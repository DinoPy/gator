-- name: CreateFeed :one
INSERT INTO feeds (
	id,
	name,
	url,
	user_id
) VALUES (
	$1, 
	$2,
	$3,
	$4
) RETURNING *;

-- name: GetFeeds :many
SELECT 
	f.name,
	f.url,
	u.name AS username
FROM feeds f
JOIN users u ON f.user_id = u.id;

-- name: GetIdFeedByUrl :one
SELECT 
	id
FROM 
	feeds
WHERE 
	url = $1;

-- name: MarkFeedFetched :one 
UPDATE feeds 
SET
	last_fetched_at = $1
WHERE
	id = $2
RETURNING *;

-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;

-- name: CreateFeedFollow :one
WITH insert_feed_follow AS (
	INSERT INTO feeds_follows (
		id,
		user_id,
		feed_id,
		created_at,
		updated_at
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
	) RETURNING *
) SELECT 
	insert_feed_follow.*,
	feeds.name as feed_name,
	users.name as user_name
FROM insert_feed_follow
INNER JOIN users ON users.id = insert_feed_follow.user_id
INNER JOIN feeds ON feeds.id = insert_feed_follow.feed_id;

-- name: GetFeedFollowsForUser :many 
SELECT 
	f.name as feed_name,
	u.name as user_name,
	f.url,
	ff.created_at,
	ff.updated_at,
	ff.id
FROM feeds_follows ff
INNER JOIN users u ON u.id = ff.user_id
INNER JOIN feeds f ON f.id = ff.feed_id
WHERE u.id = $1;

-- name: DeleteFeedFollow :exec
DELETE FROM feeds_follows
WHERE feed_id = $1 
	AND user_id = $2;
