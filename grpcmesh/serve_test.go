package grpcmesh_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
)

var garbage = []byte{0xff, 0xff, 0xff, 0xff}

// statusOf decodes a reply carrying a MeshError.
func statusOf(t *testing.T, reply mesh.Message) *status.Status {
	t.Helper()
	var st status.Status
	if err := proto.Unmarshal(reply.Payload, &st); err != nil {
		t.Fatal(err)
	}
	return &st
}

func TestEndpointRepliesWithTheEncodedResponse(t *testing.T) {
	var got *testproto.ApiKey
	ep := grpcmesh.NewEndpoint(testproto.ApiKeyTargets.Search, func(_ context.Context, req *testproto.ApiKey) (*testproto.ApiKey, error) {
		got = req
		return apiKey("reply"), nil
	})

	reply, err := ep.Handler(context.Background(), mesh.Message{Payload: mustMarshal(t, apiKey("ask"))})
	if err != nil {
		t.Fatal(err)
	}

	if !ep.Target.Equal(testproto.ApiKeyTargets.Search) {
		t.Errorf("endpoint target = %v", ep.Target)
	}
	if got.GetFirstName() != "ask" {
		t.Errorf("handler received %v", got)
	}
	var resp testproto.ApiKey
	if err := proto.Unmarshal(reply.Payload, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.GetFirstName() != "reply" {
		t.Errorf("reply decodes to %v", &resp)
	}
	if reply.Metadata[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("reply metadata = %v, want Content-Type", reply.Metadata)
	}
	if _, ok := reply.Metadata[grpcmesh.GrpcStatusKey]; ok {
		t.Error("a successful reply carries Grpc-Status")
	}
}

func TestEndpointExposesTheIncomingMetadata(t *testing.T) {
	var seen map[string]string
	ep := grpcmesh.NewEndpoint(testproto.ApiKeyTargets.Search, func(ctx context.Context, _ *testproto.ApiKey) (*testproto.ApiKey, error) {
		seen = grpcmesh.IncomingMetadata(ctx)
		return &testproto.ApiKey{}, nil
	})

	if _, err := ep.Handler(context.Background(), mesh.Message{Metadata: map[string]string{"Request-Id": "7"}}); err != nil {
		t.Fatal(err)
	}

	if seen["Request-Id"] != "7" {
		t.Errorf("IncomingMetadata = %v", seen)
	}
}

func TestEndpointRepliesAMeshErrorAsAStatus(t *testing.T) {
	me := grpcmesh.NewMeshError(code.Code_NOT_FOUND, "no such key", &errdetails.ErrorInfo{Reason: "GONE"})
	ep := grpcmesh.NewEndpoint(testproto.ApiKeyTargets.Search, func(context.Context, *testproto.ApiKey) (*testproto.ApiKey, error) {
		return nil, me
	})

	reply, err := ep.Handler(context.Background(), mesh.Message{})
	if err != nil {
		t.Fatal(err)
	}

	if reply.Metadata[grpcmesh.GrpcStatusKey] != "5" {
		t.Errorf("Grpc-Status = %q, want 5", reply.Metadata[grpcmesh.GrpcStatusKey])
	}
	if reply.Metadata[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("reply metadata = %v, want Content-Type", reply.Metadata)
	}
	st := statusOf(t, reply)
	if st.GetCode() != 5 || st.GetMessage() != "no such key" || len(st.GetDetails()) != 1 {
		t.Errorf("status = %v", st)
	}
}

func TestEndpointRepliesUnknownForAnyOtherError(t *testing.T) {
	ep := grpcmesh.NewEndpoint(testproto.ApiKeyTargets.Search, func(context.Context, *testproto.ApiKey) (*testproto.ApiKey, error) {
		return nil, errors.New("database down")
	})

	reply, err := ep.Handler(context.Background(), mesh.Message{})
	if err != nil {
		t.Fatal(err)
	}

	if reply.Metadata[grpcmesh.GrpcStatusKey] != "2" {
		t.Errorf("Grpc-Status = %q, want 2", reply.Metadata[grpcmesh.GrpcStatusKey])
	}
	st := statusOf(t, reply)
	if st.GetCode() != 2 || st.GetMessage() != "database down" {
		t.Errorf("status = %v", st)
	}
}

func TestEndpointRepliesUnknownForAPanic(t *testing.T) {
	ep := grpcmesh.NewEndpoint(testproto.ApiKeyTargets.Search, func(context.Context, *testproto.ApiKey) (*testproto.ApiKey, error) {
		panic("boom")
	})

	reply, err := ep.Handler(context.Background(), mesh.Message{})
	if err != nil {
		t.Fatal(err)
	}

	if reply.Metadata[grpcmesh.GrpcStatusKey] != "2" {
		t.Errorf("Grpc-Status = %q, want 2", reply.Metadata[grpcmesh.GrpcStatusKey])
	}
	if st := statusOf(t, reply); !strings.Contains(st.GetMessage(), "boom") {
		t.Errorf("status message = %q, want the panic value in it", st.GetMessage())
	}
}

func TestEndpointRepliesInternalForAnUndecodableRequest(t *testing.T) {
	called := false
	ep := grpcmesh.NewEndpoint(testproto.ApiKeyTargets.Search, func(context.Context, *testproto.ApiKey) (*testproto.ApiKey, error) {
		called = true
		return nil, nil
	})

	reply, err := ep.Handler(context.Background(), mesh.Message{Payload: garbage})
	if err != nil {
		t.Fatal(err)
	}

	if called {
		t.Error("handler ran on an undecodable request")
	}
	if reply.Metadata[grpcmesh.GrpcStatusKey] != "13" {
		t.Errorf("Grpc-Status = %q, want 13", reply.Metadata[grpcmesh.GrpcStatusKey])
	}
	if st := statusOf(t, reply); st.GetCode() != 13 || st.GetMessage() == "" {
		t.Errorf("status = %v, want INTERNAL with text", st)
	}
}

func TestSubscriberDeliversTheDecodedMessageAndMetadata(t *testing.T) {
	var got *testproto.ApiKey
	var seen map[string]string
	sub := grpcmesh.NewSubscriber(testproto.ApiKeyTargets.Created, func(ctx context.Context, req *testproto.ApiKey) error {
		got, seen = req, grpcmesh.IncomingMetadata(ctx)
		return nil
	})

	err := sub.Handler(context.Background(), mesh.Message{
		Metadata: map[string]string{"Event-Id": "9"},
		Payload:  mustMarshal(t, apiKey("made")),
	})

	if err != nil {
		t.Fatal(err)
	}
	if !sub.Target.Equal(testproto.ApiKeyTargets.Created) {
		t.Errorf("subscriber target = %v", sub.Target)
	}
	if got.GetFirstName() != "made" {
		t.Errorf("handler received %v", got)
	}
	if seen["Event-Id"] != "9" {
		t.Errorf("IncomingMetadata = %v", seen)
	}
}

func TestSubscriberReturnsTheHandlerErrorUnchanged(t *testing.T) {
	failure := errors.New("cannot record")
	sub := grpcmesh.NewSubscriber(testproto.ApiKeyTargets.Created, func(context.Context, *testproto.ApiKey) error {
		return failure
	})

	err := sub.Handler(context.Background(), mesh.Message{})

	if err != failure {
		t.Errorf("err = %v, want the handler's error itself", err)
	}
}

func TestSubscriberReturnsTheDecodeErrorWithoutRunningTheHandler(t *testing.T) {
	called := false
	sub := grpcmesh.NewSubscriber(testproto.ApiKeyTargets.Created, func(context.Context, *testproto.ApiKey) error {
		called = true
		return nil
	})

	err := sub.Handler(context.Background(), mesh.Message{Payload: garbage})

	if err == nil {
		t.Fatal("err = nil, want the decode error")
	}
	if called {
		t.Error("handler ran on an undecodable message")
	}
}

func TestIncomingMetadataOutsideAHandlerIsNil(t *testing.T) {
	if md := grpcmesh.IncomingMetadata(context.Background()); md != nil {
		t.Errorf("IncomingMetadata = %v, want nil", md)
	}
}
