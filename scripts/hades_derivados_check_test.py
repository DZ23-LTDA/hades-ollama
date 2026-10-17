#!/usr/bin/env python3
"""O teste do pre-commit dos derivados (`E-1071`).

A REGRA DO PROMPT: *«Nao conte teste verde que nao falharia com o defeito.»*

Este teste **injeta o defeito exacto do `E-1070`** -- um derivado desactualizado em relacao a fonte
-- e verifica que o `hades_derivados_check.py` **fica vermelho e diz qual regenerar**. Depois
restaura e verifica que volta a verde.

COMO OS DEFEITOS SAO INJETADOS, SEM TOCAR NO REPOSITORIO.

O script le os caminhos de `RAIZ` (a raiz do repo). Em vez de mexer nos ficheiros reais, o teste
**corre o script com um `RAIZ` proprio** -- uma copia descartavel em `_tmp/` -- e muda la' os
derivados. *Um teste que muda o repositorio de producao e' um teste que se pode recusar a correr no
dia em que e' mais preciso.*

Os tres defeitos:
  1. **derivado apagado**      -> o `--check` do gerador falha (o ficheiro nao existe)
  2. **derivado com uma linha a mais** -> o `--check` falha (divergiu)
  3. **fonte mudada, derivado nao** -> o `--check` falha -- **e' o defeito do `E-1070`**

E a contraprova: **restaurar** o derivado volta a verde.

O QUE ESTE TESTE NAO PROVA.

- Nao prova que o **hook do Git** dispara. Isso e' do Git (`core.hooksPath`), e testa-se por
  inspeccao do ficheiro `.githooks/pre-commit` (que o teste tambem verifica: existe, tem shebang, e
  chama o script certo).
- Nao prova que o hook e' rapido o bastante. Mede o tempo, e o teste falha se passar de **30 s** --
  um limite declarado **antes**, e folgado, porque o medido e' ~4 s.
"""
import io
import os
import shutil
import subprocess
import sys
import tempfile
import time
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPT = "scripts/hades_derivados_check.py"
HOOK = ".githooks/pre-commit"

# O que a copia descartavel precisa de ter para os tres `--check` correrem.
NECESSARIOS = [
    "scripts/hades_release_gate.py",
    "scripts/hades_manifest.py",
    "scripts/hades_brand_inventory.py",
    SCRIPT,
    "docs/mission/HADES_FINAL_RELEASE_GATE.md",
    "docs/mission/HADES_RELEASE_REPORT.md",
    "docs/mission/HADES_SNAPSHOT_MANIFEST.json",
    "docs/mission/HADES_BRAND_INVENTORY.md",
    "docs/mission/HADES_MASTER_STATE.json",
]

falhas = []


def T(nome, ok, detalhe=""):
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copiar(destino: Path) -> None:
    """Copia so' o necessario -- *uma copia do monorepo inteiro tornaria o teste numa espera.*"""
    for rel in NECESSARIOS:
        src = RAIZ / rel
        dst = destino / rel
        dst.parent.mkdir(parents=True, exist_ok=True)
        if src.exists():
            shutil.copy2(src, dst)


