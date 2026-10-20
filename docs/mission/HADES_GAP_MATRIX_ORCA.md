# HADES × ORCA × OPENJARVIS — MATRIZ DE LACUNAS (FASE 0 + FASE 1)

**Este documento existe porque a FASE 1 do adendo V3 nunca tinha sido escrita.** Estava declarada
como ausente em todas as rondas anteriores. Sem esta matriz, cada fatia de código ficou sem
justificação rastreável de prioridade.

---

## FASE 0 — HEADs AUDITADOS

| Referência | HEAD auditado | Data (UTC) | Licença | Linguagem | Estrelas |
|---|---|---|---|---|---|
| `stablyai/orca` | `1938bd055aedd349d07172056fec6810992d5e0e` | 2026-10-18T15:11:52Z | MIT | TypeScript | 96 169 |
| `open-jarvis/OpenJarvis` | `2817faa5fe9dbf0612f56be46f77d7c5c38e271c` | 2026-10-18T06:15:06Z | Apache-2.0 | Python | 11 511 |
| `DZ23-LTDA/hades-ollama` | ver `docs/mission/HADES_MASTER_STATE.json` | — | — | Go + TS | — |

**Método:** consulta à API pública do GitHub (`/repos`, `/commits/main`, `/git/trees/main`).
Nenhum script, instalador ou binário dos repositórios externos foi executado. Nenhum código foi
copiado. As afirmações de README foram tratadas como alegações até confirmação no código.

**Fonte por capacidade:** as linhas abaixo marcam `[C]` quando a capacidade foi observada na
**árvore de ficheiros do commit auditado** (evidência de código) e `[D]` quando vem de
documentação. Não usei `[E]` (execução) para nenhum repositório externo porque não os executei —
e não os executei de propósito. *Uma capacidade que eu não corri não é uma capacidade que eu
verifiquei.*

**O que a árvore do Orca mostra, em concreto** `[C]`: `src/main/ssh/` e
`src/main/ipc/runtime-environments/` (runtime remoto por SSH), `src/main/ephemeral-vm-runtime-ssh.ts`
e `src/shared/ephemeral-vm-recipes` (sandbox descartável por receita), `src/main/ipc/pty/` e
`src/renderer/src/components/terminal-pane/` (terminal com panes), `WorktreeCard.tsx` (worktrees na
UI), `src/shared/protocol-version.ts` e `tests/e2e/cross-version-wire/` (protocolo versionado com
testes N/N-1), `src/main/browser/paired-runtime-browser-*` (browser controlado emparelhado),
`src/main/ai-vault/` (arquivo de sessões de IA), `src/main/daemon/daemon-server.ts` (servidor
persistente), `config/reliability-gates.jsonc` (gates com invariante + oráculo + comandos + refs de
asserção), `skill-guides/`, `skill-stubs/`, `skills/`.

---

## A LIÇÃO METODOLÓGICA QUE ESTA MATRIZ TEVE DE CORRIGIR

A primeira passagem do inventário procurou **ficheiros** com nomes que eu inventei
(`*ssh*`, `*terminal*`, `*protocolo*`) e devolveu **8 capacidades "AUSENTE"**. A segunda passagem
procurou **conteúdo** com nomes alternativos e encontrou:

| o que eu tinha chamado AUSENTE | o que existe de facto |
|---|---|
| design-mode | `inspec` **162**, `inspector` 2, `overlay` 1 |
| runtime SSH | `SSH` **88**, `Remoto` 28 |
| terminal | `pty` **987**, `Shell` 50 |
| protocolo versionado | `Capability` **291**, `handshake` 12, `protocol` 16 |
| Computer Use | `ComputerUse` 7, `computer_use` 5, `Acessibilidade` 10 |
| engine/hardware | `VRAM` **27** |
| API compatível OpenAI | `openai` **56**, `/v1/chat` 33 |

**Pela quinta vez, o inventário errou na mesma direção:**

> *«O inventário errou duas vezes na mesma direcção: procurei o nome que inventei, não o nome que existe.»*

**Consequência para esta matriz:** nenhuma linha abaixo diz `AUSENTE` com base numa busca por nome
de ficheiro. `AUSENTE` só aparece quando **a busca por conteúdo também falhou** e o resultado está
escrito ao lado.

---

## FASE 1 — MATRIZ DE LACUNAS

Colunas: **capacidade** · **fonte no Orca/OpenJarvis** · **situação HADES** · **lacuna real** ·
**decisão** · **evidência**.

### Grupo A — Orca (cockpit, worktrees, terminais, supervisão, execução paralela, runtime remoto)

