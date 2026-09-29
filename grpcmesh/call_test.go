package grpcmesh_test

import (
	"context"
	"errors"
	"strings"
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
	var got *testproto.Order
	hub := serveMem(t, testproto.OrderService{
		Place: func(_ context.Context, req *testproto.Order) (*testproto.Order, error) {
			got = req
			return order("reply"), nil
		},
	})

	resp, _, err := grpcmesh.Call[*testproto.Order, *testproto.Order](context.Background(), testproto.OrderTargets.Place, order("ask"), nil)
	if err != nil {
		t.Fatal(err)
	}

	if got.GetId() != "ask" {
		t.Errorf("handler received %v", got)
	}
	if resp.GetId() != "reply" {
		t.Errorf("Call returned %v", resp)
	}
	sent := hub.Runtimes()[0].Client().(*memtransport.Client).Requests()
	if len(sent) != 1 {
		t.Fatalf("Request ran %d times, want 1", len(sent))
	}
	if !sent[0].Message.Target.Equal(testproto.OrderTargets.Place) {
		t.Errorf("sent to %v", sent[0].Message.Target)
	}
	if sent[0].Message.Metadata[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("sent metadata = %v, want Content-Type", sent[0].Message.Metadata)
	}
	var wire testproto.Order
	if err := proto.Unmarshal(sent[0].Message.Payload, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.GetId() != "ask" {
		t.Errorf("sent payload decodes to %v", &wire)
	}
}

func TestCallSendsTheMetadataAsMessageMetadata(t *testing.T) {
	hub := serveMem(t, testproto.OrderService{
		Place: func(context.Context, *testproto.Order) (*testproto.Order, error) { return &testproto.Order{}, nil },
	})

	_, _, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, map[string]string{"Request-Id": "7", grpcmesh.ContentTypeKey: "text/plain"})
	if err != nil {
		t.Fatal(err)
	}

	sent := hub.Runtimes()[0].Client().(*memtransport.Client).Requests()[0]
	if sent.Message.Metadata["Request-Id"] != "7" {
		t.Errorf("sent metadata = %v, want Request-Id", sent.Message.Metadata)
	}
	if sent.Message.Metadata[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("Content-Type = %q, want the package's value over the caller's", sent.Message.Metadata[grpcmesh.ContentTypeKey])
	}
	if len(sent.Options) != 0 {
		t.Errorf("options = %v, want none", sent.Options)
	}
}

func TestCallPassesOptionKeysAsTransportOptions(t *testing.T) {
	hub := serveMem(t, testproto.OrderService{
		Place: func(context.Context, *testproto.Order) (*testproto.Order, error) { return &testproto.Order{}, nil },
	})
	md := map[string]string{"Request-Id": "7", "Mesh-Option-request_timeout": "2s"}

	if _, _, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, md); err != nil {
		t.Fatal(err)
	}

	sent := hub.Runtimes()[0].Client().(*memtransport.Client).Requests()[0]
	if len(sent.Options) != 1 || sent.Options["request_timeout"] != "2s" {
		t.Errorf("options = %v, want only request_timeout 2s", sent.Options)
	}
	if len(sent.Message.Metadata) != 2 || sent.Message.Metadata["Request-Id"] != "7" {
		t.Errorf("sent metadata = %v, want Request-Id and Content-Type only", sent.Message.Metadata)
	}
	if len(md) != 2 || md["Mesh-Option-request_timeout"] != "2s" {
		t.Errorf("caller's map = %v, want it unchanged", md)
	}
}

func TestCallReturnsTheMeshErrorAReplyCarries(t *testing.T) {
	serveMem(t, testproto.OrderService{
		Place: func(context.Context, *testproto.Order) (*testproto.Order, error) {
			return nil, grpcmesh.NewMeshError(code.Code_NOT_FOUND, "no such key", &errdetails.ErrorInfo{Reason: "GONE"})
		},
	})

	resp, _, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, nil)

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

func TestCallReturnsTheReplyMetadataOfASuccessfulReply(t *testing.T) {
	serveMem(t, testproto.OrderService{
		Place: func(ctx context.Context, _ *testproto.Order) (*testproto.Order, error) {
			grpcmesh.SetReplyMetadata(ctx, map[string]string{"Request-Id": "7"})
			return &testproto.Order{}, nil
		},
	})

	_, md, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(md) != 2 || md["Request-Id"] != "7" || md[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("reply metadata = %v, want Request-Id and Content-Type", md)
	}
}

func TestCallReturnsTheReplyMetadataOfAMeshErrorReply(t *testing.T) {
	serveMem(t, testproto.OrderService{
		Place: func(ctx context.Context, _ *testproto.Order) (*testproto.Order, error) {
			grpcmesh.SetReplyMetadata(ctx, map[string]string{"Retry-After": "30"})
			return nil, grpcmesh.NewNotFoundError("no such key")
		},
	})

	_, md, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, nil)

	var me *grpcmesh.MeshError
	if !errors.As(err, &me) {
		t.Fatalf("err = %v, want *MeshError", err)
	}
	if md["Retry-After"] != "30" || md[grpcmesh.GrpcStatusKey] != "5" {
		t.Errorf("reply metadata = %v, want Retry-After and Grpc-Status 5", md)
	}
}

