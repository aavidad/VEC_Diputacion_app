package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

type consultorPublicacionPrueba struct {
	pagina ports.PaginaVersionesPublicacionOficial
	err    error
}

func (c consultorPublicacionPrueba) ConsultarPublicacionesOficiales(context.Context) (ports.PaginaVersionesPublicacionOficial, error) {
	return c.pagina, c.err
}

type preparadorPublicacionPrueba struct {
	err     error
	alterar bool
}

func (p preparadorPublicacionPrueba) PrepararDeclaracionPublicacionOficial(_ context.Context, d ports.DeclaracionPublicacionOficial) (ports.SolicitudDeclaracionPublicacionOficial, error) {
	if p.alterar {
		d.Medio = "boe"
	}
	return ports.SolicitudDeclaracionPublicacionOficial{Declaracion: d}, p.err
}

type servicioPublicacionPrueba struct {
	recibo   ports.ReciboPublicacionOficial
	err      error
	recibida *ports.DeclaracionPublicacionOficial
}

func (s servicioPublicacionPrueba) Declarar(_ context.Context, q ports.SolicitudDeclaracionPublicacionOficial) (ports.ReciboPublicacionOficial, error) {
	if s.recibida != nil {
		*s.recibida = q.Declaracion
	}
	return s.recibo, s.err
}

const cuerpoPublicacionPrueba = `{"convocatoria_ref":"cv1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_v1","huella_version_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","medio":"bop_granada","boletin_numero":201,"anuncio_ref":"ENSAYO-BOP-2026-000201","fecha_publicacion":"2026-10-15","direccion":null,"sustituye_ref":null}`

func peticionPublicacionPrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaPublicacionesOficialesBases, strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "f195-clave-0001")
	return r
}

