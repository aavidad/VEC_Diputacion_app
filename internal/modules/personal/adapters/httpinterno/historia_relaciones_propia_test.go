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
)

type consultaHistoriaRelacionesHTTPPrueba struct {
	solicitud domain.SolicitudHistoriaRelacionesPropia
	llamadas  int
	err       error
	ajena     bool
}

func (c *consultaHistoriaRelacionesHTTPPrueba) Consultar(_ context.Context, s domain.SolicitudHistoriaRelacionesPropia) (ports.ResultadoHistoriaRelacionesPropia, error) {
	c.llamadas++
	c.solicitud = s
	if c.err != nil {
		return ports.ResultadoHistoriaRelacionesPropia{}, c.err
	}
	m, err := domain.NuevoMaterialHistoriaRelacionesPropia(s)
	if err != nil {
		return ports.ResultadoHistoriaRelacionesPropia{}, err
	}
	h := domain.HistoriaRelacionesPropia{EmpleadoRef: m.EmpleadoRef(), Corte: s.Corte, Cobertura: "parcial", Revisiones: []domain.RevisionRelacionPropia{{
		RelacionRef: "rel_" + strings.Repeat("B", 24), Estado: "vigente", Regimen: "Funcionarial", Modalidad: "Temporal", Unidad: "Unidad sintética", Puesto: "Técnico/a", Situacion: "Servicio activo",
		Traza: domain.TrazaEmpleadoB2{Desde: "2020-01-01", RegistradaEn: s.Corte.ConocidoEn.Add(-time.Hour), Version: 2, ActoRef: "acto:relacion", FuenteRef: "fuente:personal", FuenteVersion: 3},
	}}}
	if c.ajena {
		h.EmpleadoRef = "emp_" + strings.Repeat("Z", 24)
	}
	ref := "aud_v3_" + strings.Repeat("a", 32)
	return ports.ResultadoHistoriaRelacionesPropia{Historia: h, Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: ref, DecisionRef: "dec_privada", EfectoRef: m.EmpleadoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: ref, ConsultadaEn: s.Corte.ConocidoEn.Add(time.Microsecond)}}, nil
}

type registroHistoriaRelacionesHTTPPrueba struct {
	intentos []ports.IntentoHistoriaRelacionesPropia
	err      error
}

func (*registroHistoriaRelacionesHTTPPrueba) VerificarRegistroHistoriaRelacionesPropia(context.Context) error {
	return nil
}
func (r *registroHistoriaRelacionesHTTPPrueba) RegistrarIntentoHistoriaRelacionesPropia(_ context.Context, i ports.IntentoHistoriaRelacionesPropia) error {
	r.intentos = append(r.intentos, i)
	return r.err
}

func peticionHistoriaRelacionesHTTPPrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaHistoriaRelacionesPropia, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func manejadorHistoriaRelacionesHTTPPrueba(t *testing.T, c *consultaHistoriaRelacionesHTTPPrueba, registro *registroHistoriaRelacionesHTTPPrueba) *ManejadorHistoriaRelacionesPropia {
	t.Helper()
	m, err := NuevoManejadorHistoriaRelacionesPropia(actorFichaPropiaHTTP{actor: actorHistoriaHTTPPrueba(t)}, c, registro, func() time.Time {
		return time.Date(2026, 9, 25, 10, 0, 0, 123456789, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHistoriaRelacionesHTTPDosFechasYProyeccionCerrada(t *testing.T) {
	c, registro := &consultaHistoriaRelacionesHTTPPrueba{}, &registroHistoriaRelacionesHTTPPrueba{}
	m := manejadorHistoriaRelacionesHTTPPrueba(t, c, registro)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, peticionHistoriaRelacionesHTTPPrueba(`{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01"}`))
	if w.Code != 200 || c.llamadas != 1 || len(registro.intentos) != 0 || c.solicitud.Corte.ConocidoEn.Nanosecond() != 123456000 {
		t.Fatalf("consulta/corte: %d, %d", w.Code, c.llamadas)
	}
	for _, esperado := range []string{`"relacion_ref":"rel_`, `"traza":{`, `"version":2`, `"efectos_hasta":"2027-01-01"`} {
		if !strings.Contains(w.Body.String(), esperado) {
			t.Fatalf("falta %s: %s", esperado, w.Body.String())
		}
	}
	for _, prohibido := range []string{"empleado_ref", "emp_", "decision_ref", "consumo_huella", "auditoria_ref", "dec_privada"} {
		if strings.Contains(w.Body.String(), prohibido) {
			t.Fatal("dato interno expuesto", prohibido)
		}
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("cabeceras de respuesta")
	}
}

func TestHistoriaRelacionesHTTPEntradaAjenayAuditoriaObligatoria(t *testing.T) {
	for _, cuerpo := range []string{`{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01","empleado_ref":"emp_ajeno"}`, `{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01","conocido_en":"2026-01-01"}`, `{"efectos_desde":"2027-01-01","efectos_hasta":"2020-01-01"}`} {
		c, registro := &consultaHistoriaRelacionesHTTPPrueba{}, &registroHistoriaRelacionesHTTPPrueba{}
		w := httptest.NewRecorder()
		manejadorHistoriaRelacionesHTTPPrueba(t, c, registro).ServeHTTP(w, peticionHistoriaRelacionesHTTPPrueba(cuerpo))
		if w.Code != 400 || c.llamadas != 0 || len(registro.intentos) != 1 || registro.intentos[0].Motivo != "entrada_invalida" {
			t.Fatalf("entrada admitida: %s, %d", cuerpo, w.Code)
		}
	}
	c, registro := &consultaHistoriaRelacionesHTTPPrueba{}, &registroHistoriaRelacionesHTTPPrueba{err: errors.New("auditoría caída")}
	w := httptest.NewRecorder()
	manejadorHistoriaRelacionesHTTPPrueba(t, c, registro).ServeHTTP(w, peticionHistoriaRelacionesHTTPPrueba(`{}`))
	if w.Code != 503 || c.llamadas != 0 || strings.Contains(w.Body.String(), "auditoría") {
		t.Fatal("fallo de auditoría reveló validación o datos")
	}
}

func TestHistoriaRelacionesHTTPNoEntregaRespuestaAjenaNiDuplicaFalloDelServicio(t *testing.T) {
	c, registro := &consultaHistoriaRelacionesHTTPPrueba{ajena: true}, &registroHistoriaRelacionesHTTPPrueba{}
	w := httptest.NewRecorder()
	manejadorHistoriaRelacionesHTTPPrueba(t, c, registro).ServeHTTP(w, peticionHistoriaRelacionesHTTPPrueba(`{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01"}`))
	if w.Code != 503 || len(registro.intentos) != 1 || strings.Contains(w.Body.String(), "emp_") {
		t.Fatal("respuesta ajena expuesta")
	}
	c, registro = &consultaHistoriaRelacionesHTTPPrueba{err: domain.ErrHistoriaRelacionesPropiaDenegada}, &registroHistoriaRelacionesHTTPPrueba{}
	w = httptest.NewRecorder()
	manejadorHistoriaRelacionesHTTPPrueba(t, c, registro).ServeHTTP(w, peticionHistoriaRelacionesHTTPPrueba(`{"efectos_desde":"2020-01-01","efectos_hasta":"2027-01-01"}`))
	if w.Code != 403 || len(registro.intentos) != 0 {
		t.Fatal("se duplicó la auditoría del servicio")
	}
}
