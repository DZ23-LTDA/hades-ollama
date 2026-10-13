# HADES — INVENTÁRIO DE REFERÊNCIAS (PARIDADE UNIVERSAL / BEST-OF-BREED)

**Origem:** ADENDO DE PARIDADE UNIVERSAL E BEST-OF-BREED, anexado ao HADES MASTER V2.2 (sem reiniciar o objetivo).
**Artefato exigido em:** §2 (inventário obrigatório) e §6 (matriz de paridade).
**Evidência de criação:** E-316.

---

## 0. Regras deste inventário (não são enfeite — são o critério)

1. **Nada aqui é claim de superioridade.** A meta de superar concorrentes só vale onde **métrica honesta** sustentar;
   resultado ausente é **`NAO_MEDIDO`**, nunca “ok”.
2. **Licença é `NAO_VERIFICADA` até ser lida.** Nenhum trecho de código de referência é copiado antes de verificar
   licença da raiz, subprojetos, assets, modelos e dependências. Marcar como verificado sem ler seria a mesma mentira
   que declarar capacidade sem teste.
3. **Comparar comportamento observável**, não aparência: workflows autorizados, entradas equivalentes, resultados
   auditáveis.
4. **`existe módulo` ≠ capacidade homologada.** As colunas de estado usam os estados canônicos do V2.2.
5. **Duplicatas contam como um registro** (ex.: `codenotch` aparece repetido na lista original ⇒ um único `ref_id`).
6. Referências **novas** só entram com justificativa, e sem deixar as anteriores sem triagem.

### Legenda

| Campo | Valores |
| --- | --- |
| **Estado no Hades** | `IMPLEMENTADO_E_TESTADO` · `IMPLEMENTADO_NAO_HOMOLOGADO` · `PARCIAL` · `APENAS_ADAPTER` · `APENAS_CATALOGO` · `AUSENTE` · `BLOCKED_BY_EXTERNAL_DEPENDENCY` · `NAO_MEDIDO` |
| **Abordagem** (§4) | **a** capacidade nativa Hades · **b** adaptador oficial (API/MCP/harness) substituível · **c** benchmark/referência sem dependência · **d** rejeitado com justificativa |
| **Prioridade** | P0 (gate) · P1 · P2 |
| **Licença** | `NAO_VERIFICADA` (padrão) até leitura efetiva da licença e das dependências |

> **Nota de honestidade sobre esta versão:** as colunas `estado_no_Hades` refletem **medição contra os arquivos e
> evidências do repositório** (E-001…E-315) nas capacidades que **já** foram auditadas. Onde não houve medição
> específica, está escrito **`NAO_MEDIDO`** — em vez de “provavelmente funciona”.

---

## 1. Produtos e plataformas (47 entradas)

