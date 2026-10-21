// Dicionário base da UI em português do Brasil.
// Chaves semânticas e planas: `namespace.componente.chave`.
// Interpolação usa `{nome}` e é resolvida por `translate()` em `./index.ts`.
export const ptBRMessages: Record<string, string> = {
  // Comum
  "common.delete": "Excluir",
  "common.remove": "Remover",
  "common.enable": "Habilitar",
  "common.disable": "Desabilitar",
  "common.save": "Salvar",
  "common.edit": "Editar",
  "common.create": "Criar",
  "common.creating": "Criando…",
  "common.retry": "Tentar novamente",
  "common.id": "ID",
  "common.provider": "Provider",
  "common.version": "Versão",
  "common.description": "Descrição",

  // Página / cabeçalho por tipo
  "productWorkspace.projects.title": "Projetos",
  "productWorkspace.projects.eyebrow": "Contexto persistente",
  "productWorkspace.projects.description":
    "Organize memória, fontes, tarefas, membros e builders em workspaces isolados.",
  "productWorkspace.projects.action": "Novo projeto",
  "productWorkspace.skills.title": "Habilidades",
  "productWorkspace.skills.eyebrow": "Skills e MCP",
  "productWorkspace.skills.description":
    "Veja manifestos, versões, escopos e confiança das habilidades carregadas pelo runtime.",
  "productWorkspace.skills.action": "Abrir configuração",
  "productWorkspace.plugins.title": "Registro avançado",
  "productWorkspace.plugins.eyebrow": "Conectores, MCP e skills",
  "productWorkspace.plugins.description":
    "Inspecione GitHub, Google Workspace, Claude, Codex, OmniRoute, MCP e outros serviços sem expor segredos.",
  "productWorkspace.plugins.action": "Abrir configuração",
  "productWorkspace.tasks.title": "Tarefas",
  "productWorkspace.tasks.eyebrow": "Inbox de missões",
  "productWorkspace.tasks.description":
    "Veja missões em execução, aguardando approval, concluídas ou em recuperação.",
  "productWorkspace.tasks.action": "Nova tarefa",

  // Estados genéricos
  "productWorkspace.resource.delete": "Excluir recurso",
  "productWorkspace.loading": "Consultando contratos agentic…",
  "productWorkspace.loadError":
    "Não foi possível carregar os dados persistidos.",

  // Projetos
  "productWorkspace.projects.updatedAt": "· atualizado",
  "productWorkspace.projects.localWorkspace": "workspace local gerenciado",
  "productWorkspace.projects.empty":
    "Crie um projeto para manter instruções, arquivos e contexto entre missões.",
  "productWorkspace.projects.emptyAction": "Criar projeto",
  "productWorkspace.projects.created":
    "Projeto criado e persistido no ContextStore.",
  "productWorkspace.projects.createFailed": "Falha ao criar projeto.",
  "productWorkspace.projects.updated": "Projeto atualizado.",
  "productWorkspace.projects.updateFailed": "Falha ao atualizar projeto.",
  "productWorkspace.projects.deleted": "Projeto excluído.",
  "productWorkspace.projects.deleteFailed": "Falha ao excluir projeto.",
  "productWorkspace.projects.deleteConfirm":
    "Excluir o projeto {name}? Memórias locais associadas também serão removidas.",
  "productWorkspace.projects.nameLabel": "Nome do novo projeto",
  "productWorkspace.projects.namePlaceholder": "Nome do projeto",
  "productWorkspace.projects.workspaceLabel": "Workspace do novo projeto",
  "productWorkspace.projects.workspacePlaceholder": "Workspace (opcional)",
  "productWorkspace.projects.importTitle": "Importar projeto inteiro",
  "productWorkspace.projects.importDescription":
    "GitHub e ZIP são baixados, isolados em worktree e indexados. Código importado nunca é executado automaticamente.",
  "productWorkspace.projects.importAction": "Importar projeto",
  "productWorkspace.projects.askTitle": "Perguntar aos documentos",
  "productWorkspace.projects.askDescription":
    "As respostas vêm somente dos documentos indexados do projeto, com as fontes citadas.",
  "productWorkspace.projects.askSelect": "Projeto para consultar",

  // Tarefas
  "productWorkspace.tasks.created": "Tarefa criada",
  "productWorkspace.tasks.state": "· estado",
  "productWorkspace.tasks.missingTitle":
    "A tarefa criada não apareceu na listagem",
  "productWorkspace.tasks.missingHint":
    "Atualize a lista ou verifique a organização ativa.",
  "productWorkspace.tasks.refresh": "Atualizar lista",
  "productWorkspace.tasks.empty":
    "Nenhuma tarefa criada ainda. Crie a primeira para acompanhar plano, approvals e artifacts aqui.",
  "productWorkspace.tasks.emptyAction": "Nova tarefa",
  "productWorkspace.tasks.listLabel": "Tarefas",
  "productWorkspace.tasks.meta":
    "{id} · {steps} passos · {artifacts} artefatos · {model}",
  "productWorkspace.tasks.defaultModel": "modelo padrão",
  "productWorkspace.tasks.cancelBeforeDelete":
    "Cancele a tarefa antes de excluir",
  "productWorkspace.tasks.deleteLabel": "Excluir tarefa",
  "productWorkspace.tasks.deleted": "Tarefa excluída.",
  "productWorkspace.tasks.deleteFailed": "Falha ao excluir a tarefa.",
  "productWorkspace.tasks.deleteConfirm":
    'Excluir a tarefa "{objective}"? Esta ação não pode ser desfeita.',

  // Skills
  "productWorkspace.skills.catalogTitle": "Catálogo por contexto",
  "productWorkspace.skills.catalogDescription":
    "Ative skills por projeto, agente ou modelo e veja o que o servidor selecionou para o orçamento informado. A ordem e as exclusões vêm do servidor; a tela não decide sozinha.",
  "productWorkspace.skills.meta": "{count} tools · {trust}",
  "productWorkspace.skills.trusted": "confiável",
  "productWorkspace.skills.needsReview": "requer revisão",
  "productWorkspace.skills.empty":
    "Nenhuma skill foi carregada pelo runtime. Instale um manifesto revisado no diretório de skills do servidor.",
  "productWorkspace.skills.emptyAction": "Abrir configuração",
  "productWorkspace.skills.resolveFailed":
    "Não foi possível resolver o catálogo de skills.",
  "productWorkspace.skills.activateFailed": "Não foi possível ativar a skill.",
  "productWorkspace.skills.deactivateFailed":
    "Não foi possível desativar a skill.",

  // Plugins / conectores / MCP
  "productWorkspace.plugins.statusError":
    "Não foi possível ler o estado dos plugins.",
  "productWorkspace.plugins.installedTitle": "Plugins instalados — estado real",
  "productWorkspace.plugins.installedDescription":
    "O servidor informa o estado de cada plugin a partir de dados verificáveis. “Pronto para usar” só aparece quando existe execução real comprovada; catálogo não é integração.",
  "productWorkspace.plugins.connectorTitle": "Registrar connector seguro",
  "productWorkspace.plugins.connectorDescription":
    "Cadastre endpoint, operações e referências de credencial. O valor do token nunca é aceito nesta tela; use uma variável de ambiente do servidor ou o identificador de um OAuth já autorizado.",
  "productWorkspace.plugins.fieldBaseUrl": "Base URL HTTPS",
  "productWorkspace.plugins.fieldTokenEnv":
    "Nome da variável de token (opcional)",
  "productWorkspace.plugins.fieldOAuthProvider":
    "ID do provider OAuth (opcional)",
  "productWorkspace.plugins.fieldMethods": "Métodos permitidos",
  "productWorkspace.plugins.fieldPathPrefix": "Prefixo de caminho",
  "productWorkspace.plugins.registerTenant": "Registrar no tenant atual",
  "productWorkspace.plugins.connectorRegistered":
    "Connector registrado no tenant atual. O runtime mantém apenas referências de credencial; o valor deve existir no ambiente do servidor ou no OAuth autorizado.",
  "productWorkspace.plugins.connectorRegisterFailed":
    "Falha ao registrar connector.",
  "productWorkspace.plugins.mcpTitle": "Registrar MCP e skills",
  "productWorkspace.plugins.mcpDescription":
    "O cadastro grava somente manifestos. MCP local usa executável absoluto validado; Remote MCP usa HTTPS/SSRF/DNS pinning; skills começam não confiáveis. Não cole tokens nesta tela.",
  "productWorkspace.plugins.mcpStdio": "MCP stdio",
  "productWorkspace.plugins.registerLocalMcp": "Registrar MCP local",
  "productWorkspace.plugins.remoteMcp": "Remote MCP",
  "productWorkspace.plugins.tokenEnvOptional": "TOKEN_ENV (opcional)",
  "productWorkspace.plugins.registerRemoteMcp": "Registrar Remote MCP",
  "productWorkspace.plugins.skillManifest": "Skill manifest",
  "productWorkspace.plugins.skillScopes": "Scopes separados por vírgula",
  "productWorkspace.plugins.skillTools": "Tools separados por vírgula",
  "productWorkspace.plugins.registerSkill": "Registrar skill",
  "productWorkspace.plugins.mcpRegistered":
    "MCP stdio registrado. O executável é validado e executado somente no host/workspace permitido pelo servidor.",
  "productWorkspace.plugins.mcpRegisterFailed": "Falha ao registrar MCP stdio.",
  "productWorkspace.plugins.remoteMcpRegistered":
    "Remote MCP registrado. O endpoint deve passar HTTPS/SSRF/DNS pinning; tokens permanecem no ambiente do servidor.",
  "productWorkspace.plugins.remoteMcpRegisterFailed":
    "Falha ao registrar Remote MCP.",
  "productWorkspace.plugins.skillRegistered":
    "Manifesto de skill registrado como não confiável e habilitado pelo servidor para revisão/approval.",
  "productWorkspace.plugins.skillRegisterFailed": "Falha ao registrar skill.",
  "productWorkspace.plugins.toggleFailed":
    "Falha ao alterar lifecycle do plugin.",
  "productWorkspace.plugins.removeFailed": "Falha ao remover o plugin.",
  "productWorkspace.plugins.toggleNotice": "{id} {status}.",
  "productWorkspace.plugins.enabled": "habilitado",
  "productWorkspace.plugins.disabled": "desabilitado",
  "productWorkspace.plugins.removeNotice": "{id} removido do manifest local.",
  "productWorkspace.plugins.removeConfirm":
    "Remover {id} do manifest durável? Essa ação não revoga credenciais upstream.",
  "productWorkspace.plugins.appsTitle": "Aplicativos e conectores",
  "productWorkspace.plugins.appsDescription":
    "Catálogo amplo de apps, APIs e MCPs. O botão de configuração nunca inventa uma conexão: cada provider exige OAuth, API key ou homologação própria.",
  "productWorkspace.plugins.catalogCount": "{count} disponíveis no catálogo",
  "productWorkspace.plugins.search": "Pesquisar conectores",
  "productWorkspace.plugins.categories": "Categorias de conectores",
  "productWorkspace.plugins.allCategories": "Todos",
  "productWorkspace.plugins.auth": "· autenticação:",
  "productWorkspace.plugins.statusAvailable": "Disponível para configuração",
  "productWorkspace.plugins.statusProviderSelection":
    "Escolha o provedor fiscal",
  "productWorkspace.plugins.statusOperatorConfig":
    "Configuração do operador necessária",
  "productWorkspace.plugins.statusConfiguredDisabled":
    "Configurado · desabilitado",
  "productWorkspace.plugins.statusConfiguredCredential":
    "Configurado · habilitado · credencial presente",
  "productWorkspace.plugins.statusConfiguredNoCredential":
    "Configurado · habilitado · credencial ausente",
  "productWorkspace.plugins.viewState": "Ver estado",
  "productWorkspace.plugins.configureProviders": "Configurar em Provedores",
  "productWorkspace.plugins.configuredNotice":
    "{name}: veja o estado no bloco de connectors configurados abaixo. O runtime não chama a integração sem credencial e approval.",
  "productWorkspace.plugins.noConnectorMatch":
    "Nenhum conector corresponde à pesquisa ou categoria selecionada.",
  "productWorkspace.plugins.configuredConnectors":
    "Connectors configurados ({count})",
  "productWorkspace.plugins.connectorOperations":
    "{count} operações · segredos mantidos no servidor",
  "productWorkspace.plugins.connectorEnabledCredential":
    "habilitado · credencial presente",
  "productWorkspace.plugins.connectorEnabledNoCredential":
    "habilitado · credencial ausente",
  "productWorkspace.plugins.noConnectors":
    "Nenhum connector configurado. O catálogo acima mostra o que pode ser conectado sem fingir autorização.",
  "productWorkspace.plugins.mcpServers": "MCP local e remoto ({count})",
  "productWorkspace.plugins.remoteEndpoint": "endpoint remoto",
  "productWorkspace.plugins.mcpMethods":
    "{count} métodos allowlisted · credencial somente no servidor",
  "productWorkspace.plugins.noMcpServers":
    "Nenhum servidor MCP carregado. O comando ou endpoint precisa estar allowlisted no servidor.",
  "productWorkspace.plugins.cliNote":
    "{count} CLI(s) de integração detectado(s). A instalação e o registro continuam protegidos no ambiente do operador.",

  // Indicadores e rodapé
  "productWorkspace.stats.status": "Status",
  "productWorkspace.stats.security": "Segurança",
  "productWorkspace.stats.persistence": "Persistência",
  "productWorkspace.stats.running": "{count} em execução",
  "productWorkspace.stats.projects": "{count} projetos",
  "productWorkspace.stats.realCatalog": "Catálogo real",
  "productWorkspace.stats.activeApprovals": "Aprovações ativas",
  "productWorkspace.stats.localFirst": "Local-first",
  "productWorkspace.footer.persisted": "Dados persistidos",
  "productWorkspace.footer.persistedDescription":
    "Esta superfície consulta o contrato agentic, não fixtures visuais.",
  "productWorkspace.footer.policy": "Política ativa",
  "productWorkspace.footer.secrets": "Secrets não aparecem na interface.",
  "productWorkspace.footer.approvals": "Ações externas exigem approval.",
  "productWorkspace.footer.crossTenant":
    "Cross-tenant é rejeitado no servidor.",
  "productWorkspace.footer.nextSteps": "Próximos passos",
  "productWorkspace.footer.nextStepsDescription":
    "Para criar uma missão complexa, use o console com seleção de provider, projeto, timeline e approvals.",
  "productWorkspace.footer.openConsole": "Abrir Agentic Console",

  // Studio — paleta de componentes e modelos de projeto
  "studio.palette.ai.title": "Gerar com IA",
  "studio.palette.ai.placeholder": "Descreva o app/site que você quer…",
  "studio.palette.ai.replace": "Gerar do zero (substituir o canvas)",
  "studio.palette.ai.generating": "Gerando com modelo local…",
  "studio.palette.ai.generate": "Gerar componentes",
  "studio.palette.components": "Componentes",
  "studio.palette.addHint": "Clique para adicionar",
  "studio.palette.templates": "Modelos de Projeto",
  "studio.palette.mobileAdd": "Adicionar:",
  "studio.palette.template.site": "Site / Landing Page",
  "studio.palette.template.dashboard": "Dashboard Analítico",
  "studio.palette.template.slides": "Slides / Pitch Deck",
  "studio.palette.template.game": "Jogo 2D Interativo",
  "studio.palette.template.app": "Aplicação Web App",

  // Studio — modelos de componentes (rótulos da paleta)
  "studio.template.heading.label": "Título",
  "studio.template.paragraph.label": "Parágrafo",
  "studio.template.button.label": "Botão de Ação",
  "studio.template.card.label": "Card / Recurso",
  "studio.template.metric.label": "Métrica / KPI",
  "studio.template.navbar.label": "Barra de Navegação",
  "studio.template.hero.label": "Hero / Destaque",
  "studio.template.image.label": "Imagem",
  "studio.template.input.label": "Campo de Formulário",
  "studio.template.link.label": "Link",
  "studio.template.list.label": "Lista",
  "studio.template.divider.label": "Divisor",
  "studio.template.pricing.label": "Plano / Preço",
  "studio.template.testimonial.label": "Depoimento",
  "studio.template.faq.label": "FAQ (pergunta)",
  "studio.template.footer.label": "Rodapé",

  // Studio — conteúdo inicial dos modelos
  "studio.template.heading.text": "Novo Título de Impacto",
  "studio.template.paragraph.text":
    "Descreva sua ideia, solução ou produto com detalhes claros e objetivos.",
  "studio.template.button.text": "Clique Aqui",
  "studio.template.card.title": "Destaque do Produto",
  "studio.template.card.body":
    "Explicação dos benefícios chave da solução construída.",
  "studio.template.metric.sampleLabel": "Taxa de Sucesso",
  "studio.template.navbar.links": "Início, Recursos, Preços, Contato",
  "studio.template.hero.title": "Sua ideia, pronta em minutos",
  "studio.template.hero.subtitle": "Monte, publique e evolua — tudo local.",
  "studio.template.hero.cta": "Começar agora",
  "studio.template.image.alt": "Descrição da imagem",
  "studio.template.input.sampleLabel": "Seu e-mail",
  "studio.template.input.placeholder": "voce@exemplo.com",
  "studio.template.link.text": "Saiba mais",
  "studio.template.list.items": "Primeiro item, Segundo item, Terceiro item",
  "studio.template.pricing.plan": "Plano Pro",
  "studio.template.pricing.price": "R$ 49/mês",
  "studio.template.pricing.features":
    "Tudo do Básico, Suporte prioritário, Sem limites",
  "studio.template.pricing.cta": "Assinar",
  "studio.template.testimonial.quote":
    "Mudou completamente o nosso fluxo de trabalho.",
  "studio.template.testimonial.author": "Cliente satisfeito",
  "studio.template.faq.question": "Pergunta frequente?",
  "studio.template.faq.answer": "Resposta clara e objetiva para a dúvida.",
  "studio.template.footer.text": "© 2026 Minha Empresa",
  "studio.template.footer.links": "Sobre, Contato, Privacidade",

  // Studio — fallbacks de renderização no canvas
  "studio.render.paragraphFallback": "Texto do parágrafo",
  "studio.render.buttonFallback": "Botão",
  "studio.render.cardTitleFallback": "Título do Card",
  "studio.render.cardBodyFallback": "Corpo do card explicativo.",
  "studio.render.metricFallback": "Métrica",
  "studio.render.navbarBrandFallback": "Brand",
  "studio.render.navbarLinksFallback": "Link 1, Link 2",
  "studio.render.heroTitleFallback": "Título do Hero",
  "studio.render.heroSubtitleFallback": "Subtítulo explicativo",
  "studio.render.heroCtaFallback": "Ação",
  "studio.render.imageEmpty": "Imagem (defina a URL nas propriedades)",
  "studio.render.inputLabelFallback": "Rótulo",
  "studio.render.listFallback": "Item 1, Item 2",
  "studio.render.pricingFeaturesFallback": "Recurso 1, Recurso 2",
  "studio.render.footerLinksFallback": "Sobre, Contato",

  // Studio — cabeçalho
  "studio.header.title": "Editor do Studio",
  "studio.header.subtitle":
    "Monte um site ou app adicionando componentes com um clique e reordenando a lista; veja a prévia e exporte em ZIP.",
  "studio.header.undo": "Desfazer (Undo)",
  "studio.header.redo": "Refazer (Redo)",
  "studio.header.tabEditor": "Editor",
  "studio.header.tabPreview": "Prévia",
  "studio.header.export": "Exportar ZIP",
  "studio.header.exportTitle": "Exportar ZIP com checksum SHA-256",
  "studio.header.downloadHtml": "Baixar HTML",
  "studio.header.downloadHtmlTitle":
    "Baixar o app como um arquivo HTML standalone",
  "studio.header.instantPreview": "Ver agora",
  "studio.header.instantPreviewTitle":
    "Ver o app agora em uma nova aba (prévia instantânea, sem servidor)",
  "studio.header.publish": "Publicar",

  // Studio — canvas
  "studio.canvas.area": "Área do Canvas ({count} componentes)",
  "studio.canvas.hint": "Clique em um componente para editar suas propriedades",
  "studio.canvas.loading": "Carregando canvas do Studio...",
  "studio.canvas.empty": "O canvas está vazio",
  "studio.canvas.emptyHint":
    "Clique em um componente na barra lateral esquerda para começar a montar.",
  "studio.canvas.dragHint": "Arraste para reordenar",
  "studio.canvas.moveUp": "Mover para cima",
  "studio.canvas.moveDown": "Mover para baixo",
  "studio.canvas.delete": "Excluir",

  // Studio — inspector de propriedades
  "studio.inspector.title": "Propriedades",
  "studio.inspector.type": "Tipo",
  "studio.inspector.text": "Texto",
  "studio.inspector.titleField": "Título",
  "studio.inspector.body": "Corpo",
  "studio.inspector.label": "Rótulo",
  "studio.inspector.value": "Valor",
  "studio.inspector.brand": "Marca",
  "studio.inspector.links": "Links (separados por vírgula)",
  "studio.inspector.change": "Variação",
  "studio.inspector.action": "Ação",

  // Studio — prévia
  "studio.preview.authNote":
    "Prévia autenticada: carregada com a sua sessão; a URL interna é protegida e não é compartilhável.",
  "studio.preview.expand": "Expandir prévia",
  "studio.preview.frameTitle": "Prévia ao vivo",
  "studio.preview.dialogLabel": "Prévia do app",
  "studio.preview.dialogTitle": "Prévia — {name}",
  "studio.preview.close": "Fechar ✕",
  "studio.preview.closeLabel": "Fechar prévia",
  "studio.preview.expandedDialogLabel": "Prévia expandida",
  "studio.preview.expandedDialogTitle": "Prévia autenticada — {name}",
  "studio.preview.expandedFrameTitle": "Prévia autenticada expandida",

  // Studio — publicação
  "studio.deploy.title": "Publicar projeto",
  "studio.deploy.description":
    "Selecione o provedor de nuvem para hospedar seu projeto. Sem credenciais ativas, o sistema mantém status honesto.",
  "studio.deploy.requiresCredentials": "Requer credenciais",
  "studio.deploy.close": "Fechar",
  "studio.deploy.provider.vercel": "Vercel",
  "studio.deploy.provider.cloudflare": "Cloudflare Pages",
  "studio.deploy.provider.netlify": "Netlify",
  "studio.deploy.provider.ssh": "SSH / Servidor Próprio",
  "studio.deploy.credential.vercel": "VERCEL_TOKEN",
  "studio.deploy.credential.cloudflare":
    "CLOUDFLARE_API_TOKEN e CLOUDFLARE_ACCOUNT_ID",
  "studio.deploy.credential.netlify": "NETLIFY_AUTH_TOKEN",
  "studio.deploy.credential.ssh":
    "chave SSH e host (SSH_PRIVATE_KEY, SSH_HOST, SSH_USER)",
  "studio.deploy.publishing": "Publicando…",
  "studio.deploy.sending":
    "Enviando o projeto para {provider} pelo Deploy Adapter…",
  "studio.deploy.published": "Publicado",
  "studio.deploy.publishedWithUrl": "Publicado em {provider}. URL: {url}",
  "studio.deploy.processing": "Em processamento",
  "studio.deploy.accepted":
    "{provider}: solicitação de publicação aceita pelo runtime. Acompanhe a conclusão e a URL final.",
  "studio.deploy.acceptedWithStatus":
    "{provider}: solicitação de publicação aceita pelo runtime (estado: {status}). Acompanhe a conclusão e a URL final.",
  "studio.deploy.blocked": "Bloqueado (dependência externa)",
  "studio.deploy.failed":
    "Não foi possível publicar em {provider}: {reason}. Configure a credencial no ambiente do servidor: {credential}. O sistema não falsifica a publicação.",

  // Studio — avisos
  "studio.notice.loadError": "Erro ao carregar projeto: {error}",
  "studio.notice.previewAuthError":
    "Erro ao carregar preview autenticado: {error}",
  "studio.notice.canvasUpdated": "Canvas atualizado (v{version})",
  "studio.notice.canvasSaveError": "Erro ao salvar canvas: {error}",
  "studio.notice.generating": "Gerando componentes com o modelo local…",
  "studio.notice.noModel":
    "Nenhum modelo disponível. Baixe um modelo em Configurações primeiro.",
  "studio.notice.invalidComponents":
    "A IA não retornou componentes válidos. Tente descrever de outro jeito.",
  "studio.notice.generatedFresh":
    "{count} componentes gerados do zero pelo modelo local ({model}).",
  "studio.notice.generatedAdded":
    "{count} componentes adicionados pelo modelo local ({model}).",
  "studio.notice.generateError": "Falha ao gerar com IA: {error}",
  "studio.notice.previewError": "Não foi possível gerar a prévia: {error}",
  "studio.notice.downloaded": "HTML do app baixado.",
  "studio.notice.downloadError": "Não foi possível baixar o HTML: {error}",
  "studio.notice.undo": "Undo executado (v{version})",
  "studio.notice.undoError": "Nada para desfazer: {error}",
  "studio.notice.redo": "Redo executado (v{version})",
  "studio.notice.redoError": "Nada para refazer: {error}",
  "studio.notice.previewGenerated":
    "Preview ao vivo gerado no backend com sucesso!",
  "studio.notice.previewGenerateError": "Erro ao gerar preview: {error}",
  "studio.notice.exported":
    "Projeto exportado com sucesso (SHA-256 verificado)!",
  "studio.notice.exportError": "Erro ao exportar: {error}",

  // Studio — criação a partir de modelo
  "studio.create.confirm":
    'Criar "{name}"? O projeto atualmente aberto no editor será substituído por um novo projeto em branco.',
  "studio.create.confirmLabel": "Criar",
  "studio.create.success": 'Novo projeto "{name}" criado.',
  "studio.create.error": "Erro ao criar projeto a partir do modelo: {error}",
  "studio.templateName.site": "Novo Site Web",
  "studio.templateName.dashboard": "Novo Dashboard",
  "studio.templateName.slides": "Nova Apresentação",
  "studio.templateName.game": "Novo Jogo Web",
  "studio.templateName.app": "Novo App Móvel",

  // Studio — projeto inicial
  "studio.seed.hero": "Bem-vindo ao Studio do Hades",
  "studio.seed.lead":
    "Editor visual interativo de ponta a ponta com preview ao vivo, undo/redo e export rastreável.",
  "studio.seed.cta": "Experimentar Agora",

  // Automações — cabeçalho e alternância de vista
  "automations.title": "Automações",
  "automations.description":
    "Automatize fluxos recorrentes, configure gatilhos e deixe o agente trabalhar em segundo plano.",
  "automations.new": "Nova automação",
  "automations.view.list": "Lista",
  "automations.view.flow": "Editor visual (beta)",

  // Automações — editor visual de fluxo
  "automations.flow.hint":
    "Monte o fluxo arrastando nós da paleta, conecte as etapas e valide antes de publicar. O rascunho fica salvo neste navegador.",
  "automations.flow.published":
    "Fluxo publicado como automação. Veja em Rotinas Ativas.",

  // Automações — acompanhamento da missão criada
  "automations.followMission": "Acompanhar no Console de Missões →",

  // Automações — instruções do webhook
  "automations.webhook.title": "Gatilho de webhook criado — como acionar",
  "automations.webhook.close": "Fechar",
  "automations.webhook.intro":
    "Este gatilho roda apenas quando o endpoint abaixo recebe uma requisição autenticada (não roda por tempo). Envie um",
  "automations.webhook.introTail":
    "com o cabeçalho do segredo e uma chave de idempotência única por evento.",
  "automations.webhook.endpoint": "Endpoint (POST)",
  "automations.webhook.secretHeader": "Cabeçalho do segredo",
  "automations.webhook.idempotency":
    "Chave de idempotência (obrigatória, única por evento)",
  "automations.webhook.secretValue":
    "X-Ollama-Agent-Secret: <valor da variável de ambiente {env}>",
  "automations.webhook.idempotencyExample":
    "Idempotency-Key: <id único do evento>",
  "automations.webhook.secretExplain": "O valor de",
  "automations.webhook.secretExplainMid":
    "deve ser igual ao conteúdo da variável de ambiente",
  "automations.webhook.secretExplainTail":
    "configurada no servidor. Requisições sem a chave de idempotência ou com segredo inválido são recusadas.",

  // Automações — Cartão: agendamento
  "automations.card.schedule.title": "Agendamento",
  "automations.card.schedule.description":
    "Automatize trabalhos recorrentes escolhendo quando e com que frequência o Hades deve ser executado.",
  "automations.card.schedule.presetProgress":
    "⚡ Relatório diário de progresso (24h)",
  "automations.card.schedule.presetSecurity":
    "⚡ Varredura semanal de segurança (7d)",
  "automations.card.schedule.action": "Criar agendamento",
  "automations.card.schedule.objectiveProgress":
    "Auditar arquivos e gerar relatório de progresso",
  "automations.card.schedule.objectiveSecurity":
    "Verificar atualizações de dependências e segurança",

  // Automações — Cartão: gatilho
  "automations.card.trigger.title": "Gatilho",
  "automations.card.trigger.description":
    "Configure um webhook autenticado; o endpoint exige segredo em variável de ambiente e chave de idempotência.",
  "automations.card.trigger.presetTests":
    "⚡ Testar repositório ao detectar commits",
  "automations.card.trigger.presetInvoices":
    "⚡ Extrair dados de faturas recebidas",
  "automations.card.trigger.action": "Criar gatilho",
  "automations.card.trigger.objectiveTests":
    "Disparar testes e validar build quando novo commit for detectado",
  "automations.card.trigger.objectiveInvoices":
    "Processar arquivos e faturas adicionadas na pasta local",

  // Automações — Cartão: automação avançada
  "automations.card.advanced.title": "Automação avançada",
  "automations.card.advanced.description":
    "Descreva com suas próprias palavras o que o Hades deve fazer e escolha a frequência abaixo. A interpretação de horários em linguagem natural ainda não está disponível: por ora a automação roda em intervalo fixo.",
  "automations.card.advanced.placeholder":
    "Ex: Monitore meu repositório e gere um resumo...",
  "automations.card.advanced.frequency": "Frequência",
  "automations.card.advanced.start": "Começar",
  "automations.card.advanced.missingPrompt":
    "Descreva o que o Hades deve fazer antes de começar.",

  // Automações — intervalos
  "automations.interval.hour1": "A cada 1 hora",
  "automations.interval.hour6": "A cada 6 horas",
  "automations.interval.hour12": "A cada 12 horas",
  "automations.interval.day24": "Diariamente (24 horas)",
  "automations.interval.week7": "Semanalmente (7 dias)",
  "automations.interval.day24Long": "Diariamente (a cada 24 horas)",
  "automations.interval.week7Long": "Semanalmente (a cada 7 dias)",
  "automations.interval.everyHours": "A cada {hours} horas",

  // Automações — modal de criação
  "automations.modal.title": "Configurar Nova Rotina de Automação",
  "automations.modal.objective": "Objetivo / Instrução para o Agente",
  "automations.modal.objectivePlaceholder":
    "Ex: Comparar dados de concorrentes e atualizar relatório",
  "automations.modal.triggerType": "Tipo de gatilho",
  "automations.modal.triggerInterval": "Intervalo recorrente",
  "automations.modal.triggerWebhook": "Webhook/evento",
  "automations.modal.secretEnv": "Variável do segredo webhook",
  "automations.modal.secretEnvPlaceholder": "NOME_DA_VARIAVEL",
  "automations.modal.webhookNote":
    "Gatilhos de webhook não rodam por tempo: são acionados apenas pelo endpoint autenticado, cuja URL e instruções aparecem após salvar.",
  "automations.modal.frequency": "Frequência de Execução",
  "automations.modal.cancel": "Cancelar",
  "automations.modal.saving": "Salvando...",
  "automations.modal.save": "Salvar automação",

  // Automações — lista de rotinas
  "automations.list.title": "Rotinas Ativas ({count})",
  "automations.list.openConsole": "Abrir Console de Missões →",
  "automations.list.loading": "Carregando automações...",
  "automations.list.empty": "Nenhuma rotina configurada",
  "automations.list.emptyHint":
    "Use os cartões acima para criar seu primeiro agendamento ou gatilho autônomo.",

  // Automações — item da lista
  "automations.item.webhookOnly": "Acionada por webhook (não roda por tempo)",
  "automations.item.nextRun": "{interval} • Próxima execução: {when}",
  "automations.item.runNow": "Executar agora",
  "automations.item.runNowTitle": "Executar imediatamente",
  "automations.item.reprocess": "Reprocessar pendentes",
  "automations.item.reprocessTitle":
    "Reprocessar apenas os passos ignorados ou que falharam",
  "automations.item.approve": "Aprovar {step}",
  "automations.item.approveTitle": 'Aprovar "{step}" para esta execução',
  "automations.item.reject": "Recusar",
  "automations.item.rejectTitle": 'Recusar "{step}" para esta execução',
  "automations.item.pause": "Pausar",
  "automations.item.resume": "Retomar",
  "automations.item.deleteTitle": "Excluir rotina",

  // Automações — erros
  "automations.error.load": "Não foi possível carregar as automações.",
  "automations.error.missingObjective":
    "Informe o objetivo/instrução da automação.",
  "automations.error.missingSecret":
    "Informe a variável de ambiente com o segredo do webhook.",
  "automations.error.create": "Não foi possível criar a automação.",
  "automations.error.approval": "Não foi possível registrar a decisão.",
  "automations.error.reprocess": "Não foi possível reprocessar a automação.",
  "automations.error.run": "Não foi possível executar a automação.",
  "automations.error.toggle": "Não foi possível alterar o estado da automação.",
  "automations.error.delete": "Não foi possível excluir a automação.",

  // Automações — sucesso
  "automations.success.created": "Automação criada com sucesso!",
  "automations.success.approved":
    'Aprovação registrada em "{step}". Use "Reprocessar pendentes" para retomar o fluxo.',
  "automations.success.rejected":
    'Recusa registrada em "{step}". O fluxo não seguirá por esse caminho.',

  // Automações — reprocessamento e execução
  "automations.reprocess.failed": "Reprocessamento falhou: {error}",
  "automations.reprocess.errorUnknown": "erro não informado",
  "automations.reprocess.clean":
    "Reprocessamento concluído sem pendências: {summary}",
  "automations.reprocess.pending":
    "Reprocessamento concluído, ainda com {count} passo(s) pendente(s): {summary}",
  "automations.run.partial": "Rotina executada parcialmente: {summary}",
  "automations.run.done": "Rotina executada: {summary}",
  "automations.run.failed": "Execução da rotina falhou ({verdict}): {error}",

  // Automações — exclusão
  "automations.delete.confirm":
    "Tem certeza que deseja excluir esta automação?",
  "automations.delete.confirmLabel": "Excluir",

  // Empresa — página / cabeçalho
  "companyWorkspace.page.title": "Empresa",
  "companyWorkspace.eyebrow": "Sistema da Empresa",
  "companyWorkspace.title": "Crie e opere sua empresa",
  "companyWorkspace.description":
    "Estratégia, produto, engenharia, marketing, vendas, suporte e operações em um tenant com ciclos autônomos, métricas, approvals e pausa segura.",
  "companyWorkspace.refresh": "Atualizar",
  "companyWorkspace.loadError":
    "Não foi possível carregar o Sistema da Empresa.",
  "companyWorkspace.actionFailed": "A operação falhou.",
  "companyWorkspace.loading": "Carregando o Sistema da Empresa…",
  "companyWorkspace.created": "Empresa criada como tenant local.",
  "companyWorkspace.tenant": "tenant {id}",

  // Empresa — criação
  "companyWorkspace.start.title": "Comece com uma empresa",
  "companyWorkspace.start.namePlaceholder": "Nome da empresa",
  "companyWorkspace.start.missionPlaceholder":
    "Missão e resultado que a empresa entrega",
  "companyWorkspace.start.positioningPlaceholder":
    "Posicionamento e diferencial",
  "companyWorkspace.start.audiencePlaceholder": "Público-alvo",
  "companyWorkspace.start.offerPlaceholder": "Oferta principal",
  "companyWorkspace.start.offerLabel": "Oferta",
  "companyWorkspace.start.budgetPlaceholder": "Limite mensal em BRL",
  "companyWorkspace.start.budgetLabel": "Limite mensal",
  "companyWorkspace.start.create": "Criar empresa e tenant",
  "companyWorkspace.field.mission": "Missão",
  "companyWorkspace.field.positioning": "Posicionamento",
  "companyWorkspace.field.businessModel": "Modelo de negócio",
  "companyWorkspace.field.name": "Nome",

  // Empresa — Tel-Agent
  "companyWorkspace.telAgent.title": "Tel-Agent — canal textual operacional",
  "companyWorkspace.telAgent.description":
    "Comande esta empresa por texto e receba o retorno persistido no tenant. As operações são allowlisted e locais; telefonia, WhatsApp e mensagens externas permanecem não configurados até uma conexão homologada.",
  "companyWorkspace.telAgent.operationLabel": "Operação Tel-Agent",
  "companyWorkspace.telAgent.operationReport": "Ler relatório operacional",
  "companyWorkspace.telAgent.operationBacklog": "Criar item no backlog",
  "companyWorkspace.telAgent.operationCampaign": "Criar rascunho de campanha",
  "companyWorkspace.telAgent.operationTitlePlaceholder": "Título da operação",
  "companyWorkspace.telAgent.priorityPlaceholder": "Prioridade",
  "companyWorkspace.telAgent.priorityLabel": "Prioridade Tel-Agent",
  "companyWorkspace.telAgent.descriptionPlaceholder":
    "Objetivo do rascunho (não publica)",
  "companyWorkspace.telAgent.descriptionLabel": "Objetivo da campanha",
  "companyWorkspace.telAgent.messagePlaceholder":
    "Ex.: crie uma tarefa para revisar o onboarding",
  "companyWorkspace.telAgent.messageLabel": "Mensagem Tel-Agent",
  "companyWorkspace.telAgent.history":
    "Histórico limitado às últimas 100 trocas; entradas são redigidas antes da persistência.",
  "companyWorkspace.telAgent.run": "Executar operação",
  "companyWorkspace.telAgent.approvalPending":
    "Tel-Agent criou um rascunho local; approval continua obrigatório.",
  "companyWorkspace.telAgent.completed":
    "Tel-Agent concluiu a operação local.",
  "companyWorkspace.telAgent.failed":
    "Tel-Agent não conseguiu concluir a operação.",

  // Empresa — indicadores
  "companyWorkspace.stats.status": "Estado",
  "companyWorkspace.stats.backlog": "Backlog aberto",
  "companyWorkspace.stats.goalsOnTrack": "Metas em dia",
  "companyWorkspace.stats.activeCycles": "Ciclos ativos",
  "companyWorkspace.stats.budgetUsed": "Orçamento usado",
  "companyWorkspace.stats.anomalies": "Anomalias",
  "companyWorkspace.status.paused": "pausada",
  "companyWorkspace.status.active": "ativa",
  "companyWorkspace.status.none": "sem empresa",
  "companyWorkspace.cycleState.enabled": "ativo",
  "companyWorkspace.cycleState.paused": "pausado",

  // Empresa — identidade
  "companyWorkspace.identity.title": "Identidade, posicionamento e modelo",
  "companyWorkspace.identity.nameUpdated": "Nome atualizado.",
  "companyWorkspace.identity.missionUpdated": "Missão atualizada.",
  "companyWorkspace.identity.positioningUpdated":
    "Posicionamento atualizado.",
  "companyWorkspace.identity.businessModelUpdated":
    "Modelo de negócio atualizado.",
  "companyWorkspace.departments.title": "Departamentos virtuais",

  // Empresa — roadmap
  "companyWorkspace.roadmap.title": "Roadmap estratégico",
  "companyWorkspace.roadmap.placeholder": "Próximo resultado estratégico",
  "companyWorkspace.roadmap.itemLabel": "Item de roadmap",
  "companyWorkspace.roadmap.added": "Item adicionado ao roadmap.",
  "companyWorkspace.roadmap.empty": "Nenhum marco definido.",
  "companyWorkspace.add": "Adicionar",

  // Empresa — metas
  "companyWorkspace.goals.title": "Metas e KPIs",
  "companyWorkspace.goals.titlePlaceholder": "Meta",
  "companyWorkspace.goals.titleLabel": "Título da meta",
  "companyWorkspace.goals.metricPlaceholder": "KPI: leads, receita, clientes",
  "companyWorkspace.goals.metricLabel": "Métrica",
  "companyWorkspace.goals.targetLabel": "Alvo",
  "companyWorkspace.goals.created": "Meta criada.",
  "companyWorkspace.goals.empty": "Os KPIs da empresa aparecerão aqui.",
  "companyWorkspace.create": "Criar",

  // Empresa — backlog
  "companyWorkspace.backlog.title": "Backlog priorizado",
  "companyWorkspace.backlog.placeholder": "Tarefa ou experimento",
  "companyWorkspace.backlog.itemLabel": "Item de backlog",
  "companyWorkspace.backlog.priorityLabel": "Prioridade",
  "companyWorkspace.backlog.added": "Item adicionado e priorizado.",
  "companyWorkspace.backlog.empty": "Backlog vazio.",

  // Empresa — ciclos
  "companyWorkspace.cycles.title": "Ciclos autônomos",
  "companyWorkspace.cycles.nameLabel": "Nome do ciclo",
  "companyWorkspace.cycles.objectiveLabel": "Objetivo do ciclo",
  "companyWorkspace.cycles.frequencyLabel": "Frequência",
  "companyWorkspace.cycles.frequencyDaily": "Diário",
  "companyWorkspace.cycles.frequencyWeekly": "Semanal",
  "companyWorkspace.cycles.create": "Criar ciclo",
  "companyWorkspace.cycles.created": "Ciclo criado e ligado ao scheduler.",
  "companyWorkspace.cycles.defaultName": "Ciclo diário",
  "companyWorkspace.cycles.defaultObjective":
    "Revisar métricas, roadmap e executar a próxima tarefa priorizada",

  // Empresa — segurança e gastos
  "companyWorkspace.security.title": "Segurança, budget e pausa",
  "companyWorkspace.security.approvalsNotice":
    "Ações de gasto, anúncio, contrato e mensagem exigem approval.",
  "companyWorkspace.security.amountPlaceholder": "Valor em centavos",
  "companyWorkspace.security.amountLabel": "Valor de gasto",
  "companyWorkspace.security.categoryLabel": "Categoria",
  "companyWorkspace.security.categoryAds": "Anúncios",
  "companyWorkspace.security.categoryContract": "Contrato",
  "companyWorkspace.security.categoryInfrastructure": "Infraestrutura",
  "companyWorkspace.security.categorySales": "Vendas",
  "companyWorkspace.security.recordSpend": "Registrar gasto",
  "companyWorkspace.security.spendRequested":
    "Gasto solicitado; decision pendente antes de atualizar o budget.",
  "companyWorkspace.security.resumed": "Empresa retomada.",
  "companyWorkspace.security.paused": "Empresa pausada com segurança.",
  "companyWorkspace.security.resume": "Retomar",
  "companyWorkspace.security.pause": "Pausar",
  "companyWorkspace.signal.title": "Sinal operacional",
  "companyWorkspace.signal.monthlyBudget": "Orçamento mensal",
  "companyWorkspace.signal.consumed": "Consumido",
  "companyWorkspace.signal.persistence":
    "Persistência local, tenant e auditoria server-side.",

  // Endpoint — página e cabeçalho
  "endpoint.title": "Computadores & Endpoint",
  "endpoint.heading": "Computadores",
  "endpoint.description":
    "Gerencie seus computadores locais e na nuvem para execução segura de missões e Computer Use.",
  "endpoint.cloud.create": "Criar computador na nuvem",
  "endpoint.cloud.prerequisiteStrong": "Pré-requisito:",
  "endpoint.cloud.prerequisiteText":
    "computadores na nuvem exigem um provedor de infraestrutura configurado (credenciais e região). Este build local-first não inclui um provedor de nuvem, portanto a criação ainda não está disponível. Enquanto isso, use o seu computador local, que já aparece conectado abaixo.",

  // Endpoint — host local
  "endpoint.host.thisComputer": "Este computador",
  "endpoint.host.osLocal": "Local",
  "endpoint.host.statusOnline": "Host Ativo",
  "endpoint.host.statusOffline": "Host Offline",
  "endpoint.host.statusChecking": "Verificando host",
  "endpoint.host.osLine": "• {os} / Local {version}",
  "endpoint.host.portLabel": "Porta da API:",
  "endpoint.host.osLabel": "Sistema:",
  "endpoint.host.versionLabel": "Versão do servidor:",
  "endpoint.remote.button": "Acesso remoto",
  "endpoint.remote.before":
    "Por padrão este host responde apenas localmente. Para acesso remoto é preciso ativar",
  "endpoint.remote.strong": "“Expor o Hades na rede”",
  "endpoint.remote.afterStart": "em Configurações e proteger a porta",
  "endpoint.remote.afterPort":
    "com TLS/autenticação (por exemplo, um proxy reverso). Enquanto isso não for feito, o acesso remoto permanece desligado por segurança.",
  "endpoint.settings": "Configurações",
  "endpoint.tasks": "Tarefas",

  // Endpoint — conectar dispositivo
  "endpoint.connect.title": "Conectar outro dispositivo",
  "endpoint.connect.description":
    "Adicione nós remotos, instâncias na nuvem ou conecte seu smartphone para controlar tarefas em andamento.",
  "endpoint.connect.computer": "Conectar meu computador",
  "endpoint.connect.computerHintBefore":
    "Para conectar outro computador como nó remoto é preciso que ele rode o Hades e seja alcançável pela rede (host e porta",
  "endpoint.connect.computerHintAfter":
    "expostos com TLS). Não há, neste build local-first, um provedor que faça esse provisionamento automaticamente.",
  "endpoint.pairing.generating": "Gerando código…",
  "endpoint.pairing.button": "Parear o seu telefone",
  "endpoint.pairing.startFailed":
    "Não foi possível iniciar o pareamento agora.",
  "endpoint.pairing.codeLabel": "Código de pareamento:",
  "endpoint.pairing.instructions":
    "Abra o app complementar do Hades no seu celular e informe este código para concluir o pareamento.",
  "endpoint.pairing.expiresAt": "O código expira às {time}.",
  "endpoint.pairing.onlyPhone":
    "O pareamento só se conclui a partir do app no telefone; nada é enviado para fora deste computador.",

  // Endpoint — endereços da API
  "endpoint.api.title": "Endereços da API & Conexão",
  "endpoint.api.description":
    "Use os endpoints abaixo para conectar Claude Code, Codex, Cursor e SDKs locais.",
  "endpoint.api.secure": "Seguro",
  "endpoint.api.native": "Ollama Nativo",
  "endpoint.api.openai": "Compatível com OpenAI",
  "endpoint.api.openaiLabel": "compatível com OpenAI",
  "endpoint.api.anthropic": "Compatível com Anthropic (Claude Code)",
  "endpoint.api.anthropicLabel": "compatível com Anthropic",
  "endpoint.copy.aria": "Copiar {label}",

  // Endpoint — gateway
  "endpoint.gateway.title": "Configurar inferência de terceiros",
  "endpoint.gateway.description":
    "Um único endereço para os modelos locais e para as APIs de terceiros. Se um provedor recusar a chamada, o gateway tenta o próximo candidato sem trocar a configuração do Claude ou do Codex.",
  "endpoint.gateway.rotationDefault": "Rotação padrão · 3 tentativas",
  "endpoint.gateway.baseUrl": "URL base do gateway",
  "endpoint.gateway.apiKey": "Chave de API do gateway",
  "endpoint.gateway.keyStored":
    "Guardada fora do arquivo de configuração, na variável",
  "endpoint.gateway.copyKey": "chave do gateway",
  "endpoint.gateway.hide": "Ocultar",
  "endpoint.gateway.show": "Mostrar",
  "endpoint.gateway.noKey":
    "Nenhuma chave ativa. Sem chave, o gateway aceita apenas pedidos deste computador — é o que mantém a inferência local privada.",
  "endpoint.gateway.generating": "Gerando…",
  "endpoint.gateway.rotate": "Gerar nova chave",
  "endpoint.gateway.generate": "Gerar chave do gateway",
  "endpoint.gateway.rotateFailed":
    "Não foi possível gerar a chave do gateway agora.",
  "endpoint.gateway.loopbackBefore": "O gateway está no modo",
  "endpoint.gateway.loopbackStrong": "somente este computador",
  "endpoint.gateway.loopbackAfter":
    ": clientes remotos recebem 401 até existir uma chave. Gere a chave acima antes de apontar outro dispositivo para este endereço.",
  "endpoint.gateway.aliases": "Apelidos aceitos no campo model:",
  "endpoint.gateway.providersOne": "{count} provedor declarado em",
  "endpoint.gateway.providersMany": "{count} provedores declarados em",

  // Endpoint — integração rápida
  "endpoint.integration.heading": "Comandos de Integração Rápida",
  "endpoint.integration.activeModel": "Modelo ativo:",
  "endpoint.integration.commandAria": "Comando de integração {title}",
  "endpoint.claude.description":
    "Execução com modelos locais através da rota compatível.",
  "endpoint.claude.aria": "Comando de integração Claude",
  "endpoint.codex.description":
    "Ollama configura e abre o Codex com o modelo escolhido.",
  "endpoint.codex.aria": "Comando de integração Codex",

  // Onboarding — comum
  "onboarding.signIn": "Entrar",
  "onboarding.continue": "Continuar",
  "onboarding.retry": "Tentar de novo",

  // Onboarding — introdução
  "onboarding.intro.alt": "Hades dando boas-vindas",
  "onboarding.intro.title": "Bem-vindo ao Hades!",
  "onboarding.intro.tagline":
    "Rode modelos abertos com seus agentes de código para gastar menos mantendo seus dados privados.",
  "onboarding.features.connectApps.title": "Conecte seus apps",
  "onboarding.features.connectApps.description":
    "Turbine seus apps de código com modelos abertos",
  "onboarding.features.switchModel.title": "Troque de modelo facilmente",
  "onboarding.features.switchModel.description":
    "Alterne entre modelos de ponta em um clique.",
  "onboarding.features.yourData.title": "Seus dados continuam seus",
  "onboarding.features.yourData.description":
    "Modelos locais processam seus pedidos neste computador. Modelos de nuvem só recebem pedidos quando você escolhe usá-los.",

  // Onboarding — conta
  "onboarding.welcome.title": "Crie uma conta",
  "onboarding.welcome.subtitle":
    "Crie sua conta para acessar modelos abertos maiores e mais rápidos.",
  "onboarding.welcome.privacy":
    "Modelos locais processam seus pedidos neste computador. Se você ativar os modelos de nuvem, os pedidos são enviados ao provedor escolhido quando você os usa.",
  "onboarding.welcome.openingApps": "Abrindo os apps…",
  "onboarding.welcome.finishInBrowser": "Conclua no seu navegador…",
  "onboarding.welcome.createAccount": "Criar conta",
  "onboarding.welcome.cloudOptIn":
    "Ativar modelos de nuvem grátis da minha conta",
  "onboarding.welcome.useLocal": "Não, obrigado — vou usar os modelos locais",
  "onboarding.welcome.saveCloudFailed":
    "Não foi possível salvar sua escolha de modelos de nuvem. A ativação não foi confirmada; tente novamente.",

  // Onboarding — primeiro modelo
  "onboarding.run.title": "Baixe seu primeiro modelo",
  "onboarding.run.description":
    "Escolha um modelo local para começar. O Hades baixa e roda o modelo no seu computador — sem precisar de terminal.",
  "onboarding.run.modelLabel": "Modelo local",
  "onboarding.run.modelPlaceholder": "ex.: {model}",
  "onboarding.run.preparing": "Preparando download…",
  "onboarding.run.downloading": "Baixando…",
  "onboarding.run.download": "Baixar",
  "onboarding.run.fileNote":
    "O arquivo pode ser grande; mantenha o Ollama em execução durante o processo.",
  "onboarding.run.modelReady": "Modelo {model} pronto para uso local.",
  "onboarding.run.downloadFailed":
    "Não foi possível baixar o modelo neste computador.",

  // Onboarding — Claude conectado
  "onboarding.claudeIntro.alt":
    "Exemplo de mapeamento de modelos do Claude nas configurações do Hades",
  "onboarding.claudeIntro.title": "Acesse os modelos do Hades no seu Claude",
  "onboarding.claudeIntro.description":
    "O Hades roteia os modelos do Claude automaticamente. Abra as configurações do Hades para escolher quais modelos o Claude Desktop usa.",

  // Onboarding — apps
  "onboarding.apps.recommended": "Recomendados",
  "onboarding.apps.titleApps": "Apps",
  "onboarding.apps.titleOthers": "Outros apps",
  "onboarding.apps.empty": "Nenhum app encontrado.",
  "onboarding.apps.loadFailed": "Não foi possível carregar as integrações.",
  "onboarding.apps.checking": "Verificando integrações…",
  "onboarding.apps.copyAria": "Copiar comando do {name}",
  "onboarding.apps.copiedAria": "Comando do {name} copiado",
  "onboarding.apps.copyFailedTitle":
    "Não foi possível copiar o comando do {name}",
  "onboarding.apps.copiedTitle": "Comando de inicialização copiado.",
  "onboarding.apps.pasteHint": "Cole no seu terminal",
  "onboarding.apps.selectHint":
    "Selecione e copie o comando abaixo e cole no seu terminal.",
  "onboarding.chatgpt.description": "Use os modelos do Hades no ChatGPT",

  // Onboarding — conexão com o Claude
  "onboarding.claude.readStatusFailed":
    "O Hades não conseguiu ler o status da conexão com o Claude.",
  "onboarding.claude.openFailed":
    "O Hades conectou o Claude, mas não conseguiu abrir o app.",
  "onboarding.claude.connectFailed":
    "O Hades não conseguiu conectar ao Claude.",
  "onboarding.claude.disconnectFailed":
    "O Hades não conseguiu desconectar do Claude.",
  "onboarding.claude.timeoutOpen":
    "O Claude está demorando demais para abrir. Verifique o Claude e tente de novo.",
  "onboarding.claude.timeoutConnect":
    "O Claude está demorando demais para conectar. Verifique o Claude e tente de novo.",
  "onboarding.claude.timeoutAction":
    "O Claude está demorando demais para {action}. Tente de novo.",
  "onboarding.claude.actionConnect": "conectar",
  "onboarding.claude.actionDisconnect": "desconectar",
  "onboarding.claude.completeFailed":
    "O Hades não conseguiu concluir a conexão com o Claude.",
  "onboarding.claude.installNotDetected":
    "A instalação do Claude não foi detectada. Tente novamente.",
  "onboarding.claude.installedRestart":
    "O Claude está instalado. Ative o Conectar para reiniciá-lo com o Hades.",
  "onboarding.claude.installerFailed":
    "O Hades não conseguiu abrir o instalador do Claude.",
  "onboarding.claude.macosOnly":
    "A conexão com o Claude está disponível no app do Hades para macOS.",
  "onboarding.claude.restartToUse":
    "Reiniciar Claude Desktop para usar Ollama? Qualquer tarefa em execução será interrompida.",
  "onboarding.claude.restartToRemove":
    "Reiniciar Claude Desktop para remover Ollama? Qualquer tarefa em execução será interrompida.",
  "onboarding.claude.restartLabel": "Reiniciar",
  "onboarding.claude.connected": "Conectado ao Hades · {count}",
  "onboarding.claude.installing":
    "O Hades está baixando o instalador do Claude…",
  "onboarding.claude.waitingInstall":
    "Conclua a instalação do Claude. O Hades vai conectá-lo automaticamente.",
  "onboarding.claude.connectingDetail": "Conectando o Claude ao Hades…",
  "onboarding.claude.launchingDetail": "Abrindo o Claude…",
  "onboarding.claude.disconnectingDetail":
    "Restaurando a conexão normal do Claude…",
  "onboarding.claude.notInstalled":
    "Vamos baixar o Claude e conectá-lo ao Hades.",
  "onboarding.claude.ready": "Use os modelos do Hades no seu Claude Code.",
  "onboarding.claude.disconnect": "Desconectar Claude",
  "onboarding.claude.connecting": "Conectando Claude",
  "onboarding.claude.connect": "Conectar Claude",
  "onboarding.claude.disconnectTitle": "Desconectar",
  "onboarding.claude.connectTitle": "Conectar",
  "onboarding.claude.statusDownloading": "Baixando…",
  "onboarding.claude.statusFinishingInstall": "Conclua a instalação…",
  "onboarding.claude.statusConnecting": "Conectando…",
  "onboarding.claude.statusLaunching": "Abrindo…",
  "onboarding.claude.statusDisconnecting": "Desconectando…",

  // Agent Center (AgentCenterPanel)
  "agentCenter.title": "Agentes e Equipas",
  "agentCenter.subtitle": "ledger determinístico · drift por hash",
  "agentCenter.actions.list": "Listar agentes",
  "agentCenter.actions.catalog": "Ver catálogo",
  "agentCenter.actions.evaluation": "Ver critérios de avaliação",
  "agentCenter.actions.teams": "Ver equipas",
  "agentCenter.actions.runEvaluation": "Correr avaliação real",
  "agentCenter.actions.quarantine": "Reter em quarentena",
  "agentCenter.actions.triage": "Triar importação",
  "agentCenter.actions.backup": "Backup e restauro",
  "agentCenter.actions.drift": "Verificar integridade",
  "agentCenter.actions.remove": "Remover instalacao",
  "agentCenter.actions.update": "Actualizar (com backup)",
  "agentCenter.actions.install": "Registar instalação",
  "agentCenter.target.aria": "Formato de destino",
  "agentCenter.destination.placeholder": "destino relativo",
  "agentCenter.destination.aria": "Destino relativo",
  "agentCenter.idle": "Nada consultado nesta sessão.",
  "agentCenter.loading": "A consultar...",
  "agentCenter.list.countOne": "{count} instalação registada",
  "agentCenter.list.countMany": "{count} instalações registadas",
  "agentCenter.catalog.version":
    "versão {version} · {total} persona(s) · {evaluated} avaliada(s)",
  "agentCenter.catalog.noneEvaluated":
    "Nenhuma persona do catálogo foi avaliada. Isto não é «aprovado»: é «não medido».",
  "agentCenter.catalog.unrated": "NÃO AVALIADO",
  "agentCenter.evaluation.noNote": "sem nota do servidor",
  "agentCenter.evaluation.inferenceOn": "execução ligada a um motor",
  "agentCenter.evaluation.inferenceOff":
    "execução contra modelo NÃO ligada neste runtime",
  "agentCenter.evaluation.suiteLine":
    "{division} · v{version} · {cases} caso(s) · {refusals} de recusa/injeção",
  "agentCenter.evaluation.states":
    "{count} estados declarados · {terminal} terminal(is)",
  "agentCenter.teams.summary":
    "{total} equipa(s) · {functional} funcional(is) · {missing} com ferramentas em falta · {invalid} inválida(s)",
  "agentCenter.teams.noneProven":
    "nenhuma equipa foi provada numa missão real",
  "agentCenter.teams.rolesOne": "{count} papel",
  "agentCenter.teams.rolesMany": "{count} papéis",
  "agentCenter.teams.valid": "válida",
  "agentCenter.teams.invalid": "INVÁLIDA ({count})",
  "agentCenter.teams.missingTools": "falta: {tools}",
  "agentCenter.teams.noMissingTools": "sem ferramentas em falta declaradas",
  "agentCenter.triage.placeholder":
    "Cole o manifesto da persona (JSON) para triagem",
  "agentCenter.triage.status":
    "estado: {state} · {blocks} bloqueio(s) · {warnings} aviso(s) · {install}",
  "agentCenter.triage.installed": "INSTALADO",
  "agentCenter.triage.notInstalled": "nada instalado",
  "agentCenter.triage.note":
    "a triagem lê e recusa; não escreve nem ativa nada",
  "agentCenter.realEval.status":
    "estado: {state} · {cases} caso(s) · {failed} falha(s) · motor: {engine}",
  "agentCenter.realEval.note":
    "corre a suite contra o motor local; não aprova para produção",
  "agentCenter.realEval.passed": "passou",
  "agentCenter.realEval.failed": "falhou",
  "agentCenter.quarantine.countOne": "{count} persona retida",
  "agentCenter.quarantine.countMany": "{count} personas retidas",
  "agentCenter.quarantine.release": "libertar",
  "agentCenter.quarantine.empty":
    "Nenhuma persona retida. Reter uma cria a primeira entrada.",
  "agentCenter.quarantine.reviewerPlaceholder":
    "revisor (obrigatório para libertar)",
  "agentCenter.quarantine.releaseNote":
    "Libertar leva a SECURITY_REVIEWED e exige revisor. Não aprova para produção.",
  "agentCenter.backups.countOne": "{count} backup guardado",
  "agentCenter.backups.countMany": "{count} backups guardados",
  "agentCenter.backups.restore": "restaurar",
  "agentCenter.backups.restored": "Restaurado do backup",
  "agentCenter.backups.empty":
    "Nenhum backup ainda. Guardar um cria o primeiro.",
  "agentCenter.backups.restoreNote":
    "Restaurar exige escolher um id: não há «restaurar o mais recente».",
  "agentCenter.install.created":
    "201 criado · {id} · {bytes} bytes renderizados",
  "agentCenter.install.existing":
    "200 já registado · {id} · {bytes} bytes renderizados",
  "agentCenter.install.noId": "sem id",
  "agentCenter.update.done": "Actualizado com backup",
  "agentCenter.update.bytes": "bytes",
  "agentCenter.remove.line": "Removido do ledger: {id} · {state}",
  "agentCenter.remove.fileKept": "o ficheiro no destino foi preservado",
  "agentCenter.remove.noFile": "sem ficheiro a preservar",
  "agentCenter.error.centerMissing":
    "O Agent Center não está ligado neste runtime (503): não é «nenhum agente», é «sem armazenamento».",
  "agentCenter.error.conflict":
    "Já existe um registo com esta chave e hashes diferentes: resolva o conflito, não reinstale por cima.",
  "agentCenter.error.generic": "Não foi possível correr.",
  "agentCenter.footer":
    "Registar uma instalação não escreve no destino: calcula os hashes e grava no ledger. O destino tem de ser relativo e o servidor recusa `..` — um destino que o cliente escreve é um destino que o cliente escolhe.",
  // Settings (Settings.tsx)
  "settings.interfaceMode.title": "Modo da interface",
  "settings.interfaceMode.description":
    "Simples mostra só o essencial. Avançado também mostra MCP, diagnósticos e detalhes técnicos.",
  "settings.interfaceMode.simple": "Simples",
  "settings.interfaceMode.advanced": "Avançado",
  // Idioma da interface (controlo em SettingsLanguage.tsx)
  "settings.language.title": "Idioma da interface",
  "settings.language.description":
    "Escolhe o idioma dos textos da interface. A escolha fica guardada neste dispositivo.",
  "settings.language.pt-BR": "Português (Brasil)",
  "settings.language.en": "English",
  "settings.discardDrafts.message":
    "Descartar as alterações de modelos dos apps que ainda não foram aplicadas?",
  "settings.discardDrafts.confirm": "Descartar",
  "settings.reset.failed":
    "Não foi possível restaurar todas as configurações. Confira as opções acima e tente novamente.",
  "settings.reset.restoring": "Restaurando…",
  "settings.reset.action": "Restaurar padrões",
  "settings.native.loading": "Consultando configuração nativa…",
  "settings.native.unavailable": "Configuração nativa indisponível neste momento",
  "settings.native.unavailableHint":
    "O painel agentic acima continua disponível para diagnóstico sanitizado; inicie o servidor Ollama para editar as preferências nativas.",
  "settings.account.upgrade": "Fazer upgrade",
  "settings.account.manage": "Gerenciar",
  "settings.account.disconnect": "Desconectar",
  "settings.account.connect": "Entrar",
  "settings.account.label": "Conta Ollama",
  "settings.account.notConnected":
    "Não conectada · entre para usar modelos na nuvem e busca na web",
  "settings.account.title": "Conta & Operador ({name})",
  "settings.account.storageLabel": "Armazenamento Local:",
  "settings.account.storageFallback": "Workspace local ativo",
  "settings.account.databaseLabel": "Banco de Dados:",
  "settings.account.databaseValue": "SQLite (armazenamento local)",
  "settings.account.modeLabel": "Modo:",
  "settings.account.modeValue": "Local-first (sem dependência de nuvem)",
  "settings.cloud.label": "Nuvem",
  "settings.cloud.forcedOff":
    "A variável de ambiente OLLAMA_NO_CLOUD está forçando a nuvem desligada.",
  "settings.cloud.description": "Ativar modelos na nuvem e busca na web.",
  "settings.appsMenu.label": "Mostrar apps no menu",
  "settings.appsMenu.description":
    "Exibir os apps conectados no topo do menu do Ollama.",
  "settings.autoUpdate.label": "Baixar atualizações automaticamente",
  "settings.autoUpdate.on":
    "Baixar atualizações automaticamente quando disponíveis.",
  "settings.autoUpdate.off": "As atualizações não serão baixadas automaticamente.",
  "settings.expose.label": "Expor o Hades na rede",
  "settings.expose.description":
    "Permitir que outros dispositivos ou serviços acessem o Hades.",
  "settings.modelsDir.label": "Local dos modelos",
  "settings.modelsDir.description": "Local onde os modelos são armazenados.",
  "settings.modelsDir.browse": "Procurar",
  "settings.context.label": "Tamanho do contexto",
  "settings.context.description":
    "O tamanho do contexto define quanto da conversa o modelo consegue lembrar e usar para gerar as respostas. Valores maiores mantêm mais histórico, mas consomem mais memória.",
  "settings.apps.heading": "Apps",
  "settings.agentMode.label": "Ativar modo agente",
  "settings.agentMode.description":
    "Usar ferramentas em múltiplas etapas para cumprir solicitações",
  "settings.toolsMode.label": "Ativar modo ferramentas",
  "settings.toolsMode.description":
    "Usar ferramentas em uma única etapa para cumprir solicitações",
  "settings.personalization.title": "Personalização",
  "settings.personalization.description":
    "Instruções personalizadas e memória de longo prazo para orientar o comportamento do agente.",
  "settings.personalization.languageLabel": "Idioma:",
  "settings.personalization.languageBefore": "o Hades responde sempre em",
  "settings.personalization.languageName": "português do Brasil",
  "settings.personalization.languageAfter":
    "por padrão (a menos que você peça outro idioma na conversa). Isso já está ativo.",
  "settings.personalization.instructionsLabel": "Instruções personalizadas",
  "settings.personalization.instructionsBefore":
    "Anote preferências de estilo/resposta. Estas instruções são salvas neste computador e",
  "settings.personalization.instructionsStrong":
    "enviadas ao modelo em toda conversa",
  "settings.personalization.instructionsAfter":
    ", junto com o comportamento padrão do Hades.",
  "settings.personalization.instructionsPlaceholder":
    "Ex.: respostas curtas e diretas; sempre com exemplos.",
  "settings.personalization.memoryTitle": "Memória de Longo Prazo",
  "settings.personalization.memoryDescription":
    "Consulta de contexto entre sessões ainda não está configurada neste ambiente.",
  "settings.personalization.memoryUnavailable": "Indisponível",
  "settings.saved": "Salvo",
  "settings.operator.defaultName": "Operador local",
  "settings.operator.defaultPlan": "Modo Local-first",
  "settings.error.connectUrl": "Failed to get connect URL",
  "settings.error.connectFailed": "Failed to connect to Ollama account",
  "settings.error.disconnectFailed": "Failed to disconnect Ollama account",
  // Company campaign scheduling (CompanyCampaignSchedulingPanel)
  "companyCampaign.load.panelFailed": "Painel de campanhas indisponível.",
  "companyCampaign.load.channelsFailed": "Catálogo de marketplaces indisponível.",
  "companyCampaign.action.failed": "A operação de campanha falhou.",
  "companyCampaign.sync.done":
    "Sincronização executada contra o dublê de sandbox.",
  "companyCampaign.sync.failed": "A sincronização falhou.",
  "companyCampaign.eyebrow": "Campanhas e marketplaces",
  "companyCampaign.title":
    "Agendamento, métricas medidas e sincronização em sandbox",
  "companyCampaign.intro.before":
    "Agendar exige campanha aprovada e início no futuro; métrica exige campanha que já executou e apenas soma. A sincronização de marketplace envia",
  "companyCampaign.intro.items": "os itens que você montar",
  "companyCampaign.intro.after": "contra um dublê local:",
  "companyCampaign.intro.never": "nada é publicado em plataforma externa",
  "companyCampaign.refresh": "Atualizar painel",
  "companyCampaign.external.label": "Publicação externa:",
  "companyCampaign.schedule.heading": "Agendar e medir campanha",
  "companyCampaign.fields.hours": "Início em horas",
  "companyCampaign.fields.impressions": "Impressões medidas",
  "companyCampaign.fields.clicks": "Cliques medidos",
  "companyCampaign.campaigns.empty":
    "Nenhuma campanha ainda: crie uma no painel Growth OS acima.",
  "companyCampaign.scheduledFor": "Agendada para {date}",
  "companyCampaign.measured":
    "Medido: {impressions} impressões · {clicks} cliques · {conversions} conversões",
  "companyCampaign.action.schedule": "Agendar",
  "companyCampaign.action.metrics": "Registrar métricas",
  "companyCampaign.success.scheduled": "Campanha agendada no sandbox.",
  "companyCampaign.success.metrics":
    "Métricas medidas registradas (somam ao total).",
  "companyCampaign.total":
    "Total medido: {impressions} impressões · {clicks} cliques · CTR {ctr} · conversão {conversion} · gasto {spend}",
  "companyCampaign.ingestions.prefix":
    "Entregas aceitas (idempotência): {count} — reentrega com a mesma chave",
  "companyCampaign.ingestions.noSum": "não soma",
  "companyCampaign.manual.prefix":
    "Nenhuma entrega automática registrada: os números acima vêm de",
  "companyCampaign.manual.badge": "registro manual",
  "companyCampaign.marketplaces.heading": "Marketplaces",
  "companyCampaign.marketplaces.descriptionBefore":
    "O catálogo informa o estado",
  "companyCampaign.marketplaces.descriptionStrong": "real",
  "companyCampaign.marketplaces.descriptionAfter":
    "do canal. Sem credencial configurada, o canal não está pronto — e o painel diz isso em vez de exibir um selo verde.",
  "companyCampaign.channels.empty": "Nenhum canal no catálogo.",
  "companyCampaign.liveTested": "Testado ao vivo: {value}",
  "companyCampaign.yes": "sim",
  "companyCampaign.no": "não",
  "companyCampaign.items.heading": "Itens a sincronizar",
  "companyCampaign.items.noProducts":
    "Nenhum produto no catálogo: cadastre um produto no Growth OS para montar o item.",
  "companyCampaign.items.stock": "Estoque",
  "companyCampaign.items.add": "Adicionar item",
  "companyCampaign.items.line":
    "{entity} · {sku} · versão {version} · estoque {quantity}",
  "companyCampaign.items.syncOne": "Sincronizar {count} item (sandbox)",
  "companyCampaign.items.syncMany": "Sincronizar {count} itens (sandbox)",
  "companyCampaign.items.needOne":
    "Adicione ao menos um item: sincronizar sem item é recusado pelo servidor.",
  "companyCampaign.sync.summary":
    "{applied} aplicados · {skipped} ignorados · {conflicts} conflitos · {used}/{limit} operações · {homologation}",

  // Home (HomePage.tsx)
  "home.actions.chat.description": "Chat local-first com modelos do Ollama e provedores configurados.",
  "home.actions.chat.eyebrow": "Chat e Modelos",
  "home.actions.chat.title": "Conversa com modelo",
  "home.actions.compile.description": "Planeje, execute em split-screen, aprove etapas e gere código com artifacts.",
  "home.actions.compile.eyebrow": "Compilar >",
  "home.actions.compile.title": "Sites, aplicativos e jogos",
  "home.actions.create.description": "Documentos executivos, apresentações e mídias geradas pelas missões.",
  "home.actions.create.eyebrow": "Criar >",
  "home.actions.create.title": "Slides, imagens e vídeos",
  "home.actions.local.description": "Abrir e orquestrar arquivos e projetos locais com contexto persistente.",
  "home.actions.local.eyebrow": "Workspace local",
  "home.actions.local.title": "Começar a partir de arquivo local",
  "home.attachments.aria": "Arquivos anexados",
  "home.attachments.attachAria": "Anexar arquivo",
  "home.attachments.chooseAria": "Escolher arquivos para anexar",
  "home.attachments.limitCount": "{filename}: limite de {count} arquivos por tarefa",
  "home.attachments.limitTotal": "{filename}: limite total de {megabytes} MB por tarefa",
  "home.attachments.note": "Anexos são enviados ao Hades para indexação local (até 10 arquivos, 10 MB cada e 32 MB no total). Imagens ainda não são indexadas. O modo automático recebe somente leitura; escrita exige consentimento e revisão.",
  "home.attachments.projectTitle": "Anexos: {objective}",
  "home.attachments.remove": "Remover {filename}",
  "home.commands.hint": "Comandos reais: /goal delega, /plan apenas planeja, /test executa testes e /review revisa. Ctrl+Enter para iniciar.",
  "home.commands.reviewObjective": "Revisar o código ou artefato atual, apontar riscos e recomendações acionáveis. Contexto: {objective}",
  "home.commands.testObjective": "Rodar o runner de testes do projeto atual e registrar o resultado. Contexto: {objective}",
  "home.consent.remoteSend": "Entendo que o provider remoto selecionado pode receber meu pedido e trechos dos anexos para gerar a resposta.",
  "home.consent.workspaceWrite": "Permitir alterações nos arquivos do projeto. Se marcar, a tarefa não executará automaticamente: você revisará e iniciará a missão no console.",
  "home.error.attachmentsFailed": "Falha ao enviar ou indexar os anexos.",
  "home.error.projectNotConfirmed": "O servidor não confirmou a criação do projeto dos anexos.",
  "home.error.remoteConsentRequired": "Confirme que entende que o provider remoto selecionado pode receber o pedido e o contexto dos anexos.",
  "home.heading.description": "Um runtime local-first: converse, compile, pesquise, automatize e acompanhe missões com aprovações e dados no seu ambiente.",
  "home.model.auto": "Automático · grátis-primeiro",
  "home.model.choose": "Escolher modelo",
  "home.model.free": "Grátis",
  "home.model.manual": "Manual",
  "home.model.modeTitle": "IA da tarefa",
  "home.model.paid": "Pago",
  "home.objective.aria": "Objetivo da nova tarefa",
  "home.objective.label": "Descreva uma tarefa",
  "home.objective.placeholder": "Atribua uma tarefa ou digite / para mais opções...",
  "home.quickActions.code.label": "Analisar código",
  "home.quickActions.code.prompt": "Inspecionar o repositório, auditar segurança e listar recomendações",
  "home.quickActions.research.label": "Pesquisa profunda",
  "home.quickActions.research.prompt": "Realizar pesquisa aprofundada de mercado com síntese e fontes citadas",
  "home.quickActions.site.label": "Criar site",
  "home.quickActions.site.prompt": "Criar uma landing page moderna responsiva com Tailwind e React",
  "home.quickActions.slides.label": "Criar slides",
  "home.quickActions.slides.prompt": "Criar apresentação profissional de slides sobre inovação em IA",
  "home.quickActions.title": "Ações rápidas:",
  "home.startOptions.aria": "Formas de começar",
  "home.startOptions.attach": "Anexar arquivo",
  "home.startOptions.import": "Importar projeto",
  "home.startOptions.remove": "Remover",
  "home.startOptions.removeSelected": "Remover projeto selecionado",
  "home.startOptions.selected": "Projeto selecionado: {name}",
  "home.startOptions.title": "Começar por:",
  "home.submit.aria": "Iniciar no Console agentic",
  "home.submit.review": "Continuar para revisar",
  "home.submit.sending": "Enviando…",
  "home.submit.start": "Iniciar",

  // Console Codex Desktop (CodexDesktopRow.tsx)
  "codexDesktop.action.addModels": "Adicionar os modelos do Hades ao ChatGPT",
  "codexDesktop.action.addModelsTitle": "Adicionar modelos do Hades",
  "codexDesktop.action.connecting": "Conectando ChatGPT",
  "codexDesktop.action.finishConnection": "Concluir a conexão do ChatGPT",
  "codexDesktop.action.installTitle": "Instalar o ChatGPT e adicionar os modelos do Hades",
  "codexDesktop.action.removeModels": "Remover os modelos do Hades do ChatGPT",
  "codexDesktop.action.removeModelsTitle": "Remover modelos do Hades",
  "codexDesktop.action.retry": "Tentar novamente",
  "codexDesktop.action.retryAria": "Tentar salvar o progresso de novo",
  "codexDesktop.confirm.restartLabel": "Reiniciar",
  "codexDesktop.confirm.restartToAdd": "Reiniciar ChatGPT para adicionar modelos Ollama? Qualquer tarefa em execução será interrompida.",
  "codexDesktop.confirm.restartToRemove": "Reiniciar ChatGPT para remover modelos Ollama? Qualquer tarefa em execução será interrompida.",
  "codexDesktop.description.connectedMany": "Conectado ao Hades · {count} requisições nesta sessão",
  "codexDesktop.description.connectedOne": "Conectado ao Hades · {count} requisição nesta sessão",
  "codexDesktop.description.downloadIntro": "Vamos baixar o ChatGPT e conectá-lo ao Hades.",
  "codexDesktop.description.useCodexModels": "Use os modelos do Hades no modo Codex do ChatGPT.",
  "codexDesktop.error.addModels": "O Hades não conseguiu adicionar seus modelos ao ChatGPT.",
  "codexDesktop.error.completeConnection": "O Hades não conseguiu concluir a conexão com o ChatGPT.",
  "codexDesktop.error.installFailed": "O Hades não conseguiu instalar o ChatGPT.",
  "codexDesktop.error.installNotDetected": "A instalação do ChatGPT não foi detectada. Tente de novo.",
  "codexDesktop.error.integrationUnavailable": "A integração com o ChatGPT está indisponível.",
  "codexDesktop.error.readStatus": "O Hades não conseguiu ler o status da conexão com o ChatGPT.",
  "codexDesktop.error.removeModels": "O Hades não conseguiu remover seus modelos do ChatGPT.",
  "codexDesktop.error.restartToConnect": "O ChatGPT está instalado. Clique em Conectar para reiniciá-lo com os modelos do Hades.",
  "codexDesktop.error.saveProgress": "O Hades não conseguiu salvar seu progresso. Tente de novo.",
  "codexDesktop.notice.modelsAdded": "Modelos do Hades adicionados ao lado dos modelos do Codex",
  "codexDesktop.notice.modelsRemoved": "Modelos do Hades removidos · os modelos do Codex continuam disponíveis",
  "codexDesktop.progress.connecting.description": "Conectando o ChatGPT ao Hades…",
  "codexDesktop.progress.connecting.label": "Conectando…",
  "codexDesktop.progress.disconnecting.description": "Restaurando a conexão normal do ChatGPT…",
  "codexDesktop.progress.disconnecting.label": "Desconectando…",
  "codexDesktop.progress.installing.description": "O Hades está baixando o instalador do ChatGPT…",
  "codexDesktop.progress.installing.label": "Baixando…",
  "codexDesktop.progress.saving.description": "Salvando seu progresso…",
  "codexDesktop.progress.saving.label": "Salvando…",
  "codexDesktop.progress.waitingForInstall.description": "Conclua a instalação do ChatGPT. O Hades vai conectá-lo automaticamente.",
  "codexDesktop.progress.waitingForInstall.label": "Conclua a instalação…",

  // Catalogo de conectores (lib/connectorCatalog.ts)
  // `category` fica de fora: e' identificador de contrato do backend, nao copy.
  "connectorCatalog.gmail.description": "Redija respostas, pesquise a caixa de entrada e resuma conversas por e-mail.",
  "connectorCatalog.outlookMail.description": "Escreva, pesquise e gerencie e-mails do Outlook sem sair do fluxo.",
  "connectorCatalog.slack.description": "Leia canais, envie mensagens e acompanhe conversas de equipe.",
  "connectorCatalog.whatsapp.description": "Envie e receba mensagens em fluxos aprovados de atendimento.",
  "connectorCatalog.discord.description": "Acompanhe servidores e canais e publique atualizações com aprovação.",
  "connectorCatalog.googleWorkspace.description": "Acesse arquivos do Drive, Docs e Sheets e deixe o agente ajudar no contexto.",
  "connectorCatalog.googleCalendar.description": "Entenda a agenda, gerencie eventos e otimize seu tempo.",
  "connectorCatalog.notion.description": "Pesquise o workspace, atualize notas e automatize fluxos de trabalho.",
  "connectorCatalog.microsoft365.description": "Word, Excel, Outlook e Teams em um único conector de produtividade.",
  "connectorCatalog.jira.description": "Consulte issues, sprints e boards e crie tarefas com aprovação.",
  "connectorCatalog.linear.description": "Acompanhe issues e ciclos de engenharia e proponha atualizações.",
  "connectorCatalog.github.description": "Repositórios, issues e pull requests com contexto de código.",
  "connectorCatalog.gitlab.description": "Repositórios, merge requests e pipelines em um só lugar.",
  "connectorCatalog.supabase.description": "Consulte tabelas, funções e logs do seu projeto Postgres gerenciado.",
  "connectorCatalog.instagram.description": "Gere e publique posts, stories e reels com aprovação.",
  "connectorCatalog.metaAds.description": "Insights e otimização de anúncios para economizar horas e maximizar retorno.",
  "connectorCatalog.tiktok.description": "Publique conteúdo e acompanhe métricas de alcance e engajamento.",
  "connectorCatalog.linkedin.description": "Publique atualizações e pesquise conexões e páginas com aprovação.",
  "connectorCatalog.videoEditor.description": "Crie e refine vídeos em uma timeline editável.",
  "connectorCatalog.higgsfield.description": "Gere imagens e vídeos cinematográficos com Higgsfield.",
  "connectorCatalog.canva.description": "Crie e edite designs e exporte assets para suas missões.",
  "connectorCatalog.similarweb.description": "Analise tráfego e dados de SEO de qualquer domínio ou URL.",
  "connectorCatalog.worldBank.description": "Estatísticas oficiais do World Bank para qualquer país, região ou faixa de renda.",
  "connectorCatalog.bigquery.description": "Consulte grandes datasets e cruze com o contexto das missões.",
  "connectorCatalog.stripe.description": "Leia pagamentos, assinaturas e métricas financeiras (somente leitura).",
  "connectorCatalog.googleAnalytics.description": "Métricas de audiência, aquisição e conversão dos seus sites.",

  // AgenticConsole
  "agenticConsole.approval.send":
    "Pode enviar ou publicar conteúdo fora do Hades. Revise destinatário, texto e escopo antes de decidir.",
  "agenticConsole.approval.spend":
    "Pode gerar gasto, compromisso financeiro ou anúncio. Confirme valor, limite e destinatário antes de decidir.",
  "agenticConsole.approval.write":
    "Pode alterar arquivos, publicar ou modificar um serviço externo. Verifique o resumo e o projeto afetado.",
  "agenticConsole.approval.blocked":
    "Esta ação foi bloqueada preventivamente pelo servidor até uma decisão explícita. Revise o contexto antes de decidir.",
  "agenticConsole.provider.ollamaLocal": "Ollama local",
  "agenticConsole.provider.local": "local",
  "agenticConsole.provider.configured": "configurado",
  "agenticConsole.provider.noKey": "sem chave",
  "agenticConsole.error.load": "Não foi possível carregar o runtime agentic.",
  "agenticConsole.error.autoMission": "Falha ao iniciar a missão automática.",
  "agenticConsole.error.createMission": "Falha ao criar a missão.",
  "agenticConsole.error.reasonRequired": "Informe o motivo antes de decidir a aprovação.",
  "agenticConsole.error.decision": "Falha ao registrar a decisão.",
  "agenticConsole.error.run": "Falha ao iniciar a missão.",
  "agenticConsole.error.orchestration": "Falha ao iniciar a orquestração.",
  "agenticConsole.error.research": "Falha na pesquisa profunda.",
  "agenticConsole.aria.console": "Console agentic",
  "agenticConsole.header.eyebrow": "Runtime de Agentes DZ23",
  "agenticConsole.header.title": "Console de Missões",
  "agenticConsole.header.description":
    "Planeje, orquestre, pesquise, aprove, execute e observe operações com a mesma trilha persistente usada pela API local.",
  "agenticConsole.objective.label": "Nova tarefa",
  "agenticConsole.objective.placeholder":
    "Descreva o que você quer construir, pesquisar, revisar ou automatizar",
  "agenticConsole.engine.label": "Motor",
  "agenticConsole.model.label": "Modelo",
  "agenticConsole.project.label": "Projeto",
  "agenticConsole.project.none": "Sem projeto",
  "agenticConsole.isolate.label": "Executar em cópia isolada",
  "agenticConsole.allowWrite.label": "Permitir escrita",
  "agenticConsole.create": "Criar missão",
  "agenticConsole.help.providerLabel": "Motor selecionado:",
  "agenticConsole.help.providerDetail":
    "Providers remotos só aparecem quando o catálogo os publica; a missão falha fechado se não houver adapter/credencial.",
  "agenticConsole.help.write":
    "A missão começa somente com leitura; habilite “Permitir escrita” quando a tarefa precisar alterar arquivos.",
  "agenticConsole.help.isolate":
    "“Executar em cópia isolada” cria um snapshot Git por missão e não altera a origem; exige projeto associado à organização e exclui caminhos/arquivos sensíveis.",
  "agenticConsole.help.isolateAvailable": "A opção só fica disponível com projeto selecionado.",
  "agenticConsole.help.scopes":
    "Escopos elevados (terminal, navegador, desktop, CLIs, MCP, conectores) só aparecem em «Permissões da missão», derivados do catálogo de ferramentas do runtime.",
  "agenticConsole.help.server": "O servidor valida grants e exige approval para efeitos de escrita.",
  "agenticConsole.help.isolateHint":
    "A cópia isolada é opt-in; selecione um projeto. Segredos, arquivos ignorados e symlinks não são copiados.",
  "agenticConsole.metrics.created": "Criadas",
  "agenticConsole.metrics.completed": "Concluídas",
  "agenticConsole.metrics.retries": "Novas tentativas",
  "agenticConsole.metrics.tools": "Ferramentas",
  "agenticConsole.orchestration.eyebrow": "Multiagente",
  "agenticConsole.orchestration.title": "Orquestrar especialistas",
  "agenticConsole.orchestration.placeholder":
    "Ex.: pesquisar concorrentes, revisar segurança e propor implementação",
  "agenticConsole.orchestration.start": "Iniciar orquestração",
  "agenticConsole.orchestration.summary": "Ver síntese",
  "agenticConsole.research.eyebrow": "Pesquisa profunda",
  "agenticConsole.research.title": "Fontes e citações",
  "agenticConsole.research.queryPlaceholder": "Pergunta de pesquisa",
  "agenticConsole.research.urlsPlaceholder": "Uma URL HTTPS pública por linha",
  "agenticConsole.research.submit": "Pesquisar",
  "agenticConsole.view.split": "Visão dividida",
  "agenticConsole.view.classic": "Visão Clássica",
  "agenticConsole.plan.title": "Plano",
  "agenticConsole.plan.step": "Etapa {current} de {total}",
  "agenticConsole.plan.approval": "aprovação",
  "agenticConsole.run": "Executar",
  "agenticConsole.approvals.title": "Aprovações",
  "agenticConsole.approvals.empty": "Nenhuma aprovação pendente.",
  "agenticConsole.approvals.note":
    "Nada será executado enquanto a decisão não for validada pelo servidor. A aprovação não pode ser feita automaticamente pela missão.",
  "agenticConsole.approvals.reasonLabel": "Motivo da decisão",
  "agenticConsole.approvals.reasonAria": "Motivo da decisão para {step}",
  "agenticConsole.approvals.reasonPlaceholder":
    "Explique por que esta ação deve ser aprovada ou rejeitada",
  "agenticConsole.approvals.approve": "Aprovar",
  "agenticConsole.approvals.reject": "Rejeitar",

  // BuildPlanPanel
  "buildPlan.title": "Build & Release",
  "buildPlan.start.accepted": "Build de «{target}» aceite{suffix} — acompanhe em «Estado dos builds».",
  "buildPlan.start.incompletePlan": "O plano de «{target}» está incompleto: {message}",
  "buildPlan.intro.before": "Este painel apenas",
  "buildPlan.intro.detect": "detecta",
  "buildPlan.intro.middle": "o projeto e",
  "buildPlan.intro.probe": "sonda",
  "buildPlan.intro.after":
    "as ferramentas instaladas. Não instala nada, não constrói nada e não assina nada.",
  "buildPlan.repo.label": "Caminho do projeto",
  "buildPlan.repo.placeholder": "ex.: C:\\caminho\\do\\projeto",
  "buildPlan.loading": "A analisar o projeto…",
  "buildPlan.error.analyze": "Não foi possível analisar o projeto:",
  "buildPlan.field.project": "Projeto",
  "buildPlan.field.stacks": "Stacks",
  "buildPlan.stacks.none": "nenhuma reconhecida",
  "buildPlan.field.packageManagers": "Gestores de pacotes",
  "buildPlan.packageManagers.none": "nenhum detetado",
  "buildPlan.field.targets": "Alvos",
  "buildPlan.targets.none": "nenhum",
  "buildPlan.field.tools": "Ferramentas",
  "buildPlan.tools.ready": "{ready} de {total} prontas",
  "buildPlan.tools.missing": ", {count} em falta",
  "buildPlan.noTargets":
    "O projeto não declara nenhum alvo que o HADES conheça. Não há plano de build a construir — isto não é um erro, é um resultado.",
  "buildPlan.blockers.title": "O que falta",
  "buildPlan.reasons.title": "Porquê",
  "buildPlan.independent.title": "O que pode avançar sem desbloquear nada",
  "buildPlan.plan.caption": "Plano por alvo",
  "buildPlan.col.target": "Alvo",
  "buildPlan.col.status": "Estado",
  "buildPlan.col.actions": "Ações",
  "buildPlan.col.tool": "Ferramenta",
  "buildPlan.col.detail": "Detalhe",
  "buildPlan.status.planFailed": "não foi possível planear",
  "buildPlan.status.ready": "pronto",
  "buildPlan.status.blocked": "bloqueado",
  "buildPlan.blockers.none": "nada",
  "buildPlan.start.starting": "A iniciar…",
  "buildPlan.start.action": "Iniciar build",
  "buildPlan.probes.details": "Detalhe das ferramentas sondadas",
  "buildPlan.probes.caption": "Ferramentas por alvo",
  "buildPlan.warnings.title": "Avisos da deteção",
  "buildPlan.artifact.title": "Verificar um artefacto",
  "buildPlan.artifact.intro.before": "Mede o SHA-256 de um ficheiro que",
  "buildPlan.artifact.intro.exists": "já existe",
  "buildPlan.artifact.intro.after":
    "e, se der a referência, compara. Não constrói, não assina e não instala. O caminho tem de ser",
  "buildPlan.artifact.intro.absolute": "absoluto",
  "buildPlan.artifact.path.label": "Caminho do artefacto",
  "buildPlan.artifact.path.placeholder": "ex.: C:\\caminho\\hades.exe",
  "buildPlan.artifact.sha.label": "SHA-256 esperado (opcional para medir)",
  "buildPlan.artifact.sha.placeholder": "64 caracteres hexadecimais",
  "buildPlan.artifact.measure": "Medir",
  "buildPlan.artifact.verify": "Verificar",
  "buildPlan.artifact.measuring": "A medir o artefacto…",
  "buildPlan.artifact.checksumAria": "Checksum medido",
  "buildPlan.artifact.file": "Ficheiro",
  "buildPlan.artifact.size": "Tamanho",
  "buildPlan.artifact.resultAria": "Resultado da verificação",
  "buildPlan.artifact.verdict": "Veredito",
  "buildPlan.artifact.shaMeasured": "SHA-256 medido",
  "buildPlan.artifact.shaExpected": "SHA-256 esperado",
  "buildPlan.artifact.signature": "Assinatura",
  "buildPlan.artifact.measureFailed": "não foi possível medir o artefacto",

  // CompanyGrowthPanel
  "companyGrowth.error.report": "Growth OS indisponível.",
  "companyGrowth.error.action": "A operação de Growth OS falhou.",
  "companyGrowth.title": "Marketing, afiliados e dropshipping controlados",
  "companyGrowth.intro":
    "Tudo começa em sandbox local. Publicação, anúncio, parceiro e fulfillment exigem approval e connector autorizado.",
  "companyGrowth.refresh": "Atualizar growth",
  "companyGrowth.stats.campaigns": "Campanhas",
  "companyGrowth.stats.active": "Ativas",
  "companyGrowth.stats.programs": "Programas",
  "companyGrowth.stats.conversions": "Conversões",
  "companyGrowth.stats.products": "Produtos",
  "companyGrowth.stats.ordersDone": "Pedidos concluídos",
  "companyGrowth.campaign.title": "Campanha",
  "companyGrowth.campaign.name": "Nome da campanha",
  "companyGrowth.campaign.objectivePlaceholder": "Objetivo: gerar leads",
  "companyGrowth.campaign.objective": "Objetivo da campanha",
  "companyGrowth.campaign.channel": "Canal da campanha",
  "companyGrowth.campaign.channelSocial": "Redes sociais",
  "companyGrowth.campaign.channelEmail": "E-mail",
  "companyGrowth.campaign.channelAffiliate": "Afiliados",
  "companyGrowth.campaign.channelContent": "Conteúdo",
  "companyGrowth.campaign.create": "Criar campanha",
  "companyGrowth.campaign.created": "Campanha criada como rascunho; approval pendente.",
  "companyGrowth.campaign.approve": "Aprovar",
  "companyGrowth.campaign.approved": "Campanha aprovada no sandbox.",
  "companyGrowth.campaign.launch": "Iniciar",
  "companyGrowth.campaign.launched": "Campanha iniciada no sandbox.",
  "companyGrowth.program.title": "Programa de afiliados",
  "companyGrowth.program.name": "Nome do programa",
  "companyGrowth.program.networkPlaceholder": "Rede ou parceiro",
  "companyGrowth.program.network": "Rede de afiliados",
  "companyGrowth.program.create": "Criar programa",
  "companyGrowth.program.created": "Programa criado; aprovação pendente.",
  "companyGrowth.program.approve": "Aprovar programa",
  "companyGrowth.program.approved": "Programa de afiliados aprovado no sandbox.",
  "companyGrowth.product.title": "Produto e pedido",
  "companyGrowth.product.name": "Nome do produto",
  "companyGrowth.product.supplierPlaceholder": "Fornecedor sandbox",
  "companyGrowth.product.supplier": "Fornecedor",
  "companyGrowth.product.priceAria": "Preço em centavos",
  "companyGrowth.product.pricePlaceholder": "Preço cents",
  "companyGrowth.product.inventory": "Estoque",
  "companyGrowth.product.add": "Adicionar produto",
  "companyGrowth.product.added": "Produto criado no catálogo sandbox.",
  "companyGrowth.order.customer": "Referência do cliente",
  "companyGrowth.order.quantity": "Quantidade",
  "companyGrowth.order.created": "Pedido criado; aprovação pendente.",
  "companyGrowth.order.create": "Criar pedido",
  "companyGrowth.order.approve": "Aprovar pedido",
  "companyGrowth.order.approved": "Pedido aprovado no sandbox.",
  "companyGrowth.order.fulfill": "Fulfill sandbox",
  "companyGrowth.order.fulfilled": "Pedido fulfillado no sandbox.",

  // ProvidersPage
  "providers.common.cancel": "Cancelar",
  "providers.common.close": "Fechar",
  "providers.common.saving": "Salvando…",
  "providers.models.queryError":
    "Não foi possível consultar os modelos deste provedor.",
  "providers.models.loading": "Consultando modelos do provedor…",
  "providers.models.listError": "Não foi possível listar os modelos: {message}",
  "providers.models.stale":
    "Não existem mais no provedor e foram desmarcados: {models}",
  "providers.models.filterPlaceholder": "Filtrar {count} modelos",
  "providers.models.filterAria": "Filtrar modelos",
  "providers.models.saveError":
    "Não foi possível salvar os modelos selecionados.",
  "providers.models.use": "Usar {count} modelo(s)",
  "providers.card.model.one": "modelo",
  "providers.card.model.other": "modelos",
  "providers.card.keyConfigured": "Chave configurada",
  "providers.card.noKeyNeeded": "Sem chave necessária",
  "providers.card.noKey": "Sem chave",
  "providers.card.replaceKey": "Trocar chave",
  "providers.card.addKey": "Adicionar chave",
  "providers.card.models": "Modelos ({count})",
  "providers.card.removeKey": "Remover chave",
  "providers.card.keyPlaceholder": "Cole a chave de {provider}",
  "providers.card.keyAria": "Chave de API de {provider}",
  "providers.card.save": "Salvar",
  "providers.form.nameRequired": "Dê um nome ao provedor.",
  "providers.form.urlRequired": "Informe o endpoint (base URL) do provedor.",
  "providers.form.modelsRequired": "Informe ao menos um ID de modelo.",
  "providers.form.createError": "Não foi possível cadastrar o provedor.",
  "providers.form.aria": "Adicionar provedor de IA",
  "providers.form.title": "Adicionar provedor",
  "providers.form.intro":
    "Escolha um modelo de serviço compatível ou use \"Personalizado\". O preço e a cota dependem sempre do provedor escolhido; o cadastro aqui não torna o serviço gratuito.",
  "providers.form.service": "Serviço",
  "providers.form.name": "Nome",
  "providers.form.namePlaceholder": "ex.: openai",
  "providers.form.endpoint": "Endpoint (base URL)",
  "providers.form.urlPlaceholder":
    "https://api.exemplo.com/v1 ou http://localhost:1234/v1",
  "providers.form.urlHint":
    "Serviços externos exigem HTTPS. HTTP é aceito apenas para serviços locais (localhost / 127.0.0.1), como LM Studio ou llama.cpp.",
  "providers.form.models": "IDs de modelos",
  "providers.form.modelsPlaceholder":
    "Um por linha ou separados por vírgula. Ex.: gpt-4o, gpt-4o-mini",
  "providers.form.modelsCount": "{count} modelo(s) reconhecido(s).",
  "providers.form.key": "Chave de API (opcional)",
  "providers.form.keyPlaceholder":
    "Cole aqui a chave do provedor, se necessária",
  "providers.form.keyHint":
    "A chave é guardada protegida no seu usuário do sistema e nunca volta para esta tela. Serviços locais geralmente não precisam de chave.",
  "providers.form.submitting": "Cadastrando…",
  "providers.form.submit": "Cadastrar provedor",
  "providers.page.configTitle": "Configurações",
  "providers.page.title": "Provedores de IA",
  "providers.page.intro":
    "Modelos de outros provedores usados pelo roteador, ao lado dos modelos locais e do Ollama Cloud. As chaves ficam protegidas no seu usuário do sistema e nunca voltam para esta tela.",
  "providers.page.add": "Adicionar provedor",
  "providers.page.created":
    "{name}: provedor cadastrado. O runtime aplicará a configuração na próxima inicialização.",
  "providers.page.ready": "{ready} de {total} provedores prontos",
  "providers.page.retry": "Tentar novamente",
  "providers.page.loading": "Carregando provedores…",
  "providers.page.empty.intro":
    "Você ainda não cadastrou nenhum provedor externo. Use o botão \"Adicionar provedor\" acima para cadastrar um serviço compatível (como OpenAI, Groq ou OpenRouter) colando a chave de API — não é preciso editar nenhum arquivo.",
  "providers.page.empty.local":
    "Os modelos locais e o Ollama Cloud (com login) continuam funcionando normalmente, sem precisar de nenhum provedor aqui.",
  "providers.page.changed":
    "{name}: alteração salva. As alterações foram salvas; o runtime aplicará a configuração na próxima inicialização.",

  // WhatsAppGatewayPanel
  "whatsapp.status.connected": "Conectado",
  "whatsapp.status.notConfigured": "Não configurado",
  "whatsapp.status.failed": "Falha",
  "whatsapp.status.connecting": "Conectando",
  "whatsapp.header.title": "WhatsApp Gateway — Controle Remoto do Agente",
  "whatsapp.header.subtitle":
    "Dispare missões, consulte progresso e aprove ações sensíveis pelo celular.",
  "whatsapp.header.connected": "Conectado (PASS)",
  "whatsapp.header.refreshAria": "Atualizar status",
  "whatsapp.backend.active": "ATIVO",
  "whatsapp.backend.status": "Status: {status}",
  "whatsapp.backend.evolution.description":
    "Gateway aberto de alta velocidade para instâncias próprias.",
  "whatsapp.backend.cloud.description":
    "Endpoint oficial Meta Graph API para contas corporativas.",
  "whatsapp.evolution.title": "Conectar Evolution API",
  "whatsapp.evolution.intro":
    "Informe o endereço e a chave do seu servidor Evolution API. As credenciais são guardadas com segurança no servidor local (0600) e nunca são exibidas de volta.",
  "whatsapp.evolution.baseUrl": "URL do servidor",
  "whatsapp.evolution.baseUrlPlaceholder": "https://evolution.seudominio.com",
  "whatsapp.evolution.instance": "Instância",
  "whatsapp.evolution.instancePlaceholder": "nome-da-instancia",
  "whatsapp.evolution.apiKey": "API Key",
  "whatsapp.evolution.apiKeyPlaceholder": "chave da API",
  "whatsapp.evolution.webhookSecret": "Segredo do webhook",
  "whatsapp.evolution.webhookSecretPlaceholder":
    "segredo para validar o webhook",
  "whatsapp.evolution.save": "Salvar e conectar",
  "whatsapp.evolution.saving": "Salvando…",
  "whatsapp.evolution.saved": "Credenciais salvas. Verificando status…",
  "whatsapp.evolution.hint":
    "Dica: configure o webhook do seu Evolution API para apontar para o endpoint do Hades (/api/agent/v1/whatsapp/webhook) usando o mesmo segredo.",
  "whatsapp.bridge.execution": "Execução real no runtime local",
  "whatsapp.bridge.off":
    "Configure um backend acima (Evolution API ou Cloud API da Meta) para ativar o Command Bridge, a aprovação humana e a proteção contra loops (DLQ). Esses recursos ficam disponíveis apenas após a conexão.",
  "whatsapp.card.security.title": "Segurança e aprovação humana",
  "whatsapp.card.security.value": "Aprovação via celular (APROVAR/REJEITAR)",
  "whatsapp.card.security.note":
    "Ações sensíveis bloqueadas até autorização",
  "whatsapp.card.idempotency.title": "Idempotência & DLQ",
  "whatsapp.card.idempotency.value": "Filtro fromMe + Dedupe TTL 24h",
  "whatsapp.card.idempotency.note": "Zero loops de resposta e DLQ ativa",
  "whatsapp.allowlist.title": "Allowlist de Donos e Contatos Autorizados",
  "whatsapp.allowlist.intro":
    "Apenas os números cadastrados abaixo têm permissão para interagir com o agente. Contatos fora da lista são bloqueados imediatamente.",
  "whatsapp.allowlist.empty":
    "Nenhum contato cadastrado na allowlist. Adicione o seu número abaixo para começar.",
  "whatsapp.allowlist.removeAria": "Remover contato",
  "whatsapp.allowlist.phonePlaceholder": "Telefone (ex: 5511999999999)",
  "whatsapp.allowlist.namePlaceholder": "Nome do contato",
  "whatsapp.allowlist.role.owner": "Dono (Owner - Acesso Total)",
  "whatsapp.allowlist.role.operator": "Operador (Executa missões)",
  "whatsapp.allowlist.role.viewer": "Visualizador (Somente leitura)",
  "whatsapp.allowlist.add": "Adicionar",

  // AppSidebar
  "appSidebar.header.workspace": "Espaço de trabalho local",
  "appSidebar.newTask": "Nova tarefa",
  "appSidebar.search": "Pesquisar",
  "appSidebar.section.content": "Conteúdo",
  "appSidebar.nav.library": "Biblioteca",
  "appSidebar.nav.creations": "Criações",
  "appSidebar.nav.capabilities": "Capacidades",
  "appSidebar.nav.signing": "Assinatura",
  "appSidebar.section.agents": "Agentes",
  "appSidebar.nav.computers": "Computadores",
  "appSidebar.nav.agents": "Agentes",
  "appSidebar.badge.new": "Novo",
  "appSidebar.nav.localRuntimeTitle": "Runtime local",
  "appSidebar.nav.automations": "Automações",
  "appSidebar.nav.plugins": "Plugins",
  "appSidebar.section.tools": "Ferramentas",
  "appSidebar.nav.skills": "Habilidades",
  "appSidebar.nav.company": "Empresa",
  "appSidebar.nav.tasks": "Tarefas",
  "appSidebar.nav.compareModels": "Comparar modelos",
  "appSidebar.section.projects": "Projetos",
  "appSidebar.projects.new": "Novo projeto",
  "appSidebar.nav.allProjects": "Todos os projetos",
  "appSidebar.section.system": "Sistema",
  "appSidebar.nav.settings": "Configurações",
  "appSidebar.nav.help": "Ajuda e sobre",
  "appSidebar.profile.personal": "Pessoal",
  "appSidebar.profile.credits": "Créditos:",
  "appSidebar.profile.creditsUnlimited": "Ilimitado (Local)",
  "appSidebar.login.opening": "Abrindo…",
  "appSidebar.login.action": "Entrar com conta Ollama",
  "appSidebar.login.hint":
    "Opcional — só para modelos na nuvem e busca na web. Os modelos locais funcionam sem conta.",
  "appSidebar.profile.account": "Conta",
  "appSidebar.profile.personalization": "Personalização",
  "appSidebar.profile.shortcuts": "Atalhos de teclado",
  "appSidebar.notifications.title": "Notificações",
  "appSidebar.notifications.close": "Fechar notificações",
  "appSidebar.notifications.loading": "Carregando eventos…",
  "appSidebar.notifications.empty":
    "Nenhum evento de missão registrado ainda.",
  "appSidebar.notifications.listAria": "Eventos recentes de missões",
  "appSidebar.notifications.loadFailed":
    "Não foi possível carregar notificações.",
  "appSidebar.notifications.invalidStream":
    "O stream de notificações retornou dados inválidos.",
  "appSidebar.notifications.streamUnavailable":
    "Atualização ao vivo indisponível; os eventos já carregados permanecem visíveis.",
  "appSidebar.shortcuts.title": "Atalhos de Teclado",
  "appSidebar.shortcuts.newChat": "Novo Chat rápido",
  "appSidebar.shortcuts.sendMission": "Enviar e iniciar missão",
  "appSidebar.shortcuts.close": "Fechar",
  "appSidebar.signOut.confirm": "Tem certeza de que deseja sair?",
  "appSidebar.signOut.asEmail": "Sair do Hades como {email}?",
  "appSidebar.signOut.stay": "Manter-se conectado",
  "appSidebar.signOut.pending": "Saindo…",
  "appSidebar.signOut.action": "Sair",
  "appSidebar.signOut.failed": "Não foi possível encerrar a sessão.",
  "appSidebar.nav.aria": "Navegação principal",
  "appSidebar.chatShortcut": "Chat",

  // Rótulos de enums do backend (lib/labels.ts)
  "labels.missionState.pending": "Pendente",
  "labels.missionState.queued": "Na fila",
  "labels.missionState.planning": "Planejando",
  "labels.missionState.running": "Em execução",
  "labels.missionState.observing": "Observando",
  "labels.missionState.recovering": "Recuperando",
  "labels.missionState.awaitingApproval": "Aguardando aprovação",
  "labels.missionState.paused": "Pausada",
  "labels.missionState.blocked": "Bloqueada",
  "labels.missionState.blockedExternal": "Bloqueada (dependência externa)",
  "labels.missionState.completed": "Concluída",
  "labels.missionState.failed": "Falhou",
  "labels.missionState.cancelled": "Cancelada",
  "labels.missionState.notExecuted": "Não executada",
  "labels.missionState.pass": "OK",
  "labels.eventType.missionCreated": "Missão criada",
  "labels.eventType.missionPlanned": "Plano gerado",
  "labels.eventType.missionRun": "Missão iniciada",
  "labels.eventType.missionRunning": "Em execução",
  "labels.eventType.missionCompleted": "Missão concluída",
  "labels.eventType.missionFailed": "Missão falhou",
  "labels.eventType.missionCancelled": "Missão cancelada",
  "labels.eventType.missionRepairAttempt": "Tentando corrigir",
  "labels.eventType.missionRepairSucceeded": "Correção aplicada",
  "labels.eventType.missionRepairFailedTest": "Teste ainda falhando",
  "labels.eventType.missionRepairError": "Erro ao corrigir",
  "labels.eventType.missionRepairExhausted":
    "Tentativas de correção esgotadas",
  "labels.eventType.missionQueueFailed": "Falha ao enfileirar",
  "labels.eventType.stepStarted": "Passo iniciado",
  "labels.eventType.stepSucceeded": "Passo concluído",
  "labels.eventType.stepFailed": "Passo falhou",
  "labels.eventType.stepAwaitingApproval": "Passo aguardando aprovação",
  "labels.eventType.stepRetryScheduled": "Nova tentativa agendada",
  "labels.eventType.browserOperator": "Navegador",
  "labels.eventType.browserFrame": "Quadro do navegador",
  "labels.eventType.browserApproval": "Aprovação do navegador",
  "labels.eventType.gitMergeSucceeded": "Merge concluído",
  "labels.eventType.gitMergeRejected": "Merge recusado",
  "labels.stepKind.workspaceRead": "Ler arquivo",
  "labels.stepKind.workspaceWrite": "Escrever arquivo",
  "labels.stepKind.workspacePatch": "Alterar arquivo",
  "labels.stepKind.workspaceList": "Listar arquivos",
  "labels.stepKind.browserOperator": "Operar navegador",
  "labels.stepKind.projectTestRun": "Rodar testes",

  // Página de conectores
  "connectors.page.title": "Plugins & Conectores",
  "connectors.page.heading": "Plugins",
  "connectors.page.description":
    "Conecte serviços externos, contas OAuth e servidores MCP para expandir as capacidades do agente.",
  "connectors.page.manage": "Gerenciar Conectores ({count})",
  "connectors.page.create": "Criar Conector",
  "connectors.categories.all": "Todos",
  "connectors.categories.productivity": "Produtividade",
  "connectors.categories.creativity": "Criatividade e mídia",
  "connectors.categories.business": "Negócios",
  "connectors.categories.development": "Desenvolvimento",
  "connectors.categories.infra": "Infraestrutura",
  "connectors.categories.finance": "Finanças",
  "connectors.categories.data": "Dados e métricas",
  "connectors.categories.connected": "Conectados",
  "connectors.categories.whatsapp": "WhatsApp Gateway",
  "connectors.search.placeholder": "Buscar conectores e plugins...",
  "connectors.search.aria": "Buscar conectores",
  "connectors.create.title": "Cadastrar Conector / API Personalizada",
  "connectors.create.description":
    "Adicione endpoints REST ou servidores MCP locais/remotos para o agente invocar com segurança.",
  "connectors.create.nameLabel": "Nome do Serviço / Provedor",
  "connectors.create.namePlaceholder": "Ex: Minha API Interna",
  "connectors.create.baseUrlLabel": "URL Base da API",
  "connectors.create.tokenEnvLabel":
    "Variável de Ambiente com a Chave (opcional)",
  "connectors.create.tokenEnvPlaceholder":
    "Ex: MINHA_API_KEY (o segredo nunca é exposto)",
  "connectors.create.pathLabel": "Caminho permitido (prefixo)",
  "connectors.create.pathPlaceholder": "Ex: /v1/clientes",
  "connectors.create.pathHint":
    "O agente só poderá chamar rotas sob este prefixo. Mantenha o menor escopo possível.",
  "connectors.create.allowWriteLabel":
    "Permitir escrita (POST/PUT/PATCH/DELETE) neste caminho.",
  "connectors.create.allowWriteHint":
    "Por padrão o conector é somente leitura (apenas GET). Habilite a escrita apenas se este endereço realmente precisar executar alterações.",
  "connectors.create.cancel": "Cancelar",
  "connectors.create.saving": "Salvando...",
  "connectors.create.submit": "Registrar conector",
  "connectors.create.failedBackend": "Falha ao registrar conector no backend",
  "connectors.create.failed": "Falha ao registrar conector",
  "connectors.list.loading": "Carregando catálogo de plugins...",
  "connectors.list.empty": "Nenhum conector encontrado nesta categoria.",
  "connectors.card.active": "Ativo",
  "connectors.card.oauthTitle": "Requer um app OAuth registrado pela DZ23",
  "connectors.card.oauthRequires": "Requer configuração OAuth",
  "connectors.card.disabled": "Desativado",
  "connectors.card.connect": "Conectar",

  // Painel de BI / status de negócio
  "businessStatus.errors.panel": "Painel de negócio indisponível.",
  "businessStatus.errors.trend": "Série temporal indisponível.",
  "businessStatus.errors.record": "Não foi possível registrar a amostra.",
  "businessStatus.errors.export":
    "Não foi possível gerar a exportação agora. Tente novamente.",
  "businessStatus.record.created": "Amostra registrada agora.",
  "businessStatus.record.existing":
    "Já havia amostra nesta hora: a série usa uma por janela e não duplica o mesmo instante.",
  "businessStatus.indicators.companies": "Empresas",
  "businessStatus.indicators.orders": "Pedidos",
  "businessStatus.indicators.paid": "Pagos",
  "businessStatus.indicators.revenue": "Receita",
  "businessStatus.indicators.financeIncome": "Receita financeira",
  "businessStatus.indicators.financeExpense": "Despesa financeira",
  "businessStatus.indicators.net": "Líquido",
  "businessStatus.indicators.reconciled": "Conciliado",
  "businessStatus.indicators.campaigns": "Campanhas",
  "businessStatus.indicators.scheduled": "Agendadas",
  "businessStatus.indicators.impressions": "Impressões",
  "businessStatus.indicators.clicks": "Cliques",
  "businessStatus.indicators.proposals": "Propostas",
  "businessStatus.indicators.followUpsOverdue": "Follow-ups vencidos",
  "businessStatus.indicators.conversations": "Conversas",
  "businessStatus.indicators.slaViolations": "Violações de SLA",
  "businessStatus.indicators.incompleteChannels": "Canais incompletos",
  "businessStatus.header.eyebrow": "BI e status de negócio",
  "businessStatus.header.title":
    "Indicadores reais, com origem e lacunas declaradas",
  "businessStatus.header.leadBefore":
    "Cada número vem de um módulo interno e diz de onde veio. O que este painel",
  "businessStatus.header.leadStrong": "não mede",
  "businessStatus.header.leadAfter":
    "aparece logo abaixo, com o motivo: zero sem medição não é resposta.",
  "businessStatus.actions.export": "Exportar CSV",
  "businessStatus.actions.refreshing": "Atualizando…",
  "businessStatus.actions.refresh": "Atualizar painel",
  "businessStatus.exportNote.before": "A exportação é CSV com esquema único",
  "businessStatus.exportNote.afterSchema": "e leva",
  "businessStatus.exportNote.gaps": "as lacunas",
  "businessStatus.exportNote.conjunction": "e a",
  "businessStatus.exportNote.origin": "origem de cada número",
  "businessStatus.exportNote.after": ": fora do produto a tela não vai junto.",
  "businessStatus.export.done":
    "{file} gerado: o arquivo inclui o que NÃO é medido e a origem de cada número.",
  "businessStatus.empty.text":
    "Sem dados ainda nesta empresa: os indicadores aparecem quando houver pedido, lançamento, campanha, proposta ou conversa registrados.",
  "businessStatus.trend.title": "Série temporal",
  "businessStatus.trend.record": "Registrar amostra agora",
  "businessStatus.trend.unavailable": "Série indisponível.",
  "businessStatus.trend.samplesOne":
    "{count} amostra · retenção máxima {max}",
  "businessStatus.trend.samplesMany":
    "{count} amostras · retenção máxima {max}",
  "businessStatus.trend.discardedOne": "{count} fora do recorte",
  "businessStatus.trend.discardedMany": "{count} fora do recorte",
  "businessStatus.trend.noBase": "Sem tendência: {reason}",
  "businessStatus.trend.interval": "Intervalo comparado: {interval}",
  "businessStatus.chart.title": "Gráfico da série",
  "businessStatus.chart.metricLabel": "Métrica do gráfico",
  "businessStatus.naoMedido.title": "O que este painel NÃO mede",
  "businessStatus.sources.title": "Origem de cada número",
  "businessStatus.sources.external": "Publicação externa:",

  // Codex Desktop — modelos (CodexDesktopModelsSettings)
  "codexModels.status.cloudOff": "Modelos de nuvem desativados",
  "codexModels.status.signIn": "Login necessário",
  "codexModels.status.upgradePlan": "Requer plano {plan}",
  "codexModels.status.upgradeRequired": "Requer upgrade de plano",
  "codexModels.status.unavailable": "Acesso indisponível",
  "codexModels.status.notInstalled": "Não instalado",
  "codexModels.search.placeholder": "Buscar modelo…",
  "codexModels.search.aria": "Buscar modelo do ChatGPT",
  "codexModels.search.empty": "Nenhum modelo encontrado",
  "codexModels.error.version": "As configurações de modelos do ChatGPT não estão disponíveis nesta versão do Ollama.",
  "codexModels.error.load": "Não foi possível carregar as configurações de modelos do ChatGPT.",
  "codexModels.error.macosOnly": "As configurações de modelos do ChatGPT estão disponíveis apenas no macOS por enquanto.",
  "codexModels.error.selectOne": "Escolha pelo menos um modelo para o ChatGPT.",
  "codexModels.confirm.restartToUpdate": "Reiniciar ChatGPT para atualizar os modelos Ollama? Qualquer tarefa em execução será interrompida.",
  "codexModels.error.openTimeout": "O ChatGPT está demorando mais que o esperado para abrir. Tente novamente.",
  "codexModels.error.apply": "Não foi possível aplicar os modelos do ChatGPT.",
  "codexModels.error.reset": "Não foi possível restaurar os modelos do ChatGPT.",
  "codexModels.pick.upToOne": "Escolha até {count} modelo Ollama para usar no ChatGPT.",
  "codexModels.pick.upToMany": "Escolha até {count} modelos Ollama para usar no ChatGPT.",
  "codexModels.action.saveAndRestart": "Salvar e reiniciar ChatGPT",
  "codexModels.action.restart": "Reiniciar ChatGPT",
  "codexModels.action.saveAndStart": "Salvar e iniciar ChatGPT",
  "codexModels.action.start": "Iniciar ChatGPT",
  "codexModels.action.restarting": "Reiniciando…",
  "codexModels.action.starting": "Iniciando…",
  "codexModels.notice.macosOnly": "Disponível apenas no macOS por enquanto. A configuração de modelos do ChatGPT depende do aplicativo Ollama para macOS.",
  "codexModels.picker.addAria": "Adicionar modelo do ChatGPT",
  "codexModels.picker.chooseAria": "Escolher modelos do ChatGPT",
  "codexModels.picker.removeAria": "Remover {model}",
  "codexModels.picker.loading": "Carregando modelos…",
  "codexModels.picker.select": "Selecionar modelos",

  // Claude Desktop — modelos (ClaudeDesktopModelsSettings)
  "claudeModels.picker.placeholder": "Selecione um modelo",
  "claudeModels.picker.routeAria": "Modelo Ollama para {route}",
  "claudeModels.search.placeholder": "Buscar modelo…",
  "claudeModels.search.aria": "Buscar modelo para {route}",
  "claudeModels.search.empty": "Nenhum modelo encontrado",
  "claudeModels.error.readStatus": "Não foi possível ler o estado de conexão do Claude.",
  "claudeModels.error.loadModels": "Não foi possível carregar seus modelos.",
  "claudeModels.confirm.restartDesktop": "Reiniciar Claude Desktop? Qualquer tarefa em execução será interrompida.",
  "claudeModels.confirm.restartAutoMode": "Reiniciar Claude para alterar o modo automático? Qualquer tarefa em execução será interrompida.",
  "claudeModels.error.macosRouting": "O roteamento de modelos do Claude está disponível apenas no macOS por enquanto.",
  "claudeModels.error.selectOne": "Escolha pelo menos um modelo Ollama para o Claude.",
  "claudeModels.error.invalidMapping": "Escolha modelos disponíveis para sua conta e seu dispositivo.",
  "claudeModels.error.apply": "Não foi possível aplicar os modelos do Claude.",
  "claudeModels.error.autoModeUnavailable": "O modo automático está disponível apenas no macOS por enquanto.",
  "claudeModels.error.autoMode": "Não foi possível atualizar o modo automático do Claude.",
  "claudeModels.error.reset": "Não foi possível restaurar os modelos do Claude.",
  "claudeModels.autoMode.draftHint": "Inicie ou reinicie o Claude para aplicar as alterações antes de mudar o modo automático.",
  "claudeModels.autoMode.description": "Deixe o Claude decidir quando pedir confirmação antes de fazer alterações.",
  "claudeModels.autoMode.selectCloud": "Selecione um modelo na nuvem do Ollama.com para usar o modo automático.",
  "claudeModels.autoMode.selectOne": "Selecione um destes modelos para usar o modo automático: {models}.",
  "claudeModels.autoMode.needCloud": "O modo automático precisa de um modelo na nuvem disponível na sua conta Ollama.com.",
  "claudeModels.autoMode.toggle": "Ativar modo automático",
  "claudeModels.guidance.restart": "Reiniciar Claude interromperá qualquer tarefa em execução.",
  "claudeModels.section.aria": "Configurações do Claude",
  "claudeModels.section.heading": "Aplicativos",
  "claudeModels.description": "Escolha qual modelo Ollama o Claude usa para cada opção.",
  "claudeModels.action.restoring": "Restaurando…",
  "claudeModels.action.restarting": "Reiniciando…",
  "claudeModels.action.starting": "Iniciando…",
  "claudeModels.action.restart": "Reiniciar Claude",
  "claudeModels.action.start": "Iniciar Claude",
  "claudeModels.notice.macosOnly": "Disponível apenas no macOS por enquanto. O roteamento de modelos do Claude depende do aplicativo Ollama para macOS.",
  "claudeModels.list.two": "{first} ou {last}",
  "claudeModels.list.many": "{items} ou {last}",

  // Fluxo de automação (lib/flowGraph)
  "flowGraph.connect.missingNodes": "Nó de origem ou destino não existe.",
  "flowGraph.connect.self": "Um nó não pode se conectar a si mesmo.",
  "flowGraph.connect.triggerTarget": "Um gatilho não pode receber conexões de entrada.",
  "flowGraph.connect.duplicate": "Essa conexão já existe.",
  "flowGraph.connect.cycle": "Essa conexão criaria um ciclo no fluxo.",
  "flowGraph.validate.empty": "O fluxo está vazio: adicione um gatilho para começar.",
  "flowGraph.validate.noTrigger": "O fluxo precisa de pelo menos um gatilho de entrada.",
  "flowGraph.validate.cycle": "O fluxo tem um ciclo: remova a conexão que volta para um nó anterior.",
  "flowGraph.validate.danglingNode": "O nó \"{label}\" está solto (sem conexão de entrada).",
  "flowGraph.steps.empty": "O fluxo está vazio: nada a persistir como grafo de passos.",
  "flowGraph.steps.singleTrigger": "O grafo de passos exige exatamente um gatilho.",
  "flowGraph.steps.unsupportedNode": "O nó \"{label}\" ({type}) ainda não roda como passo persistido; o fluxo segue como plano textual.",
  "flowGraph.steps.tooMany": "O grafo de passos aceita no máximo {max} passos.",
  "flowGraph.steps.cycle": "O fluxo tem um ciclo e não pode ser persistido.",
  "flowGraph.steps.triggerDepends": "O gatilho \"{label}\" não pode depender de outro passo.",
  "flowGraph.steps.actionNoInput": "A ação \"{label}\" precisa de uma conexão de entrada.",
  "flowGraph.steps.missingObjective": "Preencha o objetivo da ação \"{label}\".",
  "flowGraph.describe.missionMissingObjective": "Preencha o objetivo da ação \"Rodar missão\".",
  "flowGraph.describe.httpMissingUrl": "Preencha a URL da ação \"Requisição HTTP\".",
  "flowGraph.describe.connectorMissing": "Escolha o conector da ação \"Chamar conector\".",
  "flowGraph.compile.singleTrigger": "A execução por agendamento aceita exatamente um gatilho por fluxo.",
  "flowGraph.compile.needsMission": "O fluxo precisa de uma ação \"Rodar missão\" para ser executado por agendamento.",
  "flowGraph.compile.webhookSecret": "Informe a variável de ambiente com o segredo do gatilho de webhook.",

  // Comum (rótulo de confirmação partilhado)
  "common.restart": "Reiniciar",
  // Saude do sistema (HealthCenter) + titulo da rota inicial
  "health.title": "Saúde do sistema",
  "health.loading": "Verificando os serviços…",
  "health.error": "Não foi possível consultar o diagnóstico do backend.",
  "health.retry": "Tentar novamente",
  "health.checkedAt": "Diagnóstico recebido do backend em {at}.",
  "health.now": "agora",
  "health.details": "Detalhes",
  "health.detailSuffix": ". A ação corretiva depende da configuração exibida e não é executada automaticamente.",
  "health.status.ok": "Operacional",
  "health.status.attention": "Atenção",
  "health.status.unavailable": "Indisponível",
  "health.subsystem.agent": "Motor de missões",
  "health.subsystem.store": "Persistência",
  "health.subsystem.queue": "Fila de execução",
  "health.subsystem.sandbox": "Isolamento",
  "route.home.title": "Início",
  "backend.offline.message": "O backend local ainda não respondeu. Ele pode estar iniciando — aguarde alguns segundos e tente novamente.",

  // Nova tarefa (NewTaskPage)
  "newTask.title": "Nova tarefa",
  "newTask.back": "Voltar para Tarefas",
  "newTask.eyebrow": "Missão agentic",
  "newTask.description":
    "A tarefa é criada no runtime local com leitura do workspace e sem execução automática. Você acompanha plano, approvals e artifacts em Tarefas.",
  "newTask.objective.label": "Objetivo da tarefa",
  "newTask.objective.placeholder":
    "Ex.: Revisar o README e listar o que falta para rodar o projeto localmente",
  "newTask.objective.hint": "{count}/{max} caracteres · Ctrl+Enter para criar",
  "newTask.validation.empty": "Descreva o objetivo da tarefa antes de criar.",
  "newTask.error.title": "A tarefa não foi criada.",
  "newTask.error.create": "Não foi possível criar a tarefa.",
  "newTask.submit": "Criar tarefa",
  "newTask.submit.pending": "Criando tarefa…",
  "common.cancel": "Cancelar",
};
