-- +goose Up
create extension if not exists pgcrypto;

create table users (
  id uuid primary key default gen_random_uuid(),
  created_at timestamp default now(),
  updated_at timestamp default now(),
  email text unique not null
);

-- +goose Down
drop table users;
