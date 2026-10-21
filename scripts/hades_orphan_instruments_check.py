#!/usr/bin/env python3
"""Catraca dos INSTRUMENTOS ORFAOS (`E-1010`).

**O achado que a originou.** *Ao fechar a lacuna declarada duas vezes -- «a catraca nao foi ligada
ao `hades_audit_instruments_test.py`» (`E-1007`, `E-1008`) -- medi **todos** os `.py` de
`scripts/` contra os gates, em vez de olhar so' para as catracas novas:* **18 ficheiros sem gate
e sem importador**. => *E entre eles, **dois testes**, cujas proprias docstrings dizem porque
existem:*

    hades_mission_check_invariantes_test.py
        ***Um instrumento sem teste proprio e' a lacuna que ele mesmo denuncia.***
        ...os dois verificadores ... ficaram **sem** teste ate' agora...

    hades_timeout_class_check_test.py
        A catraca so' vale se ela reprova a classe certa **e** deixa passar o que nao e' da classe.

=> *Ou seja: existem testes escritos para **fechar** a lacuna «instrumento sem teste», e **esses
testes nunca correm**. => *E' o `E-997` ao contrario: la' o instrumento lia o disco em vez do
commit; aqui o teste **existe e nao e' executado por ninguem**.*

**O que esta catraca verifica**, para cada `.py` de `scripts/`:

1. **Ligado a um gate** (CI ou bateria local), OU
2. **Importado** por outro `.py` de `scripts/` (um helper e' legitimo), OU
3. **Declarado Isento** em `ISENTOS`, com **motivo escrito**.

**Nada mais passa.** => *E a lista `ISENTOS` e' **deliberadamente curta** e **exige motivo**: o
`E-992` diz que uma allowlist que esconde divida nova e' defeito, logo uma isencao **sem motivo**
nao compila -- o proprio teste verifica que todas tem motivo.*

Diferente do `hades_audit_instruments.py` (`E-256`), que reexecuta **scanners** com baseline
declarada (`oraculo`, `rota`), este mede **cobertura de execucao**: *quem corre cada instrumento*.

Uso:
    python scripts/hades_orphan_instruments_check.py            # mede e reporta
    python scripts/hades_orphan_instruments_check.py --check    # exit 1 se houver orfao novo
"""

import argparse
import ast
import io
import os
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPTS = RAIZ / "scripts"

# **Os gates**: onde um instrumento pode ser executado. *Duas fontes, e nao uma.*
FONTES_DE_GATE = [
    ".github/workflows/class-a-plus-integrity.yaml",
    ".github/workflows/test.yaml",
    ".github/workflows/pr-gate.yaml",
    ".github/workflows/dz23-agentic-quality.yaml",
    "scripts/hades_local_gates.ps1",
]

