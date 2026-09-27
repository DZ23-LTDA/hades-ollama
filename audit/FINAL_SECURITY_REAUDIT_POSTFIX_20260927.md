# Relatório final de reauditoria de segurança — Ollama Full

**Data:** 27/09/2026

**Tipo:** auditoria pós-remediação, somente leitura
**Escopo:** implementação Go/Gin em `/home/ubuntu/ollama-full-recovery`

## Resumo executivo

As três revisões independentes confirmam que várias remediações importantes estão presentes e testadas, especialmente nos caminhos de Git controlado, snapshots, integridade de artefatos, descritores de ferramentas, filas locais/Redis, persistência e DLP comum. **Contudo, o candidato ainda requer mudanças antes do release.**

Foi reproduzido um bypass de execução no `terminal.exec`: `git` é resolvido pelo `PATH` do processo pai durante `exec.Command`, antes de o `PATH` restrito ser atribuído ao processo filho. Também permanecem exposições de catálogo MCP entre organizações, URLs de provedor com parâmetros assinados, chaves JSON de assinatura não cobertas pelo DLP, crescimento/trabalho não limitados do outbox e uma fuga de metadados em listagens de jobs inconsistentes.

O PostgreSQL RLS **não está corrigido**: o contexto de organização baseado em GUCs pode ser forjado pelo papel de banco. O servidor público atualmente contém o risco ao recusar PostgreSQL na inicialização/construção da API; isso é contenção, não uma correção da fronteira de isolamento. PostgreSQL deve continuar desabilitado até existir contexto não forjável e prova de integração.

## Correções verificadas

- **Git de inspeção e snapshots:** uso de resolvedor absoluto confiável, diretórios de ferramentas seguros, ambiente Git restrito (`GIT_CONFIG_NOSYSTEM=1`, configuração global nula, lazy fetch/optional locks desabilitados), argumentos somente leitura, limites de saída e, no Linux, `cwd` ancorado por descritor.
- **Snapshots e fontes:** rejeição de `.git`/metadados por ponteiros ou symlinks perigosos, filtros externos, escapes de diretório pai e substituições de caminho; verificação adicional por identidade, tamanho e hash.
- **Execução isolada e planejamento:** validação de snapshot/manifesto, uso de `os.Root`, desativação explícita de ferramentas pathname-based em snapshots isolados e rejeição de drift do `ToolDescriptor` antes da execução.
- **Artefatos e entrega HTTP:** manifests com arquivos regulares e contenção de workspace; releitura ancorada, verificação de tamanho/SHA-256 e limpeza pós-entrega do snapshot temporário verificado.
- **Configuração/ownership:** cópia protegida por mutex, checagens de organização em deploy, cópia de coleções MCP, validação de nomes de headers/variáveis e rejeição de headers restritos.
- **Egress e redaction já cobertos:** validação fail-closed de payloads aninhados/JSON escapado, limites de forma/tamanho e redaction de chaves e URLs sensíveis comuns, incluindo variantes normais de `signature`/`x.sig` em URLs.
- **Filas e persistência:** locks/flock, persistência atômica, CAS e cópias profundas no JSONStore; leases com fencing token, heartbeat, reclaim, backoff e transições Lua atômicas no Redis; SecretStore recarregado e persistido sob lock; replay de jobs já proprietários e execução rejeitam organização estrangeira.
- **Contenção PostgreSQL:** o servidor público recusa configuração PostgreSQL e runtimes PostgreSQL antes de servir rotas, independentemente de autenticação. Isso foi verificado como fail-closed, mas não torna o RLS seguro.

## Bloqueadores remanescentes

