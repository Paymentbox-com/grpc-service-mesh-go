package grpcmesh_test

import (
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

func TestCodeConstructorCarriesMessageAndDetails(t *testing.T) {
	me := grpcmesh.NewNotFoundError("no such key", &errdetails.ErrorInfo{Reason: "GONE"})

	if me.Code() != code.Code_NOT_FOUND {
		t.Errorf("Code() = %v, want NOT_FOUND", me.Code())
	}
	if me.Message() != "no such key" {
		t.Errorf("Message() = %q, want %q", me.Message(), "no such key")
	}
	if len(me.Details()) != 1 {
		t.Errorf("Details() has %d entries, want 1", len(me.Details()))
	}
}

func TestEachCodeConstructorSetsItsCode(t *testing.T) {
	check := func(name string, me *grpcmesh.MeshError, want code.Code) {
		if me.Code() != want {
			t.Errorf("%s: Code() = %v, want %v", name, me.Code(), want)
		}
	}
	check("Cancelled", grpcmesh.NewCancelledError(""), code.Code_CANCELLED)
	check("Unknown", grpcmesh.NewUnknownError(""), code.Code_UNKNOWN)
	check("InvalidArgument", grpcmesh.NewInvalidArgumentError(""), code.Code_INVALID_ARGUMENT)
	check("DeadlineExceeded", grpcmesh.NewDeadlineExceededError(""), code.Code_DEADLINE_EXCEEDED)
	check("NotFound", grpcmesh.NewNotFoundError(""), code.Code_NOT_FOUND)
	check("AlreadyExists", grpcmesh.NewAlreadyExistsError(""), code.Code_ALREADY_EXISTS)
	check("PermissionDenied", grpcmesh.NewPermissionDeniedError(""), code.Code_PERMISSION_DENIED)
	check("Unauthenticated", grpcmesh.NewUnauthenticatedError(""), code.Code_UNAUTHENTICATED)
	check("ResourceExhausted", grpcmesh.NewResourceExhaustedError(""), code.Code_RESOURCE_EXHAUSTED)
	check("FailedPrecondition", grpcmesh.NewFailedPreconditionError(""), code.Code_FAILED_PRECONDITION)
	check("Aborted", grpcmesh.NewAbortedError(""), code.Code_ABORTED)
	check("OutOfRange", grpcmesh.NewOutOfRangeError(""), code.Code_OUT_OF_RANGE)
	check("Unimplemented", grpcmesh.NewUnimplementedError(""), code.Code_UNIMPLEMENTED)
	check("Internal", grpcmesh.NewInternalError(""), code.Code_INTERNAL)
	check("Unavailable", grpcmesh.NewUnavailableError(""), code.Code_UNAVAILABLE)
	check("DataLoss", grpcmesh.NewDataLossError(""), code.Code_DATA_LOSS)
}

func TestCodeConstantsAreTheGoogleRpcCodes(t *testing.T) {
	if grpcmesh.OK != code.Code_OK || grpcmesh.NotFound != code.Code_NOT_FOUND {
		t.Errorf("constants differ from google.rpc.Code")
	}
}
