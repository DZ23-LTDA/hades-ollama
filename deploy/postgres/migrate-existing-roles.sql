-- Preparation step for a dedicated PostgreSQL cluster and database. Run over
-- TCP as the legacy trusted PostgreSQL superuser connected to `ollama_agent`.
-- It provisions new credentials, fences/drains the old superuser immediately,
-- reconnects as the new admin, then transfers the allowlisted Ollama objects.
\set ON_ERROR_STOP on
\getenv admin_password OLLAMA_AGENT_POSTGRES_ADMIN_PASSWORD
\getenv runtime_password OLLAMA_AGENT_RUNTIME_PASSWORD
\getenv migrator_password OLLAMA_AGENT_MIGRATOR_PASSWORD

SET search_path = pg_catalog, public;

SELECT :'admin_password' <> ''
   AND :'runtime_password' <> ''
   AND :'migrator_password' <> ''
   AND :'admin_password' <> :'runtime_password'
   AND :'admin_password' <> :'migrator_password'
   AND :'runtime_password' <> :'migrator_password' AS secrets_valid
\gset
\if :secrets_valid
\else
  \echo "ERROR: set three non-empty, distinct admin/runtime/migrator passwords in the environment"
  \quit 2
\endif

DO $fence_preflight$
BEGIN
  IF pg_catalog.current_database() <> 'ollama_agent' THEN
    RAISE EXCEPTION 'connect to the existing ollama_agent database before running this legacy-only upgrade';
  END IF;
  IF pg_catalog.inet_server_addr() IS NULL THEN
    RAISE EXCEPTION 'connect over TCP so this script can reconnect as ollama_agent_admin immediately after fencing the legacy login';
  END IF;
  IF current_user NOT IN ('ollama_agent','ollama_agent_admin') OR NOT EXISTS
     (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper) THEN
    RAISE EXCEPTION 'connect as the trusted legacy superuser or the fenced cutover administrator';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='ollama_agent'
                 AND ((rolsuper AND rolcanlogin) OR
                      (NOT rolsuper AND NOT rolcanlogin AND NOT rolcreatedb AND NOT rolcreaterole
                       AND NOT rolreplication AND NOT rolbypassrls AND NOT rolinherit))) THEN
    RAISE EXCEPTION 'expected the legacy ollama_agent role to be either the original superuser login or the safely fenced retry state';
  END IF;
END
$fence_preflight$;

BEGIN;

