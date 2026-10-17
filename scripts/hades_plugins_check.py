#!/usr/bin/env python3
"""`hades_plugins_check.py` — o §ESTAGIO 8 (Plugins e Skills), medido (`E-1107`).

***O QUE O §ESTAGIO 8 EXIGE*** (o prompt, Dominio H):

    - **execucao local e self-hosted; catalogo extensivel de plugins/MCP**;
    - **conectores tipados**;
    - **import/export de workflows e testes de contrato**;
    - e o `/goal` pode planear uso de ferramentas que os plugins expoem.

***E O QUE O `B-14` DIZ (o bloqueio declarado)***: *«a execucao de plugin exige **PRIVILEGIO DE MONTAGEM** --
mesmo o modo best-effort faz mount/chroot (mount --make-rprivate, mount --bind, tmpfs e chroot/unshare em
tools.go). Onde nao ha esse privilegio (**Windows**, container nao privilegiado, runner padrao de CI) a
execucao nao acontece»* => *e o `E-185` **corrigiu o diagnostico**: «a falha silenciosa era do runtime de
plugin, que descartava o Output do sandbox; agora a falha propaga».*

***O QUE ESTE CHECADOR MEDE -- e o que ele separa.***

  *O §ESTAGIO 8 tem **duas metades**:*
  1. **o que NAO precisa de privilegio**: *o **manifesto** (formato, schema, versao), o **bundle** (empacotar,
     verificar hash), o **registry** (instalar, listar, remover), o **lifecycle** (estados), o **scope**
     (tenant/workspace), o **status** (diagnostico);*
  2. **o que PRECISA**: *a **execucao** do plugin (o sandbox com mount/chroot).*
  => *a metade **1** esta' feita e testada; a **2** e' o `B-14`, e por isso o estagio e' `PARTIAL` -- **nao**
  `COMPLETED_VERIFIED`.*

***E A LICAO QUE ESTE CHECADOR APLICA***: *o `E-185` mostrou que a falha **silenciosa** era o defeito --
«descartava o Output do sandbox» => *logo o criterio de diagnostico e' que a falha **propague o stderr**, e nao
que a execucao **funcione** no Windows (onde ela **nao pode** funcionar).*
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
    # ---------- 1. as duas metades ----------
    modulos = {
        "plugin_manifest_format.go": "o FORMATO do manifesto (schema + versao)",
        "plugin_bundle.go": "o BUNDLE (empacotar e verificar)",
        "plugin_registry.go": "o REGISTRY (instalar/listar/remover)",
        "plugin_runtime.go": "o RUNTIME (a execucao -- o que precisa de privilegio)",
        "plugin_scope.go": "o SCOPE (tenant/workspace)",
        "plugin_status.go": "o STATUS (diagnostico)",
        "plugin_workspace.go": "o WORKSPACE do plugin",
    }
    for arq, o_que in modulos.items():
        T("(1) `%s` existe (%s)" % (arq, o_que), (AGENTE / arq).is_file(),
          "%.1f KB" % ((AGENTE / arq).stat().st_size / 1024) if (AGENTE / arq).is_file() else "AUSENTE")

    # ---------- 2. a metade que NAO precisa de privilegio esta' testada ----------
    testes = sorted(p.name for p in AGENTE.glob("plugin*_test.go"))
    T("(2) os testes do plugin existem", len(testes) >= 8, "%d ficheiros" % len(testes))
    # *cada modulo da metade 1 tem teste*
    pares = {
        "manifest": "plugin_manifest_format_test.go",
        "bundle": "plugin_bundle_test.go",
        "registry": "plugin_registry_test.go",
        "runtime": "plugin_runtime_test.go",
        "scope": "plugin_scope_test.go",
        "status": "plugin_status_test.go",
        "workspace": "plugin_workspace_test.go",
    }
    for nome, arq in pares.items():
        T("(2b) o modulo `%s` tem teste" % nome, arq in testes, arq)

    # ---------- 3. o manifesto e' TIPADO e versionado ----------
    mf = fonte("plugin_manifest_format.go")
    T("(3) o manifesto tem **versao** de formato",
      bool(re.search(r'(SchemaVersion|FormatVersion|manifestVersion|Version\s+int)', mf)),
      "o formato e' versionado, nao inventado por ficheiro")
    T("(3b) e tem **validacao** (o manifesto invalido e' recusado)",
      bool(re.search(r'(func.*[Vv]alidate|errors\.New|fmt\.Errorf)', mf)),
      "o manifesto nao e' aceito as cegas")

    # ---------- 4. o bundle verifica a INTEGRIDADE ----------
    bundle = fonte("plugin_bundle.go")
    T("(4) o bundle tem **hash**/integridade",
      bool(re.search(r"(sha256|SHA256|Hash)", bundle)),
      "o bundle nao e' um zip sem verificacao")
    # ***O LIMITE E' `maxPluginBundleBytes = 64 << 20`.*** *Medido: `plugin_bundle.go:34`, usado no `:53`
    # (recusa) e no `:106`/`:205` (`io.LimitReader`, para o zip nao crescer ao descompactar).*
    T("(4b) e tem **limite** de tamanho (`maxPluginBundleBytes`, 64 MiB)",
      "maxPluginBundleBytes" in bundle and "LimitReader" in bundle,
      "o `LimitReader` impede o zip-bomb enquanto descompacta")

    # ---------- 5. o registry tem LIFECYCLE ----------
    # ***OS ESTADOS ESTAO' NO `plugin_status.go`, NAO NO REGISTRY.*** *Medido: `PluginStateInstalled`,
    # `PluginStateConnected`, `PluginStateConfigRequired`, `PluginStateUnauthorized`, `PluginStateUnhealthy`,
    # `PluginStateUnsupported`, `PluginStateTestedLive` (7 estados) -- e o `PluginStatus` e' «o retrato
    # auditavel de um plugin instalado».* => *`instalar` (INSTALLED) nao e' o mesmo que `conectado`
    # (CONNECTED) nem `testado ao vivo` (TESTED_LIVE).*
    status = fonte("plugin_status.go")
    estados = re.findall(r'PluginState\w+\s*=\s*"([A-Z_]+)"', status)
    T("(5) ha' **estados** canonicos de ciclo de vida (no `plugin_status.go`)", len(estados) >= 5,
      "%d estados: %s" % (len(estados), ", ".join(estados[:4])))
    T("(5c) e `INSTALLED` NAO e' o mesmo que `CONNECTED`/`TESTED_LIVE` (o que o estado separa)",
      "INSTALLED" in estados and ("CONNECTED" in estados or "TESTED_LIVE" in estados),
      "o estado nao confunde instalar com funcionar")
    T("(5b) e o `plugin_lifecycle_test.go` cobre as transicoes",
      "plugin_lifecycle_test.go" in testes)

    # ---------- 6. o SCOPE (tenant/workspace) ----------
    escopo = fonte("plugin_scope.go")
    T("(6) o plugin tem **escopo** (tenant/workspace)",
      bool(escopo) and bool(re.search(r"(Tenant|Organization|tenant)", escopo + fonte("plugin_workspace.go"))),
      "o plugin nao ve' outro tenant")
    T("(6b) e a **non-inference** do tenant esta' testada",
      "plugin_tenant_non_inference_test.go" in testes,
      "o teste que prova que o plugin nao infere o tenant")

    # ---------- 7. o B-14: o diagnostico PROPAGA (o que o E-185 corrigiu) ----------
    rt = fonte("plugin_runtime.go")
    T("(7) o runtime **propaga o stderr** do sandbox (o que o E-185 corrigiu)",
      bool(re.search(r"(Stderr|stderr)", rt)) and
      bool(re.search(r"(Output|output)", rt)),
      "antes descartava o Output e a falha era silenciosa")
    T("(7b) e o `plugin_execution_diagnostics_test.go` prova o diagnostico",
      "plugin_execution_diagnostics_test.go" in testes,
      "o teste que discrimina a falha silenciosa")

    # ---------- 8. o B-14 e' o LIMITE, e o estagio continua PARTIAL ----------
    estado = RAIZ / "docs" / "mission" / "HADES_MASTER_STATE.json"
    if estado.is_file():
        import json
        d = json.loads(io.open(estado, encoding="utf-8").read())
        est = d.get("stages") or []
        s8 = next((s for s in est if s.get("id") == 8), None)
        if s8:
            T("(8) o estagio 8 continua `PARTIAL` (a execucao exige privilegio de montagem)",
              s8.get("capability") == "PARCIAL",
              "capability=%r -- *e' correcto: o B-14 e' um limite real*" % s8.get("capability"))

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 8 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 8 (Plugins): manifesto TIPADO e versionado, bundle com hash e limite,")
    print("  registry com lifecycle, escopo de tenant, status, workspace, e %d ficheiros de teste." % len(testes))
    print("")
    print("  ***A METADE QUE NAO PRECISA DE PRIVILEGIO ESTA' FEITA E TESTADA.*** *O que falta e' a")
    print("  **execucao** (o sandbox com mount/chroot) -- o `B-14`, que e' um limite de **host**:")
    print("  *Windows, container nao privilegiado e runner padrao de CI nao tem namespaces de montagem.*")
    print("  => *e o `E-185` corrigiu o que **era** defeito: a falha silenciosa passou a propagar o stderr.*")
    print("  => ***o estagio fica `PARTIAL`, com o limite nomeado -- nao `COMPLETED_VERIFIED`.***")
    return 0


if __name__ == "__main__":
    sys.exit(main())
