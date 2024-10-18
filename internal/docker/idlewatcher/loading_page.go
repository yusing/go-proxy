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

const headerCheckRedirect = "X-Goproxy-Check-Redirect"

func (w *Watcher) makeLoadingPageBody() []byte {
	msg := fmt.Sprintf("%s is starting...", w.ContainerName)

	data := new(templateData)
	data.CheckRedirectHeader = headerCheckRedirect
	data.Title = w.ContainerName
	data.Message = strings.ReplaceAll(msg, " ", "&ensp;")

	buf := bytes.NewBuffer(make([]byte, len(loadingPage)+len(data.Title)+len(data.Message)+len(headerCheckRedirect)))
	err := loadingPageTmpl.Execute(buf, data)
	if err != nil { // should never happen in production
		panic(err)
	}
	return buf.Bytes()
}