-- The new bootstrap administrator must be available for the immediate
-- reconnect after this transaction. Do not mutate application ACLs, data,
-- memberships, or runtime roles until the old superuser has been fenced and
-- its existing sessions have been drained below.
SELECT pg_catalog.format(
  'CREATE ROLE ollama_agent_admin LOGIN SUPERUSER PASSWORD %L',
  :'admin_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'ollama_agent_admin')
\gexec
SELECT pg_catalog.format(
  'ALTER ROLE ollama_agent_admin LOGIN SUPERUSER PASSWORD %L',
  :'admin_password'
)
\gexec

ALTER ROLE ollama_agent NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION;
COMMIT;

-- Fence new logins first, then reconnect as the independent administrator and
-- drain any legacy sessions that raced the read-only preflight.
\setenv PGPASSWORD :admin_password
\connect -reuse-previous=on ollama_agent ollama_agent_admin
SET search_path = pg_catalog, public;

-- A login granted ollama_agent can retain an authenticated session and
-- SET ROLE after the target role is NOLOGIN. Snapshot all direct/transitive
-- members under a catalog lock, revoke both membership directions, then drain
-- sessions by session_user (pg_stat_activity.usename).
BEGIN;
SET LOCAL lock_timeout = '5s';
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
CREATE TEMP TABLE ollama_legacy_session_principals (rolname NAME PRIMARY KEY) ON COMMIT PRESERVE ROWS;
INSERT INTO pg_temp.ollama_legacy_session_principals (rolname)
WITH RECURSIVE legacy_members(member_oid) AS (
  SELECT memberships.member FROM pg_catalog.pg_auth_members memberships
  WHERE memberships.roleid=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
  UNION
  SELECT memberships.member FROM pg_catalog.pg_auth_members memberships
  JOIN legacy_members nested ON memberships.roleid=nested.member_oid
)
SELECT role_row.rolname FROM pg_catalog.pg_roles role_row
WHERE role_row.oid=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
   OR role_row.oid IN (SELECT member_oid FROM legacy_members);
SELECT DISTINCT pg_catalog.format('REVOKE %I FROM %I', granted.rolname, member.rolname)
FROM pg_catalog.pg_auth_members memberships
JOIN pg_catalog.pg_roles granted ON granted.oid=memberships.roleid
JOIN pg_catalog.pg_roles member ON member.oid=memberships.member
WHERE member.rolname='ollama_agent' OR granted.rolname='ollama_agent'
\gexec
COMMIT;

DO $drain_legacy_sessions$
DECLARE
  active_sessions BIGINT;
  zero_samples INTEGER := 0;
BEGIN
  FOR attempt IN 1..300 LOOP
    PERFORM pg_catalog.pg_stat_clear_snapshot();
    SELECT count(*) INTO active_sessions FROM pg_catalog.pg_stat_activity
    WHERE usename IN (SELECT rolname FROM pg_temp.ollama_legacy_session_principals)
      AND pid<>pg_catalog.pg_backend_pid();
    IF active_sessions=0 THEN
      zero_samples := zero_samples + 1;
      EXIT WHEN zero_samples >= 5;
    ELSE
      zero_samples := 0;
      PERFORM pg_catalog.pg_terminate_backend(pid) FROM pg_catalog.pg_stat_activity
      WHERE usename IN (SELECT rolname FROM pg_temp.ollama_legacy_session_principals)
        AND pid<>pg_catalog.pg_backend_pid();
    END IF;
    IF attempt=300 THEN RAISE EXCEPTION 'legacy sessions remain; keep Ollama stopped and retry as ollama_agent_admin'; END IF;
    PERFORM pg_catalog.pg_sleep(0.1);
  END LOOP;
END
$drain_legacy_sessions$;

-- Validate the full dedicated-cluster and object inventory only after the old
-- superuser has been fenced and all sessions drained.
DO $preflight$
BEGIN
  IF pg_catalog.current_database() <> 'ollama_agent' THEN
    RAISE EXCEPTION 'connect to the existing ollama_agent database before running this legacy-only upgrade';
  END IF;
  IF pg_catalog.inet_server_addr() IS NULL THEN
    RAISE EXCEPTION 'connect over TCP so this script can reconnect as ollama_agent_admin immediately after fencing the legacy login';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_database WHERE NOT datistemplate AND datname NOT IN ('ollama_agent','postgres')) THEN
    RAISE EXCEPTION 'Ollama runtime requires a dedicated PostgreSQL cluster; remove or migrate non-Ollama databases before cutover';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper) THEN
    RAISE EXCEPTION 'legacy role upgrade requires a trusted superuser connection';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='ollama_agent'
                 AND ((rolsuper AND rolcanlogin) OR
                      (NOT rolsuper AND NOT rolcanlogin AND NOT rolcreatedb AND NOT rolcreaterole
                       AND NOT rolreplication AND NOT rolbypassrls AND NOT rolinherit))) THEN
    RAISE EXCEPTION 'expected the legacy ollama_agent role to be either the original superuser login or the safely fenced retry state';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_stat_activity WHERE usename='ollama_agent' AND pid<>pg_catalog.pg_backend_pid()) THEN
    RAISE EXCEPTION 'active ollama_agent sessions detected; stop the legacy Ollama service and retry the maintenance cutover';
  END IF;
  IF pg_catalog.to_regclass('public.agent_missions') IS NULL OR pg_catalog.to_regclass('public.agent_events') IS NULL THEN
    RAISE EXCEPTION 'legacy agent_missions and agent_events tables must exist before role upgrade';
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_class c
    WHERE c.relnamespace='public'::pg_catalog.regnamespace
      AND c.relname IN ('agent_missions','agent_events','agent_tenant_context_key','agent_tenant_context_keys')
      AND ((c.relname IN ('agent_missions','agent_events') AND c.relkind NOT IN ('r','p'))
        OR (c.relname IN ('agent_tenant_context_key','agent_tenant_context_keys') AND c.relkind<>'r')
        OR c.relowner NOT IN ((SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent'),
                              (SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent_migrator')))
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_proc p
    WHERE p.pronamespace='public'::pg_catalog.regnamespace AND p.proname='agent_tenant_context_matches'
      AND NOT (p.prokind='f' AND p.pronargs=1 AND pg_catalog.oidvectortypes(p.proargtypes)='text'
               AND p.prorettype='pg_catalog.bool'::pg_catalog.regtype
               AND p.proowner IN ((SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent'),
                                  (SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent_migrator')))
  ) THEN
    RAISE EXCEPTION 'existing Ollama objects have an unsupported relation kind, verifier signature, or owner; inventory and repair them before cutover';
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_database d
    WHERE d.datdba=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
      AND d.oid<>(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_shdepend d
    WHERE d.refclassid='pg_catalog.pg_authid'::pg_catalog.regclass
      AND d.refobjid=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
      AND d.deptype='o'
      AND d.dbid NOT IN (0,(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database()))
  ) THEN
    RAISE EXCEPTION 'legacy role owns another database or objects outside ollama_agent; inventory and migrate those objects before role preparation';
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_namespace n
    WHERE n.nspname NOT IN ('pg_catalog','information_schema','public')
      AND n.nspname NOT LIKE 'pg_toast%'
      AND n.nspowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
    WHERE n.nspname NOT IN ('pg_catalog','information_schema','public')
      AND n.nspname NOT LIKE 'pg_toast%'
      AND c.relowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname NOT IN ('pg_catalog','information_schema','public')
      AND p.proowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
    WHERE n.nspname NOT IN ('pg_catalog','information_schema','public')
      AND t.typowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname='public'
      AND p.prokind NOT IN ('f','p')
      AND p.proowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_type t
    WHERE t.typnamespace='public'::pg_catalog.regnamespace
      AND t.typowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
      AND t.typtype NOT IN ('c','d','e')
      AND NOT (t.typtype='b' AND t.typelem<>0)
  ) THEN
    RAISE EXCEPTION 'legacy role owns application objects outside public; inventory and migrate those schemas/objects explicitly before role preparation';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_extension e
             WHERE e.extnamespace='public'::pg_catalog.regnamespace AND e.extname<>'pgcrypto') THEN
    RAISE EXCEPTION 'public schema contains a non-pgcrypto extension; migrate it to a separately reviewed dedicated database before cutover';
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_class c
    WHERE c.relnamespace='public'::pg_catalog.regnamespace
      AND c.relkind IN ('r','p','v','m','S','f')
      AND c.relname NOT IN ('agent_missions','agent_events','agent_tenant_context_key','agent_tenant_context_keys')
      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend d
                      WHERE d.classid='pg_catalog.pg_class'::pg_catalog.regclass AND d.objid=c.oid
                        AND d.refclassid='pg_catalog.pg_extension'::pg_catalog.regclass AND d.deptype='e')
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_proc p
    WHERE p.pronamespace='public'::pg_catalog.regnamespace
      AND p.proname <> 'agent_tenant_context_matches'
      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend d
                      WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=p.oid
                        AND d.refclassid='pg_catalog.pg_extension'::pg_catalog.regclass AND d.deptype='e')
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_type t
    WHERE t.typnamespace='public'::pg_catalog.regnamespace AND t.typrelid=0
      AND t.typtype IN ('c','d','e')
      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend d
                      WHERE d.classid='pg_catalog.pg_type'::pg_catalog.regclass AND d.objid=t.oid
                        AND d.refclassid='pg_catalog.pg_extension'::pg_catalog.regclass AND d.deptype='e')
  ) THEN
    RAISE EXCEPTION 'public schema contains unrelated non-extension objects; move them before the dedicated Ollama cutover';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_shdepend d
             WHERE d.refclassid='pg_catalog.pg_authid'::pg_catalog.regclass
               AND d.refobjid=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
               AND d.deptype='o' AND d.dbid=0
               AND NOT (d.classid='pg_catalog.pg_database'::pg_catalog.regclass
                        AND d.objid=(SELECT oid FROM pg_catalog.pg_database WHERE datname='ollama_agent'))) THEN
    RAISE EXCEPTION 'legacy role owns shared objects such as tablespaces; migrate them explicitly before cutover';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_largeobject_metadata WHERE lomowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')) THEN
    RAISE EXCEPTION 'legacy role owns large objects outside the supported Ollama schema; export and migrate them explicitly before cutover';
  END IF;
