# HADES x OPENJARVIS -- MATRIZ DE LACUNAS (adendo V3, FASE 1, segunda metade)

> **OpenJarvis auditado:** `open-jarvis/OpenJarvis` HEAD `091068cb94804f4dfbe57af2114adba89245612b`
> (2026-10-17T06:14:57Z), **Apache-2.0** (Copyright 2025 The OpenJarvis Authors), 11.451 stars, 88 MB.
> **Paper oficial declarado no README:** arXiv **2605.17172** (`https://arxiv.org/abs/2605.17172`),
> descoberto a partir do proprio repositorio como o adendo exige. **Nao foi lido** -- registado como
> referencia, nao como prova.
> **Fonte de cada linha:** `DOC` = README/documentacao; `CODE` = codigo lido; `RUN` = executado.
> **Nenhuma linha e' `CODE` nem `RUN`** -- o repo nao foi clonado e o adendo proibe executar scripts
> externos. O que se afirma e' o que o OpenJarvis DECLARA; o que o HADES TEM e' medido no checkout.

## Os tres falsos negativos que esta matriz apanhou

Medir por um nome inventado da' zero e mente. Aconteceu **tres vezes** nesta FASE 1:

| nome que procurei | deu | nome real | medido |
| --- | --- | --- | --- |
| `ReviewComment` | 0 | `Comment` | `collaboration.go`, 253 linhas + 2 testes |
| `RemoteRuntime` | 0 | `RemoteHost` + terminal + worktree | 20 + 10 + `CreateGitWorktree` |
| `engine-local` | 0 | `Ollama` / `/api/chat` | **771** e **25** ficheiros |
| `Preset` | 11 | presets de **PROVIDER**, nao de missao | 13 provider, **0 missao** |

*Tres dos quatro primeiros 'ausente' eram nomes meus. O quarto era um nome real a apontar para
outra coisa. Uma matriz construida sobre a primeira medicao teria pedido quatro reimplementacoes
de coisas que ja' existiam.*

## A matriz

