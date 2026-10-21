package agent

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// ErrNotaNaoApagada recusa declarar sucesso quando o ficheiro ainda existe.
var ErrNotaNaoApagada = errors.New("note was not removed")

// NomeDoLivroDeAuditoria e' o registo append-only das remocoes, dentro da raiz do vault.
// Ele NAO guarda o conteudo apagado (isso contradiria o direito de esquecer): guarda o FACTO,
// quem, quando, em que tenant e qual o caminho relativo.
const NomeDoLivroDeAuditoria = ".hades-auditoria.log"

// RemocaoGovernada apaga uma nota do vault sob a mesma autoridade do resto do adaptador.
type RemocaoGovernada struct {
	Raiz     string
	Politica PoliticaDeAcesso
	// Agora permite fixar o instante nos testes; vazio usa time.Now.
	Agora func() time.Time
}

// PedidoDeRemocao nao tem campos de identidade, tenant ou permissao: como nos restantes pedidos,
// nao ha' onde as impor.
type PedidoDeRemocao struct {
	Identidade string
	Tenant     string
	Relativo   string
}

func (r RemocaoGovernada) agora() time.Time {
	if r.Agora != nil {
		return r.Agora()
	}
	return time.Now().UTC()
}

// Apagar remove a nota se, e so' se: houver raiz, identidade, tenant, permissao de ESCRITA no
// escopo do tenant, o caminho estiver dentro da raiz, a nota existir, for legivel e pertencer ao
// tenant da sessao.
//
// Nota de outro tenant, nota inexistente e nota ilegivel devolvem TODAS `ErrAcessoNegado`, pela
// mesma razao da leitura: um erro diferente revelaria existencia.
func (r RemocaoGovernada) Apagar(p PedidoDeRemocao) (bool, error) {
	if strings.TrimSpace(r.Raiz) == "" {
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
	// Apagar e' escrita: exige o bit de escrita, nao o de leitura.
	if err := r.Politica.Autorizar(identidade, EscopoTenant, PermissaoEscrever); err != nil {
		return false, err
	}

	caminho, err := CaminhoAutorizado(r.Raiz, p.Relativo)
	if err != nil {
		if errors.Is(err, ErrForaDaRaiz) {
			return false, err
		}
		return false, ErrAcessoNegado
	}

	dados, err := os.ReadFile(caminho)
	if err != nil {
		return false, ErrAcessoNegado
	}
	meta, _, err := LerNota(dados)
	if err != nil {
		return false, ErrAcessoNegado
	}
	if strings.TrimSpace(meta.Tenant) != tenant {
		return false, ErrAcessoNegado
	}

	if err := os.Remove(caminho); err != nil {
		return false, fmt.Errorf("%w: %v", ErrNotaNaoApagada, err)
	}
	// Prova de que apagou: o ficheiro tem de deixar de existir.
	if _, err := os.Stat(caminho); err == nil {
		return false, ErrNotaNaoApagada
	}

	if err := r.registarRemocao(identidade, tenant, p.Relativo); err != nil {
		// A nota ja' nao existe; falhar o registo nao a ressuscita. Devolvemos o erro para o
		// chamador saber que a auditoria ficou incompleta -- nao fingimos sucesso completo.
		return true, fmt.Errorf("%w: %v", ErrNotaNaoApagada, err)
	}
	return true, nil
}

// registarRemocao escreve UMA linha por remocao. Nunca inclui o conteudo apagado.
func (r RemocaoGovernada) registarRemocao(identidade, tenant, relativo string) error {
	livro := strings.TrimSpace(r.Raiz) + string(os.PathSeparator) + NomeDoLivroDeAuditoria
	linha := fmt.Sprintf("%s\tremocao\ttenant=%s\tidentidade=%s\trelativo=%s\n",
		r.agora().Format(time.RFC3339), tenant, identidade, relativo)
	f, err := os.OpenFile(livro, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(linha); err != nil {
		return err
	}
	return f.Sync()
}
