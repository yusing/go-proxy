package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/santhosh-tekuri/jsonschema"
	E "github.com/yusing/go-proxy/internal/error"
	"gopkg.in/yaml.v3"
)

func ValidateYaml(schema *jsonschema.Schema, data []byte) E.NestedError {
	var i any

	err := yaml.Unmarshal(data, &i)
	if err != nil {
		return E.FailWith("unmarshal yaml", err)
	}

	m, err := json.Marshal(i)
	if err != nil {
		return E.FailWith("marshal json", err)
	}

	err = schema.Validate(bytes.NewReader(m))
	if err == nil {
		return nil
	}

	errors := E.NewBuilder("yaml validation error")
	for _, e := range err.(*jsonschema.ValidationError).Causes {
		errors.AddE(e)
	}
	return errors.Build()
}

// Serialize converts the given data into a map[string]any representation.
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
// - result: The resulting map[string]any representation of the data.
// - error: An error if the data type is unsupported or if there is an error during conversion.
func Serialize(data any) (SerializedObject, E.NestedError) {
	result := make(map[string]any)

	// Use reflection to inspect the data type
	value := reflect.ValueOf(data)

	// Check if the value is valid
	if !value.IsValid() {
		return nil, E.Invalid("data", fmt.Sprintf("type: %T", data))
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
				if err.HasError() {
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
		return nil, E.Unsupported("type", value.Kind())
	}

	return result, nil
}

func Deserialize(src SerializedObject, target any) E.NestedError {
	if src == nil || target == nil {
		return nil
	}
	// convert data fields to lower no-snake
	// convert target fields to lower no-snake
	// then check if the field of data is in the target
	mapping := make(map[string]string)
	t := reflect.TypeOf(target).Elem()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		snakeCaseField := ToLowerNoSnake(field.Name)
		mapping[snakeCaseField] = field.Name
	}
	tValue := reflect.ValueOf(target)
	if tValue.IsZero() {
		return E.Invalid("value", "nil")
	}
	for k, v := range src {
		kCleaned := ToLowerNoSnake(k)
		if fieldName, ok := mapping[kCleaned]; ok {
			prop := reflect.ValueOf(target).Elem().FieldByName(fieldName)
			propType := prop.Type()
			isPtr := prop.Kind() == reflect.Ptr
			if prop.CanSet() {
				val := reflect.ValueOf(v)
				vType := val.Type()
				switch {
				case isPtr && vType.ConvertibleTo(propType.Elem()):
					ptr := reflect.New(propType.Elem())
					ptr.Elem().Set(val.Convert(propType.Elem()))
					prop.Set(ptr)
				case vType.ConvertibleTo(propType):
					prop.Set(val.Convert(propType))
				case isPtr:
					var vSerialized SerializedObject
					vSerialized, ok = v.(SerializedObject)
					if !ok {
						if vType.ConvertibleTo(reflect.TypeFor[SerializedObject]()) {
							vSerialized = val.Convert(reflect.TypeFor[SerializedObject]()).Interface().(SerializedObject)
						} else {
							return E.Failure(fmt.Sprintf("convert %s (%T) to %s", k, v, reflect.TypeFor[SerializedObject]()))
						}
					}
					propNew := reflect.New(propType.Elem())
					err := Deserialize(vSerialized, propNew.Interface())
					if err.HasError() {
						return E.Failure("set field").With(err).Subject(k)
					}
					prop.Set(propNew)
				default:
					return E.Invalid("conversion", k).Extraf("from %s to %s", vType, propType)
				}
			} else {
				return E.Unsupported("field", k).Extraf("type %s is not settable", propType)
			}
		} else {
			return E.Unexpected("field", k)
		}
	}

	return nil
}

func DeserializeJson(j map[string]string, target any) E.NestedError {
	data, err := E.Check(json.Marshal(j))
	if err.HasError() {
		return err
	}
	return E.From(json.Unmarshal(data, target))
}

func ToLowerNoSnake(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", ""))
}

type SerializedObject = map[string]any
