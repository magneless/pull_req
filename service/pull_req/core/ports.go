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
	Create(ctx context.Context, prID, name, authorID string) (PullRequest, error)
	Merge(ctx context.Context, id string) (PullRequest, error)
	Reassign(ctx context.Context, prID, oldReviewerID string) (PullRequest, string, error)
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
	Get(ctx context.Context, id string) (PullRequest, error)
	GetActiveTeamMemberIDsByUserID(ctx context.Context, userID string) ([]string, error)
	Add(ctx context.Context, prID, name, authorID string, reviewersID []string) error
	UpdateMerged(ctx context.Context, id string) (PullRequest, error)
	UpdateReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) (PullRequest, error)
	GetByReviewer(ctx context.Context, reviewerID string) ([]PullRequestShort, error)
}
