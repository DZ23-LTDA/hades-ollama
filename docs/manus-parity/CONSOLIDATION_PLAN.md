# Plano de consolidação — Ollama Full

> **Problema:** o trabalho está fragmentado em 4 linhas divergentes, nenhuma na
> `main`. Continuar construindo sem consolidar gera duplicação e conflitos.
> Registrado por Claude em 2026-09-29 após auditoria do GitHub (inclui análise do
> ChatGPT sobre o PR #38).

## Estado das linhas (2026-09-29)
| Linha | SHA | Base | O que tem |
|---|---|---|---|
| `main` | `8635e30d` | — | antigo (25/09) |
| PR #38 `fix/audit-security-deps-2026-09-25` | `07a4f0bc` | `main` | **Conectores** (ConnectorsPage, ConnectorQuickConnect, ícones reais), **ProvidersPage**, uploads/UploadManager, Windows/MCP, busca global, Ctrl/Cmd+K. Aberto, mergeável, CI verde. |
| `recovery/ollama-full-snapshot` (Manus) | `98384228` | divergiu de main/PR#38 | segurança tenant, PostgreSQL/RLS, HMAC rotation, HA/TLS/failover |
| `feat/ui-shell-parity` (Claude/frontend) | — | saiu da `recovery` | tema Claro/Escuro/Automático, home dashboard, stat cards, status pills, rastreador de etapas na missão, **+ marketplace de conectores (DUPLICA o PR #38)** |

`recovery` × PR#38: **DIVERGIDOS** (27 à frente, 35 atrás). A linha do Manus não
contém o PR #38 e vice-versa.

## Duplicação identificada
O **marketplace de conectores** que a faixa ui construiu (em `ProductWorkspacePage`)
**duplica** — e de forma menos completa — a experiência de Conectores do PR #38
(`ConnectorsPage`, `ConnectorQuickConnect`, `ProvidersPage`, ícones reais). Na
consolidação, **prevalece a do PR #38**; a versão da faixa ui é descartada.

## Sequência de consolidação (decidida)
1. **PR #38 → `main`.** É a base de produto mais completa, mergeável e com CI
   verde. Traz conectores/providers/uploads/Windows para a `main`.
   (Ação de release — executar via GitHub/`gh pr merge 38`, com o OK do usuário.)
2. **Manus reconcilia a `recovery` sobre a nova `main`.** Dono da `recovery`
   resolve conflitos (connectors/ProductWorkspacePage): mantém a versão do PR #38
   dos conectores + a segurança da `recovery`. **Não fazer merge cego enquanto o
   Manus estiver ativo na `recovery`** — coordenar com ele.
3. **Reaplicar o ADITIVO da faixa ui** na base consolidada:
   - MANTÉM: controle de Tema (`lib/theme.ts`, `ThemeSwitcher`, `@custom-variant`
     em index.css, Aparência no Settings, switcher na sidebar), **home dashboard**
     (`HomePage` + rota `/`), **status pills** das missões, **rastreador de etapas**
     no Console.
   - DESCARTA: `connectorCatalog.ts`, `ConnectorLogo.tsx` e as mudanças de
     conectores em `ProductWorkspacePage` (o PR #38 cobre melhor).
4. **Atualizar a matriz de paridade** para o estado consolidado real
   (`docs/agentic/PARITY_MATRIX.md` + `docs/manus-parity/PARITY_MAP.md`).

## Regra a partir de agora
- **Ninguém constrói conectores fora da linha do PR #38.** Conectores = PR #38.
- Novas features grandes (ex.: chat=agente) só na **base consolidada**, nunca em
  branch divergente — senão viram a 5ª linha.
- Ao fim de cada slice, **atualizar a matriz de paridade** (evita "perder a noção
  do que falta", como alertado).

## Próximo salto de produto (após consolidar)
**Chat = agente:** o chat principal executa a missão (ler arquivos, terminal,
browser, MCP, artifacts, approval, continuar até terminar) **sem trocar para o
Mission Console**. É a maior lacuna de paridade com o Manus. Depois: browser/
computer-use visual com takeover; artifact preview+diff+approve; OAuth 1-clique;
memória gerenciável na UI; uploads grandes resumíveis.