| ref_id | referência | categoria | diferencial que interessa | estado no Hades | gap principal | abordagem | licença | prioridade |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| REF-P01 | Builder.io | app builder / visual | edição visual com componentes e publicação | `PARCIAL` (Product Builder + canvas com testes de artefato real) | preview seguro e deploy staging | b | NAO_VERIFICADA | P0-C |
| REF-P02 | FlutterFlow | app builder (mobile) | geração de app mobile a partir de UI visual | `AUSENTE` | geração mobile nativa | c | NAO_VERIFICADA | P2 |
| REF-P03 | GitHub Copilot Workspace | coding agent | issue → plano → PR orquestrado | `PARCIAL` (mesmo ciclo provado no próprio repo) | benchmark comparativo | c | NAO_VERIFICADA | P0-B |
| REF-P04 | OpenHands | coding agent OSS | sandbox de execução de agente de código | `AUSENTE` como harness | adaptador de sessão | b | NAO_VERIFICADA | P0-B |
| REF-P05 | Devin | agente autônomo de engenharia | execução longa com recuperação | `PARCIAL` (checkpoints/retomada provados) | benchmark de tarefa longa | c | NAO_VERIFICADA | P0-B |
| REF-P06 | Plandex | coding agent | planos versionados com diffs por etapa | `AUSENTE` | diff por hunk auditável | c | NAO_VERIFICADA | P1 |
| REF-P07 | Roo Code | coding agent (IDE) | modos de operação e ferramentas | `AUSENTE` como harness | adapter de harness | b | NAO_VERIFICADA | P1 |
| REF-P08 | Cline | coding agent (IDE) | aprovação por ação no editor | `PARCIAL` (approvals server-side já existem) | integração com ações do editor | b | NAO_VERIFICADA | P1 |
| REF-P09 | Aider | coding agent CLI | edição multi-arquivo por diff | `AUSENTE` como harness | adapter + benchmark | b | NAO_VERIFICADA | P0-B |
| REF-P10 | Claude Code | harness CLI | sessão real com streaming e cancelamento | `BLOCKED_BY_EXTERNAL_DEPENDENCY` (exige login) | login do operador | b | NAO_VERIFICADA | P0-B |
| REF-P11 | Void | editor IA | edição contextual | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P12 | Trae | IDE IA | fluxo de agente no editor | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P13 | Zed | editor | performance e edição colaborativa | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P14 | Windsurf | IDE agentic | cascata de ações com contexto | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P15 | Cursor | IDE agentic | edição multi-arquivo e revisão | `AUSENTE` como harness | benchmark | c | NAO_VERIFICADA | P0-B |
| REF-P16 | Databutton | app builder | app + dados gerenciados | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P17 | Marblism | app builder | geração full-stack rápida | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P18 | Create.xyz | app builder | prototipagem com componentes | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P19 | Replit Agent | cloud IDE + agente | ambiente hospedado com execução | `AUSENTE` (Hades é local-first por decisão) | — | c | NAO_VERIFICADA | P2 |
| REF-P20 | v0 (Vercel) | UI generator | geração de UI de alta qualidade | `PARCIAL` (Design System no Builder) | qualidade visual medida | c | NAO_VERIFICADA | P1 |
| REF-P21 | Base44 | app builder | app com auth/DB prontos | `PARCIAL` (auth/RBAC/DB existem) | staging real | c | NAO_VERIFICADA | P1 |
| REF-P22 | Bolt.diy | app builder OSS | execução local de geração | `PARCIAL` | staging + export | c | NAO_VERIFICADA | P1 |
| REF-P23 | Bolt.new | app builder | preview imediato no browser | `PARCIAL` | preview seguro | c | NAO_VERIFICADA | P1 |
| REF-P24 | Lovable | app builder | app full-stack com deploy | `PARCIAL` | deploy staging + rollback (T3) | c | NAO_VERIFICADA | P0-C |
| REF-P25 | Perplexity | pesquisa | resposta com **fontes rastreáveis** | `PARCIAL` (pesquisa existe; falta dedupe/data da consulta) | fontes + data + dedupe | c | NAO_VERIFICADA | P0-A |
| REF-P26 | Polsia | agentes/negócios | operação empresarial assistida | `PARCIAL` (Company Factory + supervisor) | E2E de empresa (T7) | c | NAO_VERIFICADA | P1-B |
| REF-P27 | Manus | agente universal | missão E2E com navegador e artefatos | `PARCIAL` (jornada E2E de negócio provada; navegador não) | navegador autorizado + artefatos (T1) | c | NAO_VERIFICADA | **P0-A** |
| REF-P28 | OmniRoute | roteamento multi-provedor | roteamento/custo entre provedores | `PARCIAL` (Model Gateway com fallback/custo) | benchmark de roteamento | c | NAO_VERIFICADA | P0-D |
| REF-P29 | DeepSeek | modelo/API | modelos de código e raciocínio | `IMPLEMENTADO_NAO_HOMOLOGADO` (provider configurável; chave de teste já expirou) | credencial válida | b | NAO_VERIFICADA | P0-D |
| REF-P30 | Codex | harness CLI | engenharia autônoma em repo | `BLOCKED_BY_EXTERNAL_DEPENDENCY` (login) | login do operador | b | NAO_VERIFICADA | **P0-B** |
| REF-P31 | ChatGPT | modelo/assistente | tool calling e raciocínio | `IMPLEMENTADO_NAO_HOMOLOGADO` (API compatível) | credencial + tool calling medido | b | NAO_VERIFICADA | P0-D |
| REF-P32 | Gemini | modelo/API | contexto longo e multimodal | `IMPLEMENTADO_NAO_HOMOLOGADO` | credencial válida | b | NAO_VERIFICADA | P0-D |
| REF-P33 | Hermes Agent | agente | execução de tarefas com memória | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P34 | Hercules.app | agente desktop | automação de desktop | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P35 | Verdent.ai | agentes | orquestração de tarefas | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P36 | SWE-agent (Princeton) | agente de código | resolução de issues com ferramentas | `AUSENTE` como harness | benchmark (T2) | c | NAO_VERIFICADA | P0-B |
| REF-P37 | MetaGPT | multiagente | papéis de empresa em software | `PARCIAL` (agentes especializados existem) | E2E de 3 especialistas (T4) | c | NAO_VERIFICADA | **P0-E** |
| REF-P38 | ChatDev | multiagente | pipeline de papéis | `PARCIAL` | T4 | c | NAO_VERIFICADA | P0-E |
| REF-P39 | AutoGen | multiagente | conversa entre agentes com ferramentas | `PARCIAL` | T4 | c | NAO_VERIFICADA | P0-E |
| REF-P40 | CrewAI | multiagente | times com tarefas e delegação | `PARCIAL` | T4 | c | NAO_VERIFICADA | P0-E |
| REF-P41 | Continue.dev | IDE assistant | contexto de código no editor | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P42 | Ollama | motor local | execução local de modelos | `IMPLEMENTADO_E_TESTADO` (**motor padrão**, preservado) | — | a | MIT (upstream) — **reverificar** | P0-D |
| REF-P43 | vLLM | servidor de inferência | throughput alto | `APENAS_ADAPTER` (endpoint compatível) | teste real com provedor | b | NAO_VERIFICADA | P1 |
| REF-P44 | E2B | sandbox de execução | sandbox remoto para código | `PARCIAL` (sandbox local governado) | sandbox remoto é dependência externa | b | NAO_VERIFICADA | P1 |
| REF-P45 | Grok Bot | bot/agente | interação em tempo real | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P46 | Blink | dev tool | produtividade de engenharia | `AUSENTE` | — | c | NAO_VERIFICADA | P2 |
| REF-P47 | Qoder | IDE/agente | compreensão de codebase | `AUSENTE` | mapa arquitetural (existe parcial) | c | NAO_VERIFICADA | P2 |
| REF-P48 | TypingMind | interface de chat | UX de chat multi-modelo | `PARCIAL` (UI própria em pt-BR) | polimento de UX | c | NAO_VERIFICADA | P1 |

