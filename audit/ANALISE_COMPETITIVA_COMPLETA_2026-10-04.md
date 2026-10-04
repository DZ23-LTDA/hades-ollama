# Hades × 50+ concorrentes — análise competitiva completa e honesta (2026-10-04)

## Metodologia e honestidade (leia antes)
- **O que é verificável aqui:** as capacidades do **Hades** marcadas ✅ foram
  implementadas e testadas nesta linha do tempo (testes automatizados e/ou
  verificação no navegador, citadas no `MEMORIA.md` e em `docs/evidencias/`).
- **O que NÃO é benchmark head-to-head:** não tenho acesso para rodar Devin,
  Manus, Cursor, v0 etc. ao vivo. As colunas dos concorrentes refletem o
  **posicionamento público** de cada um (o que eles se propõem a fazer bem),
  não medições minhas. Portanto, **este documento não "prova" superioridade
  numérica** — ele mapeia, com honestidade, onde o Hades ganha, empata e perde.
- **Conclusão honesta de cabeça:** nenhuma ferramenta é a melhor em tudo. O
  diferencial real e defensável do Hades é a **combinação** (local-first +
  agente de código + app builder + automação por nós + conectores OAuth +
  segurança por tenant **num app só, rodando na máquina do usuário**). Contra um
  **especialista** no eixo dele (ex.: Cursor em edição de código, n8n em
  automação, v0 em UI), o Hades hoje **empata ou perde** no eixo isolado — e
  **ganha** quando o critério é "tudo junto, local e seguro". Dizer o contrário
  seria o "fake" que o projeto proíbe.

Legenda: **G** = Hades à frente · **=** = empate técnico · **P** = Hades atrás ·
(evidência entre parênteses quando G/=).

## 1. Agentes de código / IDEs agênticos
| Concorrente | Força principal deles | Hades vs ele | Nota honesta |
|---|---|---|---|
| Cursor | IDE+IA, edição inline multi-arquivo, tab-complete | **P** | Cursor lidera em DX de edição; Hades não é um IDE |
| Windsurf | IDE agêntico, fluxos "Cascade" | **P** | idem |
| Zed | editor nativo rápido + IA | **P** | Zed ganha em editor; Hades não compete em editor |
| Void / Trae / Roo Code / Cline / Continue.dev | extensões/forks de IDE com IA | **=/P** | Hades tem missões agênticas, mas não plugin de IDE |
| Aider | pair-programming no terminal, git-aware | **=** | Hades roda Aider/Claude como harness (cross-platform) e tem git worktree + merge-por-diff |
| Claude Code / Codex / Copilot-Workspace | agentes de código de ponta (nuvem/CLI) | **P** no raciocínio; **G** em local-first | Hades os **orquestra** como harness em vez de competir no modelo |
| Devin / OpenHands / SWE-agent / Plandex | agente autônomo issue→PR | **P** (fim-a-fim) / **G** (local+seguro) | Hades tem missões+worktree+approval-por-diff, mas E2E issue→PR provado ainda falta |
| Verdent.ai / Hercules.app / agente hermes | agentes de código recentes | **=/P** | pouca base pública p/ afirmar; não exagerar |

**Veredito da categoria:** especialistas de IDE/edição **ganham** no eixo deles.
O trunfo do Hades é **rodar os melhores agentes (Claude/Codex/Aider) por dentro,
local**, com segurança — não ser um IDE melhor que o Cursor.

## 2. App builders / geração de UI
| Concorrente | Força | Hades vs ele | Nota |
|---|---|---|---|
| v0 (Vercel) / Bolt.new / Bolt.diy / Lovable.dev | prompt→app React de alta qualidade | **P** na riqueza; **G** em offline/local | Studio do Hades: 12 componentes, drag-and-drop, preview+export HTML offline ✅; v0/Bolt geram apps mais ricos |
| Builder.io | visual CMS + design-to-code | **P** | Builder lidera em visual/CMS maduro |
| Replit Agent / Base44 / Databutton / Create.xyz / Marblism | build+deploy na nuvem | **P** (nuvem) / **G** (local) | Hades constrói e publica local sem conta obrigatória |
| FlutterFlow | apps mobile Flutter visuais | **P** | nicho mobile que o Hades não cobre |
| blink | build assistido | **=/P** | base pública limitada |

**Veredito:** v0/Bolt/Lovable **ganham** em qualidade de app gerado. Hades
**ganha** quando o critério é local-first + export offline + fazer parte de um
app maior (chat+agente+automação).

## 3. Automação / workflow
| Concorrente | Força | Hades vs ele | Nota |
|---|---|---|---|
| n8n | ~400 integrações, editor de nós maduro | **P** (amplitude) / **=** (editor visual) | Hades agora tem **editor visual de fluxo por nós** ✅ + **publicar cria automação real** (201 ✅), mas só gatilho→missão executa; conector/HTTP/condição ainda não disparam (listados honestamente) |
| dify | apps LLM + workflow + RAG | **P/=** | dify lidera em RAG/orquestração LLM visual |
| Make / E2B | automação / sandboxes de código | **P** | nichos específicos |

