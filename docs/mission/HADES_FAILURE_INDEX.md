# Índice de falhas e limites — estágio 2 (gerado, não editar à mão)

> Gerado por `python scripts/hades_baseline_report.py` a partir de `HADES_MASTER_STATE.json`: um bloqueio que
> existe no estado aparece aqui, e um que sai do estado sai daqui. O `--check` no CI detecta divergência.

Total: **17** — 8 em aberto · 9 resolvidos.

## 1. Em aberto

| ID | Tipo | O que é |
| --- | --- | --- |
| B-02 | EXTERNO (MEDIDO: PARCIAL) | MEDIDO (E-204/E-205), nao suposto. O que o B-02 dizia ('conta real logada em Codex/Claude') esta parcialmente vencido: o CODEX esta INSTALADO e LOGADO (metodo chatgpt, tier account), o Gemini esta instalado SEM login, o OpenCode esta instalado com verificacao de login falhando, e Claude Code, Qwen Code, Copilot CLI e Cursor Agent estao AUSENTES. O que barra  |
| B-03 | EXTERNO | Tokens reais de canais (Telegram bot token, WhatsApp) |
| B-04 | EXTERNO | Provedor de telefonia/voz real |
| B-05 | EXTERNO | Contas reais de marketplace/social/pagamento |
| B-06 | EXTERNO | Identidade de assinatura de release (certificado). ESTADO MEDIDO (E-201): a autenticacao de NOTARIZACAO foi PROVADA com a chave de API do App Store Connect (JWT ES256, HTTP 200 em /v1/apps) - o que faltava ali esta resolvido no codigo (E-199) e provado na credencial. O que ainda falta e o certificado de ASSINATURA: a leitura de /v1/certificates com essa chav |
| B-07 | EXTERNO | Chaves de provedor de modelo para homologacao de chamada real |
| B-09 | AMBIENTAL (MEDIDO E CORRIGIDO, COM PROCEDIMENTO) | CORRIGIDO POR MEDICAO (E-202). A descricao anterior ('Host sem Python/sh: 4 testes falham por ambiente') estava errada em duas frentes: Python EXISTE no host (3.12.10) e bash esta no PATH (so `sh` nao); e o numero real e 45 funcoes de teste, nao 4. Causa raiz unica e medida: o host NAO TEM COMPILADOR C (gcc/cc ausentes) => CGO_ENABLED=0 => go-sqlite3 devolve |
| B-14 | AMBIENTE (DIAGNOSTICO RESOLVIDO) | LIMITE (E-184): a execucao de plugin exige PRIVILEGIO DE MONTAGEM - mesmo o modo best-effort faz mount/chroot (mount --make-rprivate, mount --bind, tmpfs e chroot/unshare em tools.go). Onde nao ha esse privilegio (Windows, container nao privilegiado, runner padrao de CI) a execucao nao acontece e o sintoma e 'exit status 1' com stdout e stderr VAZIOS - falha |

## 2. Resolvidos

| ID | O que era | Situação |
| --- | --- | --- |
| B-01 | RESOLVIDO em 2026-10-10: o usuario concedeu autorizacao de merge ampla. #71 e #73 mesclados individualmente; os outros 13 PRs das rodadas 6-13 foram integrados pelo merge train do PR #86 (23/23 checks verdes, main limpa) | resolvido, com teste contra regressão onde aplicável |
| B-10 | Conectores: 104 de 122 exigem configuracao do operador (available=17) || RESOLVIDO (E-200): verificado por leitura E por teste. O catalogo declara 123 entradas com vocabulario FECHADO (available 17, operator_setup_required 105, provider_selection_required 1) e `available` exige caminho de auth publicado - ja guardado em internal/agent. O que faltava era o gu | resolvido, com teste contra regressão onde aplicável |
| B-11 | 7 PRs do dependabot com CI vermelho, tratados um a um SEM merge cego: #26 (mobile-tooling: react-dom/react-native/react-native-web) e #34 (eslint-plugin-storybook 10) estavam VERDES apos rebase na main atual e foram integrados; #30 (tablewriter 1.1.5) foi CORRIGIDO por migracao de codigo (PR #90, 20/20 checks); #27 (typescript 7), #29 (expo-notifications 58) | resolvido, com teste contra regressão onde aplicável |
| B-12 | PR #31 (grupo web-dev-tooling) quebrava 'Web and mobile quality', 'test (ubuntu-latest)' e 'Web E2E': causa raiz provada no log do CI — o grupo subia react-test-renderer para 19.3.0 enquanto react/react-dom ficavam em 19.1.0, e o lockfile resolvia react=19.3.0 com react-dom=19.1.0 (React exige versoes EXATAS iguais). Corrigido no PR #92 alinhando react/react | resolvido, com teste contra regressão onde aplicável |
| B-13 | DEFEITO (E-183): a execucao de plugin falha no layout real de instalacao - plugin_runtime.go:215 passa Workspace = installation.ContentPath, e ContentPath e o ARQUIVO plugins/content/<org>/<id>-<versao>.zip (plugin_bundle.go:113); o executor trata Workspace como diretorio e falha com 'not a directory'. Instalacao e verificacao de conteudo funcionam; execucao | resolvido, com teste contra regressão onde aplicável |
| B-15 | ORACULO DE ENUMERACAO EM TODA A SUPERFICIE DE EMPRESA (E-189): o guarda compartilhado `companyForRequest` (server/company_routes.go), usado por ~55 pontos em 11 arquivos, devolve 404 para empresa INEXISTENTE e 403 para empresa de OUTRO tenant - a diferenca de status revela a existencia do ID a qualquer tenant autenticado, sem acessar dado nenhum. O vazamento | resolvido, com teste contra regressão onde aplicável |
| B-16 | MESMA CLASSE DE ORACULO DO B-15 EM OUTROS QUATRO MODULOS (E-191, scanner calibrado): methods GetForOrganization de builder (internal/agent/builder.go:352), devices (devices.go:210) e swarm (swarm.go:218), e TelAgentSessionForOrganization (tel_agent_session.go:181), devolvem o erro CRU da busca quando o recurso nao existe e um erro PROPRIO quando e de outro t | resolvido, com teste contra regressão onde aplicável |
| B-17 | CANDIDATO A ORACULO NAO VERIFICADO (E-192): internal/agent/mcp_remote.go:606 (`call`) devolve 'remote MCP server ... is not registered' ANTES do portao de organizacao, que devolve ErrPluginOrganizationScope - a mesma forma do B-15/B-16 (existencia distinguivel da negativa). NAO foi verificado com teste que compare as duas recusas, e a metodologia declarada e | resolvido, com teste contra regressão onde aplicável |
| B-18 | FALHA INTERMITENTE NA CI DA MAIN (E-193): TestServiceAgentAuditKeepsContactTextOut (DLP: conteudo do contato nao pode entrar na trilha de auditoria) falhou uma vez no passo 'CGO_ENABLED=1 go test ./... -count=1' do job dz23-agentic-quality, em bc3c708f6. NAO reproduzida em SEIS condicoes medidas: 10x isolado (Windows), 20x isolado sob CGO=1 (Linux/golang:1.2 | resolvido, com teste contra regressão onde aplicável |

## 3. Limites declarados

- O texto de cada limite é o do estado; este índice **não** julga se o desbloqueio é suficiente.
- Defeito corrigido **sem** teste que o impeça de voltar não entra como resolvido por conveniência: a coluna de
  situação diz onde há teste.
- Este arquivo é um **índice**: a evidência de cada item está em `HADES_EVIDENCE_INDEX.md` e nos documentos de
  estágio citados lá.
