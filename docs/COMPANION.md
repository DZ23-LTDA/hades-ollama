# HADES Companion — o pet contextual

O **Companion** é a representação visual do estado real de uma missão. Ele não é um chat, não é
um assistente e não decide nada: ele **lê** o estado que o servidor derivou de eventos verificados e
o **diz**.

> ***A regra que governa tudo: o Companion não inventa estado nenhum.*** Um mascote que anima sozinho
> é um mascote que **decide** que trabalho está a acontecer — e essa decisão não lhe pertence.

---

## Para quem usa

### O que ele mostra

Cada estado tem um título e uma explicação. A explicação é sempre verificável contra a **razão**
(o evento que produziu o estado), que aparece por baixo do texto em «Procedência».

| Estado | O que significa |
| --- | --- |
| **Sem missão em curso** | Não há eventos desta missão. *Não estou a fazer nada — e digo-o em vez de rodar.* |
| **A planear** | A missão foi criada e está a ser planeada. |
| **A trabalhar** | Há um passo aberto. *É o único estado em que posso dizer isto.* |
| **À espera da sua decisão** | Há uma aprovação pedida e não resolvida. *Não estou a avançar: estou à sua espera.* |
| **Bloqueado** | A missão declarou um bloqueio. *Ele não se levanta sozinho.* |
| **Concluído** | A missão terminou com um evento de conclusão. |
| **Falhou** | A missão declarou falha. O motivo está na razão. |
| **Cancelado** | A missão foi cancelada. |

### O que ele **não** faz

- **Não responde perguntas.** Não é um chat. *Um mascote que inventa respostas seria uma alucinação
  com olhos.*
- **Não fala.** Não há provider de fala ligado neste ambiente; o botão diz «Voz indisponível» em vez
  de fingir. É `BLOCKED_BY_EXTERNAL_DEPENDENCY`.
- **Não cria missão, não executa ferramenta, não autoriza ação e não gasta recursos.** O botão de voz
  só é chamado quando **você** carrega.

### O que você controla

Três botões, e a escolha persiste **neste dispositivo**:

| Botão | Efeito |
| --- | --- |
| **Ocultar Companion** / **Mostrar Companion** | Não desenha nada. *Desligar não perde nenhuma funcionalidade central.* |
| **Silenciar avisos** / **Ouvir avisos** | O anúncio para leitores de ecrã passa a `off`. **O texto continua na página** — o leitor deixa de ser interrompido, mas o estado continua legível. |
| **Restaurar** | Repõe o default: visível, a falar baixo, a animar como os outros. |

Se a escolha **não** puder ser guardada (armazenamento indisponível, modo privado), o Companion
**diz isso**: «A escolha não foi guardada (armazenamento indisponível). Vale só nesta sessão.»

### Acessibilidade

- **`prefers-reduced-motion`** desliga as animações. *O estado continua a ser lido e anunciado.*
- O mascote em si é `aria-hidden` — *o estado é anunciado pelo texto ao lado, e um leitor de ecrã não
  precisa de ouvir duas vezes a mesma coisa.*
- O `aria-live` é `polite`, e só passa a `assertive` quando há **uma aprovação à sua espera** — o
  único caso em que a ação é sua.
- O estado nunca é comunicado **só** por cor ou animação: há sempre título, explicação e procedência.

---

## Para quem desenvolve

### Arquitectura

```
eventos persistidos da missão                  (o servidor)
        │
        ▼  internal/agent/companion_eventos.go
CompanionEventoVerificado[]                    (vocabulário FECHADO, 12 eventos)
        │
        ▼  internal/agent/companion_estado.go
(CompanionEstado, razao)                       (8 estados, precedência explícita)
        │
        ▼  server/agent_routes.go  GET /api/agent/v1/missions/:id/companion
{mission_id, estado, razao, scope}             (o `scope` é DECLARADO)
        │
        ▼  app/ui/app/src/lib/companionApi.ts
valida o estado na fronteira (ESTADOS_CONHECIDOS)
        │
        ├──▶ CompanionMascote.tsx   (o componente: só diz o que recebeu)
        └──▶ companionPreferencias.ts (o que o utilizador escolheu)
                 │
                 ▼  CompanionLigado.tsx   (a rota: AgenticSplitShell.tsx:729)
```

### As quatro regras do derivador

1. **Sem eventos ⇒ `ausente`.** *É o «inventar agente trabalhando» no seu caso mais puro.*
2. **Evento desconhecido ⇒ RECUSA** (`ErrCompanionEventoDesconhecido`). *Recusar é melhor do que
   mapear por semelhança — «parece um evento de sucesso» não é um evento de sucesso.*
