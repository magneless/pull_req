package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const baseURL = "http://localhost:8080"

var client = &http.Client{
	Timeout: 5 * time.Second,
}


type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type PullRequestReq struct {
	PRID     string `json:"pull_request_id"`
	PRName   string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
}

type PullRequestResp struct {
	PR struct {
		ID        string   `json:"pull_request_id"`
		Status    string   `json:"status"`
		Reviewers []string `json:"assigned_reviewers"`
	} `json:"pull_request"`
}

type ReassignReq struct {
	PRID      string `json:"pull_request_id"`
	OldUserID string `json:"old_user_id"`
}

type ReassignResp struct {
	ReplacedBy string `json:"replaced_by"`
	PR         struct {
		Reviewers []string `json:"assigned_reviewers"`
	} `json:"pr"`
}

type MergeReq struct {
	PRID string `json:"pull_request_id"`
}

func TestHappyPath_CreatePR(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	suffix := fmt.Sprintf("%d", rnd.Int())
	
	teamName := "team_alpha_" + suffix
	author := "u_author_" + suffix
	user1 := "u_rev1_" + suffix
	user2 := "u_rev2_" + suffix
	
	team := Team{
		TeamName: teamName,
		Members: []TeamMember{
			{UserID: author, Username: "Author", IsActive: true},
			{UserID: user1, Username: "Reviewer1", IsActive: true},
			{UserID: user2, Username: "Reviewer2", IsActive: true},
		},
	}
	createTeam(t, team)

	prID := "pr_100_" + suffix
	prReq := PullRequestReq{
		PRID:     prID,
		PRName:   "Feature X",
		AuthorID: author,
	}
	
	prResp := createPR(t, prReq)

	require.Equal(t, "OPEN", prResp.PR.Status)
	require.Len(t, prResp.PR.Reviewers, 2, "Should assign exactly 2 reviewers")
	require.Contains(t, prResp.PR.Reviewers, user1)
	require.Contains(t, prResp.PR.Reviewers, user2)
	require.NotContains(t, prResp.PR.Reviewers, author, "Author should not be a reviewer")
}

func TestReassignReviewer(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	suffix := fmt.Sprintf("%d", rnd.Int())

	teamName := "team_reassign_" + suffix
	users := []string{"u1_" + suffix, "u2_" + suffix, "u3_" + suffix, "u4_" + suffix}
	
	members := make([]TeamMember, len(users))
	for i, u := range users {
		members[i] = TeamMember{UserID: u, Username: "User", IsActive: true}
	}

	createTeam(t, Team{TeamName: teamName, Members: members})

	prID := "pr_reassign_" + suffix
	prResp := createPR(t, PullRequestReq{
		PRID:     prID,
		PRName:   "Fix Bug",
		AuthorID: users[0],
	})

	require.Len(t, prResp.PR.Reviewers, 2)
	
	oldReviewer := prResp.PR.Reviewers[0]
	
	reassignResp := reassignPR(t, ReassignReq{
		PRID:      prID,
		OldUserID: oldReviewer,
	})

	require.NotEqual(t, oldReviewer, reassignResp.ReplacedBy, "New reviewer must be different")
	require.NotContains(t, reassignResp.PR.Reviewers, oldReviewer, "Old reviewer must be removed")
	require.Contains(t, reassignResp.PR.Reviewers, reassignResp.ReplacedBy, "New reviewer must be in list")
	require.NotContains(t, reassignResp.PR.Reviewers, users[0], "Author must not be assigned")
}

func TestMergeLifecycle(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	suffix := fmt.Sprintf("%d", rnd.Int())

	teamName := "team_merge_" + suffix
	u1, u2, u3 := "m1_"+suffix, "m2_"+suffix, "m3_"+suffix
	createTeam(t, Team{
		TeamName: teamName, 
		Members: []TeamMember{
			{UserID: u1, IsActive: true, Username: "A"}, 
			{UserID: u2, IsActive: true, Username: "B"},
			{UserID: u3, IsActive: true, Username: "C"},
		},
	})

	prID := "pr_merge_" + suffix
	createPR(t, PullRequestReq{PRID: prID, PRName: "Final", AuthorID: u1})

	mergePR(t, prID)
	resp, status := sendMergeRequest(t, prID)
	require.Equal(t, http.StatusOK, status, "Second merge should be OK (idempotent)")
	require.Equal(t, "MERGED", resp.PR.Status)

	reassignPayload := ReassignReq{PRID: prID, OldUserID: u2}
	body, _ := json.Marshal(reassignPayload)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/pullRequest/reassign", bytes.NewBuffer(body))
	
	httpResp, err := client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	require.Equal(t, http.StatusConflict, httpResp.StatusCode, "Should not allow reassign on merged PR")
}

func createTeam(t *testing.T, team Team) {
	body, err := json.Marshal(team)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/team/add", bytes.NewBuffer(body))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK, 
		"Failed to create team, status: %d", resp.StatusCode)
}

func createPR(t *testing.T, payload PullRequestReq) PullRequestResp {
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/pullRequest/create", bytes.NewBuffer(body))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result PullRequestResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return result
}

func reassignPR(t *testing.T, payload ReassignReq) ReassignResp {
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/pullRequest/reassign", bytes.NewBuffer(body))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result ReassignResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return result
}

func mergePR(t *testing.T, prID string) PullRequestResp {
	result, status := sendMergeRequest(t, prID)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "MERGED", result.PR.Status)
	return result
}

func sendMergeRequest(t *testing.T, prID string) (PullRequestResp, int) {
	payload := MergeReq{PRID: prID}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/pullRequest/merge", bytes.NewBuffer(body))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result PullRequestResp
	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
	}
	return result, resp.StatusCode
}