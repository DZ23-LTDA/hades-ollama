-- Preparation step for an existing volume. Run as the legacy trusted
-- PostgreSQL superuser connected to the existing `ollama_agent` database.
-- This transaction preserves application rows and does not retire the old login;
-- verify the new DSNs before running retire-legacy-role.sql.
\set ON_ERROR_STOP on
\getenv admin_password OLLAMA_AGENT_POSTGRES_ADMIN_PASSWORD
\getenv runtime_password OLLAMA_AGENT_RUNTIME_PASSWORD
\getenv migrator_password OLLAMA_AGENT_MIGRATOR_PASSWORD

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

DO $preflight$
BEGIN
  IF current_database() <> 'ollama_agent' THEN
    RAISE EXCEPTION 'connect to the existing ollama_agent database before running this legacy-only upgrade';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
    RAISE EXCEPTION 'legacy role upgrade requires a trusted superuser connection';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='ollama_agent' AND rolsuper AND rolcanlogin) THEN
    RAISE EXCEPTION 'expected the legacy login ollama_agent to exist and be a superuser';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND usename='ollama_agent' AND pid<>pg_backend_pid()) THEN
    RAISE EXCEPTION 'active ollama_agent sessions detected; stop the legacy Ollama service and retry the maintenance cutover';
  END IF;
  IF to_regclass('public.agent_missions') IS NULL OR to_regclass('public.agent_events') IS NULL THEN
    RAISE EXCEPTION 'legacy agent_missions and agent_events tables must exist before role upgrade';
  END IF;
END
$preflight$;

BEGIN;

SELECT format(
  'CREATE ROLE ollama_agent_admin LOGIN SUPERUSER PASSWORD %L',
  :'admin_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ollama_agent_admin')
\gexec
SELECT format(
  'ALTER ROLE ollama_agent_admin LOGIN SUPERUSER PASSWORD %L',
  :'admin_password'
)
\gexec

SELECT format(
  'CREATE ROLE ollama_agent_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION PASSWORD %L',
  :'migrator_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ollama_agent_migrator')
\gexec
SELECT format(
  'ALTER ROLE ollama_agent_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION PASSWORD %L',
  :'migrator_password'
)
\gexec

SELECT format(
  'CREATE ROLE ollama_agent_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION PASSWORD %L',
  :'runtime_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ollama_agent_runtime')
\gexec
SELECT format(
  'ALTER ROLE ollama_agent_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION PASSWORD %L',
  :'runtime_password'
)
\gexec

-- The dedicated identities must have no role membership, including SET ROLE
-- paths left by hand-configured legacy deployments.
SELECT format('REVOKE %I FROM %I', granted.rolname, member.rolname)
FROM pg_auth_members memberships
JOIN pg_roles granted ON granted.oid = memberships.roleid
JOIN pg_roles member ON member.oid = memberships.member
WHERE member.rolname IN ('ollama_agent_runtime', 'ollama_agent_migrator')
\gexec

ALTER DATABASE ollama_agent OWNER TO ollama_agent_migrator;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_migrator, ollama_agent_runtime;
ALTER SCHEMA public OWNER TO ollama_agent_migrator;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO ollama_agent_runtime;

SELECT format('ALTER TABLE public.%I OWNER TO ollama_agent_migrator', table_name)
FROM unnest(ARRAY['agent_missions', 'agent_events', 'agent_tenant_context_key']) AS table_name
WHERE to_regclass(format('public.%I', table_name)) IS NOT NULL
\gexec

SELECT 'ALTER FUNCTION public.agent_tenant_context_matches(TEXT) OWNER TO ollama_agent_migrator'
WHERE to_regprocedure('public.agent_tenant_context_matches(text)') IS NOT NULL
\gexec

COMMIT;
