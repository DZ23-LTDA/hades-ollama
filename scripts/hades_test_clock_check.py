#!/usr/bin/env python3
"""E-422 — Checker: TESTE QUE USA O RELÓGIO DA MÁQUINA COMO INSTANTE DE NEGÓCIO.

**A CLASSE, MEDIDA DUAS VEZES EM PRODUÇÃO**: um teste que deriva um **instante de negócio** (agendamento,
vencimento de lembrete, janela) de `time.Now()` **passa 23 horas por dia e falha na 24ª**.
  - E-417: `scheduling_test.go:17` — `agora := Now()+24h`, `agora.Add(90min)` ⇒ *às 23:41 cai em 01:11 do dia
    seguinte, fora de um turno `00:00–24:00`*;
  - E-421: `scheduling_waitlist_test.go:187` — **deixou o CI vermelho** ✗ — `agora := Now()`, `agora.Add(90min)`
    ⇒ *às 22:23 cai em 23:53, e o serviço termina depois da meia-noite*.

**POR QUE ESTE CHECKER EXISTE**: a varredura do E-418 (feita à mão, descartável) procurou **`time.Now()` seguido
de `.Add(...)` na MESMA linha** — e por isso **não viu** a forma em duas linhas
(`agora := time.Now()` … `agora.Add(...)`), que é exatamente a do E-421. ***O ponto cego não era do produto: era
do instrumento.*** Este checker cobre **as duas formas**.

O QUE ELE **NÃO** É: ele não tenta provar que o instante é perigoso. Ele **triagem** — aponta o sítio, e a
inspeção decide. *Excesso de zelo aqui custa uma leitura; falta de zelo custa um CI vermelho de madrugada.*

CLASSIFICAÇÃO (por que alguns usos são legítimos):
  - **vencimento** (`ClaimDue(now+2min)`, `NextRunAt = now-1min`): *"está vencido?"* não depende do calendário;
  - **timestamp** (`CreatedAt: now`, `UpdatedAt`, `ExpiresAt`): registra quando aconteceu, não escolhe quando;
  - **`now` puro**: passado como "agora" para uma função que compara;
  - **instante de negócio** (o que este checker procura): um `StartsAt`/`marcado`/`alvo`/`slot` **derivado** do
    relógio e deslocado — *é este que cruza fronteiras de calendário.*

------------------------------------------------------------------------------------------------------
E-909 — QUATRO DEFEITOS DO PROPRIO INSTRUMENTO, CORRIGIDOS (findings 2-6 do PR #438)
------------------------------------------------------------------------------------------------------
Cada um foi **confirmado por leitura** antes de ser corrigido, e cada correcao tem teste que a discrimina.

  (a) **CLASSIFICAVA A LINHA, NAO O USO.** `classificar` recebia a linha inteira e usava `any(... in linha)`.
      Consequencia medida: `OBSERVACAO` contem `"At:"`, e **`NextRunAt:` contem `At:`** ⇒
      `NextRunAt: now.Add(2*time.Hour)` era classificado **timestamp** - *apesar de `NextRunAt` estar em
      `NEGOCIO`*. (`StartsAt` escapava so' porque estava na exclusao da condicao seguinte.) **Cura:** `usos()`
      extrai **cada** `.Add(` com o seu **proprio contexto** (o rotulo/campo a que pertence), e cada uso e'
      classificado isoladamente. *Um uso `SEGURO` na mesma linha **nao pode** esconder um uso de negocio.*

  (b) **PARAVA NO PRIMEIRO `.Add(`.** O `break` fazia com que, numa variavel com dois deslocamentos
      (*um seguro e um de negocio*), so' o primeiro fosse visto. **Cura:** a janela devolve **todos** os usos.

  (c) **TRIAGADOS POR `arquivo:linha`.** O proprio comentario da lista admitia *«Pinar NUMERO DE LINHA e'
      brittleness conhecida»* - *e o E-432 ja' tinha medido o custo: reescrever um cabecalho moveu o sitio de
      **28** para **36**, e a catraca acusou o **movimento**, nao a **mudanca**.* **Cura:** a chave e'
      `(arquivo, ASSINATURA DE CONTEUDO)`. *Mover o ficheiro ou inserir linhas acima **nao** reabre o achado;
      **mudar a linha** reabre* - que e' o que a catraca deve fazer.

  (d) **ACUSAVA COMENTARIO.** A varredura lia as linhas cruas, e um dos dois `TRIAGADOS` era **literalmente**
      *«a linha do COMENTARIO do E-421»*: o instrumento acusava **texto**, nao codigo. **Cura:**
      `sem_comentarios()` remove comentarios Go (`//` e `/* */`) **preservando o numero de linha** - *a lista
      `linhas` mantem o mesmo comprimento e os mesmos indices, so' o conteudo e' limpo.*

E a **duplicacao** (finding 6): o `_test.py` mantinha `conhecidos` **proprio**, com **3** sitios, enquanto
`TRIAGADOS` tinha **2** ⇒ *duas allowlists com o mesmo proposito e conteudos diferentes: um sitio no teste e nao
no checker faz o `--check` **reprovar** e o teste **aprovar**.* **Cura:** o teste deriva de `checker.TRIAGADOS`.

USO: python3 scripts/hades_test_clock_check.py [--check]
  --check  sai != 0 se houver sítio de ALTO risco (para a bateria)
"""

