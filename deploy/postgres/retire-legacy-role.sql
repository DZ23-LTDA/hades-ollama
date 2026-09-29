-- Run only after migrate-existing-roles.sql commits and the operator has
-- successfully connected with the new admin, migrator, and runtime DSNs.
-- This final transaction disables the old superuser application login.
\set ON_ERROR_STOP on

DO $preflight$
DECLARE
  unsafe_runtime BOOLEAN;
  unsafe_migrator BOOLEAN;
BEGIN
  IF current_database() <> 'ollama_agent' THEN
    RAISE EXCEPTION 'connect to the existing ollama_agent database';
  END IF;
  IF current_user <> 'ollama_agent_admin' THEN
    RAISE EXCEPTION 'connect as the verified ollama_agent_admin login before retiring the legacy role';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='ollama_agent' AND rolsuper AND rolcanlogin) THEN
    RAISE EXCEPTION 'legacy ollama_agent login is absent or already retired';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname='ollama_agent' AND datdba=(SELECT oid FROM pg_roles WHERE rolname='ollama_agent_migrator')) THEN
    RAISE EXCEPTION 'database ownership has not been transferred to ollama_agent_migrator';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=to_regclass('public.agent_missions') AND relowner=(SELECT oid FROM pg_roles WHERE rolname='ollama_agent_migrator'))
     OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=to_regclass('public.agent_events') AND relowner=(SELECT oid FROM pg_roles WHERE rolname='ollama_agent_migrator')) THEN
    RAISE EXCEPTION 'agent tables must be owned by ollama_agent_migrator before legacy retirement';
  END IF;
  SELECT rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication
      OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=r.oid)
    INTO unsafe_runtime FROM pg_roles r WHERE rolname='ollama_agent_runtime';
  IF unsafe_runtime IS DISTINCT FROM FALSE THEN
    RAISE EXCEPTION 'ollama_agent_runtime is missing or has unsafe role privileges/membership';
  END IF;
  SELECT rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication
      OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=r.oid)
    INTO unsafe_migrator FROM pg_roles r WHERE rolname='ollama_agent_migrator';
  IF unsafe_migrator IS DISTINCT FROM FALSE THEN
    RAISE EXCEPTION 'ollama_agent_migrator is missing or has unsafe role privileges/membership';
  END IF;
END
$preflight$;

BEGIN;
REVOKE ALL ON DATABASE ollama_agent FROM ollama_agent;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM ollama_agent;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM ollama_agent;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM ollama_agent;
REVOKE CREATE ON SCHEMA public FROM ollama_agent;
DO $sessions$
DECLARE
  active_sessions BIGINT;
  terminated_sessions BIGINT;
BEGIN
  SELECT count(*) INTO active_sessions
  FROM pg_stat_activity
  WHERE usename='ollama_agent' AND pid <> pg_backend_pid();
  SELECT count(*) INTO terminated_sessions FROM (
    SELECT pg_terminate_backend(pid) AS terminated
    FROM pg_stat_activity
    WHERE usename='ollama_agent' AND pid <> pg_backend_pid()
  ) AS termination_results WHERE terminated;
  IF terminated_sessions <> active_sessions THEN
    RAISE EXCEPTION 'could not terminate all remaining legacy ollama_agent sessions (% of %)', terminated_sessions, active_sessions;
  END IF;
END
$sessions$;
ALTER ROLE ollama_agent NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION;
COMMIT;
