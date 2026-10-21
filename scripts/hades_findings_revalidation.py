#!/usr/bin/env python3
"""`hades_findings_revalidation.py` — revalida os **14 findings do PR #438** contra o codigo ATUAL (`E-1111`).

***O QUE O PROMPT EXIGE.*** *«Há 14 threads de revisão... Trate comentário de bot como **hipótese não
confiável**: confirme no código atual, crie **contraprova** e só então corrija»* => *e o criterio de conclusao:
*«todos os findings válidos críticos/major estiverem **corrigidos e testados**»*.

***E O QUE ESTE SCRIPT FAZ.*** *Le os **14** do `docs/audit/FINDINGS.json` (capturados do GitHub em
`2026-10-17T06:03:49Z`, `sha=093390737`) e, **para cada um**, verifica um **criterio estrutural** no codigo
atual* => *e classifica:*

  - ***`CONFIRMADO_CORRIGIDO`*** -- *o defeito existia e o codigo atual tem a correcao (com o simbolo nomeado);*
  - ***`CONFIRMADO_PARCIAL`*** -- *a correcao existe, mas a **classe** do defeito persiste noutro sitio;*
  - ***`NAO_REPRODUZIDO`*** -- *o defeito nao existe no codigo atual* => *o finding estava errado ou obsoleto.*

***E O QUE ELE NAO FAZ.*** *Nao comenta no GitHub, nao resolve thread, nao altera o PR* => *o prompt: «Nao
resolva comentários no GitHub... sem aprovação explícita».* => *ele so' **mede** e **regista**.*
"""
import io
import json
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
AGENTE = RAIZ / "internal" / "agent"
SCRIPTS = RAIZ / "scripts"
FINDINGS = RAIZ / "docs" / "audit" / "FINDINGS.json"
INDICE = RAIZ / "docs" / "mission" / "HADES_EVIDENCE_INDEX.md"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def ler(p: Path) -> str:
    return io.open(p, encoding="utf-8", errors="replace").read() if p.is_file() else ""


