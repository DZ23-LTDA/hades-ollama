-- Preparation step for an existing volume. Run as the legacy trusted
-- PostgreSQL superuser connected to the existing `ollama_agent` database.
-- This transaction preserves application rows and does not retire the old login;
-- verify the new DSNs before running retire-legacy-role.sql.
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

DO $preflight$
BEGIN
  IF pg_catalog.current_database() <> 'ollama_agent' THEN
    RAISE EXCEPTION 'connect to the existing ollama_agent database before running this legacy-only upgrade';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper) THEN
    RAISE EXCEPTION 'legacy role upgrade requires a trusted superuser connection';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='ollama_agent' AND rolsuper AND rolcanlogin) THEN
    RAISE EXCEPTION 'expected the legacy login ollama_agent to exist and be a superuser';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_stat_activity WHERE usename='ollama_agent' AND pid<>pg_catalog.pg_backend_pid()) THEN
    RAISE EXCEPTION 'active ollama_agent sessions detected; stop the legacy Ollama service and retry the maintenance cutover';
  END IF;
  IF pg_catalog.to_regclass('public.agent_missions') IS NULL OR pg_catalog.to_regclass('public.agent_events') IS NULL THEN
    RAISE EXCEPTION 'legacy agent_missions and agent_events tables must exist before role upgrade';
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
END
$preflight$;

BEGIN;

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

ALTER DATABASE ollama_agent OWNER TO ollama_agent_migrator;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_migrator, ollama_agent_runtime;
REVOKE CREATE, TEMPORARY ON DATABASE ollama_agent FROM PUBLIC;
REVOKE CREATE, TEMPORARY ON DATABASE ollama_agent FROM ollama_agent_runtime;
ALTER SCHEMA public OWNER TO ollama_agent_migrator;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO ollama_agent_runtime;

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
    SELECT l.oid
    FROM pg_catalog.pg_largeobject_metadata l
    WHERE l.lomowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent')
  LOOP
    command_text := pg_catalog.format('ALTER LARGE OBJECT %s OWNER TO ollama_agent_migrator', object_row.oid);
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
