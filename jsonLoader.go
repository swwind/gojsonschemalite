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
//
// description		Strategies to load JSON: raw bytes or an already-unmarshaled Go value.
//					No file system or network access is performed.
//
// created          01-02-2015

package gojsonschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/xeipuuv/gojsonreference"
)

// JSONLoader defines the JSON loader interface
type JSONLoader interface {
	JsonSource() interface{}
	LoadJSON() (interface{}, error)
	JsonReference() (gojsonreference.JsonReference, error)
	LoaderFactory() JSONLoaderFactory
}

// JSONLoaderFactory defines the JSON loader factory interface
type JSONLoaderFactory interface {
	// New creates a new JSON loader for the given source
	New(source string) JSONLoader
}

// defaultJSONLoaderFactory creates loaders that resolve canonical schema URLs
// against the embedded meta-schema cache (see draft.go). It performs no file
// system or network I/O.
type defaultJSONLoaderFactory struct{}

func (defaultJSONLoaderFactory) New(source string) JSONLoader {
	return &metaSchemaLoader{url: source}
}

// metaSchemaLoader resolves a canonical meta-schema URL (e.g. the draft-04
// schema URL) using the embedded meta-schema cache.
type metaSchemaLoader struct {
	url string
}

func (l *metaSchemaLoader) JsonSource() interface{} {
	return l.url
}

func (l *metaSchemaLoader) JsonReference() (gojsonreference.JsonReference, error) {
	return gojsonreference.NewJsonReference(l.url)
}

func (l *metaSchemaLoader) LoaderFactory() JSONLoaderFactory {
	return defaultJSONLoaderFactory{}
}

func (l *metaSchemaLoader) LoadJSON() (interface{}, error) {
	if metaSchema := getMetaSchema(l.url); metaSchema != "" {
		return decodeJSONUsingNumber(strings.NewReader(metaSchema))
	}

	return nil, errors.New(formatErrorDescription(Locale.RemoteNotSupported(), ErrorDetails{"reference": l.url}))
}

// JSON bytes loader

type jsonBytesLoader struct {
	source []byte
}

func (l *jsonBytesLoader) JsonSource() interface{} {
	return l.source
}

func (l *jsonBytesLoader) JsonReference() (gojsonreference.JsonReference, error) {
	return gojsonreference.NewJsonReference("#")
}

func (l *jsonBytesLoader) LoaderFactory() JSONLoaderFactory {
	return defaultJSONLoaderFactory{}
}

// NewBytesLoader creates a new JSONLoader, taking a `[]byte` as source
func NewBytesLoader(source []byte) JSONLoader {
	return &jsonBytesLoader{source: source}
}

func (l *jsonBytesLoader) LoadJSON() (interface{}, error) {
	return decodeJSONUsingNumber(bytes.NewReader(l.JsonSource().([]byte)))
}

// JSON raw loader
// Used when the JSON is already unmarshaled into interface{}, e.g. maps, slices, structs.

type jsonRawLoader struct {
	source interface{}
}

// NewRawLoader creates a new JSON raw loader for the given already-unmarshaled source
func NewRawLoader(source interface{}) JSONLoader {
	return &jsonRawLoader{source: source}
}
func (l *jsonRawLoader) JsonSource() interface{} {
	return l.source
}
func (l *jsonRawLoader) LoadJSON() (interface{}, error) {
	return l.source, nil
}
func (l *jsonRawLoader) JsonReference() (gojsonreference.JsonReference, error) {
	return gojsonreference.NewJsonReference("#")
}
func (l *jsonRawLoader) LoaderFactory() JSONLoaderFactory {
	return defaultJSONLoaderFactory{}
}

func decodeJSONUsingNumber(r io.Reader) (interface{}, error) {

	var document interface{}

	decoder := json.NewDecoder(r)
	decoder.UseNumber()

	err := decoder.Decode(&document)
	if err != nil {
		return nil, err
	}

	return document, nil

}
