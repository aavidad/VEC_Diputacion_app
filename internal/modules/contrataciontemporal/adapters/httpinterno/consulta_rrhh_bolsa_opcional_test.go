package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
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
