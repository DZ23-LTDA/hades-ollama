#!/usr/bin/env python3
"""Teste automático de CRONOLOGIA e ENUMERAÇÃO do estado canônico — V2.2, seção 2.

O QUE ESTE TESTE PEGA (defeitos REAIS medidos em E-311, antes da correção):
  - `updated_at` **161 horas no FUTURO** (2026-10-19 contra a hora real 2026-10-13);
  - `git.main_sha` de **2026-10-09**, cerca de 30 PRs atrás do HEAD;
  - **duas fontes de estado** (`stage_states` e `stages[].state`) divergindo em **16 estágios** — inclusive um caso de
    `IN_PROGRESS` contra `COMPLETED_VERIFIED`.

***Um documento de estado que não pode ser contradito por um teste é uma narrativa, não um estado.***

As direções negativas são testadas com fixtures: o teste não vale se só souber dizer "ok" ao repositório real.
"""

from __future__ import annotations

import json
import re
import sys
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ESTADO = RAIZ / "docs/mission/HADES_MASTER_STATE.json"
INDICE = RAIZ / "docs/mission/HADES_EVIDENCE_INDEX.md"

ESTADOS_CANONICOS = {
    "NOT_STARTED",
    "IN_PROGRESS",
    "TESTING",
    "COMPLETED_VERIFIED",
    "FAILED",
    "BLOCKED_BY_EXTERNAL_DEPENDENCY",
}
RELEASE_STATES = {"NOT_RELEASE_READY", "RELEASE_READY", "RELEASE_CANDIDATE"}

# E-959 (FASE 7.1/7.5): a faixa histórica ausente é DECLARADA no índice e lida daqui. O padrão vive
# numa constante para que o teste negativo use a MESMA fonte — allowlist divergente é dívida nova.
# Formato exigido no `HADES_EVIDENCE_INDEX.md`:  FAIXA HISTORICA AUSENTE: E-571..E-777
FAIXA_HISTORICA_AUSENTE = re.compile(
    r"(?m)^[^\n]*FAIXA\s+HIST[OÓ]RICA\s+AUSENTE[^\n]*?E-(\d+)\s*\.\.\s*E-(\d+)",
    re.IGNORECASE,
)


def _agora() -> datetime:
    return datetime.now(timezone.utc)


def _parse(instante: str) -> datetime:
    """Interpreta um timestamp ISO. Exige timezone explícita: sem ela, não há cronologia comparável."""
    valor = datetime.fromisoformat(instante.replace("Z", "+00:00"))
    if valor.tzinfo is None:
        raise ValueError(f"timestamp sem timezone: {instante!r}")
    return valor


def auditar_cronologia(estado: dict, agora: datetime | None = None) -> list[str]:
    """Devolve as violações de cronologia e de campos medidos."""
    agora = agora or _agora()
    violacoes: list[str] = []

    for campo in ("updated_at", "measured_at"):
        valor = estado.get(campo)
        if not valor:
            violacoes.append(f"{campo} ausente (V2.2 seção 2 exige a marca de tempo real)")
            continue
        try:
            quando = _parse(valor)
        except ValueError as erro:
            violacoes.append(f"{campo}: {erro}")
            continue
        # Tolerância pequena: o arquivo é escrito e o teste roda em seguida.
        if quando > agora + timedelta(minutes=5):
            violacoes.append(f"{campo} está NO FUTURO: {valor} (agora {agora.isoformat(timespec='seconds')})")

    if not estado.get("measured_commit"):
        violacoes.append("measured_commit ausente: sem ele não se sabe QUAL revisão foi medida")
    return violacoes


def auditar_duas_fontes(estado: dict) -> list[str]:
    """`stage_states` precisa ser DERIVADO de `stages[].state` — duas fontes mentem sozinhas."""
    violacoes: list[str] = []
    derivado: dict[str, list[int]] = {}
    for e in estado.get("stages", []):
        derivado.setdefault(e.get("state", ""), []).append(e.get("id"))
    for chave, ids in derivado.items():
        if chave not in ESTADOS_CANONICOS:
            violacoes.append(f"estado fora do enum canônico em stages[]: {chave!r}")
        declarado = sorted((estado.get("stage_states") or {}).get(chave, []))
        if declarado != sorted(i for i in ids if i is not None):
            violacoes.append(f"stage_states[{chave}] divergente de stages[].state: {declarado} != {sorted(ids)}")
    return violacoes