import re
import sys
from pathlib import Path

# Nomes de variável/campo que indicam INSTANTE DE NEGÓCIO (escolhido, não observado).
NEGOCIO = (
    "StartsAt", "marcado", "alvo", "slot", "NextRunAt", "RunAt", "Due", "due",
    "janela", "livre", "abertura", "fechamento",
)
# Formas que tornam o uso legítimo, mesmo com deslocamento.
SEGURO = (
    "-time.Second", "-time.Minute", "-time.Hour", "-1*time.Minute", "-2*time.Hour",
    "Truncate(24 * time.Hour)", "time.Date(",
)
# 🚨 E-422 (MEDIDO pelo próprio teste do checker): **consulta de VENCIMENTO** não é instante de negócio.
# `ClaimDue(now+1s)` e `ExpiresAt = now-1s` perguntam *"já venceu?"* — e a resposta **compara instantes**, não
# dias de calendário ⇒ *um `+1s` nunca cruza uma fronteira de dia de forma que mude a resposta*. O meu
# classificador marcava `ClaimDue(Now+Add(time.Second))` como ALTO ✗ porque `Due` estava na lista de negócio e o
# deslocamento era **positivo** (só os negativos estavam em SEGURO). *Um verificador que acusa vencimento obriga a
# inspecionar falso positivo toda vez — e falso positivo repetido ensina a ignorar o instrumento.*
VENCIMENTO = ("ClaimDue", "claimDue", "IsDue", "isDue", "vencido", "Vencido", "due(", "Due(", "Expira", "expira")
# Campos que são claramente observação, não escolha.
#
# E-909(a): `"At:"` e `"at:"` foram **REMOVIDOS** desta lista. Eles existiam para pegar `CreatedAt:`/`UpdatedAt:`
# - mas `"At:"` e' **substring** de `NextRunAt:`, `RunAt:` e `StartsAt:`, *todos de NEGOCIO*. Eram um padrao
# largo demais a fazer o trabalho de um específico. Os nomes completos (`CreatedAt`, `UpdatedAt`, `ObservedAt`,
# `RecordedAt`) ficam, e a classificacao por **uso** (E-909a) torna o padrao largo desnecessario.
OBSERVACAO = ("CreatedAt", "UpdatedAt", "ObservedAt", "RecordedAt", "ExpiresAt")

DIRETO = re.compile(r"time\.Now\(\)[^\n;]*?\.Add\(")
VARIAVEL = re.compile(r"(\w+)\s*:=\s*time\.Now\(\)")
ADD = re.compile(r"\.Add\(([^)]*)\)")
# Inicio de um "contexto": apos `{`, `,`, `;`, `(` ou o inicio da linha. O rotulo/campo do uso esta' entre o
# inicio do contexto e o `.Add(`.
INICIO_DE_CONTEXTO = re.compile(r"[{,;(]")


