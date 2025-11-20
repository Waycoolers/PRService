package services

import (
	"PRService/internal/domain"
	"context"
	"errors"
	"fmt"
)

func (s *Service) SetActive(ctx context.Context, userID string, isActive bool) (domain.User, error) {
	user, err := s.repo.SetUserActive(ctx, userID, isActive)
	if err != nil {
		return domain.User{}, domain.ErrNotFound
	}
	return user, nil
}

func (s *Service) GetPRsForReviewer(ctx context.Context, userID string) ([]domain.PullRequest, error) {
	return s.repo.ListPRsByReviewer(ctx, userID)
}

func (s *Service) DeactivateUsers(ctx context.Context, userIDs []string) (map[string]string, error) {
	results := make(map[string]string)

	for _, userID := range userIDs {
		prs, err := s.repo.ListPRsByReviewer(ctx, userID)
		if err != nil {
			results[userID] = fmt.Sprintf("failed to list PRs: %v", err)
			continue
		}

		canDeactivate := true
		for _, pr := range prs {
			_, _, err := s.ReassignReviewer(ctx, pr.PullRequestID, userID)
			if err != nil {
				if errors.Is(err, domain.ErrNoCandidate) {
					results[userID] = fmt.Sprintf("PR %s has no available replacement", pr.PullRequestID)
				} else {
					results[userID] = fmt.Sprintf("failed to reassign PR %s: %v", pr.PullRequestID, err)
				}
				canDeactivate = false
				break
			}
		}

		if canDeactivate {
			_, err := s.repo.SetUserActive(ctx, userID, false)
			if err != nil {
				results[userID] = fmt.Sprintf("failed to deactivate: %v", err)
			} else {
				results[userID] = "success"
			}
		}
	}

	return results, nil
}
