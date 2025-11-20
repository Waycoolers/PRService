package ports

import (
	"PRService/internal/domain"
	"context"
	"time"
)

type Repository interface {
	CreateTeam(ctx context.Context, team domain.Team) error
	GetTeam(ctx context.Context, teamName string) (domain.Team, error)
	UpsertUsers(ctx context.Context, users []domain.User) error
	SetUserActive(ctx context.Context, userID string, isActive bool) (domain.User, error)
	GetUser(ctx context.Context, userID string) (domain.User, error)
	ListActiveTeamMembers(ctx context.Context, teamName string, excludeIDs []string, limit int) ([]domain.User, error)

	CreatePR(ctx context.Context, pr domain.PullRequest, reviewers []string) error
	GetPR(ctx context.Context, prID string) (domain.PullRequest, error)
	UpdatePRStatusMerged(ctx context.Context, prID string, mergedAt *time.Time) (domain.PullRequest, error)
	ReplaceReviewer(ctx context.Context, prID string, oldUserID, newUserID string) (domain.PullRequest, error)
	ListPRsByReviewer(ctx context.Context, userID string) ([]domain.PullRequest, error)

	GetReviewerStats(ctx context.Context) (map[string]int, error)
	GetPRStats(ctx context.Context) (map[string]int, error)
}
