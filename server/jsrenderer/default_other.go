// +build !windows

package jsrenderer

import "net/http"

func NewDefaultOrDie(jsCode string, local http.Handler) Renderer {
	r, err := NewV8(jsCode, local)
	if err != nil {
		panic(err)
	}
	return r
}
