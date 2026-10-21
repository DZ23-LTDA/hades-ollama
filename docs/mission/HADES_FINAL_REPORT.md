# HADES — RELATÓRIO FINAL DE EXECUÇÃO

> **Veredito: `NOT_RELEASE_READY`.**
>
> Este relatório é a **saída obrigatória** da missão, com os **15 itens** que o prompt exige. Tudo o que está
> afirmado tem **comando, resultado e evidência** — e tudo o que **não** foi feito está na secção **9 e 10**.
>
> **O que este relatório NÃO é:** não é «100% pronto», não é «release ready», não é «paridade total», não é
> «superior a todos», e **não é** `ENGINEERING_COMPLETE_AWAITING_INDEPENDENT_REVIEW` — *ver a secção 11*.

---

## 1. Identidade

| | |
|---|---|
| **Host** | Windows 11 Pro build 26300 · i9-13900HX (24c/32t) · 47,7 GB RAM · RTX 4060 Laptop 4 GB |
| **Data (BRT)** | **2026-10-17** |
| **Checkout** | `C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama` |
| **Branch** | `fix/checker-relogio` |
| **HEAD** | **`cf3429dfa`** (pushed, == remoto) |
| **`origin/main`** | `4b3e21854` — *e é **ancestral** do HEAD (`git merge-base --is-ancestor origin/main HEAD` = 0)* |
| **Dirty/WIP** | **0** com diff real · 11 entradas são **só fim-de-linha** (CRLF) · **0** não rastreados |
| **Stashes** | **0** |
| **PR #438** | **aberto**, não mesclado, não draft · **499 commits** · **536 ficheiros** · **14 threads, 0 resolvidas** (proibido resolver) |
| **CI do HEAD** | `cf3429dfa` — *ver secção 8* |

---

## 2. Arquivos alterados/criados e por quê

**18 instrumentos novos** (todos com teste discriminador), **1 pacote**, e **4 documentos canónicos**:

| ficheiro | por quê |
|---|---|
| `scripts/hades_requisitos.py` + `_test.py` | **REQ-ID → arquivo → teste → comando → resultado**: 10 requisitos ativos, 11 colunas |
| `scripts/hades_t2_contexto.py` + `_test.py` | o **ambiente** do T2 medido (hardware real, Ollama 0.40.1, 9 modelos) + 4 lacunas declaradas |
| `scripts/hades_testes_hermeticos_check.py` + `_test.py` | a **hermeticidade**: 107 de 192 chamadas sem `DataRoot` (tecto, não allowlist) |
| `scripts/hades_ux_onboarding_check.py` + `_test.py` | `/goal` no **mesmo** campo do texto; 5 testes discriminantes novos |
| `scripts/hades_harness_check.py` + `_test.py` | as **7 capacidades** do harness, nomeadas por teste |
| `scripts/hades_agents_check.py` + `_test.py` | o swarm com **`MaxAgents` APLICADO** (semáforo) e orçamento de 4 limites |
| `scripts/hades_platform_matrix.py` | os **6 alvos × 7 eixos** (SOURCE/BUILD/PACKAGE/INSTALL/DEVICE_E2E/assinatura/hash) |
| `scripts/hades_audit_inventory_check.py` + `_test.py` | a matriz, os 15 workflows, e as **instruções destrutivas marcadas** |
| `scripts/hades_architecture.py` + `_check.py` + `_test.py` | **1 Runtime, 1 Mission, 0 motores paralelos** |
| `scripts/hades_security_check.py` + `_test.py` | os **9 eixos** do Domínio K + os 4 instrumentos ligados |
| `scripts/hades_extension_check.py` + `_test.py` | a extensão MV3, **25 verificações** |
| `scripts/hades_mcp_check.py` + `_test.py` | o MCP real (spawn, transporte, SSRF) |
| `scripts/hades_plugins_check.py` + `_test.py` | o plugin em **2 metades** (a que não precisa de privilégio, e o `B-14`) |
| `scripts/hades_stability_check.py` + `_test.py` | o **flake do macOS** (`NAO_REPRODUZIDO`) e a dívida de 107 |
| `scripts/hades_competitive_check.py` + `_test.py` | a **matriz competitiva**: 27/27 repos + 49/49 produtos |
| `scripts/hades_traceability_check.py` + `_test.py` | os **12 IDs perdidos** marcados `EVIDENCE_LOST_HISTORICAL` |
| `scripts/hades_findings_revalidation.py` + `_test.py` | os **14 findings** do PR revalidados (13 corrigidos, 1 parcial) |
| `scripts/hades_linux_package.py` | o **pacote Linux** (cross-compile + tarball + hash) |
| `app/ui/app/e2e-extension-real.mjs` | o **E2E em motor real** da extensão (12 verificações) |

