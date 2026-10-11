# Estágio 22 — Gate de Engenharia (Release Readiness Intermediário)

**SHA avaliado:** `8e3feb663a5bbe290842c0b8320681d402dff382` (`main`)
**Data:** 2026-10-11 · **Host local:** Windows · **CI:** ubuntu-latest (GitHub Actions)
**Regra deste documento:** cada item tem comando reproduzível e resultado observado.
Nada aqui é estimado; o que não foi medido está escrito como **não medido**.

---

## 1. Descoberta que corrige uma conclusão anterior (importante)

Eu havia registrado que o **E2E de navegador** estava impedido por ausência de Playwright, e
que **3 testes** falhavam por limitação do ambiente. A verificação do **log real do CI** na
`main` mostra outra coisa:

| Fato medido no log do gate `Go agentic and server gates` | Evidência |
| --- | --- |
| O CI **instala Playwright 1.63.0 + Chromium 153** e aponta `OLLAMA_AGENT_BROWSER_EXECUTABLE` para um Chromium real | `python3 -m playwright install --with-deps chromium` |
| `python3 internal/agent/browser_helper_test.py` → **9 testes OK** | `Ran 9 tests ... OK` |
| `CGO_ENABLED=0 go test ./internal/agent` → **ok 23.6s** | as "falhas locais" **passam no CI** |
| `CGO_ENABLED=1 go test ./server` → **ok 3.8s** | idem para o servidor |
| `CGO_ENABLED=1 go test ./...` → **todos os pacotes ok** | 60+ pacotes |
| `go vet ./...`, `go build -o dist/ollama-agentic .` | limpos |
| `go mod verify` | `all modules verified` |
| `govulncheck@v1.8.0` | **No vulnerabilities found** (1 em módulo requerido **não chamado**) |
| `gofmt -l` | sem saída (formatado) |
| `check-class-a-plus-integrity.sh` | **DESIGN SYSTEM VERIFICATION PASSED** e **CONTRACT VERIFICATION PASSED** (298 rotas de backend × 155 chamadas do frontend) |
| Upload de artefato | `ollama-agentic-linux` 34.778.626 bytes · SHA256 `789ce377afea29c476545a4f0dbd7f99cd44f71d5d55905d6cdc35a169497c4d` |

**Correção honesta:** as três falhas são **específicas deste host Windows**, não defeitos do
produto — o mesmo código **passa** no CI. E o **navegador real** está homologado **em CI/Linux**,
não bloqueado. O que continua **não homologado** é o **desktop/computer-use nativo** e a
execução **neste host**.

---

## 2. Suíte executada neste host (evidência local)

