package administracion

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

type autoridadCopiasPrueba struct {
	recurso   string
	operacion string
	recursos  map[string]bool
	err       error
	llamadas  int
}

func (a *autoridadCopiasPrueba) AutorizarCopias(_ context.Context, _ p.Sesion, op p.Operacion, ref string) error {
	a.llamadas++
	if op != p.Consultar || ref != "copias" && ref != a.recurso && ref != a.operacion && !a.recursos[ref] {
		return p.ErrDenegado
	}
	return a.err
}

type auditorCopiasPrueba struct {
	recursos     []string
	falloRecurso string
	err          error
	llamadas     int
}

func (a *auditorCopiasPrueba) RegistrarLecturaCopias(_ context.Context, _ p.Sesion, _ string, recurso string, _ string) error {
	a.llamadas++
	a.recursos = append(a.recursos, recurso)
	if recurso == a.falloRecurso {
		return p.ErrNoDisponible
	}
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

// El documento y todos los recursos se leen del árbol real; no hay CSS ni HTML
// de fixture que pueda ocultar un fallo en las clases o versiones del montaje.
func TestGrafoDocumentoRealCopiasPasaFronteraDeRecursos(t *testing.T) {
	ses := sesionPrueba(t)
	deps := DependenciasCopias{Autoridad: &autoridadCopiasPrueba{}, ResolverSesion: func(context.Context, *http.Request) (p.Sesion, error) { return ses, nil }, AuditorFrontera: func(context.Context, adapter.Denegacion) error { return nil }}
	h := nuevaSuperficieCopias(Configuracion{Host: "admin.example.test"}, deps, os.DirFS("../../../web/static"), &auditorCopiasPrueba{}, http.NotFoundHandler())
	recursos := []string{"https://admin.example.test/admin/copias/"}
	visitadas := map[string]bool{}
	htmlLinks := regexp.MustCompile(`(?:href|src)="([^"]+)"`)
	importsJS := regexp.MustCompile(`from\s+["']([^"']+)["']`)
	importsCSS := regexp.MustCompile(`@import\s+url\(["']([^"']+)["']\)`)
	for len(recursos) > 0 {
		recurso := recursos[0]
		recursos = recursos[1:]
		if visitadas[recurso] {
			continue
		}
		visitadas[recurso] = true
		req := httptest.NewRequest("GET", recurso, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("grafo real bloqueado %s: %d", req.URL.RequestURI(), w.Code)
		}
		contenido := w.Body.String()
		if req.URL.Path == "/admin/copias/" && (!strings.Contains(contenido, `body class="portal-empleado-app"`) || !strings.Contains(contenido, `class="portal-superficie contenido-portal"`)) {
			t.Fatal("clases comunes ausentes")
		}
		var referencias [][]string
		switch {
		case req.URL.Path == "/admin/copias/":
			referencias = htmlLinks.FindAllStringSubmatch(contenido, -1)
		case strings.HasSuffix(req.URL.Path, ".js"):
			referencias = importsJS.FindAllStringSubmatch(contenido, -1)
		case strings.HasSuffix(req.URL.Path, ".css"):
			referencias = importsCSS.FindAllStringSubmatch(contenido, -1)
		}
		for _, r := range referencias {
			ref, err := url.Parse(r[1])
			if err != nil {
				t.Fatal(err)
			}
			recursos = append(recursos, req.URL.ResolveReference(ref).String())
		}
	}
	if len(visitadas) < 14 {
		t.Fatalf("grafo incompleto: %d", len(visitadas))
	}
}
