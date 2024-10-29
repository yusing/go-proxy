package docker

import (
	"reflect"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
	U "github.com/yusing/go-proxy/internal/utils"
)

/*
Formats:
  - namespace.attribute
  - namespace.target.attribute
  - namespace.target.attribute.namespace2.attribute
*/
type (
	Label struct {
		Namespace string
		Target    string
		Attribute string
		Value     any
	}
	NestedLabelMap map[string]U.SerializedObject
)

var (
	ErrApplyToNil    = E.New("label value is nil")
	ErrFieldNotExist = E.New("field does not exist")
)

func (l *Label) String() string {
	if l.Attribute == "" {
		return l.Namespace + "." + l.Target
	}
	return l.Namespace + "." + l.Target + "." + l.Attribute
}

// Apply applies the value of a Label to the corresponding field in the given object.
//
// Parameters:
//   - obj: a pointer to the object to which the Label value will be applied.
//   - l: a pointer to the Label containing the attribute and value to be applied.
//
// Returns:
//   - error: an error if the field does not exist.
func ApplyLabel[T any](obj *T, l *Label) E.Error {
	if obj == nil {
		return ErrApplyToNil.Subject(l.String())
	}
	switch nestedLabel := l.Value.(type) {
	case *Label:
		var field reflect.Value
		objType := reflect.TypeFor[T]()
		for i := range reflect.TypeFor[T]().NumField() {
			if objType.Field(i).Tag.Get("yaml") == l.Attribute {
				field = reflect.ValueOf(obj).Elem().Field(i)
				break
			}
		}
		if !field.IsValid() {
			return ErrFieldNotExist.Subject(l.Attribute).Subject(l.String())
		}
		dst, ok := field.Interface().(NestedLabelMap)
		if !ok {
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
			} else {
				field = field.Addr()
			}
			err := U.Deserialize(U.SerializedObject{nestedLabel.Namespace: nestedLabel.Value}, field.Interface())
			if err != nil {
				return err.Subject(l.String())
			}
			return nil
		}
		if dst == nil {
			field.Set(reflect.MakeMap(reflect.TypeFor[NestedLabelMap]()))
			dst = field.Interface().(NestedLabelMap)
		}
		if dst[nestedLabel.Namespace] == nil {
			dst[nestedLabel.Namespace] = make(U.SerializedObject)
		}
		dst[nestedLabel.Namespace][nestedLabel.Attribute] = nestedLabel.Value
		return nil
	default:
		err := U.Deserialize(U.SerializedObject{l.Attribute: l.Value}, obj)
		if err != nil {
			return err.Subject(l.String())
		}
		return nil
	}
}

func ParseLabel(label string, value string) *Label {
	parts := strings.Split(label, ".")

	if len(parts) < 2 {
		return &Label{
			Namespace: label,
			Value:     value,
		}
	}

	l := &Label{
		Namespace: parts[0],
		Target:    parts[1],
		Value:     value,
	}

	switch len(parts) {
	case 2:
		l.Attribute = l.Target
	case 3:
		l.Attribute = parts[2]
	default:
		l.Attribute = parts[2]
		nestedLabel := ParseLabel(strings.Join(parts[3:], "."), value)
		l.Value = nestedLabel
	}

	return l
}
