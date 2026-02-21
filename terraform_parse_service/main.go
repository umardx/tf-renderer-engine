package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/umardx/tf-renderer-engine/api"
	"github.com/umardx/tf-renderer-engine/middleware"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	r := chi.NewRouter()

	r.Use(middleware.Recover(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Post("/render", api.RenderHandler(logger))

	port := getPort()
	logger.Info("starting server", zap.String("port", port))

	http.ListenAndServe(":"+port, r)
}

func getPort() string {
	if p := os.Getenv("HTTP_PORT"); p != "" {
		return p
	}
	return "8080"
}