| # | Capacidade no OpenJarvis | Fonte | Estado no HADES | Implementacao canonica | Lacuna funcional | Risco | Dependencias | Teste discriminante | Criterio de aceite | Decisao | Justificativa | Evidencia |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Local-first personal AI (inteligencia na maquina, nao na nuvem) | DOC | EXISTE_NAO_PROVADO | `Ollama` em **771** ficheiros `.go`; `/api/chat` em 25; `OLLAMA_HOST` em 45. Ollama e' o motor do produto. | Nao ha' prova E2E de que o caminho por omissao nunca toca a nuvem | Um fallback silencioso para API remota anula o 'local-first' | Ollama 0.40.1 em `127.0.0.1:11434` | Missao sensivel com `LOCAL_ONLY`: provar que nenhum endpoint remoto e' tocado (nao basta nao configurar) | E2E com captura de trafego provando zero chamadas externas | SUPERAR | O HADES ja' e' local-first por construcao (Ollama e' o motor). O OpenJarvis declara-o como diferencial; aqui e' a base. | E-1174 |
| 2 | **Presets de MISSAO** (`morning-digest`, `deep-research`, `code-assistant`, `scheduled-monitor`, `chat-simple`) | DOC | AUSENTE | `internal/multillm/presets.go` tem **13 presets de PROVIDER** (ollama, vllm, llamacpp, lmstudio, openai, anthropic, gemini, deepseek, groq, mistral, openrouter, xai, openai-compatible) e **0 de missao** | O HADES sabe escolher o MOTOR; nao tem configuracao versionada de TAREFA (agentes, ferramentas, permissoes, orcamento, criterio de conclusao) | Preset e' configuracao executavel: um preset errado corre com permissoes erradas | FASE 23 do adendo; `AgentOrchestrator` (7 ficheiros) | Carregar dois presets e provar que produzem topologias/ferramentas DIFERENTES, e que um preset sem permissao e' recusado | Preset inspeccionavel antes de executar, personalizavel, e ligado ao `/goal` | ADOTAR | O adendo (FASE 23) pede exactamente isto e o HADES nao tem. Nao confundir com os presets de provider, que ja' existem e nao sao a mesma coisa. | E-1174 |
| 3 | Oito agentes em tres modos (on-demand, scheduled, continuous) | DOC | PARCIAL | `internal/agent/schedule_flow.go` (agendado com aprovacao, dead-letter, laco com tecto); `SupervisorScheduler`; **47 ficheiros** com `Schedule`/`Cron`/`Ticker` em `internal/agent` | Existe agendamento. **Nao existe o modo CONTINUO** (`Monitor`=0, `Watchdog`=0, `Continuo`=0) | Um monitor continuo sem dono nem condicao de parada e' um loop eterno a gastar | `Heartbeat` (5), `Ticker` (4) ja' existem como base | Um agente continuo com intervalo, orcamento e condicao de parada: provar que para sozinho e que nao duplica apos reinicio | Pausa, cancelamento, orcamento e anti-duplicacao provados num monitor real | ADAPTAR | O agendado e' nosso e e' mais rico do que o README do OpenJarvis descreve (aprovacao por passo, dead-letter). O que falta e' so' o modo continuo. | E-1174 |
| 4 | Skill install de catalogos externos (Hermes ~150, OpenClaw) | DOC | PARCIAL | `skill_activation.go`, `HadesPluginSkill`; 9 ficheiros com `skills/`; **2** com install | Nao ha' ingestao de catalogo externo com quarentena, licenca, SBOM e secret scan | Importar 150 skills de terceiros sem quarentena e' supply chain em massa | FASE 10 e 20 do adendo | Importar uma skill externa e provar que fica em QUARENTENA e que nao corre ate' aprovacao | Skill importada, avaliada e so' entao promovida, com rollback | ADAPTAR | O HADES tem o registro; falta a ingestao governada. O adendo proibe importar sem quarentena -- o OpenJarvis instala directamente. | E-1174 |
| 5 | Skill benchmark (`jarvis bench skills --seeds`) | DOC | EXISTE_NAO_PROVADO | **34 ficheiros** `.go` com `Benchmark`/`benchmark` | Existe benchmark; nao ha' prova de comparacao `sem skill` vs `skill` vs `skill otimizada` com sementes | Benchmark sem baseline e' numero sem significado | FASE 20 do adendo | Tres condicoes (sem skill, original, otimizada) x N sementes: provar que a diferenca e' medida, nao afirmada | Relatorio com taxa de invocacao correta, qualidade, regressoes e rollback | ADAPTAR | O HADES ja' tem benchmark. Falta o desenho experimental que o adendo exige. | E-1174 |
| 6 | Skill optimize (`--policy dspy`) | DOC | PARCIAL | 10 ficheiros `.go` com `Optimiz`/`optimiz` | Nao ha' prova de que a optimizacao seja OVERLAY versionado com aprovacao e rollback (o adendo exige) | Otimizar em producao sem baseline e sem aprovacao e' autoimplantacao | FASE 20 do adendo | Optimizar uma skill e provar que a original NUNCA e' sobrescrita e que o rollback restaura exactamente | Overlay versionado, aprovacao humana e rollback imediato | ADAPTAR | Parcialmente presente. O adendo exige mais rigidez do que o OpenJarvis descreve. | E-1174 |
| 7 | Trace history para aprender | DOC | PARCIAL | 6 ficheiros `.go` com `TraceStore`/`StepTrace` | Nao ha' prova de que os traces sejam redigidos, exportaveis, apagaveis e ligados a decisao de roteamento | Um trace que guarda o prompt inteiro e' um registo de dados sensiveis | FASE 18 do adendo | Gerar um trace com um secret-canario e provar que o secret NAO aparece | Trace local, redigido, exportavel, apagavel, com ID de decisao | ADAPTAR | O HADES tem traces. O adendo exige a garantia de redaction que nenhum dos dois prova. | E-1174 |
| 8 | Memoria com recuperacao (retrieval) | DOC | PARCIAL | **1** ficheiro `.go` com `ContextMemory` (`context_memory_lock.go` tem o flock) | Uma camada de memoria, sem busca hibrida nem citacao de fonte/trecho/versao | Memoria sem fonte nao e' memoria: e' boato com data | FASE 17 do adendo; Obsidian (17.1) | Recuperar uma memoria e provar que a resposta cita nota, trecho e versao -- e que uma IA sem scope recebe NEGACAO | Isolamento negativo entre usuarios/tenants e direito de esquecer comprovado | ADAPTAR | O HADES tem o esqueleto e o flock. A FASE 17.1 (Obsidian) e' o trabalho de fundo. | E-1174 |
| 9 | TTS / saida por audio | DOC | EXISTE_NAO_PROVADO | 14 ficheiros `.go` com `TTS`/`Speech`; 33 com `voz` | Nao ha' prova de TTS local com voz/velocidade/idioma configuraveis nem de barge-in | Voz sem indicador visivel e sem forma de parar e' controlo sem testemunha | FASE 15 do adendo | Sintetizar, interromper a meio e provar que o audio parou e que o microfone ficou inactivo | Idioma/voz/velocidade configuraveis, interrupcao real, sem gravacao permanente | ADAPTAR | Presente e nao provado. O adendo acrescenta os requisitos de privacidade e de interrupcao. | E-1174 |
| 10 | Deploy em Docker | DOC | EXISTE_NAO_PROVADO | 17 ficheiros `.go` com `Docker`/`docker` | Nao ha' prova de imagem pinada, SBOM e limites no deploy do HADES | Imagem sem pin e sem SBOM e' um deploy que nao se sabe o que corre | FASE 25 do adendo | Construir a imagem, verificar o digest pinado e provar que um caminho bloqueado dentro do sandbox e' negado | Imagem pinada, SBOM, limites de CPU/RAM/PIDs e limpeza de orfaos | ADAPTAR | Presente. O adendo (FASE 25) exige o sandbox e a guarda de dados sensiveis que faltam. | E-1174 |
| 11 | **Leaderboard publico** (`open-jarvis.github.io/OpenJarvis/leaderboard`) | DOC | AUSENTE | Nada. `Leaderboard` = 0 ficheiros | O HADES nao publica nem mantem comparacao publica de modelos por classe de tarefa | Um leaderboard publico sem metodologia e' marketing; o adendo proibe usa-lo como benchmark | FASE 21 do adendo | Se for construido: benchmark reproduzivel com warm-up, N amostras e ambiente registado -- e NAO um ranking sem metodo | Benchmark versionado por classe de tarefa, hardware, modelo e data | NAO_ADOTAR | O adendo (FASE 21) exige benchmark reproduzivel LOCAL, nao um ranking publico. Construir um leaderboard seria a coisa errada: o que falta e' medicao honesta, nao posicao num quadro. | E-1176 |

