#!/usr/bin/env python3
"""Confere a ESCRITURAÇÃO da missão: nenhuma afirmação sem lastro (E-180).

Por que este script existe
--------------------------
A seção 3 das diretrizes exige que toda capacidade promovida tenha **evidência anexada ao índice**. Isso só vale
se alguém conferir: um estágio pode citar `E-999` que nunca existiu, dois estágios podem disputar o mesmo ID, um
bloqueio pode ser referenciado sem estar declarado — e nada disso apareceria como falha. Enquanto o índice de
evidências for mantido à mão (e ele é, por ser prosa), o mínimo é **verificar a estrutura dele por máquina**.

O que é conferido
-----------------
1. `HADES_EVIDENCE_INDEX.md`: linhas `| E-NNN |` com IDs **únicos** e **sem buracos** a partir de E-001.
2. Toda evidência citada por um estágio (`state.stages[].evidence`) **existe** no índice.
3. Todo `blocker` citado por um estágio (`state.stages[].blocker`) **existe** na lista de bloqueios.
4. `execution` de cada estágio está no conjunto permitido pela seção 7, e nenhum estágio ficou sem o campo.
5. O veredito global é permitido e coerente: `COMPLETED_VERIFIED` exige que **nenhum** requisito da matriz esteja
   pendente — o script falha se alguém declarar release completo com item em aberto.
6. O gate gerado continua consistente (`scripts/hades_release_gate.py --check`).

Uso
---
    python scripts/hades_mission_check.py
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ESTADO_PADRAO = RAIZ / "docs/mission/HADES_MASTER_STATE.json"
INDICE_PADRAO = RAIZ / "docs/mission/HADES_EVIDENCE_INDEX.md"
RELATORIO_PADRAO = RAIZ / "docs/mission/HADES_STAGE_REPORT.md"
BLOQUEIOS_PADRAO = RAIZ / "docs/mission/HADES_BLOCKERS.md"

# Declaracao explicita de que o vinculo de estagio NAO foi registrado (linhas legadas do indice).
# Sem ela, a linha precisa ser citada por algum estagio: evidencia orfa silenciosa e registro falso.
MARCADOR_SEM_VINCULO = "VINCULO_NAO_DECLARADO"

EXECUCOES_PERMITIDAS = {
    "NOT_STARTED",
    "IN_PROGRESS",
    "TESTING",
    "FAILED",
    "BLOCKED_BY_EXTERNAL_DEPENDENCY",
    "COMPLETED_VERIFIED",
}
VEREDITOS_PERMITIDOS = {"NOT_RELEASE_READY", "RELEASE_CANDIDATE", "COMPLETED_VERIFIED"}


# EVIDENCIAS OBRIGATORIAS DA SECAO 5 QUE NOMEIAM DOCUMENTO (E-186).
#
# A secao 5 lista, por estagio, "Evidencias obrigatorias". Varias descrevem CATEGORIAS (logs, testes, metricas) e nao
# tem caminho fixo; as que NOMEIAM um documento entram aqui, porque um documento nomeado e um artefato que pode
# existir ou nao - e enquanto ninguem confere, "obrigatorio" e decoracao. Esta tabela e a parte da secao 5 que a
# checagem consegue provar.
DOCUMENTOS_OBRIGATORIOS_POR_ESTAGIO: dict[int, tuple[str, ...]] = {
    1: (
        "audit/HADES_MASTER_AUDIT.md",
        "audit/HADES_CAPABILITY_MATRIX.md",
        "audit/HADES_RELEASE_BLOCKERS.md",
        "docs/mission/HADES_STAGE_MAP.md",
    ),
    2: (
        "docs/mission/HADES_BASELINE_REPORT.md",
        "docs/mission/HADES_FAILURE_INDEX.md",
    ),
    22: ("docs/mission/HADES_ENGINEERING_GATE.md",),
}


def verificar_documentos_obrigatorios(raiz: Path) -> tuple[list[str], dict]:
    """Confere os documentos que a secao 5 nomeia como evidencia OBRIGATORIA de cada estagio.

    Recebe `raiz` para ser TESTAVEL com fixture: um diretorio temporario prova o caminho de falha sem depender do
    repositorio real.
    """
    problemas: list[str] = []
    conferidos = 0
    for estagio, caminhos in sorted(DOCUMENTOS_OBRIGATORIOS_POR_ESTAGIO.items()):
        for caminho in caminhos:
            conferidos += 1
            if not (raiz / caminho).is_file():
                problemas.append(
                    f"evidencia OBRIGATORIA da secao 5 ausente: estagio {estagio} exige `{caminho}`"
                )
    return problemas, {"documentos_obrigatorios_conferidos": conferidos}


def verificar_mapa_de_estagios(estado: dict) -> tuple[list[str], list[str], dict]:
    """Confere o mapa entre os estágios TRACKADOS e os estágios do prompt master (E-187).

    Duas naturezas de achado, de propósito:

    - **problema** (falha o build): o mapa está MECANICAMENTE incompleto — estágio do estado sem linha, linha para
      estágio que não existe. Isso é erro de escrituração e tem conserto objetivo.
    - **aviso** (aparece, não falha): dívida de GOVERNANÇA — estágio do master sem cobertura, coberto duas vezes, ou
      com escopo divergente. Aqui falhar não conserta nada: reconciliar exige decidir se o estado muda de numeração,
      e renumerar 38 estágios muda o significado das evidências acumuladas. Esconder também não: por isso o número
      aparece no resumo e no documento gerado.
    """
    problemas: list[str] = []
    avisos: list[str] = []
    resumo: dict = {}
    try:
        import hades_stage_map as mapa
    except ImportError as erro:
        return [], [f"mapa de estágios indisponível para conferência: {erro}"], {}

    ids_do_estado = {e.get("id") for e in estado.get("stages", [])}
    ids_do_mapa = {linha[0] for linha in mapa.MAPA}
    for ident in sorted(ids_do_estado - ids_do_mapa):
        problemas.append(f"estágio {ident} do estado NÃO tem linha no mapa de estágios (scripts/hades_stage_map.py)")
    # O sentido inverso (linha no mapa sem estágio no estado) NÃO é problema: o mapa é uma tabela estática que
    # cobre 1..38 por construção, e fixtures de teste usam subconjuntos do estado. A garantia de que o mapa cobre
    # exatamente 1..38 pertence ao TESTE do próprio mapa, não a uma checagem que roda contra qualquer estado.
    if ids_do_mapa != set(range(1, 39)):
        problemas.append(
            "mapa de estágios precisa cobrir exatamente 1..38; cobre "
            + f"{len(ids_do_mapa)} estágios"
        )

    contagem = mapa.situacoes()
    sem_cobertura = mapa.mestres_nao_cobertos()
    duplicados = mapa.mestres_cobertos_mais_de_uma_vez()
    divergentes = contagem.get(mapa.ESCOPO_DIVERGENTE, 0) + contagem.get(mapa.SEM_PAR_NO_MESTRE, 0)
    if sem_cobertura:
        avisos.append(
            "estágios do MASTER sem nenhum estágio do estado apontando: "
            + ", ".join(str(i) for i in sem_cobertura)
        )
    if duplicados:
        avisos.append(
            "estágios do master apontados por MAIS DE UM estágio do estado: "
            + ", ".join(f"{i} (x{c})" for i, c in duplicados.items())
        )
    if divergentes:
        avisos.append(f"{divergentes} estágios do estado têm escopo divergente do master (dívida de governança declarada)")
    # Quantas dessas divergências já têm DECISÃO registrada (o procedimento do mapa exige decisão por estágio)?
    # A dívida continua VISÍVEL (o aviso acima permanece); o que se acrescenta é o placar do que falta decidir.
    def _obs(linha: object) -> str:
        try:
            return str(linha[4])  # (id, titulo, masters, situacao, observacao)
        except (TypeError, IndexError):
            return ""

    divergentes_do_mapa = [linha for linha in mapa.MAPA if _obs(linha).startswith("DECISAO_")]
    cobertos = sum(1 for linha in divergentes_do_mapa if _obs(linha).startswith("DECISAO_A"))
    lacunas = sum(1 for linha in divergentes_do_mapa if _obs(linha).startswith("DECISAO_B"))
    sem_decisao = divergentes - len(divergentes_do_mapa)
    avisos.append(
        f"decisões por estágio divergente: {cobertos} cobertos (DECISAO_A), {lacunas} lacunas declaradas "
        f"(DECISAO_B), {sem_decisao} SEM decisão — de {divergentes} divergências"
    )
    if lacunas:
        avisos.append(
            f"lacunas declaradas (escopo do master SEM dono no estado): {lacunas} — ver HADES_STAGE_MAP.md seção do mapa"
        )
    resumo = {
        "estagios_sem_cobertura_no_master": len(sem_cobertura),
        "estagios_master_com_cobertura_dupla": len(duplicados),
        "estagios_com_escopo_divergente": divergentes,
    }
    return problemas, avisos, resumo



# Marcas de texto DUPLA-Codificado (UTF-8 lido como ANSI e regravado como UTF-8). Portugues acentuado vira
# "Ã©", "Ã§"..., e travessao vira "â€". Documento assim e ilegivel e nao corresponde ao produto.
MOJIBAKE = re.compile(r"Ã[©¡³§µºª¢­‰°¼½¾]|â€|Ãƒ")


def verificar_mojibake(raiz: Path) -> tuple[list[str], dict]:
    """Acusa documento com texto duplamente codificado. Devolve (problemas, resumo)."""
    problemas: list[str] = []
    alvos = sorted((raiz / "docs" / "mission").glob("*.md")) + sorted((raiz / "audit").glob("*.md"))
    varridos = 0
    for caminho in alvos:
        try:
            conteudo = caminho.read_text(encoding="utf-8")
        except (UnicodeDecodeError, OSError):
            problemas.append(f"documento ilegivel em UTF-8: {caminho.relative_to(raiz)}")
            continue
        varridos += 1
        achados = MOJIBAKE.findall(conteudo)
        if achados:
            problemas.append(
                f"documento com TEXTO CORROMPIDO (dupla codificacao): {caminho.relative_to(raiz)} "
                f"({len(achados)} ocorrencias, ex.: {achados[0]!r})"
            )
    return problemas, {"documentos_varridos": varridos}

def verificar(
    estado_path: Path,
    indice_path: Path,
    conferir_gate: bool = True,
    raiz: Path | None = None,
) -> tuple[list[str], list[str], dict]:
    """Devolve (problemas, avisos, resumo). Aceita caminhos para ser TESTÁVEL com fixtures."""
    problemas: list[str] = []
    avisos: list[str] = []

    estado = json.loads(estado_path.read_text(encoding="utf-8"))
    indice = indice_path.read_text(encoding="utf-8")

    # 1. IDs do índice.
    ids = re.findall(r"^\|\s*(E-\d+)\s*\|", indice, re.M)
    if not ids:
        problemas.append("índice de evidências sem nenhuma linha `| E-NNN |`")
    duplicados = sorted({i for i in ids if ids.count(i) > 1})
    if duplicados:
        problemas.append(f"IDs de evidência DUPLICADOS no índice: {', '.join(duplicados)}")
    numeros = sorted(int(i.split("-")[1]) for i in ids)
    if numeros:
        esperado = list(range(1, max(numeros) + 1))
        faltando = sorted(set(esperado) - set(numeros))
        if faltando:
            problemas.append(
                "buracos na numeração do índice (IDs ausentes): "
                + ", ".join(f"E-{n:03d}" for n in faltando[:12])
                + ("..." if len(faltando) > 12 else "")
            )
    conjunto = set(ids)

    # 2 e 3. Referências dos estágios.
    ids_de_bloqueio = {b.get("id") for b in estado.get("blockers", [])}
    for estagio in estado["stages"]:
        ident = estagio.get("id")
        for evidencia in estagio.get("evidence") or []:
            if evidencia not in conjunto:
                problemas.append(f"estágio {ident} cita evidência que NÃO existe no índice: {evidencia}")
        bloqueio = estagio.get("blocker")
        if bloqueio and bloqueio not in ids_de_bloqueio:
            problemas.append(f"estágio {ident} cita bloqueio que NÃO está declarado: {bloqueio}")


    # 5. Vínculo de estágio (o mestre exige "estágio e tarefa associados" em cada evidência).
    citadas_por_estagio = {e for st in estado["stages"] for e in (st.get("evidence") or [])}
    sem_vinculo: list[str] = []
    for linha in indice.splitlines():
        casamento = re.match(r"^\|\s*(E-\d+)\s*\|", linha)
        if not casamento:
            continue
        ident = casamento.group(1)
        if ident not in citadas_por_estagio and MARCADOR_SEM_VINCULO not in linha:
            sem_vinculo.append(ident)
    if sem_vinculo:
        problemas.append(
            "evidências SEM vínculo de estágio e SEM a declaração "
            f"{MARCADOR_SEM_VINCULO}: "
            + ", ".join(sem_vinculo[:12])
            + ("..." if len(sem_vinculo) > 12 else "")
        )

    # 4. Campos de execução/estado.
    for estagio in estado["stages"]:
        ident = estagio.get("id")
        execucao = estagio.get("execution")
        if not execucao:
            problemas.append(f"estágio {ident} SEM o campo `execution` exigido pela seção 7")
        elif execucao not in EXECUCOES_PERMITIDAS:
            problemas.append(f"estágio {ident} tem `execution` fora do permitido: {execucao}")
        if not estagio.get("state"):
            avisos.append(f"estágio {ident} sem `state` legível")

    # 4b. Pelo menos uma evidencia por estagio: "Evidencias obrigatorias" sem nenhuma evidencia nao e cobertura.
    for estagio in estado["stages"]:
        if not (estagio.get("evidence") or []):
            problemas.append(
                f"estágio {estagio.get('id')} nao declara NENHUMA evidencia (a seção 5 exige evidências obrigatórias)"
            )

    # 4c. Documentos que a seção 5 nomeia como evidência obrigatória.
    if raiz is not None:
        problemas_documentos, resumo_documentos = verificar_documentos_obrigatorios(raiz)
        problemas.extend(problemas_documentos)
    else:
        resumo_documentos = {"documentos_obrigatorios_conferidos": 0}

    # 4d. Mapa entre os estágios trackados e os estágios do prompt master (E-187).
    problemas_mapa, avisos_mapa, resumo_mapa = verificar_mapa_de_estagios(estado)
    problemas.extend(problemas_mapa)
    avisos.extend(avisos_mapa)

    # 5. Veredito coerente com o estado.
    veredito = estado.get("release_state")
    if veredito not in VEREDITOS_PERMITIDOS:
        problemas.append(f"`release_state` fora do permitido: {veredito}")
    if veredito == "COMPLETED_VERIFIED":
        pendentes = [
            e.get("id")
            for e in estado["stages"]
            if e.get("execution") not in ("COMPLETED_VERIFIED",)
        ]
        if pendentes:
            problemas.append(
                "veredito COMPLETED_VERIFIED com estágios fora de COMPLETED_VERIFIED: "
                + ", ".join(str(p) for p in pendentes[:12])
            )
        externos = [b for b in estado.get("blockers", []) if "EXTERNO" in (b.get("tipo") or "").upper()]
        if externos:
            problemas.append(
                "veredito COMPLETED_VERIFIED com bloqueio externo em aberto: "
                + ", ".join(str(b.get("id")) for b in externos)
            )

    # 6. Gate gerado consistente.
    if conferir_gate:
        resultado = subprocess.run(
            [sys.executable, str(RAIZ / "scripts/hades_release_gate.py"), "--check"],
            capture_output=True,
            text=True,
        )
        if resultado.returncode != 0:
            problemas.append("gate gerado DIVERGENTE do estado: " + (resultado.stdout or "").strip())

    resumo = {
        "evidencias": len(ids),
        "ultimo_id": max(numeros) if numeros else 0,
        "estagios": len(estado["stages"]),
        "bloqueios": len(ids_de_bloqueio),
        "veredito": veredito,
        **resumo_documentos,
    **resumo_mapa,
}
    return problemas, avisos, resumo


def verificar_relatorio(relatorio_path: Path, conjunto_ids: set[str]) -> tuple[list[str], dict]:
    """Confere a NUMERAÇÃO do relatório de estágios e as evidências que ele cita (E-181).

    O relatório é prosa, mas a numeração é ÍNDICE: item duplicado ou faltando torna impossível citar uma rodada
    sem ambiguidade. E uma evidência citada no relatório que não existe no índice é afirmação sem lastro.
    """
    problemas: list[str] = []
    texto = relatorio_path.read_text(encoding="utf-8")
    linhas = texto.splitlines()

    numeros: list[int] = []
    for linha in linhas:
        casamento = re.match(r"^(\d+)\. \*\*", linha)
        if casamento:
            numeros.append(int(casamento.group(1)))

    if not numeros:
        problemas.append("relatório de estágios sem nenhum item numerado (`N. **`)")
        return problemas, {"itens": 0, "ultimo": 0}

    duplicados = sorted({n for n in numeros if numeros.count(n) > 1})
    if duplicados:
        problemas.append(
            "itens DUPLICADOS no relatório de estágios: "
            + ", ".join(str(n) for n in duplicados)
            + " (numeração ambígua: duas rodadas com o mesmo índice)"
        )
    buracos = sorted(set(range(1, max(numeros) + 1)) - set(numeros))
    if buracos:
        problemas.append(
            "buracos na numeração do relatório (itens ausentes): "
            + ", ".join(str(n) for n in buracos[:12])
            + ("..." if len(buracos) > 12 else "")
        )

    # ID entre crases e MENC AO (exemplo/identificador), nao CITACAO: no proprio relatorio isso acontece quando
    # um item explica o que o verificador pega e cita "E-999" como exemplo. Sem esta regra, a explicacao vira
    # falso positivo - foi o que a primeira execucao mostrou.
    CRASE = chr(96)
    sem_crase = re.sub(CRASE + r"[^" + CRASE + r"]*" + CRASE, " ", texto)
    citadas = sorted(set(re.findall(r"\bE-\d+\b", sem_crase)))
    fantasmas = [e for e in citadas if e not in conjunto_ids]
    if fantasmas:
        problemas.append(
            "relatório cita evidências que NÃO existem no índice: " + ", ".join(fantasmas[:12])
        )
    return problemas, {"itens": len(numeros), "ultimo": max(numeros), "citadas": len(citadas)}


def verificar_cruzamento_de_auditoria(estado: dict) -> tuple[list[str], dict]:
    """🚨 E-443: cruza `proximos_ataques` com `achados` - o cruzamento que faltava.

    **MEDIDO**: dos 4 itens em `proximos_ataques`, TRÊS já tinham achado correspondente marcado CORRIGIDO/FECHADO
    (ADV-60, ADV-63, ADV-64). A lista de ataques **duplicava** a lista ao lado, e ninguém cruzou as duas ⇒ *conferir
    item a item funcionaria, mas **o dado já estava lá***.

    *Uma lista que só cresce e nunca é relida não é backlog: é arquivo morto com aparência de plano.*

    A regra: um id **não pode** estar em `proximos_ataques` **e** em `achados` com estado que o próprio registro dá por
    encerrado. Se está, ou a lista está vencida, ou o achado foi reaberto sem retirar o ataque — *nos dois casos há duas
    verdades sobre o mesmo id, e nenhuma autoridade.*
    """
    problemas: list[str] = []
    auditoria = estado.get("v22", {}).get("auditoria_adversarial", {})
    ataques = auditoria.get("proximos_ataques", [])

    encerrado = ("CORRIGIDO", "FECHADO", "RESOLVIDO", "RETRATADO", "ATENDIDO", "SEM DEFEITO", "PARCIALMENTE FECHADO")
    por_id: dict[str, str] = {}
    for achado in auditoria.get("achados", []):
        identificador = str(achado.get("id", "")).split(" ")[0].strip()
        if identificador:
            por_id[identificador] = str(achado.get("estado", ""))

    duplicados = []
    for item in ataques:
        # O id do item pode vir com sufixo (`"ADV-60 (P0)"`) - *foi a falta desta normalização que produziu o registro
        # falso de E-443, onde o ADV-60 recebeu a conferência do ADV-62.*
        bruto = item.get("id", "") if isinstance(item, dict) else str(item)
        identificador = str(bruto).split(":")[0].split(" ")[0].strip()
        if not identificador:
            continue
        estado_do_achado = por_id.get(identificador)
        if estado_do_achado and any(marca in estado_do_achado.upper() for marca in encerrado):
            duplicados.append("%s (achado diz: %s)" % (identificador, estado_do_achado[:70]))

    for duplicado in duplicados:
        problemas.append(
            "ATaque E ACHADO AO MESMO TEMPO: %s — *ou a lista de ataques está VENCIDA, ou o achado foi reaberto sem "
            "retirar o ataque. Duas verdades sobre o mesmo id, e nenhuma autoridade*" % duplicado
        )

    return problemas, {"proximos_ataques": len(ataques), "ataques_duplicando_achado_encerrado": len(duplicados)}


def verificar_testes_citados(raiz: Path, estado: dict) -> tuple[list[str], dict]:
    """🚨 E-446: todo teste CITADO no estado tem de EXISTIR no codigo.

    **POR QUE ISTO EXISTE**: em sete rodadas de auditoria, **7 de 7** itens de lista estavam vencidos, e **3** deles
    citavam testes que **existiam com outro nome** (E-442, E-443, E-444). *Um achado que diz "CORRIGIDO: TestX" e cujo
    `TestX` nao existe mais nao esta' corrigido - esta' **com a prova podre**, e ninguem percebe porque a frase continua
    legitima.*

    ***A mesma licao de E-432 (instrumento que nunca dispara) e E-434 (validacao que recusa tudo), do lado da EVIDENCIA:
    prova citada que nao existe e' afirmacao sem lastro com aparencia de lastro.***

    **O QUE ESTA INVARIANTE NAO FAZ**: ela **nao** prova que o teste cita passa, nem que ele cobre o que o achado diz ✗
    - so' prova que o NOME existe. *Nome existente com corpo vazio seria aprovado aqui, e e' por isso que a conferencia
    humana continua necessaria.*
    """
    problemas: list[str] = []
    auditoria = estado.get("v22", {}).get("auditoria_adversarial", {})
    texto_do_estado = json.dumps(auditoria, ensure_ascii=False)

    citados = set(re.findall(r"\bTest[A-Z][A-Za-z0-9_]{3,}\b", texto_do_estado))
    if not citados:
        return problemas, {"testes_citados": 0, "testes_citados_inexistentes": 0}

    definidos: set[str] = set()
    for caminho in raiz.rglob("*_test.go"):
        partes = set(caminho.parts)
        if "_tmp" in partes or "vendor" in partes or ".git" in partes:
            continue
        try:
            conteudo = caminho.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        definidos.update(re.findall(r"func (Test[A-Za-z0-9_]{3,})\s*\(", conteudo))

    inexistentes = sorted(citados - definidos)
    for citado in inexistentes:
        problemas.append(
            "TESTE CITADO QUE NAO EXISTE: %s — *um achado que cita um teste inexistente tem a PROVA PODRE: a frase "
            "continua legitima e o lastro sumiu*" % citado
        )
    return problemas, {"testes_citados": len(citados), "testes_citados_inexistentes": len(inexistentes)}


def verificar_bloqueios(bloqueios_path: Path, ids_do_estado: set[str]) -> tuple[list[str], dict]:
    """Confere o documento de bloqueios contra o estado (E-181), nos DOIS sentidos.

    Bloqueio declarado e ausente do documento é impedimento que ninguém lê; linha no documento sem bloqueio no
    estado é fantasma. Os dois casos são divergência silenciosa.
    """
    problemas: list[str] = []
    texto = bloqueios_path.read_text(encoding="utf-8")
    no_documento = set(re.findall(r"\|\s*(B-\d+)\s*\|", texto))

    so_no_estado = sorted(ids_do_estado - no_documento)
    if so_no_estado:
        problemas.append(
            "bloqueios declarados no ESTADO e AUSENTES de HADES_BLOCKERS.md: " + ", ".join(so_no_estado)
        )
    so_no_documento = sorted(no_documento - ids_do_estado)
    if so_no_documento:
        problemas.append(
            "bloqueios listados em HADES_BLOCKERS.md e INEXISTENTES no estado: " + ", ".join(so_no_documento)
        )
    return problemas, {"no_documento": len(no_documento), "no_estado": len(ids_do_estado)}


MARCADORES_DE_CONFLITO = ("<<<<<<<", ">>>>>>>")


def verificar_sem_marcadores_de_conflito(caminhos: list[Path]) -> list[str]:
    """Reprova documento da missão que contenha marcador de merge não resolvido.

    MOTIVO (E-201): eu commitei um `HADES_MASTER_STATE.json` com marcadores de conflito e quem pegou foi o CI, por
    acidente — o `json.loads` estourou. O checker **passava**, porque ele valida estrutura, numeração e citações, não o
    texto bruto: um índice com `<<<<<<<` era considerado "consistente". Marcador de conflito é falha dura: significa
    que duas versões da escrituração foram misturadas sem decisão.
    """
    problemas: list[str] = []
    for caminho in caminhos:
        if not caminho.exists():
            continue
        texto = caminho.read_text(encoding="utf-8", errors="replace")
        encontrados = [m for m in MARCADORES_DE_CONFLITO if m in texto]
        if encontrados:
            problemas.append(f"{caminho.name} tem marcador de conflito de merge não resolvido: {' '.join(encontrados)}")
    return problemas


def main() -> int:
    parser = argparse.ArgumentParser(description="Confere a escrituração da missão.")
    parser.add_argument("--state", type=Path, default=ESTADO_PADRAO)
    parser.add_argument("--index", type=Path, default=INDICE_PADRAO)
    parser.add_argument("--report", type=Path, default=RELATORIO_PADRAO)
    parser.add_argument("--blockers", type=Path, default=BLOQUEIOS_PADRAO)
    parser.add_argument("--no-gate-check", action="store_true", help="não confere o gate gerado")
    args = parser.parse_args()

    raiz_do_repositorio = Path(__file__).resolve().parent.parent
    problemas, avisos, resumo = verificar(args.state, args.index, conferir_gate=not args.no_gate_check, raiz=raiz_do_repositorio)

    # 7. Relatório de estágios (numeração + evidências citadas).
    estado = json.loads(args.state.read_text(encoding="utf-8"))
    indice = args.index.read_text(encoding="utf-8")
    conjunto_ids = set(re.findall(r"^\|\s*(E-\d+)\s*\|", indice, re.M))
    problemas_mojibake, resumo_mojibake = verificar_mojibake(RAIZ)
    problemas += problemas_mojibake
    resumo.update(resumo_mojibake)
    problemas_relatorio, resumo_relatorio = verificar_relatorio(args.report, conjunto_ids)
    problemas += problemas_relatorio

    # 8. Documento de bloqueios ⇄ estado (nos dois sentidos).
    ids_do_estado = {b.get("id") for b in estado.get("blockers", [])}
    problemas_bloqueios, resumo_bloqueios = verificar_bloqueios(args.blockers, ids_do_estado)
    problemas += problemas_bloqueios

    # 10. 🚨 E-443: CRUZAMENTO ENTRE AS DUAS LISTAS DE AUDITORIA.
    #
    # **MEDIDO**: dos 4 itens em `proximos_ataques`, **TRÊS** já tinham achado correspondente em `achados` marcado
    # `CORRIGIDO`/`FECHADO` ✗ ⇒ *a lista de ataques **duplicava** a lista ao lado, e ninguém cruzou as duas.* Somando
    # os retratados antes, **6 de 7** itens de lista estavam vencidos ou parciais.
    #
    # ***Conferir item a item funcionaria — mas o dado já estava lá. O que faltava era o CRUZAMENTO.*** *Uma lista que
    # só cresce e nunca é relida não é backlog: é arquivo morto com aparência de plano.*
    problemas_cruzamento, resumo_cruzamento = verificar_cruzamento_de_auditoria(estado)
    problemas += problemas_cruzamento
    resumo.update(resumo_cruzamento)

    # 11. 🚨 E-446: TODO TESTE CITADO NO ESTADO TEM DE EXISTIR.
    #
    # Em sete rodadas, **7 de 7** itens de lista estavam vencidos, e **3** citavam testes que existiam **com outro
    # nome** (E-442/443/444). *Um achado que diz "CORRIGIDO: TestX" e cujo `TestX` não existe mais não está corrigido —
    # está com a **prova podre**, e ninguém percebe porque a frase continua legítima.*
    problemas_citados, resumo_citados = verificar_testes_citados(raiz_do_repositorio, estado)
    problemas += problemas_citados
    resumo.update(resumo_citados)

    # 9. Nenhum documento da escrituração (nem os gerados) pode conter marcador de merge não resolvido.
    problemas += verificar_sem_marcadores_de_conflito(
        [
            args.state,
            args.index,
            args.report,
            args.blockers,
            RAIZ / "docs/mission" / "HADES_FINAL_RELEASE_GATE.md",
            RAIZ / "docs/mission" / "HADES_RELEASE_REPORT.md",
            RAIZ / "docs/mission" / "HADES_BASELINE_REPORT.md",
            RAIZ / "docs/mission" / "HADES_FAILURE_INDEX.md",
            RAIZ / "docs/mission" / "HADES_STAGE_MAP.md",
        ]
    )

    print(f"índice: {resumo['evidencias']} evidências (E-001..E-{resumo['ultimo_id']:03d})")
    print(f"estágios: {resumo['estagios']} · bloqueios declarados: {resumo['bloqueios']}")
    print(f"relatório: {resumo_relatorio['itens']} itens (até {resumo_relatorio['ultimo']}) · "
          f"{resumo_relatorio.get('citadas', 0)} evidências citadas")
    print(f"bloqueios no documento: {resumo_bloqueios['no_documento']}")
    print(f"veredito: {resumo['veredito']}")
    if avisos:
        print("avisos:")
        for aviso in avisos:
            print("  -", aviso)
    if problemas:
        print("PROBLEMAS:")
        for problema in problemas:
            print("  -", problema)
        return 1
    print("ESCRITURAÇÃO CONSISTENTE: nenhuma evidência fantasma, nenhum bloqueio órfão, veredito coerente.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
