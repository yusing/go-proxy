package docker

import (
	"strings"

	E "github.com/yusing/go-proxy/error"
	U "github.com/yusing/go-proxy/utils"
)

type Label struct {
	Namespace string
	Target    string
	Attribute string
	Value     any
}

// Apply applies the value of a Label to the corresponding field in the given object.
//
// Parameters:
//   - obj: a pointer to the object to which the Label value will be applied.
//   - l: a pointer to the Label containing the attribute and value to be applied.
//
// Returns:
//   - error: an error if the field does not exist.
func ApplyLabel[T any](obj *T, l *Label) E.NestedError {
	return U.SetFieldFromSnake(obj, l.Attribute, l.Value)
}

type ValueParser func(string) (any, E.NestedError)
type ValueParserMap map[string]ValueParser

func ParseLabel(label string, value string) (*Label, E.NestedError) {
	parts := strings.Split(label, ".")

	if len(parts) < 2 {
		return &Label{
			Namespace: label,
			Value:     value,
		}, E.Nil()
	}

	l := &Label{
		Namespace: parts[0],
		Target:    parts[1],
		Value:     value,
	}

	if len(parts) == 3 {
		l.Attribute = parts[2]
	} else {
		l.Attribute = l.Target
	}

	// find if namespace has value parser
	pm, ok := labelValueParserMap[l.Namespace]
	if !ok {
		return l, E.Nil()
	}
	// find if attribute has value parser
	p, ok := pm[l.Attribute]
	if !ok {
		return l, E.Nil()
	}
	// try to parse value
	v, err := p(value)
	if err.IsNotNil() {
		return nil, err
	}
	l.Value = v
	return l, E.Nil()
}

func RegisterNamespace(namespace string, pm ValueParserMap) {
	labelValueParserMap[namespace] = pm
}

// namespace:target.attribute -> func(string) (any, error)
var labelValueParserMap = make(map[string]ValueParserMap)
