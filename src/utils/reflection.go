package utils

import (
	"net/http"
	"reflect"
	"strings"

	E "github.com/yusing/go-proxy/error"
)

func snakeToPascal(s string) string {
	toHyphenCamel := http.CanonicalHeaderKey(strings.ReplaceAll(s, "_", "-"))
	return strings.ReplaceAll(toHyphenCamel, "-", "")
}

func SetFieldFromSnake[T, VT any](obj *T, field string, value VT) E.NestedError {
	field = snakeToPascal(field)
	prop := reflect.ValueOf(obj).Elem().FieldByName(field)
	if prop.Kind() == 0 {
		return E.Invalid("field", field)
	}
	prop.Set(reflect.ValueOf(value))
	return E.Nil()
}
