package e2e

import (
	"log"
	"net/http/httptest"
	"os"
	"testing"

	httphandler "PRService/internal/adapters/http"
	"PRService/internal/adapters/postgres"
	"PRService/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var server *httptest.Server

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL env variable is required")
	}

	repo := postgres.NewPostgresRepo(dbURL)
	if err := repo.Migrate(); err != nil {
		log.Fatal("migration failed:", err)
	}

	service := services.NewService(repo)

	handler := &httphandler.Handler{S: service}
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

	server = httptest.NewServer(r)
	code := m.Run()
	server.Close()
	os.Exit(code)
}
