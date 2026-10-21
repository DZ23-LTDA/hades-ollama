# BACKLOG_PRODUTO — itens autorizados, com ACEITE ESCRITO ANTES

Regra desta lista: **um item por commit**, com verde da PR lido **por id de job** antes do
próximo. Cada item só é DONE quando os critérios de aceite abaixo estiverem provados com
**teste positivo + negativo** e, havendo UI, **E2E real** (captura de ecrã não é prova).

Estimativa não é medição. Tudo o que tocar build/instalador/antivírus deve ser multiplicado
por 2–3.

---

## I0a — Anexos/zip: o erro deixa de ser silencioso

**Aceite (escrito antes da implementação):**

| # | critério | prova exigida |
| --- | --- | --- |
| A1 | Um zip cujo conteúdo excede os limites (`ErrZipLimit`) produz **mensagem visível ao utilizador**, com o limite que foi excedido — não apenas `slog.Warn`. | teste negativo em Go + saída |
| A2 | Um zip **sem** ficheiros de texto suportados produz **erro explícito** com a contagem de entradas ignoradas. | teste negativo em Go |
| A3 | Um zip válido com ficheiros suportados continua a produzir os anexos esperados — **sem regressão**. | teste positivo em Go |
| A4 | As entradas ignoradas passam a ser **resumidas ao utilizador** (contagem + motivo), não só `slog.Debug`. | teste + saída |
| A5 | O ficheiro problemático **não desaparece em silêncio**: o utilizador sabe que o anexo falhou. | E2E/medição |

**Causa-raiz medida** (`app/cmd/app/webview.go:348-361`):

```go
report, err := attachments.ReadZipText(filename, textExts, attachments.DefaultZipLimits)
if err != nil {
    slog.Warn("zip attachment rejected", ...)   // <- so' LOGA
}
for _, skipped := range report.Skipped { slog.Debug(...) }   // <- so' DEBUG
for _, entry := range report.Entries { files = append(...) } // <- vazio quando falha
```

Quando `ReadZipText` devolve erro (ou quando todas as entradas são filtradas), o utilizador
recebe **zero anexos e nenhuma mensagem**: o zip «quebra» sem explicação.

---

## I0b — QR do Evolution

**Aceite:**

| # | critério |
| --- | --- |
| B1 | O painel mostra o QR de emparelhamento da instância Evolution configurada, obtido do **próprio servidor Evolution** (proxy autenticado), não inventado. |
| B2 | Sem Evolution configurado, a UI diz o que falta — não mostra um QR vazio. |
| B3 | Instância já ligada (`state=open`) → a UI diz «ligado», não pede QR. |
| B4 | Segredo do webhook/API key **nunca** aparece na resposta do proxy. |
| B5 | Teste negativo: Evolution inacessível → erro claro, sem QR falso. |

**Medido hoje:** não existe endpoint de QR no Hades (`/whatsapp`: webhook, status, send, dlq,
allowlist, config). O QR é gerado pelo servidor Evolution do utilizador.

---

## I0c — Barra de permissões por missão — **DONE** (ver commit)

**Causa-raiz medida:** o MissionCapabilityPicker existia mas vivia dentro de um <details> FECHADO — na prática o nível de acesso não aparecia.

**Aceite:**

| # | critério |
| --- | --- |
| C1 | Ao criar/editar uma missão, o nível de acesso é escolhido num controlo **sempre visível** (não escondido num passo). |
| C2 | O servidor **recusa** a missão quando a capacidade pedida não foi concedida (já medido: `400 tool capability is not granted to this mission: workspace.write`). |
| C3 | A UI mostra o nível efetivo da missão em execução e permite mudar antes de aprovar. |
| C4 | Teste negativo: pedir `workspace:write` sem conceder → recusa com mensagem; positivo: concedido → aceita. |

---

## I1 — Preview ao vivo COMPLETO

