package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// NÃO INFERÊNCIA NA SUPERFÍCIE DE PLUGINS/CONECTORES/MCP (E-194, fechamento de B-17).
//
// O scanner calibrado (E-191) apontou UM candidato (`mcp_remote.go:606`). Ao VERIFICAR o candidato — em vez de
// corrigi-lo de imediato — a varredura do módulo mostrou que a forma se repete em **seis** métodos `*ForOrganization`
// de três gerenciadores: o caminho "não existe" devolvia `ErrPluginNotFound` e o caminho "é de outro tenant"
// devolvia `ErrPluginOrganizationScope`. São recusas DIFERENTES para a mesma pergunta de um tenant, e a diferença
// revela a existência do ID.
//
// UM CASO QUE **NÃO** É VAZAMENTO, e fica registrado para não ser "corrigido" por engano: `PluginRegistry.getLocked`
// procura em `byOrg[organizationID]`, ou seja, no mapa DA PRÓPRIA organização — "não está no meu mapa" não diz nada
// sobre outros tenants, então `ErrPluginNotFound` ali é correto.
//
// Este teste compara as DUAS recusas e exige que sejam a mesma. Se uma delas voltar a divergir, ele falha.
func recusasIguais(t *testing.T, operacao string, errOutroTenant, errInexistente error) {
	t.Helper()
	if errOutroTenant == nil || errInexistente == nil {
		t.Fatalf("%s: as duas negações precisam existir (outro=%v, inexistente=%v)", operacao, errOutroTenant, errInexistente)
	}
	if errOutroTenant.Error() != errInexistente.Error() {
		t.Fatalf(
			"%s: ORÁCULO DE ENUMERAÇÃO — recusa de outro tenant (%q) difere da recusa de inexistente (%q)",
			operacao, errOutroTenant.Error(), errInexistente.Error())
	}
}

func TestRemoteMCPTenantOperationsDoNotRevealExistence(t *testing.T) {
	gerenciador := NewRemoteMCPManager()
	config := RemoteMCPServerConfig{
		ID:             "srv_da_org_a",
		OrganizationID: "org-a",
		URL:            "https://exemplo.invalid/mcp",
		AllowedMethods: []string{"tools/list"},
	}
	if err := gerenciador.RegisterForOrganization("org-a", config); err != nil {
		t.Fatalf("registrar servidor da org-a: %v", err)
	}

	const intrusa = "org-intrusa"
	const inexistente = "srv_que_nao_existe"

	t.Run("SetEnabledForOrganization", func(t *testing.T) {
		errOutro := gerenciador.SetEnabledForOrganization(intrusa, config.ID, false)
		errInexistente := gerenciador.SetEnabledForOrganization(intrusa, inexistente, false)
		recusasIguais(t, "RemoteMCP.SetEnabledForOrganization", errOutro, errInexistente)
	})

	t.Run("RemoveForOrganization", func(t *testing.T) {
		errOutro := gerenciador.RemoveForOrganization(intrusa, config.ID)
		errInexistente := gerenciador.RemoveForOrganization(intrusa, inexistente)
		recusasIguais(t, "RemoteMCP.RemoveForOrganization", errOutro, errInexistente)
	})

	t.Run("CallForOrganization", func(t *testing.T) {
		_, errOutro := gerenciador.CallForOrganization(context.Background(), intrusa, config.ID, "tools/list", nil)
		_, errInexistente := gerenciador.CallForOrganization(context.Background(), intrusa, inexistente, "tools/list", nil)
		recusasIguais(t, "RemoteMCP.CallForOrganization", errOutro, errInexistente)
	})
}

