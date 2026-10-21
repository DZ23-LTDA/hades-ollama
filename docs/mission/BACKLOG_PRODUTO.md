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