**Aceite:** editar o artefacto e **correr** a app gerada (servidor real, não `srcdoc` mock),
com reload automático; teste positivo (ficheiro alterado → preview muda) e negativo
(artefacto inválido → erro visível, sem preview em branco).

**Estado:** existe painel de passos + preview de artefacto (Studio → «Construção ao vivo»).
Falta edição e execução real.

---

## I2 — Roteamento multi-provedor

**Aceite:** escolher local / Ollama Cloud / externo por missão; **fallback explicável**
(qual provedor foi usado, porquê, e o que falhou antes); teste negativo (provedor sem
credencial → erro claro, sem cair para outro em silêncio).

---

## D — Enviar enquanto a IA responde (fila de envio)

**Aceite (escrito antes):**

| # | critério | prova |
| --- | --- | --- |
| D1 | Submeter durante o stream **enfileira** a mensagem; não descarta nem cancela a resposta em curso. | teste negativo do texto vazio + teste da fila |
| D2 | Ao terminar o stream, a fila é enviada por ordem, automaticamente. | teste positivo da fila |
| D3 | A UI mostra a contagem de mensagens na fila. | componente + i18n |
| D4 | **Parar** (botão ■) esvazia a fila: parar é parar. | teste da função de limpeza |
| D5 | Nada é perdido em silêncio: o texto enfileirado aparece na contagem. | teste |

**Causa-raiz medida** (`ChatForm.tsx:533`):

```ts
if (!message.content.trim() || isStreaming || isDownloading) return;   // descarta em silêncio
```

e os botões do composer em `ChatForm.tsx:1103/1108` usam `isStreaming || isDownloading`.
---

## I1 — Preview ao vivo COMPLETO — **medição feita, aceite revisto**

**Medido em 2026-10-21 (antes de escrever código):**

| facto | evidência |
| --- | --- |
| O artefacto de missão é servido por **HTTP** pelo próprio servidor, com `X-Artifact-SHA256` — **não** é `srcdoc` nem mock | `server/agent_routes.go:4435` (`artifact`), rota `GET /missions/:id/artifacts/:artifact_id` |
| Existe histórico por artefacto e **restauro** | rotas `.../versions` e `.../restore`; testes `artifact_restore_*` |
| O builder tem **Preview, Export, Undo, Redo** | `internal/agent/builder.go:270/309/396/430` |
| O builder tem **staging, publish e rollback** com testes de realidade | `builder_staging.go`, `builder_publish_report*.go`, `builder_rollback_test.go`, `builder_published_e2e_test.go` |
| O painel «Construção ao vivo» já mostra passos + preview do artefacto com reload a cada 4 s | `LiveBuildPanel` (separador do Studio) |

**Consequência:** «correr a app gerada» **já existe do lado do servidor** (HTTP + staging/publish). O que falta é
(a) **ligar** esse caminho ao painel de forma explícita (mostrar a URL de execução e o estado de publish, não
um blob), e (b) a **edição** — que NÃO deve ser um textarea a escrever ficheiros arbitrários: tem de usar o
caminho versionado (versões + restauro) ou o `UpdateVisual` do builder, com limites e auditoria.

**Aceite revisto, em dois passos (um commit cada):**

| passo | critério | prova exigida |
| --- | --- | --- |
| **I1a — correr** | O painel mostra a app gerada **servida por HTTP** (URL do artefacto/publish) em vez de um blob temporário; erro de fetch mostra mensagem, nunca preview em branco. | teste positivo (200 + SHA256) e negativo (404 → erro visível) |
| **I1b — editar** | Editar o conteúdo passa pelo **caminho versionado** (nova versão + possibilidade de restauro), com limite de tamanho e recusa de payload inválido. | positivo (salvar → nova versão → preview muda) e negativo (payload inválido/grande → 400) |

**Decisão de segurança registada:** não será criado um endpoint que escreva ficheiros arbitrários no
workspace. A edição usa o versionamento existente.
---

