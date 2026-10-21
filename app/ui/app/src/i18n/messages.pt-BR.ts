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
};