**Documentos canónicos criados/actualizados**: `HADES_ARCHITECTURE.md` (26 pacotes, 317 rotas) ·
`HADES_EVIDENCE_LOST_HISTORICAL.md` (os 12 com alcance e impacto) ·
`HADES_PLATFORM_MATRIX.md` (6 alvos) · `HADES_COMPETITIVE_CAPABILITY_MATRIX.md/.csv` (**130 capacidades × 12 colunas**).

**A bateria cresceu de 51 para 72 blocos** (`scripts/hades_local_gates.ps1`: 1 130 linhas).

---

## 3. Findings do PR #438

**14 threads · 0 resolvidas no GitHub** (*proibido sem autorização do proprietário*) · **revalidação técnica
completa** (`docs/audit/FINDINGS.json` tem agora o campo `revalidacao`).

| veredicto | n | quais |
|---|---|---|
| **`CONFIRMADO_CORRIGIDO`** | **13** | *incluindo o **P0** (`4266197772`)* |
| **`CONFIRMADO_PARCIAL`** | **1** | *`4266327642` — o pipe no índice* |

**O P0, medido**: o `service_agent_approvals.go` tem o **padrão público + `Locked`** em **6 pares**
(`SetApprovers`/`setApproversLocked`, `Save`/`saveLocked`, `expireStale`/`expireStaleLocked`,
`Request`/`requestLocked`, `Decide`/`decideLocked`, `Consume`/`consumeLocked`) com **9** `withFileLock` — e os
**5 testes de corrida passam**:

```
--- PASS: TestDeploymentApprovalPersistsAcrossRestartAndConsumesOnce (0.03s)
--- PASS: TestServiceAgentApprovalConsumeRefusesExpiredEvenIfApproved (0.06s)
--- PASS: TestExpireStaleNaoRessuscitaAprovacaoConsumida (0.05s)
--- PASS: TestSaveESetApproversNaoRessuscitamAprovacaoConsumida (0.08s)
--- PASS: TestConsumeConcorrenteComEscritoresNaoDuplicaExecucao (0.64s)
```

**O `CONFIRMADO_PARCIAL`**: o pipe foi corrigido **no caso**, mas a **classe** persiste em **831** linhas do
índice (os comandos reproduzíveis contêm `|`) → corrigi-la exigiria **reescrever a evidência**, o que o prompt
proíbe.

⇒ ***`um finding fechado no caso e aberto na classe não é um finding fechado.`***

---

## 4. Matriz dos 10 estágios livres

