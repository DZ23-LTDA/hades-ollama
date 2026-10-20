"""E-488 - o 16o checker: os testes Go, COM a receita completa, e um anti-armadilha.

O QUE FALTAVA: a bateria de 15 checkers nao corria `go test`. *Foi por isso que uma rodada inteira passou com dois testes
a 'falhar' sem que ninguem visse que a causa era o ambiente* (E-487).

E O `AGENTS.md` JA' AVISAVA: *'o sintoma e' sempre "o teste nao roda" - NUNCA "o teste falha"'* - com a receita:
`GOPATH`, `GOMODCACHE`, `GOCACHE`, `GOTMPDIR`, `APPDATA`, `LOCALAPPDATA` **todos** sob `_tmp/`.

O ANTI-ARMADILHA, e e' o ponto deste script: **se o `LOCALAPPDATA` nao estiver sob `_tmp`, o script RECUSA correr** e diz
por que. *Um instrumento que corre com a receita incompleta produz um 'FAIL' que nao e' do produto - e a pior coisa que
um gate pode fazer e' reprovar o inocente.*

SAIDAS:
  0  = os testes passaram
  1  = os testes FALHARAM (e' o produto)
  2  = a RECEITA esta' incompleta (nao e' o produto, e o script recusa medir)
  3  = `go` indisponivel -> BLOCKED_BY_EXTERNAL_DEPENDENCY
"""

import os
import shutil
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent

# 🚨 E-488 (CONTRA-PROVADO): a versão anterior verificava o `LOCALAPPDATA` **depois** de o ter sobrescrito ⇒ ***o
# anti-armadilha era CÓDIGO MORTO: nunca podia disparar*** ✓✓ — *foi o contra-prova que o mostrou: com um
# `LOCALAPPDATA` errado por fora, o script **continuou a correr** em vez de recusar.*
# A correcção é guardar o valor **HERDADO** antes de mexer nele — *é esse que diz se o chamador veio com a receita certa.*
HERDADO_LOCALAPPDATA = os.environ.get("LOCALAPPDATA", "")

# A RECEITA, com os nomes exatos que o AGENTS.md manda redireccionar.
REDIRECIONADOS = {
    "GOPATH": "_tmp/gopath",
    "GOMODCACHE": "_tmp/gomodcache",
    "GOCACHE": "_tmp/gocache1269",
    "GOTMPDIR": "_tmp/gotmp",
    "APPDATA": "_tmp/appdata",
    "LOCALAPPDATA": "_tmp/localappdata",
    "TMP": "_tmp/e2e-tmp",
    "TEMP": "_tmp/e2e-tmp",
    "GOFLAGS": "-mod=mod",
    "GOTOOLCHAIN": "local",
    "CGO_ENABLED": "0",
}

ambiente = dict(os.environ)
for chave, relativo in REDIRECIONADOS.items():
    if chave in ("GOFLAGS", "GOTOOLCHAIN", "CGO_ENABLED"):
        ambiente[chave] = relativo
        continue
    destino = RAIZ / relativo
    destino.mkdir(parents=True, exist_ok=True)
    ambiente[chave] = str(destino)

# O GOROOT: o toolchain local vive sob `_tmp`. *Sem ele, o `go` pode nao existir.*
goroot = RAIZ / "_tmp" / "goroot1269" / "go"
go = str(goroot / "bin" / "go.exe") if (goroot / "bin" / "go.exe").exists() else shutil.which("go")

print("  === CHECKER DOS TESTES GO (E-488) ===")
if not go:
    print("  BLOCKED_BY_EXTERNAL_DEPENDENCY: `go` nao encontrado (nem sob _tmp, nem no PATH)")
    print("  *Nao e' falha do produto: e' ausencia do instrumento, e ela e' declarada.*")
    sys.exit(3)

