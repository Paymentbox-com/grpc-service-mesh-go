package grpcmesh_test

import (
	"strings"
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

func TestNewMeshErrorDetailsRoundTripThroughProto(t *testing.T) {
	info := &errdetails.ErrorInfo{Reason: "EXPIRED", Domain: "shop"}
	me := grpcmesh.NewMeshError(code.Code_FAILED_PRECONDITION, "expired", info)

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

func TestNewMeshErrorWithUnpackableDetailIsInternal(t *testing.T) {
	bad := &errdetails.ErrorInfo{Reason: "\xff"} // invalid UTF-8 in a proto3 string

	me := grpcmesh.NewMeshError(code.Code_NOT_FOUND, "no such key", bad)

	if me.Code() != code.Code_INTERNAL {
		t.Errorf("Code() = %v, want INTERNAL", me.Code())
	}
	if !strings.HasPrefix(me.Message(), "packing *errdetails.ErrorInfo detail: ") {
		t.Errorf("Message() = %q, want the detail type and the packing failure", me.Message())
	}
	if len(me.Details()) != 0 {
		t.Errorf("Details() = %v, want none", me.Details())
	}
}

func TestMeshErrorTextIsCodeNameAndMessage(t *testing.T) {
	err := error(grpcmesh.NewMeshError(code.Code_PERMISSION_DENIED, "not yours"))

	if err.Error() != "PERMISSION_DENIED: not yours" {
		t.Errorf("Error() = %q", err.Error())
	}
}
