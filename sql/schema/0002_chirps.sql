-- +goose Up
create table chirps (
  id uuid primary key default gen_random_uuid(),
  created_at timestamp default now(),
  updated_at timestamp default now(),
  body text not null,
  user_id uuid not null,
  foreign key (user_id) references users(id) on delete cascade
);

-- +goose Down
drop table chirps;
