package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestCarteraNoConfiguradaNoDesmontaAreaPersonal(t *testing.T) {
	preferencias := &autoridadPreferenciasUsuariosDesarrollo{
		base:        &autoridadRutasDietasDesarrollo{},
		ruta:        usuarioshttp.RutaMisPreferenciasAreaPersonal,
		manejador:   http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		registrador: &registradorDenegacionPreferenciasPrueba{},
	}
	h, err := nuevaAPIPersonalPortalExterno(&resolvedorIdentidadDesarrollo{},
		vecports.EmisorIncidenciasTecnicasNulo{}, preferencias)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{meritoshttp.RutaCartera + "?filtro=todos&lang=es", meritoshttp.RutaRetiradasCartera} {
		metodo := http.MethodGet
		if strings.HasPrefix(ruta, meritoshttp.RutaRetiradasCartera) {
			metodo = http.MethodPost
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(metodo, ruta, nil))
		if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "meritos.error.cartera_no_disponible") {
			t.Fatalf("M4 sin configurar: %s %d %s", ruta, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, usuarioshttp.RutaMisPreferenciasAreaPersonal, nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("M4 ausente alteró Usuarios: %d %s", w.Code, w.Body.String())
	}
}
