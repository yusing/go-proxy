package notif

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func formatMarkdown(extras map[string]interface{}) string {
	msg := bytes.NewBufferString("")
	for k, v := range extras {
		msg.WriteString("#### ")
		msg.WriteString(k)
		msg.WriteRune('\n')
		msg.WriteString(fmt.Sprintf("%v", v))
		msg.WriteRune('\n')
	}
	return msg.String()
}

func formatDiscord(extras map[string]interface{}) (string, error) {
	fieldsMap := make([]map[string]any, len(extras))
	i := 0
	for k, extra := range extras {
		fieldsMap[i] = map[string]any{
			"name":  k,
			"value": extra,
		}
		i++
	}
	fields, err := json.Marshal(fieldsMap)
	if err != nil {
		return "", err
	}
	return string(fields), nil
}
