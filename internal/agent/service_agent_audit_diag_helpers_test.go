package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Ferramentas de diagnóstico de DLP na trilha (E-193).
//
// Por que existem: `TestServiceAgentAuditKeepsContactTextOut` falhou UMA vez na CI e a mensagem antiga dizia apenas
// "o texto não pode ser gravado" — sem dizer QUAL campo. Uma falha que não nomeia o mecanismo não é investigável
// quando não se consegue reproduzir. Estes dois helpers fazem a falha entregar o campo e um identificador do texto
// **sem revelar** o conteúdo de terceiro no log.

// campoDaTrilhaComTexto devolve qual campo do primeiro registro carrega o texto, no formato `entries[i].campo`.
// Devolve "" quando a trilha está limpa (ou ilegível e sem o texto).
func campoDaTrilhaComTexto(conteudo, texto string) string {
	var arquivo serviceAgentAuditFile
	if err := json.Unmarshal([]byte(conteudo), &arquivo); err != nil {
		// Formato inesperado: não dá para nomear campo, mas se o texto está no arquivo bruto isso precisa aparecer —
		// silenciar aqui esconderia vazamento por mudança de formato.
		if contemTexto(conteudo, texto) {
			return "arquivo (formato não reconhecido)"
		}
		return ""
	}
	for indice, entrada := range arquivo.Entries {
		// TEXTO LIVRE: é onde conteúdo de terceiro poderia entrar, então vale a busca pelo texto completo E pelo
		// fragmento do cartão.
		livres := []struct{ nome, valor string }{
			{"summary", entrada.Summary},
			{"reason", entrada.Reason},
		}
		for _, campo := range livres {
			// E-333: mascara identificadores gerados ANTES da busca por fragmento. Sem isto, um identificador como
			// "apr_28741117-50a" casa com o fragmento "4111" do segredo e acusa vazamento inexistente (a mesma classe
			// de falso positivo que E-196 já havia corrigido para campos que SÃO identificadores).
			if contemTexto(mascaraIdentificadoresGerados(campo.valor), texto) {
				return fmt.Sprintf("entries[%d].%s", indice, campo.nome)
			}
		}
		// IDENTIFICADORES: aqui só o texto COMPLETO conta. Procurar um fragmento de 4 dígitos num UUID acusa por
		// acaso (e o relógio também tem dígitos: `decided_at` com `.411114Z` foi o falso positivo medido em E-196).
		identificadores := []struct{ nome, valor string }{
			{"tool_name", entrada.ToolName},
			{"effect", entrada.Effect},
			{"status", entrada.Status},
			{"agent_id", entrada.AgentID},
			{"conversation_id", entrada.ConversationID},
			{"model_id", entrada.ModelID},
		}
		for _, campo := range identificadores {
			if strings.Contains(campo.valor, texto) {
				return fmt.Sprintf("entries[%d].%s", indice, campo.nome)
			}
		}
	}
	return ""
}

// contemTexto cobre o texto completo E o prefixo de cartão usado nos fixtures: a invariante é "conteúdo de terceiro
// não entra", e o número de cartão é o caso concreto que os testes exercitam. Só vale para campos de TEXTO LIVRE —
// um fragmento de dígitos em identificador ou timestamp acusa por acaso (E-196).
func contemTexto(valor, texto string) bool {
	return strings.Contains(valor, texto) || strings.Contains(valor, "4111")
}

// mascararTexto devolve comprimento + impressão digital, para localizar o campo no log SEM revelar o conteúdo.
func mascararTexto(texto string) string {
	soma := sha256.Sum256([]byte(texto))
	return fmt.Sprintf("%d caracteres, sha256:%x", len(texto), soma[:6])
}

// mascaraIdentificadoresGerados remove identificadores cunhados pelo PRÓPRIO sistema antes de procurar conteúdo de
// terceiro num campo de TEXTO LIVRE.
//
// POR QUE ISSO É CAUSA-RAIZ E NÃO ABRANDAMENTO (E-333): o campo `reason` mistura texto fixo ("escrita exige aprovação
// humana") com um identificador gerado ("apr_28741117-50a"). O segredo da varredura contém "4111", e o identificador
// também — então a busca por FRAGMENTO acusava vazamento onde não havia. O detector já fazia isso para campos que SÃO
// identificadores (lição de E-196); faltava o caso de identificador EMBUTIDO em texto livre.
//
// O que permanece: um vazamento real do conteúdo do contato continua sendo acusado, porque o texto de terceiro não vive
// dentro de um identificador gerado.
func mascaraIdentificadoresGerados(texto string) string {
	if texto == "" {
		return texto
	}
	// Prefixos cunhados pelo sistema (mission, aprovação, conversa, sessão de live, evento, pedido, produto, etc.).
	// E-824: `ct` (contato) FALTAVA. O `summary` do `crm.contact.read` é texto legítimo COM um identificador
	// embutido ("contato Ana Souza (ct_4111e7ed-f7e), telefone +55...") e o `ct_4111e7ed-f7e` casava com o fragmento
	// "4111" do segredo da varredura — a MESMA classe de falso positivo que E-196 (campo identificador) e E-333
	// (identificador embutido em texto livre) já haviam corrigido. Medido na CI da `main` (`4b3e21854`, run
	// 38499987288): era este o motivo do vermelho, não um vazamento.
	comPrefixo := regexp.MustCompile(`\b(?:apr|aud|ct|cv|tr|sp|ms|apv|run|wf|svc|res|brk|clo|live|cev|mod|wl|prod|ped|appt|shift|org|proj|agent|mission)_[0-9a-zA-Z][0-9a-zA-Z_-]*`)
	texto = comPrefixo.ReplaceAllString(texto, "<id>")
	// UUIDs completos (com ou sem hífens) e sequências longas de hexadecimal.
	uuidCompleto := regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
	texto = uuidCompleto.ReplaceAllString(texto, "<id>")
	return texto
}
