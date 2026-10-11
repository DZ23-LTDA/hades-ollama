# Estágio 25 — superfície HTTP do marketplace (E-128)

## 1. O limite que eu mesmo declarei

No documento anterior do estágio eu registrei, entre os limites: *"**sem rotas HTTP** e sem interface
visual"*. Esta rodada fecha a parte de rotas.

## 2. Rotas entregues

| Rota | Status |
| --- | --- |
| `GET /companies/:id/marketplace/channels` | **200** (catálogo com estado real) · **404** empresa |
| `POST /companies/:id/marketplace/sync` | **200** · **400** pedido inválido · **404** canal desconhecido · **429** teto excedido |
| `GET /companies/:id/marketplace/mirror/:channel_id/:sku` | **200** (espelho do **duble**) |

## 3. Decisões

- **429, não 400, no teto de operações**: exceder o teto por execução é **limitação de taxa** — o
  pedido está bem formado, o volume é que não cabe. Devolver 400 esconderia a natureza do problema e
  sugeriria erro do cliente.
- **Nenhuma publicação externa**: a sincronização roda contra o **duble** e o relatório traz
  `sandbox: true` e `homologacao_live: BLOCKED_BY_EXTERNAL_DEPENDENCY` com motivo. A rota do espelho
  também declara `sandbox: true`.
- **Credencial não é operação**: o catálogo deriva o estado apenas da **existência** das variáveis
  de credencial. Os **valores nunca são lidos** para exibição, resposta ou registro: só o nome é
  consultado, e o resultado é booleano.
- **Escopo por organização**: todas as rotas passam por `companyForRequest`, então a sincronização
  usa o `organization_id` da empresa da sessão — nunca um valor vindo do corpo da requisição.

## 4. Verificação (1 teste de rota cobrindo cada status)

`TestMarketplaceRoutesCatalogSyncAndMirror` **PASS**: catálogo **200** com canais declarando
`CONFIG_REQUIRED`/`UNAUTHORIZED`, **sem** teste live e **com** procedimento e capacidades;
sincronização **200** aplicando `1` e declarando sandbox e bloqueio; reexecução **ignora**
(`0` aplicados, `1` ignorado); espelho **200** refletindo `4` do duble e declarando sandbox; canal
desconhecido **404**; pedido vazio **400**; **acima do teto 429** e — comprovadamente — **nada foi
aplicado** apesar do pedido grande; empresa inexistente **404**.

## 5. Limites declarados

- **Homologação live**: `BLOCKED_BY_EXTERNAL_DEPENDENCY` — exige conta autorizada e scopes válidos;
  desbloqueio: definir a variável de credencial do canal e executar teste de conexão real com
  evidência de handshake.
- **Pull de pedidos** segue declarado e não exercitado (depende da API real).
- **Teto global**, não o limite real de cada plataforma.
- **Sem interface visual**: a superfície entregue é a HTTP; a tela fica para a frente de UX.

## 6. Verificação final

- `go test ./server/ -run TestMarketplaceRoutes` → **PASS**.
- `go test ./server/` → **ok** · `go test ./internal/agent/` → **ok**.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` → **0 issues**.
