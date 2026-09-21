package grpcmesh

import (
	"context"

	"github.com/Paymentbox-com/service-mesh-go/mesh"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
)

// Call sends req to the ROUTE target t through the client that
// DefaultTransportRouter holds for t's transport metadata and decodes the
// reply into a fresh Resp. The message carries the metadata set with
// WithOutgoingMetadata plus Content-Type, and the transport receives the
// options set with WithTransportOptions.
//
// Grpc-Status on the reply is read before the payload. When it is set, the
// payload is decoded as google.rpc.Status and returned as a *MeshError. When
// it is not, the payload is decoded as Resp. A payload that does not decode,
// or a req that does not encode, returns an INTERNAL *MeshError. Router,
// Service Mesh API, and transport errors are returned unchanged.
func Call[Req, Resp proto.Message](ctx context.Context, t mesh.Target, req Req) (Resp, error) {
	var zero Resp
	c, msg, err := prepare(ctx, t, req)
	if err != nil {
		return zero, err
	}
	reply, err := c.Request(ctx, msg, transportOptions(ctx))
	if err != nil {
		return zero, err
	}
	if _, ok := reply.Metadata[GrpcStatusKey]; ok {
		st, err := decode[*status.Status](reply.Payload)
		if err != nil {
			return zero, NewMeshError(code.Code_INTERNAL, err.Error())
		}
		return zero, MeshErrorFromProto(st)
	}
	resp, err := decode[Resp](reply.Payload)
	if err != nil {
		return zero, NewMeshError(code.Code_INTERNAL, err.Error())
	}
	return resp, nil
}

// Publish sends req to the TOPIC target t through the client that
// DefaultTransportRouter holds for t's transport metadata. Metadata and
// options come from the context as for Call. A req that does not encode
// returns an INTERNAL *MeshError. Router, Service Mesh API, and transport
// errors are returned unchanged.
func Publish[Req proto.Message](ctx context.Context, t mesh.Target, req Req) error {
	c, msg, err := prepare(ctx, t, req)
	if err != nil {
		return err
	}
	return c.Publish(ctx, msg, transportOptions(ctx))
}

// prepare resolves the client for t and encodes req as a message to t.
func prepare(ctx context.Context, t mesh.Target, req proto.Message) (mesh.Client, mesh.Message, error) {
	c, err := DefaultTransportRouter.Client(t.Metadata[TransportKey])
	if err != nil {
		return nil, mesh.Message{}, err
	}
	payload, err := proto.Marshal(req)
	if err != nil {
		return nil, mesh.Message{}, NewMeshError(code.Code_INTERNAL, err.Error())
	}
	msg := mesh.Message{Target: t, Metadata: contentType(outgoingMetadata(ctx)), Payload: payload}
	return c, msg, nil
}