END
$preflight$;

BEGIN;
-- Recheck under the ownership transaction so a membership granted after the
-- first drain cannot race the privilege transfer. Drain again after commit.
SET LOCAL lock_timeout = '5s';
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
INSERT INTO pg_temp.ollama_legacy_session_principals (rolname)
WITH RECURSIVE legacy_members(member_oid) AS (
  SELECT memberships.member FROM pg_catalog.pg_auth_members memberships
  WHERE memberships.roleid=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
  UNION
  SELECT memberships.member FROM pg_catalog.pg_auth_members memberships
  JOIN legacy_members nested ON memberships.roleid=nested.member_oid
)
SELECT role_row.rolname FROM pg_catalog.pg_roles role_row
WHERE role_row.oid=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
   OR role_row.oid IN (SELECT member_oid FROM legacy_members)
ON CONFLICT (rolname) DO NOTHING;
SELECT pg_catalog.format(
  'CREATE ROLE ollama_agent_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION PASSWORD %L',
  :'migrator_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'ollama_agent_migrator')
\gexec
SELECT pg_catalog.format(
  'ALTER ROLE ollama_agent_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION PASSWORD %L',
  :'migrator_password'
)
\gexec

SELECT pg_catalog.format(
  'CREATE ROLE ollama_agent_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION PASSWORD %L',
  :'runtime_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'ollama_agent_runtime')
\gexec
SELECT pg_catalog.format(
  'ALTER ROLE ollama_agent_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION PASSWORD %L',
  :'runtime_password'
)
\gexec

-- Dedicated runtime/migrator identities must have no role membership,
-- including SET ROLE paths left by a hand-configured legacy deployment.
SELECT DISTINCT pg_catalog.format('REVOKE %I FROM %I', granted.rolname, member.rolname)
FROM pg_catalog.pg_auth_members memberships
JOIN pg_catalog.pg_roles granted ON granted.oid = memberships.roleid
JOIN pg_catalog.pg_roles member ON member.oid = memberships.member
WHERE member.rolname IN ('ollama_agent_runtime', 'ollama_agent_migrator')
   OR granted.rolname IN ('ollama_agent_runtime', 'ollama_agent_migrator')
\gexec

SELECT DISTINCT pg_catalog.format('REVOKE %I FROM %I', granted.rolname, member.rolname)
FROM pg_catalog.pg_auth_members memberships
JOIN pg_catalog.pg_roles granted ON granted.oid=memberships.roleid
JOIN pg_catalog.pg_roles member ON member.oid=memberships.member
WHERE member.rolname='ollama_agent' OR granted.rolname='ollama_agent'
\gexec

-- With legacy logins fenced and sessions drained, prepare the least-privilege
-- roles and transfer ownership/ACLs as one retryable transaction.
REVOKE ALL ON DATABASE ollama_agent FROM PUBLIC, ollama_agent_runtime, ollama_agent;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_admin, ollama_agent_migrator, ollama_agent_runtime;
SELECT pg_catalog.format('REVOKE ALL ON DATABASE %I FROM PUBLIC, ollama_agent_runtime, ollama_agent_migrator, ollama_agent', datname)
FROM pg_catalog.pg_database WHERE datname <> pg_catalog.current_database()
\gexec
REVOKE USAGE, CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO ollama_agent_runtime;

ALTER DATABASE ollama_agent OWNER TO ollama_agent_migrator;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_migrator, ollama_agent_runtime;
REVOKE CONNECT ON DATABASE ollama_agent FROM PUBLIC;
REVOKE CREATE, TEMPORARY ON DATABASE ollama_agent FROM PUBLIC;
REVOKE CREATE, TEMPORARY ON DATABASE ollama_agent FROM ollama_agent_runtime;
ALTER SCHEMA public OWNER TO ollama_agent_migrator;
REVOKE USAGE, CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO ollama_agent_runtime;

DO $revoke_column_acl$
DECLARE object_row RECORD;
BEGIN
  FOR object_row IN
    SELECT c.relname, pg_catalog.string_agg(pg_catalog.format('%I',a.attname),',' ORDER BY a.attnum) AS columns
    FROM pg_catalog.pg_class c JOIN pg_catalog.pg_attribute a ON a.attrelid=c.oid
    WHERE c.relnamespace='public'::pg_catalog.regnamespace AND c.relkind IN ('r','p','v','m','f')
      AND a.attnum>0 AND NOT a.attisdropped
    GROUP BY c.relname
  LOOP
    EXECUTE pg_catalog.format('REVOKE ALL (%s) ON TABLE public.%I FROM PUBLIC, ollama_agent_runtime',object_row.columns,object_row.relname);
  END LOOP;
END
$revoke_column_acl$;

-- Transfer every supported application relation, routine and user-defined
-- composite/domain/enum type in public. Phase two checks remaining ownership.
DO $transfer_public_objects$
DECLARE
  object_row RECORD;
  command_text TEXT;
BEGIN
  FOR object_row IN
    SELECT c.relname, c.relkind
    FROM pg_catalog.pg_class c
    WHERE c.relnamespace='public'::pg_catalog.regnamespace
      AND c.relowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
      AND c.relkind IN ('r','p','v','m','S','f')
  LOOP
    command_text := CASE object_row.relkind
      WHEN 'S' THEN pg_catalog.format('ALTER SEQUENCE public.%I OWNER TO ollama_agent_migrator', object_row.relname)
      WHEN 'v' THEN pg_catalog.format('ALTER VIEW public.%I OWNER TO ollama_agent_migrator', object_row.relname)
      WHEN 'm' THEN pg_catalog.format('ALTER MATERIALIZED VIEW public.%I OWNER TO ollama_agent_migrator', object_row.relname)
      WHEN 'f' THEN pg_catalog.format('ALTER FOREIGN TABLE public.%I OWNER TO ollama_agent_migrator', object_row.relname)
      ELSE pg_catalog.format('ALTER TABLE public.%I OWNER TO ollama_agent_migrator', object_row.relname)
    END;
    EXECUTE command_text;
  END LOOP;

  FOR object_row IN
    SELECT p.proname, p.prokind, pg_catalog.pg_get_function_identity_arguments(p.oid) AS identity_arguments
    FROM pg_catalog.pg_proc p
    WHERE p.pronamespace='public'::pg_catalog.regnamespace
      AND p.proowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
      AND p.prokind IN ('f','p')
      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=p.oid AND d.deptype='e')
  LOOP
    command_text := pg_catalog.format('ALTER %s %I.%I(%s) OWNER TO ollama_agent_migrator',
      CASE object_row.prokind WHEN 'p' THEN 'PROCEDURE' ELSE 'FUNCTION' END,
      'public', object_row.proname, object_row.identity_arguments);
    EXECUTE command_text;
  END LOOP;

  FOR object_row IN
    SELECT t.typname, t.typtype
    FROM pg_catalog.pg_type t
    WHERE t.typnamespace='public'::pg_catalog.regnamespace
      AND t.typowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
      AND t.typtype IN ('c','d','e')
  LOOP
    command_text := pg_catalog.format('ALTER %s public.%I OWNER TO ollama_agent_migrator',
      CASE object_row.typtype WHEN 'd' THEN 'DOMAIN' ELSE 'TYPE' END,
      object_row.typname);
    EXECUTE command_text;
  END LOOP;

END
$transfer_public_objects$;

-- Also transfer extension ownership and any catalog object kinds not handled
-- by the explicit supported public-schema inventory above.
REASSIGN OWNED BY ollama_agent TO ollama_agent_migrator;

-- Remove inherited PUBLIC and explicit runtime grants from every existing
-- public object, including legacy SECURITY DEFINER routines. The sole direct
-- helper grant is pgcrypto hmac for the migrator-owned SECURITY DEFINER verifier.
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC, ollama_agent_runtime;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC, ollama_agent_runtime;
REVOKE ALL ON ALL ROUTINES IN SCHEMA public FROM PUBLIC, ollama_agent_runtime;
GRANT EXECUTE ON FUNCTION public.hmac(bytea, bytea, text) TO ollama_agent_migrator;

ALTER DEFAULT PRIVILEGES FOR ROLE ollama_agent_migrator REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE ollama_agent_migrator REVOKE EXECUTE ON FUNCTIONS FROM ollama_agent_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE ollama_agent_migrator REVOKE ALL ON TABLES FROM PUBLIC, ollama_agent_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE ollama_agent_migrator REVOKE ALL ON SEQUENCES FROM PUBLIC, ollama_agent_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE ollama_agent_migrator IN SCHEMA public REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE ollama_agent_migrator IN SCHEMA public REVOKE EXECUTE ON FUNCTIONS FROM ollama_agent_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE ollama_agent_migrator IN SCHEMA public REVOKE ALL ON TABLES FROM PUBLIC, ollama_agent_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE ollama_agent_migrator IN SCHEMA public REVOKE ALL ON SEQUENCES FROM PUBLIC, ollama_agent_runtime;

COMMIT;

-- The second locked snapshot may have found a member connected after the
-- initial drain. Revoke prevents new role use; terminate and verify those
-- already-authenticated sessions before declaring phase one complete.
DO $drain_post_transfer_legacy_sessions$
DECLARE active_sessions BIGINT;
BEGIN
  FOR attempt IN 1..300 LOOP
    PERFORM pg_catalog.pg_stat_clear_snapshot();
    SELECT count(*) INTO active_sessions FROM pg_catalog.pg_stat_activity
    WHERE usename IN (SELECT rolname FROM pg_temp.ollama_legacy_session_principals)
      AND pid<>pg_catalog.pg_backend_pid();
    EXIT WHEN active_sessions=0;
    PERFORM pg_catalog.pg_terminate_backend(pid) FROM pg_catalog.pg_stat_activity
    WHERE usename IN (SELECT rolname FROM pg_temp.ollama_legacy_session_principals)
      AND pid<>pg_catalog.pg_backend_pid();
    IF attempt=300 THEN RAISE EXCEPTION 'legacy or member sessions remain after ownership transfer; keep runtime stopped and retry'; END IF;
    PERFORM pg_catalog.pg_sleep(0.1);
  END LOOP;
  PERFORM pg_catalog.pg_stat_clear_snapshot();
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_stat_activity
             WHERE usename IN (SELECT rolname FROM pg_temp.ollama_legacy_session_principals)
               AND pid<>pg_catalog.pg_backend_pid()) THEN
    RAISE EXCEPTION 'legacy or member session appeared after ownership transfer';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members memberships
             JOIN pg_catalog.pg_roles granted ON granted.oid=memberships.roleid
             JOIN pg_catalog.pg_roles member ON member.oid=memberships.member
             WHERE member.rolname='ollama_agent' OR granted.rolname='ollama_agent') THEN
    RAISE EXCEPTION 'legacy role membership was recreated during cutover; retry after all other superusers are quiesced';
  END IF;
END
$drain_post_transfer_legacy_sessions$;
