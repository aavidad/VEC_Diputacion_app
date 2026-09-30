package bootstrap

import (
	"errors"
	"strings"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestIncorporacionB2SeleccionNoSustituyeFuentesCT(t *testing.T) {
	a := ct.AntecedentesPlanNominalB2{DocumentoRef: "documento:formalizacion", DocumentoSHA256: strings.Repeat("a", 64)}
	d := ct.DetalleExpedienteRRHH{Solicitud: ct.SolicitudOperativaRRHH{MotivoClave: "sustitucion"}, Analisis: &ct.AnalisisOperativoRRHH{PeriodoInicio: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), PeriodoFin: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)}}
	s := ct.SolicitudPlanNominalB2{DocumentoRef: a.DocumentoRef, DocumentoSHA256: a.DocumentoSHA256, Desde: "2026-09-30", Hasta: "2026-12-01", MotivoClave: "sustitucion"}
	if e := validarSeleccionFuenteCTB2(s, a, d); e != nil {
		t.Fatal(e)
	}
	for _, cambiar := range []func(*ct.SolicitudPlanNominalB2){
		func(s *ct.SolicitudPlanNominalB2) { s.DocumentoRef = "documento:ajeno" },
		func(s *ct.SolicitudPlanNominalB2) { s.DocumentoSHA256 = strings.Repeat("b", 64) },
		func(s *ct.SolicitudPlanNominalB2) { s.Desde = "2026-10-01" },
		func(s *ct.SolicitudPlanNominalB2) { s.Hasta = "" },
		func(s *ct.SolicitudPlanNominalB2) { s.MotivoClave = "otra_causa" },
	} {
		alterada := s
		cambiar(&alterada)
		if e := validarSeleccionFuenteCTB2(alterada, a, d); !errors.Is(e, ct.ErrPlanNominalB2Conflicto) {
			t.Fatalf("selección ajena admitida: %v", e)
		}
	}
	d.Analisis = nil
	if e := validarSeleccionFuenteCTB2(s, a, d); !errors.Is(e, ct.ErrPlanNominalB2Conflicto) {
		t.Fatalf("ausencia de análisis admitida: %v", e)
	}
}

func TestIncorporacionB2TraduccionConservaInstrumentosYProcedencia(t *testing.T) {
	plantilla := "11111111-1111-4111-8111-111111111111"
	rpt := "22222222-2222-4222-8222-222222222222"
	m := domct.PlanIncorporacionPersonalB2{VersionPlazaRef: "plantilla:" + plantilla, VersionPuestoRef: "rpt:" + rpt, FuentePlantilla: domct.InstrumentoPlanPersonalB2{Ref: plantilla, Revision: 2, FuenteRef: "fuente:plantilla", FuenteSHA256: strings.Repeat("a", 64)}, FuenteRPT: domct.InstrumentoPlanPersonalB2{Ref: rpt, Revision: 3, FuenteRef: "fuente:rpt", FuenteSHA256: strings.Repeat("b", 64)}, RevisionPlaza: 4, RevisionPuesto: 5}
	c := ct.ContratoPlanNominalB2{Material: m, PlanRef: "plan:ct", PlanReciboRef: "recibo:plan", PlanVersion: 1, PlanSHA256: strings.Repeat("c", 64), IdempotenciaPersonalUUID: "33333333-3333-4333-8333-333333333333"}
	r := contratoAplicacionB2(c)
	if r.Protocolo != inc.ProtocoloPersonalB2V1 || r.DatosPersonal.VersionPlantillaRef != m.VersionPlazaRef || r.DatosPersonal.VersionRPTRef != m.VersionPuestoRef || r.DatosPersonal.RevisionPlaza != 4 || r.DatosPersonal.RevisionPuesto != 5 || r.FuenteRPT.Version != 3 || r.FuenteRPT.HuellaSHA256 != m.FuenteRPT.FuenteSHA256 {
		t.Fatal("traducción mezcló instrumentos, revisiones o fuente")
	}
	p := r.DatosPersonal.Procedencia
	if p.ActoRef != c.PlanRef || p.FuenteRef != c.PlanRef || p.FuenteVersion != 1 || p.FuenteHuellaSHA256 != c.PlanSHA256 || p.IdempotenciaRef != c.IdempotenciaPersonalUUID {
		t.Fatal("traducción sustituyó la procedencia del plan durable")
	}
}
