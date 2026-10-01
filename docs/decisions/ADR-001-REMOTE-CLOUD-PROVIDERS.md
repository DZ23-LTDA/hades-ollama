# ADR-001: Adapters para Provedores Remotos e Proteção de Segredos

**Status:** APROVADO  
**Data:** 29 de Setembro de 2026  
**Contexto:**  
O Ollama Full visa paridade funcional com o Manus Desktop, permitindo alternar entre modelos locais (via Ollama engine) e modelos avançados de raciocínio externo (Claude, Codex, xAI Grok, OmniRoute, HarnessRouter). No entanto, como um runtime prioritariamente local-first e enterprise-grade, credenciais de API de terceiros não devem ser gravadas em repositórios, expostas em fixtures de teste ou renderizadas no frontend sem proteção.

**Decisão:**  
1. Todos os provedores externos são operados através de arquitetura de **Adapters** no servidor Go (`server/` e `internal/agent/multillm`), com validação estrita de endpoint, timeouts de segurança e injeção server-side de tokens.
2. Nenhuma credencial externa é gravada em arquivos versionados do Git nem exibida em texto claro na interface.
3. Quando executado sem credenciais de nuvem, o app permanece 100% operacional no modo local-first utilizando os modelos locais instalados ou o `RulePlanner` determinístico de fallback.

**Consequências:**  
- O status desses provedores na matriz de paridade é classificado como `ADAPTER IMPLEMENTADO`, cumprindo a paridade sem comprometer o isolamento de segurança.