| # | Capacidade | Fonte | Situação HADES | Lacuna real | Decisão | Evidência |
|---|---|---|---|---|---|---|
| A1 | Worktrees isolados por agente | `[C]` `WorktreeCard.tsx`, `git worktree` | **COMPLETO_VERIFICADO** | nenhuma nas operações implementadas | **SUPERAR** (já supera: estados + diagnóstico + registo) | E-1226..E-1238 |
| A2 | Ciclo de vida de worktree (9 estados) | `[C]` árvore não mostra enum | **COMPLETO_VERIFICADO** | — | **SUPERAR** | E-1227, E-1228, E-1234 |
| A3 | Comparação objetiva de soluções | `[C]` não observada na árvore | **COMPLETO_VERIFICADO** | — | **SUPERAR** | E-1239, E-1240 |
| A4 | Terminal com panes e PTY | `[C]` `src/main/ipc/pty/`, `terminal-pane/` | **PARCIAL** — **MEDIDO (E-1453): 0 criacao de PTY em Go, nenhuma dependencia de pty, 0 paineis na UI -- o `PARCIAL` vinha das 987 ocorrencias do lado ORCA, nao deste repo**** (`pty` 987 ocorrências) | divisão vertical/horizontal, reconexão, scrollback com busca | **ADAPTAR** | a medir |
| A5 | Runtime remoto por SSH | `[C]` `src/main/ssh/`, `ipc/runtime-environments/` | **PARCIAL** — **PROVADO (E-1455): 3 testes com servidor SSH efemero real -- comando executado, host key divergente recusada, ausencia de callback recusada** — **MEDIDO (E-1454): existe cliente SSH real (`SSHEnvironment` -> `runOverSSH` -> `ssh.NewClientConn`), com `HostKeyCallback` OBRIGATORIA e sem `InsecureIgnoreHostKey`; 22 testes PASS** (`SSH` 88) | interface comum de execução com provedores (local/SSH/servidor/contêiner/VM) | **ADAPTAR** | a medir |
| A6 | Sandbox descartável por receita | `[C]` `ephemeral-vm-runtime-ssh.ts`, `ephemeral-vm-recipes` | **AUSENTE** — busca por `ephemeral`, `receita`, `sandboxVM`, `VM efémera` no `internal/agent` devolveu 0; **actualizacao E-1462**: o unico `--sandbox` do repo e' DELEGADO a CLI externa (codex), nao implementado aqui | receitas versionadas de ambiente por workspace | **ADOTAR** | por registar |
| A7 | Servidor persistente (`daemon-server`) | `[C]` `src/main/daemon/daemon-server.ts` | **EXISTE_NÃO_PROVADO** — **MEDIDO (E-1449): `serve` persiste entre 3 sessoes independentes e apos 8 s de ocio, mesmo PID, 200/200/200/200; sem reinicio automatico nem supervisor**** | modo servidor persistente com projetos/worktrees/agentes/terminais/sessões | **ADAPTAR** | a medir |
| A8 | Protocolo versionado + testes N/N-1 | `[C]` `protocol-version.ts`, `tests/e2e/cross-version-wire/` | **PARCIAL** — **MEDIDO (E-1457): versao com recusa explicita (`v !== PROTOCOLO` -> `protocolo_incompativel`, provado com `v: 99`); compatibilidade N/N-1 NAO existe** (`Capability` 291, `handshake` 12) | teste explícito cliente N-1 contra servidor N | **ADOTAR** | a medir |
| A9 | Browser controlado emparelhado | `[C]` `paired-runtime-browser-*` | **PARCIAL** — **MEDIDO (E-1456): 22 ficheiros, checker com oraculo exit=0, 3 suites JS exit=0; falta emparelhamento com Chrome real** (`apps/hades-browser-bridge/`) | emparelhamento autenticado, lease, reconexão | **ADAPTAR** | a medir |
| A10 | Gates com invariante + oráculo + refs de asserção | `[C]` `config/reliability-gates.jsonc` | **PARCIAL** — **MEDIDO (E-1450/E-1451/E-1452): 17 de 38 checkers com oraculo, 12 sem, 9 sem teste proprio; 10 requisitos, 10 citados; violar REQ-3A-03 deixa o gate vermelho**** (gates existem, formato diferente) | declarar por gate o invariante e a asserção que o prova | **ADAPTAR** | a medir |
| A11 | Arquivo de sessões de IA (`ai-vault`) | `[C]` `src/main/ai-vault/` | **PARCIAL** — **MEDIDO (E-1458): vault governado com 15 ficheiros, 76 testes Go PASS e 75 assercoes de E2E; arquiva MEMORIA com rotulo/fonte/hash, nao transcricoes de sessao** (`internal/agent/historico*`) | busca em histórico de sessões | **ADAPTAR** | a medir |
| A12 | App móvel como cliente de supervisão | `[C]` `mobile/` | **PARCIAL** — **MEDIDO (E-1460): 50 ficheiros; 3 testes de politica (offline, contexto de missao, cliente) PASSAM em node puro sem Android; login/aprovacao/outbox e build Android continuam BLOCKED_EXTERNAL; `tsc --noEmit` exit=0 com 0 erros (E-1461)** (`apps/mobile-agentic`, 17 ficheiros) | allowlist de RPC no celular | **ADAPTAR** | a medir |

