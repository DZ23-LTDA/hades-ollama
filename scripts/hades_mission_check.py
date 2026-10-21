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
# "[e-agudo-duplo]", "Ã§"..., e travessao vira "â€". Documento assim e ilegivel e nao corresponde ao produto.
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
        # 🚨 E-778: ***UM BURACO DECLARADO NAO E' UM BURACO - E' DIVIDA.***
        # *A sequencia viva do projecto vai em `E-777` (cunhada em **codigo e commits**, nao no indice) enquanto o
        # indice registava ate' `E-570`* => *com a regra anterior, **contiguidade** (esta regra) e **nao-colisao**
        # (o `E-778`) eram **mutuamente exclusivas**: saltar para `E-778` criava **207** buracos, e nao saltar
        # reutilizava numeros **com dono*** ⇒ ***a cura nao e' escolher qual das duas regras se quebra: e' DECLARAR
        # a divida*** ⇒ *um ID ausente que o estado declara em `evidence_ids_nao_registrados` e' **divida
        # declarada**; o que continua a ser falha sao os ausentes **nao declarados**.*
        tolerados_buraco = set()
        for item in estado.get("evidence_ids_nao_registrados") or []:
            achado_num = re.search(r"(\d{1,4})", str(item))
            if achado_num:
                tolerados_buraco.add(int(achado_num.group(1)))
        # 🚨 E-778: *a divida da era paralela e' uma FAIXA FECHADA (`E-571..E-777`), declarada no estado* => *um
        # ausente **dentro** da faixa declarada nao e' «buraco»: e' **divida declarada**; um ausente **fora** dela
        # continua a FALHAR* - *assim a regra mantem o que serve (detectar linha apagada) e deixa de exigir o que o
        # projecto nao pode cumprir (contiguidade com uma sequencia cunhada fora do indice).*
        faixas_buraco: list[tuple[int, int]] = []
        for faixa in estado.get("evidence_ids_nao_registrados_faixas") or []:
            try:
                faixas_buraco.append((int(faixa[0]), int(faixa[1])))
            except (TypeError, ValueError, IndexError):
                continue

        def dentro_da_divida(numero: int) -> bool:
            return numero in tolerados_buraco or any(lo <= numero <= hi for lo, hi in faixas_buraco)

        esperado = list(range(1, max(numeros) + 1))
        faltando = sorted(n for n in esperado if n not in set(numeros) and not dentro_da_divida(n))
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
            # TEXTO COM errors="replace", NAO text=True.
            #
            # Medido: `text=True` manda o Python descodificar o stdout do filho como UTF-8 estrito
            # dentro da sua reader-thread. Quando o filho emite um byte cp1252 (0xe3 = "a" til), a
            # thread morre com UnicodeDecodeError, o TRACEBACK e' impresso no meio do relatorio, e a
            # saida do filho **perde-se** -- com o checker a terminar em exit 0. Um instrumento que
            # perde a saida do filho e continua "OK" nao esta' a medir: esta' a fingir.
            [sys.executable, str(RAIZ / "scripts/hades_release_gate.py"), "--check"],
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
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

    # **E-913 / finding 14 do bot, CONFIRMADO por leitura:** `PARCIALMENTE FECHADO` estava nesta lista e
    # **nao e' um estado terminal** -- o proprio texto do achado diz que a integracao de producao continua
    # aberta. => *um achado parcialmente fechado e' **meio aberto**, e a lista de ataques aponta-lo e'
    # **legitimo**: ha' trabalho por fazer.* => *inclui-lo aqui acusava de «duas verdades sobre o mesmo id»
    # um caso em que so' ha' **uma**: a integracao esta' pendente e o ataque continua na lista.*
    #
    # *A regra passa a ser: **so' estado TERMINAL encerra** -- o que nao deixa trabalho pendente.*
    encerrado = ("CORRIGIDO", "FECHADO", "RESOLVIDO", "RETRATADO", "ATENDIDO", "SEM DEFEITO")
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
        # 🚨 E-447: casamento por **PREFIXO**, nao por substring. *O meu proprio teste de controle pegou isto*: o estado
        # `"ABERTO: ainda nao corrigido"` **contem** `"CORRIGIDO"` ⇒ com `in` ele era acusado como encerrado ✗✓.
        # ***Substring sobre campo de texto livre e' casamento por acidente: "nao corrigido" casa com "CORRIGIDO".***
        # *O veredito e' o comeco da frase; o resto e' explicacao.*
        if estado_do_achado and any(
            estado_do_achado.strip().upper().startswith(marca) for marca in encerrado
        ):
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
    # ⚡ E-447: varredura RESTRITA as raizes de codigo. *O `rglob` no repositorio inteiro levava 245 s* ✗ - e um
    # verificador lento e' um verificador que as pessoas deixam de rodar. *As raizes estao declaradas, nao adivinhadas.*
    for sub in ("internal", "server", "cmd"):
        sub_raiz = raiz / sub
        if not sub_raiz.exists():
            continue
        for caminho in sub_raiz.rglob("*_test.go"):
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


