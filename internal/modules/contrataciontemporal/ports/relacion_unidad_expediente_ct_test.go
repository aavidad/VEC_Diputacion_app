package ports

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func relacionUnidadExpedienteCTPrueba() RelacionUnidadExpedienteCT {
	return RelacionUnidadExpedienteCT{
		Solicitud: SolicitudRelacionUnidadExpedienteCT{OrganizacionRef: "organizacion:ct:sintetica", ExpedienteRef: "expediente:ct:sintetico", UnidadRefEsperada: "unidad:ct:gestora", VersionExpedienteSolicitada: 7},
		UnidadRef: "unidad:ct:gestora", UnidadSnapshotSolicitadoRef: "unidad:ct:gestora", VersionExpedienteActual: 9, VersionOrigenVinculo: 4,
		OperacionOrigenRef: "reserva:ct:asignacion", ReservaAsignacionRef: "reserva:ct:asignacion", ReciboAsignacionRef: "recibo:ct:asignacion",
		PruebaSnapshotOrigenHuellaSHA256: strings.Repeat("a", 64), TipoEventoOrigen: TipoEventoOrigenRelacionUnidadCT,
		EventoAsignacionRef: "evento:ct:asignacion", EventoPayloadHuellaSHA256: strings.Repeat("b", 64),
		AsignacionConfirmadaEn: time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC),
	}
}

func TestRelacionUnidadCTConservaActualHistoricaYOrigenSeparados(t *testing.T) {
	r := relacionUnidadExpedienteCTPrueba()
	if err := r.ValidarPara(r.Solicitud); err != nil {
		t.Fatal(err)
	}
	if r.VersionExpedienteActual == r.Solicitud.VersionExpedienteSolicitada || r.VersionOrigenVinculo == r.Solicitud.VersionExpedienteSolicitada {
		t.Fatal("el escenario debe conservar tres versiones distintas")
	}
	anterior := r
	anterior.VersionExpedienteActual = anterior.Solicitud.VersionExpedienteSolicitada
	if err := r.ValidarRevalidacion(anterior); err != nil {
		t.Fatalf("avance sin cambio de vínculo: %v", err)
	}
}

