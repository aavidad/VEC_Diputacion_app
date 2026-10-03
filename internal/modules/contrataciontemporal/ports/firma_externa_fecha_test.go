package ports

import (
	"errors"
	"strings"
	"testing"
)

func TestFechaFirmaExternaCanonica(t *testing.T) {
	for _, fecha := range []string{
		"2026-10-02T10:00:00Z",
		"2026-10-02T10:00:00.123456Z",
	} {
		if _, ok := FechaFirmaExternaCanonica(fecha); !ok {
			t.Fatalf("fecha canonica rechazada: %s", fecha)
		}
	}
	for _, fecha := range []string{
		"2026-13-02T10:00:00Z",
		"2026-10-02T10:00:00+00:00",
		"2026-10-02T10:00:00.1234567Z",
		"2026-10-02T10:00:00.120000Z",
	} {
		if _, ok := FechaFirmaExternaCanonica(fecha); ok {
			t.Fatalf("fecha no canonica admitida: %s", fecha)
		}
	}
	malformada := "2026-13-02T10:00:00Z"
	_, err := fechaFirmaExternaParseada(malformada)
	if !errors.Is(err, errFechaFirmaExternaNoCanonica) || strings.Contains(err.Error(), malformada) {
		t.Fatalf("el error de fecha debe ser un centinela sin contenido de entrada: %v", err)
	}
}
