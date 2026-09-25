package httppersonal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ejecutorDisposicionPrueba struct {
	llamadas      int
	oferta, clave string
	err           error
	repetida      bool
}

func (e *ejecutorDisposicionPrueba) ManifestarDisposicion(_ context.Context, _ mibolsa.Orden, oferta, clave string) (puertosbolsa.ReciboDisposicionPortal, error) {
	e.llamadas++
	e.oferta, e.clave = oferta, clave
	return puertosbolsa.ReciboDisposicionPortal{Reutilizada: e.repetida, OfertaRef: oferta, ReciboRef: "recibo:disposicion:" + strings.Repeat("e", 64), ManifestadaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}, e.err
}

func TestDisposicionRegistraYRepite(t *testing.T) {
	e := new(ejecutorDisposicionPrueba)
	h, err := NuevoDisposicion(new(preparadorPrueba), e)
	if err != nil {
		t.Fatal(err)
	}
	oferta := "oferta:" + strings.Repeat("a", 64)
	cuerpo := `{"oferta":"` + oferta + `","clave":"clave-disposicion-1"}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaDisposiciones, cuerpo, nil))
	if w.Code != 201 || e.oferta != oferta || e.clave != "clave-disposicion-1" || !strings.Contains(w.Body.String(), `"estado":"manifestada"`) {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	e.repetida = true
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaDisposiciones, cuerpo, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"repetida":true`) {
		t.Fatalf("repetición: %d %s", w.Code, w.Body.String())
	}
	if AccionPortalEn(http.MethodPost, RutaMiBolsaDisposiciones)[0] != puertosbolsa.AccionManifestarDisposicionPropia || !EsRutaPortal(RutaMiBolsaDisposiciones) {
		t.Fatal("ruta o acción de la disposición")
	}
}

func TestDisposicionRechazaYTraduce(t *testing.T) {
	e := new(ejecutorDisposicionPrueba)
	h, _ := NuevoDisposicion(new(preparadorPrueba), e)
	valido := `{"oferta":"oferta:` + strings.Repeat("a", 64) + `","clave":"clave-disposicion-1"}`
	for nombre, r := range map[string]*http.Request{
		"otro origen": peticionPortal(RutaMiBolsaDisposiciones, valido, map[string]string{"Sec-Fetch-Site": "cross-site"}),
		"cookie":      peticionPortal(RutaMiBolsaDisposiciones, valido, map[string]string{"Cookie": "a=b"}),
		"campo ajeno": peticionPortal(RutaMiBolsaDisposiciones, `{"oferta":"x","clave":"clave-disposicion-1","candidato":"can_x"}`, nil),
		"consulta":    peticionPortal(RutaMiBolsaDisposiciones+"?x=1", valido, nil),
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 || w.Code >= 500 {
			t.Fatalf("%s: status %d", nombre, w.Code)
		}
	}
	if e.llamadas != 0 {
		t.Fatal("ejecutó una petición rechazada")
	}
	for err, codigo := range map[error]string{
		puertosbolsa.ErrPortalOfertaNoAbierta:          "oferta_no_abierta",
		puertosbolsa.ErrPortalDisposicionYaManifestada: "disposicion_ya_manifestada",
	} {
		h, _ := NuevoDisposicion(new(preparadorPrueba), &ejecutorDisposicionPrueba{err: err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPortal(RutaMiBolsaDisposiciones, valido, nil))
		if w.Code != 409 || !strings.Contains(w.Body.String(), codigo) {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
}

func TestRespuestaIncluyeOfertasSinDatosAjenos(t *testing.T) {
	manifestada := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	i := puertosbolsa.InstantaneaMiBolsa{Ofertas: []puertosbolsa.OfertaPortalCandidato{{
		OfertaRef: "oferta:" + strings.Repeat("b", 64), Bolsa: "bolsa:01",
		Datos:       dominiobolsa.DatosOferta{Categoria: "Auxiliar", Centro: "Residencia", FechaInicio: "2026-10-01", Descripcion: "Sustitución"},
		PublicadaEn: manifestada.Add(-time.Hour), VenceAntesDe: manifestada.Add(48 * time.Hour), Estado: puertosbolsa.EstadoOfertaPortalAbierta,
		Disposicion: &puertosbolsa.DisposicionPropiaPortal{Recibo: "recibo:disposicion:x", ManifestadaEn: manifestada},
	}}}
	salida, err := json.Marshal(respuestaOfertas(i))
	if err != nil || !strings.Contains(string(salida), `"estado":"abierta"`) || !strings.Contains(string(salida), `"fecha_fin":null`) ||
		!strings.Contains(string(salida), `"manifestada_en":"2026-09-25T09:00:00.000000Z"`) {
		t.Fatalf("ofertas: %s %v", salida, err)
	}
	if respuestaOfertas(puertosbolsa.InstantaneaMiBolsa{}) != nil {
		t.Fatal("sin ofertas compuestas la respuesta no debe traerlas")
	}
}
