# HADES — MATRIZ DE CAPACIDADES COMPETITIVAS

> **Documento vivo.** Decompõe cada domínio em **capacidades atómicas** (`CAP-ID`) e associa
> cada referência apenas ao que foi **confirmado por fonte primária** — *nunca ao produto
> inteiro*. Marca de um produto ≠ capacidade desse produto.
>
> **Regra de honestidade**: uma capacidade só recebe `SUPERIOR_MEASURED` ou `PARITY_MEASURED`
> com **benchmark reproduzível**. Sem acesso legal/técnico ao concorrente, o estado é
> `CLAIM_ONLY` ou `NOT_MEASURED`. *Marketing não é benchmark.*

**Gerado em**: 2026-10-16 (BRT) · **branch** `fix/checker-relogio` · **HEAD** `1aefee17c`
**Nota**: este documento é **fonte** de `HADES_COMPETITIVE_CAPABILITY_MATRIX.csv` (mesma data).
Ele **não** substitui `docs/mission/HADES_EVIDENCE_INDEX.md`, que é a trilha de evidências.

---

## 1. Estado dos 5 clones medidos (proveniência verificada)

Cada clone abaixo foi **clonado localmente**, e os números foram **medidos** — não copiados
de descrição. `_tmp/thirdparty/` está em `.gitignore` **e** `.dockerignore`.

| # | Repositório | owner | commit | data do commit | licença | `SKILL.md` | `SECURITY` |
|---|---|---|---|---|---|---|---|
| 1 | `addyosmani/agent-skills` | addyosmani | `aeac41d` | 2026-10-13 | **MIT** | **25** | não |
| 2 | `heygen-com/hyperframes` | heygen-com | `4d6b5be` | 2026-10-16 | **Apache-2.0** | **40** | sim |
| 3 | `googleworkspace/cli` | googleworkspace | `a3768d0` | 2026-03-31 | **Apache-2.0** | **95** | sim |
| 4 | `apify/apify-mcp-server` | apify | `d161bc4` | 2026-10-16 | **MIT** | 3 | sim |
| 5 | `microsoft/playwright-mcp` | microsoft | `b8b4183` | 2026-10-08 | **Apache-2.0** | 0 | sim |

**Total medido: 163 `SKILL.md` nas 5 fontes**, contra **42** no HADES (`skills/`).
⇒ *Isso é uma **medida de volume de catálogo**, não de qualidade nem de paridade.*

**Distribuição de licenças**: MIT ×2, Apache-2.0 ×3. **Nenhuma copyleft** — todas compatíveis
com o HADES (MIT). *Atribuição obrigatória nos 5 casos; `NOTICE` obrigatório no Apache-2.0.*

**Ficheiros de licença encontrados** (medidos por nome, no topo de cada clone):
`agent-skills/LICENSE` · `apify-mcp-server/LICENSE.md` + `SECURITY.md` ·
`googleworkspace-cli/LICENSE` + `SECURITY.md` · `hyperframes/LICENSE` + `SECURITY.md` ·
`playwright-mcp/LICENSE` + `SECURITY.md`.

⇒ **4 de 5** têm `SECURITY.md`; o único sem é o `agent-skills`. *Ler o `SECURITY.md` de cada
um é passo obrigatório antes de qualquer integração.*

---

## 2. Como ler os estados

| estado | significado |
|---|---|
| `SUPERIOR_MEASURED` | o HADES venceu critérios pré-definidos, com benchmark reproduzível |
| `PARITY_MEASURED` | mesmo resultado dentro da tolerância declarada |
| `IMPLEMENTED_UNMEASURED` | existe e funciona, **sem** benchmark contra a referência |
| `PARTIAL` | existe incompleto (falta caminho de utilizador, teste ou wiring) |
| `ABSENT_APPROVED` | não existe e **pertence ao escopo aprovado** |
| `OUT_OF_SCOPE` | não pertence ao escopo aprovado (com motivo) |
| `BLOCKED_EXTERNAL` | depende de credencial/conta/dispositivo indisponível |
| `CLAIM_ONLY` | a referência **afirma** e não demonstra; ou não houve acesso para medir |

---

## 3. Domínio A — Engenharia autónoma fim a fim

