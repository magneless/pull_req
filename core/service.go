package core

import (
	"context"
	"log/slog"
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

func NewPRService(log *slog.Logger, db PRDB) *PRService {
	return &PRService{
		log: log,
		db:  db,
	}
}
func (pr *PRService) Create(ctx context.Context, pqID, name, authorID string) (PullRequest, error) {
	pullReq, err := pr.db.Add(ctx, pqID, name, authorID)
	if err != nil {
		pr.log.Error("failed to create pr", "error", err)
		return PullRequest{}, err
	}
	return pullReq, nil
}
func (pr *PRService) Merge(ctx context.Context, id string) (PullRequest, error) {
	pullReq, err := pr.db.UpdateMerged(ctx, id)
	if err != nil {
		pr.log.Error("failed to merge pr", "error", err)
		return PullRequest{}, err
	}
	return pullReq, nil
}
func (pr *PRService) Reassign(ctx context.Context, pqID, oldReviewerID string) (PullRequest, error) {
	pullReq, err := pr.db.UpdateReviewer(ctx, pqID, oldReviewerID)
	if err != nil {
		pr.log.Error("failed to reassign pr", "error", err)
		return PullRequest{}, err
	}
	return pullReq, nil
}
func (pr *PRService) ListByReviewer(ctx context.Context, reviewerID string) ([]PullRequestShort, error) {
	pullReqs, err := pr.db.GetByReviewer(ctx, reviewerID)
	if err != nil {
		pr.log.Error("failed to get list by reviewer", "error", err)
		return nil, err
	}
	return pullReqs, nil
}
