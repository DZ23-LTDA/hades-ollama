package agent

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// POLÍTICA DE DURABILIDADE DA ESCRITA (E-168).
//
// E-166/E-167 mediram onde o tempo da escrita atômica está: o `fsync` domina (**1,08–1,93 ms** de 2,6–3,6 ms no
// arquivo de 200 memórias). A recomendação registrada então foi *"agrupar ou tornar opcional o `fsync` — decisão
// de produto explícita, não ajuste silencioso"*. É isso que este arquivo implementa:
//
//   - `durable` (PADRÃO): grava, sincroniza o arquivo e o diretório. Sobrevive a queda de energia/processo.
//   - `relaxed`: grava e renomeia (troca atômica continua), SEM sincronizar. Mais rápido, e o **último lote de
//     escritas pode ser perdido** numa queda abrupta — risco DECLARADO, nunca escondido.
//
// O padrão NÃO muda: quem quiser velocidade assume o risco explicitamente por configuração
// (`HADES_WRITE_DURABILITY=relaxed`), e o modo EFETIVO é sempre consultável por `WriteDurability()` — inclusive
// quando um valor inválido cai de volta para o seguro.
const (
	// WriteDurabilityDurable sincroniza arquivo e diretório: sobrevive a queda abrupta.
	WriteDurabilityDurable = "durable"
	// WriteDurabilityRelaxed pula o `fsync`: troca atômica sem garantia contra queda abrupta.
	WriteDurabilityRelaxed = "relaxed"
	// WriteDurabilityBatched sincroniza em LOTE: a troca atômica acontece sempre e o `fsync` é feito no máximo
	// uma vez por janela. É o meio-termo entre perder tudo desde a última sincronização e pagar um `fsync` por
	// escrita — a perda possível fica LIMITADA À JANELA, e não ao volume desde o início do processo.
	WriteDurabilityBatched = "batched"
)

// WriteSyncInterval é a janela máxima entre sincronizações no modo `batched`.
//
// 2 s é curto o bastante para o pior caso ser "perdi o que escrevi nos últimos 2 s" e longo o bastante para
// tirar o `fsync` do caminho de cada escrita — que é exatamente o custo medido em E-166/E-167.
const WriteSyncInterval = 2 * time.Second

// estado do modo `batched`: quando foi a última sincronização e quais arquivos/diretórios estão pendentes.
var (
	writeSyncMu      sync.Mutex
	writeSyncLast    time.Time
	writeSyncPending map[string]struct{}
)

// writeDurability guarda o modo efetivo. 0 = durable, 1 = relaxed (atômico para não custar lock por escrita).
var writeDurability atomic.Int32

// SetWriteDurability aplica o modo e devolve o modo EFETIVO.
//
// Um valor desconhecido **não** vira "relaxed" por engano nem falha a escrita: cai para o padrão seguro e a
// divergência é devolvida — o chamador precisa poder DECLARAR que pediu uma coisa e ficou com outra.
func SetWriteDurability(modo string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(modo)) {
	case "", WriteDurabilityDurable:
		writeDurability.Store(0)
		return WriteDurabilityDurable, nil
	case WriteDurabilityRelaxed:
		writeDurability.Store(1)
		return WriteDurabilityRelaxed, nil
	case WriteDurabilityBatched:
		writeDurability.Store(2)
		// Entra no lote com o relógio ZERADO: a primeira escrita já sincroniza, em vez de deixar a janela
		// inteira descoberta logo depois de trocar de modo.
		writeSyncMu.Lock()
		writeSyncLast = time.Time{}
		if writeSyncPending == nil {
			writeSyncPending = map[string]struct{}{}
		}
		writeSyncMu.Unlock()
		return WriteDurabilityBatched, nil
	default:
		writeDurability.Store(0)
		return WriteDurabilityDurable, fmt.Errorf("modo de durabilidade desconhecido %q: usando %q", modo, WriteDurabilityDurable)
	}
}

