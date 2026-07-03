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
// description      Defines Schema, the main entry to every subSchema.
//                  Contains the parsing logic and error checking.
//
// created          26-02-2013

package gojsonschema

import (
	"errors"
	"math/big"
	"reflect"
	"regexp"
	"text/template"

	"github.com/xeipuuv/gojsonreference"
)

var (
	// Locale is the default locale to use
	// Library users can overwrite with their own implementation
	Locale locale = DefaultLocale{}

	// ErrorTemplateFuncs allows you to define custom template funcs for use in localization.
	ErrorTemplateFuncs template.FuncMap
)

// NewSchema instances a schema using the given JSONLoader
func NewSchema(l JSONLoader) (*Schema, error) {
	return NewSchemaLoader().Compile(l)
}

// Schema holds a schema
type Schema struct {
	documentReference gojsonreference.JsonReference
	rootSchema        *subSchema
	pool              *schemaPool
	referencePool     *schemaReferencePool
}

func (d *Schema) parse(document interface{}, draft Draft) error {
	d.rootSchema = &subSchema{property: STRING_ROOT_SCHEMA_PROPERTY, draft: &draft}
	return d.parseSchema(document, d.rootSchema)
}

// SetRootSchemaName sets the root-schema name
func (d *Schema) SetRootSchemaName(name string) {
	d.rootSchema.property = name
}

