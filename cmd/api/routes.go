package main

import (
	"net/http"

	appMiddleware "snippet-service/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (app *Config) Routes() http.Handler {
	mux := chi.NewRouter()

	corsMiddleware := cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"https://*",
			"http://*",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	//specify who is allowed to connect
	mux.Use(corsMiddleware)

	mux.Use(middleware.Heartbeat("/ping"))

	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		payload := jsonResponse{
			Error:   false,
			Message: "healthy",
		}
		app.writeJSON(w, http.StatusOK, payload)
	})
	mux.Get("/", app.SnippetHome)
	mux.Get("/search", app.SearchSnippet)
	mux.Get("/snippet/{id}", app.GetSnippetById)

	protectedRouter := chi.NewRouter()
	protectedRouter.Use(corsMiddleware)
	protectedRouter.Use(appMiddleware.JWTMiddleware)
	protectedRouter.Post("/snippet", app.CreateSnippet)
	protectedRouter.Get("/user/snippets", app.GetUserSnippets)

	mux.Mount("/", protectedRouter)

	return mux
}
