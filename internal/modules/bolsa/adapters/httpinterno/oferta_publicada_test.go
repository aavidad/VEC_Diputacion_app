package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorOfertasPrueba struct {
	publicar  *EntradaPublicarOferta
	resolver  *EntradaResolverOferta
	consultar string
	limite    int
}

func (p *preparadorOfertasPrueba) PrepararSolicitudPublicarOferta(_ context.Context, e EntradaPublicarOferta) (ports.SolicitudPublicarOferta, error) {
	p.publicar = &e
	return ports.SolicitudPublicarOferta{BolsaRef: e.BolsaRef, Datos: e.Datos, ClaveIdempotencia: e.ClaveIdempotencia}, nil
}
func (p *preparadorOfertasPrueba) PrepararSolicitudResolverOferta(_ context.Context, e EntradaResolverOferta) (ports.SolicitudResolverOferta, error) {
	p.resolver = &e
	return ports.SolicitudResolverOferta{BolsaRef: e.BolsaRef, OfertaRef: e.OfertaRef, ParticipacionRef: e.ParticipacionRef}, nil
}
func (p *preparadorOfertasPrueba) PrepararSolicitudConsultarOfertas(_ context.Context, bolsa string, limite int) (ports.SolicitudConsultarOfertas, error) {
	p.consultar, p.limite = bolsa, limite
	return ports.SolicitudConsultarOfertas{BolsaRef: bolsa, Limite: limite}, nil
}

type operadorOfertasPrueba struct {
	err         error
	reutilizada bool
}

func (o operadorOfertasPrueba) PublicarOferta(_ context.Context, q ports.SolicitudPublicarOferta) (ports.OfertaPublicada, error) {
	return ports.OfertaPublicada{OfertaRef: "oferta:1", BolsaRef: q.BolsaRef, Datos: q.Datos, Estado: "abierta", Reutilizada: o.reutilizada}, o.err
}
func (o operadorOfertasPrueba) ResolverOferta(_ context.Context, q ports.SolicitudResolverOferta) (ports.OfertaPublicada, error) {
	return ports.OfertaPublicada{OfertaRef: q.OfertaRef, Estado: "adjudicada", Reutilizada: o.reutilizada}, o.err
}
func (o operadorOfertasPrueba) ConsultarOfertas(context.Context, ports.SolicitudConsultarOfertas) ([]ports.OfertaPublicada, error) {
	return []ports.OfertaPublicada{}, o.err
}

func peticionOferta(metodo, ruta, cuerpo, clave string) *http.Request {
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	if cuerpo != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	r.Header.Set("Accept", "application/json")
	if clave != "" {
		r.Header.Set("Idempotency-Key", clave)
	}
	return r
}

