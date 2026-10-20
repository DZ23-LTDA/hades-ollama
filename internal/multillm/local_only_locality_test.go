package multillm

import (
	"errors"
	"testing"
)

// montarRegistryLocal monta um registry minimo com um provedor remoto e um local.
// Testes em pacote interno: usamos os mapas nao exportados de proposito.
func montarRegistryLocal(remotoAllowPrivate bool, localidadeLocal string) *Registry {
	remoto := Provider{
		Name:         "remoto",
		Type:         ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.exemplo.invalid/v1",
		AllowPrivate: remotoAllowPrivate,
		Locality:     "cloud",
	}
	local := Provider{
		Name:     "local",
		Type:     ProviderTypeOpenAICompatible,
		BaseURL:  "http://127.0.0.1:11434/v1",
		Locality: localidadeLocal,
	}
	r := &Registry{
		providers: map[string]Provider{"remoto": remoto, "local": local},
		models: map[string]Model{
			"remoto/modelo": {ID: "remoto/modelo", Provider: "remoto", Available: true},
			"local/modelo":  {ID: "local/modelo", Provider: "local", Available: true},
		},
	}
	return r
}

// O DEFEITO, agora fechado: `LocalOnly` filtrava por `AllowPrivate` (conectividade), nao por
// localidade. Um provedor REMOTO com rede privada permitida era candidato a uma tarefa que o
// utilizador exigiu que nao saisse da maquina.
func TestLocalOnlyNaoEscolheRemotoComAllowPrivate(t *testing.T) {
	// SO' o remoto: e' o cenario que prova o defeito.
	r := &Registry{
		providers: map[string]Provider{"remoto": {
			Name: "remoto", Type: ProviderTypeOpenAICompatible,
			BaseURL: "https://api.exemplo.invalid/v1", AllowPrivate: true, Locality: "cloud",
		}},
		models: map[string]Model{"remoto/modelo": {ID: "remoto/modelo", Provider: "remoto", Available: true}},
	}

	todos, err := r.RouteCandidates(RouteRequest{})
	if err != nil {
		t.Fatalf("sem LocalOnly devia haver candidatos: %v", err)
	}
	if len(todos) == 0 {
		t.Fatal("sem LocalOnly o registry devia ter candidatos")
	}

	// Com LOCAL_ONLY e sem provedor declaradamente local -> ErrNoRoute (recusa honesta).
	_, err = r.RouteCandidates(RouteRequest{LocalOnly: true})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("LOCAL_ONLY com apenas remoto (AllowPrivate=true) devia dar ErrNoRoute, veio %v", err)
	}

	// E `Route` (o caminho usado pelo produto) tambem recusa, com o mesmo erro.
	if _, err := r.Route(RouteRequest{LocalOnly: true}); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("Route com LOCAL_ONLY devia recusar, veio %v", err)
	}
}

// POSITIVO: quem declara `Locality: "local"` e' escolhido sob LOCAL_ONLY.
func TestLocalOnlyEscolheProvedorDeclaradoLocal(t *testing.T) {
	r := montarRegistryLocal(true, "local")
	candidatos, err := r.RouteCandidates(RouteRequest{LocalOnly: true})
	if err != nil {
		t.Fatalf("devia haver candidato local: %v", err)
	}
	if len(candidatos) != 1 {
		t.Fatalf("esperava exactamente 1 candidato, veio %d", len(candidatos))
	}
	if candidatos[0].Model.Provider != "local" {
		t.Fatalf("o escolhido devia ser o local, veio %q", candidatos[0].Model.Provider)
	}
}

// FAIL-CLOSED: localidade desconhecida (vazia) NAO conta como local.
func TestLocalOnlyRecusaDesconhecidaComBaseURLPublico(t *testing.T) {
	r := &Registry{
		providers: map[string]Provider{"p": {Name: "p", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.publico.invalid/v1"}},
		models:    map[string]Model{"p/m": {ID: "p/m", Provider: "p", Available: true}},
	}
	if _, err := r.RouteCandidates(RouteRequest{LocalOnly: true}); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("localidade desconhecida devia ser tratada como nao-local, veio %v", err)
	}
}
