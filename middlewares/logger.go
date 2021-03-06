package middlewares

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/lovego/goa"
	loggerPkg "github.com/lovego/logger"
	"github.com/lovego/tracer"
)

type Logger struct {
	Logger           *loggerPkg.Logger
	PanicHandler     func(*goa.Context)
	ShouldLogReqBody func(*goa.Context) bool
	ShouldLogResBody func(*goa.Context) bool
}

func NewLogger(logger *loggerPkg.Logger) *Logger {
	return &Logger{
		Logger:           logger,
		PanicHandler:     defaultPanicHandler,
		ShouldLogReqBody: defaultShouldLogBody,
		ShouldLogResBody: defaultShouldLogBody,
	}
}

func (l *Logger) Record(c *goa.Context) {
	debug := c.URL.Query()["_debug"] != nil
	l.Logger.RecordWithContext(c.Context(), func(tracerCtx context.Context) error {
		if debug {
			tracerCtx = tracer.SetDebug(tracerCtx)
		}
		c.Set("context", tracerCtx)
		c.Next()
		return c.GetError()
	}, func() {
		if l.PanicHandler != nil {
			l.PanicHandler(c)
		}
	}, func(fields *loggerPkg.Fields) {
		l.setFields(fields, c, debug)
	})

}

func (l *Logger) setFields(f *loggerPkg.Fields, c *goa.Context, debug bool) {
	req := c.Request
	f.With("requestId", c.RequestId())
	f.With("host", req.Host)
	f.With("method", req.Method)
	f.With("path", req.URL.Path)
	f.With("rawQuery", req.URL.RawQuery)
	// f.With("query", req.URL.Query())
	f.With("status", c.Status())
	f.With("reqBodySize", req.ContentLength)
	f.With("resBodySize", c.ResponseBodySize())
	f.With("ip", c.ClientAddr())
	f.With("agent", req.UserAgent())
	f.With("refer", req.Referer())
	if sess := c.Get("session"); sess != nil {
		f.With("session", sess)
	}
	if debug || l.ShouldLogReqBody != nil && l.ShouldLogReqBody(c) {
		reqBody, err := c.RequestBody()
		if err != nil {
			if c.GetError() == nil {
				c.SetError(err)
			}
		} else {
			f.With("reqBody", string(reqBody))
		}
	}
	if debug || l.ShouldLogResBody != nil && l.ShouldLogResBody(c) {
		f.With("resBody", string(c.ResponseBody()))
	}
}

func defaultPanicHandler(c *goa.Context) {
	if c.ResponseBodySize() <= 0 {
		c.JsonWithCode(map[string]string{"code": "server-err", "message": "Fatal Server Error."}, 500)
	} else {
		c.WriteHeader(500)
	}
}

func defaultShouldLogBody(c *goa.Context) bool {
	method := c.Request.Method
	return method == http.MethodPost ||
		method == http.MethodDelete ||
		method == http.MethodPut
}

func tryUnmarshal(b []byte) interface{} {
	var v map[string]interface{}
	err := json.Unmarshal(b, &v)
	if err == nil {
		return v
	} else {
		return string(b)
	}
}
