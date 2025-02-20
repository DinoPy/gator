-- name: CreatePost :one
INSERT INTO posts (
	id,
	created_at,
	updated_at,
	title,
	url,
	description,
	published_at,
	feed_id
) VALUES (
	$1,
	$2,
	$3,
	$4,
	$5, 
	$6,
	$7,
	$8
) RETURNING *;

-- name: GetPostsForUser :many
SELECT p.*
FROM posts p
INNER JOIN feeds_follows ff ON p.feed_id = ff.feed_id
INNER JOIN users u ON u.id = ff.user_id
WHERE u.id = $1
ORDER BY published_at ASC
LIMIT $2;