// Parses a subSchema
//
// Pretty long function ( sorry :) )... but pretty straight forward, repetitive and boring
// Not much magic involved here, most of the job is to validate the key names and their values,
// then the values are copied into subSchema struct
//
func (d *Schema) parseSchema(documentNode interface{}, currentSchema *subSchema) error {

	if currentSchema.draft == nil {
		if currentSchema.parent == nil {
			return errors.New("Draft not set")
		}
		currentSchema.draft = currentSchema.parent.draft
	}

	if !isKind(documentNode, reflect.Map) {
		return errors.New(formatErrorDescription(
			Locale.ParseError(),
			ErrorDetails{
				"expected": STRING_SCHEMA,
			},
		))
	}

	m := documentNode.(map[string]interface{})

	if currentSchema.parent == nil {
		currentSchema.ref = &d.documentReference
		currentSchema.id = &d.documentReference
	}

	if currentSchema.id == nil && currentSchema.parent != nil {
		currentSchema.id = currentSchema.parent.id
	}

	keyID := KEY_ID
	if existsMapKey(m, keyID) && !isKind(m[keyID], reflect.String) {
		return errors.New(formatErrorDescription(
			Locale.InvalidType(),
			ErrorDetails{
				"expected": TYPE_STRING,
				"given":    keyID,
			},
		))
	}
	if k, ok := m[keyID].(string); ok {
		jsonReference, err := gojsonreference.NewJsonReference(k)
		if err != nil {
			return err
		}
		if currentSchema == d.rootSchema {
			currentSchema.id = &jsonReference
		} else {
			ref, err := currentSchema.parent.id.Inherits(jsonReference)
			if err != nil {
				return err
			}
			currentSchema.id = ref
		}
	}

	// definitions
	if existsMapKey(m, KEY_DEFINITIONS) {
		if isKind(m[KEY_DEFINITIONS], reflect.Map) {
			for _, dv := range m[KEY_DEFINITIONS].(map[string]interface{}) {
				if isKind(dv, reflect.Map) {

					newSchema := &subSchema{property: KEY_DEFINITIONS, parent: currentSchema}

					err := d.parseSchema(dv, newSchema)

					if err != nil {
						return err
					}
				} else {
					return errors.New(formatErrorDescription(
						Locale.InvalidType(),
						ErrorDetails{
							"expected": STRING_ARRAY_OF_SCHEMAS,
							"given":    KEY_DEFINITIONS,
						},
					))
				}
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.InvalidType(),
				ErrorDetails{
					"expected": STRING_ARRAY_OF_SCHEMAS,
					"given":    KEY_DEFINITIONS,
				},
			))
		}

	}

	// title
	if existsMapKey(m, KEY_TITLE) && !isKind(m[KEY_TITLE], reflect.String) {
		return errors.New(formatErrorDescription(
			Locale.InvalidType(),
			ErrorDetails{
				"expected": TYPE_STRING,
				"given":    KEY_TITLE,
			},
		))
	}
	if k, ok := m[KEY_TITLE].(string); ok {
		currentSchema.title = &k
	}

	// description
	if existsMapKey(m, KEY_DESCRIPTION) && !isKind(m[KEY_DESCRIPTION], reflect.String) {
		return errors.New(formatErrorDescription(
			Locale.InvalidType(),
			ErrorDetails{
				"expected": TYPE_STRING,
				"given":    KEY_DESCRIPTION,
			},
		))
	}
	if k, ok := m[KEY_DESCRIPTION].(string); ok {
		currentSchema.description = &k
	}

	// $ref
	if existsMapKey(m, KEY_REF) && !isKind(m[KEY_REF], reflect.String) {
		return errors.New(formatErrorDescription(
			Locale.InvalidType(),
			ErrorDetails{
				"expected": TYPE_STRING,
				"given":    KEY_REF,
			},
		))
	}

	if k, ok := m[KEY_REF].(string); ok {

		jsonReference, err := gojsonreference.NewJsonReference(k)
		if err != nil {
			return err
		}

		currentSchema.ref = &jsonReference

		if sch, ok := d.referencePool.Get(currentSchema.ref.String()); ok {
			currentSchema.refSchema = sch
		} else {
			err := d.parseReference(documentNode, currentSchema)

			if err != nil {
				return err
			}

			return nil
		}
	}

	// type
	if existsMapKey(m, KEY_TYPE) {
		if isKind(m[KEY_TYPE], reflect.String) {
			if k, ok := m[KEY_TYPE].(string); ok {
				err := currentSchema.types.Add(k)
				if err != nil {
					return err
				}
			}
		} else {
			if isKind(m[KEY_TYPE], reflect.Slice) {
				arrayOfTypes := m[KEY_TYPE].([]interface{})
				for _, typeInArray := range arrayOfTypes {
					if reflect.ValueOf(typeInArray).Kind() != reflect.String {
						return errors.New(formatErrorDescription(
							Locale.InvalidType(),
							ErrorDetails{
								"expected": TYPE_STRING + "/" + STRING_ARRAY_OF_STRINGS,
								"given":    KEY_TYPE,
							},
						))
					}
					if err := currentSchema.types.Add(typeInArray.(string)); err != nil {
						return err
					}
				}

			} else {
				return errors.New(formatErrorDescription(
					Locale.InvalidType(),
					ErrorDetails{
						"expected": TYPE_STRING + "/" + STRING_ARRAY_OF_STRINGS,
						"given":    KEY_TYPE,
					},
				))
			}
		}
	}

	// properties
	if existsMapKey(m, KEY_PROPERTIES) {
		err := d.parseProperties(m[KEY_PROPERTIES], currentSchema)
		if err != nil {
			return err
		}
	}

	// additionalProperties
	if existsMapKey(m, KEY_ADDITIONAL_PROPERTIES) {
		if isKind(m[KEY_ADDITIONAL_PROPERTIES], reflect.Bool) {
			currentSchema.additionalProperties = m[KEY_ADDITIONAL_PROPERTIES].(bool)
		} else if isKind(m[KEY_ADDITIONAL_PROPERTIES], reflect.Map) {
			newSchema := &subSchema{property: KEY_ADDITIONAL_PROPERTIES, parent: currentSchema, ref: currentSchema.ref}
			currentSchema.additionalProperties = newSchema
			err := d.parseSchema(m[KEY_ADDITIONAL_PROPERTIES], newSchema)
			if err != nil {
				return errors.New(err.Error())
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.InvalidType(),
				ErrorDetails{
					"expected": TYPE_BOOLEAN + "/" + STRING_SCHEMA,
					"given":    KEY_ADDITIONAL_PROPERTIES,
				},
			))
		}
	}

	// patternProperties
	if existsMapKey(m, KEY_PATTERN_PROPERTIES) {
		if isKind(m[KEY_PATTERN_PROPERTIES], reflect.Map) {
			patternPropertiesMap := m[KEY_PATTERN_PROPERTIES].(map[string]interface{})
			if len(patternPropertiesMap) > 0 {
				currentSchema.patternProperties = make(map[string]*subSchema)
				for k, v := range patternPropertiesMap {
					_, err := regexp.MatchString(k, "")
					if err != nil {
						return errors.New(formatErrorDescription(
							Locale.RegexPattern(),
							ErrorDetails{"pattern": k},
						))
					}
					newSchema := &subSchema{property: k, parent: currentSchema, ref: currentSchema.ref}
					err = d.parseSchema(v, newSchema)
					if err != nil {
						return errors.New(err.Error())
					}
					currentSchema.patternProperties[k] = newSchema
				}
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.InvalidType(),
				ErrorDetails{
					"expected": STRING_SCHEMA,
					"given":    KEY_PATTERN_PROPERTIES,
				},
			))
		}
	}

	// dependencies
	if existsMapKey(m, KEY_DEPENDENCIES) {
		err := d.parseDependencies(m[KEY_DEPENDENCIES], currentSchema)
		if err != nil {
			return err
		}
	}

	// items
	if existsMapKey(m, KEY_ITEMS) {
		if isKind(m[KEY_ITEMS], reflect.Slice) {
			for _, itemElement := range m[KEY_ITEMS].([]interface{}) {
				if isKind(itemElement, reflect.Map) {
					newSchema := &subSchema{parent: currentSchema, property: KEY_ITEMS}
					newSchema.ref = currentSchema.ref
					currentSchema.itemsChildren = append(currentSchema.itemsChildren, newSchema)
					err := d.parseSchema(itemElement, newSchema)
					if err != nil {
						return err
					}
				} else {
					return errors.New(formatErrorDescription(
						Locale.InvalidType(),
						ErrorDetails{
							"expected": STRING_SCHEMA + "/" + STRING_ARRAY_OF_SCHEMAS,
							"given":    KEY_ITEMS,
						},
					))
				}
				currentSchema.itemsChildrenIsSingleSchema = false
			}
		} else if isKind(m[KEY_ITEMS], reflect.Map) {
			newSchema := &subSchema{parent: currentSchema, property: KEY_ITEMS}
			newSchema.ref = currentSchema.ref
			currentSchema.itemsChildren = append(currentSchema.itemsChildren, newSchema)
			err := d.parseSchema(m[KEY_ITEMS], newSchema)
			if err != nil {
				return err
			}
			currentSchema.itemsChildrenIsSingleSchema = true
		} else {
			return errors.New(formatErrorDescription(
				Locale.InvalidType(),
				ErrorDetails{
					"expected": STRING_SCHEMA + "/" + STRING_ARRAY_OF_SCHEMAS,
					"given":    KEY_ITEMS,
				},
			))
		}
	}

	// additionalItems
	if existsMapKey(m, KEY_ADDITIONAL_ITEMS) {
		if isKind(m[KEY_ADDITIONAL_ITEMS], reflect.Bool) {
			currentSchema.additionalItems = m[KEY_ADDITIONAL_ITEMS].(bool)
		} else if isKind(m[KEY_ADDITIONAL_ITEMS], reflect.Map) {
			newSchema := &subSchema{property: KEY_ADDITIONAL_ITEMS, parent: currentSchema, ref: currentSchema.ref}
			currentSchema.additionalItems = newSchema
			err := d.parseSchema(m[KEY_ADDITIONAL_ITEMS], newSchema)
			if err != nil {
				return errors.New(err.Error())
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.InvalidType(),
				ErrorDetails{
					"expected": TYPE_BOOLEAN + "/" + STRING_SCHEMA,
					"given":    KEY_ADDITIONAL_ITEMS,
				},
			))
		}
	}

	// validation : number / integer

	if existsMapKey(m, KEY_MULTIPLE_OF) {
		multipleOfValue := mustBeNumber(m[KEY_MULTIPLE_OF])
		if multipleOfValue == nil {
			return errors.New(formatErrorDescription(
				Locale.InvalidType(),
				ErrorDetails{
					"expected": STRING_NUMBER,
					"given":    KEY_MULTIPLE_OF,
				},
			))
		}
		if multipleOfValue.Cmp(big.NewRat(0, 1)) <= 0 {
			return errors.New(formatErrorDescription(
				Locale.GreaterThanZero(),
				ErrorDetails{"number": KEY_MULTIPLE_OF},
			))
		}
		currentSchema.multipleOf = multipleOfValue
	}

	if existsMapKey(m, KEY_MINIMUM) {
		minimumValue := mustBeNumber(m[KEY_MINIMUM])
		if minimumValue == nil {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfA(),
				ErrorDetails{"x": KEY_MINIMUM, "y": STRING_NUMBER},
			))
		}
		currentSchema.minimum = minimumValue
	}

	if existsMapKey(m, KEY_EXCLUSIVE_MINIMUM) {
		exclusiveMinimum, minimum, err := parseExclusiveBound(m[KEY_EXCLUSIVE_MINIMUM], KEY_EXCLUSIVE_MINIMUM, KEY_MINIMUM, currentSchema.minimum)
		if err != nil {
			return err
		}
		currentSchema.exclusiveMinimum = exclusiveMinimum
		currentSchema.minimum = minimum
	}

	if existsMapKey(m, KEY_MAXIMUM) {
		maximumValue := mustBeNumber(m[KEY_MAXIMUM])
		if maximumValue == nil {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfA(),
				ErrorDetails{"x": KEY_MAXIMUM, "y": STRING_NUMBER},
			))
		}
		currentSchema.maximum = maximumValue
	}

	if existsMapKey(m, KEY_EXCLUSIVE_MAXIMUM) {
		exclusiveMaximum, maximum, err := parseExclusiveBound(m[KEY_EXCLUSIVE_MAXIMUM], KEY_EXCLUSIVE_MAXIMUM, KEY_MAXIMUM, currentSchema.maximum)
		if err != nil {
			return err
		}
		currentSchema.exclusiveMaximum = exclusiveMaximum
		currentSchema.maximum = maximum
	}

	// validation : string

	if existsMapKey(m, KEY_MIN_LENGTH) {
		minLengthIntegerValue := mustBeInteger(m[KEY_MIN_LENGTH])
		if minLengthIntegerValue == nil {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_MIN_LENGTH, "y": TYPE_INTEGER},
			))
		}
		if *minLengthIntegerValue < 0 {
			return errors.New(formatErrorDescription(
				Locale.MustBeGTEZero(),
				ErrorDetails{"key": KEY_MIN_LENGTH},
			))
		}
		currentSchema.minLength = minLengthIntegerValue
	}

	if existsMapKey(m, KEY_MAX_LENGTH) {
		maxLengthIntegerValue := mustBeInteger(m[KEY_MAX_LENGTH])
		if maxLengthIntegerValue == nil {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_MAX_LENGTH, "y": TYPE_INTEGER},
			))
		}
		if *maxLengthIntegerValue < 0 {
			return errors.New(formatErrorDescription(
				Locale.MustBeGTEZero(),
				ErrorDetails{"key": KEY_MAX_LENGTH},
			))
		}
		currentSchema.maxLength = maxLengthIntegerValue
	}

	if currentSchema.minLength != nil && currentSchema.maxLength != nil {
		if *currentSchema.minLength > *currentSchema.maxLength {
			return errors.New(formatErrorDescription(
				Locale.CannotBeGT(),
				ErrorDetails{"x": KEY_MIN_LENGTH, "y": KEY_MAX_LENGTH},
			))
		}
	}

	if existsMapKey(m, KEY_PATTERN) {
		if isKind(m[KEY_PATTERN], reflect.String) {
			regexpObject, err := regexp.Compile(m[KEY_PATTERN].(string))
			if err != nil {
				return errors.New(formatErrorDescription(
					Locale.MustBeValidRegex(),
					ErrorDetails{"key": KEY_PATTERN},
				))
			}
			currentSchema.pattern = regexpObject
		} else {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfA(),
				ErrorDetails{"x": KEY_PATTERN, "y": TYPE_STRING},
			))
		}
	}

	if existsMapKey(m, KEY_FORMAT) {
		formatString, ok := m[KEY_FORMAT].(string)
		if !ok {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfType(),
				ErrorDetails{"key": KEY_FORMAT, "type": TYPE_STRING},
			))
		}
		currentSchema.format = formatString
	}

	// validation : object

	if existsMapKey(m, KEY_MIN_PROPERTIES) {
		minPropertiesIntegerValue := mustBeInteger(m[KEY_MIN_PROPERTIES])
		if minPropertiesIntegerValue == nil {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_MIN_PROPERTIES, "y": TYPE_INTEGER},
			))
		}
		if *minPropertiesIntegerValue < 0 {
			return errors.New(formatErrorDescription(
				Locale.MustBeGTEZero(),
				ErrorDetails{"key": KEY_MIN_PROPERTIES},
			))
		}
		currentSchema.minProperties = minPropertiesIntegerValue
	}

	if existsMapKey(m, KEY_MAX_PROPERTIES) {
		maxPropertiesIntegerValue := mustBeInteger(m[KEY_MAX_PROPERTIES])
		if maxPropertiesIntegerValue == nil {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_MAX_PROPERTIES, "y": TYPE_INTEGER},
			))
		}
		if *maxPropertiesIntegerValue < 0 {
			return errors.New(formatErrorDescription(
				Locale.MustBeGTEZero(),
				ErrorDetails{"key": KEY_MAX_PROPERTIES},
			))
		}
		currentSchema.maxProperties = maxPropertiesIntegerValue
	}

	if currentSchema.minProperties != nil && currentSchema.maxProperties != nil {
		if *currentSchema.minProperties > *currentSchema.maxProperties {
			return errors.New(formatErrorDescription(
				Locale.KeyCannotBeGreaterThan(),
				ErrorDetails{"key": KEY_MIN_PROPERTIES, "y": KEY_MAX_PROPERTIES},
			))
		}
	}

	if existsMapKey(m, KEY_REQUIRED) {
		if isKind(m[KEY_REQUIRED], reflect.Slice) {
			requiredValues := m[KEY_REQUIRED].([]interface{})
			for _, requiredValue := range requiredValues {
				if isKind(requiredValue, reflect.String) {
					if isStringInSlice(currentSchema.required, requiredValue.(string)) {
						return errors.New(formatErrorDescription(
							Locale.KeyItemsMustBeUnique(),
							ErrorDetails{"key": KEY_REQUIRED},
						))
					}
					currentSchema.required = append(currentSchema.required, requiredValue.(string))
				} else {
					return errors.New(formatErrorDescription(
						Locale.KeyItemsMustBeOfType(),
						ErrorDetails{"key": KEY_REQUIRED, "type": TYPE_STRING},
					))
				}
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_REQUIRED, "y": TYPE_ARRAY},
			))
		}
	}

	// validation : array

	if existsMapKey(m, KEY_MIN_ITEMS) {
		minItemsIntegerValue := mustBeInteger(m[KEY_MIN_ITEMS])
		if minItemsIntegerValue == nil {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_MIN_ITEMS, "y": TYPE_INTEGER},
			))
		}
		if *minItemsIntegerValue < 0 {
			return errors.New(formatErrorDescription(
				Locale.MustBeGTEZero(),
				ErrorDetails{"key": KEY_MIN_ITEMS},
			))
		}
		currentSchema.minItems = minItemsIntegerValue
	}

	if existsMapKey(m, KEY_MAX_ITEMS) {
		maxItemsIntegerValue := mustBeInteger(m[KEY_MAX_ITEMS])
		if maxItemsIntegerValue == nil {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_MAX_ITEMS, "y": TYPE_INTEGER},
			))
		}
		if *maxItemsIntegerValue < 0 {
			return errors.New(formatErrorDescription(
				Locale.MustBeGTEZero(),
				ErrorDetails{"key": KEY_MAX_ITEMS},
			))
		}
		currentSchema.maxItems = maxItemsIntegerValue
	}

	if existsMapKey(m, KEY_UNIQUE_ITEMS) {
		if isKind(m[KEY_UNIQUE_ITEMS], reflect.Bool) {
			currentSchema.uniqueItems = m[KEY_UNIQUE_ITEMS].(bool)
		} else {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfA(),
				ErrorDetails{"x": KEY_UNIQUE_ITEMS, "y": TYPE_BOOLEAN},
			))
		}
	}

	// validation : all

	if existsMapKey(m, KEY_ENUM) {
		if isKind(m[KEY_ENUM], reflect.Slice) {
			for _, v := range m[KEY_ENUM].([]interface{}) {
				is, err := marshalWithoutNumber(v)
				if err != nil {
					return err
				}
				if isStringInSlice(currentSchema.enum, *is) {
					return errors.New(formatErrorDescription(
						Locale.KeyItemsMustBeUnique(),
						ErrorDetails{"key": KEY_ENUM},
					))
				}
				currentSchema.enum = append(currentSchema.enum, *is)
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_ENUM, "y": TYPE_ARRAY},
			))
		}
	}

	// validation : subSchema

	if existsMapKey(m, KEY_ONE_OF) {
		if isKind(m[KEY_ONE_OF], reflect.Slice) {
			for _, v := range m[KEY_ONE_OF].([]interface{}) {
				newSchema := &subSchema{property: KEY_ONE_OF, parent: currentSchema, ref: currentSchema.ref}
				currentSchema.oneOf = append(currentSchema.oneOf, newSchema)
				err := d.parseSchema(v, newSchema)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_ONE_OF, "y": TYPE_ARRAY},
			))
		}
	}

	if existsMapKey(m, KEY_ANY_OF) {
		if isKind(m[KEY_ANY_OF], reflect.Slice) {
			for _, v := range m[KEY_ANY_OF].([]interface{}) {
				newSchema := &subSchema{property: KEY_ANY_OF, parent: currentSchema, ref: currentSchema.ref}
				currentSchema.anyOf = append(currentSchema.anyOf, newSchema)
				err := d.parseSchema(v, newSchema)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_ANY_OF, "y": TYPE_ARRAY},
			))
		}
	}

	if existsMapKey(m, KEY_ALL_OF) {
		if isKind(m[KEY_ALL_OF], reflect.Slice) {
			for _, v := range m[KEY_ALL_OF].([]interface{}) {
				newSchema := &subSchema{property: KEY_ALL_OF, parent: currentSchema, ref: currentSchema.ref}
				currentSchema.allOf = append(currentSchema.allOf, newSchema)
				err := d.parseSchema(v, newSchema)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_ANY_OF, "y": TYPE_ARRAY},
			))
		}
	}

	if existsMapKey(m, KEY_NOT) {
		if isKind(m[KEY_NOT], reflect.Map) {
			newSchema := &subSchema{property: KEY_NOT, parent: currentSchema, ref: currentSchema.ref}
			currentSchema.not = newSchema
			err := d.parseSchema(m[KEY_NOT], newSchema)
			if err != nil {
				return err
			}
		} else {
			return errors.New(formatErrorDescription(
				Locale.MustBeOfAn(),
				ErrorDetails{"x": KEY_NOT, "y": TYPE_OBJECT},
			))
		}
	}

	return nil
}

