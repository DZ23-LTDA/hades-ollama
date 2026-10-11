# Estágio 35 — o teste que media a máquina (E-155)

## 1. O que estava frágil

A auditoria dos testes do pacote `internal/agent` encontrou os pontos em que uma espera **curta** decidia o
resultado:

```
13 esperas de 2 s e 3 s em testes que apenas afirmam "o evento acontece":
  worker reivindica o job · ferramenta começa · ferramenta para ao cancelar ·
  runner do swarm inicia/retorna · notificação termina após liberar o provedor ·
  revogação interrompe I/O · cancelamento durável propaga · lock em processo termina
```

Nenhum deles **testa** que algo estoura por tempo: todos afirmam que algo **acontece**. Isso significa que
2 s não era um critério de correção — era um teto para a velocidade da máquina. Em contêiner carregado (o
gate Linux) isso vira resultado de teste. É a mesma família de E-142 (o caso que falhava 3 s antes da virada
da hora) e a hipótese mais provável para a falha de paridade **não capturada** de E-154.

## 2. O que mudou

| Mudança | Efeito |
| --- | --- |
| `waitBudget(t)` (novo helper de teste) | Orçamento **5–30 s**, derivado do prazo do teste e **limitado** |
| 12 esperas de 2 s/3 s trocadas | A asserção continua idêntica; some a dependência da máquina |
| Fixture SSE do MCP | Passou a esperar o **cliente desconectar** em vez de dormir |
| `scripts/verify-linux-parity.ps1` | **Grava sempre** a saída e imprime o nome dos testes que falharam |

O limite de 30 s é deliberado: uma espera que nunca termina precisa falhar rápido o bastante para **dizer
qual evento não aconteceu**, em vez de estourar o timeout do pacote e esconder o nome do teste.

## 3. O achado que só a medição por teste revelou

Depois das 12 trocas, a suíte foi de ~64 s para **98,261 s** — e um único teste apareceu com **30,00 s**:

```
30.00s  TestRemoteMCPSSEReturnsCorrelatedErrorBeforeEOF
```

Ali a espera **não** era um evento: era a **fixture** segurando o `text/event-stream` aberto para provar que o
erro correlacionado chega **antes do EOF**. O sono arbitrário atrapalhava; o certo é esperar o fim da
requisição (`<-r.Context().Done()`), que é determinístico.

- antes: **30,00 s** · depois: **0,05 s**
- suíte: **98,261 s** → **63,901 s**, com **nenhum teste ≥ 20 s**

Se eu tivesse assumido que "trocar 2 s por 30 s é inofensivo", teria deixado a suíte 55% mais lenta **e** um
teste preso a um relógio. Foi a medição por teste, não a intuição, que mostrou isso.

## 4. O flake de E-154: o que ficou provado

| Execução | Resultado |
| --- | --- |
| Paridade completa, 3× (E-154) | OK nas três |
| **Pacote do agente no Linux, `-count=5`** | **ok 96,563s, NENHUMA falha** |
| Paridade completa depois do endurecimento | **OK** (agente 22,102s · servidor 6,790s) |

São **9 execuções limpas** da região afetada desde a única falha. **Não** afirmo qual teste falhou — o nome se
perdeu por erro meu de processo — e por isso **não** declaro a falha "resolvida": declaro que a **classe** de
fragilidade que a explicaria foi removida, e que o defeito segue sem causa provada.

O erro de processo foi corrigido na fonte: a paridade agora **sempre** grava
`_tmp/linux-parity-last.log` (mais uma cópia com carimbo de tempo) e, em caso de falha, imprime os nomes dos
testes que falharam — ou diz explicitamente que a falha foi de compilação/`vet`/imagem.

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -count=5` (Linux, contêiner) | **ok 96,563s · 0 falhas** |
| `go test ./internal/agent/ -count=1` | **ok 64,798s** (sem teste ≥ 20 s na medição por teste) |
| `go test ./server/ -count=1` | **ok 23,562s** |
| `go test ... -run 'TestServiceAgent\|TestRuntimeServiceAgents'` | **78/78 PASS** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agente 22,102s · servidor 6,790s) e log gravado |
| `gofmt -l internal/agent server` | **sem saída** |

## 6. Limites declarados

- O orçamento é de **5 a 30 s**: se a máquina estiver pior que isso, o teste falha — a intenção é não medir a
  máquina, não afirmar que ela é infinita.
- As esperas de 5 s e 10 s do arquivo **gated por Redis** não foram tocadas (sem
  `OLLAMA_AGENT_TEST_REDIS_URL` elas nem executam).
- **Não há prova** de que a falha de E-154 era uma dessas esperas: são 9 execuções limpas, não uma
  reprodução. O defeito permanece **sem causa provada**.
- Rodar no Linux local **não** substitui o CI: o endurecimento reduz a chance de falha por lentidão, não a
  elimina por decreto.
