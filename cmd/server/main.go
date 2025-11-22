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
	defer repo.Close()
	if err := repo.Migrate(); err != nil {
		log.Fatal("migrate: ", err)
	}

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
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
