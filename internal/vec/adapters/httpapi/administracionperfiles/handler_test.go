package administracionperfiles

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type sesionPrueba struct{ llamadas int }

func (s *sesionPrueba) ResolverSesionADMIN(context.Context, *http.Request) (SesionConfiable, error) {
	s.llamadas++
	return SesionConfiable{}, ErrAccesoDenegado
}

type lecturasPrueba struct{ llamadas int }

func (l *lecturasPrueba) Capacidades(context.Context, domain.ContextoActor) (Capacidades, error) {
	l.llamadas++
	return Capacidades{}, nil
}
func (l *lecturasPrueba) BuscarPersonas(context.Context, domain.ContextoActor, string, string) (PaginaPersonas, error) {
	l.llamadas++
	return PaginaPersonas{}, nil
}
func (l *lecturasPrueba) ConsultarPersona(context.Context, domain.ContextoActor, string) (FichaPersona, error) {
	l.llamadas++
	return FichaPersona{}, nil
}
func (l *lecturasPrueba) ListarRoles(context.Context, domain.ContextoActor) (Roles, error) {
	l.llamadas++
	return Roles{}, nil
}
func (l *lecturasPrueba) ListarPropuestas(context.Context, domain.ContextoActor) (PaginaPropuestas, error) {
	l.llamadas++
	return PaginaPropuestas{}, nil
}
func (l *lecturasPrueba) ConsultarPropuesta(context.Context, domain.ContextoActor, string) (Propuesta, error) {
	l.llamadas++
	return Propuesta{}, nil
}
func (l *lecturasPrueba) ConsultarRecibo(context.Context, domain.ContextoActor, string) (domain.ReciboAdministracionPerfiles, error) {
	l.llamadas++
	return domain.ReciboAdministracionPerfiles{}, nil
}

type catalogoPrueba struct{ llamadas int }

func (c *catalogoPrueba) ResolverRolAdministrable(context.Context, string) (ports.RolAdministrable, error) {
	c.llamadas++
	return ports.RolAdministrable{}, nil
}

type actosPrueba struct{ llamadas int }

type auditorPrueba struct {
	llamadas int
	err      error
}

func (a *auditorPrueba) RegistrarDenegacionADMIN(context.Context, string) error {
	a.llamadas++
	return a.err
}

func (a *actosPrueba) AplicarOrdinario(context.Context, domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	a.llamadas++
	return domain.ReciboAdministracionPerfiles{}, nil
}
func (a *actosPrueba) ProponerSensible(context.Context, domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error) {
	a.llamadas++
	return ports.PropuestaAdministracionPerfiles{}, nil
}
func (a *actosPrueba) CerrarPropuestaSensible(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	a.llamadas++
	return ports.CierrePropuestaAdministracionPerfiles{}, nil
}

func TestFronteraADMINRechazaAntesDeResolverSesion(t *testing.T) {
	s := &sesionPrueba{}
	l := &lecturasPrueba{}
	c := &catalogoPrueba{}
	a := &actosPrueba{}
	aud := &auditorPrueba{}
	h, err := NuevoHandler("https://admin.example.test", s, l, c, a, aud)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, metodo, ruta string
		preparar             func(*http.Request)
		estado               int
	}{
		{"sin certificado", http.MethodGet, PrefijoV1 + "/capacidades", func(r *http.Request) { r.TLS = nil }, http.StatusUnauthorized},
		{"sin cadena verificada", http.MethodGet, PrefijoV1 + "/capacidades", func(r *http.Request) { r.TLS.VerifiedChains = nil }, http.StatusUnauthorized},
		{"host distinto", http.MethodGet, PrefijoV1 + "/capacidades", func(r *http.Request) { r.Host = "vec.example.test" }, http.StatusUnauthorized},
		{"authorization", http.MethodGet, PrefijoV1 + "/capacidades", func(r *http.Request) { r.Header.Set("Authorization", "Bearer x") }, http.StatusUnauthorized},
		{"cookie", http.MethodGet, PrefijoV1 + "/capacidades", func(r *http.Request) { r.Header.Set("Cookie", "sesion=x") }, http.StatusUnauthorized},
		{"proxy authorization", http.MethodGet, PrefijoV1 + "/capacidades", func(r *http.Request) { r.Header.Set("Proxy-Authorization", "Basic x") }, http.StatusUnauthorized},
		{"origen cruzado", http.MethodGet, PrefijoV1 + "/capacidades", func(r *http.Request) { r.Header.Set("Origin", "https://vec.example.test") }, http.StatusForbidden},
		{"fetch cruzado", http.MethodGet, PrefijoV1 + "/capacidades", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, http.StatusForbidden},
		{"post sin origin", http.MethodPost, PrefijoV1 + "/actos-ordinarios", func(r *http.Request) {}, http.StatusForbidden},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			r := httptest.NewRequest(caso.metodo, "https://admin.example.test"+caso.ruta, nil)
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			caso.preparar(r)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || s.llamadas != 0 || l.llamadas != 0 || c.llamadas != 0 || a.llamadas != 0 || aud.llamadas == 0 {
				t.Fatalf("estado=%d sesion=%d lecturas=%d catalogo=%d actos=%d", w.Code, s.llamadas, l.llamadas, c.llamadas, a.llamadas)
			}
			if !strings.Contains(w.Body.String(), `"clave_i18n":"api.admin.perfiles.error.`) || w.Header().Get("Set-Cookie") != "" || w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Cache-Control") == "" {
				t.Fatal("respuesta de error fuera del contrato privado")
			}
		})
	}
}

