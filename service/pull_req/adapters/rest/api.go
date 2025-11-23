package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"pull_req/pull_req/core"
	"time"
)

const (
	codeTeamExists  = "TEAM_EXISTS"
	codePrExists    = "PR_EXISTS"
	codePrMerged    = "PR_MERGED"
	codeNotAssigned = "NOT_ASSIGNED"
	codeNoCandidate = "NO_CANDIDATE"
	codeNotFound    = "NOT_FOUND"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSONError(w http.ResponseWriter, status int, code string, message string) error {
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{Code: code, Message: message},
	})
	return err
}

type TeamMember struct {
	ID       string `json:"user_id"`
	Name     string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	Name    string       `json:"team_name"`
	Members []TeamMember `json:"members"`
}

type TeamResponse struct {
	Team Team `json:"team"`
}

func NewAddTeamHandler(log *slog.Logger, t core.TeamPort) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var team Team

		err := json.NewDecoder(r.Body).Decode(&team)
		if err != nil {
			log.Error("decode body problem", "error", err)
			http.Error(w, "Bad request body", http.StatusBadRequest)
			return
		}

		members := make([]core.TeamMember, len(team.Members))
		for i, m := range team.Members {
			members[i].ID = m.ID
			members[i].Name = m.Name
			members[i].IsActive = m.IsActive
		}
		err = t.Create(r.Context(), core.Team{
			Name:    team.Name,
			Members: members,
		})
		if err != nil {
			if errors.Is(err, core.ErrAlreadyExists) {
				log.Error("team already exists", "error", err)
				err := writeJSONError(w, http.StatusBadRequest, codeTeamExists, "Team already exists")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			log.Error("create team problem", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}

		resp := TeamResponse{Team: team}

		w.WriteHeader(http.StatusCreated)
		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("encoding problem", "error", err)
		}
	}
}

func NewGetTeamHandler(log *slog.Logger, t core.TeamPort) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("team_name")
		if name == "" {
			log.Error("empty or missed team_name")
			http.Error(w, "team_name should not be empty", http.StatusBadRequest)
			return
		}

		team, err := t.Get(r.Context(), name)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				log.Error("team not found", "error", err)
				err := writeJSONError(w, http.StatusNotFound, codeNotFound, "Team not found")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			log.Error("get team problem", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		membersResp := make([]TeamMember, len(team.Members))
		for i, m := range team.Members {
			membersResp[i].ID = m.ID
			membersResp[i].Name = m.Name
			membersResp[i].IsActive = m.IsActive
		}
		teamResp := Team{
			Name:    team.Name,
			Members: membersResp,
		}
		resp := TeamResponse{
			Team: teamResp,
		}

		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("encoding problem", "error", err)
		}
	}
}

type SetIsActiveReq struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type User struct {
	ID       string `json:"user_id"`
	Name     string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type UserResponse struct {
	User User `json:"user"`
}

func NewSetIsActiveHandler(log *slog.Logger, u core.UserPort) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SetIsActiveReq

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Error("decode body problem", "error", err)
			http.Error(w, "Bad request body", http.StatusBadRequest)
			return
		}

		user, err := u.SetFlag(r.Context(), req.UserID, req.IsActive)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				log.Error("user not found", "error", err)
				err := writeJSONError(w, http.StatusNotFound, codeNotFound, "User not found")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			log.Error("internal error", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		userResp := User{
			ID:       user.ID,
			Name:     user.Name,
			TeamName: user.TeamName,
			IsActive: user.IsActive,
		}
		resp := UserResponse{
			User: userResp,
		}

		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("encoding problem", "error", err)
		}
	}
}

type CreatePRReq struct {
	PRID     string `json:"pull_request_id"`
	PRName   string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
}

type PullRequestShort struct {
	ID       string `json:"pull_request_id"`
	Name     string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
	Status   string `json:"status"`
}

type PullRequest struct {
	PullRequestShort
	Reviewers []string   `json:"assigned_reviewers"`
	MergedAt  *time.Time `json:"mergedAt"`
}

type PullRequestResponse struct {
	PullRequest PullRequest `json:"pull_request"`
}