### Grupo B — OpenJarvis (inteligência local, engines, memória, agentes pessoais, traces, skills)

| # | Capacidade | Fonte | Situação HADES | Lacuna real | Decisão | Evidência |
|---|---|---|---|---|---|---|
| B1 | Interface canónica de inferência | `[D]` README «Personal AI, On Personal Devices» | **PARCIAL** — **MEDIDO (E-1463): 12 metodos publicos, 3 tipos de provider (openai-compatible/anthropic/cli), 28 ficheiros de teste, 164 PASS** (`internal/multillm.Registry`) | localidade explícita além de `AllowPrivate` | **ADAPTAR** | FASE 29 |
| B2 | Detecção de hardware e recomendação | `[C]` `VRAM` 27 no HADES | **MEDIDO** | recomendação de quantização/contexto/offload a partir de VRAM medida | **ADAPTAR** | FASE 16 |
| B3 | Memória com fontes e escopos | `[D]` | **PARCIAL** — **MEDIDO (E-1464): 20 ficheiros, 9 escopos (SESSION..SHARED_AUTHORIZED), Fonte/Aprovador/ContentHash como campos, 53 PASS; a imposicao do escopo esta' na ACL, nao no tipo** (`internal/agent/memory*`, 20 ficheiros) | `ObsidianMemoryAdapter` + 12 E2E da FASE 17.1 | **ADAPTAR** | FASE 17.1 |
| B4 | Traces auditáveis e aprendizado medido | `[D]` | **PARCIAL** (2 ficheiros) | gate de 10 passos para política aprendida | **ADAPTAR** | FASE 18 |
| B5 | Skills com quarentena e benchmark | `[D]` | **PARCIAL** — **MEDIDO (E-1465): 7 ficheiros, 25 PASS (com recusas de travessia/duplicado); quarentena existe para PERSONAS, nao para skills; benchmark de skills AUSENTE** (`skill*` 7 ficheiros) | comparar sem/original/optimizada com várias seeds | **ADAPTAR** | FASE 20 |
| B6 | Eficiência e custo medidos | `[D]` | **PARCIAL** (`metricas.go`) | energia/VRAM marcados `NAO_MEDIDOS` com método declarado | **ADAPTAR** | FASE 21 |
| B7 | Agentes pessoais agendados/contínuos | `[D]` | **PARCIAL** — **MEDIDO (E-1466): 15 ficheiros (nao 21), 65 PASS com aprovacao+lease+dead-letter+compensacao; `continuo` NAO existe (sem laco perpetuo/daemon)** (`schedule*` 21 ficheiros) | anti-duplicação após perda de conexão | **ADAPTAR** | FASE 22 |
| B8 | Assistente local-first | `[D]` descrição do repositório | **MEDIDO** | conversão conversa→/goal com motivo explicado | **ADAPTAR** | FASE 15 |

### Grupo C — Transversais do adendo (não vêm de nenhum dos dois)

| # | Capacidade | Situação HADES | Lacuna real | Decisão | Evidência |
|---|---|---|---|---|---|
| C1 | Build/Signing/Publishing Hub | **PARCIAL** (2 ficheiros de toolchain) | **MEDIDO** | **ADOTAR** | FASE 28 |
| C2 | Computer Use com perfis de autonomia | **PARCIAL** (`ComputerUse` 7) | **MEDIDO** | **ADOTAR** | FASE 27 |
| C3 | Design Studio com validação de conteúdo | **PARCIAL** (17 ficheiros + validador E-1241) | **MEDIDO** | **ADOTAR** | E-1241 |
| C4 | CLI determinística | **PARCIAL** (`cmd/` 31 ficheiros) | **MEDIDO** | **ADAPTAR** | FASE 8 |
| C5 | Presets verificáveis | **PARCIAL** (`preset*` 6 ficheiros) | **MEDIDO** | **ADAPTAR** | FASE 23 |
| C6 | API compatível OpenAI | **PARCIAL** (`openai` 56, `/v1/chat` 33) | **MEDIDO** | **ADAPTAR** | FASE 24 |

