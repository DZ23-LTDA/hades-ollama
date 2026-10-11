# Estágio 29 — caixa de entrada unificada (e três comandos destravados)

## 1. O que faltava

`/inbox`, `/handoff` e `/support` estavam registrados como **indisponíveis**, com o
motivo *"a caixa de entrada unificada ainda não existe nesta versão"* — honesto, e era
exatamente o que faltava. Com a caixa implementada, os três ganharam rota real e a
**lista de comandos indisponíveis ficou VAZIA**: todos os **35** comandos do §6 têm rota.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **Uma conversa por contato e canal** | mensagem nova **reaproveita** a conversa aberta; criar uma conversa por mensagem destruiria histórico e contexto — o oposto do que o estágio pede |
| **Deduplicação de entrada por `ExternalID`** | webhook reentregue **não** pode duplicar mensagem (§11.7). Checagem e inserção sob o mesmo lock, então vale para entregas **concorrentes** |
| **SLA medido, nunca presumido** | o relógio para na **primeira resposta real**; sem resposta, a conversa aparece **violada** depois do prazo |
| **Handoff é estado, não texto** | exige **motivo**, registra responsável e move para `pending_human`; a passagem fica **na própria conversa** |
| **Consentimento consultado no ENVIO** | responder quem pediu opt-out é **recusado**; a caixa de entrada não é porta dos fundos para contatar quem não autorizou |
| **Conversa encerrada não é reaproveitada** | a próxima mensagem abre conversa nova, para o histórico encerrado não ser reaberto por acidente |

## 3. Alcançável pelo produto

| Rota | Comportamento |
| --- | --- |
| `GET /inbox/conversations?status=` | lista por status, escopada por organização |
| `POST /inbox/messages` | **201** na primeira entrega; **200 com a mensagem ORIGINAL** e `deduplicated=true` na reentrega |
| `GET /inbox/conversations/:id` | conversa + mensagens + **aviso de resposta bloqueada** por consentimento |
| `POST /inbox/conversations/:id/reply` | resposta; **recusa** sem consentimento válido |
| `POST /inbox/conversations/:id/assign` · `.../handoff` · `.../close` | responsável, transferência (com motivo) e encerramento |
| `GET /inbox/metrics` | SLA violado, aguardando primeira resposta, fila humana, volumes |

## 4. Meus erros, corrigidos pelos testes

1. **`/support` não casou** com o texto que eu esperava no registro (o motivo era outro);
   ajustei a substituição ao texto **real** em vez de presumir.
2. **Três testes codificavam a verdade antiga** ("existe comando indisponível"): o do
   registro de comandos, o de `ParseSlashInvocation` e o de `runCommand` (que exigia
   **501** para `/inbox`). Atualizei os três para a **nova** verdade, sem remover
   cobertura:
   - o caminho **501** passou a ser exercitado por um spec **sintético**
     (`checkSlashCommandAvailable`), então não depende de existir comando sem rota — e
     extraí a checagem para um **ponto único**, em vez de criar API só para o teste;
   - `/inbox` passou a ser verificado como **executando** (200), não como indisponível.

## 5. Verificação

- **6 testes de motor**: reaproveitamento de conversa por contato e canal (e nova
  conversa após encerramento) · **deduplicação de webhook**, incluindo **10 entregas
  concorrentes** do mesmo evento gerando **uma** mensagem, e mensagem sem chave externa
  **não** deduplicada · SLA medido (antes do prazo sem violação, depois com violação,
  média real de 120s) · handoff com motivo obrigatório e indicador de fila humana ·
  resposta respeitando consentimento (sem consentimento → recusa; com → envia; opt-out →
  recusa) · reinício com índice de deduplicação reconstruído + isolamento por organização.
- **3 testes de rota**: dedupe pelo HTTP (201 depois 200 com a **mesma** mensagem),
  detalhe com `reply_blocked`, resposta recusada, handoff sem motivo 400, métricas,
  filtro por status, encerramento, 400 sem corpo · isolamento por organização ·
  os três comandos disponíveis apontando para `/inbox/`.
- A **guarda do §11.1** (E-103) continua verde e agora cobre os três comandos novos.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes
  apenas com as **falhas ambientais conhecidas**.

## 6. Limites declarados

1. **Canais reais não conectados**: a caixa recebe mensagens pelas rotas; ligar WhatsApp
   e Telegram reais depende de credencial (dependência externa).
2. **Sem webhook público**: a entrega do provedor é simulada por `POST /inbox/messages`;
   o endpoint de webhook com validação de assinatura é trabalho do estágio 34.
3. **Sem base de conhecimento** nem resposta automática por IA dentro da conversa.
4. **Sem anexos e sem áudio**: só texto nesta rodada.
5. **SLA de primeira resposta apenas** (sem SLA de resolução nem calendário útil).
