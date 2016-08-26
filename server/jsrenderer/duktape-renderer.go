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

// addHeaders wraps Server and adds all of the provided headers to any
// request processed by it.  This can be used to copy cookies from a client
// request to all fetch calls during server-side rendering.
type addHeaders struct {
	Server  http.Handler
	Headers http.Header
}

func (a addHeaders) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for key, vals := range a.Headers {
		for _, val := range vals {
			r.Header.Add(key, val)
		}
	}
	a.Server.ServeHTTP(w, r)
}

func NewDukTape(jsCode string, local http.Handler) (Renderer, error) {
	ctx := duktape.New()
	ctx.PevalString(`var console = {log:print,warn:print,error:print,info:print}`)
	ah := &addHeaders{local, nil}
	fetch.PushGlobal(ctx, ah)

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
	d.local.Headers = p.Headers

	data, err := json.Marshal(p)
	if err != nil {
		panic(err) // should never happen
	}

	ch := make(chan resAndError, 1)
	d.ctx.PushGlobalGoFunction("__goServerCallback__", func(ctx *duktape.Context) int {
		jsonResult := ctx.SafeToString(-1)
		var res resAndError
		res.error = json.Unmarshal([]byte(jsonResult), &res.Result)
		ch <- res
		return 0
	})

	if err := d.ctx.PevalString(`main(` + string(data) + `, __goServerCallback__)`); err != nil {
		return Result{}, fmt.Errorf("Could not call main(...): %v", err)
	}

	select {
	case res := <-ch:
		return res.Result, res.error
	case <-time.After(time.Second):
		return Result{}, errors.New("Timed out")
	}
}
