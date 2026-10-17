#!/usr/bin/env python3
"""`hades_security_check.py` — o §ESTAGIO 18 (Seguranca, permissoes e multitenancy) (`E-1104`).

***O QUE O §ESTAGIO 18 EXIGE*** (do prompt, Dominio K — seguranca, governanca e confiabilidade):

    - **RBAC/ABAC, tenant isolation, autenticacao e sessoes**;
    - **vault/secrets, redaction, rotacao** e nenhuma credencial em chat/log;
    - **policy engine, approvals de uso unico, idempotencia e non-inference**;
    - **supply-chain**: lockfiles, SBOM, SCA e provenance;
    - **audit log** resistente a adulteracao;
    - **backup/restore, migracoes e rollback**;
    - **protecao contra prompt injection, tool poisoning** e conteudo externo nao confiavel;
    - **controles de custo, recursos, dados e privacidade**.

***O QUE ESTE CHECADOR MEDE.*** *Cada eixo tem de ter **teste** (nao comentario) e, quando ha' instrumento, ele
tem de estar ligado a um gate.* => *e o criterio do §ESTAGIO 18 e' que a seguranca **nao seja uma promessa**:
tem de ser um **teste que falha** quando a protecao cai.*
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
AGENTE = RAIZ / "internal" / "agent"
SERVER = RAIZ / "server"
SCRIPTS = RAIZ / "scripts"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def testes_que_casam(chaves: tuple[str, ...]) -> list[str]:
    out: list[str] = []
    for d in (AGENTE, SERVER):
        if not d.is_dir():
            continue
        for p in d.glob("*_test.go"):
            try:
                t = io.open(p, encoding="utf-8").read()
            except UnicodeDecodeError:
                continue
            nomes = re.findall(r"func (Test\w+)", t)
            if any(any(c in n for c in chaves) for n in nomes):
                out.append(p.name)
    return sorted(set(out))


def main() -> int:
    # ---------- 1. os eixos do Dominio K, cada um com teste ----------
    eixos = {
        # *o eixo `tenant` usa SO' `Tenant`/`tenant`: o `Scope` e o `Org` sao OUTROS eixos (RBAC e
        # organizacao), e misturá-los fazia o critério passar mesmo quando os testes de tenant eram apagados
        # (medido: 32 nomes de `Scope`/`Org` sobreviviam a' injecção).*
        "tenant isolation / RBAC": ("Tenant", "tenant"),
        "autenticacao e sessoes": ("Auth", "Session", "Bearer", "Token"),
        "secrets / redaction / DLP": ("Secret", "Redact", "DLP", "Dlp", "Mask"),
        "approvals de uso unico": ("Approval", "ApprovalSingle", "SingleUse"),
        "non-inference": ("NonInference", "Inference", "Heuristic"),
        "audit log": ("Audit", "Journal", "Trace"),
        "prompt injection / tool poisoning": ("Injection", "Poison", "Injecao", "Malicious"),
        "backup / restore / rollback": ("Backup", "Restore", "Rollback", "Snapshot"),
        "custos / quotas / recursos": ("Cost", "Budget", "Quota", "Limit"),
    }
    for nome, chaves in eixos.items():
        achados = testes_que_casam(chaves)
        T("(1) `%s` tem teste" % nome, len(achados) >= 1,
          "%d ficheiro(s), ex.: %s" % (len(achados), achados[0] if achados else "-"))

    # ---------- 2. os instrumentos EXISTEM e estao' LIGADOS ----------
    instrumentos = {
        "hades_tenant_scope_check.py": "o isolamento de tenant",
        "hades_non_inference_scan.py": "a non-inference",
        "hades_approval_hook_ratchet.py": "o encaixe da aprovacao",
        "hades_test_clock_check.py": "o relogio de teste (o tempo nao e' injectavel por engano)",
    }
    for arq, o_que in instrumentos.items():
        p = SCRIPTS / arq
        T("(2) o instrumento `%s` existe (%s)" % (arq, o_que), p.is_file(),
          "%.1f KB" % (p.stat().st_size / 1024) if p.is_file() else "AUSENTE")

    # ---------- 3. os instrumentos estao' na BATERIA ----------
    gates = SCRIPTS / "hades_local_gates.ps1"
    T("(3) a bateria local existe", gates.is_file())
    if gates.is_file():
        g = io.open(gates, encoding="utf-8").read()
        for arq in instrumentos:
            T("(3b) `%s` corre na bateria" % arq, arq in g,
              "ligado a um gate" if arq in g else "NAO ligado -- seria um instrumento orfao")

    # ---------- 4. a seguranca esta' DECLARADA no inventario de marca/limitacoes ----------
    lim = RAIZ / "docs" / "audit" / "LIMITACOES.md"
    T("(4) o `LIMITACOES.md` existe (as limitacoes declaradas)", lim.is_file())
    if lim.is_file():
        l = io.open(lim, encoding="utf-8").read()
        T("(4b) e declara limitacoes de seguranca",
          "segur" in l.lower() or "tenant" in l.lower(),
          "as limitacoes ficam escritas, nao escondidas")

    # ---------- 5. nenhum segredo no repositorio ----------
    # *procurar padroes de chave em ficheiros rastreados (sem imprimir valores)*
    suspeitos = 0
    padroes = (r"sk-[A-Za-z0-9]{20,}", r"ghp_[A-Za-z0-9]{30,}", r"AKIA[0-9A-Z]{16}",
               r"-----BEGIN [A-Z ]*PRIVATE KEY-----")
    # ***O SCAN E' SOBRE O CODIGO DE PRODUTO, NAO SOBRE OS INSTRUMENTOS.*** *O `scripts/` tem os
    # proprios checkers (e os testes deles) que contem **fixtures de segredo** -- incluir o `scripts/` no scan
    # fazia o detector acusar-se a si mesmo (medido: o `baseline` da copia falhou porque a copia incluia o
    # `_check.py` com o `sk-...` da injecao).*
    for base in (RAIZ / "internal", RAIZ / "server", RAIZ / "app" / "ui" / "app" / "src"):
        if not base.is_dir():
            continue
        for p in base.rglob("*"):
            if not p.is_file() or p.suffix in (".png", ".jpg", ".webp", ".zip", ".exe", ".gz"):
                continue
            # ***UM TESTE QUE USA UMA CHAVE FALSA NAO E' UM SEGREDO VAZADO: E' O CONTRARIO.***
            # *Medido: o detector acusou 10 -- `Bearer abcdefghijklmnop1234` (secrets_test.go), `const
            # secret = "..."` (runtime_test.go) -- todos **fixtures** que existem PARA testar o detector.*
            if p.name.endswith("_test.go"):
                continue
            if p.stat().st_size > 2_000_000:
                continue
            try:
                t = io.open(p, encoding="utf-8", errors="ignore").read()
            except OSError:
                continue
            for pad in padroes:
                if re.search(pad, t):
                    suspeitos += 1
                    print("    SUSPEITO (sem imprimir o valor): %s" % p.relative_to(RAIZ))
                    break
    T("(5) nenhum segredo em texto plano no codigo", suspeitos == 0,
      "%d suspeito(s)" % suspeitos)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 18 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 18 (Seguranca): os 9 eixos do Dominio K tem teste, os 4 instrumentos")
    print("  correm na bateria, as limitacoes estao declaradas, e nenhum segredo em texto plano.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
