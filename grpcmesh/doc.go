// Package grpcmesh is the Go library of the gRPC Service Mesh API. It carries
// protobuf messages over any transport that implements the Service Mesh API
// contract in github.com/Paymentbox-com/service-mesh-go/mesh.
//
// The package holds the non-generated types of the specification: the
// TransportRouter, the Registry, the RPCRuntime, and MeshError, plus the
// generic helpers that generated code calls: NewEndpoint, NewSubscriber,
// Call, and Publish. Generated code lives in the definitions project and
// refers to the two process singletons, DefaultTransportRouter and
// DefaultRegistry, which the application configures at boot.
//
// Every message the package sends carries metadata Content-Type set to
// application/x-protobuf. A reply that carries a MeshError also carries
// Grpc-Status, the google.rpc.Code as a decimal string, and its payload is
// the encoded google.rpc.Status.
package grpcmesh

// Metadata keys the package writes and reads.
const (
	// ContentTypeKey is the metadata key every sent message carries.
	ContentTypeKey = "Content-Type"

	// ContentTypeProtobuf is the value of ContentTypeKey on every sent message.
	ContentTypeProtobuf = "application/x-protobuf"

	// GrpcStatusKey marks a reply whose payload is an encoded google.rpc.Status.
	// Its value is the code as a decimal integer string.
	GrpcStatusKey = "Grpc-Status"

	// TransportKey is the Target metadata key naming the transport a Target
	// is served over. The TransportRouter resolves the value.
	TransportKey = "transport"
)
