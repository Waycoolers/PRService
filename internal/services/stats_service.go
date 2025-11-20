package services

import (
	"PRService/internal/domain"
	"context"
)

func (s *Service) GetStats(ctx context.Context) (domain.Stats, error) {
	reviewerStats, err := s.repo.GetReviewerStats(ctx)
	if err != nil {
		return domain.Stats{}, err
	}

	prStats, err := s.repo.GetPRStats(ctx)
	if err != nil {
		return domain.Stats{}, err
	}

	return domain.Stats{
		ReviewerAssignments: reviewerStats,
		PRAssignments:       prStats,
	}, nil
}
