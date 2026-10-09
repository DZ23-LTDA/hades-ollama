# Security

The Ollama maintainer team takes security seriously and will actively work to resolve security issues.

## Reporting a vulnerability

> **Escolha o canal certo pelo componente afetado.** Este repositório é o fork
> **Hades** (DZ23), com uma superfície agentic própria sobre o Ollama.

### A) Vulnerabilidade no núcleo do Ollama upstream

Se a falha está no motor Ollama original (não na camada agentic/Hades),
reporte ao canal do upstream, **não** a este repositório: envie para
hello@ollama.com. Dê tempo hábil para investigação antes da divulgação pública.

### B) Vulnerabilidade específica do Hades (este fork)

Se a falha está na camada Hades — runtime agentic, connectors, MCP, deploy,
Company OS, HarnessRouter, companion, UI/mobile deste repositório — **não abra
issue pública** e **não** use hello@ollama.com. Reporte de forma privada por:

1. **GitHub Private Vulnerability Reporting** — aba **Security** deste
   repositório → **Report a vulnerability** (recomendado; cria um advisory
   privado com histórico e correção coordenada).
2. Se o botão acima não aparecer, o recurso ainda não foi habilitado pelo
   mantenedor (veja a nota abaixo); nesse caso, contate de forma privada o
   mantenedor do repositório público **DZ23-LTDA/hades-ollama** pelo
   GitHub para combinar um canal privado antes de qualquer divulgação.

Inclua sempre: descrição, passos de reprodução, impacto avaliado, mitigações
possíveis, e a **referência do commit + arquivo afetado**. Nunca coloque
credenciais reais em relatórios, issues, PRs, screenshots, fixtures ou logs.

> **Nota de habilitação:** o *GitHub Private Vulnerability Reporting* deste fork
> foi **habilitado em 2026-09-24**. Use o item B.1 (aba **Security → Report a
> vulnerability**). Não foi criado nenhum e-mail de segurança específico do fork.

Please include the following details in your report:
- A description of the vulnerability
- Steps to reproduce the issue
- Your assessment of the potential impact
- Any possible mitigations

## Security best practices

While the maintainer team does its best to secure Ollama, users are encouraged to implement their own security best practices, such as:

- Regularly updating to the latest version of Ollama
- Securing access to hosted instances of Ollama
- Monitoring systems for unusual activity

## Contact

For any other questions or concerns related to security, please contact us at hello@ollama.com

## Hades

O runtime agentic pode executar ferramentas, acessar conectores, controlar companions, manipular arquivos e publicar builders. Em instalações públicas, mantenha `OLLAMA_AGENT_AUTH_REQUIRED=true`, use TLS/mTLS quando houver dispositivo remoto, mantenha tokens em um secrets manager e não habilite modos `DEV` ou `ALLOW_INSECURE` fora de loopback.

**PostgreSQL multi-tenant está em implementação candidata e ainda não aprovado para declarar produção/isolamento enterprise.** O novo caminho substitui os GUCs confiáveis anteriores por `app.tenant_context`: valor com HMAC, tenant e expiração curta, verificado pela policy RLS por função `SECURITY DEFINER` com `search_path` fixo e segredo em tabela sem `SELECT` para runtime. O startup usa apenas a role `ollama_agent_runtime` (sem ownership, `CREATE`, superuser ou `BYPASSRLS`); DDL ocorre somente via `ollama agent migrate-postgres` e DSN separada `OLLAMA_AGENT_MIGRATOR_DATABASE_URL`. A migration verifica que as tabelas pertencem ao migrator, trava as tabelas e recusa ownership de tenant ambíguo, eventos órfãos ou owner divergente em vez de inferir backfill. Eventos estão vinculados ao par `(mission_id, organization_id)` por FK e policy RLS checa a missão pai. A recuperação de jobs/agenda percorre IDs da `AuthStore` confiável usando runtimes por tenant; não concede enumeração global à role SQL. Volumes legados exigem janela de manutenção, backup validado e execução administrativa de `deploy/postgres/migrate-existing-roles.sql`; o processo antigo deve ser parado e a role privilegiada antiga é desativada pelo procedimento antes do migrator/runtime entrarem em operação. A chave HMAC (`OLLAMA_AGENT_TENANT_CONTEXT_KEY`) precisa permanecer secreta, persistente e idêntica no migrator/runtime. **A rotação é feita pelo comando `ollama agent rotate-postgres-key`** (implementado em `internal/agent/postgres_key_rotation.go`): ele faz fencing dos logins de runtime, aguarda a quiescência das sessões e troca a chave de forma transacional, usando versões de chave. As variáveis envolvidas são `OLLAMA_AGENT_TENANT_CONTEXT_KEY_VERSION` (versão atual), `OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT` + `OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT_VERSION` (chave/versão nova) e a DSN administrativa `OLLAMA_AGENT_POSTGRES_ADMIN_DATABASE_URL` (além da `OLLAMA_AGENT_MIGRATOR_DATABASE_URL`). Mesmo com o comando, **não** troque a variável isoladamente nem execute migrations com outra chave fora dele; isso provoca indisponibilidade. O uso em produção compartilhada permanece bloqueado até que essa rotação seja **ensaiada e registrada em janela de manutenção de staging** — com quiescência dos processos, verificação e rollback — e até existir um runbook documentado de chave comprometida: o comando existe, o ensaio operacional e o runbook são os gates pendentes.