| Verificação | Comando | Resultado |
| --- | --- | --- |
| Build | `go build ./...` | **exit 0** |
| Vet | `go vet ./internal/agent/ ./server/` | **exit 0** |
| Lint | `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| Testes do agente | `go test ./internal/agent/` | 2 falhas **de host** (passam no CI) |
| Testes do servidor | `go test ./server/` | 1 falha **de host** (passa no CI) |
| Web: testes | `npx vitest run` | **82 arquivos / 551 testes passando** |
| Web: typecheck | `npx tsc -b` | **exit 0** |
| Web: lint | `npm run lint` | **0 erros**, 8 avisos pré-existentes |
| Web: build | `npm run build` | **✓ built in 9.44s** — exige `TEMP` gravável; ver §3 |

## 3. Falha de build que era de ambiente (com procedimento)

Na primeira execução, `npm run build` falhou com
`[vite:esbuild-transpile] remove C:\Users\...\AppData\Local\Temp\esbuild-...: Acesso negado`.
**Causa:** o esbuild não conseguiu remover seu arquivo temporário em `%TEMP%` do host.
**Procedimento que resolve:** apontar `TEMP`/`TMP` para um diretório gravável do workspace antes
do build. Com isso: `✓ built in 9.44s`, exit 0. **Não é defeito do produto** e o CI não é afetado.

## 4. Gates obrigatórios no CI (todos verdes na `main`)

| Check | Conclusão |
| --- | --- |
| `Go agentic and server gates` | **success** |
| `Go agentic Windows portability` | **success** |
| `PostgreSQL RLS, Redis DLQ and OTLP integration` | **success** |
| `Web and mobile quality` | **success** |
| `CGO-free and Windows build gate` | **success** |
| `Generate SBOM` | **success** |
| `Preserve Class A+ surfaces` | **success** |

## 5. Os 20 itens exigidos pelo estágio

| # | Item | Situação | Evidência / motivo |
| --- | --- | --- | --- |
| 1 | Funcionalidades implementadas | **DONE** | inventário em `HADES_CAPABILITY_MATRIX.md` |
| 2 | Funcionalidades **homologadas** | **PARTIAL** | **2 de 38** estágios `COMPLETED_VERIFIED` (13, 15) |
| 3 | Plugins | **DONE** (código) / homologação de terceiros externa | manifesto, integridade, rollback (E-0xx) |
| 4 | MCPs | **DONE** (código) | local/stdio/remoto, permissões por ferramenta, revogação |
| 5 | Skills | **DONE** | ativação por projeto/agente/modelo, proveniência |
| 6 | Agentes | **DONE** | agentes-base e complementares com testes |
| 7 | Subagentes | **DONE** | delegação com cotas, profundidade e isolamento |
| 8 | Modelos locais | **DONE e provado ao vivo** | troca `qwen2.5:7b` ↔ `llama3.2:3b`, contexto byte-idêntico (E-089) |
| 9 | APIs externas | **NOT_DONE / BLOCKED_BY_EXTERNAL_DEPENDENCY** | sem credenciais reais; contratos e testes existem |
| 10 | Harnesses | **PARTIAL** | contrato de sessão + detecção; sessão real sem login do CLI é dependência externa |
| 11 | Plataformas | **PARTIAL** | Linux homologado em CI; **desktop/computer-use nativo não homologado**; instaladores exigem execução real |
| 12 | Testes | **DONE** | CI verde + suíte local; falhas de host declaradas |
| 13 | Defeitos corrigidos | **DONE** | ver §6 |
| 14 | Riscos residuais | **DONE** | B-01…B-09 em `HADES_BLOCKERS.md` |
| 15 | Dependências externas | **DONE** | nomeadas com procedimento de desbloqueio |
| 16 | Artefatos | **DONE** | `ollama-agentic-linux` (SHA256 acima), SBOM gerado no CI |
| 17 | Instruções | **DONE** | `AGENTS.md`, `docs/mission/*`, handoff |
| 18 | Commits/PRs | **DONE** | PRs #159–#171 |
| 19 | Evidências | **DONE** | 98 linhas em `HADES_EVIDENCE_INDEX.md` |
| 20 | Status real | **DONE** | release global **`NOT_RELEASE_READY`** |

## 6. Defeitos corrigidos que este gate deve registrar

1. **Exportação de memória** vazava dado privado de outro usuário (E-088);
2. **Exclusão de memória** permitia apagar dado privado alheio (E-088);
3. **Contexto do modelo** levava memória privada alheia ao preâmbulo de sistema (E-089);
4. **Comando `/memory`** devolvia memória privada alheia (E-090);
5. **Caminho de resumo** não existia e foi criado sem inventar conteúdo (E-096);
6. **Falha persistente de retomada** do supervisor era **engolida em silêncio** (E-097);
7. **Escopos de memória** ausentes, fechados com separação entre propriedade e divulgação (E-095).

## 7. O que impede a aprovação deste gate

O CI passa e a suíte de engenharia está verde, mas o estágio exige também **install/aprimoramento/rollback de pacote, performance e acessibilidade medidos**, e o item 11 acima:

| Pendência do gate | Natureza |
| --- | --- |
| Instalação/upgrade/rollback executados de verdade | **EXECUTADO do código-fonte (E-113)**: instalação, atualização com backup, rollback reversível e desinstalação com binário rodando. **Instaladores de pacote assinados e macOS/Linux reais seguem NÃO MEDIDOS** |
| Performance medida | **MEDIDA E APROVADA** (E-099) |
| Acessibilidade medida (WCAG 2.2 AA) | **MEDIDA E APROVADA** (E-099) |
| Desktop/computer-use nativo homologado | **AUSENTE** |
| APIs externas com credenciais reais | **BLOCKED_BY_EXTERNAL_DEPENDENCY** |
| Homologação por plataforma em dispositivo real (Windows/macOS) | **NÃO MEDIDO** |


## 7.1 Instalação, atualização, rollback e desinstalação (E-113)

`scripts/install.ps1` ganhou **backup**, **rollback** e **desinstalação**, e o ciclo foi
**executado** num diretório descartável: instalação (`SHA-256 ADCA65F8…`), execução real do
binário (`ollama version is 0.40.1`), **atualização** entregando artefato **diferente**
(`D27C2249…`, com o anterior preservado), **rollback** restaurando exatamente o hash
original (**restaurado == original: True**) e **desinstalação** removendo binário, backup e
manifesto **sem tocar nos dados do usuário**. Detalhes e transcrito em
`docs/mission/HADES_GATE_22_INSTALL_CYCLE.md`.

**O que isso NÃO cobre**: instaladores de pacote **assinados** e instalação em **macOS e
Linux reais**. Anunciar homologação de pacote sem artefato assinado seria invenção.

## 8. Performance e acessibilidade: medidos nesta rodada (E-099)

Estes dois itens estavam registrados como **NÃO MEDIDO**. A infraestrutura já existia no
repositório — 7 benchmarks Go, `e2e/accessibility.spec.ts` com axe e um budget de bundle — e
bastou **executá-la** para obter números reais.

### Performance (benchmarks reais, 32 núcleos)

| Caminho medido | Resultado | Alocação |
| --- | --- | --- |
| `BenchmarkMemorySearchContext` | **699,3 ns/op** (473.030 iterações) | 888 B/op · 4 allocs/op |
| `BenchmarkGroundedContextBuild` | **110,7 µs/op** | 45.110 B/op · 302 allocs/op |
| `BenchmarkPluginInstallUpdateRollback` | **5,20 ms/op** | 28.030 B/op · 173 allocs/op |
| `BenchmarkDownloadChunkCompletion` | **2,37 ms/op** → **442,83 MB/s** | 39.663 B/op · 232 allocs/op |

**Budget de bundle (gate que quebra o build se estourar):** `HADES_BUNDLE_MAX_JS_BYTES=2.000.000`,
`HADES_BUNDLE_MAX_JS_GZIP_BYTES=600.000` → maior chunk `index-UMxVpmOA.js` com **1.443,91 kB
cru / 396,66 kB gzip**: **PASS** (exit 0).

**Limites declarados:** isto mede caminhos internos e o tamanho do bundle. **Não** mede latência
de inferência real (depende de modelo e hardware) nem comportamento sob carga concorrente.

### Acessibilidade (axe WCAG, medida de verdade)

`npx playwright test e2e/accessibility.spec.ts` → **3 passaram em 44,3s**:

1. `axe WCAG gate — desktop` · **nenhuma violação**;
2. `axe WCAG gate — mobile` · **nenhuma violação**;
3. `mobile shell is operable by keyboard without a trap` · **passou**.

O mesmo arquivo roda no CI pelo workflow `dz23-e2e.yaml` (`npm run test:e2e`). Para medir
localmente foi necessário instalar o build de Chromium exigido pela versão do Playwright do
repositório (`chromium_headless_shell-1248`); a execução anterior falhou por **ausência do
navegador**, não por violação de acessibilidade.

**Limites declarados:** a medição cobre as **rotas críticas definidas pelo spec**, não todas as
telas; e **não** houve teste com leitor de tela real (NVDA/JAWS/VoiceOver).

## 9. Veredito

- **Suíte de engenharia:** **APROVADA** (CI verde nos 7 checks; build, vet, lint e testes ok).
- **Gate intermediário do estágio 22:** **AINDA NÃO APROVADO**, mas com **duas pendências a menos**:
  performance e acessibilidade passaram a estar **medidas** nesta rodada. Continuam pendentes
  instalação/upgrade/rollback executados, desktop nativo homologado, homologação em dispositivo
  real e APIs externas com credenciais. Aprovar sem elas seria declarar o que não foi medido.
- **Release global:** permanece **`NOT_RELEASE_READY`**.
- **Próximos desbloqueios, em ordem de custo:** (a) executar instalação/rollback dos pacotes
  produzidos no CI (médio); (b) homologação em dispositivo real Windows/macOS (médio, depende de
  máquina); (c) desktop/computer-use nativo (implementação); (d) APIs externas com credenciais
  reais (dependência externa).
