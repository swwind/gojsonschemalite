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
	"bytes"
	"errors"
	"reflect"
)

// SchemaLoader is used to load schemas
type SchemaLoader struct {
	AutoDetect bool
	Validate   bool
	Draft      Draft
}

// NewSchemaLoader creates a new NewSchemaLoader
func NewSchemaLoader() *SchemaLoader {
	return &SchemaLoader{
		AutoDetect: true,
		Validate:   false,
		Draft:      Draft4,
	}
}

func (sl *SchemaLoader) validateMetaschema(documentNode interface{}) error {
	var (
		schema string
		err    error
	)
	if sl.AutoDetect {
		schema, _, err = parseSchemaURL(documentNode)
		if err != nil {
			return err
		}
	}

	// If no explicit "$schema" is used, use the default metaschema associated with the draft used
	if schema == "" {
		schema = drafts.GetSchemaURL(sl.Draft)
	}

	metaSchemaJSON := drafts.GetMetaSchema(schema)
	if metaSchemaJSON == "" {
		return errors.New("Unsupported metaschema: " + schema)
	}

	//Disable validation when loading the metaschema to prevent an infinite recursive loop
	sl.Validate = false

	metaSchema, err := sl.Compile(NewBytesLoader([]byte(metaSchemaJSON)))
	if err != nil {
		return err
	}

	sl.Validate = true

	result := metaSchema.validateDocument(documentNode)
	if !result.Valid() {
		var res bytes.Buffer
		for _, err := range result.Errors() {
			res.WriteString(err.String())
			res.WriteString("\n")
		}
		return errors.New(res.String())
	}

	return nil
}

// Compile loads and compiles a schema
func (sl *SchemaLoader) Compile(rootSchema JSONLoader) (*Schema, error) {
	d := Schema{}

	doc, err := rootSchema.LoadJSON()
	if err != nil {
		return nil, err
	}

	// If the loaded schema is an array (slice), wrap it into {"type": "array", "items": doc} as ESLint does
	if isKind(doc, reflect.Slice) {
		doc = map[string]interface{}{
			"type":  "array",
			"items": doc,
		}
	}

	if sl.Validate {
		if err := sl.validateMetaschema(doc); err != nil {
			return nil, err
		}
	}

	draft := sl.Draft
	if sl.AutoDetect {
		_, detectedDraft, err := parseSchemaURL(doc)
		if err != nil {
			return nil, err
		}
		if detectedDraft != nil {
			draft = *detectedDraft
		}
	}

	err = d.parse(doc, draft)
	if err != nil {
		return nil, err
	}

	return &d, nil
}
