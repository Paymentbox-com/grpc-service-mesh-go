package grpcmesh

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"

	"github.com/Paymentbox-com/service-mesh-go/mesh"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/protobuf/proto"
)

// NewEndpoint wraps fn as the mesh.Endpoint for a ROUTE target. The handler
// decodes the request into a fresh Req, exposes the message metadata through
// IncomingMetadata, and replies with the encoded Resp under Content-Type
// application/x-protobuf.
//
// The reply carries the metadata fn sets with SetReplyMetadata, on a
// successful reply and on a MeshError reply. The package writes Content-Type
// and Grpc-Status after the application's values, so its own values win, and
// a successful reply carries no Grpc-Status.
//
// A *MeshError from fn becomes a reply whose payload is the encoded
// google.rpc.Status and whose Grpc-Status metadata is the code as a decimal
// string. Any other error, or a panic, is reported the same way as UNKNOWN
// with the failure's text. A request that does not decode, or a response
// that does not encode, is reported as INTERNAL with the protobuf error's
// text. The handler never returns an error to the runtime.
func NewEndpoint[Req, Resp proto.Message](t mesh.Target, fn func(context.Context, Req) (Resp, error)) mesh.Endpoint {
	return mesh.Endpoint{
		Target: t,
		Handler: func(ctx context.Context, m mesh.Message) (mesh.Message, error) {
			req, err := decode[Req](m.Payload)
			if err != nil {
				return statusReply(NewMeshError(code.Code_INTERNAL, err.Error()), contentType()), nil
			}
			reply := map[string]string{}
			ctx = withHandlerReply(withIncomingMetadata(ctx, m.Metadata), reply)
			resp, err := serve(ctx, req, fn)
			if err != nil {
				return statusReply(asMeshError(err), withContentType(reply)), nil
			}
			payload, err := proto.Marshal(resp)
			if err != nil {
				return statusReply(NewMeshError(code.Code_INTERNAL, err.Error()), withContentType(reply)), nil
			}
			md := withContentType(reply)
			delete(md, GrpcStatusKey)
			return mesh.Message{Metadata: md, Payload: payload}, nil
		},
	}
}

// NewSubscriber wraps fn as the mesh.Subscriber for a TOPIC target. The
// handler decodes the message into a fresh Req, exposes the message metadata
// through IncomingMetadata, and returns fn's error to the runtime unchanged.
// A message that does not decode returns the protobuf error. Panics are not
// recovered.
func NewSubscriber[Req proto.Message](t mesh.Target, fn func(context.Context, Req) error) mesh.Subscriber {
	return mesh.Subscriber{
		Target: t,
		Handler: func(ctx context.Context, m mesh.Message) error {
			req, err := decode[Req](m.Payload)
			if err != nil {
				return err
			}
			return fn(withIncomingMetadata(ctx, m.Metadata), req)
		},
	}
}

// serve runs fn and turns a panic into an error.
func serve[Req, Resp proto.Message](ctx context.Context, req Req, fn func(context.Context, Req) (Resp, error)) (resp Resp, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return fn(ctx, req)
}

// asMeshError returns err as a *MeshError, or UNKNOWN with err's text.
func asMeshError(err error) *MeshError {
	var me *MeshError
	if errors.As(err, &me) {
		return me
	}
	return NewMeshError(code.Code_UNKNOWN, err.Error())
}

// statusReply encodes me as a reply message whose metadata is md with
// Grpc-Status set.
func statusReply(me *MeshError, md map[string]string) mesh.Message {
	payload, err := proto.Marshal(me.Proto())
	if err != nil {
		// A Status holding only a code and text always encodes.
		me = NewMeshError(code.Code_INTERNAL, err.Error())
		payload, _ = proto.Marshal(me.Proto())
	}
	md[GrpcStatusKey] = strconv.Itoa(int(me.Code()))
	return mesh.Message{Metadata: md, Payload: payload}
}

// decode unmarshals payload into a fresh message of type M, obtained from
// the type's zero value, so M is a pointer to a generated message type.
func decode[M proto.Message](payload []byte) (M, error) {
	var zero M
	m := zero.ProtoReflect().New().Interface().(M)
	if err := proto.Unmarshal(payload, m); err != nil {
		return zero, err
	}
	return m, nil
}

// contentType returns a new map holding Content-Type.
func contentType() map[string]string {
	return map[string]string{ContentTypeKey: ContentTypeProtobuf}
}

// withContentType returns a copy of md with Content-Type set.
func withContentType(md map[string]string) map[string]string {
	out := make(map[string]string, len(md)+1)
	maps.Copy(out, md)
	out[ContentTypeKey] = ContentTypeProtobuf
	return out
}
