package grpcmesh_test

import (
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
)

func TestNewMeshErrorCarriesCodeAndMessage(t *testing.T) {
	me := grpcmesh.NewMeshError(code.Code_NOT_FOUND, "no such key")

	if me.Code() != code.Code_NOT_FOUND {
		t.Errorf("Code() = %v, want NOT_FOUND", me.Code())
	}
	if me.Message() != "no such key" {
		t.Errorf("Message() = %q, want %q", me.Message(), "no such key")
	}
	if len(me.Details()) != 0 {
		t.Errorf("Details() = %v, want none", me.Details())
	}
	if me.Proto().GetCode() != int32(code.Code_NOT_FOUND) || me.Proto().GetMessage() != "no such key" {
		t.Errorf("Proto() = %v, want code 5 and the message", me.Proto())
	}
}

func TestWithDetailsRoundTripsThroughProto(t *testing.T) {
	info := &errdetails.ErrorInfo{Reason: "EXPIRED", Domain: "pbx"}
	me, err := grpcmesh.NewMeshError(code.Code_FAILED_PRECONDITION, "expired").WithDetails(info)
	if err != nil {
		t.Fatal(err)
	}

	back := grpcmesh.MeshErrorFromProto(me.Proto())

	if back.Code() != code.Code_FAILED_PRECONDITION || back.Message() != "expired" {
		t.Errorf("round trip gave %v %q", back.Code(), back.Message())
	}
	if len(back.Details()) != 1 {
		t.Fatalf("Details() has %d entries, want 1", len(back.Details()))
	}
	var got errdetails.ErrorInfo
	if err := back.Details()[0].UnmarshalTo(&got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(&got, info) {
		t.Errorf("detail = %v, want %v", &got, info)
	}
}

func TestWithDetailsLeavesTheReceiverUnchanged(t *testing.T) {
	base := grpcmesh.NewMeshError(code.Code_INVALID_ARGUMENT, "bad")

	derived, err := base.WithDetails(&errdetails.ErrorInfo{Reason: "R"})
	if err != nil {
		t.Fatal(err)
	}

	if len(base.Details()) != 0 {
		t.Errorf("receiver gained %d details", len(base.Details()))
	}
	if len(derived.Details()) != 1 {
		t.Errorf("result has %d details, want 1", len(derived.Details()))
	}
}

func TestMeshErrorTextIsCodeNameAndMessage(t *testing.T) {
	err := error(grpcmesh.NewMeshError(code.Code_PERMISSION_DENIED, "not yours"))

	if err.Error() != "PERMISSION_DENIED: not yours" {
		t.Errorf("Error() = %q", err.Error())
	}
}