// Sin consumidor del núcleo la composición no tiene servicio: deniega cerrado.
func TestPublicacionOficialHTTPSinServicioNuncaAcepta(t *testing.T) {
	h := NuevoHandlerPublicacionesOficialesBases(nil, preparadorPublicacionPrueba{}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPublicacionPrueba(cuerpoPublicacionPrueba))
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"publicacion_oficial_no_habilitada"`) {
		t.Fatalf("sin servicio: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPublicacionesOficialesBases, nil))
	if w.Code != 503 {
		t.Fatalf("lectura sin consultor: %d", w.Code)
	}
}

func TestPublicacionOficialHTTPDeclaraConClaveDeCabecera(t *testing.T) {
	var recibida ports.DeclaracionPublicacionOficial
	for _, caso := range []struct {
		reutilizada bool
		estado      int
	}{{false, 201}, {true, 200}} {
		s := servicioPublicacionPrueba{recibo: ports.ReciboPublicacionOficial{PublicacionRef: "publicacion_bases:1", Reutilizada: caso.reutilizada}, recibida: &recibida}
		w := httptest.NewRecorder()
		NuevoHandlerPublicacionesOficialesBases(nil, preparadorPublicacionPrueba{}, s).ServeHTTP(w, peticionPublicacionPrueba(cuerpoPublicacionPrueba))
		if w.Code != caso.estado || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("estado %d, esperado %d: %s", w.Code, caso.estado, w.Body.String())
		}
	}
	if recibida.ClaveIdempotencia != "f195-clave-0001" || recibida.BoletinNumero == nil || *recibida.BoletinNumero != 201 || recibida.Direccion != nil {
		t.Fatalf("declaración entregada: %+v", recibida)
	}
}

func TestPublicacionOficialHTTPRechazaEntradasAjenas(t *testing.T) {
	s := servicioPublicacionPrueba{}
	for nombre, r := range map[string]*http.Request{
		"declarante_en_cuerpo": peticionPublicacionPrueba(strings.Replace(cuerpoPublicacionPrueba, `"medio"`, `"declarada_por":"x","medio"`, 1)),
		"ambito_en_cuerpo":     peticionPublicacionPrueba(strings.Replace(cuerpoPublicacionPrueba, `"medio"`, `"ambito_ref":"x","medio"`, 1)),
		"dos_objetos":          peticionPublicacionPrueba(cuerpoPublicacionPrueba + "{}"),
		"sin_clave": func() *http.Request {
			r := peticionPublicacionPrueba(cuerpoPublicacionPrueba)
			r.Header.Del("Idempotency-Key")
			return r
		}(),
		"consulta_en_url": httptest.NewRequest(http.MethodGet, RutaPublicacionesOficialesBases+"?ambito_ref=x", nil),
	} {
		w := httptest.NewRecorder()
		NuevoHandlerPublicacionesOficialesBases(consultorPublicacionPrueba{}, preparadorPublicacionPrueba{}, s).ServeHTTP(w, r)
		if w.Code != 400 && w.Code != 404 {
			t.Fatalf("%s admitido: %d", nombre, w.Code)
		}
	}
	w := httptest.NewRecorder()
	NuevoHandlerPublicacionesOficialesBases(nil, preparadorPublicacionPrueba{alterar: true}, s).ServeHTTP(w, peticionPublicacionPrueba(cuerpoPublicacionPrueba))
	if w.Code != 503 {
		t.Fatalf("el preparador cambió la declaración: %d", w.Code)
	}
	w = httptest.NewRecorder()
	NuevoHandlerPublicacionesOficialesBases(nil, nil, nil).ServeHTTP(w, httptest.NewRequest(http.MethodPut, RutaPublicacionesOficialesBases, nil))
	if w.Code != 405 || w.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("PUT: %d", w.Code)
	}
}

func TestPublicacionOficialHTTPCodigosCatalogados(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
		codigo string
	}{
		{vd.ErrAutorizacionDenegada, 403, "acceso_denegado"},
		{ports.ErrPublicacionOficialNoVigente, 409, "version_no_vigente"},
		{ports.ErrPublicacionOficialConflicto, 409, "publicacion_en_conflicto"},
		{ports.ErrPublicacionOficialInvalida, 400, "solicitud_invalida"},
		{ports.ErrPublicacionOficialNoDisponible, 503, "publicacion_oficial_no_habilitada"},
		{errors.Join(ports.ErrPublicacionOficialNoConfirmada, vd.ErrAutorizacionDenegada), 503, "resultado_no_confirmado"},
		{errors.New("otro"), 503, "servicio_no_disponible"},
	} {
		w := httptest.NewRecorder()
		NuevoHandlerPublicacionesOficialesBases(nil, preparadorPublicacionPrueba{}, servicioPublicacionPrueba{err: caso.err}).ServeHTTP(w, peticionPublicacionPrueba(cuerpoPublicacionPrueba))
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"`+caso.codigo+`"`) {
			t.Fatalf("%v: %d %s", caso.err, w.Code, w.Body.String())
		}
	}
}

func TestPublicacionOficialHTTPListaSinAmbitos(t *testing.T) {
	pagina := ports.PaginaVersionesPublicacionOficial{Completa: true, Items: []ports.VersionPublicacionOficial{{
		ConvocatoriaRef: "cv1_x_v1", Secuencia: 1, Titulo: "Bolsa", HuellaVersionSHA256: strings.Repeat("c", 64),
		UnidadRef: "unidad:secreta", AmbitoRef: "ambito:secreto"}}}
	w := httptest.NewRecorder()
	NuevoHandlerPublicacionesOficialesBases(consultorPublicacionPrueba{pagina: pagina}, nil, nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPublicacionesOficialesBases, nil))
	if w.Code != 200 || strings.Contains(w.Body.String(), "secret") || !strings.Contains(w.Body.String(), `"publicacion":null`) {
		t.Fatalf("lista: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	NuevoHandlerPublicacionesOficialesBases(consultorPublicacionPrueba{err: vd.ErrAutorizacionDenegada}, nil, nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPublicacionesOficialesBases, nil))
	if w.Code != 403 {
		t.Fatalf("lista sin concesión: %d", w.Code)
	}
}
