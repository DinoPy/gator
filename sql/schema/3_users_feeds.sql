-- +goose Up
CREATE TABLE feeds_follows (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	feed_id TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE,
	UNIQUE(user_id, feed_id)
);

-- +goose Down
DROP TABLE feeds_follows;
