# Auditoria do Hades instalado — Codex — 2026-10-03

## Resumo executivo

O runtime instalado ficou operacional após a atualização ocorrida durante a auditoria. A versão final observada foi `0.2.0-rc.1-237-geb245f5`; os processos `ollama app.exe` e `ollama.exe` permaneceram ativos e a API local voltou a responder normalmente em `127.0.0.1:11434`.

Não foi encontrada evidência de crash persistente no Windows. Foram encontrados dois defeitos reproduzíveis na infraestrutura de qualidade do repositório e uma inconsistência de desenvolvimento que pode produzir falsos erros de conexão.

## Escopo e método

- Inspeção dos processos instalados e da porta local do Hades.
- Exercício de 14 rotas da interface atual contra o backend real, sem enviar mensagens, alterar configurações ou executar ações externas.
- Verificação visual desktop e mobile `390x844`.
- Provas diretas dos endpoints locais principais.
- Leitura de eventos do Windows e do log local do proxy, sem expor segredos.
- Execução de build, lint, Vitest, testes Go e Playwright.

Limitação: o controlador desta sessão não expôs a janela nativa do Windows. A navegação foi feita pela superfície HTTP equivalente conectada ao mesmo backend. A UI embutida exata do executável não pôde ser automatizada diretamente.

## Evidências positivas

- API local: `GET /api/version`, `GET /api/tags`, `POST /api/me`, `GET /api/v1/settings`, `GET /api/v1/integrations`, missões, rotinas, projetos e skills responderam HTTP 200.
- Estabilidade da rota de configurações: 30 leituras sequenciais, 0 falhas, média de 4,2 ms e p95 de 3 ms após o reinício.
- Interface: Home, Computadores, Agents, Biblioteca, Criações, Studio, Automações, Plugins, Habilidades, Empresa, Tarefas, Projetos, Configurações e Harnesses & Codex renderizaram.
- Mobile: a Home ficou operável em `390x844`, com menu recolhido, composer, seletor de modelo e sugestões visíveis.
- Logs: 0 eventos Warning/Error/Critical do Hades/Ollama no log Application do Windows nos sete dias consultados; 0 linhas de erro no `codex-proxy.log` consultado.
- Vitest: 58 arquivos e 328 testes aprovados.
- Go: `go test ./server ./internal/agent` aprovado.
- ESLint: aprovado.
- Build Vite/TypeScript: aprovado.
- Bundle medido manualmente: `index-DjQpazbe.js`, 1.292.706 bytes bruto e 355.350 bytes gzip, dentro dos limites declarados de 2.000.000/600.000 bytes.

## Achados reproduzíveis

### AUD-CX-01 — `build:check` quebra no Windows

Severidade: média. Impacto: gate local/CI de orçamento de bundle não funciona no Windows, apesar de o bundle estar dentro do orçamento.

Comando:

```powershell
cd D:\IA\Trabalhos\DZ23-LTDA\ollama-classe-a-plus\app\ui\app
npm run build:check
```

Falha:

```text
ENOENT: no such file or directory, open 'D:\D:\IA\Trabalhos\...\dist\assets\abap-....js'
```

Causa localizada em `app/ui/app/scripts/check-bundle-budget.mjs:16`: `new URL(assetsDir).pathname` produz um pathname iniciado por `/D:/...`; `join()` o transforma em `D:\D:\...` no Windows. Deve-se usar `fileURLToPath(assetsDir)` ou `new URL(name, assetsDir)` diretamente.

### AUD-CX-02 — suíte Playwright está desatualizada e instável

Severidade: alta para prevenção de regressões; não comprova falha equivalente no runtime.

Resultado: 4 falhas e 1 aprovação.

- `e2e/shell.spec.ts` ainda procura a antiga Home (`heading Hades` e `Sites, aplicativos e jogos`), mas `/` agora é o chat unificado `O que posso fazer por você?`.
- O teste mobile de `/agentic` ainda procura o textbox `Nova tarefa` e o botão `Criar missão`, que não correspondem ao estado atual carregado.
- Os gates axe desktop/mobile expiram em 60 segundos porque usam `page.goto(..., { waitUntil: "networkidle" })` numa aplicação que mantém tráfego/polling local.
- Antes da execução, `@axe-core/playwright` estava declarado no `package.json`/lockfile, mas ausente em `node_modules`; `npm install --ignore-scripts --no-audit --no-fund` restaurou a dependência sem alterar arquivos versionados.

### AUD-CX-03 — configuração de desenvolvimento pode simular backend offline

Severidade: média para desenvolvimento; o build de produção usa mesma origem.

`app/ui/app/src/lib/config.ts` usa `http://127.0.0.1:3001` como fallback em desenvolvimento, enquanto `vite.config.ts` configura o proxy `/api` para `http://127.0.0.1:11434`. Rodar apenas `npm run dev` com o Hades instalado em `11434` pode ignorar o proxy e produzir `Failed to fetch`/estado offline falso. A auditoria precisou definir explicitamente `VITE_API_URL` para a origem do Vite.

## Sinais transitórios que não foram promovidos a defeito

- Configurações apareceu vazia durante navegação rápida, mas renderizou integralmente após estabilização.
- Harnesses & Codex exibiu erro transitório de integrações, depois carregou o estado vazio honesto `Nenhum app encontrado`.
- Houve recusas de conexão enquanto o instalador novo reiniciava o aplicativo. A versão mudou de `g8627e0e` para `geb245f5`, os PIDs foram recriados e a API voltou a HTTP 200; isso foi classificado como atualização concorrente, não crash confirmado.

## Próximo passo recomendado

1. Corrigir `AUD-CX-01` com conversão de URL de arquivo compatível com Windows e adicionar teste Windows.
2. Atualizar `shell.spec.ts` para a Home unificada e trocar `networkidle` por `domcontentloaded` mais esperas semânticas nos testes axe.
3. Unificar o fallback de desenvolvimento com o proxy do Vite ou documentar/validar obrigatoriamente `VITE_API_URL`.
4. Reexecutar Playwright desktop/mobile contra o binário `geb245f5`, incluindo cadastro de provider, remoção de anexo e retomada de upload ZIP, sem usar contas pagas ou efeitos externos.