def verificar_referencias_de_arquivo(raiz: Path, estado: dict) -> tuple[list[str], dict]:
    """🚨 E-451: toda referencia `arquivo.go:NNN` no estado tem de apontar para arquivo existente COM a linha dentro.

    **POR QUE ISTO EXISTE**: a varredura de E-446 provava que os **testes citados** existem, mas **nao** que as
    **referencias de arquivo** existem - *duas conferencias diferentes, e a segunda faltava*. Ao fazê-la, ela achou em
    **30** referencias **uma** que apontava para o vazio: `audit_test.go:65`, abreviada por mim na E-439 quando o arquivo
    e' `service_agent_audit_test.go`.

    ***Referencia para linha inexistente e' prova DESLOCADA: a frase continua legitima e o lastro aponta para o vazio.***

    **O QUE ELA NAO FAZ**: nao prova que a linha **contem** o que o achado diz ✗ - *isso exigiria julgamento, e inventar
    heuristica aqui produziria alarme falso (E-432/E-434/E-447)*. E **nao** confere referencias que citam apenas **nome de
    funcao** ✗.

    ⚡ *O indice de nomes e' montado UMA vez, FORA do laco* - a primeira versao da varredura tinha `rglob` dentro do
    laco e ficou **O(n^2)** sobre 1346 arquivos, a correr em background. ***Instrumento lento e' instrumento que nao se
    roda (E-447).***
    """
    problemas: list[str] = []
    auditoria = estado.get("v22", {}).get("auditoria_adversarial", {})
    texto = json.dumps(auditoria, ensure_ascii=False)
    referencias = re.findall(r"([A-Za-z0-9_./-]+\.go):(\d+)(?:-(\d+))?", texto)
    if not referencias:
        return problemas, {"referencias_de_arquivo": 0, "referencias_deslocadas": 0}

    # INDICE UMA VEZ. *Fora do laco - a licao de E-451.*
    por_nome: dict[str, list[Path]] = {}
    for sub in ("internal", "server", "cmd", "scripts"):
        sub_raiz = raiz / sub
        if not sub_raiz.exists():
            continue
        for caminho in sub_raiz.rglob("*.go"):
            if any(parte in {"_tmp", "vendor", ".git"} for parte in caminho.parts):
                continue
            por_nome.setdefault(caminho.name, []).append(caminho)

    deslocadas = []
    for arquivo, inicio, fim in referencias:
        candidatos = por_nome.get(Path(arquivo).name, [])
        if not candidatos:
            deslocadas.append("%s:%s (arquivo nao existe)" % (arquivo, inicio))
            continue
        alvo = int(fim or inicio)
        if not any(alvo <= len(caminho.read_text(encoding="utf-8", errors="replace").split("\n")) for caminho in candidatos):
            deslocadas.append("%s:%s (linha alem do fim do arquivo)" % (arquivo, inicio))

    for deslocada in sorted(set(deslocadas)):
        problemas.append(
            "REFERENCIA DESLOCADA: %s — *a frase continua legitima e o lastro aponta para o vazio*" % deslocada
        )
    return problemas, {"referencias_de_arquivo": len(referencias), "referencias_deslocadas": len(set(deslocadas))}


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


