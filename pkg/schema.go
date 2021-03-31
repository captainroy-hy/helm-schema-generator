package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/encoding/openapi"
	"cuelang.org/go/encoding/yaml"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pkg/errors"
)

// GenerateSchemaFromValues generate OpenAPIv3 schema based on Chart Values
// file.
func GenerateSchemaFromValues(values []byte) ([]byte, error) {
	r := cue.Runtime{}
	// convert Values yaml to CUE
	ins, err := yaml.Decode(&r, "", string(values))
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode Values.yaml to CUE")
	}
	// get the streamed CUE including the comments which will be used as
	// 'description' in the schema
	c, err := format.Node(ins.Value().Syntax(cue.Docs(true)), format.Simplify())
	if err != nil {
		return nil, errors.Wrap(err, "cannot format CUE generated from Values.yaml")
	}

	valuesIdentifier := "values"
	// cue openapi encoder only works on top-level identifier, we have to add
	// an identifier manually
	valuesStr := fmt.Sprintf("#%s:{\n%s\n}", valuesIdentifier, string(c))

	r = cue.Runtime{}
	ins, err = r.Compile("-", valuesStr)
	if err != nil {
		return nil, errors.Wrap(err, "cannot compile CUE generated from Values.yaml")
	}
	if ins.Err != nil {
		return nil, errors.Wrap(ins.Err, "cannot compile CUE generated from Values.yaml")
	}
	// generate OpenAPIv3 schema through cue openapi encoder
	rawSchema, err := openapi.Gen(ins, &openapi.Config{})
	if err != nil {
		return nil, errors.Wrap(ins.Err, "cannot generate OpenAPIv3 schema")
	}
	rawSchema, err = makeSwaggerCompatible(rawSchema)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot make CUE-generated schema compatible with Swagger")
	}

	var out = &bytes.Buffer{}
	_ = json.Indent(out, rawSchema, "", "   ")
	// load schema into Swagger to validate it compatible with Swagger OpenAPIv3
	fullSchemaBySwagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(out.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "cannot load schema by SwaggerLoader")
	}
	valuesSchema := fullSchemaBySwagger.Components.Schemas[valuesIdentifier].Value
	changeEnumToDefault(valuesSchema)

	b, err := valuesSchema.MarshalJSON()
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshall Values schema")
	}
	_ = json.Indent(out, b, "", "   ")
	return out.Bytes(), nil
}

// cue openapi encoder converts default in Chart Values as enum in schema
// changing enum to default makes the schema consistent with Chart Values
func changeEnumToDefault(schema *openapi3.Schema) {
	t := schema.Type
	switch t {
	case "object":
		for _, v := range schema.Properties {
			s := v.Value
			changeEnumToDefault(s)
		}
	case "array":
		if schema.Items != nil {
			changeEnumToDefault(schema.Items.Value)
		}
	}
	// change enum to default
	if len(schema.Enum) > 0 {
		schema.Default = schema.Enum[0]
		schema.Enum = nil
	}
	// remove all required fields, because fields in Values.yml are all optional
	schema.Required = nil
}

// cue openapi encoder converts 'items' field in an array type field into array,
// that's not compatible with OpenAPIv3. 'items' field should be an object.
func makeSwaggerCompatible(d []byte) ([]byte, error) {
	m := map[string]interface{}{}
	err := json.Unmarshal(d, &m)
	if err != nil {
		return nil, errors.Wrap(err, "cannot unmarshall schema")
	}
	handleItemsOfArrayType(m)
	b, err := json.Marshal(m)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshall schema")
	}
	return b, nil
}

// handleItemsOfArrayType will convert all 'items' of array type from array to object
// and remove enum in the items
func handleItemsOfArrayType(t map[string]interface{}) {
	for _, v := range t {
		if next, ok := v.(map[string]interface{}); ok {
			handleItemsOfArrayType(next)
		}
	}
	if t["type"] == "array" {
		if i, ok := t["items"].([]interface{}); ok {
			if len(i) > 0 {
				if itemSpec, ok := i[0].(map[string]interface{}); ok {
					handleItemsOfArrayType(itemSpec)
					itemSpec["enum"] = nil
					t["items"] = itemSpec
				}
			}
		}
	}
}
