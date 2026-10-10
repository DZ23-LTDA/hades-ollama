# TypingMind × Hades — veredito honesto por eixo (2026-10-09)

Você apontou `https://www.typingmind.com/` como o melhor harness que encontrou.
Este documento responde a isso com honestidade: **em qual eixo** ele é melhor, o
que ele é exatamente, e o que o Hades ganha ou perde frente a ele.

Método: as características do TypingMind abaixo vêm da **documentação do próprio
fornecedor** (dados de terceiros, não verificados por mim — não instalei nem
testei o produto). As características do Hades foram **verificadas no código**
deste repositório. Onde não verifiquei, está escrito "não verificado".

Fontes: [typingmind.com](https://www.typingmind.com/),
[Feature List](https://docs.typingmind.com/feature-list),
[Multi-Agent Workflows](https://docs.typingmind.com/ai-agents/build-multi-agent-workflow),
[documentação](https://docs.typingmind.com/).

## 1. O que o TypingMind é

Um **front-end de chat para modelos de IA** (aplicativo web estático) em que o
usuário traz as próprias chaves de API. Segundo a documentação dele:

- Modelos de OpenAI, Anthropic, Google, Mistral, Grok, DeepSeek, Kimi, além de
  OpenRouter, DeepInfra, Groq, Moonshot, Perplexity, xAI e Z.ai; modelos custom
  por endpoint compatível.
- "Multi-model responses" (vários modelos respondendo à mesma mensagem).
- Prompts com cache automático, sintaxe de mensagem, pastas de projeto, memória.
- "AI Agents" com instrução de sistema, modelo base, conhecimento próprio e
  skills; "Multi-Agent Workflows" executados por **menção `@agente` dentro de uma
  mesma conversa** (orquestração em nível de prompt, não um motor de execução).
- Plugins (busca/scrape, imagem, RAG, produtividade: Slack, Zapier, Todoist),
  suporte a servidores MCP, "Agent Skills".
- Dados de chat e prompts **no dispositivo**; chave de API criptografada
  localmente; sincronização em nuvem opcional; self-host estático com licença.

Ou seja: é um **harness de conversa** muito bem acabado. Não é um runtime que
executa comandos, edita arquivos, roda testes ou aprova efeitos externos.

## 2. Por eixo

| Eixo | TypingMind (doc do fornecedor) | Hades (verificado no repo) | Quem ganha |
| --- | --- | --- | --- |
| Amplitude de modelos | Lista ampla de provedores + endpoint compatível | `internal/multillm/` (catalog, registry, router, proxy, discovery, spend) + 122 conectores no catálogo, dos quais 17 `available` e 104 exigem configuração do operador | **TypingMind** (na prática do usuário final: plug-and-play) |
| Vários modelos lado a lado | Recurso nativo | Não encontrado em `app/ui` (busca por "lado a lado"/"side-by-side"/biblioteca de prompts = 0) | **TypingMind** |
| Biblioteca de prompts | Nativa, com cache automático | Não encontrada na UI | **TypingMind** |
| Memória / pastas de projeto | Nativo | Verificar caso a caso (não verificado nesta rodada) | Empate a confirmar |
| RAG / conversar com arquivos | Nativo na UI | Backend existe e está testado: `internal/agent/rag.go`, `embedding.go`, `rag_test.go` | **TypingMind** na UI; Hades já tem o motor |
| Agentes | Agentes com prompt/modelo/conhecimento/skills | Missões e "company OS" com execução real, eventos e persistência | **Hades** |
| Workflows multiagente | Orquestração por menção `@agente` na conversa | Execução com fila, quotas, replay, escopo por organização | **Hades** (execução real, não só prompt) |
| Execução de ferramentas no PC | Não é o propósito do produto | `terminal.exec` restrito a `pwd`/`ls`, `git.repo.inspect` endurecido, cwd fixado por descritor, timeout, saída limitada, DLP | **Hades** |
| Aprovação humana | Aprovação não é o modelo do produto | Aprovação vinculada ao diff, com motivo e redação de DLP | **Hades** |
| Segurança corporativa | Chave local criptografada; multiusuário na versão Teams | Egress zero-trust, política única de egresso, DLP recursivo, escopo por organização em MCP/conectores/deploys/jobs, 6 achados bloqueantes reauditados e fechados | **Hades** |
| Multitenancy / governança | Administração na versão Teams | Isolamento por organização testado (RPS); RLS PostgreSQL segue **desabilitado** por decisão de projeto | **Hades**, com ressalva do RLS |
| Local-first / offline | App estático, dados locais, mas inferência via API de terceiros | Executa modelos locais (Ollama/MLX) sem nuvem obrigatória | **Hades** |
| Self-host | Self-host **estático** com licença | Binário compilado, servidor local, instaladores por SO | Empate |
| Voz | Nativo | Não encontrado | **TypingMind** |
| Sync/backup | Sync em nuvem opcional | Não verificado | **TypingMind** |
| Observabilidade/CI | Não é verificável publicamente | Workflows com gates obrigatórios, SBOM, release-readiness | **Hades** |

## 3. Onde o TypingMind realmente é melhor

1. **Tempo até o primeiro valor**: colar a chave e conversar. No Hades, conectar
   provedor/agente hoje exige mais passos.
2. **Escolha de modelo**: troca entre modelos frontier sem fricção, incluindo
   comparação lado a lado — o Hades não tem esse recurso na UI.
3. **Biblioteca de prompts** com cache automático: ergonomia que o Hades não tem.
4. **Plugins com ecossistema pronto** (Slack, Zapier, Todoist) contra os 104
   conectores do Hades que ainda exigem configuração do operador.
5. **Acabamento de produto de chat**: memória, pastas, voz, sync — detalhes que
   o usuário final percebe imediatamente.

## 4. Onde o Hades é melhor

1. **Execução real com isolamento**: roda comandos, git, arquivos, navegador;
   TypingMind conversa.
2. **Aprovação humana vinculada ao diff** + DLP recursivo + egresso zero-trust.
3. **Multitenancy de verdade** (escopo por organização testado) e trilha de
   auditoria por missão/evento.
4. **Local-first sem nuvem obrigatória**, incluindo inferência local.
5. **Engenharia verificável**: gates de CI obrigatórios, SBOM, quarentena de
   RLS fail-closed em vez de segurança de fachada.

## 5. Veredito honesto

- "TypingMind é o melhor harness" é **verdadeiro no eixo de chat/UX e amplitude
  de modelos** — e falso como descrição de capacidade de agente que executa,
  aprova e isola. São produtos com propósitos diferentes que **se sobrepõem** em
  "conectar muitos modelos + MCP + agentes".
- Não vou declarar o Hades "melhor e mais completo que o TypingMind" em todos os
  eixos: hoje ele **perde** claramente em tempo até o primeiro valor, escolha
  comparada de modelos, biblioteca de prompts e acabamento de chat.
- O que é defensável e verificável: o Hades é mais completo **quando o eixo é
  executar com segurança e governança**; o TypingMind é mais completo **quando o
  eixo é conversar com o melhor modelo, agora**.

## 6. O que copiar do TypingMind (entra no backlog)

Cada item abaixo é de **UI/UX sobre backend que já existe** — custo menor que os
itens estruturais do backlog principal:

| ID | Item | Aceite verificável | Backend existente |
| --- | --- | --- | --- |
| P0-4 | Respostas multi-modelo lado a lado na mesma pergunta | Teste E2E que envia 1 mensagem a 2 modelos e mostra 2 respostas distintas | `internal/multillm/` (router, registry/proxy) |
| P0-5 | Biblioteca de prompts com variáveis, pastas e reuso | Teste E2E criando/salvando/reusando prompt; persistência local | a definir (persistência de conversas já existe) |
| P1-6 | Conversar com arquivos (RAG) pela UI | E2E: anexar arquivo → resposta cita o conteúdo; cobre `rag.go` pela UI | `internal/agent/rag.go`, `embedding.go` (+ testes) |
| P2-7 | Voz (STT/TTS) | E2E opt-in dependente de motor local; pular sem motor | inexistente |
| P2-8 | Sync/backup opt-in e criptografado, sem nuvem obrigatória | Teste de export/import com dados criptografados | export já existe no Studio? verificar |

## 7. Oportunidade de diferenciação

O TypingMind prova que **um chat front-end excelente vende sozinho**. O Hades já
tem o mais difícil (execução isolada, aprovação, DLP, multitenancy). A lacuna não
é capacidade de motor: é **tempo até o primeiro valor**. Um PR bem feito de
P0-4 + P0-5 muda a percepção de produto mais do que qualquer novo adaptador
isolado — e não depende de credencial externa para ser testado.

Referência cruzada: priorização geral em
`audit/BACKLOG_LACUNAS_PRIORIZADO_20261009.md`.
