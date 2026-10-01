# Matriz de referências e capacidades — Ollama Classe A+

## Finalidade e limite

Esta matriz transforma o complemento do Prompt Master V3 em decisões rastreáveis para o único produto em desenvolvimento, o **Ollama Classe A+**. Ela não cria um segundo produto, não declara integração instalada e não trata a existência de um adapter como homologação operacional.

A triagem abaixo é documental. As referências de produto e os repositórios não foram executados nesta rodada. A coluna **Evidência no Classe A+** registra o estado do código atual conforme a matriz de paridade e os gates já executados; não é uma afirmação sobre a implementação dos projetos de referência.

A arquitetura segue cinco camadas: produto; coordenação; inferência; execução especializada; e evidências. Ollama continua sendo o motor local padrão. Providers externos, harnesses, MCP, companions, builders e deploys permanecem opt-in, subordinados a uma missão, e limitados pela política de capabilities, tenant, aprovação e privacidade local.

## Estados usados

| Estado | Significado |
|---|---|
| `VALIDADA LOCALMENTE` | Implementação e testes reproduzíveis no repositório, sem afirmar serviço externo. |
| `ADAPTER IMPLEMENTADO` | Existe contrato e adapter, mas a jornada ponta a ponta depende de provider, conta, hardware ou infraestrutura externa. |
| `PARCIAL` | Existe uma parte funcional, mas ainda faltam superfícies ou gates essenciais. |
| `REFERÊNCIA / PENDENTE` | A fonte foi catalogada para estudo, mas não há integração ou prova suficiente no Classe A+. |

## Decisões por referência de produto

Todas as linhas abaixo são referências de capacidade, não dependências obrigatórias. Uma capacidade pode ser implementada nativamente uma única vez e atender várias referências.

