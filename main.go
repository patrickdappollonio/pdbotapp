package main

import (
	"log"
	"net/http"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/patrickdappollonio/env"
	"github.com/patrickdappollonio/pdbotapp/handlers"
)

var port = env.GetDefault("PORT", "8080")

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", handlers.Home)

	log.Println("Starting HTTP server on port", port)
	err := gracehttp.Serve(&http.Server{Addr: ":" + port, Handler: r})

	if err != nil {
		log.Fatalf("Error while setting up graceful server: %s", err.Error())
	}
}
