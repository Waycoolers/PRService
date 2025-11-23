package postgres

import (
	"PRService/internal/domain"
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Repo struct {
	db *sqlx.DB
}

func NewPostgresRepo(dbURL string) *Repo {
	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}

	return &Repo{db: db}
}
func (r *Repo) Close() error {
	return r.db.Close()
}

func (r *Repo) CreateTeam(ctx context.Context, team domain.Team) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO teams (team_name) VALUES ($1)`,
		team.TeamName,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrTeamExists
		}
		return err
	}
	return nil
}

func (r *Repo) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	var t domain.Team
	t.TeamName = teamName

	var members []domain.TeamMember
	err := r.db.SelectContext(ctx, &members,
		`SELECT user_id, username, is_active 
		 FROM users WHERE team_name = $1`,
		teamName,
	)

	if err != nil {
		return domain.Team{}, err
	}
	if len(members) == 0 {
		return domain.Team{}, domain.ErrNotFound
	}

	t.Members = members
	return t, nil
}

func (r *Repo) UpsertUsers(ctx context.Context, users []domain.User) error {
	if len(users) == 0 {
		return nil
	}

	query := `
	INSERT INTO users (user_id, username, team_name, is_active)
	VALUES (:user_id, :username, :team_name, :is_active)
	ON CONFLICT (user_id) DO UPDATE SET
	    username = EXCLUDED.username,
	    team_name = EXCLUDED.team_name,
	    is_active = EXCLUDED.is_active`

	q, args, err := sqlx.Named(query, users)
	if err != nil {
		return err
	}

	q, args, err = sqlx.In(q, args...)
	if err != nil {
		return err
	}

	q = r.db.Rebind(q)

	_, err = r.db.ExecContext(ctx, q, args...)
	return err
}

func (r *Repo) SetUserActive(ctx context.Context, userID string, isActive bool) (domain.User, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET is_active=$1 WHERE user_id=$2`,
		isActive, userID,
	)
	if err != nil {
		return domain.User{}, err
	}

	return r.GetUser(ctx, userID)
}

func (r *Repo) GetUser(ctx context.Context, userID string) (domain.User, error) {
	var u domain.User
	err := r.db.GetContext(ctx, &u,
		`SELECT user_id, username, team_name, is_active
		 FROM users WHERE user_id=$1`,
		userID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}

	return u, nil
}

func (r *Repo) ListActiveTeamMembers(ctx context.Context, teamName string, excludeIDs []string, limit int) ([]domain.User, error) {
	baseQuery := `
	SELECT user_id, username, team_name, is_active
	FROM users
	WHERE team_name = ? AND is_active = true
	`

	if len(excludeIDs) > 0 {
		baseQuery += " AND user_id NOT IN (?)"
	}

	baseQuery += " LIMIT ?"

	args := []interface{}{teamName}
	if len(excludeIDs) > 0 {
		args = append(args, excludeIDs)
	}
	args = append(args, limit)

	query, args, err := sqlx.In(baseQuery, args...)
	if err != nil {
		return nil, err
	}

	query = r.db.Rebind(query)

	var users []domain.User
	if err := r.db.SelectContext(ctx, &users, query, args...); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Repo) CreatePR(ctx context.Context, pr domain.PullRequest, reviewers []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO pull_requests 
		    (pull_request_id, pull_request_name, author_id, status, created_at, merged_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		pr.PullRequestID,
		pr.PullRequestName,
		pr.AuthorID,
		pr.Status,
		pr.CreatedAt,
		pr.MergedAt,
	)
	if err != nil {
		_ = tx.Rollback()
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrPrExists
		}
		return err
	}

	for _, reviewer := range reviewers {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`,
			pr.PullRequestID, reviewer,
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (r *Repo) GetPR(ctx context.Context, prID string) (domain.PullRequest, error) {
	var pr domain.PullRequest

	err := r.db.GetContext(ctx, &pr,
		`SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
	     FROM pull_requests WHERE pull_request_id=$1`,
		prID,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PullRequest{}, domain.ErrNotFound
		}
		return domain.PullRequest{}, err
	}

	var reviewers []string
	err = r.db.SelectContext(ctx, &reviewers,
		`SELECT user_id FROM pr_reviewers WHERE pull_request_id=$1`,
		prID,
	)
	if err != nil {
		return domain.PullRequest{}, err
	}

	pr.AssignedReviewers = reviewers
	return pr, nil
}

func (r *Repo) UpdatePRStatusMerged(ctx context.Context, prID string, mergedAt *time.Time) (domain.PullRequest, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pull_requests
		 SET status='MERGED', merged_at=$1
		 WHERE pull_request_id=$2`,
		mergedAt, prID,
	)
	if err != nil {
		return domain.PullRequest{}, err
	}

	return r.GetPR(ctx, prID)
}

func (r *Repo) ReplaceReviewer(ctx context.Context, prID, oldUserID, newUserID string) (domain.PullRequest, error) {
	log.Printf("Replacing reviewer: PR=%s, oldUser=%s, newUser=%s", prID, oldUserID, newUserID)

	res, err := r.db.ExecContext(ctx,
		`UPDATE pr_reviewers 
		 SET user_id = $1 
		 WHERE pull_request_id = $2 AND user_id = $3`,
		newUserID, prID, oldUserID,
	)
	if err != nil {
		return domain.PullRequest{}, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return domain.PullRequest{}, err
	}
	if n == 0 {
		return domain.PullRequest{}, domain.ErrNotAssigned
	}

	return r.GetPR(ctx, prID)
}

func (r *Repo) ListPRsByReviewer(ctx context.Context, userID string) ([]domain.PullRequest, error) {
	var prs []domain.PullRequest

	err := r.db.SelectContext(ctx, &prs,
		`SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status,
		        pr.created_at, pr.merged_at
		 FROM pull_requests pr
		 JOIN pr_reviewers r ON pr.pull_request_id = r.pull_request_id
		 WHERE r.user_id=$1`,
		userID,
	)

	if err != nil {
		return nil, err
	}

	for i := range prs {
		var reviewers []string
		err := r.db.SelectContext(ctx, &reviewers,
			`SELECT user_id FROM pr_reviewers WHERE pull_request_id=$1`,
			prs[i].PullRequestID,
		)
		if err != nil {
			return nil, err
		}
		prs[i].AssignedReviewers = reviewers
	}

	return prs, nil
}

func (r *Repo) GetReviewerStats(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, COUNT(*) AS assignments
		FROM pr_reviewers
		GROUP BY user_id
	`)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}(rows)

	stats := make(map[string]int)
	for rows.Next() {
		var userID string
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, err
		}
		stats[userID] = count
	}
	return stats, nil
}

func (r *Repo) GetPRStats(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT pull_request_id, COUNT(*) AS reviewers
		FROM pr_reviewers
		GROUP BY pull_request_id
	`)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}(rows)

	stats := make(map[string]int)
	for rows.Next() {
		var prID string
		var count int
		if err := rows.Scan(&prID, &count); err != nil {
			return nil, err
		}
		stats[prID] = count
	}
	return stats, nil
}
