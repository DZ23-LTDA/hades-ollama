-- Docker entrypoint executes this as the bootstrap administrator only on a
-- fresh volume. Runtime never receives ownership, migration credentials, or
-- CREATE privileges on the application schema.
\getenv runtime_password OLLAMA_AGENT_RUNTIME_PASSWORD
\getenv migrator_password OLLAMA_AGENT_MIGRATOR_PASSWORD

CREATE ROLE ollama_agent_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS PASSWORD :'migrator_password';
CREATE ROLE ollama_agent_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS PASSWORD :'runtime_password';

ALTER DATABASE ollama_agent OWNER TO ollama_agent_migrator;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_runtime;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_migrator;
REVOKE CREATE, TEMPORARY ON DATABASE ollama_agent FROM PUBLIC;
REVOKE CREATE, TEMPORARY ON DATABASE ollama_agent FROM ollama_agent_runtime;

\connect ollama_agent
-- pgcrypto is a trusted database prerequisite but extension creation remains
-- privileged on supported PostgreSQL builds. Install it as the bootstrap
-- administrator, never as the runtime or migrator role.
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
ALTER SCHEMA public OWNER TO ollama_agent_migrator;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO ollama_agent_runtime;
