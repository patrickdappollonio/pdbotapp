package main

import (
	"log"
	"net/http"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/patrickdappollonio/env"
	"github.com/patrickdappollonio/pdbotapp/chat"
	"github.com/patrickdappollonio/pdbotapp/handlers"
)

var (
	port    = env.GetDefault("PORT", "8080")
	channel = env.GetDefault("CHANNEL", "patrickdappollonio")
)

func main() {
	if err := chat.Setup(); err != nil {
		log.Fatalf("Error connecting: %s", err.Error())
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", handlers.Home)

	log.Println("Starting HTTP server on port", port)
	listen(r)
}

func listen(r http.Handler) {
	if err := gracehttp.Serve(&http.Server{Addr: ":" + port, Handler: r}); err != nil {
		log.Fatalf("Error while setting up graceful server: %s", err.Error())
	}
}
