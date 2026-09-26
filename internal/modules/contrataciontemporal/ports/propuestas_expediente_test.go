package ports

import (
	"testing"
	"time"
)

func TestPropuestasExpedienteValidas(t *testing.T) {
	t0 := time.Date(2026, 9, 6, 1, 28, 30, 697897000, time.UTC)
	sustituida := EstadoPropuestaExpediente{Orden: 1, VersionResultante: 7, ConfirmadaEn: t0, ReciboRef: "recibo:a",
		Sustitucion: &SustitucionPropuestaExpediente{NoIncorporacionReciboRef: "recibo:ni", MotivoClave: "no_presentado", RegistradaEn: t0.Add(time.Hour)}}
	vigente := EstadoPropuestaExpediente{Orden: 2, VersionResultante: 9, ConfirmadaEn: t0.Add(2 * time.Hour), ReciboRef: "recibo:b", Vigente: true}
	if !PropuestasExpedienteValidas(nil) || !PropuestasExpedienteValidas([]EstadoPropuestaExpediente{sustituida, vigente}) {
		t.Fatal("historia válida rechazada")
	}
	unica := vigente
	unica.Orden = 1
	if !PropuestasExpedienteValidas([]EstadoPropuestaExpediente{unica}) {
		t.Fatal("propuesta única rechazada")
	}
	for nombre, alterar := range map[string]func(p []EstadoPropuestaExpediente){
		"dos vigentes":      func(p []EstadoPropuestaExpediente) { p[0].Vigente, p[0].Sustitucion = true, nil },
		"última sustituida": func(p []EstadoPropuestaExpediente) { p[1].Vigente = false },
		"vigente con sustitución": func(p []EstadoPropuestaExpediente) {
			p[1].Sustitucion = p[0].Sustitucion
		},
		"orden roto":       func(p []EstadoPropuestaExpediente) { p[1].Orden = 3 },
		"versión repetida": func(p []EstadoPropuestaExpediente) { p[1].VersionResultante = 7 },
		"referencia vacía": func(p []EstadoPropuestaExpediente) { p[1].ReciboRef = "" },
		"motivo inválido": func(p []EstadoPropuestaExpediente) {
			p[0].Sustitucion = &SustitucionPropuestaExpediente{NoIncorporacionReciboRef: "recibo:ni", MotivoClave: "No Presentado", RegistradaEn: t0}
		},
		"instante no UTC": func(p []EstadoPropuestaExpediente) { p[1].ConfirmadaEn = time.Time{} },
	} {
		p := []EstadoPropuestaExpediente{sustituida, vigente}
		alterar(p)
		if PropuestasExpedienteValidas(p) {
			t.Fatalf("%s admitido", nombre)
		}
	}
	muchas := make([]EstadoPropuestaExpediente, MaximoPropuestasExpediente+1)
	if PropuestasExpedienteValidas(muchas) {
		t.Fatal("historia sin límite admitida")
	}
	e := EstadoIncorporacionAcreditada{Propuestas: []EstadoPropuestaExpediente{vigente}}
	if e.Valido() {
		t.Fatal("estado con orden inválido admitido")
	}
}
