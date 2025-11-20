package services

import (
	"PRService/internal/ports"
)

type Service struct {
	repo ports.Repository
}

func NewUseCase(repo ports.Repository) *Service {
	return &Service{repo: repo}
}
