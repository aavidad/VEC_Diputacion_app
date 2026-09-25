package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverResolucionPrueba struct {
	llamadas int
	err      error
}

func (r *resolverResolucionPrueba) ResolverResolucionPermisos(*http.Request) (ports.OrdenResolucionPermisos, error) {
	r.llamadas++
	return ports.OrdenResolucionPermisos{}, r.err
}
func (r *resolverResolucionPrueba) ResolverAvisosPropios(*http.Request) (ports.OrdenAvisosPropios, error) {
	r.llamadas++
	return ports.OrdenAvisosPropios{}, r.err
}

type casoResolucionPrueba struct {
	err      error
	replay   bool
	paso     domain.PasoPermiso
	peticion ports.PeticionResolucionPermiso
	archivo  ports.PeticionArchivoAviso
}

func (c *casoResolucionPrueba) ConsultarBandeja(_ context.Context, _ ports.OrdenResolucionPermisos, paso domain.PasoPermiso) (ports.BandejaPermisos, error) {
	c.paso = paso
	return ports.BandejaPermisos{Paso: paso, Pendientes: []ports.SolicitudPendiente{}}, c.err
}
func (c *casoResolucionPrueba) ResolverPermiso(_ context.Context, _ ports.OrdenResolucionPermisos, p ports.PeticionResolucionPermiso) (ports.ReciboResolucionPermiso, error) {
	c.peticion = p
	return ports.ReciboResolucionPermiso{ResolucionRef: "permiso:cronos:resolucion:" + p.ClaveOperacion, Estado: domain.EstadoPermisoConcedido, Version: p.VersionEsperada + 1, InstanteUTC: time.Now().UTC(), Replay: c.replay}, c.err
}
func (c *casoResolucionPrueba) ConsultarAvisos(context.Context, ports.OrdenAvisosPropios) (ports.ConsultaAvisosPropios, error) {
	return ports.ConsultaAvisosPropios{Avisos: []ports.AvisoPropio{}}, c.err
}
func (c *casoResolucionPrueba) ArchivarAviso(_ context.Context, _ ports.OrdenAvisosPropios, p ports.PeticionArchivoAviso) (ports.ReciboArchivoAviso, error) {
	c.archivo = p
	return ports.ReciboArchivoAviso{ArchivoRef: "aviso:cronos:archivo:" + p.ClaveOperacion, AvisoRef: p.AvisoRef, Replay: c.replay}, c.err
}

const cuerpoResolucion = `{"clave_operacion":"res-va-00001","solicitud_ref":"permiso:cronos:solicitud:perm-va-00001","paso":"administracion","decision":"denegar","motivo":"Coincide con el cierre","version_esperada":"2"}`

