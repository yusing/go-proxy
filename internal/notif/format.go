package notif

import (
	"bytes"
	"encoding/json"
)

func formatMarkdown(extras LogFields) string {
	msg := bytes.NewBufferString("")
	for _, field := range extras {
		msg.WriteString("#### ")
		msg.WriteString(field.Name)
		msg.WriteRune('\n')
		msg.WriteString(field.Value)
		msg.WriteRune('\n')
	}
	return msg.String()
}

func formatDiscord(extras LogFields) (string, error) {
	fields, err := json.Marshal(extras)
	if err != nil {
		return "", err
	}
	return string(fields), nil
}
