package cmd

import "testing"

// Defeito medido em 2026-10-21: `ollama list` panicava com
// "slice bounds out of range [:12] with length 11" porque o codigo fazia m.Digest[:12].
// Qualquer aplicacao que usasse o CLI via crash, embora /api/tags respondesse bem.
func TestShortDigestNaoRebentaComDigestCurto(t *testing.T) {
	casos := []struct {
		nome   string
		digest string
		quer   string
	}{
		{"digest de 11 (o que provocava o panic)", "0123456789a", "0123456789a"},
		{"digest vazio", "", ""},
		{"digest de 1", "a", "a"},
		{"digest exactamente 12", "0123456789ab", "0123456789ab"},
		{"digest de 64 (caso normal)", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "0123456789ab"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			var obtido string
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("PANIC com digest de %d caracteres: %v", len(caso.digest), r)
					}
				}()
				obtido = shortDigest(caso.digest)
			}()
			if obtido != caso.quer {
				t.Fatalf("shortDigest(%q) = %q, queria %q", caso.digest, obtido, caso.quer)
			}
		})
	}
}