def auditar_enum_release(estado: dict) -> list[str]:
    """O release global precisa estar no enum e ser coerente com a existência de bloqueios."""
    violacoes: list[str] = []
    release = estado.get("release_state")
    if release not in RELEASE_STATES:
        violacoes.append(f"release_state fora do enum: {release!r}")
    bloqueados = [e.get("id") for e in estado.get("stages", []) if e.get("state") == "BLOCKED_BY_EXTERNAL_DEPENDENCY"]
    if bloqueados and release == "RELEASE_READY":
        violacoes.append(f"release_state RELEASE_READY com estágios bloqueados: {bloqueados}")
    return violacoes


def auditar_evidence_ids(conteudo: str) -> list[str]:
    """E-IDs únicos e contíguos — numeração com buraco ou repetida torna o índice não auditável.

    E-959 (FASE 7.1): a faixa `E-571..E-777` **não existe** e não pode ser inventada. Ela passou a
    ser VISÍVEL no CI no momento em que as evidências E-560+ foram commitadas: antes, o maior ID do
    índice era ~E-570 e o `max(ids)` ficava **abaixo** de 571, então o buraco não aparecia. Isso é
    exatamente o que o prompt descreve — *«um auditor do GitHub não as vê»* — e o guarda estava
    certo em disparar.

    A cura **não** é preencher 207 textos fictícios (proibido), nem afrouxar a contiguidade. É
    **declarar** a faixa no próprio índice, de forma legível por máquina, e exigir que ela continue
    vazia: qualquer ID novo dentro de uma faixa declarada é violação, para que a dívida histórica
    nunca vire licença para reutilizar números (FASE 7.6).
    """
    violacoes: list[str] = []
    ids = [int(m.group(1)) for m in re.finditer(r"(?m)^\| E-(\d+) ", conteudo)]
    if not ids:
        return ["nenhuma evidência encontrada no índice"]
    repetidos = {i for i in ids if ids.count(i) > 1}
    if repetidos:
        violacoes.append(f"E-IDs repetidos: {sorted(repetidos)}")

    faixas_declaradas = [
        (int(a), int(b)) for a, b in FAIXA_HISTORICA_AUSENTE.findall(conteudo)
    ]
    # Uma faixa declarada que VOLTOU a ter IDs e' reutilizacao de numero perdido: falha sempre.
    for inicio, fim in faixas_declaradas:
        invadidos = sorted(i for i in set(ids) if inicio <= i <= fim)
        if invadidos:
            violacoes.append(
                "IDs dentro de faixa declarada como ausente (E-%d..E-%d): %s"
                % (inicio, fim, invadidos)
            )

    def declarado(i: int) -> bool:
        return any(inicio <= i <= fim for inicio, fim in faixas_declaradas)

    faltando = [
        i for i in sorted(set(range(min(ids), max(ids) + 1)) - set(ids)) if not declarado(i)
    ]
    if faltando:
        violacoes.append(f"buraco na numeração de evidências: {faltando}")
    return violacoes