def verificar_frescura_dos_documentos_de_retomada(
    raiz: Path, ultimo_id: int, estado: dict, tolerancia: int = 40
) -> tuple[list[str], dict]:
    """Confere que os documentos de RETOMADA mencionam evidência recente (E-544 / E-548).

    **A DÍVIDA, MEDIDA.** *A FASE 7 manda actualizar «continuamente» CINCO documentos, e o `E-544` mediu que TRÊS
    estavam parados há ~16 h — e que este checker **não os lia**: `grep HANDOFF|PENDING` devolvia vazio. ⇒ Três
    documentos avisavam, **eles próprios**, que «os números envelhecem» … e os três envelheceram.*

    ***A REGRA NÃO ERA O PROBLEMA: FALTAVA QUEM A VERIFICASSE.*** *Este passo é essa metade em falta — a segunda
    natureza da correcção que o `E-544` declarou e o `E-548` registou como **não feita**.*

    **O CRITÉRIO, E PORQUE É ESTE.** Não se exige o número *exacto* de evidências: ele muda a cada registo, e um
    documento é actualizado a cada poucas dezenas. Exige-se que mencione **alguma** evidência dos últimos
    `tolerancia` registos. ⇒ *Um documento parado há 62 rodadas (*o `NEXT_ACTION` dizia «rodada 39»/«353»*) não
    menciona nenhum `E-5xx` recente, e é **isso** que se detecta. Um documento fresco menciona pelo menos um.*

    *Devolve `(problemas, resumo)`, como as outras verificações deste ficheiro.*
    """
    # 🚨 E-556: ***A LISTA VEM DO ESTADO, NÃO DO SCRIPT.*** *Até aqui era literal, e por isso a lista do `E-549` e a
    # do `E-555` eram **afirmações sem dono** sobre o que existe.* ⇒ *A medição do `E-556` mostrou que a cura **não**
    # podia ser derivar (o estado **não** tinha chave nenhuma) — *tinha de ser **declarar**: `checkpoint_documents`.*
    #
    # *A lista literal fica como **retrocompatibilidade**: se a chave faltar, o passo usa-a e **declara** que caiu nela.*
    nomes = (estado.get("checkpoint_documents") or {}).get("retomada") or []
    origem = "estado"
    if not nomes:
        nomes = ["HADES_HANDOFF.md", "HADES_PENDING_WORK.md", "HADES_NEXT_ACTION.md"]
        origem = "literal (retrocompatibilidade: o estado nao declara checkpoint_documents)"
    caminhos = [(nome, raiz / "docs/mission" / nome) for nome in nomes]
    piso = max(1, ultimo_id - tolerancia)
    problemas: list[str] = []
    desactualizados: list[str] = []
    for nome, caminho in caminhos:
        if not caminho.exists():
            problemas.append(f"documento de retomada AUSENTE: {nome}")
            continue
        texto = caminho.read_text(encoding="utf-8")
        # *Qualquer `E-NNN` citado, seja na forma `E-547` ou `E-0547`.*
        citados = [int(m) for m in re.findall(r"\bE-?(\d{3,4})\b", texto)]
        recentes = [n for n in citados if n >= piso]
        if not recentes:
            maior = max(citados) if citados else 0
            desactualizados.append(nome)
            problemas.append(
                "documento de retomada DESACTUALIZADO: "
                f"{nome} nao cita nenhuma evidencia >= E-{piso:03d} "
                f"(a mais recente que cita e' E-{maior:03d}) - *um documento de retomada que so' fala do passado "
                "nao serve para retomar*"
            )
    return problemas, {
        "retomada_desactualizados": len(desactualizados),
        "retomada_piso": piso,
        "retomada_lista_de": origem,
    }