def usos(linha: str) -> list[tuple[str, str]]:
    """Devolve o CONTEXTO de cada `.Add(` da linha - um por uso, na ordem em que aparecem.

    O contexto e' o trecho desde o delimitador anterior (`{`, `,`, `;`, `(`) ou o inicio da linha ate' ao
    `.Add(`. E' o que diz **a que campo/variavel aquele deslocamento pertence**.

    E-909(a): existir para que `CreatedAt: now.Add(...)` e `StartsAt: now.Add(...)` na **mesma linha** sejam
    dois usos distintos, classificados em separado. Antes desta funcao, a linha inteira era uma unidade e um
    rotulo seguro escondia o inseguro.

    E-909(b): devolve **todos**, nao o primeiro. *Uma variavel com dois deslocamentos - um seguro e um de
    negocio - tem de expor o de negocio.*
    """
    contextos: list[tuple[str, str]] = []
    for m in ADD.finditer(linha):
        inicio = 0
        for d in INICIO_DE_CONTEXTO.finditer(linha, 0, m.start()):
            inicio = d.end()
        contextos.append((linha[inicio:m.start()], m.group(1)))
    return contextos


def classificar(rotulo: str, argumento: str, linha: str) -> str:
    """Classifica UM uso `.Add(`, pelo seu ROTULO, pelo seu ARGUMENTO e pela linha inteira.

    Ordem de decisao - a **primeira** que casar manda:
      1. **argamento** desloca para TRAS (`-time.Minute`) ou e' construcao de calendario (`time.Date(`,
         `Truncate(24h)`) => `vencimento/janela-segura`. *Deslocar para tras pergunta "ja' passou?" - e a
         resposta compara instantes, nao dias de calendario.*
      2. **rotulo** e' consulta de vencimento (`ClaimDue`, `due(`) => `vencimento`.
      3. **rotulo** e' observacao (`CreatedAt`, `ExpiresAt`) => `timestamp`.
      4. **rotulo** e' campo de negocio (`StartsAt`, `NextRunAt`, `alvo`) => **ALTO**.
      5. sem rotulo conclusivo, a **linha inteira** decide (formas seguras por construcao).

    E-909(f): o argumento e' conferido **primeiro** porque o sentido do deslocamento e' mais específico que o
    nome do campo. *Sem isso, `NextRunAt = agora.Add(-time.Minute)` ("ja' venceu?") era ALTO so' por causa do
    rotulo - e um achado falso repetido ensina a ignorar o instrumento.*
    """
    if any(s in argumento for s in SEGURO):
        return "vencimento/janela-segura"
    if any(v in rotulo for v in VENCIMENTO):
        return "vencimento"
    if any(o in rotulo for o in OBSERVACAO):
        return "timestamp"
    if any(n in rotulo for n in NEGOCIO):
        return "ALTO"
    # Sem rotulo conclusivo, a linha inteira decide (formas seguras por construcao).
    if any(s in linha for s in SEGURO):
        return "vencimento/janela-segura"
    if any(v in linha for v in VENCIMENTO):
        return "vencimento"
    if any(o in linha for o in OBSERVACAO):
        return "timestamp"
    if any(n in linha for n in NEGOCIO):
        return "ALTO"
    return "baixo"


