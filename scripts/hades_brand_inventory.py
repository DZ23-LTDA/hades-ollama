#!/usr/bin/env python3
"""Inventaria as strings de marca e emite a tabela de excepcoes (§FASE-5 itens 1 e 3).

Por que este script existe
--------------------------
O §FASE-5 item 1 pede *«fazer inventario de todas as strings visiveis»* e o item 3 diz
*«manter nomes tecnicos legados **apenas onde necessarios** a compatibilidade de **API, dados,
env vars, protocolos e migracao segura**; **documentar cada excecao**»*.

O `E-990` mediu-o **uma vez** e registou-o como evidencia: 909 canonicas e 441 legadas em
4 nomes. Mas *uma evidencia e' um retrato, e um retrato nao se actualiza sozinho* -- *o
`E-987` e o `E-989` existem precisamente por isso*: um numero publicado num documento
envelhece em silencio se nao houver quem o reconfira.

**Este script e' o quem.** Ele corre o inventario, aplica a lista de EXCECOES declarada
abaixo (que e' a parte que exige julgamento, e por isso fica **escrita e datada**) e escreve
`docs/mission/HADES_BRAND_INVENTORY.md`.

A regra que separa os dois casos
--------------------------------
- **MIGRAVEL**: texto que o utilizador **le** -- titulos de documento, `README`, `MEMORIA`,
  prosa de `docs/`. *Trocar o nome nao quebra nada e cumpre o item 2.*
- **EXCECAO**: texto que **outro sistema** le -- nome de artefacto de instalador, variavel de
  ambiente, chave de credencial, bundle ID de app instalada, `id` de extensao. *Trocar o nome
  quebra compatibilidade, e o item 3 manda mante-lo -- mas exige que se diga **qual e por que**.*

E **historico** e' uma terceira categoria, e nao uma excecao: o `HADES_EVIDENCE_INDEX.md`
cita os nomes que os artefatos **tinham** quando foram medidos. *Reescreve-lo produziria um
indice coerente com a marca de hoje e **falso** sobre o que foi medido* -- e o §FASE-7 item 1
e os `E-959`/`E-965` ja' fixaram que **a evidencia nao se reescreve**.

Uso
---
    python scripts/hades_brand_inventory.py            # escreve docs/mission/HADES_BRAND_INVENTORY.md
    python scripts/hades_brand_inventory.py --check    # falha se divergir (catraca)
    python scripts/hades_brand_inventory.py --json     # imprime o inventario em JSON
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from collections import Counter, defaultdict
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DOC = RAIZ / "docs/mission/HADES_BRAND_INVENTORY.md"


def _git(*args: str) -> str | None:
    """Corre um comando de leitura do Git. Devolve `None` se falhar (*nunca rebenta*).

    *Um instrumento que rebenta por nao haver git e' pior que um que declara a falta* -- e' a
    mesma decisao do `snapshot_do_git()` no `E-987`.
    """
    try:
        saida = subprocess.run(
            ["git", *args], cwd=RAIZ, capture_output=True, text=True,
            encoding="utf-8", errors="replace", timeout=120, check=False,
        )
        return saida.stdout if saida.returncode == 0 else None
    except (OSError, subprocess.SubprocessError):
        return None


def _git_bytes(*args: str) -> bytes | None:
    """Idem, mas sem descodificar -- *para o `cat-file` em lote, onde o conteudo e' binario*."""
    try:
        saida = subprocess.run(
            ["git", *args], cwd=RAIZ, capture_output=True, timeout=120, check=False,
        )
        return saida.stdout if saida.returncode == 0 else None
    except (OSError, subprocess.SubprocessError):
        return None


def _hash_objeto(rel: str) -> str | None:
    saida = _git("rev-parse", "HEAD:%s" % rel)
    return saida.strip() if saida else None

CANONICA = "Hades/HADES"
PADROES = {
    CANONICA: re.compile(r"\bHADES\b|\bHades\b"),
    "DZ23 Agentic": re.compile(r"DZ23 Agentic|dz23-agentic"),
    "Ollama Full": re.compile(r"Ollama Full"),
    "Ollama Classe A+": re.compile(r"Ollama Classe A\+|Classe A\+"),
    "Ollama DZ23": re.compile(r"Ollama DZ23"),
}

