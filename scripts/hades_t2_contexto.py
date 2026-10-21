#!/usr/bin/env python3
"""`hades_t2_contexto.py` — o AMBIENTE da medicao T2, medido (`E-1093`).

***O QUE O §ESTAGIO 21 EXIGE, E O QUE FALTAVA.***

O prompt pede, para os benchmarks: *«definir cenarios reproduziveis, **hardware, versao, modelo, contexto,
warm/cold, repeticoes e erro**; separar tempo de carregamento, geracao, ferramentas e wall time; comparar
apenas o que puder ser executado legalmente e com equivalencia de tarefa; sem acesso a concorrente, marcar
NOT_MEASURED; nao usar marketing como benchmark.»*

=> *o `HADES_T2_MATRIX.md` e o `HADES_T2_REPETICAO.md` tem os **cenarios**, os **modelos**, as
**repeticoes** (N=3) e o **veredito** -- e **nao** tem o **hardware**, a **versao**, o **contexto**, o
**warm/cold**, o **erro**, nem a **separacao dos tempos**.* => ***logo os numeros nao sao reproduziveis:
quem os quiser repetir nao sabe em que maquina, com que versao e com que contexto foram medidos.***

***O QUE ESTE GERADOR FAZ.*** *Mede o **ambiente** -- CPU, RAM, GPU, SO, versao do Ollama, modelos
instalados -- e escreve `docs/parity/HADES_T2_AMBIENTE.md`. *Nao inventa: se o Ollama nao responder, diz
`NAO_MEDIDO`, e nao «desconhecido» como se fosse um valor.*

***E O QUE ELE NAO FAZ.*** *Nao re-mede os 8 cenarios -- isso exige o modelo a correr e tempo (o T2 levou
**599,88 s**). => *o que ele faz e' **fechar o contexto** que os torna reproduziveis.*
"""
import hashlib
import io
import json
import os
import platform
import shutil
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

# Escolhe o `_tmp/goroot*` de MAIOR versao: um literal fixo expira em silencio (ja' aconteceu).
def _versao_do_goroot(raiz):
    candidatos = []
    base = raiz / "_tmp"
    if base.is_dir():
        for d in base.iterdir():
            if d.is_dir() and d.name.startswith("goroot") and d.name[6:].isdigit():
                if (d / "go" / "bin" / "go.exe").exists() or (d / "go" / "bin" / "go").exists():
                    candidatos.append((int(d.name[6:]), d.name))
    if candidatos:
        return sorted(candidatos, reverse=True)[0][1]
    return "goroot1269"

RAIZ = Path(__file__).resolve().parent.parent
SAIDA = RAIZ / "docs" / "parity" / "HADES_T2_AMBIENTE.md"
OLLAMA = "http://127.0.0.1:11434"


def http_json(caminho: str, timeout: float = 5.0):
    try:
        with urllib.request.urlopen(OLLAMA + caminho, timeout=timeout) as r:
            return json.loads(r.read().decode("utf-8"))
    except (urllib.error.URLError, urllib.error.HTTPError, TimeoutError, OSError, ValueError):
        return None


def cpu() -> str:
    if platform.system() == "Windows":
        try:
            r = subprocess.run(["powershell.exe", "-NoProfile", "-Command",
                                "(Get-CimInstance Win32_Processor).Name"],
                               capture_output=True, text=True, encoding="utf-8", errors="replace",
                               timeout=20)
            nome = (r.stdout or "").strip()
            if nome:
                return nome
        except Exception:
            pass
    return platform.processor() or "NAO_MEDIDO"


def ram_gb() -> str:
    if platform.system() == "Windows":
        try:
            r = subprocess.run(["powershell.exe", "-NoProfile", "-Command",
                                "[math]::Round((Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory/1GB,1)"],
                               capture_output=True, text=True, encoding="utf-8", errors="replace",
                               timeout=20)
            v = (r.stdout or "").strip()
            if v:
                return v + " GB"
        except Exception:
            pass
    return "NAO_MEDIDO"


def gpu() -> str:
    if platform.system() == "Windows":
        try:
            r = subprocess.run(["powershell.exe", "-NoProfile", "-Command",
                                "(Get-CimInstance Win32_VideoController | Select-Object -First 1).Name"],
                               capture_output=True, text=True, encoding="utf-8", errors="replace",
                               timeout=20)
            v = (r.stdout or "").strip()
            if v:
                return v
        except Exception:
            pass
    return "NAO_MEDIDO"


