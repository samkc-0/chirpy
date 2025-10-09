-- name: CreateUser :one
insert into users (
  email,
  hashed_password
) values (
  $1, $2
) returning id, created_at, updated_at, email;

-- name: DeleteAllUsers :exec
delete from users;

-- name: GetUserByEmail :one
select * from users where email = $1 limit 1;

-- name: GetUser :one
select * from users where id = $1 limit 1;
