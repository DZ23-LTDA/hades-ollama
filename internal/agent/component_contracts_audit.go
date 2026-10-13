package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ComponentEntryPoints mapeia cada componente canônico ao SEU ponto de entrada no código (arquivo dentro deste
// pacote). O mapa é declarado, não inferido: inferir "qual arquivo é o componente" produziria falso positivo.
// ComponentEntryPoints mapeia cada componente canônico ao SEU ponto de entrada no código (arquivo dentro deste
// pacote). O mapa é CONFERIDO contra o disco: nome de arquivo suposto NÃO entra — se o arquivo não existe, o
// componente fica sem ponto declarado e a auditoria o ACUSA, em vez de o mapa afirmar o que ninguém verificou.
func ComponentEntryPoints() map[string]string {
	candidatos := map[string][]string{
		"hades_core":             {"service_agents.go"},
		"mission_control":        {"auth.go"},
		"model_gateway":          {"service_agents.go"},
		"harness_runtime":        {"service_agents.go"},
		"agent_orchestrator":     {"runtime.go", "swarm.go"},
		"tool_execution_runtime": {"service_agent_tools.go"},
		"mcp_manager":            {"mcp.go"},
		"plugin_manager":         {"plugin_registry.go", "connectors.go"},
		"skill_manager":          {"skill_activation.go", "capability_policy.go"},
		"memory_context_engine":  {"context.go"},
		"computer_use_runtime":   {"browser.go", "browser_environment.go"},
		"product_builder":        {"builder.go"},
		// Fluxo de automação: só aceita arquivo cujo NOME declare workflow/automação. Catalogar automação
		// (connector_catalog.go) NÃO é o motor — apontar para o catálogo seria afirmar o que não foi medido.
		"workflow_automation_engine":      {"workflow.go", "automation.go", "workflow_engine.go"},
		"security_policy_engine":          {"egress_zero_trust.go"},
		"artifact_delivery_engine":        {"artifacts.go"},
		"observability_evaluation_engine": {"benchmark_suite.go"},
	}
	dir := packageDir()
	pontos := map[string]string{}
	for componente, opcoes := range candidatos {
		for _, opcao := range opcoes {
			if _, err := os.Stat(filepath.Join(dir, opcao)); err == nil {
				pontos[componente] = opcao
				break
			}
		}
	}
	return pontos
}

// ContractPresence é o resultado da auditoria para UM componente.
type ContractPresence struct {
	Component  string `json:"component"`
	EntryPoint string `json:"entry_point"`
	Present    bool   `json:"present"`
}

// ContractPresenceReport é a auditoria completa.
type ContractPresenceReport struct {
	Audited  int                `json:"audited"`
	Present  int                `json:"present"`
	Missing  []ContractPresence `json:"missing"`
	Findings []ContractPresence `json:"findings"`
}

// packageDir devolve o diretório DESTE pacote por runtime.Caller, sem depender do diretório de trabalho dos testes.
func packageDir() string {
	_, arquivo, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Dir(arquivo)
}

// AuditContractPresence confere se o ponto de entrada de cada contrato registrado EXISTE no pacote.
//
// O que esta auditoria NÃO faz (declarado para não inflar evidência): ela não prova compatibilidade semântica entre
// código e versão declarada; prova apenas que o componente não desapareceu por baixo do contrato.
func AuditContractPresence(registry *ContractRegistry) (ContractPresenceReport, error) {
	return auditContractPresenceAt(registry, ComponentEntryPoints(), packageDir())
}

// auditContractPresenceAt e' a costura testavel: permite apontar para um diretorio/mapa controlados sem depender do
// disco real, para PROVAR que a auditoria ACUSA ausencia (uma auditoria que so sabe confirmar nao audita nada).
func auditContractPresenceAt(registry *ContractRegistry, pontos map[string]string, dir string) (ContractPresenceReport, error) {
	if registry == nil {
		return ContractPresenceReport{}, fmt.Errorf("%w: registro ausente", ErrContractInvalid)
	}
	relatorio := ContractPresenceReport{}
	contratos := registry.Contracts()
	for _, contrato := range contratos {
		relatorio.Audited++
		ponto, declarado := pontos[contrato.Component]
		presenca := ContractPresence{Component: contrato.Component, EntryPoint: ponto}
		if !declarado || strings.TrimSpace(ponto) == "" {
			// Sem ponto de entrada declarado, a auditoria NÃO pode dizer "presente": diria o que não mediu.
			presenca.Present = false
			relatorio.Missing = append(relatorio.Missing, presenca)
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, ponto)); err == nil {
			presenca.Present = true
			relatorio.Present++
		} else {
			relatorio.Missing = append(relatorio.Missing, presenca)
		}
		relatorio.Findings = append(relatorio.Findings, presenca)
	}
	sort.Slice(relatorio.Missing, func(i, j int) bool { return relatorio.Missing[i].Component < relatorio.Missing[j].Component })
	sort.Slice(relatorio.Findings, func(i, j int) bool { return relatorio.Findings[i].Component < relatorio.Findings[j].Component })
	return relatorio, nil
}
