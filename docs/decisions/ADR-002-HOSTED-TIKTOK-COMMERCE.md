# ADR-002: Comércio Eletrônico Social e APIs Externas de Terceiros

**Status:** APROVADO  
**Data:** 29 de Setembro de 2026  
**Contexto:**  
Funcionalidades de social commerce (TikTok Shop, Instagram Shopping, Shopify) dependem de contas de vendedor aprovadas, credenciais OAuth federadas nos respectivos Partner Centers e webhooks públicos para processamento de pagamentos reais. Em um ambiente de desenvolvimento ou execução local-first, executar transações financeiras reais sem supervisão humana violaria as políticas de segurança.

**Decisão:**  
1. O módulo Company OS e Growth OS implementa os esquemas de dados, catálogos de produtos, roteiros e contratos de automação em nível de sandbox local.
2. Qualquer chamada de checkout, débito ou publicação comercial externa é obrigatoriamente retida sob o gate de aprovação humana in-line (`AWAITING_APPROVAL`) antes da transmissão externa.
3. No ambiente padrão local sem credenciais de vendedor, as rotas simulam o sandbox empresarial sem gerar cobranças financeiras.

**Consequências:**  
- O status na matriz de paridade permanece como `PARCIAL COM SANDBOX LOCAL ATIVO`, garantindo segurança financeira e arquitetura extensível.
