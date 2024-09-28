package http

import "net/http"

func IsSuccess(status int) bool {
	return status >= http.StatusOK && status < http.StatusMultipleChoices
}
