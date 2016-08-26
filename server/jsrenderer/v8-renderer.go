// +build !windows

package jsrenderer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/augustoroman/v8"
	"github.com/augustoroman/v8/v8console"
	"github.com/augustoroman/v8fetch"
)

var createdCount int32

func NewV8(jsCode string, local http.Handler) (Renderer, error) {
	ctx := v8.NewIsolate().NewContext()
	ah := &addHeaders{Server: local}
	v8fetch.Inject(ctx, ah)
	num := atomic.AddInt32(&createdCount, 1)
	v8console.Config{fmt.Sprintf("Context #%d> ", num), os.Stdout, os.Stderr, true}.Inject(ctx)
	_, err := ctx.Eval(jsCode, "bundle.js")
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize context from bundle code: %v", err)
	}
	ctx.Eval("console.info('Initialized new context')", "<server>")
	return v8Renderer{ctx, ah}, nil
}

type v8Renderer struct {
	ctx   *v8.Context
	local *addHeaders
}

func (r v8Renderer) Render(p Params) (Result, error) {
	// Update the console log bindings to prefix logs with the current request UUID.
	v8console.Config{p.UUID + "> ", os.Stdout, os.Stderr, true}.Inject(r.ctx)
	// Update the local server to include the current request's cookies.
	r.local.Headers = p.Headers

	params, err := r.ctx.Create(p)
	if err != nil {
		return Result{}, fmt.Errorf("Cannot create params ob: %v", err)
	}

	res := make(chan resAndError, 1)
	callback := r.ctx.Bind("rendered_result", func(l v8.Loc, args ...*v8.Value) (*v8.Value, error) {
		res <- r.resultCallback(args)
		return nil, nil
	})

	main, err := r.ctx.Global().Get("main")
	if err != nil {
		return Result{}, fmt.Errorf("Can't get global main(): %v", err)
	}
	_, err = main.Call(main, params, callback)
	if err != nil {
		return Result{}, fmt.Errorf("Call to main(...) failed: %v", err)
	}

	select {
	case resp := <-res:
		return resp.Result, resp.error
	case <-time.After(time.Second):
		r.ctx.Terminate() // TODO(aroman): re-initialize ctx
		return Result{}, errors.New("Timed out")
	}
}

func (v *v8Renderer) resultCallback(args []*v8.Value) resAndError {
	if len(args) < 1 {
		return resAndError{error: errors.New("No result returned from rendering engine.")}
	}
	jsonVal := args[0].String()
	var res struct {
		Error string `json:"error"`
		Result
	}
	if err := json.Unmarshal([]byte(jsonVal), &res); err != nil {
		return resAndError{res.Result, err}
	} else if res.Error != "" {
		return resAndError{res.Result, errors.New(res.Error)}
	}
	return resAndError{res.Result, nil}
}
