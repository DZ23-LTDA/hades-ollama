"""Fault injection do `hades_traceability_check.py` -- `E-1110`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas tem de **falhar** quando a rastreabilidade mente.*

=> *injeta 3 defeitos numa **copia**:*

  1. **remover um dos 12 IDs do estado** (`evidence_ids_nao_registrados`) => (2) falha (o ID citado deixa de
     estar declarado);
  2. **remover a linha de um ID** do `HADES_EVIDENCE_LOST_HISTORICAL.md` => (3b) falha;
  3. **meter um ID perdido no `REQUISITOS.csv`** (um requisito ativo a depender de evidencia perdida) => (5b)
     falha.

***E A CONTRAPROVA.*** *Sem o injectado, o checador continua a passar.*
"""
import io
import json
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
MISSAO = RAIZ / "docs" / "mission"
AUDIT = RAIZ / "docs" / "audit"
CHECKER = RAIZ / "scripts" / "hades_traceability_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    for src, rel in ((MISSAO, "docs/mission"), (AUDIT, "docs/audit")):
        d = tmp / rel
        d.mkdir(parents=True)
        for p in src.glob("*"):
            if p.is_file():
                shutil.copyfile(p, d / p.name)
    # *os SBOMs na raiz*
    for p in RAIZ.glob("HADES_SBOM_*.csv"):
        shutil.copyfile(p, tmp / p.name)
    return tmp


def correr(raiz: Path) -> tuple[int, str]:
    alvo = raiz / "scripts" / "_check.py"
    alvo.parent.mkdir(parents=True, exist_ok=True)
    t = io.open(CHECKER, encoding="utf-8").read()
    t = t.replace(
        'RAIZ = Path(__file__).resolve().parent.parent',
        'RAIZ = __import__("pathlib").Path(r"%s")' % raiz,
    )
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(t)
    r = subprocess.run([sys.executable, "-X", "utf8", str(alvo)],
                       capture_output=True, text=True, encoding="utf-8", errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    if not CHECKER.is_file():
        print("  ERRO: o checador nao existe")
        return 2

    raiz = copia()
    est = raiz / "docs" / "mission" / "HADES_MASTER_STATE.json"
    perd = raiz / "docs" / "mission" / "HADES_EVIDENCE_LOST_HISTORICAL.md"
    req = raiz / "docs" / "audit" / "REQUISITOS.csv"

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0,
      (saida.strip().splitlines()[-6] if saida.strip() else "")[:70])

    orig_est = io.open(est, encoding="utf-8").read()
    orig_perd = io.open(perd, encoding="utf-8").read()
    orig_req = io.open(req, encoding="utf-8").read()

    # ---- (1) remover um ID do estado ----
    d = json.loads(orig_est)
    alvo = "E-618"
    antes = len(d.get("evidence_ids_nao_registrados") or [])
    d["evidence_ids_nao_registrados"] = [x for x in (d.get("evidence_ids_nao_registrados") or []) if x != alvo]
    assert len(d["evidence_ids_nao_registrados"]) == antes - 1, "a remocao nao mudou nada"
    io.open(est, "w", encoding="utf-8", newline="").write(json.dumps(d, indent=4, ensure_ascii=False))
    rc, saida = correr(raiz)
    T("(1) sem o `%s` no estado -> FALHA" % alvo, rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(est, "w", encoding="utf-8", newline="").write(orig_est)

    # ---- (2) remover a linha de um ID do documento ----
    sem = re.sub(r"^\|\s*`E-672`\s*\|.*?$", "", orig_perd, flags=re.M)
    assert sem != orig_perd, "a remocao da linha nao mudou nada"
    io.open(perd, "w", encoding="utf-8", newline="\n").write(sem)
    rc, saida = correr(raiz)
    T("(2) sem a linha do `E-672` no documento -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(perd, "w", encoding="utf-8", newline="\n").write(orig_perd)

    # ---- (3) um requisito ativo a depender de um ID perdido ----
    # ***O `REQUISITOS.csv` NAO TEM LINHA PARA ACRESCENTAR ASSIM.*** *Medido: a 1.a linha e' o cabecalho e as
    # outras comecam por `REQ-` => *o filtro `not l.startswith("REQ")` nao apanhava nenhuma* => *a cura: meter
    # o `E-672` numa linha `REQ-*` mesmo.*
    linhas = orig_req.split("\n")
    for k, l in enumerate(linhas):
        if l.startswith("REQ-"):
            linhas[k] = l + " E-672"
            break
    novo_req = "\n".join(linhas)
    assert novo_req != orig_req, "a injecao no REQUISITOS nao mudou nada"
    io.open(req, "w", encoding="utf-8", newline="").write(novo_req)
    rc, saida = correr(raiz)
    T("(3) um requisito ativo a citar `E-672` (perdido) -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(req, "w", encoding="utf-8", newline="").write(orig_req)

    # ---- contraprova ----
    raiz2 = copia()
    rc, saida = correr(raiz2)
    T("contraprova: a copia original continua a PASSAR", rc == 0)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  ==== AS 3 INJECOES + A CONTRAPROVA PASSARAM ====")
    return 0


if __name__ == "__main__":
    sys.exit(main())
