package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/augustoroman/go-react-v8-ssr/server/jsrenderer"

	"github.com/augustoroman/sandwich"
	"github.com/nu7hatch/gouuid"
)

type ReactPage struct {
	r   jsrenderer.Renderer
	tpl *template.Template
}

func (v *ReactPage) Render(
	w http.ResponseWriter,
	r *http.Request,
	UUID *uuid.UUID,
	e *sandwich.LogEntry,
) error {
	start := time.Now()
	// First, we execute the react javascript to process the main content of
	// the page.  The Helmet components will also return information about
	// titles & meta tags.
	res, err := v.r.Render(jsrenderer.Params{
		Url:     r.URL.String(),
		Headers: http.Header{"Cookie": r.Header["Cookie"]},
		UUID:    UUID.String(),
	})
	e.Note["react-render"] = time.Since(start).String()

	var data = struct {
		jsrenderer.Result
		UUID  string
		Error error
	}{res, UUID.String(), err}

	return v.tpl.ExecuteTemplate(w, "react.html", data)
}
