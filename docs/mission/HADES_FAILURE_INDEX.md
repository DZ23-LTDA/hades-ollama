# Índice de falhas e limites — estágio 2 (gerado, não editar à mão)

> Gerado por `python scripts/hades_baseline_report.py` a partir de `HADES_MASTER_STATE.json`: um bloqueio que
> existe no estado aparece aqui, e um que sai do estado sai daqui. O `--check` no CI detecta divergência.

Total: **15** — 10 em aberto · 5 resolvidos.

## 1. Em aberto

| ID | Tipo | O que é |
| --- | --- | --- |
| B-02 | EXTERNO | Conta real logada em Codex/Claude para homologar harnesses |
| B-03 | EXTERNO | Tokens reais de canais (Telegram bot token, WhatsApp) |
| B-04 | EXTERNO | Provedor de telefonia/voz real |
| B-05 | EXTERNO | Contas reais de marketplace/social/pagamento |
| B-06 | EXTERNO | Identidade de assinatura de release (certificado) |
| B-07 | EXTERNO | Chaves de provedor de modelo para homologacao de chamada real |
| B-09 | AMBIENTAL | Host sem Python/sh e limites de caminho: 4 testes falham por ambiente |
| B-10 | PRODUTO | Conectores: 104 de 122 exigem configuracao do operador (available=17) |
| B-14 | AMBIENTE (DIAGNOSTICO RESOLVIDO) | LIMITE (E-184): a execucao de plugin exige PRIVILEGIO DE MONTAGEM - mesmo o modo best-effort faz mount/chroot (mount --make-rprivate, mount --bind, tmpfs e chroot/unshare em tools.go). Onde nao ha esse privilegio (Windows, container nao privilegiado, runner padrao de CI) a execucao nao acontece e o sintoma e 'exit status 1' com stdout e stderr VAZIOS - falha |
| B-16 | PRODUTO (SEGURANCA) | MESMA CLASSE DE ORACULO DO B-15 EM OUTROS QUATRO MODULOS (E-191, scanner calibrado): methods GetForOrganization de builder (internal/agent/builder.go:352), devices (devices.go:210) e swarm (swarm.go:218), e TelAgentSessionForOrganization (tel_agent_session.go:181), devolvem o erro CRU da busca quando o recurso nao existe e um erro PROPRIO quando e de outro t |

## 2. Resolvidos

| ID | O que era | Situação |
| --- | --- | --- |
| B-01 | RESOLVIDO em 2026-10-10: o usuario concedeu autorizacao de merge ampla. #71 e #73 mesclados individualmente; os outros 13 PRs das rodadas 6-13 foram integrados pelo merge train do PR #86 (23/23 checks verdes, main limpa) | resolvido, com teste contra regressão onde aplicável |
| B-11 | 7 PRs do dependabot com CI vermelho, tratados um a um SEM merge cego: #26 (mobile-tooling: react-dom/react-native/react-native-web) e #34 (eslint-plugin-storybook 10) estavam VERDES apos rebase na main atual e foram integrados; #30 (tablewriter 1.1.5) foi CORRIGIDO por migracao de codigo (PR #90, 20/20 checks); #27 (typescript 7), #29 (expo-notifications 58) | resolvido, com teste contra regressão onde aplicável |
| B-12 | PR #31 (grupo web-dev-tooling) quebrava 'Web and mobile quality', 'test (ubuntu-latest)' e 'Web E2E': causa raiz provada no log do CI — o grupo subia react-test-renderer para 19.3.0 enquanto react/react-dom ficavam em 19.1.0, e o lockfile resolvia react=19.3.0 com react-dom=19.1.0 (React exige versoes EXATAS iguais). Corrigido no PR #92 alinhando react/react | resolvido, com teste contra regressão onde aplicável |
| B-13 | DEFEITO (E-183): a execucao de plugin falha no layout real de instalacao - plugin_runtime.go:215 passa Workspace = installation.ContentPath, e ContentPath e o ARQUIVO plugins/content/<org>/<id>-<versao>.zip (plugin_bundle.go:113); o executor trata Workspace como diretorio e falha com 'not a directory'. Instalacao e verificacao de conteudo funcionam; execucao | resolvido, com teste contra regressão onde aplicável |
| B-15 | ORACULO DE ENUMERACAO EM TODA A SUPERFICIE DE EMPRESA (E-189): o guarda compartilhado `companyForRequest` (server/company_routes.go), usado por ~55 pontos em 11 arquivos, devolve 404 para empresa INEXISTENTE e 403 para empresa de OUTRO tenant - a diferenca de status revela a existencia do ID a qualquer tenant autenticado, sem acessar dado nenhum. O vazamento | resolvido, com teste contra regressão onde aplicável |

## 3. Limites declarados

- O texto de cada limite é o do estado; este índice **não** julga se o desbloqueio é suficiente.
- Defeito corrigido **sem** teste que o impeça de voltar não entra como resolvido por conveniência: a coluna de
  situação diz onde há teste.
- Este arquivo é um **índice**: a evidência de cada item está em `HADES_EVIDENCE_INDEX.md` e nos documentos de
  estágio citados lá.