class CronologiaDoEstadoTest(unittest.TestCase):
    # ---- o repositório REAL: é este o teste que o CI executa ----

    def test_estado_real_tem_cronologia_e_enum_coerentes(self) -> None:
        if not ESTADO.exists():
            self.skipTest("estado canônico ausente neste checkout")
        estado = json.loads(ESTADO.read_text(encoding="utf-8"))
        violacoes = (
            auditar_cronologia(estado)
            + auditar_duas_fontes(estado)
            + auditar_enum_release(estado)
            + auditar_evidence_ids(INDICE.read_text(encoding="utf-8"))
        )
        self.assertEqual([], violacoes, "estado inconsistente:\n" + "\n".join(violacoes))

    # ---- direções NEGATIVAS: um guarda que só sabe dizer "ok" não guarda nada ----

    def test_timestamp_no_futuro_e_acusado(self) -> None:
        futuro = (_agora() + timedelta(hours=160)).isoformat(timespec="seconds")
        violacoes = auditar_cronologia({"updated_at": futuro, "measured_at": futuro, "measured_commit": "abc1234"})
        self.assertTrue(any("NO FUTURO" in v for v in violacoes), violacoes)

    def test_timestamp_sem_timezone_e_acusado(self) -> None:
        violacoes = auditar_cronologia({"updated_at": "2026-10-13T04:43:14", "measured_at": "2026-10-13T04:43:14+00:00", "measured_commit": "abc1234"})
        self.assertTrue(any("sem timezone" in v for v in violacoes), violacoes)

    def test_measured_commit_ausente_e_acusado(self) -> None:
        agora = _agora().isoformat(timespec="seconds")
        violacoes = auditar_cronologia({"updated_at": agora, "measured_at": agora})
        self.assertTrue(any("measured_commit ausente" in v for v in violacoes), violacoes)

    def test_duas_fontes_divergentes_sao_acusadas(self) -> None:
        estado = {
            "stages": [{"id": 13, "state": "COMPLETED_VERIFIED"}],
            "stage_states": {"COMPLETED_VERIFIED": [], "IN_PROGRESS": [13]},
        }
        violacoes = auditar_duas_fontes(estado)
        self.assertTrue(any("divergente" in v for v in violacoes), violacoes)

    def test_estado_fora_do_enum_e_acusado(self) -> None:
        violacoes = auditar_duas_fontes({"stages": [{"id": 1, "state": "QUASE_PRONTO"}], "stage_states": {}})
        self.assertTrue(any("fora do enum" in v for v in violacoes), violacoes)

    def test_release_ready_com_bloqueio_e_acusado(self) -> None:
        estado = {"release_state": "RELEASE_READY", "stages": [{"id": 5, "state": "BLOCKED_BY_EXTERNAL_DEPENDENCY"}]}
        violacoes = auditar_enum_release(estado)
        self.assertTrue(any("RELEASE_READY com estágios bloqueados" in v for v in violacoes), violacoes)

    def test_buraco_e_repeticao_de_evidence_id_sao_acusados(self) -> None:
        self.assertTrue(auditar_evidence_ids("| E-1 | a |\n| E-3 | c |\n"))
        self.assertTrue(auditar_evidence_ids("| E-1 | a |\n| E-1 | b |\n"))

    # ---- E-959: a faixa histórica DECLARADA não é buraco; a mesma faixa NÃO declarada é. ----

    def test_faixa_declarada_como_ausente_nao_e_buraco(self) -> None:
        conteudo = (
            "> FAIXA HISTORICA AUSENTE: E-2..E-4 (EVIDENCE_LOST_HISTORICAL)\n"
            "| E-1 | a |\n| E-5 | b |\n"
        )
        self.assertEqual([], auditar_evidence_ids(conteudo))

    def test_mesma_faixa_sem_declaracao_continua_sendo_buraco(self) -> None:
        # O controle: sem a declaração, o MESMO conteúdo é acusado. Sem isto, o teste acima
        # poderia estar a passar por o auditor não ver buraco nenhum.
        self.assertTrue(auditar_evidence_ids("| E-1 | a |\n| E-5 | b |\n"))

    def test_id_novo_dentro_da_faixa_declarada_e_acusado(self) -> None:
        # FASE 7.6: a dívida declarada NÃO pode virar licença para reutilizar números perdidos.
        conteudo = (
            "> FAIXA HISTORICA AUSENTE: E-2..E-4 (EVIDENCE_LOST_HISTORICAL)\n"
            "| E-1 | a |\n| E-3 | REUTILIZADO |\n| E-5 | b |\n"
        )
        violacoes = auditar_evidence_ids(conteudo)
        self.assertTrue(any("faixa declarada como ausente" in v for v in violacoes), violacoes)

    def test_buraco_fora_da_faixa_declarada_e_acusado(self) -> None:
        conteudo = (
            "> FAIXA HISTORICA AUSENTE: E-2..E-4 (EVIDENCE_LOST_HISTORICAL)\n"
            "| E-1 | a |\n| E-5 | b |\n| E-9 | c |\n"
        )
        violacoes = auditar_evidence_ids(conteudo)
        self.assertTrue(any("buraco na numeração" in v for v in violacoes), violacoes)


if __name__ == "__main__":
    unittest.main(verbosity=2)
