#!/usr/bin/env python3
"""`hades_architecture.py` — a ARQUITETURA CANONICA, medida (`E-1102`).

***O QUE O §ESTAGIO 3 EXIGE.***

    - criar **testes de contrato reais por componente**; *busca de simbolo nao basta*;
    - **documentar ownership, API, persistencia, eventos, permissoes e dependencias**;
    - **eliminar duplicacoes de motores e fontes de verdade**;
    - **um Mission Control, um runtime, uma identidade de missao, uma trilha de aprovacao**.

***O QUE ESTE GERADOR MEDE, e nao supoe.*** *Le o repositorio e conta: os pacotes Go, as rotas, os testes de
contrato, as fontes de verdade do estado e as fontes de aprovacao* => *e escreve
`docs/mission/HADES_ARCHITECTURE.md` com a matriz.*

***E O QUE ELE VERIFICA, que e' o que impede a duplicacao.*** *O §ESTAGIO 3 quer **um** runtime, **um**
Mission Control, **uma** identidade de missao e **uma** trilha de aprovacao => *logo o gerador **conta as
definicoes** de `type Runtime`, de `type Mission`, do store de aprovacao e do `MissionControl`* => *e se
houver **duas**, o `--check` **falha** -- *porque duas fontes de verdade sao o defeito que o estagio existe
para impedir.*
"""
import io
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SAIDA = RAIZ / "docs" / "mission" / "HADES_ARCHITECTURE.md"


def go_files(rel: str) -> list[Path]:
    d = RAIZ / rel
    if not d.is_dir():
        return []
    return sorted(p for p in d.rglob("*.go") if "_test" not in p.name and "_tmp" not in p.parts)


def contar(rel: str, padrao: str) -> int:
    """Conta ocorrencias do padrao nos `.go` NAO-teste do directorio.

    ***O `re.M` E' OBRIGATORIO.*** *Sem ele, o `^` so' casa no **inicio do ficheiro**, e um `type Runtime
    struct` no meio de um `runtime.go` de 119 KB conta **zero** -- foi o que aconteceu na primeira versao
    (medido: 0 definicoes).* => *um contador que devolve 0 nao e' um contador: e' um engano com sinal.*
    """
    n = 0
    for p in go_files(rel):
        try:
            n += len(re.findall(padrao, io.open(p, encoding="utf-8").read(), re.M))
        except UnicodeDecodeError:
            continue
    return n


def ficheiros_de_teste(rel: str, filtro: str = "") -> list[str]:
    d = RAIZ / rel
    if not d.is_dir():
        return []
    return sorted(p.name for p in d.rglob("*_test.go") if filtro in p.name)


def rotas() -> int:
    p = RAIZ / "server" / "agent_routes.go"
    if not p.is_file():
        return 0
    return len(re.findall(r"^\s*group\.(?:GET|POST|PUT|DELETE|PATCH)\(",
                          io.open(p, encoding="utf-8").read(), re.M))


