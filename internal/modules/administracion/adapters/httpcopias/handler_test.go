package httpcopias

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	app "vec-diputacion-granada/internal/modules/administracion/application/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

type autoridadTest struct {
	err      error
	llamadas int
	op       p.Operacion
	recurso  string
}

func (a *autoridadTest) AutorizarCopias(_ context.Context, _ p.Sesion, op p.Operacion, ref string) error {
	a.llamadas++
	a.op = op
	a.recurso = ref
	return a.err
}

type backendTest struct {
	llamadas int
	consulta int
	err      error
}

func (b *backendTest) Listar(context.Context, p.Sesion, string, int) (p.Pagina, error) {
	b.consulta++
	return p.Pagina{Version: 5, Copias: []p.Copia{}}, b.err
}
func (b *backendTest) Detalle(_ context.Context, _ p.Sesion, ref string) (p.Copia, error) {
	b.consulta++
	return p.Copia{CopiaRef: ref}, b.err
}
func (b *backendTest) Calendario(context.Context, p.Sesion) (p.Configuracion, error) {
	b.consulta++
	return p.Configuracion{}, b.err
}
func (b *backendTest) Retencion(context.Context, p.Sesion) (p.Configuracion, error) {
	b.consulta++
	return p.Configuracion{}, b.err
}
func (b *backendTest) Propuestas(context.Context, p.Sesion) ([]p.Propuesta, error) {
	b.consulta++
	return []p.Propuesta{}, b.err
}
func (b *backendTest) Lanzar(_ context.Context, _ p.Sesion, v p.SolicitudLanzamiento) (p.Recibo, error) {
	b.llamadas++
	if v.VersionEsperada != 5 {
		return p.Recibo{}, p.ErrConflicto
	}
	return p.Recibo{ReciboRef: "recibo_aaaaaaaa", OperacionRef: v.OperacionRef, RecursoRef: "copia_aaaaaaaa", Version: 6, Estado: "reservada", RegistradoEn: time.Now().UTC()}, b.err
}
func (b *backendTest) ConfigurarCalendario(context.Context, p.Sesion, p.SolicitudPolitica) (p.Recibo, error) {
	b.llamadas++
	return p.Recibo{}, b.err
}
func (b *backendTest) ConfigurarRetencion(context.Context, p.Sesion, p.SolicitudPolitica) (p.Recibo, error) {
	b.llamadas++
	return p.Recibo{}, b.err
}
func peticion(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, "https://admin.example.test"+path, strings.NewReader(body))
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	if method == http.MethodPost {
		for k, v := range map[string]string{"Origin": "https://admin.example.test", "Sec-Fetch-Site": "same-origin", "Sec-Fetch-Mode": "cors", "Sec-Fetch-Dest": "empty", "Content-Type": "application/json"} {
			r.Header.Set(k, v)
		}
	}
	return r
}
func preparar(t *testing.T) (*Handler, *autoridadTest, *backendTest) {
	t.Helper()
	ses := sesionPrueba(t)
	a, b := &autoridadTest{}, &backendTest{}
	h, e := Nuevo("https://admin.example.test", func(context.Context, *http.Request) (p.Sesion, error) { return ses, nil }, func(context.Context, Denegacion) error { return nil }, &app.Servicio{Autoridad: a, Lecturas: b, Cambios: b})
	if e != nil {
		t.Fatal(e)
	}
	return h, a, b
}

const lanzamiento = `{"operacion_ref":"operacion_aaaaaaaa","version_esperada":5,"tipo":"completa"}`

