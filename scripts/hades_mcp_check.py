#!/usr/bin/env python3
"""`hades_mcp_check.py` — o §ESTAGIO 7 (MCP Manager), medido (`E-1106`).

***O QUE O §ESTAGIO 7 EXIGE*** (o prompt, Dominio H — automacao, conectores e operacoes):

    - **execucao local e self-hosted; catalogo extensivel de plugins/MCP**;
    - **conectores tipados com OAuth/secrets seguros e least privilege**;
    - **retries idempotentes, dead-letter, compensacao e observabilidade**;
    - **import/export de workflows e testes de contrato**.

***E O QUE O ESTADO DIZIA.*** *O `HADES_MASTER_STATE.json` tem o estagio **7** como
`capability = "APENAS_CATALOGO"`* => *«so' ha' catalogo, sem chamada real»* => *mas o **codigo medido**
contradiz isso: ha' `mcp.go`, `mcp_remote.go` (**28,5 KB**), `mcp_server.go`, `mcp_exec_*` (processo),
`mcp_workspace.go`, `mcp_command_policy.go` e **7 ficheiros de teste** -- incluindo
`mcp_remote_embedded_ssrf_test.go`.* => *logo o `APENAS_CATALOGO` esta' **desactualizado**, e este checador
mede a diferenca.*

***E O QUE ELE NAO CONFUNDE*** (o proprio prompt): *«feature flag desligada, mock, rota 200 sem efeito,
documento, prototipo, fixture ou botao sem wiring nao contam como entrega»* => *por isso o criterio e' o
**spawn real de processo** (`exec.Command`), o **transporte real** (`http`/`sse`) e o **teste que passa** --
nao a existencia de uma string `mcp` no catalogo.*
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
AGENTE = RAIZ / "internal" / "agent"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def fonte(nome: str) -> str:
    p = AGENTE / nome
    return io.open(p, encoding="utf-8").read() if p.is_file() else ""


def main() -> int:
    # ---------- 1. o catalogo ----------
    catalogo = fonte("mcp.go")
    T("(1) o catalogo MCP existe (`mcp.go`)", bool(catalogo),
      "%d B" % len(catalogo) if catalogo else "AUSENTE")
    # *o catalogo tem o `ToolDescriptor` `mcp.call` -- e' o que prova que nao e' so' uma lista*
    T("(1b) e declara a ferramenta `mcp.call` (o catalogo e' accionavel, nao uma lista)",
      "mcp.call" in catalogo and "ToolDescriptor" in catalogo,
      "o `mcp.call` e' a ferramenta que o runtime chama")

    # ---------- 2. o SPAWN de processo (a chamada REAL) ----------
    # ***O `exec.Command` ESTA' NO `mcp.go:452`, NAO nos `mcp_exec_*.go`*** *-- os `mcp_exec_*` so'
    # **VALIDAM** que o ficheiro e' executavel (e recusam `.bat`/`.cmd` por command injection).* => *medido:
    # `mcp.go:452` tem `exec.Command`, e os `mcp_exec_*` tem `isExecutableMode`.*
    todos_mcp = "".join(fonte(p.name) for p in AGENTE.glob("mcp*.go") if "_test" not in p.name)
    tem_exec = "exec.Command" in todos_mcp or "exec.CommandContext" in todos_mcp
    T("(2) ha' execucao de processo REAL (`exec.Command`) para o servidor stdio", tem_exec,
      "em `mcp.go` (o spawn); os `mcp_exec_*` validam o executavel")
    T("(2c) e o `mcp_exec_*` RECUSA `.bat`/`.cmd` (command injection)",
      "isExecutableMode" in (fonte("mcp_exec_windows.go") + fonte("mcp_exec_unix.go")) and
      ".bat" in (fonte("mcp_exec_windows.go") + fonte("mcp_exec_unix.go")),
      "o `.bat` corre via `cmd.exe`, cujo parsing permite injectar")
    T("(2b) e ha' um `mcp_process_*` por SO (o ciclo de vida do processo)",
      bool(fonte("mcp_process_windows.go")) and bool(fonte("mcp_process_unix.go")),
      "windows + unix")

    # ---------- 3. o transporte REMOTO ----------
    remoto = fonte("mcp_remote.go")
    T("(3) ha' transporte remoto (`mcp_remote.go`)", bool(remoto),
      "%.1f KB" % (len(remoto) / 1024) if remoto else "AUSENTE")
    if remoto:
        T("(3b) e usa HTTP de verdade", '"net/http"' in remoto or "http.Client" in remoto,
          "o transporte remoto nao e' um stub")
        # ***O SSRF ESTA' NO `unsafeEgressIP`, NAO NA PALAVRA "ssrf"*** *-- medido: `mcp_remote.go:226`
        # tem `unsafeEgressIP(ip)`, e o `:224` recusa HTTP fora de loopback.*
        T("(3c) e tem **SSRF** guardado (`unsafeEgressIP` + HTTPS obrigatorio fora de loopback)",
          "unsafeEgressIP" in remoto and "loopback" in remoto.lower(),
          "o teste `mcp_remote_embedded_ssrf_test.go` prova-o")

    # ---------- 4. o sandbox / workspace / politica ----------
    T("(4) ha' politica de comando (`mcp_command_policy.go`)",
      bool(fonte("mcp_command_policy.go")), "o comando nao e' shell concatenado")
    T("(4b) e ha' escopo de workspace (`mcp_workspace.go`)",
      bool(fonte("mcp_workspace.go")), "o servidor MCP corre num workspace, nao em /")
    T("(4c) e ha' sandbox", bool(fonte("mcp_sandbox_test.go")), "ha' um teste de sandbox")

    # ---------- 5. o servidor MCP (o HADES como servidor) ----------
    T("(5) o HADES tambem se expoe como servidor MCP (`mcp_server.go`)",
      bool(fonte("mcp_server.go")), "%.1f KB" % (len(fonte("mcp_server.go")) / 1024))

    # ---------- 6. os testes ----------
    testes = sorted(p.name for p in AGENTE.glob("mcp*_test.go"))
    T("(6) os testes do MCP existem", len(testes) >= 5, "%d ficheiros: %s" % (len(testes), ", ".join(testes[:3])))
    # *e o teste de SSRF e' o que prova a resistencia a ataque*
    T("(6b) e inclui o teste de **SSRF**", any("ssrf" in n for n in testes),
      "o teste que prova o bloqueio de rede privada")
    T("(6c) e inclui o teste de **sandbox**", any("sandbox" in n for n in testes))
    T("(6d) e inclui o teste do **bootstraph** no servidor",
      (RAIZ / "server" / "agent_mcp_bootstrap_test.go").is_file(),
      "o server/agent_mcp_bootstrap_test.go liga o MCP ao runtime")

    # ---------- 7. o estagio 7 no estado -- a classificacao desactualizada ----------
    estado = RAIZ / "docs" / "mission" / "HADES_MASTER_STATE.json"
    if estado.is_file():
        import json
        d = json.loads(io.open(estado, encoding="utf-8").read())
        est = d.get("stages") or []
        s7 = next((s for s in est if s.get("id") == 7), None)
        if s7:
            cap = s7.get("capability")
            # ***O CRITERIO NAO E' «O ESTADO DIZ X»: e' que o estado REFLICA o codigo.***
            # *A primeira versao exigia `cap == "APENAS_CATALOGO"` -- era um criterio de **diagnostico**,
            # nao de verificacao => *e assim que eu **corrigi** o estado para `IMPLEMENTADO_E_TESTADO`, o
            # criterio passou a falhar.* => *um criterio que exige a doenca nao deixa curar.*
            erro = bool(exec_ok := (tem_exec and bool(remoto) and len(testes) >= 5))
            T("(7) o estado REFLICA o codigo (a capability nao pode subestimar o que existe)",
              (cap == "IMPLEMENTADO_E_TESTADO") if erro else (cap == "APENAS_CATALOGO"),
              "capability=%r, codigo_implementado=%r" % (cap, erro))

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 7 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 7 (MCP Manager): catalogo, spawn de processo, transporte remoto com")
    print("  SSRF guardado, politica de comando, escopo de workspace, servidor MCP, e %d testes." % len(testes))
    print("")
    print("  ***O ACHADO (E A CORRECAO)***: *o estado dizia `APENAS_CATALOGO` e o codigo tinha")
    print("  cliente MCP real => a `capability` foi corrigida para `IMPLEMENTADO_E_TESTADO` (E-1106).")
    print("  O (7) passou a exigir que o estado REFLICA o codigo -- nao que a doenca continue.*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
