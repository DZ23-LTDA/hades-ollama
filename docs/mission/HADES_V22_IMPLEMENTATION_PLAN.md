# HADES V2.2 — PLANO DE IMPLEMENTAÇÃO, RETOMADA, EXPANSÃO E HOMOLOGAÇÃO

**Repositório:** https://github.com/DZ23-LTDA/hades-ollama
**Base:** HADES MASTER V2.2 (retomada da janela V2.1 esgotada em 256/256)
**Estado global:** `NOT_RELEASE_READY`
**Medição inicial (V2.2 §2):** ver `HADES_MASTER_STATE.json → measured_at` / `measured_commit` e o bloco
`ESTADO MEDIDO` no `HADES_HANDOFF.md` §0. Nenhum SHA deste documento é "verdade eterna": o próprio merge avança a
`main`, então a referência é **quando** e **qual revisão** foi medida, com o comando para remedir.

---

## 0. Delta V2.1 → V2.2

| Item | V2.1 (encerrada) | V2.2 (esta janela) |
| --- | --- | --- |
| Escopo | 38 estágios | 38 estágios **+ estágios 39, 40, 41** e dois incrementos transversais |
| Distribuição | Windows não assinado, demais plataformas ausentes | **Windows + Linux + Android + iOS** sob arquitetura única |
| Empresas | criadas pelo Hades | **também empresas já existentes** (onboarding por integração/importação) |
| Agenda | inexistente como motor | **Scheduling Engine** multiempresa/multiunidade/antidouble-booking |
| Descoberta | README e docs internas | **Estágio 39** SEO/GEO + open source |
| Criação | Product Builder (apps) | **Estágio 40** Creator Studio / YouTube (pipeline real de vídeo) |
| Diagnóstico | ausente | **Estágio 41** REA (reverse engineering governado) |
| Estado | `updated_at` **no futuro** e duas fontes divergentes | **reconciliado** + teste automático de cronologia/enumeração (E-311) |

## 1. Ordem obrigatória (V2.2 §0 e §15) e onde estamos

| # | Etapa | Estado | Evidência |
| --- | --- | --- | --- |
| A | Retomada fiel e reconciliação do estado | **COMPLETED_VERIFIED** | E-311 |
| B | Remediação (lint pt-BR, matriz CI, escopos, RLS/IDOR/SSRF, instalação) | **IN_PROGRESS** | E-306/E-309 + esta matriz |
| C | Incremento I — Scheduling + onboarding de empresas existentes | NOT_STARTED | — |
| C | Incremento II — Distribuição Win/Linux/Android/iOS | NOT_STARTED | — |
| C | Estágio 39 — Discoverability/SEO/GEO/GitHub | NOT_STARTED | — |
| C | Estágio 40 — Creator Studio / YouTube | NOT_STARTED | — |
| C | Estágio 41 — REA Reverse Engineering Lab | NOT_STARTED | — |
| D | Testes por funcionalidade a cada PR | **IN_PROGRESS** | bateria descrita no §4 |
| E | Auditoria profunda independente (após as integrações) | NOT_STARTED | `HADES_V22_RELEASE_AUDIT.md` (só depois) |
| F | Correção, reteste, homologação multiplataforma e de canais | NOT_STARTED | — |
| G | Release somente com os gates aplicáveis | **NOT_RELEASE_READY** | matriz do gate global |

> **A auditoria final NÃO vem antes dos incrementos.** Auditoria preliminar de entendimento e segurança já foi feita
> (leituras do §2 do V2.2 + E-311); a profunda acontece **depois** de E, conforme a ordem.

## 2. Tarefas com dependências e critério de aceite

### T-B1 — Lint pt-BR (`misspell`) sem ignorar tudo — P1
- **Depende de:** nada (a versão do linter já está fixada desde E-306/E-309).
- **Aceite:** dicionário/exclusões **mínimas e justificadas**; nenhuma regra global desligada; teste que falha se a
  exclusão crescer sem justificativa; CI verde.
- **Limite declarado:** subir a versão do `golangci-lint` é decisão de política com PR próprio.

### T-B2 — Matriz CI completa (Go, web, Windows, Linux, macOS, segurança, race, SBOM, smoke) — P1
- **Depende de:** nada. **Medição:** hoje a matriz `test (*)` está vermelha por motivo **pré-existente** e **não
  obrigatório** (medido em E-306); os gates que bloqueiam são `Go agentic and server gates`,
  `Go agentic Windows portability`, `PostgreSQL RLS...`, `Web and mobile quality` e os agregadores.
- **Aceite:** cada job com estado medido, motivo e dono; nenhum job "vermelho por costume" sem explicação.