func codigoOferta(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var sobre struct {
		Error struct{ Codigo string } `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &sobre)
	return sobre.Error.Codigo
}

const cuerpoOfertaPrueba = `{"bolsa_ref":"bolsa:1","datos":{"categoria":"Auxiliar","centro":"Residencia","fecha_inicio":"2026-10-01","descripcion":"Sustitución"}}`

func TestHandlerOfertasPublicaYDistingueReplay(t *testing.T) {
	for _, caso := range []struct {
		reutilizada bool
		estado      int
	}{{false, http.StatusCreated}, {true, http.StatusOK}} {
		p := &preparadorOfertasPrueba{}
		h, _ := NuevoHandlerOfertasPublicadas(p, operadorOfertasPrueba{reutilizada: caso.reutilizada})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionOferta(http.MethodPost, RutaOfertasPublicadas, cuerpoOfertaPrueba, "clave-oferta-1"))
		if w.Code != caso.estado || p.publicar == nil || p.publicar.ClaveIdempotencia != "clave-oferta-1" || p.publicar.Datos.Centro != "Residencia" {
			t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
		}
	}
}

func TestHandlerOfertasRechazaEntradasSinLlegarAlPreparador(t *testing.T) {
	casos := map[string]*http.Request{
		"sin clave":          peticionOferta(http.MethodPost, RutaOfertasPublicadas, cuerpoOfertaPrueba, ""),
		"campo ajeno":        peticionOferta(http.MethodPost, RutaOfertasPublicadas, `{"bolsa_ref":"b","actor":"x","datos":{}}`, "clave-oferta-1"),
		"dos documentos":     peticionOferta(http.MethodPost, RutaOfertasPublicadas, cuerpoOfertaPrueba+cuerpoOfertaPrueba, "clave-oferta-1"),
		"participación ''":   peticionOferta(http.MethodPost, RutaResolucionesOferta, `{"bolsa_ref":"b","oferta_ref":"oferta:1","participacion_ref":""}`, "clave-resol-1"),
		"consulta sin bolsa": peticionOferta(http.MethodGet, RutaOfertasPublicadas, "", ""),
		"parámetro ajeno":    peticionOferta(http.MethodGet, RutaOfertasPublicadas+"?bolsa_ref=b&actor=x", "", ""),
		"límite excesivo":    peticionOferta(http.MethodGet, RutaOfertasPublicadas+"?bolsa_ref=b&limite=101", "", ""),
	}
	for nombre, r := range casos {
		p := &preparadorOfertasPrueba{}
		h, _ := NuevoHandlerOfertasPublicadas(p, operadorOfertasPrueba{})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || p.publicar != nil || p.resolver != nil || p.consultar != "" {
			t.Errorf("%s: estado=%d", nombre, w.Code)
		}
	}
}

func TestHandlerOfertasResuelveLlamamientoDirectoConParticipacionNula(t *testing.T) {
	p := &preparadorOfertasPrueba{}
	h, _ := NuevoHandlerOfertasPublicadas(p, operadorOfertasPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionOferta(http.MethodPost, RutaResolucionesOferta, `{"bolsa_ref":"b","oferta_ref":"oferta:1","participacion_ref":null}`, "clave-resol-1"))
	if w.Code != http.StatusCreated || p.resolver == nil || p.resolver.ParticipacionRef != "" {
		t.Fatalf("estado=%d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionOferta(http.MethodGet, RutaOfertasPublicadas+"?bolsa_ref=bolsa:1&limite=5", "", ""))
	if w.Code != http.StatusOK || p.consultar != "bolsa:1" || p.limite != 5 || !strings.Contains(w.Body.String(), `"vec.bolsa.rrhh.ofertas.v1"`) {
		t.Fatalf("consulta estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
}

func TestHandlerOfertasTraduceErrores(t *testing.T) {
	casos := []struct {
		err    error
		estado int
		codigo string
	}{
		{dominiovec.ErrAutorizacionDenegada, http.StatusForbidden, "acceso_denegado"},
		{ports.ErrOfertaConflicto, http.StatusConflict, "clave_divergente"},
		{ports.ErrOfertaYaResuelta, http.StatusConflict, "oferta_ya_resuelta"},
		{ports.ErrOfertaPlazoAbierto, http.StatusConflict, "plazo_abierto"},
		{ports.ErrOfertaPropuestaCambiada, http.StatusConflict, "propuesta_cambiada"},
		{ports.ErrOfertaInvalida, http.StatusUnprocessableEntity, "oferta_invalida"},
		{ports.ErrPlazoOfertaNoConfigurado, http.StatusServiceUnavailable, "plazo_no_configurado"},
		{context.DeadlineExceeded, http.StatusServiceUnavailable, "servicio_no_disponible"},
	}
	for _, caso := range casos {
		h, _ := NuevoHandlerOfertasPublicadas(&preparadorOfertasPrueba{}, operadorOfertasPrueba{err: caso.err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionOferta(http.MethodPost, RutaResolucionesOferta, `{"bolsa_ref":"b","oferta_ref":"oferta:1","participacion_ref":"p:1"}`, "clave-resol-1"))
		if w.Code != caso.estado || codigoOferta(t, w) != caso.codigo {
			t.Errorf("%v: estado=%d cuerpo=%s", caso.err, w.Code, w.Body.String())
		}
	}
	h, _ := NuevoHandlerOfertasPublicadas(&preparadorOfertasPrueba{}, operadorOfertasPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionOferta(http.MethodGet, RutaResolucionesOferta, "", ""))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "POST" {
		t.Fatalf("método: %d %q", w.Code, w.Header().Get("Allow"))
	}
}

func TestAuditoriaAnotaLasRutasDeOfertas(t *testing.T) {
	for _, r := range []*http.Request{
		peticionOferta(http.MethodPost, RutaOfertasPublicadas, cuerpoOfertaPrueba, "clave-oferta-1"),
		peticionOferta(http.MethodGet, RutaOfertasPublicadas+"?bolsa_ref=b", "", ""),
		peticionOferta(http.MethodPost, RutaResolucionesOferta, "{}", "clave-resol-1"),
	} {
		if _, _, aplicable := intentoAuditableBorradorLlamamiento(r); !aplicable {
			t.Errorf("%s %s sin auditoría de frontera", r.Method, r.URL.Path)
		}
	}
}