3. **Evento de outra missão ⇒ ignorado.** *O Companion de uma missão não fala de outra.*
4. **`concluido` e `falhou` SÓ com os seus eventos.** *Nunca por omissão, nunca por tempo, nunca por
   ausência de sinal — um silêncio não é um sucesso.*

### Os 12 eventos verificados

```
mission.created              mission.approval.requested    mission.completed
mission.planning             mission.approval.granted      mission.failed
mission.step.started         mission.approval.denied       mission.cancelled
mission.step.finished        mission.blocked
                             mission.repair_exhausted
```

Qualquer outro evento (`mission.planned`, por exemplo) é **descartado** pelo conversor — e o descarte
é **contado** (`ConversorEventosDoCompanionComRelatorio`), para não ser silencioso.

### Os 8 estados e o mapa para os 14 do prompt

| Prompt | HADES | Nota |
| --- | --- | --- |
| `IDLE` | `ausente` | |
| `PLANNING` | `a_planear` | |
| `WORKING` | `a_trabalhar` | |
| `APPROVAL_REQUIRED` | `a_aprovar` | |
| `BLOCKED` | `bloqueado` | |
| `ERROR` | `falhou` | |
| `CANCELLED` | `cancelado` | |
| `SUCCEEDED` | `concluido` | |
| `DISABLED` | — | É **preferência do utilizador**, não estado de missão. |
| `LISTENING` | — | Não há captação de áudio; voz = `BLOCKED_BY_EXTERNAL_DEPENDENCY`. |
| `WAITING_TOOL` | — | Não há evento `mission.tool.*`. O passo aberto cobre «a trabalhar». |
| `WAITING_USER` | — | `a_aprovar` **é** a espera pelo humano. |
| `PAUSED` | — | **Não há `mission.paused`** no código; inventar o estado sem o evento seria derivar do nada. |
| `OFFLINE` | — | É estado de **conexão**; o derivador não lê rede, de propósito. |

### Preferências: onde ficam e porquê

`companionPreferencias.ts` guarda **três booleanos** em `localStorage`, com a chave **versionada**
`hades.companion.preferencias.v1`.

- **Porquê `localStorage` e não POST:** a preferência é do **utilizador, neste dispositivo**. Um POST
  por cada mudança seria a «chamada em loop» que o contrato proíbe, e uma chamada de rede para
  decidir se um desenho aparece.
- **`validarPreferencias` não confia no armazenamento:** um campo com o tipo errado cai no default
  **desse** campo. *O `localStorage` é escrito por qualquer script da mesma origem e é editável à mão.*
- **`guardarPreferencias` devolve `false`** quando não conseguiu guardar — e a UI **diz** isso. *A
  ausência declara-se.*

### O que **não** tem, e porquê

- **Timers.** Zero `setInterval`, `setTimeout` e `requestAnimationFrame`. *A animação é CSS, derivada
  de `data-state` e das preferências.*
- **Listeners globais.** Zero `addEventListener`, zero `window.*`, zero `document.*`.
- **Estado próprio.** O component não calcula o estado: recebe-o.

### Testes

| Ficheiro | Testes | O que prova |
| --- | --- | --- |
| `internal/agent/companion_estado_test.go` | 9 | As quatro regras do derivador. |
| `internal/agent/companion_eventos_test.go` | 7 | A redação/DLP, o vocabulário, a ordem, o isolamento. |
| `server/agent_companion_routes_test.go` | 1 | A rota, o `scope`, e `org-b` recusada. |
| `src/lib/companionPreferencias.test.ts` | 21 | Validação, persistência, precedência. |
| `src/components/CompanionMascote.test.tsx` | 18 | Estados, anúncio, preferências, regressão ao byte. |
| `src/components/CompanionLigado.preferences.test.tsx` | 11 | As 3 ações + o contrato estrutural. |
| `src/components/companionPerformance.test.ts` | 7 | Zero timers, zero listeners, cleanup. |
| `e2e/companion.spec.ts` | 2 | HTTP real contra o backend mínimo (2/2, ~29 s). |

### Limitações declaradas

- **Não há E2E da jornada completa** do pet com o backend Go real + Ollama (`idle → /goal → planning →
  working → approval → resume → success`). O `companion.spec.ts` prova a **fiação**, não a jornada.
- **Não há medição de CPU/RAM/bateria** com o componente montado por minutos. A catraca mede a
  **causa** (timer, listener, fetch sem guarda), não o sintoma.
- **Não há provider de fala.** O botão declara a ausência.
