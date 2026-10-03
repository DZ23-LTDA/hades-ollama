# Correções P0 de jornadas e UX — plano de implementação

> **Para agentes:** executar as tarefas incrementalmente, com testes e revisão do diff entre fatias. Preservar estado e artefatos locais do usuário.

**Objetivo:** corrigir os gaps mais graves percebidos por usuários na Home/Agentic e tornar explícitos os limites de segurança, privacidade e recuperação documental.

**Abordagem:** manter o runtime local-first e as APIs Ollama compatíveis. Construir anexos por um fluxo autenticado e limitado, reaproveitando o import/indexador existente; reduzir privilégios padrão do autorun no cliente e no servidor; alinhar o texto de privacidade às opções reais; exibir trechos e fontes retornados pelo RAG, sem afirmar que foi sintetizada uma resposta que nunca foi gerada.

**Tech Stack:** Go, Gin, runtime `internal/agent`, React 19, TypeScript, Vitest, npm, testes Go.

**Spec de referência:** `outputs/auditoria_hades_ollama_2026-10-03.txt` (inclui achados auditados, SHA de referência e limitações).

## Restrições de execução

- Continuar somente em `recovery/ollama-full-snapshot`, após revalidar branch/HEAD/status antes da implementação.
- Não alterar `main`, não descartar arquivos locais e preservar `outputs/`, `.work/` e `.manus/`.
- Não fazer commit, push, merge ou publicar mudanças.
- Não transmitir conteúdo de arquivo por URL, query string ou storage do navegador como se fosse upload concluído.
- Não habilitar inferência remota por implicação. A tela documental desta fatia é recuperação de trechos, não geração de resposta.
- Aceitar somente formatos que o indexador realmente processa; rejeições devem ser claras. Limitar número/tamanho agregado, sanitizar nomes e manter isolamento por organização.

## Tarefas

### 1. Corrigir anexos perdidos na Home

- Trocar a navegação que carrega bytes/metadados de anexos para query string por um envio real do navegador ao servidor.
- Adicionar endpoint autenticado multipart, com limites de request, contagem/tamanho por arquivo, validação de extensão/nome e erro legível. Reaproveitar o `ProjectImporter`/`DocumentIngestor` para criar um projeto contextual e indexar arquivos seguros; não executar conteúdo importado.
- Criar a missão somente após confirmação de que importação/indexação terminou e associar o `project_id` devolvido. Se upload falhar, não criar missão órfã e manter mensagem acionável.
- Cobrir limites, formato inválido, ownership/organização, arquivos aceitos, resposta de importação e teste UI que assegure que nenhum byte/conteúdo entra na URL.

### 2. Menor privilégio no autorun

- Determinar o modelo atual de capabilities/approval no backend antes de modificar defaults.
- Fazer o autorun começar sem capabilities de escrita, shell ou browser. Qualquer elevação deve estar visível, deliberada e passar pelas mesmas validações server-side; a UI não será a única barreira.
- Manter o modo de execução manual funcional e não ampliar permissões implícitas.
- Cobrir criação e execução com capabilities perigosas, defaults do cliente e estados acessíveis de confirmação.

### 3. Privacidade e localização do onboarding

- Remover promessas absolutas incompatíveis com providers remotos ou serviços opcionais. Distinguir processamento com Ollama local de conteúdo enviado a provider externo, usando estados/consentimentos já existentes; se a implementação não distinguir, usar texto factual e qualificado.
- Unificar o idioma do onboarding com a configuração atual da interface e preservar o sentido do setup.
- Atualizar testes de contrato de texto; não alterar a política legal fora do escopo sem evidência.

### 4. Tornar a busca documental clara e utilizável

- Renomear a ação para “buscar/localizar nos documentos” enquanto não houver geração de resposta.
- Renderizar snippet, fonte e estado sem resultados de cada citação retornada; nunca renderizar `context` (que contém instruções internas do prompt) como resposta ao usuário.
- Preservar proteção contra XSS usando o renderer existente/escaped text e acessibilidade do formulário e resultados.
- Cobrir snippets, fonte, estado vazio e ausência de contexto interno na UI.

## Critérios de aceite

- Uploads de arquivos permitidos têm confirmação do servidor antes da missão; falhas não criam missões silenciosamente; conteúdo não aparece em URL/log de navegação.
- Importação usa limites e isolamento organizacional; arquivos importados não são executados.
- Autorun não recebe escrita/shell/browser por padrão e o servidor valida o pedido, não apenas o React.
- Onboarding não faz alegações absolutas quando há providers externos e não fica parcialmente em idioma divergente.
- Busca documental mostra os trechos citados e fontes, não o prompt interno nem uma resposta inventada.
- Testes focados, `npm run lint`, `npm run build` e testes Go relevantes passam; `git diff --check` passa. Registrar falhas ambientais reais sem ocultá-las.
- Relatório final lista arquivos alterados, decisões e validações, e mantém explícito que nada foi publicado.


## Emenda — 2026-10-03, 10:15 BRT

O usuário posteriormente autorizou explicitamente corrigir e subir as melhorias ao GitHub. Isso substitui a restrição original de “não fazer commit/push” apenas para esta execução: fazer commit e push normal para `recovery/ollama-full-snapshot`; não abrir PR, não tocar em `main` e nunca fazer force-push.

Escopo concluído nesta primeira fatia: anexos por multipart autenticado e indexação com limites; defaults de menor privilégio para autorun de missões; consentimento para provider remoto com anexos; copy de privacidade condicional; painel RAG honesto com trechos e sem devolver contexto interno. A localização integral do onboarding não foi incluída e deve continuar no backlog.

