package bootstrap

import (
	"bytes"
	"encoding/json"
	"testing"

	"vec-diputacion-granada/config"
)

func TestTokensAvisoOfertaConservaGeneracionYSeparaUso(t *testing.T) {
	actual := tokensAvisoOferta{nuevoDerivadorIdempotenciaPrueba(t, 3, 2)}
	anterior := tokensAvisoOferta{nuevoDerivadorIdempotenciaPrueba(t, 2, 1)}
	previo, version, err := anterior.DerivarTokenAvisoOferta(t.Context(), "oferta:uno", "")
	if err != nil {
		t.Fatal(err)
	}
	recuperado, v, err := actual.DerivarTokenAvisoOferta(t.Context(), "oferta:uno", version)
	if err != nil || v != version || !bytes.Equal(previo, recuperado) {
		t.Fatal("no recuperó la generación persistida")
	}
	otro, _, err := actual.DerivarTokenAvisoOferta(t.Context(), "oferta:otra", version)
	if err != nil || bytes.Equal(previo, otro) {
		t.Fatal("token reutilizable en otra oferta")
	}
	if _, _, err = actual.DerivarTokenAvisoOferta(t.Context(), "oferta:uno", "inexistente"); err == nil {
		t.Fatal("aceptó generación retirada")
	}
}
func TestCatalogoAvisoOfertaYEnlaceOpcional(t *testing.T) {
	var catalogo map[string]any
	if err := json.Unmarshal(config.CatalogoAvisoOfertaBolsa(), &catalogo); err != nil {
		t.Fatal(err)
	}
	for _, enlace := range []string{"", "http://aviso.invalid/area-personal/", "https://usuario:secreto@aviso.invalid/area-personal/", "https://aviso.invalid/otro/"} {
		t.Setenv(config.EnvBolsaAvisoOfertaEnlace, enlace)
		if componerDespachadorAvisoOferta(nil, nil, nil) != nil {
			t.Fatal("configuración inválida habilitó envío")
		}
	}
}
