-- Local Docker Compose credentials only. Provision a distinct secret-managed
-- login in each deployed environment and grant that login app_runtime.
DO $role$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_login') THEN
        CREATE ROLE app_login
            LOGIN
            PASSWORD 'app'
            NOSUPERUSER
            NOCREATEDB
            NOCREATEROLE
            NOINHERIT
            NOBYPASSRLS;
    END IF;
END
$role$;

GRANT app_runtime TO app_login;
