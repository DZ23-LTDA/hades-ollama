# Diagnóstico de falha de execução de plugin (E-185, limite B-14)

## 1. O que B-14 dizia e o que eu fui verificar

O limite registrado em E-184 dizia: fora de um host com privilégio de montagem, a execução de plugin falha com
`exit status 1` e **`stdout`/`stderr` VAZIOS** — falha silenciosa. Fui atrás da causa **antes** de aceitar o sintoma.

## 2. A causa não estava no sandbox — estava em QUEM DESCARTAVA o diagnóstico

O executor de sandbox **sempre** devolve `stdout`/`stderr` no `Value`, inclusive no caminho de erro
(`tools.go`). O defeito era do runtime de plugin: no caminho de falha ele devolvia

    PluginExecutionResult{Record: record}, fmt.Errorf(...)

ou seja, **jogava fora o `Output`**. Toda falha chegava ao operador como `exit status 1` e nada mais. O stderr existia
e era descartado — o que eu tinha lido como "stderr vazio" era, na verdade, um campo **nunca propagado**.

Segundo defeito menor, medido: no caminho de falha o sandbox **não** publicava `exit_code` (só no sucesso).

## 3. A correção

- `plugin_runtime.go`: a falha agora devolve `Output: result.Value` — o diagnóstico acompanha o erro;
- `tools.go`: o caminho de falha publica `exit_code` a partir de `*exec.ExitError` quando o processo chegou a rodar.

## 4. Prova: o que o operador vê agora (container `node:22-bookworm`, best-effort aprovado)

    diagnóstico do sandbox: exit_code=1 stderr="unshare: unshare failed: Operation not permitted\n" stdout=""

Antes: `exit status 1` **sem mensagem**. Agora: **código de saída e a razão exata** — e a razão confirma o limite de
ambiente (o container não tem privilégio para `unshare`, isto é, para os namespaces que o sandbox exige).

## 5. Teste que trava o comportamento

`TestPluginExecutionFailureCarriesSandboxDiagnostics` usa um plugin que falha de propósito (`process.exit(3)`) e
exige: (a) o registro de auditoria **não** pode dizer sucesso; (b) a falha precisa virar erro; (c) o diagnóstico
precisa carregar o mapa do sandbox com `stderr` (ou `exit_code`); (d) precisa haver registro de auditoria. O stderr
medido é **impresso no log do teste**, então o CI passa a mostrar a razão da falha.

Resultado: **PASS no Linux** (container) e **SKIP declarado no Windows** (onde o executor recusa antes de rodar).

## 6. O que continua aberto em B-14

O **pré-requisito de host permanece**: o sandbox exige namespaces de montagem (`unshare --user --map-root-user
--mount`), indisponíveis no Windows, em container não privilegiado e no runner padrão de CI. O que **deixou de
existir** é a **falha silenciosa**: agora a razão aparece. Desbloqueio do restante: executar em host com esses
namespaces (ou com o modo `strict` configurado via `OLLAMA_AGENT_SANDBOX_CGROUP_ROOT`).

## 7. Limites declarados

- Não corrigi o pré-requisito de host: não tenho host privilegiado nesta sessão, e forjar isso seria mentira.
- O teste prova que o diagnóstico chega; **não** prova execução bem-sucedida de plugin.
- `exit_code` só aparece quando o processo chegou a rodar; falhas anteriores (recusa de plataforma) seguem sem ele —
  e é por isso que a asserção aceita `stderr` **ou** `exit_code`.
