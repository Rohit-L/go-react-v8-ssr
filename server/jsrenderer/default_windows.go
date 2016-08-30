// +build windows

package jsrenderer

import "net/http"

// NewDefaultOrDie constructs and initializes a Renderer with the default
// implementation for the current platform.  On windows the default
// implementation is the duktape-based renderer.  On any other platform, the
// default is the V8-based renderer.
func NewDefaultOrDie(jsCode string, local http.Handler) Renderer {
	r, err := NewDukTape(jsCode, local)
	if err != nil {
		panic(err)
	}
	return r
}
