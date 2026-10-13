package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestPreparadorCarteraNoTomaPersonaDeURLNiSinSesion(t *testing.T) {
	a := &autoridadPreferenciasUsuariosDesarrollo{ruta: meritoshttp.RutaCartera,
		superficie: vec.SuperficieAutenticacionExternaPersonalV1}
	p := preparadorCarteraExterna{autoridad: a, ruta: meritoshttp.RutaCartera}
	for _, ruta := range []string{meritoshttp.RutaCartera + "?persona_ref=per_ajena", meritoshttp.RutaCartera + "?filtro=todos"} {
		_, err := p.PrepararSolicitudCartera(httptest.NewRequest(http.MethodGet, ruta, nil))
		if !errors.Is(err, meritosports.ErrCarteraSinSesion) {
			t.Fatalf("petición sin sesión %s: %v", ruta, err)
		}
	}
	_, err := p.PrepararSolicitudCartera(httptest.NewRequest(http.MethodGet,
		meritoshttp.RutaRetiradasCartera, nil))
	if !errors.Is(err, meritosports.ErrCarteraDenegada) {
		t.Fatalf("ruta hermana en autoridad ajena: %v", err)
	}
}
