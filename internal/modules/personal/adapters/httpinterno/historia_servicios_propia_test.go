package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type consultaHistoriaHTTPPrueba struct {
	solicitud  domain.SolicitudHistoriaServiciosPropia
	llamadas   int
	err        error
	ajena      bool
	revisiones []domain.RevisionServicioPropio
}

func (c *consultaHistoriaHTTPPrueba) Consultar(_ context.Context, s domain.SolicitudHistoriaServiciosPropia) (ports.ResultadoHistoriaServiciosPropia, error) {
	c.llamadas++
	c.solicitud = s
	if c.err != nil {
		return ports.ResultadoHistoriaServiciosPropia{}, c.err
	}
	m, e := domain.NuevoMaterialHistoriaServiciosPropia(s)
	if e != nil {
		return ports.ResultadoHistoriaServiciosPropia{}, e
	}
	h := domain.HistoriaServiciosPropia{EmpleadoRef: m.EmpleadoRef(), Corte: s.Corte, Cobertura: "no_acreditada", Revisiones: []domain.RevisionServicioPropio{}}
	h.Revisiones = append(h.Revisiones, c.revisiones...)
	if c.ajena {
		h.EmpleadoRef = "emp_" + strings.Repeat("Z", 24)
	}
	return ports.ResultadoHistoriaServiciosPropia{Historia: h, Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "aud_v3_" + strings.Repeat("a", 32), DecisionRef: "dec_privada", EfectoRef: m.EmpleadoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "aud_v3_" + strings.Repeat("a", 32), ConsultadaEn: s.Corte.ConocidoEn.Add(time.Microsecond)}}, nil
}

type registroHistoriaHTTPPrueba struct {
	intentos []ports.IntentoHistoriaServiciosPropia
	err      error
}

