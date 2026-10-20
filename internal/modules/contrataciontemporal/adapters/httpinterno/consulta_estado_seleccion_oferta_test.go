package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultorEstadoSeleccionOfertaHTTPPrueba struct {
	resultado application.DatosEstadoSeleccionOfertaParaAdaptador
	err       error
	llamadas  int
}

func (c *consultorEstadoSeleccionOfertaHTTPPrueba) ConsultarEstadoSeleccionOfertaParaAdaptador(
	_ context.Context, ref string,
) (application.DatosEstadoSeleccionOfertaParaAdaptador, error) {
	c.llamadas++
	if c.resultado.ExpedienteRef == "" {
		c.resultado.ExpedienteRef = ref
	}
	return c.resultado, c.err
}

func TestConsultaEstadoSeleccionOfertaGETDurableSinUUIDCliente(t *testing.T) {
	const ref = "expediente:ct:prueba"
	c := &consultorEstadoSeleccionOfertaHTTPPrueba{resultado: application.DatosEstadoSeleccionOfertaParaAdaptador{
		Estado: "pendiente", ExpedienteRef: ref, VersionEsperada: 6,
		ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", VentanaAbierta: true,
	}}
	h, err := NuevoManejadorConsultaEstadoSeleccionOferta(c)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		RutaConsultaEstadoSeleccionOferta+"?expediente_ref="+url.QueryEscape(ref), nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"estado":"pendiente"`) ||
		!strings.Contains(w.Body.String(), `"clave_idempotencia":"018f47a2-6b31-4c80-8a95-4d2e707c5a11"`) ||
		strings.Contains(w.Body.String(), "origen_ct") || c.llamadas != 1 {
		t.Fatalf("GET pendiente: estado=%d cuerpo=%s llamadas=%d", w.Code, w.Body, c.llamadas)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		RutaConsultaEstadoSeleccionOferta+"?expediente_ref="+url.QueryEscape(ref)+"&expediente_ref="+url.QueryEscape(ref), nil))
	if w.Code != http.StatusBadRequest || c.llamadas != 1 {
		t.Fatalf("parámetro duplicado: estado=%d llamadas=%d", w.Code, c.llamadas)
	}
}

func TestConsultaEstadoSeleccionOfertaSingularPendiente(t *testing.T) {
	const ref = "expediente:ct:prueba"
	c := &consultorEstadoSeleccionOfertaHTTPPrueba{resultado: application.DatosEstadoSeleccionOfertaParaAdaptador{
		Estado: "pendiente", TipoSeleccion: "llamamiento", ExpedienteRef: ref,
		VersionEsperada: 6, ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11",
		Efecto: "solicitar_llamamiento", VentanaAbierta: true,
	}}
	h, err := NuevoManejadorConsultaEstadoSeleccionOferta(c)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		RutaConsultaEstadoSeleccionOferta+"?expediente_ref="+url.QueryEscape(ref), nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"tipo_seleccion":"llamamiento"`) ||
		!strings.Contains(w.Body.String(), `"efecto":"solicitar_llamamiento"`) ||
		!strings.Contains(w.Body.String(), `"clave_idempotencia":"018f47a2-6b31-4c80-8a95-4d2e707c5a11"`) ||
		strings.Contains(w.Body.String(), `"oferta"`) || strings.Contains(w.Body.String(), `"descarte"`) {
		t.Fatalf("singular pendiente no recuperado: %d %s", w.Code, w.Body)
	}
}

func TestConsultaEstadoSeleccionOfertaNoConvierteConflictoEnAusencia(t *testing.T) {
	const ref = "expediente:ct:prueba"
	c := &consultorEstadoSeleccionOfertaHTTPPrueba{err: errors.Join(
		application.ErrSeleccionOfertaConcurrente, ports.ErrVinculoOfertaBolsaCTInvalido)}
	h, err := NuevoManejadorConsultaEstadoSeleccionOferta(c)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		RutaConsultaEstadoSeleccionOferta+"?expediente_ref="+url.QueryEscape(ref), nil))
	if w.Code != http.StatusConflict || strings.Contains(w.Body.String(), `"estado":"ausente"`) || c.llamadas != 1 {
		t.Fatalf("conflicto disfrazado: estado=%d cuerpo=%s llamadas=%d", w.Code, w.Body, c.llamadas)
	}
}

func TestConsultaEstadoSeleccionOfertaTerminalYAusenteConservanRecibos(t *testing.T) {
	const ref = "expediente:http:001"
	base := application.DatosEstadoSeleccionOfertaParaAdaptador{ExpedienteRef: ref,
		VersionEsperada: 6, ClaveIdempotencia: claveSeleccionLlamamientoHTTPPrueba}
	for _, caso := range []struct {
		nombre string
		datos  application.DatosEstadoSeleccionOfertaParaAdaptador
		campo  string
	}{
		{"confirmada", func() application.DatosEstadoSeleccionOfertaParaAdaptador {
			d := base
			d.Estado = "confirmada"
			d.Oferta = reciboSeleccionOfertaHTTPPrueba().Oferta
			return d
		}(), `"recibo_vinculo_ref":"recibo:ct:oferta:001"`},
		{"descartada", func() application.DatosEstadoSeleccionOfertaParaAdaptador {
			d := base
			d.Estado = "descartada"
			d.Descarte = reciboDescarteOfertaHTTPPrueba().Descarte
			return d
		}(), `"recibo_descarte_ref":"recibo:descarte:001"`},
		{"ausente", application.DatosEstadoSeleccionOfertaParaAdaptador{Estado: "ausente", ExpedienteRef: ref}, `"estado":"ausente"`},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := &consultorEstadoSeleccionOfertaHTTPPrueba{resultado: caso.datos}
			h, err := NuevoManejadorConsultaEstadoSeleccionOferta(c)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
				RutaConsultaEstadoSeleccionOferta+"?expediente_ref="+url.QueryEscape(ref), nil))
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), caso.campo) {
				t.Fatalf("terminal no recuperado: %d %s", w.Code, w.Body)
			}
			if caso.nombre == "ausente" && strings.Contains(w.Body.String(), "clave_idempotencia") {
				t.Fatalf("ausencia expuso UUID: %s", w.Body)
			}
		})
	}
}
