-- +goose Up
create table ping_log (
    id         bigserial   primary key,
    message    text        not null,
    created_at timestamptz not null default now()
);

-- +goose Down
drop table ping_log;
