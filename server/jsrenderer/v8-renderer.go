// +build !windows

package jsrenderer

import (
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

// NewV8 constructs a Renderer backed by the V8 javascript engine. A given
// Renderer instance is not safe for concurrent usage.
//
// The provided javascript code should be all of the javascript necessary to
// render the desired react page, including all dependencies bundled together.
// It is assumed that the react code exposes a single function:
//
//   main(params, callback)
//
// where params corresponds to the Params struct in this package and callback
// is a function that receives a Result struct serialized to a JSON string.
// For example:
//
//   function main(params, callback) {
//     result = { app: '<div>hi there!</div>', title: '<title>The app</title>' };
//     callback(JSON.stringify(result));
//   }
//
// In addition to the javascript code, you should also provide an http.Handler
// to call for any requests to the local server.
//
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
	// Update the local addHeaders server to include the current request's cookies.
	r.local.Headers = p.Headers

	// Convert the params Go struct into a javascript object.
	params, err := r.ctx.Create(p)
	if err != nil {
		return Result{}, fmt.Errorf("Cannot create params ob: %v", err)
	}

	// Setup the callback function for the current handler.
	res := make(chan resAndError, 1)
	callback := r.ctx.Bind("rendered_result", func(in v8.CallbackArgs) (*v8.Value, error) {
		res <- parseJsonFromCallback(in.Arg(0).String(), nil)
		return nil, nil
	})

	// Get and call main() in the js code to render the react page.
	main, err := r.ctx.Global().Get("main")
	if err != nil {
		return Result{}, fmt.Errorf("Can't get global main(): %v", err)
	}
	_, err = main.Call(main, params, callback)
	if err != nil {
		return Result{}, fmt.Errorf("Call to main(...) failed: %v", err)
	}

	// Wait for a response.  If it times out, kill the V8 engine and return
	// an error.
	select {
	case resp := <-res:
		return resp.Result, resp.error
	case <-time.After(time.Second):
		r.ctx.Terminate() // TODO(aroman): re-initialize ctx
		return Result{}, errors.New("Timed out")
	}
}
