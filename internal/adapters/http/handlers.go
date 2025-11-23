package http

import (
	"PRService/internal/domain"
	"PRService/internal/services"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	_ "github.com/go-chi/chi/v5"
)

type Handler struct {
	S *services.Service
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var team domain.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.S.CreateTeam(r.Context(), team); err != nil {
		if errors.Is(err, domain.ErrTeamExists) {
			http.Error(w, "TEAM_EXISTS", http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err := json.NewEncoder(w).Encode(map[string]domain.Team{"team": team})
	if err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		http.Error(w, "team_name required", http.StatusBadRequest)
		return
	}

	team, err := h.S.GetTeam(r.Context(), teamName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "TEAM_NOT_FOUND", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(team)
	if err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := h.S.SetActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "USER_NOT_FOUND", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(map[string]domain.User{"user": user})
	if err != nil {
		log.Printf("error encoding user: %v", err)
	}
}

func (h *Handler) GetUserPRs(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	prs, err := h.S.GetPRsForReviewer(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Printf("error getting PRs for reviewer: %v", err)
		return
	}

	if prs == nil {
		http.Error(w, "PR_NOT_FOUND", http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Printf("error writing response: %v", err)
	}
}

func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	pr := domain.PullRequest{
		PullRequestID:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
	}

	pr, err := h.S.CreatePR(r.Context(), pr)
	if err != nil {
		if errors.Is(err, domain.ErrPrExists) {
			http.Error(w, "PR_EXISTS", http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "AUTHOR_OR_TEAM_NOT_FOUND", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Printf("error creating PR: %v", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(map[string]domain.PullRequest{"pr": pr})
	if err != nil {
		log.Printf("error encoding PR: %v", err)
	}
}

func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	pr, err := h.S.MergePR(r.Context(), req.PullRequestID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "PR_NOT_FOUND", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(map[string]domain.PullRequest{"pr": pr})
	if err != nil {
		log.Printf("error encoding PR: %v", err)
	}
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_reviewer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	pr, replacedBy, err := h.S.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		if errors.Is(err, domain.ErrPrMerged) {
			http.Error(w, "PR_MERGED", http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrNotAssigned) {
			http.Error(w, "NOT_ASSIGNED", http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrNoCandidate) {
			http.Error(w, "NO_CANDIDATE", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"pr":          pr,
		"replaced_by": replacedBy,
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Printf("error reassigning reviewer: %v", err)
	}
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.S.GetStats(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(stats)
	if err != nil {
		log.Printf("error encoding stats: %v", err)
	}
}

func (h *Handler) DeactivateUsersHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(req.UserIDs) == 0 {
		http.Error(w, "user_ids required", http.StatusBadRequest)
		return
	}

	results, err := h.S.DeactivateUsers(r.Context(), req.UserIDs)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"results": results,
	}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Printf("error encoding results: %v", err)
	}
}
