package jsrenderer

import (
	"encoding/json"
	"errors"
)

func parseJsonFromCallback(jsonData string, err error) resAndError {
	if err != nil {
		return resAndError{error: err}
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
