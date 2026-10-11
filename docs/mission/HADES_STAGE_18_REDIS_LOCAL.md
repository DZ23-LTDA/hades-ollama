# Estágio 18 (continuação) — Redis, OTLP e a suíte distribuída medidas LOCALMENTE

## 1. O limite que eu tinha declarado

Na medição anterior (E-121) eu registrei: *"o trecho dependente de Redis foi pulado nesta execução
(`OLLAMA_AGENT_TEST_REDIS_URL is not configured`), ficando com a evidência do CI"*. Como Docker
está disponível nesta máquina, dava para **fechar esse limite** em vez de deixá-lo no CI.

## 2. O que foi medido

Pilha do CI subida localmente (`deploy/docker-compose.agentic.yml`) com portas próprias para evitar
conflito: PostgreSQL 55433, Redis 56379, OTLP HTTP 14318 / gRPC 14317. Redis respondeu `PONG` e o
coletor OTLP ficou `healthy`.

### 2.1 Redis + OTLP: **25 PASS, 0 FAIL, 0 SKIP** (8,58 s)

```
go test -tags integration ./internal/agent/ \
  -run 'TestDistributedRedis|TestDistributedOTLP' -count=1 -v
→ ok  github.com/ollama/ollama/internal/agent  8,582s
→ PASS=25  FAIL=0  SKIP=0
```

Isso cobre fila, retry, DLQ, replay de carta morta, *lease* e recuperação de expiração,
idempotência de *claim*, cotas por tenant e global, ordenação de retry, rejeição de registros
órfãos/inconsistentes, drenagem com *handler* não cooperativo e cancelamento — e o coletor OTLP.

### 2.2 Suíte distribuída COMPLETA: **28 PASS, 1 FAIL, 2 SKIP** (23,87 s)

| Resultado | Quantidade |
| --- | --- |
| PASS | **28** |
| FAIL | **1** |
| SKIP | **2** |

## 3. A falha e o que ela é (declarada, não escondida)

`TestDistributedPostgresRuntimeReadinessForResidualLegacyContract` falha **localmente** com:

```
ERROR: role "ollama_agent" does not exist (SQLSTATE 42704)
```

Isso **não é defeito do produto**: o teste simula um estado **legado pré-existente** e o CI
**cria essa fixture de propósito** antes de rodar:

```sql
CREATE ROLE ollama_agent LOGIN SUPERUSER PASSWORD :'legacy_password';
CREATE ROLE ollama_agent_reverse_member LOGIN PASSWORD :'legacy_password';
GRANT ollama_agent TO ollama_agent_reverse_member;
GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_reverse_member;
```

**Procedimento exato para completar a medição local** (não feito nesta rodada por orçamento):
criar esses dois papéis na instância local e **aplicar o patch de HBA** que o CI aplica (a linha
`host ollama_agent ollama_agent ...` passa a apontar para `ollama_agent_reverse_member`), com
`OLLAMA_AGENT_TEST_POSTGRES_LEGACY_URL` apontando para o papel de *reverse member*.

Também vale registrar um ponto de robustez: esse teste **falha** quando a fixture não existe, em
vez de **pular** com a declaração do pré-requisito — diferente dos outros, que checam o ambiente
antes. É um comportamento a melhorar no teste (não no produto), e fica anotado.

`TestDistributedPostgresRuntimeReadinessForLegacyRole` **PASSA** (0,18 s) quando o DSN do
*migrator* está presente — eu o havia omitido na primeira tentativa, o que produzia a segunda
falha; a causa era **minha configuração**, não o produto.

## 4. Conclusão

O limite "Redis só tem evidência no CI" deixa de existir: **25 testes de Redis e OTLP passaram
nesta máquina**. A suíte distribuída local fecha em **28 PASS**, com **uma** falha explicada por
**fixture ausente** (com procedimento de desbloqueio declarado) e **2 skips**.

Senhas usadas na pilha local são **efêmeras e não versionadas**; a pilha foi **derrubada** ao
final (contêineres e volumes removidos).
