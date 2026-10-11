# Estágio 18 — RLS e não-inferência medidos contra PostgreSQL REAL (local)

## 1. O que o estágio pede, e o que eu vinha declarando

O estágio 18 exige **RLS com papéis distintos** e **prova em banco real** de que o tenant A não
acessa **nem infere** o tenant B. Eu vinha registrando isso como pendente por falta de banco —
mas **Docker está disponível nesta máquina**, então dava para medir de verdade em vez de esperar
o CI.

## 2. Como foi medido

1. `docker compose -f deploy/docker-compose.agentic.yml up -d --wait` — a **mesma** pilha que o
   CI usa: PostgreSQL 16.15, Redis 7 e coletor OTLP, com os papéis
   `ollama_agent_admin` / `ollama_agent_migrator` / `ollama_agent_runtime` e a política de HBA do
   repositório.
2. Execução dos testes de integração com a tag que o CI usa (`-tags integration`) e as variáveis
   de DSN e chave de tenant.

## 3. Resultado (executado nesta máquina, Windows)

| Teste | Resultado |
| --- | --- |
| `TestDistributedPostgresRLSAndEvents` | **PASS** (4,43 s) |
| `TestDistributedPostgresRuntimeRejectsRLSDrift` | **PASS** (7,68 s) |
| `TestDistributedPostgresTenantKeyRotation` | **PASS** (1,64 s) |

```
go test -tags integration ./internal/agent/ \
  -run 'TestDistributedPostgresRLSAndEvents|TestDistributedPostgresRuntimeRejectsRLSDrift|TestDistributedPostgresTenantKeyRotation' \
  -count=1 -v
→ ok  github.com/ollama/ollama/internal/agent  17,627s
```

## 4. A prova de NÃO-INFERÊNCIA já está no teste

Inspecionando o que o teste executa contra o banco real:

- **Oráculo por contagem**: `SELECT COUNT(*) FROM agent_missions WHERE id=$1` devolve **0** para a
  missão de OUTRO tenant, em transação crua e pelo caminho do runtime — a contagem **não revela**
  a linha alheia;
- **Colisão de identidade**: uma tentativa de gravar sobre a missão de outro tenant é recusada
  (`must not overwrite another tenant`), ou seja, o erro **não** serve para confirmar que a
  identidade existe no outro tenant;
- **Referência cruzada**: o banco **não aceita** evento apontando para missão de outro tenant;
- **Sem contagem global**: o papel de runtime não consegue contar todas as missões.

Além disso, a política é criada com `ENABLE` **e** `FORCE ROW LEVEL SECURITY`, o papel de runtime
é `NOBYPASSRLS` e o contexto do tenant é **assinado** (`agent_tenant_context_matches`), então a
política não depende de um `SET` que o chamador poderia escolher.

## 5. O que continua não medido aqui

- O trecho do teste que depende de **Redis** foi pulado nesta execução
  (`OLLAMA_AGENT_TEST_REDIS_URL is not configured`): fila/DLQ distribuídos ficam com a evidência
  do **CI**, que sobe o Redis e exporta a URL.
- Não há prova de **não-inferência por tempo de execução** (timing): medir diferença de latência
  exige repetição estatística e um ambiente controlado; o que existe é a prova de que a
  **contagem** e as **colisões** não revelam dados alheios.
- A pilha local usa senhas de teste efêmeras, **nunca** versionadas; a pilha foi **derrubada**
  depois da medição.

## 6. Conclusão honesta

O estágio 18 deixa de ser "pendente por falta de banco": a RLS e a não-inferência por contagem e
colisão foram **verificadas contra PostgreSQL real** nesta máquina, além do CI que já cobria esse
caminho. Permanecem como limites declarados a inferência por **timing** e a parte de **Redis**
(que tem evidência no CI).