func TestFronteraDeniegaSinIdentidadNiPermiso(t *testing.T) {
	for _, c := range []struct {
		name   string
		modify func(*http.Request)
		err    error
		status int
	}{
		{"sin TLS", func(r *http.Request) { r.TLS = nil }, nil, 401},
		{"sin cadena", func(r *http.Request) { r.TLS.VerifiedChains = nil }, nil, 401},
		{"identidad libre", func(r *http.Request) { r.Header.Set("X-Remote-User", "admin") }, nil, 401},
		{"cookie", func(r *http.Request) { r.Header.Set("Cookie", "session=x") }, nil, 401},
		{"origen ajeno", func(r *http.Request) { r.Header.Set("Origin", "https://evil.example.test") }, nil, 403},
		{"sin permiso", func(*http.Request) {}, p.ErrDenegado, 403},
		{"autoridad caida", func(*http.Request) {}, p.ErrNoDisponible, 503},
	} {
		t.Run(c.name, func(t *testing.T) {
			h, a, b := preparar(t)
			a.err = c.err
			r := peticion("POST", PrefijoV1+"/lanzamientos", lanzamiento)
			c.modify(r)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != c.status || b.llamadas != 0 {
				t.Fatalf("status %d effects %d", w.Code, b.llamadas)
			}
			if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") == "" {
				t.Fatal("private response headers")
			}
		})
	}
}
func TestJSONEstrictoNoAceptaAutoridadNiDuplicados(t *testing.T) {
	bodies := []string{`{"operacion_ref":"operacion_aaaaaaaa","tipo":"completa"}`, `{"operacion_ref":"operacion_aaaaaaaa","version_esperada":5,"tipo":"completa","actor":"admin"}`, `{"operacion_ref":"operacion_aaaaaaaa","version_esperada":5,"tipo":"completa","permisos":["*"]}`, `{"operacion_ref":"operacion_aaaaaaaa","version_esperada":5,"tipo":"completa","version_esperada":4}`, lanzamiento + ` {}`, `null`, `{"operacion_ref":null}`, `{"operacion_ref":"operacion_aaaaaaaa","version_esperada":5,"tipo":"incremental"}`, `{"operacion_ref":"../../etc/passwd","version_esperada":5,"tipo":"completa"}`}
	for _, body := range bodies {
		h, a, b := preparar(t)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion("POST", PrefijoV1+"/lanzamientos", body))
		if w.Code != 400 || b.llamadas != 0 || a.llamadas != 0 {
			t.Fatalf("accepted %q: status=%d effects=%d", body, w.Code, b.llamadas)
		}
	}
	h, _, b := preparar(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion("POST", PrefijoV1+"/lanzamientos", strings.Repeat("a", maxCuerpo+1)))
	if w.Code != 413 || b.llamadas != 0 {
		t.Fatal("unbounded input")
	}
}
func TestLanzamientoExactoCASYDependenciaAusente(t *testing.T) {
	h, a, b := preparar(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion("POST", PrefijoV1+"/lanzamientos", lanzamiento))
	if w.Code != 200 || b.llamadas != 1 || a.op != p.Lanzar || a.recurso != "copias" || !strings.Contains(w.Body.String(), `"version":6`) {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion("POST", PrefijoV1+"/lanzamientos", strings.Replace(lanzamiento, ":5", ":4", 1)))
	if w.Code != 409 {
		t.Fatal("CAS ignored")
	}
	h.servicio.Cambios = nil
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion("POST", PrefijoV1+"/lanzamientos", lanzamiento))
	if w.Code != 503 {
		t.Fatal("missing backend returned success")
	}
}
func TestSesionRevalidadaYAudiFallaCerrado(t *testing.T) {
	h, _, b := preparar(t)
	h.resolver = func(context.Context, *http.Request) (p.Sesion, error) { return p.Sesion{}, p.ErrAutenticacion }
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion("GET", PrefijoV1, ""))
	if w.Code != 401 || b.consulta != 0 {
		t.Fatal("missing session read data")
	}
	h.auditor = func(context.Context, Denegacion) error { return errors.New("audit offline") }
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion("GET", PrefijoV1, ""))
	if w.Code != 503 {
		t.Fatal("missing audit not fail closed")
	}
}
func TestLecturaRechazaCuerpoQueryYCancelacion(t *testing.T) {
	for _, r := range []*http.Request{peticion("GET", PrefijoV1, "{}"), peticion("GET", PrefijoV1+"?limite=101", ""), peticion("GET", PrefijoV1+"?actor=admin", "")} {
		h, _, b := preparar(t)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 || b.consulta != 0 {
			t.Fatal("invalid read reached backend")
		}
	}
	h, _, b := preparar(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion("GET", PrefijoV1, "").WithContext(ctx))
	if w.Code != 503 || b.consulta != 0 {
		t.Fatal("cancelled request reached backend")
	}
}
