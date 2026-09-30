# ADR-004: Isolamento Multi-tenant com PostgreSQL RLS e Fallback Local-First

**Status:** APROVADO  
**Data:** 29 de Setembro de 2026  
**Contexto:**  
O Ollama Full foi desenhado para operar tanto em implantações enterprise multi-tenant (com segregação criptográfica estrita por organização via PostgreSQL Row-Level Security) quanto em desktops locais de desenvolvedores (single-user local-first sem dependência de banco de dados relacional externo).

**Decisão:**  
1. Quando a variável `OLLAMA_AGENT_DATABASE_URL` não for fornecida, o runtime opera automaticamente no modo `JSONStore` local, vinculado à organização reservada `LocalOrganizationID = "local"`.
2. As rotas HTTP de gerenciamento de sessões, approvals e catálogo adotam fallback automático para `local`, eliminando mensagens de erro de autenticação no desktop.
3. Em ambientes corporativos, a migração e as políticas RLS são aplicadas exclusivamente pelo comando `ollama agent migrate-postgres` com usuário migrador separado, protegendo os dados de cada inquilino por HMAC de transação.

**Consequências:**  
- Experiência plug-and-play imediata no desktop sem configuração de infraestrutura complexa, preservando ao mesmo tempo a capacidade enterprise comprovada em testes automatizados.