---

## 2. Repositórios open source (22 entradas)

| ref_id | repositório | categoria | diferencial | estado no Hades | abordagem | licença | prioridade |
| --- | --- | --- | --- | --- | --- | --- | --- |
| REF-R01 | Dpro-at/Tel-Agent | telefonia/Voz | STT/TTS + chamadas com ferramentas | `APENAS_ADAPTER` (voz e chamada ativa governadas; sem provedor) | b (**processo separado**) | ⚠️ **AGPL-3.0 lida** (`be3f7b28…`) — copyleft de rede (§13) | P1-D |
| REF-R02 | wickedapp/openclaw-office | suíte office | documentos/planilhas | `AUSENTE` | c | NAO_VERIFICADA | P2 |
| REF-R03 | block/buzz | — | infraestrutura/ferramenta | `AUSENTE` | c | NAO_VERIFICADA | P2 |
| REF-R04 | lemonade-sdk/lemonade | inferência local | NPU/GPU local | `APENAS_ADAPTER` (endpoint compatível) | b | NAO_VERIFICADA | P1 |
| REF-R05 | agentconnect-md/agentconnect | agentes | interconexão de agentes | `AUSENTE` | c | NAO_VERIFICADA | P2 |
| REF-R06 | nexu-io/open-design | design | design system/canvas | `PARCIAL` (canvas com testes) | c | NAO_VERIFICADA | P2 |
| REF-R07 | renatoasse/opensquad | multiagente | squads de agentes | `PARCIAL` | T4 | c | NAO_VERIFICADA | P0-E |
| REF-R08 | vinzdg/codenotch | (duplicata na lista) | — | `NAO_MEDIDO` | c | NAO_VERIFICADA | P2 |
| REF-R09 | MakazhanAlpamys/Soup | fine-tuning | treino leve de modelos | `AUSENTE` | c | NAO_VERIFICADA | P2 |
| REF-R10 | dyad-sh/dyad | app builder local | builder OSS local-first | `PARCIAL` | c | NAO_VERIFICADA | P1 |
| REF-R11 | BerriAI/litellm | gateway de modelos | proxy multi-provedor | `PARCIAL` (gateway próprio) | b | NAO_VERIFICADA | P0-D |
| REF-R12 | langgenius/dify | automação/LLM apps | workflow + agentes + UI | `PARCIAL` (workflow com retry/DLQ) | c | NAO_VERIFICADA | P1-A |
| REF-R13 | HarnessRouter/harnessrouter | harness | roteamento de harnesses | `PARCIAL` (prompt routing existe) | b | NAO_VERIFICADA | P0-B |
| REF-R14 | decolua/9router | roteamento | roteamento/custo | `PARCIAL` | c | NAO_VERIFICADA | P1 |
| REF-R15 | tldraw/tldraw | canvas | quadro infinito colaborativo (licença **deve** ser lida) | `AUSENTE` | c | NAO_VERIFICADA | P2 |
| REF-R16 | xyflow/xyflow | canvas | grafos de nós (React Flow) | `AUSENTE` | c | NAO_VERIFICADA | P2 |
| REF-R17 | n8n-io/n8n | automação | nós reais + retry + DLQ | `PARCIAL` | c | NAO_VERIFICADA | P1-A |
| REF-R18 | agentscope-ai/agentscope-java | multiagente | orquestração de agentes | `PARCIAL` | c | NAO_VERIFICADA | P0-E |
| REF-R19 | langchain-ai/langgraph | multiagente | grafos com estado | `PARCIAL` | c | NAO_VERIFICADA | P0-E |
| REF-R20 | TypingMind/typingmind | UI de chat | UX multi-modelo | `PARCIAL` | c | NAO_VERIFICADA | P1 |
| REF-R21 | OleLehmann/youtube-automation-agent (TUBEGENT) | creator/YouTube | 7 agentes: roteiro→vídeo→upload | `AUSENTE` | **b/a** (seletivo, com atribuição) | ✅ **MIT lida** (`de3dcc3f…`) | P1-E |
| REF-R22 | morluto/rea | engenharia reversa | MCP + skill de análise | `AUSENTE` | **b** (MCP/skill, não vendorização) | ✅ **MIT lida** (`2d476d24…`) | P1-F |

