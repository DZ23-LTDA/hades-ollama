# Evaluation OS e provider router

O Ollama Full agora possui uma camada local de avaliação determinística para transformar paridade funcional em evidência reproduzível. O catálogo padrão cobre coding, browser, tools, segurança, memória, planejamento e recuperação. Cada caso possui identificador, domínio, timeout e resultado explícito; falhas e timeouts não são convertidos em sucesso.

O provider router seleciona entre modelos registrados usando capacidades requeridas, estado de saúde, latência observada, custo, privacidade e qualidade histórica. A decisão é explicável e pode ser limitada por `local_only`, orçamento, provider preferido e capacidades multimodais. O router não cria credenciais nem afirma que um provider está disponível: modelos sem registro, sem saúde ou fora das restrições são excluídos.

## Grok Live

O cliente `internal/grok` implementa Responses API, streaming SSE, retries limitados, circuito de falha e status sanitizado. O endpoint local `GET /api/agent/v1/grok/status` informa apenas provider, modelo, estado, autenticação, saúde, latência e erro resumido. O endpoint `POST /api/agent/v1/grok/responses` exige `XAI_API_KEY` server-side; o browser nunca recebe a chave. Sem credencial, o estado permanece `cataloged` e a execução responde indisponível.

Grok Live distingue memória do projeto de evidência ao vivo. Citações recebidas de uma pesquisa devem ser mantidas como fontes verificáveis, enquanto conteúdo de memória é rotulado como contexto e não como confirmação atual. A integração não incorpora o Grok Bot hospedado nem promete web search, Voice ou Imagine sem credenciais, escopos e testes autorizados.

## Verificação

Os testes unitários cobrem seleção por restrições, saúde, custo, latência, qualidade, casos aprovados, falhos e timeout. A promoção de qualquer provider para uma conclusão de produção ainda exige dataset, quota, judge autorizado, smoke externo, proteção de dados e métricas históricas por domínio.
