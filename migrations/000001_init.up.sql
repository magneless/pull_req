CREATE TABLE teams (
    name TEXT PRIMARY KEY
);

CREATE TABLE users (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    is_active BOOLEAN NOT NULL,
    team_name TEXT NOT NULL REFERENCES teams(name) ON DELETE CASCADE
);

CREATE TABLE prs (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    author_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status     TEXT NOT NULL CHECK (status IN ('OPEN','MERGED')),
    reviewers  TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    merged_at  TIMESTAMP
);
