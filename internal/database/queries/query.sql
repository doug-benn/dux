-- name: QueryLink :one
SELECT * FROM links WHERE id = ?;

-- name: ListLinks :many
SELECT
  *
FROM
  links
ORDER BY
  hits;

-- name: CreateLink :one
INSERT INTO
  links (name, url, icon, category, colour, hits)
VALUES
  (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateLink :exec
UPDATE links
SET
  name = sqlc.arg('name'),
  url = sqlc.arg('url'),
  icon = COALESCE(NULLIF(CAST(sqlc.arg('icon') AS TEXT), ''), icon),
  category = sqlc.arg('category'),
  colour = sqlc.arg('colour')
WHERE
  id = sqlc.arg('id');

-- name: DeleteLink :exec
DELETE FROM links
WHERE
  id = ?;

-- name: RecordHit :exec
UPDATE links
SET
  hits = hits + 1
WHERE
  id = ?;

-- name: SearchLinks :many
SELECT
  *
FROM
  links
WHERE
  LOWER(name) LIKE LOWER('%' || sqlc.arg('query') || '%')
  OR LOWER(url) LIKE LOWER('%' || sqlc.arg('query') || '%')
ORDER BY
  hits DESC;