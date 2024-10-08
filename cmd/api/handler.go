package main

import (
	"net/http"
	"snippet-service/data"
)

func (app *Config) SnippetHome(w http.ResponseWriter, r *http.Request) {
	payload := data.JsonResponse{
		Error:   false,
		Message: "Hit the snippet service",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}
