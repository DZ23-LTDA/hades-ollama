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
};
