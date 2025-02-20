-- +goose Up
CREATE TABLE users(
	id TEXT PRIMARY KEY,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	name TEXT UNIQUE NOT NULL
);

-- +goose Down
DROP TABLE users;


-- export PATH="/opt/homebrew/opt/postgresql@17/bin:$PATH"