func TestConstructorADMINFallaCerradoSinFuente(t *testing.T) {
	s := &sesionPrueba{}
	l := &lecturasPrueba{}
	c := &catalogoPrueba{}
	a := &actosPrueba{}
	aud := &auditorPrueba{}
	for _, origen := range []string{"", "http://admin.example.test", "https://admin.example.test/ruta", "https://admin.example.test?x=1"} {
		if _, err := NuevoHandler(origen, s, l, c, a, aud); err == nil {
			t.Fatalf("origen aceptado: %q", origen)
		}
	}
	if _, err := NuevoHandler("https://admin.example.test", nil, l, c, a, aud); err == nil {
		t.Fatal("sesión ausente aceptada")
	}
	if _, err := NuevoHandler("https://admin.example.test", s, nil, c, a, aud); err == nil {
		t.Fatal("lecturas ausentes aceptadas")
	}
	if _, err := NuevoHandler("https://admin.example.test", s, l, nil, a, aud); err == nil {
		t.Fatal("catálogo ausente aceptado")
	}
	if _, err := NuevoHandler("https://admin.example.test", s, l, c, nil, aud); err == nil {
		t.Fatal("actos ausentes aceptados")
	}
	var sesionNula *sesionPrueba
	if _, err := NuevoHandler("https://admin.example.test", sesionNula, l, c, a, aud); err == nil {
		t.Fatal("sesión nula tipada aceptada")
	}
	if _, err := NuevoHandler("https://admin.example.test", s, l, c, a, nil); err == nil {
		t.Fatal("auditor ausente aceptado")
	}
}

func TestEscrituraRechazaAutoridadYTextoFueraDelContrato(t *testing.T) {
	for _, cuerpo := range []string{
		`{"operacion_ref":"acto_admin:00000000000000000000000000000001","actor":{"persona_ref":"per_falsa"}}`,
		`{"motivo":{"catalogo_id":"motivos","catalogo_version":1,"catalogo_huella_sha256":"x","entrada_clave":"a","etiqueta":"texto del cliente"}}`,
		`{} {}`,
	} {
		r := httptest.NewRequest(http.MethodPost, "https://admin.example.test"+PrefijoV1+"/actos-ordinarios", strings.NewReader(cuerpo))
		w := httptest.NewRecorder()
		var dto SolicitudActo
		if decodificar(w, r, &dto) || w.Code != http.StatusBadRequest {
			t.Fatalf("contenido fuera de contrato admitido: %s", cuerpo)
		}
	}
}

func TestDenegacionADMINFallaCerradoSinAuditoria(t *testing.T) {
	aud := &auditorPrueba{err: errors.New("auditoria caída")}
	h, err := NuevoHandler("https://admin.example.test", &sesionPrueba{}, &lecturasPrueba{}, &catalogoPrueba{}, &actosPrueba{}, aud)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "https://admin.example.test"+PrefijoV1+"/capacidades", nil)
	r.Header.Set("Cookie", "x=1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || aud.llamadas != 1 {
		t.Fatalf("estado=%d auditoria=%d", w.Code, aud.llamadas)
	}
}