func TestConnectorTenantOperationsDoNotRevealExistence(t *testing.T) {
	gerenciador := NewConnectorManager()
	config := ConnectorConfig{ID: "con_da_org_a", OrganizationID: "org-a", Provider: "teste", BaseURL: "https://exemplo.invalid"}
	if err := gerenciador.RegisterForOrganization("org-a", config); err != nil {
		t.Fatalf("registrar conector da org-a: %v", err)
	}

	const intrusa = "org-intrusa"
	const inexistente = "con_que_nao_existe"

	t.Run("SetEnabledForOrganization", func(t *testing.T) {
		errOutro := gerenciador.SetEnabledForOrganization(intrusa, config.ID, false)
		errInexistente := gerenciador.SetEnabledForOrganization(intrusa, inexistente, false)
		recusasIguais(t, "Connector.SetEnabledForOrganization", errOutro, errInexistente)
	})

	t.Run("RemoveForOrganization", func(t *testing.T) {
		errOutro := gerenciador.RemoveForOrganization(intrusa, config.ID)
		errInexistente := gerenciador.RemoveForOrganization(intrusa, inexistente)
		recusasIguais(t, "Connector.RemoveForOrganization", errOutro, errInexistente)
	})
}

func TestMCPManagerTenantOperationsDoNotRevealExistence(t *testing.T) {
	gerenciador := NewMCPManager()
	// O comando precisa ser caminho ABSOLUTO de executavel (validação do registro). O binário de teste serve: os
	// ramos sob teste (não existe × outro tenant) retornam ANTES de qualquer spawn.
	comando := os.Args[0]
	if abs, err := filepath.Abs(comando); err == nil {
		comando = abs
	}
	config := MCPServerConfig{ID: "mcp_da_org_a", OrganizationID: "org-a", Command: comando, AllowedMethods: []string{"ping"}}
	if err := gerenciador.RegisterForOrganization("org-a", config); err != nil {
		t.Fatalf("registrar MCP da org-a: %v", err)
	}

	const intrusa = "org-intrusa"
	const inexistente = "mcp_que_nao_existe"

	t.Run("SetEnabledForOrganization", func(t *testing.T) {
		errOutro := gerenciador.SetEnabledForOrganization(intrusa, config.ID, false)
		errInexistente := gerenciador.SetEnabledForOrganization(intrusa, inexistente, false)
		recusasIguais(t, "MCP.SetEnabledForOrganization", errOutro, errInexistente)
	})

	t.Run("RemoveForOrganization", func(t *testing.T) {
		errOutro := gerenciador.RemoveForOrganization(intrusa, config.ID)
		errInexistente := gerenciador.RemoveForOrganization(intrusa, inexistente)
		recusasIguais(t, "MCP.RemoveForOrganization", errOutro, errInexistente)
	})
}

// CONTROLE POSITIVO: o dono continua operando os próprios recursos. Sem isto, "sempre negar" passaria nos testes
// acima e a suíte estaria provando nada.
func TestOwnersStillOperateTheirPlugins(t *testing.T) {
	conectores := NewConnectorManager()
	config := ConnectorConfig{ID: "con_da_org_a", OrganizationID: "org-a", Provider: "teste", BaseURL: "https://exemplo.invalid"}
	if err := conectores.RegisterForOrganization("org-a", config); err != nil {
		t.Fatal(err)
	}
	if err := conectores.SetEnabledForOrganization("org-a", config.ID, false); err != nil {
		t.Fatalf("o dono precisa poder desativar o próprio conector: %v", err)
	}
	if err := conectores.RemoveForOrganization("org-a", config.ID); err != nil {
		t.Fatalf("o dono precisa poder remover o próprio conector: %v", err)
	}

	remotos := NewRemoteMCPManager()
	if err := remotos.RegisterForOrganization("org-a", RemoteMCPServerConfig{ID: "srv_da_org_a", OrganizationID: "org-a", URL: "https://exemplo.invalid/mcp", AllowedMethods: []string{"tools/list"}}); err != nil {
		t.Fatal(err)
	}
	if err := remotos.RemoveForOrganization("org-a", "srv_da_org_a"); err != nil {
		t.Fatalf("o dono precisa poder remover o próprio servidor MCP: %v", err)
	}
}
