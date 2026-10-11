# CRM de contatos, leads e oportunidades — e o comando `/crm` real

## 1. O que faltava

O comando `/crm` estava **registrado como indisponível**, com o motivo *"não existe CRM
nesta versão: sem rota de contatos/leads"*. Isso era **honesto** — e era exatamente o
problema: a diretriz §11.1 exige que todo comando do catálogo **mapeie para rota real**,
com testes. O estágio 27 também pede *"integração a CRM comum"*, e as etapas 29 e 35
reusam contatos.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **Deduplicação por identidade normalizada** | o mesmo cliente entrando por canais diferentes **não pode virar três contatos**: isso quebra histórico, consentimento e cobrança |
| **Nono dígito do celular brasileiro normalizado** | bases legadas guardam `551188887777` e `5511988887777` para o MESMO número. A regra vale só para celular (após o DDD, iniciando em 6-9); **fixo não é mesclado** |
| **Contato sem telefone e sem e-mail é recusado** | sem identidade não há como reconhecer a pessoa **nem respeitar o opt-out** |
| **Consentimento NÃO é copiado para o contato** | vem do `ConsentRegistry`, fonte única do opt-out. Duas verdades sobre consentimento seriam pior do que nenhuma |
| **Fase terminal é terminal** | `won`/`lost` não voltam atrás; a data de fechamento fica gravada. Reafirmar a mesma fase é permitido (idempotente); reabrir, não |
| **Escopo por organização** em toda leitura e escrita | contato e oportunidade de outra organização não são lidos, alterados nem cruzados |

## 3. Alcançável pelo produto

| Rota | Comportamento |
| --- | --- |
| `GET /crm/contacts?q=` | lista e busca por nome, telefone, e-mail ou etiqueta |
| `POST /crm/contacts` | **201** na criação, **200** com o **mesmo contato** quando a identidade repete |
| `GET /crm/contacts/:id` | contato + **consentimento real por canal** + oportunidades dele |
| `POST /crm/contacts/:id/opportunities` | abre oportunidade |
| `POST /crm/opportunities/:id/stage` | muda fase; **recusa** reabrir encerrada e fase inválida |
| `GET /crm/pipeline` | funil por fase: contagem, totais, abertas, ganhas, perdidas |
| `POST /crm/contacts/:id/consent` · `.../consent/revoke` | concede e revoga consentimento **pelo próprio CRM** (motivo obrigatório) |

E o comando `/crm` passou a apontar para `GET /api/agent/v1/crm/contacts`, deixando de
ser `Available: false`.

## 4. Fechamento de um limite declarado

Em **E-101** o consentimento existia mas **não tinha rota**: o limite foi declarado
naquele momento. Aqui ele é fechado: conceder e revogar consentimento passaram a ter
rota, com o opt-out bloqueando o envio pelo gateway (mesma verdade, sem sincronização).

## 5. Meus erros, corrigidos pelos testes

1. **9º dígito**: meu teste exigia que `551188887777` e `5511988887777` fossem o mesmo
   contato, e o código não normalizava. O **teste estava certo e o código errado** —
   implementei a normalização (restrita a celular, para não mesclar fixo com celular).
2. **Contagem desatualizada**: depois de incluir o telefone fixo, a asserção que
   esperava 2 contatos passou a estar errada (são 3: celular, fixo e e-mail). Ajustei a
   expectativa, mantendo a verificação de que **fixo não é mesclado**.
3. **Teste do registro de comandos codificava a verdade antiga** (`/crm` indisponível).
   Atualizei para afirmar a **nova** verdade — disponível **e** apontando para a rota
   real — em vez de remover a verificação.

## 6. Verificação

- **5 testes de motor**: deduplicação (três formatos do mesmo celular, fixo **não**
  mesclado, e-mail ignorando caixa) · contato sem identidade recusado · funil com
  indicadores e fase terminal · **consentimento lido do registro, nunca copiado** (com
  opt-out refletindo imediatamente) · reinício + isolamento por organização + busca.
- **3 testes de rota**: deduplicação pelo HTTP (201 depois 200 com o **mesmo ID**),
  400 sem identidade, funil, fase (ganho → reabertura recusada → fase inválida 400),
  contato inexistente recusado · concessão/revogação de consentimento pelo CRM, com
  revogação sem motivo em 400 e **verificação de que o opt-out bloqueia o envio** ·
  isolamento por organização.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes
  apenas com as **falhas ambientais conhecidas**.

## 7. Limites declarados

1. **Sem integração com CRM externo** (HubSpot/Salesforce seguem **apenas catálogo**):
   sincronizar com eles exige conta e credencial — dependência externa.
2. **Sem histórico de conversa** por contato (a caixa de entrada é o estágio 29).
3. **Sem agentes comerciais** (consultor, qualificador, propostas, pós-venda).
4. **Sem anexos, propostas em PDF ou automação de follow-up**.
5. **Sem deduplicação por nome** (só por identidade forte: telefone ou e-mail) — de
   propósito: mesclar homônimos apagaria histórico de pessoas diferentes.
