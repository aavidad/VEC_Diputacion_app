package domain

import (
	"strings"
	"testing"
	"time"
)

func TestDeclaracionRCDeclaradaPorCentro(t *testing.T) {
	fecha := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	rc := func(numero, documento string) DeclaracionRC {
		return DeclaracionRC{Existe: true, Numero: numero, Fecha: fecha,
			Importe: Importe{Centimos: 123456, Moneda: "EUR"}, DocumentoRef: documento}
	}
	if err := (DeclaracionRC{}).ValidarDeclaradaPorCentro(); err != nil {
		t.Fatalf("sin RC debe admitirse: %v", err)
	}
	if err := rc("220260001234", "rc-declarado:220260001234").ValidarDeclaradaPorCentro(); err != nil {
		t.Fatalf("número de operación válido rechazado: %v", err)
	}
	largo := "A" + strings.Repeat("1", 146)
	if err := rc(largo, PrefijoRCDeclarada+largo).ValidarDeclaradaPorCentro(); err != nil {
		t.Fatalf("147 caracteres deben admitirse: %v", err)
	}
	for nombre, caso := range map[string]DeclaracionRC{
		"dni":                  rc("12345678Z", "rc-declarado:12345678Z"),
		"nie":                  rc("X1234567L", "rc-declarado:X1234567L"),
		"dni con separadores":  rc("1234.5678-Z", "rc-declarado:1234.5678-Z"),
		"etiqueta dni":         rc("dni-2026", "rc-declarado:dni-2026"),
		"documento custodiado": rc("220260001234", "doc:rc:001"),
		"número distinto":      rc("220260001234", "rc-declarado:220260009999"),
		"espacios":             rc("2202 6000", "rc-declarado:2202 6000"),
		"demasiado largo":      rc(largo+"1", PrefijoRCDeclarada+largo+"1"),
	} {
		if err := caso.ValidarDeclaradaPorCentro(); err == nil {
			t.Errorf("%s: debía rechazarse", nombre)
		}
	}
}
