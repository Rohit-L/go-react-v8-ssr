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
		ch <- parseJsonFromCallback(ctx.SafeToString(-1), nil)
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