## CLI — `ollama list` panicava com digests curtos — **DONE** (verde 28/28)

**Report do dono:** «o hades não funciona em outros aplicativos».

**Causa-raiz:** `cmd/cmd.go:1184` e `1236` faziam `m.Digest[:12]`. Modelos remotos têm digest de 11
(`remote:aliba`, `remote:anthr`) → `panic: slice bounds out of range [:12] with length 11` em
`cmd.ListHandler`. O `/api/tags` respondia 200 em 0,1 s: o servidor estava bom, o **binário** morria.

**Prova:** `TestShortDigestNaoRebentaComDigestCurto` 5/5 (11, vazio, 1, 12, 64) + `ollama list`
reconstruído → tabela completa, exit 0, sem panic. Entregue ao produto: instalador reconstruído,
reinstalado, e o CLI **instalado** lista 7 modelos.

---

## Autostart do motor — **DONE** (verde 28/28, commit `dd7c2fe14`)

**Dois defeitos:**
1. `createLoginShortcut()` copia `{app}\lib\Ollama.lnk`, mas o `.iss` criava `lib\Hades.lnk`
   (`{#MyAppName}`). E ao corrigir o nome, o Inno gerou `Ollama.lnk.lnk` — a forma certa é `Name`
   **sem** extensão.
2. A chave `HKCU Run\Ollama` apontava para `...\Programs\Ollama\ollama app.exe`, **caminho morto**
   desde a desinstalação do Ollama antigo.

**Prova medida no sistema após instalar:** `lib\Ollama.lnk: True` e
`Run\Ollama = ...\Programs\Hades\ollama app.exe --hide`.

**Limite declarado:** a prova do arranque **no login** exige reiniciar a sessão — não executada.
---

## I1b — Editar o artefacto (caminho versionado)

**Medição feita antes de código (2026-10-21):**

| facto | evidência |
| --- | --- |
| O artefacto tem **histórico, comparação e restauro** com rotas | `server/artifact_versions_routes.go:37` (`artifactVersions`) e `:66` (`artifactRestore`); rotas em `agent_routes.go:983-984` |
| O comentário das rotas declara que o domínio já existia e testado em `internal/agent`, mas não tinha rota | `agent_routes.go:980-982` |
| A escrita de `restore` passa por CAS (`missionVersionMatches`), como todas as escritas | `agent_routes.go:982` |
| O runtime expõe **apenas leitura** de artefacto | `internal/agent/runtime.go:2769` (`Runtime.Artifact`) |

**Aceite:**

| # | critério | prova exigida |
| --- | --- | --- |
| E1 | Editar cria uma **nova versão** do artefacto; a anterior continua restaurável. | teste positivo (version→N+1, restore→N) |
| E2 | `GET ...?inline=1` passa a servir o **conteúdo novo** (o preview acompanha a edição). | teste positivo de conteúdo |
| E3 | Payload inválido é **recusado**: tamanho acima do limite, content-type não suportado, corpo vazio. | teste negativo (3 casos, 400) |
| E4 | Missão com versão desactualizada é recusada pelo CAS existente (sem sobrescrever trabalho novo). | teste negativo (409) |
| E5 | Nenhum caminho escreve fora do armazém de artefactos (sem `path` arbitrário no corpo). | revisão + teste |

**Decisão de segurança registada:** a edição **não** aceita caminho de ficheiro. Escreve conteúdo no
armazém de artefactos versionado, com limite de tamanho e CAS. Nada de `path` vindo do cliente.
---

## FATIA 2 — Multi-provedor com fallback EXPLICÁVEL

**Medição feita antes de código (2026-10-21):**

