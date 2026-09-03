-- PostgreSQL project-tenancy baseline
-- Target: PostgreSQL 15+
--
-- Trust model:
--   * The API validates the OIDC access token before touching this schema.
--   * The API connects with a login role that SET ROLEs to app_runtime.
--   * app_runtime must never own these objects and must not have BYPASSRLS.
--   * Every application transaction sets app.user_id from the validated token.

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE SCHEMA IF NOT EXISTS app;
REVOKE ALL ON SCHEMA app FROM PUBLIC;

-- Managed services may require this role to be created outside the migration.
DO $role$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_runtime') THEN
        CREATE ROLE app_runtime
            NOLOGIN
            NOSUPERUSER
            NOCREATEDB
            NOCREATEROLE
            NOINHERIT
            NOBYPASSRLS;
    END IF;
END
$role$;

CREATE TABLE app.user_account (
    id                      uuid PRIMARY KEY DEFAULT public.gen_random_uuid(),
    primary_email           text,
    primary_email_verified  boolean NOT NULL DEFAULT false,
    display_name            text NOT NULL DEFAULT 'User',
    avatar_url              text,
    status                  text NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'disabled')),
    version                 bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at              timestamptz NOT NULL DEFAULT statement_timestamp(),
    updated_at              timestamptz NOT NULL DEFAULT statement_timestamp(),

    CHECK (primary_email IS NULL OR (
        btrim(primary_email) <> '' AND char_length(primary_email) <= 320
    )),
    CHECK (char_length(display_name) BETWEEN 1 AND 200)
);

-- Email is a contact and invitation attribute, never an authentication key.
CREATE UNIQUE INDEX user_account_verified_email_uq
    ON app.user_account (lower(primary_email))
    WHERE primary_email IS NOT NULL AND primary_email_verified;

CREATE TABLE app.oidc_identity (
    id                       uuid PRIMARY KEY DEFAULT public.gen_random_uuid(),
    user_id                  uuid NOT NULL
                                 REFERENCES app.user_account(id) ON DELETE CASCADE,
    issuer                   text NOT NULL,
    subject                  text NOT NULL,
    provider_email           text,
    provider_email_verified  boolean NOT NULL DEFAULT false,
    last_login_at            timestamptz NOT NULL DEFAULT statement_timestamp(),
    created_at               timestamptz NOT NULL DEFAULT statement_timestamp(),

    UNIQUE (issuer, subject),
    CHECK (btrim(issuer) <> ''),
    CHECK (btrim(subject) <> '')
);

CREATE INDEX oidc_identity_user_idx ON app.oidc_identity(user_id);

CREATE TABLE app.project (
    id              uuid PRIMARY KEY DEFAULT public.gen_random_uuid(),
    owner_user_id   uuid NOT NULL
                        REFERENCES app.user_account(id) ON DELETE RESTRICT,
    name            text NOT NULL,
    description     text,
    version         bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at      timestamptz NOT NULL DEFAULT statement_timestamp(),
    updated_at      timestamptz NOT NULL DEFAULT statement_timestamp(),

    CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    CHECK (description IS NULL OR char_length(description) <= 5000)
);

CREATE INDEX project_owner_created_idx
    ON app.project(owner_user_id, created_at DESC, id);

