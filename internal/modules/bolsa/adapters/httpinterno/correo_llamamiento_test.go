package httpinterno

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorCorreoPrueba struct{ err error }

func (p preparadorCorreoPrueba) PrepararSolicitudVistaPreviaLlamamiento(_ context.Context, e EntradaVistaPreviaLlamamiento) (ports.SolicitudVistaPreviaLlamamiento, error) {
	return ports.SolicitudVistaPreviaLlamamiento{BolsaRef: e.BolsaRef, ParticipacionRef: e.ParticipacionRef, Configuracion: e.Configuracion}, p.err
}
func (p preparadorCorreoPrueba) PrepararConsultaPlantillaCorreoLlamamiento(context.Context) (dominiovec.ContextoActor, error) {
	return dominiovec.ContextoActor{PersonaRef: "per_prueba"}, p.err
}

type operadorCorreoPrueba struct {
	err    error
	idioma *string
}

func (o operadorCorreoPrueba) VistaPreviaLlamamiento(_ context.Context, q ports.SolicitudVistaPreviaLlamamiento) (ports.VistaPreviaCorreoLlamamiento, error) {
	return ports.VistaPreviaCorreoLlamamiento{Asunto: "Aviso", Cuerpo: "Hola " + q.ParticipacionRef, Caracteres: 10, Limite: 4000}, o.err
}
func (o operadorCorreoPrueba) PlantillaCorreoLlamamiento(_ context.Context, _ dominiovec.ContextoActor, idioma string) (ports.PlantillaCorreoLlamamientoPublica, error) {
	if o.idioma != nil {
		*o.idioma = idioma
	}
	return ports.PlantillaCorreoLlamamientoPublica{PlantillaVersion: "bolsa-llamamiento-v2", Marcadores: []ports.MarcadorCorreoLlamamientoPublico{{Clave: "nombre"}}}, o.err
}

func peticionCorreo(metodo, ruta, cuerpo string) *http.Request {
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	if cuerpo != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func TestHandlerCorreoLlamamientoPlantillaYVistaPrevia(t *testing.T) {
	var idioma string
	h, err := NuevoHandlerCorreoLlamamiento(preparadorCorreoPrueba{}, operadorCorreoPrueba{idioma: &idioma}, "es")
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCorreo(http.MethodGet, RutaPlantillaCorreoLlamamiento, ""))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"plantilla_version":"bolsa-llamamiento-v2"`) || idioma != "es" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("plantilla: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionCorreo(http.MethodPost, RutaVistaPreviaCorreoLlamamiento, `{"bolsa_ref":"bolsa:01","participacion_ref":"part:01","configuracion":{"cuerpo":"Hola {nombre}"}}`))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"cuerpo":"Hola part:01"`) {
		t.Fatalf("vista previa: %d %s", w.Code, w.Body.String())
	}
}

func TestHandlerCorreoLlamamientoRechazos(t *testing.T) {
	h, _ := NuevoHandlerCorreoLlamamiento(preparadorCorreoPrueba{}, operadorCorreoPrueba{}, "es")
	casos := []struct {
		nombre string
		r      *http.Request
		estado int
	}{
		{"plantilla por POST", peticionCorreo(http.MethodPost, RutaPlantillaCorreoLlamamiento, `{}`), 405},
		{"vista previa por GET", peticionCorreo(http.MethodGet, RutaVistaPreviaCorreoLlamamiento, ""), 405},
		{"plantilla con consulta", peticionCorreo(http.MethodGet, RutaPlantillaCorreoLlamamiento+"?idioma=en", ""), 400},
		{"campo desconocido", peticionCorreo(http.MethodPost, RutaVistaPreviaCorreoLlamamiento, `{"bolsa_ref":"b","extra":1}`), 400},
		{"dos documentos", peticionCorreo(http.MethodPost, RutaVistaPreviaCorreoLlamamiento, `{"bolsa_ref":"b"}{}`), 400},
		{"ruta ajena", peticionCorreo(http.MethodGet, RutaEmisionesLlamamiento+"/otra", ""), 404},
	}
	for _, c := range casos {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, c.r)
		if w.Code != c.estado {
			t.Fatalf("%s: %d %s", c.nombre, w.Code, w.Body.String())
		}
	}
	sinTipo := peticionCorreo(http.MethodPost, RutaVistaPreviaCorreoLlamamiento, `{}`)
	sinTipo.Header.Del("Content-Type")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, sinTipo)
	if w.Code != 400 {
		t.Fatalf("sin Content-Type: %d", w.Code)
	}
}

func TestHandlerCorreoLlamamientoCodigosDeError(t *testing.T) {
	invalida := func(err error) error { return fmt.Errorf("%w: %w", ports.ErrEmisionLlamamientoInvalida, err) }
	for _, caso := range []struct {
		err    error
		estado int
		codigo string
	}{
		{invalida(dominiobolsa.ErrCorreoLlamamientoExcedeLimite), 422, "correo_excede_limite"},
		{invalida(dominiobolsa.ErrDatosCorreoLlamamientoIncompletos), 422, "datos_incompletos"},
		{invalida(dominiobolsa.ErrPlantillaCorreoLlamamiento), 422, "plantilla_invalida"},
		{ports.ErrEmisionLlamamientoInvalida, 422, "emision_invalida"},
		{dominiovec.ErrAutorizacionDenegada, 403, "acceso_denegado"},
		{ports.ErrEmisionLlamamientoNoDisponible, 503, "servicio_no_disponible"},
	} {
		h, _ := NuevoHandlerCorreoLlamamiento(preparadorCorreoPrueba{}, operadorCorreoPrueba{err: caso.err}, "es")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionCorreo(http.MethodPost, RutaVistaPreviaCorreoLlamamiento, `{"bolsa_ref":"bolsa:01","participacion_ref":"part:01","configuracion":{}}`))
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"codigo":"`+caso.codigo+`"`) {
			t.Fatalf("%v: %d %s", caso.err, w.Code, w.Body.String())
		}
	}
	h, _ := NuevoHandlerCorreoLlamamiento(preparadorCorreoPrueba{err: dominiovec.ErrAutorizacionDenegada}, operadorCorreoPrueba{}, "es")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionCorreo(http.MethodGet, RutaPlantillaCorreoLlamamiento, ""))
	if w.Code != 403 {
		t.Fatalf("plantilla sin sesión: %d", w.Code)
	}
}

func TestAuditoriaClasificaRutasCorreoLlamamiento(t *testing.T) {
	for _, r := range []*http.Request{peticionCorreo(http.MethodGet, RutaPlantillaCorreoLlamamiento, ""), peticionCorreo(http.MethodPost, RutaVistaPreviaCorreoLlamamiento, "{}")} {
		accion, clase, ok := intentoAuditableBorradorLlamamiento(r)
		if !ok || accion != ports.AccionIntentoConsultarBorradorLlamamiento || clase != ports.ClaseRutaEmisionesLlamamiento {
			t.Fatalf("%s %s: %v %v %v", r.Method, r.URL.Path, accion, clase, ok)
		}
	}
	if _, _, ok := intentoAuditableBorradorLlamamiento(peticionCorreo(http.MethodPost, RutaPlantillaCorreoLlamamiento, "{}")); ok {
		t.Fatal("un método no admitido no se clasifica como consulta")
	}
	if _, err := NuevoHandlerCorreoLlamamiento(nil, operadorCorreoPrueba{}, "es"); err == nil {
		t.Fatal("sin preparador no se compone")
	}
}
