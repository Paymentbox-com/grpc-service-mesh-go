package grpcmesh

import (
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// MeshError is an application failure carried as a google.rpc.Status. A
// ROUTE handler returns one as its error; a caller receives one from Call
// through errors.As.
type MeshError struct {
	st *status.Status
}

// NewMeshError returns a MeshError with code c and message msg and no details.
func NewMeshError(c code.Code, msg string) *MeshError {
	return &MeshError{st: &status.Status{Code: int32(c), Message: msg}}
}

// MeshErrorFromProto wraps st. The MeshError holds st itself.
func MeshErrorFromProto(st *status.Status) *MeshError {
	return &MeshError{st: st}
}

// WithDetails returns a copy of e with details packed into google.protobuf.Any
// and appended. e is unchanged. The error is anypb's when a detail cannot be
// encoded.
func (e *MeshError) WithDetails(details ...proto.Message) (*MeshError, error) {
	st := proto.Clone(e.st).(*status.Status)
	for _, d := range details {
		a, err := anypb.New(d)
		if err != nil {
			return nil, err
		}
		st.Details = append(st.Details, a)
	}
	return &MeshError{st: st}, nil
}

// Code returns the google.rpc.Code.
func (e *MeshError) Code() code.Code {
	return code.Code(e.st.GetCode())
}

// Message returns the human-readable text.
func (e *MeshError) Message() string {
	return e.st.GetMessage()
}

// Details returns the packed detail messages.
func (e *MeshError) Details() []*anypb.Any {
	return e.st.GetDetails()
}

// Proto returns the wrapped google.rpc.Status.
func (e *MeshError) Proto() *status.Status {
	return e.st
}

// Error returns the code name and the message, as in "NOT_FOUND: no such key".
func (e *MeshError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code(), e.Message())
}
