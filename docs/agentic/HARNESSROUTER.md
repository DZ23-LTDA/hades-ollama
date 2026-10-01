# HarnessRouter no Ollama Full

O [HarnessRouter Community Edition](https://github.com/HarnessRouter/harnessrouter) é um complemento relevante para o Ollama Full: ele fornece uma interface unificada para executar harnesses como Codex, Claude Code, Hermes e outros por uma API compatível com OpenAI Responses. O repositório público informa licença Apache-2.0, sessões persistentes, streaming, arquivos, artifacts, cancelamento, falhas estruturadas e o Unified Harness Protocol (UHP).

## Decisão arquitetural

O HarnessRouter deve ser tratado como **backend opcional e substituível**, não como substituto do runtime local do Ollama Full. O Ollama Full continua responsável por planner, approvals, política local-only, isolamento de workspace, memória, projetos, observabilidade, artifacts e seleção de provider. Quando configurado, o HarnessRouter pode executar um harness especializado atrás do gateway multi-provider.

A integração inicial usa o endpoint OpenAI Responses-compatible do HarnessRouter e injeta `metadata.harness_id` server-side. Assim, o frontend pode escolher `harnessrouter/codex` ou `harnessrouter/claude-code` sem inserir metadata manualmente e sem expor a chave no navegador. O preset seguro está em [`examples/dz23-harnessrouter.json`](../../examples/dz23-harnessrouter.json).

## Configuração local

A Community Edition é self-hosted. O quickstart oficial usa Docker, volume persistente e a porta loopback `3000`; a instância precisa de uma chave de provider própria e não inclui modelo, chave de teste ou conta cloud. O operador deve alterar as credenciais padrão da Console antes de expor a instância.

```bash
docker run -d --name harnessrouter \
  -p 127.0.0.1:3000:3000 \
  -v harnessrouter:/data \
  harnessrouter/harnessrouter

export HARNESSROUTER_API_KEY='chave-fornecida-pelo-HarnessRouter'
export OLLAMA_DZ23_GATEWAY_KEY='chave-de-entrada-do-gateway-local'
export OLLAMA_DZ23_CONFIG=/caminho/para/examples/dz23-harnessrouter.json
```

A chave deve existir somente no ambiente do servidor. Ela nunca deve ser colocada em `localStorage`, screenshots, fixtures, logs, commits ou payloads enviados pelo browser.

## Mapeamento de capabilities

| Seleção no Ollama Full | Provider/model | Metadata enviado ao HarnessRouter | Estado inicial |
|---|---|---|---|
| Codex via HarnessRouter | `harnessrouter/codex` | `harness_id=codex` | Adapter implementado; requer HarnessRouter, chave e harness disponível |
| Claude Code via HarnessRouter | `harnessrouter/claude-code` | `harness_id=claude-code` | Adapter implementado; requer HarnessRouter, chave e harness disponível |
| Ollama local | modelo local Ollama | Nenhum | Não depende do HarnessRouter |
| OmniRoute | `omniroute/auto` | Nenhum | Gateway local opcional e separado |

## Limites e segurança

A integração de protocolo não comprova que Codex ou Claude Code estejam instalados, autenticados ou aptos a executar uma missão neste ambiente. Cada CLI mantém sua própria licença e seus requisitos; a licença Apache-2.0 do HarnessRouter não relicencia os harnesses, modelos ou ferramentas instalados por ele.

O endpoint HTTP sem TLS é aceito somente para loopback quando as flags de rede privadas e inseguras estão explicitamente ativadas. Para hosts externos, o Ollama Full exige HTTPS. A política `local-only` impede que uma missão local faça fallback silencioso para HarnessRouter. Routing, conexão de contas, publicação, escrita externa e operações de alto impacto continuam sujeitos a approval.

A validação completa exige uma instância HarnessRouter local real, provider configurado, harness instalado, chave server-side, teste de streaming, cancelamento, follow-up por sessão, artifacts e falhas. Até esse gate, a matriz de paridade deve classificar a integração como **ADAPTER IMPLEMENTADO**, não como disponibilidade validada.
