package core

import "context"

type TeamPort interface {
	Create(context.Context, Team) error
	Get(ctx context.Context, name string) (Team, error)
}

type UserPort interface {
	SetFlag(ctx context.Context, id string, isActive bool) (User, error)
}

type PRPort interface {
	Create(ctx context.Context, pqID, name, authorID string) (PullRequest, error)
	Merge(ctx context.Context, id string) (PullRequest, error)
	Reassign(ctx context.Context, pqID, oldReviewerID string) (PullRequest, error)
	ListByReviewer(ctx context.Context, reviewerID string) ([]PullRequestShort, error)
}

type TeamDB interface {
	Add(context.Context, Team) error
	Get(ctx context.Context, name string) (Team, error)
}

type UserDB interface {
	UpdateIsActive(ctx context.Context, id string, isActive bool) (User, error)
}

type PRDB interface {
	Add(ctx context.Context, pqID, name, authorID string) (PullRequest, error)
	UpdateMerged(ctx context.Context, id string) (PullRequest, error)
	UpdateReviewer(ctx context.Context, pqID, oldReviewerID string) (PullRequest, error)
	GetByReviewer(ctx context.Context, reviewerID string) ([]PullRequestShort, error)
}