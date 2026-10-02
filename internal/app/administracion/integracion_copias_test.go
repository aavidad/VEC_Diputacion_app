package administracion

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

type autoridadCopiasPrueba struct {
	recurso  string
	err      error
	llamadas int
}

func (a *autoridadCopiasPrueba) AutorizarCopias(_ context.Context, _ p.Sesion, op p.Operacion, ref string) error {
	a.llamadas++
	if op != p.Consultar || ref != "copias" && ref != a.recurso {
		return p.ErrDenegado
	}
	return a.err
}

type auditorCopiasPrueba struct {
	recurso  string
	err      error
	llamadas int
}

func (a *auditorCopiasPrueba) RegistrarLecturaCopias(context.Context, p.Sesion, string, string, string) error {
	a.llamadas++
	return a.err
}
func TestSuperficieCopiasExigeSesionVigenteYPermisoEspecifico(t *testing.T) {
	ses := sesionPrueba(t)
	autoridad := &autoridadCopiasPrueba{}
	auditor := &auditorCopiasPrueba{}
	denegaciones := 0
	falloSesion := error(nil)
	deps := DependenciasCopias{Autoridad: autoridad, ResolverSesion: func(context.Context, *http.Request) (p.Sesion, error) { return ses, falloSesion }, AuditorFrontera: func(context.Context, adapter.Denegacion) error { denegaciones++; return nil }}
	h := nuevaSuperficieCopias(Configuracion{Host: "admin.example.test"}, deps, fstest.MapFS{"admin/copias/index.html": {Data: []byte("<main></main>")}}, auditor, http.NotFoundHandler())
	pedir := func(path string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "https://admin.example.test"+path, nil)
		h.ServeHTTP(w, r)
		if w.Header().Get("Set-Cookie") != "" {
			t.Fatal("cookie")
		}
		return w.Code
	}
	if got := pedir("/admin/copias/"); got != 200 || autoridad.llamadas != 1 || auditor.llamadas != 1 {
		t.Fatalf("montaje: %d", got)
	}
	// Una sesión autenticada/perfil válido no suple la concesión de copias.
	autoridad.err = p.ErrDenegado
	if got := pedir("/admin/copias/"); got != 403 || denegaciones != 1 {
		t.Fatalf("permiso ajeno: %d", got)
	}
	autoridad.err = nil
	falloSesion = p.ErrAutenticacion
	antes := autoridad.llamadas
	if got := pedir("/admin/copias/"); got != 403 || autoridad.llamadas != antes {
		t.Fatalf("sesion revocada: %d", got)
	}
	falloSesion = nil
	auditor.err = errors.New("fallo_sintetico")
	if got := pedir("/admin/copias/"); got != 503 {
		t.Fatalf("sin recibo auditoria: %d", got)
	}
	if got := pedir("/textos/es/otro.json"); got != 404 {
		t.Fatalf("arbol general publicado: %d", got)
	}
	if got := pedir("/admin/copias/../../secreto.json"); got != 404 {
		t.Fatalf("ruta ajena: %d", got)
	}
}
