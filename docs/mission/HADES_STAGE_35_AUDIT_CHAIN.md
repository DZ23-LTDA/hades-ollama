# Estágio 35 — a trilha de auditoria ganhou cadeia de hashes (E-147)

## 1. O limite que esta rodada fecha

O estado declarava: *"a trilha não é assinada nem encadeada criptograficamente — um operador com acesso ao
disco pode editá-la"*.

Agora cada registro carrega `PreviousHash` e `Hash`, formando uma **cadeia**: editar, remover ou reordenar
um registro passa a ser **detectável** por `VerifyTrail`, e a verificação acompanha o `AuditStatus` (logo,
a rota que já existe a expõe).

## 2. O que a cadeia dá — e o que ela NÃO dá

| Dá | Não dá |
| --- | --- |
| **Detecção** de conteúdo alterado, registro removido ou reordenado | **Prevenção**: quem tem acesso ao disco pode reescrever a trilha inteira e recalcular a cadeia |

Detectar já é a diferença entre *"a trilha pode ter sido editada sem deixar rastro"* e *"a edição aparece na
verificação"*. A **assinatura com chave** (que impediria o recálculo) é o passo seguinte e **não** está
feita.

## 3. Duas decisões de honestidade

1. **Recusa em estender trilha adulterada.** `Append` verifica a cadeia antes de gravar: continuar
   escrevendo sobre uma trilha já editada produziria um documento que **parece íntegro a partir dali**,
   escondendo a adulteração anterior. A recusa vira `ErrServiceAgentAuditChainBroken` e aparece em
   `turn.AuditError`.
2. **Adulteração ≠ idade.** Registros escritos antes da cadeia (sem hash) **não** são declarados válidos —
   mas também **não** são acusados de adulteração: `Broken` (evidência de edição) foi separado de `Legacy`
   (não verificável). Sem essa separação, uma trilha antiga **congelaria a auditoria para sempre** — e foi
   exatamente o que meu primeiro teste pegou, porque eu havia escrito `Append` recusando qualquer trilha
   `!Valid`.

## 4. O defeito que só apareceu ao repetir a execução

A suíte completa falhou uma vez em `TestServiceAgentAuditIsolationHoldsOnDisk`; na execução seguinte passou.
**Repeti o teste 30 vezes e ele falhou 3** — um flake de ~1/16.

Causa: eu afirmava o isolamento procurando a **substring** `aud_b` no texto do arquivo. O ID gerado é
`aud_` + os primeiros 12 caracteres de um UUID; quando o primeiro caractere sorteado é `b`, o ID
**legítimo** de `org-a` contém `aud_b`. A asserção acusava isolamento violado onde não havia.

Corrigido para conferência **estrutural** (ler o JSON e comparar os CAMPOS `OrganizationID` e `ToolName`).
Prova: **50/50 execuções** do teste antes flaky, **50/50** do teste de teto e **120/120** dos seis testes de
cadeia, além de **duas suítes completas seguidas** (`ok 63,709s` e `ok 63,487s`).

**Lição institucionalizada:** asserção por substring em identificador aleatório é armadilha — compare o
**campo**, não o texto. E `-count=N` é o instrumento que revela flake que uma execução só esconde; foi
também assim em E-142 (o teste que só falhava 3 s antes da virada da hora).

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestServiceAgent` | **30/30 PASS** (6 novos de cadeia) |
| `go test ./internal/agent/ -count=1` (2×) | **ok 63,709s** e **ok 63,487s** |
| `go test ./server/ -count=1` | **ok 22,570s** |
| `-count=50` no teste antes flaky · `-count=20` nos 6 de cadeia | **50/50 · 120/120** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 20,713s · server 5,919s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

Os seis testes novos: cadeia intacta verifica; **conteúdo adulterado** detectado com o registro apontado;
**registro removido** detectado; **reordenação** detectada; **recusa** em estender cadeia quebrada; e
registros **anteriores à cadeia** não declarados válidos (com a limitação explícita, e sem bloquear a
gravação futura).

## 6. Limites declarados

- **Sem assinatura com chave**: a cadeia detecta edição, não impede recálculo completo por quem controla o
  disco. É o próximo passo natural, e depende de gestão de chave (a criptografia de credenciais já é opt-in
  por `OLLAMA_AGENT_CREDENTIAL_KEY`).
- **Sem exportação** da trilha e **sem retenção por tempo** (o teto é por quantidade).
- **Gravação síncrona** no turno, escolhida em troca de latência para não perder registro em queda abrupta.
- **O teto por organização corta as entradas mais antigas**: a cadeia permanece coerente para o que fica,
  mas o que saiu do teto não é verificável depois — declarado, não escondido.