| CAP-ID | capacidade | referências (fonte primária) | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-A-01` | ingestão de issue/especificação | OpenHands, SWE-agent, Copilot Workspace | MIT/Apache | `PARTIAL` | `internal/agent/spec*`, `/goal` | falta ligar a issue tracker real — **integrar** |
| `CAP-A-02` | planeamento hierárquico editável | MetaGPT, Plandex, Devin | MIT/Apache | `IMPLEMENTED_UNMEASURED` | `internal/agent` planner | falta edição pré-execução na UI — **integrar** |
| `CAP-A-03` | indexação de repos grandes | Aider, Cursor, Zed | Apache/MIT | `IMPLEMENTED_UNMEASURED` | índice local | falta medir latência — **medir** |
| `CAP-A-04` | edição multi-arquivo + patch mínimo | Aider, Cline, Roo Code | Apache | `IMPLEMENTED_UNMEASURED` | `ContarHunksDoDiff` | **medido em fixtures** (`E-344`) |
| `CAP-A-05` | terminal + auto-correção limitada | OpenHands, Devin | MIT | `PARTIAL` | `sandboxExecTool` | *o loop real existe* (`specialist_repair_loop.go`) — **medir** |
| `CAP-A-06` | testes gerados + fault injection | SWE-agent | MIT | `IMPLEMENTED_UNMEASURED` | `harness_benchmark.go` | fixtures T2 medidos |
| `CAP-A-07` | checkpoints / rollback / worktrees | Devin, Claude Code | proprietária (`CLAIM_ONLY`) | `PARTIAL` | `hades_resume.ps1` | *sem worktree isolado* — **implementar** |
| `CAP-A-08` | filas / orçamento / timeout / cancelamento | Manus, Devin | proprietária (`CLAIM_ONLY`) | `IMPLEMENTED_UNMEASURED` | `internal/agent` queue | *a cura do `E-968` **mediu** o orçamento* |
| `CAP-A-09` | revisão de código + preparação de PR | CodeRabbit, Copilot | proprietária | `PARTIAL` | 12 findings do PR #438 | *sem publicação automática (correto)* |
| `CAP-A-10` | aprovação humana + trilha de decisão | todos | várias | **`PARITY_MEASURED`** | `service_agent_approvals.go` + P0 `E-960` | **aprovado uma vez não executa duas** — teste 5/5 |

## 4. Domínio B — IDE/workspace e inteligência de código

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-B-01` | índice incremental com exclusões | Cursor, Windsurf, Trae | proprietária | `IMPLEMENTED_UNMEASURED` | índice local | falta apagamento verificável — **implementar** |
| `CAP-B-02` | busca semântica / símbolos / referências | Cursor, Continue, Zed | Apache | `PARTIAL` | busca local | *o `E-964` mediu 6 falsos negativos de padrão* |
| `CAP-B-03` | autocomplete de baixa latência | Cursor, Windsurf | proprietária | `ABSENT_APPROVED` | — | **decisão: adiar** (exige modelo dedicado) |
| `CAP-B-04` | chat + diff + apply/revert parcial | Cline, Roo Code, Aider | Apache | `IMPLEMENTED_UNMEASURED` | `app/ui/app` | *revert parcial sem E2E* |
| `CAP-B-05` | contexto de editor/terminal/browser/Git | Cursor, Windsurf | proprietária | `PARTIAL` | `browser.operator` | **a FASE 3-B fecha** |
| `CAP-B-06` | memória por workspace + compactação | Continue, Zed | Apache | `IMPLEMENTED_UNMEASURED` | missions | *sem limite medido* |
| `CAP-B-07` | LSP/DAP + múltiplas linguagens | Void, Zed | Apache | `OUT_OF_SCOPE` | — | *o HADES não é IDE* |
| `CAP-B-08` | startup / RAM / CPU / índice medidos | Zed, Void | Apache | `ABSENT_APPROVED` | — | **medir** (o `§ESTÁGIO 21` exige) |