func NewCreatePRHandler(log *slog.Logger, pr core.PRPort) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreatePRReq

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Error("decode body problem", "error", err)
			http.Error(w, "Bad request body", http.StatusBadRequest)
			return
		}

		pullReq, err := pr.Create(r.Context(), req.PRID, req.PRName, req.AuthorID)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				log.Error("Author/team not found", "error", err)
				err := writeJSONError(w, http.StatusNotFound, codeNotFound, "Author/team not found")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			if errors.Is(err, core.ErrAlreadyExists) {
				log.Error("pr exists", "error", err)
				err := writeJSONError(w, http.StatusConflict, codePrExists, "PR exists")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			log.Error("internal error", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		pullReqResp := PullRequest{
			PullRequestShort: PullRequestShort{
				ID:       pullReq.ID,
				Name:     pullReq.Name,
				AuthorID: pullReq.AuthorID,
				Status:   pullReq.Status,
			},
			Reviewers: pullReq.Reviewers,
		}
		resp := PullRequestResponse{
			PullRequest: pullReqResp,
		}

		w.WriteHeader(http.StatusCreated)
		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("encoding problem", "error", err)
		}
	}
}

type MergePRReq struct {
	PRID string `json:"pull_request_id"`
}

func NewMergePRHandler(log *slog.Logger, pr core.PRPort) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MergePRReq

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Error("decode body problem", "error", err)
			http.Error(w, "Bad request body", http.StatusBadRequest)
			return
		}

		pullReq, err := pr.Merge(r.Context(), req.PRID)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				log.Error("pr not found", "error", err)
				err := writeJSONError(w, http.StatusNotFound, codeNotFound, "PR not found")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			log.Error("internal error", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		pullReqResp := PullRequest{
			PullRequestShort: PullRequestShort{
				ID:       pullReq.ID,
				Name:     pullReq.Name,
				AuthorID: pullReq.AuthorID,
				Status:   pullReq.Status,
			},
			Reviewers: pullReq.Reviewers,
		}
		resp := PullRequestResponse{
			PullRequest: pullReqResp,
		}
		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("encoding problem", "error", err)
		}
	}
}

type ReassignPRReq struct {
	PRID      string `json:"pull_request_id"`
	OldUserID string `json:"old_user_id"`
}

type ReassignResponse struct {
	PR     PullRequest `json:"pr"`
	NewRev string      `json:"replaced_by"`
}

func NewReassignPRHandler(log *slog.Logger, pr core.PRPort) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReassignPRReq

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Error("decode body problem", "error", err)
			http.Error(w, "Bad request body", http.StatusBadRequest)
			return
		}

		pullReq, newRev, err := pr.Reassign(r.Context(), req.PRID, req.OldUserID)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				log.Error("pr or user not found", "error", err)
				err := writeJSONError(w, http.StatusNotFound, codeNotFound, "PR/user not found")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			if errors.Is(err, core.ErrAlredyMerged) {
				log.Error("pr merged", "error", err)
				err := writeJSONError(w, http.StatusConflict, codePrMerged, "PR merged")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			if errors.Is(err, core.ErrNotAssigned) {
				log.Error("not assigned", "error", err)
				err := writeJSONError(w, http.StatusConflict, codeNotAssigned, "Reviewer is not assigned to this PR")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			if errors.Is(err, core.ErrNoCandidate) {
				log.Error("no candidate", "error", err)
				err := writeJSONError(w, http.StatusConflict, codeNoCandidate, "No active replacement candidate in team")
				if err != nil {
					log.Error("write json error problem", "error", err)
				}
				return
			}
			log.Error("internal error", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		pullReqResp := PullRequest{
			PullRequestShort: PullRequestShort{
				ID:       pullReq.ID,
				Name:     pullReq.Name,
				AuthorID: pullReq.AuthorID,
				Status:   pullReq.Status,
			},
			Reviewers: pullReq.Reviewers,
		}
		resp := ReassignResponse{
			PR:     pullReqResp,
			NewRev: newRev,
		}
		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("encoding problem", "error", err)
		}
	}
}

type GetReviewResponse struct {
	UserID string             `json:"user_id"`
	PR     []PullRequestShort `json:"pull_requests"`
}

func NewGetReviewHandler(log *slog.Logger, pr core.PRPort) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			log.Error("empty or missed user_id")
			http.Error(w, "user_id should not be empty", http.StatusBadRequest)
			return
		}

		prs, err := pr.ListByReviewer(r.Context(), userID)
		if err != nil {
			log.Error("internal error", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		prsResp := make([]PullRequestShort, len(prs))
		for i, p := range prs {
			prsResp[i].ID = p.ID
			prsResp[i].Name = p.Name
			prsResp[i].AuthorID = p.AuthorID
			prsResp[i].Status = p.Status
		}
		resp := GetReviewResponse{
			UserID: userID,
			PR:     prsResp,
		}
		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("encoding problem", "error", err)
		}
	}
}
