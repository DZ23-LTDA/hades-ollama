# Estágio 35 — quórum: valores altos exigem duas pessoas (E-153)

## 1. O limite que esta rodada fecha

O estado declarava: *"SEM QUÓRUM — uma decisão de ator autorizado basta"*. Agora, **acima de um limite de
valor**, uma decisão **não** basta.

| Peça | Comportamento |
| --- | --- |
| `QuorumAboveCents` | Limite a partir do qual o quórum vale. **Padrão R$ 1.000**; `0` desliga |
| `QuorumApprovals` | Quantas decisões de **pessoas distintas** são exigidas (padrão 2) |
| `RequiredApprovals` | **Gravado no pedido**: mudar a configuração depois **não** altera o que já foi exigido |
| `Decisions[]` | **Cada** decisão é registrada com autor e instante — sobrescrever apagaria o primeiro "sim" |
| Progresso | O motivo do turno diz *"2 aprovações … 1 de 2 decisões"*, e a API devolve `decisions` |

## 2. Quatro decisões de desenho

1. **Quórum LIGADO por padrão**: proteção que precisa ser lembrada de ligar não protege. Desligar é uma
   escolha explícita (`QuorumAboveCents = 0`), não um esquecimento.
2. **A MESMA pessoa não cumpre quórum.** A contagem é de **atores distintos** — é o ponto inteiro da
   exigência, e há teste específico para isso.
3. **Uma recusa encerra o pedido.** Quórum existe para **somar cuidados**, não para superar objeção: com um
   "não", o pedido vira negado e a tentativa seguinte é bloqueada com o nome de quem recusou.
4. **Abaixo do limite, uma decisão continua bastando.** Compatível com o que já existia — os testes
   anteriores seguem os mesmos, sem afrouxamento.

## 3. Transparência: `misspell` — idioma, não permissão

Quatro achados de lint nesta rodada eram **palavras portuguesas corretas** (`corporativo`, `autor`,
`argumentos`, `validade`, `eles`, `correspondente`, `confidencial`, `transfere`), que o dicionário **inglês**
do `misspell` sugere trocar por cognatos. O repositório **já** resolvia isso no `.golangci.yaml` com
`ignore-rules` para palavras portuguesas (`inferir`, `persistente`, `cliente`…) — segui o **precedente do
próprio projeto** em vez de distorcer o texto (em rodadas anteriores troquei *transfere* por *encaminha* só
para calar o linter, o que é pior).

E para não deixar a mudança como "linter afrouxado", a prova de que o `misspell` **segue ativo**: uma sonda
com erro real de inglês continua sendo pega.

```
$ golangci-lint run --no-config --enable-only misspell <sonda>
misspell_probe.go:3:41: `recieve` is a misspelling of `receive` (misspell)
* misspell: 1
```

`ignore-rules` declara **idioma**; não esconde erro de digitação em inglês.

## 4. Erro meu nesta rodada (registrado)

Escrevi uma asserção assim:

```go
if strings.Contains(call.Reason, "aprovacões") || strings.Contains(call.Reason, "aprovações") && ...
```

O primeiro termo tinha **erro de digitação** (`aprovacões`) e portanto **nunca** casava: a metade da
verificação era decorativa. **Uma asserção que não pode falhar é pior que nenhuma** — passei a checar só o
que importa (`"distintas"` no motivo), que é o texto que o código realmente emite quando há quórum.

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|TestRuntimeServiceAgents'` | **72/72 PASS** (5 novos de quórum) |
| `go test ./internal/agent/ -run TestServiceAgentQuorum -count=5` | **ok 6,372s** (sem flake) |
| `go test ./internal/agent/ -count=1` | **ok 64,536s** |
| `go test ./server/ -count=1` | **ok 21,012s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** (+ sonda provando que o `misspell` segue ativo) |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 22,871s · server 5,963s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes novos: **duas pessoas distintas** cumprem o quórum e a escrita acontece **uma vez**; com **uma**
decisão o pedido segue pendente e **nada é escrito**; a **mesma pessoa** decidindo duas vezes **não** cumpre;
**recusa encerra** o pedido; **abaixo do limite** uma decisão basta; a exigência fica **gravada** no pedido
(mesmo com a configuração desligada depois); e com **lista de aprovadores**, um forasteiro é recusado e o
quórum exige **duas pessoas autorizadas**.

## 6. Limites declarados

- **O limite é global do store**, não por ferramenta: hoje R$ 1.000 vale para toda escrita com valor; uma
  política por ferramenta/risco exigiria configuração por organização (não existe).
- **Quórum não substitui a lista de aprovadores**: sem lista, qualquer ator identificado decide (e
  `RoleChecked = false` permanece declarado); com lista, o papel é conferido em **cada** decisão.
- **Sem notificação** ao segundo aprovador: o pedido aparece na listagem, mas ninguém é avisado de que falta
  a segunda assinatura.
- **Aprovação parcial não expira por si**: o prazo (`ApprovalTTL`) só começa a contar quando o quórum é
  cumprido; um pedido com uma assinatura pode ficar pendente indefinidamente até alguém decidir.
- **Sem trilha separada por assinatura no store**: cada decisão está no pedido (`Decisions`), e cada uma
  entra na trilha de auditoria como `approval.decide`; não há assinatura criptográfica por aprovador.
