# Entrega da fase 5 — Infraestrutura distribuída e operação segura

## Escopo implementado

O adapter PostgreSQL para missões/eventos existe e mantém operações transacionais testáveis, mas **o servidor público recusa `OLLAMA_AGENT_DATABASE_URL`** até que papéis runtime/migrator e contexto tenant não-forjável substituam o desenho RLS baseado em GUCs caller-settable. PostgreSQL não está habilitado como store compartilhado de produção; consulte [`SECURITY.md`](../../SECURITY.md). O JSON store local continua sendo o padrão de runtime.

A fila Redis opcional é ativada por `OLLAMA_AGENT_REDIS_URL` e suporta enqueue, claim, backoff, retries, dead-letter, replay e worker persistente. O caminho local continua sendo o default. O protocolo Redis foi mantido em uma camada pequena e testável, sem credenciais em arquivos de configuração.

O runtime possui provider OpenTelemetry opcional por `OLLAMA_AGENT_OTLP_ENDPOINT`. O endpoint deve ser HTTPS, salvo override explícito para desenvolvimento. O TraceStore local continua preservando a auditoria de missão mesmo quando o collector está indisponível.

Companions podem abrir `GET /api/agent/v1/devices/:id/connect` por WebSocket. O handshake requer frame `hello` com token do device e capabilities; o transporte exige TLS por padrão, oferece verificação mTLS opcional, limita Origins e aceita heartbeat/ping. O transporte não concede execução arbitrária: tools e approvals continuam sendo a fronteira de autorização.

O mobile Expo ganhou cache de missão, fila de ações offline e sincronização automática após retorno de conectividade. A sessão continua no SecureStore e a URL do servidor no AsyncStorage.

A pipeline `.github/workflows/dz23-agentic-quality.yaml` executa gates Go, typechecks web/mobile e gera SBOM CycloneDX. `deploy/docker-compose.agentic.yml` fornece PostgreSQL, Redis e OpenTelemetry Collector para desenvolvimento local.

## Gates

Os testes `CGO_ENABLED=0 go test ./internal/agent -count=1` e `CGO_ENABLED=1 go test ./server -count=1` passaram após as alterações. O `npm ci` e `npm run typecheck` do cliente mobile também passaram. O teste de parser RESP2 e o teste do provider OpenTelemetry noop passam junto da suíte agentic.

## Limites honestos

O adapter Redis usa protocolo RESP2; há agora um teste local contra Redis real cobrindo enqueue, binding do owner, rejeição cross-tenant e replay, mas TLS Redis e política de cluster continuam sem validação suficiente para produção. PostgreSQL continua bloqueado até migração para autoridade/contexto tenant não-forjável, backfill e testes adversariais; autenticação HTTP não corrige RLS. O WebSocket exige terminação TLS real, rotação de certificados, mTLS configurado, replay protection e testes em companions físicos. O mobile offline ainda precisa de resolução de conflitos, push notification remoto e testes Android/iOS assinados. A pipeline SBOM não publica nem assina release por conta própria; os workflows de release existentes continuam sendo a autoridade de distribuição.
