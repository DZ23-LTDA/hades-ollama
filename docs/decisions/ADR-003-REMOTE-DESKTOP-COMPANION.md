# ADR-003: Arquitetura de Computer Use e Companion Desktop

**Status:** APROVADO  
**Data:** 29 de Setembro de 2026  
**Contexto:**  
O controle direto de periféricos do sistema operacional hospedeiro (mouse, teclado, captura de tela raw) exige privilégios de acessibilidade e segurança elevados em cada plataforma (Windows, macOS, Linux). O acesso descontrolado de ferramentas de IA ao sistema operacional pode causar ações acidentais em janelas confidenciais.

**Decisão:**  
1. O acesso ao desktop utiliza o protocolo **Desktop Commander** via MCP (Model Context Protocol) com transporte Streamable HTTP ou stdio local autenticado por token efêmero.
2. Toda operação destrutiva de processo ou modificação fora do diretório de workspace requer confirmação expressa na interface do agente.
3. Para ambientes onde o companion não está conectado, o runtime agentic opera no modo isolado via Playwright browser headless/headed com espelhamento visual por frame JPEG.

**Consequências:**  
- Paridade de navegação e visualização é garantida pelo operador Playwright com evento `browser.frame`, enquanto automações de sistema operacional utilizam o companion autorizado.
