package core

import "time"

type TeamMember struct {
	ID       string
	Name     string
	IsActive bool
}

type Team struct {
	Name    string
	Members []TeamMember
}

type User struct {
	ID       string
	Name     string
	TeamName string
	IsActive bool
}

type PullRequestShort struct {
	ID       string
	Name     string
	AuthorID string
	Status   string
}

type PullRequest struct {
	PullRequestShort
	Reviewers []string
	MergedAt  *time.Time
}
