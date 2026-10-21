package multillm

import "testing"

// O DEFEITO, do lado do PROXY: `Resolve` recusava TUDO com `policy.LocalOnly`, mesmo um modelo
// LOCAL pedido pelo nome. `RouteCandidates` filtrava por localidade e `Resolve` negava sem olhar
// ao modelo -- duas superficies de roteamento com semantica oposta, e a jornada `LOCAL_ONLY` era
// impossivel de servir pelo proxy mesmo com Ollama local saudavel.
//
// Agora a regra e' a mesma nos dois sitios: exige provedor local, nao proibe o local.
func TestResolveComLocalOnlyAceitaProvedorLocal(t *testing.T) {
	r := montarRegistryLocal(true, "local")
	modelo, ok := r.Resolve("local/modelo", Policy{LocalOnly: true})
	if !ok {
		t.Fatal("LOCAL_ONLY devia servir um modelo LOCAL declarado")
	}
	if modelo.Provider != "local" {
		t.Fatalf("o resolvido devia ser o local, veio %q", modelo.Provider)
	}
}

func TestResolveComLocalOnlyRecusaProvedorRemotoMesmoComAllowPrivate(t *testing.T) {
	r := montarRegistryLocal(true, "local")
	if _, ok := r.Resolve("remoto/modelo", Policy{LocalOnly: true}); ok {
		t.Fatal("LOCAL_ONLY nao pode resolver um modelo de provedor remoto")
	}
}

// Sem a restricao, o remoto continua a resolver normalmente: a correcao nao parte nada.
func TestResolveSemLocalOnlyAceitaRemoto(t *testing.T) {
	r := montarRegistryLocal(true, "local")
	if _, ok := r.Resolve("remoto/modelo", Policy{}); !ok {
		t.Fatal("sem LOCAL_ONLY o modelo remoto devia resolver")
	}
}

// Compatibilidade: provedor sem `Locality` mas com BaseURL de loopback continua a ser local.
func TestResolveComLocalOnlyAceitaLoopbackSemLocalityDeclarada(t *testing.T) {
	r := &Registry{
		providers: map[string]Provider{"ollama": {
			Name: "ollama", Type: ProviderTypeOpenAICompatible, BaseURL: "http://127.0.0.1:11434/v1",
		}},
		models: map[string]Model{"ollama/llama": {ID: "ollama/llama", Provider: "ollama", Available: true}},
	}
	if _, ok := r.Resolve("ollama/llama", Policy{LocalOnly: true}); !ok {
		t.Fatal("um provider em loopback sem Locality devia contar como local (compatibilidade)")
	}
}
