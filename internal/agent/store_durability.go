package agent

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
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
	default:
		writeDurability.Store(0)
		return WriteDurabilityDurable, fmt.Errorf("modo de durabilidade desconhecido %q: usando %q", modo, WriteDurabilityDurable)
	}
}

// WriteDurability devolve o modo efetivo em uso (seguro por padrão).
func WriteDurability() string {
	if writeDurability.Load() == 1 {
		return WriteDurabilityRelaxed
	}
	return WriteDurabilityDurable
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

// durableWriteAtivo diz se a escrita deve sincronizar (arquivo e diretório).
func durableWriteAtivo() bool {
	return writeDurability.Load() == 0
}
