# HADES — GOAL MASTER (V2.1 CONSOLIDADO)

**mission_id:** `hades-universal-v21`
**Repositório oficial:** https://github.com/DZ23-LTDA/hades-ollama
**Base de integração:** `main` = `889750abe4006985b3bf9037bca2323d1e40cef0`
**Estado de release:** `NOT_RELEASE_READY`
**Documento de origem:** prompt master V2.1 (38 estágios, seções 0–11).

## 1. Objetivo

Transformar o Hades em uma plataforma universal de inteligência artificial,
desenvolvimento de software, automação, criação visual, pesquisa, operação
empresarial e coordenação multiagente — **local-first, segura, modular,
extensível, profissional e comprovadamente utilizável pelo usuário final**.

O objetivo permanente é: **avanço automático, sucesso comprovado, segurança
obrigatória e conclusão nunca inventada.**

## 2. Conceito canônico

Hades = **sistema operacional universal de agentes**. O Ollama é o motor local
padrão, **não** a única fonte de inteligência. Os modelos são substituíveis; o
Hades controla ferramentas, permissões, memória, projeto, missão, orçamento,
interfaces e resultados. Modelos locais ou externos só recebem capacidades
verificadas; **credencial nunca cria privilégio**.

## 3. Regras de engajamento (limites de autonomia)

Decisões reversíveis e de baixo risco: **autonomia total, sem perguntar.**
Exigem autorização humana explícita:

- merge em `main` (governança do repositório);
- publicação externa, envio de mensagens a terceiros, criação de contas;
- chamadas/telefonia, pagamentos, movimentação de dinheiro;
- alterações em produção, operações destrutivas ou irreversíveis;
- custo financeiro relevante.

Proibido em qualquer hipótese: desativar teste/lint/gate para ficar verde;
mascarar falha; simular integração real com mock e declarar produção; exibir
botão/tela/conector que finge funcionar; publicar segredo em Git, log ou
navegador; spam, engajamento artificial, perfil falso ou violação de termos.

## 4. Estados canônicos

- **Capacidade:** `IMPLEMENTADO_E_TESTADO` · `IMPLEMENTADO_NAO_HOMOLOGADO` ·
  `PARCIAL` · `APENAS_ADAPTER` · `APENAS_CATALOGO` · `AUSENTE` ·
  `BLOQUEADO_EXTERNAMENTE`.
- **Execução (estágio):** `NOT_STARTED` · `IN_PROGRESS` · `TESTING` · `FAILED` ·
  `BLOCKED_BY_EXTERNAL_DEPENDENCY` · `COMPLETED_VERIFIED`.
- **Release global:** `NOT_RELEASE_READY` · `RELEASE_CANDIDATE` ·
  `COMPLETED_VERIFIED`.

Capacidade e bloqueio são **eixos separados**: um requisito pode estar
`IMPLEMENTADO_NAO_HOMOLOGADO` e `BLOCKED_BY_EXTERNAL_DEPENDENCY` ao mesmo tempo.

## 5. Convenções de honestidade desta missão

1. `estado: IN_PROGRESS` é o valor **conservador** para "não comprovado como
   concluído". Ele **não** afirma que houve trabalho executado.
2. `COMPLETED_VERIFIED` só com evidência reproduzível no
   `HADES_EVIDENCE_INDEX.md` (comando + ambiente + resultado observado).
3. `auditoria: PENDENTE` significa que a superfície **não foi inspecionada nesta
   rodada**; ausência de evidência **não** é prova de ausência nem de existência.
4. Campo `capability: PENDENTE_DE_AUDITORIA` é uma **extensão declarada** deste
   repositório: não é uma classificação da taxonomia canônica e não promove
   nenhuma capacidade.
5. Documentos de auditoria anteriores (datados) são tratados como **pistas**, não
   como prova: só valem como evidência após reconfirmação do comando.

## 6. Ondas de execução

| Onda | Tema | Estágios |
| --- | --- | --- |
| I | baseline, segurança, arquitectura base | 1–3, 18 |
| II | mission control, modelos, harnesses, MCP, agentes | 4–11, 15, 17 |
| III | ferramentas, builder, workflows, browser | 12–14 |
| IV | canais, empresa, comércio | 16, 23–37 |
| V | UX, distribuição, benchmarks, gate final | 19–22, 38 |

O **Estágio 22 é gate intermediário de engenharia**; o gate **final** só existe
após 1–38. Estágios 16, 28, 29, 33–37 compartilham o Omnichannel Runtime e
**não podem duplicar adaptadores**. Estágios 23–32 reutilizam a infraestrutura
de agentes/modelos/segurança dos anteriores.

## 7. Artefatos obrigatórios de checkpoint

| Artefato | Papel |
| --- | --- |
| `docs/mission/HADES_MASTER_GOAL.md` | este documento — objetivo, limites, convenções |
| `docs/mission/HADES_MASTER_STATE.json` | estado de máquina (canônico, retomável) |
| `docs/mission/HADES_STAGE_REPORT.md` | relatório por estágio (1–38) |
| `docs/mission/HADES_BLOCKERS.md` | bloqueios + procedimento exato de desbloqueio |
| `docs/mission/HADES_EVIDENCE_INDEX.md` | índice de evidências com comando reproduzível |
| `docs/mission/HADES_HANDOFF.md` | como retomar a missão do zero |
| `docs/mission/HADES_PENDING_WORK.md` | backlog priorizado restante |
| `docs/mission/HADES_NEXT_GOAL_READINESS.md` | matriz dos critérios finais de aceite |

Artefatos de auditoria: `audit/HADES_MASTER_AUDIT.md`,
`audit/HADES_CAPABILITY_MATRIX.md`, `audit/HADES_RELEASE_BLOCKERS.md`.

## 8. Ciclo de execução

```
AUDITAR → ENTENDER → PLANEJAR → IMPLEMENTAR → TESTAR
        → INSPECIONAR → CORRIGIR → RETESTAR → DOCUMENTAR → AVANÇAR
```

Ao final de cada estágio: rodar testes, registrar evidência, corrigir falhas,
atualizar checkpoint, avaliar critérios de aceite e avançar para o próximo
estágio elegível. Um bloqueio externo **não** interrompe tarefas independentes.

## 9. Regra final

O avanço é automático; o sucesso é comprovado; a segurança é obrigatória; a
conclusão nunca é inventada.
