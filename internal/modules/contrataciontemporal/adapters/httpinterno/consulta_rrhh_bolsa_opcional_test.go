package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaDetalleBolsaSoloConSQLActivado(t *testing.T) {
	for _, activo := range []bool{false, true} {
		t.Run(map[bool]string{false: "HZ15_sin_CT201", true: "CT201_activo"}[activo], func(t *testing.T) {
			consultor := &consultorDetalleRRHHPrueba{detalle: detalleRRHHPrueba()}
			var pidioResultado bool
			consultor.alConsultar = func(ctx context.Context) {
				pidioResultado = ports.ResultadoBolsaRRHHSolicitado(ctx)
			}
			h, err := NuevoManejadorConsultaDetalleRRHH(consultor)
			if err != nil {
				t.Fatal(err)
			}
			if activo {
				h, err = ConfigurarResultadoBolsaConsultaDetalleRRHH(h)
				if err != nil {
					t.Fatal(err)
				}
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, cuerpoDetalleRRHHPrueba()))
			if w.Code != http.StatusOK || consultor.llamadas != 1 || pidioResultado != activo {
				t.Fatalf("estado=%d llamadas=%d resultado_bolsa=%t activo=%t", w.Code, consultor.llamadas, pidioResultado, activo)
			}
		})
	}
}

func TestConsultaDetalleBolsaPaginaExigeMontajeYPropagaCursor(t *testing.T) {
	cursor := "2026-10-09T11:00:00.123456Z#llamamiento:" + strings.Repeat("a", 64)
	cuerpo := strings.TrimSuffix(string(cuerpoDetalleRRHHPrueba()), "}") + `,"resultado_bolsa_cursor":"` + cursor + `"}`
	consultor := &consultorDetalleRRHHPrueba{detalle: detalleRRHHPrueba()}
	var recibido string
	consultor.alConsultar = func(ctx context.Context) { recibido = ports.CursorResultadoBolsaRRHH(ctx) }
	h, err := NuevoManejadorConsultaDetalleRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	sinMontaje := httptest.NewRecorder()
	h.ServeHTTP(sinMontaje, nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, cuerpo))
	if sinMontaje.Code != http.StatusUnprocessableEntity || consultor.llamadas != 0 {
		t.Fatalf("cursor sin montaje: estado=%d consultas=%d", sinMontaje.Code, consultor.llamadas)
	}
	h, err = ConfigurarResultadoBolsaConsultaDetalleRRHH(h)
	if err != nil {
		t.Fatal(err)
	}
	conMontaje := httptest.NewRecorder()
	h.ServeHTTP(conMontaje, nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, cuerpo))
	if conMontaje.Code != http.StatusOK || consultor.llamadas != 1 || recibido != cursor {
		t.Fatalf("cursor no propagado: estado=%d consultas=%d", conMontaje.Code, consultor.llamadas)
	}
}
