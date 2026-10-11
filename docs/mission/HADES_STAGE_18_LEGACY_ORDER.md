# Estágio 18 — o ciclo legado exige ORDEM, não um passo isolado (E-123)

## 1. O que eu tentei fechar

Na medição anterior (E-122) a suíte distribuída local fechou em **28 PASS / 1 FAIL / 2 SKIP**, e a
única falha era `TestDistributedPostgresRuntimeReadinessForResidualLegacyContract`, por **fixture de
papel legado ausente**. Eu tinha declarado o procedimento e decidi executá-lo.

## 2. O que a tentativa revelou

O teste (lido por dentro) **cria a própria fixture** (`CREATE ROLE ollama_agent_legacy_fixture
NOLOGIN`, `GRANT ollama_agent TO …`, `ALTER ROLE ollama_agent CREATEDB`) e usa apenas os DSNs de
**runtime** e **admin**. Logo, o único pré-requisito externo era o papel **`ollama_agent`** existir
— exatamente como o CI o cria (`CREATE ROLE ollama_agent LOGIN SUPERUSER …`).

Criei o papel e rodei a suíte:

| Estado | PASS | FAIL | SKIP |
| --- | --- | --- | --- |
| Sem o papel legado (E-122) | 28 | **1** | 2 |
| **Com** o papel legado criado isoladamente | **25** | **5** | 2 |

**Criar o papel piorou o resultado**: os testes de RLS (`TestDistributedPostgresRLSAndEvents`,
`TestDistributedPostgresRuntimeRejectsRLSDrift`) passaram a **falhar**, junto com os dois de
prontidão legada. Ou seja: a suíte **não** é um conjunto de testes independentes contra um estado
arbitrário — ela depende do **ciclo de vida ordenado** que o CI executa
(*ACL fresco → papel legado → migração → retirada do legado*). Foi por isso que o CI tem **passos
explícitos** para o legado, e não uma única criação de papel.

## 3. Estado restaurado e remedido

Removi a fixture (`DROP ROLE ollama_agent`) e **rodei a suíte completa de novo**:

```
go test -tags integration ./internal/agent/ -run TestDistributed -count=1 -v
→ PASS=29  FAIL=1  SKIP=2
```

**29 PASS**, com a **mesma** falha de antes (`...ForResidualLegacyContract`) e os dois skips
conhecidos (`...ForStaleLegacySession` e `...CannotConnectOutsideApplicationDatabase`, que exigem
DSNs adicionais que o CI provê).

## 4. Correção de um diagnóstico meu

Numa checagem intermediária o teste de RLS apareceu falhando com
*"OLLAMA_AGENT_TEST_POSTGRES_RUNTIME_URL, …MIGRATOR_URL, and a 64+ byte hex …TENANT_KEY are
required"*. A causa era **minha**: eu havia omitido `OLLAMA_AGENT_TEST_POSTGRES_MIGRATOR_URL`
naquela execução. Vale registrar o comportamento: o teste **falha** quando **alguns** DSNs estão
presentes e outros não, em vez de pular — a mensagem é clara, mas a robustez poderia ser melhor
(mesma observação feita em E-122).

## 5. Conclusão honesta

- O que eu **consegui**: medir a suíte distribuída local com **29 PASS** e declarar exatamente o
  que falta.
- O que eu **não** consegui: deixar a suíte 100% verde aqui. E agora sei **por quê**: o caminho
  legado exige a **ordem** do CI, não a criação de um papel. Reproduzir isso significa orquestrar a
  sequência (ACL fresco → legado → migração → retirada), que é o que o job do CI faz em passos
  próprios.
- **Procedimento exato de desbloqueio** (para a próxima rodada): seguir os passos do job
  `PostgreSQL RLS, Redis DLQ and OTLP integration` na **mesma ordem**, incluindo a criação do papel
  legado **antes** da migração e a **retirada** (`retire-legacy-role.sql`) depois — não basta criar
  o papel.

Senhas da pilha local foram **efêmeras e não versionadas**; a pilha foi **derrubada** ao final.