## 5. Domínio C — Studio/Builder full-stack

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-C-01` | prompt → app funcional | v0, Bolt.new, Lovable | proprietária | `PARTIAL` | `internal/agent/builder*` | *demo estática não conta* — **implementar** |
| `CAP-C-02` | canvas visual + edição bidirecional | Builder.io, FlutterFlow, tldraw, xyflow | MIT (tldraw/xyflow) | `PARTIAL` | builder routes | *sem edição bidirecional* |
| `CAP-C-03` | nós/grafos com validação | n8n, xyflow, LangGraph | MIT | `IMPLEMENTED_UNMEASURED` | `builder` graph | — |
| `CAP-C-04` | frontend+backend+DB+auth+testes | Replit, Databutton | proprietária | `ABSENT_APPROVED` | — | **implementar** (`§ESTÁGIO 12`) |
| `CAP-C-05` | import de design/imagem | Builder.io, FlutterFlow | proprietária | `ABSENT_APPROVED` | — | **adiar** (exige visão) |
| `CAP-C-06` | preview / hot reload / snapshots / rollback | Bolt, Replit | proprietária | `PARTIAL` | preview routes | — |
| `CAP-C-07` | exportação sem lock-in | Bolt.diy, Dyad | MIT | `PARTIAL` | export builder | *o `E-941` mediu BUNDLE_BUDGET=PASS* |
| `CAP-C-08` | publicação só com aprovação | todos | várias | **`PARITY_MEASURED`** | `requestDeploymentApproval` | *aprovado, secrets fora do artefacto* |

## 6. Domínio D — Mission Control e multiagentes

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-D-01` | decomposição em papéis + dependências | MetaGPT, CrewAI, AutoGen | MIT | `IMPLEMENTED_UNMEASURED` | `internal/agent` planner | — |
| `CAP-D-02` | grafos duráveis + branches/joins | LangGraph, Dify | MIT/Apache | `PARTIAL` | missions | *sem compensação* — **implementar** |
| `CAP-D-03` | delegação com contrato e orçamento | Paperclip, Teamily | `CLAIM_ONLY` | `PARTIAL` | `HERMES_TASK_DIRECTIVE` | *o YAML **não** prova execução* |
| `CAP-D-04` | comunicação por eventos/mailbox | AutoGen, AgentScope | MIT | `IMPLEMENTED_UNMEASURED` | events + SSE | — |
| `CAP-D-05` | HITL + escalonamento + separação | todos | várias | **`PARITY_MEASURED`** | approvals + roles | — |
| `CAP-D-06` | equipas/backlog/metas | OpenClaw Office, Paperclip | a confirmar | `PARTIAL` | `company_*` | — |
| `CAP-D-07` | memória escopada (user/org/mission) | Dify, Continue | Apache | **`PARITY_MEASURED`** | tenant scope check | *o `E-967` **declarou** um ramo inalcançável* |
| `CAP-D-08` | prevenção de órfãos/loops/deadlock | todos | várias | `IMPLEMENTED_UNMEASURED` | `§ANTILOOP` | *a cura do `E-968` é prova* |
| `CAP-D-09` | visualização por eventos reais + pet | — | — | `PARTIAL` | `companion_*` | **a FASE 3-A fecha** |
| `CAP-D-10` | avaliação de contribuição / reexecução | Devin | proprietária | `PARTIAL` | `specialist_repair_loop` | — |