## O que esta matriz NAO prova

1. Nenhuma linha do OpenJarvis foi lida no CODIGO -- so' no README. As capacidades sao o que ele
   DECLARA. Confirmar exigiria clonar 88 MB e ler o codigo, que nao foi feito.
2. O paper arXiv 2605.17172 esta' REGISTADO mas nao LIDO. Nao e' citado como prova de nada.
3. Nenhuma linha do HADES esta' `COMPLETO_VERIFICADO`.
4. `NAO_ADOTAR` no item 11 e' uma decisao de produto: o adendo pede benchmark local reproduzivel, nao
   um ranking publico. Nao e' um juizo sobre a qualidade do leaderboard do OpenJarvis.

## Resumo dos veredictos

| veredicto | quantas | quais |
| --- | --- | --- |
| SUPERAR | 1 | local-first (o HADES ja' e' Ollama por construcao) |
| ADOTAR | 1 | **presets de missao** (a unica lacuna liquida desta matriz) |
| ADAPTAR | 8 | agentes continuos, skills, benchmark, optimize, traces, memoria, TTS, Docker |
| NAO_ADOTAR | 1 | leaderboard publico |

*Uma matriz que diz ADOTAR a tudo nao e' uma matriz. Esta diz ADOTAR uma vez em onze -- e a unica que
diz ADOTAR e' a que o HADES nao tem mesmo: presets de missao, medidos como 0 contra 13 presets de
provider que sao outra coisa.*