Os testes adversariais locais em PostgreSQL 16 e Redis 7 verificaram SQL raw para spoofing de tenant/evento, chave errada, leitura da chave, `SET ROLE`, DDL, `row_security=off`, migração sem owner e recuperação tenant-scoped. Isso **não prova ainda production-readiness**: antes de ativar serviço compartilhado, repetir e registrar as verificações em staging com política TLS, dados/backups representativos e plano de restore; revisar tenant provisioning/auth directory, concorrência/race e implementar/testar um procedimento de rotação/recuperação; obter aprovação da auditoria independente final. O usuário do processo Hades conhece a chave HMAC e, se esse processo for integralmente comprometido, pode assinar contextos de qualquer tenant; o mecanismo protege a boundary SQL contra role runtime/SQL arbitrário sem esse segredo, não contra comprometimento completo do processo. Não exponha DSNs/segredos nem marque RLS como certificada antes dos gates.

A camada agentic executa inspeção DLP antes dos egressos cobertos e redige tokens reconhecidos, propriedades JSON sensíveis e erros/resultados de ferramentas antes de persistir ou serializar. Propriedades sensíveis encontradas dentro de strings livres causam ocultação da string inteira. A inspeção decodifica escapes Unicode e camadas de JSON string até um limite fixo de oito passos; nomes de propriedade acima de 256 bytes e entradas além do limite falham fechados. **Isso é uma defesa heurística, não classificação semântica nem garantia de que dados pessoais/comerciais desconhecidos serão identificados**; aplique minimização de dados, approvals e controle de destino, e não trate a redação como substituta de uma política de dados do deployment.

Em missões isoladas no Linux, `sandbox.exec` reutiliza o `os.Root` já aberto pelo Runtime e passa o diretório aprovado ao namespace por descritor herdado, não por novo lookup do pathname. Isso protege contra substituição do caminho após a abertura; não converte o modo best-effort em isolamento estrito. O fallback não-Linux continua best-effort/path-based e não reivindica essa mesma garantia de ancoragem. O modo `strict` continua dependente de namespaces, seccomp e cgroup v2 delegado, e falha fechado se esses controles não estiverem disponíveis.

Snapshots físicos ancoram o `DataRoot`, diretórios de tenant/mission/snapshot, árvore `tree`, arquivos temporários, rename do conteúdo e escrita do manifesto em handles `os.Root`. No Linux, a enumeração Git usa o diretório-fd herdado e a troca do pathname da origem é detectada antes da cópia; fora do Linux, o fallback Git continua path-based. Uma substituição do pathname depois de abrir a árvore não redireciona a cópia para o destino substituto; rollback/remove ficam confinados ao handle do `DataRoot`, e pais vazios criados pela operação só são removidos quando tracked e ainda vazios. A regressão `TestCopyWorkspaceSnapshotFileRemainsAnchoredAfterDestinationSwap` cobre a troca após a abertura do destino. Isso protege a operação do runtime contra corrida de pathname, mas não contra um processo comprometido com o mesmo UID que possa ler diretamente os dados privados do processo.

Todos os `Runtime.WithOrganization` recebem um decorator de Store que verifica ownership de missão em leituras, listas, gravações e eventos também para JSON/memory fallback; PostgreSQL acrescenta RLS e escopo transacional. Os endpoints de métricas agregadas são ocultados quando a autenticação multi-organização está habilitada, pois os contadores atuais não são particionados por tenant. `JSONStore` armazena em cache, persiste e devolve cópias redigidas de missões; na inicialização, arquivos legados de missão/evento são redigidos e regravados sob lock. Tokens de push persistidos usam a mesma cifra AES-GCM protegida por `OLLAMA_AGENT_CREDENTIAL_KEY`; arquivos legados em texto claro são migrados na inicialização. A chave externa precisa estar configurada e estável antes de abrir subscriptions cifradas; sem ela, a inicialização falha fechada em vez de perder ou expor tokens.

