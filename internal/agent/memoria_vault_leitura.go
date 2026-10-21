package agent

import (
	"errors"
	"os"
	"strings"
)

// ErrTenantDivergente recusa entregar uma nota cujo tenant nao e' o da sessao. E' um erro PRÓPRIO,
// mas as chamadas publicas devolvem a MESMA negacao de `ErrAcessoNegado` para nao revelar que a
// nota existe. Ver `LeituraGovernada.Ler`.
var ErrTenantDivergente = errors.New("note belongs to another tenant")

// LeituraGovernada le uma nota do vault sob a mesma autoridade do append: identidade de sessao,
// permissao de LEITURA separada e escopo do tenant.
type LeituraGovernada struct {
	Raiz     string
	Politica PoliticaDeAcesso
}

// PedidoDeLeitura e' o que o chamador pode pedir. Como no append, NAO ha' campo para identidade,
// tenant ou permissao.
type PedidoDeLeitura struct {
	Identidade string
	Tenant     string
	Relativo   string
}

// Ler devolve a memoria se, e so' se: houver raiz, identidade, tenant, permissao de LEITURA no
// escopo do tenant, o caminho estiver dentro da raiz, a nota pertencer ao tenant da sessao e o
// conteudo bater com o hash gravado.
//
// ISOLAMENTO NEGATIVO, E O CUIDADO QUE ISTO EXIGE: nota de outro tenant, nota inexistente e nota
// ilegivel devolvem TODAS `ErrAcessoNegado`. Distinguir "existe mas nao e' tua" de "nao existe"
// entrega a existencia por omissao -- *um erro mais explicito e' uma fuga de informacao mais
// explicita.*
func (l LeituraGovernada) Ler(p PedidoDeLeitura) (Memoria, error) {
	if strings.TrimSpace(l.Raiz) == "" {
		return Memoria{}, ErrRaizDoVaultAusente
	}
	identidade := strings.TrimSpace(p.Identidade)
	if identidade == "" {
		return Memoria{}, ErrIdentidadeDeSessaoAusente
	}
	tenant := strings.TrimSpace(p.Tenant)
	if tenant == "" {
		return Memoria{}, ErrTenantDaSessaoAusente
	}
	if err := l.Politica.Autorizar(identidade, EscopoTenant, PermissaoLer); err != nil {
		return Memoria{}, err
	}

	caminho, err := CaminhoAutorizado(l.Raiz, p.Relativo)
	if err != nil {
		if errors.Is(err, ErrForaDaRaiz) {
			return Memoria{}, err
		}
		return Memoria{}, ErrAcessoNegado
	}

	dados, err := os.ReadFile(caminho)
	if err != nil {
		// Nao distinguir "nao existe" de "nao posso ler": ambos sao negacao.
		return Memoria{}, ErrAcessoNegado
	}

	meta, memoria, err := LerNota(dados)
	if err != nil {
		// Nota ilegivel tambem nao pode ser um oraculo de existencia.
		return Memoria{}, ErrAcessoNegado
	}
	if strings.TrimSpace(meta.Tenant) != tenant {
		return Memoria{}, ErrAcessoNegado
	}
	if err := IntegridadeDaNota(meta, memoria); err != nil {
		return Memoria{}, err
	}
	// O hash ja' define o conteudo canonico SEM a terminacao de linha (`HashDoConteudo` apara `\n`).
	// Se a leitura devolvesse o corpo com a terminacao, escrita -> leitura deixaria de ser identica e
	// quem compara strings veria uma diferenca que nao existe no conteudo. Mesma licao de E-1437,
	// agora do lado da leitura.
	memoria.Conteudo = strings.TrimRight(memoria.Conteudo, "\n")
	return memoria, nil
}
