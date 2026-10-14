package agent

import (
	"errors"
	"fmt"
	"strings"
)

// E-532 (Estagio 47 - Universal Artifact Drive): o CONTRATO do modelo canonico de artefacto.
//
// ***POR QUE UM CONTRATO, E NAO SO' CAMPOS.*** Campos sem regra sao uma estrutura; regra sem contraprova e' uma
// intencao. *O `E-532` acrescentou `TenantID`/`UserID`/`ProjectID`/`AgentID`/`Version`/`RetentionDays`/`ACL`/
// `MIMEVerified` ao `ArtifactManifest` - e este ficheiro e' o que impede que esses campos sejam decorativos.*
//
// ***A REGRA QUE IMPORTA, E POR QUE ELA E' A PRIMEIRA.***
//
// Um artefacto **sem tenant** nao pode existir. Nao e' preferencia de estilo: *«sem tenant» e' exactamente o estado em
// que uma busca atravessa organizacoes sem ninguem notar* - e a FASE 3 da missao proibe ajustes cosmeticos que escondam
// problemas reais. A partir do momento em que este contrato existe, **todo** o artefacto novo tem dono organizacional.
var (
	// ErrArtifactSemTenant e' devolvido quando um artefacto nao declara a organizacao a que pertence.
	ErrArtifactSemTenant = errors.New("artifact must declare a tenant")
	// ErrArtifactACLInvalida e' devolvido quando uma entrada de ACL nao tem a forma `tipo:id`.
	ErrArtifactACLInvalida = errors.New("artifact ACL entry must be `type:id`")
)

// ValidateArtifactScope verifica o contrato do artefacto. *Devolve erro em vez de corrigir em silencio* - um validador
// que "arranja" o que valida nao e' um validador: e' um co-autor.
func ValidateArtifactScope(m ArtifactManifest) error {
	if strings.TrimSpace(m.TenantID) == "" {
		return fmt.Errorf("%w: id=%q mission=%q", ErrArtifactSemTenant, m.ID, m.MissionID)
	}
	if strings.TrimSpace(m.ID) == "" {
		return errors.New("artifact must have an id")
	}
	if strings.TrimSpace(m.SHA256) == "" {
		return errors.New("artifact must declare its sha256")
	}
	// Versao: 0 e' aceite como "nao declarada" para nao quebrar manifestos antigos; **negativa nunca**.
	if m.Version < 0 {
		return fmt.Errorf("artifact version cannot be negative: %d", m.Version)
	}
	if m.RetentionDays < 0 {
		return fmt.Errorf("artifact retention cannot be negative: %d", m.RetentionDays)
	}
	for _, entrada := range m.ACL {
		limpa := strings.TrimSpace(entrada)
		if limpa == "" {
			return fmt.Errorf("%w: entrada vazia", ErrArtifactACLInvalida)
		}
		// `tipo:id`, com as duas partes nao vazias. *Nao valido `tipo` contra uma lista fixa: a lista de tipos e' do
		// dominio (user/group/tenant/service...), e fecha-la aqui obrigaria a mexer neste ficheiro sempre que o dominio
		// crescesse - e um validador que se tem de editar para aceitar o que e' legitimo acaba por ser desligado.*
		partes := strings.SplitN(limpa, ":", 3)
		if len(partes) < 2 {
			return fmt.Errorf("%w: %q", ErrArtifactACLInvalida, entrada)
		}
		if strings.TrimSpace(partes[0]) == "" || strings.TrimSpace(partes[1]) == "" {
			return fmt.Errorf("%w: %q", ErrArtifactACLInvalida, entrada)
		}
	}
	return nil
}

// ArtifactVisivelPara diz se um artefacto pode ser visto por um sujeito, no tenant dado.
//
// ***AS TRES REGRAS, NA ORDEM EM QUE FALHAM FECHADO.***
//
//  1. **Tenant primeiro, e sempre.** *Um sujeito de outro tenant nao ve' nada - nem o proprio artefacto do seu id.*
//  2. **O dono ve' o seu.** *Se `UserID` bate, nao e' preciso ACL.*
//  3. **ACL so' ACRESCENTA.** *ACL vazia significa «ninguem alem do dono», NUNCA «todos»* - o sentido do vazio escolhe-se
//     pelo lado que falha fechado.
//
// *Consequencia deliberada: um artefacto sem tenant nao e' visivel para NINGUEM. E' o comportamento certo - ele tambem
// nao devia existir.*
func ArtifactVisivelPara(m ArtifactManifest, tenantID, sujeito string) bool {
	if strings.TrimSpace(tenantID) == "" || !strings.EqualFold(strings.TrimSpace(m.TenantID), strings.TrimSpace(tenantID)) {
		return false
	}
	sujeito = strings.TrimSpace(sujeito)
	if sujeito == "" {
		return false
	}
	if dono := strings.TrimSpace(m.UserID); dono != "" && strings.EqualFold(dono, sujeito) {
		return true
	}
	for _, entrada := range m.ACL {
		partes := strings.SplitN(strings.TrimSpace(entrada), ":", 2)
		if len(partes) == 2 && strings.EqualFold(strings.TrimSpace(partes[1]), sujeito) {
			return true
		}
	}
	return false
}
