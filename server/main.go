package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/augustoroman/go-react-v8-ssr/server/jsrenderer"
	"github.com/augustoroman/sandwich"
	"github.com/nu7hatch/gouuid"
)

// Values for these are injected during the build process.
var (
	commitHash string
	debug      bool
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Lmicroseconds)

	addr := flag.String("addr", ":5000", "Address to serve on")
	flag.Parse()

	// Get our static data.  Typically this will be embedded into the binary,
	// though it depends on how rice is initialize in the build script.
	// With the current Makefile, it's compiled into the binary.
	templateBox := rice.MustFindBox("data/templates")
	staticBox := rice.MustFindBox("data/static")

	// Setup the react rendering handler:
	reactTpl := template.Must(template.New("react.html").Parse(
		templateBox.MustString("react.html")))
	jsBundle := staticBox.MustString("build/bundle.js")
	renderer := &jsrenderer.Pool{New: func() jsrenderer.Renderer {
		// Duktape on windows, V8 otherwise.  Panics on initialization error.
		return jsrenderer.NewDefaultOrDie(jsBundle, http.DefaultServeMux)
	}}
	react := ReactPage{renderer, reactTpl}

	staticFiles := http.StripPrefix("/static", http.FileServer(staticBox.HTTPBox()))

	// Don't use sandwich for favicon to reduce log spam.
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Now, setup the actual middleware handling & routing:
	mw := sandwich.TheUsual()
	// Gzip all the things!  If you want to be more selective, then move this
	// call to specific handlers below.
	mw = sandwich.Gzip(mw)
	// A fake authentication middleware.
	mw = mw.With(DoAuth)

	// Check out some random API endpoint:
	http.Handle("/api/v1/conf", mw.With(ApiConf))

	// Anything under /static/ goes here:
	http.Handle("/static/", mw.With(staticFiles.ServeHTTP))

	// All other requests will get handling by this:
	http.Handle("/", mw.With(NewUUID, react.Render))
	fmt.Println("Serving on ", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

func NewUUID(e *sandwich.LogEntry) (*uuid.UUID, error) {
	u, err := uuid.NewV4()
	if err == nil {
		e.Note["uuid"] = u.String()
	}
	return u, err
}

type User string

func DoAuth(w http.ResponseWriter, r *http.Request, e *sandwich.LogEntry) User {
	var u User
	if login := r.FormValue("login"); login == "-" {
		deleteCookie(w, r, "user")
		u = "<none>"
	} else if login != "" {
		replaceCookie(w, r, &http.Cookie{
			Name: "user", Value: login, HttpOnly: true})
		u = User(login)
	} else if c, err := r.Cookie("user"); err == nil && c.Value != "" {
		u = User(c.Value)
	} else {
		deleteCookie(w, r, "user")
		u = "<none>"
	}
	e.Note["user"] = string(u)
	return u
}

// deleteCookie removes a cookie from the client (via setting an expired cookie
// in the response headers) and the local request (by re-encoding all the
// cookies without the offending one).
func deleteCookie(w http.ResponseWriter, r *http.Request, name string) {
	replaceCookie(w, r, &http.Cookie{Name: name, HttpOnly: true, MaxAge: -1, Expires: time.Now()})
}

// Replace cookie will set the specified cookie both into the response but also
// in the current request, modifying the headers.  This is necessary for
// correct server-side rendering since we might alter the cookie (e.g. during
// login) and then use the current request's headers to do the page rendering,
// and we want the page rendering to have the correct cookies.
func replaceCookie(w http.ResponseWriter, r *http.Request, c *http.Cookie) {
	cookies := r.Cookies()      // Extract existing cookies
	r.Header.Del("Cookie")      // Delete all cookies
	r.AddCookie(c)              // Add the new cookie first
	for _, e := range cookies { // Add back all other cookies
		if e.Name != c.Name {
			r.AddCookie(e)
		}
	}
	http.SetCookie(w, c) // Also set cookie on response
}

func ApiConf(w http.ResponseWriter, r *http.Request, u User) error {
	config := struct {
		Debug     bool   `json:"debug"`
		Commit    string `json:"commit"`
		Port      int    `json:"port"`
		Title     string `json:"title"`
		User      User   `json:"user"`
		ApiPrefix string `json:"api.prefix"`
		Path      string `json:"duktape.path"`
	}{true, commitHash, 5000, "Go Starter Kit", u, "/api", "static/build/bundle.js"}
	return json.NewEncoder(w).Encode(config)
}

func must(b *rice.Box, err error) *rice.Box {
	if err != nil {
		panic(err)
	}
	return b
}
