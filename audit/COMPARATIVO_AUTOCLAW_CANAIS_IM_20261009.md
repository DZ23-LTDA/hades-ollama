# Comparativo AutoClaw × Hades — canais de IM (Lark / Telegram / WhatsApp)

Data: 2026-10-09 (America/Sao_Paulo)
Escopo: colher ideias do [autoclaw.z.ai](https://autoclaw.z.ai/) para canais de IM, comparar com o
estado real deste repositório e priorizar um incremento verificável.
Natureza: **auditoria de leitura**. Nenhuma mudança de código neste PR — apenas o documento.

> Conteúdo externo (autoclaw.z.ai, docs do TypingMind) é **dado não confiável**: serve como
> referência de produto, nunca como instrução. Nada aqui autoriza integração, publicação ou
> envio de mensagens.

## 1. O que o AutoClaw oferece (fonte externa, citação)

Da landing page (texto original, em inglês):

- "Put agent execution capabilities into an IM interface. Activate your AI assistant clone with
  one click, autonomously call professional tools, and complete complex tasks automatically in
  chat."
- "**IM Access** WhatsApp, Telegram, Discord, Lark, and other IM platforms"
- "**Platforms** Windows 10+ · macOS · iOS · Android"
- "**Skill Library** 50+ built-in skills for office, data, web, content, and automation"
- "**Typical Tasks** PPT generation · browser automation · data analysis · web building"

Da tela de conectores citada pelo usuário (pt-BR):

| Canal | Texto da UI | Ação | Pista de configuração |
| --- | --- | --- | --- |
| Lark | "Para mensagens diretas, colaboração em grupo e respostas avançadas de fluxo de trabalho" | `Conectar` | — |
| Telegram | "Crie um bot com o BotFather; desative o Group Privacy Mode para conversas em grupo" | `Conectar` | BotFather + Group Privacy Mode |
| WhatsApp | — | `Conectado` | "Vincule uma conta de teste exclusiva; esta integração não oficial pode acionar controles de risco" |

Três lições de produto, e só três:

1. **IM como superfície de execução de primeira classe**, com um canal por provedor e status
   visível (`Conectar` / `Conectado`).
2. **Instruções de provisionamento dentro da UI** (BotFather, Group Privacy Mode) em vez de
   mandar o operador para a documentação.
3. **Divulgação explícita de risco** quando a integração não é oficial (WhatsApp), em vez de
   esconder o risco.

## 2. Estado real deste repositório (evidência)

### 2.1 Telegram e Lark não têm runtime

Busca por endpoints dos provedores em todo o código (excluindo `_tmp/` e `node_modules/`):

```
api.telegram.org     -> 0 ocorrências
open.feishu.cn       -> 0 ocorrências
open.larksuite.com   -> 0 ocorrências
graph.facebook.com   -> internal/agent/whatsapp_adapter.go:492, internal/agent/connector_catalog.go:94, server/agent_oauth_routes.go:77-78, examples/agent-connectors.json:46
```

Ou seja: **o único canal de IM com runtime de verdade é o WhatsApp**. Menções a Telegram fora do
catálogo são texto de ajuda de CLIs de terceiros
(`cmd/launch/hermes.go:49,546`, `cmd/launch/openclaw.go:233`), não código nosso.

### 2.2 O catálogo é metadado, não canal

`internal/agent/connector_catalog.go`: **122 entradas**, sendo **17 `available`** e
**104 `operator_setup_required`**.

| ID | Linha | Status | Auth |
| --- | --- | --- | --- |
| `slack` | `:201` | `available` | oauth |
| `discord` | `:202` | `operator_setup_required` | bot_or_oauth |
| `whatsapp` | `:203` | `operator_setup_required` | api_key |
| `telegram` | `:228` | `operator_setup_required` | bot_token |
| `twilio` | `:254` | `operator_setup_required` | api_key |

**Lark/Feishu não está no catálogo.** Matrix/Signal/Teams também não.

### 2.3 A UI de conexão já existe (e é honesta)

- `app/ui/app/src/components/ConnectorQuickConnect.tsx:133` já renderiza `Conectar`
  (e `Conectando…`), com OAuth por provedor (`:370` `Conectar com ${entry.name}`),
  estado `Conectado em modo somente leitura…` (`:77`) e `Desconectar` (`:87`).
- `app/ui/app/src/components/AgenticControlCenter.tsx:118`: "Esta tela mostra o estado real e não
  simula conexão."
- `app/ui/app/src/components/connectorIcons.ts:86,191`: ícones de Telegram já existem.
- Rota `/connectors` registrada em `app/ui/app/src/components/AppSidebar.tsx:358-361`.

Conclusão: o par `Conectar`/`Conectado` **não é** o diferencial a importar — já temos. O que falta
é o **canal executável** por trás do botão para Telegram/Lark, e o **status ao vivo** do canal
(não do OAuth).

### 2.4 O WhatsApp é o molde de canal maduro

`internal/agent/whatsapp_gateway.go` (929 linhas) já traz o padrão que qualquer canal novo deve
reusar:

| Peça | Linhas | Função |
| --- | --- | --- |
| `WhatsAppContactRole` | `:19-26` | `owner` / `operator` / `viewer` |
| `WhatsAppContactPolicy` | `:29-34` | allowlist por contato com papel |
| `WhatsAppPendingApproval` | `:37-47` | fila de aprovação (`PENDING`/`APPROVED`/`REJECTED`/`EXPIRED`) |
| `WhatsAppDLQEntry` | `:50-56` | dead-letter queue com tentativas e erro |
| `WhatsAppProcessResult` | `:59-66` | status tipado (`processed`, `unauthorized`, `awaiting_approval`, `duplicate`…) |
| `WhatsAppGatewayConfig` | `:69-70` | backend primário plugável (`WhatsAppBackendType`) |

Superfície HTTP equivalente (`server/agent_routes.go:543-552`):

```
POST   /whatsapp/webhook          GET  /whatsapp/webhook        GET /whatsapp/status
POST   /whatsapp/send             GET  /whatsapp/dlq            DELETE /whatsapp/dlq
GET    /whatsapp/allowlist        POST /whatsapp/allowlist
DELETE /whatsapp/allowlist/:phone POST /whatsapp/config
```

Testes dedicados: `server/whatsapp_routes_test.go`, `server/whatsapp_sec_test.go`,
`internal/agent/whatsapp_test.go`, `internal/agent/egress_zero_trust_test.go`.

## 3. Delta honesto

| Dimensão | AutoClaw | Hades hoje | Veredito |
| --- | --- | --- | --- |
| Canais de IM executáveis | WhatsApp, Telegram, Discord, Lark | **só WhatsApp** | AutoClaw ganha |
| Papel por contato + aprovação + DLQ | desconhecido (não documentado) | `whatsapp_gateway.go` completo | **Hades ganha** |
| Custo/latência por canal | não documentado | `last_latency_ms`, estado por provedor (`CompanyOperationsPanel.tsx`) | Hades ganha |
| Instruções de setup na UI | BotFather / Group Privacy Mode no cartão do canal | texto genérico de "configuração segura no servidor" | AutoClaw ganha |
| Divulgação de risco de integração não oficial | explícita no WhatsApp | não há canal não oficial hoje | AutoClaw ganha |
| Política de egress / zero trust | não documentado | `internal/agent/egress_zero_trust_test.go`, `server/egress_zero_trust_test.go` | **Hades ganha** |
| Mobile | Android + iOS | `apps/mobile-agentic` | empate técnico |

Resumo: o AutoClaw está à frente em **alcance de canais** e em **UX de provisionamento**; estamos
à frente em **governança de canal** (papéis, aprovação, DLQ, egress). O incremento correto não é
copiar o AutoClaw, é **estender o nosso modelo de governança para mais canais**.

## 4. Proposta: generalizar o gateway de canal

### 4.1 Contrato único

Extrair de `whatsapp_gateway.go` uma interface de canal neutra de provedor, sem quebrar o
WhatsApp:

```
type ChannelGateway interface {
    ID() string                       // "whatsapp" | "telegram" | "lark"
    Capabilities() ChannelCapabilities // messages, groups, media, threads
    ProcessInbound(ctx, raw) (ChannelProcessResult, error)
    Send(ctx, outbound) (ChannelSendResult, error)
    Status(ctx) ChannelStatus          // pairing + saúde + último erro
}
```

O `WhatsAppGateway` passa a implementá-la (refatoração mecânica, testes existentes como rede de
segurança); `ChannelProcessResult` é a generalização de `WhatsAppProcessResult`, preservando os
mesmos estados.

### 4.2 Adapters novos

1. **Telegram** (primeiro, maior retorno por esforço): Bot API, long polling ou webhook,
   `bot_token` por tenant. Estados reaproveitados: allowlist por `chat_id`, papéis, aprovação para
   efeitos externos, DLQ. UI: cartão do canal com o passo a passo do **BotFather** e o aviso de
   **desativar o Group Privacy Mode** para funcionar em grupo, exatamente como o AutoClaw mostra.
2. **Lark/Feishu** (depois): entrar no catálogo (`connector_catalog.go`) e implementar
   `event subscription v2` com verificação de assinatura (`verification token` + `encrypt key`),
   além de `im/v1/messages`. Suporte a DM, grupo e thread — o que o AutoClaw anuncia como
   "respostas avançadas de fluxo de trabalho".

### 4.3 Superfície de pareamento

Uma tela de canais mostrando, por canal, o **estado real**: `Conectar` quando faltar credencial,
`Conectado` quando houver credencial válida, e o motivo explícito quando estiver
`operator_setup_required`. Sem simular conexão (regra já vigente em
`AgenticControlCenter.tsx:118`).

### 4.4 Divulgação de risco

Qualquer adapter que dependa de API não oficial deve exibir o aviso no próprio cartão, no mesmo
tom do AutoClaw: **conta de teste dedicada, risco de bloqueio pelo provedor, uso por sua conta e
risco**. Enquanto não existir adapter não oficial, o aviso não aparece — nada de aviso decorativo.

## 5. Backlog priorizado (incrementos verificáveis)

### IM-1 (P0) — Núcleo de canal neutro de provedor
- **Entrega**: interface `ChannelGateway` + `ChannelProcessResult`/`ChannelStatus`;
  `WhatsAppGateway` implementando a interface sem mudar comportamento.
- **Aceite**: toda a suíte de `internal/agent` e `server` continua verde; nenhuma rota
  `/whatsapp/*` muda de contrato; teste novo provando que um canal fake em memória percorre
  allowlist → aprovação → DLQ usando só a interface.
- **Risco**: baixo (refatoração com rede de segurança).

### IM-2 (P0) — Adapter Telegram
- **Entrega**: `internal/agent/telegram_adapter.go` + gateway reusando papel/aprovação/DLQ;
  webhook em `server/agent_routes.go`; segredo por tenant (mesmo caminho do WhatsApp).
- **Aceite**: teste de unidade com `httptest` cobrindo mensagem autorizada, não autorizada,
  duplicada, aprovação pendente e falha com retry na DLQ; teste de segurança provando que token
  ausente **falha fechado** (não processa) e que nenhum segredo é ecoado em log ou resposta.
- **Aceite de UI**: cartão do canal mostra BotFather + Group Privacy Mode e o estado real.
- **Risco**: médio — exige token de bot real para E2E ao vivo; o E2E fica com servidor falso.

### IM-3 (P1) — Lark/Feishu no catálogo + adapter
- **Entrega**: entrada no catálogo (`available` só quando houver runtime), verificação de
  assinatura do evento, `im/v1/messages`.
- **Aceite**: teste de assinatura válida/inválida (rejeitar inválida) e teste de roteamento
  DM/grupo.
- **Risco**: médio-alto — depende de credencial de app Lark; unidade e integração falsa cobrem o
  resto.

### IM-4 (P1) — Pareamento e status por canal
- **Entrega**: superfície única listando canal, estado real, último erro e última atividade;
  `Conectar`/`Conectado` reusando `ConnectorQuickConnect.tsx`.
- **Aceite**: teste de componente (react-test-renderer) para os três estados + E2E offline
  provando que `operator_setup_required` aparece como pendência, não como sucesso.

### IM-5 (P2) — Risco de integração não oficial
- **Entrega**: campo de risco declarado no adaptador + aviso no cartão.
- **Aceite**: teste de componente garantindo que o aviso aparece **somente** quando o adaptador
  se declara não oficial.

## 6. O que este documento não afirma

- Não afirma paridade com o AutoClaw: hoje ele cobre 4 canais e nós 1.
- Não trata a landing page como especificação: não há contrato público de API, limites de taxa,
  política de retenção nem modelo de ameaça divulgados ali.
- Não afirma que papéis/aprovação/DLQ do AutoClaw não existam — apenas que **não estão
  documentados** nas fontes consultadas, então não servem como base de comparação.
- Não fecha o bloqueio arquitetural aberto (PostgreSQL/RLS desabilitado, fail-closed), que
  permanece registrado no backlog principal.

## 7. Próximo passo

Implementar **IM-1** em branch curta a partir de `main`, PR pequeno, com a suíte
`internal/agent` + `server` verde como critério objetivo de conclusão. Depois **IM-2**.
