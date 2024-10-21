package notif

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	U "github.com/yusing/go-proxy/internal/utils"
)

func FieldsAsTitle(entry *logrus.Entry) string {
	if len(entry.Data) == 0 {
		return ""
	}
	var parts []string
	for k, v := range entry.Data {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	parts[0] = U.Title(parts[0])
	return strings.Join(parts, ", ")
}
