package jsrenderer

import (
	"html/template"
	"net/http"
	"sync"
)

type Renderer interface {
	Render(Params) (Result, error)
}

type Params struct {
	Url     string      `json:"url"`
	Headers http.Header `json:"headers"`
	UUID    string      `json:"uuid"`
}

type Result struct {
	Redirect string `json:"redirect"`
	Rendered string `json:"app"`
	Title    string `json:"title"`
	Meta     string `json:"meta"`
	Initial  string `json:"initial"`
}

func (r Result) HTMLApp() template.HTML   { return template.HTML(r.Rendered) }
func (r Result) HTMLTitle() template.HTML { return template.HTML(r.Title) }
func (r Result) HTMLMeta() template.HTML  { return template.HTML(r.Meta) }

type Pool struct {
	New func() Renderer

	mu        sync.Mutex
	available []Renderer
}

func (p *Pool) Render(params Params) (Result, error) {
	r := p.get()
	defer p.put(r)
	return r.Render(params)
}
func (p *Pool) get() Renderer {
	p.mu.Lock()
	if N := len(p.available); N > 0 {
		r := p.available[N-1]
		p.available = p.available[:N-1]
		p.mu.Unlock()
		return r
	}
	p.mu.Unlock()
	return p.New()
}

func (p *Pool) put(r Renderer) {
	p.mu.Lock()
	p.available = append(p.available, r)
	p.mu.Unlock()
}
