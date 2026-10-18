package httpinterno

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestComprobacionConservaSuIntentoNominal(t *testing.T) {
	for _, estado := range []int{400, 403, 503, 201} {
		r := &registradorIntentoBorradorDoble{}
		h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(estado)
		}), r, nil)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost,
			"/api/vec/bolsa/bolsas/bolsa:uno/candidatos/participacion:uno/comprobaciones-ofrecimiento", nil))
		if len(r.intentos) != 1 || r.intentos[0].Validar() != nil ||
			r.intentos[0].Accion != ports.AccionIntentoRegistrarComprobacionOfrecimiento ||
			r.intentos[0].ClaseRuta != ports.ClaseRutaComprobacionOfrecimiento {
			t.Fatalf("intento distinto o perdido para %d: %+v", estado, r.intentos)
		}
		i := r.intentos[0]
		i.Accion = ports.AccionIntentoCambiarSituacionParticipacion
		if i.Validar() == nil {
			t.Fatal("la comprobación se etiquetó como cambio de situación")
		}
	}
}

func TestComprobacionAuditaLaConfirmacionIncierta(t *testing.T) {
	r := &registradorIntentoBorradorDoble{}
	h := nuevaAuditoriaBorradorPrueba(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		responderCodigoComprobacion(w, 503, "resultado_no_confirmado")
	}), r, nil)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost,
		"/api/vec/bolsa/bolsas/bolsa:uno/candidatos/participacion:uno/comprobaciones-ofrecimiento", nil))
	if len(r.intentos) != 1 || r.intentos[0].Resultado != ports.ResultadoIntentoIndeterminadoBorradorLlamamiento {
		t.Fatalf("se perdió la incertidumbre: %+v", r.intentos)
	}
}
