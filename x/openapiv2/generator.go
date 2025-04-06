package openapiv2

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/kralicky/codegen/pkg/extensions"
	"github.com/kralicky/grpc-gateway/v2/pkg/descriptor"
	"github.com/kralicky/grpc-gateway/v2/protoc-gen-openapiv2/pkg/genopenapi"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"
)

var Generator = generator{}

type generator struct{}

func (generator) Name() string {
	return "openapiv2"
}

func (g generator) Generate(plugin *protogen.Plugin) error {
	toGenerate := map[string]struct{}{}
	for path, file := range plugin.FilesByPath {
		if !file.Generate {
			continue
		}
		if !proto.HasExtension(file.Desc.Options(), E_Generator) {
			continue
		}
		ext, ok := extensions.Lookup[*GeneratorOptions](file.Desc, E_Generator)
		if !ok || !ext.GetEnabled() {
			continue
		}
		toGenerate[path] = struct{}{}
	}

	reg := descriptor.NewRegistry()
	reg.SetUseAllOfForRefs(true)

	reg.LoadFromPlugin(plugin)

	gen := genopenapi.New(reg, genopenapi.FormatJSON)
	targets := make([]*descriptor.File, 0, len(plugin.Request.FileToGenerate))
	for _, target := range plugin.Request.FileToGenerate {
		if _, ok := toGenerate[target]; !ok {
			continue
		}
		f, err := reg.LookupFile(target)
		if err != nil {
			return err
		}
		targets = append(targets, f)
	}
	out, err := gen.Generate(targets)
	if err != nil {
		return err
	}
	for _, f := range out {
		content := f.GetContent()
		var swagger openapi2.T
		if err := json.Unmarshal([]byte(content), &swagger); err != nil {
			return err
		}
		oapiv3, err := openapi2conv.ToV3(&swagger)
		if err != nil {
			return err
		}
		v3 := struct {
			OpenAPI      string                        `json:"openapi" yaml:"openapi"` // Required
			Info         *openapi3.Info                `json:"info" yaml:"info"`       // Required
			Servers      openapi3.Servers              `json:"servers,omitempty" yaml:"servers,omitempty"`
			Security     openapi3.SecurityRequirements `json:"security,omitempty" yaml:"security,omitempty"`
			Tags         openapi3.Tags                 `json:"tags,omitempty" yaml:"tags,omitempty"`
			Paths        *openapi3.Paths               `json:"paths" yaml:"paths"` // Required
			Components   *openapi3.Components          `json:"components,omitempty" yaml:"components,omitempty"`
			ExternalDocs *openapi3.ExternalDocs        `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
		}{
			OpenAPI:      oapiv3.OpenAPI,
			Info:         oapiv3.Info,
			Servers:      oapiv3.Servers,
			Security:     oapiv3.Security,
			Tags:         oapiv3.Tags,
			Paths:        oapiv3.Paths,
			Components:   oapiv3.Components,
			ExternalDocs: oapiv3.ExternalDocs,
		}
		v3Yaml, err := yaml.Marshal(v3)
		if err != nil {
			return err
		}
		dir, file := filepath.Split(f.GetName())
		filename := filepath.Join(dir, strings.Replace(file, "swagger.json", "openapi.yaml", 1))
		plugin.NewGeneratedFile(filename, protogen.GoImportPath(f.GoPkg.String())).Write(v3Yaml)
	}
	return nil
}
