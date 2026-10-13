package agent

import (
	"fmt"
	"time"
)

// CanonicalContractSeeds são os contratos INICIAIS dos 16 componentes canônicos (E-277).
//
// Todos entram como `beta` de propósito: o registro DESCREVE o contrato, ele não audita se o código cumpre a versão
// declarada (limite declarado em E-276). Marcar tudo como `estavel` afirmaria homologação que ninguém mediu.
func CanonicalContractSeeds() []ComponentContract {
	descricao := "contrato declarado; conformidade do código com esta versão é verificada manualmente, não auditada"
	sementes := []struct {
		componente string
		dono       string
		resumo     string
	}{
		{"hades_core", "cto", "núcleo modular que hospeda missões, agentes, modelos, ferramentas e políticas"},
		{"mission_control", "platform", "orquestrador de missões, checkpoints, delegação e estado persistente"},
		{"model_gateway", "ia", "roteamento unificado para modelos locais e APIs externas"},
		{"harness_runtime", "platform", "hospedagem de harnesses especializados por contrato de sessão"},
		{"agent_orchestrator", "platform", "coordenação de agentes e subagentes com cotas e isolamento"},
		{"tool_execution_runtime", "security", "execução governada de ferramentas com validação e auditoria"},
		{"mcp_manager", "platform", "servidores MCP locais e remotos com permissão por ferramenta"},
		{"plugin_manager", "platform", "plugins versionados com integridade, atualização e rollback"},
		{"skill_manager", "platform", "skills com ativação por projeto, agente e modelo"},
		{"memory_context_engine", "dados", "memória governada com escopo, proveniência e isolamento"},
		{"computer_use_runtime", "security", "automação de navegador e desktop sob sandbox e limites"},
		{"product_builder", "produto", "studio de criação de aplicações full-stack reais"},
		{"workflow_automation_engine", "platform", "automação determinística com fila, retry e DLQ"},
		{"security_policy_engine", "security", "autorização, DLP e políticas aplicadas no servidor"},
		{"artifact_delivery_engine", "produto", "artefatos, versionamento, exportação e entrega"},
		{"observability_evaluation_engine", "sre", "métricas, tracing, avaliação e benchmark"},
	}
	contratos := make([]ComponentContract, 0, len(sementes))
	for _, semente := range sementes {
		contratos = append(contratos, ComponentContract{
			Component: semente.componente,
			Version:   "1.0",
			Owner:     semente.dono,
			Status:    ContractStatusBeta,
			Summary:   semente.resumo + " — " + descricao,
		})
	}
	return contratos
}

// SeedCanonicalContracts registra os contratos que FALTAM e devolve quantos foram criados.
//
// IDEMPOTENTE por desenho: componente já registrado NÃO é tocado. Se a semeadura sobrescrevesse, ela apagaria
// versão e histórico de migração já promovidos — exatamente o que o registro existe para proteger.
func SeedCanonicalContracts(registry *ContractRegistry, now time.Time) (int, error) {
	if registry == nil {
		return 0, fmt.Errorf("%w: registro ausente", ErrContractInvalid)
	}
	criados := 0
	for _, contrato := range CanonicalContractSeeds() {
		if _, existe := registry.Contract(contrato.Component); existe {
			continue
		}
		if _, err := registry.Register(contrato, now); err != nil {
			// CORRIDA (outro já registrou): confere o ESTADO, não o TEXTO do erro. Casar mensagem de erro é frágil e
			// esconderia uma semente inválida de verdade, que precisa aparecer.
			if _, agoraExiste := registry.Contract(contrato.Component); agoraExiste {
				continue
			}
			return criados, err
		}
		criados++
	}
	return criados, nil
}

// CanonicalContractsComplete diz se o registro cobre os 16 canônicos e, quando não cobre, QUEM falta.
func CanonicalContractsComplete(registry *ContractRegistry) (bool, []string) {
	if registry == nil {
		return false, CanonicalComponentIDs()
	}
	faltando := registry.MissingCanonicalComponents()
	return len(faltando) == 0, faltando
}
