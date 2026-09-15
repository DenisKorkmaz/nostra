-- name: InsertPing :exec
insert into ping_log (message) values ($1);

-- name: PingStats :one
select count(*)::bigint as ping_count, now()::timestamptz as db_time from ping_log;
