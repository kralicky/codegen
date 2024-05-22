package cli

import (
	"github.com/kralicky/codegen/pkg/extensions"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func getGeneratorOptions(file protoreflect.Descriptor) (*GeneratorOptions, bool) {
	return extensions.Lookup[*GeneratorOptions](file, E_Generator)
}

func getCommandGroupOptions(svc protoreflect.Descriptor) (*CommandGroupOptions, bool) {
	return extensions.Lookup[*CommandGroupOptions](svc, E_CommandGroup)
}

func getCommandOptions(mtd protoreflect.Descriptor) (*CommandOptions, bool) {
	return extensions.Lookup[*CommandOptions](mtd, E_Command)
}

func getFlagOptions(fld protoreflect.Descriptor) (*FlagOptions, bool) {
	return extensions.Lookup[*FlagOptions](fld, E_Flag)
}

func getFlagSetOptions(fld protoreflect.Descriptor) (*FlagSetOptions, bool) {
	return extensions.Lookup[*FlagSetOptions](fld, E_FlagSet)
}

func applyOptions(desc protoreflect.Descriptor, out proto.Message) {
	var opts proto.Message
	var ok bool
	switch out.(type) {
	case *GeneratorOptions:
		opts, ok = getGeneratorOptions(desc)
	case *CommandGroupOptions:
		opts, ok = getCommandGroupOptions(desc)
	case *CommandOptions:
		opts, ok = getCommandOptions(desc)
	case *FlagOptions:
		opts, ok = getFlagOptions(desc)
	case *FlagSetOptions:
		opts, ok = getFlagSetOptions(desc)
	}
	if ok {
		proto.Merge(out, opts)
	}
}
