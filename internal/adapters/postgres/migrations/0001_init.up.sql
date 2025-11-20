CREATE TABLE teams
(
    team_name TEXT PRIMARY KEY
);

CREATE TABLE users
(
    user_id   TEXT PRIMARY KEY,
    username  TEXT    NOT NULL,
    team_name TEXT    NOT NULL REFERENCES teams (team_name) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE pull_requests
(
    pull_request_id   TEXT PRIMARY KEY,
    pull_request_name TEXT                     NOT NULL,
    author_id         TEXT                     NOT NULL REFERENCES users (user_id),
    status            TEXT                     NOT NULL CHECK (status IN ('OPEN', 'MERGED')) DEFAULT 'OPEN',
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL                                      DEFAULT now(),
    merged_at         TIMESTAMP WITH TIME ZONE NULL
);

CREATE TABLE pr_reviewers
(
    pull_request_id TEXT REFERENCES pull_requests (pull_request_id) ON DELETE CASCADE,
    user_id         TEXT REFERENCES users (user_id) ON DELETE CASCADE,
    PRIMARY KEY (pull_request_id, user_id)
);

CREATE INDEX idx_users_team_active ON users (team_name, is_active);
CREATE INDEX idx_pr_reviewers_user ON pr_reviewers (user_id);
CREATE INDEX idx_pr_status ON pull_requests (status);
