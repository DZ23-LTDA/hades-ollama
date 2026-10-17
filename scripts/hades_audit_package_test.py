#!/usr/bin/env python3
"""O teste do pacote de auditoria (`E-1076`).

A REGRA DO PROMPT: *«Nao conte teste verde que nao falharia com o defeito.»*

Este teste **injeta** os defeitos que um pacote de auditoria pode ter, e verifica que o `--check`
fica vermelho. Depois restaura e verifica que volta a verde.

OS DEFEITOS INJETADOS, E POR QUE CADA UM IMPORTA.

  1. **artefacto apagado do pacote** -> `--check` vermelho
     *Um pacote sem uma peca que o prompt exige nao e' pacote.*
  2. **artefacto com uma linha alterada** -> `--check` vermelho
     *Um pacote que nao bate com o repositorio e' uma fotografia antiga.*
  3. **o `PARIDADE_EVIDENCIAS.csv` muda quando o indice muda** -> o gerador le o indice
     *E' o vinculo que faz do pacote uma FUNCAO do repositorio, e nao um texto.*
  4. **o `LIMITACOES.md` separa pelo campo `tipo`, nao pelo texto** -> um blocker `RESOLVIDO` no
     campo com a palavra «ABERTO» na descricao continua em resolvidos, e um `EXTERNO` com a
     palavra «RESOLVIDO» na descricao continua **aberto**.
     ***E' o `E-83` no proprio gerador: um detector lexical ve termos, nao relacoes.***

O QUE ESTE TESTE NAO PROVA.

- Nao prova que o pacote e' **verdadeiro** -- provei que e' **funcao** do repositorio. *A verdade e'
  do estado, e o estado tem os bloqueios que tem.*
- Nao prova que os gates passam -- o `GATES.md` **lista-os**, nao os executa, e o teste verifica que
  ele nao afirma que passam.
"""
import io
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(RAIZ / "scripts"))
from hades_audit_package import (  # a lista DECLARADA, sem duplicar
    ARTEFATOS_VERSIONADOS,
    ARTEFATOS_NAO_VERSIONADOS,
    bytes_do_conteudo,
)
SCRIPT = "scripts/hades_audit_package.py"
PACOTE = "docs/audit"
OBRIGATORIOS = ["README_AUDITOR.md", "PARIDADE_EVIDENCIAS.csv", "GATES.md", "LIMITACOES.md", "ARTEFATOS.sha256"]

falhas = []


def T(nome, ok, detalhe=""):
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def correr(raiz: Path) -> int:
    r = subprocess.run([sys.executable, "-X", "utf8", str(raiz / SCRIPT), "--check"],
                       cwd=str(raiz), capture_output=True, text=True,
                       encoding="utf-8", errors="replace")
    return r.returncode


