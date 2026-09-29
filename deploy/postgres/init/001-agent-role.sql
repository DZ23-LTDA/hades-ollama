-- Docker entrypoint executes this as the bootstrap administrator only on a
-- fresh volume. Runtime never receives ownership, migration credentials, or
-- CREATE privileges on the application schema.
SET search_path = pg_catalog, public;
\getenv runtime_password OLLAMA_AGENT_RUNTIME_PASSWORD
\getenv migrator_password OLLAMA_AGENT_MIGRATOR_PASSWORD

CREATE ROLE ollama_agent_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS PASSWORD :'migrator_password';
CREATE ROLE ollama_agent_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS PASSWORD :'runtime_password';

ALTER DATABASE ollama_agent OWNER TO ollama_agent_migrator;
REVOKE CONNECT ON DATABASE ollama_agent FROM PUBLIC;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_runtime;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_migrator;
REVOKE CREATE, TEMPORARY ON DATABASE ollama_agent FROM PUBLIC;
REVOKE CREATE, TEMPORARY ON DATABASE ollama_agent FROM ollama_agent_runtime;
SELECT pg_catalog.format('REVOKE ALL ON DATABASE %I FROM PUBLIC, ollama_agent_runtime, ollama_agent_migrator', datname)
FROM pg_catalog.pg_database
WHERE datname <> pg_catalog.current_database()
\gexec

\connect ollama_agent
SET search_path = pg_catalog, public;
-- pgcrypto is a trusted database prerequisite but extension creation remains
-- privileged on supported PostgreSQL builds. Install it as the bootstrap
-- administrator, never as the runtime or migrator role.
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
ALTER SCHEMA public OWNER TO ollama_agent_migrator;
REVOKE USAGE, CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO ollama_agent_runtime;
DO $column_acl$
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
$column_acl$;
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
