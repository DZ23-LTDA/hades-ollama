## JANELA E-1387

**A cabeca do indice e' E-1387.** Este bloco substitui a janela anterior (E-1363) e cobre o que foi feito
desde entao. O documento seguinte continua a ser o historico; este e' o que se le' primeiro.

### O QUE MUDOU ENTRE E-1364 E E-1376

- **E-1364** — a confirmacao local do MFA protege contra o engano, nao contra o ataque.
- **E-1365** — modelo canonico de persona: `AgentPersona` com validacao, hash de conteudo e render por
  ferramenta. *Nao avaliado e um estado, e e diferente de aprovado.*
- **E-1366** — ledger por organizacao com `requireOrganization` em `Record`, `Load` e `Entries`. *Leitura
  tambem e scoped: ver o ledger de outra organizacao e tao grave como escrever nele.*
- **E-1367** — surface HTTP do Agent Center (`GET /agents`, `POST /agents/install`, `GET /agents/drift`).
- **E-1368** — catalogo com 6 personas HADES nativas, ordem estavel. *Um catalogo que muda de ordem sozinho
  faz o painel parecer que mudou alguma coisa.*
- **E-1369** — harness de avaliacao: 10 estados, transicoes, suites com POSITIVE/REFUSAL/INJECTION.
  *A ausencia da capacidade e a garantia mais forte que existe.*
- **E-1370** — equipas e presets: `Why`/`Deliverable`/`MissingTools` por papel, ciclo recusado por DFS, 4
  presets **todos `Functional:false`**.
- **E-1371** — backup e rollback: id derivado do CONTEUDO, `Restore` valida tudo antes de escrever.
  *Um rollback que restaura metade e um segundo defeito.*
- **E-1372** — ingestao segura: padrao literal com campo+trecho+motivo, `QUARANTINE` na maquina de estados.
- **E-1373** — quarentena PERSISTENTE: sobrevive a reabertura, `Release` exige revisor e leva so' a
  `SECURITY_REVIEWED`. *Quarentena e um estado, nao um log.*
- **E-1374** — **o ciclo completo apanhou um defeito P2 de PERDA DE DADOS**: `Entries`/`Record` liam o mapa e
  nao o disco -- um ledger reaberto dizia zero e um `Record` novo apagava as entradas anteriores. Corrigido
  com `loadLocked`. *Uma lista vazia e um erro nao sao a mesma coisa.*
- **E-1375** — harness contra o motor REAL: `qwen2.5:7b`, **estado=EVALUATED_FAILED**, positivos passam e
  negativas falham. *Um positivo sem o negativo ao lado prova que algo aconteceu, nao que aconteceu o que se
  diz.*
- **E-1376** — a UI corre a avaliacao: `POST /agents/evaluation/run`, `inference_available` medido ao vivo.
- **E-1377** — gate de contrato painel<->servidor, e o painel que lia `catalog_note` de um servidor que
  emitia `note`: mostrava «catalogo nao ligado» sempre. *Um painel que le uma chave que ninguem emite nao
  mostra o vazio: mostra o fallback.*

### ESTADO NO HEAD

`2d2e1da2e` (antes deste commit) · branch `fix/checker-relogio` · PR #438 · **1169** entradas
`go test ./internal/agent` ok 414s · `go test ./server` ok · `go vet` 0 · 20 vitest · `tsc -b` 0
**362 rotas / 16 orfas / 0 lacuna (teto 0)** · mission/enc/non_inference 0/0/0

### O QUE FALTA (dito)

1. A UI a falar com o servidor **num navegador real** -- painel provado contra mock e servidor provado com
   `httptest` nao provam que um fala com o outro.
2. `Evaluate` correu contra `qwen2.5:7b` e **reprovou** a recusa e a injecao: o modelo nao esta' apto para
   essas personas. Nao e' um bug do harness -- e' a medicao.
3. Android/iOS `BLOCKED_EXTERNAL`; Windows RC `NotSigned`.
4. Gitleaks historico (E-1217/E-1218) -- decisao do proprietario.
5. FASE 0/1: 19 linhas `ADAPTAR` continuam hipoteses nao medidas.

Veredito: **NOT_RELEASE_READY**.

## JANELA E-1363

**A LACUNA DE PRODUTO FECHOU: 93 -> 0.** A cabeça do índice é **E-1363**.

- **E-1363** — `PluginBundlePanel` consome `POST /plugins/bundle` (**201**). O servidor resolve o caminho
  com `filepath.Abs` + **`EvalSymlinks` ANTES de comparar** + `Rel`; **403 desligado** (falta
  `OLLAMA_AGENT_PLUGIN_BUNDLE_DIR`) **≠ 403 caminho recusado** — estados distintos na UI. **34.ª medição:
  previ 1, mediu 1.**
- **E-1362** — **ACHADO P1: MFA sem reautenticação**; painel declara em vez de fingir; 4 → 1.
- **E-1361** — inbox: 201 nova ≠ 200 reentrega; dedup só por `external_id`; 5 → 4.
- **E-1360** — empresa: 422 traz o relatório; 7 → 5.

## O GATE

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **16** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **0** |

**`TETO_ORFAS = 0`.** *Qualquer rota nova que o backend ganhe e que o frontend não consuma falha
imediatamente, porque o número de hoje é zero.*

**ZERO NÃO QUER DIZER PRODUTO PRONTO.** Quer dizer que **todas as 348 rotas têm superfície no frontend ou
são consumidor externo deliberado**. *Um gate a zero mede uma propriedade; não absolve o resto.*

## O BUG QUE ISTO EXPÔS

O gate tinha **duas atribuições `TETO_ORFAS`** — L231 `= 0` e L316 `= 1`. **O Python usa a última.** Cada
`str.replace` corrigia a primeira e **deixava a segunda a valer**, então os valores medidos e os valores
declarados divergiam **sem que nenhum `assert` falhasse**.

*Um ficheiro com duas atribuições da mesma constante não tem dois valores: tem um valor e um comentário
que se lhe parece. O `assert` procurava a substring e encontrava as duas.*

Reparado: **uma só atribuição, `py_compile` antes de correr o gate**. *Um gate que não arranca não é um
gate: é um ficheiro com nome de gate.*

## O QUE CONTINUA ABERTO

- **Build NÃO assinado e NÃO instalado.**
- `sign.start` / `publish.request_approval` / `submit` / `rollback` **inexistentes**.
- **`gitleaks` histórico — a única falha de CI.**
- Instalação limpa `BLOCKED_EXTERNAL`; Android sem toolchain.
- **Sem catraca contra testes flaky.**
- **FASE 30 (Agent Center) não iniciada**; **FASE 17.1 Obsidian 4/12**.
- **Achado P1: MFA sem reautenticação.**
- O transporte descarta corpo **e** status de erro.
- O A1 da DZ23 nunca foi avaliado.

**Veredito: `NOT_RELEASE_READY`.**

---
## JANELA E-1362

A cabeça do índice é **E-1362**.

- **E-1362** — **ACHADO P1: nenhum dos três handlers de MFA exige reautenticação.** `MfaPanel` cobre
  `/auth/mfa/enable`, `/disable` e `/recovery/generate`. Os três verificam **apenas** `agent.user`;
  `disableMFA` e `generateRecoveryCodes` **nem corpo têm** — um bearer token válido desativa o MFA e gera
  códigos novos com um pedido vazio. *Um segundo factor que qualquer portador do primeiro pode remover não
  é um segundo factor: é uma cerimónia.* **Registado como finding, não corrigido** (a correcção é no
  servidor). O painel **não finge**: exige confirmação, **declara** que é trava de clique, e mostra o
  achado no topo. **33.ª medição: previ 3, mediu 3** (4 → 1).
- **E-1361** — inbox: 201 nova vs 200 reentrega; dedup só por `external_id`; 5 → 4.
- **E-1360** — empresa: fábrica NÃO abre; 422 traz o RELATÓRIO; isolamento precisa de 2.ª org; 7 → 5.
- **E-1359** — aprovadores: ato mais sensível; lista vazia = portão ABERTO; 8 → 7.

## A distância

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **17** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **1** (era 93) |

## A ÚLTIMA ROTA

`plugins/bundle` — **`POST /plugins/bundle`**, **201 CREATED**. **Fecha fechado (403)** sem
`OLLAMA_AGENT_PLUGIN_BUNDLE_DIR`. `ResolvePluginBundlePath` faz `filepath.Abs` em ambos e
**`filepath.EvalSymlinks` antes de comparar** — *um caminho que passa por um symlink não é o caminho que
se vê*. Corpo: `{manifest: PluginManifest, bundle_path, signature: SignedArtifact}`.

## O ACHADO P1, escrito por extenso

| operação | corpo | reautenticação |
|---|---|---|
| `/auth/mfa/enable` | `{secret}` | **nenhuma** |
| `/auth/mfa/disable` | **vazio** | **nenhuma** |
| `/auth/mfa/recovery/generate` | **vazio** | **nenhuma** |

A correcção correcta é **no servidor**: exigir senha, TOTP actual ou código de recuperação antes de
desactivar o MFA e antes de gerar códigos novos. **Não é corrigível nesta fatia** — e um painel que pedisse
a senha para a enviar a um servidor que a ignora seria **teatro de segurança**.

O que continua aberto: **1 rota**; **build não assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; **`gitleaks` histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **sem
catraca contra testes flaky**; **FASE 30 não iniciada**. Achados abertos: **MFA sem reautenticação (P1,
NOVO)**; o transporte descarta corpo e status de erro; o A1 nunca foi avaliado.

---
## JANELA E-1361

A cabeça do índice é **E-1361**.

- **E-1361** — **201 nova e 200 reentrega são estados distintos.** `InboxIngestPanel` consome
  `POST /inbox/messages`. O handler devolve **201** quando cria e **200** quando `Deduplicated` — a mesma
  entrega duas vezes devolve a original, não duplica. *Uma reentrega contada como mensagem nova inflaciona
  a caixa de entrada — e a caixa inflacionada faz o utilizador procurar uma conversa que nunca aconteceu.*
  **A dedup é só pelo `external_id`**, e o servidor não inventa a chave. **32.ª medição: previ 1, mediu 1**
  (5 → 4). `InboxPanel` existe mas **não consome** esta rota — sem duplicação (verificado).
- **E-1360** — empresa: fábrica NÃO abre (artefactos+checklist); 422 traz o RELATÓRIO; isolamento precisa
  de 2.ª org; 7 → 5.
- **E-1359** — aprovadores: ato mais sensível; lista vazia = portão ABERTO; duas identidades; 8 → 7.
- **E-1358** — push token com `json:"-"`; 501 ≠ 503; fase terminal de CRM não reabre; 10 → 8.

## A distância

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **20** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **4** (era 93) |

## AS 4 RESTANTES

| domínio | n | rotas |
|---|---|---|
| `auth` | 3 | `/auth/mfa/disable` · `/auth/mfa/enable` · `/auth/mfa/recovery/generate` |
| `plugins` | 1 | `/plugins/bundle` |

**A fatia de MFA é própria:** as três tocam credenciais, devem **recusar sem reautenticação** e exigem
teste de negação explícito. As três partilham o prefixo `/auth/mfa/` — **o gate conta 3 rotas, e são 3
prefixos distintos** (`/auth/mfa/disable`, `/enable`, `/recovery`) ⇒ **3 pontos**.

**`/plugins/bundle` (1):** **fecha fechado (403)** sem `OLLAMA_AGENT_PLUGIN_BUNDLE_DIR`.
`ResolvePluginBundlePath` faz `filepath.Abs` em ambos e **`filepath.EvalSymlinks` antes de comparar** —
*um caminho que passa por um symlink não é o caminho que se vê*. **201 CREATED** no sucesso.

## Lição desta ronda, corrigida a meio

Um teste exigia que a reentrega **não** contivesse «criada» — mas o texto diz *«não foi criada outra»*.
**A negação é o que se quer.** A asserção passou a exigir a ausência do marcador `· nova ·`.
*Uma asserção que proíbe a palavra que a própria negação usa está a testar o dicionário, não o estado.*

O que continua aberto: **4 rotas** sem UI; **build não assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; **`gitleaks` histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **sem
catraca contra testes flaky**; **FASE 30 não iniciada**. Achados abertos: **o transporte descarta corpo e
status de erro**; **o A1 nunca foi avaliado**.

---
## JANELA E-1360

A cabeça do índice é **E-1360**.

- **E-1360** — **A fábrica não abre a empresa; e o 422 traz o relatório junto.** `CompanyCockpitPanel`
  consome `POST /company/factory/launch` (**201**) e `POST /business/certify` (**200**/**422**). O handler
  diz: *«a fábrica NÃO cria cliente nem emite registro legal»*. *Um botão que promete «abrir a empresa» e
  entrega um ZIP de marca não abriu a empresa: entregou o prospecto.* O 422 devolve `report` **E**
  `error`: *um relatório que só aparece quando tudo corre bem é um prémio, não um diagnóstico.* A prova de
  isolamento **sempre** usa uma **segunda organização**. **31.ª medição: previ 2, mediu 2** (7 → 5).
- **E-1359** — aprovadores: ato mais sensível; lista vazia = portão ABERTO; duas identidades; 8 → 7.
- **E-1358** — push token com `json:"-"`; 501 ≠ 503; fase terminal de CRM não reabre; 10 → 8.
- **E-1357** — media/ocr+tone: caminho contido no workspace; 502 ≠ erro do pedido; 12 → 10.

## A distância

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **21** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **5** (era 93) |

## AS 5 RESTANTES

| domínio | n | rotas |
|---|---|---|
| `auth` | 3 | `/auth/mfa/disable` · `/auth/mfa/enable` · `/auth/mfa/recovery/generate` |
| `inbox` | 1 | `/inbox/messages` |
| `plugins` | 1 | `/plugins/bundle` |

Nenhuma partilha prefixo: **cada uma renderá exactamente 1**.

## Lições já lidas nos handlers restantes

- `/inbox/messages`: **201 nova / 200 reentrega** via `external_id` — *webhook duplicado não vira
  mensagem duplicada*.
- `/plugins/bundle`: **fecha fechado** (403) sem `OLLAMA_AGENT_PLUGIN_BUNDLE_DIR`; `ResolvePluginBundlePath`
  resolve o caminho dentro do directório permitido — *para que a API nunca vire leitura arbitrária de
  arquivo do host*.
- `/auth/mfa/*`: **fatia própria** — credenciais, reautenticação, teste de negação.

O que continua aberto: **5 rotas** sem UI; **build não assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; **`gitleaks` histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **sem
catraca contra testes flaky**; **FASE 30 não iniciada**. Achados abertos: **o transporte descarta corpo e
status de erro**; **o A1 nunca foi avaliado**.

---
## JANELA E-1359

A cabeça do índice é **E-1359**.

- **E-1359** — **Definir quem aprova é o ato mais sensível, e o próprio código o diz.** `ApproversPanel`
  consome `PUT /service-agents/approval-approvers`. O E-429 registou o incidente: um `RoleViewer` redefiniu
  a lista para `["eu-mesmo"]` e recebeu **HTTP 200**; agora `requireApprovalApprover` exige **owner/admin**
  e a trilha grava o ator **da sessão**. *Definir quem aprova é definir quem pode autorizar tudo o resto.*
  **Lista vazia não é «sem aprovadores»: é «sem checagem de papel»** (`role_checked: len(approvers) > 0`).
  **Duas identidades:** `changed_by` do corpo vs trilha da sessão. **A trilha pode falhar** e a resposta
  declara (`audit_recorded`/`audit_reason`). **30.ª medição: previ 1, mediu 1** (8 → 7).
- **E-1358** — push token com `json:"-"`; 501 ≠ 503; fase terminal de CRM não reabre; 10 → 8.
- **E-1357** — media/ocr+tone: caminho contido no workspace; 502 ≠ erro do pedido; 12 → 10.
- **E-1356** — waitlist+no-show: faltar ≠ cancelar; fila vazia = `offered:false`; 15 → 12.

## A distância

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **23** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **7** (era 93) |

## AS 7 RESTANTES

| domínio | n | rotas |
|---|---|---|
| `auth` | 3 | `/auth/mfa/disable` · `/auth/mfa/enable` · `/auth/mfa/recovery/generate` |
| `inbox` | 1 | `/inbox/messages` |
| `business` | 1 | `/business/certify` |
| `company` | 1 | `/company/factory/launch` |
| `plugins` | 1 | `/plugins/bundle` |

Nenhuma partilha prefixo: **cada uma renderá exactamente 1**.

## Superfícies adjacentes não duplicadas

`ApprovalsPanel` (existente) **aprova decisões** (`/service-agents/approvals`); `ApproversPanel` (novo)
**define quem aprova** (`/service-agents/approval-approvers`). **São dois atos distintos**: um decide um
caso, o outro decide quem decide. Sem colisão de `data-testid`.

## Lições já lidas nos handlers restantes

- `/inbox/messages`: **201 nova, 200 reentrega** via `external_id` — *webhook duplicado não vira mensagem
  duplicada*.
- `/plugins/bundle`: **fecha fechado** (403) sem `OLLAMA_AGENT_PLUGIN_BUNDLE_DIR`.
- `/business/certify`: corpo **opcional**; a prova de isolamento **sempre precisa de uma SEGUNDA
  organização** — *um teste de isolamento com uma só organização prova que uma organização se vê a si
  mesma*.
- `/company/factory/launch`: a fábrica **não cria cliente nem emite registro legal**.
- `/auth/mfa/*`: **fatia própria** — credenciais, reautenticação, teste de negação.

O que continua aberto: **7 rotas** sem UI; **build não assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; **`gitleaks` histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **sem
catraca contra testes flaky**; **FASE 30 não iniciada**. Achados abertos: **o transporte descarta corpo e
status de erro**; **o A1 nunca foi avaliado**.

---
## JANELA E-1358

A cabeça do índice é **E-1358**.

- **E-1358** — **O token de push não tem como sair; a fase terminal de CRM não reabre.**
  `push.go` tem `Token string `json:"-"`` — **não tem como ser serializado**; o handler ainda redige. *Um
  campo que não se pode serializar não precisa de ser redigido; redigi-lo na mesma é dizer que quem
  escreveu o handler sabia que o campo existia.* **501 ≠ 503**: sem serviço de push é configuração em
  falta, não dependência caída. `ErrCRMStageTerminal`: *«reopening would erase the outcome»* — *reabrir
  apagaria o resultado, e o resultado é a única coisa que o funil existe para guardar.* O painel recusa
  **antes** de chamar; o actor vem do token, não do corpo. **29.ª medição: previ 2, mediu 2** (10 → 8).
- **E-1357** — media/ocr+tone: caminho **contido no workspace**; 502 ≠ erro do pedido; 12 → 10.
- **E-1356** — waitlist+no-show: faltar ≠ cancelar; fila vazia = `offered:false`; 15 → 12.
- **E-1355** — finance/orders: receita de pedido não pago recusada; as 3 não devolvem o mesmo status; 18 → 15.

## A distância

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **24** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **8** (era 93) |

## AS 8 RESTANTES

| domínio | n | rotas |
|---|---|---|
| `auth` | 3 | `/auth/mfa/disable` · `/auth/mfa/enable` · `/auth/mfa/recovery/generate` |
| `inbox` | 1 | `/inbox/messages` |
| `service-agents` | 1 | `/service-agents/approval-approvers` (**PUT**) |
| `business` | 1 | `/business/certify` |
| `company` | 1 | `/company/factory/launch` |
| `plugins` | 1 | `/plugins/bundle` |

Nenhuma partilha prefixo com outra: **cada uma renderá exactamente 1**.

## Lições já lidas nos handlers restantes

- `/inbox/messages`: **201 nova, 200 reentrega** — `external_id` deduplica; *webhook duplicado não pode
  virar mensagem duplicada*.
- `/plugins/bundle`: **fecha fechado** (403) sem `OLLAMA_AGENT_PLUGIN_BUNDLE_DIR` — *para que a API
  nunca vire leitura arbitrária de arquivo do host*.
- `/auth/mfa/*`: **fatia própria** — tocam credenciais, exigem reautenticação e teste de negação
  explícito. 3 rotas, 3 pontos.

O que continua aberto: **8 rotas** sem UI; **build não assinado nem instalado**;
`sign.start`/`publish.*` inexistentes; **`gitleaks` histórico — a única falha de CI**; instalação limpa
`BLOCKED_EXTERNAL`; **sem catraca contra testes flaky**; **FASE 30 não iniciada**. Achados abertos: **o
transporte descarta corpo e status de erro**; **o A1 nunca foi avaliado** por `SigningProvider`.

---
## JANELA E-1357

A cabeça do índice é **E-1357**.

- **E-1357** — **O caminho de entrada é contido no workspace — e essa é a fronteira.** `MediaPanel` consome
  `POST /media/ocr` e `POST /media/tone`. O handler do OCR faz `containedPath(workspace, request.InputPath)`:
  o caminho **não é usado como veio**, é resolvido dentro do workspace da missão. *Um caminho que escapa do
  workspace não é um caminho: é uma leitura arbitrária com nome de parâmetro.* O painel recusa o óbvio
  (absoluto, `..`) mas **NÃO reimplementa `containedPath`**: *uma segunda verdade sobre o que está «dentro»
  seria uma segunda fronteira, e a mais fraca é a que fica.*
  **Sem missão não há fronteira** — `mission_id` é obrigatório nos dois.
  **Os dois devolvem 201** porque **criam um Artifact**: *não «correu bem», mas «existe uma coisa nova».*
  **O 502 do OCR não é erro do pedido** — é a dependência a falhar, e o painel diz isso.
  **VIGÉSIMA OITAVA MEDIÇÃO — previsão CERTA:** previ **2**, mediu **2** (12 → 10).
- **E-1356** — waitlist+no-show: faltar ≠ cancelar; fila vazia = `offered:false`, não erro; 15 → 12.
- **E-1355** — finance/orders: receita de pedido não pago recusada; as 3 não devolvem o mesmo status; 18 → 15.
- **E-1354** — scheduling/reschedule: a recusa vem do MOTOR; remarcar ≠ cancelar+criar; tecto medido depois.
- **E-1353** — scheduling/cancel: motivo obrigatório = auditabilidade; id no CORPO; 19 → 18.
- **E-1352** — scheduling/book: E-456 contrato separado; E-458 reentrega ≠ erro; `busy_until` ≠ `ends_at`; 20 → 19.

## A REGRA DO GATE, CONFIRMADA OUTRA VEZ

```python
prefixo = caminho.split(":")[0].rstrip("/") or caminho
if prefixo not in fonte:      # SUBSTRING
```

`/media/ocr` e `/media/tone` **não têm `:param`** → dois prefixos distintos → **duas rotas, dois pontos**.
E `scheduling/reschedule` vs `reschedule/slots` → **um prefixo** → zero. A regra explica todas as medições
desde E-1352.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **26** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **10** (era 93) |

## AS 10 RESTANTES — LISTA EXACTA DO GATE

| domínio | n | rotas |
|---|---|---|
| `auth` | 3 | `/auth/mfa/disable` · `/auth/mfa/enable` · `/auth/mfa/recovery/generate` |
| `notifications` | 1 | `/notifications/register` |
| `crm` | 1 | `/crm/opportunities/:id/stage` |
| `inbox` | 1 | `/inbox/messages` |
| `service-agents` | 1 | `/service-agents/approval-approvers` |
| `business` | 1 | `/business/certify` |
| `company` | 1 | `/company/factory/launch` |
| `plugins` | 1 | `/plugins/bundle` |

**`scheduling` e `media` estão FECHADOS.** As 10 restantes são `auth` (MFA, 3) e sete domínios isolados —
**nenhum deles partilha prefixo com outro**, por isso cada um renderá exactamente 1.

**Vinte e oito medições, doze previsões certas** — as últimas **cinco seguidas**.

O que continua aberto: **10 rotas** sem UI (todas escritas); **20 ORPHAN de 209**; o build **não é assinado
nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks` histórico — a única falha de CI**;
instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra testes flaky**; **FASE 30 não iniciada**.
Achados abertos: **o transporte descarta corpo e status de erro**; **o A1 nunca foi avaliado** por
`SigningProvider`.

## Próximo passo

As de MFA (`/auth/mfa/*`) exigem **fatia própria**: tocam credenciais, devem **recusar sem reautenticação**
e precisam de teste de negação explícito. As outras sete (`notifications/register`, `crm/opportunities/:id/stage`,
`inbox/messages`, `service-agents/approval-approvers`, `business/certify`, `company/factory/launch`,
`plugins/bundle`) são escritas isoladas de 1 ponto cada.

---
## JANELA E-1356

A cabeça do índice é **E-1356**.

- **E-1356** — **Faltar não é cancelar.** `WaitlistOpsPanel` consome `/scheduling/waitlist/{offer,confirm}` e
  `/scheduling/no-show`. O comentário: *«Quem faltou nao e' o mesmo que quem cancelou, e a agenda precisa
  distinguir.»* O painel: **«marcado como FALTOU · agora está `<estado>` · isto não é um cancelamento»**.
  *Dois estados que o servidor distingue e o painel junta são dois estados que o utilizador perdeu.*
  **Fila vazia não é erro:** o servidor responde **200 com `offered: false`** — *«Fila vazia nao e' erro: e'
  ausencia de oferta — e o status tem de dizer isso, nao 500.»* O painel mostra isso **sem cair em erro nem
  em recusa** (testado). O `now` não vai no corpo (testado), e a oferta **expira** — *«o motor decide, nao a
  rota»*.
  **VIGÉSIMA SÉTIMA MEDIÇÃO — previsão CERTA:** previ **3**, mediu **3** (15 → 12).
- **E-1355** — finance/orders: receita de pedido não pago recusada; as 3 não devolvem o mesmo status; 18 → 15.
- **E-1354** — scheduling/reschedule: a recusa vem do MOTOR; remarcar ≠ cancelar+criar; tecto medido depois.
- **E-1353** — scheduling/cancel: motivo obrigatório = auditabilidade; id no CORPO; 19 → 18.
- **E-1352** — scheduling/book: E-456 contrato separado; E-458 reentrega ≠ erro; `busy_until` ≠ `ends_at`; 20 → 19.
- **E-1351** — business/tick: 4 desfechos separados; `errors` vem COM resultado; 21 → 20.

## A REGRA DO GATE, E PORQUE ELA IMPORTA PARA O CUSTO

```python
prefixo = caminho.split(":")[0].rstrip("/") or caminho
if prefixo not in fonte:      # SUBSTRING, não igualdade
```

**O prefixo corta no `:` e a comparação é por substring.** Consequência directa:

| escrito na fonte | fecha |
|---|---|
| `/scheduling/waitlist` | `/waitlist` · `/waitlist/offer` · `/waitlist/confirm` |
| `/finance/orders` | `/revenue` · `/cost` · `/reconcile` |
| `/scheduling/reschedule` | o POST `reschedule` **e** `/reschedule/slots` |

***Uma escrita num prefixo que já contém outros fecha tudo o que ele contém.*** Foi por isso que as últimas
quatro medições acertaram: a regra está no instrumento, e lê-se antes de prever.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **28** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **12** (era 93) |

## AS 12 RESTANTES — LISTA EXACTA DO GATE

| domínio | n | rotas |
|---|---|---|
| `auth` | 3 | `/auth/mfa/disable` · `/auth/mfa/enable` · `/auth/mfa/recovery/generate` |
| `media` | 2 | `/media/ocr` · `/media/tone` |
| `notifications` | 1 | `/notifications/register` |
| `crm` | 1 | `/crm/opportunities/:id/stage` |
| `inbox` | 1 | `/inbox/messages` |
| `service-agents` | 1 | `/service-agents/approval-approvers` |
| `business` | 1 | `/business/certify` |
| `company` | 1 | `/company/factory/launch` |
| `plugins` | 1 | `/plugins/bundle` |

**`scheduling` está FECHADO nas escritas.** Restam `auth` (MFA, 3), `media` (2) e sete domínios isolados.

**Vinte e sete medições, onze previsões certas** — as últimas quatro seguidas, depois de ler o gate.

O que continua aberto: **12 rotas** sem UI (todas escritas); **20 ORPHAN de 209**; o build **não é assinado
nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks` histórico — a única falha de CI**;
instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra testes flaky**; **FASE 30 não iniciada**.
Achados abertos: **o transporte descarta corpo e status de erro**; **o A1 nunca foi avaliado** por
`SigningProvider`.

## Próximo passo

`/media/ocr` e `/media/tone` são as próximas mais simples (criam Artifacts, `201`). As de **MFA** exigem
fatia própria: tocam credenciais e devem **recusar sem reautenticação**, com teste de negação explícito.

---
## JANELA E-1355

A cabeça do índice é **E-1355**.

- **E-1355** — **Receita de pedido não pago é recusada pelo servidor.** `FinanceOrderPanel` consome as
  **três** escritas de `/finance/orders/:id/`. O comentário: *«lanca a receita de um pedido PAGO. Pedido
  aguardando pagamento e' recusado (409): lancar receita dele seria receita ficticia.»* O painel **não
  adivinha** o estado do pagamento — mostra **«O servidor recusou o lançamento: `<motivo>`»**.
  *Receita fictícia é o pior número que um sistema financeiro pode produzir: não é um erro de arredondamento,
  é um valor que entra no relatório e que ninguém vai cobrar.*
  **As três não devolvem o mesmo status:** `revenue` **201 ou 200** (idempotente), `cost` **201 sempre**,
  `reconcile` **200 sempre** e *«liga o pedido à linha do extrato no MESMO lançamento»* — não cria nada.
  **O factor 100 fica visível ANTES do erro:** *«1250 centavos = R$ 12,50»*.
  **VIGÉSIMA SEXTA MEDIÇÃO — previsão CERTA:** previ **3**, mediu **3** (18 → 15).
- **E-1354** — scheduling/reschedule: a recusa vem do MOTOR; remarcar ≠ cancelar+criar; tecto medido depois.
- **E-1353** — scheduling/cancel: motivo obrigatório = auditabilidade; id no CORPO; 19 → 18.
- **E-1352** — scheduling/book: E-456 contrato separado; E-458 reentrega ≠ erro; `busy_until` ≠ `ends_at`; 20 → 19.
- **E-1351** — business/tick: 4 desfechos separados; `errors` vem COM resultado; 21 → 20.
- **E-1350** — reschedule/slots: dois campos ditos por nome; responde ANTES; 23 → 21.

## A REGRA DO GATE — AGORA LIDA NO CÓDIGO

```python
prefixo = caminho.split(":")[0].rstrip("/") or caminho
if prefixo not in fonte:      # SUBSTRING, não igualdade
    orfas.append((metodo, caminho))
```

**O prefixo corta no `:` — o parâmetro de rota.** Por isso:

| rotas | prefixo | rende |
|---|---|---|
| `finance/orders/:id/revenue` · `/cost` · `/reconcile` | `/finance/orders` | **3 de uma vez** |
| `scheduling/book` | `/scheduling/book` | 1 |
| `scheduling/reschedule` + `reschedule/slots` | `/scheduling/reschedule` (substring) | 0 |

***Ler o instrumento antes de prever custa menos do que prever três vezes.***

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **31** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **15** (era 93) |

## AS 15 RESTANTES — LISTA EXACTA DO GATE

| domínio | n | rotas |
|---|---|---|
| `auth` | 3 | `/auth/mfa/disable` · `/auth/mfa/enable` · `/auth/mfa/recovery/generate` |
| `scheduling` | 3 | `/scheduling/no-show` · `/scheduling/waitlist/confirm` · `/scheduling/waitlist/offer` |
| `media` | 2 | `/media/ocr` · `/media/tone` |
| `notifications` | 1 | `/notifications/register` |
| `crm` | 1 | `/crm/opportunities/:id/stage` |
| `inbox` | 1 | `/inbox/messages` |
| `service-agents` | 1 | `/service-agents/approval-approvers` |
| `business` | 1 | `/business/certify` |
| `company` | 1 | `/company/factory/launch` |
| `plugins` | 1 | `/plugins/bundle` |

**Vinte e seis medições, dez previsões certas** — e as três últimas foram certas depois de eu ler o gate.

O que continua aberto: **15 rotas** sem UI (todas escritas); **20 ORPHAN de 209**; o build **não é assinado
nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks` histórico — a única falha de CI**;
instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra testes flaky**; **FASE 30 não iniciada**.
Achados abertos: **o transporte descarta corpo e status de erro** (o 409 do E-1352 e o 201-vs-200 daqui);
**o A1 nunca foi avaliado** por `SigningProvider`.

---
## JANELA E-1354

A cabeça do índice é **E-1354**.

- **E-1354** — **A rota não decide se cabe — o motor decide.** `RescheduleMutationPanel` consome
  `POST /scheduling/reschedule`. O comentário: *«A recusa do motor e' preservada: sobreposicao, fechamento e
  turno continuam sendo decididos por ele, nao por esta rota.»* O painel valida **só** o identificador e o
  RFC 3339; a recusa aparece como **«O motor recusou este instante: <motivo>»**, com teste a verificar que
  não cai no erro genérico. *Uma rota que decide o que cabe duplica o motor com menos informação — o cliente
  não sabe o turno, nem os fechamentos, nem quem mais marcou naquele minuto.*
  **Remarcar não é cancelar+criar:** o painel diz *«É o MESMO agendamento (ag-9) noutro instante»*, e o
  teste confirma que o corpo **não** leva `reason` nem `resource_id`.
  **VIGÉSIMA QUINTA MEDIÇÃO — previsão ERRADA:** previ **1**, mediu **0** (`reschedule/slots` do E-1350 já
  fechara o prefixo). **E baixei o tecto antes de medir — o gate acusou; tecto reposto a 18.**
- **E-1353** — scheduling/cancel: motivo obrigatório = auditabilidade; id no CORPO; 19 → 18.
- **E-1352** — scheduling/book: E-456 contrato separado; E-458 reentrega ≠ erro; `busy_until` ≠ `ends_at`; 20 → 19.
- **E-1351** — business/tick: 4 desfechos separados; `errors` vem COM resultado; 21 → 20.
- **E-1350** — reschedule/slots: dois campos ditos por nome; responde ANTES; 23 → 21.
- **E-1349** — `OpenSlots` ≠ `Availability`; 0 slots ≠ dia livre; 24 → 23.

## A REGRA DO GATE, MEDIDA

**O gate agrupa por PREFIXO DE CAMINHO, sem o método.**

| rota | prefixo | rende |
|---|---|---|
| `scheduling/book` | `scheduling/book` | **1** |
| `scheduling/cancel` | `scheduling/cancel` | **1** |
| `scheduling/reschedule` + `reschedule/slots` | `scheduling/reschedule` | **0** (já fechado) |

*`book`, `cancel` e `reschedule` são **três** prefixos; `reschedule` e `reschedule/slots` são **um**. Foi por
não distinguir isto que a previsão falhou duas vezes seguidas.*

**E a segunda lição: o tecto mede-se DEPOIS.** Baixar antes de medir é adivinhar — e o gate que existe para
travar o próximo aumento passa a travar o trabalho feito.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **34** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **18** (era 93) |

## AS 18 RESTANTES — TODAS ESCRITAS

`scheduling` 3 (`no-show`, `waitlist/offer`, `waitlist/confirm`, `waitlist`, `reminders/sent`) · `auth` 3
(MFA) · `finance` 3 · `media` 2 · `business` 1 (`certify`) · `notifications` 1 · `crm` 1 · `inbox` 1 ·
`service-agents` 1 (PUT) · `company` 1 · `plugins` 1.

**Vinte e cinco medições, nove previsões certas** — e a regra que faltava está agora escrita acima e no
próprio tecto.

O que continua aberto: **18 rotas** sem UI (todas escritas); **20 ORPHAN de 209**; o build **não é assinado
nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks` histórico — a única falha de CI**;
instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra testes flaky**; **FASE 30 não iniciada**.
Achados abertos: **o transporte descarta o corpo do 409**; **o A1 nunca foi avaliado** por `SigningProvider`.

---
## JANELA E-1353

A cabeça do índice é **E-1353**.

- **E-1353** — **O motivo é obrigatório, e o próprio handler diz porquê.** `CancellationPanel` consome
  `POST /scheduling/cancel`. O comentário: *«Escrita com motivo obrigatorio — cancelamento sem motivo nao e'
  auditavel.»* Não é validação de conveniência: *um cancelamento sem motivo explica o QUE aconteceu e não o
  PORQUE — e o porquê é a única parte que alguém vai querer ler seis meses depois.* O painel recusa **antes**
  de chamar, com `expect(mockAgentFetch).not.toHaveBeenCalled()`.
  **O `appointment_id` vai no CORPO, não no caminho** — com teste da ausência (`expect(caminho).not.toContain("ag-9")`).
  *Um id concatenado num caminho é um id que pode reescrever o caminho.*
  **VIGÉSIMA QUARTA MEDIÇÃO — previsão ERRADA:** previ **0** (o `book` já teria fechado o prefixo `scheduling`),
  mediu **1** (19 → 18). *O gate agrupa por PREFIXO mas LISTA por ROTA — duas propriedades, e eu confundi-as.*
- **E-1352** — scheduling/book: E-456 contrato separado; E-458 reentrega ≠ erro; `busy_until` ≠ `ends_at`; 20 → 19.
- **E-1351** — business/tick: 4 desfechos separados; `errors` vem COM resultado; 21 → 20.
- **E-1350** — reschedule/slots: dois campos ditos por nome; responde ANTES; 23 → 21.
- **E-1349** — `OpenSlots` ≠ `Availability`; 0 slots ≠ dia livre; 24 → 23.
- **E-1348** — scheduling/waitlist: janela PEDIDA ≠ slot OFERECIDO; 26 → 24.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **34** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **18** (era 93) |

## AS 18 RESTANTES — TODAS ESCRITAS

`scheduling` 4 (`reschedule`, `no-show`, `waitlist/offer`, `waitlist/confirm`, `reminders/sent`) · `auth` 3
(MFA) · `finance` 3 · `media` 2 · `business` 1 (`certify`) · `notifications` 1 · `crm` 1 · `inbox` 1 ·
`service-agents` 1 (PUT) · `company` 1 · `plugins` 1.

**Nota acumulada — vinte e quatro avisos:** o gate mede o **domínio** ao agrupar e a **rota** ao listar.
Vinte e quatro medições, **nove** previsões certas. A última errada por confundir as duas propriedades —
e o erro está documentado no próprio tecto para não se repetir.

## ACHADOS ABERTOS (para o backlog)

1. **O transporte descarta o corpo do 409** (E-1352). O servidor devolve o agendamento idempotente; o
   `agentFetch` lança e não expõe o body. Resolve-se devolvendo `{ok, status, body}` — afecta **todos** os
   clientes, por isso é fatia própria.
2. **O A1 da DZ23 nunca foi avaliado** por `SigningProvider`; sem EKU de Code Signing.
3. **Não há catraca contra novos testes flaky** — o E-1345 foi encontrado por sorte.

O que continua aberto: **18 rotas** sem UI (todas escritas); **20 ORPHAN de 209**; o build **não é assinado
nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks` histórico — a única falha de CI**;
instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1352

A cabeça do índice é **E-1352**.

- **E-1352** — **E-456: o contrato da superfície é separado do motor, e o modo de falha é silencioso.**
  `BookingPanel` consome `POST /scheduling/book`. O comentário: *«Expor a struct INTERNA como contrato de
  API acopla a API a implementação — e o modo de falha é silencioso: o pedido chega vazio, e o erro culpa os
  campos que o cliente mandou.»* O corpo manda `resource_id`/`starts_at`; a struct interna espera
  `ResourceID`/`StartsAt` porque **não tem json tags** → corpo vira **zero silencioso**.
  **E-458: reentrega não é erro do cliente.** O servidor responde **409 com o MESMO recurso** —
  *«Idempotência de verdade é devolver o MESMO RECURSO»*. **Mas o `agentFetch` lança em erro HTTP e não
  expõe o corpo**, pelo que o painel diz **«o agendamento existente NÃO foi duplicado»** sem inventar o id.
  **`busy_until` ≠ `ends_at`** — inclui o buffer; mostrado como «recurso ocupado até … (buffer de 15 min)».
  **A organização não vai no corpo** (vem do middleware), com teste a provar.
  **VIGÉSIMA TERCEIRA MEDIÇÃO — previsão CERTA:** previ **1**, mediu **1** (20 → 19).
- **E-1351** — business/tick: 4 desfechos separados; `errors` vem COM resultado; 21 → 20.
- **E-1350** — reschedule/slots: dois campos ditos por nome; responde ANTES; 23 → 21.
- **E-1349** — `OpenSlots` ≠ `Availability`; 0 slots ≠ dia livre; 24 → 23.
- **E-1348** — scheduling/waitlist: janela PEDIDA ≠ slot OFERECIDO; 26 → 24.
- **E-1347** — scheduling/reminders/due: `now` não vem do corpo; 28 → 26.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **35** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **19** (era 93) |

## AS 19 RESTANTES — TODAS ESCRITAS

`scheduling` 5 (`cancel`, `reschedule`, `no-show`, `waitlist`, `waitlist/offer`, `waitlist/confirm`,
`reminders/sent`) · `auth` 3 (MFA) · `finance` 3 · `media` 2 · `business` 1 (`certify`) · `notifications` 1 ·
`crm` 1 · `inbox` 1 · `service-agents` 1 (PUT) · `company` 1 · `plugins` 1.

**Três previsões certas nas últimas três rondas com escrita** — o método estabilizou quando passou a medir
uma rota consumida em vez de um domínio.

**Nota acumulada — vinte e três avisos:** o gate mede o **domínio**, não a rota. Vinte e três medições,
**nove** previsões certas.

## ACHADO NOVO DESTA JANELA (para o backlog)

**O transporte descarta o corpo do 409.** O servidor devolve `{"appointment": …, "idempotent_replay": true}`
com o agendamento idempotente, mas o `agentFetch` lança e não expõe o body. O painel é honesto (diz
«reentrega detectada»), mas **o cliente não consegue mostrar o agendamento que já existia**. Isto é uma
lacuna da camada de transporte, não do painel: resolver exige o `agentFetch` devolver
`{ok, status, body}` em vez de lançar — mudança que afecta **todos** os clientes e por isso é fatia própria.

O que continua aberto: **19 rotas** sem UI (todas escritas); o gate **mede prefixos**; **20 ORPHAN de
209**; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks`
histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra novos
testes flaky**; **FASE 30 não iniciada**.

---
## JANELA E-1351

A cabeça do índice é **E-1351**.

- **E-1351** — **A primeira ESCRITA da série — e é uma escrita com confirmação.** `BusinessTickPanel`
  consome `POST /business/tick`. O painel **não dispara ao montar**: um tick executa ações autónomas,
  regista incidentes e gasta orçamento. Há botão e caixa de confirmação; sem ela o painel recusa e **não
  chama**. *Um painel que corre um ciclo ao abrir executou sem ninguém pedir — e a supervisão promete o
  contrário.*
  **Quatro desfechos que não são o mesmo número:** `autonomous_executed`, `awaiting_approval`,
  `blocked_by_budget`, `incidents_recorded` — mostrados **separados**. *Somar isto num «N ações» é a mesma
  mentira do E-1340: uma ação bloqueada por orçamento e uma à espera de aprovação são ambas «pendentes» e
  não são a mesma coisa.*
  **`errors` vem COM resultado** — *«é ele que mostra o que foi observado, executado, bloqueado»*.
  **O número do tick não pode recuar**; o orçamento mostra **dois** números.
  **VIGÉSIMA SEGUNDA MEDIÇÃO — previsão CERTA:** previ **1**, mediu **1** (21 → 20).
- **E-1350** — reschedule/slots: dois campos ditos por nome; responde ANTES; 23 → 21.
- **E-1349** — `OpenSlots` ≠ `Availability`; 0 slots ≠ dia livre; 24 → 23.
- **E-1348** — scheduling/waitlist: janela PEDIDA ≠ slot OFERECIDO; 26 → 24.
- **E-1347** — scheduling/reminders/due: `now` não vem do corpo; 28 → 26.
- **E-1346** — commerce: `on_hand`/`reserved`/`available` são três; 30 → 28.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **36** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **20** (era 93) |

## AS LEITURAS ACABARAM

**As 20 restantes são todas `POST` ou `PUT`:**

`scheduling` 6 · `auth` 3 (MFA) · `finance` 3 · `media` 2 · `business` 1 · `notifications` 1 · `crm` 1 ·
`inbox` 1 · `service-agents` 1 (PUT) · `company` 1 · `plugins` 1.

**Uma escrita não se fecha com um `GET`.** Exige: cliente tipado, painel com botão, **confirmação
explícita**, teste que prove que **sem confirmação nada acontece**, e um caminho de erro que não
esconda efeito parcial. A primeira (E-1351) provou que é fazível; o custo é ~2× o de uma leitura.

**Nota acumulada — vinte e dois avisos:** o gate mede o **domínio**, não a rota. Vinte e duas medições,
**oito** previsões certas — e as duas últimas foram certas seguidas.

O que continua aberto: **20 rotas** sem UI (todas escritas); o gate **mede prefixos**; **20 ORPHAN de
209**; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks`
histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra novos
testes flaky**; **FASE 30 não iniciada**.

---
## JANELA E-1350

A cabeça do índice é **E-1350**.

- **E-1350** — **A peça que quase sempre falta.** `ReschedulePanel` consome
  `GET /scheduling/reschedule/slots`. O comentário: *«remarcar sem saber o que está livre obriga o cliente a
  adivinhar, e a adivinhação vira horário inexistente — que o motor recusa **depois**, quando o cliente já
  escolheu. Aqui o motor responde **antes**.»*
  **Dois campos obrigatórios, a falta dita por nome:** `appointment_id` **e** `date`.
  **`/slots` e `/reschedule/slots` são perguntas diferentes:** uma serve para marcar, a outra para remarcar
  — e esta conhece o agendamento que está a ser movido.
  **Zero horários não é sucesso:** é a resposta negativa, dada a tempo.
  **VIGÉSIMA PRIMEIRA MEDIÇÃO — previsão errada por 1:** previ **1**, mediu **2** (23 → 21).
- **E-1349** — `OpenSlots` ≠ `Availability`; data vazia ≠ hoje; 0 slots ≠ dia livre; 24 → 23.
- **E-1348** — scheduling/waitlist: janela PEDIDA ≠ slot OFERECIDO; 26 → 24.
- **E-1347** — scheduling/reminders/due: `now` não vem do corpo; 28 → 26.
- **E-1346** — commerce: `on_hand`/`reserved`/`available` são três; 30 → 28.
- **E-1345** — RetentionPanel flaky: `setState` fora de `act`.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **37** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **21** (era 93) |

## `scheduling` ESTÁ FECHADO NAS LEITURAS — 6 DAS 21

O painel consome **6 leituras** (`services`, `appointments`, `reminders/due`, `waitlist`, `slots`,
`reschedule/slots`). Restam **6 rotas**, **todas escritas**: `book`, `cancel`, `reschedule`, `no-show`,
`waitlist` (POST), `waitlist/offer`, `waitlist/confirm`, `reminders/sent`.

**As 21 restantes são quase todas escritas:** `auth` 3 (MFA), `finance` 3, `media` 2, `business` 2,
`notifications` 1, `crm` 1, `inbox` 1, `service-agents` 1, `company` 1, `plugins` 1, `scheduling` 6.

**Uma escrita não se fecha com uma leitura.** Exige botão, confirmação, teste de negação e um fluxo que
o utilizador consiga completar -- é uma fatia maior do que qualquer leitura desta série.

**Nota acumulada — vinte e um avisos:** o gate mede o **domínio**, não a rota. Vinte e uma medições,
**sete** previsões certas.

O que continua aberto: **21 rotas** sem UI; o gate **mede prefixos**; **20 ORPHAN de 209**; o build **não é
assinado nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra novos testes flaky**; **FASE 30 não
iniciada**.

---
## JANELA E-1349

A cabeça do índice é **E-1349**.

- **E-1349** — **`OpenSlots` não é `Availability` — e o handler diz porquê (ADV-60).** `SlotsPanel` consome
  `GET /scheduling/slots`. O comentário: *«`Availability` decide apenas por agendamentos — não olha turno,
  fechamento por data, intervalo nem fuso; um slot marcado livre ali pode estar às 03:00 de um domingo com a
  empresa fechada»*. E fecha: *«A função não mudou de comportamento — mudou de promessa.»*
  O painel escreve a promessa inteira: **«dentro do turno, fora de fechamento e livre de ocupação»**.
  **`date` é obrigatório — vazio não é «hoje»**: o painel recusa antes de chamar.
  **`step` tem tecto de 240**; a duração de cada slot é **medida** contra o passo.
  **Zero horários não é dia livre.** E um slot com `free=false` numa lista de abertos é **contradição
  mostrada**, não escondida.
  **VIGÉSIMA MEDIÇÃO — PRIMEIRA PREVISÃO CERTA EM QUATRO RONDAS:** previ **1**, mediu **1** (24 → 23).
- **E-1348** — scheduling/waitlist: janela PEDIDA ≠ slot OFERECIDO; oferta sem prazo não é oferta; 26 → 24.
- **E-1347** — scheduling/reminders/due: `now` não vem do corpo; é só a metade pendente; 28 → 26.
- **E-1346** — commerce: `on_hand`/`reserved`/`available` são três; saldo derivado e conferido; 30 → 28.
- **E-1345** — RetentionPanel flaky: `setState` fora de `act`.
- **E-1344** — verify-contracts: `call.raw` vs `call.path`.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **39** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **23** (era 93) |

## O QUE RESTA EM `scheduling` — 8 DAS 23

O painel consome **5 leituras** (`services`, `appointments`, `reminders/due`, `waitlist`, `slots`).
Resta **1 leitura** (`slots/reschedule`) e **8 escritas**.

**Nota acumulada — vinte avisos:** o gate mede o **domínio**, não a rota. Vinte medições, **sete**
previsões certas — e a sétima veio depois de quatro erradas seguidas.

O que continua aberto: **23 rotas** sem UI; o gate **mede prefixos**; **20 ORPHAN de 209**; o build **não é
assinado nem instalado**; `sign.start`/`publish.*` inexistentes; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra novos testes flaky**; **FASE 30 não
iniciada**.

---
## JANELA E-1348

A cabeça do índice é **E-1348**.

- **E-1348** — **A janela PEDIDA não é o slot OFERECIDO.** `WaitlistPanel` consome
  `GET /scheduling/waitlist`. O contrato tem **dois pares de tempo**: `from`/`to` é a janela que o cliente
  **pediu**; `offered_slot` é o que o motor **ofereceu**; `offer_expires_at` é quando a oferta **deixa de
  valer**. O painel escreve «pediu de … a …» e nunca chama a isso horário marcado.
  **Uma oferta sem prazo não é uma oferta:** `ofertaValida()` exige `offered_slot` **e** prazo futuro.
  Sem prazo → «oferta não é válida» e a ação diz «aguardar», não «aceitar». Expirada → «a oferta expirou».
  `appointment_id` presente ⇒ «já convertida em agendamento».
  **DÉCIMA NONA MEDIÇÃO — previsão errada por 1:** previ **1**, mediu **2** (26 → 24).
- **E-1347** — scheduling/reminders/due: `now` não vem do corpo; é só a metade pendente; 28 → 26.
- **E-1346** — commerce: `on_hand`/`reserved`/`available` são três; saldo derivado e conferido; 30 → 28.
- **E-1345** — RetentionPanel flaky: `setState` fora de `act`.
- **E-1344** — verify-contracts: `call.raw` vs `call.path`.
- **E-1343** — wakeups: despertar informativo não é trabalho enfileirado; 31 → 30.
- **E-1342** — solutions: POST que não muta; 32 → 31.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **40** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **24** (era 93) |

## O QUE RESTA EM `scheduling` — 10 DAS 24

O painel consome **4 leituras** (`services`, `appointments`, `reminders/due`, `waitlist`).
Restam **2 leituras** (`slots`, `reschedule/slots`) e **8 escritas**.

**Nota acumulada — dezanove avisos:** o gate mede o **domínio**, não a rota. Dezanove medições, **seis**
previsões certas.

O que continua aberto: **24 rotas** sem UI; o gate **mede prefixos**; **20 ORPHAN de 209**; o build **não é
assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel do build **não se actualiza
sozinho**; **`gitleaks` histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **não há
catraca contra novos testes flaky**; **FASE 30 não iniciada**.

---
## JANELA E-1347

A cabeça do índice é **E-1347**.

- **E-1347** — **«`now` não vem do corpo» — e o próprio handler diz porquê:** *«um corpo que escolhe
  «agora» escolhe quem deve ser avisado»*. `RemindersPanel` consome `GET /scheduling/reminders/due` e
  **não envia `now`** — há teste a provar que o caminho não o contém.
  **Esta rota é só UMA metade.** O comentário: *«a superfície tem de expor as DUAS metades: quem deve ser
  avisado, e o registo de que foi.»* O painel escreve «o registo do que já foi enviado está noutra rota»,
  e na lista vazia: «isto NÃO diz que os anteriores foram enviados». *Um painel que se chame "lembretes"
  está a chamar ao trabalho PENDENTE o trabalho FEITO.*
  **`kind` é obrigatório — vazio não é «todos»**: o servidor responde 400, e o painel recusa antes de
  chamar. `lead_minutes` validado nos dois lados.
  **DÉCIMA OITAVA MEDIÇÃO — previsão errada por 1, a favor:** previ **1**, mediu **2** (28 → 26).
- **E-1346** — commerce: `on_hand`/`reserved`/`available` são três; saldo derivado e conferido; 30 → 28.
- **E-1345** — RetentionPanel flaky: `setState` fora de `act`.
- **E-1344** — verify-contracts: `call.raw` vs `call.path`.
- **E-1343** — wakeups: despertar informativo não é trabalho enfileirado; 31 → 30.
- **E-1342** — solutions: POST que não muta; 32 → 31.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **42** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **26** (era 93) |

## O QUE RESTA EM `scheduling` — 12 DAS 26

`scheduling` tem **14 rotas**: o painel consome **3 leituras** (`services`, `appointments`, `reminders/due`).
Restam **3 leituras** (`slots`, `reschedule/slots`, `waitlist`) e **8 escritas** (`book`, `cancel`,
`reschedule`, `waitlist`, `waitlist/offer`, `waitlist/confirm`, `no-show`, `reminders/sent`).

**Nota acumulada sobre o contador — dezoito avisos:** o gate mede o **domínio**, não a rota. Dezoito
medições, **seis** previsões certas.

O que continua aberto: **26 rotas** sem UI (19 escritas, 7 leituras); o gate **mede prefixos**; **20 ORPHAN
de 209**; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel do
build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de CI**; instalação limpa
`BLOCKED_EXTERNAL`; **não há catraca contra novos testes flaky**; **FASE 30 não iniciada**.

---
## JANELA E-1346

A cabeça do índice é **E-1346**.

- **E-1346** — **`on_hand`, `reserved` e `available` são três números — e o painel mostra os três.**
  `StockPanel` consome `GET /commerce/stock/:sku`. O `CommerceStockBalance` declara os três: *o que existe
  no armazém, o que existe mas já tem dono, e o que se pode vender agora*. O painel escreve «7 disponível(is)
  · 10 em armazém · 3 reservado(s)». *Um painel que mostre só `available` esconde o que está reservado; um
  que mostre `on_hand` como "disponível" vende mercadoria que já tem dono.*
  **O saldo é DERIVADO — e o painel confere a soma** contra os movimentos que vieram no mesmo payload.
  Sem movimentos, devolve `null`: *ausência de prova não é prova de conformidade.*
  **`available` tem de ser `on_hand − reserved`**, e o painel verifica.
  **DÉCIMA SÉTIMA MEDIÇÃO — previsão errada por 1:** previ **1**, mediu **2** (30 → 28).
- **E-1345** — RetentionPanel flaky: `setState` fora de `act`; 1184/1185 → 1185 ×3.
- **E-1344** — verify-contracts: `call.raw` vs `call.path`; falso negativo provado e corrigido.
- **E-1343** — wakeups: despertar informativo não é trabalho enfileirado; 31 → 30.
- **E-1342** — solutions: POST que não muta; `integra_automaticamente` sempre falso; 32 → 31.
- **E-1341** — company: `customers_created=0` é fronteira, não contagem; 33 → 32.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **44** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **28** (era 93) |

## 70% FECHADAS — E O QUE RESTA

`scheduling` **12** · `auth` 3 (MFA) · `finance` 3 · `media` 2 (201 Created) · `business` 2 ·
`company` 1 · `notifications` 1 · `crm` 1 · `inbox` 1 · `service-agents` 1 · `plugins` 1.

**Todas são escritas que mutam estado real.** *Um painel que mostra uma rota de escrita não a prova: é
preciso um botão, uma confirmação e um teste de negação.*

**Nota acumulada sobre o contador — dezassete avisos:** o gate mede o **domínio**, não a rota. Dezassete
medições, **seis** previsões certas.

O que continua aberto: **28 rotas** sem UI (todas escritas); o gate **mede prefixos**; **20 ORPHAN de
209**; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel do build
**não se actualiza sozinho**; **`gitleaks` histórico — a única falha de CI**; instalação limpa
`BLOCKED_EXTERNAL`; **não há catraca contra novos testes flaky**; **FASE 30 não iniciada**.

---
## JANELA E-1345

A cabeça do índice é **E-1345**.

- **E-1345** — **um teste que às vezes passa não é um teste: é uma moeda ao ar que se pode citar.** A
  suíte completa acusou `RetentionPanel > aplicar mostra os números reais do que removeu` com
  **1184/1185**. O teste **passava isolado, duas vezes seguidas**. Causa na linha 149: `await
  botao.props.onClick()` chamado **fora de `agir()`** — um `setState` fora de `act`. **O teste já sabia:**
  os outros 7 usam `agir()`; a linha 149 era a única que chamava o handler directamente. *O ficheiro
  mostrava o padrão correcto ao lado do incorrecto.* Corrigido na causa: **3 passadas consecutivas de
  1185/1185** — sem baixar testes, sem skip, sem aumentar timeout. O defeito era **latente e não era meu**:
  o `RetentionPanel.test.tsx` é anterior a esta série; um flaky precisa de uma passada azarada, não de uma
  regressão.
- **E-1344** — verify-contracts: `call.raw` vs `call.path`; falso negativo provado com probe e corrigido.
- **E-1343** — wakeups: despertar informativo não é trabalho enfileirado; 31 → 30.
- **E-1342** — solutions: POST que não muta; `integra_automaticamente` sempre falso; 32 → 31.
- **E-1341** — company: `customers_created=0` é fronteira, não contagem; 33 → 32.
- **E-1340** — business: orçamento não é aprovação; 35 → 33.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **46** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **30** (era 93) |

**Duas rondas, dois instrumentos — e nenhuma rota.** O E-1344 corrigiu o gate de contratos; o E-1345
corrigiu um teste intermitente. *O contador de rotas é o indicador mais visível e não é o único: um
instrumento que mente e um teste que oscila custam mais do que uma rota sem painel — a rota em falta
sabe-se, o falso negativo e o falso vermelho ensinam a ignorar o próximo aviso.*

**Nota acumulada sobre o contador — dezasseis avisos:** o gate mede o **domínio**, não a rota. Dezasseis
medições, **seis** previsões certas.

O que continua aberto: **30 rotas** sem UI (todas escritas); `hades_rotas_orfas.py` **mede prefixos**;
**20 ORPHAN listados de 209**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **não há catraca contra novos testes flaky** — este foi
encontrado por sorte; **FASE 30 não iniciada**.

---
## JANELA E-1344

A cabeça do índice é **E-1344**.

- **E-1344** — **o gate de contratos comparava o texto cru com o caminho normalizado — e acusava o produto
  pelo que ele fazia.** `matches()` usava `call.raw`. Com `agentFetch("/plugins")`, `raw = "/plugins"` e
  `path = "/api/agent/v1/plugins"`. O gate comparava os dois e escrevia **`[FAIL] GET
  /api/agent/v1/plugins -> NO MATCH`** — *a rota que o log diz não encontrar está no prefixo do próprio
  log.* Passou despercebido porque todos os clientes usam o caminho completo, onde `raw == path`.
  **Defeito latente desde o início, provado com um probe real:** `__probe_contrato.ts` com
  `agentFetch("/plugins")` → **antes 1 FAIL, depois 0**. O probe foi removido — não era fixture, era prova.
  *Um gate que compara o texto cru com o caminho normalizado não compara dois caminhos: compara duas
  maneiras de escrever o mesmo caminho, e chama-lhes nomes diferentes.*
  **Teste permanente que falha com o defeito:** `scripts/verify_contracts_test.py` (o 46.º script) →
  correcção **5/5 exit 0**; defeito reintroduzido **2 FAIL exit 1**; restaurado **PASSED**.
- **E-1343** — wakeups: despertar informativo não é trabalho enfileirado; 31 → 30.
- **E-1342** — solutions: POST que não muta; `integra_automaticamente` sempre falso; 32 → 31.
- **E-1341** — company: `customers_created=0` é fronteira, não contagem; 33 → 32.
- **E-1340** — business: orçamento não é aprovação; 35 → 33.
- **E-1339** — supervisor: reconhecer não é resolver; 39 → 35.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **46** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **30** (era 93) |

**O contador não desceu nesta ronda — e está certo.** Esta fatia não fechou rota nenhuma: **fechou um
instrumento**. *Um gate que mente sobre o produto custa mais do que uma rota sem painel: a rota em falta
sabe-se, o falso negativo ensina a ignorar o próximo vermelho verdadeiro.*

## O QUE FOI VERIFICADO E **NÃO** ADOTADO

`/media/ocr` e `/media/tone` pareciam cálculos puros como o `solutions/compare`. **Não são:** ambos
respondem **`201 Created`** e criam um `Artifact` no workspace da missão. *Um POST que responde 201 não é
um cálculo: é uma escrita com endereço.*

Ficam as 30 — todas escritas que mutam estado real.

**Nota acumulada sobre o contador — dezasseis avisos:** o gate mede o **domínio**, não a rota. Dezasseis
medições, **seis** previsões certas.

O que continua aberto: **30 rotas** sem UI (todas escritas); `hades_rotas_orfas.py` **mede prefixos**;
**20 ORPHAN listados de 209**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1343

A cabeça do índice é **E-1343**.

- **E-1343** — **um despertar informativo e um trabalho enfileirado são ambos «acordar» e não são a mesma
  coisa.** `WakeupsPanel` consome `POST /wakeups` — **a primeira lacuna fechada por uma rota de escrita que
  MUTA**, e a única que pode ser repetida sem medo. O handler: *«`EnqueueMission` é idempotente por missão
  […] é o que torna esta rota segura de chamar periodicamente.»* O painel mostra **as duas listas lado a
  lado** — `wakeups` e `enqueued` — porque só uma delas mente: *mostrar só `enqueued` esconde missões que o
  sistema viu; mostrar só `wakeups` sugere que todas vão correr.*
  **O informativo não vira trabalho**: *«enfileirar uma missão que espera aprovação seria executá-la sem a
  decisão que ela própria pediu»*. O painel escreve **«N despertar(es) informativo(s) — NÃO foram
  enfileirados»**.
  **Falha parcial não é sucesso**: se nada foi enfileirado e houve falhas, a rota é **500**. O painel tem
  tipo próprio (`WakeupsSemEfeitoError`) porque o `agentFetch` não expõe o corpo do erro.
  **O escopo é declarado, não presumido global.**
  **SEXTA PREVISÃO CERTA DO GATE**: previ **1**, mediu **1** (31 → 30) — a primeira descida por uma rota
  que escreve.
- **E-1342** — solutions: POST que não muta; `integra_automaticamente` sempre falso; 32 → 31.
- **E-1341** — company: `customers_created=0` é fronteira, não contagem; 33 → 32.
- **E-1340** — business: orçamento não é aprovação; custo ao lado da ação; 35 → 33.
- **E-1339** — supervisor: reconhecer não é resolver; 200-vazio não prova supervisor; 39 → 35.
- **E-1338** — `PluginsPanel`: scope pedido não é concedido; integridade em 4 estados; 41 → 39.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **46** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **30** (era 93) |

## 78% DAS LACUNAS FORAM FECHADAS — E AS ÚLTIMAS SÃO OUTRA COISA

De **93** para **30**. Mas as 30 restantes são **todas escritas que mutam estado real**, e só uma
(`wakeups`) é idempotente.

| domínio | lacunas | o que muta |
|---|---|---|
| `scheduling` | **12** | book, cancel, no-show, reminders |
| `auth` | 3 | MFA — enable, disable, recovery |
| `finance` | 3 | cost, reconcile, revenue por pedido |
| `media` | 2 | ocr, tone |
| `commerce` | 2 | stock |
| `business` | 2 | certify, tick |
| `company` | 1 | factory/launch |
| `notifications` | 1 | register |
| `crm` | 1 | stage |
| `inbox` | 1 | messages |
| `service-agents` | 1 | approval-approvers |
| `plugins` | 1 | bundle |

**Isto é a fronteira honesta do contador.** *Um painel que mostra uma rota de escrita não a prova: é
preciso um botão, uma confirmação e um teste de negação. Cada uma das 29 restantes é a sua própria
fatia — e algumas (MFA, certify, launch) tocam em credenciais ou criam empresas.*

**Nota acumulada sobre o contador — dezasseis avisos:** o gate mede o **domínio**, não a rota. Dezasseis
medições, **seis** previsões certas.

O que continua aberto: **30 rotas** sem UI (todas escritas, 29 mutam); o gate **mede prefixos**; a
assimetria `call.raw`/`call.path`; **20 ORPHAN listados de 209**; o build **não é assinado nem
instalado**; `sign.start`/`publish.*` inexistentes; o painel do build **não se actualiza sozinho**;
**`gitleaks` histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não
iniciada**.

---
## JANELA E-1342

A cabeça do índice é **E-1342**.

- **E-1342** — **um POST que não muda nada é um GET que não caberia no URL.** `SolutionsPanel` consome
  `POST /solutions/compare` — a **primeira lacuna fechada por uma rota de escrita que não escreve**. O
  comentário do handler: *«O método é POST, mas o efeito é nenhum; o cliente não deve ter de inferir isso
  do método»*. A resposta traz **`somente_leitura: true`**, e o painel **lê o campo** em vez de inferir do
  método. *A partir de hoje, «é POST» não significa «muta»: significa «o corpo não cabe num URL».*
  **`integra_automaticamente` é SEMPRE falso** — *«o sistema pode recomendar um vencedor; não pode escolher
  por quem assina o merge»*. Há um teste para o caso do servidor um dia dizer `true`: o painel **não
  executa**.
  **Empate não é falha** (decisão humana). **`nao_medidos` é «ninguém mediu», não «falhou»**.
  **`medidos_parcialmente` é o campo que mais se esquece** — *«um critério medido num candidato e não
  noutro não compara: informa»* — e o painel distingue-o de `nao_medidos` em dois blocos.
  **As medições são entradas, não saída**: JSON inválido é recusado **antes** de chamar o servidor.
  **QUINTA PREVISÃO CERTA DO GATE**: previ **1**, mediu **1** (32 → 31).
- **E-1341** — company: `customers_created=0` é fronteira, não contagem; valor de tabela não é receita; 33 → 32.
- **E-1340** — business: orçamento não é aprovação; custo ao lado da ação; 35 → 33.
- **E-1339** — supervisor: reconhecer não é resolver; 200-vazio não prova supervisor; 39 → 35.
- **E-1338** — `PluginsPanel`: scope pedido não é concedido; integridade em 4 estados; 41 → 39.
- **E-1337** — `SessionPanel`: `omitempty` em booleano = `false`; `Public()` limpa segredos; 42 → 41.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **47** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **31** (era 93) |

## AS LACUNAS DE LEITURA ESTÃO ESGOTADAS

**As 31 restantes são TODAS POST que MUTAM:**

| domínio | lacunas | o que muta |
|---|---|---|
| `scheduling` | **12** | book, cancel, no-show, reminders |
| `auth` | 3 | MFA — enable, disable, recovery |
| `finance` | 3 | cost, reconcile, revenue por pedido |
| `media` | 2 | ocr, tone |
| `commerce` | 2 | stock |
| `business` | 2 | certify, tick |
| `company` | 1 | factory/launch |
| `notifications` | 1 | register |
| `wakeups` | 1 | wakeups |
| `crm` | 1 | stage |
| `inbox` | 1 | messages |
| `service-agents` | 1 | approval-approvers |
| `plugins` | 1 | bundle |

**Isto muda a natureza do trabalho.** *Um painel que só lê não fecha uma rota que escreve. As 31
restantes exigem UI de ação — botão, confirmação, approval — e cada uma merece a sua própria fatia com
teste de negação. Não é o mesmo trabalho, e o contador não o vai reflectir depressa.*

**Nota acumulada sobre o contador — quinze avisos:** o gate mede o **domínio**, não a rota. Quinze
medições, **cinco** previsões certas.

O que continua aberto: **31 rotas** sem UI (todas escritas); o gate **mede prefixos**; a assimetria
`call.raw`/`call.path`; **20 ORPHAN listados de 209**; o build **não é assinado nem instalado**;
`sign.start`/`publish.*` inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks`
histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1341

A cabeça do índice é **E-1341**.

- **E-1341** — **`customers_created: 0` não é «zero clientes»: é «esta fábrica nunca cria clientes».**
  `CompanyFactoryPanel` consome `GET /company/factory/report/:id`. O próprio tipo declara: *«CustomersCreated
  é sempre 0: a fábrica NÃO inventa clientes»*; e as rotas: *«a fábrica NÃO cria cliente nem emite registro
  legal»*. O painel escreve **«0 clientes criados — a fábrica não inventa clientes»**. *Um painel que mostre
  «0 clientes» ao lado de «3 produtos» está a sugerir uma empresa que arrancou sem clientes — quando o que
  aconteceu foi a fábrica ter feito tudo o que NÃO depende de um humano.*
  **`catalog_value_cents` é valor de TABELA, não receita** — o painel diz «(não é receita)».
  **O checklist separa** o que a fábrica fez do que fica para um humano: «1 feito(s) pela fábrica · 1 à
  espera de terceiro». **O id é codificado no caminho** (`a/b` → `a%2Fb`).
  **QUARTA PREVISÃO CERTA DO GATE**: previ **1**, mediu **1** (33 → 32). `company` tem 2 rotas e só
  **uma** é leitura.
- **E-1340** — business: orçamento não é aprovação; custo ao lado da ação; 35 → 33.
- **E-1339** — supervisor: reconhecer não é resolver; 200-vazio não prova supervisor; 39 → 35.
- **E-1338** — `PluginsPanel`: scope pedido não é concedido; integridade em 4 estados; 41 → 39.
- **E-1337** — `SessionPanel`: `omitempty` em booleano = `false`; `Public()` limpa segredos; 42 → 41.
- **E-1336** — `FinancePanel`: líquido com cobertura ao lado; recebível não é receita; 43 → 42.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **48** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **32** (era 93) |

## O MAPA REVELOU O QUE FALTA — E NÃO É O QUE PARECIA

| domínio | lacunas | natureza |
|---|---|---|
| `scheduling` | **12** | **todas de escrita** (`book`, `cancel`, `no-show`, reminders) |
| `auth` | 3 | MFA — escritas (`enable`, `disable`, recovery) |
| `finance` | 3 | escritas (cost, reconcile, revenue por pedido) |
| `media` | 2 | **ambas POST** (`ocr`, `tone`) |
| `commerce` | 2 | escritas (stock) |
| `business` | 2 | escritas (certify, tick) |
| `company` | 1 | escrita (factory/launch) |
| outros | 7 | 1 cada, quase todas escritas |

**A conclusão que isto força:** das 32 restantes, **17 são POST declaradas**. *As lacunas de leitura
estão esgotadas — o que sobra é escrita, e escrita não se fecha com um painel que só lê.*

**Nota acumulada sobre o contador — catorze avisos:** o gate mede o **domínio**, não a rota. Catorze
medições, **quatro** previsões certas.

O que continua aberto: **32 rotas** sem UI (17 escritas); o gate **mede prefixos**; a assimetria
`call.raw`/`call.path`; **20 ORPHAN listados de 209**; o build **não é assinado nem instalado**;
`sign.start`/`publish.*` inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks`
histórico — a única falha de CI**; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1340

A cabeça do índice é **E-1340**.

- **E-1340** — **bloqueada por orçamento e aguardando aprovação são ambas «pendentes» e não são a mesma
  coisa.** `BusinessActionsPanel` consome `GET /business/actions`. `BusinessAction.status` tem **seis**
  valores: `AUTONOMA` (correu sem humano), `REQUER_APROVACAO`, `EXECUTADA`, `AGUARDANDO_APROVACAO`,
  `RECUSADA` e **`BLOQUEADA_ORCAMENTO`**. O painel escreve deste último **«BLOQUEADA POR ORÇAMENTO —
  aprovar não resolve»**. *Uma ação bloqueada por orçamento e uma ação à espera de aprovação são ambas
  «pendentes» e não são a mesma coisa: uma precisa de dinheiro, a outra precisa de uma pessoa.*
  **O resumo separa as duas filas**: «1 à espera de uma pessoa · 1 à espera de orçamento · 1 noutro
  estado».
  **O custo não é decorativo**: `cost_cents` aparece ao lado de cada ação, com o risco. *Aprovar uma ação
  sem ver o custo é aprovar um número que não se leu.*
  **Só a leitura chega ao produto**: `certify`, `tick` e `decide` são escritas — o comentário do `decide`
  diz *«aprovar EXECUTA, recusar não executa»*, e o painel não decide. **503 ≠ «nenhuma ação pendente»**.
  **DÉCIMA LEITURA ERRADA DO GATE.** Previ **1**, mediu **2** (35 → 33). *Treze medições, três previsões
  certas. O erro desta vez foi supor que contar as rotas de LEITURA bastava. Não basta: um prefixo pode
  ter mais do que uma entrada na lacuna.*
- **E-1339** — supervisor: reconhecer não é resolver; 200-vazio não prova supervisor; 39 → 35.
- **E-1338** — `PluginsPanel`: scope pedido não é concedido; integridade em 4 estados; 41 → 39.
- **E-1337** — `SessionPanel`: `omitempty` em booleano = `false`; `Public()` limpa segredos; 42 → 41.
- **E-1336** — `FinancePanel`: líquido com cobertura ao lado; recebível não é receita; 43 → 42.
- **E-1335** — `OmnichannelPanel`: DLQ declara durabilidade, segredo pelo nome; 48 → 43.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **49** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **33** (era 93) |

**Os maiores domínios ainda sem UI:** `media` **2**, `company` **2** — o resto é disperso.

**Nota acumulada sobre o contador — treze avisos:** o gate mede o **domínio**, não a rota. Treze medições,
**três** previsões certas. A décima errou por contar rotas de leitura: um prefixo pode ter mais do que
uma entrada.

O que continua aberto: **33 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 209**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **3 rotas de `/business` sem UI por serem escritas —
declarado, não escondido**; **FASE 30 não iniciada**.

---
## JANELA E-1339

A cabeça do índice é **E-1339**.

- **E-1339** — **reconhecer não é resolver — e o próprio handler o diz.** `CompanySupervisorPanel` passou a
  consumir `/supervisor/incidents` e `/supervisor/ticks`. O comentário do acknowledge: *«registra CIÊNCIA
  humana sem declarar resolução»*. O painel escreve **«reconhecido por X — reconhecer NÃO é resolver»**, e
  um incidente reconhecido continua a mostrar `state` aberto e **«sem resolução registada»**.
  **Lista vazia não prova que o supervisor está ligado — e esta fronteira é diferente da do
  `omnichannel`.** Sem runtime, `supervisorIncidents` devolve **200** `{incidents: [], count: 0}`; o
  `omnichannel` devolve **503**. *Duas rotas que respondem ao mesmo problema com fronteiras diferentes:
  uma falha fechado, a outra mente por omissão.* O painel diz isso em vez de dizer «tudo bem».
  **Os ciclos mostram o que o supervisor bloqueia**: `blocked_external_actions` e `risk_paused_companies`
  não são métricas decorativas.
  **Ampliar, não duplicar:** o `CompanySupervisorPanel.tsx` já existia (195 linhas) — acrescentei-lhe a
  secção, em vez de criar um segundo painel de supervisor. A regra que o E-1326 custou caro.
  **A NONA LEITURA ERRADA — e foi a favor.** Previ **2**, mediu **4** (39 → 35). *Num domínio
  PARCIALMENTE consumido, fechar as últimas leituras não custa o número de leituras: custa o domínio. O
  contador sempre mediu domínios — só parecia medir rotas quando todas as rotas de um domínio eram
  lacuna.*
- **E-1338** — `PluginsPanel`: scope pedido não é concedido; integridade em 4 estados; 41 → 39.
- **E-1337** — `SessionPanel`: `omitempty` em booleano = `false`; `Public()` limpa segredos; 42 → 41.
- **E-1336** — `FinancePanel`: líquido com cobertura ao lado; recebível não é receita; 43 → 42.
- **E-1335** — `OmnichannelPanel`: DLQ declara durabilidade, segredo pelo nome; 48 → 43.
- **E-1334** — catálogo de agentes usado no painel; mock com ordem trocada; tecto 48 inalterado.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **51** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **35** (era 93) |

**Os maiores domínios ainda sem UI:** `business` **4** (parcial), `media` **2**, `company` **2**.

**Nota acumulada sobre o contador — doze avisos:** o gate mede o **domínio**, não a rota. Doze medições,
**três** previsões certas. A nona errou **a favor**: um domínio parcialmente consumido fecha inteiro.

O que continua aberto: **35 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 211**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **`/supervisor/incidents` e `/supervisor/ticks` devolvem
200-vazio sem runtime — fronteira mais fraca que o 503 do `omnichannel`, agora declarada no painel**;
**FASE 30 não iniciada**.

---
## JANELA E-1338

A cabeça do índice é **E-1338**.

- **E-1338** — **um scope pedido não é um scope concedido — e o `PluginStatus` só traz os pedidos.**
  `PluginsPanel` consome as **três** leituras de `plugins`. `PluginInstallation` tem
  `requested_scopes` **e** `granted_scopes`; `PluginStatus` tem **só** `requested_scopes`. O painel diz
  **«pediu: X — o estado não informa o que foi concedido»**. *Um ecrã que mostre `requested_scopes`
  debaixo do rótulo «permissões» está a dizer que o plugin TEM o que ele PEDIU. Não é a mesma lista.*
  **Pedido vs concedido lado a lado nomeia o que falta**: «3 pedido(s), 2 concedido(s) — NÃO
  concedido(s): net.out».
  **`ready_to_use: false` sem as razoes é uma linha vermelha sem causa** — o painel mostra `reasons`, e
  quando o servidor não manda nenhuma escreve «motivo não declarado».
  **`trusted` e `content_verified` são coisas diferentes** — quatro combinações distintas.
  **O formato é o contrato aceito**: `hades-plugin/v1`, 5 `kinds`, 5 `runtimes`, e o handler diz porquê
  — *«para que o autor do plugin saiba o que o runtime faz antes de o escrever»*.
  **OITAVA LEITURA ERRADA DO GATE — e o erro desta vez foi de outra natureza.** Previ **3**, mediu **2**
  (41 → 39). `/plugins/format` é **ESTÁTICA**: não depende de organização nem de estado, nunca esteve na
  lacuna. *As três previsões certas anteriores foram as que contaram rotas de LEITURA; esta falhou
  porque supus que uma rota de leitura é sempre uma lacuna. Uma rota estática não falta a ninguém.*
- **E-1337** — `SessionPanel`: `omitempty` em booleano = `false`; `Public()` limpa segredos; 42 → 41.
- **E-1336** — `FinancePanel`: líquido com cobertura ao lado; recebível não é receita; 43 → 42.
- **E-1335** — `OmnichannelPanel`: DLQ declara durabilidade, segredo pelo nome; 48 → 43.
- **E-1334** — catálogo de agentes usado no painel; mock com ordem trocada; tecto 48 inalterado.
- **E-1333** — `ApprovalsPanel` com `role_checked`/quórum/`overdue`; 53 → 48.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **55** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **39** (era 93) |

**Os maiores domínios ainda sem UI:** `supervisor` **4** (parcial), `business` **4** (parcial),
`media` **2**, `company` **2**.

**Nota acumulada sobre o contador — onze avisos:** o gate mede o **domínio**, não a rota. Onze medições,
**três** previsões certas. E a oitava falhou por uma razão nova: uma rota **estática** não é lacuna.

O que continua aberto: **39 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 211**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **6 rotas de `/plugins` sem UI por serem escritas —
declarado, não escondido**; **FASE 30 não iniciada**.

---
## JANELA E-1337

A cabeça do índice é **E-1337**.

- **E-1337** — **`omitempty` num booleano faz «ausente» significar `false` — e o painel não lhe pode
  chamar «desconhecido».** `SessionPanel` consome `GET /auth/session`. `auth` tem **13 rotas**, mas só
  **uma** é leitura sem parâmetro; as outras 12 são escritas (logout, MFA, OAuth, SAML) — **o painel não
  as chama e não inventa botões**.
  **`Public()` é a fronteira dos segredos, e é legível no código:** limpa `MFASecretCiphertext` e
  `RecoveryCodesCiphertext`, e `omitempty` faz o resto. O teste verifica que `mfa_secret`,
  `recovery_codes` e `Ciphertext` não aparecem em nenhum sítio da árvore renderizada. *A fronteira não
  está no painel: está na função que o servidor chama antes de serializar.*
  **O achado: `mfa_enabled` tem `omitempty`** ⇒ ausente é `false`, nunca «desconhecido». O teste proíbe
  a palavra. *Um campo booleano com `omitempty` que se leia como «não sei» transforma um «desligado» em
  dúvida, e a dúvida num alarme falso.*
  **`mode: "local"` não é sessão inválida** (200, `authenticated: false`). **Erro de rede não é sessão
  anónima** — o teste distingue `sessao-erro` de `sem-utilizador`.
  **TERCEIRA PREVISÃO CERTA DO GATE.** Previ **1**, mediu **1** (42 → 41). *Dez medições, três
  previsões certas — e as três últimas foram as únicas em que contei as rotas de LEITURA em vez de contar
  as rotas do domínio.*
- **E-1336** — `FinancePanel`: líquido com cobertura ao lado; recebível não é receita; 43 → 42.
- **E-1335** — `OmnichannelPanel`: DLQ declara durabilidade, segredo pelo nome; 48 → 43.
- **E-1334** — catálogo de agentes usado no painel; mock com ordem trocada; tecto 48 inalterado.
- **E-1333** — `ApprovalsPanel` com `role_checked`/quórum/`overdue`; 53 → 48.
- **E-1332** — `SalesPanel` com a conta aberta; frete não declarado ≠ R$ 0,00; 61 → 53.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **57** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **41** (era 93) |

**Os maiores domínios ainda sem UI:** `supervisor` **4** (parcial), `business` **4** (parcial),
`plugins` **3**, `media` **2**, `company` **2**.

**Nota acumulada sobre o contador — dez avisos:** o gate mede o **domínio**, não a rota. Dez medições,
**três** previsões certas. As três certas foram as que contaram **rotas de leitura**.

O que continua aberto: **41 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 211**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **12 rotas de `auth` sem UI por serem escritas — declarado,
não escondido**; **FASE 30 não iniciada**.

---
## JANELA E-1336

A cabeça do índice é **E-1336**.

- **E-1336** — **o líquido sobre 0% de reconciliação não é um líquido — é uma soma.** `FinancePanel`
  consome `GET /finance/report`. Quatro números de dinheiro que não são a mesma coisa: `income_cents`
  (entrou, contabilizado), `receivable_cents` (*«o que ainda NÃO entrou»* — o próprio tipo),
  `unposted_revenue_cents` (existe e não está contabilizado) e `refund_due_cents` (a devolver). Somar os
  quatro e chamar-lhe receita era a mentira mais fácil do ecrã; o painel separa-os. *Uma receita líquida
  calculada sobre 0% de reconciliação não é uma receita líquida: é uma soma.*
  **A cobertura acompanha o líquido, sempre** — com 0% escreve **«líquido PROVISÓRIO»**. *Um número sem a
  sua qualidade ao lado é um número que se cita mal.*
  **As divergências não são escondidas** (`kind`/`amount`/`detail`); sem divergências com lançamentos
  diz «nenhuma divergência registada», não «está tudo bem». **O 503 não é «a empresa não tem números»**
  (`ErrFinanceCompanyMissing`). **Só a leitura chega ao produto**: as 3 escritas (receita, custo,
  reconciliar) exigem acção e approval — o painel não inventa os botões. **Uma só verdade sobre o mesmo
  número**: importa `centavosParaReais` do `commerceClient`.
  **SÉTIMA LEITURA ERRADA DO GATE.** Previ 4, mediu **1** (43 → 42): `finance` tem 4 rotas mas só
  **uma** estava na lacuna. *Nove medições, duas previsões certas. O instrumento não está a melhorar: eu
  é que continuo a supor que uma rota consumida é uma lacuna fechada, e o gate mede outra coisa.*
- **E-1335** — `OmnichannelPanel`: DLQ declara durabilidade, segredo pelo nome; 48 → 43.
- **E-1334** — catálogo de agentes usado no painel; mock com ordem trocada; tecto 48 inalterado.
- **E-1333** — `ApprovalsPanel` com `role_checked`/quórum/`overdue`/notificação; 53 → 48.
- **E-1332** — `SalesPanel` com a conta aberta; frete não declarado ≠ R$ 0,00; 61 → 53.
- **E-1331** — `InboxPanel`; `reply_blocked` não esconde a conversa; 68 → 61.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **58** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **42** (era 93) |

**Os maiores domínios ainda sem UI:** `auth` **4**, `supervisor` **4** (parcial), `business` **4**
(parcial), `plugins` **3**, `media` **2**, `company` **2**.

**Nota acumulada sobre o contador — nove avisos:** o gate mede o **domínio**, não a rota. Nove medições,
**duas** previsões certas. Corrigi-lo é trabalho próprio.

O que continua aberto: **42 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 211**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1335

A cabeça do índice é **E-1335**.

- **E-1335** — **a DLQ declara a sua própria durabilidade — e o segredo aparece pelo nome.** `omnichannel`
  tinha **6 rotas** e nenhum componente. Entregues `omnichannelClient.ts` + `OmnichannelPanel.tsx`,
  consumindo `/omnichannel/connectors`, `/omnichannel/events` e `/omnichannel/dlq`.
  **(1) Durabilidade não se presume, e o handler diz porquê:** *«O ESTADO acompanha a lista: durabilidade
  não se presume, e falha de persistência precisa ser visível para quem opera — uma fila "em memória"
  que se apresenta como durável é pior que a ausência dela.»* O painel escreve **«fila em memória — não
  sobrevive a reinício»** quando `persisted=false`, e mostra o último erro de persistência **com a fila
  de pé**. *Uma fila que promete durabilidade que não tem é pior do que uma fila que admite que a perde.*
  **(2) `secret_env` é o NOME da variável, não o segredo.** *«O valor nunca é guardado nem devolvido pela
  API.»* **Terceira vez que este padrão aparece** (E-1325 `TokenEnv`, E-1329 PII mascarada, aqui) — é a
  mesma família: **o identificador nunca viaja como o conteúdo.**
  **(3) `organization_id` do evento vem do conector, não do payload** — isolamento de tenant na
  ingestão. **(4) A DLQ não é uma lista de erros genéricos**: traz `reason` e `attempts`. **(5) Fila
  vazia não é fila ausente.** **(6) 503 não é lista vazia.**
  **PREVISÃO CERTA, SEGUNDA VEZ.** Previ 5; mediu 5. *Duas em sete: o instrumento continua errado, e
  continuar a prever é a única forma de saber quanto.*
- **E-1334** — catálogo de agentes usado no painel; mock com ordem trocada; tecto 48 inalterado.
- **E-1333** — `ApprovalsPanel` com `role_checked`/quórum/`overdue`/notificação; 53 → 48.
- **E-1332** — `SalesPanel` com a conta aberta; frete não declarado ≠ R$ 0,00; 61 → 53.
- **E-1331** — `InboxPanel`; `reply_blocked` não esconde a conversa; 68 → 61.
- **E-1330** — detalhe do contacto com consentimento real; teto 68 inalterado e explicado.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **59** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **43** (era 93) |

**Os maiores domínios ainda sem UI:** `auth` **4**, `supervisor` **4** (parcial), `finance` **4**,
`business` **4** (parcial), `plugins` **3**, `media` **2**, `company` **2**.

**Nota acumulada sobre o contador — oito avisos:** o gate mede o **domínio**, não a rota. Sete medições,
**duas** previsões certas (E-1331 e E-1335). Corrigi-lo é trabalho próprio.

O que continua aberto: **43 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 211**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1334

A cabeça do índice é **E-1334**.

- **E-1334** — **uma função de cliente sem chamador — e um mock com a ordem trocada.** O E-1333 deixou
  `listarAgentesDeServico` escrita em `approvalsClient.ts` e **o produto nunca a chamava**. Esta fatia
  usa-a: o painel mostra o catálogo de agentes, e o teste verifica a rota **e** o texto. *Um teste que
  chama a função que o produto não chama prova que a função funciona, não que o produto a usa.*
  **O defeito que isso revelou foi no mock, não no produto.** Ao acrescentar o catálogo como segundo
  pedido, três testes falharam porque o mock respondia na ordem `approvals → auditoria → agentes` e o
  painel pede `approvals → agentes → auditoria`. Corrigi a **ordem do mock**, não a do painel.
  ***Um mock com a ordem trocada testa uma sequência que o produto não faz — e passa a testar o mock.***
  **SEXTA LEITURA DO MESMO GATE, E O TETO NÃO MEXEU.** Mediu **48**, não os 45 previstos.
  `/service-agents` já estava coberta pelo prefixo de `/approvals` e `/audit` (E-1333). **Consumir uma
  rota a mais e o contador não mexer** é a sexta vez. A conta inteira: previ 3/caiu 10 (E-1328), 2/7
  (E-1329), 1/0 (E-1330), 7/7 (E-1331), 7/8 (E-1332), 3/5 (E-1333), **3/0 (E-1334)**. *Sete medições,
  uma previsão certa: o instrumento é que está errado.*
- **E-1333** — `ApprovalsPanel` com `role_checked`/quórum/`overdue`/notificação; 53 → 48.
- **E-1332** — `SalesPanel` com a conta aberta; frete não declarado ≠ R$ 0,00; 61 → 53.
- **E-1331** — `InboxPanel`; `reply_blocked` não esconde a conversa; 68 → 61.
- **E-1330** — detalhe do contacto com consentimento real; teto 68 inalterado e explicado.
- **E-1329** — `CRMPanel` com PII mascarada; 75 → 68.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **64** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **48** (era 93) |

**Os maiores domínios ainda sem UI:** `omnichannel` **5**, `auth` **4**, `supervisor` **4** (parcial),
`finance` **4**, `business` **4** (parcial), `plugins` **3**, `media` **2**, `company` **2**.

**Nota acumulada sobre o contador — sete avisos:** o gate mede o **domínio**, não a rota. Sete medições,
**uma** previsão certa. Corrigi-lo é trabalho próprio, e continua a ser a dívida de instrumento mais
clara do projecto.

O que continua aberto: **48 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 216**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — a única falha de
CI**; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1333

A cabeça do índice é **E-1333**.

- **E-1333** — **a fila de aprovação humana passou a ter superfície — é a FASE 4 e a FASE 11
  juntas.** `service-agents` tinha **6 rotas** e `service_agent_approvals.go` **45 KB** (quórum,
  política, fingerprint, expiração, notificação cross-processo) sem nenhum componente. Entregues
  `approvalsClient.ts` + `ApprovalsPanel.tsx`, consumindo `/service-agents/approvals` e
  `/service-agents/audit`.
  **(1) `role_checked=false` é uma declaração que o painel não pode esconder.** O tipo: *«`false` é
  declarado: sem lista configurada o sistema exige ator identificado, mas NÃO verifica se ele pode
  aprovar — e isso precisa aparecer no registro, em vez de virar confiança implícita.»* O painel escreve
  **«sem lista de aprovadores — apenas ator identificado»**. *Uma aprovação que não verificou QUEM
  aprovou não é a mesma aprovação.*
  **(2) Quórum é parcial, não binário** — «2 de 3 aprovações», nunca «pendente» a seco. **(3) `overdue`
  separa pendente de esquecido** — «PASSOU DO LIMIAR». **(4) A notificação pode ter falhado com a fila
  de pé** — «a fila não avisa ninguém» quando não há canal. **(5) O fingerprint é truncado** —
  identificador, não conteúdo. **(6) A falha da auditoria não apaga a fila**; e sem trilha durável a
  leitura diz «nada é guardado» — o servidor: *«Sem trilha durável a leitura é recusa declarada, não
  lista vazia: vazio pareceria "nada aconteceu".»*
  **QUINTA LEITURA DO GATE, E A CONTA FICA FEITA:** errada por 2 (E-1328), errada por 5 (E-1329),
  **certa (E-1331)**, errada por 1 (E-1332), errada por 2 (E-1333). *Cinco medições, uma previsão
  certa: o instrumento é que está errado, não a minha contabilidade.*
- **E-1332** — `SalesPanel` com a conta aberta; frete não declarado ≠ R$ 0,00; 61 → 53.
- **E-1331** — `InboxPanel`; `reply_blocked` não esconde a conversa; 68 → 61.
- **E-1330** — detalhe do contacto com consentimento real; teto 68 inalterado e explicado.
- **E-1329** — `CRMPanel` com PII mascarada; 75 → 68.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **64** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **48** (era 93) |

**Os maiores domínios ainda sem UI:** `omnichannel` **5**, `auth` **4**, `supervisor` **4** (parcial),
`finance` **4**, `business` **4** (parcial), `plugins` **3**, `media` **2**, `company` **2**.

**Nota acumulada sobre o contador — seis avisos:** o gate mede o **domínio**, não a rota. Cinco medições,
uma previsão certa. Corrigi-lo é trabalho próprio, e continua a ser a dívida de instrumento mais clara
do projecto.

O que continua aberto: **48 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 216**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; **`gitleaks` histórico — 2 de 17
check-runs deste HEAD, e as duas falhas são gitleaks**; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30
não iniciada**.

---
## JANELA E-1332

A cabeça do índice é **E-1332**.

- **E-1332** — **a conta fica aberta: não se mostra o total, mostra-se o total e as parcelas.** `sales`
  era o maior domínio restante (**8 rotas**), `sales_workforce.go` 23 KB, nenhum componente `*ales*`.
  Entregues `salesClient.ts` + `SalesPanel.tsx`, consumindo `/sales/proposals`,
  `/sales/proposals/:id` e `/sales/follow-ups`.
  **(1) O próprio tipo diz porquê, com o ID da evidência:** *«PRECIFICAÇÃO (E-269): a conta fica ABERTA
  na proposta. Um total único esconderia quanto foi desconto, quanto foi imposto e quanto foi frete.»*
  O painel mostra **subtotal, desconto, imposto, frete e total**; o teste falha se o total vier sem as
  parcelas ou com valores em centavos. *Uma proposta que mostra só o total não mostra a proposta: mostra
  o preço.*
  **(2) Frete não declarado não é R$ 0,00.** `shipping_declared`: *«Zero não informado não é zero
  cobrado.»* O painel escreve **«frete não declarado»**, e o teste verifica que a linha do frete **não
  contém `R$ 0,00`**. *Zero não informado e zero cobrado são o mesmo número e duas afirmações
  diferentes.*
  **(3) Pontos-base não são percentagem:** `500` → **5%**, `1250` → **12,5%**.
  **(4) `overdue` é query booleana**, com a caixa «Só acompanhamentos atrasados»; lista vazia nesse modo
  diz **«isto é diferente de não haver acompanhamentos»**.
  **(5) 503 não é lista vazia** (estado `sem-vendas`).
  **A PREVISÃO FALHOU POR 1, E ISSO FICA REGISTADO.** Previ 54; o gate mediu **53**. `sales` tem 8 rotas
  e **todas** caíram com 3 consumidas. Corrigi para o valor medido: *uma previsão que acerta uma vez não
  é um método: é uma coincidência que se pode repetir ou não.*
- **E-1331** — `InboxPanel`; `reply_blocked` não esconde a conversa; 68 → 61.
- **E-1330** — detalhe do contacto com consentimento real; teto 68 inalterado e explicado.
- **E-1329** — `CRMPanel` com PII mascarada; 75 → 68.
- **E-1328** — `CommercePanel`; 85 → 75.
- **E-1327** — `SchedulingPanel`; 87 → 85.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **69** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **53** (era 93) |

**Os maiores domínios ainda sem UI:** `service-agents` **6**, `omnichannel` **5**, `auth` **4**,
`supervisor` **4** (parcial), `finance` **4**, `business` **4** (parcial), `plugins` **3**, `media`
**2**, `company` **2**.

**Nota acumulada sobre o contador — cinco avisos:** E-1328 (3 de 10), E-1329 (2 de 8, 1 de detalhe
aberta), E-1330 (**1 consumida, contador inalterado**), E-1331 (3 de 8, previsão certeira), E-1332
(3 de 8, **previsão errada por 1**). O gate mede o **domínio**; corrigi-lo continua a ser trabalho
próprio.

O que continua aberto: **53 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 221**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; `gitleaks`; instalação limpa
`BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1331

A cabeça do índice é **E-1331**.

- **E-1331** — **`reply_blocked` não é um erro, é a decisão central desta fatia.** `inbox` era o maior
  domínio restante (**8 rotas**), com SLA, atribuição, handoff e conhecimento, e nenhum componente
  `*nbox*` (confirmado por glob **antes** de escrever — regra do E-1326). Entregues `inboxClient.ts` +
  `InboxPanel.tsx`, consumindo `/inbox/conversations`, `/inbox/conversations/:id` e `/inbox/metrics`.
  **(1) Um bloqueio de resposta não esconde a conversa.** O servidor: *«O consentimento do contato é
  anexado à visão para o atendente saber, sem consultar outro lugar, se pode responder.»* O painel
  mostra o bloqueio **com o motivo** e **continua a mostrar a conversa e as mensagens**. O teste
  verifica as **duas** coisas ao mesmo tempo — era o único que falharia se o painel escondesse a
  conversa ao ver o bloqueio. *Uma conversa que se pode ler e não se pode responder não é uma conversa
  bloqueada: é uma conversa com uma porta fechada.*
  **(2) Segundos não são minutos.** `125` aparece como **«2 min 5 s»**; o valor cru não aparece.
  **(3) `deduplicated` não é mensagem nova** — marcada «(reentrega — não é mensagem nova)».
  **(4) `last_message` é uma prévia** — o próprio tipo o diz; o painel acrescenta **(prévia)**.
  **(5) 503 não é lista vazia**; e `reply_blocked=false` diz «pode responder».
  **O número:** 68 → **61**, exactamente as 7 rotas do domínio `inbox`. Terceira fatia consecutiva com
  o efeito de prefixo — **e a primeira em que a previsão escrita ANTES da medição bateu certo**.
- **E-1330** — detalhe do contacto com consentimento real; teto 68 inalterado e explicado.
- **E-1329** — `CRMPanel` com PII mascarada; 75 → 68.
- **E-1328** — `CommercePanel`; 85 → 75.
- **E-1327** — `SchedulingPanel`; 87 → 85.
- **E-1326** — falso progresso: painel já existia, sobrescrito e restaurado (teto inalterado).

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **77** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **61** (era 93) |

**Os maiores domínios ainda sem UI:** `sales` **8**, `service-agents` **6**, `omnichannel` **5**,
`auth` **4**, `supervisor` **4** (parcial), `finance` **4**, `business` **4** (parcial), `plugins` **3**,
`media` **2**, `company` **2**.

**Nota acumulada sobre o contador — quatro avisos:** E-1328 (3 de 10), E-1329 (2 de 8, 1 de detalhe
aberta), E-1330 (**1 consumida, contador inalterado**), E-1331 (3 de 8, previsão certeira). O gate mede
o **domínio**; corrigi-lo continua a ser trabalho próprio.

O que continua aberto: **61 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 229**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; `gitleaks`; instalação limpa
`BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1330

A cabeça do índice é **E-1330**.

- **E-1330** — **o limite declarado no E-1329 passou a ser um dado lido.** O E-1329 dizia: «o
  `blocked_channels` só vem em `/crm/contacts/:id`, que esta fatia NÃO consome — o painel diz «sem
  informação de consentimento» em vez de inventar». Esta fatia consome-a: clicar no nome pede
  `/crm/contacts/:id` e mostra o `blocked_channels` **real**, o estado por canal e a contagem de
  oportunidades — tudo do servidor.
  **(1) A prova documental do consentimento não se mostra.** `ConsentRecord` traz `evidence` (URL do
  formulário, referência do contrato) e `reason`. São prova de terceiros e podem conter PII que o
  proprietário não pediu para ver neste ecrã. O painel mostra o **estado** e o **canal**; a prova fica
  onde está, e o painel di-lo. O teste discriminante verifica que **a URL da evidência e o número de
  documento não aparecem**. *Uma prova de consentimento que se mostra sem pedido é uma segunda recolha
  de dados pessoais.*
  **(2) Sem registo não é consentimento implícito** — «nenhum canal está autorizado», nunca «tudo
  permitido por omissão». **(3) A falha do detalhe não apaga a lista.**
  **TERCEIRO AVISO DO MESMO GATE — E O MAIS CLARO.** O teto **não baixou**: continua **68**. O
  `/crm/contacts/:id` já estava coberto pelo prefixo `/crm/contacts` da lista (E-1329). O painel passou
  a consumir **uma rota a mais** e o contador **não mexeu**. *Um instrumento que não distingue duas
  rotas do mesmo prefixo não conta rotas: conta prefixos com nome de rota.* **Três avisos em três
  fatias** (E-1328, E-1329, E-1330): o número que este gate baixa é do **domínio**; o número que eu
  fecho é o que os testes cobrem.
- **E-1329** — `CRMPanel` com PII mascarada; 75 → 68.
- **E-1328** — `CommercePanel`; 85 → 75.
- **E-1327** — `SchedulingPanel`; 87 → 85.
- **E-1326** — falso progresso: painel já existia, sobrescrito e restaurado (teto inalterado).
- **E-1325** — `DeploymentsPanel`; 88 → 87.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **84** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **68** (era 93) |

**Os maiores domínios ainda sem UI:** `inbox` **8**, `sales` **8**, `service-agents` **6**,
`omnichannel` **5**, `auth` **4**, `supervisor` **4** (parcial), `finance` **4**, `business` **4**
(parcial), `plugins` **3**, `media` **2**, `company` **2**.

**Nota acumulada sobre o contador — três avisos:** o gate mede o **domínio**, não a rota. E-1328 (3 de
10), E-1329 (2 de 8, com 1 de detalhe aberta), E-1330 (**1 rota nova consumida, contador inalterado**).
O instrumento tem um limite de medida que já ensinou três vezes a mesma lição; corrigi-lo é trabalho
próprio, não um efeito colateral de fechar painéis.

O que continua aberto: **68 rotas** sem UI; o gate **mede prefixos**; a assimetria `call.raw`/`call.path`;
**20 ORPHAN listados de 236**; o build **não é assinado nem instalado**; `sign.start`/`publish.*`
inexistentes; o painel do build **não se actualiza sozinho**; `gitleaks`; instalação limpa
`BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1329

A cabeça do índice é **E-1329**.

- **E-1329** — **PII mascarada por omissão numa lista — é a decisão central desta fatia.** `crm` tinha
  **8 rotas**, `crm.go` 18 KB, store com consentimento por canal, e nenhum componente `CRM*`.
  Entregues `crmClient.ts` + `CRMPanel.tsx`, consumindo `/crm/contacts` e `/crm/pipeline`.
  **(1) `phone` e `email` não se mostram inteiros numa lista.** O servidor devolve-os porque a API é do
  dono dos dados; **uma lista no ecrã é uma superfície diferente de uma resposta de API**. O painel
  mascara (`ana.silva@exemplo.com` → `a•••@exemplo.com`; `+351912345678` → `+351••••••678`) e diz que
  mascarou. O teste verifica que os valores **completos não aparecem** — é esse o teste que falharia
  com o defeito presente. *Uma lista de contactos que mostra os contactos todos não é uma lista: é um
  relatório que se pode fotografar.* Há «Mostrar dados pessoais», que diz que está a mostrar.
  **(2) Centavos não são reais** — reusa o `centavosParaReais` do `commerceClient` em vez de duplicar.
  *Uma segunda função de conversão é uma segunda verdade sobre o mesmo número.*
  **(3) 503 não é lista vazia** (estado `sem-crm`).
  **(4) Consentimento vem do registro, não do contacto** — o servidor diz: *«Consentimento NÃO é
  duplicado aqui: o estado vem do `ConsentRegistry`»*. **Limite declarado:** o `blocked_channels` só
  vem em `/crm/contacts/:id`, que esta fatia **não** consome; o painel diz «sem informação de
  consentimento» em vez de inventar.
  **UM ACHADO SOBRE O GATE, OUTRA VEZ:** o teto medido foi **68**, não os 67 esperados. Consumir
  `/crm/contacts` e `/crm/pipeline` resolveu 7 das 8 rotas; **`/crm/contacts/:id` continua órfão**
  porque o prefixo não cobre o sufixo `/:id`. *Um gate que compara prefixos não vê as rotas de
  detalhe: quem fecha a lista fecha a lista, não o detalhe.*
- **E-1328** — `CommercePanel`; 85 → 75 (3 consumidas, 7 por prefixo).
- **E-1327** — `SchedulingPanel`; 87 → 85.
- **E-1326** — falso progresso: painel já existia, sobrescrito e restaurado (teto inalterado).
- **E-1325** — `DeploymentsPanel`; 88 → 87.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **84** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **68** (era 93) |

**Os maiores domínios ainda sem UI:** `inbox` **8**, `sales` **8**, `service-agents` **6**,
`omnichannel` **5**, `auth` **4**, `supervisor` **4** (parcial), `finance` **4**, `business` **4**
(parcial), `plugins` **3**, `media` **2**, `company` **2**. Mais as **rotas de detalhe** dos domínios
já fechados (`/crm/contacts/:id`, `/commerce/orders/:id`).

**Nota acumulada sobre o contador:** o gate mede o **domínio**, não a rota. Declarar sempre quantas
rotas foram **de facto** consumidas — E-1328 (3 de 10) e E-1329 (2 de 8, com 1 de detalhe aberta).

O que continua aberto: **68 rotas** sem UI; a assimetria `call.raw`/`call.path`; **20 ORPHAN listados
de 236**; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel do
build **não se actualiza sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não
iniciada**.

---
## JANELA E-1328

A cabeça do índice é **E-1328**.

- **E-1328** — **o maior domínio restante passou a ter superfície. Teto 85 → 75.** `commerce` tinha
  **12 rotas** e um motor maduro (`commerce.go` 54 KB, `live_commerce.go` 29 KB, ~12 ficheiros de
  teste) sem nenhum componente `Commerce*`.
  Entregues `commerceClient.ts` + `CommercePanel.tsx`, consumindo `/commerce/products`,
  `/commerce/orders` e `/commerce/summary`.
  **(1) Centavos não são reais.** O painel converte; o teste recusa **tanto** `125000` **como**
  `R$ 1250,00` — a segunda asserção foi acrescentada ao ver que a primeira versão produzia o valor
  **sem separador de milhar**. A função passou a usar `toLocaleString('pt-BR')`. *Um valor de dinheiro
  na unidade errada não é um arredondamento: é uma mentira com duas casas decimais.* E *num valor de
  dinheiro, legibilidade não é estética.*
  **(2) 503 não é lista vazia.** Estado `sem-loja` separado, com «isto não é «não há produtos»: é «não
  há loja»» — exigida por teste.
  **(3) `duplicated` não é um pedido novo.** *«Duplicated marca a devolução idempotente do MESMO
  pedido.»* Contados em separado e marcados na linha.
  **(4) Estado desconhecido** sem rótulo inventado. **(5) Só lê**, três URLs exactas testadas.
  **UM ACHADO SOBRE O PRÓPRIO GATE:** o `hades_rotas_orfas.py` mede por **PREFIXO**, não por rota.
  Consumir 3 rotas de `/commerce` fez cair **10** lacunas de uma vez (85 → 75), não 3. O teto foi
  corrigido para **75** — o valor REAL — e a nota ficou escrita no próprio ficheiro: *um gate que
  agrupa por prefixo mede o domínio, não a rota: é bom para achar domínios esquecidos e enganador para
  contar rotas fechadas.* Registar 82 seria creditar-me 7 rotas que não fechei.
- **E-1327** — `SchedulingPanel`; 87 → 85.
- **E-1326** — falso progresso: painel já existia, sobrescrito e restaurado (teto inalterado).
- **E-1325** — `DeploymentsPanel`; 88 → 87.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **91** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **75** (era 93) |

**Os maiores domínios ainda sem UI:** `crm` **8**, `inbox` **8**, `sales` **8**, `service-agents`
**6**, `omnichannel` **5**, `auth` **4**, `supervisor` **4** (parcial), `finance` **4**, `business`
**4** (parcial), `plugins` **3**, `media` **2**, `company` **2**.

**Nota para as próximas rondas:** o mesmo efeito de prefixo vai repetir-se. Fechar `crm` (8 rotas) com
um painel de leitura pode valer 8 no contador — declarar sempre quantas foram **de facto** consumidas.

O que continua aberto: **75 rotas** sem UI; a assimetria `call.raw`/`call.path`; **20 ORPHAN listados
de 243**; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel do
build **não se actualiza sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não
iniciada**.

---
## JANELA E-1327

A cabeça do índice é **E-1327**.

- **E-1327** — **o maior domínio sem superfície passou a ter uma. Teto 87 → 85.** `scheduling` era o
  maior domínio órfão (**14 rotas**), com um motor maduro (`scheduling.go` 21 KB,
  `scheduling_waitlist.go` 17 KB, `scheduling_hours.go` 15 KB, ~20 ficheiros de teste) e **nenhum
  componente no frontend**.
  **A regra do E-1326 aplicada ANTES de escolher:** cruzei as 87 lacunas por domínio com os componentes
  existentes. `scheduling` não tinha nenhum `Scheduling*`. Desta vez o painel **não existia**.
  Entregues `schedulingClient.ts` + `SchedulingPanel.tsx`, consumindo `/scheduling/services` e
  `/scheduling/appointments`.
  **(1) 503 NÃO É LISTA VAZIA.** O servidor: *«sem store não há superfície — e dizer isso é melhor que
  devolver lista vazia, que pareceria 'não há serviços' em vez de 'não há motor ligado'.»* Estado
  `sem-motor` separado, com a frase «isto não é «não há serviços»: é «não há motor»» — exigida por
  teste.
  **(2) `busy_until` NÃO É `ends_at`.** O buffer é o intervalo real em que o recurso não pode receber
  outro atendimento; o painel mostra «+10 min de buffer» quando diferem.
  **(3) Estado fechado** — fora da lista aparece como «estado desconhecido (valor)», sem rótulo
  inventado.
  **(4) Motor ligado com zero serviços** diz isso e é diferente de motor ausente.
  **(5) Só lê**, prefixo explícito, URLs exactas testadas.
- **E-1326** — falso progresso: painel já existia, sobrescrito e restaurado (teto **inalterado**).
- **E-1325** — `DeploymentsPanel`; 88 → 87.
- **E-1324** — `RoutingPanel`; 89 → 88.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **101** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **85** (era 93) |

**Os maiores domínios ainda sem UI:** `commerce` **12**, `crm` **8**, `inbox` **8**, `sales` **8**,
`service-agents` **6**, `omnichannel` **5**, `auth` **4**, `supervisor` **4** (parcial), `finance`
**4**, `business` **4** (parcial), `plugins` **3**, `media` **2**, `company` **2**.

O que continua aberto: **85 rotas** sem UI; a assimetria `call.raw`/`call.path`; **20 ORPHAN listados
de 253**; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel do
build **não se actualiza sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não
iniciada**.

---
## JANELA E-1326

A cabeça do índice é **E-1326**.

- **E-1326** — **não é progresso de produto: é um erro meu, a recuperação dele, e a lição.**
  Escolhi `/worktrees` como próxima lacuna porque tem **três rotas**. **O painel já existia** — 334
  linhas, FASE 2 (E-1226..E-1236), montado no `CapabilitiesPage` L314, 12 testes a passar, e o
  `worktreeClient.ts` a consumir as três rotas.
  **O que fiz de errado:** chamei `write` sobre o `WorktreesPanel.tsx` **sem o ler primeiro**.
  Sobrescrevi 334 linhas com 190 piores.
  **O sinal que ignorei:** o `write` devolveu **«Updated file»**, não «Created file».
  *Um `write` que diz «Updated» está a dizer que o ficheiro já existia — e «Updated» é a palavra que
  devia parar a mão.*
  **A recuperação, provada:** `git show HEAD:<path>` + `WriteAllLines` (**sem `git checkout --`**, que a
  FASE 0 proíbe) → `git diff --exit-code` **exit 0**; **sha HEAD = sha disco =
  `f8540c5db1a5fae4ad8afc9b6516f0954060bb5e`**; `tsc` **0**; **12/12 PASS**. O `git status` continuou a
  dizer `M` — era **só o stat cache de mtime**. *Um `git status` que diz «modificado» quando os hashes
  são iguais está a reportar o relógio do filesystem, não o conteúdo.*
  **Duplicação apagada:** o meu `worktreesClient.ts` (6734 B) duplicava o `worktreeClient.ts` (5233 B)
  existente. Removido. *Dois clientes para a mesma rota não são dois clientes: são a mesma rota com
  dois contratos que um dia divergem.*
  **A lição dupla:**
  **(1)** Escolhi a fatia pelo lado **errado** do gate — li a lista de lacunas e **presumi** ausência de
  painel. *Um inventário de lacunas que mede o frontend não serve para escolher a próxima fatia sem se
  confirmar no frontend que ela falta.*
  **(2)** Falhou a disciplina mais básica: **ler antes de escrever**. *O erro mais caro de uma sessão
  longa não é o mais difícil: é o mais básico, repetido quando a confiança já é alta.*
  **Estado do produto: INALTERADO.** Teto continua em **87**. Registar isto como progresso seria mentir.
- **E-1325** — `DeploymentsPanel`; teto 88 → 87.
- **E-1324** — `RoutingPanel`; teto 89 → 88.
- **E-1323** — `TracesPanel`; teto 90 → 89.

## A distância, agora (inalterada)

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **103** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **87** (era 93) |

**Regra nova para a próxima ronda:** antes de escolher uma lacuna, **confirmar no frontend que o painel
falta** (grep no `src/components/` e no `CapabilitiesPage`), e não apenas que a rota existe no backend.

O que continua aberto: **87 rotas** sem UI; a assimetria `call.raw`/`call.path`; **20 ORPHAN listados de
253**; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel do
build **não se actualiza sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não
iniciada**.

---
## JANELA E-1325

A cabeça do índice é **E-1325**.

- **E-1325** — **a quarta das 93 passou a ser produto. Teto 88 → 87.** `/deployments` mostra os
  providers de deploy configurados, e o contrato tem **duas armadilhas de segurança** que o painel
  resolveu em vez de contornar.
  **(1) `TokenEnv` é o NOME da variável, não o token** — o painel escreve «token lido de `VERCEL_TOKEN`
  (nome da variável, não o valor)», e o teste exige essa frase. *Um campo chamado «token» que mostra o
  nome da variável é lido como o token por quem tem pressa — e a pressa é o estado normal de quem
  opera.*
  **(2) `base_url` pode trazer credenciais em userinfo** — `https://user:senha@host/`. `redigirBaseURL`
  remove o userinfo, o painel mostra o selo **«credenciais removidas»** e um aviso que diz que a
  configuração no servidor não foi alterada. O teste verifica que **`senha-secreta` e `utilizador` NÃO
  aparecem** — é esse o teste que falharia com o defeito presente. *Uma credencial escondida sem aviso
  é uma credencial que ninguém sabe que existe.*
  **(3) Lista vazia não é «pronto»** — sem provider não há para onde publicar, e o painel di-lo.
  **(4) Um provider configurado não é uma publicação feita** — escrito na fronteira, exigido por teste.
  **(5) Só lê**, testado.
  **O padrão das quatro rotas fechou:** `/retention`, `/traces`, `/routing`, `/deployments` — prefixo
  explícito e só-GET provado, nas quatro, antes do CI.
- **E-1324** — `RoutingPanel`; teto 89 → 88.
- **E-1323** — `TracesPanel`; teto 90 → 89.
- **E-1322** — o gate de contrato na bateria local.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **103** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **87** (era 93) |

O que continua aberto: **87 rotas** sem UI; a assimetria `call.raw`/`call.path` no
`verify-contracts.mjs`; **20 ORPHAN listados de 253** sem catraca no total; o build **não é assinado nem
instalado**; `sign.start`/`publish.*` inexistentes; o painel do build **não se actualiza sozinho**;
`gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1324

A cabeça do índice é **E-1324**.

- **E-1324** — **a terceira das 93 passou a ser produto. Teto 89 → 88.** `/routing` é a rota que
  responde **porquê o HADES escolheu um modelo** — e até agora ninguém a podia perguntar. É a FASE 19
  (roteamento multiobjetivo e fronteira de Pareto).
  Entregues `routingClient.ts` + `RoutingPanel.tsx`, montado no `CapabilitiesPage`.
  **As decisões que o painel não pode errar:**
  **(1) O 422 é RECUSA, não avaria.** Quando nenhum modelo satisfaz as restrições o servidor responde
  422 com as restrições no texto; o painel mostra «Nenhum modelo satisfaz» e diz que é resposta do
  roteador. *Uma recusa que explica as restrições que a produziram é informação; a mesma recusa
  mostrada como «falhou» é ruído.*
  **(2) Os factores aparecem** — o próprio código do servidor diz *«Um score sem os componentes que o
  somam obriga quem o lê a confiar em quem o escreveu»*. O teste exige `capacidade=0.90`, `custo=1.00`,
  `latencia=0.60` ao lado do score.
  **(3) A fronteira de Pareto não é um ranking** — cinco eixos, rotulados como alternativas.
  **(4) A query não leva parâmetros vazios** — o teste da primeira consulta exige exactamente
  `/api/agent/v1/routing?preference=auto`.
  **(5) `cost_cents_per_1k = 0` diz-se GRÁTIS**, não `R$ 0,00`.
  *Três rotas seguidas com as mesmas duas armadilhas (prefixo explícito, só GET) resolvidas antes do
  CI — a lição virou regra, não remendo.*
- **E-1323** — `TracesPanel`; teto 90 → 89.
- **E-1322** — o gate de contrato na bateria local.
- **E-1321** — o gate compara o path cru; o cliente omitia o prefixo.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **104** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **88** (era 93) |

O que continua aberto: **88 rotas** sem UI; a assimetria `call.raw`/`call.path` no
`verify-contracts.mjs`; **20 ORPHAN listados de 254** sem catraca no total; o build **não é assinado nem
instalado**; `sign.start`/`publish.*` inexistentes; o painel do build **não se actualiza sozinho**;
`gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1323

A cabeça do índice é **E-1323**.

- **E-1323** — **a segunda das 93 passou a ser produto. Teto 90 → 89.** `/traces` era uma rota de
  leitura com backend pronto e nenhuma forma de ver o que devolvia.
  Entregues `tracesClient.ts` + `TracesPanel.tsx`, montado no `CapabilitiesPage`.
  **As decisões que o painel não pode errar:**
  **(1) `duration_ns` não é duração para humano** — o `time.Duration` do Go serializa como inteiro de
  nanossegundos; o painel diz `4.82 s` e o teste exige que `4820000000` **não apareça**.
  **(2) Um span sem `end_at` está ABERTO, não concluído.** *Um span aberto mostrado como concluído é um
  trace que mente sobre o tempo que levou.*
  **(3) O erro vence o rótulo** — `status:"ok"` + `error` preenchido é FALHADO.
  **(4) Lista vazia e `spans:null` não são erro.** **(5) Só lê** — um teste exige `method === "GET"` em
  todas as chamadas.
  **O prefixo explícito, aprendido no E-1321:** `/api/agent/v1/traces`, não `/traces`. A armadilha foi
  aplicada **antes** de o CI a apanhar. *Uma lição que só se aplica onde doeu não é uma lição: é um
  remendo.*
  **Um defeito meu no teste:** `texto()` lia `props.children` de um nível, mas os números estão em
  `<span>` aninhados — três asserções falharam sem números. Passou a ler o `toJSON()` recursivamente.
  *Uma função de teste que só lê um nível mede um nível.*
- **E-1322** — o gate de contrato na bateria local.
- **E-1321** — o gate compara o path cru; o cliente omitia o prefixo.
- **E-1320** — `RetentionPanel`; teto 93 → 90.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **105** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **89** (era 93) |

O que continua aberto: **89 rotas** sem UI; a assimetria `call.raw`/`call.path` no
`verify-contracts.mjs`; **20 ORPHAN listados de 255** sem catraca no total; o build **não é assinado nem
instalado**; `sign.start`/`publish.*` inexistentes; o painel do build **não se actualiza sozinho**;
`gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1322

A cabeça do índice é **E-1322**.

- **E-1322** — **o gate que apanhou o E-1321 não estava na bateria local.** O `verify-contracts.mjs`
  corre no CI (`Preserve Class A+ surfaces`) desde antes do E-1321 — e **nunca** correu na minha
  bateria. Foi por isso que um cliente meu com o path sem prefixo **viajou até ao PR**.
  *Um gate que não está na bateria local é um gate que só existe no CI, e a distância entre os dois é o
  tempo de uma ronda.*
  **É a TERCEIRA VEZ na mesma sessão que o padrão aparece:** o E-1319 escreveu um instrumento que o
  `hades_orphan_instruments_check.py` acusou de não estar invocado; agora descobri que um instrumento
  **que já existia** também não estava.
  *Um gate que existe e não está na bateria é um gate que só protege quem corre o CI — e o CI corre
  depois do push, quando o defeito já viajou.*
  **Entregue:** `node scripts/verify-contracts.mjs` no `hades_local_gates.ps1`, depois do
  `hades_rotas_orfas.py`. `.ps1` **ASCII puro** (0 bytes > 127), parser 5.1 **0 erros**.
  **O que isto muda:** escrever um `agentFetch` fora da convenção passa a falhar **na máquina**.
  **O que não resolve:** a assimetria `call.raw` vs `call.path` fica como **dívida conhecida** — não a
  corrigi, porque alargar o gate acomodaria o cliente mal escrito.
- **E-1321** — o gate compara o path cru; o cliente omitia o prefixo.
- **E-1320** — `RetentionPanel`; teto 93 → 90.
- **E-1319** — auditoria de rotas órfãs.

## O que continua aberto

A assimetria `call.raw` vs `call.path` no `verify-contracts.mjs`; **20 ORPHAN listados de 256** (o gate
mostra os primeiros 20 e o total não tem catraca); **90 rotas** sem superfície no UI; o build **não é
assinado nem instalado**; `sign.start`/`publish.*` inexistentes; `gitleaks`; instalação limpa
`BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1321

A cabeça do índice é **E-1321**.

- **E-1321** — **o gate de contrato compara o path CRU, e o `agentFetch` aceita o path sem prefixo.**
  Escrevi `agentFetch("/retention")`. Em runtime funcionava — o painel leu e gravou a política. O
  `verify-contracts.mjs` reprovou: `routeMatches(call.raw, route.path)` compara o path **cru**
  (`/retention`) com o path **com prefixo** (`/api/agent/v1/retention`).
  *Um cliente que funciona em runtime e reprova no contrato não está errado no produto: está fora da
  convenção que o gate mede — e a convenção é o que mantém 202 chamadas verificáveis.*
  **E o CI apanhou-o, o que é o sistema a funcionar.** O job `Preserve Class A+ surfaces` reportou os
  três paths em 12 minutos. Na minha bateria local eu **não corria** o `verify-contracts.mjs`.
  *Um gate que não está na bateria local é um gate que só existe no CI, e a distância entre os dois é o
  tempo de uma ronda.*
  **A investigação:** suspeitei do gate, não do cliente. Instrumentei-o e vi que **os dois lados
  estavam lá** — 3 rotas no backend, 3 chamadas no frontend. O desalinhamento estava num argumento:
  `matches` usa `call.raw`, não `call.path`. Todos os 202 clientes existentes passam o prefixo
  **explícito**; o meu era o único a omiti-lo.
  **Correcção:** os três paths passaram a `/api/agent/v1/retention` e `.../retention/apply`, com a
  armadilha documentada no topo do ficheiro.
  **O que NÃO fiz:** não alarguei o `verify-contracts.mjs` para aceitar as duas formas.
  *Um gate que se alarga para acomodar o defeito deixa de medir o que media.*
- **E-1320** — `RetentionPanel`; teto 93 → 90.
- **E-1319** — auditoria de rotas órfãs.
- **E-1318** — `PodarDisco` ligado ao arranque.

## O que continua aberto

**O `verify-contracts.mjs` não está na bateria local** — só corre no CI, e foi por isso que o defeito
viajou até ao PR. **20 ORPHAN listados de 256**: o gate só mostra os primeiros 20 e o total não tem
catraca. **90 rotas** ainda sem superfície no UI; o build **não é assinado nem instalado**;
`sign.start`/`publish.*` inexistentes; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não
iniciada**.

---
## JANELA E-1320

A cabeça do índice é **E-1320**.

- **E-1320** — **a primeira das 93 lacunas passou a ser produto.** `/retention` (GET/PUT) e
  `/retention/apply` (POST) eram três das 93 medidas no E-1319 — e as mais irónicas, porque o E-1316,
  E-1317 e E-1318 foram todos sobre retenção que existia e não acontecia.
  *Uma rota que existe e não é consumida é uma promessa do backend, não uma funcionalidade do produto.*
  Entregues `retentionClient.ts` + `RetentionPanel.tsx`, montado no `CapabilitiesPage`. O painel lê a
  política, grava-a e pede a aplicação, mostrando os **números reais** do que foi removido.
  **As decisões que o painel não pode errar:**
  **(1) `max_memories_per_project = 0` significa SEM TECTO** — o servidor aceita 0 como legítimo
  (0..10000). Dizer «0 memórias por projecto» diria o oposto do que a política faz.
  *Zero não é sempre um limite: às vezes é a ausência dele, e uma interface que confunde os dois mente
  sobre o que vai apagar.*
  **(2) «Não configurada» é um ESTADO, não um erro.** **(3) Gravar não é aplicar** — o aviso diz
  «Nada foi removido ainda». **(4) Zero removido diz-se em palavras.**
  **Dois defeitos meus, apanhados pelos gates:** `TS2367` numa condição que não filtrava nada; e o
  teste do `act` — no mesmo `act` o `onChange` e o clique não deixam re-renderizar, o clique apanhava
  o estado antigo e o PUT saía. *Um teste que mede o estado anterior aprova o defeito que devia travar.*
  **O TETO DESCEU: 93 → 90**, e o `TETO_ORFAS` acompanhou.
  *Baixar o teto é progresso; deixar o teto em 93 depois de fechar 3 rotas seria guardar crédito por
  trabalho já feito.*
- **E-1319** — auditoria de rotas órfãs: 93 lacunas, com gate e teto.
- **E-1318** — `PodarDisco` ligado ao arranque.
- **E-1317** — retenção nos dois sítios; `misspell` resolvido por `ignore-rules`.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **106** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **90** (era 93) |

O que continua aberto: **90 rotas** ainda sem superfície no UI; as 93 não foram reclassificadas uma a
uma; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel de
estado do build **não se actualiza sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`;
**FASE 30 não iniciada**.

---
## JANELA E-1319

A cabeça do índice é **E-1319**.

- **E-1319** — **duas rondas seguidas com o mesmo padrão, e eu a procurar um de cada vez.**
  O E-1316 foi `persistir()` escrito e nunca invocado; o E-1318 foi `PodarDisco()` escrito e nunca
  invocado. Achei os dois por sorte, dentro da fatia em que estava a trabalhar. Se o padrão aconteceu
  duas vezes em dois subsistemas que eu conhecia, o número real tinha de ser **medido**.
  *Um defeito que se encontra por sorte não foi encontrado: foi tropeçado.*
  **MEDIDO:** 348 rotas no backend · **109 não referenciadas** no frontend nem no E2E · dessas
  **16 são orfandade CORRECTA** (webhooks, `/metrics`, `/auth/saml/*`, `/mcp/rpc`, pareamento) e
  **93 são lacuna de produto**. Maiores: **scheduling 14, commerce 12, crm 8, inbox 8, sales 8,
  service-agents 6, omnichannel 5** — 21 domínios.
  *Uma rota que existe e não é consumida é uma promessa do backend, não uma funcionalidade do produto.*
  **Entregue** `scripts/hades_rotas_orfas.py` + teste, com **`TETO_ORFAS = 93`**: falha se subir, baixa
  quando descer, e **exige razão escrita para cada prefixo de consumidor-externo** — para que
  «orfandade correcta» não vire a gaveta onde se esconde o que não se quer ver.
  *Um teto não é uma aprovação: é o número de hoje, congelado para que a próxima ronda não o aumente
  sem que alguém escreva porquê.*
  **E o gate está LIGADO ao `hades_local_gates.ps1`** — sem isso seria o terceiro instrumento escrito e
  nunca invocado na mesma sessão. *Um gate que existe e não é invocado é um gate que não existe.*
  **O que o instrumento NÃO prova:** não prova que as 93 sejam todas necessárias no UI — algumas podem
  ser legitimamente só de CLI e estar mal classificadas. Reclassificar é a próxima fatia.
- **E-1318** — `PodarDisco` ligado ao arranque.
- **E-1317** — retenção nos dois sítios; `misspell` resolvido por `ignore-rules`.
- **E-1316** — persistência do store ligada ao `DataRoot`.

## A distância medida

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **109** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **93** |

O que continua aberto: as 93 não foram reclassificadas uma a uma; **nenhuma** foi ligada ao UI nesta
ronda; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel de
estado **não se actualiza sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não
iniciada**.

---
## JANELA E-1318

A cabeça do índice é **E-1318**.

- **E-1318** — **a função de limpeza existia e nunca corria.** Entreguei `PodarDisco()` no E-1317 com
  quatro testes verdes — e **nenhum chamador no produto**. O `storeDeBuilds()` chamava
  `CarregarDoDisco()` e devolvia. Um servidor que arranca com 500 ficheiros e fica parado nunca volta a
  chamar `podar` (que só corre a cada build novo), portanto o disco nunca descia.
  *Uma função de limpeza sem chamador é a mesma família do store que persistia e nunca escrevia: código
  que existe e não acontece.*
  **É a SEGUNDA VEZ no mesmo subsistema.** No E-1316 o defeito foi `persistir()` escrito e nunca
  invocado; agora `PodarDisco()` escrito e nunca invocado. Os dois passaram nos testes porque o teste
  chamava a função diretamente.
  *Um teste que chama a função que o produto não chama prova que a função funciona, não que o produto a
  usa.*
  Entregue `PodarDisco()` no arranque, depois do `CarregarDoDisco()`; `MaxBuildsGuardadosParaTeste()`
  expõe o teto; teste `TestArranqueDoStoreAplicaRetencao` com 212 ficheiros.
  **Um `nolintlint` FALSO:** com o typecheck do teste quebrado, o `golangci-lint` reportou dois
  `//nolint:nilerr` como «unused» **noutro ficheiro**. Corrigidos os imports, 0 issues.
  *Um linter que reporta noutro ficheiro quando um terceiro não compila ensina a procurar o defeito onde
  ele não está.*
- **E-1317** — retenção nos dois sítios; `misspell` resolvido por `ignore-rules`.
- **E-1316** — persistência do store ligada ao `DataRoot`.
- **E-1315** — botão de iniciar build.

## O que continua aberto

Build **não assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel de estado **não se
atualiza sozinho**; **a poda no arranque corre uma vez** — um servidor com uptime longo volta a crescer
até ao próximo build, e não há poda periódica; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30
não iniciada**.

---
## JANELA E-1317

A cabeça do índice é **E-1317**.

- **E-1317** — **dois defeitos: um de retenção, um de ferramenta.**
  **(1) A retenção só valia para metade dos sítios.** O `podar()` descartava os registos terminados do
  mapa em memória e **deixava os ficheiros em disco a crescer**. *Uma política de retenção que só conta
  os registos que caberam em memória não é uma política de retenção: é uma política de memória.*
  Corrigido: `podar()` apaga o `.json` com o mesmo critério, e há `PodarDisco()` para o caso que o
  `podar` não cobria — um processo que arranca, carrega 200 registos e fica parado nunca volta a chamar
  `podar`. `PodarDisco` **nunca remove um build a correr neste processo**.
  **(2) A oitava ocorrência do `misspell` — e a causa era outra.** O `.golangci.yaml` **já tinha
  `ignore-rules`** para o `misspell` com cinco palavras portuguesas legítimas. `discriminante` não
  estava lá. E o `misspell` **só flagra o SINGULAR** — o plural `discriminantes` já passava, como
  `toolchain_plan_test.go:3` provava sem que eu reparasse. As outras ocorrências estão em `.ts`/`.tsx`,
  que o `golangci-lint` não analisa.
  **Passei oito rondas a reescrever comentários quando a solução era uma linha que o repo já tinha.**
  *Reescrever o comentário oito vezes é tratar o sintoma de um corretor ortográfico com a ferramenta
  errada.* — e o comentário reescrito ficava **pior**, porque perdia o termo técnico exacto.
  **Prova da ignore-rule:** repus a palavra de propósito num comentário e `golangci-lint` deu **0
  issues** (antes: `misspell: 1`). A regra foi testada com o caso que falhava, não aceite por confiança.
  **Medido:** 4 testes de retenção PASS · `./server` ok 44.664 s · `./internal/agent` ok 347.894 s.
- **E-1316** — persistência do store ligada ao `DataRoot`; builds abandonados marcados `FALHOU`.
- **E-1315** — botão de iniciar build; alvo bloqueado não ganha botão.
- **E-1314** — painel de estado; o 409 de `cancel` é resultado, não falha.

## `PodarDisco` — o que ainda falta

`PodarDisco` existe e está testado, mas **não está ligado a nenhum agendador**: só corre quando alguém o
chama. A retenção em disco funciona no caminho do `podar()` (a cada build novo), mas um servidor parado
com disco cheio não se limpa sozinho.

O que continua aberto: build **não assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o
painel de estado **não se actualiza sozinho**; `PodarDisco` sem agendador; `gitleaks`; instalação limpa
`BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1316

A cabeça do índice é **E-1316**.

- **E-1316** — **o store de builds do E-1313 tinha persistência MORTA.** `NovoBuildStoreComDirectorio` e
  `persistir()` existiam, e o único chamador da fábrica com diretório era **o próprio teste**. O
  `storeDeBuilds()` usava `NovoBuildStore()`, sem diretório ⇒ `persistir()` nunca escrevia nada.
  *Um store que existe e não persiste é um store que perde o trabalho quando o processo reinicia — e um
  reinício não é um caso raro: é a próxima vez que o utilizador abrir o produto.*
  **E o pior: declarei a limitação em três evidências como se fosse desenho.** Não era — era código
  escrito e não ligado. *Declarar não é corrigir.*
  Entregues `CarregarDoDisco()` e `Directorio()`; o `storeDeBuilds()` passou a usar
  `Runtime.DataRoot()/builds` (o getter já existia — não inventei caminho novo).
  **A decisão que o carregamento não pode errar: um registo `A_EXECUTAR` em disco NÃO está a correr.**
  O processo morreu. Passam a `FALHOU` com motivo explícito «interrompido».
  *Um build que o processo abandonou não está a correr: está interrompido, e o estado tem de o dizer.*
  **Ficheiro corrompido não derruba o arranque**, mas também não conta como build válido.
  **Medido:** `go build ./...` 0 · `./server` **ok 54 s** · `./internal/agent` **ok 382 s** · lint 0 ·
  4 testes novos PASS. **Sexta ocorrência do `misspell` a recusar «discriminante».**
- **E-1315** — botão de iniciar build; alvo bloqueado não ganha botão. 131 ficheiros / 944 testes.
- **E-1314** — painel de estado de build; o 409 de `cancel` é resultado, não falha.
- **E-1313** — `build.status`/`logs`/`artifacts`/`cancel` + `BuildStore`.

## Ciclo de build — estado real

| passo | backend | UI | persiste |
|---|---|---|---|
| planear | **existe** | **existe** | — |
| iniciar | **existe** | **existe (E-1315)** | **sim (E-1316)** |
| acompanhar | **existe** | **existe (E-1314)** | **sim (E-1316)** |
| logs / artefactos | **existe** | **existe (E-1314)** | **sim (E-1316)** |
| cancelar | **existe** | **existe (E-1314)** | **sim (E-1316)** |
| assinar | **AUSENTE** | — | — |
| publicar | **AUSENTE** | — | — |

O que continua aberto: build **não assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o
painel de estado **não se actualiza sozinho**; **os ficheiros em disco não são podados** (o tecto de
200 é só em memória); `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1315

A cabeça do índice é **E-1315**.

- **E-1315** — **o `build.start` não tinha botão.** O E-1313 deu-lhe modo assíncrono e o E-1314 deu
  painel ao estado; entre os dois faltava o gesto que cria o build. *Um backend completo e um produto
  incompleto não são a mesma coisa, e a distância entre os dois é a interface.*
  Entregues `iniciarBuild()` em `buildStatusClient.ts` (`POST /build/start`) e, no `BuildPlanPanel`,
  uma coluna **Ações** com o botão por alvo pronto.
  **A decisão que um painel apressado erraria: alvo bloqueado NÃO ganha botão, nem desactivado.**
  *Um botão que nunca funciona ensina a ignorar botões* — e um desactivado sem explicação ensina o
  mesmo. Para um alvo bloqueado o painel mostra `—`, e a razão já está na coluna «O que falta».
  **O modo assíncrono é o padrão do botão.** *Um `build.start` síncrono é um pedido que só responde
  quando já não há nada a cancelar* — e o `cancel` só tem alvo enquanto o build corre.
  **O 412 é um resultado:** o painel diz **«O plano de «X» está incompleto»**, não «erro de rede».
  Mesma família da decisão do E-1314 sobre o 409.
  **Medido:** `tsc -b` 0 · `vitest` **131 ficheiros / 944 testes PASS** (era 130/939) · `eslint` 0
  errors · `vite build` 0 · bundle **PASS** (435 057 ≤ 600 000) · 5 testes novos (os 17 pré-existentes
  do painel continuam PASS).
- **E-1314** — painel de estado de build; 409 de `cancel` é resultado, não falha. 130/939 testes.
- **E-1313** — `build.status`/`logs`/`artifacts`/`cancel` + `BuildStore`.
- **E-1312** — CI verde (`d75b2ff16`). Instalação limpa `BLOCKED_EXTERNAL` com prova.
- **E-1311** — `ResumoDeBlockers` mentia; **uma correção de ~20 linhas fechou as 7 falhas de CI.**

## Ciclo de build no UI — fechado

| passo | backend | UI |
|---|---|---|
| planear | `GET /build/plan` | **existe** |
| **iniciar** | `POST /build/start` | **existe (E-1315)** |
| acompanhar | `GET /build/status/:id` | **existe (E-1314)** |
| logs / artefactos | `GET /build/logs`, `/artifacts` | **existe (E-1314)** |
| cancelar | `POST /build/cancel/:id` | **existe (E-1314)** |
| assinar | `sign.start` | **AUSENTE** |
| publicar | `publish.submit` | **AUSENTE** |

O que continua aberto: build iniciado **não é assinado nem instalado**; store **não sobrevive a
reinício**; `sign.start`/`publish.*` mutantes inexistentes; **o painel de estado não se actualiza
sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1314

A cabeça do índice é **E-1314**.

- **E-1314** — **as cinco rotas do E-1313 não tinham cliente.** O servidor sabia responder
  `build.status`/`logs`/`artifacts`/`cancel`; o produto não perguntava. *Uma rota que existe e não é
  consumida é uma promessa do backend, não uma funcionalidade do produto.*
  Entregues `app/ui/app/src/lib/buildStatusClient.ts` e
  `app/ui/app/src/components/BuildStatusPanel.tsx`, montado em `CapabilitiesPage`.
  **A decisão que um painel ingénuo erraria:** o **409** de `cancel` é um **resultado**, não uma
  falha. Cancelar um build que já acabou devolve 409; mostrar «falhou ao cancelar» seria contar a
  história errada sobre um servidor que funcionou. O painel diz **«já tinha terminado — cancelar não
  tem efeito»**. Um erro de rede real continua a aparecer como erro.
  **O painel não calcula nada:** `nada_assinado`/`nada_publicado`/`nada_instalado` e `cancelavel` vêm
  do servidor; `EstadoDeBuild` é texto e não é remapeado.
  **Medido:** `tsc -b` 0 · `vitest` **130 ficheiros / 939 testes PASS** (era 129/931) · `eslint` 0
  errors (31 warnings pré-existentes) · `vite build` 0 · bundle **PASS** (434 737 ≤ 600 000) ·
  8 testes novos do painel.
- **E-1313** — `build.status`/`logs`/`artifacts`/`cancel` + `BuildStore`. `cancel` de terminado → 409;
  build cancelado **não** é des-cancelado. `./server` ok 49 s, `./internal/agent` ok 378 s.
- **E-1312** — CI verde (`d75b2ff16`: 21 success, só `gitleaks`). Instalação limpa
  `BLOCKED_EXTERNAL` com prova.
- **E-1311** — `ResumoDeBlockers` mentia: `alvos_bloqueados=3` com `blockers=[]`. **Uma correção de
  ~20 linhas fechou as 7 falhas de CI.**

## Cobertura do Release Hub

| superfície | backend | cliente (UI) |
|---|---|---|
| `toolchain.*`, `project.detect` | **existe** | **existe** |
| `build.plan` | **existe** | **existe** |
| `build.start` | **existe** (síncrono + assíncrono) | **AUSENTE** |
| **`build.status`/`logs`/`artifacts`/`cancel`** | **existe** | **existe (E-1314)** |
| `build.verify`, `build.smoke`, `artifact.*`, `sbom.generate` | **existe** | **existe** |
| `sign.plan`, `credentials.status`, `signing.*` | **existe** | **existe** |
| `sign.start`/`sign.status`/`sign.verify` | **AUSENTE** | — |
| `publish.preflight` | **existe** | **existe** |
| `publish.request_approval`/`submit`/`status`/`rollback` | **AUSENTE** | — |
| `provenance.attest`, `policy.evaluate` | **AUSENTE** | — |

O que continua aberto: **`build.start` não tem botão no UI** (o assíncrono muito menos); o store não
sobrevive a reinício; nada assinado/publicado/instalado; `sign.start`/`publish.submit`/`rollback`
inexistentes; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1313

A cabeça do índice é **E-1313**.

- **E-1313** — **as quatro rotas que faltavam, e o store que nenhuma delas pode dispensar.** O
  doc-comment do `build.verify` dizia *«não existe registo de builds»*. Entregues
  `internal/agent/build_store.go` (`BuildStore`, `RegistoDeBuild`, 5 estados, cancelamento por
  contexto, tectos de saída e de registos) e **5 rotas**: `GET /build/status/:id`,
  `GET /build/logs/:id`, `GET /build/artifacts/:id`, `GET /build/registos`,
  `POST /build/cancel/:id`. O `build.start` passou a **registar** e a devolver `build_id`, com modo
  `assincrono` (**202**, corre em segundo plano). As duas promessas travadas: `cancel` sobre build
  terminado → **409**; e um build cancelado **não** é des-cancelado quando o processo morre depois.
  *Um cancelamento que o resultado desfaz não é um cancelamento: é uma sugestão.*
  **O modo assíncrono não é extra:** se o `start` só responde no fim, o `cancel` só encontra builds
  terminados e o 409 recusa todos.
  **Medido:** `go build ./...` 0 · `./server` **ok 49 s** · `./internal/agent` **ok 378 s** · lint 0 ·
  17+10+2 testes PASS. Ciclo real: `start`→200 com `build_id`; `status`→200 `CONCLUIDO`;
  assíncrono→202 `A_EXECUTAR` e termina; `cancel` de terminado→409.
  **Duas ocorrências aqui:** `itoa` colidiu com o `itoa` de `omnichannel_dedupe_test.go`; e o
  **`misspell` recusou «discriminante»** — **quinta** vez nesta sessão.
- **E-1312** — CI verde (`d75b2ff16`: 21 success, só `gitleaks` vermelho). Instalação limpa
  `BLOCKED_EXTERNAL` com prova: `.iss` usa `{autopf}` → exige UAC; ficou pendurado 600 s sem log;
  **nada foi instalado** (sem directório, sem registo, zero processos).
- **E-1311** — `ResumoDeBlockers` mentia: `alvos_bloqueados=3` com `blockers=[]`. Segunda ocorrência
  do mesmo defeito. **Uma correção de ~20 linhas fechou as 7 falhas de CI.**
- **E-1310** — `ProviderDeInstaladorWindows`; `HadesSetup-teste.exe` 3 363 167 B, **`NotSigned`**.

## Cobertura do Release Hub

| superfície | estado |
|---|---|
| `toolchain.*` | **existe** |
| `project.detect`, `build.plan`, `build.start` | **existe** |
| **`build.status`/`logs`/`artifacts`/`cancel`** | **existe (E-1313)** |
| `build.verify`, `build.smoke`, `artifact.*`, `sbom.generate` | **existe** |
| `sign.plan`, `credentials.status`, `signing.*` | **existe** |
| `sign.start`/`sign.status`/`sign.verify` | **AUSENTE** |
| `publish.preflight` | **existe** |
| `publish.request_approval`/`submit`/`status`/`rollback` | **AUSENTE** (por desenho: o provider só tem `Preflight`) |
| `provenance.attest`, `policy.evaluate` | **AUSENTE** |

O que continua aberto: o store **não sobrevive a reinício**; nenhum artefacto assinado nem instalado;
`sign.start`/`publish.submit` inexistentes; **nenhum painel do UI consome as rotas novas**;
`gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1312

A cabeça do índice é **E-1312**.

- **E-1312** — **o CI do E-1311 está VERDE.** `d75b2ff16` fechou com **21 success e 2 failure**, e as
  duas são o `gitleaks (historico do PR)` — a dívida conhecida do proprietário. Os três gates que
  estavam vermelhos passaram: **`Go agentic and server gates`**, **`PR gate`** e
  **`Go agentic Windows portability`**. **A comparação mede a correção:** no commit anterior
  (`f86bcce8e`) eram **7 falhas**, incluindo `test (macos-latest)`, `test (ubuntu-latest)` e
  `race (ubuntu-latest)` — o `TestRotaPlanoBlockersEMotivosNoTopo` partia o `./server` nas **três
  plataformas**, e o `PR gate` reprovava por consequência. Uma correção de ~20 linhas no
  `ResumoDeBlockers` fechou sete.
- **A INSTALAÇÃO LIMPA ficou `BLOCKED_EXTERNAL`, com prova.** O `.iss` de teste usa
  `DefaultDirName={autopf}` → `C:\Program Files`, que exige elevação. O instalador com `/VERYSILENT`
  ficou **pendurado 600 s**, sem log e sem exit code: à espera de um UAC que não aparece em sessão
  não-interativa. **O que isso NÃO deixou:** nada — `C:\Program Files\HADESTeste` não existe,
  `%LOCALAPPDATA%\HADESTeste` não existe, registo `*HADES*` vazio, zero processos. A tentativa foi
  limpa. **Desbloqueio exato:** VM Windows descartável + `/VERYSILENT /LOG=...`, depois desinstalar e
  verificar resíduos — ação do proprietário, não do agente.
- **E-1311** — `ResumoDeBlockers` mentia: `alvos_bloqueados=3` com `blockers=[]`. Segunda ocorrência
  do mesmo defeito (o E-1307 corrigiu-o no provider e deixou-o no resumo). Corrigido com
  `MotivosDeNaoPronto()` + prefixo do alvo. `internal/agent` ok 344 s, `server` ok 44 s.
- **E-1310** — `ProviderDeInstaladorWindows`: `windows-installer` → 200, `HadesSetup-teste.exe`
  3 363 167 B, `'Inno Setup'` presente, **`Status = NotSigned`**.
- **E-1309** — o artefacto **corre** no alvo: `HADES-ARTEFACTO-OK so=windows arch=amd64`; ELF e
  Mach-O **não** produziram a marca.

## Estado por alvo

| alvo | build | formato | assinatura | instalação |
|---|---|---|---|---|
| `windows` | **200** | PE `.exe` | — | **EXECUTADO** |
| `linux` | **200** | ELF | — | **NÃO EXECUTADO** (sem host Linux) |
| `macos` | **200** | Mach-O | — | **NÃO EXECUTADO** (sem host macOS) |
| `windows-installer` | **200** | Inno Setup `.exe` | **NotSigned** (por desenho) | **`BLOCKED_EXTERNAL`** |

O que continua aberto: a instalação limpa (`BLOCKED_EXTERNAL`), nenhum artefacto assinado (por
desenho), `build.status`/`logs`/`cancel`/`artifacts` inexistentes, `gitleaks (historico do PR)`,
build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não
iniciada**.

---
## JANELA E-1311

A cabeça do índice é **E-1311**.

- **E-1311** — **o CI do E-1309 estava VERMELHO e o defeito era meu.** `0666cca86` fechou com
  `Go agentic and server gates = failure` e `PR gate = failure` (o `PR gate` falha *porque* o gate
  profundo falhou). Log: `--- FAIL: TestRotaPlanoBlockersEMotivosNoTopo` /
  `plano tem 3 alvo(s) bloqueado(s) mas 'blockers' esta' vazio`. **Causa:** o E-1308 tirou o
  `inno-setup` do alvo `windows`, os três alvos ficaram `PRESENT_COMPATIBLE_UNVERIFIED`, e
  `ResumoDeBlockers()` lia `.Blockers` (que só tem ausências) enquanto `AlvosBloqueados` conta
  `Pronto()==false`. Medido: `alvos_bloqueados=3`, `blockers=[]` — **o painel mostrava «3 bloqueados»
  e nenhum blocker**.
  **É a SEGUNDA ocorrência do mesmo defeito** — o E-1307 corrigiu-o no `ProviderDeBuildGo` e deixou-o
  vivo no resumo do projecto. *Uma correção num sítio não é uma correção no sistema.*
  Corrigido: `ResumoDeBlockers()` usa `MotivosDeNaoPronto()` e **prefixa o alvo** (o dedupe global
  colapsava o `go` de `windows` e `linux` numa linha, apagando a contagem).
  **`go test ./internal/agent` ok (344 s)**, **`go test ./server` ok (44 s)**, lint 0.
  **O processo falhou, não só o código:** depois do E-1308 corri `-run TestInicioDeBuild` em vez da
  suíte completa do `server`. *Um `-run` que escolhe o teste que eu escrevi é um teste que confirma o
  que eu já sabia.*
- **E-1310** — `ProviderDeInstaladorWindows`: `windows-installer` → **200**,
  `HadesSetup-teste.exe` 3 363 167 B, `'Inno Setup'` no binário, **`Status = NotSigned`**.
- **E-1309** — o artefacto **corre** no alvo: `HADES-ARTEFACTO-OK so=windows arch=amd64`; ELF e
  Mach-O **não** produziram a marca.
- **E-1308** — o alvo `windows` prometia um instalador que nenhum provider produzia.
- **E-1307** — o `build.start` passou a construir. Quatro defeitos.

O que continua aberto: nenhum instalador foi **instalado nem executado**; `build.status`/`logs`/
`cancel`/`artifacts` não existem; **nenhum artefacto assinado** (por desenho); `gitleaks (historico do
PR)`; build Android e arranque do app (`BLOCKED_EXTERNAL`); `INSTALLED=NAO` (E-1252); **FASE 30 não
iniciada**.

---
## JANELA E-1310

A cabeça do índice é **E-1310**.

- **E-1310** — o quarto alvo deixou de recusar: existe o **`ProviderDeInstaladorWindows`**. O E-1308
  separou `windows` de `windows-installer` porque o provider de binários nunca invocaria o Inno Setup;
  o alvo ficou a responder **412**. Agora o `build.start` escolhe o provider **pelo alvo**.
  **Medido:** `windows-installer` → **200**, **`HadesSetup-teste.exe` 3 363 167 B**, magic `4D 5A`,
  string `Inno Setup` dentro do binário, e `Get-AuthenticodeSignature` →
  **`Status = NotSigned`**, `SignerCert = (nenhum)` — **não assinado, que é o requisito da RC**.
  **8 testes PASS**, lint 0. O provider **não adivinha o nome** do artefacto (lista o destino antes e
  depois e mede o que apareceu), **não apaga o `dist`**, e **recusa** com dois `.iss` em vez de
  escolher por nós.
- **E-1309** — o artefacto **corre** no alvo: binário Windows executado devolveu
  `HADES-ARTEFACTO-OK so=windows arch=amd64`; ELF e Mach-O **não** produziram a marca (negativo
  discriminante). O discriminante é a **marca na saída**, não o exit code.
- **E-1308** — o alvo `windows` prometia um instalador que nenhum provider produzia.
- **E-1307** — o `build.start` passou a construir. Quatro defeitos: mensagem `0 blocker(s)`, espera
  circular, `GOOS` do host, catálogo sem `macos`.
- **E-1306** — `POST /build/start`, a primeira rota do Release Hub que executa.

## Estado por alvo

| alvo | build | formato | execução no alvo |
|---|---|---|---|
| `windows` | **200** | PE `.exe` | **EXECUTADO** — `so=windows arch=amd64` |
| `linux` | **200** | ELF | **NÃO EXECUTADO** (sem host Linux) |
| `macos` | **200** | Mach-O | **NÃO EXECUTADO** (sem host macOS) |
| `windows-installer` | **200** | Inno Setup `.exe` | **NÃO INSTALADO** — não assinado, como a RC exige |

O que continua aberto: **nenhum instalador foi instalado nem executado**; o `.iss` usado é de
**teste** (`HADES Teste`), não o `app/ollama.iss` canónico; `build.status`/`logs`/`cancel`/`artifacts`
não existem; **nenhum artefacto assinado** (por desenho); `gitleaks (historico do PR)`; build Android e
arranque do app (`BLOCKED_EXTERNAL`); `INSTALLED=NAO` (E-1252); **FASE 30 não iniciada**.

---
## JANELA E-1309

A cabeça do índice é **E-1309**.

- **E-1309** — o elo que os três builds verdes ainda escondiam: **o artefacto corre no sistema alvo?**
  Provado para o único alvo que este host pode executar. O binário **Windows** produzido pelo
  `build.start` foi executado e respondeu **`HADES-ARTEFACTO-OK so=windows arch=amd64`** (exit 0) —
  identifica-se como Windows/amd64 **ao correr**, o que é mais forte do que o `MZ` no cabeçalho.
  **Negativo discriminante:** os binários `linux` (ELF) e `macos` (Mach-O) foram tentados no mesmo
  host e **não produziram a marca**. O discriminante é a **marca na saída**, não o exit code — o
  `exit=0` das tentativas falhadas é do PowerShell, e contá-lo teria dado **três verdes falsos**.
- **E-1308** — o alvo `windows` prometia um instalador que nenhum provider produz. Separado em
  `windows` (binário) e `windows-installer` (recusa honesta).
- **E-1307** — o `build.start` passou a construir. Quatro defeitos: mensagem `0 blocker(s)`, espera
  circular, `GOOS` do host, catálogo sem `macos`.
- **E-1306** — `POST /build/start`, a primeira rota do Release Hub que executa. 8 testes PASS.
- **E-1305** — o `misspell` leu *Resolvemos* como erro de inglês. **Verde confirmado em `aea7076be`.**

## Estado por alvo — a distinção que o adendo exige

| alvo | build | formato | execução no alvo |
|---|---|---|---|
| `windows` | **200** | PE `.exe` | **EXECUTADO** — `so=windows arch=amd64` |
| `linux` | **200** | ELF | **NÃO EXECUTADO** (sem host Linux) |
| `macos` | **200** | Mach-O | **NÃO EXECUTADO** (sem host macOS) |
| `windows-installer` | **412** | — | sem provider |

O que continua aberto: o instalador Windows não existe, `build.status`/`logs`/`cancel`/`artifacts` não
existem, **nenhum artefacto foi assinado**, `gitleaks (historico do PR)` (decisão do proprietário),
build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não
iniciada**.

---
## JANELA E-1308

A cabeça do índice é **E-1308**.

- **E-1308** — o alvo `windows` prometia um **instalador** que nenhum provider produz. O catálogo dizia
  *«Instalador .exe não assinado (Inno Setup)»* e exigia `inno-setup` + `powershell`; o único provider
  corre `go build`. **Medido** com `ISCC.exe` (1 456 272 B) e PowerShell no PATH: **200** e
  `build-alvo-windows.exe` (2 458 624 B, `MZ`) — e **instaladores: 0**. Separado em `windows` (binário,
  só `go`) e `windows-installer` (instalador declarado, `SEM PROVIDER: o build.start recusa`).
  **Os três alvos de binário constroem com a plataforma certa:** `linux`→ELF, `macos`→Mach-O,
  `windows`→PE. `go test ./internal/agent` **ok (361 s)**, lint 0.
- **E-1307** — o `build.start` passou a **construir**. Quatro defeitos: mensagem dizia `0 blocker(s)`;
  **espera circular** (`Pronto()` exigia o health que o próprio `go build` produz); `GOOS` vinha do
  host (artefacto `-linux` com bytes `MZ`); catálogo **sem o alvo `macos`** que o provider prometia.
- **E-1306** — `POST /build/start`: a primeira rota do Release Hub que executa. 8 testes PASS.
- **E-1305** — o `misspell` leu *Resolvemos* como erro de inglês. **Verde em `aea7076be`**:
  `test (windows-latest)` success, `test (ubuntu-latest)` success.
- **E-1304** — o E2E do `build.plan` nunca tinha corrido contra o backend real (faltava `HADES_E2E_REPO`).

O que continua aberto: **o instalador Windows não existe** (`windows-installer` sem provider e o
`ISCC.exe` fora do PATH do servidor), `build.status`/`logs`/`cancel`/`artifacts` não existem,
**nenhum artefacto foi assinado, instalado ou executado no sistema alvo**, `gitleaks (historico do PR)`
(decisão do proprietário), build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO`
(E-1252) e a **FASE 30 não iniciada**.

---
## JANELA E-1307

A cabeça do índice é **E-1307**.

- **E-1307** — o `build.start` do E-1306 passou a **construir de verdade**. Quatro defeitos encontrados
  ao executá-lo contra o backend real: (1) a mensagem dizia `0 blocker(s): ` quando o plano não estava
  pronto por uma ferramenta `PRESENT_COMPATIBLE_UNVERIFIED`; (2) **espera circular** — o `Construir`
  exigia `plano.Pronto()`, cujo health funcional para o `go` **é o próprio `go build`**; (3) `GOOS`
  vinha do host, não do alvo: pedir `linux` produzia um ficheiro chamado `build-alvo-linux` com bytes
  `4D 5A 90 00` (Windows PE); (4) o catálogo **não tinha o alvo `macos`** que o provider prometia.
  Medido: `linux` → **ELF** (`7F 45 4C 46`), `macos` → **Mach-O** (`CF FA ED FE`), `windows` → 412
  honesto (`inno-setup` ABSENT). `golangci-lint` 0 issues, testes **ok**.
- **E-1306** — `POST /build/start`: a primeira rota do Release Hub que **executa**. O `ProviderDeBuildGo`
  existia desde o E-1261 e nenhuma rota o expunha. 8 testes PASS.
- **E-1305** — o E-1303 corrigiu o macOS e partiu windows/ubuntu: `misspell` leu *Resolvemos* como erro.
  **Verde confirmado em `aea7076be`:** `test (windows-latest)` success, `test (ubuntu-latest)` success.
- **E-1304** — o E2E do `build.plan` nunca tinha corrido contra o backend real (faltava `HADES_E2E_REPO`).
- **E-1303** — `/var`→`/private/var`: `filepath.Clean` não resolve symlinks.

O que continua aberto: **o alvo `windows` não constrói** (`inno-setup` ABSENT — o alvo produz
instalador), `build.status`/`logs`/`cancel`/`artifacts` não existem, **nenhum artefacto foi assinado,
instalado ou executado no sistema alvo**, `gitleaks (historico do PR)` (decisão do proprietário),
build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não
iniciada**.

---
## JANELA E-1306

A cabeça do índice é **E-1306**.

- **E-1306** — `POST /build/start`: a primeira rota do Release Hub que **executa** alguma coisa. O
  `ProviderDeBuildGo` existia desde o E-1261 e **nenhuma rota o expunha**. As defesas já estavam no
  provider (plano pronto, alvo no catálogo, raiz absoluta com `go.mod`, timeout 10 min, saída truncada,
  `dry_run`). A rota acrescenta validação do corpo, plano vindo do `ResolverDeToolchain` e aviso de
  fronteira no corpo. **8 testes PASS**, `golangci-lint ./server` 0 issues. Contra o binário real:
  412 (raiz relativa/vazia), 400 (timeout acima do tecto / negativo / corpo inválido), 200 (`dry_run`).
  **O caminho de sucesso de uma build real NÃO foi executado.**
- **E-1305** — o E-1303 corrigiu o macOS e partiu windows/ubuntu: o `misspell` leu *Resolvemos* como
  erro de inglês. Corrigido; `golangci-lint` 0 issues nos três pacotes.
- **E-1304** — o E2E do `build.plan` nunca tinha corrido contra o backend real: sem `HADES_E2E_REPO`
  3 passed / 4 failed; com ela 7 passed. O spec passou a falhar alto sem a variável.
- **E-1303** — `/var`→`/private/var`: `filepath.Clean` não resolve symlinks.
- **E-1302** — `publish.preflight` contra o binário Go real: 9 passed no modo real, 9 no fixture.

O que continua aberto: **caminho de sucesso de `build.start` não provado**,
`build.status`/`logs`/`cancel`/`artifacts` não existem (falta store), `gitleaks (historico do PR)`
(decisão do proprietário), build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO`
(E-1252) e a **FASE 30 não iniciada**.

---
## JANELA E-1305

A cabeça do índice é **E-1305**.

- **E-1305** — o E-1303 corrigiu o macOS mas partiu o **Windows e o Ubuntu**: o `misspell` do
  `golangci-lint` leu a palavra portuguesa *Resolvemos* num comentário como erro de inglês. Medido:
  `f5fd1af7a` windows/ubuntu **success**; `63ea97085` **failure** em cada. Corrigido removendo a
  palavra; `golangci-lint run ./server/` **0 issues**, `./internal/agent/` e `./internal/multillm/`
  **0 issues**. **O gate local existia e não foi invocado.**
- **E-1304** — o E2E do `build.plan` nunca tinha corrido contra o binário Go real: sem
  `HADES_E2E_REPO` dava 3 passed / **4 failed**; com ela **7 passed**. O spec passou a falhar alto sem
  a variável.
- **E-1303** — `/var`→`/private/var` no macOS: `filepath.Clean` não resolve symlinks. Corrigido com
  `EvalSymlinks` nos dois lados.
- **E-1302** — E2E do `publish.preflight` contra o backend Go real: 9 passed no modo real, 9 no
  fixture, suíte 136/11.
- **E-1301/E-1300** — `Preserve Class A+ surfaces` verde; a causa era o checker de `.ps1` a exigir
  `powershell.exe` no Ubuntu.

O que continua aberto: `build.start`/`status`/`logs`/`cancel`/`artifacts` não existem,
`gitleaks (historico do PR)` (decisão do proprietário), build Android e arranque do app
(`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não iniciada**.

---
## JANELA E-1304

A cabeça do índice é **E-1304**.

- **E-1304** — o E2E do `build.plan` **tinha** modo `BACKEND_REAL_AGENT_RUNTIME` mas nunca foi
  executado contra o binário Go real. Executado: **3 passed / 4 failed / 2 skipped**. A causa não era o
  produto: faltava `HADES_E2E_REPO`, e o spec caía em silêncio para `/repo-fixture`. Com a variável:
  **7 passed / 2 skipped**. O spec passa a **falhar alto** sem ela.
- **E-1303** — `test`/`race` do macOS vermelhos por `filepath.Clean` não resolver symlinks
  (`/var`→`/private/var`); corrigido com `EvalSymlinks` nos dois lados.
- **E-1302** — E2E do `publish.preflight` contra o binário Go real: seis canais a 200, 400/412 nos
  negativos; **9 passed** no modo real e **9** no fixture.
- **E-1301/E-1300** — `Preserve Class A+ surfaces` **verde**; a causa era o checker de `.ps1` a exigir
  `powershell.exe` no Ubuntu.
- **E-1299/E-1298/E-1289** — cadeia `publish.preflight` completa.

O que continua aberto: **`build.start`/`status`/`logs`/`cancel`/`artifacts` não existem** (o adendo
pede-os), `gitleaks (historico do PR)` (decisão do proprietário), build Android e arranque do app
(`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não iniciada**.

---
## JANELA E-1303

A cabeça do índice é **E-1303**.

- **E-1303** — `test (macos-latest)` e `race (macos-latest)` ficaram **vermelhos**: o meu teste do
  E-1286 (`TestDeteccaoNaoUsaARaizDoPedido`) comparava o caminho com `filepath.Clean`, que **não
  resolve symlinks**. No macOS `/var` é symlink de `/private/var`, e o teste chamava diferentes a dois
  nomes do mesmo sítio. Corrigido com `filepath.EvalSymlinks` nos **dois** lados.
- **E-1302** — E2E do `publish.preflight` contra o **binário Go real**: seis canais a 200, canal
  inválido a 400, artefacto ausente a 412; Playwright **9 passed** em `BACKEND_REAL_AGENT_RUNTIME` e
  **9 passed** em `UI_FIXTURE`; suíte completa **136 passed / 11 skipped**.
- **E-1301** — `Preserve Class A+ surfaces` **verde** (era falha em cinco commits).
- **E-1300** — a causa era o checker de `.ps1` a exigir `powershell.exe` no Ubuntu; corrigido com
  `exit 3` declarado + `skipUnless` com motivo + três ramos explícitos, **sem** `|| true`.
- **E-1299/E-1298/E-1289** — cadeia `publish.preflight` completa (painel, cliente, rota).

O que continua aberto: `gitleaks (historico do PR)` (decisão do proprietário), `BuildProvider` sem
cliente/E2E, build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a
**FASE 30 não iniciada**.

---
## ACHADO P1 (ronda 107) -- RESOLVIDO na ronda 108 (E-1290): o build de release Android assinava com a DEBUG KEY
**RESOLUCAO (E-1290, ronda 108).** O bloco `buildTypes.release` deixou de usar `signingConfigs.debug`. A credencial vem de quatro variaveis de ambiente (`HADES_ANDROID_STORE_FILE`, `_STORE_PASSWORD`, `_KEY_ALIAS`, `_KEY_PASSWORD`); sem as quatro, `signingConfig` fica `null` e um guard em `assembleRelease`/`bundleRelease` lanca `GradleException` antes do empacotamento. A auditoria CNG/bare exigida acima foi feita: `android/` tem 35 ficheiros rastreados (bare), por isso a correcao persiste. A catraca `scripts/hades_android_signing_check.py` corre em CI via o seu `*_test.py`, e e' discriminante: reintroduzir o defeito produz um P0. **O build real continua `BLOCKED_EXTERNAL`** (sem JDK 17, Android SDK e Gradle nesta maquina).


`apps/mobile-agentic/android/app/build.gradle` tem, no template do React Native:

```
release {
    // Caution! In production, you need to generate your own keystore file.
    signingConfig signingConfigs.debug
}
```

**Porque e' um achado:** a debug keystore usa a senha publica `android` e o alias `androiddebugkey`,
conhecidos desde 2010. Um APK/AAB assinado com ela e' **forjavel por qualquer pessoa**, e o Google Play
**rejeita** AAB assinado com debug key.

**O que NAO e':** nao e' vazamento de segredo. `storePassword 'android'` e' a senha publica do debug
keystore; `debug.keystore` nao e' rastreada pelo git. A varredura do gate anti-vazamento confirmou:
0 ficheiros de credencial rastreados, 0 blocos PEM no repositorio.

**O caminho real de build Android e' o EAS**, nao `gradlew assembleRelease`: o `eas.json` declara
`credentialsSource` por perfil. Enquanto o EAS gerir a credencial, esta linha nunca corre.

**Risco residual:** o caminho LOCAL esta' aberto e silencioso. Quem correr `assembleRelease` obtem um
artefacto assinado com chave publica e nao descobre isso ate' o Play recusar -- ou pior, ate' alguem
forjar uma actualizacao.

**Porque nao foi corrigido agora:** o adendo 28.0 proibe `expo prebuild --clean` e a regeneracao de
`android/` antes de entender a estrategia CNG/bare e o WIP concorrente. Alterar o `signingConfig` sem
essa auditoria substituiria configuracao nativa gerada e seria perdido no proximo prebuild.

**Proximo passo:** fatia propria com auditoria CNG/bare, `signingConfigs.release` lendo credencial do
secret broker, e prova de que `assembleRelease` local falha fechado quando a credencial nao existe --
em vez de cair silenciosamente para a debug key.
## JANELA E-1289

**Ronda 107. O `PublishingProvider` ganhou porta — e o inventário foi corrigido.**

- **E-1289** — `server/agent_publish_preflight_route.go` + `_test.go`: `POST /publish/preflight/:canal`,
  **13 PASS** (8 testes + 5 subtestes).
- **CORREÇÃO DE INVENTÁRIO:** o resumo do projeto dizia que `CapabilityResolver` e
  `PublishingProvider` estavam **ausentes**. **Falso nos dois.** O `CapabilityResolver` **existe como
  `ResolverDeToolchain`** (`toolchain_plan.go`, 17 143 B) com `CapabilityPlan`, 6 `ToolState`, 10
  alvos, `ArgsDeVersao` e **3 rotas**; a UI **já o consome** (`toolchainClient.ts`, 20+ exportações,
  `CapabilitiesPage`, `InstallProfilesPanel`, `install-profiles.spec.ts` com 21 referências). E
  `PlanoDeInstalação` existe com 2 rotas.
  *Um inventário que procura nomes inventados declara ausente o que existe com outro nome* — 13.ª
  ocorrência do defeito, e a primeira em que o erro era **a favor** do projeto e mesmo assim foi
  corrigido.
- **A lacuna real:** o `PublishingProvider` tinha 6 canais e 9 testes e **ZERO rotas**. A rota é
  `publish.preflight`, **não** `publish.submit`: o provider não tem `Publicar()`.
- **Caminhos vêm do ambiente:** `HADES_ARTEFACTO_A_PUBLICAR` e `HADES_CREDENCIAL_DE_PUBLICACAO` —
  variáveis próprias, distintas das outras quatro fronteiras. O pedido só escolhe o CANAL.
- **Garantias medidas:** `publicado` **sempre** false e `nada_publicado` **sempre** true, em campos
  separados; **canal público NUNCA fica `pronto`** — os 5 testados; canal desconhecido recusado com
  **400 antes de tocar no filesystem**; **canário de vazamento** `CANARIO-QUE-NAO-PODE-VAZAR-12345`
  ausente do detalhe e do corpo serializado ("presente (N bytes); conteúdo NAO lido").

Medido: **13 PASS** · `go test ./server/` ok · `go build ./...` 0 · `go vet` 0 ·
`golangci-lint ./server` 0 · `gofumpt` limpo.

**Lacuna:** falta cliente + painel + E2E do preflight. `publish.submit` é deliberadamente inexistente.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1288

**Ronda 106. O E2E do `ProjectDetectPanel`: o nono elo fechado no browser real.**

- **E-1288** — `e2e/project-detect-panel.spec.ts` + rota `project/detect` no fixture:
  **10 passed / 1 skipped**.
- **O teste que só o browser faz aqui:** o fixture devolve as stacks `go`/`node` e os gestores
  `go-modules`/`npm` — os valores REAIS do backend — e o spec exige os rótulos traduzidos **"Go",
  "Node", "Go modules", "npm"**. Se o `switch` tivesse inventado `golang`/`nodejs`, o teste unitário
  passaria (usaria o mesmo valor inventado) e **o E2E falharia**.
- **Não tem campo de raiz, provado no browser:** `input` `toHaveCount(0)` e
  `Object.keys(postDataJSON()) === []`, com ausência de `raiz`/`caminho`/`path`/`directorio`/`root`.
- **Não existe botão de instalar, construir, assinar, publicar nem build:** um único botão, cinco
  rótulos proibidos. **A prova não promete toolchain:** exige "NÃO prova que a toolchain esteja
  instalada" e `toolchain.doctor`.
- **No modo de backend real a asserção é acionável:** sem `HADES_RAIZ_PROJECTO` o servidor recusa com
  412 e o spec exige o erro **nomeando a variável**, sem relatório. Não fabrica resultado.

Medido: **10 passed / 1 skipped** · `npx playwright test` **127 passed / 11 skipped** (era 117/10) ·
`npx vitest run` 128 ficheiros / 921 testes · `tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 ·
`BUNDLE_BUDGET=PASS`.

**Lacuna:** falta E2E com raiz REAL. O detector **não alimenta** o `BuildProvider`.
**2 providers sem rota nem painel** (`CapabilityResolver`, `PublishingProvider`).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1287

**Ronda 105. O oitavo provider fechado nos três elos: `ProjectDetectPanel` consome `project.detect`.**

- **E-1287** — `projectDetectClient.ts` + `ProjectDetectPanel.tsx` + **12 testes**, montado em
  `CapabilitiesPage`. A E-1286 deu rota ao `ProjectDetector`, mas **nenhum consumidor na UI**: a rota
  era inalcançável a partir do produto.
- **Os valores foram lidos antes de escrito, 12.ª vez:** stacks `go`/`node`/`expo`/`python`/`rust`/
  `dotnet`/`flutter` e gestores `go-modules`/`npm`/`pnpm`/`yarn`/`bun`/`poetry`/`uv`/`cargo`/`nuget`/
  `pub` — **todos extraídos de `project_detector.go`**, nenhum inventado.
- **Não tem campo de raiz:** `input` `toHaveLength(0)` e corpo do pedido `{}`. Um único botão, com
  instalar/construir/assinar/publicar/build proibidos.
- **A prova não promete toolchain:** o teste exige "NÃO prova que a toolchain esteja instalada" e
  `toolchain.doctor`. E quando há alvos mas **nenhum gestor**, o painel diz "NENHUM gestor de pacotes
  reconhecido" em vez de mostrar uma lista que parece boa.
- **Os avisos do detector são mostrados:** `avisos` tem `omitempty`; o painel mostra-os quando existem
  e não inventa um estado quando não.

Medido: **12 testes PASS** · `npx vitest run` **128 ficheiros / 921 testes** (era 127/909) ·
`tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS` ·
`npx playwright test` **117 passed / 10 skipped**.

**Lacuna:** sem E2E do painel. O detector **não alimenta** o `BuildProvider`.
**2 providers sem rota nem painel** (`CapabilityResolver`, `PublishingProvider`).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1286

**Ronda 104. O `ProjectDetector` ganhou porta: `POST /project/detect`.**

- **E-1286** — `server/agent_project_detect_route.go` + **8 testes**. O detector existe desde a E-1245
  com 12 KB e teste próprio, mas **nenhuma rota**: só o programador o podia invocar.
- **A fronteira, e o teste que a prova:** a raiz vem de `HADES_RAIZ_PROJECTO`, **nunca do pedido**.
  `TestDeteccaoNaoUsaARaizDoPedido` manda no corpo `/caminho/inventado/pelo/cliente` e exige **200 com
  a raiz do ambiente** — falha se a rota tiver obedecido.
- **Uma suposição minha desmentida pelo teste:** escrevi o teste a esperar **200 com listas vazias**; o
  detector devolve **409** com *"nenhuma stack reconhecida"*. **O contrato real é melhor que a minha
  suposição:** *uma lista vazia e um erro não são a mesma coisa: quem lê um relatório vazio acredita
  que o projeto não tem stack; quem lê "nenhuma stack reconhecida" sabe que o detector não soube
  classificar.*
- **Segundo sintoma mal lido:** o teste seguinte exigia o caminho completo no erro; o caminho **estava**
  lá, mas com barras escapadas (`\\`). `filepath.Base` resolveu. *O sintoma parecia falta de informação
  e era o escape do JSON.*
- **O aviso declara que os alvos são SUGESTÃO:** o teste exige `SUGESTAO` e `toolchain.doctor`. *Um
  alvo derivado de um ficheiro não prova que a toolchain exista.*

Medido: **8 testes PASS** · `go build ./...` 0 · `go vet` 0 · `golangci-lint` **0** · `gofumpt` limpo ·
`npx vitest run` 127 ficheiros / 909 testes · `tsc -b` 0.

**Lacuna:** sem cliente, sem painel e sem E2E — o `ProjectDetector` continua a não ser consumido pelo
produto. **2 providers sem rota nem painel** (`CapabilityResolver`, `PublishingProvider`).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1285

**Ronda 103. O E2E do `VerificationPanel`: o oitavo elo fechado no browser real.**

- **E-1285** — `e2e/verification-panel.spec.ts` + rota `build/verify` no fixture: **11 passed / 1 skipped**.
- **O teste que só o browser consegue fazer:** o fixture devolve `sem_assinatura: 1` e
  `nao_verificadas: 1` — **o mesmo valor nos dois**. Se o painel os somasse, um campo leria **2**. O E2E
  lê cada `data-testid` e exige **1 e 1**. Um teste unitário que constrói o próprio objecto não
  apanharia isto; **o browser apanha porque lê o DOM que o painel realmente pintou**.
- **Não existe botão de assinar, construir, instalar ou publicar:** um único botão, quatro rótulos
  proibidos, `input` `toHaveCount(0)`, `Object.keys(corpo) === ["artefactos"]`, `sha256` casado com
  `/^[a-f0-9]{64}$/`.
- **O resumo não pode dizer "assinado":** o E2E exige `not.toContainText(/assinad/i)` e a frase "NAO
  foi/foram inspeccionado".
- **No modo de backend real a asserção é disjunta** (`erro` OU `relatorio`) — os caminhos de
  demonstração não existem no router de produção, e o teste **não fabrica** um resultado para fingir
  que passou.

Medido: **11 passed / 1 skipped** · `npx playwright test` **117 passed / 10 skipped** (era 106/9) ·
`npx vitest run` 127 ficheiros / 909 testes · `tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 ·
`BUNDLE_BUDGET=PASS`.

**Lacuna:** falta ligação ao `BuildProvider` para verificar artefactos REAIS. `BuildPlanPanel` sem E2E.
**3 providers sem rota nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1284

**Ronda 102. O sétimo provider fechado: `VerificationPanel` consome `build.verify`.**

- **E-1284** — `verificationClient.ts` + `VerificationPanel.tsx` + **12 testes**, montado em
  `CapabilitiesPage`. A rota existia desde E-1256 mas **nenhum consumidor na UI**.
- **A distinção que o painel não pode perder, e há teste que a trava:** `sem_assinatura` (o adaptador
  olhou, não havia) e `nao_verificadas` (ninguém olhou) em `data-testid` **separados**, proibido somar.
  Se fossem somados, o painel diria "N com assinatura presente" para artefactos que ninguém inspecionou.
- **Inventado-vs-actual, apanhado antes de testar, 11.ª vez:** escrevi `v.integridade`; o campo é
  `integros`. Corrigido lendo `build_provider.go`. E o `eslint` apanhou `SHA_A` declarada e nunca usada
  — **o gate apanhou o meu próprio teste**.
- **`aprovada: true` não é prova de assinatura:** o teste exige "nada foi assinado" e "não é prova de
  assinatura".
- **Não tem campo nem botão de assinar/construir/instalar:** `input` `toHaveLength(0)`, um botão,
  `Object.keys(corpo) === ["artefactos"]`. Caminhos de demonstração declarados como tal.

Medido: **12 testes PASS** · `npx vitest run` **127 ficheiros / 909 testes** (era 126/897) ·
`tsc -b` 0 · `eslint .` **0 erros** · `vite build` 0 · `BUNDLE_BUDGET=PASS` ·
`npx playwright test` **106 passed / 9 skipped**.

**Lacuna:** sem E2E do painel; caminhos são de demonstração (falta ligação ao `BuildProvider`).
**3 providers sem rota nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1283

**Ronda 101. O E2E do `SbomPanel`: o sétimo elo fechado no browser real.**

- **E-1283** — `e2e/sbom-panel.spec.ts` + a rota `sbom/generate` no fixture.
- **Os tipos são os do BACKEND, e o spec verifica:** o fixture emite `MODULO_GO` e `PACOTE_NPM`, e o
  spec exige que o painel mostre "modulo Go" e "pacote npm". **É o antídoto do defeito
  inventado-vs-actual**: se o painel inventasse `GO_MODULE`, o E2E falhava — não o teste unitário, que
  usava o mesmo valor inventado.
- **Não tem campo de raiz, provado no browser:** `input` com `toHaveCount(0)` e
  `Object.keys(postDataJSON()) === []`, com ausência de `raiz`/`caminho`/`path`/`directorio`/`root`.
- **Não existe botão de assinar nem de atestar:** um único botão; `assinado` e `provenance` leem
  `nao`; o aviso contém "SBOM MINIMO", "sempre false" e "nunca do pedido".

**ACHADO DE CI — `cancelled` NÃO É FALHA:** em `5718d7f25`, 17 success + **6 `cancelled`** + 2
`failure` (gitleaks). Os 6 `cancelled` são **concurrency cancel-in-progress** — o push seguinte
cancelou-os (`started_at` 22:52, commit seguinte no mesmo minuto). **Ler `cancelled` como falha
inflacionaria o vermelho de 2 para 8.**

Medido: **9 passed / 1 skipped** · `npx playwright test` **106 passed / 9 skipped** (era 97/8) ·
`npx vitest run` 126 ficheiros / 897 testes · `tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 ·
`BUNDLE_BUDGET=PASS`.

**Lacuna:** no modo de backend real só a recusa 412 corre. **4 providers sem rota nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1282

**Ronda 100. O sexto provider fechado nos três elos: `SbomPanel` consome `sbom.generate`.**

- **E-1282** — `sbomClient.ts` + `SbomPanel.tsx` + **11 testes**, montado em `CapabilitiesPage`.
  **Décimo sétimo reabrir do defeito, sétimo fecho.**
- **Defeito do "inventado vs actual", apanhado ANTES de correr:** escrevi
  `GO_MODULE | NPM_PACKAGE | DESCONHECIDO`; os valores reais são **`MODULO_GO` / `PACOTE_NPM` /
  `BINARIO`**. Corrigido lendo `compliance_provider.go` antes de testar. *Um teste que inventa o valor
  que o backend devolve nunca falha por causa do backend.* **Décimo reabrir deste defeito desde
  E-1245.**
- **Não tem campo de raiz, e há teste que o prova:** `input` com `toHaveLength(0)` e o corpo do
  pedido é `{}` — `Object.keys(corpo) === []`.
- **Não existe botão de assinar nem de atestar:** um único botão, com "assinar"/"atestar" proibidos.
  **`nao_produzido` é mostrado por extenso.**
- **Um inventário de zero componentes não é erro:** **zero** `role="alert"`, `total=0` e a frase
  "nenhuma fonte de dependência encontrada". **`assinado`/`provenance` verdadeiros GRITAM.**

Medido: **11 testes PASS** · `npx vitest run` **126 ficheiros / 897 testes** · `tsc -b` 0 ·
`eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright. O SBOM não cobre transitivas nem attestation. **4 providers sem rota
nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1281

**Ronda 99. O sexto provider ligado ao produto: `sbom.generate` consome o `ComplianceProvider`.**

- **E-1281** — `server/agent_sbom_route.go` + **6 testes**. **Décimo sexto reabrir do defeito, sexto
  fecho.**
- **A raiz vem do ambiente e é uma variável própria:** `HADES_RAIZ_SBOM`, não `HADES_ARTEFACTOS`.
  *`HADES_ARTEFACTOS` é a fronteira dos binários; `HADES_RAIZ_SBOM` é a fronteira do código. São cofres
  diferentes, logo variáveis diferentes — e nenhuma das duas é escolhida pelo cliente.* Há teste que
  envia um `raiz` no corpo a apontar para um diretório vazio e exige que o ambiente ganhe.
- **Recusas:** sem a variável **412**; raiz que não é directório **412**; falha do provider **409** —
  *um 500 aqui confundiria "o servidor falhou" com "o projecto não tem fontes"*.
- **`assinado` e `provenance` são SEMPRE false**, e o aviso declara "MINIMO", "NAO inclui", "sempre
  false" e "nunca do pedido". *Um SBOM que inclui o que o programador supõe é pior do que nenhum.*
- **Um directório vazio não é erro:** **200** com `total: 0` e `nao_produzido` **não vazio** — um SBOM
  vazio e válido seria uma mentira silenciosa.

Medido: **6 PASS** · `./server` ok 51,3 s · `go build ./...` 0 · `go vet` 0 · lint 0 issues.

**CI medido:** `6ee5ebb64` → **17 success · 2 failure (só os gitleaks históricos) · 3 skipped**.
Nenhum vermelho novo.

**Lacuna:** a rota **não tem painel**. O SBOM não cobre transitivas, notices, provenance nem
attestation. **4 providers sem rota nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1280

**Ronda 98. O E2E do `SigningPanel`: o sexto elo fechado no browser real.**

- **E-1280** — `e2e/signing-panel.spec.ts` + a rota `sign/plan` no fixture.
- **Sem campo de senha, provado no browser:** `input` com `toHaveCount(2)`,
  `input[type="password"]` com contagem **zero**, e o texto sem "senha" nem "chave privada:".
- **Sem botão de assinar, provado no browser:** `getByRole("button")` tem de ser **exactamente um**,
  com rótulo a conter "avaliar" e não "assinar".
- **O corpo do pedido é inspeccionado:** `postDataJSON()` exige `[identidade, tipo]` e, dentro da
  identidade, `[ambiente, caminho_do_certificado, rotulo]`; o JSON serializado não pode conter
  `senha`, `password`, `chave_privada`, `private_key`, `pfx` nem `p12`.
- **Um certificado sem EKU não pode ser tratado como apto:** `apta=nao`, frase do EKU, OID
  `1.3.6.1.5.5.7.3.3` e `prova=nao assinavel`.

Medido: **9 passed / 1 skipped** · `npx playwright test` **97 passed / 8 skipped** (era 88/7) ·
`npx vitest run` 125 ficheiros / 886 testes · `tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 ·
`BUNDLE_BUDGET=PASS`.

**CI no momento desta ronda:** 17 check-runs **pendentes, 0 vermelhos** em `6ee5ebb64`.

**Lacuna:** sem E2E com certificado REAL do kit. **5 providers sem painel.** Nenhuma assinatura real
aplicada.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1279

**Ronda 97. O quinto provider ligado ao produto: `SigningPanel` consome `sign/plan`.**

- **E-1279** — `signingPlanClient.ts` + `SigningPanel.tsx` + **11 testes**, montado em
  `CapabilitiesPage`. **Décimo quinto reabrir do defeito, quinto fecho.**
- **Defeito real, causado por mim, apanhado pelo `tsc`:** escrevi o cliente **por cima** de
  `signingClient.ts`, que **já existia** e serve o Signing Identity Center (FASE 28.5). Partiu
  `SigningIdentitiesPage.tsx` com **8 erros**. *Um nome que parece livre não prova que o ficheiro não
  existe: a pasta não foi lida.* Corrigido: original restaurado do `HEAD`, novo cliente em
  `signingPlanClient.ts`, tipo renomeado para `IdentidadeParaAvaliar`.
- **Sem campo para segredo, e há teste que o prova:** os `input` têm de ser **exactamente**
  `[assinatura-certificado, assinatura-rotulo]`, nenhum `type="password"`, e o corpo do pedido tem de
  ter `Object.keys(identidade) === [ambiente, caminho_do_certificado, rotulo]`.
- **Cada negação tem a sua frase, e o teste discrimina:** `SEM_EKU` mostra o OID; **`EXPIRADA` mostra
  "expirada" e o teste exige que a frase de EKU NÃO apareça** — *dizer "falta o EKU" a um certificado
  expirado manda quem lê corrigir a coisa errada.* Valor desconhecido **falha alto** no `default` do
  `switch` com `never`.

Medido: **11 testes PASS** · `npx vitest run` **125 ficheiros / 886 testes** · `tsc -b` 0 ·
`eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright. **5 dos 10 providers sem painel.** O provider **não valida cadeia
criptográfica nem revogação** — só o nome da CA emissora.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1278

**Ronda 96. O E2E do `RollbackPanel`: o quinto elo fechado no browser real.**

- **E-1278** — `e2e/rollback-panel.spec.ts` + a rota `publish/rollback` no fixture.
- **Duas classificações:** `UI_FIXTURE` (plano com candidato, ou **409** do Play sem service account)
  e `BACKEND_REAL_AGENT_RUNTIME` (atravessa o proxy; **`HADES_ARTEFACTOS` não existe em CI** ⇒ **412**,
  e o spec exige **erro accionável nomeando a variável**, nunca um plano).
- **Nenhum botão de restaurar, provado no browser:** `toHaveCount(1)` nos botões do painel e proibição
  dos rótulos `restaurar`, `aplicar`, `executar rollback`.
- **O corpo do pedido é inspeccionado no browser:** `postDataJSON()` exige `Object.keys === ["tipo"]`
  e a ausência de `directorio`.
- **O 409 do Play não é alerta genérico:** mostra *"service account JSON"* em `role="alert"` e
  **nenhum plano** aparece.

Medido: **7 passed / 1 skipped** · `npx playwright test` **88 passed / 7 skipped** (era 81/6) ·
`npx vitest run` 124 ficheiros / 875 testes · `tsc -b` 0 · `eslint .` 0 erros.

**A lição da E-1275 foi aplicada antes de correr:** o `readBody` devolve string, logo `JSON.parse`
desde o início — não custou uma falha.

**Lacuna:** no modo de backend real só a recusa 412 corre. **6 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1277

**Ronda 95. O quarto provider ligado ao produto: `RollbackPanel` consome `publish.rollback`.**

- **E-1277** — `rollbackClient.ts` + `RollbackPanel.tsx` + 8 testes, montado em `CapabilitiesPage`.
  **Décimo quarto reabrir do defeito, quarto fecho.**
- **Não existe botão de restaurar, e há teste que o prova:** conta os botões e exige **exactamente um**
  ("Planear rollback"), proibindo os rótulos "restaurar", "aplicar" e "executar rollback".
- **O pedido leva só o `tipo`, nunca um `directorio`:** o teste faz `JSON.parse` do corpo e exige
  `Object.keys(corpo) === ["tipo"]`.
- **O painel grita se o servidor declarar `executado: true`** — mostra *"ATENCAO: o servidor declarou
  que EXECUTOU o rollback"* em vez de o esconder. *Um estado que só o programador pode nomear inválido
  não é um estado.*
- **Canal não suportado não é alerta:** o 409 traz a explicação e o painel mostra-a como informação.

Medido: **8 testes PASS** · `npx vitest run` **124 ficheiros / 875 testes** · `tsc -b` 0 ·
`eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright. **6 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1276

**Ronda 94. O terceiro provider ligado ao produto: `publish.rollback` consome o `RollbackProvider`.**

- **E-1276** — `server/agent_rollback_route.go` + 6 testes. A E-1268 provou o provider com 7 testes,
  **mas só os testes o invocavam**. **Décimo terceiro reabrir do defeito, terceiro fecho.**
- **O directório vem do AMBIENTE, não do pedido:** lê `HADES_ARTEFACTOS` — a **mesma** variável que
  `build.smoke`. *Duas rotas que leem o mesmo cofre não podem ter duas fronteiras diferentes.* Há teste
  que envia um `directorio` no corpo e exige que seja **ignorado**.
- **`tipo` vazio é recusado (400):** *o canal de rollback não tem valor por omissão* — escolher um
  canal em nome de quem pede seria inventar a intenção.
- **Canal sem credencial devolve 409, não 500:** *um 500 aqui confundiria "o servidor falhou" com "o
  canal não existe"*.
- **`executado` é SEMPRE false** e o `aviso_de_fronteira` garante que nenhum ficheiro foi movido.

**Oito rotas do Release Hub:** `build/plan` · `build/artifact/checksum` · `build/artifact/verify` ·
`build/verify` · `build/smoke` · `publish/rollback` · `sign/plan` · `credentials/status`.

Medido: 6 PASS; `./server` ok 58,9 s; build 0; vet 0; lint 0.

**Lacuna:** a rota **não tem painel**. **7 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1275

**Ronda 93. O E2E do `SmokePanel`: o segundo elo fechado no browser real.**

- **E-1275** — `e2e/smoke-panel.spec.ts` + a rota `build/smoke` no fixture.
- **Duas classificações:** `UI_FIXTURE` (smoke aprovado com aviso) e `BACKEND_REAL_AGENT_RUNTIME`
  (atravessa o proxy; **`HADES_ARTEFACTOS` não existe em CI** ⇒ **412** — e o spec exige que o painel
  mostre isso como **erro accionável nomeando a variável**, nunca como resultado).
- **Quatro garantias nos dois modos:** diz que executa antes; botão **desactivado** sem caminho;
  **nenhum resultado sem pedido**; e **o pedido HTTP sai de verdade**.
- **No modo fixture:** saída `0.40.1`, duração `42 ms`, **aviso de sandbox** com *"NAO ha' sandbox"*,
  e o painel **não pode conter "binário seguro"**.

**Defeito real apanhado na execução:** a fixture devolvia 400 a dizer que faltava o `caminho`. A causa
não era o painel — era eu: **`readBody` devolve uma string** e eu testava `typeof corpo === "object"`.
**O sintoma parecia defeito do cliente e era do fixture.** Corrigido com `JSON.parse`.

Medido: **7 passed / 1 skipped**; `npx playwright test` **81 passed / 6 skipped**;
`npx vitest run` 123 ficheiros / 867 testes; `tsc -b` 0; `eslint .` 0 erros.

**Lacuna:** no modo de backend real só a recusa 412 corre — falta E2E com raiz REAL e binário REAL.
**8 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1274

**Ronda 92. O segundo provider ligado ao produto: `SmokePanel` consome `build.smoke`.**

- **E-1274** — `smokeClient.ts` + `SmokePanel.tsx` + 11 testes, montado em `CapabilitiesPage`. A
  E-1273 provou a rota com 6 testes, mas **nenhum painel a consumia**. **Décimo segundo reabrir do
  defeito, segundo fecho** — agora `CredentialProvider` e `TestProvider` têm rota + cliente + painel.
- **Três obrigações de um painel que executa código, todas com teste:**
  1. **dizer que executa antes** do botão;
  2. **mostrar o aviso de sandbox do backend por inteiro** — *um aviso que fica na documentação não
     chega a quem correu o binário*;
  3. **não chamar "funciona" ao que só arrancou** — o painel escreve *"provado que arranca e responde;
     NÃO provado que faz o que promete"*, e um teste proíbe a frase "binário seguro".
- **Não há campo para a raiz:** o painel não escolhe onde estão os binários. *Um cliente que escolhe a
  raiz dos binários é um cliente que decide o que corre.*
- **`separarArgumentos` não imita o shell:** respeita aspas mas **não expande `$VAR`** — teste com
  `$HOME/x` a exigir que fique literal.
- **Botão desactivado sem caminho**; **timeout mostrado como processo que não terminou**, nunca como
  aprovado.

Medido: **11 testes PASS** · `npx vitest run` **123 ficheiros / 867 testes** · `tsc -b` 0 ·
`eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright para o `SmokePanel`. **8 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1273

**Ronda 91. A primeira rota que EXECUTA um binário — e a fronteira que a torna segura.**

- **E-1273** — `server/agent_smoke_route.go` + 6 testes. `build/plan`, `artifact/*` e `sign/plan` são
  GET e só leem; esta é **POST e executa**. Sem fronteira seria um serviço de execução remota:
  *uma rota que executa o que lhe mandarem não é um TestProvider: é um shell com JSON*.
- **A fronteira é uma RAIZ, não uma lista de extensões:** o caminho tem de estar dentro de
  `HADES_ARTEFACTOS`. Sem a variável, a rota recusa com **412**.
- **Três verificações, nesta ordem:** normalizar; **resolver symlinks nos dois lados**; conter com
  `filepath.Rel`. A do meio é a que importa — *uma verificação de prefixo sem resolver symlinks é uma
  verificação que aprova o link, não o alvo*. Há teste com symlink a escapar a exigir 403.
- **Não há shell:** `exec.Command` com argumentos como lista. *Um argumento que passa por um shell é
  um argumento que o shell pode reescrever.*
- **Recusa sem argumentos**, recusa >32 argumentos (413) e **corta o timeout para o máximo** em vez de
  o aceitar por escrito. *Um timeout escolhido pelo cliente é uma pausa que o servidor aceita por
  escrito.*
- **A fronteira vai no resultado:** `sem_sandbox: true` + `aviso_de_sandbox`.

**Sete rotas do Release Hub:** `build/plan` · `build/artifact/checksum` · `build/artifact/verify` ·
`build/verify` · `build/smoke` · `sign/plan` · `credentials/status`.

Medido: 5 PASS / 1 SKIP (symlink no Windows); `./server` ok 54,2 s; build 0; lint 0.

**CI dos três commits anteriores: 0 red novos** — só os 2 gitleaks históricos.

**Lacuna:** a rota **não tem painel**.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1272

**Ronda 90. O E2E que faltava: o painel de credenciais no browser real.**

- **E-1272** — `e2e/credentials-panel.spec.ts` + a rota `credentials/status` no fixture. A E-1271
  provou o painel com transporte **mockado**; a E-1270 provou a rota com `httptest`; **os dois nunca
  falaram um com o outro**.
- **Duas classificações, e o spec diz qual corre:** `UI_FIXTURE` (kit com certificado e segredo) e
  `BACKEND_REAL_AGENT_RUNTIME` (atravessa o proxy até ao router de produção, onde o kit não está
  configurado em CI — e o spec exige a mensagem própria, **não um alerta**).
- **Cinco garantias nos dois modos:** painel com nome acessível; **o pedido HTTP sai de verdade**
  (`page.waitForRequest`); **"não configurado" não é erro**; aviso de fronteira no DOM; **nenhum botão
  existe** no painel.
- **No modo fixture, três asserções discriminantes:** certificado com sujeito e validade; segredo com
  **"NAO lido"**; e o painel **não pode conter "verificado com sucesso"**.

**Dois defeitos apanhados na execução:**
1. **`LOCALAPPDATA` redireccionado para `_tmp`** fez o Playwright procurar os browsers no sítio errado
   — ele guarda-os em `%LOCALAPPDATA%\ms-playwright`, e esse redireccionamento não serve para ele.
2. A fixture usava **`json(200, ...)`** mas o helper é **`sendJSON(response, status, payload)`** — o
   `ReferenceError` derrubava o servidor inteiro (daí `ECONNRESET` em todas as rotas).

Medido: **7 passed / 1 skipped** no spec; `npx playwright test` **74 passed / 5 skipped**;
`npx vitest run` **122 ficheiros / 856 testes**; `tsc -b` 0; `eslint .` 0 erros.

**Lacuna:** no modo de backend real só a garantia de "não configurado" corre — falta o E2E com o kit
REAL configurado no servidor.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1271

**Ronda 88. O primeiro painel que consome um dos dez providers: `CredentialsPanel`.**

- **E-1271** — `credentialsClient.ts` + `CredentialsPanel.tsx` + 7 testes, montado em `CapabilitiesPage`.
  A E-1270 provou a rota com 7 testes, mas **nenhum painel a consumia**: *um painel provado contra um
  mock e um backend provado contra `httptest` não provam que um fala com o outro*.
- **Décimo primeiro reabrir do defeito recorrente, primeiro fecho** — o elo rota→cliente→painel
  existe agora inteiro.
- **Não há campo para o valor de um segredo:** o tipo `CredencialPublica` não o tem, o backend não o
  envia e o painel não tem onde o guardar. *Um painel que tem um campo para o segredo acaba por o
  receber.* `eSegredo()` é uma lista **positiva** dos que não são segredo — um tipo novo cai em
  "é segredo".
- **Três estados que não se confundem, todos com teste:** não configurado (mensagem própria, não
  alerta), erro (`role="alert"`) e kit vazio.
- **"Não lido" é informação, não falha:** um teste exige que o painel **não** contenha "verificado com
  sucesso".
- **O aviso de fronteira vem do backend e é mostrado por inteiro** — o painel não o resume.

Medido: **7 testes PASS** · `npx vitest run` **122 ficheiros / 856 testes** · `tsc -b` 0 ·
`eslint .` 0 erros (31 warnings preexistentes) · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright — o elo está provado por teste de componente com `agentFetch` mockado.
**9 dos 10 providers continuam sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1270

**Ronda 87. O primeiro provider ligado ao produto: `credentials.status`.**

- **E-1270** — `server/agent_credentials_route.go` + 7 testes. A E-1265 media o kit real mas **só os
  testes a invocavam**: *uma comparação que só o programador consegue invocar não é uma
  funcionalidade: é uma função*.
- **O caminho vem do AMBIENTE (`HADES_KIT_CREDENCIAIS`), não do pedido.** Há teste que passa um
  `?raiz=` para outro directório e exige que seja ignorado. *Uma rota que aceita qualquer caminho é
  uma rota que lê qualquer caminho.*
- **Redacção:** devolve só caminhos relativos e **não devolve a raiz** — teste serializa a resposta e
  exige que a string da raiz não apareça nela. *A rota que devolve o mapa do cofre não é a rota que
  mostra o estado do cofre.*
- **Segredos continuam não lidos:** `conteudo_lido: false`, `fingerprint` vazio, frase "NAO lido" no
  `.p12` — tudo exigido por teste.
- **Não configurado não é erro:** responde **200** com `configurado: false` e diz qual variável
  definir. Um 404 faria um cliente concluir que o servidor está quebrado.

**Seis rotas do Release Hub:** `build/plan` · `build/artifact/checksum` · `build/artifact/verify` ·
`build/verify` · `sign/plan` · `credentials/status`.

Medido: 7 PASS.

**Lacuna:** a rota **não tem painel** no frontend. O defeito recorrente deu um passo, não fechou.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1268 e E-1269 -- OS DEZ PROVIDERS DO ADENDO 28.1 COMPLETOS

**Ronda 86. Fecham os dois últimos: `RollbackProvider` e `PublishingProvider`.**

### E-1268 -- `RollbackProvider`: o plano que NUNCA executa

- `Planear` monta o plano e verifica a integridade do alvo; **não restaura**. Restaurar um binário de
  produção é exactamente a acção que o adendo 28.5 põe atrás de *"confirmação humana no momento da
  acção"*. Um `Executar()` que só devolvesse "não autorizado" seria *um botão morto*.
- **`Executado` é SEMPRE false** e o aviso garante que *nenhum ficheiro foi movido, copiado ou
  apagado* — há teste que exige as duas frases.
- **Cada candidato tem SHA-256 medido em fluxo** e o teste recalcula por fora e compara.
- **Canal sem credencial é declarado, não fingido:** Play (`CONFIG_REQUIRED`), App Store (sem macOS),
  EAS Update (sem canal), OCI (sem registry) — 4 canais com `ErrRollbackNaoSuportado` e motivo.

### E-1269 -- `PublishingProvider`: e ele NÃO PUBLICA, por desenho

- A autorização proíbe publicar, por isso o provider implementa **`publish.preflight`** (o nome do
  adendo 28.7) e **não** `publish.submit`. A diferença é a diferença entre verificar e agir.
- **`Publicado` é SEMPRE false** nos 6 canais.
- **`LOCAL_ONLY` NÃO é deploy público:** a explicação diz que o artefacto *está no disco, NÃO que foi
  publicado* — e há teste que exige a frase.
- **Canal público nunca é declarado disponível:** a condição fica insatisfeita com o detalhe a nomear
  a proibição; `RequerAprovacao` é verdadeiro neles e falso no `LOCAL_ONLY`.
- **A credencial é verificada por existência, sem abrir:** há teste com **sentinela** de texto a
  provar que ela não vazou para o resultado nem para a explicação.

### Os dez providers

| # | Provider | Ficheiro |
|---|---|---|
| 1 | `ProjectDetector` | `project_detector.go` |
| 2 | `CapabilityResolver` | (via `PlanoParaProjeto`) |
| 3 | `BuildProvider` | `build_provider.go` |
| 4 | `SigningProvider` | `signing_provider.go` |
| 5 | `VerificationProvider` | `verification_provider.go` |
| 6 | `PublishingProvider` | `publishing_provider.go` |
| 7 | `CredentialProvider` | `credential_provider.go` |
| 8 | `ComplianceProvider` | `compliance_provider.go` |
| 9 | `TestProvider` | `test_provider.go` |
| 10 | `RollbackProvider` | `rollback_provider.go` |

**Nota honesta:** o item 2 (`CapabilityResolver`) tem apenas **1 ocorrência** em `internal/agent` —
é o nome no comentário do contrato, não uma abstracção própria. O plano de capability existe em
`PlanoParaProjeto`, mas **não há interface `CapabilityResolver`**. Declarado como lacuna.

Medido: `./internal/agent` **ok 340,1 s**; build 0; vet 0; lint 0.

**Lacuna transversal:** **nenhum dos dez tem rota HTTP própria nem painel** — só `sign.plan`,
`build/*` e `artifact/*` têm rotas. O defeito recorrente continua.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1267

**Ronda 85. O oitavo dos dez providers: `TestProvider` — o primeiro teste que EXECUTA o artefacto.**

- **E-1267** — `test_provider.go` + `process_tree.go` + 8 testes.
- A E-1261 prova integridade do ficheiro; **não prova que o binário corre**. Este provider fecha o
  degrau `build → verificar checksum → EXECUTAR e observar`.
  *Um ficheiro com bytes não é um artefacto utilizável.*
- **Três fronteiras no resultado:** `sem_sandbox: true` com aviso (sem Docker nesta máquina), não
  instala (E-1252), e **não aceita comando arbitrário** — só o próprio artefacto com os argumentos
  declarados. *Um evento sem allowlist não é um evento: é código com um nome diferente.*
- **Recusa executar sem argumentos** — *executar um binário sem argumentos é executá-lo no modo que o
  programador não escolheu*.
- **DEFEITO REAL encontrado pelo próprio teste:** `exec.CommandContext` matava o `cmd.exe` e deixava
  o `ping` filho vivo a segurar o pipe — **59,5 s para um timeout de 400 ms**. *Um timeout que mata o
  pai e deixa o filho não é um timeout: é uma promessa.* Corrigido com `cmd.Cancel` →
  `taskkill /T /F /PID` + `WaitDelay`. **Depois: 0,61 s.**
- **Alvos declarados com honestidade:** `AlvosTestaveis` = `BINARIO_DO_HOST`/`EXE_WINDOWS`;
  `AlvosNaoTestaveis` declara `EMULADOR_ANDROID`, `SIMULADOR_IOS`, `VM_EFEMERA`,
  `CONTENTOR_DOCKER`, `DISPOSITIVO_FISICO` — todos `BLOCKED_EXTERNAL` com motivo.

**Medido contra o artefacto REAL da E-1251:** `hades-windows-amd64.exe --version` →
`ollama version is 0.40.1`, **exit 0**, **2960 ms**.

Medido: 8 PASS; `./internal/agent` **ok 367,7 s**; build 0; vet 0; lint 0.

**8 dos 10 providers.** Faltam 2: `Publishing`, `Rollback`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1266

**Ronda 84. O sétimo dos dez providers: `ComplianceProvider` — o SBOM que recusa produzir o que não consegue.**

- **E-1266** — `compliance_provider.go` + 9 testes. Lê `go.mod` (bloco `require`, linha única,
  `// indirect`) e `package.json` (separando `dependencies` de `devDependencies`).
- **As três coisas que o adendo 28.9 pede e este provider NÃO faz, ditas no corpo:**
  `THIRD_PARTY_NOTICES` (exige o TEXTO da licença), árvore transitiva (exigiria subprocessos),
  `provenance` (exige builder e artefacto). *Um SBOM que inclui o que o programador supõe é pior do
  que nenhum.*
- **`Licenca` vazio NÃO é "sem licença":** significa **NÃO DECLARADA** — factos diferentes, e há
  teste que exige a frase no `aviso`.
- **`package.json` inválido não inventa componentes:** entra como `INVALIDO` em `FicheirosLidos`.
- **`node_modules` não é percorrido** — teste com dependência inventada a provar.
- **Assinatura estável:** *um SBOM que muda de ordem a cada leitura não permite dizer se mudou.*

**Medido contra o próprio repositório HADES:** **113 módulos Go**, 113 sem licença declarada,
**0 ficheiros npm** — e o limite foi encontrado: os `package.json` estão **dois níveis abaixo** e o
provider só olha a raiz e subdirectórios imediatos. **Declarado, não escondido.**

**Defeito meu apanhado:** `TipoDeComponente` somado com `string` sem conversão. E o `misspell` acusou
`directos` → "directors" (a grafia correcta é `diretos`).

Medido: 8 PASS; PASS contra o repo real; `./internal/agent` **ok 354,1 s**; build 0; vet 0; lint 0.

**7 dos 10 providers.** Faltam 3: `Publishing`, `Test`, `Rollback`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1265

**Ronda 83. O sexto dos dez providers: `CredentialProvider` — e a regra que o define é NÃO LER.**

- **E-1265** — `credential_provider.go` + 9 testes. Abre **certificados públicos**; faz **apenas
  `os.Stat`** em `.p12`, `.jks`, `.p8`, `.dpapi`, `.mobileprovision`.
  *Não leia o segredo para provar que ele existe: o tamanho e a data já provam que o ficheiro está
  lá, e a leitura é o risco.*
- **A prova de que não abre não é o comentário:** um teste tira a permissão de leitura a um `.jks`
  (`chmod 000`) e verifica que ele **continua** no inventário com o tamanho certo. Um `os.Open`
  teria falhado.
- **`DESCONHECIDA` também não é lida:** *na dúvida, não se abre.*
- **Três defeitos apanhados na própria ronda:**
  1. `Certificados` contava o TIPO e não os válidos — corrigido para exigir fingerprint.
  2. **`misspell` acusou `inventario` → "inventor"** — mesma armadilha da E-1255, resolvida com
     `ignore-rules`. *O `misspell` segue LIGADO.*
  3. **`nilerr` acusou dois `return nil` que engoliam erros** — e tinha razão: agora vão para
     `Avisos` + `NaoLegiveis`. *Um erro que não aborta não é um erro que não aconteceu.*

**Medido contra o kit real da DZ23 LTDA:** **29 ficheiros, 15 lidos, 14 NÃO lidos, 6 certificados,
0 expirados**. Certificados lidos de verdade: `hades-upload-certificate.pem` ("HADES", 2054), 3 DER
do Google Play (2056), 2 Apple Distribution ("leandro prado (K8G36CUP3S)", 2027-10-17). **Os 14
segredos não foram abertos.**

Medido: 8 PASS; PASS contra o kit real; `./internal/agent` **ok 352,1 s**; build 0; vet 0; lint 0.

**6 dos 10 providers** implementados. Faltam 4: `Publishing`, `Compliance`, `Test`, `Rollback`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1264

**Ronda 82. `sign.plan`: o `SigningProvider` deixou de ser função que só os testes invocam.**

- **E-1264** -- `POST /api/agent/v1/sign/plan` + 7 testes. **5 rotas do Release Hub** no router.
- **Não há `sign.start`** por decisão: a autorização proíbe assinar, e uma rota que só devolvesse 403
  seria *um botão morto — e um botão morto é pior do que um botão ausente: o ausente não mente*.
- **Dois defeitos reais apanhados pelos testes:**
  1. **`ErrCertificadoIlegivel` devolvia 500** (não mapeado) — um 500 diz "o servidor falhou" quando
     o que falhou foi o ficheiro indicado. Corrigido para **400**.
  2. O teste do certificado Apple **reescrevia o `Issuer` em memória** — mas o PEM vinha do DER
     original. *Um teste que reescreve o campo em memória e verifica o ficheiro no disco testa a
     memória.* Corrigido gerando a folha a partir de uma CA fictícia; o Apple passou a ser recusado
     com `EMISSOR_INESPERADO`.
- **O aviso de fronteira ensina a ler `apta`:** não significa que o artefacto foi assinado, e a
  cadeia criptográfica e a revogação **NÃO** são validadas — há teste que exige as duas frases.

Medido: 7 PASS na rota; **23 PASS** no conjunto; `./server` **ok 46,9 s**; build 0; vet 0; lint 0.

**Ausente:** nenhum painel nem E2E chama `sign.plan`. A1 da DZ23 não avaliado (store do Windows).
Faltam **5 dos 10** providers.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1263

**Ronda 81. O quinto dos dez providers: `SigningProvider` — e um certificado REAL expôs um defeito meu.**

- **E-1263** -- `signing_provider.go` + 12 testes. **Nao** e' um `Assinar()` que devolve "nao
  autorizado": a missao proibe assinar, e *um botao morto e' pior do que um botao ausente*.
- `AvaliarCertificado` verifica **validade → emissor → EKU**, com 8 valores de aptidao e o motivo.
- **O defeito que o certificado real expôs:** `IdentidadeDeOutraCA` estava declarado e **nunca era
  produzido** -- codigo morto a fingir de verificacao. Medindo o `distribution.cer` **real** do kit,
  descobri que ele **TEM** o EKU de Code Signing `1.3.6.1.5.5.7.3.3`; sem `emissorCompativel` o
  provider declararia **um certificado Apple apto a assinar um `.exe` do Windows**.
- **A frase do adendo provada contra o ficheiro real:** *"Apple Distribution nao substitui Developer
  ID"* -- `APPLE_DEVELOPER_ID` → **`SEM_EKU_DE_ASSINATURA`**; `AUTHENTICODE` →
  **`EMISSOR_INESPERADO`**.
- **Validade antes do EKU:** *dizer "falta o EKU" a um expirado manda corrigir a coisa errada.*
- **Formato sem regra nao e' aprovado:** Android/OpenPGP → `FORMATO_DESCONHECIDO`, sem afirmar nada.
- **Limite declarado no codigo:** verifica o NOME da CA, nao a cadeia nem a revogacao.

Medido: 12 PASS; `./internal/agent` **ok 499,1 s**; build 0; vet 0; lint 0.

**Ausente:** sem rota HTTP nem painel. A1 da DZ23 **nao avaliado** (vive no store do Windows).
Faltam **5 dos 10** providers: `Publishing`, `Credential`, `Compliance`, `Test`, `Rollback`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1262

**Ronda 80. `build.verify`: a verificacao deixou de ser funcao que so' os testes invocam.**

- **E-1262** -- `POST /api/agent/v1/build/verify` + 8 testes. 4 rotas `build/*` no router.
- **POST e nao GET:** a lista de artefactos e checksums nao cabe num URL -- *uma rota que se le' mas
  exige POST obriga quem a usa a saber mais do que precisa*.
- **Recusas:** lote vazio 400 · sem `sha256` 400 · sem `caminho` 400 · **>64 artefactos 413** ·
  ficheiro apagado **404 e nao 500**, sem veredito.
- **Nao inventa registo de builds:** nao procura por ID -- *um ID que resolve para um ficheiro que
  ninguem registou e' um apelido, nao uma identidade*.
- **O aviso de fronteira ENSINA A LER o resultado:** `aprovada` nao significa que exista assinatura;
  o teste exige essa frase.
- **Adulterado devolve 200** com `aprovada=false` -- *um 4xx confundiria "pedido invalido" com
  "artefacto adulterado"*.

Medido: 8 PASS; `./server` **ok 70,2 s**; build 0; vet 0; lint 0.

**Ausente:** nenhum painel nem E2E chama `build.verify` -- o defeito recorrente volta a abrir-se
nesta rota. Sem adaptador de assinatura.

Proximo passo: o painel a chamar `build.verify` e um E2E que o atravesse.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1261

**Ronda 79. A ligacao que faltava: construir -> VERIFICAR.**

- **E-1261** -- `VerificarConstrucao` + 5 testes. O SHA medido no build e' a REFERENCIA; o
  verificador mede o ficheiro AGORA.
- **Tres defeitos meus apanhados pelos proprios testes:**
  1. `errado` colidia com uma **funcao** do pacote;
  2. `artefacto` (com "c") e' **funcao** do pacote; a variavel era `artefato` -- o nome errado
     **resolveu para a funcao**. *Um identificador local que coincide com um do pacote nao
     sombreia.*
  3. **o importante:** a explicacao contava `NAO_VERIFICADA` como **"assinatura presente"**.
     *"Nao consegui olhar" e "olhei e nao ha'" sao factos diferentes.* Corrigido com `NaoVerificadas`
     separado; o teste agora **proibe a frase** quando ninguem inspeccionou.
- **Build vazio nao e' aprovacao:** `ErrSemArtefactosParaVerificar`.
- **Erro de infra nao vira veredito:** ficheiro apagado aborta.

Medido: 5 PASS; `./internal/agent` **ok 446,2 s**; build 0; vet 0; lint 0.
**Artefacto real da E-1251 verificado:** 69 206 016 B, `fcef9d9d...` aprovado; o MESMO ficheiro
reprovado com referencia errada.

**Nao medido / ausente:** `VerificarConstrucao` **nao tem chamador fora dos testes** -- o mesmo
defeito que esta fatia corrigiu para o `ConstruirAlvos`. Sem verificacao de assinatura real.

Proximo passo: rota `build.verify` a chamar `VerificarConstrucao`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1260

**Ronda 78. O E2E encontrou um P1 que nenhum teste de componente podia encontrar.**

- **E-1260** -- `e2e/artifact-verify.spec.ts` (9 testes) + fixture com negacoes + **correcao do
  `agentFetch`**.
- **O defeito:** `agenticClient.ts:677` lia `body?.error` (ingles); o backend responde
  `{"erro": ...}`. Todo o painel que mostre um erro de rota dizia **`Bad Request`** em vez da razao.
  Corrigido lendo os dois campos.
- **Porque ninguem apanhara antes:** o teste de componente mocka exactamente essa funcao --
  *um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
  com o outro*; e *um teste que substitui o cliente testa o mock, nao o produto*.
- **O fixture aprendeu a negar:** relativo -> 400, `inexistente` -> 404, referencia de 16 chars ->
  400. *Um fixture que so' serve o caminho feliz nao prova que o painel distingue os outros.*
- A regra de caminho absoluto foi repetida no fixture **de proposito** -- se o fixture fosse mais
  permissivo que o servidor, o spec passaria num comportamento que a producao recusa.
- O spec carrega o modo no nome do describe; o teste de ficheiro inexistente faz `skip` declarado.

Medido: 9 E2E PASS; **E2E completo 67 passed / 4 skipped / 0 failed** (eram 58); 121/849 vitest;
tsc 0; eslint 0.

**Nao medido:** o modo `BACKEND_REAL_AGENT_RUNTIME` nao foi executado nesta ronda.

Proximo passo: o `BuildProvider` a chamar o verificador no fluxo de build.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1259

**Ronda 77. As rotas da E-1258 deixaram de estar so' no servidor: o painel chama-as.**

- **E-1259** -- `artifactClient.ts` + seccao **dentro** do `BuildPlanPanel` (nao painel paralelo) +
  7 testes. *Tres pecas provadas em separado nao sao uma integracao provada.*
- **Dois nomes que nao se confundem:** `obterChecksum` mede, `verificarArtefacto` compara -- *uma
  funcao que so' sabe verificar obriga quem so' quer o hash a inventar uma referencia.*
- **O tipo obriga a cobrir os quatro estados:** `const inesperado: never = estado` -- *se o backend
  ganhar um estado novo, o TypeScript falha AQUI e nao no ecra.*
- **O aviso de fronteira vem do backend** e e' mostrado tal e qual.
- **"Verificar" desactivado sem referencia**, com guarda tambem no handler.
- **Dois defeitos meus:** `vi.importActual`+spread voltou a dar `__spreadValues` (mesma armadilha da
  E-1248 -- corrigido mockando so' o `agentFetch`); e **9 dos 10 testes antigos falharam** com
  `Expected 1 but found 3 instances with node type: "input"` -- a seleccao passou a ser pelo `id`,
  **mais precisa**.
- **O bug `$T`/`$t` do PowerShell apareceu pela TERCEIRA vez.**

Medido: 7 + 10 PASS; **121 ficheiros / 849 testes**; tsc 0; eslint 0; build 0; budget PASS.

Proximo passo: E2E Playwright da seccao; depois o `BuildProvider` a chamar o verificador.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1258

**Ronda 76. O verificador deixou de ser uma funcao Go: `artifact.checksum` e `artifact.verify`.**

- **E-1258** -- `server/agent_artifact_verify_route.go` + 8 testes PASS. Adendo 28.7.
- **Duas rotas porque sao duas operacoes:** `checksum` mede; `verify` compara com referencia.
  *Junta-las obrigaria quem so' quer o hash a inventar uma referencia -- e uma referencia inventada
  "passa" sempre.*
- Recusas com significado: sem `caminho` 400; caminho **relativo** 400; inexistente 404; sem
  `sha256` 400; referencia **curta** 400.
- **Adulterado devolve 200 com `adulterado=true`, NAO 4xx** -- *um 4xx confundiria "pedido
  invalido" com "artefacto adulterado"*.
- A fronteira vai no CORPO (`somente_leitura`, `aviso_de_fronteira`), nao so' na documentacao.
- **Nao procuram por ID:** um registo de artefactos nao existe. *Um ID que resolve para um ficheiro
  que ninguem registou e' um apelido, nao uma identidade.*
- **Bug meu:** `$R` (caminho) e `$r` (conteudo) sao a MESMA variavel em PowerShell. O ficheiro ficou
  intacto (verificado) e as rotas foram registadas pela ferramenta de edicao.
  => *duas variaveis que diferem so' na caixa sao a mesma variavel.*

**CI do `993804798` (correcao do `darwin`): 0 red** no momento da medicao -- terceira causa fechada.

Medido: 8 testes PASS; `go test ./server` ok 55,4 s; lint 0; `go build ./...` 0.

Proximo passo: um painel ou o fluxo de build a chamar estas rotas.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1257

**Ronda 75. O terceiro red era o mesmo `darwin`!=`macos`, no sitio que a correcao anterior nao tocou.**

- **E-1257** -- `--- FAIL: TestBuildProviderConstruirAlvosSaltaBloqueados`, `artefactos = 0; esperado
  1`. O teste tinha `{Alvo: runtime.GOOS, Plano: plannerPronto(alvoDoHost())}`: a E-1253 substituiu
  o segundo mas **deixou o campo `Alvo`**. No macOS `darwin` != `macos` -> alvo **pulado**.
- => *Uma correccao aplicada por substituicao de texto corrige as ocorrencias que a STRING descreve,
  nao as que o DEFEITO tem.* **Quarto avatar** do mesmo defeito -- e **dentro do proprio teste que eu
  escrevi para o corrigir**.
- **Corrigido** (`Alvo: alvoDoHost()`) **e guardado**: duas assercoes **independentes de
  plataforma** -- `cobreAlvo(alvoDoHost())` true e `alvoDoHost()` != `darwin`. Sem elas o teste so'
  falhava no macOS: *um teste cuja falha depende do sistema onde corre e' um teste que so' se
  descobre na CI mais lenta.*
- A mensagem de falha passou a incluir `NaoConstruidos`, para nao ser preciso ir ao log.

Medido: 13 testes PASS; vet 0; lint **0 issues**; gofmt limpo.

Proximo passo: confirmar o commit verde; depois INSTALLED (E-1252).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1256

**Ronda 74. O item 5 de dez: `VerificationProvider` -- e o teste que me obrigou a decidir.**

- **E-1256** -- `internal/agent/verification_provider.go` + teste, **14 testes PASS**. O
  `BuildProvider` calculava o hash e devolvia-o: o unico verificador era quem produzia.
  => *um artefacto cuja unica verificacao e' o hash calculado por quem o produziu nao e'
  verificado: e' assinado por si mesmo.*
- Recusa verificar **sem referencia** (`ErrSemReferencia`) e **recusa um PREFIXO** de checksum
  (16 caracteres do checksum real sao recusados): *uma referencia curta nao e' uma referencia: e'
  uma esperanca com hexadecimal*.
- **O teste que importa:** adultera-se UM byte -> `Adulterado=true`, `Aprovado=false`.
- Assinatura com **quatro** estados, incluindo `NAO_VERIFICADA`: *"nao consegui olhar" e "olhei e
  nao ha'" sao factos diferentes, e um campo que os junta mente sobre um dos dois.*
- **O teste obrigou-me a decidir:** eu esperava `Aprovado=false` quando o adaptador falha; o teste
  estava errado, nao o codigo. Reprovar ali tornaria o verificador inutil fora do Windows.
  `Aprovado` passou a significar "passou os gates que consegui aplicar".
- **Validacao cruzada:** corrido contra o artefacto real da E-1251, mediu
  `fcef9d9d4303a75b0b9c658f155c9ae6a7ccb84912b4face2268dc51f8161df6` -- o mesmo que a E-1251
  declarou.

Providers fechados: **5 de 10**. Proximo passo: rota HTTP para o verificador; depois
`SigningProvider`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1255

**Ronda 73. O segundo red do CI era `misspell` -- e a palavra era portugues correcto.**

- **E-1253 VALIDADA:** com a correccao `darwin`->`macos`, `test (macos-latest)` passou a **success**
  no GitHub. Quatro dos cinco jobs de `go test` ficaram verdes.
- **O que restava:** `test (windows-latest)` RED, causa dada pelas anotacoes do check-run --
  `server/agent_e2e_harness_test.go:175/179`: **`projecto` -> `projection` (misspell)**.
- **Reproduzido antes de corrigir:** `golangci-lint run ./server` **exit 1**, 3 ocorrencias.
  *O texto que explica porque o fixture tem de ser um projecto verdadeiro foi o texto que o linter
  recusou.*
- **Correccao:** `projecto`/`projectos` no `ignore-rules`, pelo criterio que o proprio ficheiro ja'
  documenta. **O linter segue ligado.**
- **Medido:** `./server` exit 0 (era 1); `./internal/agent` exit 0 (config valida, nao exit 3);
  `./...` **0 misspell**; os 8 `unused` restantes sao **preexistentes**.
- **Dado novo:** `internal/agent` tem 35 `projecto` e nunca foi acusado, o `server` tinha 3.
  => *o dicionario casa contexto, nao so' a palavra; assumir "esta' ou nao esta' na lista" erra
  por omissao nos dois sentidos.*

Proximo passo declarado: confirmar o commit verde no GitHub; depois INSTALLED (E-1252).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1254

**Ronda 72. A proveniencia mentirosa do instalador foi corrigida -- e o gate falha de proposito.**

- **E-1254** -- `scripts\hades_installer_windows.ps1` passou a **medir cada alvo** (SHA-256, caminho,
  data) e a comparar a data com a do commit HEAD. Medido no `5cef8330f`: **3 de 3 alvos anteriores
  ao HEAD** -> declara `COMPOSICAO` e grava o aviso no `PROVENIENCIA.txt`.
- **TESTE NEGATIVO:** `-ExigirAlvosDoHead` sai com **exit 4** e **nao gera um unico `.exe`** -- o
  gate foi verificado pela **ausencia do artefacto**.
- **A HEURISTICA E'' DECLARADA:** *um timestamp nao e'' uma proveniencia: e'' um indicio de
  proveniencia.*
- **INSTALACAO:** quarto sintoma -- `& HadesSetup.exe /VERYSILENT /LOG=` devolve **exit 0 em 0 s**
  sem criar ficheiro nem log. O binario foi verificado como Inno Setup legitimo (`MZ`+`PE`, string
  `Inno Setup`, `VersionInfo` produto `Hades`). **4 tentativas, sintomas diferentes, parar.**

Proximo passo declarado: ligar `-ExigirAlvosDoHead` ao CI e reconstruir o app desktop no HEAD; a
instalacao exige diagnostico proprio.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1253

**Ronda 71. O CI ficou vermelho por minha causa -- e a causa era um nome.**

- **E-1253** -- `d547d7986` passou de 2 para **6 red**: os quatro jobs que correm `go test`
  (`windows`, `ubuntu`, `macos`, `race macos`). **Causa:** os testes do `BuildProvider` passavam
  `runtime.GOOS` como alvo; no macOS isso e' `darwin`, o catalogo diz `macos`.
  => *um teste que passa na maquina onde foi escrito e falha nas outras tres nao testa o produto:
  testa o sistema de quem o escreveu* -- **a mesma familia do defeito de caminhos** que ja' apareceu
  em tres avatares, agora num nome de sistema.
- **Corrigido em dois eixos:** `alvoDoHostPara(goos)` com tabela que cobre os tres sistemas; e
  `ambienteDeBuild()` que sobrepoe `CGO_ENABLED=0`/`GOOS`/`GOARCH` em vez de herdar o `GOOS=windows`
  que o CI exporta.

Proximo passo declarado: confirmar os 4 jobs verdes no GitHub; depois atacar a instalacao (E-1252).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1252

**Ronda 70. O instalador Windows existe -- e a proveniencia dele mente.**

- **E-1252** -- `HadesSetup-d547d7986.exe`: **39,21 MB**, SHA-256
  `E8A733B8A45682A8E3B27D1F7D1BBEBC49CB3D65911C99805E62D99EF67CC50D`, **`NotSigned`** (o requisito
  da RC), produto **`Hades`**, versao `0.2.0.9012`. Compilado por ISCC 6.7.3 em 41,3 s.
- **O ACHADO:** o `PROVENIENCIA.txt` declara **um** SHA, mas o instalador empacota **tres** origens:
  motor reconstruido do HEAD (`b7f8ee93`), app desktop do commit `1d2ce88c9`, runner de 16/10.
  => *um artefacto composto que declara um SHA para partes de tres origens mente -- e mente mais
  alto do que um sem proveniencia, porque quem le' acredita.*
- **INSTALACAO NAO PROVADA:** tres execucoes nao instalaram um ficheiro nem escreveram log. Protocolo
  antiloop aplicado: 3 tentativas, mesmo sintoma, **parar**. Causa nao determinada; lacuna interna,
  nao `BLOCKED_EXTERNAL`.

Proximo passo declarado: corrigir a proveniencia composta (o script tem de listar o SHA de cada
alvo) e determinar porque o instalador nao escreve. Depois `SigningProvider`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1251

**Ronda 69. C1 CONSEGUIDO: um build real do HADES produzido, medido e reproduzido.**

- **E-1251** -- `internal/agent/build_provider.go` (item 3 do adendo 28.1) + 11 testes. **Artefacto:**
  `hades-windows-amd64.exe`, **66 MB**, SHA-256 `fcef9d9d4303a75b0b9c658f155c9ae6a7ccb84912b4face2268dc51f8161df6`,
  **`NotSigned`**, `--version` responde. **Segundo build = mesmo SHA-256: REPRODUZIVEL.**
- O provider recusa plano incompleto antes de criar `dist`, recusa alvo sem provider, dry-run nao
  executa, checksum recalculado no teste, binario **executado** e saida verificada, e nao tenta
  alvos bloqueados.
- **Erro meu corrigido na leitura:** o `go version -m` imprime o caminho do **ficheiro analisado**;
  eu quase declarei que o binario continha o caminho do build.

Proximo passo declarado: **instalar** o artefacto num Windows limpo (28.10 item 14) e gerar o
instalador Inno Setup. Depois `SigningProvider`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1250

**Ronda 68.** O modo `BACKEND_REAL_AGENT_RUNTIME` foi exercitado -- e apanhou um bug que o fixture
nunca apanharia.

- **E-1250** -- harness a servir o **router de producao** em `127.0.0.1:43219`; `GET /build/plan`
  devolveu `200` com `stacks=go alvos=[linux windows] blockers=inno-setup (ABSENT)` sobre git real.
  E2E modo real **6/2/0**; modo fixture **9/0/0**.
- **O ACHADO:** o painel comparava `saudavel`/`ausente` -- **nomes que eu inventei**. Os `ToolState`
  reais sao `PRESENT_COMPATIBLE_HEALTHY`/`PRESENT_BROKEN`/`ABSENT`/`UNKNOWN`. **O fixture tambem os
  tinha inventado**, por isso passava nos dois lados: *um teste que inventa o valor que o backend
  devolve nunca falha por causa do backend* -- a mesma licao da E-1238. Corrigido no painel, no
  fixture e no mock do teste.

Proximo passo declarado: **C1** -- um build real produzido e instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1249

**Ronda 67.** O painel e a rota passaram a falar um com o outro no browser real.

- **E-1249** -- `e2e/build-plan-panel.spec.ts`, duas classificacoes declaradas
  (`BACKEND_REAL_AGENT_RUNTIME` / `UI_FIXTURE`), rotas novas no fixture com **dois** planos (um com
  alvo pronto e um bloqueado; outro `sem_alvos`). **9 testes PASS**; suite E2E **58 passed / 4
  skipped / 0 failed** (antes 49/4/0). Dois testes sao **skipped no modo real** com a razao escrita
  -- *um teste que finge que o fixture e'' o backend prova o fixture, nao o backend*.

Proximo passo declarado: **exercitar o modo `BACKEND_REAL_AGENT_RUNTIME`** com o harness. Depois
**C1**: um build real instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1248

**Ronda 66.** O plano de build passou a ter painel: o utilizador ve'' o que o projeto pede e o que
falta, sem abrir Go.

- **E-1248** -- `BuildPlanPanel.tsx` + `buildPlanClient.ts`, montado na pagina de Capacidades
  **depois de `WorktreesPanel`**, nao num painel paralelo. **10 testes**. O painel diz **no corpo**
  que apenas detecta e sonda; projeto sem alvos tem mensagem propria, nao erro; estados traduzidos
  para texto humano; blockers e motivos no topo. **Frontend inteiro verde:** `tsc` 0, `eslint .` 0,
  **120 ficheiros / 842 testes**, `vite build` 0, budget PASS.

**Quatro defeitos meus apanhados** -- `vi.importActual` com spread quebrava a transpilacao; campo
no sitio errado (`plano.gestores` vs `plano.projeto.gestores`); assumi que `agentFetch` devolvia
`Response` quando devolve o objecto tipado; e o mock devolvia um `Response` que o produto nao usa.

Proximo passo declarado: **E2E Playwright** do painel contra backend real. Depois **C1**: um build
real instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1247

**Ronda 65.** O plano de release deixou de ser uma funcao que so'' o programador invoca: tem rota.

- **E-1247** -- `GET /api/agent/v1/build/plan?repo=`. **Nao aceita `alvo`** (ha'' teste que pede
  `&alvo=android` num repo Go e exige que seja ignorado) e declara a fronteira **no corpo**
  (`somente_leitura`, `nada_instalado`, `nada_construido`, `aviso_de_fronteira`). Codigos
  distinguidos por significado: **422** quando o caminho existe mas nao e'' um projecto. 10 testes.
  Auto-teste pela rota no repo HADES real: `alvos=[linux web windows]`.

Proximo passo declarado: **painel do frontend** que consome a rota -- e so'' depois **C1**: um build
real instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1246

**Rondas 63-64.** O Build Hub deixou de receber o alvo escrito a mao: detecta o projecto e plana
contra o que ele PEDE.

- **E-1245** -- `ProjectDetector`: 7 stacks, gestor por lockfile, alvos filtrados pelo catalogo
  real, pastas nativas, manifests de prova. 15 testes. **Dois defeitos meus apanhados:** a regra
  de precedencia de lockfile apagava o npm quando era o npm o vencedor; e o `nilerr` acusou dois
  ramos onde eu engolia um erro de leitura em silencio.
- **E-1246** -- o ELO. `PlanoParaProjeto(ctx, raiz)` **nao aceita alvo**: o alvo vem do projecto.
  Cada alvo detectado tem plano ou erro explicito e nunca e'' omitido; `SemAlvos` e'' estado
  proprio, nao sucesso mudo. 9 testes.

Proximo passo declarado: **rota HTTP** para o Mission Control consumir o plano. Depois **C1**
propriamente dito -- um build real instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1244

**Rondas 60-62.** A matriz de lacunas existe, o validador do Studio esta'' ligado ao pipeline, e o
cenario E2E minimo da FASE 3 foi cumprido ponta a ponta.

- **E-1242** -- matriz Orca x HADES e OpenJarvis x HADES. FASE 0 com HEADs auditados
  (`1938bd05…` MIT e `2817faa5…` Apache-2.0), 26 capacidades em 3 grupos, 3 SUPERAR / 4 ADOTAR /
  19 ADAPTAR. **O achado que vale mais do que a matriz:** a primeira passagem do inventario
  procurou ficheiros com nomes inventados e devolveu 8 "AUSENTE"; a segunda procurou conteudo e
  encontrou `pty` 987, `SSH` 88, `Capability` 291, `VRAM` 27, `openai` 56, `inspec` 162.
  *Pela quinta vez o inventario errou na mesma direccao.*
- **E-1243** -- C3 da matriz: os validadores da E-1241 passaram a ser um GATE. `VisualComponent`
  tem `Props`/`Style`/`Bindings`/`Events` que o renderer executa; o validador existente so' via
  estrutura e **2 dos 3** caminhos de `renderVisualHTML` nem isso. 12 eventos e 11 accoes
  allowlisted. `TestCriacaoDeProjectoVisualRecusaComponenteInseguro` falha se a ligacao for
  removida.
- **E-1244** -- o cenario E2E minimo *"comparar duas solucoes e exigir aprovacao antes de
  integrar"*: repositorio git real, **duas solucoes em worktrees separados**, medicao por
  `git diff --stat`, rota de producao, e **prova de nao-integracao** (`main` no mesmo commit,
  `branch --merged` limpo, `extra.go` ausente, branches vivas).

Proximo passo declarado: **C1** -- matriz por target do Build Hub com pelo menos um build real
instalado. Depois **B3** -- os 12 E2E do Obsidian.

Veredito global: `NOT_RELEASE_READY`.

# HADES — TRABALHO PENDENTE PRIORIZADO
## JANELA E-1231 -- FASE 2: WORKTREES COM PROTECCAO, ESTADOS E DESCOBERTA (rondas 45-49)

> Substitui a janela `E-1218`. *A catraca de janela acusou atras=13: o trabalho das rondas 45-49
> nao estava reflectido aqui.* **A protecao de trabalho nao integrado, a maquina de estados e o
> inventario de worktrees existem e estao provados com git real.** O que falta e' a superficie.

### O que ficou feito

| evidencia | o que prova |
|---|---|
| **E-1226** | `RemoveGitWorktree` apagava com `--force` + `os.RemoveAll` SEM verificar nada. Corrigido: diagnostico, erros tipados, guarda de alvo, branch preservado |
| **E-1227** | 9 estados fechados com transicoes declaradas; `InventariarWorktrees` descobre worktrees externos |
| **E-1228** | maquina LIGADA ao codigo: `CreateGitWorktree` -> `preparado`; suspender/retomar/arquivar/marcar-integrado; migracao `active`->`em_uso`; `PodeRemoverAgora` exige as duas condicoes |
| **E-1229** | CI macOS vermelho por minha causa: `/var` e' symlink de `/private/var`. Corrigido com `caminhoReal` |
| **E-1230** | `test (macos-latest)` **success** em `54df8a219`; restam so' os 2 gitleaks historicos |
| **E-1231** | mesmo defeito no runner Windows (`RUNNER~1` -> `runneradmin`); os testes passam por `caminhoReal` |

**37 testes** na suite de worktree, todos com git de verdade.

### As tres licoes de metodo que este ciclo produziu

1. *Apagar trabalho que ninguem integrou nao e' limpeza: e' perda de dados com nome tecnico.*
2. *Um caminho absoluto nao e' o mesmo que um caminho real.* (macOS `/var`)
3. *A correccao existia e o teste que a devia exercitar continuava a comparar strings.* (Windows)

A licao 3 e' a mais util: **a funcao `caminhoReal` foi criada para resolver o macOS e os testes
continuaram a usar `filepath.Clean` — corrigiram o codigo e nao o teste.** O mesmo defeito
apareceu em duas plataformas diferentes porque a comparacao era de strings, nao de alvos.

### O que continua AUSENTE (FASE 2)

- **nenhuma rota HTTP** expoe worktrees, estados, inventario ou diagnosticos;
- **nenhum painel** no Mission Control;
- suspensao/retomada **nao persiste entre processos** (o estado vive na sessao em memoria);
- o aviso de trabalho nao integrado na criacao nao foi exercitado num fluxo de missao ponta a ponta.

Estado da fase: `PARTIAL_IMPLEMENTED_NOT_E2E_VERIFIED`. Veredito global: `NOT_RELEASE_READY`.

### Proxima acao

Expor `GET /api/agent/v1/worktrees` (inventario + diagnostico por sessao) e o painel
correspondente; persistir o estado das sessoes para que suspender sobreviva a reinicio.
## JANELA E-1218 -- SIGNING IDENTITY CENTER COMPLETO; GITLEAKS HISTORICO BLOQUEADO (rondas 33-38)

> Substitui a janela `E-1205`. *A catraca de janela acusou atras=13: os documentos pararam nas
> rondas 27-28 e o trabalho das rondas 33-38 nao estava reflectido aqui. Um documento de
> retomada desactualizado e' pior do que nenhum, porque parece completo.*

### O QUE ESTA' FEITO (duas capacidades, quatro camadas cada)

**Toolchain / Capacidades (FASE 28.6-28.8).** Motor em `internal/agent/toolchain_plan.go`
(14 testes), CLI `cmd/agent_toolchain.go` (8), API `server/agent_toolchain_routes.go` (6),
UI `components/CapabilitiesPage.tsx` (6). Evidencias `E-1209` a `E-1213`.

**Signing Identity Center (FASE 28.5).** Registo `internal/multillm/signing_identity.go` (16),
carregamento `internal/multillm/signing_identity_file.go` (13), API
`server/agent_signing_routes.go` (4), UI `components/SigningIdentitiesPage.tsx` (7).
Evidencias `E-1214` a `E-1216`.

**74 testes novos**, todos a passar. O cofre canonico ja' existia
(`internal/multillm/credentials.go` com DPAPI), e o centro de identidades foi acrescentado como
indice de metadados sobre ele -- nao como segundo cofre.

### O QUE ESTA' BLOQUEADO, E POR QUEM

**`gitleaks (historico do PR)` -- 2 jobs vermelhos.** Causa provada (`E-1217`) e confirmada
(`E-1218`): o commit `5e8fc6173` contem, no historico, um fixture do teste que prova que
`private_key` e' recusado -- e esse fixture usava um valor com entropia de chave RSA real. A
arvore esta' corrigida; a copia antiga esta' num commit que a autorizacao vigente proibe
reescrever.

**Classificacao: `BLOCKED_EXTERNAL`** quanto a resolucao final. Nao falta engenharia; falta uma
decisao do proprietario: squash no merge do PR #438, rewrite autorizado, ou allowlist escopada
ao commit (esta ultima **nao aplicada**, porque o adendo proibe allowlist para findings).

### PROXIMO VALOR

1. E2E Playwright das paginas Capacidades e Assinatura -- a UI so' foi testada com HTTP mockado.
2. Povoar o registo de identidades com o kit real -- exige fatia propria com aprovacao.
3. Matriz de assinatura por plataforma (FASE 28.4).
4. Instalador modular (FASE 28.6-A).

## 0.0 REMEDIÇÃO MEDIDA (2026-10-15, agora) — a secção 0 abaixo fica preservada como HISTÓRIA

> **REMEDIÇÃO MAIS RECENTE: `E-959`** (*cadeia `E-820` → `E-866`, rodadas 181–221*).
>
> 🧑 **E o §18.3-30 («teste de pessoa leiga») foi inventariado (`E-866`) — e fica PARCIAL, com a lacuna
> nomeada:** as **peças** existem (*`Onboarding.tsx` 50 828 B — o maior componente de UI do repositório —,
> `Onboarding.test.tsx` 45 437 B, `ModelPicker.tsx`, help contextual, e onboarding de empresa com limites e
> orçamento*), mas **falta o teste da pessoa**: ninguém percorreu o produto como um leigo.
> ⚠️ E **«instalar sem terminal» tem receita e não artefacto**: `app\ollama.iss` (*Inno Setup 6*) existe e o
> **Inno Setup não está instalado** ⇒ *é a decisão **(c)** do dono, e mantém o **`G10`** em `PACKAGE_CREATED = NÃO`.*
>
> 🔎 **E a «via barata» para o `G3` foi testada em vez de suposta (`E-865`):** o `llama-server.exe`
> **existe** — no Ollama que o utilizador já tem instalado (`lib\ollama\`, com `libllama.dll`,
> `libllama-server-impl.dll` e `libllama-common.dll`). Copiei-o para o sítio onde o Hades procura e
> **o erro mudou**: de `llama-server binary not found` para **`system_info line not found in output`**.
> ⇒ ***O binário corre e não é compatível*** — *o fork modificou `llama/server`, e por isso o motor do
> Ollama oficial não serve*. **O `G3` continua `BLOCKED_EXTERNAL`, mas por um motivo mais preciso.**
> *Os 10,3 MB copiados foram removidos e o Ollama do utilizador ficou intacto.*
>
> 🔥 **E o `G3` mudou de categoria (`E-864`): saiu dos «por medir» e entrou nos `BLOCKED_EXTERNAL` —
> com causa medida e procedimento dado pelo próprio programa.** O Hades **compila** (*65,7 MB, 10,1 s*) e
> **arranca**, mas `POST /api/chat` devolve **404**; o `stderr` diz **`llama-server binary not found`**
> (*o motor de inferência nunca foi construído*) e dá o comando
> **`cmake -S llama/server --preset cpu && cmake --build --preset cpu`**.
> *Exige `cmake` + toolchain C++, que este host não tem* ⇒ *é a decisão **(a)** do dono (`mingw-w64`).*
> **Isto acrescenta um item de bloqueio real à lista:** *sem o motor, o §18.3-07 (`/goal` com modelo local)
> e o §15-G3 ficam bloqueados — e nenhum dos 19 itens abaixo cobre isso, porque não é «trabalho pendente
> de engenharia»: é **ambiente ausente**.* HEAD
> **`9f7379502`** · branch **`fix/checker-relogio`** · **656** evidências · **34** entradas não
> commitadas · **0** stashes · veredicto **`NOT_RELEASE_READY`**. A classificação `E-841` abaixo
> continua válida e fica **preservada**; o que muda é o **estado do trabalho**, medido nesta janela.
>
> **Os 19 itens deste ficheiro continuam classificados (`E-841`)** — *nenhum é defeito de produto* — e
> **todos** continuam **gated em decisões do dono ou em PRs abertos**, não em trabalho de engenharia:
> - **P0-5** — as duas PRs da primeira jornada (#61 vs #71): manter a #71, fechar a #61 só com autorização.
> - **P1-1** — PR #69, merge **não autorizado**. **P1-2** — PRs #56/#58. **P1-3** — PRs #63/#65/#57 (*bloqueado por B-03*).
> - **P1-4** — **28 ficheiros / 44 sítios** que constroem runtime sem `DataRoot`: *é a única fronteira
>   interna real e continua por fechar*.
> - **P1-5** — 104/122 conectores precisam de operador.
> - **P2-2** — Node 20 deprecado em 15 workflows (*exige commit ⇒ travado pelo congelamento*).
> - **P2-3** — replay do upstream: 73 commits, 39 ficheiros em conflito, 15 conflitos reais.
> - **P3-1** — release assinada: **`FUTURE_DISTRIBUTION`** por decisão do dono.
>
> **E o que esta janela provou que NÃO é pendência:** a inferência local funciona e foi **medida**
> (`E-862`/`E-863` — 12,07 s a frio, 0,88 s a quente, trabalho útil idêntico); as defesas de **egress,
> redirect e DNS rebinding** estão provadas (`E-855`); e as **três** matrizes de lacunas não silenciam
> nada (`E-841`/`E-842`).
>
> ⚠️ **REGRA DE MANUTENÇÃO** (*`E-861`, reconfirmada na ronda 218*): este é **um dos três** documentos
> de retomada validados pelo `hades_mission_check.py` (*com `HADES_HANDOFF.md` e `HADES_NEXT_ACTION.md`*).
> O piso de evidência **sobe a cada ronda** ⇒ **actualizar os três na mesma ronda** em que se regista
> evidência. *O checker nomeia um de cada vez; corrigir o nomeado revela o seguinte.*

> **A regra do próprio ficheiro (linha ~27) diz: «qualquer número acima envelhece. Antes de citar, remeça.»** Isto é o
> resultado de a cumprir. **A secção 0 cita `cafb00dbe` / `06cd277da`** — e os dois são **ancestrais do HEAD**
> (`git merge-base --is-ancestor` => exit 0), com o segundo **55 commits atrás** (`git rev-list --count 06cd277da..HEAD`
> = **55**). *Nada foi apagado: o histórico fica, e estes números substituem-no até nova remeção.*

| Medida | Valor **medido agora** |
| --- | --- |
| HEAD local | `9f7379502b1d5d5bb7abb8e2bd55224e92796dfd` |
| Remoto `refs/heads/fix/checker-relogio` (`git ls-remote`) | `9f7379502` => **local = remoto**, **0 por empurrar** |
| `main` remota (`git ls-remote`) | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` |
| Distância dos números da secção 0 ao HEAD | **55 commits** |
| Evidências no índice (`Select-String '^\| E-\d+ '`) | **571** linhas = `E-001..E-570` + **`E-778`** (o `E-778` é o próximo ID livre; a secção 0 diz 546 — **histórico**) |

> **REMEDIÇÃO SEGUINTE (2026-10-15, 19:14 BRT).** A linha acima **já envelheceu** — e é a regra deste próprio
> ficheiro que o exige dizer. Foi escrita com **571** linhas e **`E-778`** como próximo ID livre. Valores
> **medidos agora**, no mesmo HEAD `9f7379502`:

| Medida | Valor **medido agora** |
| --- | --- |
| Linhas do índice | **591** = `E-001..E-570` + **`E-778`..`E-798`** |
| **Próximo ID livre** | **799** (prefixo `E-`) — *herdar `E-778` da tabela acima causaria **colisão de ID**, o defeito que o próprio `E-778` documentou. Escrito como número, e não com o formato de evidência, porque o `hades_mission_check.py` exige que todo ID citado esteja registado — e um ID **futuro** não pode estar* |
| `ENTRIES` / `STASHES` | **29** / **0** |
| Checkers oficiais | **7/7** `exit 0` |

> O **`E-798`** (nesta mesma ronda) corrigiu o **item P1 3** abaixo: a classificação `BLOCKED_EXTERNAL` do pacote
> `./server` estava **refutada** pelo `E-795`. **O item P1 4 continua aberto** (hermeticidade dos testes).

> **`E-778` — COLISÃO DE IDs (o defeito mais grave desta sessão, e é meu).** A **sequência viva** do projecto vai em
> **`E-777`**, cunhada em **código e commits** e **nunca** registada no índice; este índice registava até `E-570` ⇒
> **7** das entradas `E-560..E-570` **colidem** com outro objecto já citado em código committado. O instrumento novo
> (passo 15 do `hades_mission_check.py`, `E-570`) verifica **existência** — e *a existência **não** distingue um ID
> livre de um ID já ocupado*. ⇒ Duas regras ficaram **mutuamente exclusivas**: o passo 1 exige **contiguidade** e o
> `E-778` exige **não-colisão**. Cura aplicada: um buraco **declarado** em `evidence_ids_nao_registrados` deixa de ser
> falha; o que continua a falhar é um buraco **não declarado**.
> **⇒ Ao retomar: NÃO reutilize nenhum ID abaixo de `E-778`.**
| Árvore de trabalho | **16** entradas não commitadas, **0** stashes — **WIP do proprietário preservada** (naquele momento) |
| Árvore de trabalho (**medido agora**, `E-799`) | **29** entradas não commitadas, **0** stashes — *o **16** acima envelheceu: a árvore cresceu com os artefactos e scripts G12, o bundle e a consolidação de documentos* |
| Veredito global | **`NOT_RELEASE_READY`** |

### CI DO SHA CONGELADO — medido agora, e é o primeiro registo deste número

O SHA candidato `9f7379502` **tem CI remoto concluído**, com **5 workflows `success` no MESMO `head_sha`**
(evento `pull_request`, PR **#438**, arranque `2026-10-15T17:19:55Z`):

| Workflow | Run | Resultado |
| --- | --- | --- |
| `test` | `38776430738` (#1242) | **success** |
| `dz23-agentic-quality` | `38776430666` (#2018) | **success** |
| `pr-gate` | `38776430633` (#979) | **success** |
| `dz23-e2e` | `38776430658` (#341) | **success** |
| `class-a-plus-integrity` | `38776430631` (#2116) | **success** |

=> *O `pr-gate` é o **agregador** (recusa `cancelled`/`skipped` sem mascarar a falha — `E-551`), e está verde **neste** SHA.*
***LIMITE, e importa:*** *este CI prova a **árvore commitada** de `9f7379502` — **não** as **16** alterações não commitadas
da árvore de trabalho. A revisão independente usa **um** SHA (§18.5.10), e esse SHA está verde.*

### P0/P1 ABERTOS, MEDIDOS NESTA SESSÃO (arquivo:linha + bloqueio)

1. **P0 — o seletor de modelo do Chat não persiste contra o `serve`.** `server/desktop_local_routes.go:44-89`: o `GET`
   devolve um objecto **hard-coded** e o `POST` responde **501 honesto** (antes: 200 ecoando o corpo **sem gravar**).
   Bloqueio: **CGO ausente** neste host (`CGO_ENABLED=0`, `gcc` ausente) => `app/store` (SQLite) é um *stub* em runtime.
   **Decisão pendente do dono** — trazer um toolchain C de terceiros é uma acção de **supply chain**, e a rede funciona.
   Rota real do produto: `app/cmd/app/app.go:30-32` + `app/ui/ui.go:1631,1653`. Evidências `E-560`, `E-561`, `E-563`.
2. **P0 de instrumento — o gate local podia estar VERDE com o caminho de persistência VERMELHO.** O passo 4 não incluía
   `./app/...` (o CI inclui: `dz23-agentic-quality.yaml:82`, `release.yaml:78`). **Corrigido** em
   `scripts/hades_local_gates.ps1`: passo 5 condicionado a cgo, escopo declarado no resumo, e `-First`->`-Last` no passo 2.
   Evidência `E-561`.
3. **P1 — o pacote `./server` não é reprodutivelmente verde neste host.** `whatsapp_routes_test.go:38` =>
   `internal/agent/runtime_storage.go:66` (`mkdir %APPDATA%\ollama-agent\<hash>: Acesso negado`), **~92** testes em
   cascata. Provado **não** ser ACL, **não** confinamento, **não** o caminho => **processo filho do harness**.
   `BLOCKED_EXTERNAL` para a verificação local desse pacote. Evidência `E-562`.
   **CORRIGIDO — medido em `E-795`/`E-796` (2026-10-15): esta classificação está REFUTADA.** Com
   `APPDATA`/`LOCALAPPDATA` redireccionados para um directório de trabalho, o pacote `./server` fica verde
   **3/3 corridas** (`exit=0`; `ok` **29,665 s** / **25,994 s** / **29,231 s**; `FAIL=0`; «Acesso negado»=**0**),
   e o teste `whatsapp_routes_test.go` corre e **passa** (`-run WhatsApp -v` => **7 PASS / 0 FAIL / 0 SKIP**).
   O estado deixa de ser `BLOCKED_EXTERNAL` para a verificação local: passa a **EXECUTÁVEL com isolamento
   de `APPDATA`**. O **mecanismo** do `Acesso negado` continua **por determinar** (o directório existe no
   perfil real, o `mkdir` funciona a partir do shell, e o `ollama-agent` real tem **0** subdirs): *efeito
   medido, causa não*. O **item 4 abaixo continua aberto** (hermeticidade: 42 sítios sem `DataRoot`) —
   o isolamento é uma **técnica de invocação**, não uma correcção do teste.
4. **P1 — hermeticidade dos testes de `./server`: 42 sítios sem `DataRoot` em 27 ficheiros**, e
   `agent_runtime_defaults_test.go:42` tem o **default da plataforma como objecto** (torná-lo hermético destruiria o
   teste). `XDG_CONFIG_HOME` (`:64`) é **ignorado no Windows** por `os.UserConfigDir()`. Evidência `E-563`.
   **REMEDIDO (`E-799`): o predicado «42 sítios em 27 ficheiros» NÃO é reproduzível como está escrito.** Medido:
   **150** ficheiros `*_test.go` em `server/`, **28** mencionam `DataRoot`, **122** não mencionam, **46** ocorrências
   no total. *Nenhuma destas contagens dá 42 nem 27.* O item **continua aberto** (a hermeticidade não foi provada
   nem refutada) — mas fica registado que **precisa do predicado exacto** para poder ser remedido. Sem isso,
   um auditor não consegue nem confirmar nem fechar o item.
   **PREDICADO RECUPERADO (`E-800`).** Testados **7** predicados naturais em `server/*_test.go` (**150**
   ficheiros). O que se aproxima é: *ficheiros que constroem o runtime* (`runtime.New`|`NewRuntime`|`agentRuntime`)
   *e não mencionam `DataRoot`* => **28 ficheiros / 44 ocorrências** (ocorrências dos construtores dentro desses
   ficheiros), contra os **27 / 42** aqui escritos. Diferença: **1 ficheiro e 2 sítios**.
   **Duas leituras, e o `E-800` não decide entre elas:** *(a)* o predicado é este e estes números **envelheceram**
   (a árvore ganhou 1 ficheiro e 2 sítios); *(b)* falta **mais uma condição**. O que está provado: existe um
   predicado **reproduzível** (28/44) e **nenhum** dos 7 testados dá 27/42. **O item continua aberto** — e a
   lacuna é **maior** do que aqui estava escrito, não menor. Comando: contar `*_test.go` de `server/` com
   `runtime.New|NewRuntime|agentRuntime` e sem `DataRoot`.
   *Pistas úteis do mesmo teste:* `newTestServer|setupServer|newServer` = **19** ficheiros / **131** ocorrências;
   `t.TempDir()` = **89** ficheiros / **318** ocorrências — *muitos testes isolam um diretório temporário, mas não
   o `DataRoot`.*
5. **P1 — `integration/max_queue_test.go:20`**: `t.Skip` **incondicional** (teste desligado em todos os ambientes; a
   guarda de `:22-25` é código morto). Dos **292** `t.Skip` do repositório, é o **único** incondicional. Evidência `E-564`.

### DÍVIDAS DE INSTRUMENTO FECHADAS (para o auditor não as reabrir)

- `E-553` (o `t.Skip` do symlink): **condição lida** — `internal/agent/media_test.go:290` salta por **capacidade**
  (`os.Symlink` falhou), depois da asserção do caminho seguro (`:281-288`, que corre sempre) e antes da de recusa
  (`:293-298`) => **não esconde nada que pudesse ter corrido**. A dívida declarada era **falsa**. Fechada em `E-564`.

## 0. ESTADO MEDIDO — `fix/checker-relogio` `cafb00dbe` (esta é a leitura canônica; não herde número de texto antigo)

> **BLOQUEIO MEDIDO (2026-10-15, item 13 do complemento 48 — teste de browser).** O browser mode do `vitest 3.2.7` **está quase montado**: existem `@vitest/browser`, `playwright`, `playwright-core`, `@playwright/test`, `react-dom` e **Chromium instalado** (`ms-playwright/chromium-1228,1243,1248`). **Faltam exactamente duas peças** — `@vitest/browser-playwright` (*o provider que o Vitest ≥3.2 exige*) e `vitest-browser-react` (*renderizador React para browser mode*); `vitest.config.ts` só tem `environment: "node"`. ⇒ **`BLOCKED_BY_EXTERNAL_DEPENDENCY`** com a peça nomeada. **DECISÃO TOMADA:** *não* adicionei as dependências ao `package.json` — fazê-lo **sem regravar o lockfile quebraria o `--frozen-lockfile` do CI**, trocando um teste em falta por um CI vermelho. **Comando correcto (operador, com rede):** `pnpm add -D @vitest/browser-playwright vitest-browser-react` **+ commit do lockfile no mesmo commit**. *Todo o resto da cadeia da sala está `TESTED`; falta só a fiação React de 3 linhas.*

> **FRENTES 45/46/47 (prioridade 7 da FASE 6) — MEDIDAS (2026-10-15), e uma correcção minha.** Eu tinha escrito «prioridades 7 ainda intocadas»: **a medição diz que não.** *Companion* tem **domínio + testes + UI**: `internal/agent/companion_estado.go` (conjunto **fechado** de 8 estados; `DerivarEstadoCompanion` devolve **estado E razão**; `CompanionPodeDizer(estado, frase)` é a guarda contra o mascote afirmar o que não pode provar; invariante **«sem missão não há estado honesto»** ⇒ `ErrCompanionMissaoAusente`) + `companion_estado_test.go` (9 casos) + `CompanionMascote.tsx` (174 linhas) com `CompanionMascote.test.tsx` (movimento, anúncio para quem não vê, **ausência declarada**, onboarding). *Studio* tem página real (`StudioCanvasPage.tsx` ≈950 linhas, rota `studio.tsx`, item no `AppSidebar`, aviso extraído para ser testável). **LACUNA MEDIDA:** `DerivarEstadoCompanion` **não aparece em `server/`** ⇒ *o Companion tem domínio e UI mas **não tem rota** que alimente a UI com o estado* — a mesma classe do `EvaluationRunner`, medida por nome. **Artifact Drive — MEDIDO no vocabulário correcto (`Artefato|Biblioteca|Library`), e o meu «0» era o TERCEIRO falso negativo meu:** existe **`LibraryPage.tsx`** + rota `/library` + item «Biblioteca» no `AppSidebar` + entrada na busca (`page-library`) — agrega artefatos de **todas** as missões, com **busca e filtro por categoria**; **`ArtifactsViewer.tsx`** faz download real (`/api/agent/v1/missions/:id/artifacts/:artifactId`) e mostra **`sha256` rotulado «Hash de Proveniência SHA-256»**; **`AgenticSplitShell.tsx`** integra o painel «Artefatos da Missão (n)» (**com teste**: `AgenticSplitShell.test.tsx` exige «Artefatos (1)»); **`MissionDeliveryCard.tsx`** entrega com sha256; **`StudioCanvasPage.security.test.ts`** lê o **próprio fonte** do `ArtifactsViewer` (padrão de teste de segurança sobre o código). ⇒ **UI ✅ + proveniência ✅ + busca ✅ + prévia ✅ + testes ✅**. **NÃO medido:** versionamento *por artefato*, permissões *por artefato* em multi-inquilino e dedupe *real* (o sha256 é o substrato, não a prova do dedupe). **Lição registada: três falsos negativos meus, todos por procurar o nome em inglês num domínio de identificadores pt-BR.**

> **COMPANION — a lacuna é MAIOR do que «falta a rota» (medido 2026-10-15).** `CompanionEventoVerificado` aparece **11 vezes no repositório inteiro**: na definição, dentro de `DerivarEstadoCompanion` e no `companion_estado_test.go`. **Não há PRODUTOR em lado nenhum** — *nada no sistema constrói este tipo a partir de eventos reais de missão.* A forma exacta (para ser accionável, não uma queixa): `Tipo string` (conjunto fechado), `MissaoID`, `PassoID` (opcional), `Quando time.Time`, `Detalhe string` — **e o próprio código declara a exigência**: *«`Detalhe` é texto já sanitizado a montante: este ficheiro não inventa detalhe, só o transporta»* e *«`Quando` existe para que um evento futuro não seja tratado como passado»*. ⇒ **Faltam DUAS peças: (1) conversor `eventos reais → CompanionEventoVerificado` (só eventos que ACONTECERAM; `Detalhe` sanitizado a montante/DLP; `Quando` real) e (2) rota `GET …/companion/:missao_id` → estado + razão + escopo.** *Item 46 = `IMPLEMENTED_PARTIAL`/domain-only, com o caminho cortado em dois pontos — é a doutrina «provar o caminho completo da UI até ao efeito real» a acusar exactamente onde dói.* **A MEDIR (tentativa falhada, padrões declarados):** `type EventoDaMissao|type MissionEvent|EventosDaMissao|type Evento struct|Eventos []Evento` ⇒ **0** em `internal/agent` — *o tipo de evento de missão tem outro nome e tem de ser medido antes de escrever o conversor* (lição dos três falsos negativos: **não inventar o nome**).

> **COMPANION — RESOLVIDO (2026-10-15): o teste de handler encontrou um defeito REAL na minha própria entrega.** Publicado em `5c403c8c9`, o conversor passava **todos** os eventos a `DerivarEstadoCompanion` — que **recusa os que não pertencem ao vocabulário**. E `Runtime.CreateMission` emite **`mission.planned`**, que **não** está em `EventosVerificadosDeMissao` (12 nomes exportados em `companion_estado.go:48`). Consequência medida: **a rota respondia `400` para TODA missão recém-criada** — *o Companion ficaria permanentemente indisponível em produção: um controlo que existe, que é testado, e que ninguém consegue usar.* **Causa-raiz:** *um adaptador que entrega a uma função o que ela declara rejeitar não é um adaptador — é um erro à espera de acontecer.* **Correcção (`831d8dad6`):** o conversor **filtra ao vocabulário** (`CompanionEventoEhVerificado`, reutilizando a lista exportada — *não guarda cópia que envelhece*) e **conta os descartados** (`…ComRelatorio`), *porque um descarte invisível é indistinguível de uma perda de dados*. **Provas:** `go test ./internal/agent -run Companion` → **ok**; `go test ./server -run CompanionRouteReportsHonestState` → **PASS**; checker de codificação → OK; gofmt limpo. **Duas expectativas MINHAS que estavam erradas** (e o que ensinaram): missão nova tem `mission.planning` ⇒ o estado honesto é **`a_planear`**, não «ausente»; e o conversor **ordena**, logo indexar por posição de entrada estava errado — *a falha provou que a ordenação actua*. **O que faltou ter feito antes:** declarar `IMPLEMENTED_UNTESTED` e **não** afirmar «cadeia completa» no commit — *foi o teste que a meta exige que impediu a entrega avariada.* **E2E:** nenhuma das três tem passagem E2E provada.

| Medida | Valor medido |
| --- | --- |
| Commit medido (**local = remoto**, 0 por empurrar) | `06cd277da` |
| Remoto da branch | `06cd277da` (*nenhum commit local pendente*) |
| `main` remota | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` (**medido agora** com `git ls-remote`, 2026-10-15 — *não é valor herdado*) |
| Evidências no índice | **546** (**histórico** — não re-medido; *não herde este número*) |
| Estágios com evidência | **38 de 38** (**histórico**) |
| Bloqueios declarados | **17** (**histórico**) |
| Matriz do gate final | `BLOCKED_BY_EXTERNAL_DEPENDENCY: 5 · DONE: 19` (**histórico**) |
| Veredito global | **`NOT_RELEASE_READY`** |
| **Checkpoint 2026-10-15 (BRT) — medido** | **CI do SHA `06cd277da`: `NOT MEASURED`** — o run `38741470515` ficou **preso do lado do GitHub** (`in_progress`, `updated_at` congelado em 13:17:43Z), com `Verify formatting` e `Test multi-provider contracts` já **verdes**. *Último veredito medido: `abd3771ab` ⇒ `class-a-plus-integrity` **success** + `dz23-multi-provider` **success** (13:04:43Z).* **Provas locais no SHA corrente:** `python scripts/hades_encoding_check_test.py` **OK** · `prettier --check` **limpo** · **`vitest run` na UI inteira: 100 ficheiros / 661 testes PASS** · `tsc -b` exit 0 · `gofmt` limpo · `go build ./...` exit 0 · `golangci-lint` **0 issues** (`./internal/agent/...` e `./server/...`) · `go test ./server -run TestCollab` **5/5 PASS**. **Entregue no complemento 48 (F2) — cadeia da sala:** domínio → **rotas HTTP** (`GET/POST /api/agent/v1/collab/:project_id/members`, `.../rooms`, `.../rooms/:room_id/members`; *origem = ator autenticado; sala inexistente ⇒ **404**, nunca lista vazia*) → cliente `salaApi.ts` (*falha nunca vira lista vazia*) → componente `SalaMembros.tsx` → painel `SalaPainel.tsx` (*estado em união discriminada*) — **todos `TESTED`**; **`SalaLigada.tsx` = `IMPLEMENTED_UNTESTED`** (*medido: sem `jsdom`/`happy-dom`/`testing-library`; existe `@vitest/browser` ⇒ é esse o teste que falta*). **Bloqueio humano declarado:** PR #438 `blocked` por revisão (`LMPrado-DZ23`) — **não contornar**. **Próximas ações:** ① teste de browser da `SalaLigada` (para `E2E_VERIFIED`); ② ler o veredito de `06cd277da` quando houver runner; ③ retomar as prioridades da FASE 6 (Paperclip → Cloudroom/Teleport → Studios → homologação e benchmarks reais). |

**Medido em `2026-10-14 16:35 UTC`.**

**Regra desta seção (E-301, aplicada em E-310, e outra vez em E-544):** qualquer número acima **envelhece**. Esta seção
já ficou desatualizada **duas** vezes (dizia `f190ec8ea` / 300, e depois `fafda19e7` / 310) e foi remedida. ⇒ *E o `E-544`
mediu o custo: este documento e o `HADES_HANDOFF.md` estiveram **parados ~16 h**, sem mencionar **um único** commit ou
evidência recente.* **Antes de citar, remeça:**

```powershell
git rev-parse --short HEAD
(Select-String -Path docs/mission/HADES_EVIDENCE_INDEX.md -Pattern '^\| E-\d+ ').Count
python scripts/hades_mission_check.py                      # ESCRITURAÇÃO CONSISTENTE (exit 0)
python scripts/hades_ci_reproducibility_check.py --check    # nenhum gate flutuante (E-309)
Select-String -Path docs/mission/HADES_FINAL_RELEASE_GATE.md -Pattern 'Resumo: '
```

**Decisão preservada:** `docs/mission/HADES_CURRENT_STATE.json` **não existe por decisão explícita** — dois arquivos
de estado divergiriam. O estado canônico é `HADES_MASTER_STATE.json`. **Não criar** o outro.

**Retomada exata (comandos reais, nenhum passo presumido):**

> ⚠️ **A PRIMEIRA LINHA DESTRÓI TRABALHO LOCAL. LEIA A CONDIÇÃO ANTES DE A CORRER.**
> *`git reset --hard origin/main` é o passo **certo** para quem retoma **da `main`** — e **destrutivo** para quem tem
> commits locais por empurrar.* ⇒ **Confirme primeiro:**
> ```powershell
> git rev-list --count origin/fix/checker-relogio..HEAD   # 0 = seguro; >0 = HA' TRABALHO LOCAL
> ```
> *Se der **> 0**, **não** corra o `reset --hard`: empurre primeiro (um push por corrida — `E-541`), ou guarde o
> trabalho numa branch.* **O `reset --hard` não avisa: ele apaga.**

```powershell
git fetch origin main; git reset --hard origin/main
$repo = (Get-Location).Path
$env:GOROOT = "$repo\_tmp\goroot1269\go"; $env:GOPATH = "$repo\_tmp\gopath"
$env:GOMODCACHE = "$repo\_tmp\gomodcache"; $env:GOCACHE = "$repo\_tmp\gocache1269"
$env:GOTMPDIR = "$repo\_tmp\gotmp"; $env:GOFLAGS = '-mod=mod'; $env:GOTOOLCHAIN = 'local'
$env:PATH = "$env:GOROOT\bin;$env:PATH"
gofmt -l internal/agent internal/multillm server/agent_routes.go   # tem de sair VAZIO
go test ./internal/agent/ -count=1
python scripts/hades_release_gate.py; python scripts/hades_baseline_report.py
python scripts/hades_stage_map.py; python scripts/hades_mission_check.py
python scripts/hades_ci_reproducibility_check.py --check
git checkout -b feat/<escopo>   # branch curta + PR; nunca commit direto em main, nunca force-push
```

**Onde o trabalho parou (para quem retoma):** estágios **36–38** com métricas, publicação, jornada E2E e a fronteira
cruzada já provadas; faltam **STT/TTS e telefonia reais** (dependem de provedor/número/credencial —
`BLOCKED_BY_EXTERNAL_DEPENDENCY`), **104/122 conectores** que exigem operador (`APENAS_CATALOGO`), artefato **não
assinado e só Windows**, e a **decisão de política** do `misspell` (dicionário inglês) ao subir a versão do linter.
# 1. baseline: main integrada e limpa
git fetch origin main; git reset --hard origin/main
# 2. toolchain fixada do repositório (Go declarado em go.mod)
$repo = (Get-Location).Path
$env:GOROOT = "$repo\_tmp\goroot1269\go"; $env:GOPATH = "$repo\_tmp\gopath"
$env:GOMODCACHE = "$repo\_tmp\gomodcache"; $env:GOCACHE = "$repo\_tmp\gocache1269"
$env:GOTMPDIR = "$repo\_tmp\gotmp"; $env:GOFLAGS = '-mod=mod'; $env:GOTOOLCHAIN = 'local'
$env:PATH = "$env:GOROOT\bin;$env:PATH"
# 3. bateria obrigatória ANTES de qualquer push
gofmt -l internal/agent internal/multillm server/agent_routes.go   # tem de sair VAZIO
go test ./internal/agent/ -count=1
python scripts/hades_release_gate.py; python scripts/hades_baseline_report.py
python scripts/hades_stage_map.py; python scripts/hades_mission_check.py
# 4. branch curta + PR (NUNCA commit direto em main, NUNCA force-push)
git checkout -b feat/<escopo>
```

**A lição que mais custou rodadas (E-300b):** o gate `Preserve Class A+ surfaces` falha com
`test -z "$(gofmt -l internal/agent internal/multillm server/agent_routes.go)"` — **reproduza esse comando localmente
antes de corrigir**, em vez de supor a causa.


> Ordem de execução: **P0 → P1 → P2 → P3**. Nada aqui é declaração de conclusão;
> é backlog com critério de pronto.

## P0 — bloqueia o release e a honestidade do produto

## CLASSIFICAÇÃO EXPLÍCITA DOS 5 ITENS SEM ESTADO (E-841)

> **§18.5-2**: *«Não deixar "falta implementar" genérico — cada pendência tem
> arquivo:linha, teste falho, pessoa responsável, bloqueio concreto ou decisão de
> escopo.»* Os 19 itens deste documento foram auditados: **14 tinham classificação
> explícita** e estes **5 não tinham**. Cada um fica agora com **responsável e
> bloqueio concretos**, sem inventar causa.

| item | classificação | bloqueio concreto | responsável |
|---|---|---|---|
| **P0-5** · jornada de primeira execução duplicada | **DECISÃO DO DONO** | *O próprio item exige: «fechar o redundante **somente com autorização explícita»**. Recomendação registada: manter **PR #71** (já gateado no CI) e fechar **#61** **após** o merge de #71. Não é trabalho de engenharia — é um clique de governação.* | proprietário |
| **P1-1** · saída de ferramenta na linha do tempo | **DECISÃO DO DONO** | *«PR #69 (`feat/saida-ferramenta-no-chat`, `525856a75`) está **verde**. Falta **autorização de merge**.»* *A parte opcional já tem **arquivo nomeado**: `AgenticSplitShell.tsx` — e o item declara que a linha do tempo **já mostra `payload` genérico**, logo o essencial **existe**.* | proprietário |
| **P2-2** · aviso de depreciação do Node 20 no CI | **ACCIONÁVEL — bloqueado só pela regra de congelamento** | *Trabalho **mecânico e medido**: `actions/checkout@11d5960a…` e `actions/setup-node@49933ea5…` (v4 com SHA fixo) emitem `Node.js 20 is deprecated` — **confirmado em log real de CI** (`E-823`). A acção é uma **PR dedicada** subindo os `uses:` para v5 nos **15** workflows. **Exige commit**, e commitar move o SHA congelado `9f7379502` — que é a única prova de CI verde desta fase. **Não é FAIL: é sequenciamento.** | proprietário (abrir PR) |
| **P2-3** · replay curado do upstream | **SINCRONIZAÇÃO PLANEADA — não é defeito** | *O item **quantifica** o trabalho: «replay linear temático sobre `upstream/main` (**73 commits, 39 arquivos em conflito, 15 conflitos reais**). 4 gates obrigatórios + smoke Chromium antes de abrir PR.» É **sincronização com o Ollama upstream** (§00.2 manda preservá-lo), não um bug por corrigir. | engenharia, em janela dedicada |
| **P3-1** · release assinado por SO | **`FUTURE_DISTRIBUTION`** | *Depende do **`B-06`** (certificado). E a decisão do proprietário é **explícita**: «**NÃO VAMOS COMPRAR OU CONFIGURAR AGORA CERTIFICADOS COMERCIAIS** de assinatura Windows/macOS, nem publicar nas lojas.» Pelo §18.5-7, fica como **`FUTURE_DISTRIBUTION`** e **não** reprova o gate de engenharia/build. | proprietário (fase futura) |

**Efeito no veredicto:** *nenhum destes 5 itens é um defeito de produto por
corrigir — **2** são decisões de governação (P0-5, P1-1), **1** é sequenciamento
(P2-2), **1** é sincronização planeada (P2-3) e **1** é distribuição futura já
decidida pelo dono (P3-1).* ⇒ *Depois desta classificação, **19/19** itens têm
estado, bloqueio e responsável.*

---

### P0-1 · Checkpoints da missão V2.1 (PR #72)
- **Pronto quando:** os 11 artefatos existem, o JSON é válido, `PR gate` verde.
- **Estado:** **ENTREGUE** — PR #72 aberto com os 11 artefatos e **15/15 checks
  sem falha** (E-036); JSON válido (`stages=38`, `blockers=9`). Falta autorização
  de merge.

### P0-2 · Cobertura E2E do Studio — **ENTREGUE (PR #73)**
- **Correção:** o drag-and-drop do Studio **existe** (reorder por arraste em
  `StudioCanvasPage.tsx:975-988` e por botão/teclado `1012/1024`, ambos
  persistindo via `updateBuilderVisual`, com 5 testes unitários de
  `reorderById`). O gap real era **falta de spec Playwright que assegure o
  comportamento**, não falta de funcionalidade. Nunca reafirmar "DnD ausente".
- **Entregue:** `e2e/studioReorder.spec.ts` (3 testes) + `e2e/studio-builder-fixture.mjs`,
  provando ordem vinda do servidor, persistência do arraste, paridade pelo botão
  acessível e sobrevivência ao reinício. Verde no CI: **23/23 checks**, com os 3
  testes passando no job `Web E2E (shell smoke)` (E-034, E-035).
- **Restante do escopo:** preview/export, undo/redo e edição de propriedades sob
  E2E — em backlog (P1/P2).

### P0-3 · Corrigir o backlog histórico — **ENTREGUE (PR #73)**
- `audit/BACKLOG_LACUNAS_PRIORIZADO_20261009.md` afirmava "arraste real
  inexistente" no Studio. Corrigido com a evidência do probe (E-008/E-009), mais
  a contagem honesta dos 6 itens do roadmap e a ordem recomendada.

### P0-4 · Decidir o gate de onboarding quando `GET /settings` falha
- **Situação:** `routes/index.tsx` engole o erro de `getSettings` e cai no shell
  sem redirecionar ao onboarding ⇒ usuário novo sem backend vê a aplicação e não
  o onboarding. Pode ser comportamento desejado (modo local) ou defeito.
- **Critério de pronto:** decisão documentada + teste que fixe o comportamento
  escolhido (hoje há apenas o comportamento implícito).
- **REMEDIDO (E-816, 2026-10-15): o critério está cumprido, e o desfecho é a
  favor do produto.** Lido o código, **não é um erro engolido**: `routes/index.tsx:17-20`
  tem o motivo **escrito** no `catch` («The local-first shell must remain usable
  when the optional preferences API is offline. Chat/agent requests will show
  their own safe states.»), e o onboarding **é** alcançado quando a API responde
  (`:21-26` — `OnboardingVersion < CURRENT_ONBOARDING_VERSION` ⇒
  `throw redirect({ to: "/onboarding" })`). A decisão está **certa por mérito**:
  um usuário **novo sem backend** não teria onde **gravar** a versão de onboarding,
  logo o shell é o comportamento correto — coerente com o §02.3 («Chat local
  básico funciona sem login cloud e sem rede»). O teste que faltava existe agora:
  `app/ui/app/src/routes/index.onboarding-gate.test.tsx`, **3/3 PASS**, fixando os
  três casos (API offline ⇒ chat; versão antiga ⇒ onboarding; versão atual ⇒ chat).

### P0-5 · Jornada de primeira execução duplicada
- PR **#61** (`test/e2e-jornada-primeira-execucao`, `bb45d7944`) e PR **#71**
  (`feat/e2e-primeira-execucao`, `3b5459ef5`) cobrem a mesma jornada.
- **Critério de pronto:** decidir qual supersede o outro e fechar o redundante
  **somente com autorização explícita**. Recomendação: manter #71 (já gateado no
  CI) e fechar #61 após o merge de #71.

## P1 — paridade competitiva (TypingMind / AutoClaw / Manus)

### P1-1 · Saída de ferramenta visível na linha do tempo
- PR #69 (`feat/saida-ferramenta-no-chat`, `525856a75`) está verde. Falta
  autorização de merge. Opcional: renderizar o excerto de `output` em
  `AgenticSplitShell.tsx` (a linha do tempo já mostra `payload` genérico).

### P1-2 · Paridade de chat TypingMind
- PR #56 (chat) e #58 (biblioteca de prompts) abertos. Próximo: comparação lado
  a lado multi-modelo e biblioteca de prompts usável de ponta a ponta.

### P1-3 · Canais de IM (Telegram/WhatsApp/inbox)
- PRs #63 (Telegram) e #65 (guia de conexão) abertos; PR #57 traz o comparativo
  AutoClaw. **Bloqueio real:** homologação com token exige B-03. Manter a
  classificação `APENAS_ADAPTER` até haver chamada real.

### P1-4 · Harness CLI governado — **PROVADO (E-830); o merge é decisão do dono**
- **REMEDIDO (E-830, medido):** o item pede *«prova de que a aprovação obrigatória
  (`RequiresApproval: true`, `Risk: RiskWrite`) é exigida **de fato**, com teste
  reproduzível»*. **O teste existe e usa os dois campos exactos**:
  `internal/agent/mutation_gate_test.go:60` → **`TestMutationHarnessHITLApprovalBypass`**,
  com `Step{… Risk: RiskWrite (**:64**), RequiresApproval: true (**:65**) …}` — e a versão
  actual chama as funções **reais** (`approvalsReady`, `stepApproved`,
  `validateMissionApprovalBindings`) sobre um `Runtime` e uma `Mission` de verdade.
- **E o ficheiro documenta um achado que vale mais que o item**: `:47-59` — *«**E-401 /
  ADV-58/1 (5ª auditoria independente, MEDIDO)**: esta função era **TEATRO**. As linhas
  59-64 e 72-75 eram **closures LOCAIS** que reimplementavam a condição
  `if s.RequiresApproval && !approved`, e a asserção final comparava **uma closure com a
  outra**… A contraprova do auditor é decisiva: com **TODOS** os pontos de aprovação do
  runtime desligados, este teste continuava **PASSANDO**.»* ⇒ ***«Teste de mutação que não
  toca o produto não é teste de mutação: é um espelho.»***
- **E um segundo achado (`:72-74`)**: *com `Kind: "git.merge.origin"`, a guarda
  anti-secreto **recusa persistir o payload*** ⇒ *duas defesas no mesmo caminho, e uma
  encobria a outra no teste*.
- **E a fronteira do merge**: *o item diz que o PR #62 é **branch-local*** ⇒ **merge não
  autorizado** (review pendente) ⇒ **decisão do dono**.
- **Situação original (história):**
- **REMEDIDO (E-829, medido):** o item pedia *«prova de que a aprovação obrigatória
  (`RequiresApproval: true`, `Risk: RiskWrite`) é exigida de fato, com teste
  reproduzível»*. **Medido**: existe **`internal/agent/cli_harness_tool.go`** e
  **`cli_harness_test.go`**; e `harness.cli`/`RequiresApproval`/`RiskWrite` tem **126**
  ocorrências em `internal`, `server`, `cmd` — lideradas por **`planner.go` (16)**,
  `runtime_test.go` (12), `runtime.go` (10), `cli_harness_tool.go` (6),
  **`mutation_gate_test.go` (6)**, `approval_payload_fields_test.go` (4),
  `cli_harness_test.go` (4), `hardening_test.go` (3).
- **E os testes correram**: `-run 'MutationGate|CliHarness|CLIHarness|ApprovalPayload|Hardening'`
  ⇒ **exit=0**, **PASS=2 / FAIL=0** (`ok` 6,072 s), com
  `TestApprovalPayloadHashRejectsUnserializableInput` e
  `…RejectsInputChangedByPersistenceRedaction` — *ambos sobre o **hash do payload***.
- **O que **não** está provado**: *se o `mutation_gate` exige mesmo `RequiresApproval: true`
  **para `RiskWrite`*** — *é o que o item pede por nome, e o `-run` apanhou só **2** testes,
  não o conjunto dos dois ficheiros citados.* ***«Os testes existem e passam» e «o teste
  prova o que o item pede» são duas afirmações, e a segunda exige ler o teste.»***
- **E a fronteira do merge**: *o item diz que o PR #62 é **branch-local** e que falta o
  merge* ⇒ **merge não autorizado** (review humano pendente) ⇒ **decisão do dono**.
- **Situação original (história):**
- PR #62 (`afea7e726`) é **branch-local**: `harness.cli` não está em `main`.
  Próximo: merge e prova de que a aprovação obrigatória (`RequiresApproval:
  true`, `Risk: RiskWrite`) é exigida de fato, com teste reproduzível.

### P1-5 · Catálogo honesto de conectores
- 104/122 exigem operador. Promover conectores **um por um**, com evidência de
  chamada real e mantendo `Auth` não-vazio. A metade "promoção automática de
  status" para linhas só-chave é **inviável** (`BLOCKED_BY_EXTERNAL_DEPENDENCY`).

## P2 — engenharia e infraestrutura

### P2-1 · Workflow Engine — **CUMPRIDO (E-827); obsolescência medida**
- **REMEDIDO (E-827, medido por leitura do código e dos testes):** o item pede **três**
  coisas e as três **já existem**.
  1. **Execução real de schedule persistido** ⇒ `schedule_flow.go:570`
     (`// ExecuteScheduleFlow runs a persisted schedule flow…`) e
     `supervisor.go:318` (`// Execute the persisted flow graph of the schedule…`),
     coberto por `schedule_flow_test.go`.
  2. **Desfecho por passo persistido** ⇒ `runtime.go:1906`
     (`// recordScheduleRun persiste o resultado da execução…`) com
     `Schedule.LastRun`, *«tanto em sucesso quanto em [falha]»*; e `schedule_flow.go:543`
     — *«o gravador persiste no schedule para que a **pausa sobreviva ao tick**»*.
  3. **`action.connector` implementado** (`schedule_flow.go:51`, dois `case` em `:338` e
     `:1009`, e `schedule_connector_test.go` com os casos **ausente**/**cruzado**/**quebrado**);
     **`action.http` não existe — e não pode existir**, porque `schedule_flow.go:15`
     declara *«`ScheduleStepKind` is the **closed vocabulary**»* ⇒ é a **recusa fechada**
     que o item admitia, feita **por construção** e não por `if`.
  ⇒ **O que resta é só a nota opcional** (`condition.if` com seletor de `expect`,
  PRs #67/#68) ⇒ **decisão do dono**, não trabalho por fazer.
- **Situação original (história):**
- Provar execução real de schedule persistido (não só validação), persistir
  desfecho por passo, e implementar ou **recusar de forma fechada**
  `action.http`/`action.connector`; opcional `condition.if` com seletor de
  `expect` no editor de fluxo (PRs #67/#68).

### P2-2 · Aviso de depreciação do Node 20 no CI
- `actions/checkout@11d5960a…` e `actions/setup-node@49933ea5…` (v4 com SHA
  fixo) emitem `Node.js 20 is deprecated`. PR dedicada subindo os `uses:`
  fixados por SHA para v5 nos 15 workflows.

### P2-3 · Replay curado do upstream
- Branch `chore/upstream-YYYYMMDD` a partir de `main`; replay linear temático
  sobre `upstream/main` (73 commits, 39 arquivos em conflito, 15 conflitos
  reais). 4 gates obrigatórios + smoke Chromium antes de abrir PR.

### P2-4 · Falhas ambientais de teste (B-09) — **ACTUALIZADO (E-820), não fechado**
- **REMEDIDO (E-820, medido):** o `B-09` no JSON diz *«AMBIENTAL (**MEDIDO E
  CORRIGIDO**, COM PROCEDIMENTO) — CORRIGIDO POR MEDICAÇÃO (`E-202`). A descrição
  anterior ('Host sem Python/sh: 4 testes falham por ambiente') estava…»* ⇒ **o
  problema original foi corrigido**. **Mas marcar este item como resolvido seria
  falso:** este host tem **três** falhas ambientais **distintas**, medidas nesta
  sessão e com contorno conhecido —
  1. **`CGO_ENABLED=0`** derivado pelo próprio Go porque **não existe toolchain C**
     (`E-815`: 9 compiladores e 6 directórios canónicos ausentes) ⇒ **14** falhas
     de stub `go-sqlite3` (*`app/server` é 21 PASS / 13 FAIL por esta causa*, `E-814`);
  2. **`OLLAMA_CONTEXT_LENGTH=32768`** no perfil do utilizador (`HKCU\Environment`,
     `E-809`/`E-810`) ⇒ 1 falha de `context_window`, curável só **antes** do processo
     arrancar (*é variável de pacote, `envconfig/config.go:230`);
  3. **`%TEMP%` não-apagável** (`E-796`) ⇒ falha de `go build` e de
     `vite:esbuild-transpile`; contorno = `TMP`/`TEMP` para dentro do repo.
  ⇒ **O item mudou de conteúdo**: já não é «sanear o host» (o problema do `B-09`),
  é **declarar e contornar as três falhas ambientais deste host** — o que está feito.
- **Situação original (história):**
- Sanear o host de teste (Python + `sh`, caminho curto) **sem** desativar teste.

### P2-5 · Prettier não governado na raiz — **FECHADO (E-820): `FORA_DE_ESCOPO`**
- **REMEDIDO (E-820, medido):** decisão tomada — **declarar formalmente fora de
  escopo**, e não adoptar na raiz. As quatro provas: (a) **não existe**
  `.prettierrc` na raiz (nem `.json`, nem `.js`); (b) o **único** `.prettierrc`
  do repositório é `app/ui/app/.prettierrc`; (c) **0 dos 15** workflows em
  `.github/workflows/` mencionam `prettier`; (d) o script `prettier:check`
  **existe** no `package.json` da UI mas **nenhum workflow o corre**. ⇒ Adoptar na
  raiz seria **criar** governança de formatação num escopo onde nada é governado —
  o §00.2 proíbe trabalho paralelo sem justificativa validada em diff e ADR.
- **Situação original (história):**
- `.prettierrc` existe só em `app/ui/app`; nenhum workflow roda
  `prettier:check`. Decidir: adotar na raiz ou declarar formalmente fora de
  escopo.

## P3 — release verificado e arquitetura

### P3-1 · Release assinado por SO
- Certificado (B-06), smoke nativo por SO, artefatos assinados publicados.

### P3-2 · Isolamento por organização (multitenancy) — **CUMPRIDO (E-828)**
- **REMEDIDO (E-828, medido):** o item pede *«decisão arquitectural + **teste** de que uma
  organização não lê dados de outra»*. **Ambas existem, e a segunda em abundância:**
  `grep` em `internal/agent/*_test.go` devolve **250 de 325** correspondências
  (*o grep cortou no limite; o total é 325*), cobrindo **plugins/conectores**
  (`ErrPluginOrganizationScope`, incluindo *mesmo id em tenants diferentes*), **MCP**
  (local, remoto e server), **fila/jobs** (`ErrQueueJobForbidden`), **CRM/comércio/financeiro**,
  **inbox/omnichannel**, **agendamento** (inclusive *a trava é por organização, não global*),
  **dispositivos/push** (`ErrDeviceForbidden`), **egress**, **memória/retenção**,
  **aprovações**, **builder/deploy**, **media upload**, **consentimento**, **histórico** —
  e **RLS em SQL real** (`distributed_integration_test.go`, com INSERT cross-tenant
  **recusado pelo banco**).
- **A decisão arquitectural tem erros nomeados**: `ErrPluginOrganizationScope`,
  `ErrQueueJobForbidden`, `ErrDeviceForbidden`, `ErrBuilderForbidden` e o guarda
  compartilhado `companyForRequest` (`B-15`, RESOLVIDO); o `hades_tenant_scope_check.py`
  **passa** (7/7 checkers).
- **Três testes de qualidade excepcional**: `service_agent_approval_tenant_test.go:17-23`
  (*descobriu que o teste que alegava isolamento **«era satisfeito pela colisão de caminho»**
  e escreveu um que **não depende** dela*), `scheduling_lock_scope_proof_test.go:82`
  (*prova que a trava não é GLOBAL*) e `oauth_refresh_non_inference_test.go:23`
  (`…DoesNotRevealExistenceAcrossTenants` — *mais forte que «não lê»: **não revela se existe***).
- **E vai além do pedido**: `inbox_isolation_test.go:10` cobre **escrever**
  (*atribuir, transferir, fechar*), não só ler.
- **Situação original (história):**
- Decisão arquitetural + teste de que uma organização não lê dados de outra.

### P3-3 · Benchmarks com metodologia auditável — **CUMPRIDO (E-829)**
- **REMEDIDO (E-829, medido):** o artefacto é **`docs/mission/HADES_BENCHMARK_METHOD.md`**,
  e o cabeçalho diz literalmente o que o item pedia: *«Artefato do **estágio 21**.
  **Substitui**, no que é mensurável localmente, o **comparativo puramente documental**
  que existia antes.»*
- **Os três requisitos, um a um**: **comando** (`§1`, com 4 notas de método);
  **ambiente** (`§2`: Sistema, **CPU i9-13900HX**, **Toolchain `go1.27.2`**, **Ponto de
  partida `main@889750abe`**, Iterações); **resultado** (`§4`, com `ns/op`, `B/op`, `allocs/op`).
- **E vai além do pedido**: `§3` tem coluna **«Não mede»** por benchmark; e `:6-11`
  **recusa** a comparação competitiva (*«sem a MESMA tarefa, a MESMA máquina e a
  licença/conta de cada produto, qualquer número seria **inventado**»*).
- **E a metodologia foi reproduzida**: corri o comando **do documento** ⇒ **exit=0**,
  **296,5 s**, **22** benchmarks (*a suíte cresceu: `OmnichannelIngest*`, `ServiceAgentTurn/Plan`,
  `MemorySearch*`, `WritePhase_*` não estão no `§4`*). Os números são da **mesma ordem**,
  mas **não idênticos** — *e as duas variáveis estão **declaradas** no documento*
  (toolchain e ponto de partida diferentes). **Não atribuo a divergência a nenhuma
  das duas** (seria o erro do `E-824`).
- **E o `METHOD` fecha-se com duas secções que valem por si (`E-830`)**: `§5 «Como
  invalidar este resultado»` — *«Duas medições com o mesmo commit e a mesma máquina devem
  ficar na mesma ordem de grandeza; se não ficarem, o **ambiente** precisa ser registado»*
  — e `§6 «O que ainda falta para um comparativo competitivo»`: tabela com **4** requisitos
  **«não feito»/«pendentes»**, cada um com **bloqueio nomeado** (*licenciar cada produto*,
  **`B-07`**, *PRs #75/#77*), terminando em *«o estágio 21 permanece **`PARCIAL`**, sem
  nenhuma alegação de supremacia»*. ⇒ *e o critério do `§5` foi **accionado** pela minha
  reprodução (460× de divergência no `MemorySearchContext`), e o ambiente foi registado.*
- **Situação original (história):**
- Substituir o comparativo documental (`audit/COMPARATIVO_TYPINGMIND_20261009.md`,
  `docs/RELATORIO_BENCHMARK_COMPETITIVO_HADES_2026-10-04.txt`) por benchmarks
  reproduzíveis com versão, ambiente e comando.

### P3-4 · Estágios 5, 8, 15 e 34–37 — **FECHADO (E-821); premissa corrigida em (E-820)**
- **FECHADO (E-821, medido):** o predicado que o `E-820` deixou nomeado — *existência
  real de código* — está cumprido, e os três **têm implementação e testes**:
  - **5 · Context Bootstrap** ⇒ `internal/agent/context_bootstrap.go`; **7** ficheiros
    `*bootstrap*`, dos quais **3** de teste (`_test`, `_handoff_test`, `_isolation_test`);
    **325** ocorrências em **19** ficheiros.
  - **8 · Plugins e Skills** ⇒ `plugin_runtime.go`, `plugin_status.go`, `plugin_bundle.go`,
    `skill_activation.go`; **29** `*plugin*` + **19** `*skill*` = **48** ficheiros, com
    testes de ciclo de vida (`plugin_lifecycle_test.go`) e diagnóstico
    (`plugin_execution_diagnostics_test.go`).
  - **15 · Memoria e RAG local** ⇒ `rag.go`, `embedding.go`, `context_memory_governance.go`,
    `context_reembed_job.go`; **31** `*memory*` + **8** `*embed*` + **4** `*rag*` = **43**
    ficheiros; **970** ocorrências em **200** ficheiros — *o maior dos três, e coerente
    com o relatório, que já o declarava um dos 2 `COMPLETED_VERIFIED`*.
  ⇒ **Resultado dos 7 estágios: 4 externos por `B-03`/`B-04` e 3 com código e testes.**
  **0 dos 7 exige implementação nova** — o que falta é **classificação** no
  `HADES_STAGE_MAP.md` (*documento gerado: regenera-se, não se edita à mão*).
- **Situação anterior (E-820, história):**
- **REMEDIDO (E-820, medido):** o item diz *«nada foi inspecionado»* como se a
  informação não existisse — **ela existe, noutro documento**. Medido em
  `HADES_MASTER_STATE.json`: o campo `stages` tem **38** entradas com `id`+`title`
  e **`status` ausente**; e o campo `stage_states` tem **apenas 6** entradas,
  **nenhuma** com `status` ⇒ **a classificação dos 38 vive no
  `HADES_STAGE_MAP.md`**, não neste JSON.
- **Os `title` reais, lidos:** **5** = `Context Bootstrap / contexto de projeto` ·
  **8** = `Plugins e Skills` · **15** = `Memoria e RAG local` · **34** = `Canal
  WhatsApp` · **35** = `Inbox unificado omnicanal` · **36** = `Voz e transcricao` ·
  **37** = `Chamadas e follow-up`.
- **34–37 já têm bloqueio externo numerado:** são canais e voz ⇒ mapeiam em
  **`B-03`** («Tokens reais de canais», tipo `EXTERNO`) e **`B-04`** («Provedor de
  telefonia/voz real», tipo `EXTERNO`) ⇒ classificação honesta =
  **`BLOCKED_BY_EXTERNAL_DEPENDENCY`**. *A resposta estava no item **P1-3**, duas
  secções acima.*
- **O que REALMENTE falta (predicado nomeado):** verificar a **existência real de
  código** de **3** dos 7 — **5** (`Context Bootstrap`), **8** (`Plugins e Skills`)
  e **15** (`Memoria e RAG local`, que tem indício de já ser `COMPLETED_VERIFIED`).
  Os 34–37 **não** precisam: são externos por `B-03`/`B-04`.
- **Situação original (história):**
- Auditar existência real (nada foi inspecionado) antes de qualquer
  classificação; depois avançar conforme a dependência externa permitir.

### P0 · Complementos MASTER V2.2 (MEMÓRIA · PAPERCLIP · CLOUDROOM)

Estado medido em 2026-10-14, com evidência no `HADES_MEMORY_CONTEXT_RELIABILITY.md`:

1. **MEMÓRIA / COMPACTAÇÃO UNIVERSAL** — fechado no que é verificável:
   compactação automática (E2E por HTTP), histórico original (append-only + cadeia de hashes),
   recuperação das fontes, memória governada (8 rotas com escopo auditado), §4 handoff
   (contrato → produtores → chat → UI desktop → mobile) e §6 retomada de missões (já existia e
   está testada). Decisões D-1..D-12 registadas. **Correcção registada**: o `memory.search` do
   E-618 era DUPLICADO do `knowledge.search` (E-152) e foi revertido no E-619 — a «lacuna» era
   um defeito de busca, não do produto.
2. **PAPERCLIP — Agent Organization OS** — pendente: matriz de paridade (20 capacidades, com
   referência ao código real dos dois lados) e depois implementação incremental, reutilizando
   Company OS, Mission Control, memória e tarefas existentes. **Não** copiar o backend Node.
3. **CLOUDROOM — Session Teleport** — pendente: matriz de paridade (20 capacidades) e depois
   adaptadores por harness/executor. Diferenciar sempre `Native Resume` / `Checkpoint Restore` /
   `Context Handoff`; nada de segundo runtime.
4. **HADES Collaborative Intelligence — Teamily Parity** (complemento **48 PROPOSTO**; *a numeração
   foi CONFERIDA, não herdada*: `stages` do `HADES_MASTER_STATE.json` = **38** ids `1..38` sem
   lacuna, e as extensões 39–42 / complementos 43–47 vivem em texto, não na lista do estado — logo
   o próximo livre é 48, e a atribuição fica **pendente de confirmação no schema canônico**) —
   **NOVO (2026-10-15)**, recebido como complemento da MASTER V2.2. Objetivo: humanos e múltiplos
   agentes **na mesma sala**, com delegação que executa trabalho real. **Estado medido hoje: NADA
   implementado** (as 15 fases estão `PENDING`; o primeiro passo é a **matriz de paridade com
   evidência**, como manda a FASE 1). ✅ **FASE 1 CONCLUÍDA — auditoria medida no commit `93f12ac19`:** a matriz
   das 20 capacidades está em `docs/mission/HADES_TEAMILY_PARITY_MATRIX.md` (estado por capacidade, evidência com
   **arquivo e linha**, o que já existe para **reusar**, e o que ficou **`A MEDIR`**). *Resultado curto: salas,
   menções, respostas encadeadas, presença com origem e participante-agente **não existem em nenhuma camada**
   (runtime, backend, UI); o que existe e deve ser reusado é o chat com injeção de contexto e compactação, o
   `inbox`, o store de colaboração, o canal externo com DLP, os orçamentos/aprovações e o Companion.* Fases: 0 recuperar contexto/memória · 1 auditoria de paridade ·
   2 Team Chat (salas, menções `@agente`/`@equipe`/`@todos`, anexos, tarefas, presença com origem
   verificável, reconexão) · 3 Social Orchestrator (plano → delegação → paralelismo seguro →
   aprovações) · 4 Living Memory (com os rótulos `FACT_VERIFIED`/`DECISION_APPROVED`/`PROPOSAL`/
   `HYPOTHESIS`/`UNTRUSTED_CONTENT`) · 5 roteamento inteligente na conversa · 6 humanos e agentes no
   mesmo workspace · 7 Collaborative Studios · 8 Companion/assistentes · 9 agentes proativos com
   governança · 10 continuidade entre chats e dispositivos · 11 segurança e privacidade (isolamento
   por organização, anti-injeção, DLP) · 12 UX A+++ · 13 testes e homologação · 14 integração no
   backlog canônico · 15 critérios de conclusão.
   **REGRAS QUE VINCULAM (do próprio complemento):** ① **não** criar segundo chat engine, runtime,
   memory engine, permission engine ou sistema de tarefas — reutilizar Cowork, inbox, eventos,
   memória, Company OS e tarefas existentes; ② **primeiro** resolver os bloqueios do **PR #438** e
   estabilizar a `main`, depois implementar incrementalmente; ③ uma menção só dispara execução com
   **autorização + capacidade + orçamento** — *e o orçamento por passo já existe medido (E-669)*;
   ④ testes simulados têm de ser **declarados como simulados**, e UI responsiva **não** é prova de
   sincronização.

Regra que estes dois blocos herdam do erro do E-619: ***procurar pelos NOMES que o código usa,
com padrões que não excluam o alvo*** — antes de declarar qualquer lacuna.

## Checkpoint E-650/E-651 — reparacao do CI vermelho do PR #438

**Medido em 2026-10-15T01:35Z (2026-10-14T22:35 BRT).** NAO substitui `HADES_MASTER_STATE.json`
(fonte canonica unica); e' o registo de trabalho desta fatia.

### Estado medido (nao narrado)
| item | valor | origem |
| --- | --- | --- |
| `origin/main` | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` | `git rev-parse origin/main` |
| branch | `fix/checker-relogio` | `git rev-parse --abbrev-ref HEAD` |
| head ANTES desta fatia | `3c2edf466` | igual ao head que a auditoria externa registou |
| head DEPOIS | `f461ec86a` (E-650, E-651, E-650-ui) | `git rev-parse HEAD` |
| PR #438 | aberto, **nao mergeado**, `mergeable_state=blocked` | revisao humana pendente — **nao contornada** |
| checks do head `3c2edf466` | **6 falhas, 17 verdes** | `get_check_runs` |
| checks do head `e432a9091` | as 6 falhas caem para **1** (so' o E2E) | `get_check_runs` |

### As 6 falhas, com causa raiz medida por log (nao por suposicao)
| check | causa raiz | correccao | prova |
| --- | --- | --- | --- |
| `Preserve Class A+ surfaces` (12 s) | `verify-contracts.mjs`: `DELETE /api/agent/v1/skills/:id` **NO MATCH** — a rota foi APAGADA no E-642, quando usei a linha como ANCORA de edicao e nao a repus | rota reposta + comentario do incidente | *«CONTRACT VERIFICATION PASSED»*, exit 0 |
| `Go agentic and server gates` | mesmo `verify-contracts.mjs` | idem | idem |
| `PR gate` | agregava os anteriores | idem | verde no CI |
| `test (ubuntu-latest)` | `TestLembretesNaSuperficie`: `time.Now().UTC().Add(2h)` + servico de 30 min dentro de turno `00:00-24:00` ⇒ as 23:56 recusa **com razao** ⇒ ***a cor do CI dependia da hora*** | proximo meio-dia UTC (sempre < 24 h e com folga na janela) | reproduzido as 21:56 BRT e depois `--- PASS` + `go test ./server/` ok (29,7 s) |
| `test (macos-latest)` | mesma causa | idem | verde no CI |
| `Web E2E (shell smoke)` | **`PerformanceCard` derrubava a PAGINA**: o stub do E2E responde `{}` a rotas nao tratadas ⇒ `dados.performance` indefinido ⇒ `TypeError` ⇒ error boundary | guarda + mensagem honesta + teste de regressao com payload mau | `npx playwright test e2e/first-run-journey.spec.ts` = **1 passed (28,2 s)** (era `1 failed`) |

### A causa de eu nao os ter apanhado (a parte reutilizavel)
A bateria local tinha **16** verificadores e **nao continha o gate do CI**: `verify-contracts.mjs`
e' **Node**, nao Python. Passa a existir `_tmp/bateria_local.ps1`, **espelho dos passos do job**
`Preserve Class A+ surfaces`: **1 node + 27 python = 28/28 TUDO VERDE**. *Bateria que nao contem o
gate do CI nao substitui o CI* (mesma classe do E-636).

### Correccao de um numero que EU declarei mal
A mensagem do commit `146350696` (E-651) diz *«633 passed (633)»*; o **medido foi 630**. O numero
errado fica na historia (commit empurrado) e o correcto e' este: **630**. Licao: nao escrever
numeros de memoria numa mensagem de commit — colar o que a ferramenta mediu.

### Proximo passo e comando de retomada segura
1. Confirmar o CI do head final (`f461ec86a`): os 6 checks que estavam vermelhos.
2. Correr a **suite E2E inteira** (`npx playwright test`), nao so' o spec da jornada.
3. Actualizar `HADES_MASTER_STATE.json` (valores medidos desta fatia), sem criar fonte concorrente.
4. `PR #438` continua a exigir **revisao humana** (`LMPrado-DZ23`): nao fazer merge, nao contornar.

```bash
# retomada segura
cd hades-ollama && git fetch origin && git status --porcelain=v1
git rev-parse HEAD origin/fix/checker-relogio origin/main
powershell -NoProfile -ExecutionPolicy Bypass -File _tmp/bateria_local.ps1   # 28 gates do CI
```

---

## ACTUALIZACAO V3.0 (RONDA 71) - O QUE MUDOU DESDE O `E-959`

> **Requisito**: o `mission_check` exige que os documentos de retomada citem evidencia
> **recente** (>= `E-960`). Este documento parava no `E-959` - *41 evidencias de atraso*.
> **Isto e' o §FASE-7 e o ESTAGIO 1** (*«reconciliar documentos canonicos e historicos»*).

### Estado medido nesta ronda
```
branch  fix/checker-relogio     HEAD  31be202d6     origin/main  4b3e21854
WIP     14 entradas             stashes 0          indice  793 entradas (E-001..E-1000)
CI      os 6 workflows VERDES no d1f4e7915 (test, pr-gate, dz23-e2e, dz23-agentic-quality,
        dz23-multi-provider, class-a-plus-integrity com os 9 passos)
bateria 16 OK / 0 FAIL
```

### Evidencias novas (E-960 .. E-1000), por tema

**Adendo do ecossistema de plugins/skills/app/MCP**
- `E-960` a `E-967` - as **5 fontes** do adendo (*agent-skills, hyperframes, googleworkspace-cli,
  apify-mcp-server, playwright*), a matriz de encaixe e a fatia vertical do `POST /plugin`.
- `E-972` - o **formato aberto `hades-plugin.format/v1`** e a rota de formato.

**Fechamento dos 12 findings do PR #438**
- `E-961`, `E-962` - os **5 P1 do checker do relogio** e os **12 findings** revalidados.
- `E-963` a `E-965` - as **5 fontes** conferidas e a correccao factual do `E-959`.

**§ESTAGIO 22 - o gate de engenharia (3 dos 4 itens)**
- `E-987` - o gate deixa de ser **espelho do `MASTER_STATE`** e passa a provar o **snapshot real**
  (*`gate_head`, `gate_branch`, `gate_dirty`, `gate_snapshot`*).
- `E-988` - o **MANIFESTO do snapshot**: SBOM com `sha256`, plataformas, limitacoes, `source_sha`.
- `E-989` - **catraca** do manifesto no CI e na bateria local.
- `E-993` - o gate estava **impossivel de satisfazer no CI**: eram **4 volateis**, nao 1.
- `E-994` - o **mesmo defeito no manifesto** (`source_sha`).
- `E-996` - o **terceiro instrumento** (`host`) e a **INVERSAO DA REGRA**: so' se compara o que o
  manifesto **PROVA**, e cada campo tem de ser **classificado**.
- `E-997` - o inventario **media o disco**; passa a ler o **commit** e a comparar **estrutura**.

**§FASE-5 - marca canonica**
- `E-990` - **inventario da marca**: **909** canonicas contra **441** legadas em 4 nomes.
- `E-991` - o **gerador** do inventario com **8 excepcoes** declaradas (*razao + desbloqueio*).
- `E-997` - as contagens passam a vir do commit, e nao do disco.

**Instrumentos e rastreabilidade**
- `E-992` - **catraca do ASCII nos `.ps1`**: 34 caracteres nao-ASCII em **string** no `install.ps1`.
- `E-995` - o **`E-559` restaurado do Git**, engolido pelo `E-993`.
- `E-998` - a **quinta instancia** do defeito (*o checker media o disco e passava por sorte*).
- `E-999` - o **`\n` LITERAL** do here-string duplicou o `E-559` e engoliu o `E-560`.
- `E-1000` - **evidencia que citava codigo NAO COMMITADO**: o `E-998` descrevia uma cura que
  ficou **fora** do commit; curado no `31be202d6` (*1 file changed, 113 insertions*).

### As QUATRO falhas do CI, com o run que provou cada cura
```
run 38922978710  passo 6 (gate)        gate_head/gate_branch volateis  -> E-993  bddeb7ebd
run 38924032504  passo 7 (manifesto)   source_sha/source_branch        -> E-994  a4ddaa4e0
run 38926745194  passo 7 (host)        host Windows vs Linux           -> E-996  c025eee59
run 38932562362  passo 8 (inventario)  media o DISCO                  -> E-997  ecd8bb31b + d1f4e7915
run 38935930565  *** 6/6 SUCCESS ***
```

### Licoes medidas (nao supostas)
1. **Curar o que o CI aponta e' perseguir sintomas** - 3 instrumentos, e em todos curei o que ele
   apontou e deixei o resto. Dai' a **INVERSAO**: allowlist do que e' **estavel**, volatil por
   omissao, **+ classificacao obrigatoria**.
2. **Nao ler git nao impede medir WIP** - a segunda pergunta: *o ficheiro que eu leio esta'
   commitado?*
3. **O `HEAD` do CI nao e' a branch** - em `pull_request` e' `refs/pull/N/merge`. Mesma pergunta
   em 3 lugares.
4. **Performance tambem e' correccao**: 2459 `git show` -> **11**, de **minutos** para **1,8 s**.
5. **Uma evidencia sem commit e' uma afirmacao** (`E-1000`).

### Divida declarada (nao escondida)
```
5a instancia de "mede o disco": os 3 DIVIDA_STRING e os 4 tectos de comentario do
  hades_ascii_ps1_check.py medem o DISCO, e o CI mede o checkout -> declarada, NAO corrigida
install.ps1: 34 caracteres nao-ASCII em STRING -> fecha com o FASE-6 (teste de instalacao limpa)
duplicacao intra-linha: o mission_check conta LINHAS e nao ve' uma copia embutida -> declarada
```

### Retomada segura (comandos que existem neste repositorio)
```bash
cd hades-ollama && git fetch origin && git status --porcelain=v1
git rev-parse HEAD origin/fix/checker-relogio origin/main
python scripts/hades_release_gate.py --check          # o gate bate com o estado da missao
python scripts/hades_mission_check.py                 # escrituracao: sem fantasma, veredito coerente
python -X utf8 scripts/hades_manifest.py --check      # o manifesto bate com a arvore
python -X utf8 scripts/hades_brand_inventory.py --check
python -X utf8 scripts/hades_ascii_ps1_check.py       # ASCII nos .ps1
```
*Os caminhos em `_tmp/` **nao** vao para o Git e **nao** sao instrucoes de retomada validas --
a bateria canonica e' a lista acima, mais `scripts/hades_local_gates.ps1`.*

---

## ACTUALIZACAO V3.0 (RONDA 86) - O QUE MUDOU DESDE O `E-1000`

> **Requisito**: a catraca `hades_handoff_window_check.py` (`E-1008`) mede a **janela** de
> citacao -- *o maior ID citado tem de estar a menos de **12** evidencias da cabeca do indice*.
> Este documento parava no `E-1000`, e a cabeca estava em `E-1008`: **8 de atraso**, medidos.
> *A correccao fecha o `atras` para **0**.*

### As 8 evidencias novas, e o que cada uma fechou
``````
E-1001  os 3 documentos de retomada paravam no E-959 (42 de atraso)   aec62f0eb
E-1002  a divida declarada A MAIS (a 5a instancia ja' estava curada)  aec62f0eb
E-1003  o E-1002 PARTIDO em 2 linhas - e NENHUM checker viu          2e3de0dc2
E-1004  o normalizar so' apanhava METADE das tabelas de contagem      55d69ea5d
E-1005  as 5 strings do FASE-5 + o marcador que mutilava `DZ23`       a10e52cdc
E-1006  o E-1005 partiu o passo 8 -> comparar CONJUNTOS, nao listas   b1f1f0d86
E-1007  a catraca da FORMA do indice (5 regras, fault injection)      dbadd69b6
E-1008  a JANELA de citacao dos documentos de retomada (janela 12)    db1ee1fae
``````

### Estado medido nesta ronda
``````
HEAD db1ee1fae · branch fix/checker-relogio · main 4b3e21854 · WIP 13
indice 801 entradas (E-001..E-1008) · bateria 19 OK / 0 FAIL
CI: b1f1f0d86 e dbadd69b6 sem failure · db1ee1fae a correr com os 3 passos novos
    (inventario, FORMA do indice, JANELA de citacao)
PR #438: OPEN · mergeState=BLOCKED · reviewDecision VAZIO (FALTA REVISAO HUMANA)
``````

### As 6 causas de CI vermelho, todas curadas e provadas
``````
run 38922978710  passo 6  gate_head/gate_branch volateis      -> E-993
run 38924032504  passo 7  source_sha/source_branch            -> E-994
run 38926745194  passo 7  host Windows vs Linux               -> E-996
run 38932562362  passo 8  inventario media o DISCO            -> E-997
run 38939790452  passo 8  normalizar so' via 2 colunas        -> E-1004
run 38941538186  passo 8  o E-1005 partiu-o (tabela aberta)   -> E-1006
``````

### Licoes medidas nesta janela
1. Um checker que conta LINHAS nao ve' uma linha que NAO E' LINHA (`E-1007`).
2. Um risco DECLARADO e' uma medicao a fazer -- e as 2 declaracoes cumpriram-se (`E-1006`).
3. Uma cura pode ter o MESMO defeito que cura (`E-1005`).
4. Sobrestimar tambem e' defeito (`E-1002`).
5. Um limiar que mede «SE HA» nao mede «QUAO RECENTE» (`E-1008`).
6. Um detector de texto nao distingue uma MENCAO de uma OCORRENCIA (`E-1007`, `E-1008`).
7. Uma catraca sem teste e' uma crenca -- o teste do `E-1007` deu 3 defeitos da propria.

### Bloqueios externos reais
``````
race / go test -race   CGO_ENABLED=0, mingw-w64 ausente
pacotes / instalador   FASE-6 nao iniciada
revisao humana         reviewDecision VAZIO no PR #438
``````

### O que falta (actualizado na janela `E-1009`..`E-1021`)

**Fechado nesta janela**: instalador Windows (`E-1012`), pacote Linux (`E-1013`), inventario de
marca (`E-1015`), MV3 provado inexistente antes (`E-1016`/`E-1017`), Android medido ate' a
fronteira (`E-1019`), extensao MV3 criada (`E-1020`) e **completada e carregavel** (`E-1021`),
`.gitignore` de material de assinatura (`E-1020`, commit `82228563f`).

**Continua em falta, sem disfarce**:
- `.zip` da extensao com hash e instrucoes -- **proximo passo**;
- **pairing real** (`native messaging` + `HADES Local Companion`): **nao existe**; o popup mostra
  `host: - (native host nao instalado)`;
- ligacao da extensao ao **Mission Control** (hoje `missao` e `aprovado` ficam a `null`);
- E2E em Chrome/Edge reais; Firefox e Safari **nao** tem suporte demonstrado;
- APK Android: bloqueado pela DLL nativa JNI neste host (`E-1019`), passo de desbloqueio = runner Linux;
- macOS e iOS: `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial -- o material Apple
  fornecido (3 `AuthKey` + 1 `.cer`) serve para **submeter/notarizar**, o `.cer` sozinho **nao
  assina**, e `codesign`/`notarytool` **so' existem no macOS**;
- script + doc de notarizacao/assinatura: **pedido pelo dono, ainda nao escrito**;
- iOS sem build de simulador; dispositivo fisico e TestFlight `BLOCKED_EXTERNAL`;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado; `SUPERIOR_MEASURED` nao reclamado.

### O que falta (actualizado na janela `E-1022`)

**Fechado nesta janela**: o pacote nao assinado da extensao -- `.zip` + `SHA256SUMS` + `LEIA-ME`
ASCII + verificador versionado, reprodutivel e provado contra 3 defeitos reais (`E-1022`). O
defeito do separador foi medido, curado e **provado por injecao nos bytes crus**.

**Continua em falta, sem disfarce**:
- **pairing**: `native messaging` + `HADES Local Companion` **nao existem**; o popup mostra
  `host: - (native host nao instalado)` e `missao`/`aprovado` ficam a `null`;
- ligacao da extensao ao **Mission Control** (nada alimenta os campos de estado);
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nao carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**;
- **SBOM do pacote da extensao** e actualizacao do `THIRD_PARTY_NOTICES` com os icones
  reutilizados de `app/ui/app/public/`;
- medicao de latencia/CPU/RAM do `observar()` e backpressure medido;
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- script e doc de notarizacao/assinatura (pedido pelo dono);
- benchmark reproduzivel (ESTAGIO 21) nao iniciado; `SUPERIOR_MEASURED` nao reclamado.

### O que falta (actualizado na janela `E-1023`)

**Fechado nesta janela**: o contrato do canal local (`E-1023`) -- schema, sequencia, expiracao,
replay, pairing por codigo curto, chave por dispositivo, rotacao, revogacao e recusa de LAN.
23 assercoes, e 2 delas apanharam defeitos reais de codigo.

**Continua em falta, sem disfarce**:
- **acordo de chaves**: o contrato usa HMAC **simetrico** e **assume** que a chave foi
  partilhada com seguranca -- a derivacao (HKDF a partir do codigo curto) **nao existe**;
- **transporte**: nao ha native messaging, nao ha socket, nao ha `HADES Local Companion`;
  o popup continua a mostrar `host: - (native host nao instalado)`;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**;
- **SBOM do pacote da extensao** e `THIRD_PARTY_NOTICES` com os icones reutilizados;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1024`)

**Fechado nesta janela**: a derivacao de chave por HKDF com contexto e codigo de uso unico,
o manifesto template do native host com allowlist, e a permissao `nativeMessaging` movida
para `optional_permissions` (`E-1024`). O `mission_check` apanhou o `E-1024` citado em
codigo antes de existir -- a mesma classe do `E-1020`, por um vector que a catraca do
indice nao cobre.

**Continua em falta, sem disfarce**:
- **o `HADES Local Companion` nao existe**: o manifesto e template com placeholders, e nao
  ha executavel, transporte, stdio pipe nem handshake;
- **o codigo curto em trafego**: num canal `loopback` sem TLS ele viaja em claro no
  `localhost`; falta o handshake que prova posse da chave derivada e o rate-limit;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**;
- **SBOM do pacote da extensao** e `THIRD_PARTY_NOTICES` com os icones reutilizados;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1026`)

**Fechado**: o handshake de prova de posse com desafio de uso unico e prazo, a taxa por instalacao
com janela expiravel, e a canonizacao propria do desafio (`E-1025`). E **tres defeitos reais**
curados no mesmo sitio (`E-1026`): `verificar` chamado com 3 argumentos numa funcao de 2 (a guarda
de prova morria sempre), `registro.revogadaEm` (campo que nunca existiu -- a guarda de revogacao
estava MORTA), e `>` em vez de `>=` no limite de expiracao.

**Continua em falta, sem disfarce**:
- **o TRANSPORTE**: o handshake prova identidade, nao sigilo. O codigo curto viaja em claro no
  `localhost` no primeiro salto; nao ha stdio pipe nem TLS de loopback;
- **o `HADES Local Companion` nao existe** (o manifesto e' template com placeholders);
- **a taxa vive em memoria**: reiniciar o host apaga a contagem; sem rate-limit do lado do SO;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**; sem SBOM do pacote;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1028`)

**Fechado**: o transporte (`E-1027`) -- enquadramento de native messaging (4 bytes little-endian)
com erros tipados, corpo parcial tratado como parcial, limites de comprimento zero e gigante
antes de alocar, e um canal em memoria que prova o protocolo completo pelo fio, inclusive partido
em pedacos de 1 byte. E **tres premissas minhas erradas** curadas no mesmo teste (`E-1028`):
contagem de bytes a mao (*duas vezes*), exigir deteccao de lixo no inicio (*impossivel num
enquadramento de comprimento*) e um nome de motivo inventado.

**Continua em falta, sem disfarce**:
- **o processo Companion nao existe**: o enquadramento e' testado em memoria e **nunca** correu
  contra `process.stdin`/`stdout` de um processo real;
- **`allowed_origins` e' template**: nao ha' host instalado nem origem validada em execucao;
- **sem TLS de loopback** -- o codigo curto continua a viajar em claro no primeiro salto;
- **a taxa vive em memoria**: reiniciar o host apaga a contagem; sem rate-limit do lado do SO;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**; sem SBOM do pacote;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1030`)

**Fechado**: o `HADES Local Companion` **como processo** (`E-1029`) -- le' do `stdin`, responde no
`stdout`, implementa so' o pairing e o handshake, encerra com codigo 2 quando o fluxo dessincroniza, e
mantem o diagnostico no `stderr`. E o defeito de protocolo que so' um processo real expos (`E-1030`):
**o host respondia sem ecoar o `id`**, e um pedido pendente para sempre aparece como **silencio**, nao
como erro. Tres sondas passaram antes de a quarta o revelar -- nenhuma correlacionava.

**Continua em falta, sem disfarce**:
- **o host nao esta instalado**: `allowed_origins` e' template e nao ha' registo no navegador nem
  prova de que o SO respeita a fronteira;
- **o estado vive em memoria**: taxa, pairing e dispositivos desaparecem ao reiniciar o processo;
- **sem TLS de loopback** -- o codigo curto continua a viajar em claro no primeiro salto;
- **sem rate-limit do lado do SO**;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**; sem SBOM do pacote;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1031`)

**Fechado**: o `allowed_origins` **deixou de ser placeholder** (`E-1031`). O ID e' derivado da
chave publica (`sha256` do DER, hex, `0-f` -> `a-p`), o `manifest.json` tem `key`, o host
instala-se no `HKCU` para Chrome e Edge com manifesto em `%LOCALAPPDATA%`, e o `-Desinstalar`
deixa **zero** registo sujo -- medido. 25 assercoes, com 3 fault injections que provam a catraca.

**Continua em falta, sem disfarce**:
- **o host nao foi exercitado por um navegador**: instalei-o e desinstalei-o, mas **nao abri o
  Chrome**. O E2E real (carregar a extensao, ver o ID, falar por native messaging) e' o proximo
  passo e e' o que separa «protocolo correto» de «produto instalado»;
- **Firefox**: nao usa registo do Windows; o `.xpi` continua por fazer;
- **Safari**: alvo separado, exige macOS/Xcode e projeto proprio;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuicao esta' fora do Git** (`_tmp/ext-chave-privada.pem`): sem ela
  nao se empacota um `.crx` com o MESMO ID; quem for empacotar precisa dela ou de outra chave;
- **taxa e pairing em memoria** -- reiniciar o processo do Companion apaga-os;
- sem TLS de loopback, sem rate-limit do SO, sem DNS rebinding a nivel de socket;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1032`)

**Fechado**: o E2E real num navegador. O Edge carrega a extensao, o ID bate com o derivado, e a
extensao fala com o Companion por native messaging -- medido, com desinstalacao sem residuo.
Quatro defeitos reais corrigidos: a permissao nunca pedida, o `permissions.request` que trava,
o gesto que nao sobrevive ao `sendMessage`, e o `path` a apontar para `.mjs`.

**Continua em falta, sem disfarce**:
- **controle de aba ponta a ponta**: o canal funciona; observar DOM, clicar, preencher e
  **recusar campo de senha** numa pagina real ainda **nao** foi provado;
- **takeover e revogacao por evento** nao testados com uma pagina a serio;
- **o Chrome real do utilizador nao foi exercitado**: a versao 155 deste host **nao carrega**
  por `--load-extension` (medido: 0 service workers). Exige carregamento manual;
- **Firefox** (nao usa registo do Windows; falta o `.xpi`) e **Safari** (macOS/Xcode);
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuicao esta' fora do Git** (`_tmp/ext-chave-privada.pem`);
- **`CAPACIDADES` VAZIO**: o Companion **nao** controla o PC. Computer use nao existe;
- **taxa e pairing em memoria** -- reiniciar o Companion apaga-os;
- sem TLS de loopback, sem rate-limit do SO, sem DNS rebinding a nivel de socket;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1033`)

**Fechado**: o controle de aba ponta a ponta. Observar o DOM real, escrever num campo
legítimo com o valor **verificado no DOM**, **recusar senha e cartão** com o DOM a ficar
vazio, badge `ON`, revogar e recusar depois. E o opt-in de rede privada deixou de ser
código morto: passou a ser decisão do utilizador, visível no popup, desligado por omissão.

**Continua em falta, sem disfarce**:
- **takeover humano**: o scope `browser:takeover` existe e **não** foi exercitado;
- **revogação por mudança de origem**: tem código e **não** tem E2E;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados (o `preencher` recusa `type=file`, sem E2E);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension` (medido); exige carregamento manual;
- **Firefox** (não usa registo do Windows; falta o `.xpi`) e **Safari** (macOS/Xcode);
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git** (`_tmp/ext-chave-privada.pem`);
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC. Computer use não existe;
- **taxa e pairing em memória**;
- sem TLS de loopback, sem rate-limit do SO, sem DNS rebinding a nível de socket;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1034`)

**Fechado**: o takeover humano. O content script observa `input`/`keydown`/`pointerdown`,
marca o takeover por 30s, e o agente é **recusado** com `takeover_humano_ativo` enquanto
durar -- com o **valor do humano preservado** e o autor (`human`/`agent`) registado. E o
popup dá ao utilizador o botão para assumir ou devolver.

Dois defeitos reais corrigidos no caminho: a **escrita do agente accionava o takeover**
(flag `escrevendoPeloAgente`), e o **`lerEstado` apagava o campo novo** (reconstruía o
objecto com lista fixa).

**Continua em falta, sem disfarce**:
- **revogação por mudança de origem**: tem código no `tabs.onUpdated` e **não** tem E2E;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos) e não detecta actividade fora do DOM;
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** (não usa registo do Windows; falta o `.xpi`) e **Safari** (macOS/Xcode);
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- sem TLS de loopback, sem rate-limit do SO, sem DNS rebinding a nível de socket;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1035`)

**Fechado**: a revogação por mudança de origem, agora com E2E. Duas abas na mesma origem
coexistem; a que navega para outra origem perde a concessão e o badge; a vizinha continua;
e voltar não restaura. E o **terceiro** local do defeito do opt-in (`tratarAccao`) foi
curado.

**Continua em falta, sem disfarce**:
- **uma catraca sobre o CÓDIGO**: o defeito do opt-in apareceu em **três** locais
  sucessivos; a cura pontual não fecha a classe. Falta um teste que falhe quando um novo
  chamador validar URL sem o opt-in;
- **`pushState`/`replaceState`**: um SPA que mude de origem por History API **não** dispara
  `tabs.onUpdated` com URL novo -- **não testado**, e pode ser um buraco real;
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1036`)

**Fechado**: o defeito do opt-in deixou de ser uma classe aberta. A catraca
`hades_bridge_optin_ratchet.py` mede o **código** -- falha se um novo chamador de
`validarUrl`/`guardarRota` não passar `permitirPrivado` -- e 13 asserções provam que ela
**discrimina** nos dois sentidos, numa **cópia** (o repositório real nunca é tocado).

**Continua em falta, sem disfarce**:
- **`pushState`/`replaceState`**: pode ser buraco real -- um SPA que mude de origem por
  History API **não** dispara `tabs.onUpdated` com URL novo. **Não medido**;
- **a catraca olha duas linhas**: uma chamada com o opt-in mais longe seria falso positivo;
  e `permitirPrivado: false` literal passaria (mede a menção, não o valor);
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1037`)

**Fechado**: dois defeitos reais que a dívida do `E-1035` escondia -- o evento `onUpdated`
**sem URL** (que revogava a concessão a cada reload) e o **badge que ficava vazio** com a
concessão viva. E o `pushState` foi medido: **não pode mudar de origem** (o browser recusa
com `SecurityError`), logo não era buraco.

**Continua em falta, sem disfarce**:
- **BFCache** (voltar atrás): **não testado**, e pode repor uma página de outra origem
  sem disparar `onUpdated` com URL novo;
- `pushState` **dentro de iframes**, `document.domain`, service workers de página: não testados;
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1038`)

**Fechado**: a família `onUpdated`. A ida revoga (com ou sem BFCache), o regresso **traz
URL** e não revive nada, o badge não ressuscita, e a acção segue recusada. E há uma
declaração explícita: **o BFCache em si não chegou a correr** (`persisted: false`).

**Continua em falta, sem disfarce**:
- **BFCache sem execução observada**: o Edge recarregou em vez de restaurar; forçá-lo
  exigiria ausência de `unload`, `Cache-Control` permissivo e uma decisão do browser;
- **frames e shadow DOM**: o contrato exige suportá-los «com limites» e o `observar`
  actual só olha o documento principal -- **não medido**;
- `pushState` **dentro de iframes**, `document.domain`, service workers de página: não testados;
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1039`)

**Fechado**: frames e shadow DOM. O `observar` já não devolve uma lista parcial como se
fosse completa — **atravessa** shadow roots abertos e iframes de mesma origem (2 → 5 nós)
e **declara** os limites com contagens e profundidades. E o shadow fechado é declarado
**indetectável**, em vez de contado por palpite.

**Continua em falta, sem disfarce**:
- **downloads e uploads**: nada gere downloads, e o `preencher` recusa `type=file` --
  o contrato exige caminhos autorizados, manifesto com hash e aprovação -- **não implementado**;
- **shadow roots fechados**: invisíveis por design do browser, não por defeito do HADES;
- **cross-origin**: contado, nunca lido;
- `document.domain`, iframes `srcdoc`/`about:blank` e service workers de página: não testados;
- **BFCache** sem execução observada (`E-1038`, `persisted: false`);
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1040`)

**Fechado**: o scope `browser:files` decide. Política com 9 regras ordenadas, 17 asserções
unitárias, E2E real no Edge, registo auditável sem o hash, e a **cura na raiz** da lista
fixa do `lerEstado` (que já tinha mordido no `E-1037`).

**Continua em falta, sem disfarce**:
- **a extensão NÃO executa a transferência**: decide sobre ela. Não há hook em
  `chrome.downloads` nem em `type=file`;
- **a aprovação chega por `mensagem.aprovado`** -- um campo que **quem pede** controla.
  *Quem pede não certifica o que pede.* A ligação às approvals do Mission Control
  **não existe**;
- **`sha256` é medido onde os bytes existem**: a extensão **não pode verificar** que os
  bytes transferidos correspondem ao manifesto -- só a decisão é auditável;
- o registo vive em `storage.session` (morre com a sessão) e **não é assinado**;
- **shadow roots fechados**: invisíveis por design do browser;
- **cross-origin**: contado, nunca lido;
- `document.domain`, iframes `srcdoc`/`about:blank`, service workers de página: não testados;
- **BFCache** sem execução observada (`E-1038`, `persisted: false`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1041`)

**Fechado**: a aprovação deixou de ser um booleano. Prova com **id**, **impressão do
pedido** e **autor**; um booleano solto é recusado com o nome certo; a prova cruzada é
recusada; a autoria fica no registo. E a forma adoptada é a que o backend **já tinha**
(`FindApproved` + `Consume`), não um sistema novo.

**Continua em falta, sem disfarce**:
- **a extensão NÃO verifica a prova contra o store** -- não tem acesso. Uma prova **bem
  formada mas falsa passaria**. *A forma está implementada e testada; a autoridade não*;
- **falta ligar a rota de approvals no runtime**: o `nativeMessaging` funciona (`E-1032`),
  mas o Companion **não consulta** `FindApproved`;
- **a extensão não executa a transferência** -- decide sobre ela;
- **`sha256` é medido onde os bytes existem**: a extensão não pode verificar que os bytes
  transferidos correspondem ao manifesto;
- o registo vive em `storage.session` (morre com a sessão) e **não é assinado**;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1042`)

**Fechado**: a decisão de ficheiros passa a **declarar a autoridade**. `autoridade:
forma_apenas` acompanha todo o veredicto (mesmo os `permitido`), a extensão **nunca**
diz `verificada`, e `avisoDeAutoridade` diz o limite **por palavras** — visível no popup
e guardado no registo.

**A fronteira medida**: a rota `serviceAgentApprovals` existe mas é de **sessão**; o
Companion é um processo local **sem sessão**. Dar-lhe a sessão do utilizador seria pior.

**Continua em falta, sem disfarce**:
- **a extensão continua SEM verificar a prova** -- o campo `autoridade` **declara** o
  limite, não o resolve. Uma prova bem formada mas inexistente passaria;
- **falta a rota no Companion**: é preciso que ele use o **canal emparelhado** para se
  autenticar perante o runtime e consultar `FindApproved`. Só então `verificada` tem
  quem o possa afirmar;
- **a extensão não executa a transferência** -- decide sobre ela;
- **`sha256` é medido onde os bytes existem**: a extensão não pode verificar que os bytes
  transferidos correspondem ao manifesto;
- o registo vive em `storage.session` (morre com a sessão) e **não é assinado**;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1043`)

**Fechado**: a pergunta existe. O Companion trata `aprovacao:consultar`, valida o par e o
fingerprint, e responde **sempre** — com `verificada: false` e o motivo constante. A
extensão pergunta e **não depende da resposta**: um canal em baixo nunca faz a decisão
parecer melhor.

**Continua em falta, sem disfarce**:
- **a verificação continua a NÃO acontecer** -- o que existe é a pergunta e a recusa
  honesta. O encaixe é `MOTIVO_SEM_ROTA`, **uma constante**;
- **não há teste que falhe no dia em que o runtime expuser a rota** -- o passo de
  desbloqueio está **escrito, não vigiado**;
- **a extensão não executa a transferência** -- decide sobre ela;
- **`sha256` é medido onde os bytes existem**: a extensão não pode verificar que os bytes
  transferidos correspondem ao manifesto;
- o registo vive em `storage.session` (morre com a sessão) e **não é assinado**;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1044`)

**Fechado**: o encaixe do `E-1043` deixou de ser uma frase. Há uma catraca
(`hades_approval_hook_ratchet.py`) que acorda quando o runtime expuser a rota, e 17
asserções provam que ela **discrimina** — com 4 injeções e uma contraprova, numa cópia.

**Continua em falta, sem disfarce**:
- **o encaixe continua por ligar**: a autoridade da extensão continua `forma_apenas`;
- **a catraca vigia SINAIS**, não a existência real de um canal: um canal feito por outra
  via (socket, ficheiro, segundo binário) **não a faria acordar**;
- **a lista de formas de sessão tem sete entradas e não é exaustiva por construção** --
  uma forma nova daria falso positivo (o modo seguro de falhar, mas ainda assim falso);
- **os 4 findings P2 do PR #438** continuam abertos;
- **a extensão não executa a transferência** -- decide sobre ela;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1045`)

**Fechado**: os 12 findings do PR #438 foram medidos contra o HEAD. **Dez estão corrigidos
ou são falsos positivos do bot.** E **nenhuma thread do GitHub foi tocada** — proibido sem
autorização.

**Continua em falta, sem disfarce**:
- **#8 não totalmente revalidado**: `declared_categories_measured_test.go` foi inspeccionado
  por **nome de teste**, não pela alegação contra o **comportamento real** do checker;
- **a medição do #11 usa o `marked` e um cabeçalho de 5 colunas** -- outro renderizador
  (GitHub, Pandoc) podia divergir em casos de borda; **não testei no GitHub real**;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (autoridade `forma_apenas`);
- **a extensão não executa a transferência** -- decide sobre ela;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1046`)

**Fechado**: os 12 findings do PR #438. Oito corrigidos (`E-909`, `E-910`, `E-912`), dois
falsos positivos do bot (#10, #11), um tratado antes (#1), e o #12 feito por construção.
O #8 foi fechado com a contraprova que faltava: **o teste a correr** (`ok … 5.620s`).

**Continua em falta, sem disfarce**:
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido sem autorização);
- **ESTÁGIO 21 (benchmarks) não iniciado** -- nenhuma medição warm/cold, p50/p95, repetições;
- **o encaixe da aprovação continua por ligar** (`E-1044`); a extensão decide sem autoridade;
- **a extensão não executa a transferência** -- decide sobre ela;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1047`)

**Fechado**: o ESTÁGIO 21 foi **medido**. A matriz T2 real correu com dois modelos locais
(`PASS`, 599,88 s), os veredictos repetiram-se um dia depois, e o `cmd/bench` mediu o
Ollama directo (`load` 19,95 s · `ttft` 20,00 s · `prefill` 1445 tok/s · `generate`
92,17 tok/s).

**Continua em falta, sem disfarce**:
- **a repetição (`HADES_T2_REPETICAO.md`, `46+/0`) não foi corrida nesta janela** -- o
  número é do WIP e **está por revalidar**;
- **nenhuma comparação externa foi executada** -- `NAO_EXECUTADA`, não `PARITY_MEASURED`;
- **a taxa é 3/8** com modelos locais de 3–7B: *um modelo, um host, uma quantização*;
- **sem GPU/VRAM medidos por época** -- só tempo de parede;
- **o warm/cold do `cmd/bench` não existe** (frio por construção, `bench.go:504`);
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1048`)

**Fechado**: o inventário do worktree. **Zero trabalho real por integrar** — os 11 `M`
são só fim-de-linha (`--ignore-cr-at-eol` = 0). O WIP do prompt existe em disco e foi
integrado na FASE 1. Há agora um instrumento que **lista** (passo 31).

**Continua em falta, sem disfarce**:
- **a repetição T2 (`46+/0`) está por revalidar** -- a corrida está lançada;
- **nenhuma comparação externa foi executada** -- `NAO_EXECUTADA`, não `PARITY_MEASURED`;
- **a taxa da matriz T2 é 3/8** com modelos locais de 3–7B: *um modelo, um host, uma quantização*;
- **sem GPU/VRAM medidos por época** -- só tempo de parede;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1049`)

**Fechado**: a repetição T2. Corrida agora (`PASS`, 209,78 s); veredictos idênticos três
dias depois; o artefacto passou de `2026-10-13` para `2026-10-16T23:52:54Z` e o número é
medido. O `46+/0` **está confirmado**.

**Continua em falta, sem disfarce**:
- **N=3 por fixture**: contagem, não probabilidade, sem intervalo de confiança;
- **a taxa da repetição é 1 de 4 fixtures** (`soma` `3/3`; as outras três `0/3`, trapaceando);
- **nenhuma comparação externa foi executada** -- `NAO_EXECUTADA`, não `PARITY_MEASURED`;
- **um modelo, um host, uma quantização**;
- **sem GPU/VRAM medidos por época** -- só tempo de parede;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1050`)

**Fechado**: o mecanismo do vermelho do `pr-gate`. Não é defeito de código — é o push
seguinte a cancelar os gates que ele espera (`pr-gate.yaml:91`, deadline em `:61`).

**Continua em falta, sem disfarce**:
- **`55327160c` continua VERMELHO no GitHub** -- e quem o vir não sabe, sem ler o log,
  que a causa foi um push;
- **o `cda756f82` vai repetir o padrão** -- declarado antes de acontecer;
- **uma decisão do proprietário em aberto**: distinguir «cancelado pelo push» de
  «cancelado por falha» no `pr-gate` **toca a política do gate** e não é decidível pelo
  executor; a cura por baixo (aceitar `cancelled`) está **proibida**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **nenhuma comparação externa executada** -- `NAO_EXECUTADA`;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1051`)

**Fechado**: o mecanismo do `pr-gate`, provado com três SHAs. E o procedimento de push
mudou por causa disso.

**Continua em falta, sem disfarce**:
- **`55327160c` continua VERMELHO no GitHub** -- e quem o vir não sabe, sem ler o log,
  que a causa foi um push;
- **o `cda756f82` vai repetir o padrão** -- declarado antes de acontecer, não observado ainda;
- **uma decisão do proprietário em aberto**: distinguir «cancelado pelo push» de
  «cancelado por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está
  **proibida**;
- **o estágio 21 tem uma lacuna de integração**: os documentos canónicos de método não
  referenciam a matriz T2 real nem a repetição;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **nenhuma comparação externa executada** -- `NAO_EXECUTADA`;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1052`)

**Fechado**: o mecanismo do `pr-gate` — provado, previsto e confirmado. A disciplina de
push produziu **quatro 6/6 consecutivos** (`4157a459b`, `ba62baf88`, `67581a993`,
`2e0873635`).

**Continua em falta, sem disfarce**:
- **`55327160c` e `cda756f82` continuam VERMELHOS no GitHub** -- e quem os vir não sabe,
  sem ler o log, que a causa foi um push; **não os «limpei»** (reenviar mudaria o SHA);
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **`882acb121` não foi medido até ao fim** -- o `pr-gate` dele ainda não tinha fechado
  quando medi; **não afirmo** o seu estado;
- **o estágio 21 tem uma lacuna de integração**: os documentos canónicos de método não
  referenciam a matriz T2 real nem a repetição;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **nenhuma comparação externa executada** -- `NAO_EXECUTADA`;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1053`)

**Fechado**: a ligação do estágio 21 aos artefactos. Os três documentos referenciam
`HADES_T2_MATRIX.md` e `HADES_T2_REPETICAO.md`, com comandos reproduzíveis na secção 6.

**Continua em falta, sem disfarce**:
- **a ligação é textual**: se alguém renomear um artefacto, **nada acusa** -- não há teste;
- **os comandos da secção 6 exigem Ollama local e 25–30 min cada**;
- **a comparação externa continua `NAO_EXECUTADA`** -- nada disto produz número de concorrente;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **`882acb121` não foi medido até ao fim** -- não afirmo o seu estado;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1054`)

**Fechado**: a ligação do estágio 21 deixou de ser textual. A catraca
(`hades_stage_artifacts_check.py`) acusa artefacto apagado, artefacto vazio, ponteiro
removido e documento ausente — com 4 injeções e contraprova.

**Continua em falta, sem disfarce**:
- **o mapa `LIGACOES` cobre só o estágio 21** -- 3 documentos e 2 artefactos: uma ligação
  que não esteja no mapa **não é vigiada**;
- **o limite de 400 bytes é um palpite calibrado**, não uma medida;
- **a catraca verifica a presença do nome**, não que o texto descreva o artefacto certo:
  *um ponteiro errado apontaria para o mesmo nome*;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1055`)

**Fechado**: a primeira varredura de ligações docs→artefactos. **Zero defeitos reais** —
os 10 «ponteiros quebrados» e o 1 «artefacto órfão» eram **quatro erros do meu detector**.
E **não alarguei o mapa `LIGACOES`** com eles.

**Continua em falta, sem disfarce**:
- **não medi a pergunta certa**: quais documentos **afirmam produzir** um artefacto e não
  o citam -- a próxima acção;
- **o mapa `LIGACOES` continua com 6 entradas** (`E-1054`), só do estágio 21;
- **o detector de varredura não entrou na bateria** -- *um instrumento que produz falsos
  positivos não deve virar gate*;
- **`INSTALACAO.json` existe quando o instalador corre**, num caminho que eu não medi;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1056`)

**Fechado**: a linha de investigação da rastreabilidade. **Duas perguntas, dois zeros.**
Os quatro detectores de varredura **não entraram na bateria** — deliberado.

**Continua em falta, sem disfarce**:
- **os artefactos não assinados do candidato actual** (§FASE-6) -- o trabalho de produto
  que falta; **nenhum pacote do candidato `2bca9392f` foi produzido**;
- **o mapa `LIGACOES` continua com 6 entradas** (`E-1054`), só do estágio 21;
- **um detector léxico não vê uma ligação descrita por outras palavras** -- o limite fica;
- **`eas.json` existe**, mas num caminho que o detector só encontrou corrigido;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1057`)

**Fechado**: o **artefacto portátil Windows do candidato `6093a6692`** — 48 MB, verificado
com resposta real do modelo. E a lacuna do **filho órfão** no script (17 → 0).

**Continua em falta, sem disfarce**:
- **o `HadesSetup.exe` não assinado não foi gerado** -- o `ISCC.exe` existe; falta executar;
- **`INSTALLED=NAO` e `DEVICE_E2E=NAO`** -- instalar escreve em `%LOCALAPPDATA%\Programs\Hades`
  e **exige aprovação do dono**;
- **o bundle é Windows/amd64 apenas**: Linux, macOS, Android, iOS e Web/PWA continuam por
  produzir neste candidato;
- **o runner é pré-construído** (`_tmp\hadesbin`): o ZIP **não é** build reproduzível da fonte;
- **não assinado**, não publicado (`SIGNED_FOR_PUBLIC_RELEASE=NAO`);
- **Linux/macOS `BLOCKED_EXTERNAL` pela máquina**; Android bloqueado pela DLL JNI (`E-1019`);
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem assinatura, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1058`)

**Fechado**: o **`HadesSetup.exe` não assinado do candidato `d7ace6c12`** — 24,26 MB,
gerado com binários Go construídos da fonte. E a dívida do `[UninstallRun]` sem
`RunOnceId`, curada (o `Warning` desapareceu).

**Continua em falta, sem disfarce**:
- **`INSTALLED=NAO` e `DEVICE_E2E=NAO`** -- **não executei o instalador**: instalar escreve
  em `%LOCALAPPDATA%\Programs\Hades` e **exige aprovação do dono**;
- **o `lib\ollama` (42 ficheiros, 39,7 MB) é pré-construído**: o instalador **não é** build
  100% reproduzível da fonte; só os binários Go são;
- **não assinado** -- `SIGNED_FOR_PUBLIC_RELEASE=NAO`;
- **ARM64 não foi gerado** (o `.iss` protege com `Check: IsArm64()`);
- **Linux, macOS, Android, iOS e Web/PWA continuam por produzir** neste candidato;
- **Linux/macOS `BLOCKED_EXTERNAL` pela máquina**; Android pela DLL JNI (`E-1019`);
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1059`)

**Fechado**: o **tarball Linux instalável** — 11 entradas, com `install.sh`/`uninstall.sh`
e `VERSION`, construído **sem bash e sem compilador C**.

**Continua em falta, sem disfarce**:
- ***`BUILD_DESTE_HEAD=NAO` no Linux*** -- o binário é de `9f7379502b1d`; reconstruí-lo
  exige **CGO + gcc/clang**, ausentes;
- ***`INSTALLED=NAO` e `DEVICE_E2E=NAO` no Linux*** -- não há Linux nativo aqui, e o
  `install.sh` **não foi executado**;
- **a sintaxe do `install.sh` não foi validada por um bash** -- só o conteúdo, que vem do
  `package_linux.sh`;
- **macOS, Android, iOS e Web/PWA continuam por produzir** neste candidato;
- **`INSTALLED=NAO` e `DEVICE_E2E=NAO` no Windows** -- instalar exige aprovação do dono;
- **`lib\ollama` é pré-construído**: o instalador Windows **não é** build 100% reproduzível;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1060`)

**Fechado**: o build Web/PWA de produção (exit 0, orçamento PASS) e o **defeito dos três
ícones a 404** — que já estava no HEAD e agora resolve 3 de 3.

**Continua em falta, sem disfarce**:
- ***`SERVICE_WORKER=NAO`*** -- **zero** service workers: sem cache, sem offline, sem
  instalação PWA completa ⇒ **`PWA_INSTALAVEL=PARCIAL`**;
- **o `offline` é `NAO_IMPLEMENTADO`**, não «offline honesto»;
- **não instalei o PWA num browser real** nesta janela;
- **não testei upgrade/rollback do cache** -- não há cache;
- **warning real de chunks >500 kB** (`cpp` 626 kB, `emacs-lisp` 779 kB, `wasm` 622 kB) --
  são gramáticas carregadas sob demanda, e **não foi escondido**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1061`)

**Fechado**: o **service worker** com escopo honesto — `/api/*` network-only, cache do shell
e dos assets, limpeza de versões antigas. **8 testes + contraprova de 3 defeitos.**

**Continua em falta, sem disfarce**:
- ***`INSTALADO_EM_BROWSER=NAO`*** -- o SW **nunca foi instalado num browser real**;
- ***`E2E_EM_BROWSER_REAL=NAO`*** -- os 8 testes usam `caches` **simulado**:
  *«um teste com `caches` simulado prova a lógica; não prova a instalação»*;
- ***`OFFLINE=PARCIAL`*** -- o shell abre offline; o chat, as missões e as aprovações **não**,
  e a página offline di-lo;
- **sem teste de upgrade entre versões reais do SW** -- só a limpeza;
- **não medi** `prefers-reduced-motion`, memória, CPU nem bateria do SW;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1062`)

**Fechado**: o PWA num **Chromium real** — 8/8, incluindo os 3 ícones a 200 e `/api/*`
fora do cache, provado no browser.

**Continua em falta, sem disfarce**:
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar é humano e **não foi executado**;
- **Firefox e WebKit/Safari não testados** -- o E2E só correu em **Chromium**;
- **sem teste de upgrade entre versões reais do SW num browser** -- só a limpeza;
- **sem o servidor Ollama real por trás do `/api/*`** -- o preview serve só o estático;
- **não medi** memória, CPU nem bateria do SW;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missões e as aprovações **não**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1063`)

**Fechado**: o PWA em **Chromium, Firefox e WebKit** — 8/8 cada. Os dois motores que
faltavam foram instalados (Firefox 157.0, WebKit 27.2).

**Continua em falta, sem disfarce**:
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar é humano e **não foi executado**
  em nenhum dos três motores;
- **o WebKit do Playwright não é o Safari do macOS** -- é o motor; não prova Safari numa
  máquina Apple;
- **sem teste de upgrade entre versões reais do SW** num browser;
- **sem medição de memória, CPU, tempo de `install`/`activate` nem tamanho do cache**;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missões e as aprovações **não**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari** reais (extensão); sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1064`)

**Fechado**: os **recursos do SW** (9 entradas, 160,6 KB, **448 ms**) e o **upgrade
v1→v2** num Chromium real — a v1 é apagada e o `/api/*` continua fora do cache.

**Continua em falta, sem disfarce**:
- ***recursos e upgrade só em Chromium*** -- Firefox e WebKit ficaram no `E-1063` com
  os 8 pontos, mas **sem** medição de recursos nem upgrade;
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar é humano e **não foi executado**;
- **sem medição de CPU nem bateria** -- só tamanho do cache e latência;
- **`performance.memory` não foi reportado** pelo Chromium headless aqui;
- **o WebKit do Playwright não é o Safari do macOS**;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missões e as aprovações **não**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari** reais (extensão); sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1065`)

**Fechado**: recursos e upgrade do SW nos **tres motores** -- 9/9 cada, com os tres a apagar a v1
e a manter `/api/*` fora do cache.

**Continua em falta, sem disfarce**:
- ***o Firefox e' 3,9x mais lento a activar o SW (1646 ms vs 423 ms)*** -- medido, **nao explicado**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar e' humano e **nao foi executado** em
  nenhum dos tres;
- **sem medicao de CPU nem bateria** -- so' tamanho do cache e latencia;
- **o WebKit do Playwright nao e' o Safari do macOS**;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missoes e as aprovacoes **nao**;
- **a comparacao externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, nao por defeito;
- **decisao do proprietario em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a politica do gate**;
- **N=3 por fixture** na repeticao: contagem, nao probabilidade;
- **as 14 threads do PR continuam abertas** -- nao resolvi nenhuma (proibido);
- **o encaixe da aprovacao continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execucao
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versao 155 **nao carrega** por `--load-extension`;
- **Firefox** e **Safari** reais (extensao); sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **nao** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1066`)

**Fechado**: o perfil do `install`/`activate` do SW nos tres motores. O custo esta' no `install`
(18-132 ms); o `activate` e' **0 ms**; e os 1646 ms do `E-1065` eram o **arranque do browser**.

**Continua em falta, sem disfarce**:
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar e' humano;
- **sem medicao de CPU nem memoria** -- o perfil e' so' de tempo, e de **um registo a frio**;
- **o WebKit do Playwright nao e' o Safari do macOS**;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missoes e as aprovacoes **nao**;
- ***Android, iOS e macOS continuam por produzir*** -- Android e' a proxima viavel;
- **a comparacao externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, nao por defeito;
- **decisao do proprietario em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a politica do gate**;
- **N=3 por fixture** na repeticao: contagem, nao probabilidade;
- **as 14 threads do PR continuam abertas** -- nao resolvi nenhuma (proibido);
- **o encaixe da aprovacao continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execucao
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versao 155 **nao carrega** por `--load-extension`;
- **Firefox** e **Safari** reais (extensao); sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **nao** controla o PC.

### O que falta (actualizado na janela `E-1067`)

**Medido e fechado**: o inventario do Android -- o projecto esta' **completo**, e **JDK 17 +
SDK 34 + Gradle 8.13** estao' presentes. O bloqueio esta' **numa DLL do Gradle**, com causa nomeada.

**Continua em falta, sem disfarce**:
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- o Gradle 8.13 nao carrega a `native-platform.dll`
  (`NativeLibraryLocator.java:46`); **nao e' bloqueio externo** -- e' ambiente, com passo concreto;
- **sem E2E em emulador** e **sem prova de que o `App.tsx` compila**;
- ***a marca Android continua `DZ23 Agentic`*** -- `"name": "DZ23 Agentic"`,
  `"package": "com.lmprado.dz23agentic"`, `namespace 'com.lmprado.dz23agentic'`; a migracao para
  **`com.dz23ltd.hades`** que o dono decidiu **nao foi feita**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **macOS e iOS continuam por produzir**;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparacao externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, nao por defeito;
- **decisao do proprietario em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a politica do gate**;
- **N=3 por fixture** na repeticao: contagem, nao probabilidade;
- **as 14 threads do PR continuam abertas** -- nao resolvi nenhuma (proibido);
- **o encaixe da aprovacao continua por ligar** (`E-1044`);
- **o Chrome real do utilizador**: a versao 155 **nao carrega** por `--load-extension`;
- **`CAPACIDADES` VAZIO**: o Companion **nao** controla o PC.

### O que falta (actualizado na janela `E-1068`)

**Fechado**: a **marca Android** — `HADES`, `com.dz23ltd.hades`, com as chaves de storage
preservadas e o plano de migração escrito.

**Continua em falta, sem disfarce**:
- ***a marca no resto do monorepo*** -- o inventário mostra **231 `DZ23 Agentic`** e **49
  `Ollama DZ23`** ainda; a migração do mobile reduziu, mas não fechou;
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- o Gradle 8.13 não carrega a
  `native-platform.dll` (`E-1067`, 4 tentativas, 2 hipóteses novas); **não é bloqueio externo**;
- ***a migração não foi testada numa build*** -- *«uma migração de marca que nunca compilou
  está pronta a compilar — não está verificada»*;
- **sem E2E em emulador**;
- **macOS e iOS continuam por produzir**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1069`)

**Fechado**: a classificação das 231 ocorrências — **~250 técnicas e 19 histórico**, nenhuma
marca visível. O §FASE-5 item 2 está cumprido.

**Continua em falta, sem disfarce**:
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- o Gradle 8.13 não carrega a `native-platform.dll`
  (`E-1067`, 4 tentativas, 2 hipóteses novas); **não é bloqueio externo**;
- **a hipótese não testada**: extrair a DLL do jar **à mão** e passar `-Dorg.gradle.native.dir`;
- ***a migração da marca não foi testada numa build*** -- *«uma migração que nunca compilou
  está pronta a compilar — não está verificada»*;
- **sem E2E em emulador**;
- **o meu classificador de contexto tem 10 padrões** -- falsos negativos possíveis, e as **49
  `Ollama DZ23`** não foram verificadas uma a uma;
- **macOS e iOS continuam por produzir**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1072`)

**Ganho**: o Gradle **arranca** com a `native-platform.dll` extraída do jar (`--version` exit 0).
Antes nem isso.

**Continua em falta, sem disfarce**:
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- o **daemon** do Gradle 8.13 não carrega a DLL
  (**6 hipóteses, 10 tentativas**); **não é bloqueio externo**;
- **a 6ª hipótese falhou** (editar o `org.gradle.jvmargs` no `gradle.properties`) -- e o
  `android/` é **gerado**, logo testar ali é descartável;
- **a pista por seguir**: o log do daemon diz que ele **já correu** (2026-10-16T06:42) e falhou
  com `No Java compiler found` -- **a JVM do daemon não vê o JDK 17**;
- **a migração da marca não foi testada numa build**;
- **sem E2E em emulador**;
- **macOS e iOS continuam por produzir**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1073`)

**Ganho mantido**: o Gradle **arranca** com a `native-platform.dll` extraída do jar
(`--version` exit 0), e o daemon **já carregou** essa DLL neste host em 2026-10-16.

**Continua em falta, sem disfarce**:
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- **12 tentativas, 8 hipoteses**, `BLOCKED`
  registado; **não é bloqueio externo**, mas o desbloqueio é **decisão do dono**
  (outro Gradle 8.14+/9.x, ou outro host/runner);
- **as duas causas candidatas foram eliminadas** (o `java.io.tmpdir` e o `~/.gradle/native/`
  inexistente);
- **a migração da marca não foi testada numa build**;
- **sem E2E em emulador**;
- **macOS e iOS continuam por produzir**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a política do gate**;
- **os 73 IDs órfãos continuam por classificar** (§FASE-7 item 3) -- **o próximo trabalho**;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1074`)

**Fechado**: os 74 órfãos classificados — **todos dentro da faixa declarada, zero fora**;
**63 em código**, **11 só em documentos**, **88 ficheiros**; e **3 falsos positivos** provados.
A catraca do §FASE-7 item 6 está **cumprida**.

**Continua em falta, sem disfarce**:
- **as 133 não-citadas continuam sem texto** -- **não as reconstruí** (§FASE-7 item 2);
- **as 74 citações não foram verificadas uma a uma** -- provei que existem, não que estão no
  sítio certo;
- **as 11 só em documentos** não foram cruzadas com requisito;
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- `BLOCKED` no daemon do Gradle (`E-1073`),
  **decisão do dono**;
- **macOS e iOS continuam por produzir**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a política do gate**;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### A cura do `E-1075` (a mesma janela)

**O `E-1074` expôs um defeito de paridade**: o `hades_handoff_window_check.py` **não** tinha o
limite inferior que o `hades_index_shape_check.py` já tinha ⇒ o `E-000` (que vira `0` no `int()`)
entrava como *«citado sem entrada»* em **qualquer** documento que o mencionasse.

**Cura**: a **mesma** regra (`_abaixo = min(registados)` e `c >= _abaixo`). ⇒ ***«Dois checkers, a
mesma regra, e um deles ficou para trás.»***

**Resultado**: `JANELA DE CITACAO - fault injection: 20 OK, 0 FAIL` (era 18/2), e a bateria voltou a
**33 OK / 0 FAIL**. ⇒ ***«O documento que documenta o falso positivo é o teste que o checker não
esperava.»***

### O que falta (actualizado na janela `E-1076`)

**Fechado**: o **pacote de auditoria independente** — 5 artefactos nomeados, **gerados** e
verificados (`--check`), com **24 verificações** e **4 defeitos injetados**. E o `LIMITACOES.md`
separa **8 blockers abertos** de **9 resolvidos** pelo campo `tipo`, não pela palavra.

**Continua em falta, sem disfarce**:
- **o item 8 também pede a lista de findings abertos/fechados** -- o `LIMITACOES.md` dá os
  **blockers**, mas os **14 findings do PR #438** ainda não estão numa lista gerada;
- **o estado do CI do mesmo snapshot** -- o pacote **não** o inclui, e é por desenho: *o SHA
  muda a cada commit*;
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- `BLOCKED` no daemon do Gradle (`E-1073`),
  **decisão do dono**;
- **macOS e iOS continuam por produzir**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a política do gate**;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1077`)

**Fechado**: o **§FASE-7 item 8** está completo — os 5 artefactos nomeados (`E-1076`), os
**SBOMs** (já existentes) e a **lista de findings** (`E-1077`): **14 findings** (5 Major,
9 Minor) capturados com data e SHA, e o MD gerado do JSON.

**Continua em falta, sem disfarce**:
- **nenhum dos 14 findings foi validado contra o código** -- o `E-1070` mostrou o método
  (reproduzir num worktree limpo) e **nenhum** passou por ele;
- **o JSON envelhece**: um finding novo não aparece até alguém correr `--capturar`;
- **o `FINDINGS.json` não tem `isResolved`** -- a API de review comments não traz o estado da
  thread, e o GraphQL falhou;
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- `BLOCKED` no daemon do Gradle (`E-1073`),
  **decisão do dono**;
- **macOS e iOS continuam por produzir**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a política do gate**;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1135` -- proveniencia dos artefactos)

**Fechado nesta janela** (todas com medicao, nao declaracao):

- **o Windows citava o instalador de outro snapshot**: a matriz usava o caminho FIXO
  `dist/HadesSetup.exe`, que era o de `2026-10-16 22:58` (`SHA256 A70B5107...`), enquanto o
  artefacto do candidato estava em `dist/hades-installer-818251093/` (`0F95833A...`). Mesmo
  tamanho (24,26 MB). Cura: leitura do `SOURCE_SHA` do `PROVENIENCIA.txt`;
- **o Linux tinha a mesma classe**: a proveniencia vive **dentro** do tarball e declara
  `SOURCE_SHA: 9a0408e31905`. Cura: leitura com a stdlib `tarfile`;
- **o Web/PWA nao pode declarar SHA** (Vite nao o escreve): passou a ser provada a
  **auto-coerencia** do bundle (5/5 assets referenciados presentes) e a data do build;
- **os dois pacotes da extensao nao estavam em inventario nenhum**: passaram a ter seccao
  propria na matriz e a declarar `SOURCE_SHA: a313019e48ef` (campo escrito pelos dois geradores,
  e o inventario passou a ler `LEIA-ME.txt` *e* `PROVENIENCIA.txt`, em qualquer nivel).

**Continua em falta, sem disfarce**:

- **smoke de instalacao do `HadesSetup`**: `BLOCKED` pela ferramenta de execucao deste host
  (`Start-Process` deixa o filho a segurar o stdio do harness). Nada foi instalado e nada ficou
  pendente. Desbloqueio: runner do `test-install.yaml`;
- **`INSTALL` e `DEVICE_E2E` continuam NAO MEDIDO em TODAS as plataformas** -- a cura foi sobre
  a proveniencia do artefacto, **nao** sobre a homologacao dele;
- **o bundle Web/PWA nao declara o SHA do fonte** (limite do Vite, dito na matriz);
- APK Android (**decisao do dono**), macOS e iOS por produzir, PWA nao instalado na UI;
- pairing (`native messaging` + Companion), ligacao da extensao ao Mission Control, E2E em
  Chrome/Edge reais;
- benchmark reproduzivel (`ESTAGIO 21`) nao iniciado; `SUPERIOR_MEASURED` nao reclamado;
- as 14 threads do PR continuam abertas -- **nenhuma foi resolvida** (proibido sem autorizacao).
