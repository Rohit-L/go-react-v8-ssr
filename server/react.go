package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/augustoroman/go-react-v8-ssr/server/jsrenderer"

	"github.com/augustoroman/sandwich"
	"github.com/nu7hatch/gouuid"
)

// ReactPage configures the rendering of a server-side rendered react page.
// It fundamentally consists of two things: the javascript engine that will
// render the react content based on the request parameters, and the go
// template that will wrap the rendered content.  The wrapping includes the
// basic HTML document structure as well as links to the react javascript (for
// any future client-side renderings) and links to style sheets as so on.  In
// our case, the wrapping also conditionally includes error-rendering.
type ReactPage struct {
	r   jsrenderer.Renderer
	tpl *template.Template
}

// Render is the actual HTTP handler portion
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
		// Include the URL to be rendered for the react router.
		Url: r.URL.String(),
		// Pass in the cookies from the client request so that they can be
		// added to any child HTTP calls made by the renderer during the server-
		// side rendering of the page.  That will allow authenticated API calls
		// to succeed as if they were made by the client itself.
		Headers: http.Header{"Cookie": r.Header["Cookie"]},
		UUID:    UUID.String(),
	})
	e.Note["react-render"] = time.Since(start).String()

	// Once we have the react content rendered, just pass the result into our
	// wrapping template and render that!  Easy-peasy!
	var templateData = struct {
		jsrenderer.Result
		UUID  string
		Error error
	}{res, UUID.String(), err}
	return v.tpl.ExecuteTemplate(w, "react.html", templateData)
}