> **Alerta herdado (V2.2 §8):** no TUBEGENT foi reportado que `getVideoStream` devolvia **JSON de simulação** e o
> fallback multimídia podia produzir resultados simulados. **Nenhum desses caminhos conta como sucesso** no Hades:
> exigimos MP4 válido, com codec/duração/áudio testáveis.

---

## 3. Estado consolidado dos gates (medido, não presumido)

| Gate | Estado | O que falta, nomeado |
| --- | --- | --- |
| **P0-A** Manus parity | `PARCIAL` | navegador/computer-use autorizado + produção de artefatos + T1 |
| **P0-B** Coding harness | `PARCIAL` | benchmark comparativo real (Codex/Claude Code/OpenHands/Aider/SWE-agent/Cursor) + T2 |
| **P0-C** Builder | `PARCIAL` | staging real (DB real) + preview seguro + rollback + T3 |
| **P0-D** Model & harness | `PARCIAL` | Ollama Cloud (login), credencial externa válida, tool calling medido por provedor + T5 |
| **P0-E** Multiagente | `PARCIAL` | E2E de **3 especialistas em worktrees isolados** com merge controlado + T4 |
| **P0-F** Segurança e entrega | `PARCIAL` | isolamento já provado em várias camadas; falta release por SO e **assinatura** |
| **P1-A** Automação | `PARCIAL` | T6 (webhook → condição → ação real → retry/idempotência) |
| **P1-B** Empresa universal | `PARCIAL` | onboarding de empresa **existente** (importação/mapeamento) |
| **P1-C** Agendamento universal | `IN_PROGRESS` | E-313/E-314 entregues; faltam intervalos no turno, lista de espera, remarcação e sync autorizada |
| **P1-D** Canais reais | `BLOCKED_BY_EXTERNAL_DEPENDENCY` | tokens Meta/Telegram, SIP, contas — roteiro em §3 do plano V2.2 |
| **P1-E** Creator Studio | `NAO_INICIADO` | auditoria do TUBEGENT + T8 (MP4 decodificável) |
| **P1-F** REA | `NAO_INICIADO` | integração MCP/skill + T9 sobre artefato próprio |
| **P1-G** Pesquisa | `PARCIAL` | fontes rastreáveis, data da consulta, dedupe e defesa contra prompt injection |
| **P1-H** Descoberta (SEO/GEO) | `NAO_INICIADO` | README PT/EN, docs versionadas, sitemap, JSON-LD — **sem prometer ranking** |
| **P2** Canvas/IDE/voz/fine-tuning | `PARCIAL`/`NAO_INICIADO` | por último, com autorização |

---

## 4. Próxima ação imediata

1. **Verificar licenças** das referências que serão efetivamente integradas **antes** de qualquer cópia —
   prioridade: `REF-R21` (TUBEGENT), `REF-R22` (REA), `REF-R01` (Tel-Agent), `REF-R11` (litellm), `REF-R15/R16` (canvas).
   Enquanto não forem lidas, permanecem `NAO_VERIFICADA` e **nenhum trecho é copiado**.
2. **P0-B/T2**: benchmark de tarefa equivalente (bug real em repo fixture) entre harnesses — sem métrica, `NAO_MEDIDO`.
3. **P0-E/T4**: três especialistas em worktrees isolados com merge controlado.
4. **P1-C**: próxima fatia do agendamento (intervalos no turno e remarcação), mantendo o padrão de teste-antes.

**Este inventário é vivo:** cada linha passa a `IMPLEMENTADO_E_TESTADO`/`HOMOLOGADO` **apenas** com evidência
reproduzível (comando + ambiente + SHA) registrada em `HADES_EVIDENCE_INDEX.md`.