// parseExclusiveBound parses a Draft4-style boolean exclusiveMinimum/exclusiveMaximum value.
// When set to true, it moves the paired bound (minimum/maximum) into the exclusive bound
// and clears the inclusive one; otherwise the inclusive bound is left untouched.
func parseExclusiveBound(value interface{}, exclusiveKey, boundKey string, bound *big.Rat) (exclusiveBound *big.Rat, newBound *big.Rat, err error) {
	if !isKind(value, reflect.Bool) {
		return nil, bound, errors.New(formatErrorDescription(
			Locale.InvalidType(),
			ErrorDetails{
				"expected": TYPE_BOOLEAN,
				"given":    exclusiveKey,
			},
		))
	}
	if bound == nil {
		return nil, nil, errors.New(formatErrorDescription(
			Locale.CannotBeUsedWithout(),
			ErrorDetails{"x": exclusiveKey, "y": boundKey},
		))
	}
	if value.(bool) {
		return bound, nil, nil
	}
	return nil, bound, nil
}

func (d *Schema) parseReference(documentNode interface{}, currentSchema *subSchema) error {
	var (
		refdDocumentNode interface{}
		dsp              *schemaPoolDocument
		err              error
	)

	newSchema := &subSchema{property: KEY_REF, parent: currentSchema, ref: currentSchema.ref}

	d.referencePool.Add(currentSchema.ref.String(), newSchema)

	dsp, err = d.pool.GetDocument(*currentSchema.ref)
	if err != nil {
		return err
	}
	newSchema.id = currentSchema.ref

	refdDocumentNode = dsp.Document
	newSchema.draft = dsp.Draft

	if err != nil {
		return err
	}

	if !isKind(refdDocumentNode, reflect.Map) {
		return errors.New(formatErrorDescription(
			Locale.MustBeOfType(),
			ErrorDetails{"key": STRING_SCHEMA, "type": TYPE_OBJECT},
		))
	}

	err = d.parseSchema(refdDocumentNode, newSchema)
	if err != nil {
		return err
	}

	currentSchema.refSchema = newSchema

	return nil

}

