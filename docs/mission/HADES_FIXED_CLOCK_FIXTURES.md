# Fixtures com relógio fixo — matando a família do flake de hora (E-143)

## 1. Por que esta rodada existe

Na rodada anterior o CI ficou vermelho por um teste que só passava em **59 dos 60 minutos** (E-142). Corrigi
o teste — mas deixei registrado o limite: *"o fixture continua usando o relógio real"*. Consertar o teste
é tratar o sintoma; **o relógio real no fixture é a raiz**, porque todo teste que dependa dele herda a
possibilidade de falhar sozinho.

## 2. A auditoria (medida, não estimada)

| Medida | Número |
| --- | --- |
| Arquivos de teste do agente que usam `time.Now()` | **51** |
| Deles, com deslocamento **pequeno** (`Add(Second\|Minute)`) — a armadilha exata | **18** |
| **Fixtures** que entregam relógio de parede | **2** (`businessStatusFixture`, `financeFixture`) |

Os dois fixtures são a raiz compartilhada: `businessStatusFixture` alimenta **3 arquivos de teste**
(`business_status`, `business_snapshots`, `business_export`) — 10 usos no total.

## 3. O que foi feito

1. **`businessFixtureClock`**: um instante fixo único (`2026-10-12T10:00:00Z`) compartilhado pelos
   fixtures de negócio do pacote. O determinismo passa a ser **estrutural**, não estatístico.
2. **`businessStatusFixture` e `financeFixture`** passaram a usá-lo — a raiz da família.
3. Um teste que ainda chamava `time.Now()` **fora** do fixture (organização vazia) passou a usar o
   instante fixo: não havia necessidade de relógio real ali.
4. **Guarda contra regressão** (`TestBusinessFixtureClockIsNotTheWallClock`): a guarda **não** fixa a
   data (isso deixaria o teste rígido a mudanças legítimas) — ela verifica a **propriedade** que importa:
   o instante do fixture não pode ser o instante atual. Se alguém voltar a escrever `time.Now()`, o teste
   falha **ali**, e não numa terça-feira às 13:59 no CI. A guarda ainda exige que o instante seja
   **estável** entre duas chamadas do fixture.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run 'TestBusinessFixtureClock\|TestBusinessStatus\|TestBusinessSnapshot\|TestBusinessTrend\|TestBusinessExport\|TestFinance'` | **16/16 PASS** |
| `go test ./internal/agent/ -count=1` (suíte completa) | **ok 58,792s** |
| `go test ./server/ -count=1` | **ok 21,012s** |
| `gofmt -l internal/agent` | **sem saída** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |

Trocar o relógio do fixture por um instante fixo poderia quebrar asserções relativas a data (vencimentos,
SLA, retenção) — por isso a suíte **inteira** foi executada depois da mudança, e não apenas os testes
diretamente tocados.

## 5. O que continua pendente, declarado com o número

- **18 arquivos** ainda combinam `time.Now()` com deslocamentos pequenos. Nem todos são flaky: a maioria
  mede intervalos onde a fronteira não é hora (janelas de replay em segundos, por exemplo). Varrer todos
  é trabalho mecânico e arriscado de fazer em bloco; fica **declarado com o número**, em vez de
  apresentado como resolvido.
- **A guarda cobre os fixtures, não os 18 arquivos**: ela impede a volta do relógio de parede na raiz
  compartilhada, que era a origem do flake real.
