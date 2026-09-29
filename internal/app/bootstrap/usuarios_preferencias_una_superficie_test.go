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
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestProcesoInternoSeparadoSoloComponeLaSuperficieCorporativa(t *testing.T) {
	// Solo el combinado compone aquí el Área personal; el externo tiene su
	// propia composición y un valor no válido nunca la activa.
	for portal, externa := range map[string]bool{"": true, "interno": false, "externo": false, "otro": false} {
		cfg := config.Config{PortalProceso: portal}
		if got := superficieExternaUsuariosEnProceso(cfg); got != externa {
			t.Fatalf("portal %q: superficie externa %t, se esperaba %t", portal, got, externa)
		}
		// Las comprobaciones previas de preferencias, correos e imagen solo
		// recorren estas superficies: en el interno nunca la del Área personal.
		superficies := superficiesUsuariosEnProceso(cfg)
		if superficies[0] != core.SuperficieAutenticacionInternaCorporativaV1 || (len(superficies) == 2) != externa ||
			(externa && superficies[1] != core.SuperficieAutenticacionExternaPersonalV1) {
			t.Fatalf("portal %q: superficies %v", portal, superficies)
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
	// Una ruta del Área personal tampoco toca la superficie ausente: pasa al
	// enrutador, que no la tiene registrada en este proceso.
	for _, ruta := range []string{usuarioshttp.RutaMisPreferenciasAreaPersonal, usuarioshttp.RutaMisCorreosAreaPersonal, usuarioshttp.RutaMiImagenAreaPersonal} {
		w = httptest.NewRecorder()
		siguiente.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("%s interceptada por la superficie corporativa: %d", ruta, w.Code)
		}
	}
	c.cerrar()
}
