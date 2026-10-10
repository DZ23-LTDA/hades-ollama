# HADES — BLOQUEIOS E DESBLOQUEIO

> Bloqueio **não** interrompe trabalho independente. Cada item traz o
> procedimento exato de desbloqueio e o que continua possível sem ele.

## Resumo

| ID | Tipo | Bloqueia | Estágios afetados | Quem desbloqueia |
| --- | --- | --- | --- | --- |
| B-01 | AGUARDANDO_DECISAO_HUMANA | merge de PRs #56–#71 | vários | usuário |
| B-02 | EXTERNO | conta real Codex/Claude | 11 | usuário |
| B-03 | EXTERNO | tokens reais de canais IM | 16, 28, 33, 34, 35 | usuário |
| B-04 | EXTERNO | provedor de telefonia/voz | 29, 36, 37 | usuário |
| B-05 | EXTERNO | contas de marketplace/social/pagamento | 24, 25, 26, 27, 30, 31 | usuário |
| B-06 | EXTERNO | certificado de assinatura de release | 20, 38 | usuário |
| B-07 | EXTERNO | chaves de provedor de modelo | 4, 5, 8, 15, 23, 32 | usuário |
| B-09 | AMBIENTAL | Python/`sh` ausentes e limite de caminho | 2, 14 | ambiente de teste |
| B-10 | PRODUTO | 104/122 conectores exigem operador | 7, 16, 18 | decisão de produto + credenciais |

## B-01 — Autorização de merge (decisão humana)

- **Situação:** apenas **#54 e #55** estão mergeados. **#56–#71 estão abertos**;
  #69, #70 e #71 estão verdes e `mergeable`.
- **O que falta:** autorização explícita do usuário, por PR.
- **Como desbloquear:** responder quais PRs podem ser mergeados; a partir daí o
  merge é executado com squash/merge conforme a política do repositório.
- **Continua possível sem isso:** todo o restante (novos PRs empilhados,
  auditoria, testes, documentação).
- **Nota:** merge é operação de governança do repositório — **não** é executado
  por autonomia própria.

## B-02 — Conta real logada (Codex/Claude)

- **O que falta:** sessão autenticada real para provar paridade de harnesses
  além do adapter.
- **Como desbloquear:** fazer login nas CLIs no host e informar; a verificação
  passa a ser uma execução real registrada no índice de evidências.
- **Continua possível:** tudo que não exige conta (contratos, política de
  escopo, testes com fixtures).

## B-03 — Tokens reais de canais de IM

- **O que falta:** `bot token` do Telegram e credenciais do WhatsApp.
- **Como desbloquear:** fornecer as credenciais por canal seguro (nunca em Git,
  log ou arquivo versionado) e autorizar a homologação.
- **Continua possível:** adapter, roteamento, catálogo, testes de contrato com
  servidor falso rotulado.
- **Proibido:** declarar canal "funcionando" com mock.

## B-04 — Telefonia e voz

- **O que falta:** provedor real (SIP/telefonia) e engine de voz.
- **Como desbloquear:** decisão explícita de provedor + credenciais. Envolve
  custo financeiro ⇒ sempre exige aprovação.

## B-05 — Contas externas de negócio

- **O que falta:** contas de marketplace, redes sociais e meio de pagamento.
- **Como desbloquear:** autorização explícita por operação; publicação externa,
  mensagens a terceiros e pagamentos **nunca** são autônomos.
- **Proibido:** simular pagamento, criar perfil falso, engajamento artificial ou
  violar termos de uso.

## B-06 — Assinatura de release

- **O que falta:** identidade de assinatura (certificado) e smoke nativo por SO.
- **Como desbloquear:** disponibilizar o certificado e autorizar a publicação
  assinada.

## B-07 — Chaves de provedor de modelo

- **O que falta:** credencial de provedor para homologar rotação/fallback e
  demais recursos que dependem de inferência real.
- **Como desbloquear:** informar quais provedores podem ser usados e
  disponibilizar as chaves fora do repositório.
- **Continua possível:** toda a engenharia determinística (roteamento,
  orçamento, política, testes com dublês rotulados).

## B-09 — Limitações ambientais de teste (host Windows)

- **Situação:** 4 testes falham por ambiente, **provados pré-existentes** em
  `main` limpo: import de upload/worktree (exit 128), browser operator (Python
  ausente, exit 9009), kill de process group (`sh` ausente), e
  `TestImportMissionAttachmentsCreatesIndexedOrgProject` (`Filename too long`).
- **Como desbloquear:** instalar Python e um `sh` utilizável no host, ou reduzir
  o comprimento do caminho do workspace; alternativamente rodar esses testes no
  runner Linux (onde passam).
- **Proibido:** desativar, `t.Skip` ou mascarar os testes para "ficar verde".

## B-10 — Conectores que exigem operador

- **Situação:** dos 122 conectores do catálogo, **104** exigem configuração do
  operador e 1 exige escolha de provedor; apenas **17** estão `available`.
- **Implicação:** a classificação honesta é `APENAS_CATALOGO` — ter a linha no
  catálogo **não** é ter integração funcional.
- **Como desbloquear:** para cada conector que se queira promover, configurar a
  credencial e anexar evidência de chamada real (mantendo `Auth` não-vazio, sob
  pena de `TestConnectorCatalogIncludesOperationalIntegrations` falhar).
