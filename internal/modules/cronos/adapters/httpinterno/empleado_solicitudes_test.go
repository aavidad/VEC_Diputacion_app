package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverSolicitudesPrueba struct {
	llamadas int
	err      error
}

func (r *resolverSolicitudesPrueba) ResolverConsultaMovimientosPropios(*http.Request) (ports.OrdenConsultaMovimientos, error) {
	r.llamadas++
	return ports.OrdenConsultaMovimientos{}, r.err
}
func (r *resolverSolicitudesPrueba) ResolverSolicitudCorreccionPropia(*http.Request) (ports.OrdenConsumoCorreccion, error) {
	r.llamadas++
	return ports.OrdenConsumoCorreccion{}, r.err
}
func (r *resolverSolicitudesPrueba) ResolverPermisosPropios(*http.Request) (ports.OrdenPermisosPropios, error) {
	r.llamadas++
	return ports.OrdenPermisosPropios{}, r.err
}

type casoSolicitudesPrueba struct {
	err      error
	replay   bool
	olvido   ports.SolicitudOlvidoMarcaje
	peticion ports.PeticionPermisoPropio
	anio     int
	periodo  ports.PeriodoSaldo
}

func (c *casoSolicitudesPrueba) ConsultarMovimientos(_ context.Context, _ ports.OrdenConsultaMovimientos, p ports.PeriodoSaldo, _, _ string) (ports.ConsultaMovimientos, error) {
	c.periodo = p
	return ports.ConsultaMovimientos{Periodo: ports.PeriodoConsultaSaldo{Tipo: p}}, c.err
}
func (c *casoSolicitudesPrueba) SolicitarOlvido(_ context.Context, _ ports.OrdenConsumoCorreccion, s ports.SolicitudOlvidoMarcaje) (ports.ReciboCorreccion, error) {
	c.olvido = s
	return ports.ReciboCorreccion{SolicitudRef: "correccion:cronos:" + s.ClaveOperacion, Estado: "pendiente_responsable", Version: 1, InstanteUTC: time.Now().UTC(), Replay: c.replay}, c.err
}
func (c *casoSolicitudesPrueba) ConsultarPermisosPropios(_ context.Context, _ ports.OrdenPermisosPropios, anio int) (ports.ConsultaPermisosPropios, error) {
	c.anio = anio
	return ports.ConsultaPermisosPropios{Anio: anio}, c.err
}
func (c *casoSolicitudesPrueba) SolicitarPermisoPropio(_ context.Context, _ ports.OrdenPermisosPropios, p ports.PeticionPermisoPropio) (ports.ReciboPermisoPropio, error) {
	c.peticion = p
	return ports.ReciboPermisoPropio{SolicitudRef: "permiso:cronos:solicitud:" + p.ClaveOperacion, Replay: c.replay}, c.err
}

func peticionJSON(metodo, ruta, cuerpo string) *http.Request {
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestMovimientosYPermisosPropiosParametrosYFalloCerrado(t *testing.T) {
	caso, resolver := &casoSolicitudesPrueba{}, &resolverSolicitudesPrueba{}
	mov, _ := NuevoManejadorMovimientosPropios(caso, resolver)
	per, _ := NuevoManejadorPermisosPropios(caso, resolver)
	for _, c := range []struct {
		h      http.Handler
		r      *http.Request
		estado int
	}{
		{mov, httptest.NewRequest(http.MethodGet, RutaConsultarMovimientosPropios+"?periodo=anio", nil), http.StatusOK},
		{mov, httptest.NewRequest(http.MethodGet, RutaConsultarMovimientosPropios+"?periodo=anio&empleado=emp_x", nil), http.StatusBadRequest},
		{mov, httptest.NewRequest(http.MethodPost, RutaConsultarMovimientosPropios+"?periodo=anio", nil), http.StatusMethodNotAllowed},
		{per, httptest.NewRequest(http.MethodGet, RutaConsultarPermisosPropios, nil), http.StatusOK},
		{per, httptest.NewRequest(http.MethodGet, RutaConsultarPermisosPropios+"?anio=2026", nil), http.StatusOK},
		{per, httptest.NewRequest(http.MethodGet, RutaConsultarPermisosPropios+"?anio=26", nil), http.StatusBadRequest},
		{per, httptest.NewRequest(http.MethodGet, RutaConsultarPermisosPropios+"?anio=2026&empleado=x", nil), http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		c.h.ServeHTTP(w, c.r)
		if w.Code != c.estado || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s: %d", c.r.Method, c.r.URL, w.Code)
		}
	}
	if caso.anio != 2026 || caso.periodo != ports.PeriodoSaldoAnio {
		t.Fatal("parámetros no llegan al caso de uso", caso.anio, caso.periodo)
	}
	resolver.err = ports.ErrEmpleadoNoAcreditado
	w := httptest.NewRecorder()
	mov.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaConsultarMovimientosPropios+"?periodo=hoy", nil))
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "sin_empleado") {
		t.Fatal("sin empleado no deniega con motivo", w.Code, w.Body.String())
	}
}

