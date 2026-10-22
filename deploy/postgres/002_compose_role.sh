#!/bin/sh
set -eu

# psql's :'name' form quotes this value as a SQL literal. Use a URL-safe
# alphanumeric value because the API also embeds it in a PostgreSQL URL.
psql \
  --set=ON_ERROR_STOP=1 \
  --set=app_password="$APP_DATABASE_PASSWORD" \
  --username "$POSTGRES_USER" \
  --dbname "$POSTGRES_DB" <<'SQL'
SELECT format(
    'CREATE ROLE app_login LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS',
    :'app_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_login')
\gexec

ALTER ROLE app_login PASSWORD :'app_password';
GRANT app_runtime TO app_login;
SQL
