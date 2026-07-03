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
	"bytes"
	"encoding/json"
	"io"
)

// JSONLoader defines the JSON loader interface
type JSONLoader interface {
	JsonSource() interface{}
	LoadJSON() (interface{}, error)
}

// JSON bytes loader
type jsonBytesLoader struct {
	source []byte
}

func (l *jsonBytesLoader) JsonSource() interface{} {
	return l.source
}

// NewBytesLoader creates a new JSONLoader, taking a `[]byte` as source
func NewBytesLoader(source []byte) JSONLoader {
	return &jsonBytesLoader{source: source}
}

func (l *jsonBytesLoader) LoadJSON() (interface{}, error) {
	return decodeJSONUsingNumber(bytes.NewReader(l.JsonSource().([]byte)))
}

// JSON Raw (types) loader
type jsonRawLoader struct {
	source interface{}
}

func (l *jsonRawLoader) JsonSource() interface{} {
	return l.source
}

// NewRawLoader creates a new JSONLoader from a given pre-unmarshaled Go object
func NewRawLoader(source interface{}) JSONLoader {
	return &jsonRawLoader{source: source}
}

func (l *jsonRawLoader) LoadJSON() (interface{}, error) {
	return l.source, nil
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
