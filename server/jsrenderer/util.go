package jsrenderer

import (
	"encoding/json"
	"errors"
	"net/http"
)

type resAndError struct {
	Result
	error
}

func parseJsonFromCallback(jsonData string, err error) resAndError {
	if err != nil {
		return resAndError{error: err}
	}
	if jsonData == "undefined" {
		return resAndError{error: errors.New("No result returned from rendering engine.")}
	}
	var res struct {
		Error string `json:"error"`
		Result
	}
	if err := json.Unmarshal([]byte(jsonData), &res); err != nil {
		return resAndError{res.Result, err}
	} else if res.Error != "" {
		return resAndError{res.Result, errors.New(res.Error)}
	}
	return resAndError{res.Result, nil}
}

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
