-- name: CreateChirp :one
insert into chirps (
  body, user_id
) values (
  $1, $2
) returning *;

-- name: GetAllChirps :many
select * from chirps order by created_at;

-- name: GetChirpsByAuthor :many
select * from chirps where user_id = $1 order by created_at;

-- name: GetChirp :one
select * from chirps where id = $1 limit 1;

-- name: DeleteChirp :exec
delete from chirps where id = $1 and user_id = $2;
