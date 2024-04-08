package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema"
	"github.com/sirupsen/logrus"
	xhtml "golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

type Utils struct {
	portsInUse      map[int]bool
	portsInUseMutex sync.Mutex
}

var utils = &Utils{
	portsInUse:      make(map[int]bool),
	portsInUseMutex: sync.Mutex{},
}

func (u *Utils) findUseFreePort(startingPort int) (int, error) {
	u.portsInUseMutex.Lock()
	defer u.portsInUseMutex.Unlock()
	for port := startingPort; port <= startingPort+100 && port <= 65535; port++ {
		if u.portsInUse[port] {
			continue
		}
		addr := fmt.Sprintf(":%d", port)
		l, err := net.Listen("tcp", addr)
		if err == nil {
			u.portsInUse[port] = true
			l.Close()
			return port, nil
		}
	}
	l, err := net.Listen("tcp", ":0")
	if err == nil {
		// NOTE: may not be after 20000
		port := l.Addr().(*net.TCPAddr).Port
		u.portsInUse[port] = true
		l.Close()
		return port, nil
	}
	return -1, NewNestedError("unable to find free port").With(err)
}

func (u *Utils) markPortInUse(port int) {
	u.portsInUseMutex.Lock()
	u.portsInUse[port] = true
	u.portsInUseMutex.Unlock()
}

func (u *Utils) unmarkPortInUse(port int) {
	u.portsInUseMutex.Lock()
	delete(u.portsInUse, port)
	u.portsInUseMutex.Unlock()
}

func (*Utils) healthCheckHttp(targetUrl string) error {
	// try HEAD first
	// if HEAD is not allowed, try GET
	resp, err := healthCheckHttpClient.Head(targetUrl)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil && resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
		_, err = healthCheckHttpClient.Get(targetUrl)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return err
}

func (*Utils) healthCheckStream(scheme, host string) error {
	conn, err := net.DialTimeout(scheme, host, streamDialTimeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func (*Utils) reloadServer() error {
	resp, err := healthCheckHttpClient.Post("http://localhost:8080/reload", "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return NewNestedError("server reload failed").Subjectf("%d", resp.StatusCode)
	}
	return nil
}

func (*Utils) snakeToPascal(s string) string {
	toHyphenCamel := http.CanonicalHeaderKey(strings.ReplaceAll(s, "_", "-"))
	return strings.ReplaceAll(toHyphenCamel, "-", "")
}

func tryAppendPathPrefixImpl(pOrig, pAppend string) string {
	switch {
	case strings.Contains(pOrig, "://"):
		return pOrig
	case pOrig == "", pOrig == "#", pOrig == "/":
		return pAppend
	case filepath.IsLocal(pOrig) && !strings.HasPrefix(pOrig, pAppend):
		return path.Join(pAppend, pOrig)
	default:
		return pOrig
	}
}

var tryAppendPathPrefix func(string, string) string
var _ = func() int {
	if logLevel == logrus.DebugLevel {
		tryAppendPathPrefix = func(s1, s2 string) string {
			replaced := tryAppendPathPrefixImpl(s1, s2)
			return replaced
		}
	} else {
		tryAppendPathPrefix = tryAppendPathPrefixImpl
	}
	return 1
}()

func htmlNodesSubPath(n *xhtml.Node, p string) {
	if n.Type == xhtml.ElementNode {
		for i, attr := range n.Attr {
			switch attr.Key {
			case "src", "href", "action": // img, script, link, form etc.
				n.Attr[i].Val = tryAppendPathPrefix(attr.Val, p)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		htmlNodesSubPath(c, p)
	}
}

func (*Utils) respHTMLSubPath(r *http.Response, p string) error {
	// remove all path prefix from relative path in script, img, a, ...
	doc, err := xhtml.Parse(r.Body)

	if err != nil {
		return err
	}

	if p[0] == '/' {
		p = p[1:]
	}
	htmlNodesSubPath(doc, p)

	var buf bytes.Buffer
	err = xhtml.Render(&buf, doc)

	if err != nil {
		return err
	}

	r.Body = io.NopCloser(strings.NewReader(buf.String()))

	return nil
}

func (*Utils) respJSSubPath(r *http.Response, p string) error {
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		return err
	}

	if p[0] == '/' {
		p = p[1:]
	}

	js := buf.String()

	re := regexp.MustCompile(`fetch\(["'].+["']\)`)
	replace := func(match string) string {
		match = match[7 : len(match)-2]
		replaced := tryAppendPathPrefix(match, p)
		return fmt.Sprintf(`fetch(%q)`, replaced)
	}
	js = re.ReplaceAllStringFunc(js, replace)

	r.Body = io.NopCloser(strings.NewReader(js))
	return nil
}

func (*Utils) fileOK(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func setFieldFromSnake[T interface{}, VT interface{}](obj *T, field string, value VT) error {
	field = utils.snakeToPascal(field)
	prop := reflect.ValueOf(obj).Elem().FieldByName(field)
	if prop.Kind() == 0 {
		return errors.New("unknown field")
	}
	prop.Set(reflect.ValueOf(value))
	return nil
}

func validateYaml(schema *jsonschema.Schema, data []byte) error {
	if noSchemaValidation {
		return nil
	}

	var i interface{}

	err := yaml.Unmarshal(data, &i)
	if err != nil {
		return NewNestedError("unable to unmarshal yaml").With(err)
	}

	m, err := json.Marshal(i)
	if err != nil {
		return NewNestedError("unable to marshal json").With(err)
	}

	err = schema.Validate(bytes.NewReader(m))
	if err != nil {
		valErr := err.(*jsonschema.ValidationError)
		ne := NewNestedError("validation error")
		for _, e := range valErr.Causes {
			ne.ExtraError(e)
		}
		return ne
	}
	return nil
}
