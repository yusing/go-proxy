package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/santhosh-tekuri/jsonschema"
	E "github.com/yusing/go-proxy/error"
	"gopkg.in/yaml.v3"
)

func ValidateYaml(schema *jsonschema.Schema, data []byte) E.NestedError {
	var i interface{}

	err := yaml.Unmarshal(data, &i)
	if err != nil {
		return E.Failure("unmarshal yaml").With(err)
	}

	m, err := json.Marshal(i)
	if err != nil {
		return E.Failure("marshal json").With(err)
	}

	err = schema.Validate(bytes.NewReader(m))
	if err == nil {
		return E.Nil()
	}

	errors := E.NewBuilder("yaml validation error")
	for _, e := range err.(*jsonschema.ValidationError).Causes {
		errors.Add(e)
	}
	return errors.Build()
}

// TryJsonStringify converts the given object to a JSON string.
//
// It takes an object of any type and attempts to marshal it into a JSON string.
// If the marshaling is successful, the JSON string is returned.
// If the marshaling fails, the object is converted to a string using fmt.Sprint and returned.
//
// Parameters:
// - o: The object to be converted to a JSON string.
//
// Return type:
// - string: The JSON string representation of the object.
func TryJsonStringify(o any) string {
	b, err := json.Marshal(o)
	if err != nil {
		return fmt.Sprint(o)
	}
	return string(b)
}

// Serialize converts the given data into a map[string]interface{} representation.
//
// It uses reflection to inspect the data type and handle different kinds of data.
// For a struct, it extracts the fields using the json tag if present, or the field name if not.
// For an embedded struct, it recursively converts its fields into the result map.
// For any other type, it returns an error.
//
// Parameters:
// - data: The data to be converted into a map.
//
// Returns:
// - result: The resulting map[string]interface{} representation of the data.
// - error: An error if the data type is unsupported or if there is an error during conversion.
func Serialize(data interface{}) (SerializedObject, error) {
	result := make(map[string]any)

	// Use reflection to inspect the data type
	value := reflect.ValueOf(data)

	// Check if the value is valid
	if !value.IsValid() {
		return nil, fmt.Errorf("invalid data")
	}

	// Dereference pointers if necessary
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	// Handle different kinds of data
	switch value.Kind() {
	case reflect.Map:
		for _, key := range value.MapKeys() {
			result[key.String()] = value.MapIndex(key).Interface()
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			jsonTag := field.Tag.Get("json") // Get the json tag
			if jsonTag == "-" {
				continue // Ignore this field if the tag is "-"
			}

			// If the json tag is not empty, use it as the key
			if jsonTag != "" {
				result[jsonTag] = value.Field(i).Interface()
			} else if field.Anonymous {
				// If the field is an embedded struct, add its fields to the result
				fieldMap, err := Serialize(value.Field(i).Interface())
				if err != nil {
					return nil, err
				}
				for k, v := range fieldMap {
					result[k] = v
				}
			} else {
				result[field.Name] = value.Field(i).Interface()
			}
		}
	default:
		return nil, fmt.Errorf("unsupported type: %s", value.Kind())
	}

	return result, nil
}

type SerializedObject map[string]any
