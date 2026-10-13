package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ESTÁGIO 42 — CORREÇÃO ENTRE PROCESSOS: LOCK POR ORGANIZAÇÃO E INVALIDAÇÃO DE CACHE (E-356).
//
// **MEDIDO**: o `LiveCommerceStore` não usava `withFileLock` **nem** invalidação de cache — exatamente o limite que o
// agendamento já tinha declarado e corrigido em E-323. Dois processos (ou duas instâncias do servidor) com o mesmo
// diretório de dados **não se viam**: cada um tinha o seu cache em memória e a sua própria trava ⇒ ***um evento aceito
// numa instância podia ser apagado pela outra, porque ambos leram o mesmo arquivo e cada um gravou a sua versão***.
//
// A CORREÇÃO TEM DUAS PARTES — e a segunda é a que costuma faltar:
//
//  1. **LOCK DE ARQUIVO por organização**, reusando `withFileLock` (o mesmo primitivo do outbox de push e do
//     agendamento; implementação real em Windows e Unix). Nada de lock novo inventado.
//  2. **INVALIDAÇÃO DE CACHE** antes de cada operação travada. Sem ela, a instância A continua respondendo do cache
//     antigo depois de a instância B gravar — *e o resultado é o mesmo defeito, só que mais difícil de ver*.

// ErrLiveStoreCrossProcess recusa operação quando o lock entre processos não pôde ser usado.
var ErrLiveStoreCrossProcess = errors.New("live commerce cross-process lock failed")

// withOrganizationLock executa `run` sob o lock de arquivo DA ORGANIZAÇÃO, invalidando o cache antes.
func (s *LiveCommerceStore) withOrganizationLock(organizationID string, run func() error) error {
	if s == nil || s.root == "" {
		// Store sem raiz (memória): não há arquivo para travar, e o mutex já cobre o mesmo processo.
		return run()
	}
	organizacao := consentScope(organizationID)
	if err := requireOrganization(organizationID); err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return fmt.Errorf("%w: %v", ErrLiveStoreCrossProcess, err)
	}
	caminhoTrava := filepath.Join(s.root, sanitizeIDComponent(organizacao)+".json.lock")
	return withFileLock(caminhoTrava, func() error {
		// ⚠️ VERDADE MEDIDA (ADV-54, 2ª auditoria adversarial independente): estas linhas são **REDUNDANTES**, e o
		// comentário anterior ("a instância precisa RELER o que outra gravou") atribuía a elas um papel que **não é
		// delas**. O que garante a releitura é a **condição** em `carregarLocked` (`live_commerce.go`:
		// `if s.carregado[org] && s.root == ""` ⇒ *com raiz, o arquivo é relido sempre*), e foi **essa** a mudança do
		// E-356 — o `git show` prova: `-if s.carregado[org] {` ⇒ `+if s.carregado[org] && s.root == "" {`.
		//
		// Consequência medida: **2 dos 3 testes do E-356 ficariam VERDES antes da correção**; o único que falha é
		// `TestDuasInstanciasNaoPerdemEventoAceito` (pela via do `ConfirmedSales`, que NÃO passa por este lock).
		//
		// **Mantidas de propósito** (e não removidas): com `root == ""` esta função retorna antes, então hoje o valor
		// nunca decide nada; mas *se `carregarLocked` mudar, a invalidação volta a ser load-bearing*. Removê-las
		// esconderia a intenção; mantê-las **com a verdade escrita** corrige o defeito real, que era a **afirmação**.
		// ***Evidência que sugere um mecanismo diferente do que realmente protege é pior que evidência ausente.***
		s.mu.Lock()
		s.carregado[organizacao] = false
		s.mu.Unlock()
		return run()
	})
}

// NOTA DE SEGURANÇA (E-357): a contraprova deste arquivo foi feita por **edição temporária** — desligar a trava e
// observar o teste FALHAR (`ARMADILHA: perda de escrita entre instâncias`) — e depois REVERTER, como em E-356.
//
// ***A tentação de deixar um interruptor de ambiente é forte e o resultado seria pior do que o defeito original: uma
// variável de ambiente que desliga o lock de segurança.*** Um back door de teste em código de produção não é
// instrumento: é vulnerabilidade com nome bonito.
