-- name: CreateUser :one
insert into users (
  email,
  hashed_password
) values (
  $1, $2
) returning id, created_at, updated_at, email, is_chirpy_red;

-- name: DeleteAllUsers :exec
delete from users;

-- name: GetUserByEmail :one
select * from users where email = $1 limit 1;

-- name: GetUser :one
select * from users where id = $1 limit 1;


-- name: UpdateUserEmail :one
update users
set email = $2
where id = $1
returning id, created_at, updated_at, email, is_chirpy_red;

-- name: UpdateUserPassword :one
update users
set hashed_password = $2
where id = $1
returning id, created_at, updated_at, email, is_chirpy_red;

-- name: UpgradeUser :one
update users
set is_chirpy_red = true
where id = $1
returning id, created_at, updated_at, email, is_chirpy_red;