## 7. Domínio E — Modelos, roteamento e inferência soberana

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-E-01` | Ollama default **offline** | Ollama | MIT | **`PARITY_MEASURED`** | `E-936` (727 testes + 13 E2E) | *motor principal confirmado* |
| `CAP-E-02` | vLLM / compatíveis opcionais | vLLM | Apache | `PARTIAL` | provider registry | — |
| `CAP-E-03` | catálogo/capabilities/tools/visão | Ollama, vLLM | MIT/Apache | `IMPLEMENTED_UNMEASURED` | model catalog | — |
| `CAP-E-04` | roteamento por custo/privacidade/latência | LiteLLM, HarnessRouter, 9router | MIT | `PARTIAL` | `HarnessRouter` | *o `E-963` mediu 31 configs MCP* |
| `CAP-E-05` | pins estritos + fallback + circuit breaker | LiteLLM | MIT | `IMPLEMENTED_UNMEASURED` | `§FASE 4` item 3 | *pin estrito sem troca silenciosa* |
| `CAP-E-06` | cache exato/semântico com isolamento | LiteLLM, vLLM | MIT/Apache | `IMPLEMENTED_UNMEASURED` | `cache_trie_test.go` | — |
| `CAP-E-07` | quotas / rate limits / custos / redaction | LiteLLM | MIT | `IMPLEMENTED_UNMEASURED` | `RedactDLP` | — |
| `CAP-E-08` | benchmark warm/cold reproduzível | — | — | **`IMPLEMENTED_UNMEASURED`** | `harness_benchmark.go` | *o `E-968` **mediu** o orçamento: 4 testes, 90 min* |
| `CAP-E-09` | gestão de modelos / pull / quantização | Ollama | MIT | **`PARITY_MEASURED`** | 9 modelos locais medidos | — |

## 8. Domínio F — Sandboxes, computador e navegador

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-F-01` | execução isolada com limites | E2B | Apache | `PARTIAL` | `sandboxExecTool` | *sem contentor* — **implementar** |
| `CAP-F-02` | imagens reproduzíveis / snapshots | E2B | Apache | `ABSENT_APPROVED` | — | **implementar** |
| `CAP-F-03` | terminal + code interpreter | E2B, Cloudroom | Apache/`CLAIM_ONLY` | **`PARITY_MEASURED`** | `E-953` (smoke em `ubuntu:24.04`) | *medido em contentor real* |
| `CAP-F-04` | automação por DOM/acessibilidade | Playwright | Apache | **`PARITY_MEASURED`** | `browser.operator` + 1 teste | — |
| `CAP-F-05` | downloads/uploads/abas/perfis | Playwright | Apache | `PARTIAL` | `browser_environment.go` | — |
| `CAP-F-06` | desktop remoto + takeover humano | Cloudroom GUI/Core | a confirmar | `PARTIAL` | `desktop.companion` | **a FASE 3-B fecha** |
| `CAP-F-07` | bloqueio SSRF/escape/injection/exfiltração | E2B, Playwright | Apache | `IMPLEMENTED_UNMEASURED` | `preparePluginWorkspace` | *travessia/symlink/zip bomb cobertos* |
| `CAP-F-08` | políticas por missão | todos | várias | `PARTIAL` | `CapabilityPolicy` | — |
| `CAP-F-09` | gravação/replay auditável | Cloudroom | a confirmar | `PARTIAL` | audit log | — |

## 9. Domínio G — Pesquisa, conhecimento e artefactos

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-G-01` | pesquisa web com citações | Perplexity, Genspark | proprietária | `PARTIAL` | `web_search` tool | *sem benchmark de grounding* |
| `CAP-G-02` | navegador de docs/código | ChatGPT, Gemini | proprietária | `IMPLEMENTED_UNMEASURED` | `web_fetch` | — |
| `CAP-G-03` | deep research interrompível | Manus, Polsia | `CLAIM_ONLY` | `ABSENT_APPROVED` | — | **implementar** |
| `CAP-G-04` | RAG local/híbrido + permissões | Dify, Continue | Apache | `IMPLEMENTED_UNMEASURED` | `agent_reembed_job` | — |
| `CAP-G-05` | memória universal escopada | ChatGPT, Gemini | proprietária | `PARTIAL` | missions | *o `E-967` declara o limite tenant* |
| `CAP-G-06` | relatórios/páginas/artefactos | Genspark, Manus | `CLAIM_ONLY` | `IMPLEMENTED_UNMEASURED` | artifacts + Mission Control | — |
| `CAP-G-07` | distinguir fonte/inferência/hipótese | — | — | **`PARITY_MEASURED`** | todo este índice | *o índice marca `NAO_MEDIDO` explicitamente* |
| `CAP-G-08` | avaliação de cobertura/frescor/grounding | Perplexity | proprietária | `ABSENT_APPROVED` | — | **medir** |
| `CAP-G-09` | `morluto/rea` (**correção factual**) | `morluto/rea` | **a confirmar** | `OUT_OF_SCOPE` | — | ⚠️ *a ênfase observada é **engenharia reversa** de software, **não** deep research web* → vai para o **Domínio J** (`CAP-J-01`) |

## 10. Domínio H — Automação, conectores e operações

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-H-01` | editor de workflows + gatilhos | n8n, Dify | Apache | `PARTIAL` | builder routes | — |
| `CAP-H-02` | conectores tipados OAuth + least privilege | n8n | Apache | **`PARITY_MEASURED`** | `connector_catalog_status_test.go` | — |
| `CAP-H-03` | retries idempotentes + dead-letter | n8n | Apache | `IMPLEMENTED_UNMEASURED` | `queue.go` | — |
| `CAP-H-04` | Telegram/WhatsApp | n8n | Apache | `BLOCKED_EXTERNAL` | `whatsapp_test.go` | *token real* — **desbloqueio: token do dono** |
| `CAP-H-05` | **telefonia/SIP** (`Dpro-at/Tel-Agent`, **correção factual**) | `Dpro-at/Tel-Agent` | **AGPL-3.0** (`E-1136`) | `ABSENT_APPROVED` | `E-1136`/`E-1137` | ⚠️ *a competência central é **telefonia/agentes, SIP** — não «controle por Telegram/WhatsApp»*. **Não** é `OUT_OF_SCOPE`: existe o **estágio 37** dedicado a implementá-la e o §FASE-8 mantém-na como externa por **número SIP real**, não fora de escopo. O código do Tel-Agent é **AGPL-3.0 e não pode ser copiado**; as bibliotecas por baixo (`livekit/agents`, `livekit/livekit` = **Apache-2.0**; `aiortc` = **BSD-3-Clause**) **são compatíveis com MIT**, logo implementação própria é viável. Falta **implementação**, não autorização. |
| `CAP-H-06` | integrações Git/cloud/DB/CRM | n8n, Dify | Apache | `PARTIAL` | `connector_*` | — |
| `CAP-H-07` | execução local/self-hosted + MCP | Dify, n8n | Apache | **`PARITY_MEASURED`** | `E-966` (rota de execução) | *o `mcp_transport_test.go` cobre* |
| `CAP-H-08` | import/export + testes de contrato | n8n | Apache | `IMPLEMENTED_UNMEASURED` | contract tests | — |

