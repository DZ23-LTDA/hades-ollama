# Mapa de paridade funcional — Ollama Full ⟵ Manus

> **Objetivo:** dar ao `ollama-classe-a-plus` (Ollama Full) as **mesmas funções e os
> mesmos caminhos/fluxos** do Manus, com a **identidade e o logo do próprio Ollama**.
> **NÃO é clone visual.** Não copiamos a UI proprietária do Manus nem embutimos
> logos de terceiros (Gmail, Slack, Notion…). Usamos ícones próprios/neutros e a
> marca Ollama. Paridade = capacidades + navegação + comportamento.
>
> Fonte: observação do Manus real (conta do usuário, tema Automático→escuro),
> telas: home/nova tarefa, Plugins, tarefa aberta, Configurações. Registrado por
> Claude em 2026-09-29.

## Tema (Aparência)
- Manus: **Claro / Escuro / Automático** (Automático = segue o SO). Idioma PT-BR.
- Ollama Full: já segue `prefers-color-scheme`. **Gap:** controle explícito com as
  3 opções (Claro/Escuro/Automático) nas Configurações.

## Navegação principal (sidebar) — Manus → Ollama Full

| Manus (função) | Caminho | Ollama Full hoje | Gap / ação |
|---|---|---|---|
| **Nova tarefa** (home: input de tarefa; atalhos "Compilar: sites/apps/jogos", "Criar: slides/imagens/vídeos", "Começar de arquivo local") | `/app` | Home redireciona pro chat; "Nova tarefa" existe | Home com atalhos de criação (Compilar/Criar) + input de tarefa |
| **Computadores** (ambientes/máquinas gerenciadas p/ computer use) | `/app/computers` | Adapter "computer use/desktop" (parcial) | Seção "Computadores" com dispositivos pareados/estado |
| **Agents** (agentes) | `/app/agents` | Agentic Console + Agentes departamentais | Seção "Agents" listando agentes/roles |
| **Biblioteca** (artifacts/criações) | `/app/library` | Biblioteca ✓ (artifacts de missões) | Refinar p/ incluir criações/arquivos |
| **Criações** (slides, imagens, vídeos) | `/app/creations` | Builders (parcial) | Seção "Criações" (galeria de slides/imagens/vídeos gerados) |
| **Automações** (recorrentes/agendadas) | `/app/automations` | Agendado ✓ | Alinhar nome/estrutura a "Automações" |
| **Plugins** (marketplace de conectores) | `/app/plugins` | Plugins ✓ (marketplace criado) | Ver seção Plugins abaixo |
| **Mais** (menu) | — | — | Menu "Mais" com itens extras |
| **Projetos → Novo projeto** | `/app` (projetos) | Projetos ✓ | ok |
| **Tarefas** (lista/inbox de tarefas) | sidebar | Tarefas ✓ | ok |
| **Empresa (Company OS)** | — (extra do Ollama) | Empresa ✓ | melhoria própria do Ollama Full |

Barra superior (em tarefa/home): seletor de modo (ex.: "Manus Flex"/"Manus 2.0
Lite"), seletor de modelo (ex.: "DeepSeek V4.1 Flash Vision · High"), créditos.
→ Ollama Full: seletor de modelo/provider já existe no Console; falta o seletor
de "modo" na barra superior e indicador de créditos/uso.

## Experiência de TAREFA (o coração do Manus)
Layout de tarefa aberta:
- **Esquerda — conversa:** mensagens do agente, comandos/ferramentas executados
  ("Leu N arquivos · N comandos"), subagentes em execução, blocos de resultado.
- **Direita — painel "Computador" ao vivo:** terminal/browser/computer que o
  agente está usando, com **controles de playback** e selo **"Ao vivo"**, e
  **rastreador de etapas** ("Etapa 4/5").
- **Rodapé:** input "Mensagem para Manus" + anexos + chips de modelo + mic + enviar.
- Topo: título da tarefa, compartilhar, menu.

→ Ollama Full: tem chat (`/c/$chatId`) e Agentic Console (`/agentic`) separados.
**Gap grande:** unir numa **visão de tarefa** = conversa + painel de execução ao
vivo (terminal/browser/computer via o operador Playwright/desktop) + rastreador de
etapas + playback. É a jornada central a perseguir.

## Plugins — marketplace de conectores
Manus: cards em destaque no topo; busca; **categorias** (Todos, Produtividade,
Criatividade, Negócios, Desenvolvimento, Finanças, Viagem, Saúde); grid de cards
(ícone + nome + descrição + toggle **+ / ✓**); botões "Gerenciar Conectores" e
"Criar". Segue tema do SO.
→ Ollama Full: marketplace de cards já criado (feat/ui-shell-parity). **Gaps:**
alinhar categorias, cards em destaque, toggle +/✓ por card, "Gerenciar Conectores".

## Configurações (settings)
Seções do Manus: **Conta**, **Uso e Faturamento**; **Configurações** (Geral
[Idioma, Tema, Preferências de chat, Preferências de comunicação], Atalhos,
Recursos beta); **Capacidades** (Personalização, Conectores, Habilidades, Mail,
Computadores); **Ativos** (Email, Telefone, Carteira); **Dados e Integrações**
(Controles de Dados, Integrações).
→ Ollama Full: settings legado do Ollama (modelos/cloud) + AgenticControlCenter.
**Gap:** organizar settings nessas seções + o controle de Tema.

## Prioridades de build (faixa ui, feat/ui-shell-parity)
1. Controle de **Tema Claro/Escuro/Automático** (visível, rápido, igual ao Manus).
2. **Home / Nova tarefa** com atalhos Compilar/Criar + input de tarefa.
3. Refinar **Plugins** (categorias, destaques, toggle +/✓).
4. **Visão de tarefa** unificada (conversa + painel ao vivo + etapas) — maior esforço.
5. Alinhar nomes/paths: Automações, Agents, Criações, Computadores.
6. Organizar **Configurações** nas seções do Manus + Tema.

> Cada item entregue com verificação no navegador (claro e escuro) e evidência.
> Identidade: **logo e marca Ollama**, ícones próprios — nunca logos de terceiros.
