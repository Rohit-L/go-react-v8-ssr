package jsrenderer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/olebedev/go-duktape-fetch.v2"
	"gopkg.in/olebedev/go-duktape.v2"
)

// NewDukTape constructs a Renderer backed by the duktape javascript engine.
// A given Renderer instance is not safe for concurrent usage.
//
// The provided javascript code should be all of the javascript necessary to
// render the desired react page, including all dependencies bundled together.
// It is assumed that the react code exposes a single function:
//
//   main(params, callback)
//
// where params corresponds to the Params struct in this package and callback
// is a function that receives a Result struct serialized to a JSON string.
//
// In addition to the javascript code, you should also provide an http.Handler
// to call for any requests to the local server.
func NewDukTape(jsCode string, local http.Handler) (Renderer, error) {
	ctx := duktape.New()
	// Setup the global console object.
	ctx.PevalString(`var console = {log:print,warn:print,error:print,info:print}`)
	// Setup the fetch() polyfill.
	ah := &addHeaders{local, nil}
	fetch.PushGlobal(ctx, ah)

	// Load the rendering code.
	if err := ctx.PevalString(jsCode); err != nil {
		return nil, fmt.Errorf("Could not load js bundle code: %v", err)
	}
	ctx.PopN(ctx.GetTop())

	return duktapeRenderer{ctx, ah}, nil
}

type duktapeRenderer struct {
	ctx   *duktape.Context
	local *addHeaders
}

func (d duktapeRenderer) Render(p Params) (Result, error) {
	// Update the local addHeaders server to include the current request's cookies.
	d.local.Headers = p.Headers

	// Convert the parameters to JSON so we can call main.
	data, err := json.Marshal(p)
	if err != nil {
		panic(err) // should never happen
	}

	// Setup the callback function.
	ch := make(chan resAndError, 1)
	d.ctx.PushGlobalGoFunction("__goServerCallback__", func(ctx *duktape.Context) int {
		ch <- parseJsonFromCallback(ctx.SafeToString(-1), nil)
		return 0
	})

	// Call main() in the js code to render the react page.
	if err := d.ctx.PevalString(`main(` + string(data) + `, __goServerCallback__)`); err != nil {
		return Result{}, fmt.Errorf("Could not call main(...): %v", err)
	}

	// Wait for a response.
	select {
	case res := <-ch:
		return res.Result, res.error
	case <-time.After(time.Second):
		// TODO(aroman) Kill the engine here.
		return Result{}, errors.New("Timed out")
	}
}
