package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ComponentPoint é o ponto de entrada de um componente: o ARQUIVO e o SÍMBOLO que precisa aparecer nele.
type ComponentPoint struct {
	File   string
	Symbol string
}

// ComponentSymbolPoints declara, por componente, o arquivo e o SÍMBOLO que identifica o componente DENTRO do arquivo.
//
// Escolha do símbolo: um identificador que o componente introduz — não uma palavra genérica que apareceria em
// qualquer arquivo. Símbolo frouxo ("func", "type") passaria sempre e a verificação não verificaria nada.
func ComponentSymbolPoints() map[string]ComponentPoint {
	return map[string]ComponentPoint{
		"hades_core":                      {File: "service_agents.go", Symbol: "ServiceAgentRuntime"},
		"mission_control":                 {File: "auth.go", Symbol: "CreateOAuthState"},
		"model_gateway":                   {File: "service_agents.go", Symbol: "ServiceAgentTurnRequest"},
		"harness_runtime":                 {File: "service_agents.go", Symbol: "ServiceAgentTurn"},
		"agent_orchestrator":              {File: "runtime.go", Symbol: "ServiceAgents"},
		"tool_execution_runtime":          {File: "service_agent_tools.go", Symbol: "ServiceAgentTool"},
		"mcp_manager":                     {File: "mcp.go", Symbol: "MCPServer"},
		"plugin_manager":                  {File: "plugin_registry.go", Symbol: "PluginInstallation"},
		"skill_manager":                   {File: "skill_activation.go", Symbol: "ActivateSkillForOrganization"},
		"memory_context_engine":           {File: "context.go", Symbol: "ContextStore"},
		"computer_use_runtime":            {File: "browser.go", Symbol: "Playwright"},
		"product_builder":                 {File: "builder.go", Symbol: "BuilderExport"},
		"workflow_automation_engine":      {File: "schedule_flow.go", Symbol: "ExecuteScheduleFlow"},
		"security_policy_engine":          {File: "egress_zero_trust.go", Symbol: "EgressDecision"},
		"artifact_delivery_engine":        {File: "artifacts.go", Symbol: "Artifact"},
		"observability_evaluation_engine": {File: "benchmark_suite.go", Symbol: "RunBenchmarkSuite"},
	}
}

// ContractSymbolFinding é o resultado por componente.
type ContractSymbolFinding struct {
	Component string `json:"component"`
	File      string `json:"file"`
	Symbol    string `json:"symbol"`
	Present   bool   `json:"present"`
	Note      string `json:"note,omitempty"`
}

// ContractSymbolReport é a auditoria por símbolo.
type ContractSymbolReport struct {
	Audited int                     `json:"audited"`
	Present int                     `json:"present"`
	Missing []ContractSymbolFinding `json:"missing"`
}

// AuditContractSymbols confere que o símbolo declarado APARECE no arquivo declarado.
//
// O que NÃO faz (declarado): não prova assinatura, tipo de retorno nem comportamento — só que o componente não
// desapareceu do arquivo para o qual o registro aponta.
func AuditContractSymbols(registry *ContractRegistry) (ContractSymbolReport, error) {
	return auditContractSymbolsAt(registry, ComponentSymbolPoints(), packageDir())
}

// auditContractSymbolsAt é a costura testável: permite apontar símbolo/arquivo controlados para PROVAR que a
// verificação acusa símbolo ausente.
func auditContractSymbolsAt(registry *ContractRegistry, pontos map[string]ComponentPoint, dir string) (ContractSymbolReport, error) {
	if registry == nil {
		return ContractSymbolReport{}, fmt.Errorf("%w: registro ausente", ErrContractInvalid)
	}
	relatorio := ContractSymbolReport{}
	for _, contrato := range registry.Contracts() {
		relatorio.Audited++
		ponto, declarado := pontos[contrato.Component]
		achado := ContractSymbolFinding{Component: contrato.Component, File: ponto.File, Symbol: ponto.Symbol}
		if !declarado || strings.TrimSpace(ponto.Symbol) == "" || strings.TrimSpace(ponto.File) == "" {
			// Sem declaração, a verificação NÃO pode dizer "presente": diria o que não mediu.
			achado.Note = "sem arquivo/simbolo declarado"
			relatorio.Missing = append(relatorio.Missing, achado)
			continue
		}
		conteudo, err := os.ReadFile(filepath.Join(dir, ponto.File))
		if err != nil {
			achado.Note = "arquivo ilegivel: " + err.Error()
			relatorio.Missing = append(relatorio.Missing, achado)
			continue
		}
		if strings.Contains(string(conteudo), ponto.Symbol) {
			achado.Present = true
			relatorio.Present++
		} else {
			// O arquivo existe mas NÃO contém o símbolo: é exatamente o caso que a checagem por arquivo deixava passar.
			achado.Note = "arquivo existe, simbolo ausente"
			relatorio.Missing = append(relatorio.Missing, achado)
		}
	}
	sort.Slice(relatorio.Missing, func(i, j int) bool { return relatorio.Missing[i].Component < relatorio.Missing[j].Component })
	return relatorio, nil
}
