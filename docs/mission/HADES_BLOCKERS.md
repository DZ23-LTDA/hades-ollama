# HADES — BLOQUEIOS E DESBLOQUEIO

> Bloqueio **não** interrompe trabalho independente. Cada item traz o
> procedimento exato de desbloqueio e o que continua possível sem ele.

## Resumo

| ID | Tipo | Bloqueia | Estágios afetados | Quem desbloqueia |
| --- | --- | --- | --- | --- |
| B-01 | RESOLVIDO (E-295) | checagem ONLINE: **zero PRs abertos** — os PRs #56–#71 já não estão abertos | vários | verificado |
| B-02 | EXTERNO | conta real Codex/Claude | 11 | usuário |
| B-03 | EXTERNO | tokens reais de canais IM | 16, 28, 33, 34, 35 | usuário |
| B-04 | EXTERNO | provedor de telefonia/voz | 29, 36, 37 | usuário |
| B-05 | EXTERNO | contas de marketplace/social/pagamento | 24, 25, 26, 27, 30, 31 | usuário |
| B-06 | EXTERNO | certificado de assinatura de release | 20, 38 | usuário |
| B-07 | EXTERNO | chaves de provedor de modelo | 4, 5, 8, 15, 23, 32 | usuário |
| B-09 | RESOLVIDO (E-116) | Falhas de teste: 1 **defeito real** de idempotência na importação + MAX_PATH do Windows | 2, 11, 14 | resta apenas Python ausente para o teste de browser |
| B-10 | PRODUTO | 104/122 conectores exigem operador | 7, 16, 18 | decisão de produto + credenciais |
| B-11 | RESOLVIDO | 7 PRs do dependabot com CI vermelho, tratados um a um **sem merge cego** (#26 e #34 entre eles, com a causa raiz provada no log) | vários | RESOLVIDO no próprio tratamento |
| B-12 | RESOLVIDO | PR #31 (grupo web-dev-tooling) quebrava `Web and mobile quality`, `test (ubuntu-latest)` e `Web E2E` — causa raiz provada no log do CI e corrigida | vários | RESOLVIDO no próprio tratamento |
| B-13 | RESOLVIDO | execucao de plugin quebra no layout real: `Workspace` aponta para o **arquivo** `.zip` (`plugin_runtime.go:215`) e o executor espera diretorio => `not a directory` (E-183) | 8 | corrigir o ponto de execucao (extrair o bundle ou apontar o diretorio extraido) | **RESOLVIDO (E-184)**: bundle extraido para diretorio de execucao temporario com protecao fail-closed e Workspace apontando para ele |
| B-14 | AMBIENTE (DIAGNOSTICO RESOLVIDO, PRE-REQUISITO ABERTO) | a execucao de plugin exige PRIVILEGIO DE MONTAGEM: mesmo `best-effort` faz `mount`/`chroot` (`tools.go`), e fora disso o sintoma e `exit status 1` com `stdout`/`stderr` VAZIOS (falha silenciosa) (E-184) | 8 | rodar em host com namespaces de montagem e propagar o stderr do launcher | **DIAGNOSTICO CORRIGIDO (E-185)**: a falha silenciosa era o runtime descartando o `Output`; agora propaga `stderr` e `exit_code` |
| B-16 | RESOLVIDO | mesma classe do B-15 em QUATRO módulos: `GetForOrganization` de `builder.go:352`, `devices.go:210`, `swarm.go:218` e `TelAgentSessionForOrganization` (`tel_agent_session.go:181`) devolvem erro **cru** quando não existe (→404) e erro **próprio** quando é de outro tenant (→403), revelando existência (E-191) | 18 | por módulo: unificar as duas recusas e converter as asserções que codificam o 404 em asserção de não inferência | **RESOLVIDO (E-192)**: recusas unificadas nos quatro módulos, com 5 testes novos (4 de não inferência + controle positivo); agent e server passaram sem conversão de teste |
| B-17 | RESOLVIDO | `internal/agent/mcp_remote.go:606` (`call`) devolve `remote MCP server … is not registered` **antes** do portão de organização — mesma forma do B-15/B-16, **sem** teste que compare as duas recusas (E-192) | 18 | montar harness de MCP remoto, comparar as negações e unificar se confirmado | **RESOLVIDO (E-194)**: a verificação achou **seis** métodos `*ForOrganization` com o mesmo oráculo; caminho de tenant unificado, caminho global declarado; 11 testes novos |
| B-18 | RESOLVIDO | `TestServiceAgentAuditKeepsContactTextOut` falhou **uma vez** na `main` (passo `CGO_ENABLED=1 go test ./...`); **não reproduzida em 6 condições**, e a varredura de DLP (30 ferramentas × 7 agentes) não achou vazamento (E-193) | 18 | re-executar o job com credencial de API e capturar o **campo** nomeado pelo diagnóstico | **RESOLVIDO (E-196)**: era o **relógio** — o fragmento `4111` casava com `decided_at` (`.411114Z`), não com conteúdo de terceiro. Checagem passou a olhar **campos**; mecanismo fixado em teste |
| B-19 | EXTERNO (3 fronteiras) | **Chrome real** (loja) / **Firefox operar** (Developer Edition) / **computer-use** (E2E nativo por SO) | 14 | usuário / sistema |
| B-15 | RESOLVIDO | oráculo de enumeração em TODA a superfície de empresa: `companyForRequest` devolve **404** para empresa inexistente e **403** para empresa de outro tenant, revelando existência de ID sem acesso a dado; o vazamento está **codificado em testes** (E-189) | 18 | unificar as duas recusas no guarda, converter cada asserção de 404 em asserção de não inferência e varrer outras rotas | **RESOLVIDO (E-190)**: recusas unificadas no guarda compartilhado; segundo mecanismo (eco do ID só no caminho que existe) encontrado pelo teste genérico e corrigido; prova com controle positivo |

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

- **Situação ATUAL (E-116): as suítes locais passam INTEIRAS.**
  `go test ./internal/agent/` e `go test ./server/` terminam **ok**, sem nenhuma falha.
- **O que o rótulo "ambiental" escondia:** ao investigar em vez de repetir a explicação, o
  teste de import de upload/worktree era **DEFEITO REAL** — `initializeImportedRepository`
  exigia `exit 0` do `git commit` e falhava com `nothing to commit` quando o snapshot já
  estava commitado, quebrando **reimportação e retomada**. Corrigido com idempotência
  (só commita quando há mudanças e valida `HEAD`).
- **MAX_PATH do Windows:** o caminho interno do git
  (`<raiz>/.git/objects/xx/<38 caracteres>`) passa de 260 quando a raiz é funda. O produto
  agora devolve `ErrImportPathTooLong` com mensagem **acionável**, e os testes usam raiz
  **curta sem o nome do teste** (`short_root_test.go`).
- **Pendência real restante:** o teste de browser operator exige **Python instalado** no
  host; é dependência declarada, não defeito.
- **Proibido (mantido):** desativar, `t.Skip` ou mascarar testes para "ficar verde". Nesta
  correção **nenhum** teste foi desativado e nenhum `//nolint` foi usado.

## B-10 — Conectores que exigem operador

- **Situação:** dos 122 conectores do catálogo, **104** exigem configuração do
  operador e 1 exige escolha de provedor; apenas **17** estão `available`.
- **Implicação:** a classificação honesta é `APENAS_CATALOGO` — ter a linha no
  catálogo **não** é ter integração funcional.
- **Como desbloquear:** para cada conector que se queira promover, configurar a
  credencial e anexar evidência de chamada real (mantendo `Auth` não-vazio, sob
  pena de `TestConnectorCatalogIncludesOperationalIntegrations` falhar).

## E-304 — DEFEITO ABERTO: a reserva de estoque não sobrevive ao reinício (severidade **ALTA**)

**Medido em `main` (303 evidências, `22307bf41`)**, com reprodução exata:

```go
primeiro, dataRoot := businessRuntime(t)                       // cria o runtime
relatorio1, _ := primeiro.CertifyBusinessJourney(entrada, now)  // certifica (aprovado)
segundo, _ := NewRuntime(RuntimeConfig{                        // REINÍCIO REAL: memória nova, MESMA raiz
    Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: raizWorkspace, DataRoot: dataRoot,
})
relatorio2, err := segundo.CertifyBusinessJourney(entrada, now)  // <- FALHA
```

Saída medida:

```
certificação após o reinício: reserva inesperada:
{SKU:CAM-001 OnHand:18 Reserved:0 Available:18 MovementCount:7}
```

`internal/agent/business_e2e.go:229` exige `Available == InitialStock - Quantity`. Depois do reinício o **pedido
continua existindo** (idempotência persistida) mas `Reserved` volta a **0** e `Available == OnHand`: **o estoque
reservado pelo pedido não está mais reservado**. Vender o que não existe é o risco de severidade **ALTA**.

**Causa provável**: `CommerceStore.Stock` (`internal/agent/commerce.go`) lê um mapa **em memória** (`s.stock`) e o
reservado não é reconstruído a partir dos pedidos abertos persistidos. É a mesma classe de E-299 (estado em memória
que o reinício esquece), agora com efeito de **estoque**.

**Por que não foi corrigido nesta rodada — declarado, não escondido**: corrigir exige mudar o modelo de estoque
(persistir/reconstruir o reservado por pedido aberto), com efeito em concorrência, comissão e migrations. O orçamento
da rodada não comporta essa mudança **com a verificação que ela exige**, e entregar sem verificar seria trocar um
defeito por uma suposição.

**Desbloqueio**: (1) reconstruir o reservado de pedidos abertos ao carregar o store; (2) teste do mesmo reinício
real, verde; (3) verificar que a jornada continua idempotente **e** que o reservado resiste. O `RecoveryProven`
atual mede **idempotência no mesmo processo** — não reinício; a evidência E-304 corrige essa leitura.

### ⚠️ Correção de E-304 (E-305) — a persistência **não** estava quebrada

Medição refeita com **diagnóstico de movimentos**: antes e depois do reinício o saldo é **idêntico** e os quatro movimentos estão presentes; o teste que já existia (`TestBusinessE2ESurvivesRealRestartWithoutDuplicating`) **passa**. ***O defeito descrito aqui acima era leitura errada.*** O defeito **real** era outro — a jornada **somava a carga inicial de novo** na reexecução (estoque 10 → 18) e falhava com erro confuso. **Corrigido em E-305**: carga inicial idempotente + recusa explícita (`ErrBusinessJourneyAlreadyCertified`) para empresa que já operou.

## B-19 — As 3 fronteiras do §ESTÁGIO 14 (todas NOMEADAS, `E-1129`)

O §ESTÁGIO 14 estava `BLOCKED_BY_EXTERNAL_DEPENDENCY` com `blocker: B-09` — e o **`B-09` está declarado como
«AMBIENTAL (MEDIDO E CORRIGIDO)»**: *um estágio bloqueado por um blocker que o próprio estado declara corrigido não
está bloqueado — está mal classificado.* As **três** fronteiras reais, medidas nos `E-1121`/`E-1124`:

1. **Chrome real** — o `--load-extension` foi **removido** e o `service_worker` não acorda (155.0.8059.79). Exige a
   **Chrome Web Store**: conta + revisão. *O que continua possível sem ela*: **Chromium 156** e **Edge 155** passam
   **10 + 25** verificações (`E-1121`).
2. **Operar no Firefox** — o `web-ext` instala como **temporary add-on** e ele **não persiste**; o Playwright
   Firefox lança mas **não vê** a extensão. Exige um **Developer Edition** com
   `xpinstall.signatures.required=false`. *O que continua possível sem ele*: o pacote, o `lint` **0/0/0** e o
   **`Installed`** (`E-1116`/`E-1122`), e o **operar** no Chromium e no Edge.
3. **Computer Use** — não há **nenhum** E2E nativo por SO. Exige E2E real em Windows/macOS/Linux, com permissões
   nativas, kill switch e testes de abuso. *O que continua possível sem ele*: os adapters compilados e o
   `desktop.companion` declarado.

**Nenhuma das três é código por escrever**: são conta/loja, um binário específico e um host nativo. E o §ESTÁGIO 14
continua **`PARTIAL` (10 de 11)** e **`BLOCKED_BY_EXTERNAL_DEPENDENCY`** — agora pelo motivo certo.