| ID | Referência | Capacidade candidata | Abordagem no Classe A+ | Evidência no Classe A+ | Lacuna e decisão |
|---|---|---|---|---|---|
| R01 | Builder.io | Builder visual e design system | Nativa no Builder existente | `ADAPTER IMPLEMENTADO` | Completar editor rico, colaboração, exportação e deploy autorizado. Não copiar UI ou internals. |
| R02 | FlutterFlow | Construção visual de apps | Nativa, com contratos por stack | `ADAPTER IMPLEMENTADO` | Validar componentes, dados, autenticação, exportação e testes por alvo. |
| R03 | Copilot Workspace | Planejamento e edição sobre repositórios | Nativa no runtime de missões | `PARCIAL` | Tratar como referência histórica até validar produto e contrato atuais; não criar dependência. |
| R04 | OpenHands | Engenharia agentic em repositórios | Integração substituível futura | `REFERÊNCIA / PENDENTE` | Exigir sessão, cancelamento, artifacts, limites e isolamento antes de homologar. |
| R05 | Devin | Execução autônoma de engenharia | Nativa no coordenador, com executores opt-in | `PARCIAL` | Medir jornada completa em repositório real, recovery e intervenção humana. |
| R06 | Plandex | Planejamento e edição em contexto grande | Nativa no planner/runtime | `PARCIAL` | Benchmark de contexto, correção de testes e retomada. |
| R07 | Roo Code | Agente de coding com modos | Nativa por capabilities e profiles | `PARCIAL` | Consolidar modos em policy server-side, sem privilégios implícitos. |
| R08 | Cline | Coding com ferramentas e approval | Nativa no Runtime e approval ledger | `VALIDADA LOCALMENTE` para enforcement local | Homologar jornadas longas, storage distribuído e UI de revisão. |
| R09 | Aider | Edição orientada a diff e testes | Nativa no fluxo de workspace | `PARCIAL` | Completar revisão de diff, regressão antes/depois e recuperação. |
| R10 | Claude Code | Harness de engenharia e tools | Integração substituível, não fusão de produto | `ADAPTER IMPLEMENTADO` para provider/session base | API não equivale ao harness completo; validar sessão autorizada, tools, streaming e limites. |
| R11 | Void | IDE local-first | Referência de UX e contexto | `REFERÊNCIA / PENDENTE` | Extrair padrões de privacidade sem incorporar uma IDE inteira. |
| R12 | Trae | IDE agentic | Referência de UX e modos | `REFERÊNCIA / PENDENTE` | Comparar jornadas observáveis; não declarar paridade por menu. |
| R13 | Zed | Editor colaborativo e performance | Referência para editor/contexto | `REFERÊNCIA / PENDENTE` | Medir colaboração e acessibilidade antes de qualquer integração. |
| R14 | Windsurf | Contexto e fluxo agentic de coding | Nativa no contexto persistente | `PARCIAL` | Benchmark de recuperação e grounding em repositório real. |
| R15 | Cursor | Navegação, contexto e edição | Nativa na UI de workspace | `PARCIAL` | Melhorar busca contextual, diff, diagnóstico e estados de erro. |
| R16 | Databutton | Construção de aplicações orientada a dados | Builder nativo | `ADAPTER IMPLEMENTADO` | Validar dados, auth, preview, export e publicação com rollback. |
| R17 | Marblism | Geração de produto e operação empresarial | Company OS + Builder nativos | `VALIDADA LOCALMENTE` para sandbox | Conectores reais, OAuth, orçamento e staging continuam externos. |
| R18 | Create.xyz | Geração de apps por linguagem natural | Builder nativo com preview | `ADAPTER IMPLEMENTADO` | Garantir código editável, testes, segurança e exportação utilizável. |
| R19 | Replit Agent | Construção iterativa com preview/deploy | Builder nativo e artifacts | `ADAPTER IMPLEMENTADO` | Completar runtime de preview, health check e rollback autorizado. |
| R20 | v0 (Vercel) | UI generation e design-to-code | Builder nativo | `ADAPTER IMPLEMENTADO` | Validar fidelidade, acessibilidade e integração com backend real. |
| R21 | Base44 | Produto full-stack gerado | Builder + Company OS | `PARCIAL` | Testar auth, dados, jobs, export e deploy sem claim universal. |
| R22 | Bolt.diy | Builder local e editável | Nativa local-first | `PARCIAL` | Avaliar licenças por módulo e isolar execução de código. |
| R23 | Bolt.new | Construção rápida com preview | Builder nativo | `ADAPTER IMPLEMENTADO` | Provar jornada completa, não apenas geração de tela. |
| R24 | Lovable.dev | Geração de produto com preview | Builder nativo | `ADAPTER IMPLEMENTADO` | Completar testes, persistência, autenticação e publicação. |
| R25 | Perplexity | Pesquisa com fontes | Pesquisa multiagente nativa | `VALIDADA LOCALMENTE` | Executar smoke com fontes externas e budgets; proteger contra prompt injection. |
| R26 | Polsia | Operação de empresa autônoma | Company OS nativo | `VALIDADA LOCALMENTE` para sandbox | Não afirmar automação de produção; faltam contas, compliance, conectores e staging. |
| R27 | Manus | Jornada unificada agentic | Produto Classe A+ nativo | `PARCIAL` conforme matriz de paridade | Comparar jornadas observáveis, não copiar internals, marca ou UI proprietária. |
| R28 | OmniRoute | Gateway e roteamento | Integração substituível | `ADAPTER IMPLEMENTADO` | Validar instância real, health, modelos, fallback e respeito a `LOCAL_ONLY`. |
| R29 | DeepSeek | Modelo/provider local ou remoto | Provider configurável | `ADAPTER IMPLEMENTADO` | Validar credencial, modelo, limites e rota efetiva por missão. |
| R30 | Codex | Harness/provider de coding | Integração substituível | `ADAPTER IMPLEMENTADO` | API/provider não prova harness completo; exigir sessão, artifacts, cancelamento e smoke autorizado. |
| R31 | ChatGPT | Assistente multimodal e agentic | Contratos nativos por capability | `PARCIAL` | Implementar somente jornadas observáveis autorizadas; não alegar fusão de serviço proprietário. |
| R32 | Gemini | Provider multimodal | Provider configurável | `ADAPTER IMPLEMENTADO` | Validar modalidade, quotas, segurança de dados e falhas por provider. |
| R33 | Hermes Agent | Skills, memória e operação local | Nativa em skills/context store | `PARCIAL` | Não promover skill não confiável a privilégio; validar retenção, origem e exclusão. |
| R34 | Hercules.app | Construção e operação de produto | Builder + Company OS | `REFERÊNCIA / PENDENTE` | Avaliar contratos atuais antes de adotar qualquer dependência. |
| R35 | Verdent.ai | Engenharia agentic | Runtime nativo e executores substituíveis | `REFERÊNCIA / PENDENTE` | Medir jornada real e manter policy de capabilities server-side. |
| R36 | SWE-agent | Reparação de software e testes | Runtime de engenharia nativo | `PARCIAL` | Aceite exige bug real, teste que falha antes, regressão verde e PR verificável. |
| R37 | MetaGPT | Equipes e papéis especializados | Company OS e agentes departamentais | `VALIDADA LOCALMENTE` | Workers persistentes e avaliação longitudinal ainda pendentes. |
| R38 | ChatDev | Colaboração entre agentes | Orquestração nativa | `PARCIAL` | Usar somente com tenant, budget, capabilities e auditoria efetivos. |
| R39 | AutoGen | Orquestração multiagente | Runtime nativo | `PARCIAL` | Não adicionar segundo núcleo obrigatório; validar cancelamento, recovery e limites. |
| R40 | CrewAI | Flows e equipes | Runtime nativo | `PARCIAL` | Comparar benefício contra complexidade; evitar roteamento circular. |
| R41 | Continue.dev | Assistência em IDE e contexto | Referência para editor/contexto | `REFERÊNCIA / PENDENTE` | Extrair contratos úteis sem integrar uma IDE inteira. |
| R42 | Ollama | Inferência local | Base preservada e padrão | `VALIDADA LOCALMENTE` | Fixar compatibilidade de funcionalidades Classe A+ contra upgrades do motor. |
| R43 | vLLM | Backend complementar de inferência | Integração opt-in | `ADAPTER IMPLEMENTADO` | Validar hardware, compatibilidade, custo e fallback explícito. |
| R44 | E2B | Sandbox de execução | Referência/integrador substituível | `PARCIAL` | Sandbox forte, limites e recovery ainda precisam de prova; nunca fallback de `LOCAL_ONLY`. |

