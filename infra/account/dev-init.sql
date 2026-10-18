-- The shared account database for a dev Postgres (libs/account). Mounted into
-- /docker-entrypoint-initdb.d; a volume that already exists gets it from
-- `make account.db`.
CREATE DATABASE account_app;