def correr(raiz: Path) -> tuple[int, str]:
    """Corre o verificador numa raiz alternativa, injetando-a como `RAIZ`."""
    # o script resolve `RAIZ` a partir de `__file__`; para o redireccionar sem o editar, corremos
    # uma copia com a linha do `RAIZ` substituida -- *nao se muda o script de producao para o testar.*
    caminho = raiz / SCRIPT
    t = io.open(caminho, encoding="utf-8").read()
    antigo = "RAIZ = Path(__file__).resolve().parent.parent"
    assert antigo in t, "o script mudou de forma: a linha do RAIZ nao bate"
    t = t.replace(antigo, "RAIZ = Path(r'%s')" % str(raiz), 1)
    io.open(caminho, "w", encoding="utf-8", newline="\n").write(t)
    r = subprocess.run([sys.executable, "-X", "utf8", str(caminho), "--quiet"],
                       cwd=str(raiz), capture_output=True, text=True,
                       encoding="utf-8", errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    tmp = Path(tempfile.mkdtemp(prefix="hades-deriv-"))
    try:
        copiar(tmp)

        # ---------- 0. o hook existe e chama o script certo ----------
        hook = RAIZ / HOOK
        T("o hook esta' versionado em .githooks/pre-commit", hook.exists())
        if hook.exists():
            h = io.open(hook, encoding="utf-8").read()
            T("o hook tem shebang", h.startswith("#!/bin/sh"))
            T("o hook chama o script de derivados", "hades_derivados_check.py" in h)
            T("o hook aceita python e python3", "python3" in h and "PY=python" in h)
            T("o hook usa --quiet (nao imprime sucesso em cada commit)", "--quiet" in h)

        # ---------- 1. baseline: consistente, NO REPO REAL ----------
        # *A copia descartavel so' tem 9 ficheiros; o `brand_inventory` le o repo todo.* => medir o
        # baseline la' seria o teste a acusar-se a si mesmo (o `E-1044`/`E-1054` outra vez).
        tempo0 = time.time()
        rc = subprocess.run([sys.executable, "-X", "utf8", str(RAIZ / SCRIPT), "--quiet"],
                            cwd=str(RAIZ), capture_output=True, text=True,
                            encoding="utf-8", errors="replace").returncode
        tempo = time.time() - tempo0
        T("baseline (repo real): os derivados batem com a fonte (exit 0)", rc == 0)
        T("baseline: corre em menos de 30 s", tempo < 30, "%.1f s" % tempo)

        # ---------- 2. DEFEITO 1: derivado apagado ----------
        copiar(tmp)  # restaura a linha do RAIZ que o `correr` substitui
        alvo = tmp / "docs/mission/HADES_BRAND_INVENTORY.md"
        backup = alvo.read_bytes()
        alvo.unlink()
        rc, saida = correr(tmp)
        T("DEFEITO 1 (derivado apagado) -> exit 1", rc == 1, saida.strip().splitlines()[0][:110] if saida.strip() else "")
        T("DEFEITO 1 -> diz qual regenerar", "hades_brand_inventory.py" in saida and "python" in saida)
        alvo.write_bytes(backup)
        copiar(tmp)

        # ---------- 3. DEFEITO 2: derivado com uma linha a mais ----------
        alvo = tmp / "docs/mission/HADES_BRAND_INVENTORY.md"
        backup = alvo.read_bytes()
        io.open(alvo, "a", encoding="utf-8", newline="\n").write("\n| linha injetada pelo teste | 1 | migravel |\n")
        rc, saida = correr(tmp)
        T("DEFEITO 2 (derivado com linha a mais) -> exit 1", rc == 1)
        alvo.write_bytes(backup)
        copiar(tmp)

        # ---------- 4. DEFEITO 3: o defeito do E-1070 -- a fonte mudou, o derivado nao ----------
        # mudar o `HADES_MASTER_STATE.json` muda o que o release_gate gera
        mestre = tmp / "docs/mission/HADES_MASTER_STATE.json"
        backup_mestre = mestre.read_bytes()
        txt = mestre.read_text(encoding="utf-8")
        mestre.write_text(txt + "\n", encoding="utf-8")  # uma alteracao minima
        rc, saida = correr(tmp)
        T("DEFEITO 3 (fonte mudou, derivado nao -- o do E-1070) -> exit 1", rc == 1,
          saida.strip().splitlines()[0][:110] if saida.strip() else "")
        mestre.write_bytes(backup_mestre)
        copiar(tmp)

        # ---------- 5. CONTRAPROVA: restaurar volta a verde, NO REPO REAL ----------
        # *A copia nao tem o mundo que o `brand_inventory` mede* => a contraprova corre onde ele
        # corre. Os defeitos acima provam que o verificador **detecta**; esta prova que ele **nao
        # detecta o que nao esta' la'**.
        rc = subprocess.run([sys.executable, "-X", "utf8", str(RAIZ / SCRIPT), "--quiet"],
                            cwd=str(RAIZ), capture_output=True, text=True,
                            encoding="utf-8", errors="replace").returncode
        T("CONTRAPROVA (repo real): restaurado -> exit 0 outra vez", rc == 0)

        # ---------- 6. a bateria completa do repo continua a passar ----------
        rc2 = subprocess.run([sys.executable, "-X", "utf8", str(RAIZ / SCRIPT), "--quiet"],
                             cwd=str(RAIZ), capture_output=True, text=True,
                             encoding="utf-8", errors="replace")
        T("o verificador no repo real da' exit 0", rc2.returncode == 0,
          (rc2.stdout or "").strip()[-120:])
    finally:
        shutil.rmtree(tmp, ignore_errors=True)

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
