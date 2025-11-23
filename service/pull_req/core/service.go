package core

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"slices"
)

type TeamService struct {
	log *slog.Logger
	db  TeamDB
}

func NewTeamService(log *slog.Logger, db TeamDB) *TeamService {
	return &TeamService{
		log: log,
		db:  db,
	}
}
func (t *TeamService) Create(ctx context.Context, team Team) error {
	err := t.db.Add(ctx, team)
	if err != nil {
		t.log.Error("failed to create team", "error", err)
		return err
	}
	return nil
}
func (t *TeamService) Get(ctx context.Context, name string) (Team, error) {
	team, err := t.db.Get(ctx, name)
	if err != nil {
		t.log.Error("failed to get team", "error", err)
		return Team{}, err
	}
	return team, nil
}

type UserService struct {
	log *slog.Logger
	db  UserDB
}

func NewUserService(log *slog.Logger, db UserDB) *UserService {
	return &UserService{
		log: log,
		db:  db,
	}
}
func (u *UserService) SetFlag(ctx context.Context, id string, isActive bool) (User, error) {
	user, err := u.db.UpdateIsActive(ctx, id, isActive)
	if err != nil {
		u.log.Error("failed to set flag", "error", err)
		return User{}, err
	}
	return user, nil
}

type PRService struct {
	log *slog.Logger
	db  PRDB
}

func getRandomStrings(src []string, n int) []string {
	if len(src) == 0 {
		return nil
	}

	rand.Shuffle(len(src), func(i, j int) {
		src[i], src[j] = src[j], src[i]
	})

	if len(src) < n {
		n = len(src)
	}

	return src[:n]
}
func removeByValue(src []string, val string) []string {
	result := src[:0]
	for _, v := range src {
		if v != val {
			result = append(result, v)
		}
	}

	return result
}
func NewPRService(log *slog.Logger, db PRDB) *PRService {
	return &PRService{
		log: log,
		db:  db,
	}
}
func (pr *PRService) Create(ctx context.Context, prID, name, authorID string) (PullRequest, error) {
	teamMembersIDs, err := pr.db.GetActiveTeamMemberIDsByUserID(ctx, authorID)
	if err != nil {
		pr.log.Error("failed to get reviewers", "error", err)
		return PullRequest{}, err
	}

	teamMembersIDs = removeByValue(teamMembersIDs, authorID)
	reviewers := getRandomStrings(teamMembersIDs, 2)

	err = pr.db.Add(ctx, prID, name, authorID, reviewers)
	if err != nil {
		pr.log.Error("failed to create pr", "error", err)
		return PullRequest{}, err
	}
	return PullRequest{
		PullRequestShort: PullRequestShort{
			ID:       prID,
			Name:     name,
			AuthorID: authorID,
			Status:   "OPEN",
		},
		Reviewers: reviewers,
	}, nil
}
func (pr *PRService) Merge(ctx context.Context, id string) (PullRequest, error) {
	pullReq, err := pr.db.UpdateMerged(ctx, id)
	if err != nil {
		pr.log.Error("failed to merge pr", "error", err)
		return PullRequest{}, err
	}
	return pullReq, nil
}
func (pr *PRService) Reassign(ctx context.Context, prID, oldReviewerID string) (PullRequest, string, error) {
	currentPR, err := pr.db.Get(ctx, prID)
	if err != nil {
		pr.log.Error("failed to get pr", "error", err)
		return PullRequest{}, "", err
	}
	if currentPR.Status == "MERGED" {
		pr.log.Error("pr already merged", "error", err)
		return PullRequest{}, "", ErrAlredyMerged
	}
	isAssigned := slices.Contains(currentPR.Reviewers, oldReviewerID)
	if !isAssigned {
		return PullRequest{}, "", ErrNotAssigned
	}

	teamMembersIDs, err := pr.db.GetActiveTeamMemberIDsByUserID(ctx, oldReviewerID)
	if err != nil {
		pr.log.Error("failed to get reviewers", "error", err)
		return PullRequest{}, "", err
	}

	for _, revID := range currentPR.Reviewers {
		teamMembersIDs = removeByValue(teamMembersIDs, revID)
	}
	teamMembersIDs = removeByValue(teamMembersIDs, currentPR.AuthorID)

	if len(teamMembersIDs) < 1 {
		pr.log.Error("there is no candidates", "error", ErrNoCandidate)
		return PullRequest{}, "", ErrNoCandidate
	}
	newReviewerID := getRandomStrings(teamMembersIDs, 1)[0]

	pullReq, err := pr.db.UpdateReviewer(ctx, prID, oldReviewerID, newReviewerID)
	if err != nil {
		pr.log.Error("failed to reassign pr", "error", err)
		return PullRequest{}, "", err
	}
	return pullReq, newReviewerID, nil
}
func (pr *PRService) ListByReviewer(ctx context.Context, reviewerID string) ([]PullRequestShort, error) {
	pullReqs, err := pr.db.GetByReviewer(ctx, reviewerID)
	if err != nil {
		pr.log.Error("failed to get list by reviewer", "error", err)
		return nil, err
	}
	return pullReqs, nil
}
