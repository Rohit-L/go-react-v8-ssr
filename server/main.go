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
	"github.com/augustoroman/sandwich"
	"github.com/nu7hatch/gouuid"
)

// Values for these are injected during the build process.
var (
	commitHash string
	debug      bool
)

func main() {
	addr := flag.String("addr", ":5000", "Address to serve on")
	flag.Parse()

	cfg := rice.Config{[]rice.LocateMethod{
		rice.LocateWorkingDirectory,
		rice.LocateFS,
		rice.LocateAppended,
		rice.LocateEmbedded,
	}}
	templateBox := must(cfg.FindBox("data/templates"))
	staticBox := must(cfg.FindBox("data/static"))

	// Setup the react rendering handler
	reactTpl := template.Must(template.New("react.html").Parse(
		templateBox.MustString("react.html")))
	jsBundle := staticBox.MustString("build/bundle.js")
	react, err := NewV8React(jsBundle, reactTpl, http.DefaultServeMux)
	if err != nil {
		log.Fatal(err)
	}

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
	http.Handle("/", mw.With(NewUUID, react.Execute, react.Render))
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
		http.SetCookie(w, &http.Cookie{Name: "user", HttpOnly: true, MaxAge: 0, Expires: time.Now()})
		u = "<none>"
	} else if login != "" {
		http.SetCookie(w, &http.Cookie{Name: "user", Value: login, HttpOnly: true})
		u = User(login)
	} else if c, err := r.Cookie("user"); err == nil && c.Value != "" {
		u = User(c.Value)
	} else {
		u = "<none>"
	}
	e.Note["user"] = string(u)
	return u
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