Vulnerabilidades envolvendo SSRF, path traversal, bypass de approval, fuga entre organizações, exposição de tokens, execução fora da sandbox ou publicação não autorizada devem ser tratadas como alta prioridade. Não coloque credenciais reais em issues, PRs, screenshots, fixtures ou logs. Para o código específico desta distribuição, encaminhe também a referência do commit e o arquivo afetado ao mantenedor do repositório público.

### Inspeção Git em missões isoladas

`git.repo.inspect` em missão isolada exige um manifesto de snapshot validado e não executa Git contra o repositório de origem nem expõe o caminho host, o índice ou metadados `.git`. Seu baseline é o início da missão; as alterações reportadas são somente artefatos persistidos pela missão, como os produzidos por `workspace.write`/`workspace.patch`. A resposta marca `change_scope=persisted_mission_artifacts_only` e `changes_complete=false`. Em particular, alterações feitas por execução arbitrária em `sandbox.exec` não são garantidamente incluídas; consumidores não devem tratar o resultado como status/diff completo nem usá-lo como aprovação para integrar alterações. O inspetor é somente leitura e não cria branch/worktree nem aplica mudanças à origem.

### Persistência e limpeza de snapshots, eventos e notificações

O runtime mantém handles `os.Root` do armazenamento e da árvore isolada durante o handoff da missão; valida as identidades contra os caminhos autorizados, cria o workspace relativamente ao handle da árvore e lê o manifesto pelo handle do snapshot. A limpeza de rollback e o orphan sweeper também operam em diretórios ancorados, inclusive ao enumerar/remover snapshots, e não seguem substituições por symlink do pathname. Em Unix, diretórios de snapshot recebem modo privado pelo descritor aberto; no Windows, cada diretório recebe DACL protegida para o usuário proprietário. A regressão cobre substituição de parent/árvore e o test binary compila para Windows; isso não substitui execução nativa Windows nem protege contra outro processo já comprometido com o mesmo usuário.

Motivos de decisão de approval passam pela redação DLP antes de persistência, evento e retorno da operação. JSONStore clona recursivamente estruturas mutáveis, redige missões/eventos antes de cache e saída e migra registros legados sob lock; gravações persistentes de missão validam owner e versão sob o mesmo lock cross-process que protege a escrita. Push outbox revalida o payload legado sob DLP ao carregar e serializa reload/read-modify-write; tokens push são cifrados em repouso, IDs incluem tenant/usuário/token, e migração sem chave correta falha fechada. Esses controles continuam dependentes de filesystem confiável e da proteção operacional de `OLLAMA_AGENT_CREDENTIAL_KEY`.

O endpoint do push é parseado como URL absoluta: HTTP só é aceito para `localhost` exato ou IP loopback; HTTPS é obrigatório para outros destinos, userinfo é rejeitado, proxies de ambiente são desativados, todos os IPs DNS são validados e o transporte conecta somente ao IP aprovado, confirmando o peer. Redirects não são seguidos. Uma resolução mista ou uma resposta que escape do loopback é recusada antes da conexão. Erros do transporte/provider retornam mensagens genéricas ou status HTTP sem ecoar URL/body não confiáveis. Revogar uma subscription é tenant/user-bound e cada envio revalida o estado corrente imediatamente antes do request; uma requisição que já começou não pode ser desfeita. Push outbox usa um UUID de fencing por claim; `Complete`/`Fail` recusam claims antigos depois que outro worker recupera o lease. Isso protege o estado persistido contra acknowledgement/retry obsoleto, mas entrega externa continua **at-least-once**: um processo pausado pode ter enviado ao provider antes da expiração e outro worker pode reenviar; o provider deve ser idempotente se duplicatas forem inaceitáveis.

Em modo multi-organização, providers de deploy precisam ser vinculados por `organization_id`; catálogos, requests de approval e execuções não aceitam providers globais/sem owner, inclusive pelo método não scoped do manager. IDs estrangeiros de approvals e do lifecycle de plugins retornam o mesmo 404 genérico que IDs desconhecidos. Connector, mídia, deploy, MCP remoto e push validam todos os IPs DNS antes de conectar, fixam o socket ao IP aprovado e comparam o peer. A exceção de loopback exige que cada resposta resolvida e o peer sejam efetivamente loopback; o Connector e a mídia também rejeitam URL com userinfo. Transcrição e visão redigem DLP antes de gravar os próprios arquivos de artifact, não somente o valor de retorno. Respostas não-2xx dos providers não são transformadas em valores de missão/erros com o corpo arbitrário; mensagens de transporte e diagnósticos upstream são genéricos, preservando somente status seguro quando aplicável. Essa regra não substitui minimização de dados, proteção de headers/env ou isolamento dos próprios logs operacionais.