def sem_comentarios(linhas: list[str]) -> list[str]:
    """Remove comentarios Go (`//` e `/* */`) PRESERVANDO o numero de linha.

    E-909(d): a varredura acusava **texto**. Um dos sitios triagados era *«a linha do COMENTARIO do E-421»* -
    o instrumento apontava uma frase sobre o defeito, nao o defeito.

    A lista devolvida tem **o mesmo comprimento** e os **mesmos indices**: cada posicao continua a ser a linha
    `i+1` do ficheiro. *Preservar a numeracao e' o requisito, nao um detalhe: o achado cita `arquivo:linha`, e
    uma numeracao deslocada tornaria o proprio relatorio falso.*

    Limitacao declarada: nao interpreta strings RAW com crase nem literais multi-linha; um `//` dentro de uma
    string de uma linha e' removido. Para a **triagem** de `*_test.go` isso e' aceitavel - *e um falso negativo
    aqui vale mais do que o falso positivo de acusar comentario.*
    """
    limpas = []
    em_bloco = False
    for linha in linhas:
        saida = []
        i = 0
        while i < len(linha):
            if em_bloco:
                fim = linha.find("*/", i)
                if fim == -1:
                    i = len(linha)
                    break
                em_bloco = False
                i = fim + 2
                continue
            if linha.startswith("//", i):
                break
            if linha.startswith("/*", i):
                em_bloco = True
                i += 2
                continue
            saida.append(linha[i])
            i += 1
        limpas.append("".join(saida))
    return limpas


def assinatura(linha: str) -> str:
    """Assinatura de CONTEUDO de uma linha, para a catraca de triagem.

    E-909(c): o `TRIAGADOS` era indexado por `arquivo:linha`, e o proprio comentario da lista admitia
    *«Pinar NUMERO DE LINHA e' brittleness conhecida»*. O custo ja' tinha sido medido no E-432: um cabecalho
    novo moveu o sitio de **28** para **36**, e a catraca acusou o **movimento** em vez da **mudanca**.

    A assinatura e' o texto **sem espacos** e **sem comentario**. *Mover o ficheiro, inserir linhas acima ou
    reindentar **nao** reabre o achado; **alterar a expressao** reabre* - que e' o comportamento que uma
    catraca deve ter.
    """
    return "".join(sem_comentarios([linha])[0].split())


def varrer(raiz: Path) -> list[tuple[str, int, str, str]]:
    """Devolve `(arquivo, linha, classe, trecho)` para cada sitio com relogio deslocado.

    A forma da tupla e' **estavel**: `_test.py` e o relatorio dependem dela.
    """
    achados: list[tuple[str, int, str, str]] = []
    for arq in sorted(raiz.rglob("*_test.go")):
        cruas = arq.read_text(encoding="utf-8").split("\n")
        linhas = sem_comentarios(cruas)
        for i, linha in enumerate(linhas):
            # Forma 1: Now()...Add(...) na MESMA linha.
            if DIRETO.search(linha):
                # E-909(a): cada uso e' classificado em separado; o pior uso decide a classe do sitio.
                us = usos(linha) or [(linha, "")]
                classes = [classificar(rot, arg, linha) for rot, arg in us]
                classe = "ALTO" if "ALTO" in classes else classes[0]
                achados.append((arq.as_posix(), i + 1, classe, linha.strip()[:120]))
                continue
            # Forma 2 (o ponto cego do E-418): variavel := Now(); ... variavel.Add(...) ate 14 linhas depois.
            m = VARIAVEL.search(linha)
            if not m:
                continue
            var = m.group(1)
            padrao = re.compile(r"\b" + re.escape(var) + r"\.Add\(")
            # E-909(b): recolhe TODOS os usos da janela, em vez de parar no primeiro.
            for j in range(i + 1, min(len(linhas), i + 15)):
                if not padrao.search(linhas[j]):
                    continue
                # E-909(b): TODOS os usos da janela, nao so' o primeiro. *Sem `break`.*
                us = usos(linhas[j]) or [(linhas[j], "")]
                classes = [classificar(rot, arg, linhas[j]) for rot, arg in us]
                classe = "ALTO" if "ALTO" in classes else classes[0]
                achados.append((arq.as_posix(), j + 1, classe, linhas[j].strip()[:120]))
    return achados