1. **`terminal.exec` permite substituição ambient-PATH de `git` — bloqueador de merge.** Em `internal/agent/tools.go:241-260`, o executável bare é resolvido antes de `command.Env` receber `PATH=/usr/bin:/bin`. A allowlist aceita `git` e `git status` é válido. A reprodução colocou um `git` falso no `PATH` pai; o marcador foi criado (`err=<nil> marker=true`). Corrigir com caminho absoluto resolvido por allowlist confiável antes de construir o comando, ou remover `git` desse surface, e adicionar teste de regressão.

2. **Listagem autenticada expõe configurações MCP ownerless/global — bloqueador de release.** `ListForOrganization` inclui configurações sem owner, enquanto `CallForOrganization` as rejeita. Assim, um tenant pode enumerar metadados de configuração de outro contexto (`URL`, `TokenEnv` e `HeadersEnv`). Exigir ownership organizacional para listagens autenticadas e retornar somente uma projeção sanitizada.

3. **URLs de provedor, inclusive query strings assinadas, são retornadas integralmente — bloqueador de release.** Listas de deployments/MCP devolvem `BaseURL`/`URL` sem sanitização; uma URL como `...?x.sig=provider-secret` reteve o segredo. Não serializar URLs brutas; retornar apenas origem não secreta ou aplicar um sanitizador central, e rejeitar credenciais em query salvo allowlist estrita.

4. **Chaves de assinatura escapam do DLP em resultados bem-sucedidos — bloqueador de release.** `sensitiveDLPKey` não cobre `signature`, `sig`, `x.sig`, `x-signature`, `provider_signature` e `auth_signature` em mapas JSON. Sondas in-package reproduziram a retenção desses valores em respostas MCP. Expandir o normalizador canônico, incluindo aliases de prefixo/sufixo e variantes normalizadas, com testes recursivos.

5. **Outbox de push sem quota nem limite de trabalho — bloqueador de release.** `Enqueue` cresce indefinidamente por itens/bytes; cada claim regrava o snapshot completo; o flush pode processar toda a fila e registros falhos permanecem até `1<<20` tentativas. Adicionar quotas agregadas e por organização, batch/tempo/cancelamento, persistência incremental/indexada, compactação e métricas de backlog.

6. **Listagem de jobs não exige concordância entre owner do job e owner da missão — bloqueador de release.** `QueueJobs`/`QueueJobsForOrganization` filtram pelo owner da missão, mas podem devolver um job persistido com `OrganizationID` divergente, vazando IDs/metadados. Execução e replay rejeitam o caso, mas isso não elimina a fuga de listagem. Exigir igualdade, tratar registros inconsistentes como quarentena/reparo e testar local/Redis.

7. **PostgreSQL RLS é um bloqueador arquitetural condicional.** As políticas usam `current_setting('app.current_organization_id')` e `app.system_access`, valores que o próprio papel de banco pode definir. Um papel comprometido pode forjar tenant ou acesso sistêmico. **Não declarar corrigido:** manter PostgreSQL desabilitado no produto público e só habilitar após papel/runtime separado e contexto não forjável (por exemplo, função/contexto controlado com privilégios revogados), mais teste de integração que prove a impossibilidade de forja.

### Recomendação adicional (não classificada como bloqueador pelos revisores)

Respostas de deploy HTTP `200` acima de 4 MiB são truncadas; JSON inválido/truncado pode retornar sucesso com identidade vazia (`internal/agent/deploy.go:477-488`). Ler `max+1`, retornar erro estável de tamanho e tratar JSON inválido como falha, preservando semântica de estado parcial.

## Testes e evidências