def disco_livre_gb() -> str:
    try:
        u = shutil.disk_usage(str(RAIZ))
        return "%.1f GB" % (u.free / (1024 ** 3))
    except Exception:
        return "NAO_MEDIDO"


def golang() -> str:
    go = RAIZ / "_tmp" / _versao_do_goroot(RAIZ) / "go" / "bin" / "go.exe"
    if not go.is_file():
        return "AUSENTE"
    try:
        r = subprocess.run([str(go), "version"], capture_output=True, text=True,
                           encoding="utf-8", errors="replace", timeout=20)
        return (r.stdout or "").strip()
    except Exception:
        return "NAO_MEDIDO"


def commit() -> str:
    try:
        r = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=str(RAIZ),
                           capture_output=True, text=True, encoding="utf-8", timeout=20)
        return (r.stdout or "").strip() or "?"
    except Exception:
        return "?"


def estado_worktree() -> str:
    r = subprocess.run(["git", "status", "--porcelain"], cwd=str(RAIZ),
                       capture_output=True, text=True, encoding="utf-8")
    linhas = [l for l in (r.stdout or "").split("\n") if l.strip()]
    return "limpo" if not linhas else "dirty(%d)" % len(linhas)


def sha_ficheiro(rel: str) -> str:
    p = RAIZ / rel
    if not p.is_file():
        return "AUSENTE"
    return hashlib.sha256(p.read_bytes()).hexdigest()[:16]