## Decisões por repositório explicitamente fornecido

Os 13 repositórios seguintes foram tratados como **triagem documental**. O fato de um README descrever uma capacidade não é prova de compatibilidade, licença de redistribuição, segurança ou operação no Classe A+.

| ID | Repositório | Capacidade estudada | Decisão | Gate obrigatório |
|---|---|---|---|---|
| G01 | [Tel-Agent](https://github.com/Dpro-at/Tel-Agent) | Voz/telefonia self-hosted e BYOK | Referência seletiva; não incorporar sem auditoria de licença e consentimento | Licença, canal, consentimento, credenciais, testes reais e limites de custo |
| G02 | [OpenClaw Office](https://github.com/wickedapp/openclaw-office) | Dashboard companion e eventos | Referência visual/observabilidade; não substituir o runtime | Eventos reais, identidade, autorização e nenhuma equipe simulada |
| G03 | [Buzz](https://github.com/block/buzz) | Colaboração, relay e eventos assinados | Referência para colaboração; não impor Nostr ou banco novo sem ADR | Modelo de identidade, assinatura, tenancy, CRDT/realtime e migração |
| G04 | [Lemonade](https://github.com/lemonade-sdk/lemonade) | Inferência local otimizada | Backend complementar opt-in | Hardware suportado, compatibilidade de API, benchmark e fallback explícito |
| G05 | [AgentConnect](https://github.com/agentconnect-md/agentconnect) | Canais de equipe e agentes ACP | Referência de sessões/canais; integração substituível | Contrato ACP, auth, tenant, cancelamento e auditoria |
| G06 | [OpenDesign](https://github.com/nexu-io/open-design) | Skills, design systems e artifacts | Referência para Builder/design; não copiar assets sem licença | Proveniência, path safety, preview, export e licença |
| G07 | [opensquad](https://github.com/renatoasse/opensquad) | Squads, pipelines e checkpoints | Referência para Company OS; prompts não substituem autorização | Policy server-side, isolamento, budget, recovery e auditoria |
| G08 | [Codenotch](https://github.com/vinzdg/codenotch) | Uso/estado de assistentes e telefone | Referência de observabilidade; não importar instruções de desativar proteções do SO | Privacidade, consentimento, device auth e testes físicos |
| G09 | [Soup](https://github.com/MakazhanAlpamys/Soup) | Fine-tuning/post-training | Extensão separada; não entra na instalação básica | Dados autorizados, licenças, GPU, orçamento, avaliação e rollback |
| G10 | [Dyad](https://github.com/dyad-sh/dyad) | Builder local | Avaliar módulos por licença; não assumir licença uniforme | Auditoria de subdiretórios, dependências, sandbox, export e atribuição |
| G11 | [LiteLLM](https://github.com/BerriAI/litellm) | SDK/gateway multi-provider | Comparar com OmniRoute antes de adicionar outra camada | Roteamento único, auth, quotas, observabilidade, fallback e sem ciclos |
| G12 | [Dify](https://github.com/langgenius/dify) | Workflows, RAG, agentes e observabilidade | Referência documental; licença e condições adicionais exigem revisão | Compatibilidade de licença, multitenancy, frontend, segurança e custo operacional |
| G13 | [HarnessRouter](https://github.com/HarnessRouter/harnessrouter) | UHP/Responses, sessões, streaming, arquivos e artifacts | Integração substituível opt-in | Instância real, harness instalado, sessão, follow-up, cancelamento, artifacts e recovery |

## Contrato canônico de harness

Qualquer executor externo deve declarar descoberta de capabilities, início e retomada de sessão, envio de turno, eventos incrementais, approval, cancelamento, artifacts, uso/quotas e encerramento. O adapter deve mapear IDs externos, status, falhas e arquivos para o contrato canônico do Classe A+.

Tenant e projeto vêm da identidade autenticada. Um agente filho recebe a interseção entre a policy do produto, a concessão do usuário e a capacidade real do executor. Ele nunca pode aumentar privilégios. Workspaces de subagentes devem ser isolados e a integração de mudanças deve ter dono, controle de concorrência, testes e revisão.

`LOCAL_ONLY` bloqueia egress de conteúdo para providers, harnesses, telemetria, embeddings e sandboxes de nuvem, salvo mudança explícita de política. Roteadores não devem ser encadeados indiscriminadamente; cada requisição precisa registrar quem decidiu modelo, executor, retry, timeout, custo e fallback.

## Aceite comparável

A promoção de qualquer linha para `VALIDADA LOCALMENTE` exige implementação, teste automatizado e execução reproduzível. Para builders, o aceite mínimo é gerar um app de estoque com autenticação e banco, editar um formulário visualmente, executar testes, criar preview, publicar em ambiente autorizado e restaurar a versão anterior. Para engenharia, o aceite é corrigir um bug real, provar o teste que falhava antes, passar regressões, entregar diff/PR e retomar após reinício. Para pesquisa, as alegações devem ter fontes, datas, exportação utilizável e nenhuma execução originada de instruções não confiáveis encontradas nas fontes.

Não usar somente screenshots, estrelas, claims comerciais ou um único LLM juiz para concluir superioridade. Registrar tempo, tokens, custo, memória, intervenção humana, cancelamento, recovery, segurança, acessibilidade e resultado funcional.

## Próxima sequência de implementação

A sequência permanece a do checkpoint do projeto: consolidar authorization por objeto e capability por skill/projeto/tenant; finalizar sandbox forte ou manter sua classificação best-effort; completar egress, DLP e sessão; homologar adapters externos com credenciais autorizadas; testar dispositivos físicos; e só então reavaliar a classificação de release. Nenhuma referência desta matriz autoriza merge em `main`, publicação em loja, compra, campanha, deploy de produção ou consumo pago.

## Referências

[1]: https://github.com/DZ23-LTDA/hades-ollama "Repositório público Ollama Classe A+"
[2]: https://github.com/DZ23-LTDA/hades-ollama/pull/1 "Pull request público do Ollama Classe A+"
[3]: ../agentic/PARITY_MATRIX.md "Matriz de paridade observável do Ollama Classe A+"
[4]: https://developers.openai.com/pt-BR/blog/codex-as-a-platform "OpenAI Codex as a platform"
[5]: https://code.claude.com/docs/en/agent-sdk/overview "Claude Agent SDK overview"
[6]: https://docs.openhands.dev/overview/introduction "OpenHands introduction"
[7]: https://docs.flutterflow.io/ "FlutterFlow documentation"
[8]: https://www.builder.io/c/docs/visual-editor "Builder.io visual editor"
[9]: https://docs.continue.dev/ "Continue documentation"
[10]: https://aider.chat/docs/repomap.html "Aider repository map"
[11]: https://docs.e2b.dev/ "E2B documentation"
[12]: https://hermes-agent.nousresearch.com/docs/user-guide/features/overview "Hermes Agent features"
[13]: https://githubnext.com/projects/ "GitHub Next projects"
[14]: https://github.com/langgenius/dify/blob/main/LICENSE "Dify license"
[15]: https://github.com/dyad-sh/dyad/blob/main/README.md "Dyad README"
