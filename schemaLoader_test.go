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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaDetection(t *testing.T) {
	loader := NewStringLoader(`{
		"$schema" : "http://json-schema.org/draft-04/schema#",
		"exclusiveMinimum" : 5
	}`)

	// The schema should produce an error in draft-04 mode
	_, err := NewSchema(loader)
	assert.NotNil(t, err)
}

const not_map_interface = "not map interface"

func TestParseSchemaURL_NotMap(t *testing.T) {
	//GIVEN
	sl := NewGoLoader(not_map_interface)
	//WHEN
	_, err := NewSchema(sl)
	//THEN
	require.Error(t, err)
	assert.EqualError(t, err, "schema is invalid")
}
