"""Teste discriminador do `hades_requisitos.py` -- `E-1091`.

***O QUE PROVA.*** O gerador passa no repo real. => *mas um gerador que so' passa nao vale nada: tem de
**falhar** quando uma prova falta.* => *aqui injecta-se **um requisito com um simbolo que nao existe** e
**um com um teste que nao existe**, numa **copia do gerador** (com a lista `REQUISITOS` substituida), e
exige-se que o `--check` os apanhe.*

***E A CONTRAPROVA.*** *A lista **real** continua a passar.*
"""
import io
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
GERADOR = RAIZ / "scripts" / "hades_requisitos.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def correr_no_repo() -> tuple[int, str]:
    r = subprocess.run([sys.executable, "-X", "utf8", str(GERADOR), "--check"],
                       cwd=str(RAIZ), capture_output=True, text=True, encoding="utf-8",
                       errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def com_lista_alterada(novo_req: str) -> tuple[int, str]:
    """Corre uma COPIA do gerador com um requisito extra (que nao tem prova)."""
    tmp = Path(tempfile.mkdtemp())
    g = tmp / "hades_requisitos.py"
    t = io.open(GERADOR, encoding="utf-8").read()
    # substituir a lista REQUISITOS por uma com o requisito extra
    ini = t.index("REQUISITOS = [")
    fim = t.index("\n]\n", ini) + 3
    lista = t[ini:fim]
    novo = lista[:-3] + novo_req + "\n]\n"
    t2 = t[:ini] + novo + t[fim:]
    # e o RAIZ aponta para o repo real (os ficheiros tem de ser os reais)
    t2 = t2.replace(
        'RAIZ = Path(__file__).resolve().parent.parent',
        'RAIZ = __import__("pathlib").Path(r"%s")' % RAIZ,
    )
    io.open(g, "w", encoding="utf-8", newline="\n").write(t2)
    # *O `--check` compara com o CSV real; como o requisito extra nao esta' la', ele falha na
    # COMPARACAO -- que e' o comportamento certo. Para testar a VALIDACAO da prova, corremos SEM
    # `--check` (modo de escrita) e lemos o aviso de `AUSENTE`.*
    r = subprocess.run([sys.executable, "-X", "utf8", str(g)],
                       cwd=str(RAIZ), capture_output=True, text=True, encoding="utf-8",
                       errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    if not GERADOR.is_file():
        print("  ERRO: o gerador nao existe")
        return 2

    # ***O CSV E' DO REPO, E O TESTE NAO PODE DEIXA-LO ALTERADO.*** *O modo de escrita do gerador
    # escreve `docs/audit/REQUISITOS.csv`; como o teste corre copias com requisitos inventados, sem
    # este guarda o CSV real ficava com eles -- e a contraprova falhava (medido).*
    csv_real = RAIZ / "docs" / "audit" / "REQUISITOS.csv"
    backup = csv_real.read_bytes() if csv_real.is_file() else None

    rc, saida = correr_no_repo()
    T("baseline: os 10 requisitos passam", rc == 0,
      (saida.splitlines()[-1] if saida.strip() else "")[:70])

    try:
        # ---- (1) um requisito cujo SIMBOLO nao existe ----
        req = (
            '    (\n'
            '        "REQ-TESTE-01",\n'
            '        "requisito inventado com simbolo que nao existe",\n'
            '        "internal/agent/companion_estado.go",\n'
            '        "SimboloQueNaoExisteEmLadoNenhum",\n'
            '        "internal/agent/companion_estado_test.go",\n'
            '        "go test ./internal/agent -run Nada -count=1",\n'
            '        "windows/amd64",\n'
            '        "MANTER",\n'
            '    ),\n'
        )
        rc, saida = com_lista_alterada(req)
        T("(1) simbolo inexistente -> o gerador FALHA", rc != 0,
          "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
        T("(1b) e diz QUAL e' o simbolo", "SimboloQueNaoExisteEmLadoNenhum" in saida,
          "aviso presente" if "SimboloQueNaoExisteEmLadoNenhum" in saida else "aviso AUSENTE")

        # ---- (2) um requisito cujo TESTE nao existe ----
        req2 = (
            '    (\n'
            '        "REQ-TESTE-02",\n'
            '        "requisito cujo teste nao existe",\n'
            '        "internal/agent/companion_estado.go",\n'
            '        "DerivarEstadoCompanion",\n'
            '        "internal/agent/teste_que_nao_existe_test.go",\n'
            '        "go test ./internal/agent -run Nada -count=1",\n'
            '        "windows/amd64",\n'
            '        "MANTER",\n'
            '    ),\n'
        )
        rc, saida = com_lista_alterada(req2)
        T("(2) teste inexistente -> o gerador FALHA", rc != 0,
          "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    finally:
        # ***RESTAURAR O CSV REAL.*** *Os modos de escrita das copias escreveram-no.*
        if backup is not None:
            csv_real.write_bytes(backup)

    # ---- (3) a contraprova: o gerador real continua a passar ----
    rc, _ = correr_no_repo()
    T("contraprova: o gerador REAL continua a PASSAR", rc == 0)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  ==== AS 2 INJECOES + A CONTRAPROVA PASSARAM ====")
    return 0

if __name__ == "__main__":
    sys.exit(main())