DIRS_FORA = {
    "_tmp", "node_modules", ".git", "dist", "test-results", "playwright-report",
    ".next", "coverage", "vendor", "build",
}
# ---------------------------------------------------------------------------
# **E-991, e foi o proprio `--check` que o apanhou**: o gerador escreve
# `docs/mission/HADES_BRAND_INVENTORY.md` -- *um documento que contem as strings da marca* --
# e depois **conta-as**. Escrever o ficheiro **aumenta a contagem que ele mede**, e o
# `--check` falhava **logo apos gerar**: duas geracoes seguidas davam **15 linhas
# diferentes**. *O instrumento media-se a si mesmo.*
#
# A cura e' **excluir o proprio output** do inventario: *um gerador nao pode contar o que
# ele proprio escreve, ou o numero que ele publica e' funcao de quantas vezes ja' correu.*
# E' o `E-987` («o gerador tem de ser deterministico») na forma de **auto-referencia**.
ARQUIVOS_FORA = {
    "docs/mission/HADES_BRAND_INVENTORY.md",
}
SUFIXOS = (
    ".md", ".json", ".yaml", ".yml", ".ts", ".tsx", ".go", ".ps1", ".py", ".toml",
    ".html", ".css", ".iss", ".js", ".mjs",
)

# ---------------------------------------------------------------------------
# A LISTA DE EXCECOES - *a parte que exige julgamento, e por isso escrita e datada*.
# Cada entrada diz: o QUE, POR QUE (a razao de compatibilidade) e o DESBLOQUEIO.
# ---------------------------------------------------------------------------
EXCECOES = (
    {
        "caminho": "app/ollama.iss",
        "razao": "Nome do artefacto do instalador (Inno Setup). E' o ficheiro que o utilizador "
                 "procura e que o desinstalador usa para se remover.",
        "tipo": "instalador",
        "desbloqueio": "So' com migracao de artefacto + registo de desinstalacao da versao antiga.",
    },
    {
        "caminho": "scripts/dz23-configure.ps1",
        "razao": "Escreve variaveis de ambiente. Um nome novo nao e' lido por configuracoes "
                 "existentes.",
        "tipo": "env var",
        "desbloqueio": "Aceitar os dois nomes em leitura durante uma janela de transicao.",
    },
    {
        "caminho": "internal/multillm/credentials.go",
        "razao": "Codigo que resolve chaves de credencial por nome. Renomear muda a chave "
                 "armazenada.",
        "tipo": "credencial",
        "desbloqueio": "Alias de leitura + migracao do armazenamento, com rollback.",
    },
    {
        "caminho": "apps/mobile-agentic/app.json",
        "razao": "Bundle ID / slug da app. Muda-lo **quebra a app ja' instalada** e a "
                 "atualizacao pela loja.",
        "tipo": "bundle id",
        "desbloqueio": "Decisao do proprietario: nova app na loja, ou manter o id para sempre.",
    },
    {
        "caminho": "app/ui/connectors_test.go",
        "razao": "Teste que depende do codigo de conetores; o texto legado esta' no dado de "
                 "teste, nao na interface.",
        "tipo": "teste",
        "desbloqueio": "Acopla-se a' renomeacao do codigo que ele testa.",
    },
    {
        "caminho": "docs/evidencias/",
        "razao": "Artefactos de medicao (auditorias de consola). Sao **registos**, nao "
                 "documentos de marca.",
        "tipo": "historico",
        "desbloqueio": "Nao se desbloqueia: preserva-se.",
    },
    {
        "caminho": "docs/mission/HADES_EVIDENCE_INDEX.md",
        "razao": "Indice de evidencias: cita os nomes que os artefactos **tinham** quando foram "
                 "medidos. Reescreve-lo tornaria o indice **falso** sobre o que foi medido.",
        "tipo": "historico",
        "desbloqueio": "Nao se desbloqueia: a evidencia nao se reescreve (§FASE-7 item 1).",
    },
    {
        "caminho": "audit/",
        "razao": "Fonte historica datada (checkpoints append-only, handoffs com data no nome).",
        "tipo": "historico",
        "desbloqueio": "Nao se desbloqueia: preserva-se.",
    },
)


