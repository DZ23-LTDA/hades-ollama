package agent

import (
	"fmt"
	"strings"
)

// Status canônico de plugin — quarta fatia do estágio 8 (V2.1 §11.3).
//
// "Catalogar não é habilitar": o servidor precisa dizer, para cada plugin, em que
// estado REAL ele está, e a interface só pode oferecer "pronto para usar" quando
// existir execução comprovada. Este resolvedor deriva o estado dos dados que já
// existem (registro, digest no disco, política e trilha de auditoria do runtime),
// sem inventar conectividade:
//
//	UNAUTHORIZED    — o servidor não considera o plugin confiável: não executa;
//	UNHEALTHY       — o conteúdo guardado não corresponde mais ao digest verificado;
//	UNSUPPORTED     — não há como executar (sem runtime/entrypoint declarado,
//	                  runtime não suportado, ou conteúdo nunca verificado);
//	CONFIG_REQUIRED — executável, mas falta configuração: desabilitado ou sem os
//	                  escopos necessários concedidos;
//	TESTED_LIVE     — já executou com sucesso (há registro real na auditoria);
//	INSTALLED       — pronto para configurar, mas nunca executado: NÃO é "pronto".
//
// `CONNECTED` existe na taxonomia para canais/conectores com vínculo vivo. Este
// resolvedor NÃO o produz, porque não há sonda de conexão real implementada —
// declarar CONNECTED aqui seria fingir integração. Fica reservado até existir
// verificação de vínculo.
const (
	PluginStateInstalled      = "INSTALLED"
	PluginStateConnected      = "CONNECTED"
	PluginStateConfigRequired = "CONFIG_REQUIRED"
	PluginStateUnauthorized   = "UNAUTHORIZED"
	PluginStateUnhealthy      = "UNHEALTHY"
	PluginStateUnsupported    = "UNSUPPORTED"
	PluginStateTestedLive     = "TESTED_LIVE"
)

// PluginStatus é o retrato auditável de um plugin instalado.
type PluginStatus struct {
	PluginID        string                 `json:"plugin_id"`
	Version         string                 `json:"version"`
	Kind            string                 `json:"kind"`
	State           string                 `json:"state"`
	Enabled         bool                   `json:"enabled"`
	Trusted         bool                   `json:"trusted"`
	ContentVerified bool                   `json:"content_verified"`
	Capabilities    []string               `json:"capabilities"`
	RequestedScopes []string               `json:"requested_scopes,omitempty"`
	Reasons         []string               `json:"reasons,omitempty"`
	LastExecution   *PluginExecutionRecord `json:"last_execution,omitempty"`
	ReadyToUse      bool                   `json:"ready_to_use"`
}

// StatusForOrganization devolve o status canônico de cada plugin da organização.
func (r *PluginRuntime) StatusForOrganization(organizationID string) ([]PluginStatus, error) {
	if r == nil || r.Registry == nil {
		return nil, fmt.Errorf("plugin runtime is not configured")
	}
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, fmt.Errorf("%w: organization is required", ErrPluginExecutionNotInstalled)
	}
	if err := r.Registry.LoadOrganization(organizationID); err != nil {
		return nil, err
	}
	installations := r.Registry.ListForOrganization(organizationID)
	statuses := make([]PluginStatus, 0, len(installations))
	for _, installation := range installations {
		statuses = append(statuses, r.statusFor(organizationID, installation))
	}
	return statuses, nil
}

func (r *PluginRuntime) statusFor(organizationID string, installation PluginInstallation) PluginStatus {
	status := PluginStatus{
		PluginID:        installation.Manifest.ID,
		Version:         installation.Manifest.Version,
		Kind:            installation.Manifest.Kind,
		Enabled:         installation.Enabled,
		Trusted:         installation.Trusted,
		ContentVerified: installation.ContentVerified,
		Capabilities:    append([]string(nil), installation.GrantedScopes...),
		RequestedScopes: append([]string(nil), installation.RequestedScopes...),
	}
	if record, ok := r.lastExecution(installation.Manifest.ID); ok {
		copied := record
		status.LastExecution = &copied
	}

	switch {
	case !installation.Trusted:
		status.State = PluginStateUnauthorized
		status.Reasons = append(status.Reasons, "assinatura não aceita pelo servidor: promoção a confiável pendente")
	case installation.ContentVerified && strings.TrimSpace(installation.ContentSHA256) != "" &&
		r.Registry.VerifyInstalledContent(organizationID, installation.Manifest.ID) != nil:
		status.State = PluginStateUnhealthy
		status.Reasons = append(status.Reasons, "conteúdo no disco difere do digest verificado: é preciso instalar o pacote de novo antes de executar")
	case !installation.ContentVerified || strings.TrimSpace(installation.ContentSHA256) == "":
		status.State = PluginStateUnsupported
		status.Reasons = append(status.Reasons, "conteúdo do pacote não verificado: execução indisponível")
	default:
		if _, _, err := pluginExecutionLanguage(installation.Manifest.Runtime); err != nil {
			status.State = PluginStateUnsupported
			status.Reasons = append(status.Reasons, "runtime declarado não suportado para execução em sandbox")
			break
		}
		if strings.TrimSpace(installation.Manifest.Entrypoint) == "" {
			status.State = PluginStateUnsupported
			status.Reasons = append(status.Reasons, "manifesto não declara entrypoint")
			break
		}
		if !installation.Enabled {
			status.State = PluginStateConfigRequired
			status.Reasons = append(status.Reasons, "plugin desabilitado para a organização")
			break
		}
		if !r.Policy.Allows(sandboxExecTool{}.Descriptor(), installation.GrantedScopes) {
			status.State = PluginStateConfigRequired
			status.Reasons = append(status.Reasons, fmt.Sprintf("escopos concedidos não cobrem %v", sandboxExecTool{}.Descriptor().Scopes))
			break
		}
		if status.LastExecution != nil && status.LastExecution.Succeeded {
			// Só aqui o plugin pode ser anunciado como pronto: houve execução real.
			status.State = PluginStateTestedLive
			status.ReadyToUse = true
			break
		}
		status.State = PluginStateInstalled
		status.Reasons = append(status.Reasons, "instalado e executável, mas sem execução comprovada: não anunciar como pronto")
	}
	return status
}

func (r *PluginRuntime) lastExecution(pluginID string) (PluginExecutionRecord, bool) {
	trail := r.AuditTrail()
	for i := len(trail) - 1; i >= 0; i-- {
		if trail[i].PluginID == pluginID {
			return trail[i], true
		}
	}
	return PluginExecutionRecord{}, false
}