- **Passaram:** `go test ./internal/agent -count=1`; suítes focadas de Deployment, RemoteMCP, PushOutbox e RuntimeFlush; testes com race de RemoteMCP/persistência/deployment; DLP/redaction/provider; filas, `RuntimeWithOrganization`, JSONStore e SecretStore; guards de PostgreSQL/API/auth; vet de `internal/agent` e `server`.
- **Passaram:** testes de leases/transições Redis e políticas de TLS/prefixo; teste isolado de replay que vincula e impõe owner da organização.
- **Reproduzidos:** execução de `git` falso via PATH pai; retorno de URL MCP/deploy com segredo em query; retenção de `signature`, `sig`, `x.sig`, `x-signature`, `provider_signature` e `auth_signature`; listagem ownerless; crescimento sem quota do outbox; listagem de job com owner divergente; GUC PostgreSQL forjável em transação.
- **Não executados:** integrações reais PostgreSQL/Redis, pois as variáveis `OLLAMA_AGENT_TEST_POSTGRES_URL` e `OLLAMA_AGENT_TEST_REDIS_URL` não estavam definidas.
- **Execuções bloqueadas pelo ambiente:** uma tentativa focada encontrou `probe-ZZYR_test.go` ausente no estado compartilhado; outra foi impedida por constraints/compilação MLX (`xgrammar` sem Go files e símbolos `mlx` indefinidos). Isso não deve ser interpretado como aprovação ou reprovação funcional dos caminhos afetados.

## Veredictos dos revisores

- **Revisor 1 — OpenAI security review subagent:** `REQUEST CHANGES`; corrigir resolução absoluta do `git` em `terminal.exec` e adicionar regressão. Os demais controles Git/snapshot/artefato/descritor foram considerados materialmente fortes.
- **Revisor 2 — segurança/concurrency:** `REQUEST_CHANGES`; bloquear até corrigir escopo/sanitização dos catálogos, aliases de assinatura no DLP e limites do outbox. Indicou também a correção recomendada para respostas de deploy truncadas.
- **Revisor 3 — arquitetura/segurança:** `REQUEST CHANGES` para qualquer implantação PostgreSQL; RLS continua um risco real, embora o servidor público falhe fechado. Para JSON local e Redis, os caminhos atuais foram considerados defensáveis após os testes focados, condicionados ao endurecimento da listagem de jobs.

## Veredicto consolidado

**Não liberar o candidato atual.** Corrigir os seis defeitos de execução/catalogação/DLP/outbox/listagem acima e repetir os testes focados em checkout limpo. Manter PostgreSQL explicitamente desabilitado até substituir o contexto GUC por mecanismo não forjável e executar integração real. Após essas mudanças e evidência de regressão, reavaliar o release; as correções já verificadas não justificam afirmar que os bloqueadores remanescentes foram resolvidos.


## Atualização de status após remediação (2026-09-27)

Este documento preserva o relatório original e seus reproductions. Os itens 1–6 e a recomendação de deploy das seções anteriores foram remediados no checkout atual com testes de regressão: (1) `terminal.exec` resolve caminhos absolutos confiáveis e rejeita Git genérico; (2) catálogos tenant-facing ocultam recursos sem owner/foreign e omitem mappings de credenciais; (3) URLs são projetadas apenas como origem e DLP cobre aliases como `signature`, `sig`, `x.sig` e formas normalizadas; (4) PushOutbox aplica quotas, limites de tamanho/batch/tempo e preserva leases ativos; (5) listagem de jobs requer concordância de owner do job e da missão; (6) responses de deploy rejeitam JSON malformado/oversized e sucesso sem identidade, inclusive Vercel e generic. O MCP stdio local agora também DLP-redige JSON de sucesso, rejeita `result` ausente e não ecoa texto de erro do provider. Git/terminal e store local de queue receberam endurecimento adicional descrito em `SECURITY.md` e na matriz de paridade. Os nomes dos testes e evidências definitivas devem ser consultados no checkpoint atualizado e no log de gates mais recente.

**Não equivale a liberação:** o item 7 continua um bloqueio real; PostgreSQL permanece recusado/fail-closed porque GUC RLS continua forjável e não há roles/contexto tenant não-forjáveis implementados. Uma terceira auditoria independente está em andamento para validar essas correções e procurar achados novos; não substituir o veredicto original por aprovação até que ela termine e os gates finais pós-alterações passem.
