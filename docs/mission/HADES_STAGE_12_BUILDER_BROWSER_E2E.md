# Estágio 12 — Builder: publicação servida e NAVEGADA por navegador real (E-178)

## 1. O item que estava NAO_COMPROVADO

A matriz de aceite dizia: *"O Builder entrega aplicações utilizáveis — **NAO_COMPROVADO**: app de estoque gerado e
testado em sandbox; falta navegação real ponta a ponta"*. Este documento fecha essa lacuna — e o passo de
navegador **encontrou um defeito** que nenhum teste hermético tinha visto.

## 2. O que foi executado (passo a passo reproduzível)

| Passo | Comando / ação | Resultado |
| --- | --- | --- |
| 1. criar o projeto pelo Builder **real** | `Create(BuilderSpec{Name:"Estoque E2E", Kind:BuilderApp, OrganizationID:"org-e2e-builder"})` | projeto criado |
| 2. aplicar componentes visuais do Studio | `ApplyVisualComponents` com `heading` + `table` + `button` | 3 componentes |
| 3. **PUBLICAR** | `PublishLocal(ctx, id)` | `published/<id>/v2` com `index.html` |
| 4. **SERVIR por HTTP** | `python -m http.server 8791 --directory <publicado>` | `GET /index.html` → **200**, 1237 bytes |
| 5. **NAVEGAR com navegador REAL** | navegador Playwright em `http://127.0.0.1:8791/index.html` | título **`Estoque E2E`**, **0 erros de console** |
| 6. provar que o **JS renderizou** | `document.querySelectorAll('[data-dz23-component]')` | **3 componentes**: `heading`/`table`/`button` com os textos `Estoque E2E`, `Produtos`, `Novo produto` |
| 7. evidência visual | captura de tela | `docs/mission/evidence/builder-published-app-navegador-real.png` |

O teste que cobre os passos 1–4 de forma **hermética** vive no repositório:
`internal/agent/builder_published_e2e_test.go` (publica, serve por HTTP de verdade e confere o HTML entregue).
Com `HADES_BUILDER_E2E_ROOT` definido, ele publica num diretório **estável** — é o que permite o navegador real
abrir **a mesma** aplicação que o teste verificou.

## 3. O defeito que o NAVEGADOR encontrou (e os testes não)

Na primeira navegação o console tinha **1 erro**: `GET /favicon.ico → 404`. Nenhum teste hermético pegaria isso,
porque nenhum deles **pede o favicon** — quem pede é o navegador. A consequência não é cosmética: **toda
aplicação publicada** pelo Builder poluía o console de quem a abre.

Correção no template (`internal/agent/builder.go`): `<link rel="icon" href="data:,">` — sem ativo binário, sem
requisição de rede. Depois da correção: **0 erros de console**.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestBuilder -count=1` (com `HADES_BUILDER_E2E_ROOT`) | **ok** |
| `GET http://127.0.0.1:8791/index.html` | **200** |
| navegador real em `/index.html` | título correto, **0 erros de console** |
| `document.querySelectorAll('[data-dz23-component]')` | **3** componentes com os textos esperados |
| `golangci-lint` / contrato / paridade Linux | ver evidência |

## 5. Limites declarados

- O que foi navegado é um app de **demonstração do Studio** (`heading`/`table`/`button`) — **não** é uma
  aplicação full-stack com banco, autenticação e CRUD, que continua sendo trabalho **não** comprovado.
- A navegação foi feita por **servidor estático local**; não houve deploy em staging nem rollback de publicação.
- A captura é de **viewport** em um único tamanho de janela: **não** é validação responsiva nem de acessibilidade
  (leitor de tela, teclado, contraste).
- O passo de navegador é **manual/interativo** (navegador do harness): o repositório guarda o teste hermético e o
  roteiro; a navegação **não** está automatizada no CI.
- `data:,"` como ícone é um marcador **vazio** de propósito: silencia a requisição, mas o app publicado continua
  **sem identidade visual** própria.
