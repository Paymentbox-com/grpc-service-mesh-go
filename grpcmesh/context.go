package grpcmesh

import (
	"context"
	"strings"
)

type contextKey int

const (
	incomingKey contextKey = iota
	handlerReplyKey
)

// IncomingMetadata returns the metadata of the message a handler is serving,
// or nil outside a handler.
func IncomingMetadata(ctx context.Context) map[string]string {
	md, _ := ctx.Value(incomingKey).(map[string]string)
	return md
}

// SetReplyMetadata merges md into the metadata of the reply the endpoint
// handler serving ctx sends. A later call adds keys and overwrites the ones
// already set. Keys that start with OptionPrefix are dropped. Content-Type
// and Grpc-Status are set by the package and override values in md. Outside
// an endpoint handler it has no effect.
func SetReplyMetadata(ctx context.Context, md map[string]string) {
	if reply, ok := ctx.Value(handlerReplyKey).(map[string]string); ok {
		for k, v := range md {
			if !strings.HasPrefix(k, OptionPrefix) {
				reply[k] = v
			}
		}
	}
}

func withIncomingMetadata(ctx context.Context, md map[string]string) context.Context {
	return context.WithValue(ctx, incomingKey, md)
}

func withHandlerReply(ctx context.Context, reply map[string]string) context.Context {
	return context.WithValue(ctx, handlerReplyKey, reply)
}
