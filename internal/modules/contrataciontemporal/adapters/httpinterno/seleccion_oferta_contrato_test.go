package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func reciboSeleccionOfertaHTTPPrueba() application.DatosReciboSeleccionLlamamientoParaAdaptador {
	return application.DatosReciboSeleccionLlamamientoParaAdaptador{Oferta: &application.DatosReciboSeleccionOfertaParaAdaptador{
		Esquema: ports.EsquemaReciboSeleccionOfertaCTV1, Estado: ports.EstadoReciboSeleccionOfertaCTV1,
		OrganizacionRef: "organizacion:http:001", ExpedienteRef: "expediente:http:001",
		BolsaRef: "bolsa:http:001", OfertaRef: "oferta:" + strings.Repeat("a", 64),
		ReciboOfertaRef:  "recibo:oferta:" + strings.Repeat("b", 64),
		ReciboVinculoRef: "recibo:ct:oferta:001", NumeroPlazas: 2,
		PublicadaEn:  time.Date(2026, 10, 15, 18, 1, 2, 123456000, time.UTC),
		VenceAntesDe: time.Date(2026, 10, 17, 18, 1, 2, 123456000, time.UTC),
	}}
}

func TestManejadorSeleccionLlamamientoPublicaReciboOfertaMinimo(t *testing.T) {
	t.Parallel()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{recibo: reciboSeleccionOfertaHTTPPrueba()}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionSeleccionLlamamientoHTTPPrueba())
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
	esperado := `{"data":{"esquema":"` + ports.EsquemaReciboSeleccionOfertaCTV1 +
		`","estado":"confirmado","organizacion_ref":"organizacion:http:001",` +
		`"expediente_ref":"expediente:http:001","bolsa_ref":"bolsa:http:001",` +
		`"oferta_ref":"oferta:` + strings.Repeat("a", 64) +
		`","recibo_oferta_ref":"recibo:oferta:` + strings.Repeat("b", 64) +
		`","recibo_vinculo_ref":"recibo:ct:oferta:001","numero_plazas":2,` +
		`"publicada_en":"2026-10-15T18:01:02.123456Z","vence_antes_de":"2026-10-17T18:01:02.123456Z"}}`
	if respuesta.Body.String() != esperado {
		t.Fatalf("salida no mínima o distinta: %s", respuesta.Body)
	}
	entrada, existe := ejecutor.ultima()
	if !existe || entrada.ExpedienteRef != "expediente:http:001" || entrada.VersionEsperada != 6 ||
		entrada.ClaveIdempotencia != claveSeleccionLlamamientoHTTPPrueba || ejecutor.total() != 1 {
		t.Fatalf("entrada distinta o delegación repetida: %#v, total=%d", entrada, ejecutor.total())
	}
}

func TestManejadorSeleccionLlamamientoRechazaOfertaNoConfiableODual(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		mutar  func(*application.DatosReciboSeleccionLlamamientoParaAdaptador)
	}{
		{"esquema", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) { r.Oferta.Esquema = "otro" }},
		{"estado", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) { r.Oferta.Estado = "pendiente" }},
		{"expediente ajeno", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.Oferta.ExpedienteRef = "expediente:http:otro"
		}},
		{"organizacion ausente", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) { r.Oferta.OrganizacionRef = "" }},
		{"oferta no autentica", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) { r.Oferta.OfertaRef = "oferta:ajena" }},
		{"recibo Bolsa no autentico", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.Oferta.ReciboOfertaRef = "recibo:oferta:ajeno"
		}},
		{"recibo vinculo ausente", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) { r.Oferta.ReciboVinculoRef = "" }},
		{"una plaza", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) { r.Oferta.NumeroPlazas = 1 }},
		{"demasiadas plazas", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) { r.Oferta.NumeroPlazas = 101 }},
		{"fecha sin UTC canonico", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.Oferta.PublicadaEn = r.Oferta.PublicadaEn.In(time.FixedZone("+1", 3600))
		}},
		{"submicrosegundo", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.Oferta.PublicadaEn = r.Oferta.PublicadaEn.Add(time.Nanosecond)
		}},
		{"vencimiento anterior", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.Oferta.VenceAntesDe = r.Oferta.PublicadaEn
		}},
		{"recibo singular", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.ReciboRef = "recibo:llamamiento:ajeno"
		}},
		{"llamamiento singular", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.LlamamientoRef = "llamamiento:ajeno"
		}},
		{"version singular", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) { r.VersionLlamamiento = 1 }},
		{"fecha singular", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.ConfirmadaEn = r.Oferta.PublicadaEn
		}},
		{"politica singular", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			v := false
			r.PoliticaProvisional = &v
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			recibo := reciboSeleccionOfertaHTTPPrueba()
			caso.mutar(&recibo)
			ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{recibo: recibo}
			respuesta := httptest.NewRecorder()
			nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor).ServeHTTP(respuesta, nuevaPeticionSeleccionLlamamientoHTTPPrueba())
			if respuesta.Code != http.StatusBadGateway || !strings.Contains(respuesta.Body.String(), `"resultado_no_confiable"`) ||
				strings.Contains(respuesta.Body.String(), "oferta_ref") || ejecutor.total() != 1 {
				t.Fatalf("salida malformada no bloqueada: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestManejadorSeleccionLlamamientoNoAceptaNumeroPlazasDesdeHTTP(t *testing.T) {
	t.Parallel()
	cuerpo := `{"expediente_ref":"expediente:http:001","version_esperada":6,"clave_idempotencia":"` +
		claveSeleccionLlamamientoHTTPPrueba + `","numero_plazas":2}`
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{recibo: reciboSeleccionOfertaHTTPPrueba()}
	respuesta := httptest.NewRecorder()
	nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor).ServeHTTP(respuesta, peticionSeleccionLlamamientoConCuerpo(cuerpo))
	if respuesta.Code != http.StatusBadRequest || ejecutor.total() != 0 {
		t.Fatalf("número de plazas HTTP alcanzó negocio: estado=%d llamadas=%d", respuesta.Code, ejecutor.total())
	}
}

func TestManejadorSeleccionLlamamientoClasificaFallosPropiosDeOferta(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		fallo  error
		estado int
		codigo string
	}{
		{"colision", application.ErrClaveSeleccionOfertaEnColision, http.StatusConflict, "conflicto_no_reintentable"},
		{"concurrente", application.ErrSeleccionOfertaConcurrente, http.StatusConflict, "conflicto_estado"},
		{"solicitud", application.ErrSolicitudSeleccionOfertaInvalida, http.StatusUnprocessableEntity, "contenido_no_valido"},
		{"respuesta ajena", application.ErrResultadoSeleccionOfertaNoConfiable, http.StatusBadGateway, "resultado_no_confiable"},
		{"indeterminada con plazo", errors.Join(application.ErrSeleccionOfertaIndeterminada, context.DeadlineExceeded), http.StatusServiceUnavailable, "seleccion_oferta_indeterminada"},
		{"no disponible", application.ErrSeleccionOfertaNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"servicio invalido", application.ErrServicioSeleccionOfertaInvalido, http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"plazo compartido", context.DeadlineExceeded, http.StatusGatewayTimeout, "plazo_agotado"},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{err: caso.fallo}
			respuesta := httptest.NewRecorder()
			nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor).ServeHTTP(respuesta, nuevaPeticionSeleccionLlamamientoHTTPPrueba())
			if respuesta.Code != caso.estado || !strings.Contains(respuesta.Body.String(), `"codigo":"`+caso.codigo+`"`) ||
				ejecutor.total() != 1 {
				t.Fatalf("clasificación incorrecta: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}