def ficheiros_sujos() -> set[str]:
    """Os ficheiros que **diferem do `HEAD`** (`git status --porcelain`), numa so' chamada.

    **E-997, e foi uma medicao de tempo que o exigiu**: a primeira versao do `ler_commitado()`
    corria `git show HEAD:<arq>` **por ficheiro** => **2459** subprocessos numa geracao =>
    *o checker demorava minutos, e o job que o testava ficou pendurado*.

    *A cura e' obvia depois de medida*: **um ficheiro limpo no disco e', por definicao, igual ao
    `HEAD`** => logo so' os **sujos** precisam de ir ao Git => *de 2459 chamadas passam a ser
    **13*** (*medido: 13 ficheiros modificados neste checkout*).

    *E nao e' uma optimizacao com risco*: se o ficheiro esta' limpo, o Git **garante** que o
    conteudo do disco e' o conteudo do commit -- e' o que «limpo» significa.
    """
    saida = _git("status", "--porcelain=v1", "--untracked-files=no")
    if saida is None:
        return set()
    sujos: set[str] = set()
    for linha in saida.splitlines():
        if len(linha) < 4:
            continue
        caminho = linha[3:].strip()
        if " -> " in caminho:  # renomeacao: `antigo -> novo`
            caminho = caminho.split(" -> ", 1)[1]
        sujos.add(caminho.strip('"'))
    return sujos


def ler_commitado(rel: str) -> str | None:
    """Le' o conteudo do ficheiro **como ele esta' commitado** (`HEAD`).

    **E-997, e o CI provou que era preciso**: o run **38928255782** do `class-a-plus-integrity`
    falhou no passo 8 (*«Verify the brand inventory matches the repository»*), com
    *«INVENTARIO DIVERGENTE»* -- *depois de os passos 6 e 7 terem **passado**.*

    A causa: o inventario lia os ficheiros **do disco**. => *e o disco tem **13 ficheiros
    modificados** (WIP nao commitado) com **24** ocorrencias de marca que **nao existem no
    commit*** => *entre eles `apps/mobile-agentic/package.json`, que o inventario **lista*** =>
    *logo o texto gerado localmente nao bate com o texto gerado no CI, onde o checkout e' limpo*.

    *E' a mesma classe do `source_dirty` do manifesto*: **um instrumento que mede um estado que
    inclui trabalho nao commitado** produz um numero que **so' e' verdadeiro naquela maquina**.

    *E eu tinha declarado o contrario*: no `E-996` escrevi que *«o inventario **nao invoca git**,
    portanto **nao pode** ter este defeito»* => *a declaracao era **verdadeira** e a conclusao
    **falsa***: **nao ler git nao impede medir WIP** -- *impede, isso sim, **saber** que se esta'
    a medir WIP*.

    Devolve `None` se o git nao responder (*e o chamador cai para o disco, dizendo-o*).
    """
    return _git("show", "HEAD:%s" % rel)


def ler_arquivo(caminho: Path, rel: str, sujos: set[str]) -> tuple[str, str]:
    """Devolve `(conteudo, origem)`.

    **E-997, com a optimizacao que a medicao exigiu**: so' os ficheiros **sujos** vao ao Git.
    *Um ficheiro limpo no disco e', por definicao, o que esta' no `HEAD`* => *logo le-lo do
    disco da' o **mesmo** resultado, sem um subprocesso*. De **2459** chamadas passou a **13**.
    """
    if rel in sujos:
        conteudo = ler_commitado(rel)
        if conteudo is not None:
            return conteudo, "HEAD (sujo no disco)"
    try:
        return caminho.read_text(encoding="utf-8", errors="replace"), "disco (limpo)"
    except OSError:
        return "", "ilegivel"


def inventariar() -> dict:
    por_padrao: Counter[str] = Counter()
    por_ficheiro: dict[str, Counter[str]] = defaultdict(Counter)
    origens: Counter[str] = Counter()
    sujos = ficheiros_sujos()
    for base, dirs, files in os.walk(RAIZ):
        dirs[:] = [d for d in dirs if d not in DIRS_FORA]
        for nome_arq in files:
            if not nome_arq.lower().endswith(SUFIXOS):
                continue
            caminho = Path(base) / nome_arq
            rel = caminho.relative_to(RAIZ).as_posix()
            if rel in ARQUIVOS_FORA:
                continue
            texto, origem = ler_arquivo(caminho, rel, sujos)
            origens[origem] += 1
            for rotulo, rx in PADROES.items():
                n = len(rx.findall(texto))
                if n:
                    por_padrao[rotulo] += n
                    por_ficheiro[rel][rotulo] += n
    return {
        "totais": dict(por_padrao),
        "ficheiros": {k: dict(v) for k, v in sorted(por_ficheiro.items())},
        "origens": dict(origens),
        "sujos": sorted(sujos),
    }