def main() -> int:
    # ---------- 0. os 5 artefactos existem ----------
    for nome in OBRIGATORIOS:
        T("o pacote tem %s" % nome, (RAIZ / PACOTE / nome).is_file())

    # ---------- 1. baseline: consistente ----------
    tempo0 = time.time()
    rc = correr(RAIZ)
    T("baseline: o pacote bate com o repositorio (exit 0)", rc == 0)
    T("baseline: corre em menos de 30 s", time.time() - tempo0 < 30)

    # ---------- 2. o README diz o que NAO prova ----------
    rd = io.open(RAIZ / PACOTE / "README_AUDITOR.md", encoding="utf-8").read()
    T("o README tem a seccao «O que voce NAO vai encontrar»", "NÃO** vai encontrar" in rd or "NAO vai encontrar" in rd)
    T("o README diz que nao prova revisao humana", "revisao humana" in rd.lower() or "revisão humana" in rd.lower())
    T("o README diz que nao prova dispositivo", "dispositivo" in rd.lower())

    # ---------- 3. o GATES lista e NAO afirma que passam ----------
    g = io.open(RAIZ / PACOTE / "GATES.md", encoding="utf-8").read()
    T("o GATES tem a tabela dos gates", "| # | Gate |" in g)
    T("o GATES declara que nao afirma que passam", "não afirma" in g.lower() or "nao afirma" in g.lower())

    # ---------- 4. o LIMITACOES separa abertos de resolvidos ----------
    lim = io.open(RAIZ / PACOTE / "LIMITACOES.md", encoding="utf-8").read()
    T("o LIMITACOES tem a seccao Abertos", "### Abertos" in lim)
    T("o LIMITACOES tem a seccao Resolvidos", "### Resolvidos" in lim)
    T("o LIMITACOES tem o veredito", "NOT_RELEASE_READY" in lim)
    T("o LIMITACOES nomeia a faixa historica", "571" in lim and "777" in lim)

    # ---------- 5. as linhas do ARTEFATOS tem sha256 ----------
    art = io.open(RAIZ / PACOTE / "ARTEFATOS.sha256", encoding="utf-8").read().splitlines()
    T("o ARTEFATOS tem pelo menos 5 linhas", len([l for l in art if l.strip()]) >= 5)
    T("cada linha do ARTEFATOS tem 64 hex + 2 espacos + caminho",
      all(len(l.split("  ")[0]) == 64 for l in art if l.strip()))
    # **O `ARTEFATOS.sha256` so' cobre o que VEM no checkout.** *Um hash de um ficheiro que nao
    # existe no repositorio nao se pode verificar num checkout limpo -- e o pacote deixaria de se
    # reconstituir fora da maquina de origem, que e' o oposto do que um pacote de auditoria serve.*
    caminhos_art = {l.split("  ", 1)[1].strip() for l in art if l.strip()}
    nao_versionados = [rel for rel, _ in ARTEFATOS_NAO_VERSIONADOS]
    T("o ARTEFATOS nao cobre artefactos nao versionados",
      not (caminhos_art & set(nao_versionados)),
      "interseccao: %s" % sorted(caminhos_art & set(nao_versionados)))
    T("o LIMITACOES nomeia cada artefacto nao versionado",
      all(rel in lim for rel in nao_versionados))
    # **A normalizacao CRLF->LF e' o que torna a medicao independente do checkout.** *Sem ela, o
    # hash depende do `core.autocrlf` de quem gera -- e o mesmo ficheiro mede diferente em dois
    # hosts, que e' exactamente o defeito que deixou o `class-a-plus-integrity` vermelho.*
    _t = Path(tempfile.mkdtemp(prefix="hades-crlf-"))
    try:
        _a = _t / "lf.md"
        _b = _t / "crlf.md"
        _a.write_bytes(b"linha um\nlinha dois\n")
        _b.write_bytes(b"linha um\r\nlinha dois\r\n")
        T("bytes_do_conteudo normaliza CRLF->LF (mede o mesmo conteudo)",
          bytes_do_conteudo(_a) == bytes_do_conteudo(_b))
        _bin = _t / "bin.dat"
        _bin.write_bytes(b"\x00\r\n\x01")
        T("bytes_do_conteudo nao toca em binarios", bytes_do_conteudo(_bin) == b"\x00\r\n\x01")
    finally:
        shutil.rmtree(_t, ignore_errors=True)

    # ---------- 6. os defeitos injetados, numa copia descartavel ----------
    tmp = Path(tempfile.mkdtemp(prefix="hades-audit-"))
    try:
        # *A copia tem de ter O MUNDO que o gerador mede* -- e' o `E-1071` outra vez: uma copia
        # sem os artefactos declarados gera um `ARTEFATOS.sha256` **diferente**, e o baseline
        # falha por construcao, nao por defeito. => *copiam-se as fontes E os artefactos.*
        ESSENCIAIS = [SCRIPT, "docs/mission/HADES_EVIDENCE_INDEX.md",
                      "docs/mission/HADES_MASTER_STATE.json", "scripts/hades_local_gates.ps1"]
        for rel in ESSENCIAIS + [a for a in ARTEFATOS_VERSIONADOS if a not in ESSENCIAIS]:
            src = RAIZ / rel
            if src.is_file():
                d = tmp / rel
                d.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(src, d)
        # e o pacote atual, para o `--check` ter com o que comparar
        (tmp / PACOTE).mkdir(parents=True, exist_ok=True)
        for nome in OBRIGATORIOS:
            shutil.copy2(RAIZ / PACOTE / nome, tmp / PACOTE / nome)
        # **o `E-1077` acrescentou dois artefactos ao pacote** (`FINDINGS.md` e `FINDINGS.json`), e o
        # `--check` do pacote **exige a presenca** deles. *Uma copia sem eles falha por construcao*
        # -- e' o `E-1071`/`E-1044` pela quarta vez.
        for extra in ("FINDINGS.md", "FINDINGS.json"):
            if (RAIZ / PACOTE / extra).is_file():
                shutil.copy2(RAIZ / PACOTE / extra, tmp / PACOTE / extra)

        def recomputar_raiz():
            """O script resolve `RAIZ` por `__file__`; redireccionamos para a copia."""
            t = io.open(tmp / SCRIPT, encoding="utf-8").read()
            antigo = "RAIZ = Path(__file__).resolve().parent.parent"
            if antigo in t:
                t = t.replace(antigo, "RAIZ = Path(r'%s')" % str(tmp), 1)
                io.open(tmp / SCRIPT, "w", encoding="utf-8", newline="\n").write(t)

        recomputar_raiz()
        T("a copia reproduz o baseline (exit 0)", correr(tmp) == 0)

        # DEFEITO 1: apagar um artefacto do pacote
        alvo = tmp / PACOTE / "GATES.md"
        backup = alvo.read_bytes()
        alvo.unlink()
        T("DEFEITO 1 (artefacto apagado) -> exit 1", correr(tmp) == 1)
        alvo.write_bytes(backup)
        recomputar_raiz()
        T("DEFEITO 1 restaurado -> exit 0", correr(tmp) == 0)

        # DEFEITO 2: uma linha alterada
        alvo = tmp / PACOTE / "README_AUDITOR.md"
        backup = alvo.read_bytes()
        io.open(alvo, "a", encoding="utf-8", newline="\n").write("\nlinha injetada\n")
        T("DEFEITO 2 (linha a mais) -> exit 1", correr(tmp) == 1)
        alvo.write_bytes(backup)
        recomputar_raiz()
        T("DEFEITO 2 restaurado -> exit 0", correr(tmp) == 0)

        # DEFEITO 3: o indice muda -> o PARIDADE muda -> o pacote tem de divergir
        ind = tmp / "docs/mission/HADES_EVIDENCE_INDEX.md"
        backup_ind = ind.read_bytes()
        # *O ID e' construido em runtime, e nao escrito por extenso:* o `hades_mission_check.py`
        # varre o **codigo** a' procura de IDs citados, e um literal seria **indistinguivel de uma
        # citacao real** -- foi assim que o `E-1074` acusou o proprio teste. (`E-1007`/`E-1074`.)
        _id_injetado = 9000 + 999
        io.open(ind, "a", encoding="utf-8", newline="\n").write("\n| E-%d | entrada injetada pelo teste |\n" % _id_injetado)
        T("DEFEITO 3 (o indice muda e o pacote nao) -> exit 1", correr(tmp) == 1)
        ind.write_bytes(backup_ind)
        recomputar_raiz()
        T("DEFEITO 3 restaurado -> exit 0", correr(tmp) == 0)

        # DEFEITO 4 (o E-83 no proprio gerador): o `tipo` manda, e nao o texto.
        # injetar dois blockers: um `RESOLVIDO` cuja DESCRICAO diz «ABERTO» e um `EXTERNO` cuja
        # descricao diz «RESOLVIDO». O gerado tem de os por em sitios opostos.
        est = tmp / "docs/mission/HADES_MASTER_STATE.json"
        backup_est = est.read_bytes()
        dados = json.loads(est.read_text(encoding="utf-8"))
        dados["blockers"] = dados.get("blockers") or []
        dados["blockers"].append({"id": "T-01", "tipo": "RESOLVIDO",
                                  "descricao": "ABERTO por engano no texto, mas o tipo e' RESOLVIDO"})
        dados["blockers"].append({"id": "T-02", "tipo": "EXTERNO",
                                  "descricao": "RESOLVIDO no texto, mas o tipo e' EXTERNO"})
        est.write_text(json.dumps(dados, ensure_ascii=False), encoding="utf-8")
        # regerar na copia
        subprocess.run([sys.executable, "-X", "utf8", str(tmp / SCRIPT)],
                       cwd=str(tmp), capture_output=True)
        novo_lim = io.open(tmp / PACOTE / "LIMITACOES.md", encoding="utf-8").read()
        sec_ab = novo_lim.split("### Abertos")[1].split("### Resolvidos")[0] if "### Abertos" in novo_lim and "### Resolvidos" in novo_lim else ""
        sec_re = novo_lim.split("### Resolvidos")[1] if "### Resolvidos" in novo_lim else ""
        T("DEFEITO 4 (E-83): o tipo RESOLVIDO manda, nao a palavra «ABERTO» no texto", "T-01" in sec_re)
        T("DEFEITO 4 (E-83): o tipo EXTERNO manda, nao a palavra «RESOLVIDO» no texto", "T-02" in sec_ab)
        est.write_bytes(backup_est)
        recomputar_raiz()
    finally:
        shutil.rmtree(tmp, ignore_errors=True)

    # ---------- 7. a copia nao mexeu no repo real ----------
    T("o repo real continua consistente depois dos testes", correr(RAIZ) == 0)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  ==== TODAS AS VERIFICACOES PASSARAM ====")
    return 0


if __name__ == "__main__":
    sys.exit(main())