// WriteDurability devolve o modo efetivo em uso (seguro por padrão).
func WriteDurability() string {
	switch writeDurability.Load() {
	case 1:
		return WriteDurabilityRelaxed
	case 2:
		return WriteDurabilityBatched
	default:
		return WriteDurabilityDurable
	}
}

// needsImmediateSync diz se ESTA escrita precisa sincronizar agora (modo `batched`) e REGISTRA a pendência quando
// não precisa. O quanto está pendente é observável por PendingDurableWrites — quem grava não precisa carregar
// esse estado de volta.
func needsImmediateSync(caminho string) bool {
	if writeDurability.Load() != 2 {
		return true
	}
	writeSyncMu.Lock()
	defer writeSyncMu.Unlock()
	if writeSyncPending == nil {
		writeSyncPending = map[string]struct{}{}
	}
	if writeSyncLast.IsZero() || time.Since(writeSyncLast) >= WriteSyncInterval {
		// Hora de sincronizar: o lote inteiro (inclusive o que estava pendente) vai junto.
		writeSyncLast = time.Now()
		writeSyncPending = map[string]struct{}{}
		return true
	}
	// Dentro da janela: grava (troca atômica) e marca como pendente. A perda possível fica LIMITADA à janela.
	writeSyncPending[caminho] = struct{}{}
	return false
}

// PendingDurableWrites diz quantas escritas estão gravadas mas ainda não sincronizadas no modo `batched`.
//
// Existe para o operador PODER VER a janela em aberto — sem isso, "batched" seria uma promessa sem medida.
func PendingDurableWrites() int {
	writeSyncMu.Lock()
	defer writeSyncMu.Unlock()
	return len(writeSyncPending)
}

// FlushDurableWrites força a sincronização do que estiver pendente e devolve quantos itens foram sincronizados.
//
// É o que fecha a janela em encerramento ordenado: sem esta chamada, o processo que sai normalmente ainda deixa
// o lote pendente para trás.
func FlushDurableWrites() error {
	writeSyncMu.Lock()
	pendencias := make([]string, 0, len(writeSyncPending))
	for caminho := range writeSyncPending {
		pendencias = append(pendencias, caminho)
	}
	writeSyncPending = map[string]struct{}{}
	writeSyncLast = time.Now()
	writeSyncMu.Unlock()
	var primeiroErro error
	for _, caminho := range pendencias {
		if err := syncFileAndParent(caminho); err != nil && primeiroErro == nil {
			primeiroErro = err
		}
	}
	return primeiroErro
}

// WriteDurabilityFromEnv lê a configuração do ambiente sem aplicar.
func WriteDurabilityFromEnv() string {
	return strings.TrimSpace(os.Getenv("HADES_WRITE_DURABILITY"))
}

// ConfigureWriteDurabilityFromEnv aplica o que estiver no ambiente e devolve o modo EFETIVO mais o aviso, se o
// valor pedido não for reconhecido (nisso o sistema segue no modo seguro).
func ConfigureWriteDurabilityFromEnv() (string, error) {
	return SetWriteDurability(WriteDurabilityFromEnv())
}

// durableWriteAtivo diz se TODA escrita deve sincronizar (arquivo e diretório) — só no modo estritamente seguro.
func durableWriteAtivo() bool {
	return writeDurability.Load() == 0
}

// syncFileAndParent sincroniza um arquivo já gravado e o diretório que o contém (usado pelo modo `batched` e pelo
// flush explícito).
func syncFileAndParent(caminho string) error {
	// Abertura com ESCRITA de propósito: `FlushFileBuffers` no Windows é negado a um handle somente leitura
	// ("Acesso negado") — foi o que o teste do modo em lote pegou. Este caminho é usado pelo FLUSH posterior à
	// escrita, quando o handle original já foi fechado.
	file, err := os.OpenFile(caminho, os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	syncParentDir(caminho)
	return nil
}
