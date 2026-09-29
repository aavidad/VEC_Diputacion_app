package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

func TestProcesoInternoSeparadoSoloComponeLaSuperficieCorporativa(t *testing.T) {
	for portal, externa := range map[string]bool{"": true, "interno": false} {
		if got := superficieExternaUsuariosEnProceso(config.Config{PortalProceso: portal}); got != externa {
			t.Fatalf("portal %q: superficie externa %t, se esperaba %t", portal, got, externa)
		}
	}
}

// Con solo la superficie interna compuesta, las rutas del Área personal no se
// registran, no se autorizan y no provocan un acceso a una autoridad nula.
func TestComposicionUsuariosSinSuperficieExterna(t *testing.T) {
	manejador := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	interna := &autoridadPreferenciasUsuariosDesarrollo{
		ruta: usuarioshttp.RutaMisPreferencias, manejador: manejador,
		correos: &autoridadPreferenciasUsuariosDesarrollo{ruta: usuarioshttp.RutaMisCorreos, manejador: manejador},
		imagen:  &autoridadPreferenciasUsuariosDesarrollo{ruta: usuarioshttp.RutaMiImagen, manejador: manejador},
	}
	c := &composicionPreferenciasUsuarios{interna: interna}
	rutas := rutaUsuariosPreferencias(c)
	if len(rutas) != 3 {
		t.Fatalf("se esperaban solo las tres rutas internas: %+v", rutas)
	}
	for _, r := range rutas {
		switch r.Ruta {
		case usuarioshttp.RutaMisPreferencias, usuarioshttp.RutaMisCorreos, usuarioshttp.RutaMiImagen:
		default:
			t.Fatalf("ruta del Área personal compuesta en el interno: %s", r.Ruta)
		}
	}
	autoridad := autoridadExactasConUsuariosPreferencias{usuarios: c}
	for _, ruta := range []string{usuarioshttp.RutaMisPreferenciasAreaPersonal, usuarioshttp.RutaMisCorreosAreaPersonal, usuarioshttp.RutaMiImagenAreaPersonal} {
		if err := autoridad.AutorizarRutaExacta(context.Background(), ruta); !errors.Is(err, vechttp.ErrAutenticacionRutaExactaRequerida) {
			t.Fatalf("%s autorizada sin superficie externa: %v", ruta, err)
		}
	}
	// El envoltorio deja pasar lo que no es suyo sin tocar la superficie ausente.
	siguiente := c.proteger(manejador)
	w := httptest.NewRecorder()
	siguiente.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/vec/otra", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("el envoltorio altero una ruta ajena: %d", w.Code)
	}
	c.cerrar()
}