func (r *registroHistoriaHTTPPrueba) VerificarRegistroHistoriaServiciosPropia(context.Context) error {
	return nil
}
func (r *registroHistoriaHTTPPrueba) RegistrarIntentoHistoriaServiciosPropia(_ context.Context, i ports.IntentoHistoriaServiciosPropia) error {
	r.intentos = append(r.intentos, i)
	return r.err
}
func actorHistoriaHTTPPrueba(t *testing.T) core.ContextoActor {
	t.Helper()
	base := actorFichaPropiaPruebaHTTP(t)
	inst := base.Instantanea
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	inst.Vinculos = []core.VinculoReferenciaContextoActor{{VinculoRef: "pep_" + z, Version: 1, Tipo: core.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}
	a, e := core.NuevoContextoActor(core.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}, inst, ahora)
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func peticionHistoriaHTTPPrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaHistoriaServiciosPropia, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}
func manejadorHistoriaHTTPPrueba(t *testing.T, c *consultaHistoriaHTTPPrueba, registro *registroHistoriaHTTPPrueba) *ManejadorHistoriaServiciosPropia {
	t.Helper()
	m, e := NuevoManejadorHistoriaServiciosPropia(actorFichaPropiaHTTP{actor: actorHistoriaHTTPPrueba(t)}, c, registro, func() time.Time { return time.Date(2026, 9, 25, 10, 0, 0, 123456789, time.UTC) })
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func TestHistoriaServiciosHTTPUsaCapturaActualYRelojServidorSinEvidenciaInterna(t *testing.T) {
	c := &consultaHistoriaHTTPPrueba{}
	registro := &registroHistoriaHTTPPrueba{}
	m := manejadorHistoriaHTTPPrueba(t, c, registro)
	llamadas := 0
	m.actor = actorFichaPropiaHTTP{actor: actorHistoriaHTTPPrueba(t), llamadas: &llamadas}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, peticionHistoriaHTTPPrueba(`{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01"}`))
	if w.Code != 200 || llamadas != 1 || c.llamadas != 1 || len(registro.intentos) != 0 || c.solicitud.Corte.ConocidoEn.Nanosecond() != 123456000 || !c.solicitud.Corte.ConocidoEn.Equal(time.Date(2026, 9, 25, 10, 0, 0, 123456000, time.UTC)) {
		t.Fatal("corte/captura divergentes", w.Code)
	}
	for _, prohibido := range []string{"empleado_ref", "emp_", "decision_ref", "consumo_huella", "auditoria_ref", "dec_privada"} {
		if strings.Contains(w.Body.String(), prohibido) {
			t.Fatal("material interno expuesto", prohibido)
		}
	}
	if !strings.Contains(w.Body.String(), `"revisiones":[]`) || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal(w.Body.String())
	}
}
func TestHistoriaServiciosHTTPJSONCerradoRegistraFalloComunSinLeer(t *testing.T) {
	for _, cuerpo := range []string{`{}`, `{"efectos_desde":null,"efectos_hasta":"2027-01-01"}`, `{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01","conocido_en":"2026-01-01"}`, `{"efectos_desde":"2020-01-01","efectos_desde":"2021-01-01","efectos_hasta":"2027-01-01"}`, `{"efectos_desde":"2027-01-01","efectos_hasta":"2020-01-01"}`, `{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01"} {}`} {
		c := &consultaHistoriaHTTPPrueba{}
		registro := &registroHistoriaHTTPPrueba{}
		m := manejadorHistoriaHTTPPrueba(t, c, registro)
		w := httptest.NewRecorder()
		m.ServeHTTP(w, peticionHistoriaHTTPPrueba(cuerpo))
		if w.Code != 400 || c.llamadas != 0 || len(registro.intentos) != 1 || registro.intentos[0].Motivo != "entrada_invalida" {
			t.Fatal("JSON no cerrado", cuerpo, w.Code)
		}
	}
	registro := &registroHistoriaHTTPPrueba{err: errors.New("sin acuse")}
	m := manejadorHistoriaHTTPPrueba(t, &consultaHistoriaHTTPPrueba{}, registro)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, peticionHistoriaHTTPPrueba(`{}`))
	if w.Code != 503 {
		t.Fatal("sin acuse revela validación")
	}
}
func TestHistoriaServiciosHTTPRechazaDatosAjenosYConservaErroresNominales(t *testing.T) {
	c := &consultaHistoriaHTTPPrueba{ajena: true}
	registro := &registroHistoriaHTTPPrueba{}
	m := manejadorHistoriaHTTPPrueba(t, c, registro)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, peticionHistoriaHTTPPrueba(`{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01"}`))
	if w.Code != 503 || len(registro.intentos) != 1 || strings.Contains(w.Body.String(), "emp_") {
		t.Fatal("respuesta ajena")
	}
	for _, caso := range []struct {
		err    error
		estado int
	}{{domain.ErrHistoriaServiciosPropiaDenegada, 403}, {domain.ErrHistoriaServiciosPropiaExcedeLimite, 422}, {domain.ErrHistoriaServiciosPropiaNoDisponible, 503}} {
		c := &consultaHistoriaHTTPPrueba{err: caso.err}
		registro := &registroHistoriaHTTPPrueba{}
		m := manejadorHistoriaHTTPPrueba(t, c, registro)
		w := httptest.NewRecorder()
		m.ServeHTTP(w, peticionHistoriaHTTPPrueba(`{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01"}`))
		if w.Code != caso.estado || len(registro.intentos) != 0 {
			t.Fatal("duplicó intento servicio", w.Code)
		}
	}
}

func TestHistoriaServiciosHTTPLigaInstantesUTCConSeisDecimales(t *testing.T) {
	for _, nanos := range []int{0, 123450000} {
		revision := domain.RevisionServicioPropio{ServicioRef: "srv_" + strings.Repeat("A", 24), RelacionRef: "rel_" + strings.Repeat("B", 24), PeriodoDesde: "2010-01-01", PeriodoHasta: "2010-12-31", DiasReconocidos: 365, Estado: "reconocido", Clase: "Servicios previos", Traza: domain.TrazaEmpleadoB2{Desde: "2020-01-01", RegistradaEn: time.Date(2025, 9, 25, 10, 0, 0, 0, time.UTC), Version: 1, ActoRef: "acto:servicios", FuenteRef: "fuente:servicios", FuenteVersion: 1}}
		c := &consultaHistoriaHTTPPrueba{revisiones: []domain.RevisionServicioPropio{revision}}
		m := manejadorHistoriaHTTPPrueba(t, c, &registroHistoriaHTTPPrueba{})
		instante := time.Date(2026, 9, 25, 10, 0, 0, nanos, time.UTC)
		m.ahora = func() time.Time { return instante }
		w := httptest.NewRecorder()
		m.ServeHTTP(w, peticionHistoriaHTTPPrueba(`{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01"}`))
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"conocido_en":"`+instante.Format("2006-01-02T15:04:05.000000Z")+`"`) || !strings.Contains(w.Body.String(), `"registrada_en":"2025-09-25T10:00:00.000000Z"`) {
			t.Fatalf("instante no canónico: %d %s", w.Code, w.Body.String())
		}
	}
}