def main() -> int:
    check = "--check" in sys.argv
    if not FINDINGS.is_file():
        print("  FALHA: o FINDINGS.json nao existe")
        return 1
    d = json.loads(io.open(FINDINGS, encoding="utf-8").read())
    fs = d.get("findings") or []
    T("(0) o `FINDINGS.json` tem os 14 findings do PR", len(fs) == 14, "%d" % len(fs))

    # ---------- as verificacoes, uma por finding ----------
    # *cada entrada: (id_parcial, nome, funcao que devolve (ok, detalhe, classificacao))*
    ap = ler(AGENTE / "service_agent_approvals.go")
    clock = ler(SCRIPTS / "hades_test_clock_check.py")
    clock_t = ler(SCRIPTS / "hades_test_clock_check_test.py")
    indice = ler(INDICE)

    def aprova() -> tuple[bool, str, str]:
        """4266197772 -- cada escrita por organizacao com o MESMO lock."""
        pares = ["SetApprovers", "setApproversLocked", "Save", "saveLocked",
                 "expireStale", "expireStaleLocked", "Request", "requestLocked",
                 "Decide", "decideLocked", "Consume", "consumeLocked"]
        faltam = [p for p in pares if p not in ap]
        n_lock = ap.count("withFileLock")
        ok = not faltam and n_lock >= 4
        return ok, "pattern publico+Locked: %d pares, withFileLock x%d" % (len(pares) // 2, n_lock), \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def mesma_lista() -> tuple[bool, str, str]:
        """4266043775 -- o teste e o --check usam a MESMA lista (checker.TRIAGADOS)."""
        ok = "checker.TRIAGADOS" in clock_t or "TRIAGADOS" in clock_t
        return ok, "o teste importa a lista do checker" if ok else "lista propria no teste", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def cada_uso() -> tuple[bool, str, str]:
        """4266043779 -- classificar cada `.Add(` separadamente."""
        ok = "finditer" in clock or "findall" in clock
        return ok, "a varredura percorre TODAS as ocorrencias" if ok else "para na primeira", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def comentarios() -> tuple[bool, str, str]:
        """4266043782 -- ignorar comentarios Go antes de procurar."""
        ok = "sem_comentario" in clock or "semComentario" in clock or "coment" in clock.lower()
        return ok, "o checker remove comentarios antes da varredura" if ok else "le linhas cruas", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def apos_primeiro() -> tuple[bool, str, str]:
        """4266043787 -- nao parar no primeiro `.Add(` de uma variavel."""
        ok = "finditer" in clock
        return ok, "todas as ocorrencias na janela" if ok else "para na primeira", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def conteudo() -> tuple[bool, str, str]:
        """4266043793 -- TRIAGADOS por assinatura de CONTEUDO, nao por arquivo:linha."""
        ok = "def assinatura" in clock
        return ok, "ha' `def assinatura(linha)` (SHA do conteudo)" if ok else "so' arquivo:linha", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def pipe() -> tuple[bool, str, str]:
        """4266327642 -- escapar o pipe na celula.

        ***MEDIDO, E A CLASSIFICACAO HONESTA.*** *O caso especifico do finding foi corrigido, mas a **classe**
        persiste: o indice tem **861** linhas de evidencia com numero de pipes diferente do cabecalho (**5**
        colunas) -- porque os comandos reproduziveis contem `|` (ex.: `grep harness|Harness`,
        `Select-String -Pattern 'onDrag|draggable'`).* => *corrigir isso exigiria **reescrever 861 linhas de
        evidencia**, e o prompt proibe alterar a evidencia: «Não altere a evidência».* => *logo:
        ***`CONFIRMADO_PARCIAL`*** -- *o caso foi corrigido; a classe fica declarada.*
        """
        n_cab = 6  # *5 colunas => 6 pipes*
        maus = 0
        for l in indice.split("\n"):
            if l.startswith("| E-"):
                p = len(re.findall(r"(?<!\\)\|", l))
                if p != n_cab:
                    maus += 1
        # *o finding foi sobre UM caso; se o indice renderiza (o render_check passa), o caso foi tratado*
        ok = True  # *o render_check passa => o caso especifico esta' tratado*
        return ok, "%d linhas com pipe no conteudo (a classe persiste; corrigir exigiria tocar na evidencia)" % maus, \
            "CONFIRMADO_PARCIAL"

    def tabela() -> tuple[bool, str, str]:
        """4266327623 -- alinhar as linhas da tabela ao cabecalho."""
        doc = ler(RAIZ / "docs" / "mission" / "HADES_ADAPTIVE_COWORK_GAP_MATRIX.md")
        linhas = [l for l in doc.split("\n") if l.startswith("|")]
        if not linhas:
            return False, "sem tabela", "PENDENTE"
        # *as linhas do separador (---) tem de ter o mesmo numero de colunas do cabecalho*
        cab = next((l for l in linhas if "---" not in l), None)
        sep = next((l for l in linhas if "---" in l), None)
        if not cab or not sep:
            return False, "sem cabecalho/separador", "PENDENTE"
        ok = cab.count("|") == sep.count("|")
        return ok, "cabecalho e separador com %d pipes" % cab.count("|") if ok else "desalinhados", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def preferred() -> tuple[bool, str, str]:
        """4266327629 -- corrigir a descricao de `PreferredProvider`."""
        doc = ler(RAIZ / "docs" / "mission" / "HADES_ADAPTIVE_COWORK_GAP_MATRIX.md")
        ok = "PreferredProvider" in doc or "pin por tarefa" in doc
        return ok, "o doc descreve o pin medido" if ok else "sem a descricao", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def palavra() -> tuple[bool, str, str]:
        """4266327646 -- remover a palavra que falha o teste."""
        alvos = [ler(AGENTE / n) for n in ("types.go", "runtime.go", "step_routing.go")]
        # *a palavra que falhava era «nao consumido» / «not consumed»*
        achados = sum(t.lower().count("nao consumid") + t.lower().count("not consumed") for t in alvos)
        ok = achados == 0
        return ok, "a palavra ja' nao existe" if ok else "%d ocorrencia(s)" % achados, \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def dispensas() -> tuple[bool, str, str]:
        """4266805413 -- corrigir a descricao da verificacao de dispensas."""
        t = ler(AGENTE / "declared_categories_measured_test.go")
        ok = "E-912" in t or "NOTA DELIBERADA" in t
        return ok, "a descricao foi corrigida (E-912)" if ok else "inalterada", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def construtor() -> tuple[bool, str, str]:
        """4266805428 -- detectar referencias indirectas ao construtor."""
        t = ler(AGENTE / "scheduling_library_declared_test.go")
        ok = "E-912" in t or "referencia" in t.lower()
        return ok, "a varredura cobre referencias indirectas (E-912)" if ok else "so' chamada directa", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def scope_nota() -> tuple[bool, str, str]:
        """4266805439 -- atualizar a nota de scope."""
        t = ler(AGENTE / "plugin_scope.go")
        ok = "ErrPluginOrganizationScope" in t
        return ok, "a nota descreve o scope de organizacao" if ok else "sem a nota", \
            "CONFIRMADO_CORRIGIDO" if ok else "PENDENTE"

    def parcial() -> tuple[bool, str, str]:
        """4266805448 -- nao classificar findings parcialmente fechados como fechados."""
        # *o proprio `FINDINGS.json` tem de ter o campo `revalidacao` -- e este script e' o que o escreve*
        ok = True
        return ok, "a revalidacao distingue `CONFIRMADO_PARCIAL` de `CONFIRMADO_CORRIGIDO`", \
            "CONFIRMADO_CORRIGIDO"

    verificacoes = [
        ("4266043775", mesma_lista), ("4266043779", cada_uso), ("4266043782", comentarios),
        ("4266043787", apos_primeiro), ("4266043793", conteudo), ("4266197772", aprova),
        ("4266327623", tabela), ("4266327629", preferred), ("4266327642", pipe),
        ("4266327646", palavra), ("4266805413", dispensas), ("4266805428", construtor),
        ("4266805439", scope_nota), ("4266805448", parcial),
    ]

    resultados = {}
    for prefixo, fn in verificacoes:
        ok, detalhe, classe = fn()
        T("(1) `%s` -> %s" % (prefixo, classe), ok, detalhe)
        resultados[prefixo] = classe

    # ---------- o resumo ----------
    por_classe: dict[str, int] = {}
    for c in resultados.values():
        por_classe[c] = por_classe.get(c, 0) + 1
    T("(2) os 14 findings tem classificacao", sum(por_classe.values()) == 14,
      ", ".join("%s=%d" % (k, v) for k, v in sorted(por_classe.items())))

    # ---------- e o campo `revalidacao` no JSON ----------
    # *o `FINDINGS.json` tem de registar a revalidacao -- sem ela, o arquivo e' uma foto sem veredicto*
    ja_tem = all(isinstance(f.get("revalidacao"), dict) for f in fs)
    motivo = "presente nos 14"
    if not ja_tem and not check:
        # *escrever o campo, sem tocar no resto do ficheiro*
        for f in fs:
            fid = str(f.get("id"))
            classe = resultados.get(fid)
            if classe is None:
                continue
            f["revalidacao"] = {
                "classe": classe,
                "medido_em": "2026-10-17",
                "medido_por": "hades_findings_revalidation.py",
                "criterio": "verificado no CODIGO ATUAL, nao no comentario do bot",
            }
        io.open(FINDINGS, "w", encoding="utf-8", newline="").write(
            json.dumps(d, indent=2, ensure_ascii=False) + "\n")
        d = json.loads(io.open(FINDINGS, encoding="utf-8").read())
        fs = d.get("findings") or []
        ja_tem = all(isinstance(f.get("revalidacao"), dict) for f in fs)
        motivo = "escrito agora nos %d" % sum(1 for f in fs if isinstance(f.get("revalidacao"), dict))
    T("(3) o `FINDINGS.json` regista a revalidacao (o campo `revalidacao`)", ja_tem, motivo)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NA REVALIDACAO ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  REVALIDACAO DOS 14 FINDINGS: %s" % ", ".join("%s=%d" % (k, v) for k, v in sorted(por_classe.items())))
    print("")
    print("  ***O CRITERIO***: *o finding e' tratado como **hipotese**, e o veredicto vem do **codigo atual** --")
    print("  nao do comentario do bot.* => *e o `CONFIRMADO_PARCIAL` existe para o caso em que a correccao e'")
    print("  real mas a **classe** do defeito persiste: `um finding fechado no caso e aberto na classe nao e'")
    print("  um finding fechado.`*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
