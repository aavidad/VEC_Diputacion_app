package application

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestHuellaConsultaCandidatosRRHHCoincideConCanonPostgreSQL(t *testing.T) {
	q := ports.ConsultaCandidatosRRHHNominal{BolsaRef: "bolsa:ejemplo", Estado: "disponible",
		Texto: "ana", Limite: 50, Modo: ports.ModoBarridoTextoCandidatosRRHH}
	got, ok := HuellaConsultaCandidatosRRHH(q)
	const sql = "ae196e8f558771d02488f07fcb1862256801f7b4d4451fdc2ee16a38f6eff687"
	if !ok || got != sql {
		t.Fatalf("huella distinta de PostgreSQL: %q, valida=%v", got, ok)
	}
	q.CursorToken = "txt.0123456789abcdef0123456789abcdef.50.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	q.SnapshotSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	q.SeleccionRefs = []string{"participacion:ejemplo"}
	q.Modo = ports.ModoPaginaTextoCandidatosRRHH
	if siguiente, valida := HuellaConsultaCandidatosRRHH(q); !valida || siguiente == got {
		t.Fatal("el cursor de otra página reutilizó la huella de la primera")
	}
	q.Texto = "Ana"
	if _, valida := HuellaConsultaCandidatosRRHH(q); valida {
		t.Fatal("admitió un texto sin el canon minúsculo de la ruta")
	}
	q.Modo, q.CursorToken, q.SnapshotSHA256, q.SeleccionRefs = ports.ModoBarridoTextoCandidatosRRHH, "", "", nil
	q.Texto = strings.ToLower(strings.Repeat("Ⱥ", 50)) // 100 bytes antes de ToLower, 150 después.
	if _, valida := HuellaConsultaCandidatosRRHH(q); !valida {
		t.Fatal("perdió una búsqueda admitida por el límite previo a ToLower")
	}
}