---

## DECISÕES TOMADAS E O QUE ELAS OBRIGAM

**`SUPERAR`** (A1, A2, A3) — o HADES já tem mais do que a referência **e tem prova**: estados
explícitos, diagnóstico de ciclo de vida, registo persistente e comparação com recusa de decidir
sozinha. Não há trabalho a fazer por imitação. *Superar não é copiar melhor: é ter a propriedade
que a referência não tem, com um teste que falharia sem ela.*

**`ADAPTAR`** (maioria) — a capacidade existe em parte no HADES; o trabalho é **fechar a lacuna
nomeada na coluna própria**, não reescrever o módulo. Isto contraria a tentação de reimplementar:
*uma capacidade parcial que já tem 987 ocorrências de `pty` não precisa de um segundo terminal.*

**`ADOTAR`** (A6, A8, C1, C2, C3) — lacuna comprovada com busca negativa escrita ao lado.

**`NÃO_ADOTAR`** — nenhuma nesta passagem. Mas fica registado o critério: adotar-se-ia apenas o que
resolvesse lacuna comprovada. *Um adendo que manda auditar dois produtos não é uma ordem para os
copiar.*

---

## O QUE ESTA MATRIZ NÃO PROVA

- **Não provei nenhuma capacidade do Orca ou do OpenJarvis em execução.** Toda a coluna «fonte» é
  `[C]` (código no commit auditado) ou `[D]` (documentação). Nenhum `[E]`.
- **As situações `PARCIAL` sem evidência ID dizem «a medir»** — são hipóteses de inventário, não
  achados. *Marcar uma linha como PARCIAL sem teste é a mesma falta de rigor que marcar
  COMPLETO_VERIFICADO sem caminho real.*
- **O HEAD do HADES nesta matriz é o do `HADES_MASTER_STATE.json`**, não um HEAD fixado aqui. Uma
  matriz que fixa o HEAD da própria casa envelhece no commit seguinte.
- **A FASE 1 não fecha nenhuma capacidade do produto.** Ela fecha a *capacidade de priorizar*.

## PRÓXIMA AÇÃO QUE ESTA MATRIZ IMPÕE

Pela ordem das lacunas comprovadas com busca negativa e maior risco:

1. **C3** — ligar os validadores da E-1241 ao pipeline do Studio (lacuna: «provados por teste,
   não usados pelo produto»);
2. **C1** — matriz por target do Build Hub com pelo menos um build real instalado;
3. **B3** — os 12 E2E obrigatórios do Obsidian (FASE 17.1);
4. **A6** — receitas de ambiente versionadas por workspace.

Veredito global: **`NOT_RELEASE_READY`**.

## Medicoes consolidadas

As oito linhas abaixo passaram de heranca/estimativa a **medicao com evidencia no indice** (E-1480 a
E-1487). O estado `MEDIDO` quer dizer *lido no codigo e verificado por comando*, nao "feito".

| linha | o que a medicao diz |
| --- | --- |
| **B2** | deteccao de VRAM e agendamento **existem** (herdados); a **recomendacao** existe agora como funcao testada (`RecomendarParaVRAM`, `E-1489`) mas **nao esta' exposta** em `doctor` nem em rota; `doctor` continua sem VRAM |
| **B8** | `/goal` e comando real (UI + endpoint de goals); a **conversao conversa->goal com motivo explicado NAO existe** |
| **C1** | build e instalacao reais; `SigningProvider` **nao assina (proibido)** e implementa `AvaliarCertificado`; o certificado A1 da DZ23 **nao tem o EKU de Code Signing**; publicacao nao existe |
| **C2** | `ComputerUse` e componente instalavel, nao perfil de autonomia; **sem** botao de emergencia; allowlist e revogacao existem noutros objectos (OAuth, persona) e estao testadas |
| **C3** | validadores **ligados** (`builder.go` -> `ValidarSegurancaDosComponentes`), `sandbox` presente **e com teste de seguranca**; falta **CSP** |
| **C4** | `--json` de uma linha testado; contrato de exit codes documentado como **observado**; 11 testes da CLI passam |
| **C5** | **11** presets de missao, **19** testes, CLI `preset list/show` ligada ao catalogo |
| **C6** | superficie com tenant (`/api/agent/v1/*`) guardada por `authMiddleware`; superficie OpenAI (`/v1/*`) servida por middleware global **sem conceito de organizacao**, com bind por omissao em `127.0.0.1` |