def verificar_procedencia_dos_derivados(raiz: Path, estado: dict) -> tuple[list[str], dict]:
    """Confere que os relatórios DERIVADOS citam o commit medido (E-552 / E-554).

    **A DÍVIDA, MEDIDA.** *A FASE 3 manda regerar os derivados «com fonte de verdade única e **identificação explícita
    do commit medido**».* ⇒ *O `E-552` mediu que **nenhum dos três** o fazia: o `RELEASE_REPORT` tinha **0** menções, o
    `FINAL_RELEASE_GATE` citação nenhuma, e o `BASELINE_REPORT` citava **a `main`** — *o que não está errado (*ele mede
    o baseline, e o baseline é da `main`*), mas era **ambíguo**.*

    ⇒ *O `E-554` corrigiu o **conteúdo** (os três geradores).* ⇒ ***E este passo é a segunda metade: **quem verifica**.***
    *Sem ele, um quarto derivado pode nascer amanhã sem procedência — e nada o diria.*

    **O CRITÉRIO.** Exige-se que cada derivado **mencione o `measured_commit` do estado**. Não se exige *qual* campo
    cita (*isso é a decisão em aberto*), exige-se que o valor **esteja escrito**: *um relatório de release sem o commit
    medido não se pode **reproduzir** nem **contestar** — e um número que não se sabe a que objecto pertence é
    indistinguível de um número inventado.*

    *Devolve `(problemas, resumo)`, como as outras verificações deste ficheiro.*
    """
    medido = (estado.get("measured_commit") or "").strip()
    # 🚨 E-556: ***A LISTA VEM DO ESTADO.*** *Mesma correcção do passo 13: o dono da lista é o estado, não o script.*
    nomes_derivados = (estado.get("checkpoint_documents") or {}).get("derivados") or []
    origem_derivados = "estado"
    if not nomes_derivados:
        nomes_derivados = ["HADES_RELEASE_REPORT.md", "HADES_FINAL_RELEASE_GATE.md", "HADES_BASELINE_REPORT.md"]
        origem_derivados = "literal (retrocompatibilidade: o estado nao declara checkpoint_documents)"
    derivados = [(nome, raiz / "docs/mission" / nome) for nome in nomes_derivados]
    problemas: list[str] = []
    sem_procedencia: list[str] = []
    if not medido:
        return (
            ["o estado nao declara `measured_commit`: nenhum derivado pode ser conferido contra nada"],
            {"derivados_sem_procedencia": 0, "derivados_medido": "NAO_MEDIDO"},
        )
    for nome, caminho in derivados:
        if not caminho.exists():
            problemas.append(f"derivado AUSENTE: {nome}")
            continue
        if medido not in caminho.read_text(encoding="utf-8"):
            sem_procedencia.append(nome)
            problemas.append(
                "derivado SEM PROCEDENCIA: "
                f"{nome} nao cita o commit medido `{medido}` - *um relatorio derivado sem o commit medido nao se pode "
                "reproduzir nem contestar*"
            )
    return problemas, {
        "derivados_sem_procedencia": len(sem_procedencia),
        "derivados_medido": medido,
        "derivados_lista_de": origem_derivados,
    }


