// Copyright 2018 johandorland ( https://github.com/johandorland )
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gojsonschema

import (
	"errors"
	"reflect"

	"github.com/xeipuuv/gojsonreference"
)

// Draft is a JSON-schema draft version
type Draft int

// Supported Draft versions
const (
	Draft4 Draft = 4
)

const (
	draft4MetaSchemaURL = "http://json-schema.org/draft-04/schema"
	draft4MetaSchema    = `{"id":"http://json-schema.org/draft-04/schema#","$schema":"http://json-schema.org/draft-04/schema#","description":"Core schema meta-schema","definitions":{"schemaArray":{"type":"array","minItems":1,"items":{"$ref":"#"}},"positiveInteger":{"type":"integer","minimum":0},"positiveIntegerDefault0":{"allOf":[{"$ref":"#/definitions/positiveInteger"},{"default":0}]},"simpleTypes":{"enum":["array","boolean","integer","null","number","object","string"]},"stringArray":{"type":"array","items":{"type":"string"},"minItems":1,"uniqueItems":true}},"type":"object","properties":{"id":{"type":"string"},"$schema":{"type":"string"},"title":{"type":"string"},"description":{"type":"string"},"default":{},"multipleOf":{"type":"number","minimum":0,"exclusiveMinimum":true},"maximum":{"type":"number"},"exclusiveMaximum":{"type":"boolean","default":false},"minimum":{"type":"number"},"exclusiveMinimum":{"type":"boolean","default":false},"maxLength":{"$ref":"#/definitions/positiveInteger"},"minLength":{"$ref":"#/definitions/positiveIntegerDefault0"},"pattern":{"type":"string","format":"regex"},"additionalItems":{"anyOf":[{"type":"boolean"},{"$ref":"#"}],"default":{}},"items":{"anyOf":[{"$ref":"#"},{"$ref":"#/definitions/schemaArray"}],"default":{}},"maxItems":{"$ref":"#/definitions/positiveInteger"},"minItems":{"$ref":"#/definitions/positiveIntegerDefault0"},"uniqueItems":{"type":"boolean","default":false},"maxProperties":{"$ref":"#/definitions/positiveInteger"},"minProperties":{"$ref":"#/definitions/positiveIntegerDefault0"},"required":{"$ref":"#/definitions/stringArray"},"additionalProperties":{"anyOf":[{"type":"boolean"},{"$ref":"#"}],"default":{}},"definitions":{"type":"object","additionalProperties":{"$ref":"#"},"default":{}},"properties":{"type":"object","additionalProperties":{"$ref":"#"},"default":{}},"patternProperties":{"type":"object","additionalProperties":{"$ref":"#"},"default":{}},"dependencies":{"type":"object","additionalProperties":{"anyOf":[{"$ref":"#"},{"$ref":"#/definitions/stringArray"}]}},"enum":{"type":"array","minItems":1,"uniqueItems":true},"type":{"anyOf":[{"$ref":"#/definitions/simpleTypes"},{"type":"array","items":{"$ref":"#/definitions/simpleTypes"},"minItems":1,"uniqueItems":true}]},"format":{"type":"string"},"allOf":{"$ref":"#/definitions/schemaArray"},"anyOf":{"$ref":"#/definitions/schemaArray"},"oneOf":{"$ref":"#/definitions/schemaArray"},"not":{"$ref":"#"}},"dependencies":{"exclusiveMaximum":["maximum"],"exclusiveMinimum":["minimum"]},"default":{}}`
)

// getMetaSchema returns the meta-schema for the given meta-schema URL, or "" if unknown.
func getMetaSchema(url string) string {
	if url == draft4MetaSchemaURL {
		return draft4MetaSchema
	}
	return ""
}

// getDraftVersion returns the Draft matching the given meta-schema URL, or nil if unknown.
func getDraftVersion(url string) *Draft {
	if url == draft4MetaSchemaURL {
		draft := Draft4
		return &draft
	}
	return nil
}

// getSchemaURL returns the meta-schema URL for the given Draft, or "" if unknown.
func getSchemaURL(draft Draft) string {
	if draft == Draft4 {
		return draft4MetaSchemaURL
	}
	return ""
}

func parseSchemaURL(documentNode interface{}) (string, *Draft, error) {

	if !isKind(documentNode, reflect.Map) {
		return "", nil, errors.New("schema is invalid")
	}

	m := documentNode.(map[string]interface{})

	if existsMapKey(m, KEY_SCHEMA) {
		if !isKind(m[KEY_SCHEMA], reflect.String) {
			return "", nil, errors.New(formatErrorDescription(
				Locale.MustBeOfType(),
				ErrorDetails{
					"key":  KEY_SCHEMA,
					"type": TYPE_STRING,
				},
			))
		}

		schemaReference, err := gojsonreference.NewJsonReference(m[KEY_SCHEMA].(string))

		if err != nil {
			return "", nil, err
		}

		schema := schemaReference.String()

		return schema, getDraftVersion(schema), nil
	}

	return "", nil, nil
}
