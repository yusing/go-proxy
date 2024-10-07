package idlewatcher

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

type templateData struct {
	CheckRedirectHeader string
	Title               string
	Message             string
}

//go:embed html/loading_page.html
var loadingPage []byte
var loadingPageTmpl = template.Must(template.New("loading_page").Parse(string(loadingPage)))

const headerCheckRedirect = "X-GoProxy-Check-Redirect"

func (w *watcher) makeRespBody(format string, args ...any) []byte {
	msg := fmt.Sprintf(format, args...)

	data := new(templateData)
	data.CheckRedirectHeader = headerCheckRedirect
	data.Title = w.ContainerName
	data.Message = strings.ReplaceAll(msg, "\n", "<br>")
	data.Message = strings.ReplaceAll(data.Message, " ", "&ensp;")

	buf := bytes.NewBuffer(make([]byte, 128)) // more than enough
	err := loadingPageTmpl.Execute(buf, data)
	if err != nil { // should never happen in production
		panic(err)
	}
	return buf.Bytes()
}