-- Presence in this table is the one and only project access level.
-- The owner also has a row here; ownership itself is stored on app.project.
CREATE TABLE app.project_member (
    project_id       uuid NOT NULL
                         REFERENCES app.project(id) ON DELETE CASCADE,
    user_id          uuid NOT NULL
                         REFERENCES app.user_account(id) ON DELETE CASCADE,
    added_by_user_id uuid
                         REFERENCES app.user_account(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT statement_timestamp(),

    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX project_member_user_idx
    ON app.project_member(user_id, project_id);

CREATE TABLE app.project_invitation (
    id                  uuid PRIMARY KEY DEFAULT public.gen_random_uuid(),
    project_id          uuid NOT NULL
                            REFERENCES app.project(id) ON DELETE CASCADE,
    email               text NOT NULL,
    token_hash          bytea NOT NULL UNIQUE,
    invited_by_user_id  uuid NOT NULL
                            REFERENCES app.user_account(id) ON DELETE RESTRICT,
    status              text NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'accepted', 'expired')),
    expires_at          timestamptz NOT NULL,
    accepted_by_user_id uuid
                            REFERENCES app.user_account(id) ON DELETE SET NULL,
    accepted_at         timestamptz,
    created_at          timestamptz NOT NULL DEFAULT statement_timestamp(),
    updated_at          timestamptz NOT NULL DEFAULT statement_timestamp(),

    CHECK (email = lower(btrim(email)) AND email <> ''),
    CHECK (octet_length(token_hash) = 32),
    CHECK (expires_at > created_at),
    CHECK (
        (status = 'accepted' AND accepted_at IS NOT NULL)
        OR
        (status <> 'accepted' AND accepted_by_user_id IS NULL AND accepted_at IS NULL)
    )
);

-- Before issuing a replacement invite, the API deletes an expired pending row.
CREATE UNIQUE INDEX project_invitation_pending_email_uq
    ON app.project_invitation(project_id, lower(email))
    WHERE status = 'pending';

CREATE INDEX project_invitation_project_idx
    ON app.project_invitation(project_id, created_at DESC, id);

CREATE TABLE app.project_service (
    project_id          uuid NOT NULL
                            REFERENCES app.project(id) ON DELETE CASCADE,
    id                  uuid NOT NULL DEFAULT public.gen_random_uuid(),
    name                text NOT NULL,
    slug                text NOT NULL,
    kind                text NOT NULL,
    description         text,
    configuration       jsonb NOT NULL DEFAULT '{}'::jsonb,
    state               text NOT NULL DEFAULT 'active'
                            CHECK (state IN ('active', 'disabled')),
    created_by_user_id  uuid NOT NULL
                            REFERENCES app.user_account(id) ON DELETE RESTRICT,
    version             bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at          timestamptz NOT NULL DEFAULT statement_timestamp(),
    updated_at          timestamptz NOT NULL DEFAULT statement_timestamp(),

    PRIMARY KEY (project_id, id),
    UNIQUE (project_id, slug),
    CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    CHECK (char_length(slug) BETWEEN 1 AND 63),
    CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CHECK (char_length(btrim(kind)) BETWEEN 1 AND 100),
    CHECK (description IS NULL OR char_length(description) <= 5000),
    CHECK (jsonb_typeof(configuration) = 'object')
);

CREATE INDEX project_service_project_created_idx
    ON app.project_service(project_id, created_at DESC, id);

CREATE INDEX project_service_kind_idx
    ON app.project_service(project_id, kind);

