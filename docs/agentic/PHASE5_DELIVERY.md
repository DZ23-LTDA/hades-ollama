# Entrega da fase 5 — Infraestrutura distribuída e operação segura

## Escopo implementado

O adapter PostgreSQL para missões/eventos agora usa uma role runtime tenant-only, assinatura HMAC não legível pelo runtime nas policies RLS, migration explicitamente executada com DSN migrator e recusa owners legados ambíguos. A integração em PostgreSQL 16 descartável valida isolamento/leitura/gravação de dois tenants, spoof de GUC legado e HMAC, leitura do segredo, assunção do migrator, DDL runtime, `row_security=off`, chave errada e recusa de backfill inseguro. **Mesmo com esses testes, PostgreSQL permanece candidato não certificado para produção** até gates completos, recuperação de jobs distribuídos e revisão independente; consulte [`SECURITY.md`](../../SECURITY.md). O JSON store local continua sendo o padrão.

A fila Redis opcional é ativada por `OLLAMA_AGENT_REDIS_URL` e suporta enqueue, claim, backoff, retries, dead-letter, replay e worker persistente. O caminho local continua sendo o default. O protocolo Redis foi mantido em uma camada pequena e testável, sem credenciais em arquivos de configuração.

O runtime possui provider OpenTelemetry opcional por `OLLAMA_AGENT_OTLP_ENDPOINT`. O endpoint deve ser HTTPS, salvo override explícito para desenvolvimento. O TraceStore local continua preservando a auditoria de missão mesmo quando o collector está indisponível.

Companions podem abrir `GET /api/agent/v1/devices/:id/connect` por WebSocket. O handshake requer frame `hello` com token do device e capabilities; o transporte exige TLS por padrão, oferece verificação mTLS opcional, limita Origins e aceita heartbeat/ping. O transporte não concede execução arbitrária: tools e approvals continuam sendo a fronteira de autorização.

O mobile Expo ganhou cache de missão, fila de ações offline e sincronização automática após retorno de conectividade. A sessão continua no SecureStore e a URL do servidor no AsyncStorage.

A pipeline `.github/workflows/dz23-agentic-quality.yaml` executa gates Go, typechecks web/mobile e gera SBOM CycloneDX. `deploy/docker-compose.agentic.yml` fornece PostgreSQL, Redis e OpenTelemetry Collector para desenvolvimento local.

## Gates

Os gates completos anteriores registrados neste documento não substituem a validação desta alteração. Nesta mudança, `go test ./internal/agent ./server ./cmd` passou e o `TestDistributedPostgresRLSAndEvents` passou em PostgreSQL 16 descartável com credenciais runtime/migrator distintas. Vet/race, suíte completa congelada, Windows cross-compile, Compose com volume novo, workflow CI e revisão independente ainda precisam ser executados antes do release.

## Limites honestos

O adapter Redis usa protocolo RESP2; há teste local contra Redis real cobrindo enqueue, binding do owner, rejeição cross-tenant e replay, mas TLS Redis e política de cluster continuam sem validação suficiente para produção. O teste PostgreSQL adversarial atual é evidência de progresso, não auditoria de produção: a key HMAC do processo Ollama Full permanece uma autoridade multi-tenant se o processo for comprometido; Redis durável é requisito do runtime PostgreSQL pois varredura global de missions/schedules não recebe bypass system; rotação coordenada, backup/restore, TLS PostgreSQL e HA continuam pendentes. O WebSocket exige terminação TLS real, rotação de certificados, mTLS configurado, replay protection e testes em companions físicos. O mobile offline ainda precisa de resolução de conflitos, push notification remoto e testes Android/iOS assinados. A pipeline SBOM não publica nem assina release por conta própria; os workflows de release existentes continuam sendo a autoridade de distribuição.
