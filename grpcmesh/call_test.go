package grpcmesh_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/memtransport"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
)

func TestCallSendsTheEncodedRequestAndDecodesTheReply(t *testing.T) {
	var got *testproto.ApiKey
	hub := serveMem(t, testproto.ApiKeyService{
		Search: func(_ context.Context, req *testproto.ApiKey) (*testproto.ApiKey, error) {
			got = req
			return apiKey("reply"), nil
		},
	})

	resp, err := grpcmesh.Call[*testproto.ApiKey, *testproto.ApiKey](context.Background(), testproto.ApiKeyTargets.Search, apiKey("ask"))
	if err != nil {
		t.Fatal(err)
	}

	if got.GetFirstName() != "ask" {
		t.Errorf("handler received %v", got)
	}
	if resp.GetFirstName() != "reply" {
		t.Errorf("Call returned %v", resp)
	}
	sent := hub.Runtimes()[0].Client().(*memtransport.Client).Requests()
	if len(sent) != 1 {
		t.Fatalf("Request ran %d times, want 1", len(sent))
	}
	if !sent[0].Message.Target.Equal(testproto.ApiKeyTargets.Search) {
		t.Errorf("sent to %v", sent[0].Message.Target)
	}
	if sent[0].Message.Metadata[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("sent metadata = %v, want Content-Type", sent[0].Message.Metadata)
	}
	var wire testproto.ApiKey
	if err := proto.Unmarshal(sent[0].Message.Payload, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.GetFirstName() != "ask" {
		t.Errorf("sent payload decodes to %v", &wire)
	}
}

func TestCallSendsOutgoingMetadataAndTransportOptions(t *testing.T) {
	hub := serveMem(t, testproto.ApiKeyService{
		Search: func(context.Context, *testproto.ApiKey) (*testproto.ApiKey, error) { return &testproto.ApiKey{}, nil },
	})
	ctx := grpcmesh.WithOutgoingMetadata(context.Background(), map[string]string{"Request-Id": "7", grpcmesh.ContentTypeKey: "text/plain"})
	ctx = grpcmesh.WithTransportOptions(ctx, map[string]string{"request_timeout": "2s"})

	if _, err := testproto.ApiKeyClient.Search(ctx, &testproto.ApiKey{}); err != nil {
		t.Fatal(err)
	}

	sent := hub.Runtimes()[0].Client().(*memtransport.Client).Requests()[0]
	if sent.Message.Metadata["Request-Id"] != "7" {
		t.Errorf("sent metadata = %v, want Request-Id", sent.Message.Metadata)
	}
	if sent.Message.Metadata[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("Content-Type = %q, want the package's value over the caller's", sent.Message.Metadata[grpcmesh.ContentTypeKey])
	}
	if sent.Options["request_timeout"] != "2s" {
		t.Errorf("options = %v, want request_timeout", sent.Options)
	}
}

func TestCallReturnsTheMeshErrorAReplyCarries(t *testing.T) {
	serveMem(t, testproto.ApiKeyService{
		Search: func(context.Context, *testproto.ApiKey) (*testproto.ApiKey, error) {
			return nil, grpcmesh.NewMeshError(code.Code_NOT_FOUND, "no such key", &errdetails.ErrorInfo{Reason: "GONE"})
		},
	})

	resp, err := testproto.ApiKeyClient.Search(context.Background(), &testproto.ApiKey{})

	if resp != nil {
		t.Errorf("resp = %v, want nil", resp)
	}
	var me *grpcmesh.MeshError
	if !errors.As(err, &me) {
		t.Fatalf("err = %v, want *MeshError", err)
	}
	if me.Code() != code.Code_NOT_FOUND || me.Message() != "no such key" {
		t.Errorf("MeshError = %v %q", me.Code(), me.Message())
	}
	var info errdetails.ErrorInfo
	if len(me.Details()) != 1 || me.Details()[0].UnmarshalTo(&info) != nil || info.GetReason() != "GONE" {
		t.Errorf("Details = %v, want one ErrorInfo GONE", me.Details())
	}
}

func TestCallReturnsInternalForAnUndecodableResponse(t *testing.T) {
	serveRaw(t, []mesh.Endpoint{{Target: testproto.ApiKeyTargets.Search, Handler: func(context.Context, mesh.Message) (mesh.Message, error) {
		return mesh.Message{Payload: garbage}, nil
	}}}, nil)

	_, err := testproto.ApiKeyClient.Search(context.Background(), &testproto.ApiKey{})

	var me *grpcmesh.MeshError
	if !errors.As(err, &me) {
		t.Fatalf("err = %v, want *MeshError", err)
	}
	if me.Code() != code.Code_INTERNAL || me.Message() == "" {
		t.Errorf("MeshError = %v %q, want INTERNAL with text", me.Code(), me.Message())
	}
}

func TestCallReturnsInternalForAnUndecodableStatusPayload(t *testing.T) {
	serveRaw(t, []mesh.Endpoint{{Target: testproto.ApiKeyTargets.Search, Handler: func(context.Context, mesh.Message) (mesh.Message, error) {
		return mesh.Message{Metadata: map[string]string{grpcmesh.GrpcStatusKey: "5"}, Payload: garbage}, nil
	}}}, nil)

	_, err := testproto.ApiKeyClient.Search(context.Background(), &testproto.ApiKey{})

	var me *grpcmesh.MeshError
	if !errors.As(err, &me) {
		t.Fatalf("err = %v, want *MeshError", err)
	}
	if me.Code() != code.Code_INTERNAL {
		t.Errorf("code = %v, want INTERNAL", me.Code())
	}
}

func TestCallPassesTransportErrorsThrough(t *testing.T) {
	serveRaw(t, nil, nil)

	_, err := testproto.ApiKeyClient.Search(context.Background(), &testproto.ApiKey{})

	if !errors.Is(err, memtransport.ErrNoReceiver) {
		t.Errorf("err = %v, want the transport's error unchanged", err)
	}
}

func TestCallUnknownTransport(t *testing.T) {
	freshSingletons(t)

	_, err := testproto.ApiKeyClient.Search(context.Background(), &testproto.ApiKey{})

	if !errors.Is(err, grpcmesh.ErrUnknownTransport) {
		t.Errorf("err = %v, want ErrUnknownTransport", err)
	}
}

func TestPublishSendsTheEncodedMessageToTheSubscriber(t *testing.T) {
	var got *testproto.ApiKey
	hub := serveMem(t, testproto.ApiKeyService{
		Created: func(_ context.Context, req *testproto.ApiKey) error {
			got = req
			return nil
		},
	})

	err := testproto.ApiKeyClient.Created(context.Background(), apiKey("made"))
	if err != nil {
		t.Fatal(err)
	}

	if got.GetFirstName() != "made" {
		t.Errorf("subscriber received %v", got)
	}
	sent := hub.Runtimes()[0].Client().(*memtransport.Client).Publishes()
	if len(sent) != 1 {
		t.Fatalf("Publish ran %d times, want 1", len(sent))
	}
	if !sent[0].Message.Target.Equal(testproto.ApiKeyTargets.Created) {
		t.Errorf("sent to %v", sent[0].Message.Target)
	}
	if sent[0].Message.Metadata[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("sent metadata = %v, want Content-Type", sent[0].Message.Metadata)
	}
}

func TestPublishPassesKindMismatchThrough(t *testing.T) {
	serveRaw(t, nil, nil)

	err := grpcmesh.Publish(context.Background(), testproto.ApiKeyTargets.Search, &testproto.ApiKey{})

	if !errors.Is(err, mesh.ErrKindMismatch) {
		t.Errorf("err = %v, want mesh.ErrKindMismatch unchanged", err)
	}
}
