# Estágio 34 — teto no cache de eventos ricos (E-174)

## 1. O último mapa sem limite

`eventsByKey` guardava o **evento completo** de cada mensagem aceita, e não tinha teto: num processo de longa
duração, **cada mensagem ficava na memória para sempre**. Era o último mapa sem limite do omnichannel — os
outros já tinham sido fechados nas rodadas anteriores (DLQ, janela de rate limit, conectores, índice de dedupe).

## 2. Por que agora é seguro limitar

Limitar este cache **antes** de E-172 seria perigoso: a dedupe dependia dele, então descartar um payload antigo
reabriria a janela de duplicação. Com o **índice de dedupe persistido**, a memória de "já entrou" vive no disco, e
o cache de eventos ricos passa a ser apenas uma **comodidade**: serve para devolver o evento completo numa
duplicata recente.

| Peça | Papel | Limite |
| --- | --- | --- |
| `dedupeEntries` (E-172) | **proteção** contra duplicação | retenção de 7 dias + teto de 5000/org |
| `eventsByKey` (E-174) | **conveniência**: payload completo | `OmnichannelMaxEventsPerOrg = 2000` |

## 3. O que foi feito

- `OmnichannelMaxEventsPerOrg` (2000) limita o cache **por organização**;
- `pruneEventsLocked` descarta os **mais antigos** primeiro, chamado a cada mensagem aceita;
- o teto é **por organização**: o volume de um tenant **não** expulsa o evento de outro;
- a variável é `var` (e não `const`) por um motivo **declarado**: o teste precisa exercitar a poda sem fazer
  milhares de ingestões.

## 4. Testes

| Teste | O que trava |
| --- | --- |
| `…EventsCacheRespectsTheCap` | o cache respeita o teto; **os mais antigos saem** e os **mais recentes ficam** |
| `…EventsCacheCapKeepsDedupeWorking` | com o **payload fora do cache**, a duplicata **continua** sendo reconhecida — é isto que torna o teto seguro |
| `…EventsCacheCapIsPerOrganization` | o volume de uma organização **não** expulsa o evento de outra |

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestOmnichannelEventsCache -count=1` | **3/3 PASS** |
| `go test ./internal/agent/ ./server/ -run '...'` | **185/185 PASS** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 6. Limites declarados

- Numa duplicata cujo payload **já saiu do cache**, a resposta traz o **registro persistido** (id, canal,
  contato, tipo, horário) com `Deduplicated=true` — não o evento completo original. É a mesma troca de E-172.
- O teto é **por organização e por processo**; 2000 é um valor escolhido, **não** medido contra um perfil de
  memória real (não há medição de RSS por tenant nesta rodada).
- A poda percorre os eventos da organização a cada mensagem aceita: com o teto em 2000 o custo é limitado, mas
  **não** foi medida a latência dessa varredura sob carga.
- `OmnichannelMaxEventsPerOrg` é `var`: um teste (ou um chamador) pode alterá-la em tempo de execução. O valor de
  produção é 2000 e **não** há configuração por ambiente exposta.
