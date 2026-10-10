# Loop de escrita em /api/v1/settings: diagnóstico e correção

Data: 2026-10-09
Escopo: `app/ui/app` (interface web)
Estado: corrigido e verificado

## Sintoma

A home (`/`, alias mascarado de `/c/new`) ficava não interativa: o renderer
saturava e nenhum clique respondia. O problema não aparecia em teste unitário —
só em execução real contra um backend que responde.

## Medição

Probe E2E dedicado (`e2e/settings-write-loop.spec.ts`), contando requisições por
método e rota durante 3 s de carregamento da home:

Antes:

```
LOOP_EMPTY_COUNT=2387
LOOP_TOP 1206 POST /api/v1/settings
LOOP_TOP 1181 GET  /api/v1/settings
LOOP_TOP 2    POST /api/me
LOOP_TOP 2    GET  /api/version
```

Todas as demais rotas ficaram com no máximo 2 requisições. O problema é
exclusivamente o par POST+GET de `/api/v1/settings`, a ~400 pares por segundo.

Depois:

```
LOOP_EMPTY_COUNT=16
LOOP_EMPTY_COUNT_LONG=16
LOOP_TOP 9 GET  /api/v1/settings
LOOP_TOP 7 POST /api/v1/settings
```

O mesmo total em 3 s e em 6 s: o fluxo converge e para. As 7 escritas restantes
são intenções distintas (cada uma com sua própria chave), não repetições.

## Causa raiz

`useSettings` grava por POST e invalida a query em caso de sucesso; a
invalidação provoca um GET que devolve um **objeto `Settings` novo**. Como
`settingsData.settings` e `setSettings` são recriados a cada leitura, todo
efeito que os tenha nas dependências roda outra vez.

Isso é inofensivo quando o backend devolve o valor gravado — o efeito vê o campo
preenchido, sua condição de parada é satisfeita e ele para. Deixa de ser
inofensivo quando o backend **aceita a escrita mas não a ecoa** (payload
parcial, proxy ou endpoint somente leitura): a condição de parada nunca é
satisfeita e o ciclo grava → invalida → relê → grava se mantém.

Dois efeitos "garanta X" caíam nesse caso:

- `src/routes/c.$chatId.tsx`: grava `LastHomeView: "chat"` e usa
  `settingsData.LastHomeView === "chat"` como condição de parada. Com payload
  vazio o campo é `undefined` para sempre.
- `src/hooks/useSelectedModel.ts`: grava o modelo padrão e usa
  `settings.selectedModel` como condição de parada, pelo mesmo motivo.

## Correção

A correção é na origem, não em cada call site: `useSettings` passa a não
repetir uma escrita idêntica enquanto o estado observado não mudar.

- `settingsWriteKey(updates)`: identifica a intenção de escrita de forma estável
  (independente da ordem das propriedades).
- `settingsSnapshot(settings)`: fotografia do último payload lido.
- `shouldSkipSettingsWrite(guard, key, snapshot)`: bloqueia a repetição somente
  quando a chave é a mesma **e** o snapshot não mudou.

Uma escrita que **falhou** não foi aceita, então não conta como repetição: o
`onError` da mutação limpa a guarda e libera o retry. Sem isso, a tela de
onboarding quebrava — "Tentar de novo" reenvia o payload idêntico e precisa
surtir efeito. Falha também não gera invalidação, portanto não realimenta o
ciclo.

`useSelectedModel` mantém a decisão de escolher o modelo padrão extraída para a
função pura `defaultModelRequest`, agora testável (seleção válida, seleção
desconhecida, modelo de provider, nada selecionado). A repetição da escrita é
responsabilidade da guarda central, não de um contador local.

## Verificação

- `src/hooks/useSettings.test.ts` (novo): 9 casos, incluindo 100 iterações
  simulando refetch sem eco, com 0 escritas repetidas.
- `src/hooks/useSelectedModel.test.ts` (novo): 7 casos.
- `e2e/settings-write-loop.spec.ts` (novo): laço de regressão permanente; afirma
  menos de 50 requisições e exige 0 requisições na segunda janela de 3 s.
- `npm test -- --run`: 412 testes em 67 arquivos, todos passando.
- `npm run lint`: 0 erros.
- `npm run build:check`: typecheck, build e orçamento de bundle OK.
- `npx playwright test`: 7 testes E2E, todos passando.

## Não afirmações

- O cenário medido é o de backend que aceita e não ecoa. Não foi medido
  comportamento contra backend lento ou que responda 5xx de forma contínua.
- A guarda elimina repetição dentro de uma instância de `useSettings`. Telas
  diferentes têm guardas independentes, o que é intencional: são intenções
  independentes.
