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

// NewMeshError builds a MeshError from a code, a message, and any number of
// detail messages, each packed into a google.protobuf.Any. A detail that
// cannot be packed makes the result an INTERNAL error naming the detail type
// and the packing failure; c and msg are dropped, since a reply must not
// claim details it does not carry.
func NewMeshError(c code.Code, msg string, details ...proto.Message) *MeshError {
	st := &status.Status{Code: int32(c), Message: msg}
	for _, d := range details {
		a, err := anypb.New(d)
		if err != nil {
			return &MeshError{st: &status.Status{
				Code:    int32(code.Code_INTERNAL),
				Message: fmt.Sprintf("packing %T detail: %v", d, err),
			}}
		}
		st.Details = append(st.Details, a)
	}
	return &MeshError{st: st}
}

// MeshErrorFromProto wraps st. The MeshError holds st itself.
func MeshErrorFromProto(st *status.Status) *MeshError {
	return &MeshError{st: st}
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
