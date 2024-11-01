package idlewatcher

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	"github.com/yusing/go-proxy/internal/common"
)

type templateData struct {
	CheckRedirectHeader string
	Title               string
	Message             string
}

//go:embed html/loading_page.html
var loadingPage []byte
var loadingPageTmpl = template.Must(template.New("loading_page").Parse(string(loadingPage)))

func (w *Watcher) makeLoadingPageBody() []byte {
	msg := w.ContainerName + " is starting..."

	data := new(templateData)
	data.CheckRedirectHeader = common.HeaderCheckRedirect
	data.Title = w.ContainerName
	data.Message = strings.ReplaceAll(msg, " ", "&ensp;")

	buf := bytes.NewBuffer(make([]byte, len(loadingPage)+len(data.Title)+len(data.Message)+len(common.HeaderCheckRedirect)))
	err := loadingPageTmpl.Execute(buf, data)
	if err != nil { // should never happen in production
		panic(err)
	}
	return buf.Bytes()
}
