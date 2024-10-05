package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	U "github.com/yusing/go-proxy/internal/utils"
)

type Trace struct {
	Time        string         `json:"time,omitempty"`
	Caller      string         `json:"caller,omitempty"`
	URL         string         `json:"url,omitempty"`
	Message     string         `json:"msg"`
	ReqHeaders  http.Header    `json:"req_headers,omitempty"`
	RespHeaders http.Header    `json:"resp_headers,omitempty"`
	RespStatus  int            `json:"resp_status,omitempty"`
	Additional  map[string]any `json:"additional,omitempty"`
}

type Traces []*Trace

var traces = Traces{}
var tracesMu sync.Mutex

const MaxTraceNum = 1000

func GetAllTrace() []*Trace {
	return traces
}

func (tr *Trace) WithRequest(req *Request) *Trace {
	if tr == nil {
		return nil
	}
	tr.URL = req.RequestURI
	tr.ReqHeaders = req.Header.Clone()
	return tr
}

func (tr *Trace) WithResponse(resp *Response) *Trace {
	if tr == nil {
		return nil
	}
	tr.URL = resp.Request.RequestURI
	tr.ReqHeaders = resp.Request.Header.Clone()
	tr.RespHeaders = resp.Header.Clone()
	tr.RespStatus = resp.StatusCode
	return tr
}

func (tr *Trace) With(what string, additional any) *Trace {
	if tr == nil {
		return nil
	}

	if tr.Additional == nil {
		tr.Additional = map[string]any{}
	}
	tr.Additional[what] = additional
	return tr
}

func (m *Middleware) EnableTrace() {
	m.trace = true
	for _, child := range m.children {
		child.parent = m
		child.EnableTrace()
	}
}

func (m *Middleware) AddTracef(msg string, args ...any) *Trace {
	if !m.trace {
		return nil
	}
	return addTrace(&Trace{
		Time:    U.FormatTime(time.Now()),
		Caller:  m.Fullname(),
		Message: fmt.Sprintf(msg, args...),
	})
}

func (m *Middleware) AddTraceRequest(msg string, req *Request) *Trace {
	return m.AddTracef("%s", msg).WithRequest(req)
}

func (m *Middleware) AddTraceResponse(msg string, resp *Response) *Trace {
	return m.AddTracef("%s", msg).WithResponse(resp)
}

func addTrace(t *Trace) *Trace {
	tracesMu.Lock()
	defer tracesMu.Unlock()
	if len(traces) > MaxTraceNum {
		traces = traces[1:]
	}
	traces = append(traces, t)
	return t
}
