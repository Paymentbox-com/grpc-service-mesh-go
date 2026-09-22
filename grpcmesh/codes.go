package grpcmesh

import (
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/protobuf/proto"
)

// The google.rpc.Code values, so handler and caller code reads them without
// importing the code package.
const (
	OK                 = code.Code_OK
	Cancelled          = code.Code_CANCELLED
	Unknown            = code.Code_UNKNOWN
	InvalidArgument    = code.Code_INVALID_ARGUMENT
	DeadlineExceeded   = code.Code_DEADLINE_EXCEEDED
	NotFound           = code.Code_NOT_FOUND
	AlreadyExists      = code.Code_ALREADY_EXISTS
	PermissionDenied   = code.Code_PERMISSION_DENIED
	Unauthenticated    = code.Code_UNAUTHENTICATED
	ResourceExhausted  = code.Code_RESOURCE_EXHAUSTED
	FailedPrecondition = code.Code_FAILED_PRECONDITION
	Aborted            = code.Code_ABORTED
	OutOfRange         = code.Code_OUT_OF_RANGE
	Unimplemented      = code.Code_UNIMPLEMENTED
	Internal           = code.Code_INTERNAL
	Unavailable        = code.Code_UNAVAILABLE
	DataLoss           = code.Code_DATA_LOSS
)

// One constructor per code. Each is NewMeshError with the code fixed.

func NewCancelledError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(Cancelled, msg, details...)
}

func NewUnknownError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(Unknown, msg, details...)
}

func NewInvalidArgumentError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(InvalidArgument, msg, details...)
}

func NewDeadlineExceededError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(DeadlineExceeded, msg, details...)
}

func NewNotFoundError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(NotFound, msg, details...)
}

func NewAlreadyExistsError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(AlreadyExists, msg, details...)
}

func NewPermissionDeniedError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(PermissionDenied, msg, details...)
}

func NewUnauthenticatedError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(Unauthenticated, msg, details...)
}

func NewResourceExhaustedError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(ResourceExhausted, msg, details...)
}

func NewFailedPreconditionError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(FailedPrecondition, msg, details...)
}

func NewAbortedError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(Aborted, msg, details...)
}

func NewOutOfRangeError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(OutOfRange, msg, details...)
}

func NewUnimplementedError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(Unimplemented, msg, details...)
}

func NewInternalError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(Internal, msg, details...)
}

func NewUnavailableError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(Unavailable, msg, details...)
}

func NewDataLossError(msg string, details ...proto.Message) *MeshError {
	return NewMeshError(DataLoss, msg, details...)
}
