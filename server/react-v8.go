package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/augustoroman/sandwich"
	"github.com/augustoroman/v8"
	"github.com/augustoroman/v8/v8console"
	"github.com/augustoroman/v8fetch"
	"github.com/nu7hatch/gouuid"
)

type V8React struct {
	tpl   *template.Template
	local http.Handler

	get <-chan *v8.Context
	ret chan<- *v8.Context
}

func NewV8React(jsCode string, tpl *template.Template, local http.Handler) (*V8React, error) {
	var createdCount int32
	newContext := func() *v8.Context {
		ctx := v8.NewIsolate().NewContext()
		v8fetch.Inject(ctx, local)
		num := atomic.AddInt32(&createdCount, 1)
		v8console.Config{fmt.Sprintf("Context #%d> ", num), os.Stdout, os.Stderr, true}.Inject(ctx)
		_, err := ctx.Eval(jsCode, "bundle.js")
		if err != nil {
			panic(err)
		}
		ctx.Eval("console.info('Initialized new context')", "<server>")
		return ctx
	}

	get, ret := newContextPool(newContext)

	return &V8React{tpl, local, get, ret}, nil
}

func (h *V8React) Execute(w http.ResponseWriter, r *http.Request, UUID *uuid.UUID, e *sandwich.LogEntry) Resp {
	resp := Resp{UUID: UUID.String(), startTime: time.Now()}

	ctx := <-h.get
	v8console.Config{UUID.String() + "> ", os.Stdout, os.Stderr, true}.Inject(ctx)
	defer func() { h.ret <- ctx }()

	renderParams, err := ctx.Create(map[string]interface{}{
		"url":     r.URL.String(),
		"headers": r.Header,
		"uuid":    UUID.String(),
	})
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	res := make(chan Resp, 1)
	callback := ctx.Bind("rendered_result", func(l v8.Loc, args ...*v8.Value) (*v8.Value, error) {
		if len(args) < 1 {
			resp.Error = "No result returned from rendering engine."
		} else {
			val := args[0].String()
			err := json.Unmarshal([]byte(val), &resp)
			if err != nil {
				resp.Error = fmt.Sprintf("Error to callback: %v\nArg: %s", err, val)
			}
		}
		res <- resp
		return nil, nil
	})
	main, err := ctx.Global().Get("main")
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	_, err = main.Call(main, renderParams, callback)
	if err != nil {
		resp.Error = fmt.Sprintf("Failed to call main(...): %v", err)
		return resp
	}
	select {
	case resp = <-res:
	case <-time.After(time.Second):
		ctx.Terminate()
		ctx = nil // throw this context out.
		resp.Error = "Timed out"
	}
	resp.UUID = UUID.String()
	return resp
}

func (h *V8React) Render(w http.ResponseWriter, r *http.Request, resp Resp, e *sandwich.LogEntry) {
	resp.RenderTime = time.Since(resp.startTime)
	e.Note["react-render"] = resp.RenderTime.String()

	if len(resp.Redirect) > 0 {
		http.Redirect(w, r, resp.Redirect, http.StatusMovedPermanently)
		return
	}
	if len(resp.Error) > 0 {
		e.Error = errors.New(resp.Error)
		w.WriteHeader(http.StatusInternalServerError)
	}

	h.tpl.ExecuteTemplate(w, "react.html", resp)
}

// Resp is a struct for convinient
// react app Response parsing.
// Feel free to add any other keys to this struct
// and return value for this key at ecmascript side.
// Keep it sync with: src/app/client/router/toString.js:23
type Resp struct {
	UUID       string        `json:"uuid"`
	Error      string        `json:"error"`
	Redirect   string        `json:"redirect"`
	App        string        `json:"app"`
	Title      string        `json:"title"`
	Meta       string        `json:"meta"`
	Initial    string        `json:"initial"`
	RenderTime time.Duration `json:"-"`
	startTime  time.Time     `json:"-"`
}

// HTMLApp returns a application template
func (r Resp) HTMLApp() template.HTML {
	return template.HTML(r.App)
}

// HTMLTitle returns a title data
func (r Resp) HTMLTitle() template.HTML {
	return template.HTML(r.Title)
}

// HTMLMeta returns a meta data
func (r Resp) HTMLMeta() template.HTML {
	return template.HTML(r.Meta)
}

func newContextPool(newContext func() *v8.Context) (<-chan *v8.Context, chan<- *v8.Context) {
	first := newContext()
	get, ret := make(chan *v8.Context), make(chan *v8.Context, 1)
	go func() {
		var pool []*v8.Context
		var next *v8.Context = first
		for {
			if next != nil {
				// We have one ready to be used.
			} else if len(pool) > 0 {
				// Pull the most recently returned from the pool.
				N := len(pool)
				pool, next = pool[:N-1], pool[N-1]
			} else {
				// Oops, the pool is empty. Create a new one.
				next = newContext()
			}

			select {
			case get <- next:
				next = nil
			case returned, open := <-ret:
				if !open {
					close(get)
					return // return channel closed, time to shutdown.
				}
				if returned != nil {
					pool = append(pool, next)
					next = returned
				}
			}
		}
	}()
	return get, ret
}
