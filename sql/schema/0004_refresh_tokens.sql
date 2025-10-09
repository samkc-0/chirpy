-- +goose Up
create table refresh_tokens (
  token text primary key,
  user_id uuid not null,
  expires_at timestamp default now() + interval '60 days',
  revoked_at timestamp,
  created_at timestamp default now(),
  updated_at timestamp default now(),
  foreign key (user_id) references users(id) on delete cascade
);

-- +goose Down
drop table refresh_tokens;