## 11. Domínio I — Multimodalidade e Creator Studio

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-I-01` | texto/imagem/áudio/vídeo plugáveis | heygen-com/hyperframes | Apache-2.0 | `PARTIAL` | `media.go` (39 KB) | *sem provedor real medido* |
| `CAP-I-02` | transcrição/diarização local | hyperframes | Apache-2.0 | `PARTIAL` | `media_pipeline.go` | *`ffmpeg` ausente* — **instalável** |
| `CAP-I-03` | edição temporal + legendas + tradução | hyperframes | Apache-2.0 | `ABSENT_APPROVED` | — | **implementar** |
| `CAP-I-04` | pipeline com roteiro→render→export | `OleLehmann/youtube-automation-agent` | **a confirmar** | `PARTIAL` | `media_process_approval.go` | — |
| `CAP-I-05` | proveniência + direitos + marca d'água | — | — | `PARTIAL` | `ArtifactManifest` | — |
| `CAP-I-06` | `block/buzz` (**correção factual**) | `block/buzz` | **a confirmar** | `OUT_OF_SCOPE` | — | ⚠️ ***não** é transcritor offline: é **colaboração humano↔agente e eventos assinados*** → **`CAP-D-11`** |
| `CAP-I-07` | automação de canal só com confirmação | youtube-automation-agent | a confirmar | `PARTIAL` | `media_process_approval` | — |

## 12. Domínio J — Engenharia reversa, fine-tuning e avaliação

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-J-01` | inspeção legal de apps/binários (`morluto/rea`) | `morluto/rea` | **a confirmar** | `OUT_OF_SCOPE` | — | *o HADES não faz RE; **e não deve** sem autorização* |
| `CAP-J-02` | análise estática/dinâmica isolada | `morluto/rea` | a confirmar | `PARTIAL` | `sandboxExecTool` | *contentor disponível (`E-953`)* |
| `CAP-J-03` | datasets versionados + consentimento | `MakazhanAlpamys/Soup` (**correção**) | **a confirmar** | `ABSENT_APPROVED` | — | *Soup = **fine-tuning**, não áudio* |
| `CAP-J-04` | fine-tuning/LoRA reproduzível | Soup | a confirmar | `ABSENT_APPROVED` | — | **implementar** (`§ESTÁGIO J`) |
| `CAP-J-05` | evals por tarefa + regressão + custo | — | — | **`IMPLEMENTED_UNMEASURED`** | `harness_benchmark.go` | *o `E-968` mediu 4 testes de modelo* |
| `CAP-J-06` | registo de modelo/dataset + rollback | — | — | `PARTIAL` | artefacts | — |
| `CAP-J-07` | `vinzdg/codenotch` (**correção factual**) | `vinzdg/codenotch` | **a confirmar** | `OUT_OF_SCOPE` | — | ⚠️ *é **monitor de limites/uso** de Codex/Claude Code — **não** um IDE* → **`CAP-K-10`** |