-- Example child entity. Domain-specific service tables should repeat project_id
-- and reference (project_id, service_id), preventing cross-tenant relationships.
CREATE TABLE app.service_resource (
    project_id  uuid NOT NULL,
    service_id  uuid NOT NULL,
    id          uuid NOT NULL DEFAULT public.gen_random_uuid(),
    resource_key text NOT NULL,
    payload     jsonb NOT NULL DEFAULT '{}'::jsonb,
    version     bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at  timestamptz NOT NULL DEFAULT statement_timestamp(),
    updated_at  timestamptz NOT NULL DEFAULT statement_timestamp(),

    PRIMARY KEY (project_id, service_id, id),
    UNIQUE (project_id, service_id, resource_key),
    FOREIGN KEY (project_id, service_id)
        REFERENCES app.project_service(project_id, id) ON DELETE CASCADE,
    CHECK (char_length(btrim(resource_key)) BETWEEN 1 AND 200),
    CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX service_resource_service_created_idx
    ON app.service_resource(project_id, service_id, created_at DESC, id);

CREATE TABLE app.project_audit_event (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id      uuid NOT NULL
                        REFERENCES app.project(id) ON DELETE CASCADE,
    actor_user_id   uuid
                        REFERENCES app.user_account(id) ON DELETE SET NULL,
    request_id      uuid,
    action          text NOT NULL,
    resource_type   text NOT NULL,
    resource_id     uuid,
    details         jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at     timestamptz NOT NULL DEFAULT statement_timestamp(),

    CHECK (char_length(action) BETWEEN 1 AND 100),
    CHECK (char_length(resource_type) BETWEEN 1 AND 100),
    CHECK (jsonb_typeof(details) = 'object')
);

CREATE INDEX project_audit_event_project_idx
    ON app.project_audit_event(project_id, occurred_at DESC, id DESC);

-------------------------------------------------------------------------------
-- Session context and authorization helpers
-------------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION app.current_user_id()
RETURNS uuid
LANGUAGE sql
STABLE
PARALLEL SAFE
AS $function$
    SELECT NULLIF(current_setting('app.user_id', true), '')::uuid
$function$;

-- SECURITY DEFINER avoids recursive RLS evaluation on project_member.
-- These functions must remain owned by the migration/table owner, not app_runtime.
CREATE OR REPLACE FUNCTION app.has_project_access(p_project_id uuid)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, app
AS $function$
    SELECT EXISTS (
        SELECT 1
        FROM app.project_member AS pm
        WHERE pm.project_id = p_project_id
          AND pm.user_id = app.current_user_id()
    )
$function$;

CREATE OR REPLACE FUNCTION app.is_project_owner(p_project_id uuid)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, app
AS $function$
    SELECT EXISTS (
        SELECT 1
        FROM app.project AS p
        WHERE p.id = p_project_id
          AND p.owner_user_id = app.current_user_id()
    )
$function$;

-- Returns only the public profile fields needed by the members endpoint. A
-- direct join to user_account would correctly hide other users under its
-- self-only RLS policy.
CREATE OR REPLACE FUNCTION app.list_project_members(
    p_project_id      uuid,
    p_limit           integer DEFAULT 51,
    p_after_joined_at timestamptz DEFAULT NULL,
    p_after_user_id   uuid DEFAULT NULL
)
RETURNS TABLE (
    user_id      uuid,
    display_name text,
    avatar_url   text,
    is_owner     boolean,
    joined_at    timestamptz
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, app
AS $function$
BEGIN
    IF NOT app.has_project_access(p_project_id) THEN
        RAISE EXCEPTION 'project_not_found' USING ERRCODE = 'P0002';
    END IF;

    IF p_limit < 1 OR p_limit > 101 THEN
        RAISE EXCEPTION 'member_page_limit_out_of_range' USING ERRCODE = '22023';
    END IF;

    IF (p_after_joined_at IS NULL) <> (p_after_user_id IS NULL) THEN
        RAISE EXCEPTION 'member_cursor_is_incomplete' USING ERRCODE = '22023';
    END IF;

    RETURN QUERY
    SELECT u.id,
           u.display_name,
           u.avatar_url,
           p.owner_user_id = u.id,
           pm.created_at
      FROM app.project_member AS pm
     JOIN app.user_account AS u ON u.id = pm.user_id
      JOIN app.project AS p ON p.id = pm.project_id
     WHERE pm.project_id = p_project_id
       AND (
           p_after_joined_at IS NULL
           OR (pm.created_at, u.id) > (p_after_joined_at, p_after_user_id)
       )
     ORDER BY pm.created_at, u.id
     LIMIT p_limit;
END
$function$;

-------------------------------------------------------------------------------
-- Integrity and version triggers
-------------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION app.touch_versioned_row()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, app
AS $function$
BEGIN
    NEW.updated_at := statement_timestamp();
    NEW.version := OLD.version + 1;
    RETURN NEW;
END
$function$;

CREATE TRIGGER user_account_touch_version
BEFORE UPDATE ON app.user_account
FOR EACH ROW EXECUTE FUNCTION app.touch_versioned_row();

CREATE TRIGGER project_touch_version
BEFORE UPDATE ON app.project
FOR EACH ROW EXECUTE FUNCTION app.touch_versioned_row();

CREATE TRIGGER project_service_touch_version
BEFORE UPDATE ON app.project_service
FOR EACH ROW EXECUTE FUNCTION app.touch_versioned_row();

CREATE TRIGGER service_resource_touch_version
BEFORE UPDATE ON app.service_resource
FOR EACH ROW EXECUTE FUNCTION app.touch_versioned_row();

CREATE OR REPLACE FUNCTION app.touch_updated_at()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, app
AS $function$
BEGIN
    NEW.updated_at := statement_timestamp();
    RETURN NEW;
END
$function$;

CREATE TRIGGER project_invitation_touch
BEFORE UPDATE ON app.project_invitation
FOR EACH ROW EXECUTE FUNCTION app.touch_updated_at();

CREATE OR REPLACE FUNCTION app.protect_project_identity()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, app
AS $function$
BEGIN
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.owner_user_id IS DISTINCT FROM OLD.owner_user_id
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'project identity and ownership are immutable'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END
$function$;

CREATE TRIGGER project_protect_identity
BEFORE UPDATE ON app.project
FOR EACH ROW EXECUTE FUNCTION app.protect_project_identity();

CREATE OR REPLACE FUNCTION app.add_owner_membership()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, app
AS $function$
BEGIN
    INSERT INTO app.project_member(project_id, user_id, added_by_user_id)
    VALUES (NEW.id, NEW.owner_user_id, NEW.owner_user_id);
    RETURN NEW;
END
$function$;

CREATE TRIGGER project_add_owner_membership
AFTER INSERT ON app.project
FOR EACH ROW EXECUTE FUNCTION app.add_owner_membership();

-------------------------------------------------------------------------------
-- OIDC identity provisioning
-------------------------------------------------------------------------------

-- Call only after the API has validated the access token signature, issuer,
-- audience and lifetime. Email never identifies an existing account here.
CREATE OR REPLACE FUNCTION app.resolve_oidc_user(
    p_issuer          text,
    p_subject         text,
    p_email           text,
    p_email_verified  boolean,
    p_display_name    text,
    p_avatar_url      text
)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, app
AS $function$
DECLARE
    v_user_id uuid;
    v_user_status text;
    v_verified_email text;
BEGIN
    IF NULLIF(btrim(p_issuer), '') IS NULL
       OR NULLIF(btrim(p_subject), '') IS NULL THEN
        RAISE EXCEPTION 'issuer and subject are required'
            USING ERRCODE = '22023';
    END IF;

    -- Serializes first-login provisioning for the same OIDC identity.
    PERFORM pg_advisory_xact_lock(
        hashtextextended(p_issuer || chr(31) || p_subject, 0)
    );

    SELECT oi.user_id, u.status
      INTO v_user_id, v_user_status
      FROM app.oidc_identity AS oi
      JOIN app.user_account AS u ON u.id = oi.user_id
     WHERE oi.issuer = p_issuer
       AND oi.subject = p_subject;

    v_verified_email := CASE
        WHEN p_email_verified
             AND NULLIF(btrim(p_email), '') IS NOT NULL
             AND char_length(btrim(p_email)) <= 320
            THEN lower(btrim(p_email))
        ELSE NULL
    END;

    IF v_user_id IS NOT NULL THEN
        IF v_user_status <> 'active' THEN
            RAISE EXCEPTION 'account_disabled' USING ERRCODE = '42501';
        END IF;

        UPDATE app.oidc_identity
           SET provider_email = NULLIF(lower(btrim(p_email)), ''),
               provider_email_verified = p_email_verified,
               last_login_at = statement_timestamp()
         WHERE issuer = p_issuer
           AND subject = p_subject;

        -- Fill a previously absent primary email, but never silently replace or
        -- link an account merely because two providers return the same email.
        IF v_verified_email IS NOT NULL THEN
            UPDATE app.user_account
               SET primary_email = v_verified_email,
                   primary_email_verified = true
             WHERE id = v_user_id
               AND primary_email IS NULL;
        END IF;

        RETURN v_user_id;
    END IF;

    IF v_verified_email IS NOT NULL AND EXISTS (
        SELECT 1
          FROM app.user_account
         WHERE lower(primary_email) = v_verified_email
           AND primary_email_verified
    ) THEN
        RAISE EXCEPTION 'account_linking_required'
            USING ERRCODE = '23505';
    END IF;

    INSERT INTO app.user_account(
        primary_email,
        primary_email_verified,
        display_name,
        avatar_url
    )
    VALUES (
        v_verified_email,
        v_verified_email IS NOT NULL,
        left(COALESCE(NULLIF(btrim(p_display_name), ''), 'User'), 200),
        NULLIF(btrim(p_avatar_url), '')
    )
    RETURNING id INTO v_user_id;

    INSERT INTO app.oidc_identity(
        user_id,
        issuer,
        subject,
        provider_email,
        provider_email_verified
    )
    VALUES (
        v_user_id,
        p_issuer,
        p_subject,
        NULLIF(lower(btrim(p_email)), ''),
        p_email_verified
    );

    RETURN v_user_id;
END
$function$;

-------------------------------------------------------------------------------
-- Invitation acceptance and audit writing
-------------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION app.accept_project_invitation(p_token text)
RETURNS TABLE (
    accepted_project_id uuid,
    membership_created boolean
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, app, public
AS $function$
DECLARE
    v_actor_id       uuid := app.current_user_id();
    v_invitation_id  uuid;
    v_project_id     uuid;
    v_invited_email  text;
    v_invited_by     uuid;
    v_status         text;
    v_expires_at     timestamptz;
    v_actor_email    text;
    v_email_verified boolean;
    v_rows_inserted  integer;
BEGIN
    IF v_actor_id IS NULL THEN
        RAISE EXCEPTION 'authentication_required' USING ERRCODE = '42501';
    END IF;

    IF NULLIF(p_token, '') IS NULL THEN
        RAISE EXCEPTION 'invitation_token_required' USING ERRCODE = '22023';
    END IF;

    SELECT i.id, i.project_id, i.email, i.invited_by_user_id,
           i.status, i.expires_at
      INTO v_invitation_id, v_project_id, v_invited_email, v_invited_by,
           v_status, v_expires_at
      FROM app.project_invitation AS i
     WHERE i.token_hash = public.digest(p_token, 'sha256')
     FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'invitation_not_found' USING ERRCODE = 'P0002';
    END IF;

    IF v_status <> 'pending' THEN
        RAISE EXCEPTION 'invitation_not_pending' USING ERRCODE = '55000';
    END IF;

    IF v_expires_at <= statement_timestamp() THEN
        RAISE EXCEPTION 'invitation_expired' USING ERRCODE = '55000';
    END IF;

    SELECT lower(primary_email), primary_email_verified
      INTO v_actor_email, v_email_verified
      FROM app.user_account
     WHERE id = v_actor_id;

    IF NOT COALESCE(v_email_verified, false)
       OR v_actor_email IS DISTINCT FROM lower(v_invited_email) THEN
        RAISE EXCEPTION 'invitation_email_mismatch' USING ERRCODE = '42501';
    END IF;

    INSERT INTO app.project_member(project_id, user_id, added_by_user_id)
    VALUES (v_project_id, v_actor_id, v_invited_by)
    ON CONFLICT (project_id, user_id) DO NOTHING;

    GET DIAGNOSTICS v_rows_inserted = ROW_COUNT;

    UPDATE app.project_invitation
       SET status = 'accepted',
           accepted_by_user_id = v_actor_id,
           accepted_at = statement_timestamp()
     WHERE id = v_invitation_id;

    INSERT INTO app.project_audit_event(
        project_id, actor_user_id, action, resource_type, resource_id
    )
    VALUES (
        v_project_id, v_actor_id, 'invitation.accepted', 'project', v_project_id
    );

    RETURN QUERY SELECT v_project_id, v_rows_inserted = 1;
END
$function$;

CREATE OR REPLACE FUNCTION app.write_project_audit_event(
    p_project_id    uuid,
    p_action        text,
    p_resource_type text,
    p_resource_id   uuid,
    p_request_id    uuid,
    p_details       jsonb DEFAULT '{}'::jsonb
)
RETURNS bigint
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, app
AS $function$
DECLARE
    v_event_id bigint;
BEGIN
    IF NOT app.has_project_access(p_project_id) THEN
        RAISE EXCEPTION 'project_access_denied' USING ERRCODE = '42501';
    END IF;

    INSERT INTO app.project_audit_event(
        project_id,
        actor_user_id,
        request_id,
        action,
        resource_type,
        resource_id,
        details
    )
    VALUES (
        p_project_id,
        app.current_user_id(),
        p_request_id,
        p_action,
        p_resource_type,
        p_resource_id,
        COALESCE(p_details, '{}'::jsonb)
    )
    RETURNING id INTO v_event_id;

    RETURN v_event_id;
END
$function$;

-------------------------------------------------------------------------------
-- Row-level security
-------------------------------------------------------------------------------

ALTER TABLE app.user_account ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.oidc_identity ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.project ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.project_member ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.project_invitation ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.project_service ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.service_resource ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.project_audit_event ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_account_select_self
ON app.user_account FOR SELECT
USING (id = app.current_user_id());

CREATE POLICY user_account_update_self
ON app.user_account FOR UPDATE
USING (id = app.current_user_id())
WITH CHECK (id = app.current_user_id());

CREATE POLICY oidc_identity_select_self
ON app.oidc_identity FOR SELECT
USING (user_id = app.current_user_id());

CREATE POLICY project_select_member
ON app.project FOR SELECT
USING (app.has_project_access(id));

CREATE POLICY project_insert_owner
ON app.project FOR INSERT
WITH CHECK (owner_user_id = app.current_user_id());

CREATE POLICY project_update_owner
ON app.project FOR UPDATE
USING (app.is_project_owner(id))
WITH CHECK (owner_user_id = app.current_user_id());

CREATE POLICY project_delete_owner
ON app.project FOR DELETE
USING (app.is_project_owner(id));

CREATE POLICY project_member_select_member
ON app.project_member FOR SELECT
USING (app.has_project_access(project_id));

-- Owners can remove another member; non-owners can remove themselves.
-- The owner cannot remove their own invariant membership.
CREATE POLICY project_member_delete_owner_or_self
ON app.project_member FOR DELETE
USING (
    (app.is_project_owner(project_id) AND user_id <> app.current_user_id())
    OR
    (user_id = app.current_user_id() AND NOT app.is_project_owner(project_id))
);

CREATE POLICY project_invitation_select_owner
ON app.project_invitation FOR SELECT
USING (app.is_project_owner(project_id));

CREATE POLICY project_invitation_insert_owner
ON app.project_invitation FOR INSERT
WITH CHECK (
    app.is_project_owner(project_id)
    AND invited_by_user_id = app.current_user_id()
    AND status = 'pending'
    AND accepted_by_user_id IS NULL
    AND accepted_at IS NULL
);

CREATE POLICY project_invitation_delete_owner
ON app.project_invitation FOR DELETE
USING (app.is_project_owner(project_id));

CREATE POLICY project_service_select_member
ON app.project_service FOR SELECT
USING (app.has_project_access(project_id));

CREATE POLICY project_service_insert_member
ON app.project_service FOR INSERT
WITH CHECK (
    app.has_project_access(project_id)
    AND created_by_user_id = app.current_user_id()
);

CREATE POLICY project_service_update_member
ON app.project_service FOR UPDATE
USING (app.has_project_access(project_id))
WITH CHECK (app.has_project_access(project_id));

CREATE POLICY project_service_delete_member
ON app.project_service FOR DELETE
USING (app.has_project_access(project_id));

CREATE POLICY service_resource_select_member
ON app.service_resource FOR SELECT
USING (app.has_project_access(project_id));

CREATE POLICY service_resource_insert_member
ON app.service_resource FOR INSERT
WITH CHECK (app.has_project_access(project_id));

CREATE POLICY service_resource_update_member
ON app.service_resource FOR UPDATE
USING (app.has_project_access(project_id))
WITH CHECK (app.has_project_access(project_id));

CREATE POLICY service_resource_delete_member
ON app.service_resource FOR DELETE
USING (app.has_project_access(project_id));

CREATE POLICY project_audit_event_select_member
ON app.project_audit_event FOR SELECT
USING (app.has_project_access(project_id));

-------------------------------------------------------------------------------
-- Least-privilege runtime grants
-------------------------------------------------------------------------------

REVOKE ALL ON ALL TABLES IN SCHEMA app FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA app FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA app FROM PUBLIC;

GRANT USAGE ON SCHEMA app TO app_runtime;

GRANT SELECT ON app.user_account TO app_runtime;
GRANT UPDATE (display_name, avatar_url) ON app.user_account TO app_runtime;
GRANT SELECT ON app.oidc_identity TO app_runtime;

GRANT SELECT ON app.project TO app_runtime;
GRANT INSERT (id, owner_user_id, name, description) ON app.project TO app_runtime;
GRANT UPDATE (name, description) ON app.project TO app_runtime;
GRANT DELETE ON app.project TO app_runtime;

GRANT SELECT, DELETE ON app.project_member TO app_runtime;

GRANT SELECT ON app.project_invitation TO app_runtime;
GRANT INSERT (
    id, project_id, email, token_hash, invited_by_user_id, expires_at
) ON app.project_invitation TO app_runtime;
GRANT DELETE ON app.project_invitation TO app_runtime;

GRANT SELECT ON app.project_service TO app_runtime;
GRANT INSERT (
    project_id, id, name, slug, kind, description, configuration,
    state, created_by_user_id
) ON app.project_service TO app_runtime;
GRANT UPDATE (
    name, slug, kind, description, configuration, state
) ON app.project_service TO app_runtime;
GRANT DELETE ON app.project_service TO app_runtime;

GRANT SELECT ON app.service_resource TO app_runtime;
GRANT INSERT (
    project_id, service_id, id, resource_key, payload
) ON app.service_resource TO app_runtime;
GRANT UPDATE (resource_key, payload) ON app.service_resource TO app_runtime;
GRANT DELETE ON app.service_resource TO app_runtime;

GRANT SELECT ON app.project_audit_event TO app_runtime;
GRANT USAGE, SELECT ON SEQUENCE app.project_audit_event_id_seq TO app_runtime;

GRANT EXECUTE ON FUNCTION app.current_user_id() TO app_runtime;
GRANT EXECUTE ON FUNCTION app.has_project_access(uuid) TO app_runtime;
GRANT EXECUTE ON FUNCTION app.is_project_owner(uuid) TO app_runtime;
GRANT EXECUTE ON FUNCTION app.list_project_members(uuid, integer, timestamptz, uuid)
    TO app_runtime;
GRANT EXECUTE ON FUNCTION app.resolve_oidc_user(text, text, text, boolean, text, text)
    TO app_runtime;
GRANT EXECUTE ON FUNCTION app.accept_project_invitation(text) TO app_runtime;
GRANT EXECUTE ON FUNCTION app.write_project_audit_event(
    uuid, text, text, uuid, uuid, jsonb
) TO app_runtime;

COMMIT;

-- Per-request transaction pattern (the API supplies the value, never the client):
--
-- BEGIN;
-- SELECT set_config('app.user_id', :validated_user_id::text, true);
-- ...all request queries...
-- COMMIT;
