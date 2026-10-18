package postgres

import (
	"strings"
	"testing"
)

func TestDecodificarGobiernoTransicionesExigeFechaDelRecibo(t *testing.T) {
	base := `{"version":2,"transiciones_guardadas":["disponible>pausa"],` +
		`"limites":{"catalogo_version":1,"catalogo_sha256":"` + strings.Repeat("a", 64) + `",` +
		`"transiciones":["disponible>pausa"]},"recibo_ref":"recibo:original"`
	const fecha = "2026-10-16T00:12:34.123456Z"
	valido := base + `,"publicada_en":"` + fecha + `"}`
	v, err := decodificarGobiernoTransicionesSituacion([]byte(valido), true)
	if err != nil || v.PublicadaEn != fecha || !v.Reutilizada {
		t.Fatalf("fecha original rechazada: %+v %v", v, err)
	}
	for _, invalida := range []string{
		base + `}`,
		base + `,"publicada_en":"ayer"}`,
		base + `,"publicada_en":"2026-10-16T02:12:34+02:00"}`,
		base + `,"publicada_en":"0001-01-01T00:00:00Z"}`,
	} {
		if _, err := decodificarGobiernoTransicionesSituacion([]byte(invalida), true); err == nil {
			t.Fatal("recibo sin fecha UTC real admitido")
		}
	}
}
