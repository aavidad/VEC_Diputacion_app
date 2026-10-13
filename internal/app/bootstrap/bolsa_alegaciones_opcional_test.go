package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

type autoridadInscripcionOpcionalPrueba struct{ llamadas int }

func (a *autoridadInscripcionOpcionalPrueba) CapturarLectura(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, string, string, inscripcion.Filtro) (inscripcion.CapturaLectura, error) {
	a.llamadas++
	return inscripcion.CapturaLectura{}, nil
}
func (a *autoridadInscripcionOpcionalPrueba) AutorizarEscritura(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, string, string, []byte) (AutorizacionEscrituraInscripcionBolsa, error) {
	a.llamadas++
	return AutorizacionEscrituraInscripcionBolsa{}, nil
}

func TestAlegacionesAusentesConservanInscripcionYCierranAmbasAcciones(t *testing.T) {
	delegado := &autoridadInscripcionOpcionalPrueba{}
	cerrada := autoridadInscripcionSinAlegaciones{AutoridadInscripcionBolsa: delegado}
	if _, err := cerrada.CapturarLectura(context.Background(), contextoSeguridadComunDesarrollo{}, AcreditacionSesionInscripcionBolsa{}, inscripcion.AccionObjetosAlegacion, "objetos", inscripcion.Filtro{}); err != inscripcion.ErrNoDisponible {
		t.Fatal("lectura de alegaciones quedó activa")
	}
	if _, err := cerrada.AutorizarEscritura(context.Background(), contextoSeguridadComunDesarrollo{}, AcreditacionSesionInscripcionBolsa{}, inscripcion.AccionPresentarAlegacion, "alegacion", nil); err != inscripcion.ErrNoDisponible {
		t.Fatal("presentación de alegaciones quedó activa")
	}
	if delegado.llamadas != 0 {
		t.Fatal("Alegaciones llegó al PDP")
	}
	if _, err := cerrada.CapturarLectura(context.Background(), contextoSeguridadComunDesarrollo{}, AcreditacionSesionInscripcionBolsa{}, inscripcion.AccionListarPropias, "propias", inscripcion.Filtro{}); err != nil || delegado.llamadas != 1 {
		t.Fatal("lectura de Inscripción no se delegó")
	}
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	h := cerrarRutasAlegacionesExterno(base)
	for _, ruta := range []string{httpinscripcion.RutaAlegacionesPropias, httpinscripcion.RutaAlegacionesPropias + "/objetos"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "servicio_no_disponible") {
			t.Fatalf("%s: %d", ruta, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, httpinscripcion.RutaPropias, nil))
	if w.Code != http.StatusAccepted {
		t.Fatal("Inscripción no conservó su handler")
	}
}