func TestBandejaYResolucionParametrosYCodigos(t *testing.T) {
	caso, resolver := &casoResolucionPrueba{}, &resolverResolucionPrueba{}
	h, err := NuevoManejadorResolucionPermisos(caso, resolver)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		r      *http.Request
		estado int
	}{
		{httptest.NewRequest(http.MethodGet, RutaBandejaPermisos+"?paso=responsable", nil), http.StatusOK},
		{httptest.NewRequest(http.MethodGet, RutaBandejaPermisos+"?paso=administracion", nil), http.StatusOK},
		{httptest.NewRequest(http.MethodGet, RutaBandejaPermisos, nil), http.StatusBadRequest},
		{httptest.NewRequest(http.MethodGet, RutaBandejaPermisos+"?paso=jefatura", nil), http.StatusBadRequest},
		{httptest.NewRequest(http.MethodGet, RutaBandejaPermisos+"?paso=responsable&empleado=emp_x", nil), http.StatusBadRequest},
		{httptest.NewRequest(http.MethodPost, RutaBandejaPermisos+"?paso=responsable", nil), http.StatusMethodNotAllowed},
		{httptest.NewRequest(http.MethodGet, RutaResolverPermiso, nil), http.StatusMethodNotAllowed},
		{peticionJSON(http.MethodPost, RutaResolverPermiso, cuerpoResolucion), http.StatusCreated},
		{peticionJSON(http.MethodPost, RutaResolverPermiso+"?x=1", cuerpoResolucion), http.StatusNotFound},
		{peticionJSON(http.MethodPost, RutaResolverPermiso, strings.Replace(cuerpoResolucion, `"2"`, `2`, 1)), http.StatusBadRequest},
		{peticionJSON(http.MethodPost, RutaResolverPermiso, strings.Replace(cuerpoResolucion, `"2"`, `"02"`, 1)), http.StatusBadRequest},
		{peticionJSON(http.MethodPost, RutaResolverPermiso, strings.Replace(cuerpoResolucion, `}`, `,"empleado_ref":"emp_x"}`, 1)), http.StatusBadRequest},
		{peticionJSON(http.MethodPost, RutaResolverPermiso, strings.Replace(cuerpoResolucion, `Coincide con el cierre`, strings.Repeat("á", domain.MaximoMotivoResolucion+1), 1)), http.StatusBadRequest},
		{peticionJSON(http.MethodPost, RutaResolverPermiso, strings.Replace(cuerpoResolucion, `Coincide con el cierre`, strings.Repeat("á", domain.MaximoMotivoResolucion), 1)), http.StatusCreated},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, c.r)
		if w.Code != c.estado || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s: %d %s", c.r.Method, c.r.URL, w.Code, w.Body.String())
		}
	}
	if caso.paso != domain.PasoAdministracion || caso.peticion.VersionEsperada != 2 || caso.peticion.Decision != domain.DecisionDenegar || caso.peticion.SolicitudRef != "permiso:cronos:solicitud:perm-va-00001" {
		t.Fatalf("parámetros no llegan al caso de uso: %+v %s", caso.peticion, caso.paso)
	}
	caso.replay = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaResolverPermiso, cuerpoResolucion))
	var cuerpo struct {
		Recibo ports.ReciboResolucionPermiso `json:"recibo"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil || !cuerpo.Recibo.Replay || cuerpo.Recibo.Version != 3 {
		t.Fatal("replay distinto", w.Code, w.Body.String())
	}
	for err, esperado := range map[error]struct {
		estado int
		codigo string
	}{
		ports.ErrResolucionNoCompetente:    {http.StatusForbidden, "no_competente"},
		ports.ErrResolucionEstadoCambiado:  {http.StatusConflict, "estado_cambiado"},
		ports.ErrClaveOperacionEnConflicto: {http.StatusConflict, "conflicto"},
		ports.ErrSolicitudCronosInvalida:   {http.StatusBadRequest, "peticion_invalida"},
		ports.ErrDependenciaNoDisponible:   {http.StatusServiceUnavailable, "no_disponible"},
		errors.New("interno"):              {http.StatusServiceUnavailable, "no_disponible"},
	} {
		caso.err = err
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaResolverPermiso, cuerpoResolucion))
		if w.Code != esperado.estado || !strings.Contains(w.Body.String(), `"`+esperado.codigo+`"`) || strings.Contains(w.Body.String(), "interno") {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
}

func TestResolucionNoLlegaAlResolverConCuerpoInvalido(t *testing.T) {
	caso, resolver := &casoResolucionPrueba{}, &resolverResolucionPrueba{}
	h, _ := NuevoManejadorResolucionPermisos(caso, resolver)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaResolverPermiso, `{"clave_operacion":"x"}`))
	if w.Code != http.StatusBadRequest || resolver.llamadas != 0 {
		t.Fatal("un cuerpo incompleto no se rechaza antes de autorizar", w.Code, resolver.llamadas)
	}
	resolver.err = ports.ErrEmpleadoNoAcreditado
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaBandejaPermisos+"?paso=responsable", nil))
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "sin_empleado") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestAvisosConsultaYArchivo(t *testing.T) {
	caso, resolver := &casoResolucionPrueba{}, &resolverResolucionPrueba{}
	h, err := NuevoManejadorAvisosPropios(caso, resolver)
	if err != nil {
		t.Fatal(err)
	}
	aviso := `{"clave_operacion":"arch-00000001","aviso_ref":"aviso:cronos:00000000-0000-4000-8000-000000000001"}`
	for _, c := range []struct {
		r      *http.Request
		estado int
	}{
		{httptest.NewRequest(http.MethodGet, RutaAvisosPropios, nil), http.StatusOK},
		{httptest.NewRequest(http.MethodGet, RutaAvisosPropios+"?empleado=emp_x", nil), http.StatusNotFound},
		{httptest.NewRequest(http.MethodPost, RutaAvisosPropios, nil), http.StatusMethodNotAllowed},
		{peticionJSON(http.MethodPost, RutaArchivarAviso, aviso), http.StatusCreated},
		{peticionJSON(http.MethodPost, RutaArchivarAviso, `{"aviso_ref":"x"}`), http.StatusBadRequest},
		{httptest.NewRequest(http.MethodGet, RutaArchivarAviso, nil), http.StatusMethodNotAllowed},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, c.r)
		if w.Code != c.estado || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s: %d", c.r.Method, c.r.URL, w.Code)
		}
	}
	if caso.archivo.AvisoRef != "aviso:cronos:00000000-0000-4000-8000-000000000001" {
		t.Fatal("el aviso no llega al caso de uso", caso.archivo)
	}
	caso.err = ports.ErrResolucionEstadoCambiado
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaArchivarAviso, aviso))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "estado_cambiado") {
		t.Fatal("un aviso ya archivado no es conflicto", w.Code, w.Body.String())
	}
}