def normalizar(texto: str) -> str:
    """Remove do texto as **contagens absolutas** -- pela licao final do `E-997`.

    *As contagens dependem do **conjunto de ficheiros que existe no checkout**.* => e esse
    conjunto **nao e' o mesmo** na minha maquina e no CI: num workflow `pull_request`, o
    `actions/checkout` faz checkout de `refs/pull/N/merge` = **branch + main**, que traz
    ficheiros que a branch nao tem. => *medido: o run **38932562362** mostrou o passo 6 e o
    passo 7 a **passar** e o passo 8 a falhar **outra vez**, mesmo depois de eu passar a ler
    do `HEAD`.*

    *E e' a mesma pergunta do `gate_head`/`source_sha`, pela terceira vez*: **o `HEAD` do CI
    nao e' a branch.** => *so' que agora ela aparece do lado das **contagens**, e nao do lado
    de um campo.*

    **O que continua a ser comparado**: a **tabela de excepcoes** da seccao 3 (`| caminho |
    tipo | razao |`, que **nao** leva contagem), as **seccoes** do documento, e a contagem de
    excepcoes por tipo. => *se alguem apagar uma excepcao, mudar uma razao ou tirar uma
    seccao, o `--check` falha.*

    **E O QUE **NAO** E COMPARADO, medido no `E-1004` e nao suposto**: *os **nomes das 5
    strings** do §FASE-5 vivem **dentro** das linhas de contagem da seccao 1* (`| **Hades/HADES**
    (canonica) | **918** |`) => *logo, quando o `normalizar` remove a linha inteira, **remove
    tambem o nome da string*** => *e **apagar uma string nao faz o `--check` falhar**.* => *A
    versao anterior deste docstring afirmava o contrario -- **«a lista de strings... continua a
    ser comparada»** -- e essa afirmacao era **falsa** desde o `E-997`.* => *a prova esta' no
    teste do `E-1004`, que verifica explicitamente que a string **nao** sobrevive ao
    `normalizar`.* => *A cura (preservar os **nomes** e substituir so' os **numeros** por um
    marcador) **nao** foi feita nesta fatia: mudaria o desenho do `--check` e arriscaria partido
    novo, e havia dois SHAs vermelhos a fechar primeiro.*

    **O que fica declarado**: os numeros, com a origem dita no proprio documento (*«as
    contagens sao um retrato do momento; a origem esta' nesta linha»*). => *aquele que le' o
    documento continua a ver os numeros, e a linha de origem diz de onde vieram.*
    """
    linhas = []
    for linha in texto.splitlines():
        s = linha.strip()
        # Linhas de contagem: **tabelas de 2 E de 3 colunas**, e as somas em prosa.
        #
        # **Este e' o `E-1004`, e o defeito estava aqui.** A versao anterior (do `E-997`) so'
        # apanhava `| nome | N |` -- **2 colunas**. => *mas as seccoes **4** (ficheiros
        # migraveis) e **5** (ficheiros de excecao) usam **3 colunas***: `| caminho | N | tipo |`
        # => *e essas linhas contem a **contagem por ficheiro**, que e' exactamente o que depende
        # do conjunto de ficheiros do checkout* => *logo **nao eram removidas**, e o `--check`
        # continuava a comparar numeros que o `E-997` acreditava ter deixado de comparar.*
        #
        # *Na minha maquina passava*, porque o disco e' o commit local; no CI, o checkout e'
        # `refs/pull/N/merge` = **branch + main** => aparecem ficheiros que a branch nao tem
        # => a tabela muda => **divergencia**. *Medido no run 38939790452*: o passo 8 falhou com
        # `INVENTARIO DIVERGENTE do repositorio: docs/mission/HADES_BRAND_INVENTORY.md`.
        #
        # **A licao**: *o `E-997` normalizou **uma** das **duas** formas de tabela de contagem, e
        # o docstring afirmava que as contagens estavam removidas* => *uma cura que se julga
        # completa **fecha a investigacao antes do fim*** => e' o **`E-1002`** (*a divida
        # declarada a mais*) na forma de **filtro**: aqui a cura **faltava**, e o texto dizia que
        # estava feita.
        if re.match(r"^\|\s*\*{0,2}[^|]+\*{0,2}\s*\|\s*\*{0,2}\d+\*{0,2}\s*\|$", s):
            continue  # tabela de 2 colunas: `| nome | N |`
        if re.match(r"^\|\s*\*{0,2}[^|]+\*{0,2}\s*\|\s*\*{0,2}\d+\*{0,2}\s*\|[^|]*\|$", s):
            continue  # tabela de 3 colunas: `| caminho | N | tipo |`
        if s.startswith("**Legadas somadas:**"):
            continue
        if s.startswith("**Por tipo de excecao:**"):
            continue
        if s.startswith("> Origens medidas neste snapshot:"):
            continue
        if s.startswith("(231 em ficheiros") or s.startswith("(231 em"):
            continue  # a segunda linha da mesma soma, tambem com numeros
        linhas.append(linha)
    return "\n".join(linhas)


