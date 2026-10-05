package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFichaPropiaNegociaHistoriasIndependientesSinConcederAcceso(t *testing.T) {
	for _, prefer := range []string{"", PreferenciaHistoriaServiciosFichaPropia, PreferenciaHistoriaRelacionesFichaPropia, PreferenciaHistoriaServiciosFichaPropia + ", " + PreferenciaHistoriaRelacionesFichaPropia, "otro"} {
		t.Run(prefer, func(t *testing.T) {
			m := manejadorFichaPropiaPrueba(t, &consultaFichaPropiaHTTP{}, &registroFichaPropiaHTTP{})
			m.historiaDisponible = false
			m.historiaRelacionesDisponible = true
			r := httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil)
			if prefer != "" {
				r.Header.Set("Prefer", prefer)
			}
			w := httptest.NewRecorder()
			m.ServeHTTP(w, r)
			var sobre struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &sobre) != nil {
				t.Fatal("consulta no disponible")
			}
			valor, existe := sobre.Data["historia_relaciones_disponible"]
			esperada := prefer == PreferenciaHistoriaRelacionesFichaPropia || prefer == PreferenciaHistoriaServiciosFichaPropia+", "+PreferenciaHistoriaRelacionesFichaPropia
			if existe != esperada || (existe && string(valor) != "true") {
				t.Fatal("historia de relaciones sin negociación o montaje propio")
			}
			if valor, existe := sobre.Data["historia_servicios_disponible"]; existe && string(valor) != "false" {
				t.Fatal("se prestó disponibilidad de servicios")
			}
			if _, existe := sobre.Data["exportacion_servicios_disponible"]; existe {
				t.Fatal("se habilitó exportación sin negociación")
			}
		})
	}
}
