package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"pull_req/pull_req/core"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	UniqueViolation     = "23505"
	ForeignKeyViolation = "23503"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func NewDB(log *slog.Logger, address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

type TeamDB struct {
	db *DB
}

func NewTeamDB(db *DB) *TeamDB {
	return &TeamDB{db}
}

type UserInsert struct {
	ID       string `db:"id"`
	Name     string `db:"name"`
	IsActive bool   `db:"is_active"`
	TeamName string `db:"team_name"`
}

func (t *TeamDB) Add(ctx context.Context, team core.Team) error {
	tx, err := t.db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO teams VALUES($1)`,
		team.Name,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == UniqueViolation {
			return core.ErrAlreadyExists
		}
		return err
	}

	if len(team.Members) > 0 {
		insertUsers := make([]UserInsert, len(team.Members))
		for i, member := range team.Members {
			insertUsers[i] = UserInsert{
				ID:       member.ID,
				Name:     member.Name,
				IsActive: member.IsActive,
				TeamName: team.Name,
			}
		}

		_, err = tx.NamedExecContext(
			ctx,
			`INSERT INTO users(id, name, is_active, team_name) 
		 	 VALUES (:id, :name, :is_active, :team_name)
		  	 ON CONFLICT (id) DO UPDATE 
    	 	 SET
			 	 name = EXCLUDED.name,
			 	 is_active = EXCLUDED.is_active,
				 team_name = EXCLUDED.team_name`,
			insertUsers,
		)
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

type Team struct {
	Name    string `db:"name"`
	Members []TeamMember
}

type TeamMember struct {
	ID       string `db:"id"`
	Name     string `db:"name"`
	IsActive bool   `db:"is_active"`
}

func (t *TeamDB) Get(ctx context.Context, name string) (core.Team, error) {
	var team Team

	err := t.db.conn.GetContext(
		ctx,
		&team,
		`SELECT name FROM teams WHERE name = $1`,
		name,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Team{}, core.ErrNotFound
		}
		return core.Team{}, err
	}

	var members []TeamMember
	err = t.db.conn.SelectContext(
		ctx,
		&members,
		`SELECT id, name, is_active
		 FROM users WHERE team_name = $1`,
		name,
	)
	if err != nil {
		return core.Team{}, err
	}

	coreMembers := make([]core.TeamMember, len(members))
	for i, m := range members {
		coreMembers[i] = core.TeamMember{
			ID:       m.ID,
			Name:     m.Name,
			IsActive: m.IsActive,
		}
	}

	return core.Team{
		Name:    team.Name,
		Members: coreMembers,
	}, nil
}

type UserDB struct {
	db *DB
}

func NewUserDB(db *DB) *UserDB {
	return &UserDB{db}
}

type User struct {
	ID       string `db:"id"`
	Name     string `db:"name"`
	TeamName string `db:"team_name"`
	IsActive bool   `db:"is_active"`
}

func (u *UserDB) UpdateIsActive(ctx context.Context, id string, isActive bool) (core.User, error) {
	var user User
	err := u.db.conn.GetContext(
		ctx,
		&user,
		`UPDATE users SET is_active = $1 WHERE id = $2
         RETURNING id, name, is_active, team_name`,
		isActive, id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.User{}, core.ErrNotFound
		}
		return core.User{}, err
	}
	return core.User{
		ID:       user.ID,
		Name:     user.Name,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}, nil
}

type PRDB struct {
	db *DB
}

func NewPRDB(db *DB) *PRDB {
	return &PRDB{db}
}
func (pr *PRDB) Get(ctx context.Context, id string) (core.PullRequest, error) {
    var pullReq PullRequest
    err := pr.db.conn.GetContext(
        ctx,
        &pullReq,
        `SELECT id, name, author_id, status, reviewers, merged_at FROM prs WHERE id = $1`,
        id,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return core.PullRequest{}, core.ErrNotFound
        }
        return core.PullRequest{}, err
    }
    return core.PullRequest{
        PullRequestShort: core.PullRequestShort{
            ID:       pullReq.ID,
            Name:     pullReq.Name,
            AuthorID: pullReq.AuthorID,
            Status:   pullReq.Status,
        },
        Reviewers: pullReq.Reviewers,
        MergedAt:  pullReq.MergedAt,
    }, nil
}
func (pr *PRDB) GetActiveTeamMemberIDsByUserID(ctx context.Context, authorID string) ([]string, error) {
	var teamName string
	err := pr.db.conn.GetContext(
		ctx,
		&teamName,
		`SELECT team_name FROM users
		 WHERE id = $1`,
		authorID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	var ids []string
	err = pr.db.conn.SelectContext(
		ctx,
		&ids,
		`SELECT id FROM users
		 WHERE team_name = $1
		 AND is_active = TRUE`,
		teamName,
	)
	if err != nil {
		return nil, err
	}

	return ids, nil
}
func (pr *PRDB) Add(ctx context.Context, prID, name, authorID string, reviewersID []string) error {
	if reviewersID == nil {
		reviewersID = []string{}
	}
	_, err := pr.db.conn.ExecContext(
		ctx,
		`INSERT INTO prs (id, name, author_id, status, reviewers)
		 VALUES ($1, $2, $3, 'OPEN', $4)`,
		prID, name, authorID, reviewersID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation {
				return core.ErrAlreadyExists
			}
			if pgErr.Code == ForeignKeyViolation {
				return core.ErrNotFound
			}
		}
		return err
	}
	return nil
}

type PullRequest struct {
	ID        string         `db:"id"`
	Name      string         `db:"name"`
	AuthorID  string         `db:"author_id"`
	Status    string         `db:"status"`
	Reviewers pq.StringArray `db:"reviewers"`
	MergedAt  *time.Time     `db:"merged_at"`
}

func (pr *PRDB) UpdateMerged(ctx context.Context, id string) (core.PullRequest, error) {
	var pullReq PullRequest
	err := pr.db.conn.GetContext(
		ctx,
		&pullReq,
		`UPDATE prs SET status = 'MERGED', merged_at = COALESCE(merged_at, NOW())
		 WHERE id = $1
		 RETURNING id, name, author_id, status, reviewers, merged_at`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.PullRequest{}, core.ErrNotFound
		}
		return core.PullRequest{}, err
	}
	return core.PullRequest{
		PullRequestShort: core.PullRequestShort{
			ID:       pullReq.ID,
			Name:     pullReq.Name,
			AuthorID: pullReq.AuthorID,
			Status:   pullReq.Status,
		},
		Reviewers: pullReq.Reviewers,
		MergedAt:  pullReq.MergedAt,
	}, nil
}
func (pr *PRDB) UpdateReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) (core.PullRequest, error) {
	var current struct {
		Status    string         `db:"status"`
		Reviewers pq.StringArray `db:"reviewers"`
	}

	err := pr.db.conn.GetContext(
		ctx,
		&current,
		`SELECT status, reviewers FROM prs WHERE id = $1`,
		prID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.PullRequest{}, core.ErrNotFound
		}
		return core.PullRequest{}, err
	}

	if current.Status == "MERGED" {
		return core.PullRequest{}, core.ErrAlredyMerged
	}
	if !slices.Contains(current.Reviewers, oldReviewerID) {
		return core.PullRequest{}, core.ErrNotAssigned
	}
	var updatedPR PullRequest
	err = pr.db.conn.GetContext(
		ctx,
		&updatedPR,
		`UPDATE prs 
		 SET reviewers = array_replace(reviewers, $2, $3)
		 WHERE id = $1
		 RETURNING id, name, author_id, status, reviewers, merged_at`,
		prID,
		oldReviewerID,
		newReviewerID,
	)
	if err != nil {
		return core.PullRequest{}, err
	}

	return core.PullRequest{
		PullRequestShort: core.PullRequestShort{
			ID:       updatedPR.ID,
			Name:     updatedPR.Name,
			AuthorID: updatedPR.AuthorID,
			Status:   updatedPR.Status,
		},
		Reviewers: updatedPR.Reviewers,
		MergedAt:  updatedPR.MergedAt,
	}, nil
}

type PullRequestShort struct {
	ID       string `db:"id"`
	Name     string `db:"name"`
	AuthorID string `db:"author_id"`
	Status   string `db:"status"`
}

func (pr *PRDB) GetByReviewer(ctx context.Context, reviewerID string) ([]core.PullRequestShort, error) {
	var prs []PullRequest

	err := pr.db.conn.SelectContext(
		ctx,
		&prs,
		`SELECT id, name, author_id, status
         FROM prs
         WHERE $1 = ANY(reviewers)`,
		reviewerID,
	)
	if err != nil {
		return nil, err
	}

	result := make([]core.PullRequestShort, len(prs))
	for i, d := range prs {
		result[i] = core.PullRequestShort{
			ID:       d.ID,
			Name:     d.Name,
			AuthorID: d.AuthorID,
			Status:   d.Status,
		}
	}

	return result, nil
}
