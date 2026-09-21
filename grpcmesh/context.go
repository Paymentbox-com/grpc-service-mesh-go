package grpcmesh

import "context"

type contextKey int

const (
	incomingKey contextKey = iota
	outgoingKey
	optionsKey
)

// IncomingMetadata returns the metadata of the message a handler is serving,
// or nil outside a handler.
func IncomingMetadata(ctx context.Context) map[string]string {
	md, _ := ctx.Value(incomingKey).(map[string]string)
	return md
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

func outgoingMetadata(ctx context.Context) map[string]string {
	md, _ := ctx.Value(outgoingKey).(map[string]string)
	return md
}

func transportOptions(ctx context.Context) map[string]string {
	opts, _ := ctx.Value(optionsKey).(map[string]string)
	return opts
}
