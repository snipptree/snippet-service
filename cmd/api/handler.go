package main

import "net/http"

func (app *Config) SnippetHome(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the snippet service",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}
