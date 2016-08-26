// +build ignore

package jsrenderer

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/robertkrimen/otto"
)

// The otto renderer is included for comparison, but it dies when trying to
// render react cause of there are regular expressions that use look-ahead which
// isn't supported by re2 that otto uses.
//
// This snippet from node_modules/react/lib/ReactChildren.js fails:
//     var userProvidedKeyEscapeRegex = /\/(?!\/)/g;  <-- this regexp fails
//     function escapeUserProvidedKey(text) {
//       return ('' + text).replace(userProvidedKeyEscapeRegex, '//');
//     }
//
// Even trying to change this to work with re2 still causes some other errors,
// so I'm not sure exactly what's going on.
//
// Also, there's not yet a fetch implementation for otto that I'm aware of.  It
// should be easy to do, however since the rest isn't working I haven't done it
// yet.

func NewOtto(jsCode string, local http.Handler) (Renderer, error) {
	ctx := otto.New()
	_, err := ctx.Run(jsCode)
	if err != nil {
		return nil, fmt.Errorf("Could not load js bundle code: %v", err)
	}
	return &ottoRenderer{ctx, &addHeaders{local, nil}}, err
}

type ottoRenderer struct {
	ctx   *otto.Otto
	local *addHeaders
}

func (o *ottoRenderer) Render(p Params) (Result, error) {
	o.local.Headers = p.Headers
	main, err := o.ctx.Get("main")
	if err != nil {
		return Result{}, fmt.Errorf("Cannot get main(...) func: %v", err)
	}
	ch := make(chan resAndError, 1)
	_, err = main.Call(main, p, func(call otto.FunctionCall) otto.Value {
		ch <- parseJsonFromCallback(call.Argument(0).ToString())
		return otto.UndefinedValue()
	})
	if err != nil {
		return Result{}, fmt.Errorf("Failed to call main(...): %v", err)
	}
	select {
	case res := <-ch:
		return res.Result, res.error
	case <-time.After(time.Second):
		return Result{}, errors.New("Timed out")
	}
}
