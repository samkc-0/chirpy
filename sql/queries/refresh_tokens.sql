-- name: CreateRefreshToken :one
insert into refresh_tokens (
  token, user_id
) values (
  $1, $2
) returning token;

-- name: ValidateRefreshToken :one
select (revoked_at is null and now() < expires_at) as valid, user_id
from refresh_tokens where token = $1 limit 1;

-- name: RevokeRefreshToken :exec
update refresh_tokens
set
  revoked_at = now(),
  updated_at = now()
where token = $1
returning token;
