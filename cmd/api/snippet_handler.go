package main

import (
	"encoding/json"
	"net/http"
	"snippet-service/data"
)

func (app *Config) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	var req data.Snippet

	// Parse the request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Unable to parse JSON", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.Title == "" || req.Snippet == "" || req.Type == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	err = app.Models.Snippet.Insert(req)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Snippet saved successfully",
	})
}