func TestSolicitudesPropiasCuerpoEstrictoYErroresNominales(t *testing.T) {
	caso, resolver := &casoSolicitudesPrueba{}, &resolverSolicitudesPrueba{}
	cor, _ := NuevoManejadorCorreccionPropia(caso, resolver)
	per, _ := NuevoManejadorPermisosPropios(caso, resolver)
	correcta := `{"clave_operacion":"corr-clave-0001","movimiento":"entrada","fecha_civil":"2026-09-24","hora_pretendida":"08:00"}`
	w := httptest.NewRecorder()
	cor.ServeHTTP(w, peticionJSON(http.MethodPost, RutaSolicitarCorreccionPropia, correcta))
	var cuerpo struct {
		Recibo map[string]any `json:"recibo"`
	}
	if w.Code != http.StatusCreated || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil || cuerpo.Recibo["solicitud_ref"] != "correccion:cronos:corr-clave-0001" || !caso.olvido.HuecoDeclarado {
		t.Fatal("corrección nueva", w.Code, w.Body.String())
	}
	caso.replay = true
	w = httptest.NewRecorder()
	cor.ServeHTTP(w, peticionJSON(http.MethodPost, RutaSolicitarCorreccionPropia, correcta))
	if w.Code != http.StatusOK {
		t.Fatal("replay no responde 200", w.Code)
	}
	llamadas := resolver.llamadas
	for _, malo := range []string{
		`{"clave_operacion":"corr-clave-0001","movimiento":"entrada","fecha_civil":"2026-09-24"}`,
		`{"clave_operacion":"corr-clave-0001","movimiento":"entrada","fecha_civil":"2026-09-24","hora_pretendida":"08:00","empleado_ref":"emp_x"}`,
		`{"clave_operacion":"a","clave_operacion":"b","movimiento":"entrada","fecha_civil":"2026-09-24","hora_pretendida":"08:00"}`,
		`{"clave_operacion":1,"movimiento":"entrada","fecha_civil":"2026-09-24","hora_pretendida":"08:00"}`,
		correcta + `{}`,
	} {
		w := httptest.NewRecorder()
		cor.ServeHTTP(w, peticionJSON(http.MethodPost, RutaSolicitarCorreccionPropia, malo))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("cuerpo inválido aceptado %s: %d", malo, w.Code)
		}
	}
	if resolver.llamadas != llamadas {
		t.Fatal("un cuerpo inválido llega a la autoridad")
	}
	permiso := `{"clave_operacion":"perm-clave-0001","permiso_ref":"permiso:cronos:asuntos-propios","desde":"2026-10-05","hasta":"2026-10-06"}`
	for err, esperado := range map[error]int{
		ports.ErrPermisoNoSolicitable:      http.StatusUnprocessableEntity,
		ports.ErrCalendarioNoPublicado:     http.StatusUnprocessableEntity,
		ports.ErrPermisoFueraDeLimites:     http.StatusUnprocessableEntity,
		ports.ErrPermisoSolapado:           http.StatusUnprocessableEntity,
		ports.ErrClaveOperacionEnConflicto: http.StatusConflict,
		ports.ErrSolicitudCronosInvalida:   http.StatusBadRequest,
		ports.ErrDependenciaNoDisponible:   http.StatusServiceUnavailable,
	} {
		caso.err = err
		w := httptest.NewRecorder()
		per.ServeHTTP(w, peticionJSON(http.MethodPost, RutaSolicitarPermisoPropio, permiso))
		if w.Code != esperado {
			t.Fatalf("%v: %d", err, w.Code)
		}
	}
	caso.err, caso.replay = nil, false
	w = httptest.NewRecorder()
	per.ServeHTTP(w, peticionJSON(http.MethodPost, RutaSolicitarPermisoPropio, permiso))
	if w.Code != http.StatusCreated || caso.peticion.PermisoRef != "permiso:cronos:asuntos-propios" {
		t.Fatal("solicitud nueva", w.Code)
	}
	w = httptest.NewRecorder()
	per.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaSolicitarPermisoPropio, nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal("GET sobre solicitudes", w.Code)
	}
}