# **Os isentos, com motivo OBRIGATORIO.** => *Uma isencao sem motivo nao e' isencao: e' esconderijo
# (o `E-992`).* => *O teste verifica que todas tem motivo, e o `--check` falha se alguma nao tiver.*
ISENTOS = {
    # *Dados, e nao instrumento:* e' importado por nome de modulo em tempo de execucao indireto.
    "hades_scope_classifier_autoteste_casos.py": "tabela de casos do autoteste; sem main() nem --check",
    "hades_tenant_scope_fixture_subpasta.py": "fixture gerada por teste; nao e' executavel por si",
    # *Geradores/ferramentas de manutencao, e nao gates:* escrevem artefato, nao medem estado.
    "dz23_provider_smoke.py": "sonda opt-in de provedor; exige credencial (BLOCKED_EXTERNAL)",
    "hades_homologacao_verificar.py": "verificador da Camada A para o revisor independente: corre os 10 geradores e os 6 gates de documento, recolhe a saida literal e escreve _tmp/HOMOLOGACAO_<sha>.md; por desenho NAO e chamado por nenhum gate (um gate que chamasse o verificador de homologacao estaria a homologar-se a si proprio) e so escreve em _tmp/",
    "hades_browser_homologacao.py": "sonda opt-in de homologacao de browser; exige playwright+chromium instalados no host, e falha com instrucao de instalacao quando ausentes -- e' a evidencia executavel do requisito 17 do gate, nao um gate de rotina",
    "hades_third_party_notices.py": "gera THIRD_PARTY_NOTICES.md; chamado pelo FASE-5, e nao por gate",
    "hades_checkers_autoteste_controle.py": "controle negativo do autoteste; roda junto com ele",
    "hades_scope_contraprova_obsoletas.py": "contraprova pontual do E-256; ferramenta de auditoria",
    # *Relatorio, e nao gate:* `sys.exit(0 if abertos == 0 else 1)` => *o `1` significa «ha'
    # limites abertos», que e' o **relatorio**, e nao uma violacao.* => *Liga-lo a um gate daria
    # vermelho permanente enquanto houver divida real -- e transformar isso em verde seria
    # esconder a divida (o `E-992`).* => *Ele fica isento com o codigo de saida documentado.*
    "hades_limites_check.py": "relatorio: exit=1 significa «ha limites abertos», nao violacao",
    # *Varredura exploratoria, e nao gate:* o `hades_link_scan.py` mediu **131 artefactos** e
    # **143 documentos** e produziu **10 «ponteiros quebrados» e 1 «artefacto orfao»** --
    # verificados um a um no `E-1055`: **ZERO defeitos reais e QUATRO erros do proprio detector**
    # (comparar `basename` contra caminhos; `fullmatch` que exclui nomes longos em silencio;
    # ficheiros gerados em runtime como o `INSTALACAO.json`; itens marcados «A CRIAR»).
    # => *Um instrumento que produz falsos positivos **nao deve virar gate**: liga-lo daria
    # vermelho sobre documentos correctos, e a catraca seria desacreditada pela propria lista.*
    # => *Fica isento com o motivo escrito; a divida e' a pergunta que ele ainda nao sabe fazer.*
    "hades_link_scan.py": "varredura exploratoria; da' falsos positivos (E-1055) -- nao deve ser gate",
    # *A QUARTA varredura exploratoria.* Ele fez a **pergunta certa** (`E-1056`): *quais
    # documentos **afirmam produzir** um artefacto e nao o citam?* -- e deu **7 achados**,
    # verificados um a um: **os 7 eram falsos positivos**, todos do mesmo erro, porque o regex
    # aceita `e' a fonte`, `gera` **e** `esta' em`, `a partir de`. => ***Um detector lexico nao ve
    # a relacao: ve os termos.*** => *Mesma razao do `hades_link_scan.py`: falso positivo nao vira
    # gate. A regra dos quatro: **varredura exploratoria fica isenta; so' entra na bateria o que
    # discrimina.***
    "hades_claim_scan.py": "varredura exploratoria; detector lexico da' falsos positivos (E-1056)",
    # *Anti-armadilha deliberado:* recusa (exit=2) se a receita `LOCALAPPDATA` vier incompleta,
    # e declara BLOCKED_BY_EXTERNAL_DEPENDENCY (exit=3) sem `go`. => *Agora corre no passo 12 da
    # bateria, que e' um gate -- logo a isencao seria desnecessaria, e o proprio `--check`
    # deteta isso (o `E-1002`: sobre-declarar tambem e' defeito).*
}


