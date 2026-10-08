package auditoria

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type identidadFuenteAuditoriaHTTPPrueba struct {
	identidad IdentidadResuelta
	fuentes   []FuenteConsulta
	bodyVacio bool
}

func (i *identidadFuenteAuditoriaHTTPPrueba) ResolverIdentidadConsulta(_ context.Context, r *http.Request, fuente FuenteConsulta) (IdentidadResuelta, error) {
	i.fuentes = append(i.fuentes, fuente)
	i.bodyVacio = r.Body == http.NoBody && r.GetBody == nil
	return i.identidad, nil
}

func cuerpoConsultaFuentePrueba(fuente, expediente string) string {
	return `{"fuente":"` + fuente + `","expediente_ref":"` + expediente + `",` +
		`"desde":"2026-09-27T00:00:00Z","hasta":"2026-09-28T00:00:00Z",` +
		`"finalidad_ref":"auditoria_rrhh",` +
		`"motivo_ref":"motivos:1:motivo_11111111111111111111111111111111","limite":10}`
}

func TestAuditoriaResuelveFuenteTipadaTrasDecodificarUnaVez(t *testing.T) {
	identidad := &identidadFuenteAuditoriaHTTPPrueba{identidad: identidadVigenteAuditoriaHTTPPrueba(t, time.Now())}
	opciones := &opcionesAuditoriaHTTPPrueba{}
	h, err := NuevoManejador(&Servicio{}, opciones, identidad)
	if err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRecorder()
	peticionGET := httptest.NewRequest(http.MethodGet, RutaOpciones, nil)
	peticionGET.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("material que no debe releerse")), nil
	}
	h.ServeHTTP(get, peticionGET)
	if get.Code != http.StatusOK || len(identidad.fuentes) != 1 || identidad.fuentes[0] != FuenteConsultaGeneral || !identidad.bodyVacio {
		t.Fatalf("GET debe usar contexto general: status=%d fuentes=%v", get.Code, identidad.fuentes)
	}
	for _, caso := range []struct {
		fuente, expediente string
		esperado           FuenteConsulta
	}{
		{"ct", "participacion:referencia-opaca", FuenteConsultaCT},
		{"bolsa", "expediente:ct:referencia-opaca", FuenteConsultaBolsa},
	} {
		r := httptest.NewRequest(http.MethodPost, RutaConsulta, strings.NewReader(cuerpoConsultaFuentePrueba(caso.fuente, caso.expediente)))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Fuente", "personal")
		r.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("material que no debe releerse")), nil
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusServiceUnavailable || identidad.fuentes[len(identidad.fuentes)-1] != caso.esperado || !identidad.bodyVacio {
			t.Fatalf("POST fuente=%q status=%d fuentes=%v bodyVacio=%v", caso.fuente, w.Code, identidad.fuentes, identidad.bodyVacio)
		}
	}
}

func TestAuditoriaGETNoEntregaCuerpoAlResolvedor(t *testing.T) {
	identidad := &identidadFuenteAuditoriaHTTPPrueba{identidad: identidadVigenteAuditoriaHTTPPrueba(t, time.Now())}
	h, err := NuevoManejador(&Servicio{}, &opcionesAuditoriaHTTPPrueba{}, identidad)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		cuerpo      string
		desconocido bool
	}{
		{"material inesperado", false},
		{"material inesperado", true},
	} {
		r := httptest.NewRequest(http.MethodGet, RutaOpciones, strings.NewReader(caso.cuerpo))
		if caso.desconocido {
			r.ContentLength = -1
			r.TransferEncoding = []string{"chunked"}
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || len(identidad.fuentes) != 0 {
			t.Fatalf("GET con cuerpo status=%d fuentes=%v", w.Code, identidad.fuentes)
		}
	}
}

func TestAuditoriaRechazaTodaCabeceraCookieAntesDeIdentidad(t *testing.T) {
	identidad := &identidadFuenteAuditoriaHTTPPrueba{identidad: identidadVigenteAuditoriaHTTPPrueba(t, time.Now())}
	h, err := NuevoManejador(&Servicio{}, &opcionesAuditoriaHTTPPrueba{}, identidad)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct{ metodo, ruta, cuerpo string }{
		{http.MethodGet, RutaOpciones, ""},
		{http.MethodPost, RutaConsulta, cuerpoConsultaFuentePrueba("ct", "expediente:ct:uno")},
	} {
		r := httptest.NewRequest(caso.metodo, caso.ruta, strings.NewReader(caso.cuerpo))
		if caso.metodo == http.MethodPost {
			r.Header.Set("Content-Type", "application/json")
		}
		r.Header.Add("Cookie", "")
		r.Header.Add("Cookie", "sesion=sintetica")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden || len(identidad.fuentes) != 0 {
			t.Fatalf("%s aceptó cookie secundaria: status=%d fuentes=%v", caso.metodo, w.Code, identidad.fuentes)
		}
	}
}

func TestAuditoriaNoResuelveIdentidadConFuenteOCuerpoInvalido(t *testing.T) {
	identidad := &identidadFuenteAuditoriaHTTPPrueba{identidad: identidadVigenteAuditoriaHTTPPrueba(t, time.Now())}
	h, err := NuevoManejador(&Servicio{}, &opcionesAuditoriaHTTPPrueba{}, identidad)
	if err != nil {
		t.Fatal(err)
	}
	base := cuerpoConsultaFuentePrueba("ct", "expediente:ct:uno")
	cuerpoSinCierre := strings.TrimSuffix(base, "}")
	casos := []string{
		cuerpoConsultaFuentePrueba("personal", "expediente:ct:uno"),
		strings.Replace(base, `"fuente":"ct"`, `"fuente":"ct","fuente":"bolsa"`, 1),
		strings.Replace(base, `"fuente":"ct"`, `"fuente":null`, 1),
		strings.Replace(base, `"fuente":"ct"`, `"fuente":"ct","actor_perfil":"admin"`, 1),
		cuerpoSinCierre,
		base + base,
		base + strings.Repeat(" ", maximoCuerpoConsulta),
	}
	for indice, cuerpo := range casos {
		r := httptest.NewRequest(http.MethodPost, RutaConsulta, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		if indice == len(casos)-1 {
			r.ContentLength = -1 // también limitar un cuerpo de tamaño no anunciado
			r.TransferEncoding = []string{"chunked"}
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || len(identidad.fuentes) != 0 {
			t.Fatalf("caso %d status=%d fuentes=%v", indice, w.Code, identidad.fuentes)
		}
	}
}
