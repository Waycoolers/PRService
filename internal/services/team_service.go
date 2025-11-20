package services

import (
	"PRService/internal/domain"
	"context"
	"errors"
	"log"
)

func (s *Service) CreateTeam(ctx context.Context, team domain.Team) error {
	err := s.repo.CreateTeam(ctx, team)
	if err != nil {
		if errors.Is(err, domain.ErrTeamExists) {
			return domain.ErrTeamExists
		}
		log.Println("CreateTeam error:", err)
		return err
	}

	users := make([]domain.User, 0, len(team.Members))
	for _, m := range team.Members {
		users = append(users, domain.User{
			UserID:   m.UserID,
			Username: m.Username,
			TeamName: team.TeamName,
			IsActive: m.IsActive,
		})
	}

	if err := s.repo.UpsertUsers(ctx, users); err != nil {
		log.Println("UpsertUsers error:", err)
		return err
	}

	return nil
}

func (s *Service) GetTeam(ctx context.Context, name string) (domain.Team, error) {
	team, err := s.repo.GetTeam(ctx, name)
	if err != nil {
		return domain.Team{}, domain.ErrNotFound
	}
	return team, nil
}
