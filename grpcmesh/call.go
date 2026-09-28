package grpcmesh

import (
	"context"
	"strings"

	"github.com/Paymentbox-com/service-mesh-go/mesh"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
)

// Call sends req to the ROUTE target t through the client that
// DefaultTransportRouter holds for t's transport metadata and decodes the
// reply into a fresh Resp.
//
// The keys of md that start with OptionPrefix are transport options: the
// transport's Request receives each one with the prefix removed. The other
// keys of md are the message metadata, and the package sets Content-Type
// over a value in md.
//
// The returned map is a copy of the reply's metadata without keys that
// start with OptionPrefix, on a successful reply and on a MeshError reply.
// It is nil when no reply arrived.
//
// Grpc-Status on the reply is read before the payload. When it is set, the
// payload is decoded as google.rpc.Status and returned as a *MeshError. When
// it is not, the payload is decoded as Resp. A payload that does not decode,
// or a req that does not encode, returns an INTERNAL *MeshError. Router,
// Service Mesh API, and transport errors are returned unchanged.
func Call[Req, Resp proto.Message](ctx context.Context, t mesh.Target, req Req, md map[string]string) (Resp, map[string]string, error) {
	var zero Resp
	c, msg, opts, err := prepare(t, req, md)
	if err != nil {
		return zero, nil, err
	}
	reply, err := c.Request(ctx, msg, opts)
	if err != nil {
		return zero, nil, err
	}
	replyMD, _ := splitOptions(reply.Metadata)
	if _, ok := reply.Metadata[GrpcStatusKey]; ok {
		st, err := decode[*status.Status](reply.Payload)
		if err != nil {
			return zero, replyMD, NewMeshError(code.Code_INTERNAL, err.Error())
		}
		return zero, replyMD, MeshErrorFromProto(st)
	}
	resp, err := decode[Resp](reply.Payload)
	if err != nil {
		return zero, replyMD, NewMeshError(code.Code_INTERNAL, err.Error())
	}
	return resp, replyMD, nil
}

// Publish sends req to the TOPIC target t through the client that
// DefaultTransportRouter holds for t's transport metadata. md is split into
// message metadata and transport options as for Call. A req that does not
// encode returns an INTERNAL *MeshError. Router, Service Mesh API, and
// transport errors are returned unchanged.
func Publish[Req proto.Message](ctx context.Context, t mesh.Target, req Req, md map[string]string) error {
	c, msg, opts, err := prepare(t, req, md)
	if err != nil {
		return err
	}
	return c.Publish(ctx, msg, opts)
}

// prepare resolves the client for t, encodes req as a message to t carrying
// the message metadata of md, and returns the transport options of md.
func prepare(t mesh.Target, req proto.Message, md map[string]string) (mesh.Client, mesh.Message, map[string]string, error) {
	c, err := DefaultTransportRouter.Client(t.Metadata[TransportKey])
	if err != nil {
		return nil, mesh.Message{}, nil, err
	}
	payload, err := proto.Marshal(req)
	if err != nil {
		return nil, mesh.Message{}, nil, NewMeshError(code.Code_INTERNAL, err.Error())
	}
	msgMD, opts := splitOptions(md)
	msgMD[ContentTypeKey] = ContentTypeProtobuf
	return c, mesh.Message{Target: t, Metadata: msgMD, Payload: payload}, opts, nil
}

// splitOptions returns a copy of md without the keys that start with
// OptionPrefix, and a map of those keys with the prefix removed.
func splitOptions(md map[string]string) (map[string]string, map[string]string) {
	rest := make(map[string]string, len(md)+1)
	opts := map[string]string{}
	for k, v := range md {
		if name, ok := strings.CutPrefix(k, OptionPrefix); ok {
			opts[name] = v
		} else {
			rest[k] = v
		}
	}
	return rest, opts
}
