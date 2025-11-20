package services

import (
	"PRService/internal/domain"
	"context"
	"time"
)

func (s *Service) CreatePR(ctx context.Context, pr domain.PullRequest) (domain.PullRequest, error) {
	if _, err := s.repo.GetPR(ctx, pr.PullRequestID); err == nil {
		return domain.PullRequest{}, domain.ErrPrExists
	}

	author, err := s.repo.GetUser(ctx, pr.AuthorID)
	if err != nil {
		return domain.PullRequest{}, domain.ErrNotFound
	}

	exclude := []string{pr.AuthorID}
	candidates, err := s.repo.ListActiveTeamMembers(ctx, author.TeamName, exclude, 2)
	if err != nil {
		return domain.PullRequest{}, err
	}

	reviewers := make([]string, 0, len(candidates))
	for _, c := range candidates {
		reviewers = append(reviewers, c.UserID)
	}

	now := time.Now().UTC()
	pr.AssignedReviewers = reviewers
	pr.Status = domain.StatusOpen
	pr.CreatedAt = &now

	if err := s.repo.CreatePR(ctx, pr, reviewers); err != nil {
		return domain.PullRequest{}, err
	}

	return pr, nil
}

func (s *Service) MergePR(ctx context.Context, prID string) (domain.PullRequest, error) {
	pr, err := s.repo.GetPR(ctx, prID)
	if err != nil {
		return domain.PullRequest{}, domain.ErrNotFound
	}

	if pr.Status == domain.StatusMerged {
		return pr, nil
	}

	now := time.Now().UTC()
	updated, err := s.repo.UpdatePRStatusMerged(ctx, prID, &now)
	if err != nil {
		return domain.PullRequest{}, err
	}

	return updated, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (domain.PullRequest, string, error) {
	pr, err := s.repo.GetPR(ctx, prID)
	if err != nil {
		return domain.PullRequest{}, "", domain.ErrNotFound
	}

	if pr.Status == domain.StatusMerged {
		return domain.PullRequest{}, "", domain.ErrPrMerged
	}

	isAssigned := false
	for _, r := range pr.AssignedReviewers {
		if r == oldUserID {
			isAssigned = true
			break
		}
	}
	if !isAssigned {
		return domain.PullRequest{}, "", domain.ErrNotAssigned
	}

	author, err := s.repo.GetUser(ctx, pr.AuthorID)
	if err != nil {
		return domain.PullRequest{}, "", domain.ErrNotFound
	}

	exclude := append(pr.AssignedReviewers, author.UserID)

	candidates, err := s.repo.ListActiveTeamMembers(ctx, author.TeamName, exclude, 1)
	if err != nil || len(candidates) == 0 {
		return domain.PullRequest{}, "", domain.ErrNoCandidate
	}

	newReviewer := candidates[0].UserID

	pr, err = s.repo.ReplaceReviewer(ctx, prID, oldUserID, newReviewer)
	if err != nil {
		return domain.PullRequest{}, "", err
	}

	return pr, newReviewer, nil
}
