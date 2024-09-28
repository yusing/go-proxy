package idlewatcher

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
)

type templateData struct {
	Title          string
	Message        string
	RequestHeaders http.Header
	SpinnerClass   string
}

//go:embed html/loading_page.html
var loadingPage []byte
var loadingPageTmpl = template.Must(template.New("loading_page").Parse(string(loadingPage)))

const (
	htmlContentType = "text/html; charset=utf-8"

	errPrefix = "\u1000"

	headerGoProxyTargetURL = "X-GoProxy-Target"
	headerContentType      = "Content-Type"

	spinnerClassSpinner   = "spinner"
	spinnerClassErrorSign = "error"
)

func (w *watcher) makeSuccResp(redirectURL string, resp *http.Response) (*http.Response, error) {
	h := make(http.Header)
	h.Set("Location", redirectURL)
	h.Set("Content-Length", "0")
	h.Set(headerContentType, htmlContentType)
	return &http.Response{
		StatusCode: http.StatusTemporaryRedirect,
		Header:     h,
		Body:       http.NoBody,
		TLS:        resp.TLS,
	}, nil
}

func (w *watcher) makeErrResp(errFmt string, args ...any) (*http.Response, error) {
	return w.makeResp(errPrefix+errFmt, args...)
}

func (w *watcher) makeResp(format string, args ...any) (*http.Response, error) {
	msg := fmt.Sprintf(format, args...)

	data := new(templateData)
	data.Title = w.ContainerName
	data.Message = strings.ReplaceAll(msg, "\n", "<br>")
	data.Message = strings.ReplaceAll(data.Message, " ", "&ensp;")
	data.RequestHeaders = make(http.Header)
	data.RequestHeaders.Add(headerGoProxyTargetURL, "window.location.href")
	if strings.HasPrefix(data.Message, errPrefix) {
		data.Message = strings.TrimLeft(data.Message, errPrefix)
		data.SpinnerClass = spinnerClassErrorSign
	} else {
		data.SpinnerClass = spinnerClassSpinner
	}

	buf := bytes.NewBuffer(make([]byte, 128)) // more than enough
	err := loadingPageTmpl.Execute(buf, data)
	if err != nil { // should never happen
		panic(err)
	}
	return &http.Response{
		StatusCode: http.StatusAccepted,
		Header: http.Header{
			headerContentType: {htmlContentType},
			"Cache-Control": {
				"no-cache",
				"no-store",
				"must-revalidate",
			},
		},
		Body:          io.NopCloser(buf),
		ContentLength: int64(buf.Len()),
	}, nil
}
