package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"gopkg.in/yaml.v3"
)

type (
	SerializedObject = map[string]any
	Converter        interface {
		ConvertFrom(value any) E.Error
	}
)

var (
	ErrInvalidType           = E.New("invalid type")
	ErrNilValue              = E.New("nil")
	ErrUnsettable            = E.New("unsettable")
	ErrUnsupportedConversion = E.New("unsupported conversion")
	ErrMapMissingColon       = E.New("map missing colon")
	ErrMapTooManyColons      = E.New("map too many colons")
	ErrUnknownField          = E.New("unknown field")
)

func ValidateYaml(schema *jsonschema.Schema, data []byte) E.Error {
	var i any

	err := yaml.Unmarshal(data, &i)
	if err != nil {
		return E.From(err)
	}

	m, err := json.Marshal(i)
	if err != nil {
		return E.From(err)
	}

	err = schema.Validate(bytes.NewReader(m))
	if err == nil {
		return nil
	}

	var valErr *jsonschema.ValidationError
	if !errors.As(err, &valErr) {
		panic(err)
	}

	b := E.NewBuilder("yaml validation error")
	for _, e := range valErr.Causes {
		b.Adds(e.Message)
	}
	return b.Error()
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
func Serialize(data any) (SerializedObject, error) {
	result := make(map[string]any)

	// Use reflection to inspect the data type
	value := reflect.ValueOf(data)

	// Check if the value is valid
	if !value.IsValid() {
		return nil, ErrInvalidType.Subjectf("%T", data)
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
		for i := range value.NumField() {
			field := value.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			jsonTag := field.Tag.Get("json") // Get the json tag
			if jsonTag == "-" {
				continue // Ignore this field if the tag is "-"
			}
			if strings.Contains(jsonTag, ",omitempty") {
				if value.Field(i).IsZero() {
					continue
				}
				jsonTag = strings.Replace(jsonTag, ",omitempty", "", 1)
			}

			// If the json tag is not empty, use it as the key
			switch {
			case jsonTag != "":
				result[jsonTag] = value.Field(i).Interface()
			case field.Anonymous:
				// If the field is an embedded struct, add its fields to the result
				fieldMap, err := Serialize(value.Field(i).Interface())
				if err != nil {
					return nil, err
				}
				for k, v := range fieldMap {
					result[k] = v
				}
			default:
				result[field.Name] = value.Field(i).Interface()
			}
		}
	default:
		return nil, errors.New("serialize: unsupported data type " + value.Kind().String())
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
func Deserialize(src SerializedObject, dst any) E.Error {
	if src == nil {
		return E.Errorf("deserialize: src is %w", ErrNilValue)
	}
	if dst == nil {
		return E.Errorf("deserialize: dst is %w", ErrNilValue)
	}

	dstV := reflect.ValueOf(dst)
	dstT := dstV.Type()

	if dstV.Kind() == reflect.Ptr {
		if dstV.IsNil() {
			return E.Errorf("deserialize: dst is %w", ErrNilValue)
		}
		dstV = dstV.Elem()
		dstT = dstV.Type()
	}

	// convert data fields to lower no-snake
	// convert target fields to lower no-snake
	// then check if the field of data is in the target

	errs := E.NewBuilder("deserialize error")

	switch dstV.Kind() {
	case reflect.Struct:
		mapping := make(map[string]reflect.Value)
		for _, field := range reflect.VisibleFields(dstT) {
			mapping[strutils.ToLowerNoSnake(field.Name)] = dstV.FieldByName(field.Name)
		}
		for k, v := range src {
			if field, ok := mapping[strutils.ToLowerNoSnake(k)]; ok {
				err := Convert(reflect.ValueOf(v), field)
				if err != nil {
					errs.Add(err.Subject(k))
				}
			} else {
				errs.Add(ErrUnknownField.Subject(k).Withf(strutils.DoYouMean(NearestField(k, dst))))
			}
		}
		return errs.Error()
	case reflect.Map:
		if dstV.IsNil() {
			dstV.Set(reflect.MakeMap(dstT))
		}
		for k := range src {
			tmp := reflect.New(dstT.Elem()).Elem()
			err := Convert(reflect.ValueOf(src[k]), tmp)
			if err != nil {
				errs.Add(err.Subject(k))
			}
			dstV.SetMapIndex(reflect.ValueOf(strutils.ToLowerNoSnake(k)), tmp)
		}
		return errs.Error()
	default:
		return ErrUnsupportedConversion.Subject("deserialize to " + dstT.String())
	}
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
func Convert(src reflect.Value, dst reflect.Value) E.Error {
	srcT := src.Type()
	dstT := dst.Type()

	if src.Kind() == reflect.Interface {
		src = src.Elem()
		srcT = src.Type()
	}

	if !dst.CanSet() {
		return ErrUnsettable.Subject(dstT.String())
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
			return ErrUnsupportedConversion.Subject(dstT.String() + " to " + srcT.String())
		}
		return Deserialize(obj, dst.Addr().Interface())
	case srcT.Kind() == reflect.Slice:
		if dstT.Kind() != reflect.Slice {
			return ErrUnsupportedConversion.Subject(dstT.String() + " to slice")
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
		return ErrUnsupportedConversion.Subjectf("%s to %s", srcT, dstT)
	}

	return converter.ConvertFrom(src.Interface())
}

func ConvertString(src string, dst reflect.Value) (convertible bool, convErr E.Error) {
	convertible = true
	if dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}
	if dst.Kind() == reflect.String {
		dst.SetString(src)
		return
	}
	switch dst.Type() {
	case reflect.TypeFor[time.Duration]():
		if src == "" {
			dst.Set(reflect.Zero(dst.Type()))
			return
		}
		d, err := time.ParseDuration(src)
		if err != nil {
			return true, E.From(err)
		}
		dst.Set(reflect.ValueOf(d))
		return
	default:
	}
	// primitive types / simple types
	switch dst.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(src)
		if err != nil {
			return true, E.From(err)
		}
		dst.Set(reflect.ValueOf(b))
		return
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(src, 10, 64)
		if err != nil {
			return true, E.From(err)
		}
		dst.Set(reflect.ValueOf(i).Convert(dst.Type()))
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(src, 10, 64)
		if err != nil {
			return true, E.From(err)
		}
		dst.Set(reflect.ValueOf(i).Convert(dst.Type()))
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
		// one liner is comma separated list
		if len(lines) == 0 {
			dst.Set(reflect.ValueOf(strutils.CommaSeperatedList(src)))
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
		errs := E.NewBuilder("invalid map")
		for i, line := range lines {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				errs.Add(ErrMapMissingColon.Subjectf("line %d", i+1))
			}
			if len(parts) > 2 {
				errs.Add(ErrMapTooManyColons.Subjectf("line %d", i+1))
			}
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			m[k] = v
		}
		if errs.HasError() {
			return true, errs.Error()
		}
		tmp = m
	}
	if tmp == nil {
		return false, nil
	}
	return true, Convert(reflect.ValueOf(tmp), dst)
}

func DeserializeJSON(j map[string]string, target any) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
