# GATES — os que estão declarados, e como os correr

**Este ficheiro lista os gates DECLARADOS em `scripts/hades_local_gates.ps1`.** Não afirma
que passam: *quem os corre é o auditor, e o resultado é do estado em que ele os correr.*

| # | Gate |
| --- | --- |
| 1 | gofmt -l internal/agent internal/multillm server/agent_routes.go |
| 2 | go build ./... |
| 3 | golangci-lint PULADO por -SemLint (isto NAO e aprovacao do gate de lint) |
| 3 | golangci-lint run ./internal/agent/... ./server/... (mesma versao do CI) |
| 4 | go test ./internal/agent ./server -count=1 -timeout 25m -skip <medicao real com modelo local> (o passo que mais reprova no CI) |
| 4 | go test PULADO por -SemTestes (isto NAO e aprovacao dos testes) |
| 5 | go test ./app/... -count=1 (persistencia: app/store usa SQLite via CGO) |
| 5 | go test ./app/... PULADO por -SemTestes (isto NAO e aprovacao de app/store) |
| 6 | manifesto do snapshot (ESTAGIO 22 item 3) - drift entre o manifesto e a arvore |
| 7 | inventario de marca (FASE-5) - drift entre o inventario e o repositorio |
| 8 | ASCII nos .ps1 - comentario com tecto, string com divida declarada |
| 9 | FORMA do indice de evidencias - entrada = uma linha, sem ID fantasma |
| 10 | JANELA de citacao dos documentos de retomada |
| 11 | instrumentos de auditoria que estavam ORFAOS |
| 12 | TESTES GO (o 16o checker, E-488) |
| 13 | INSTALADOR Windows (Inno Setup), NAO assinado |
| 14 | PACOTE da extensao HADES Browser Bridge (zip nao assinado) |
| 15 | ID deterministico da extensao e manifesto do native host |
| 16 | testes do ID deterministico (com fault injection) |
| 17 | E2E REAL: Edge -> extensao -> native messaging -> Companion |
| 18 | E2E DE ABA: concessao, DOM, campo sensivel, badge, revogacao |
| 19 | E2E DE TAKEOVER: o humano assume, o agente e recusado, a sessao volta |
| 20 | E2E DE ORIGEM: navegar para outra origem revoga, a outra aba continua |
| 21 | CATRACA do opt-in: todo o chamador de validarUrl/guardarRota passa o opt-in |
| 22 | testes da catraca (fault injection, numa COPIA) |
| 23 | E2E DO PUSHSTATE: a concessao sobrevive ao reload; a origem real revoga |
| 24 | E2E DO BFCACHE: voltar atras NAO revive a concessao |
| 25 | E2E DE FRAMES: o observar ATRAVESSA shadow aberto e iframe, e DECLARA os limites |
| 26 | E2E DE ARQUIVOS: o browser:files DECIDE -- executavel nunca fica permitido |
| 27 | TESTES da politica de ficheiros (17 assercoes, sem browser) |
| 28 | CATRACA do encaixe: acorda se o runtime expuser a rota de aprovacao |
| 29 | testes da catraca do encaixe (4 injecoes + contraprova, numa COPIA) |
| 30 | RENDER do indice: todas as entradas com 5 campos |
| 31 | INVENTARIO do worktree: `M` de conteudo vs `M` de fim-de-linha |
| 32 | CATRACA da ligacao: os docs do estagio citam os artefactos? |
| 33 | testes da ligacao (4 injecoes + contraprova, numa COPIA) |
| 34 | DOCUMENTACAO do Companion: os 8 estados, o mapa para os 14, e as limitacoes |
| 35 | fault injection da catraca da documentacao (4 injecoes + contraprova) |
| 36 | CADEIA de requisitos: REQ-ID -> arquivo:linha -> teste -> comando -> resultado |
| 37 | fault injection da cadeia de requisitos (2 injecoes + contraprova) |
| 38 | AMBIENTE do T2: hardware, SO, Ollama, modelos, lacunas e proveniencia |
| 39 | fault injection do ambiente T2 (3 injecoes + contraprova) |
| 40 | TESTES HERMETICOS: NewRuntime em testes sempre com DataRoot (divida declarada: 107) |
| 41 | fault injection da catraca dos testes hermeticos (injecao + 3 contraprovas) |
| 42 | UX/ONBOARDING: /goal no composer, onboarding real, ARIA e teclado, menu unificado |
| 43 | fault injection do UX/onboarding (3 injecoes + contraprova) |
| 44 | SNAPSHOT: o `measured_commit` do estado e' o HEAD antes de gerar os derivados |
| 45 | o gate DECLARA o snapshot (MESMO ou a distancia), sem esconder a diferenca |
| 46 | HARNESS: as 7 capacidades, cancelamento entre instancias, Ollama, e o CLI com sonda opt-in |
| 47 | fault injection do harness (3 injecoes + contraprova) |
| 48 | AGENTES: swarm com limite aplicado, orcamento, isolamento, artefactos e HITL |
| 49 | fault injection dos agentes (3 injecoes + contraprova) |
| 50 | AUDITORIA: matriz de plataformas, workflows citados, e o aviso das instrucoes destrutivas |
| 51 | fault injection da auditoria (3 injecoes + contraprova) |
| 52 | ARQUITETURA: fontes de verdade singulares, testes de contrato, e a documentacao |
| 53 | fault injection da arquitectura (3 injecoes + contraprova) |
| 54 | SEGURANCA: tenant scope, non-inference, e o relogio de teste |
| 55 | o CHECADOR da seguranca (9 eixos do Dominio K) |
| 56 | fault injection da seguranca (3 injecoes + contraprova) |
| 57 | EXTENSAO: o manifesto, as permissoes, a CSP e o native host |
| 58 | os 120 TESTES da extensao (node --test) |
| 59 | fault injection da extensao (3 injecoes + contraprova) |
| 60 | MCP: o catalogo, o spawn, o transporte, o SSRF e a politica |
| 61 | fault injection do MCP (3 injecoes + contraprova) |
| 62 | PLUGINS: o manifesto, o bundle, o registry, os estados e o escopo |
| 63 | fault injection dos plugins (3 injecoes + contraprova) |
| 64 | ESTABILIDADE: o flake, a hermeticidade, a codificacao e o comando do CI |
| 65 | fault injection da estabilidade (3 injecoes + contraprova) |
| 66 | MATRIZ COMPETITIVA: cobertura obrigatoria e nenhuma vitoria sem prova |
| 67 | fault injection da matriz competitiva (3 injecoes + contraprova) |

## Como correr a bateria completa

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/hades_local_gates.ps1
```

## Como correr os checkers de forma independente

```powershell
python -X utf8 scripts/hades_release_gate.py --check
python -X utf8 scripts/hades_manifest.py --check
python -X utf8 scripts/hades_brand_inventory.py --check
python -X utf8 scripts/hades_index_shape_check.py --check
python -X utf8 scripts/hades_handoff_window_check.py --check
python -X utf8 scripts/hades_mission_check.py
```

**Nota**: `hades_release_gate.py`, `hades_manifest.py` e `hades_brand_inventory.py` são
**geradores**; correm em modo de escrita e só depois `--check`. *Um `--check` sem gerar
acusa drift por construção.*
