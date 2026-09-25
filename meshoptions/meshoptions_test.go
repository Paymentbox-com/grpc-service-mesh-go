package meshoptions_test

import (
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/meshoptions"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestPackageRegistersTheFiveExtensionsByFullName(t *testing.T) {
	check := func(name protoreflect.FullName, want protoreflect.ExtensionType, number protoreflect.FieldNumber, extendee protoreflect.FullName) {
		got, err := protoregistry.GlobalTypes.FindExtensionByName(name)
		if err != nil {
			t.Errorf("FindExtensionByName(%q): %v", name, err)
			return
		}
		if got != want {
			t.Errorf("%s resolves to %v, want the meshoptions extension", name, got)
		}
		desc := got.TypeDescriptor()
		if desc.Number() != number {
			t.Errorf("%s has number %d, want %d", name, desc.Number(), number)
		}
		if desc.ContainingMessage().FullName() != extendee {
			t.Errorf("%s extends %s, want %s", name, desc.ContainingMessage().FullName(), extendee)
		}
	}

	check("mesh.kind", meshoptions.E_Kind, 50001, "google.protobuf.MethodOptions")
	check("mesh.consumer_group", meshoptions.E_ConsumerGroup, 50002, "google.protobuf.MethodOptions")
	check("mesh.deployment_group", meshoptions.E_DeploymentGroup, 50003, "google.protobuf.FileOptions")
	check("mesh.transport", meshoptions.E_Transport, 50004, "google.protobuf.FileOptions")
	check("mesh.root_prefix", meshoptions.E_RootPrefix, 50005, "google.protobuf.FileOptions")
}