def gerar() -> str:
    versao = http_json("/api/version")
    tags = http_json("/api/tags")

    linhas = []
    linhas.append("# HADES — T2: O AMBIENTE DA MEDIÇÃO")
    linhas.append("")
    linhas.append("**Este ficheiro é gerado.** Não edite à mão: corrija a fonte e regenere com")
    linhas.append("`python -X utf8 scripts/hades_t2_contexto.py`.")
    linhas.append("")
    linhas.append("## Porque este ficheiro existe")
    linhas.append("")
    linhas.append("O `HADES_T2_MATRIX.md` e o `HADES_T2_REPETICAO.md` têm os **cenários**, os **modelos**, as")
    linhas.append("**repetições** (N=3) e o **veredito**. O que **não** tinham era o **ambiente**: sem ele, os")
    linhas.append("números **não são reproduzíveis** — quem os quiser repetir não sabe em que máquina, com que")
    linhas.append("versão e com que contexto foram medidos.")
    linhas.append("")
    linhas.append("⇒ ***um número sem ambiente é uma curiosidade; um número com ambiente é uma medição.***")
    linhas.append("")
    linhas.append("## Hardware e sistema")
    linhas.append("")
    linhas.append("| Item | Valor medido |")
    linhas.append("| --- | --- |")
    linhas.append("| CPU | `%s` |" % cpu())
    linhas.append("| RAM | `%s` |" % ram_gb())
    linhas.append("| GPU | `%s` |" % gpu())
    linhas.append("| SO | `%s %s` (build `%s`, arch `%s`) |" % (
        platform.system(), platform.release(), platform.version(), platform.machine()))
    linhas.append("| Python | `%s` |" % platform.python_version())
    linhas.append("| Go | `%s` |" % golang())
    linhas.append("| Disco livre | `%s` |" % disco_livre_gb())
    linhas.append("| Cores lógicos | `%s` |" % (os.cpu_count() or "NAO_MEDIDO"))
    linhas.append("")
    linhas.append("## Ollama")
    linhas.append("")
    if versao:
        linhas.append("| Item | Valor medido |")
        linhas.append("| --- | --- |")
        linhas.append("| Versão | `%s` |" % versao.get("version", "?"))
        linhas.append("| Endpoint | `%s` |" % OLLAMA)
    else:
        linhas.append("**`NAO_MEDIDO`** — o Ollama não respondeu em `%s`. *Ausência não é aprovação: o T2")
        linhas.append("foi medido noutra sessão, e este ficheiro declara que **agora** não foi possível confirmar")
        linhas.append("o ambiente.*" % OLLAMA)
    linhas.append("")
    linhas.append("## Modelos instalados (agora)")
    linhas.append("")
    if tags and tags.get("models"):
        linhas.append("| Modelo | Tamanho | Digest (curto) |")
        linhas.append("| --- | --- | --- |")
        for m in tags["models"]:
            nome = m.get("name", "?")
            tam = m.get("size", 0)
            dig = (m.get("digest") or "")[:12]
            linhas.append("| `%s` | %.2f GB | `%s` |" % (nome, tam / (1024 ** 3), dig))
    else:
        linhas.append("**`NAO_MEDIDO`** — `/api/tags` não respondeu.")
    linhas.append("")
    linhas.append("## Os dois modelos do T2")
    linhas.append("")
    presentes = set()
    if tags and tags.get("models"):
        presentes = {m.get("name", "") for m in tags["models"]}
    for alvo in ("qwen2.5:7b", "llama3.2:3b"):
        linhas.append("- `%s`: **%s**" % (alvo, "instalado" if alvo in presentes else "AUSENTE agora"))
    linhas.append("")
    linhas.append("*`AUSENTE agora` não invalida a medição antiga — diz que **este ambiente** já não o tem.*")
    linhas.append("")
    linhas.append("## O que este ambiente **não** cobre")
    linhas.append("")
    linhas.append("- **warm/cold**: o T2 e o T2-REPETIÇÃO **não separaram** o tempo de carregamento do modelo do")
    linhas.append("  tempo de geração. ⇒ a coluna «duração» é **wall time**, não a soma decomposta.")
    linhas.append("- **separação de tempos**: carregamento, geração, ferramentas e wall time **não** foram medidos")
    linhas.append("  em separado. ⇒ fica como lacuna declarada, não como número inventado.")
    linhas.append("- **erro**: o T2 regista veredito por cenário, e **não** um intervalo de confiança — com N=3,")
    linhas.append("  os números são **contagem**, não probabilidade (o próprio `HADES_T2_REPETICAO.md` o diz).")
    linhas.append("- **comparação externa**: Claude Code / Codex / Manus ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` ⇒")
    linhas.append("  **`NAO_MEDIDO`**. *Sem acesso legal ao concorrente, não há comparação — e inventá-la seria")
    linhas.append("  marketing, não benchmark.*")
    linhas.append("")
    linhas.append("## Proveniência")
    linhas.append("")
    linhas.append("| Item | Valor |")
    linhas.append("| --- | --- |")
    linhas.append("| Commit | `%s` |" % commit())
    linhas.append("| Estado do worktree | `%s` |" % estado_worktree())
    linhas.append("| `HADES_T2_MATRIX.md` | sha256 `%s` |" % sha_ficheiro("docs/parity/HADES_T2_MATRIX.md"))
    linhas.append("| `HADES_T2_REPETICAO.md` | sha256 `%s` |" % sha_ficheiro("docs/parity/HADES_T2_REPETICAO.md"))
    linhas.append("")
    linhas.append("⇒ ***o hash dos dois documentos liga ESTE ambiente aos números que ele contextualiza.***")
    linhas.append("")
    return "\n".join(linhas)


def main() -> int:
    check = "--check" in sys.argv
    conteudo = gerar()
    if check:
        if not SAIDA.is_file():
            print("  FALHA: %s nao existe (corra sem --check)" % SAIDA.relative_to(RAIZ))
            return 1
        actual = io.open(SAIDA, encoding="utf-8").read()
        # *O ambiente muda (modelos instalados, disco, commit) -- o `--check` compara as seccoes
        # ESTAVEIS: o texto explicativo e a presenca dos campos, nao os valores.*
        for campo in ("CPU", "RAM", "GPU", "SO", "Python", "Go", "Cores lógicos",
                      "Ollama", "Modelos instalados", "Os dois modelos do T2",
                      "warm/cold", "separação de tempos", "erro", "comparação externa",
                      "Proveniência"):
            if campo.lower() not in actual.lower():
                print("  FALHA: o ambiente nao cobre `%s`" % campo)
                return 1
        print("  AMBIENTE T2 CONSISTENTE: hardware, SO, Ollama, modelos, lacunas e proveniencia")
        return 0
    SAIDA.parent.mkdir(parents=True, exist_ok=True)
    io.open(SAIDA, "w", encoding="utf-8", newline="\n").write(conteudo)
    print("  gerado: %s" % SAIDA.relative_to(RAIZ))
    return 0


if __name__ == "__main__":
    sys.exit(main())
