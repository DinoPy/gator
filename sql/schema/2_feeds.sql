-- +goose Up
CREATE TABLE feeds (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	url TEXT UNIQUE NOT NULL,
	user_id TEXT NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE feeds;
