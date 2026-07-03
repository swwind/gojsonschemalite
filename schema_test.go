// Copyright 2015 xeipuuv ( https://github.com/xeipuuv )
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

// author           xeipuuv
// author-github    https://github.com/xeipuuv
// author-mail      xeipuuv@gmail.com
//
// repository-name  gojsonschema
// repository-desc  An implementation of JSON Schema, based on IETF's draft v4 - Go language.

package gojsonschema

import (
	"testing"
)

func TestAdditionalPropertiesErrorMessage(t *testing.T) {
	schema := `{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "Device": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      }
    }
  }
}`
	text := `{
		"Device":{
			"Color" : true
		}
	}`
	loader := NewBytesLoader([]byte(schema))
	result, err := Validate(loader, NewBytesLoader([]byte(text)))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Errors()) != 1 {
		t.Fatal("Expected 1 error but got", len(result.Errors()))
	}

	expected := "Device.Color: Invalid type. Expected: string, given: boolean"
	actual := result.Errors()[0].String()
	if actual != expected {
		t.Fatalf("Expected '%s' but got '%s'", expected, actual)
	}
}

func TestInternalRefAndCircularRef(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"person": {
				"$ref": "#/definitions/person"
			}
		},
		"definitions": {
			"person": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"spouse": {"$ref": "#/definitions/person"}
				},
				"required": ["name"]
			}
		}
	}`

	s, err := NewSchema(NewBytesLoader([]byte(schema)))
	if err != nil {
		t.Fatal(err)
	}

	// Valid case
	validData := `{
		"person": {
			"name": "Alice",
			"spouse": {
				"name": "Bob"
			}
		}
	}`
	res, err := s.Validate(NewBytesLoader([]byte(validData)))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Valid() {
		t.Fatalf("Expected valid, got errors: %v", res.Errors())
	}

	// Invalid case
	invalidData := `{
		"person": {
			"name": "Alice",
			"spouse": {
				"spouse": {}
			}
		}
	}`
	res, err = s.Validate(NewBytesLoader([]byte(invalidData)))
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid() {
		t.Fatal("Expected invalid")
	}
}

func TestIDBasedRef(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"person": {
				"$ref": "#personRef"
			}
		},
		"definitions": {
			"person": {
				"id": "#personRef",
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name"]
			}
		}
	}`

	s, err := NewSchema(NewBytesLoader([]byte(schema)))
	if err != nil {
		t.Fatal(err)
	}

	// Valid case
	validData := `{
		"person": {
			"name": "Alice"
		}
	}`
	res, err := s.Validate(NewBytesLoader([]byte(validData)))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Valid() {
		t.Fatalf("Expected valid, got errors: %v", res.Errors())
	}

	// Invalid case
	invalidData := `{
		"person": {
			"name": 123
		}
	}`
	res, err = s.Validate(NewBytesLoader([]byte(invalidData)))
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid() {
		t.Fatal("Expected invalid")
	}
}