def _fontes_de_codigo(raiz: Path) -> list[tuple[str, str]]:
    """Devolve `(caminho, texto)` dos ficheiros de CODIGO que citam algum `E-NNN` (E-780).

    *Uma so' travessia, partilhada pelos passos **15** (existencia) e **16** (cabeca)* - *em vez de duas copias
    do mesmo percurso, que divergiriam com o tempo: e' a licao do `E-550` («duas copias de um numero sao uma
    coincidencia a espera de divergir») aplicada ao **codigo** em vez de a um numero.*

    *Custo medido: **0,52 s** sobre **410** ficheiros com ID.*
    """
    podar = {".git", "node_modules", "_tmp", "dist", "build", "vendor", ".next", "coverage", ".venv", "__pycache__"}
    exts = {".go", ".ts", ".tsx", ".js", ".mjs", ".py", ".ps1", ".sh"}
    padrao = re.compile(r"\bE-?(\d{1,4})\b")
    fontes: list[tuple[str, str]] = []
    pilha = [raiz]
    while pilha:
        atual = pilha.pop()
        try:
            entradas = list(atual.iterdir())
        except OSError:
            continue
        for entrada in entradas:
            if entrada.is_dir():
                if entrada.name not in podar:
                    pilha.append(entrada)
                continue
            if entrada.suffix not in exts:
                continue
            try:
                conteudo = entrada.read_text(encoding="utf-8", errors="replace")
            except OSError:
                continue
            if padrao.search(conteudo):
                try:
                    relativo = str(entrada.relative_to(raiz))
                except ValueError:
                    relativo = str(entrada)
                fontes.append((relativo, conteudo))
    return fontes