func TestCallDropsOptionKeysFromTheReplyMetadata(t *testing.T) {
	serveRaw(t, []mesh.Endpoint{{Target: testproto.OrderTargets.Place, Handler: func(context.Context, mesh.Message) (mesh.Message, error) {
		md := map[string]string{"Request-Id": "7", "Mesh-Option-request_timeout": "2s"}
		return mesh.Message{Metadata: md, Payload: mustMarshal(t, order("reply"))}, nil
	}}}, nil)

	_, md, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(md) != 1 || md["Request-Id"] != "7" {
		t.Errorf("reply metadata = %v, want only Request-Id", md)
	}
}

func TestCallReturnsInternalForAnUndecodableResponse(t *testing.T) {
	serveRaw(t, []mesh.Endpoint{{Target: testproto.OrderTargets.Place, Handler: func(context.Context, mesh.Message) (mesh.Message, error) {
		return mesh.Message{Payload: garbage}, nil
	}}}, nil)

	_, _, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, nil)

	var me *grpcmesh.MeshError
	if !errors.As(err, &me) {
		t.Fatalf("err = %v, want *MeshError", err)
	}
	if me.Code() != code.Code_INTERNAL || !strings.HasPrefix(me.Message(), "reply does not decode as shop.Order: ") {
		t.Errorf("MeshError = %v %q, want INTERNAL naming the response type", me.Code(), me.Message())
	}
}

func TestCallReturnsInternalForAnUndecodableStatusPayload(t *testing.T) {
	serveRaw(t, []mesh.Endpoint{{Target: testproto.OrderTargets.Place, Handler: func(context.Context, mesh.Message) (mesh.Message, error) {
		return mesh.Message{Metadata: map[string]string{grpcmesh.GrpcStatusKey: "5"}, Payload: garbage}, nil
	}}}, nil)

	_, _, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, nil)

	var me *grpcmesh.MeshError
	if !errors.As(err, &me) {
		t.Fatalf("err = %v, want *MeshError", err)
	}
	if me.Code() != code.Code_INTERNAL || !strings.HasPrefix(me.Message(), "reply does not decode as google.rpc.Status: ") {
		t.Errorf("MeshError = %v %q, want INTERNAL naming google.rpc.Status", me.Code(), me.Message())
	}
}

func TestCallPassesTransportErrorsThroughWithNoReplyMetadata(t *testing.T) {
	serveRaw(t, nil, nil)

	_, md, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, nil)

	if !errors.Is(err, memtransport.ErrNoReceiver) {
		t.Errorf("err = %v, want the transport's error unchanged", err)
	}
	if md != nil {
		t.Errorf("reply metadata = %v, want nil", md)
	}
}

func TestCallUnknownTransport(t *testing.T) {
	freshSingletons(t)

	_, _, err := testproto.OrderClient.Place(context.Background(), &testproto.Order{}, nil)

	if !errors.Is(err, grpcmesh.ErrUnknownTransport) {
		t.Errorf("err = %v, want ErrUnknownTransport", err)
	}
}

func TestPublishSendsTheEncodedMessageToTheSubscriber(t *testing.T) {
	var got *testproto.Order
	hub := serveMem(t, testproto.OrderService{
		Placed: func(_ context.Context, req *testproto.Order) error {
			got = req
			return nil
		},
	})

	err := testproto.OrderClient.Placed(context.Background(), order("made"), nil)
	if err != nil {
		t.Fatal(err)
	}

	if got.GetId() != "made" {
		t.Errorf("subscriber received %v", got)
	}
	sent := hub.Runtimes()[0].Client().(*memtransport.Client).Publishes()
	if len(sent) != 1 {
		t.Fatalf("Publish ran %d times, want 1", len(sent))
	}
	if !sent[0].Message.Target.Equal(testproto.OrderTargets.Placed) {
		t.Errorf("sent to %v", sent[0].Message.Target)
	}
	if sent[0].Message.Metadata[grpcmesh.ContentTypeKey] != grpcmesh.ContentTypeProtobuf {
		t.Errorf("sent metadata = %v, want Content-Type", sent[0].Message.Metadata)
	}
}

func TestPublishSplitsOptionKeysFromTheMessageMetadata(t *testing.T) {
	hub := serveMem(t, testproto.OrderService{
		Placed: func(context.Context, *testproto.Order) error { return nil },
	})
	md := map[string]string{"Request-Id": "7", "Mesh-Option-publish_timeout": "2s"}

	if err := testproto.OrderClient.Placed(context.Background(), &testproto.Order{}, md); err != nil {
		t.Fatal(err)
	}

	sent := hub.Runtimes()[0].Client().(*memtransport.Client).Publishes()[0]
	if len(sent.Options) != 1 || sent.Options["publish_timeout"] != "2s" {
		t.Errorf("options = %v, want only publish_timeout 2s", sent.Options)
	}
	if len(sent.Message.Metadata) != 2 || sent.Message.Metadata["Request-Id"] != "7" {
		t.Errorf("sent metadata = %v, want Request-Id and Content-Type only", sent.Message.Metadata)
	}
}

func TestPublishPassesKindMismatchThrough(t *testing.T) {
	serveRaw(t, nil, nil)

	err := grpcmesh.Publish(context.Background(), testproto.OrderTargets.Place, &testproto.Order{}, nil)

	if !errors.Is(err, mesh.ErrKindMismatch) {
		t.Errorf("err = %v, want mesh.ErrKindMismatch unchanged", err)
	}
}
