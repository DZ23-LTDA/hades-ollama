package agent

import (
	"errors"
	"strings"
)

var (
	// ErrRaizDoVaultAusente recusa escrever sem raiz configurada: o adaptador nunca escolhe uma
	// raiz por conta propria, e muito menos o vault operacional.
	ErrRaizDoVaultAusente = errors.New("vault root not configured")
	// ErrIdentidadeDeSessaoAusente recusa escrever sem identidade: a ACL nega por omissao, mas um
	// pedido sem identidade nem chega a ser avaliado.
	ErrIdentidadeDeSessaoAusente = errors.New("session identity required")
	// ErrTenantDaSessaoAusente recusa escrever sem organizacao de sessao: o escopo da nota e' do
	// tenant, e um tenant vazio nao e' "todos".
	ErrTenantDaSessaoAusente = errors.New("session tenant required")
)

// ServicoDeAppendGovernado e' a superfic e minima que liga a identidade de sessao a' escrita
// governada no vault. A identidade NAO e' um campo de pedido: entra por parametro, vinda de quem
// autenticou.
type ServicoDeAppendGovernado struct {
	// Raiz e' o diretorio do vault, vindo de configuracao do processo.
	Raiz string
	// Politica nega por omissao: sem concessao explicita, nao ha' escrita.
	Politica PoliticaDeAcesso
}

// PedidoDeAppendGovernado e' o que o chamador pode pedir. Note o que NAO existe aqui: identidade,
// tenant, escopo e permissoes. Nao ha' campo onde os impor.
type PedidoDeAppendGovernado struct {
	Identidade     string
	Tenant         string
	Relativo       string
	VersaoEsperada int
	Memoria        Memoria
}

// Anexar escreve a nota se, e so' se, a politica autorizar `PermissaoEscrever` para a identidade no
// escopo do tenant.
func (s ServicoDeAppendGovernado) Anexar(p PedidoDeAppendGovernado) (bool, error) {
	if strings.TrimSpace(s.Raiz) == "" {
		return false, ErrRaizDoVaultAusente
	}
	identidade := strings.TrimSpace(p.Identidade)
	if identidade == "" {
		return false, ErrIdentidadeDeSessaoAusente
	}
	tenant := strings.TrimSpace(p.Tenant)
	if tenant == "" {
		return false, ErrTenantDaSessaoAusente
	}

	meta := MetaDaNota{
		MemoryID:      "vault:" + strings.TrimSpace(p.Relativo),
		SchemaVersion: VersaoDoSchemaDaNota,
		Escopo:        EscopoTenant,
		Tenant:        tenant,
		Author:        identidade,
		Aprovador:     p.Memoria.Aprovador,
	}
	memoria := p.Memoria
	if strings.TrimSpace(memoria.ID) == "" {
		memoria.ID = meta.MemoryID
	}
	return AnexarAutorizado(s.Raiz, p.Relativo, s.Politica, identidade, meta, memoria, p.VersaoEsperada)
}
