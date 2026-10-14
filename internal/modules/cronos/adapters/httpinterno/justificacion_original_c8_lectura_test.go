package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type capturaLecturaOriginalC8Prueba struct{}

func (capturaLecturaOriginalC8Prueba) CapturarLecturaOriginalPropia(context.Context, ports.OrdenOriginalC8, string) (ports.CapturaLecturaOriginalC8, error) {
	return ports.CapturaLecturaOriginalC8{}, nil
}
func (capturaLecturaOriginalC8Prueba) CapturarLecturaOriginalRRHH(context.Context, ports.OrdenOriginalC8, string) (ports.CapturaLecturaOriginalC8, error) {
	return ports.CapturaLecturaOriginalC8{}, nil
}
func (capturaLecturaOriginalC8Prueba) CapturarBandejaOriginal(context.Context, ports.OrdenOriginalC8) (ports.CapturaLecturaOriginalC8, error) {
	return ports.CapturaLecturaOriginalC8{}, nil
}

type lectorHTTPOriginalC8Prueba struct{ consultas int }

func (l *lectorHTTPOriginalC8Prueba) ConsultarOriginal(_ context.Context, _ ports.CapturaLecturaOriginalC8, ref string) (ports.LecturaOriginalC8, error) {
	l.consultas++
	return ports.LecturaOriginalC8{SolicitudRef: ref, EstadoPermiso: domain.EstadoPermisoConcedido, VersionPermiso: 3,
		PendienteJustificar: true, Historia: []ports.HechoOriginalC8{}}, nil
}
func (l *lectorHTTPOriginalC8Prueba) ConsultarBandejaOriginal(context.Context, ports.CapturaLecturaOriginalC8) (ports.BandejaOriginalC8, error) {
	l.consultas++
	return ports.BandejaOriginalC8{Pendientes: []ports.PendienteRevisionOriginalC8{}}, nil
}

func TestOriginalC8HTTPGetRechazaConsultaAmbiguaYNoRepitePOST(t *testing.T) {
	r := &resolverOriginalC8Prueba{}
	l := &lectorHTTPOriginalC8Prueba{}
	m, err := NuevoManejadorLecturaOriginalC8(r, capturaLecturaOriginalC8Prueba{}, l)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{
		RutaOriginalC8Propio + "?solicitud_ref=permiso%3Acronos%3Asolicitud%3Aabcdefgh&solicitud_ref=otra",
		RutaOriginalC8BandejaRRHH + "?actor_ref=per_ajena",
	} {
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusBadRequest || r.llamadas != 0 || l.consultas != 0 {
			t.Fatalf("aceptó consulta ambigua: %d", w.Code)
		}
	}
	ref := "permiso:cronos:solicitud:abcdefgh"
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOriginalC8Propio+"?solicitud_ref="+ref, nil))
	if w.Code != http.StatusOK || r.llamadas != 1 || l.consultas != 1 {
		t.Fatalf("consulta propia: %d", w.Code)
	}
	w = httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOriginalC8BandejaRRHH, nil))
	if w.Code != http.StatusOK || l.consultas != 2 {
		t.Fatalf("bandeja: %d", w.Code)
	}
}