# O ANTI-ARMADILHA, antes de medir: a receita tem de estar COMPLETA.
# ⚠️ Verifica o valor **HERDADO**, não o que o próprio script acabou de escrever — *senão é código morto.*
if HERDADO_LOCALAPPDATA and "_tmp" not in HERDADO_LOCALAPPDATA.replace("\\", "/"):
    print("  RECEITA INCOMPLETA: o `LOCALAPPDATA` HERDADO não está sob `_tmp/` (está em %r)." % HERDADO_LOCALAPPDATA)
    print("  *O AGENTS.md documenta o sintoma: 'mkdir ...\\AppData\\Local\\ollama-agent: Acesso negado' — e o teste NÃO")
    print("   RODA, em vez de falhar. Reprovar o inocente é pior que não medir.*")
    print("  (O script JÁ corrigiu o valor para a execução; esta recusa existe para o CHAMADOR saber que veio errado.)")
    sys.exit(2)

print("  receita aplicada: %d variaveis redireccionadas para _tmp/" % len(REDIRECIONADOS))
print("  LOCALAPPDATA herdado = %r  ->  aplicado = %s" % (HERDADO_LOCALAPPDATA, ambiente["LOCALAPPDATA"]))
print("  go = %s" % go)

ALVOS = ["./internal/agent", "./server", "./internal/proxy", "./internal/multillm"]

# 🚨 E-490 (MEDIDO): a versão anterior corria `go test` nos QUATRO pacotes **numa só invocação** — e o `go test` só
# imprime **quando cada pacote termina** ⇒ ***19 minutos de SILÊNCIO*** ✓✓ — *e o utilizador perguntou «travou?», com
# razão: **um instrumento que fica mudo é indistinguível de um instrumento parado**.*
# A correcção é correr **um pacote por vez** e imprimir **ao fim de cada um** ⇒ *o tempo total é o mesmo, e deixa de
# haver silêncio longo.*
print("  correndo %d pacotes, um a um (progresso ao fim de cada):" % len(ALVOS))
falhados = []
for alvo in ALVOS:
    print("    ... %s" % alvo, flush=True)
    resultado = subprocess.run([go, "test", alvo, "-count=1", "-timeout", "30m"],
                               cwd=str(RAIZ), env=ambiente, capture_output=True, text=True)
    # PROGRESSO: as linhas uteis de cada pacote, enquanto corre.
    for linha in (resultado.stdout or "").strip().split("\n"):
        if linha.startswith("ok ") or linha.startswith("FAIL"):
            print("        " + linha)

    # FALHA: nomear o TESTE, nao apenas o pacote.
    #
    # *O `falhados.append` esteve DENTRO do laco das linhas desde a versao original.* Com uma unica
    # linha de saida o sintoma era invisivel ("1 de 4"); ao imprimir mais diagnostico apareceu
    # **"45 de 1 FALHARAM"** -- o mesmo pacote contado 45 vezes. *Um contador dentro do laco que
    # imprime o contador mede a si proprio, nao ao mundo.*
    if resultado.returncode != 0:
        falhados.append(alvo)
        # E sem o nome do teste nao ha' hipotese a testar, so' re-execucao. O filtro antigo so'
        # mostrava linhas que comecam por `---`, deixando de fora `panic:` e `*** Test killed` --
        # exactamente os casos em que o Go NAO imprime `--- FAIL`.
        saida = (resultado.stdout or "") + "\n" + (resultado.stderr or "")
        vistas = 0
        for linha in saida.split("\n"):
            limpa = linha.strip()
            if not limpa or vistas >= 12:
                continue
            if (limpa.startswith("--- FAIL") or limpa.startswith("panic:")
                    or "test timed out" in limpa
                    or limpa.startswith("*** Test killed")):
                print("        FALHA> " + limpa)
                vistas += 1
        if vistas == 0:
            print("        FALHA> (nenhuma linha reconhecida; cauda do stdout:)")
            for linha in (resultado.stdout or "").strip().split("\n")[-10:]:
                print("        | " + linha)
        for linha in (resultado.stderr or "").strip().split("\n")[-6:]:
            print("        ERR " + linha)
    print("    -> %s: %s" % (alvo, "OK" if resultado.returncode == 0 else "FALHOU"), flush=True)

if not falhados:
    print("  TESTES GO: TODOS PASSAM ✓ (%d pacotes)" % len(ALVOS))
else:
    print("  TESTES GO: %d de %d FALHARAM: %s" % (len(falhados), len(ALVOS), ", ".join(falhados)))
sys.exit(0 if not falhados else 1)
