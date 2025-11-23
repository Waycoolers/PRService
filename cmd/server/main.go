package main

import (
	httphandler "PRService/internal/adapters/http"
	"PRService/internal/adapters/postgres"
	"PRService/internal/services"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL env variable is required")
	}
	repo := postgres.NewPostgresRepo(dbURL)
	defer func(repo *postgres.Repo) {
		err := repo.Close()
		if err != nil {
			log.Printf("error closing postgres repo: %v", err)
		}
	}(repo)

	defer func() {
		if err := repo.Close(); err != nil {
			log.Printf("failed to close repo: %v", err)
		}
	}()

	service := services.NewService(repo)

	handler := &httphandler.Handler{
		S: service,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/team/add", handler.CreateTeam)
	r.Get("/team/get", handler.GetTeam)

	r.Post("/users/setIsActive", handler.SetUserActive)
	r.Post("/users/deactivate", handler.DeactivateUsersHandler) // безопасная массовая деактивация
	r.Get("/users/getReview", handler.GetUserPRs)

	r.Post("/pullRequest/create", handler.CreatePR)
	r.Post("/pullRequest/merge", handler.MergePR)
	r.Post("/pullRequest/reassign", handler.ReassignReviewer)

	r.Get("/stats", handler.GetStats)

	port := ":8080"
	log.Printf("Server listening on port %s", port)
	err := http.ListenAndServe(port, r)
	if err != nil {
		log.Printf("server failed: %v", err)
		return
	}
}
