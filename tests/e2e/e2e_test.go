package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
)

func postJSON(t *testing.T, url string, payload interface{}) *http.Response {
	data, _ := json.Marshal(payload)
	resp, err := http.Post(server.URL+url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	return resp
}

func getJSON(t *testing.T, url string) *http.Response {
	resp, err := http.Get(server.URL + url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) []byte {
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("failed to close body: %v", err)
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func createTeam(t *testing.T) (string, []string) {
	teamName := uniqueName("team")
	user1 := uniqueName("u")
	user2 := uniqueName("u")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": user1, "username": "Alice", "is_active": true},
			{"user_id": user2, "username": "Bob", "is_active": true},
		},
	}
	resp := postJSON(t, "/team/add", team)
	if resp.StatusCode != 201 {
		t.Fatalf("failed to create team: %d", resp.StatusCode)
	}
	return teamName, []string{user1, user2}
}

func createPR(t *testing.T, prID, authorID string) {
	pr := map[string]string{
		"pull_request_id":   prID,
		"pull_request_name": fmt.Sprintf("PR %s", prID),
		"author_id":         authorID,
	}
	resp := postJSON(t, "/pullRequest/create", pr)
	if resp.StatusCode != 201 {
		t.Fatalf("failed to create PR %s: %d", prID, resp.StatusCode)
	}
}

func createPRWithReviewer(t *testing.T, prID, authorID, reviewerID string) {
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": fmt.Sprintf("PR %s", prID),
		"author_id":         authorID,
		"reviewers":         []string{reviewerID},
	}
	resp := postJSON(t, "/pullRequest/create", pr)
	if resp.StatusCode != 201 {
		t.Fatalf("failed to create PR %s: %d", prID, resp.StatusCode)
	}
}

func TestCreateTeam(t *testing.T) {
	teamName, members := createTeam(t)
	resp := getJSON(t, fmt.Sprintf("/team/get?team_name=%s", teamName))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	for _, m := range members {
		if !bytes.Contains(body, []byte(m)) {
			t.Fatalf("team member %s not found in response", m)
		}
	}
}

func TestSetUserActive(t *testing.T) {
	_, members := createTeam(t)
	req := map[string]interface{}{
		"user_id":   members[1],
		"is_active": false,
	}
	resp := postJSON(t, "/users/setIsActive", req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !bytes.Contains(body, []byte(`"is_active":false`)) {
		t.Fatal("user active flag was not updated")
	}
}

func TestStats(t *testing.T) {
	teamName := uniqueName("team")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": "u100", "username": "Alice", "is_active": true},
			{"user_id": "u101", "username": "Bob", "is_active": true},
		},
	}
	resp := postJSON(t, "/team/add", team)
	if resp.StatusCode != 201 {
		t.Fatalf("failed to create team: %d", resp.StatusCode)
	}

	prID := uniqueName("pr")
	pr := map[string]string{
		"pull_request_id":   prID,
		"pull_request_name": "Test PR",
		"author_id":         "u100",
	}
	resp = postJSON(t, "/pullRequest/create", pr)
	if resp.StatusCode != 201 {
		t.Fatalf("failed to create PR: %d", resp.StatusCode)
	}

	resp = getJSON(t, "/stats")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !bytes.Contains(body, []byte(prID)) {
		t.Fatal("stats response does not contain expected PR")
	}
}

func TestGetUserPRs(t *testing.T) {
	_, members := createTeam(t)
	prID := uniqueName("pr")
	createPRWithReviewer(t, prID, members[0], members[1]) // второй участник — ревьювер

	resp := getJSON(t, fmt.Sprintf("/users/getReview?user_id=%s", members[1]))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !bytes.Contains(body, []byte(prID)) {
		t.Fatal("PR not found in user's review list")
	}
}

func TestCreatePR(t *testing.T) {
	_, members := createTeam(t)
	prID := uniqueName("pr_test")
	createPR(t, prID, members[0])
	resp := getJSON(t, fmt.Sprintf("/users/getReview?user_id=%s", members[1]))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !bytes.Contains(body, []byte(prID)) {
		t.Fatal("PR not created correctly")
	}
}

func TestMergePR(t *testing.T) {
	_, members := createTeam(t)
	prID := uniqueName("pr_merge")
	createPR(t, prID, members[0])
	req := map[string]string{"pull_request_id": prID}
	resp := postJSON(t, "/pullRequest/merge", req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !bytes.Contains(body, []byte("MERGED")) {
		t.Fatal("PR was not merged")
	}
}

func TestReassignReviewer(t *testing.T) {
	teamName := uniqueName("team")
	team := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": "u200", "username": "Charlie", "is_active": true},
			{"user_id": "u201", "username": "Dana", "is_active": true},
			{"user_id": "u202", "username": "Eve", "is_active": true},
			{"user_id": "u203", "username": "Frank", "is_active": true},
		},
	}
	resp := postJSON(t, "/team/add", team)
	if resp.StatusCode != 201 {
		t.Fatalf("failed to create team: %d", resp.StatusCode)
	}

	prID := uniqueName("pr")
	pr := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Test PR",
		"author_id":         "u200",
	}
	resp = postJSON(t, "/pullRequest/create", pr)
	if resp.StatusCode != 201 {
		t.Fatalf("failed to create PR: %d", resp.StatusCode)
	}

	reassignReq := map[string]string{
		"pull_request_id": prID,
		"old_reviewer_id": "u201",
	}
	resp = postJSON(t, "/pullRequest/reassign", reassignReq)
	if resp.StatusCode != 200 {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d, body: %s", resp.StatusCode, string(body))
	}

	body := readBody(t, resp)
	if !bytes.Contains(body, []byte("replaced_by")) {
		t.Fatal("response does not contain replaced_by field")
	}
	if !bytes.Contains(body, []byte("u203")) {
		t.Fatal("reviewer was not replaced with expected candidate u203")
	}
}