O inspetor Git direto ancora o `os.Root` e, em Linux, executa Git com cwd apontado ao descritor aberto; fora de Linux, a execução ainda usa o caminho e não possui a mesma garantia contra troca concorrente de diretório. O snapshot isolado também possui execução Git descritor-bound em Linux e fallback path-based nos demais sistemas. Escritas aprovadas serializam writers cooperativos, verificam hashes e usam rename atômico, mas não há CAS universal contra um processo externo que modifique os mesmos arquivos sem respeitar o lock; patch compensation é best-effort e não crash-atomic. Manifests e snapshots verificam hash/identidade em momentos definidos, mas não tornam um workspace modificável por outro processo imutável.


### Catálogos autenticados, execução allowlisted e limites de push

Os catálogos MCP, Connector e Deployment destinados a uma organização retornam somente recursos com `organization_id` igual à organização autenticada; entradas globais/sem owner ficam fora desse caminho. URLs de catálogo são projeções display-only de origem: path, query, fragmento e userinfo não são devolvidos; variáveis de ambiente usadas para credenciais e headers não são expostas no catálogo tenant-facing. A configuração operacional completa permanece privada ao manager.

`terminal.exec` só aceita nomes/binários allowlisted e resolve o executável por caminho absoluto em diretórios fixos confiáveis antes de construir o processo; o `PATH` herdado do serviço não seleciona o binário. O teste de regressão coloca um `git` falso no PATH pai e comprova que ele não é executado.

O push outbox tem limites explícitos: 256 itens globais, 64 por organização, 64 KiB por registro, 20 MiB por arquivo persistido, oito tentativas e até 32 envios/30 segundos por flush; itens terminais são removidos após sete dias. Quota excedida falha fechado. O arquivo continua sendo um snapshot JSON atômico, portanto limites reduzem o custo máximo mas não equivalem a uma fila incremental/índice.

Respostas 2xx de deployment são limitadas a 4 MiB e devem ser JSON object válido; payload malformado, oversized ou resposta genérica sem identidade/URL resulta em erro estável sem ecoar corpo upstream. Esse endurecimento não remove efeitos que o provider já tenha aplicado antes de devolver uma resposta inválida.


### Identidade de workspace, Git e snapshots de filas

Missões não isoladas guardam a identidade estável do diretório autorizado no momento da criação (device/inode em Unix; volume/file index no Windows). Ao iniciar cada execução, o runtime reabre e compara essa identidade; missões legadas sem identidade ou com caminho substituído falham fechado e precisam ser recriadas para reautorizar o workspace. O handle aberto é entregue às ferramentas de workspace; no Linux, Git usa cwd via descritor. O inspetor rejeita symlinks/arquivos especiais na árvore `.git` e ponteiros externos `alternates`, `http-alternates` e `commondir`. Git não está disponível por `terminal.exec`; esse comando é uma allowlist pequena, exige root fixado e, nesta versão, só executa em Linux. O inspetor Git normal continua disponível nos sistemas suportados, sujeito às ressalvas de cwd path-based descritas acima fora de Linux.

O snapshot local de jobs valida owner/estado/contadores de retry ao carregar, rejeita `MaxAttempts` fora de 1–20 e limita `jobs.json` a 20 MiB e 100.000 registros; os limites são fail-closed para arquivos legados fora do contrato. O Remote MCP SSE aplica limite agregado de 4 MiB também a comentários e eventos com IDs não correlacionados, não apenas ao evento que contém a resposta solicitada. O push outbox não compacta um retry terminal enquanto existir lease ainda válido e valida quotas tanto no carregamento quanto antes da persistência.


Respostas JSON-RPC de sucesso do MCP stdio local são decodificadas e recursivamente sanitizadas por DLP antes de voltar ao Runtime; `result` ausente ou malformado falha fechado e mensagens de erro não confiáveis do provider não são ecoadas. Vercel e providers genéricos também não reportam sucesso sem ao menos ID ou URL de deployment. Esses guards são complementares à contenção de erro/body já descrita e não revertem efeitos que um provider possa ter aplicado antes de retornar payload inválido.
