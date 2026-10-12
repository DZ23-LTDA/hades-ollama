# Estágio 22 — o gate global passou a ser GERADO e conferido pelo CI (E-179)

## 1. O problema que apareceu ao fechar a E-178

Ao atualizar a matriz do gate para refletir a navegação real do Builder (E-178), ficou visível uma fragilidade:
`HADES_FINAL_RELEASE_GATE.md` e `HADES_RELEASE_REPORT.md` são **exigidos pela seção 11.8**, mas eram escritos **à
mão** e o gerador vivia apenas em `_tmp/` (fora do repositório). Consequência: os documentos **não eram
reproduzíveis** — ninguém conseguia reobter o mesmo arquivo a partir do estado da missão — e nada impedia que a
matriz dissesse `DONE` enquanto o índice de evidências dissesse outra coisa.

Um gate que pode divergir do estado sem ninguém notar é **um gate que mente**.

## 2. O que foi feito

| Peça | O que faz |
| --- | --- |
| `scripts/hades_release_gate.py` | **gera** os dois documentos a partir de `HADES_MASTER_STATE.json` + matriz de aceite declarada |
| `--check` | **não escreve**: compara o que seria gerado com o que está no disco e **falha (exit 1)** se divergir |
| `.github/workflows/class-a-plus-integrity.yaml` | novo passo **`Verify the release gate matches the mission state (no drift)`** ⇒ a divergência passa a **quebrar o CI** |

A regra do projeto já dizia "versionar o instrumento junto com o artefato"; aqui ela foi aplicada ao **relatório
de release**, que é o artefato mais sensível de todos — o que afirma o estado do produto.

O script **não inventa status de estágio**: `execution` continua vindo da regra derivada gravada no estado, e o que
a matriz de aceite exige permanece declarado à mão (exige julgamento) — só que agora ela é **dado versionado**, não
prosa.

## 3. O `--check` funcionando (antes e depois)

| Execução | Resultado |
| --- | --- |
| `--check` **antes** de regenerar | **GATE DIVERGENTE** (exit **1**) — os documentos estavam defasados em relação ao estado |
| `python scripts/hades_release_gate.py` | documentos regenerados |
| `--check` **depois** | **GATE CONSISTENTE** (exit **0**) |

## 4. A mudança de conteúdo (o que a E-178 valeu na matriz)

| Item da matriz | Antes | Depois |
| --- | --- | --- |
| 14 — "O Builder entrega aplicações utilizáveis" | **NAO_COMPROVADO** | **DONE** — com **LIMITE** declarado: app de demonstração, **não** full-stack |
| Resumo | 15 DONE · 5 NAO_COMPROVADO | **16 DONE · 4 NAO_COMPROVADO** |

O veredito global **não** mudou: permanece **`NOT_RELEASE_READY`** — quatro itens seguem não comprovados
(onboarding, plugin de terceiro, artefatos de distribuição, bloqueios sem mitigação aceita) e três estão
bloqueados externamente.

## 5. Um detalhe que eu corrigi no caminho (qualidade de documento)

A primeira versão do gerador escrevia **sem acentos** (hábito defensivo de lidar com encoding em shell). O diff
resultante degradava documentos em pt-BR. Reescrevi o gerador com os acentos corretos e reconferi: o diff contra a
versão anterior caiu para **50 linhas no gate / 24 no relatório**, com as mudanças **intencionais** (cabeçalho de
"gerado por" + item 14 + resumo). Documento em português **não perde acento por conveniência de script**.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `python scripts/hades_release_gate.py --check` (antes) | **exit 1** — divergência detectada |
| `python scripts/hades_release_gate.py --check` (depois) | **exit 0** — consistente |
| `python scripts/classify_ci_surfaces_test.py` | **13 testes OK** (o passo novo no workflow é consistente) |
| `golangci-lint` / contrato / paridade Linux | ver evidência |

## 7. Limites declarados

- A matriz de aceite continua sendo **julgamento humano** (declarado como dado, mas humano): o `--check` garante
  que documento e estado **concordam**, **não** que o julgamento está certo.
- O `--check` só cobre **os dois** documentos que ele gera; outras páginas de missão (índice de evidências,
  relatório de estágios, bloqueios) continuam **manuais** e podem divergir do estado sem alarme.
- `scripts/check-class-a-plus-integrity.sh` **não** foi executado localmente nesta rodada: o host não tem `bash`
  utilizável (o WSL falha ao iniciar). A verificação desse script fica com o CI — e isso fica **dito**, não
  presumido.
- O `--check` no CI cobre o caminho de *pull request* e de *push* nas branches listadas no workflow; não é um
  verificador geral de documentação.
