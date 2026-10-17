"""MEDE A PERGUNTA CERTA (`E-1056`): *quais documentos AFIRMAM produzir um artefacto e nao o citam?*

A pergunta do `E-1055` era a errada: «quais documentos citam zero artefactos» da' 113 de 143 -- e
a maioria e' narrativa. **A pergunta e' sobre AFIRMACAO, nao sobre AUSENCIA.**

E este detector nasce com as **quatro correccoes** que o `E-1055` me obrigou a fazer:

  1. comparar **caminho** E **basename** -- nao so' basename contra uma lista de caminhos;
  2. **nao excluir em silencio**: o que nao se conseguiu ler e' **contado e dito**;
  3. **nao** tratar ficheiros gerados em runtime como artefactos do repositorio (exige-se que o
     documento diga **onde** o ficheiro vive);
  4. **nao** tratar «A CRIAR» como ponteiro quebrado -- *uma declaracao honesta de trabalho
     pendente nao e' um defeito.*

O que se procura: uma frase em que o documento **afirme ser a fonte / gerar / escrever** um
ficheiro nomeado, e esse ficheiro **nao seja citado** no resto do documento.
"""
import io
import os
import re
import sys

RAIZ = sys.argv[1] if len(sys.argv) > 1 else r'C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama'

# A afirmacao tem de ser EXPLICITA -- verbo de producao + nome de ficheiro na mesma frase.
VERBOS = r"(?:e' a fonte|é a fonte|fonte de|gera|gerado|escreve|escrito|produz|produzido|deriva|derivado)"
NOME_ARQ = r"`([A-Za-z0-9_./-]+\.(?:md|csv|json|sha256|txt|sbom))`"

AFIRMACAO = re.compile(VERBOS + r"[^.\n]{0,80}?" + NOME_ARQ, re.I)
# «A CRIAR» / «pendente» / «futuro» na mesma linha -> trabalho declarado, nao defeito.
PENDENTE = re.compile(r"A CRIAR|a criar|pendente|futuro|planead|TODO", re.I)
# ficheiro que o proprio documento diz ser gerado em RUNTIME (fora do repo).
RUNTIME = re.compile(r"grava|em tempo de execucao|runtime|no perfil|HKCU|LOCALAPPDATA", re.I)

docs = []
for sub in ("docs/mission", "docs", "docs/parity"):
    d = os.path.join(RAIZ, sub)
    if not os.path.isdir(d):
        continue
    for nome in sorted(os.listdir(d)):
        if nome.endswith(".md"):
            docs.append((os.path.relpath(os.path.join(d, nome), RAIZ).replace("\\", "/"),
                         os.path.join(d, nome)))

nao_lidos = 0
achados = []
for rel, caminho in docs:
    try:
        t = io.open(caminho, encoding="utf-8").read()
    except OSError:
        nao_lidos += 1                      # correccao 2: contar, nao esconder
        continue

    for m in AFIRMACAO.finditer(t):
        alvo = m.group(1).strip()
        # a linha inteira, para julgar o contexto
        inicio = t.rfind("\n", 0, m.start()) + 1
        fim = t.find("\n", m.end())
        linha = t[inicio: fim if fim > 0 else len(t)]
        if PENDENTE.search(linha):
            continue                        # correccao 4: «A CRIAR» e' honesto
        if RUNTIME.search(linha):
            continue                        # correccao 3: gerado em runtime, fora do repo
        base = os.path.basename(alvo)
        existe_repo = os.path.exists(os.path.join(RAIZ, alvo)) or any(
            os.path.exists(os.path.join(RAIZ, sub, base))
            for sub in ("", "docs", "docs/mission", "docs/parity", "scripts")
        )
        # o documento cita-o FORA da propria frase de afirmacao?
        ocorrencias = t.count(base)
        if existe_repo and ocorrencias <= 1:
            achados.append((rel, alvo, "AFIRMA gerar e so' o nomeia uma vez", linha.strip()[:96]))
        elif not existe_repo:
            achados.append((rel, alvo, "AFIRMA gerar e o ficheiro NAO existe em disco",
                            linha.strip()[:96]))

print("  A PERGUNTA CERTA: quais documentos AFIRMAM produzir um artefacto e nao o citam?")
print("  documentos lidos: %d de %d%s"
      % (len(docs) - nao_lidos, len(docs),
         ("  (** %d nao lidos **)" % nao_lidos) if nao_lidos else ""))
print("")
if not achados:
    print("  NENHUMA AFIRMACAO SEM PONTEIRO.")
else:
    print("  ACHADOS: %d -- *cada um tem de ser verificado antes de virar declaracao*" % len(achados))
    for rel, alvo, motivo, linha in achados:
        print("")
        print("    %s" % rel)
        print("      alvo: %s" % alvo)
        print("      %s" % motivo)
        print("      linha: %s" % linha)