# E-422: sítios de ALTO já **inspecionados e medidos** como legítimos. *Um instrumento que acusa sempre não serve
# de gate* — a lista existe para que o `--check` reprove **o que é NOVO**, e para que cada aceite carregue a razão.
# NUNCA acrescente aqui só para o teste passar: meça primeiro e escreva a medição.
#
# E-909(c): a chave e' `(arquivo, ASSINATURA DE CONTEUDO)` e nao `(arquivo, linha)`. *Um cabecalho novo nao
# reabre o achado; uma expressao nova reabre.*
# E-909(d): depois de `sem_comentarios`, o sitio que era **comentario** deixa de ser varrido - *e por isso saiu
# desta lista: ele nao precisava de dispensa, precisava de nao ser lido.*
TRIAGADOS = {
    # A chave e' a **assinatura MEDIDA** do trecho, nao uma escrita a mao. *A primeira versao desta lista usou
    # `assinatura("_ = BookRequest{StartsAt: agora.Add(time.Hour)}")` - uma linha **imaginada** - e o checker
    # mediu `ALTO triagado: 0`: a dispensa nao dispensava nada. Pior que nao ter dispensa e' ter uma que
    # **parece** cobrir o caso e nao cobre.*
    ("internal/agent/declared_categories_measured_test.go",
     'ContactID:"contato",Channel:"voice",StartsAt:agora.Add(time.Hour),IdempotencyKey:"k1",'):
        # E-432/E-909(c): era indexado como linha **36** - *e a linha ja' mudou duas vezes*. A dispensa e' pelo
        # CONTEUDO: e' a **organizacao vazia**, onde o portao de ESCOPO dispara ANTES do expediente, entao o
        # instante nao decide nada.
        "a organizacao e' \"\" - o portao de ESCOPO dispara ANTES do expediente, entao o instante nao decide nada",
    # ⚠️ E-909(d): a entrada de `scheduling_waitlist_test.go:187` foi **REMOVIDA**, e nao por conveniencia:
    # aquele sitio era **a linha do COMENTARIO** do E-421. Depois de `sem_comentarios()`, ele **nao e' mais
    # varrido** - *a dispensa deixou de ter objeto.* Uma dispensa que ja' nao corresponde a achado nenhum e'
    # **divida silenciosa**: ela esconderia um ALTO futuro que caísse naquela posicao. O teste
    # `test_triagem_nao_tem_dispensa_morta` verifica que toda chave de `TRIAGADOS` casa com um sitio real.
}


def main() -> int:
    # ***O STDOUT TEM DE SER UTF-8, MESMO QUANDO O TERMINAL NAO E'.***
    # Medido: o script imprime U+2705, U+1F534 e U+1F6A8, e com o stdout em **cp1252** (o default do
    # Windows quando nao ha' `-X utf8`) rebenta com
    # `UnicodeEncodeError: 'charmap' codec can't encode character '\u2705'`.
    # => o `reconfigure` com `errors="replace"` garante que o RELATORIO sai, mesmo que um simbolo
    # apareca como `?` -- ***um checker que morre a imprimir nao checa nada.***
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    check = "--check" in sys.argv
    raiz = Path("internal")
    if not raiz.is_dir():
        print("  (sem internal/ aqui — rode da raiz do repo)")
        return 0
    achados = varrer(raiz)
    altos = [a for a in achados if a[2] == "ALTO"]
    novos = [a for a in altos if (a[0], assinatura(a[3])) not in TRIAGADOS]
    print("  sitios com relogio deslocado em testes: %d" % len(achados))
    for arq, linha, classe, trecho in achados:
        if classe != "ALTO":
            continue
        razão = TRIAGADOS.get((arq, assinatura(trecho)))
        if razão:
            print("    ✅ %s:%d  [ALTO TRIAGADO]  %s" % (arq, linha, razão))
        else:
            print("    🔴 %s:%d  [ALTO NOVO]  %s" % (arq, linha, trecho))
    print("  ALTO triagado: %d | ALTO NOVO: %d" % (len(altos) - len(novos), len(novos)))
    print("  NOTA: ALTO nao prova defeito - e' TRIAGEM. Vencimento/timestamp/'now' puro nao entram porque o")
    print("        resultado nao depende do calendario.")
    if check and novos:
        print("  FALHA: %d sitio(s) de ALTO NAO TRIAGADO(S) - inspecione e, se for instante de negocio, use" % len(novos))
        print("         instanteDeTeste(t) (E-421); se for legitimo, acrescente a TRIAGADOS com a razao MEDIDA.")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
