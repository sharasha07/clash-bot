package main

import (
	"encoding/json"
	"net/http"
)

type envelope map[string]any

func (app *application) writeJSON(w http.ResponseWriter, status int, env envelope) error {
	body, err := json.MarshalIndent(env, "", "\t")
	if err != nil {
		return err
	}

	body = append(body, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body)

	return nil
}