| facto | evidência |
| --- | --- |
| O roteamento já existe e tem testes | `internal/multillm/proxy_test.go` (rotas por modelo, gateway local, metadata do harness router), `native chat` |
| A regra local-only é testada | `internal/multillm/local_only_locality_test.go:37` — local-only nao escolhe remoto |
| NÃO existe explicação de fallback | busca por `Fallback|Explain|Motivo|Razao` só devolve `stop_reason` (fim de geração, outra coisa) |

**Aceite:**

| # | critério | prova exigida |
| --- | --- | --- |
| F1 | Toda resposta traz **qual** provedor/modelo serviu e **por que** foi escolhido. | teste positivo (campo presente e correto) |
| F2 | Quando há fallback, a resposta diz **o que falhou antes** (provedor + motivo), em ordem. | teste positivo (1 falha -> 1 entrada na cadeia) |
| F3 | Sem credencial para o provedor pedido: **erro claro e imediato**, sem cair para outro em silêncio. | teste negativo (status 4xx + mensagem; assert que nenhum outro provedor foi chamado) |
| F4 | Modo `local-only` nunca escolhe remoto, mesmo que haja credencial remota e local ocupado. | teste negativo (reusa `local_only_locality_test.go`) |
| F5 | A explicação não vaza credenciais (só nomes de provedor/modelo). | teste negativo (assert de ausência de chave/token no payload) |

**Decisão registada:** «explicável» significa **dados**, não prosa: a resposta leva uma cadeia estruturada
(`serviu`, `cadeia[{provedor, motivo_da_falha}]`), e a UI apenas a mostra. Nada de log solto — o cliente
tem de poder distinguir.

**Limite declarado:** a prova de fallback com **falha real de rede** exige provedor externo; será feita com
servidor HTTP falso que devolve erro conhecido (httptest), e a prova com provedor verdadeiro fica em
PEDIDOS_EXTERNOS (credencial real).
---

## FATIA 4 — FERRAMENTAS NO CHAT (decisao do dono: 2026-10-21)

**Report que originou:** "ELE NAO FAZ" -- o dono pediu uma pasta com ficheiros no chat e recebeu instrucoes
de terminal (e comandos INVENTADOS: create-dir, create-file, list-files, open-browser nao existem no binario;
reais: serve, create, show, run, stop, pull, push, signin, signout, list, ps, cp).

**Medicao que descartou a hipotese anterior:** o dono trocou de modelo por TODOS e nada mudou. Logo nao era
o modelo. As ferramentas existem no runtime (a.runtime.ListTools()) e sao expostas pelas MISSOES; o chat
envia por useSendMessage (app/ui/app/src/hooks/useChats.ts:198), um caminho SEM ferramentas.

**Decisao do dono:** ligar as ferramentas AO CHAT. A fatia e' essa.

**Aceite:**

| # | criterio | prova exigida |
| --- | --- | --- |
| T1 | Pedido de execucao no chat ("cria a pasta X com index.html") produz FICHEIROS no disco, e nao uma lista de instrucoes. | teste positivo: ficheiro existe com o conteudo pedido |
| T2 | O chat mostra que executou (ferramenta + resultado), nao so texto. | teste positivo: evento de ferramenta visivel no stream |
| T3 | Comando/ferramenta inexistente e' RECUSADO na validacao -- o modelo nao pode inventar CLI. | teste negativo: pedido que so' existe na alucinacao -> recusa explicita |
| T4 | Quando a superficie nao tem ferramentas, a resposta NAO finge: diz que nao pode executar ali. | teste negativo: mensagem explicita, sem passos falsos |
| T5 | Nada escreve fora do workspace, como na edicao de artefactos. | teste negativo: caminho com travessia recusado |

**Restricao:** manter o default seguro (loopback). Nada de expor rede.

**Limite declarado:** T1/T2 exigem app instalada com modelo que emita tool calls (llama3.1:8b, qwen2.5:7b,
deepseek-v4.1-flash) -- modelos -coder e deepseek-r1 nao emitem, e o chat deve dize-lo em vez de fingir.