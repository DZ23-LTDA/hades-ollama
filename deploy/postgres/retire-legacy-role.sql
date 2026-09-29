-- Run as ollama_agent_admin only after the new admin/migrator/runtime DSNs
-- have been verified and the legacy Ollama service has been stopped.
-- Security ordering is intentional: commit NOLOGIN/NOSUPERUSER and privilege
-- revocation first, then terminate and wait for all old backends to disappear.
-- Runtime readiness rejects any survivor.
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
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='ollama_agent') THEN
    RAISE EXCEPTION 'legacy ollama_agent role is missing';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_database WHERE datname='ollama_agent' AND datdba=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent_migrator')) THEN
    RAISE EXCEPTION 'database ownership has not been transferred to ollama_agent_migrator';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE oid=to_regclass('public.agent_missions') AND relowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent_migrator'))
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE oid=to_regclass('public.agent_events') AND relowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent_migrator')) THEN
    RAISE EXCEPTION 'agent tables must be owned by ollama_agent_migrator before legacy retirement';
  END IF;
  SELECT NOT rolcanlogin OR rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolinherit
      OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid OR m.roleid=r.oid)
    INTO unsafe_runtime FROM pg_catalog.pg_roles r WHERE rolname='ollama_agent_runtime';
  IF unsafe_runtime IS DISTINCT FROM FALSE THEN
    RAISE EXCEPTION 'ollama_agent_runtime is missing or has unsafe role privileges/membership';
  END IF;
  SELECT NOT rolcanlogin OR rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolinherit
      OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid OR m.roleid=r.oid)
    INTO unsafe_migrator FROM pg_catalog.pg_roles r WHERE rolname='ollama_agent_migrator';
  IF unsafe_migrator IS DISTINCT FROM FALSE THEN
    RAISE EXCEPTION 'ollama_agent_migrator is missing or has unsafe role privileges/membership';
  END IF;
END
$preflight$;

-- Block new legacy connections before cleaning up existing ones. If a later
-- step fails, this state is safe and the script can be retried by the new admin.
BEGIN;
REVOKE CONNECT ON DATABASE ollama_agent FROM PUBLIC;
REVOKE ALL ON DATABASE ollama_agent FROM ollama_agent;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_admin, ollama_agent_migrator, ollama_agent_runtime;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM ollama_agent;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM ollama_agent;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM ollama_agent;
REVOKE CREATE ON SCHEMA public FROM ollama_agent;
SELECT format('REVOKE %I FROM ollama_agent', granted.rolname)
FROM pg_catalog.pg_auth_members memberships
JOIN pg_catalog.pg_roles granted ON granted.oid=memberships.roleid
JOIN pg_catalog.pg_roles member ON member.oid=memberships.member
WHERE member.rolname='ollama_agent'
\gexec
ALTER ROLE ollama_agent NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS NOREPLICATION;
COMMIT;

-- ALTER ROLE does not disconnect old sessions. NOLOGIN is already committed,
-- so reconnects cannot race in. Clear PostgreSQL's activity snapshot on every
-- bounded poll; a successful termination signal is not treated as proof.
DO $verify_retirement$
DECLARE
  active_sessions BIGINT;
  zero_samples INTEGER := 0;
BEGIN
  FOR attempt IN 1..300 LOOP
    PERFORM pg_stat_clear_snapshot();
    SELECT count(*) INTO active_sessions
    FROM pg_catalog.pg_stat_activity
    WHERE usename='ollama_agent' AND pid<>pg_catalog.pg_backend_pid();

    IF active_sessions=0 THEN
      zero_samples := zero_samples + 1;
      EXIT WHEN zero_samples >= 5;
    ELSE
      zero_samples := 0;
      PERFORM pg_terminate_backend(pid)
      FROM pg_catalog.pg_stat_activity
      WHERE usename='ollama_agent' AND pid<>pg_catalog.pg_backend_pid();
    END IF;

    IF attempt=300 THEN
      RAISE EXCEPTION 'legacy ollama_agent sessions remain after the 30-second termination window; keep runtime stopped and retry';
    END IF;
    PERFORM pg_sleep(0.1);
  END LOOP;

  PERFORM pg_stat_clear_snapshot();
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_stat_activity WHERE usename='ollama_agent' AND pid<>pg_catalog.pg_backend_pid()) THEN
    RAISE EXCEPTION 'legacy ollama_agent session appeared after termination';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname='ollama_agent' AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolbypassrls AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolinherit AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid OR m.roleid=r.oid)) THEN
    RAISE EXCEPTION 'legacy ollama_agent role did not reach the required disabled, least-privilege state';
  END IF;
END
$verify_retirement$;
