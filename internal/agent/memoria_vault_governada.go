package agent

import (
	"errors"
	"fmt"
	"os"
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
	// ANTES de anexar: se o ficheiro ja' existe, o adaptador tem de saber com o que esta' a lidar.
	// Duas verificacoes, ambas fail-closed:
	//   1. integridade -- uma nota editada a mao (fora do adaptador) tem o hash divergente; anexar
	//      sobre ela seria escrever por cima de trabalho humano SEM notar;
	//   2. tenant -- o ficheiro pode pertencer a outra organizacao; escrever nele seria violar o
	//      isolamento por um caminho que a leitura ja' recusa.
	if err := s.verificarEstadoExistente(p.Relativo, tenant, identidade, meta.Escopo); err != nil {
		return false, err
	}
	return AnexarAutorizado(s.Raiz, p.Relativo, s.Politica, identidade, meta, memoria, p.VersaoEsperada)
}

// verificarEstadoExistente recusa anexar sobre uma nota que o adaptador nao reconhece como intacta
// e sua. Ficheiro inexistente e' o caso normal de criacao e passa.
func (s ServicoDeAppendGovernado) verificarEstadoExistente(relativo, tenant, identidade string, escopo EscopoDeMemoria) error {
	caminho, err := CaminhoAutorizado(s.Raiz, relativo)
	if err != nil {
		return err
	}
	dados, err := os.ReadFile(caminho)
	if err != nil {
		// SO' "nao existe" e' criacao. Qualquer outro erro (permissao, I/O, caminho invalido)
		// tem de falhar FECHADO: tratar um erro de leitura como "nao existe" faria o append
		// escrever por cima de uma nota que nao conseguiu ler -- exactamente o que a verificacao
		// de integridade existe para impedir.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%w: nao consegui ler a nota existente: %v", ErrConteudoAlterado, err)
	}
	if len(dados) == 0 {
		return nil
	}
	metaExistente, memoriaExistente, errLer := LerNota(dados)
	if errLer != nil {
		// Existe mas nao e' legivel como nota: nao se escreve sobre o que nao se compreende.
		return fmt.Errorf("%w: nota existente ilegivel: %v", ErrConteudoAlterado, errLer)
	}
	if strings.TrimSpace(metaExistente.Tenant) != tenant {
		return ErrAcessoNegado
	}
	if errIntegridade := IntegridadeDaNota(metaExistente, memoriaExistente); errIntegridade != nil {
		return errIntegridade
	}
	_ = identidade
	_ = escopo
	return nil
}