func TestRelacionUnidadCTDeniegaAusenciasYSustituciones(t *testing.T) {
	for nombre, mutar := range map[string]func(*RelacionUnidadExpedienteCT){
		"organizacion_ajena":         func(r *RelacionUnidadExpedienteCT) { r.Solicitud.OrganizacionRef = "organizacion:otra" },
		"expediente_ajeno":           func(r *RelacionUnidadExpedienteCT) { r.Solicitud.ExpedienteRef = "expediente:otro" },
		"historica_ajena":            func(r *RelacionUnidadExpedienteCT) { r.Solicitud.VersionExpedienteSolicitada++ },
		"unidad_actor_actual":        func(r *RelacionUnidadExpedienteCT) { r.UnidadRef = "unidad:actor" },
		"unidad_actor_historica":     func(r *RelacionUnidadExpedienteCT) { r.UnidadSnapshotSolicitadoRef = "unidad:actor" },
		"unidad_esperada_vacia":      func(r *RelacionUnidadExpedienteCT) { r.Solicitud.UnidadRefEsperada = "" },
		"origen_ausente":             func(r *RelacionUnidadExpedienteCT) { r.VersionOrigenVinculo = 0 },
		"origen_posterior_historica": func(r *RelacionUnidadExpedienteCT) { r.VersionOrigenVinculo = 8 },
		"historica_futura":           func(r *RelacionUnidadExpedienteCT) { r.VersionExpedienteActual = 6 },
		"version_no_exacta":          func(r *RelacionUnidadExpedienteCT) { r.VersionExpedienteActual = maximaVersionRelacionUnidadCT + 1 },
		"reserva_ausente":            func(r *RelacionUnidadExpedienteCT) { r.ReservaAsignacionRef = "" },
		"operacion_ajena":            func(r *RelacionUnidadExpedienteCT) { r.OperacionOrigenRef = "operacion:ajena" },
		"recibo_ausente":             func(r *RelacionUnidadExpedienteCT) { r.ReciboAsignacionRef = "" },
		"prueba_ausente":             func(r *RelacionUnidadExpedienteCT) { r.PruebaSnapshotOrigenHuellaSHA256 = "" },
		"prueba_cero":                func(r *RelacionUnidadExpedienteCT) { r.PruebaSnapshotOrigenHuellaSHA256 = strings.Repeat("0", 64) },
		"prueba_mayuscula":           func(r *RelacionUnidadExpedienteCT) { r.PruebaSnapshotOrigenHuellaSHA256 = strings.Repeat("A", 64) },
		"evento_otro_tipo":           func(r *RelacionUnidadExpedienteCT) { r.TipoEventoOrigen = "contratacion_temporal.informe_emitido" },
		"evento_ausente":             func(r *RelacionUnidadExpedienteCT) { r.EventoAsignacionRef = "" },
		"payload_ausente":            func(r *RelacionUnidadExpedienteCT) { r.EventoPayloadHuellaSHA256 = "" },
		"fecha_ausente":              func(r *RelacionUnidadExpedienteCT) { r.AsignacionConfirmadaEn = time.Time{} },
		"fecha_no_utc": func(r *RelacionUnidadExpedienteCT) {
			r.AsignacionConfirmadaEn = r.AsignacionConfirmadaEn.In(time.FixedZone("sintetica", 3600))
		},
		"fecha_no_microsegundo": func(r *RelacionUnidadExpedienteCT) {
			r.AsignacionConfirmadaEn = r.AsignacionConfirmadaEn.Add(time.Nanosecond)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			r := relacionUnidadExpedienteCTPrueba()
			q := r.Solicitud
			mutar(&r)
			if err := r.ValidarPara(q); !errors.Is(err, ErrRelacionUnidadExpedienteCTInvalida) {
				t.Fatalf("sustitución aceptada: %v", err)
			}
		})
	}
}

func TestRelacionUnidadCTRevalidaProcedenciaInmutable(t *testing.T) {
	for nombre, mutar := range map[string]func(*RelacionUnidadExpedienteCT){
		"actual_retrocede": func(r *RelacionUnidadExpedienteCT) { r.VersionExpedienteActual = 8 },
		"origen_cambia":    func(r *RelacionUnidadExpedienteCT) { r.VersionOrigenVinculo = 5 },
		"operacion_reserva_cambia": func(r *RelacionUnidadExpedienteCT) {
			r.OperacionOrigenRef = "reserva:otra"
			r.ReservaAsignacionRef = "reserva:otra"
		},
		"recibo_cambia":  func(r *RelacionUnidadExpedienteCT) { r.ReciboAsignacionRef = "recibo:otro" },
		"prueba_cambia":  func(r *RelacionUnidadExpedienteCT) { r.PruebaSnapshotOrigenHuellaSHA256 = strings.Repeat("c", 64) },
		"evento_cambia":  func(r *RelacionUnidadExpedienteCT) { r.EventoAsignacionRef = "evento:otro" },
		"payload_cambia": func(r *RelacionUnidadExpedienteCT) { r.EventoPayloadHuellaSHA256 = strings.Repeat("c", 64) },
		"fecha_cambia": func(r *RelacionUnidadExpedienteCT) {
			r.AsignacionConfirmadaEn = r.AsignacionConfirmadaEn.Add(time.Microsecond)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			esperada := relacionUnidadExpedienteCTPrueba()
			r := esperada
			mutar(&r)
			if err := r.ValidarRevalidacion(esperada); !errors.Is(err, ErrRelacionUnidadExpedienteCTDivergente) {
				t.Fatalf("procedencia distinta aceptada: %v", err)
			}
		})
	}
}