def modulos(todos):
    """Devolve {ficheiro: {ficheiros que o importam}}."""
    importadores = {}
    for f in todos:
        try:
            texto = io.open(SCRIPTS / f, encoding="utf-8", errors="replace").read()
            arvore = ast.parse(texto)
        except SyntaxError:
            continue
        # **Invocacao por SUBPROCESS tambem e' dependencia.** => *E' o caso dos DOIS scanners de
        # relatorio (`hades_route_scope_scan`, `hades_tenant_oracle_scan`): eles nao tem `main()`
        # nem `sys.exit`, e sao **consumidos** pelo `hades_audit_instruments.py`, que e' gate.*
        # => *Confundir «nao executavel por si» com «orfao» seria o `E-1003` outra vez: medir o
        # nome em vez da relacao.*
        # *So conta se houver mesmo invocacao:* um nome entre aspas numa lista de politica
        # (a tabela `ISENTOS` deste proprio ficheiro) **nao** e' uma chamada.
        invoca = re.search(r"subprocess\.|check_output\(|Popen\(|\.run\(", texto)
        if invoca and f != os.path.basename(__file__):
            for m in re.finditer(r'["\'][^"\']*?([A-Za-z0-9_\-]+\.py)["\']', texto):
                g = m.group(1)
                if g != f and (SCRIPTS / g).exists():
                    importadores.setdefault(g, set()).add(f)
        for no in ast.walk(arvore):
            alvos = []
            if isinstance(no, ast.Import):
                alvos = [a.name for a in no.names]
            elif isinstance(no, ast.ImportFrom) and no.module:
                alvos = [no.module]
            for alvo in alvos:
                g = alvo.split(".")[-1] + ".py"
                if g != f and (SCRIPTS / g).exists():
                    importadores.setdefault(g, set()).add(f)
    return importadores


def gates():
    texto = ""
    for rel in FONTES_DE_GATE:
        p = RAIZ / rel
        if p.exists():
            texto += "\n" + io.open(p, encoding="utf-8", errors="replace").read()
    return texto


def medir():
    todos = sorted(f for f in os.listdir(SCRIPTS) if f.endswith(".py"))
    g = gates()
    imp = modulos(todos)
    resultado = {}
    for f in todos:
        base = f[:-3]
        no_gate = (f in g) or (base in g)
        importado = sorted(imp.get(f, ()))
        if no_gate:
            resultado[f] = ("GATE", "")
        elif importado:
            resultado[f] = ("HELPER", "importado por " + ", ".join(importado[:3]))
        elif f in ISENTOS:
            resultado[f] = ("ISENTO", ISENTOS[f])
        else:
            resultado[f] = ("ORFAO", "")
    return todos, resultado


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--check", action="store_true", help="nao escreve: exit 1 se houver orfao novo")
    args = ap.parse_args()

    todos, resultado = medir()
    por_classe = {}
    for f, (cls, _motivo) in resultado.items():
        por_classe.setdefault(cls, []).append(f)

    print("INSTRUMENTOS EM scripts/: %d" % len(todos))
    for cls in ("GATE", "HELPER", "ISENTO", "ORFAO"):
        fs = sorted(por_classe.get(cls, ()))
        print("  %-8s %d" % (cls, len(fs)))
        for f in fs:
            motivo = resultado[f][1]
            print("      %-46s %s" % (f, motivo))

    problemas = []

    # (1) **Orfao novo**: nem gate, nem importado, nem isento com motivo.
    for f in sorted(por_classe.get("ORFAO", ())):
        problemas.append(
            "INSTRUMENTO ORFAO: %s => *nao e' executado por nenhum gate, nao e' importado, e nao "
            "esta' isento* => *um instrumento que nao corre nao mede nada*" % f
        )

    # (2) **Isencao SEM motivo**: a regra do `E-992` -- uma allowlist que esconde divida.
    for f, motivo in ISENTOS.items():
        if not motivo or not motivo.strip():
            problemas.append("ISENCAO SEM MOTIVO: %s => *uma isencao sem motivo e' esconderijo*" % f)
        if f not in todos:
            problemas.append("ISENCAO OBSOLETA: %s => *isento de um ficheiro que ja' nao existe*" % f)
        # *E uma isencao que JA' e' executada e' divida declarada a mais (o `E-1002`).*
        if resultado.get(f, ("", ""))[0] == "GATE":
            problemas.append(
                "ISENCAO DESNECESSARIA: %s => *ja' esta' ligado a um gate, logo nao precisa de "
                "isencao* => *sobre-declarar tambem e' defeito (o `E-1002`)*" % f
            )

    if problemas:
        print("PROBLEMAS:")
        for p in problemas:
            print("  - %s" % p)
        return 1 if args.check else 0

    print("COBERTURA CONSISTENTE: todo instrumento corre num gate, e' helper importado,")
    print("  ou esta' isento com motivo escrito (%d isencoes)." % len(ISENTOS))
    return 0


if __name__ == "__main__":
    sys.exit(main())
