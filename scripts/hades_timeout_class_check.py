#!/usr/bin/env python3
"""Catraca da CLASSE de defeito do ADV-37/ADV-38 (E-362).

O QUE ACONTECEU (medido): um teste abria uma transação ativa e chamava a rotação de chave com
`context.WithTimeout(ctx, 150*time.Millisecond)` esperando a recusa `runtime is not quiescent`. Sob carga do CI, quem
vencia a corrida era o `context deadline exceeded` — e a asserção sobre a MENSAGEM falhava. ***Asserção sobre mensagem
+ orçamento apertado = teste inerentemente instável.***

O QUE ESTA CATRACA FAZ: marca como violação todo `context.WithTimeout`/`WithDeadline` com orçamento **abaixo de 1
segundo** que NÃO esteja explicitamente testando o próprio deadline (`DeadlineExceeded`). A classe é proibida, não o
número: *um teste que mede latência de propósito tem motivo para esperar; um teste que espera uma RECUSA não pode
correr contra ela.*

O QUE **NÃO** É VIOLAÇÃO (a distinção que evitou um estrago): valores de **domínio** (`queue.leaseDuration = 120ms`),
**instantes** de linha do tempo (`reclaimExpired(ctx, now.Add(40ms))`) e **esperas deliberadas** (`time.Sleep`) —
nenhum deles é orçamento de contexto. *Classificar antes de agir é o que separa instrumento de arma.*
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

# Dispensas DECLARADAS, cada uma com motivo. Vazio hoje: a classe está exausta.
DISPENSAS: dict[str, str] = {
    "cli_harness_real_session_test.go:121": (
        "MEDIDO: o teste cria um contexto JÁ EXPIRADO (1ms) de propósito — o comentário diz 'Contexto já expirado: o "
        "caminho governado tem de recusar/interromper rápido'. O orçamento curto É o objeto do teste."),
    "mcp_test.go:230": (
        "MEDIDO: 20ms com asserção 'expected cancelled MCP call' — o teste mede CANCELAMENTO, não espera uma recusa "
        "que corra contra o relógio."),
    "claude_desktop_test.go:1775": (
        "MEDIDO: 50ms afirma que a prontidão EM CACHE não sonda (waitForUpstream devolve nil). O objeto do teste é o "
        "cache, não o relógio."),
    "claude_desktop_test.go:1782": (
        "MEDIDO: 50ms afirma que a prontidão INVALIDADA falha ao sondar um listener FECHADO (127.0.0.1:1) — recusa de "
        "conexão em microssegundos, não espera de deadline."),
    "tools_test.go:129": (
        "MEDIDO: 50ms para matar um processo auxiliar de 10s; o teste já foi CORRIGIDO por um defeito desta mesma classe "
        "e traz a lição escrita: o limite era 2s, o runner Windows excedeu (2,37s) e passou a medir a PROPRIEDADE."),
}

PADRAO = re.compile(r"With(?:Timeout|Deadline)\(\s*[^,]+,\s*(?P<valor>\d+)\s*\*\s*time\.(?P<unidade>Millisecond|Microsecond|Second)\s*\)")


def _segundos(valor: str, unidade: str) -> float:
    numero = float(valor)
    if unidade == "Millisecond":
        return numero / 1000.0
    if unidade == "Microsecond":
        return numero / 1_000_000.0
    return numero


def auditar(raiz: Path) -> list[str]:
    violacoes: list[str] = []
    if not raiz.exists():
        return violacoes
    for caminho in sorted(raiz.rglob("*_test.go")):
        try:
            linhas = caminho.read_text(encoding="utf-8", errors="replace").split("\n")
        except OSError:
            continue
        for indice, linha in enumerate(linhas):
            for encontrado in PADRAO.finditer(linha):
                segundos = _segundos(encontrado.group("valor"), encontrado.group("unidade"))
                if segundos >= 1.0:
                    continue
                local = "%s:%d" % (caminho.name, indice + 1)
                if local in DISPENSAS:
                    continue
                # Permitido se o teste afirma o DEADLINE: aí o orçamento curto é o objeto do teste.
                contexto = "\n".join(linhas[max(0, indice - 8):indice + 25])
                # Reconhece o IDENTIFICADOR e a ASSERTIVA TEXTUAL: `tools_test.go` afirma `"deadline exceeded"` como
                # string (medido), e reprovar um teste que mede cancelamento de propósito seria a catraca errando o alvo.
                if any(marca in contexto for marca in ("DeadlineExceeded", "context.Deadline", "deadline exceeded",
                                                       "cancelled", "canceled", "Canceled")):
                    continue
                violacoes.append(
                    "%s: orçamento de %.3fs sem asserção de deadline — a classe do ADV-38 (asserção sobre recusa não "
                    "pode correr contra o orçamento)" % (local, segundos)
                )
    return violacoes


def main() -> int:
    raiz = Path(__file__).resolve().parent.parent / "internal"
    violacoes = auditar(raiz)
    if violacoes:
        for violacao in violacoes:
            print("VIOLACAO: " + violacao)
        return 1
    print("CLASSE DO ADV-38 AUSENTE: nenhum orçamento de contexto abaixo de 1s correndo contra uma asserção.")
    if DISPENSAS:
        for local, motivo in sorted(DISPENSAS.items()):
            print("  dispensado: %s — %s" % (local, motivo[:80]))
    return 0


if __name__ == "__main__":
    sys.exit(main())