### T-B3 — Escopos por rota, aprovação, ordem de recusa e side channels — P1
- **Depende de:** nada. Reaproveitar E-253/E-254/E-255/E-187 sem inventar IDOR por heurística.
- **Aceite:** teste que prova recusa **e** ordem de recusa, com evidência por rota.

### T-B4 — Multitenancy profunda (RLS real sem `BYPASSRLS`, memória, arquivos, cache, filas, storage, workers) — P1
- **Depende de:** T-B1 (para CI limpo). Reaproveitar o job `PostgreSQL RLS, Redis DLQ and OTLP integration`.
- **Aceite:** isolamento provado em **cada** camada com teste negativo.

### T-B5 — Instalação real, upgrade e desinstalação sem afetar o Ollama do usuário — P1
- **Depende de:** Incremento II (mesmos artefatos). **Aceite:** instalação limpa, upgrade, rollback, desinstalação;
  nenhum modelo duplicado; nenhum certificado privado do Ollama reutilizado.
- **Bloqueio:** assinatura (SignPath/Apple) ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` se a conta não existir.

### T-B6 — 104/122 conectores: catálogo ≠ integrado — P2
- **Depende de:** nada. **Aceite:** status honesto por conector + estratégia de expansão por impacto; nenhum
  conector não comprovado rotulado como disponível.

### INC-I — Universal Scheduling & Business Onboarding — P0
- **Depende de:** T-B4 (isolamento) porque agenda é dado sensível multiempresa.
- **Entregáveis:** onboarding por integração/importação/mapeamento com preview, dedupe, rollback, conflitos e
  conectores versionados; Scheduling Engine (serviços, durações, buffers, turnos, folgas, férias, feriados, fusos,
  capacidades, reservas simultâneas, lista de espera) com **anti-double-booking transacional** (locks/constraints),
  idempotência por canal, disponibilidade lida da **fonte da verdade**, criar/confirmar/cancelar/reagendar/lembretes/
  no-show/auditoria, sync autorizado Google Calendar/Microsoft 365 quando houver credencial, mesmas APIs para agentes
  de voz/WhatsApp/Telegram/recepção, políticas de dado sensível para setores regulados, pacotes de nicho sem hardcode
  de permissões.
- **Aceite (E2E):** dois agentes concorrendo pelo **último horário** (um ganha, o outro recebe recusa explícita);
  reinício; cancelamento; isolamento entre tenants; tudo com evidência por etapa.

### INC-II — Distribuição unificada — P0
- **Depende de:** T-B5. Reaproveitar `apps/mobile-agentic` (Expo/React Native) e scripts existentes; **não** criar
  quatro produtos.
- **Aceite por plataforma:** Windows `.exe` (instalador, desinstalador, update, rollback, logs); Linux `.deb`/`.rpm`/
  AppImage + systemd/permissões/x86_64-ARM; Android `.apk` interno e `.aab`; iOS **build real só com conta Apple**
  (senão `BLOCKED_BY_EXTERNAL_DEPENDENCY`).
- **Segurança:** identidade única, tokens em keychain/secure store, TLS, isolamento por organização; o Ollama do PC
  **só** por via autenticada — nunca exposto na internet.

### EST-39 — Discoverability, SEO/GEO e GitHub — P1
- **Depende de:** nada. **Aceite:** README PT-BR/EN, quickstart **testado**, docs públicas, casos de uso, guias de
  API, GitHub topics, SECURITY, licença + atribuições MIT, changelog, SBOM; site técnico com canonicals, sitemap,
  robots, metadados, Open Graph, Schema.org, URLs estáveis, acessibilidade e performance.
- **Proibições:** prometer ranqueamento/recomendação por buscadores ou IAs; `llms.txt` é **complementar e
  experimental**; domínio e publicação exigem autorização.

### EST-40 — Creator Studio / YouTube — P1
- **Depende de:** INC-I não; depende de Model Gateway, Artifact store, Workflow e aprovação humana.
- **Auditoria da fonte:** `OleLehmann/youtube-automation-agent` (TUBEGENT) — revalidar SHA/licenças; **não** importar o
  projeto inteiro nem criar segundo orquestrador.
- **Entregáveis:** sete papéis (estratégia, roteiro, thumbnail, SEO, produção, publicação, analytics) sob runtime
  Hades; pipeline **real** até **MP4 válido** (codec/duração/áudio testáveis), legendas, thumbnail, metadados,
  preview, aprovação, upload com stream binário verdadeiro, scheduling/playlist e métricas.
- **Achado herdado a não repetir:** caminhos de **simulação** (`getVideoStream` devolvendo JSON) **não** contam como
  sucesso. Sem OAuth de canal ⇒ `PUBLICAÇÃO_EXTERNA_BLOQUEADA`, com pipeline offline real homologado.

### EST-41 — REA Reverse Engineering Lab — P1
- **Depende de:** sandbox e política de egress/DLP.
- **Entregáveis:** integração **via MCP oficial + skills versionadas** (não vendorização maciça), seção de engenharia,
  comando `/reverse`, descoberta de capacidades e dependências (Ghidra/Hopper/JADX/ADB) com status por SO; sandbox
  forte (processos, rede, recursos, path safety, isolamento por tenant, DLP, auditoria).
- **Proibições:** contornar DRM, licença, autenticação ou controle de acesso; copiar propriedade intelectual.
  Ferramenta comercial exige licença do operador.
- **Aceite:** analisar artefato **próprio autorizado**, gerar mapa/evidência, propor mudança no código original e
  testar a correção em ambiente isolado.

### Complemento — empresa autônoma e comunicação (§10) — P1
- **Aceite:** WhatsApp priorizando **Cloud API oficial** (QR **somente** onde o protocolo permite — Evolution opcional;
  Cloud API **não** usa QR); áudios recebidos/transcritos/respostos quando houver TTS; telefonia via Tel-Agent/SIP com
  licença e conectividade reais; marketplaces apenas com API/conta; **todos** os comandos de barra só publicados com
  implementação real e teste E2E.

## 3. Bloqueios externos (não se fecham com mock)

| Bloqueio | O que falta | Desbloqueio |
| --- | --- | --- |
| B-01 Ollama Cloud | conta/API | login do proprietário |
| B-02 CLIs (Codex/Claude/Gemini) | login | autenticar nas CLIs |
| B-03 Canais de IM | tokens Meta/Telegram | conta + token |
| B-04 Telefonia/SIP | número, provedor, credencial | contrato SIP |
| B-05 Negócio (marketplace/social/pagamento) | contas e scopes | contas autorizadas |
| Apple/Expo | conta Apple Developer + EAS | conta aprovada |
| YouTube | OAuth de canal | canal de teste autorizado |
| Assinatura Windows | SignPath/certificado | conta/certificado emitido |
| REA | licenças de ferramentas comerciais | licença do operador (ou uso só de livres) |

## 4. Bateria obrigatória antes de cada push

```powershell
$repo = (Get-Location).Path
$env:GOROOT = "$repo\_tmp\goroot1269\go"; $env:GOPATH = "$repo\_tmp\gopath"
$env:GOMODCACHE = "$repo\_tmp\gomodcache"; $env:GOCACHE = "$repo\_tmp\gocache1269"
$env:GOTMPDIR = "$repo\_tmp\gotmp"; $env:GOFLAGS = '-mod=mod'; $env:GOTOOLCHAIN = 'local'
$env:PATH = "$env:GOROOT\bin;$env:PATH"
gofmt -l internal/agent internal/multillm server/agent_routes.go   # tem de sair VAZIO
go test ./internal/agent/ -count=1
python scripts/hades_release_gate.py; python scripts/hades_baseline_report.py
python scripts/hades_stage_map.py; python scripts/hades_mission_check.py
python scripts/hades_ci_reproducibility_check.py --check
python scripts/hades_state_chronology_test.py
```

## 5. Segurança — credenciais comprometidas (V2.2 §3)

Chaves coladas em chat (prioritariamente `GITHUB_TOKEN`) são consideradas **COMPROMETIDAS** até o proprietário provar
revogação. **Não** são reproduzidas aqui nem em logs/reports. O proprietário deve **revogar/rotacionar** por canal
próprio; nenhuma rotação é alegada sem ação comprovada. Trabalho independente segue normalmente.

## 6. Modelo de saída de cada rodada (V2.2 §14)

```
[RODADA/ETAPA] [SHA medido] [PR] [E-ID]
[O QUE FOI IMPLEMENTADO] [TESTES EXECUTADOS E RESULTADOS]
[O QUE ESTÁ HOMOLOGADO] [O QUE ESTÁ BLOQUEADO E POR QUÊ]
[PRÓXIMA TAREFA AUTOMÁTICA]
```

## 7. Regra de honestidade deste plano

- Código presente ≠ funcional; fixture ≠ credencial; adapter ≠ provedor homologado; CI verde ≠ E2E de usuário;
  estágio com evidência ≠ `COMPLETED_VERIFIED`.
- Nenhum incremento é marcado concluído sem teste reproduzível ligado a commit e ambiente.
- `RELEASE_READY` exige recorte explícito de release com **todos** os seus gates aprovados — e **não** afirma que a
  visão completa está pronta.
