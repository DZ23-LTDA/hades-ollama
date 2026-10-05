# Política de atualização upstream — Ollama Full

## Regra principal

O Ollama Full é um fork com uma superfície agentic própria. **Nenhuma atualização do repositório upstream altera este projeto automaticamente.** O remote `upstream` é usado somente para consulta e comparação. A integração de mudanças upstream exige uma branch manual, revisão de diff, execução dos guardrails e aprovação por Pull Request.

A base upstream aceita nesta revisão está registrada em [`UPSTREAM_BASE_COMMIT`](UPSTREAM_BASE_COMMIT). Esse marcador não é uma promessa de compatibilidade total entre versões; ele é um ponto de auditoria para saber qual motor foi analisado.

## O que fica protegido

O guardrail [`scripts/check-class-a-plus-integrity.sh`](scripts/check-class-a-plus-integrity.sh) falha se desaparecerem ou forem descaracterizados os contratos centrais:

- runtime agentic, planner, tools, sandbox, approvals, recovery e artifacts;
- multi-provider, Anthropic/Claude, Codex proxy e preset OmniRoute;
- API `/api/agent/v1`, incluindo configuração sanitizada;
- shell desktop, Agentic Console, Settings e rotas de Projetos, Biblioteca, Agendado, Habilidades, Plugins e Tarefas;
- matriz de paridade, árvore de produto e política manual de upstream;
- regras que impedem segredos, `node_modules`, chaves e certificados de entrarem no Git.

A existência do arquivo não basta. Os testes Go, build da UI, smoke Chromium e checks de segurança continuam obrigatórios.

## Procedimento de atualização do motor

1. Faça `git fetch --no-tags upstream main` sem modificar `main`, a branch de publicação ou a branch agentic ativa.
2. Crie uma branch `chore/upstream-YYYYMMDD` a partir de `main`.
3. Compare o novo commit com [`UPSTREAM_BASE_COMMIT`](UPSTREAM_BASE_COMMIT). Nunca faça force-push nem substitua o histórico público.
4. Reaplique ou resolva conflitos preservando `internal/agent`, `internal/multillm`, `server/agent_routes.go`, UI, mobile, políticas e documentação Ollama Full.
5. Atualize `UPSTREAM_BASE_COMMIT` somente depois de revisar o diff e os riscos.
6. Execute, no mínimo:

   ```bash
   scripts/check-class-a-plus-integrity.sh
   CGO_ENABLED=0 go test ./internal/agent -count=1
   CGO_ENABLED=1 go test ./server ./cmd/launch ./internal/multillm -count=1
   (cd app/ui/app && npm ci --no-audit --no-fund && npm run build)
   ```

7. Execute o smoke Chromium das rotas e botões do shell. Se uma rota, botão, provider, approval ou artifact regredir, a branch não pode ser mesclada.
8. Abra uma PR descrevendo conflitos, APIs alteradas, testes, screenshots, riscos e rollback. A publicação em `main` ocorre somente após os checks obrigatórios passarem.

## O que não será feito

Não será criado um mecanismo que sobrescreva automaticamente o fork com o upstream. Não serão removidos testes para acomodar uma atualização. Não serão preservados nomes de recursos enquanto a execução real tiver sido quebrada. Não serão copiados internals, código, assets ou interface proprietária de Manus, Claude, Codex, OmniRoute ou qualquer outro produto.

A paridade pretendida é funcional e observável. A proteção do fork é feita por isolamento de branches, marcador de base, guardrail, CI, testes, screenshots e revisão de release.
