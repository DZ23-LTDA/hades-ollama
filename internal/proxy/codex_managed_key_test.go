package proxy

import (
	"net/http"
	"testing"
)

// E-484 — **O LIMITE DO E-465, FECHADO: O CAMINHO DA CHAVE GERIDA, COM VALOR.**
//
// **O LIMITE QUE EU MESMO DECLAREI**: *«o proxy **não** tem teste que exercite o caminho da chave gerida com valor»* ✗ —
// *eu corrigi `usesManagedAPIKey` para tempo constante (E-465) e **não** deixei teste a passar por lá com uma chave.*
//
// **E POR QUE ISTO IMPORTA MAIS DO QUE PARECE**: a correcção do E-465 trocou `parts[1] == CodexDesktopManagedAPIKey` por
// um `subtle.ConstantTimeCompare` **com 'vazio nunca confere'** ⇒ *uma alteração no caminho de **autenticação**, sem
// teste, é uma promessa.* ***E a metade que eu acrescentei — «vazio nunca confere» — é exactamente a que pode recusar
// tráfego legítimo se eu tiver errado a condição.***
//
// Este teste mede as **três** direcções: *a chave certa passa; a errada não; e o vazio NÃO passa.*
func TestChaveGeridaDoCodexDesktop(t *testing.T) {
	if CodexDesktopManagedAPIKey == "" {
		t.Skip("chave gerida não definida nesta build — *o caminho não é exercitável aqui, e pular é dizer isso*")
	}

	cabecalho := func(valor string) http.Header {
		h := http.Header{}
		if valor != "" {
			h.Set("Authorization", valor)
		}
		return h
	}

	// (1) A CHAVE CERTA PASSA — *é a direcção que o 'vazio nunca confere' podia ter quebrado.*
	if !usesManagedAPIKey(cabecalho("Bearer " + CodexDesktopManagedAPIKey)) {
		t.Fatal("a chave gerida CORRECTA foi recusada — *a correccao do E-465 quebrou o caminho legitimo*")
	}
	// (2) E é indiferente a caixa do esquema: *o `EqualFold` do `Bearer` continua a valer.*
	if !usesManagedAPIKey(cabecalho("bearer " + CodexDesktopManagedAPIKey)) {
		t.Fatal("`bearer` em minusculas tinha de ser aceite — *o esquema e' comparado com EqualFold*")
	}
	// (3) UMA CHAVE DIFERENTE NÃO PASSA.
	if usesManagedAPIKey(cabecalho("Bearer " + CodexDesktopManagedAPIKey + "-errada")) {
		t.Fatal("uma chave DIFERENTE foi aceite")
	}
	// (4) **O VAZIO NÃO PASSA** — *a metade que eu acrescentei no E-465.*
	if usesManagedAPIKey(cabecalho("Bearer ")) {
		t.Fatal("ARMADILHA: `Bearer` sem chave foi aceite — *'vazio nunca confere'*")
	}
	if usesManagedAPIKey(cabecalho("")) {
		t.Fatal("ARMADILHA: sem cabecalho `Authorization` foi aceite")
	}
	// (5) E o formato tem de ser `Bearer <chave>`: *dois campos, nem um nem tres.*
	if usesManagedAPIKey(cabecalho(CodexDesktopManagedAPIKey)) {
		t.Fatal("a chave SEM o prefixo `Bearer` foi aceite")
	}
	if usesManagedAPIKey(cabecalho("Bearer " + CodexDesktopManagedAPIKey + " extra")) {
		t.Fatal("tres campos foram aceites — *o formato e' `Bearer <chave>`*")
	}
	// (6) E um esquema diferente não passa.
	if usesManagedAPIKey(cabecalho("Basic " + CodexDesktopManagedAPIKey)) {
		t.Fatal("o esquema `Basic` foi aceite")
	}
}
