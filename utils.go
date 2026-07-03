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
// description      Various utility functions.
//
// created          26-02-2013

package gojsonschema

import (
	"encoding/json"
	"math/big"
	"reflect"
)

func isKind(what interface{}, kinds ...reflect.Kind) bool {
	target := what
	if isJSONNumber(what) {
		// JSON Numbers are strings!
		target = *mustBeNumber(what)
	}
	targetKind := reflect.ValueOf(target).Kind()
	for _, kind := range kinds {
		if targetKind == kind {
			return true
		}
	}
	return false
}

func existsMapKey(m map[string]interface{}, k string) bool {
	_, ok := m[k]
	return ok
}

func isStringInSlice(s []string, what string) bool {
	for i := range s {
		if s[i] == what {
			return true
		}
	}
	return false
}

// indexStringInSlice returns the index of the first instance of 'what' in s or -1 if it is not found in s.
func indexStringInSlice(s []string, what string) int {
	for i := range s {
		if s[i] == what {
			return i
		}
	}
	return -1
}

func marshalToJSONString(value interface{}) (*string, error) {

	mBytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	sBytes := string(mBytes)
	return &sBytes, nil
}

func marshalWithoutNumber(value interface{}) (*string, error) {

	// The JSON is decoded using https://golang.org/pkg/encoding/json/#Decoder.UseNumber
	// This means the numbers are internally still represented as strings and therefore 1.00 is unequal to 1
	// One way to eliminate these differences is to decode and encode the JSON one more time without Decoder.UseNumber
	// so that these differences in representation are removed

	jsonString, err := marshalToJSONString(value)
	if err != nil {
		return nil, err
	}

	var document interface{}

	err = json.Unmarshal([]byte(*jsonString), &document)
	if err != nil {
		return nil, err
	}

	return marshalToJSONString(document)
}

func isJSONNumber(what interface{}) bool {

	switch what.(type) {

	case json.Number:
		return true
	}

	return false
}

func checkJSONInteger(what interface{}) (isInt bool) {
	switch v := what.(type) {
	case json.Number:
		bigFloat, isValidNumber := new(big.Rat).SetString(string(v))
		return isValidNumber && bigFloat.IsInt()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return v == float64(int64(v))
	case float32:
		return v == float32(int32(v))
	}
	return false
}

// same as ECMA Number.MAX_SAFE_INTEGER and Number.MIN_SAFE_INTEGER
const (
	maxJSONFloat = float64(1<<53 - 1)  // 9007199254740991.0 	 2^53 - 1
	minJSONFloat = -float64(1<<53 - 1) //-9007199254740991.0	-2^53 - 1
)

func mustBeInteger(what interface{}) *int {
	switch v := what.(type) {
	case json.Number:
		if checkJSONInteger(v) {
			int64Value, err := v.Int64()
			if err == nil {
				int32Value := int(int64Value)
				return &int32Value
			}
		}
	case int:
		val := int(v)
		return &val
	case int8:
		val := int(v)
		return &val
	case int16:
		val := int(v)
		return &val
	case int32:
		val := int(v)
		return &val
	case int64:
		val := int(v)
		return &val
	case uint:
		val := int(v)
		return &val
	case uint8:
		val := int(v)
		return &val
	case uint16:
		val := int(v)
		return &val
	case uint32:
		val := int(v)
		return &val
	case uint64:
		val := int(v)
		return &val
	case float64:
		if v == float64(int64(v)) {
			val := int(v)
			return &val
		}
	case float32:
		if v == float32(int32(v)) {
			val := int(v)
			return &val
		}
	}
	return nil
}

func mustBeNumber(what interface{}) *big.Rat {
	switch v := what.(type) {
	case json.Number:
		float64Value, success := new(big.Rat).SetString(string(v))
		if success {
			return float64Value
		}
	case int:
		return new(big.Rat).SetInt64(int64(v))
	case int8:
		return new(big.Rat).SetInt64(int64(v))
	case int16:
		return new(big.Rat).SetInt64(int64(v))
	case int32:
		return new(big.Rat).SetInt64(int64(v))
	case int64:
		return new(big.Rat).SetInt64(int64(v))
	case uint:
		return new(big.Rat).SetUint64(uint64(v))
	case uint8:
		return new(big.Rat).SetUint64(uint64(v))
	case uint16:
		return new(big.Rat).SetUint64(uint64(v))
	case uint32:
		return new(big.Rat).SetUint64(uint64(v))
	case uint64:
		return new(big.Rat).SetUint64(uint64(v))
	case float64:
		return new(big.Rat).SetFloat64(v)
	case float32:
		return new(big.Rat).SetFloat64(float64(v))
	}
	return nil
}

func convertDocumentNode(val interface{}) interface{} {

	if lval, ok := val.([]interface{}); ok {

		res := []interface{}{}
		for _, v := range lval {
			res = append(res, convertDocumentNode(v))
		}

		return res

	}

	if mval, ok := val.(map[interface{}]interface{}); ok {

		res := map[string]interface{}{}

		for k, v := range mval {
			res[k.(string)] = convertDocumentNode(v)
		}

		return res

	}

	return val
}