**Veredito:** n8n/dify **ganham** em amplitude. Hades fechou a lacuna do
**editor visual** e tem execução real do caso comum, verificada.

## 4. Roteamento de LLM / gateways
| Concorrente | Força | Hades vs ele | Nota |
|---|---|---|---|
| litellm | gateway multi-provedor padrão de mercado | **=** | Hades tem roteador multi-provedor (DZ23 Router) com metadados de harness ✅ |
| omniroute / HarnessRouter / 9router | roteadores de harness/modelo | **=/G** | Hades integra roteamento **+** app completo em volta |

**Veredito:** **empate técnico** no roteamento; Hades agrega o roteador a um
produto inteiro (vantagem de integração).

## 5. Deep research / assistentes de conhecimento
| Concorrente | Força | Hades vs ele | Nota |
|---|---|---|---|
| perplexity | busca com citações, deep research | **P** | Perplexity lidera; Hades tem pesquisa+citações básica |
| deepseek / chatgpt / gemini / grok bot / polsia | modelos/assistentes de fronteira | **P** (modelo) / **G** (local+orquestra) | Hades não treina modelos; ele **usa** os melhores, local quando possível |

**Veredito:** modelos de fronteira **ganham** em capacidade bruta. Hades os
consome (local + nuvem) dentro de um fluxo seguro — eixo diferente.

## 6. Multiagente / "company OS"
| Concorrente | Força | Hades vs ele | Nota |
|---|---|---|---|
| AutoGen / CrewAI / MetaGPT / ChatDev | frameworks multiagente | **=** | Hades tem agentes departamentais + supervisor + fan-out ✅, mas eles têm ecossistema/comunidade maior |

**Veredito:** **empate** em conceito; eles ganham em maturidade de framework,
Hades ganha em ser um produto pronto para leigo com segurança embutida.

## 7. Inferência local
| Concorrente | Força | Hades vs ele | Nota |
|---|---|---|---|
| Ollama (base) | runtime local padrão | **=** (é a base) | Hades É um fork do Ollama + camada agêntica/UI |
| vLLM | throughput de inferência servidor | **P** | vLLM ganha em servir em escala |

## 8. Browser/computer-use
| Concorrente | Força | Hades vs ele | Nota |
|---|---|---|---|
| OpenHands / Devin / Manus | automação de tela/navegador | **P** (maturidade) / **G** (setup plug-and-play local) | Hades agora tem **prontidão plug-and-play** do Browser Operator ✅ (detecta+instala Playwright/Chromium com guia), mas computer-use maduro nos 3 SOs ainda falta |

## 9. Repositórios GitHub citados (open-source)
litellm, dify, n8n, tldraw, xyflow — cobertos acima nas categorias. Os demais
(Tel-Agent, openclaw-office, buzz, lemonade, agentconnect, open-design, opensquad,
codenotch, Soup, dyad, agentscope-java) são projetos de nicho/menor tração: sem
base pública sólida para um veredito honesto — **não vou inventar comparação**.
Onde houver interesse, avalio um a um sob demanda.

## Onde o Hades genuinamente LIDERA (combinação, não eixo isolado)
1. **Local-first real** com segurança de nível enterprise (grants deny-by-default,
   approval vinculado ao diff, SSRF/DNS-pinning, isolamento por organização,
   OAuth por `(provider, subject)`) — raríssimo entre os que também fazem builder+automação.
2. **Tudo num app só** (chat + agente + builder + automação por nós + conectores
   OAuth reais + company OS) rodando no PC do usuário, sem nuvem obrigatória.
3. **Honestidade de produto**: promete = cumpre; recursos que ainda não executam
   são rotulados como tal, não fingidos.

## Onde o Hades AINDA PERDE (sem rodeio)
- Edição de código no nível de IDE (Cursor/Windsurf/Zed).
- Riqueza de app gerado (v0/Bolt/Lovable).
- Amplitude de integrações e gatilhos (n8n).
- Capacidade bruta de modelo e deep research (frontier labs / Perplexity).
- E2E issue→PR e computer-use maduros (Devin/OpenHands/Manus).

## Conclusão honesta
"Melhor e mais completo que **todos**" **não** é uma afirmação verdadeira hoje —
e eu não vou declará-la. O que é **verdadeiro e defensável**: o Hades é,
plausivelmente, **o mais completo entre os que são local-first + seguros**, e
está **fechando lacunas por capacidade** (editor de fluxo com execução real,
browser plug-and-play), cada uma verificada. O caminho para liderar de fato em
mais eixos está no roadmap priorizado do `COMPARATIVO_HADES_vs_CONCORRENTES_2026-10-04.md`.
