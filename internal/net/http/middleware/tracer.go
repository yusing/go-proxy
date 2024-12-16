package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type Tracer struct {
	name   string
	parent *Tracer
}

func (t *Tracer) Fullname() string {
	if t.parent != nil {
		return t.parent.Fullname() + "." + t.name
	}
	return t.name
}

func (t *Tracer) addTrace(msg string) *Trace {
	return addTrace(&Trace{
		Time:    strutils.FormatTime(time.Now()),
		Caller:  t.Fullname(),
		Message: msg,
	})
}

func (t *Tracer) AddTracef(msg string, args ...any) *Trace {
	if t == nil {
		return nil
	}
	return t.addTrace(fmt.Sprintf(msg, args...))
}

func (t *Tracer) AddTraceRequest(msg string, req *http.Request) *Trace {
	if t == nil {
		return nil
	}
	return t.addTrace(msg).WithRequest(req)
}

func (t *Tracer) AddTraceResponse(msg string, resp *http.Response) *Trace {
	if t == nil {
		return nil
	}
	return t.addTrace(msg).WithResponse(resp)
}
