package grpcmesh

import (
	"context"
	"maps"
)

type contextKey int

const (
	incomingKey contextKey = iota
	outgoingKey
	optionsKey
	handlerReplyKey
	callerReplyKey
)

// IncomingMetadata returns the metadata of the message a handler is serving,
// or nil outside a handler.
func IncomingMetadata(ctx context.Context) map[string]string {
	md, _ := ctx.Value(incomingKey).(map[string]string)
	return md
}

// SetReplyMetadata merges md into the metadata of the reply the endpoint
// handler serving ctx sends. A later call adds keys and overwrites the ones
// already set. Content-Type and Grpc-Status are set by the package and
// override values in md. Outside an endpoint handler it has no effect.
func SetReplyMetadata(ctx context.Context, md map[string]string) {
	if reply, ok := ctx.Value(handlerReplyKey).(map[string]string); ok {
		maps.Copy(reply, md)
	}
}

// WithReplyMetadata returns a context that makes Call set *md to a copy of
// the reply's metadata, on a successful reply and on a MeshError reply. A
// failure before a reply arrives, such as a transport error, leaves *md
// unchanged. Publish ignores it.
func WithReplyMetadata(ctx context.Context, md *map[string]string) context.Context {
	return context.WithValue(ctx, callerReplyKey, md)
}

// WithOutgoingMetadata returns a context that makes Call and Publish send md
// as the message metadata. Content-Type is set by the package and overrides
// a value in md.
func WithOutgoingMetadata(ctx context.Context, md map[string]string) context.Context {
	return context.WithValue(ctx, outgoingKey, md)
}

// WithTransportOptions returns a context that makes Call and Publish pass
// opts as the per-call options of the transport's Request or Publish.
func WithTransportOptions(ctx context.Context, opts map[string]string) context.Context {
	return context.WithValue(ctx, optionsKey, opts)
}

func withIncomingMetadata(ctx context.Context, md map[string]string) context.Context {
	return context.WithValue(ctx, incomingKey, md)
}

func withHandlerReply(ctx context.Context, reply map[string]string) context.Context {
	return context.WithValue(ctx, handlerReplyKey, reply)
}

func callerReplyMetadata(ctx context.Context) *map[string]string {
	md, _ := ctx.Value(callerReplyKey).(*map[string]string)
	return md
}

func outgoingMetadata(ctx context.Context) map[string]string {
	md, _ := ctx.Value(outgoingKey).(map[string]string)
	return md
}

func transportOptions(ctx context.Context) map[string]string {
	opts, _ := ctx.Value(optionsKey).(map[string]string)
	return opts
}