Validação: Vitest 56 arquivos/317 testes, build e ESLint frontend passaram; Go focado de `internal/agent` e `server`, mais `go vet` nesses pacotes, passou. `go test ./...` falha no ambiente local devido `CGO_ENABLED=0`/sqlite stub e compilação de testes MLX com símbolos ausentes; registrar sem mascarar. O gate remoto do novo SHA deve ser verificado após o push, sem declarar CI verde antes do resultado.


## Continuação — findings independentes no SHA 6473ed1 (11:10 BRT)

**Escopo e evidência:** seis revisões read-only fixaram o alvo em `6473ed1c4d22763bad103c2d563bafd4b11b78f9`; os dois workflows desse SHA foram confirmados `success` separadamente. O usuário autorizou continuar e publicar por fast-forward apenas na branch recovery, sem PR/main/force-push. Manter os bloqueios externos explícitos.

### Fase A — segurança de rede e release (bloqueadores antes de qualquer distribuição)

- `server/agent_tls.go`, `server/routes.go` e testes de política: listener não-loopback deve recusar startup sem TLS válido; autenticação não substitui confidencialidade; testar loopback/remoto, TLS/mTLS e falha sem certificado.
- `internal/agent/browser.go`, `browser_helper.py` e testes: validar cada request/redirect/subrecurso antes de navegar, bloquear private/link-local/downgrade e restringir bypass privado; executar no melhor isolamento host disponível e declarar o que ainda exige launcher/container/egress sandbox.
- `.github/workflows/release-readiness.yaml`, `.github/workflows/release.yaml`, workflows de build/instalação, `scripts/verify-release-artifact.sh` e documentação: validar candidate SHA/tag/check-run com workflow ID/head_sha/status/conclusion allowlisted; exigir provenance verificável/âncora independente para canal assinado e falhar fechado quando não configurada; registrar hashes verificáveis de toolchains/MLX antes de extrair/executar. Não disparar release nem fingir que credenciais externas existem.

### Fase B — filas, API e durabilidade

- `internal/agent/redis_queue.go`: aplicar quota global/tenant atomicamente; GC de índice órfão e retenção coerente; integração Redis para limite, TTL, órfão e próximo job.
- Runtime/Store/rotas: sinal durável de cancelamento observado pela instância que possui o lease; health/readiness que propaga falhas reais; cursor/paginação em missions/jobs/events e `Last-Event-ID`, heartbeat e replay limitado no SSE; estado `QUEUED` coerente no POST/GET/worker; fsync de diretório após rename; ledger PostgreSQL com migration ID/checksum e drift guard. Cada contrato tem teste negativo e teste de restart/integration quando o serviço estiver disponível.

### Fase C — jornadas mobile, onboarding e acessibilidade

- `app/ui/ui.go` + UI tests: corrigir `cloudOptIn` para o contrato `{enabled:true}`; falha deve ser visível, com retry e sem avanço silencioso.
- `apps/mobile-agentic/App.tsx`: payload `decision=approve|reject`, reason por approval ID, namespace/cache/outbox por sujeito e proteção contra troca de usuário; drenar fila em reconexão/foco ou recusar claramente operação sem sessão; orientar URL alcançável em dispositivo físico; testes de componente/contrato determinísticos.
- `HomePage.tsx`/`ImportProjectDialog.tsx`: devolver project ID e continuar para missão/projeto; sem novo upload.
- `Downloading.tsx`, `accessibility.spec.ts`, `SearchDialog.tsx`, onboarding/FileUpload: `progressbar` semântico, axe serious/critical bloqueante em fixtures, foco aprisionado/restaurado, recursos pt-BR/en completos; validar nos viewports desktop/mobile e teclado.

### Fase D — confiabilidade de produto e verdade da matriz

- `project_import.go`/ingestion: estado individual por arquivo e contagem apenas de sucesso; diferenciar vazio, unsupported, binário, parser ausente e OCR não configurado.
- Research: separar busca por pergunta de fetch por URLs; sem adapter configurado, informar claramente limite (não simular search). Não fazer egress novo sem opt-in/allowlist.
- Spend: ligar reserva/cap do `SpendLedger` ao Gateway antes do egress e reconciliar uso real, atomicamente por organização, com testes concorrentes e fake provider; preservar free/local-first e nunca fallback pago silencioso.
- Deploy: persistir estados accepted/building/healthy/failed/partial, health polling e rollback remoto aprovado/idempotente; se provider não suportar rollback, declarar `rollback_unavailable`.
- Corrigir `PARITY_MATRIX.md` para MCP stdio tenant/local como `NOT_CONFIGURED`/bloqueado até launcher estrito integrado a uma missão; atualizar `SECURITY.md` para o comando de rotação HMAC já existente, mantendo produção bloqueada até ensaio operacional.

### Ordem de execução e gates

1. Implementar e testar A, revisão de diff; não habilitar publicação externa.
2. Implementar C em subtarefas independentes; Vitest/typecheck/lint/UI E2E.
3. Implementar B; unit tests locais, Redis/PostgreSQL integration tests com serviços descartáveis, race/vet/build.
4. Implementar D; testes de contrato com fakes, tenancy, orçamento e idempotência.
5. Atualizar auditoria/mission state e matriz de paridade; rodar integrity guard, diff-check, suites aplicáveis, push normal só para recovery e observar CI desse novo SHA.
6. Itens que requerem certificado, attestation identity, staging, dispositivo, serviços pagos ou hardware ficam `BLOCKED_EXTERNAL/NOT_EXECUTED`; não declarar tudo release-ready nem produção pronta.