| estágio | antes | depois | evidência | pendência concreta |
|---|---|---|---|---|
| **1** Auditoria/inventário | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1100` | — |
| **3** Arquitetura canónica | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1102` | — |
| **6** Harness Runtime | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1098` | CLI externo `BLOCKED_EXTERNAL` (sonda opt-in) |
| **9** Agentes/subagentes | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1099` | missão multiagente real sobre Ollama (o `MaxAgents` está aplicado) |
| **10** Mission Control único | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1097` | custo/latência por etapa |
| **12** Studio/Builder | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1092` | — |
| **17** Workspace | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1094` | render por tipo |
| **19** UX/onboarding | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1095` | E2E real do onboarding |
| **21** Benchmarks | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1093` | comparação externa `NAO_EXECUTADA` |
| **22** Gate de engenharia | `IN_PROGRESS` | **`COMPLETED_VERIFIED`** | `E-1103` | — |
| **§FASE-3-A** Pet Hades | `PARTIAL` | **`COMPLETED_VERIFIED`** | `E-1095`/`E-1052` | Web/desktop cobertos; mobile não |
| **2** Estabilização | `IN_PROGRESS` | **`IN_PROGRESS`** | `E-1108` | **107** testes não-herméticos — a dívida que o estágio existe para pagar |
| **8** Plugins/Skills | `IN_PROGRESS` | **`PARTIAL`** | `E-1107` | `B-14`: a execução exige **namespaces de montagem** (`BLOCKED_EXTERNAL` legítimo) |
| **14** Browser/computer-use | `IN_PROGRESS` | **`PARTIAL`** | `E-1105` | pareamento real, DOM/click, Firefox, Chrome/Edge reais, computer-use |

⇒ **O estado canónico**: **14 `COMPLETED_VERIFIED`**, **3 `IN_PROGRESS`** (2, 8, 14), **21 `BLOCKED_BY_EXTERNAL_DEPENDENCY`** — *38 estágios no total*.

---

## 5. Ollama, Mission Control, `/goal`, menu, marca e pet

| | estado | prova |
|---|---|---|
| **Ollama como motor principal** | ✅ | *local-first; 9 modelos em `127.0.0.1:11434`; o `qwen2.5:7b` respondeu (T2, `E-1093`)* |
| **Mission Control único** | ✅ | ***0** `type MissionControl struct`* — *o MC é a **superfície** (12 rotas), o motor é o `Runtime`* (`E-1102`) |
| **`/goal` no mesmo campo do chat** | ✅ | *o `/goal` está no `ChatForm.tsx` **e** no `slashCommands.ts`, com **13 testes*** (`E-1095`) |
| **Menu unificado** | ✅ | *≤24 rotas (22), sem duplicados* |
| **Marca HADES** | ⚠️ | *a UI diz HADES; restam strings legadas em workflows/artefactos (`Ollama Classe A+`), inventariadas em `HADES_BRAND_INVENTORY.md`* |
| **Pet contextual** | ✅ | *14 estados, precedência, eventos duplicados/atrasados, reduced-motion, ARIA (14 em `Onboarding.tsx`)* (`E-1095`) |

---

## 6. Matriz por plataforma

| plataforma | SOURCE | BUILD | PACKAGE | INSTALL | DEVICE_E2E | assinatura | hash |
|---|---|---|---|---|---|---|---|
| **Windows** | ✅ | ✅ | ✅ `HadesSetup.exe` **24,3 MB** | **NAO** | **NAO** | `false` | `a70b51073d89b7f6…` |
| **Linux** | ✅ | ✅ **cross** (`CGO=0`) | ✅ **16,2 MB** | **NAO** | **NAO** | `false` | `1b7ee3046a887138…` |
| **macOS** | ✅ | **NAO** | **AUSENTE** | **NAO** | **NAO** | — | — |
| **Android** | ✅ | **NAO** | **AUSENTE** | **NAO** | **NAO** | — | — |
| **iOS** | ✅ | **NAO** | **AUSENTE** | **NAO** | **NAO** | — | — |
| **Web/PWA** | ✅ | ✅ | ✅ `index.html` 9 KB | **NAO** (`PWA_INSTALADO_NA_UI=NAO`) | **NAO** | — | `0daa2a2f233dc3d5…` |

**E as causas, todas medidas:**
- **macOS** → `GOOS=darwin CGO=0` falha com **`undefined: GetCPUMem`**: o `GetCPUMem` do darwin vive em
  `discover/gpu_darwin.go`, que usa **`import "C"`** (Metal/Foundation), e o Go **exclui** ficheiros com
  `import "C"` quando `CGO=0` ⇒ **`BLOCKED_EXTERNAL` legítimo** (Xcode + CGO). *Desbloqueio: `dz23-macos-build.yaml` em `macos-latest`.*
- **Android** → **3 camadas** (`E-1113`): o «different roots» era o **`subst`** que eu criei; sem ele, falha no
  **Kotlin** (`kotlin-stdlib 2.2.0` vs `1.9.24`); e a **doença** é o `package.json` declarar **`expo ^53.0.27`
  com `react-native ^0.87.1`**, quando o `bundledNativeModules.json` do próprio Expo 53 diz **`react-native: 0.79.6`**
  ⇒ **`BLOCKED` por DECISÃO DE PRODUTO** (*não externo*).
- **iOS** → sem macOS/Xcode.

⇒ ***`cross-compile não é teste nativo` *** ⇒ *o Linux tem **PACKAGE** e **não** tem **INSTALL** nem **DEVICE_E2E**.*

---

## 7. Rastreabilidade

| | |
|---|---|
| **Entradas no índice** | **906** (`E-001..E-1113`) |
| **Faixa histórica ausente** | **`E-571..E-777`** (207 IDs) — **declarada**, e a catraca **só a deixa diminuir** |
| **IDs citados e não registados** | **12** — **todos** marcados `EVIDENCE_LOST_HISTORICAL` com **onde são citados, o que se perdeu, o impacto e a decisão** (`HADES_EVIDENCE_LOST_HISTORICAL.md`) |
| **Placeholders** | `E-000` e `E-999` — **separados** dos perdidos |
| **Referências ativas reparadas** | **0** dependem de ID perdido — *os 10 `REQ-*` apontam para código/teste/comando/resultado medidos agora* |
| **Dívida restante** | os **61** órfãos que não foram localizados individualmente (dentro da faixa declarada) |

⇒ ***`a diferença entre um buraco declarado e um buraco escondido é a diferença entre uma dívida e uma fraude.`***

---

## 8. Testes e gates

| gate | comando | resultado |
|---|---|---|
| **Bateria local** | `hades_local_gates.ps1` (**72 blocos**) | **72 OK / 0 FAIL** |
| **Bateria manual (esta sessão)** | 108 checkers + testes | **0 FAIL** |
| **`go test ./server`** | com `APPDATA`/`LOCALAPPDATA`/`TMP` em `_tmp` | **0 falhas, ok 33,597 s** |
| **`go test ./internal/agent`** | idem | **0 falhas, ok 327,803 s** (1 557 testes) |
| **`go test -count=1 -bench=. -benchtime=1x ./server/`** | o **comando exacto do CI** | **ok 4,762 s** (51,3 s de parede) |
| **`go vet`** | `./...` | **0** |
| **`go test -race`** | — | **`NOT_SUPPORTED`** (sem CGO) — **não** renomeado para PASS |
| **Extensão** | `node --test` (6 ficheiros) | **120 pass / 0 fail** (5,3 s) |
| **E2E da extensão** | `e2e-extension-real.mjs` | **12 verificações** (motor real) |
| **CI do HEAD** | `cf3429dfa` | **6/6 verde** (`pr-gate`, `class-a-plus-integrity`, `dz23-agentic-quality`, `dz23-e2e`, `test`, `dz23-multi-provider`) |
| **CI — sequência** | últimos **13 SHAs** | **6/6 verde** em todos |

**Skipped**: `go_license` (condicional de caminho), `windows` e `linux` (matriz opcional `run_native_matrix`) —
*os 3 são **skips condicionais declarados** no workflow, não falhas escondidas.*

---

## 9. Bloqueios externos, com passo de desbloqueio

| bloqueio | passo exacto | responsável |
|---|---|---|
| **macOS build** | `dz23-macos-build.yaml` em `macos-latest` (Xcode + CGO) | owner do CI |
| **iOS build** | macOS + Xcode + conta Apple | owner |
| **Android APK** | **decisão**: `react-native → 0.79.6` **ou** `expo → ^54` | **proprietário** |
| **Chrome/Edge reais** | publicar na Chrome Web Store / Edge Add-ons (conta + revisão) | proprietário |
| **Firefox** | E2E com `web-ext` + `--load-extension` | interno (não tentado) |
| **PWA instalado** | gesto humano no browser | humano |
| **`INSTALL` Windows/Linux** | instalar num host real | humano |
| **DEVICE_E2E** | dispositivo físico | humano |
| **Comparação externa** | acesso legal aos concorrentes | proprietário |
| **Revisão humana independente** | outro revisor (não `LMPrado-DZ23`) | proprietário |
| **20 referências da matriz** | consultar a fonte primária | interno (adiado) |
| **Plugin execution** (`B-14`) | host Linux com `CAP_SYS_ADMIN` / `--privileged` | operador |

---

## 10. Riscos críticos/altos restantes

1. **`NOT_RELEASE_READY`**: *não houve homologação real, nem revisão humana independente.*
2. **O APK não existe** — *e a decisão da versão de RN é do proprietário.*
3. **3 plataformas sem pacote** (macOS, Android, iOS).
4. **107** testes não-herméticos (dívida declarada, com catraca).
5. **831** linhas do índice com pipe no conteúdo (a classe do `4266327642`).
6. **61** órfãos não localizados individualmente (dentro da faixa declarada).
7. **20** referências da matriz sem fonte primária (`UNVERIFIABLE`).
8. **0** capacidades `SUPERIOR_MEASURED` — *não há benchmark comparativo executado.*

---

## 11. Veredito honesto

# `NOT_RELEASE_READY`

**E `ENGINEERING_COMPLETE_AWAITING_INDEPENDENT_REVIEW` NÃO é autorizado**, porque o critério exige:

| critério | estado |
|---|---|
| os 10 estágios livres `COMPLETED_VERIFIED` | ✅ **sim** |
| findings críticos/major corrigidos e testados | ✅ **13/14** (1 parcial declarado) |
| WIP integrado, sem ficheiro órfão | ✅ **sim** (0 não rastreados) |
| **artefactos não assinados do candidato para as plataformas tecnicamente possíveis** | ⚠️ **Windows, Linux, Web sim; macOS, Android, iOS não** |
| **Android com APK e E2E em emulador** | ❌ **não** |
| **iOS com build/simulador ou bloqueio externo preciso** | ✅ bloqueio preciso, **sem** build |
| **Web/PWA e desktops com E2E real** | ⚠️ Web sim; **desktops não** |
| requisitos ativos não dependem das 134 perdidas | ✅ **sim** |
| gate do mesmo snapshot verde | ✅ **sim** (`gate_snapshot = MESMO`) |
| nenhum bloqueio interno mascarado como externo | ✅ **sim** (o Android está como **decisão de produto**, não externo) |

⇒ ***os 4 critérios a vermelho/amarelo impedem a declaração.*** ⇒ *e o veredito **continua** `NOT_RELEASE_READY` até revisão humana independente, homologação por plataforma e autorização do proprietário.*

---

## 12. Matriz competitiva por capacidade

**`HADES_COMPETITIVE_CAPABILITY_MATRIX.md`** (26,3 KB) + **`.csv`** (131 linhas × 12 colunas):
**27/27** repositórios e **49/49** referências de produto do prompt **cobertos** — *nenhuma ignorada.*

- **0** `SUPERIOR_MEASURED` — *nenhuma vitória sem benchmark*;
- as referências não consultadas são **`UNVERIFIABLE`** com motivo, licença e data — **não** `PARITY_MEASURED`;
- as **9 correcções factuais** do prompt aplicadas (Tel-Agent/telefonia, `block/buzz`/eventos assinados,
  `morluto/rea`/engenharia reversa, Paperclip/equipas, Soup/fine-tuning, CodeNotch/monitor, Open-claude/workspace,
  Buzz/owner-repo).

⇒ ***`uma matriz que diz «não sei» sobre 20 de 69 referências é mais útil do que uma que diz «paridade» sobre todas.`***

---

## 13. Capacidades novas efectivamente integradas

*Separadas de protótipos: só o que tem **código + teste + caminho de utilizador**.*

| CAP-ID | caminho de utilizador | código | teste | E2E | benchmark |
|---|---|---|---|---|---|
| `CAP-L-01` `/goal` no composer | ChatForm → API → Mission Control | ✅ | ✅ 13 testes | ⚠️ só backend | — |
| `CAP-D-01` Mission Control único | 12 rotas `/missions` | ✅ | ✅ 4 testes | ✅ SSE | — |
| `CAP-K-01` approvals de uso único | `Consume` com lock | ✅ | ✅ **5 de corrida** | — | — |
| `CAP-F-01` Browser Bridge MV3 | extensão → popup | ✅ | ✅ **120** | ✅ **12** em motor real | — |
| `CAP-H-01` MCP real | spawn + transporte + SSRF | ✅ | ✅ 7 ficheiros | — | — |
| `CAP-H-02` Plugin (2 metades) | manifesto/bundle/registry | ✅ | ✅ 13 ficheiros | — | — |
| `CAP-E-01` Ollama local-first | `127.0.0.1:11434` | ✅ | ✅ | ✅ 9 modelos | ⚠️ T2 sem comparação |
| `CAP-A-01` swarm com `MaxAgents` | semáforo | ✅ | ✅ | — | — |

**Protótipos / trabalho futuro** (não contam como entrega): *computer-use, o pareamento real da extensão, o plugin a executar, a missão multiagente sobre Ollama, e o benchmark comparativo.*

---

## 14. Delegações Hermes

**Nenhuma delegação ao Hermes foi executada** — *e o prompt é explícito: «Um bloco YAML não prova que Hermes executou»*.

| task_id | resultado | prova |
|---|---|---|
| — | **DISPENSADA** | *a integração real não foi provada: não há sessão/execução do Hermes exercitada com progresso, cancelamento e recuperação* |

⇒ *o trabalho foi executado **localmente** pelo DeepSeek, que é o que o prompt manda quando a integração não está provada.* ⇒ ***`não gerar diretivas em loop`*** — *e não foi gerada nenhuma.*

---

## 15. Próxima acção única

# **Decidir a versão do React Native**

**Opção (1) — recomendada**: `react-native: 0.79.6` (alinhar com o Expo 53, que é o SDK instalado e empacota
**0.79.6**). *A menos disruptiva: mantém o SDK, desce o RN.* **Impacto**: `package-lock` + `node_modules` (600 MB),
re-validar o `apps/mobile-agentic`, e depois `assembleDebug` para gerar o APK.

**Opção (2)**: `expo: ^54` (suporta RN 0.81+). *Mais moderna, mas sobe o SDK e provavelmente várias APIs.*

⇒ *a decisão é do **proprietário** porque muda o que o utilizador final recebe.* ⇒ ***e é o único item que separa o Android do `PACKAGE=sim`.***
