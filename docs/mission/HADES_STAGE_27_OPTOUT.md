# Estágio 27 — captura automática de opt-out (limite declarado fechado)

## 1. O limite que existia

O checkpoint declarava: *"o consentimento não tem captura automática de opt-out"*. Conceder e
revogar existiam, mas **revogar dependia de um humano fazer à mão**. Se um contato escrevesse
"não quero mais receber", **nada acontecia automaticamente** e o sistema continuaria enviando.

## 2. O que foi implementado

`internal/agent/consent_optout.go`:

| Decisão | Por quê |
| --- | --- |
| **Direção segura (assimétrica de propósito)** | a detecção só serve para **REVOGAR**, nunca para conceder. Falso positivo bloqueia uma mensagem (barato e reversível pelo operador); falso negativo **continuaria enviando** para quem pediu para parar (caro e irreversível) |
| **Determinística e auditável** | nenhum modelo na decisão: a **frase detectada** vira a evidência registrada na revogação, então a decisão é explicável |
| **Guarda contra dupla negação** | "não quero **parar de receber**" contém "parar de receber" e significa o **oposto**; frases negadas são descartadas **antes** de qualquer revogação |
| **Escopo explícito na entrada** | `consentScope("")` mapeia vazio para a organização local (convenção do produto), o que **engoliria** um escopo ausente; aqui a organização crua é validada, porque quem revoga consentimento precisa dizer de qual organização fala |
| **Idempotente** | pedir para parar duas vezes é comum: a segunda captura informa `already_revoked` e **não** duplica histórico |
| **Não se responde a quem pediu para parar** | a confirmação seria exatamente a mensagem que o contato pediu para não receber; o efeito declarado é `envio_bloqueado_ate_novo_consentimento_explicito` |

## 3. Verificação (4 testes, nesta máquina)

| Teste | O que prova |
| --- | --- |
| `TestCaptureOptOutRevokesConsentAndBlocksOutbound` | com consentimento o envio funciona; após o pedido o consentimento fica **revogado**, o motivo **cita a frase detectada**, o envio passa a ser **recusado** (`ErrInboxConsentRequired`) e pedir de novo é **idempotente** com **um único** registro de revogação |
| `TestCaptureOptOutDoesNotFireOnOrdinaryMessages` | cinco mensagens comuns ("quero comprar duas unidades", "não quero **cancelar** meu pedido", "pode me enviar a proposta", …) **não** revogam nada |
| `TestCaptureOptOutIgnoresDoubleNegation` | quatro formas de dupla negação **não** revogam e o consentimento **continua concedido** |
| `TestCaptureOptOutRequiresScopeAndAcceptsVariants` | escopo ausente é **erro declarado** (não revogação silenciosa) e formas com acento, maiúsculas, inglês e pontuação são reconhecidas |

Um achado **do meu próprio teste** virou correção de produto: a primeira versão validava a
organização **depois** da normalização, então `""` passava como organização local e a revogação
acontecia em silêncio. Passou a validar a **entrada crua**.

## 4. Limitação honesta desta capacidade

A lista de frases cobre pt-BR e inglês comuns — **não** todas as formas de dizer "pare". Frase
não reconhecida continua dependendo de ação humana, e por isso a capacidade fica **PARCIAL**, e
não "completa". A captura também não decide sozinha o que fazer com quem pediu para parar em um
canal mas não em outro: a revogação é por **canal + contato**, como o resto do registro.

## 5. Observação de estabilidade (declarada, não escondida)

Durante a verificação, **uma** execução das suítes completas reportou falha: a do servidor com
`TestStudioDeployRejectsWithoutCredentialsHonestGateStatus` e a do agente sem nome visível na
saída filtrada. O teste do servidor **passa isolado** e, nas **5 execuções seguintes completas**
(servidor ×3, agente ×2), **as duas suítes passaram inteiras** — a falha **não reproduziu**.
Fica registrado como **intermitência observada sob carga**, **não** como corrigida: nenhum teste
foi alterado e nenhum `t.Skip` foi adicionado para contorná-la.

## 6. Verificação final

- `go test ./internal/agent/ -run TestCaptureOptOut -count=1` → **4/4 PASS**.
- `go test ./internal/agent/` → **ok** · `go test ./server/` → **ok** (5 execuções seguidas).
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` → **0 issues** (falso positivo do
  `misspell` em "Variantes" **reescrito**, não silenciado).
