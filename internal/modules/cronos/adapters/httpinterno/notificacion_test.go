package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverNotificacionesPrueba struct {
	llamadas int
	err      error
}

func (r *resolverNotificacionesPrueba) ResolverNotificacionesPropias(*http.Request) (ports.OrdenNotificacionesPropias, error) {
	r.llamadas++
	return ports.OrdenNotificacionesPropias{}, r.err
}
func (r *resolverNotificacionesPrueba) ResolverBandejaNotificaciones(*http.Request) (ports.OrdenBandejaNotificaciones, error) {
	r.llamadas++
	return ports.OrdenBandejaNotificaciones{}, r.err
}

type casoNotificacionesPrueba struct {
	err      error
	replay   bool
	registro ports.PeticionRegistroNotificacion
	atencion ports.PeticionAtencionNotificacion
}

func (c *casoNotificacionesPrueba) ConsultarPropias(context.Context, ports.OrdenNotificacionesPropias) (ports.ConsultaNotificacionesPropias, error) {
	return ports.ConsultaNotificacionesPropias{Tipos: []ports.TipoNotificacion{}, Notificaciones: []ports.NotificacionPropia{}}, c.err
}
func (c *casoNotificacionesPrueba) RegistrarNotificacion(_ context.Context, _ ports.OrdenNotificacionesPropias, p ports.PeticionRegistroNotificacion) (ports.ReciboNotificacion, error) {
	c.registro = p
	return ports.ReciboNotificacion{NotificacionRef: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001", InstanteUTC: time.Now().UTC(), Replay: c.replay}, c.err
}
func (c *casoNotificacionesPrueba) ConsultarBandeja(context.Context, ports.OrdenBandejaNotificaciones) (ports.BandejaNotificaciones, error) {
	return ports.BandejaNotificaciones{Notificaciones: []ports.NotificacionRecibida{}}, c.err
}
func (c *casoNotificacionesPrueba) AtenderNotificacion(_ context.Context, _ ports.OrdenBandejaNotificaciones, p ports.PeticionAtencionNotificacion) (ports.ReciboAtencionNotificacion, error) {
	c.atencion = p
	return ports.ReciboAtencionNotificacion{NotificacionRef: p.NotificacionRef, Replay: c.replay}, c.err
}

const cuerpoNotificacion = `{"clave_operacion":"not-a-00001","tipo_version_ref":"notificacion:cronos:tipo:otra-comunicacion:sintetico-1","fecha_referida":"2026-09-24","texto":"Texto\nen dos líneas","adjunto_ref":"registro:sintetico:0001","adjunto_sha256":"` + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + `"}`

func TestNotificacionesPropiasRutasYCodigos(t *testing.T) {
	caso, resolver := &casoNotificacionesPrueba{}, &resolverNotificacionesPrueba{}
	h, err := NuevoManejadorNotificacionesPropias(caso, resolver)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		r      *http.Request
		estado int
	}{
		{httptest.NewRequest(http.MethodGet, RutaNotificacionesPropias, nil), http.StatusOK},
		{httptest.NewRequest(http.MethodGet, RutaNotificacionesPropias+"?empleado=emp_x", nil), http.StatusNotFound},
		{httptest.NewRequest(http.MethodPost, RutaNotificacionesPropias, nil), http.StatusMethodNotAllowed},
		{httptest.NewRequest(http.MethodGet, RutaRegistrarNotificacion, nil), http.StatusMethodNotAllowed},
		{peticionJSON(http.MethodPost, RutaRegistrarNotificacion, cuerpoNotificacion), http.StatusCreated},
		{peticionJSON(http.MethodPost, RutaRegistrarNotificacion, strings.Replace(cuerpoNotificacion, `}`, `,"empleado_ref":"emp_x"}`, 1)), http.StatusBadRequest},
		{peticionJSON(http.MethodPost, RutaRegistrarNotificacion, strings.Replace(cuerpoNotificacion, `Texto\nen dos líneas`, strings.Repeat("á", domain.MaximoTextoNotificacion+1), 1)), http.StatusBadRequest},
		{peticionJSON(http.MethodPost, RutaRegistrarNotificacion, strings.Replace(cuerpoNotificacion, `Texto\nen dos líneas`, strings.Repeat("á", domain.MaximoTextoNotificacion), 1)), http.StatusCreated},
		{peticionJSON(http.MethodPost, RutaRegistrarNotificacion, `{"clave_operacion":"not-a-00001"}`), http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, c.r)
		if w.Code != c.estado || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s: %d %s", c.r.Method, c.r.URL, w.Code, w.Body.String())
		}
	}
	if caso.registro.AdjuntoRef != "registro:sintetico:0001" || caso.registro.FechaReferida != "2026-09-24" || caso.registro.TipoVersionRef == "" {
		t.Fatalf("parámetros no llegan al caso de uso: %+v", caso.registro)
	}
	caso.replay = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaRegistrarNotificacion, cuerpoNotificacion))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"replay":true`) {
		t.Fatal("replay distinto", w.Code, w.Body.String())
	}
	for err, esperado := range map[error]struct {
		estado int
		codigo string
	}{
		ports.ErrTipoNotificacionNoVigente: {http.StatusConflict, "tipo_no_vigente"},
		ports.ErrClaveOperacionEnConflicto: {http.StatusConflict, "conflicto"},
		ports.ErrSolicitudCronosInvalida:   {http.StatusBadRequest, "peticion_invalida"},
		errors.New("interno"):              {http.StatusServiceUnavailable, "no_disponible"},
	} {
		caso.err = err
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaRegistrarNotificacion, cuerpoNotificacion))
		if w.Code != esperado.estado || !strings.Contains(w.Body.String(), `"`+esperado.codigo+`"`) || strings.Contains(w.Body.String(), "interno") {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
	resolver.err, caso.err = ports.ErrEmpleadoNoAcreditado, nil
	llamadas := resolver.llamadas
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaRegistrarNotificacion, `{}`))
	if w.Code != http.StatusBadRequest || resolver.llamadas != llamadas {
		t.Fatal("un cuerpo vacío llega a autorizar", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaNotificacionesPropias, nil))
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "sin_empleado") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestBandejaDeNotificacionesRutasYCodigos(t *testing.T) {
	caso, resolver := &casoNotificacionesPrueba{}, &resolverNotificacionesPrueba{}
	h, err := NuevoManejadorBandejaNotificaciones(caso, resolver)
	if err != nil {
		t.Fatal(err)
	}
	atencion := `{"clave_operacion":"ate-r-00001","notificacion_ref":"notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001"}`
	for _, c := range []struct {
		r      *http.Request
		estado int
	}{
		{httptest.NewRequest(http.MethodGet, RutaBandejaNotificaciones, nil), http.StatusOK},
		{httptest.NewRequest(http.MethodGet, RutaBandejaNotificaciones+"?empleado=emp_x", nil), http.StatusNotFound},
		{peticionJSON(http.MethodPost, RutaAtenderNotificacion, atencion), http.StatusCreated},
		{peticionJSON(http.MethodPost, RutaAtenderNotificacion, `{"notificacion_ref":"x"}`), http.StatusBadRequest},
		{httptest.NewRequest(http.MethodGet, RutaAtenderNotificacion, nil), http.StatusMethodNotAllowed},
		{httptest.NewRequest(http.MethodGet, RutaNotificacionesPropias, nil), http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, c.r)
		if w.Code != c.estado || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s: %d", c.r.Method, c.r.URL, w.Code)
		}
	}
	if caso.atencion.ClaveOperacion != "ate-r-00001" {
		t.Fatal("la atención no llega al caso de uso", caso.atencion)
	}
	for err, codigo := range map[error]string{ports.ErrResolucionEstadoCambiado: "estado_cambiado", ports.ErrResolucionNoCompetente: "no_competente"} {
		caso.err = err
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaAtenderNotificacion, atencion))
		if !strings.Contains(w.Body.String(), codigo) {
			t.Fatal(err, w.Code, w.Body.String())
		}
	}
}

func TestBandejaDeNotificacionesDemasiadoGrandeEsConflictoNominal(t *testing.T) {
	caso := &casoNotificacionesPrueba{err: ports.ErrBandejaNotificacionesDemasiadoGrande}
	h, err := NuevoManejadorBandejaNotificaciones(caso, &resolverNotificacionesPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaBandejaNotificaciones, nil))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"bandeja_demasiado_grande"`) || strings.Contains(w.Body.String(), "notificaciones\":") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestResolucionPendienteDeAsignacionEsConflictoNominal(t *testing.T) {
	w := httptest.NewRecorder()
	responderErrorResolucion(w, ports.ErrResolucionPendienteAsignacion)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"pendiente_asignacion"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
