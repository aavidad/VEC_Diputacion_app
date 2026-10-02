package seguridad

import (
	"bytes"
	"testing"
)

func TestHuellaAltaLigaNumeroMOADYConservaCanonHistorico(t *testing.T) {
	m := materialHuellaPrueba()
	historico, err := materialCanonicoHuellaAlta(m)
	if err != nil || string(historico) != materialHuellaAltaV1Dorado {
		t.Fatalf("canon histórico cambió: %v", err)
	}
	m.NumeroExpedienteMOAD = "2026/5487"
	a, err := materialCanonicoHuellaAlta(m)
	if err != nil || !bytes.Contains(a, []byte(`"numero_expediente_moad":"2026/5487"`)) {
		t.Fatalf("MOAD no ligado: %v", err)
	}
	m.NumeroExpedienteMOAD = "2026/5488"
	b, err := materialCanonicoHuellaAlta(m)
	if err != nil || bytes.Equal(a, b) {
		t.Fatalf("otro número produce misma huella: %v", err)
	}
	m.NumeroExpedienteMOAD = ""
	m.Solicitud.MotivoClave = "causa_sintetica_distinta"
	otraCausa, err := materialCanonicoHuellaAlta(m)
	if err != nil || bytes.Equal(historico, otraCausa) {
		t.Fatalf("una causa distinta conserva la huella histórica: %v", err)
	}
}
