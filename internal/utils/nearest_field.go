package utils

import (
	"reflect"

	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func NearestField(input string, s any) string {
	minDistance := -1
	nearestField := ""
	var fields []string
	switch s := s.(type) {
	case []string:
		fields = s
	default:
		t := reflect.TypeOf(s)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct {
			fields = make([]string, 0)
			for i := 0; i < t.NumField(); i++ {
				jsonTag, ok := t.Field(i).Tag.Lookup("json")
				if ok {
					fields = append(fields, jsonTag)
				} else {
					fields = append(fields, t.Field(i).Name)
				}
			}
		} else if t.Kind() == reflect.Map {
			keys := reflect.ValueOf(s).MapKeys()
			fields = make([]string, len(keys))
			for i, key := range keys {
				fields[i] = key.String()
			}
		} else {
			panic("unsupported type: " + t.String())
		}
	}
	for _, field := range fields {
		distance := strutils.LevenshteinDistance(input, field)
		if minDistance == -1 || distance < minDistance {
			minDistance = distance
			nearestField = field
		}
	}
	return nearestField
}