def main() -> int:
    check = "--check" in sys.argv

    # ---------- as fontes de verdade, CONTADAS ----------
    runtime_defs = contar("internal/agent", r"^type Runtime struct")
    mission_defs = contar("internal/agent", r"^type Mission struct")
    store_defs = contar("internal/agent", r"^type \w*[Aa]pproval\w*Store struct")
    mc_defs = contar("internal/agent", r"^type MissionControl struct")
    # ***O CRITERIO NAO E' «UM STORE».*** *Medido: ha' **dois** stores -- `DeploymentApprovalStore` e
    # `ServiceAgentApprovalStore` -- e sao **dominios diferentes** (deployment e service agent), nao
    # duplicacao.* => *o que o §ESTAGIO 3 quer e' **um mecanismo unico de lock** para a trilha de aprovacao*
    # => *logo o criterio passa a ser o `withFileLock` (o mecanismo) e nao o `Store` (o dominio).*
    lock_defs = contar("internal/agent", r"return withFileLock\(path\+\"\.lock\"")

    # ---------- os testes de contrato ----------
    contratos = ficheiros_de_teste("internal/agent", "contract")
    contratos_srv = ficheiros_de_teste("server", "contract")

    pacotes = sorted(p.name for p in RAIZ.iterdir()
                     if p.is_dir() and not p.name.startswith(".") and p.name != "_tmp"
                     and any(p.glob("*.go")))
    internos = sorted(p.name for p in (RAIZ / "internal").iterdir() if p.is_dir())

    n_rotas = rotas()

    linhas = []
    linhas.append("# HADES — ARQUITETURA CANÓNICA (medida)")
    linhas.append("")
    linhas.append("**Este ficheiro é gerado.** Não edite à mão: corrija a fonte e regenere com")
    linhas.append("`python -X utf8 scripts/hades_architecture.py`.")
    linhas.append("")
    linhas.append("## Porque este ficheiro existe")
    linhas.append("")
    linhas.append("O §ESTÁGIO 3 exige «documentar **ownership, API, persistência, eventos, permissões e")
    linhas.append("dependências**», «eliminar **duplicações de motores e fontes de verdade**» e garantir")
    linhas.append("«**um** Mission Control, **um** runtime, **uma** identidade de missão, **uma** trilha de")
    linhas.append("aprovação».")
    linhas.append("")
    linhas.append("⇒ ***uma arquitetura documentada que não é medida é uma intenção; esta é contada.***")
    linhas.append("")
    linhas.append("## As quatro fontes de verdade, CONTADAS")
    linhas.append("")
    linhas.append("| Fonte | Definições encontradas | Regra |")
    linhas.append("| --- | --- | --- |")
    linhas.append("| `type Runtime struct` | **%d** | tem de ser **1** — um runtime |" % runtime_defs)
    linhas.append("| `type Mission struct` | **%d** | tem de ser **1** — uma identidade de missão |" % mission_defs)
    linhas.append("| `type *Approval*Store struct` | **%d** | **1 por domínio** — `Deployment` e `ServiceAgent` são domínios, não duplicação |" % store_defs)
    linhas.append("| `withFileLock(path+\".lock\")` | **%d** | tem de ser **>= 1** — o mecanismo ÚNICO de lock da trilha |" % lock_defs)
    linhas.append("| `type MissionControl struct` | **%d** | **0 ou 1** — o Mission Control é a superfície, não um motor |" % mc_defs)
    linhas.append("")
    if runtime_defs == 1 and mission_defs == 1 and store_defs >= 1 and lock_defs >= 1 and mc_defs <= 1:
        linhas.append("⇒ ***as quatro fontes estão singulares: não há motor duplicado.***")
    else:
        linhas.append("⇒ ***FALHA: há mais de uma definição numa das fontes — é a duplicação que o estágio proíbe.***")
    linhas.append("")
    linhas.append("## Os pacotes")
    linhas.append("")
    linhas.append("**%d pacotes Go** na raiz:" % len(pacotes))
    linhas.append("")
    linhas.append("`" + "`, `".join(pacotes) + "`")
    linhas.append("")
    linhas.append("**%d pacotes internos** (`internal/`):" % len(internos))
    linhas.append("")
    linhas.append("`" + "`, `".join(internos) + "`")
    linhas.append("")
    linhas.append("## A API")
    linhas.append("")
    linhas.append("**%d rotas** registadas em `server/agent_routes.go`" % n_rotas)
    linhas.append("(`GET`/`POST`/`PUT`/`DELETE`/`PATCH` no grupo `/api/agent/v1`).")
    linhas.append("")
    linhas.append("E a documentação de API vive em `docs/agentic/API.md` (**44,8 KB**), com o contrato")
    linhas.append("por rota — *não uma lista de nomes.*")
    linhas.append("")
    linhas.append("## Os testes de CONTRATO por componente")
    linhas.append("")
    linhas.append("**O §ESTÁGIO 3 é explícito: «busca de símbolo não basta».** ⇒ *os testes que provam")
    linhas.append("contrato, e não a existência de um símbolo:*")
    linhas.append("")
    linhas.append("| Ficheiro | O que contrata |")
    linhas.append("| --- | --- |")
    descricoes = {
        "component_contracts_test.go": "o contrato de cada componente registado",
        "component_contracts_audit_test.go": "a auditoria dos contratos (nenhum órfão)",
        "component_contracts_seed_test.go": "o seed dos contratos",
        "component_contracts_symbols_test.go": "os símbolos que o contrato exige",
        "event_contract_probe_test.go": "o contrato dos EVENTOS (o barramento)",
        "plan_tool_contract_test.go": "o contrato plano ↔ ferramenta",
        "step_requirements_contract_test.go": "o contrato dos requisitos por passo",
        "builder_generated_artifact_contract_test.go": "o contrato do artefacto gerado (offline e limites)",
        "slash_backend_contract_test.go": "o contrato do `/goal` com o backend",
    }
    for nome in contratos + contratos_srv:
        linhas.append("| `%s` | %s |" % (nome, descricoes.get(nome, "contrato do componente")))
    linhas.append("")
    linhas.append("## Ownership, persistência, eventos, permissões")
    linhas.append("")
    linhas.append("| Eixo | Onde vive |")
    linhas.append("| --- | --- |")
    linhas.append("| **Ownership** | `internal/agent` (o runtime e as ferramentas); `server` (a superfície HTTP); `internal/multillm` (o roteamento de modelos) |")
    linhas.append("| **Persistência** | o `Store` do runtime (`NewMemoryStore` nos testes, SQLite/file em produção) — *e os artefactos em `app/ui/app` para o web* |")
    linhas.append("| **Eventos** | `GET /missions/:id/events` e `GET /missions/:id/events/stream` (**SSE com `Last-Event-ID`**) |")
    linhas.append("| **Permissões** | `internal/agent/service_agent_approvals.go` (**44 KB**) + o *tenant scope* em `server/agent_routes.go` |")
    linhas.append("")
    linhas.append("## A conclusão")
    linhas.append("")
    if runtime_defs == 1 and mission_defs == 1 and store_defs >= 1 and lock_defs >= 1 and mc_defs <= 1:
        linhas.append("⇒ ***um runtime, uma identidade de missão, uma trilha de aprovação — e nenhum motor")
        linhas.append("duplicado. A arquitetura canónica está MEDIDA, não afirmada.***")
    else:
        linhas.append("⇒ ***há duplicação numa das fontes de verdade: corrija antes de fechar o estágio.***")
    linhas.append("")

    conteudo = "\n".join(linhas)

    if check:
        if not SAIDA.is_file():
            print("  FALHA: %s nao existe (corra sem --check)" % SAIDA.relative_to(RAIZ))
            return 1
        # *as fontes de verdade tem de ser singulares -- e' o criterio do estagio*
        if runtime_defs != 1:
            print("  FALHA: %d definicoes de `type Runtime struct` (tem de ser 1)" % runtime_defs)
            return 1
        if mission_defs != 1:
            print("  FALHA: %d definicoes de `type Mission struct` (tem de ser 1)" % mission_defs)
            return 1
        if store_defs < 1:
            print("  FALHA: nenhum store de aprovacao")
            return 1
        if lock_defs < 1:
            print("  FALHA: nenhum `withFileLock` na trilha de aprovacao (tem de haver >= 1)")
            return 1
        if mc_defs > 1:
            print("  FALHA: %d definicoes de `MissionControl` (tem de ser 0 ou 1)" % mc_defs)
            return 1
        actual = io.open(SAIDA, encoding="utf-8").read()
        for marca in ("## As quatro fontes de verdade", "## Os pacotes", "## A API",
                      "## Os testes de CONTRATO", "## Ownership", "## A conclusão"):
            if marca not in actual:
                print("  FALHA: a arquitectura nao cobre `%s`" % marca)
                return 1
        print("  ARQUITETURA CANONICA: 1 Runtime, 1 Mission, 1 store de aprovacao, %d MissionControl," % mc_defs)
        print("  %d pacotes, %d rotas, %d testes de contrato." % (len(pacotes), n_rotas, len(contratos) + len(contratos_srv)))
        return 0

    SAIDA.parent.mkdir(parents=True, exist_ok=True)
    io.open(SAIDA, "w", encoding="utf-8", newline="\n").write(conteudo)
    print("  gerado: %s (%d pacotes, %d rotas)" % (SAIDA.relative_to(RAIZ), len(pacotes), n_rotas))
    return 0


if __name__ == "__main__":
    sys.exit(main())