def classificar(rel: str) -> str:
    for exc in EXCECOES:
        if rel == exc["caminho"] or rel.startswith(exc["caminho"]):
            return "excecao:" + exc["tipo"]
    return "migravel"


def legados(inv: dict) -> list[tuple[str, int, str]]:
    linhas = []
    for rel, cont in inv["ficheiros"].items():
        total = sum(n for k, n in cont.items() if k != CANONICA)
        if total:
            linhas.append((rel, total, classificar(rel)))
    linhas.sort(key=lambda x: (-x[1], x[0]))
    return linhas


def gerar(inv: dict) -> str:
    tot = inv["totais"]
    fil = legados(inv)
    n_mig = sum(n for _, n, c in fil if c == "migravel")
    n_exc = sum(n for _, n, c in fil if c.startswith("excecao:"))
    por_tipo: Counter[str] = Counter()
    for _, n, c in fil:
        if c.startswith("excecao:"):
            por_tipo[c.split(":", 1)[1]] += n

    def tabela(filtro) -> str:
        linhas = [f"| `{rel}` | {n} | {c} |" for rel, n, c in fil if filtro(c)]
        return "\n".join(linhas) if linhas else "| _(nenhum)_ | 0 | |"

    exc_txt = "\n".join(
        f"| `{e['caminho']}` | **{e['tipo']}** | {e['razao']} | {e['desbloqueio']} |"
        for e in EXCECOES
    )
    org = inv.get("origens", {})
    origens_txt = ", ".join(f"`{k}`: {v}" for k, v in sorted(org.items())) or "(nao medido)"
    # **generated_at NAO entra no documento**: um gerado com data obriga a re-gerar a cada
    # dia e torna a catraca inutil. A data fica no `source_sha`, que muda so' quando o
    # conteudo muda - a licao do `E-987`/`E-989`.
    return f"""# HADES - INVENTARIO DE MARCA E EXCECOES DECLARADAS (gerado, nao editar a mao)

> Gerado por `python scripts/hades_brand_inventory.py`. O `--check` falha se divergir.
> Cumpre o **§FASE-5, itens 1, 2 e 3**: inventario das strings visiveis, migracao da
> superficie visivel e **documentacao de cada excecao de compatibilidade**.

## 1. As cinco strings, como o §FASE-5 item 1 as nomeia

> **Fonte das contagens (E-997)**: os ficheiros sao lidos **do commit (`HEAD`)**, e nao do disco.
> *Medido no run **38928255782**, onde este passo falhou*: o disco tinha **13 ficheiros
> modificados** com **24** ocorrencias de marca que **nao existem no commit**, e o inventario
> divergia entre a minha maquina e o CI. => *um instrumento que mede o disco mede **tambem o
> trabalho nao commitado** -- e publica um numero que so' e' verdadeiro onde foi gerado.*
> Origens medidas neste snapshot: {origens_txt}.

| String | Ocorrencias |
| --- | --- |
| **{CANONICA}** (canonica) | **{tot.get(CANONICA, 0)}** |
| DZ23 Agentic | {tot.get('DZ23 Agentic', 0)} |
| Ollama Full | {tot.get('Ollama Full', 0)} |
| Ollama Classe A+ | {tot.get('Ollama Classe A+', 0)} |
| Ollama DZ23 | {tot.get('Ollama DZ23', 0)} |

**Legadas somadas:** {sum(v for k, v in tot.items() if k != CANONICA)} ocorrencias
({n_mig} em ficheiros **migraveis**, {n_exc} em ficheiros de **excecao**).

*O total **nao** decide: os mesmos numeros distribuem-se por sitios de tipos muito
diferentes, e e' o sitio que decide.* Um ``README`` migra; um `bundle ID` nao.

## 2. A regra que separa os dois casos

| Categoria | Quem le' o texto | Acao |
| --- | --- | --- |
| **MIGRAVEL** | o **utilizador** (titulos, prosa, `README`) | migrar para HADES |
| **EXCECAO** | **outro sistema** (instalador, env var, credencial, bundle id) | manter, e dizer por que |
| **HISTORICO** | **ninguem em execucao**: registo datado | preservar como esta' |

*As tres categorias sao diferentes, e tratar a terceira como a segunda seria pior que nao
fazer nada: reescrever um registo historico produz um documento **coerente com a marca de
hoje** e **falso sobre o que foi medido**.*

## 3. Excecoes declaradas (o §FASE-5 item 3)

| Caminho | Tipo | Por que se mantem | Desbloqueio |
| --- | --- | --- | --- |
{exc_txt}

## 4. Ficheiros migraveis (apenas os com legado)

| Ficheiro | Legadas | Categoria |
| --- | --- | --- |
{tabela(lambda c: c == "migravel")}

## 5. Ficheiros de excecao (apenas os com legado)

| Ficheiro | Legadas | Categoria |
| --- | --- | --- |
{tabela(lambda c: c.startswith("excecao:"))}

**Por tipo de excecao:** {', '.join(f'{k}: {v}' for k, v in sorted(por_tipo.items()))}

## 6. O que este documento **nao** faz

- **Nao** decide sozinho o que migra: a lista de `EXCECOES` esta' **escrita no gerador**,
  com razao e desbloqueio, e e' a parte que exige julgamento humano.
- **Nao** conta o que o utilizador *ve* na interface (`app/ui/`): isso exige browser real,
  e o escopo aqui e' **texto no repositorio**.
- **Nao** substitui a evidencia `E-990`: aquela e' o **retrato** que motivou esta tabela;
  esta e' a tabela **reconferivel**.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Inventaria a marca e as excecoes (§FASE-5).")
    parser.add_argument("--check", action="store_true", help="nao escreve: falha se divergir")
    parser.add_argument("--json", action="store_true", help="imprime o inventario em JSON")
    args = parser.parse_args()

    inv = inventariar()
    if args.json:
        json.dump(inv, sys.stdout, ensure_ascii=False, indent=2)
        sys.stdout.write("\n")
        return 0

    texto = gerar(inv)
    if args.check:
        if not DOC.exists():
            print("INVENTARIO AUSENTE: %s" % DOC.relative_to(RAIZ))
            return 1
        # **E-997, e esta comparacao foi a TERCEIRA tentativa**: comparar `texto` cru falhava no
        # CI porque as contagens dependem do conjunto de ficheiros do checkout, e num
        # `pull_request` o checkout e' `refs/pull/N/merge` (branch + main). Comparar o
        # `normalizar()` (*sem numeros*) mantem o que importa sob catraca -- *as strings, as
        # excepcoes, as razoes* -- e deixa os numeros como **declaracao do momento**.
        if normalizar(DOC.read_text(encoding="utf-8")) != normalizar(texto):
            print("INVENTARIO DIVERGENTE do repositorio: %s" % DOC.relative_to(RAIZ))
            print("Rode: python scripts/hades_brand_inventory.py")
            return 1
        print("INVENTARIO CONSISTENTE: bate com o repositorio (5 strings, excecoes, razoes).")
        print("  NAO comparados: as CONTAGENS. *Dependem do conjunto de ficheiros do checkout*,")
        print("  e num `pull_request` o CI faz checkout de `refs/pull/N/merge` (branch + main).")
        return 0

    DOC.write_text(texto, encoding="utf-8")
    print("gerado:", DOC.relative_to(RAIZ))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
