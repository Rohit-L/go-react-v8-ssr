// Package jsrenderer is an abstract API for server-side react rendering with
// several implementations.
package jsrenderer

import (
	"errors"
	"html/template"
	"net/http"
	"sync"
)

// Renderer is the primary interface for rendering react content.  Given a set
// of parameters, it will execute the react js code and return the result and
// error (if any).  A given Renderer is typically NOT SAFE for concurrent use.
// Instead, use a Pool.
type Renderer interface {
	Render(Params) (Result, error)
}

// Params describe the options that can be pass to the react rendering code.
type Params struct {
	// Url is used directly by react-router to determine what to render.
	Url string `json:"url"`
	// Headers specifies additional headers to be included for any HTTP requests
	// to the local server during rendering.  For exampe, if the rendering code
	// needs to make an authenticated API call, it's important that the
	// authentication be made on behalf of the correct user.
	Headers http.Header `json:"headers"`
	UUID    string      `json:"uuid"`
}

// Result is returned from a successful react rendering.
type Result struct {
	// The rendered react content, if any.
	Rendered string `json:"app"`
	// The URL that the client should be redirected to (if non-empty).
	Redirect string `json:"redirect"`
	// The title of the page.
	Title string `json:"title"`
	// Meta HTML tags that should be included on the page.
	Meta string `json:"meta"`
	// Initial JSON data that should be included on the page, if any.
	Initial string `json:"initial"`
}

func (r Result) HTMLApp() template.HTML   { return template.HTML(r.Rendered) }
func (r Result) HTMLTitle() template.HTML { return template.HTML(r.Title) }
func (r Result) HTMLMeta() template.HTML  { return template.HTML(r.Meta) }

// The javascript rendering engine timed out.
var ErrTimeOut = errors.New("Timed out")

// Pool is a dynamically-sized pool of Renderers that itself implements the
// Renderer interface.  You must provide a New() function that will construct
// and initialize a new Renderer when necessary.  It will grow the pool as
// necessary to accommodate rendering demands, and will tend to re-use renderers
// as much as possible. A Pool Renderer is safe for concurrent use.
type Pool struct {
	New func() Renderer

	mu        sync.Mutex
	available []Renderer
}

func (p *Pool) Render(params Params) (Result, error) {
	r := p.get()
	res, err := r.Render(params)
	if err != ErrTimeOut {
		p.put(r) // If the engine timed out, throw it away. Otherwise re-use it.
	}
	return res, err
}
func (p *Pool) get() Renderer {
	var r Renderer

	// Check to see if any are available.
	p.mu.Lock()
	if N := len(p.available); N > 0 {
		r = p.available[N-1]
		p.available = p.available[:N-1]
	}
	p.mu.Unlock()

	// Did we get one?  If not, allocate a new one.
	if r == nil {
		r = p.New()
	}

	return r
}

func (p *Pool) put(r Renderer) {
	p.mu.Lock()
	p.available = append(p.available, r)
	p.mu.Unlock()
}
