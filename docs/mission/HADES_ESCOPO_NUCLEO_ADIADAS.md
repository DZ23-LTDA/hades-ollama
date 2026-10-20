# Escopo: NÚCLEO, ADIADAS e BLOCKED_EXTERNAL

> REGRA 2 do adendo: *«registre quais jornadas ficam no NÚCLEO, quais são ADIADAS e quais são
> BLOCKED_EXTERNAL. "Falta de margem" não é bloqueio externo e não autoriza parar.»*
>
> Este documento é a classificação corrente. Ele **não** diz que o NÚCLEO está pronto: diz o que está
> dentro, o que está fora por decisão e o que está fora por impedimento alheio ao repositório.

## NÚCLEO — o que fica, com o que está provado

| jornada | estado | prova |
| --- | --- | --- |
| Vault governado: escrita/leitura/remoção/compactação com `PermissaoEscrever` e identidade vinda da sessão | implementado e medido | `E-1442`, `E-1470`, E2E do vault **0 falhas** (75 asserções) |
| Fronteira de tenant dos builds (5 handlers + registo com dono) | implementado e provado ao nível HTTP | `E-1470`, `E-1471`, `E-1472`, `E-1477`, `E-1478` |
| Recomendação de execução a partir da VRAM medida (quantização/contexto/offload parcial) | função testada, **não exposta** | `E-1489`, `E-1490`, `E-1491` |
| Proposta conversa→goal com motivo e com o que falta | função testada, **não exposta** | `E-1493` |
| Pare de emergência | primitiva testada **+ consultada** pelo executor de build e pela rota HTTP; **sem gatilho** | `E-1496`, `E-1497`, `E-1498` |
| Cabeçalhos de segurança (CSP mínima + `nosniff`) | ligado e validado no CI pelos painéis | `E-1492`, `E-1494` |
| Instrumentos da bateria (checkers, fault injection, gates) | verdes | `E-1474`, `E-1475`, `E-1476`, `E-1495` |

## ADIADAS — fora por decisão desta janela, não por impedimento

| jornada | porque está adiada |
| --- | --- |
| **Gatilho** do pare de emergência (rota/UI para acionar) | exige definição de autorização: *quem* pode parar tudo, com que credencial e com que registo. Expor sem essa definição seria um botão sem dono |
| Wiring do pare nos restantes executores (`builder_staging.go`, rotas de agente) | um executor de cada vez, com teste próprio; o build é o primeiro |
| Exposição da recomendação de hardware (`doctor`/rota) | precisa de VRAM **medida** no host e de decisão de superfície; hoje só existe a função |
| Exposição da proposta conversa→goal | precisa de decidir onde a proposta aparece e quem a aceita; a função não cria nada |
| `script-src`/`style-src`/`connect-src` na CSP | exige medir o que a SPA carrega (bundles, workers, websockets); um teste **proíbe** acrescentar sem essa medição (`E-1492`) |

## BLOCKED_EXTERNAL — impedimento alheio ao repositório

| jornada | bloqueio concreto | o que **não** é |
| --- | --- | --- |
| E2E do app móvel (login/aprovação/outbox/UI) | sem toolchain Android e sem emulador nesta máquina | não é falta de margem: as políticas e o `tsc --noEmit` estão provados |
| Assinatura e publicação de artefactos | a missão **proíbe** assinar/publicar; o `SigningProvider` mede aptidão em vez de assinar (`E-1482`) | não é falta de código |
| `race (macos)` corrido **localmente** | este host é Windows; o job de CI corre e passa (`E-1488`) | **não** é o mesmo que «CI não cobre macOS» |
| Benchmarks de MLX/Docker/Semgrep e build Android | ferramentas ausentes no host | não é ausência de desenho |
| Homologação independente | é externa **por definição**: quem construiu não homologa | não é uma tarefa que o agente possa fechar |

## O que este documento NÃO autoriza

- Não autoriza chamar «pronto» ao NÚCLEO: **três** itens do NÚCLEO estão implementados mas **não
  expostos**, e o pare não tem gatilho.
- Não autoriza transformar ADIADA em BLOCKED_EXTERNAL: as cinco adiadas dependem de **decisão**, não de
  terceiros.
- Não autoriza fechar o veredito: `release_state` continua `NOT_RELEASE_READY` até homologação
  independente.