func (d *Schema) parseProperties(documentNode interface{}, currentSchema *subSchema) error {

	if !isKind(documentNode, reflect.Map) {
		return errors.New(formatErrorDescription(
			Locale.MustBeOfType(),
			ErrorDetails{"key": STRING_PROPERTIES, "type": TYPE_OBJECT},
		))
	}

	m := documentNode.(map[string]interface{})
	for k := range m {
		schemaProperty := k
		newSchema := &subSchema{property: schemaProperty, parent: currentSchema, ref: currentSchema.ref}
		currentSchema.propertiesChildren = append(currentSchema.propertiesChildren, newSchema)
		err := d.parseSchema(m[k], newSchema)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Schema) parseDependencies(documentNode interface{}, currentSchema *subSchema) error {

	if !isKind(documentNode, reflect.Map) {
		return errors.New(formatErrorDescription(
			Locale.MustBeOfType(),
			ErrorDetails{"key": KEY_DEPENDENCIES, "type": TYPE_OBJECT},
		))
	}

	m := documentNode.(map[string]interface{})
	currentSchema.dependencies = make(map[string]interface{})

	for k := range m {
		switch reflect.ValueOf(m[k]).Kind() {

		case reflect.Slice:
			values := m[k].([]interface{})
			var valuesToRegister []string

			for _, value := range values {
				if !isKind(value, reflect.String) {
					return errors.New(formatErrorDescription(
						Locale.MustBeOfType(),
						ErrorDetails{
							"key":  STRING_DEPENDENCY,
							"type": STRING_SCHEMA_OR_ARRAY_OF_STRINGS,
						},
					))
				}
				valuesToRegister = append(valuesToRegister, value.(string))
				currentSchema.dependencies[k] = valuesToRegister
			}

		case reflect.Map:
			depSchema := &subSchema{property: k, parent: currentSchema, ref: currentSchema.ref}
			err := d.parseSchema(m[k], depSchema)
			if err != nil {
				return err
			}
			currentSchema.dependencies[k] = depSchema

		default:
			return errors.New(formatErrorDescription(
				Locale.MustBeOfType(),
				ErrorDetails{
					"key":  STRING_DEPENDENCY,
					"type": STRING_SCHEMA_OR_ARRAY_OF_STRINGS,
				},
			))
		}

	}

	return nil
}