def verificar_evidencia_citada_registada(raiz: Path, estado: dict, indice_texto: str) -> tuple[list[str], dict]:
    """Confere que toda evidência CITADA existe no índice (E-567 / E-568 / E-569).

    **A DÍVIDA, MEDIDA.** *O `E-568` mediu, por arqueologia do Git, que a cabeça **committada** do índice é `E-559`
    enquanto **94** commits cunharam `(E-6xx)` — e o maior ID cunhado é `E-672`.* ⇒ *e o `E-569` mediu que o
    `HADES_MASTER_STATE.json` registra **alguns** deles no seu `evidence_log` (**1** entrada, **6** IDs), com
    **interseção ZERO** com o índice.* ⇒ ***dois registos declarados como FONTE de verdade, e nenhum instrumento que
    os cruze.***

    **O CRITÉRIO.** Os passos **11** (`E-446`, testes citados) e **12** (`E-451`, `arquivo.go:NNN`) já conferem
    citações **do estado** — e **não** conferiam os **IDs de evidência**. *Este passo fecha essa família:* ***toda
    evidência CITADA — no estado e nos documentos de retomada — tem de existir no índice.***

    **A CATRACA.** *Com `E-618..E-669` cunhados e não registados, ligar a verificação sem catraca deixaria o gate
    **vermelho por dívida pré-existente** — e «um gate vermelho por dívida conhecida, **sem catraca**, não é uma
    correção: é um alarme novo»* ⇒ *os IDs tolerados vêm do **estado** (`evidence_ids_nao_registrados`), com
    **fallback literal declarado**, no molde do `E-556`*. ⇒ ***a lista só pode DIMINUIR: um ID novo citado e não
    registado FALHA.***

    *Devolve `(problemas, resumo)`, como as outras verificações deste ficheiro.*
    """

    def normaliza(valor: object) -> int | None:
        achado = re.search(r"(\d{1,4})", str(valor))
        return int(achado.group(1)) if achado else None

    # 🚨 E-556: ***A LISTA VEM DO ESTADO, NÃO DO SCRIPT.***
    tolerados: set[int] = set()
    for item in estado.get("evidence_ids_nao_registrados") or []:
        numero = normaliza(item)
        if numero is not None:
            tolerados.add(numero)
    # 🚨 E-778: *a era paralela cunhou uma FAIXA inteira (`E-571..E-777`) sem a registar* => *a divida e'
    # declarada como **faixa fechada**, nao como lista de 200 numeros* - *e um ID FORA da faixa continua a falhar.*
    faixas: list[tuple[int, int]] = []
    for faixa in estado.get("evidence_ids_nao_registrados_faixas") or []:
        try:
            faixas.append((int(faixa[0]), int(faixa[1])))
        except (TypeError, ValueError, IndexError):
            continue

    def tolerado(numero: int) -> bool:
        return numero in tolerados or any(lo <= numero <= hi for lo, hi in faixas)

    origem = "estado"
    if not tolerados:
        tolerados = {618, 619, 636, 642, 650, 651, 653, 654, 655, 656, 669, 0, 999}
        origem = "literal (retrocompatibilidade: o estado nao declara evidence_ids_nao_registrados)"

    registados = {int(m) for m in re.findall(r"^\|\s*E-(\d+)\s*\|", indice_texto, re.M)}

    nomes = (estado.get("checkpoint_documents") or {}).get("retomada") or []
    if not nomes:
        nomes = ["HADES_HANDOFF.md", "HADES_PENDING_WORK.md", "HADES_NEXT_ACTION.md"]
    fontes: list[tuple[str, str]] = []
    for nome in nomes:
        caminho = raiz / "docs" / "mission" / nome
        if caminho.exists():
            fontes.append((nome, caminho.read_text(encoding="utf-8")))
    # *O estado tambem cita — e' FONTE: as citacoes dele nao tinham conferencia de ID nenhuma.*
    fontes.append(("HADES_MASTER_STATE.json", json.dumps(estado, ensure_ascii=False)))

    # 🚨 E-780: ***E FALTAVA O CODIGO.*** *O `E-778` mediu **64** IDs citados em codigo que o indice
    # nao tem, e nenhum instrumento os via: o passo **16** olha ao **maximo**, nao a **existencia**.* *A
    # extensao foi **adiada** enquanto a numeracao nao estava regularizada (faixa declarada + cabeca
    # corrigida); agora esta', e sem ela o codigo - onde vivem **221+** referencias `E-6xx` - ficaria **sem**
    # conferencia de existencia para sempre.*
    fontes.extend(_fontes_de_codigo(raiz))

    problemas: list[str] = []
    faltas: set[int] = set()
    for nome, texto in fontes:
        citados = {int(m) for m in re.findall(r"\bE-?(\d{1,4})\b", texto)}
        faltam = sorted(n for n in citados if n not in registados and not tolerado(n))
        if faltam:
            faltas.update(faltam)
            problemas.append(
                f"evidencia CITADA e NAO REGISTADA em {nome}: "
                + ", ".join(f"E-{n:03d}" for n in faltam)
                + " - *uma citacao que nao existe no indice nao e' verificavel por quem audita: registe-a (medindo) "
                "ou acrescente-a a catraca declarada `evidence_ids_nao_registrados`*"
            )
    return problemas, {
        "evidencia_citada_nao_registrada": len(faltas),
        "evidencia_catraca": len(tolerados),
        "evidencia_catraca_de": origem,
    }


