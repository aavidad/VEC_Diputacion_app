package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type fuentePlanCoberturaFichaPrueba struct {
	plan     domain.PlanCompetenciaFirmaV2
	err      error
	llamadas int
}

func TestCoberturaFirmaFichaConservaHTTP404SinConsultarPlan(t *testing.T) {
	fuente := &fuentePlanCoberturaFichaPrueba{plan: planCoberturaFichaPrueba()}
	lector := &lectorDisponibilidadFichaPrueba{err: application.ErrConsultaRRHHNoObservable}
	h, err := httpinterno.NuevoManejadorConsultaDetalleRRHH(consultorDetalleConCoberturaFirmaNominalRRHH{
		lector: lector, plan: fuente, fijado: fuente.plan.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, httpinterno.RutaConsultaDetalleRRHH,
		bytes.NewBufferString(`{"expediente_ref":"expediente:prueba","version_observada":7}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound || lector.llamadas != 1 || fuente.llamadas != 0 ||
		strings.Contains(w.Body.String(), "firma_nominal") {
		t.Fatalf("HTTP=%d, lectura=%d, plan=%d, cuerpo=%s", w.Code, lector.llamadas, fuente.llamadas, w.Body.String())
	}
}

func (f *fuentePlanCoberturaFichaPrueba) Plan(context.Context) (domain.PlanCompetenciaFirmaV2, error) {
	f.llamadas++
	return f.plan, f.err
}

func planCoberturaFichaPrueba() domain.PlanCompetenciaFirmaV2 {
	v := domain.VersionPlanFirmaV2{Referencia: "plan:firma:prueba", Version: 2, HuellaSHA256: strings.Repeat("a", 64)}
	paso := domain.CompetenciaPasoFirmaV2{
		EntradaClave: "entrada:resolucion", Circuito: v, Documento: "resolucion", PasoRef: "paso:primero", PasoOrden: 1,
		PerfilEsperadoRef: "perfil:prueba", RolID: "rol:prueba", CargoRef: "cargo:prueba",
		OrganizacionRef: "organizacion:rrhh:prueba", UnidadRef: "unidad:rrhh:prueba",
		Accion: "documento.firmar", Finalidad: "formalizar", TipoRecurso: "documento",
		EsquemaContexto: "contexto.v1", MapeoVersion: 1, MapeoFuenteRef: "fuente:prueba",
	}
	duplicado := paso
	duplicado.EntradaClave, duplicado.PasoRef, duplicado.PerfilEsperadoRef = "entrada:resolucion:dos", "paso:segundo", "perfil:segundo"
	otro := paso
	otro.EntradaClave, otro.Documento, otro.PasoRef = "entrada:contrato", "contrato", "paso:tercero"
	ajeno := paso
	ajeno.EntradaClave, ajeno.Documento, ajeno.UnidadRef = "entrada:ajena", "documento_ajeno", "unidad:otra"
	return domain.PlanCompetenciaFirmaV2{Version: v, FuenteRef: "fuente:prueba", Pasos: []domain.CompetenciaPasoFirmaV2{paso, duplicado, otro, ajeno}}
}

func detalleCoberturaFichaPrueba() ports.DetalleExpedienteRRHH {
	return ports.DetalleExpedienteRRHH{
		Resumen:    ports.ResumenExpedienteRRHH{ExpedienteRef: "expediente:prueba", Version: 7, OrganizacionRef: "organizacion:rrhh:prueba"},
		Asignacion: &ports.AsignacionOperativaRRHH{UnidadRef: "unidad:rrhh:prueba"},
	}
}

func solicitudCoberturaFichaPrueba(t *testing.T) ports.SolicitudDetalleRRHH {
	t.Helper()
	solicitud, err := ports.NuevaSolicitudDetalleRRHH("expediente:prueba", 7)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func TestCoberturaFirmaFichaSoloTrasLecturaAutorizadaYUnidadReal(t *testing.T) {
	plan := planCoberturaFichaPrueba()
	fuente := &fuentePlanCoberturaFichaPrueba{plan: plan}
	lector := &lectorDisponibilidadFichaPrueba{detalle: detalleCoberturaFichaPrueba()}
	c := consultorDetalleConCoberturaFirmaNominalRRHH{lector: lector, plan: fuente, fijado: plan.Version, documentalMontado: true}
	solicitud := solicitudCoberturaFichaPrueba(t)
	detalle, err := c.Consultar(context.Background(), solicitud)
	if err != nil || lector.llamadas != 1 || fuente.llamadas != 1 ||
		detalle.CoberturaFirmaNominal.Estado != ports.CoberturaFirmaNominalDisponible ||
		detalle.EstadoBorradoresPublicados != ports.BorradoresRRHHMontado ||
		detalle.CoberturaFirmaNominal.UnidadRef != lector.detalle.Asignacion.UnidadRef ||
		detalle.CoberturaFirmaNominal.Plan != plan.Version ||
		len(detalle.CoberturaFirmaNominal.Documentos) != 2 ||
		detalle.CoberturaFirmaNominal.Documentos[0] != "contrato" ||
		detalle.CoberturaFirmaNominal.Documentos[1] != "resolucion" {
		t.Fatalf("cobertura: %+v, lectura=%d plan=%d error=%v", detalle.CoberturaFirmaNominal, lector.llamadas, fuente.llamadas, err)
	}

	lector.err = ports.ErrAutorizacionDenegada
	_, err = c.Consultar(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrAutorizacionDenegada) || fuente.llamadas != 1 {
		t.Fatalf("lectura denegada consultó plan: llamadas=%d, error=%v", fuente.llamadas, err)
	}
}

func TestCoberturaFirmaFichaNoDisponibleSinMontajeUnidadOPlanExacto(t *testing.T) {
	plan := planCoberturaFichaPrueba()
	casos := []struct {
		nombre   string
		cambiar  func(*consultorDetalleConCoberturaFirmaNominalRRHH, *fuentePlanCoberturaFichaPrueba, *lectorDisponibilidadFichaPrueba)
		llamadas int
	}{
		{"sin montaje", func(c *consultorDetalleConCoberturaFirmaNominalRRHH, _ *fuentePlanCoberturaFichaPrueba, _ *lectorDisponibilidadFichaPrueba) {
			c.plan = nil
		}, 0},
		{"sin asignacion", func(_ *consultorDetalleConCoberturaFirmaNominalRRHH, _ *fuentePlanCoberturaFichaPrueba, l *lectorDisponibilidadFichaPrueba) {
			l.detalle.Asignacion = nil
		}, 0},
		{"unidad ajena", func(_ *consultorDetalleConCoberturaFirmaNominalRRHH, _ *fuentePlanCoberturaFichaPrueba, l *lectorDisponibilidadFichaPrueba) {
			l.detalle.Asignacion.UnidadRef = "unidad:sin-cobertura"
		}, 1},
		{"organizacion ajena", func(_ *consultorDetalleConCoberturaFirmaNominalRRHH, _ *fuentePlanCoberturaFichaPrueba, l *lectorDisponibilidadFichaPrueba) {
			l.detalle.Resumen.OrganizacionRef = "organizacion:otra"
		}, 1},
		{"expediente cruzado", func(_ *consultorDetalleConCoberturaFirmaNominalRRHH, _ *fuentePlanCoberturaFichaPrueba, l *lectorDisponibilidadFichaPrueba) {
			l.detalle.Resumen.ExpedienteRef = "expediente:otro"
		}, 0},
		{"version cruzada", func(_ *consultorDetalleConCoberturaFirmaNominalRRHH, _ *fuentePlanCoberturaFichaPrueba, l *lectorDisponibilidadFichaPrueba) {
			l.detalle.Resumen.Version++
		}, 0},
		{"plan cruzado", func(c *consultorDetalleConCoberturaFirmaNominalRRHH, _ *fuentePlanCoberturaFichaPrueba, _ *lectorDisponibilidadFichaPrueba) {
			c.fijado.Version++
		}, 1},
		{"plan caido", func(_ *consultorDetalleConCoberturaFirmaNominalRRHH, f *fuentePlanCoberturaFichaPrueba, _ *lectorDisponibilidadFichaPrueba) {
			f.err = errors.New("plan no disponible")
		}, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuentePlanCoberturaFichaPrueba{plan: plan}
			lector := &lectorDisponibilidadFichaPrueba{detalle: detalleCoberturaFichaPrueba()}
			c := consultorDetalleConCoberturaFirmaNominalRRHH{lector: lector, plan: fuente, fijado: plan.Version}
			caso.cambiar(&c, fuente, lector)
			detalle, err := c.Consultar(context.Background(), solicitudCoberturaFichaPrueba(t))
			if err != nil || lector.llamadas != 1 || fuente.llamadas != caso.llamadas ||
				detalle.CoberturaFirmaNominal.Estado != ports.CoberturaFirmaNominalNoDisponible ||
				detalle.CoberturaFirmaNominal.Plan != (domain.VersionPlanFirmaV2{}) ||
				len(detalle.CoberturaFirmaNominal.Documentos) != 0 {
				t.Fatalf("cobertura no aislada: %+v, plan=%d, error=%v", detalle.CoberturaFirmaNominal, fuente.llamadas, err)
			}
		})
	}
}