## 13. Domínio K — Segurança, governança e confiabilidade

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-K-01` | RBAC/ABAC + tenant isolation | todos | várias | **`PARITY_MEASURED`** | `requirePluginAdmin`, tenant scope | *o `E-967` declara um ramo inalcançável* |
| `CAP-K-02` | vault/secrets + redaction + rotação | — | — | **`PARITY_MEASURED`** | `RedactDLP`, `app/secrets` | — |
| `CAP-K-03` | policy engine + approvals de uso único | — | — | **`PARITY_MEASURED`** | `E-960` (P0 do PR #438) | *5/5 + race ×3* |
| `CAP-K-04` | supply-chain: SBOM + SCA + provenance | — | — | `PARTIAL` | `HADES_SBOM_*.csv` | *`govulncheck` **ausente*** — **instalável** |
| `CAP-K-05` | audit log resistente a adulteração | — | — | `PARTIAL` | audit trail | — |
| `CAP-K-06` | backup/restore + DR + migrações | — | — | `PARTIAL` | — | — |
| `CAP-K-07` | atualização segura + rollback | — | — | `PARTIAL` | — | — |
| `CAP-K-08` | proteção contra prompt injection | — | — | **`PARITY_MEASURED`** | `preparePluginWorkspace` | *conteúdo do bundle é não confiável* |
| `CAP-K-09` | `block/buzz` (**eventos assinados**) | `block/buzz` | a confirmar | `OUT_OF_SCOPE` | — | *ver `CAP-I-06`* |
| `CAP-K-10` | monitor de limites/uso (`vinzdg/codenotch`) | `vinzdg/codenotch` | a confirmar | `ABSENT_APPROVED` | — | **implementar** (leve) |
| `CAP-K-11` | `Damienchakma/Open-claude` (**correção**) | `Damienchakma/Open-claude` | **a confirmar** | `OUT_OF_SCOPE` | — | ⚠️ *é **workspace multimodal** com pesquisa e artefactos — **não** sandbox do sistema* |
| `CAP-K-12` | `noemica-io/open-claude-in-chrome` | `noemica-io/open-claude-in-chrome` | a confirmar | **`OUT_OF_SCOPE`** | — | ⚠️ *técnicas que **removem barreiras de segurança de terceiros** ficam **fora** — o prompt diz explicitamente* |

## 14. Domínio L — Produto único e distribuição

| CAP-ID | capacidade | referências | licença | HADES | prova HADES | lacuna / decisão |
|---|---|---|---|---|---|---|
| `CAP-L-01` | um composer com `/goal` | todos | várias | `PARTIAL` | `ChatForm.tsx`, `slashCommands.ts`, `E-1138`, `E-1140`, `E-1141` | **Tres provas**: (a) 13 testes unitarios (`E-1138`); (b) E2E chromium -- composer+sidebar+mobile console, 2 passed 58,1 s (`E-1140`); (c) **jornada de primeira execucao**, 1 passed 38,9 s (`E-1141`) -- instalacao, onboarding, missao, artefacto, aprovacao, cancelamento e reinicio. **Lacuna precisa**: a jornada cria a missao com **texto normal**, nao com `/goal` -- o comando esta' provado em teste unitario, nao em navegador. |
| `CAP-L-02` | um Mission Control + um menu | — | — | `PARTIAL` | missions | — |
| `CAP-L-03` | onboarding local-first | — | — | `PARTIAL` | onboarding gate | — |
| `CAP-L-04` | Windows/Linux/macOS/Android/iOS/Web | — | — | `PARTIAL` | `E-938` (`HadesSetup.exe`), `E-953` (Linux), `E-955` (Android prebuild) | *sem APK, sem macOS, sem iOS* |
| `CAP-L-05` | atualizações + diagnóstico + a11y + i18n | — | — | `PARTIAL` | — | — |
| `CAP-L-06` | export/import de projetos e dados | — | — | `PARTIAL` | — | — |
| `CAP-L-07` | instalação/desinstalação/upgrade testados | — | — | `PARTIAL` | `E-938` NotSigned | *sem execução num perfil limpo* |
| `CAP-L-08` | documentação user/admin/dev/auditor | — | — | `PARTIAL` | `docs/` | *o pacote de auditoria **ainda não** deve ser criado* |

---

## 15. Referências nominais — cobertura obrigatória

Todas as referências do prompt, com o estado do **acesso** (não do produto). *Uma linha por
referência garante que nenhuma foi silenciosamente ignorada.*

| referência | acesso medido | onde entra |
|---|---|---|
| Builder.io · FlutterFlow · Databutton · Marblism · Create.xyz · Base44 · Lovable · Replit Agent | `CLAIM_ONLY` (produto fechado, sem demonstração executável) | `CAP-C-01/02/04/05` |
| GitHub Copilot Workspace · Devin · Cursor · Windsurf · Trae | `CLAIM_ONLY` | `CAP-A-01/07/09`, `CAP-B-*` |
| OpenHands · SWE-agent · Plandex · Roo Code · Cline · Aider · Continue.dev · Void · Zed · Dyad · Bolt.diy | **open source** — licença a confirmar caso a caso | `CAP-A-*`, `CAP-B-*`, `CAP-C-07` |
| Claude Code · Codex · ChatGPT · Gemini · Grok · Qoder · Blink · Verdent · Polsia · Manus · Hercules.app · Teamily · Genspark · Perplexity · TypingMind | `CLAIM_ONLY` / `BLOCKED_EXTERNAL` (login comercial) | `CAP-E-04`, `CAP-G-*`, `CAP-D-03` |
| MetaGPT · ChatDev · AutoGen · CrewAI · LangGraph · Dify · n8n · AgentScope · OpenSquad · AgentConnect · OpenClaw Office · Paperclip | open source (MIT/Apache, a confirmar) | `CAP-D-01/02/04/06`, `CAP-H-01/02` |
| Ollama · vLLM · LiteLLM · Lemonade · HarnessRouter · 9router · OmniRoute · E2B | open source | `CAP-E-*`, `CAP-F-01/02` |
| `Dpro-at/Tel-Agent` · `block/buzz` · `morluto/rea` · `paperclipai/paperclip` · `MakazhanAlpamys/Soup` · `vinzdg/codenotch` · `Damienchakma/Open-claude` · `noemica-io/open-claude-in-chrome` | **correções factuais aplicadas** (ver §16) | `CAP-H-05`, `CAP-I-06`, `CAP-J-01/03`, `CAP-D-03`, `CAP-K-11/12` |
| **DeepSeek** | **`CLAIM_ONLY` — e é o caso mais delicado da lista**: *o DeepSeek é o **executor desta própria missão***, não um concorrente a medir. *Um agente que se avalia como referência de si mesmo produz um número sem valor.* **Não há benchmark do HADES contra o DeepSeek** e **não deve haver um feito por ele próprio**: mediria o avaliador com o avaliador. ⇒ *A comparação legítima exige **avaliador independente**, e isso é `BLOCKED_EXTERNAL` até haver revisão humana independente* (`§REVISÃO HUMANA`). **Nota de conflito de interesse declarada.** | `CAP-E-04`, `CAP-G-03` (como referência de capacidade, não como baseline) |
| `wickedapp/openclaw-office` · `lemonade-sdk/lemonade` · `agentconnect-md/agentconnect` · `nexu-io/open-design` · `renatoasse/opensquad` · `davidondrej/cloudroom-gui` · `davidondrej/cloudroom-core` · `tldraw/tldraw` · `xyflow/xyflow` · `agentscope-ai/agentscope-java` · `langchain-ai/langgraph` · `OleLehmann/youtube-automation-agent` | **não clonados** — `UNVERIFIABLE` nesta janela | `CAP-D-06`, `CAP-E-*`, `CAP-C-02/03`, `CAP-F-06`, `CAP-I-04` |
| `addyosmani/agent-skills` · `heygen-com/hyperframes` · `googleworkspace/cli` · `apify/apify-mcp-server` · `microsoft/playwright-mcp` (+ core) | ✅ **CLONADOS E MEDIDOS** | §1 |

⇒ **Dívida declarada**: **12** repositórios da lista **não foram clonados** nesta janela. *Eles
ficam `UNVERIFIABLE`, não `ABSENT`* — e a diferença importa: **não medido ≠ inexistente**.

---

### 15.1 As 20 referências acrescentadas na ronda 191 (`E-1109`)

*Medido: a matriz cobria **19 de 27** repositórios e **37 de 49** referências de produto do prompt* =>
*faltavam **20**, e uma referência em falta é uma referência **silenciosamente ignorada** -- o que o prompt
proíbe.*

=> *as 20 estão agora no `.csv` como `REF-01..REF-20`, **todas** com o estado ***`UNVERIFIABLE`*** e o motivo
escrito:*

| grupo | referências | estado | porquê |
|---|---|---|---|
| 8 repositórios | `wickedapp/openclaw-office` · `lemonade-sdk/lemonade` · `agentconnect-md/agentconnect` · `nexu-io/open-design` · `renatoasse/opensquad` · `TypingMind/typingmind` · `davidondrej/cloudroom-gui` · `davidondrej/cloudroom-core` | `UNVERIFIABLE` | *não consultados com fonte primária nesta sessão* |
| 12 produtos | Marblism · Create.xyz · Base44 · OmniRoute · DeepSeek · Hercules.app · Verdent.ai · ChatDev · Continue.dev · Grok · Blink · Qoder | `UNVERIFIABLE` | *fechados (sem licença OSS) ou sem URL canónica confirmada* |

⇒ ***`UNVERIFIABLE` não é `ABSENT`, e não é `PARITY_MEASURED`.*** *A diferença importa: **não medido ≠
inexistente ≠ igual**.* => *e o prompt aceita-o explicitamente: «sem acesso legal/técnico ao concorrente, marque
`NOT_MEASURED`/`CLAIM_ONLY`; compare o HADES ao requisito observável, não invente resultado».*

⇒ **A dívida declarada**: *as 20 continuam **sem fonte primária consultada** => *a decisão é **ADIAR**, e a matriz
diz isso na coluna `lacuna_decisao` de cada uma.*

---

## 16. Correções factuais obrigatórias (aplicadas)

| referência | classificação **errada** | classificação **medida/corrigida** |
|---|---|---|
| `Dpro-at/Tel-Agent` | «controle do sistema por Telegram/WhatsApp» | **telefonia/agentes, SIP e ferramentas** → `CAP-H-05` |
| `block/buzz` | «transcritor offline» | **colaboração humano↔agente + eventos assinados** → `CAP-I-06`, `CAP-K-09` |
| `morluto/rea` | «deep research web» | **engenharia reversa de software (apps/binários)** → `CAP-J-01/02` |
| `paperclipai/paperclip` | «pesquisa/knowledge» | **organização e gestão de equipes/agentes** → `CAP-D-03` |
| `MakazhanAlpamys/Soup` | «áudio» | **sistema de fine-tuning de modelos** → `CAP-J-03/04` |
| `vinzdg/codenotch` | «IDE/editor» | **monitor de limites/uso de Codex e Claude Code** → `CAP-K-10` |
| `Damienchakma/Open-claude` | «sandbox do sistema» | **workspace multimodal com pesquisa e artefactos** → `CAP-K-11` |
| `noemica-io/open-claude-in-chrome` | — | **fora do escopo**: técnicas que removem barreiras de terceiros **não** autorizam reduzir as do HADES → `CAP-K-12` |
| «Buzz» | nome ambíguo | **identidade é `owner/repositório`**, nunca só o nome |

---

## 17. Limitações declaradas deste documento

1. **Nenhuma capacidade está `SUPERIOR_MEASURED`** — *não houve benchmark contra nenhuma
   referência*. Isso é **honesto**, não incompleto: `SUPERIOR_MEASURED` exige tarefa
   equivalente, dataset, hardware, repetições e artefactos reproduzíveis.
   ⇒ **E há um conflito de interesse declarado**: *o executor desta missão é um dos
   concorrentes listados* (**DeepSeek**). *Um agente que se compara consigo mesmo produz um
   número sem valor* — por isso a comparação exige **avaliador independente**, e está
   `BLOCKED_EXTERNAL` até isso existir.*
2. **5 fontes clonadas de 12+ nominais extraídas**; as restantes estão `UNVERIFIABLE` com
   motivo. *Não medido ≠ inexistente.*
3. **Licenças**: confirmadas nos 5 clones (MIT ×2, Apache-2.0 ×3). Para os restantes, a
   licença é **`a confirmar`** e **nenhum código será copiado** antes disso — *código sem
   licença não pode ser copiado*.
4. **`CLAIM_ONLY`** cobre produtos fechados onde não houve demonstração executável. *Marketing
   sem demonstração vira `CLAIM_ONLY`, não requisito comprovado.*
5. **Este documento é a fonte** do `.csv` derivado. *Corrigir a fonte primeiro; o derivado
   depois* (`§FASE 7` item 7).
6. **Nenhum pacote de extensão, APK, instalador novo ou publicação** foi produzido por este
   documento — *ele é um catálogo, não uma entrega*.

---

*Fonte: `HADES_COMPETITIVE_CAPABILITY_MATRIX.md` · derivado: `HADES_COMPETITIVE_CAPABILITY_MATRIX.csv`*