def verificar_cabeca_do_indice(raiz: Path, indice_texto: str) -> tuple[list[str], dict]:
    """Confere que a CABEÇA do índice alcança o maior ID cunhado no CÓDIGO (E-779).

    **O DEFEITO QUE ISTO APANHA — E QUE ACONTECEU COMIGO.** *O `E-778` mediu que **7** das minhas entradas
    `E-560..E-570` **colidiam** com trabalho committado*: *eu vi o índice terminar em `E-559` e comecei a numerar
    ali — mas a **sequência viva** do projecto já ia em `E-672` e, medido em qualquer ficheiro, em **`E-777`***.
    ⇒ *Nenhum instrumento dizia «este número já existe»* ⇒ ***este passo é esse instrumento: compara a **cabeça do
    índice** com o **maior ID citado no código**, e falha quando a cabeça fica **atrás**.***

    **PORQUE O CÓDIGO, E NÃO AS MENSAGENS DE COMMIT.** *O `E-568` propôs comparar com as mensagens de commit — mas
    um `git log` num clone **shallow** devolveria um máximo **menor**, e a verificação passaria **vaziamente**: é a
    mesma armadilha que o `E-561` mediu no CI. O **código** está sempre no checkout, custa **0,52 s** (medido sobre
    **410** ficheiros) e é onde a era paralela deixou as suas referências.*

    *Devolve `(problemas, resumo)`, como as outras verificações deste ficheiro.*
    """
    # 🚨 E-779: ***PLACEHOLDER NAO E' EVIDENCIA CUNHADA.*** *O `E-999` e o `E-000` sao **valores de exemplo** dos
    # proprios testes e da documentacao do verificador* => *sem os excluir, o maior ID «citado no codigo» seria
    # sempre **999** e este passo seria um **falso positivo permanente***. *A lista vem do **estado** (doutrina `E-556`).*
    placeholders: set[int] = set()
    try:
        estado_local = json.loads((raiz / "docs" / "mission" / "HADES_MASTER_STATE.json").read_text(encoding="utf-8"))
        for item in estado_local.get("evidence_ids_placeholder") or []:
            achado = re.search(r"(\d{1,4})", str(item))
            if achado:
                placeholders.add(int(achado.group(1)))
    except (OSError, ValueError):
        placeholders = {0, 999}
    if not placeholders:
        placeholders = {0, 999}
    maior = 0
    ficheiros = 0
    for _relativo, texto in _fontes_de_codigo(raiz):
        achados = [
            int(m) for m in re.findall(r"\bE-?(\d{1,4})\b", texto) if int(m) not in placeholders
        ]
        if achados:
            ficheiros += 1
            maior = max(maior, max(achados))
    registados = [int(m) for m in re.findall(r"^\|\s*E-(\d+)\s*\|", indice_texto, re.M)]
    cabeca = max(registados) if registados else 0
    problemas: list[str] = []
    if maior and cabeca and maior > cabeca:
        problemas.append(
            f"CABECA DO INDICE ATRAS DO CODIGO: o maior ID citado no codigo e' E-{maior:03d} e o indice registava "
            f"apenas ate' E-{cabeca:03d} - *um ID novo tem de continuar do **maior cunhado**, e nao da cabeca do "
            "indice: numerar onde o indice acaba REUTILIZA numeros com dono - foi exactamente a colisao do `E-778`*"
        )
    return problemas, {
        "cabeca_do_indice": cabeca,
        "maior_id_no_codigo": maior,
        "ficheiros_de_codigo_com_id": ficheiros,
    }


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

    # 12. 🚨 E-451: toda referencia `arquivo.go:NNN` tem de apontar para arquivo existente COM a linha dentro.
    #
    # A varredura de E-446 provava que os **testes citados** existem, mas **nao** que as **referencias de arquivo**
    # existem. *Duas conferencias diferentes, e a segunda faltava* - e ao fazê-la ela achou `audit_test.go:65` apontando
    # para o vazio (*abreviacao minha na E-439*).
    problemas_referencias, resumo_referencias = verificar_referencias_de_arquivo(raiz_do_repositorio, estado)
    problemas += problemas_referencias
    resumo.update(resumo_referencias)

    # 13. 🚨 E-544 / E-548: os TRES documentos de RETOMADA mencionam evidencia RECENTE.
    #
    # **MEDIDO (E-544)**: dos CINCO documentos que a FASE 7 manda actualizar «continuamente», TRÊS estavam em dia e
    # DOIS estavam **parados há ~16 h** — e filtrando o `HANDOFF` por commit, evidência e «evid», o resultado era
    # **ZERO linhas**. *E o `NEXT_ACTION` estava **62 rodadas** atrás.*
    #
    # ***E ESTE CHECKER NÃO OS LIA*** — *`grep HANDOFF|PENDING` no próprio ficheiro devolvia vazio*. ⇒ *Os três
    # documentos avisavam, eles próprios, que «os números envelhecem» … e envelheceram.*
    # ***A regra não era o problema: faltava QUEM A VERIFICASSE.*** *Este passo é a segunda natureza da correcção
    # que o `E-544` declarou e o `E-548` registou como NÃO feita.*
    problemas_retomada, resumo_retomada = verificar_frescura_dos_documentos_de_retomada(
        raiz_do_repositorio, resumo["ultimo_id"], estado
    )
    problemas += problemas_retomada
    resumo.update(resumo_retomada)

    # 14. 🚨 E-552 / E-554: os DERIVADOS citam o COMMIT MEDIDO.
    #
    # *A FASE 3 manda regerar os derivados «com fonte de verdade única e **identificação explícita do commit medido**».*
    # ⇒ *O `E-552` mediu que **nenhum dos três** o fazia: 0 menções no `RELEASE_REPORT`, nenhuma citação no
    # `FINAL_RELEASE_GATE`, e o `BASELINE_REPORT` citava a **`main`** — *ambíguo, porque «commit de referência» não diz
    # **se** é o medido ou o comparado*.*
    #
    # ***E é o PAR DO PASSO 13***: *o 13 (`E-549`) verifica a **frescura** dos documentos de retomada; **este** verifica
    # a **procedência** dos derivados* ⇒ ***«frescura diz **quando** foi medido; procedência diz **sobre o quê**»*** — e
    # *um relatório pode estar fresco e não dizer sobre que commit fala* ✓✓.
    #
    # *Sem este passo, um quarto derivado nasce amanhã sem procedência — e nada o diria.*
    problemas_derivados, resumo_derivados = verificar_procedencia_dos_derivados(raiz_do_repositorio, estado)
    problemas += problemas_derivados
    resumo.update(resumo_derivados)

    # 15. 🚨 E-567 / E-568 / E-569: TODA EVIDENCIA CITADA TEM DE EXISTIR NO INDICE.
    #
    # *Os passos **11** (`E-446`: todo **teste** citado existe) e **12** (`E-451`: toda referencia `arquivo.go:NNN`
    # existe) conferem citacoes **do estado** — e **nao** conferiam os **IDs de evidencia**.* ⇒ ***e foi por isso que o
    # `E-568` pode medir que o indice esta' ~113 IDs atras do proprio historico de commits COM O GATE VERDE.***
    #
    # *A familia fica fechada aqui, com catraca declarada no estado (`E-556`): um ID novo citado e nao registado
    # FALHA; a faixa `E-618..E-669`, cunhada e nao registada, fica tolerada e so' pode DIMINUIR.*
    problemas_ids_citados, resumo_ids_citados = verificar_evidencia_citada_registada(
        raiz_do_repositorio, estado, indice
    )
    problemas += problemas_ids_citados
    resumo.update(resumo_ids_citados)

    # 16. 🚨 E-779: A CABECA DO INDICE TEM DE ALCANCAR O MAIOR ID CUNHADO NO CODIGO.
    #
    # *O `E-568` propôs este instrumento e eu NAO o construí - e foi por isso que o `E-778` teve de existir:* *eu vi
    # o índice terminar em `E-559` e numerei `E-560..E-570` ali, enquanto a **sequência viva** ia em `E-777`* =>
    # ***sete dos meus onze IDs colidiram com trabalho committado***.
    #
    # *A verificacao olha ao **codigo** (nao a mensagens de commit, que um clone **shallow** tornaria menores) e a
    # cabeca do indice tem de estar **a frente** do maior ID citado.* *Custo medido: **0,52 s** sobre **410** ficheiros.*
    problemas_cabeca, resumo_cabeca = verificar_cabeca_do_indice(raiz_do_repositorio, indice)
    problemas += problemas_cabeca
    resumo.update(resumo_cabeca)

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
