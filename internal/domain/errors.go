package domain

import "errors"

var (
	ErrTeamExists  = errors.New("TEAM_EXISTS")
	ErrPrExists    = errors.New("PR_EXISTS")
	ErrPrMerged    = errors.New("PR_MERGED")
	ErrNotAssigned = errors.New("NOT_ASSIGNED")
	ErrNoCandidate = errors.New("NO_CANDIDATE")
	ErrNotFound    = errors.New("NOT_FOUND")
)
