package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type selectorDescargaDenegadaPrueba struct{ llamadas int }

func (s *selectorDescargaDenegadaPrueba) SeleccionarConsultaDescarga(context.Context, consultafirmasv2.SolicitudDescarga) (consultafirmasv2.SeleccionDescarga, error) {
	s.llamadas++
	return consultafirmasv2.SeleccionDescarga{}, ports.ErrFirmaDocumentoDenegada
}

func TestHTTPDescargaDenegadaExigeAcuseValidoAntesDeResponder(t *testing.T) {
	for _, caso := range []struct {
		nombre, fallo string
		estado        int
	}{
		{"acuse_valido", "", http.StatusNotFound},
		{"registro_indisponible", "registro", http.StatusServiceUnavailable},
		{"acuse_ajeno", "acuse", http.StatusServiceUnavailable},
		{"orden_indisponible", "orden", http.StatusServiceUnavailable},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			selector := &selectorDescargaDenegadaPrueba{}
			consulta := &consultaHTTPDescarga{}
			pdf := &pdfHTTPDescarga{}
			servicio, err := consultafirmasv2.NuevaDescarga(selector, consulta, pdf)
			if err != nil {
				t.Fatal(err)
			}
			intentos := &intentosConsultaV2Prueba{t: t, fallo: caso.fallo == "registro", acuseAjeno: caso.fallo == "acuse"}
			fabrica := &fabricaConsultaV2Prueba{t: t, registro: &registroConsultaV2Prueba{}, fallo: caso.fallo == "orden"}
			h, err := NuevoManejadorDescargaFirmaR5V2(servicio, intentos, fabrica, &garantiaHTTPDescarga{valor: "sustancial"})
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionHTTPDescargaPrueba(t, cuerpoDescargaFirmaPrueba))
			esperados := 1
			if caso.fallo == "orden" {
				esperados = 0
			}
			if w.Code != caso.estado || selector.llamadas != 1 || consulta.llamadas != 0 || pdf.llamadas != 0 ||
				fabrica.llamadas != 1 || fabrica.recursoSolicitado != "expediente:prueba" || intentos.llamadas != esperados {
				t.Fatalf("denegación sin auditoría cerrada: HTTP %d, selector %d, consulta %d, PDF %d, orden %d, apunte %d",
					w.Code, selector.llamadas, consulta.llamadas, pdf.llamadas, fabrica.llamadas, intentos.llamadas)
			}
			var respuesta map[string]any
			if json.Unmarshal(w.Body.Bytes(), &respuesta) != nil || respuesta["data"] != nil ||
				!strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
				t.Fatal("la denegación expone datos o permite almacenarla")
			}
		})
	}
}
