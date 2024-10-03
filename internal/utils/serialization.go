package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema"
	E "github.com/yusing/go-proxy/internal/error"
	"gopkg.in/yaml.v3"
)

type SerializedObject = map[string]any
type Converter interface {
	ConvertFrom(value any) (any, E.NestedError)
}

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
		return nil, E.Unsupported("type", value.Kind())
	}

	return result, nil
}

// Deserialize takes a SerializedObject and a target value, and assigns the values in the SerializedObject to the target value.
// Deserialize ignores case differences between the field names in the SerializedObject and the target.
//
// The target value must be a struct or a map[string]any.
// If the target value is a struct, the SerializedObject will be deserialized into the struct fields.
// If the target value is a map[string]any, the SerializedObject will be deserialized into the map.
//
// The function returns an error if the target value is not a struct or a map[string]any, or if there is an error during deserialization.
func Deserialize(src SerializedObject, dst any) E.NestedError {
	if src == nil {
		return E.Invalid("src", "nil")
	}

	if dst == nil {
		return E.Invalid("nil dst", fmt.Sprintf("type: %T", dst))
	}

	dstV := reflect.ValueOf(dst)
	dstT := dstV.Type()

	if dstV.Kind() == reflect.Ptr {
		if dstV.IsNil() {
			return E.Invalid("nil dst", fmt.Sprintf("type: %T", dst))
		}
		dstV = dstV.Elem()
		dstT = dstV.Type()
	}

	// convert data fields to lower no-snake
	// convert target fields to lower no-snake
	// then check if the field of data is in the target

	// TODO: use E.Builder to collect errors from all fields

	if dstV.Kind() == reflect.Struct {
		mapping := make(map[string]reflect.Value)
		for i := 0; i < dstV.NumField(); i++ {
			field := dstT.Field(i)
			mapping[ToLowerNoSnake(field.Name)] = dstV.Field(i)
		}
		for k, v := range src {
			if field, ok := mapping[ToLowerNoSnake(k)]; ok {
				err := Convert(reflect.ValueOf(v), field)
				if err != nil {
					return err.Subject(k)
				}
			} else {
				return E.Unexpected("field", k).Subjectf("%T", dst)
			}
		}
	} else if dstV.Kind() == reflect.Map && dstT.Key().Kind() == reflect.String {
		if dstV.IsNil() {
			dstV.Set(reflect.MakeMap(dstT))
		}
		for k := range src {
			tmp := reflect.New(dstT.Elem()).Elem()
			err := Convert(reflect.ValueOf(src[k]), tmp)
			if err != nil {
				return err.Subject(k)
			}
			dstV.SetMapIndex(reflect.ValueOf(ToLowerNoSnake(k)), tmp)
		}
		return nil
	} else {
		return E.Unsupported("target type", fmt.Sprintf("%T", dst))
	}

	return nil
}

// Convert attempts to convert the src to dst.
//
// If src is a map, it is deserialized into dst.
// If src is a slice, each of its elements are converted and stored in dst.
// For any other type, it is converted using the reflect.Value.Convert function (if possible).
//
// If dst is not settable, an error is returned.
// If src cannot be converted to dst, an error is returned.
// If any error occurs during conversion (e.g. deserialization), it is returned.
//
// Returns:
//   - error: the error occurred during conversion, or nil if no error occurred.
func Convert(src reflect.Value, dst reflect.Value) E.NestedError {
	srcT := src.Type()
	dstT := dst.Type()

	if src.Kind() == reflect.Interface {
		src = src.Elem()
		srcT = src.Type()
	}

	if !dst.CanSet() {
		return E.From(fmt.Errorf("%w type %T is unsettable", E.ErrUnsupported, dst.Interface()))
	}

	if dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dstT.Elem()))
		}
		dst = dst.Elem()
		dstT = dst.Type()
	}

	switch {
	case srcT.AssignableTo(dstT):
		dst.Set(src)
		return nil
	case srcT.ConvertibleTo(dstT):
		dst.Set(src.Convert(dstT))
		return nil
	case srcT.Kind() == reflect.Map:
		obj, ok := src.Interface().(SerializedObject)
		if !ok {
			return E.TypeMismatch[SerializedObject](src.Interface())
		}
		return Deserialize(obj, dst.Addr().Interface())
	case srcT.Kind() == reflect.Slice:
		if dstT.Kind() != reflect.Slice {
			return E.TypeError("slice", srcT, dstT)
		}
		newSlice := reflect.MakeSlice(dstT, 0, src.Len())
		i := 0
		for _, v := range src.Seq2() {
			tmp := reflect.New(dstT.Elem()).Elem()
			err := Convert(v, tmp)
			if err != nil {
				return err.Subjectf("[%d]", i)
			}
			newSlice = reflect.Append(newSlice, tmp)
			i++
		}
		dst.Set(newSlice)
		return nil
	case src.Kind() == reflect.String:
		if convertible, err := ConvertString(src.String(), dst); convertible {
			return err
		}
	}

	var converter Converter
	var ok bool
	// check if (*T).Convertor is implemented
	if converter, ok = dst.Addr().Interface().(Converter); !ok {
		// check if (T).Convertor is implemented
		converter, ok = dst.Interface().(Converter)
		if !ok {
			return E.TypeError("conversion", srcT, dstT)
		}
	}

	converted, err := converter.ConvertFrom(src.Interface())
	if err != nil {
		return err
	}
	c := reflect.ValueOf(converted)
	if c.Kind() == reflect.Ptr {
		c = c.Elem()
	}
	dst.Set(c)
	return nil
}

func ConvertString(src string, dst reflect.Value) (convertible bool, convErr E.NestedError) {
	convertible = true
	// primitive types / simple types
	switch dst.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(src)
		if err != nil {
			convErr = E.Invalid("boolean", src)
			return
		}
		dst.Set(reflect.ValueOf(b))
		return
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(src, 10, 64)
		if err != nil {
			convErr = E.Invalid("int", src)
			return
		}
		dst.Set(reflect.ValueOf(i))
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(src, 10, 64)
		if err != nil {
			convErr = E.Invalid("uint", src)
			return
		}
		dst.Set(reflect.ValueOf(i))
		return
	}
	// yaml like
	lines := strings.Split(strings.TrimSpace(src), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	var tmp any
	switch dst.Kind() {
	case reflect.Slice:
		// one liner is comma seperated list
		if len(lines) == 0 {
			dst.Set(reflect.ValueOf(CommaSeperatedList(src)))
			return
		}
		sl := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimLeftFunc(line, func(r rune) bool {
				return r == '-' || unicode.IsSpace(r)
			})
			if line == "" {
				continue
			}
			sl = append(sl, line)
		}
		tmp = sl
	case reflect.Map:
		m := make(map[string]string, len(lines))
		for i, line := range lines {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				convErr = E.Invalid("map", "missing colon").Subjectf("line#%d", i+1).With(line)
				return
			}
			if len(parts) > 2 {
				convErr = E.Invalid("map", "too many colons").Subjectf("line#%d", i+1).With(line)
				return
			}
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			m[k] = v
		}
		tmp = m
	}
	if tmp == nil {
		convertible = false
		return
	}
	return true, Convert(reflect.ValueOf(tmp), dst)
}

func DeserializeJson(j map[string]string, target any) E.NestedError {
	data, err := E.Check(json.Marshal(j))
	if err != nil {
		return err
	}
	return E.From(json.Unmarshal(data, target))
}

func ToLowerNoSnake(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", ""))
}
