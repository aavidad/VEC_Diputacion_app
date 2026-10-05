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
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type sesionPrueba struct {
	llamadas  int
	resultado SesionConfiable
	err       error
}

func (s *sesionPrueba) ResolverSesionADMIN(context.Context, *http.Request) (SesionConfiable, error) {
	s.llamadas++
	if s.err != nil {
		return SesionConfiable{}, s.err
	}
	return s.resultado, nil
}

type lecturasPrueba struct {
	llamadas  int
	propuesta Propuesta
}

func (l *lecturasPrueba) Capacidades(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (Capacidades, error) {
	l.llamadas++
	return Capacidades{}, nil
}
func (l *lecturasPrueba) BuscarPersonas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, ConsultaPersonas) (PaginaPersonas, error) {
	l.llamadas++
	return PaginaPersonas{}, nil
}
func (l *lecturasPrueba) ConsultarPersona(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (FichaPersona, error) {
	l.llamadas++
	return FichaPersona{}, nil
}
func (l *lecturasPrueba) ListarRoles(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (Roles, error) {
	l.llamadas++
	return Roles{}, nil
}
func (l *lecturasPrueba) ListarPropuestas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (PaginaPropuestas, error) {
	l.llamadas++
	return PaginaPropuestas{}, nil
}
func (l *lecturasPrueba) ConsultarPropuesta(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (Propuesta, error) {
	l.llamadas++
	return l.propuesta, nil
}
func (l *lecturasPrueba) ConsultarRecibo(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (domain.ReciboAdministracionPerfiles, error) {
	l.llamadas++
	return domain.ReciboAdministracionPerfiles{}, nil
}

type catalogoPrueba struct{ llamadas int }

func (c *catalogoPrueba) ResolverRolAdministrable(context.Context, string) (ports.RolAdministrable, error) {
	c.llamadas++
	return ports.RolAdministrable{}, nil
}

type actosPrueba struct {
	llamadas int
	err      error
}

type auditorPrueba struct {
	llamadas int
	err      error
	ultima   DenegacionADMIN
}

func (a *auditorPrueba) RegistrarDenegacionADMIN(_ context.Context, d DenegacionADMIN) error {
	a.llamadas++
	a.ultima = d
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
	return ports.CierrePropuestaAdministracionPerfiles{}, a.err
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
		if decodificar(w, r, &dto) == nil {
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

func sesionADMINPrueba(t *testing.T) SesionConfiable {
	t.Helper()
	ahora := time.Date(2026, 9, 30, 19, 0, 0, 0, time.UTC)
	persona := "per_" + strings.Repeat("a", 22)
	perfil := "prf_" + strings.Repeat("b", 22)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		ahora, persona, perfil, domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	actor := resultado.Contexto
	rol := domain.VersionRol{
		RolID: "tecnico_bolsa", Version: 1, Nombre: "Tecnico de bolsa",
		Estado:       domain.EstadoVersionRolPublicada,
		Concesiones:  []domain.ConcesionRol{{Accion: "bolsa.expediente.leer", ModuloID: "bolsa", TipoRecurso: "expediente", Finalidades: []string{"gestion_bolsa"}, GarantiaMinima: domain.AuthAssuranceSubstantial}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-admin", Version: 1, PerfilActivoRef: perfil, PrincipalID: persona,
			VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
			Ambitos:      []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "responsable-seguridad", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: rol,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: rol.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
	if err := snapshot.Validar(); err != nil {
		t.Fatal(err)
	}
	return SesionConfiable{Actor: actor,
		Evidencia:               domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo},
		InstantaneaAutorizacion: snapshot,
		CorrelacionRef:          "correlacion_" + strings.Repeat("e", 32)}
}

func TestSesionADMINRechazaEvidenciaAusenteOCruzada(t *testing.T) {
	base := sesionADMINPrueba(t)
	otra, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		base.Actor.ResueltoEn.Add(2*time.Minute),
		"per_"+strings.Repeat("f", 22), "prf_"+strings.Repeat("g", 22),
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, evidencia := range map[string]domain.EvidenciaSesionAdministracionPerfiles{
		"ausente":    {},
		"otro actor": {ResultadoContexto: otra, Vinculo: vinculo},
	} {
		t.Run(nombre, func(t *testing.T) {
			s := base
			s.Evidencia = evidencia
			lecturas := &lecturasPrueba{}
			h, err := NuevoHandler("https://admin.example.test", &sesionPrueba{resultado: s},
				lecturas, &catalogoPrueba{}, &actosPrueba{}, &auditorPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/capacidades", ""))
			if w.Code != http.StatusServiceUnavailable || lecturas.llamadas != 0 {
				t.Fatalf("estado=%d lecturas=%d", w.Code, lecturas.llamadas)
			}
		})
	}
}

func peticionADMIN(metodo, ruta, cuerpo string) *http.Request {
	r := httptest.NewRequest(metodo, "https://admin.example.test"+ruta, strings.NewReader(cuerpo))
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	if metodo == http.MethodPost {
		r.Header.Set("Origin", "https://admin.example.test")
		r.Header.Set("Sec-Fetch-Site", "same-origin")
		r.Header.Set("Sec-Fetch-Mode", "cors")
		r.Header.Set("Sec-Fetch-Dest", "empty")
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func TestPrefijoDeOperacionNoCruzaRutasADMIN(t *testing.T) {
	s := &sesionPrueba{resultado: sesionADMINPrueba(t)}
	l, c, a, aud := &lecturasPrueba{}, &catalogoPrueba{}, &actosPrueba{}, &auditorPrueba{}
	h, err := NuevoHandler("https://admin.example.test", s, l, c, a, aud)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct{ ruta, prefijo string }{
		{PrefijoV1 + "/actos-ordinarios", "propuesta_admin:"},
		{PrefijoV1 + "/propuestas", "acto_admin:"},
	} {
		cuerpo := `{"operacion_ref":"` + caso.prefijo + strings.Repeat("a", 32) + `"}`
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, caso.ruta, cuerpo))
		if w.Code != http.StatusBadRequest || c.llamadas != 0 || a.llamadas != 0 || aud.llamadas == 0 {
			t.Fatalf("ruta=%s estado=%d catalogo=%d actos=%d auditoria=%d", caso.ruta, w.Code, c.llamadas, a.llamadas, aud.llamadas)
		}
		if aud.ultima.ActorPersonaRef != s.resultado.Actor.PersonaRef ||
			aud.ultima.PerfilActivoRef != s.resultado.Actor.PerfilActivoRef ||
			aud.ultima.CorrelacionRef != s.resultado.CorrelacionRef {
			t.Fatal("denegacion sin contexto acreditado")
		}
	}
	aud.err = errors.New("auditoria indisponible")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/actos-ordinarios", `{"operacion_ref":"propuesta_admin:`+strings.Repeat("a", 32)+`"}`))
	if w.Code != http.StatusServiceUnavailable || c.llamadas != 0 || a.llamadas != 0 {
		t.Fatalf("auditoria caída: estado=%d catalogo=%d actos=%d", w.Code, c.llamadas, a.llamadas)
	}
}

func TestCierreADMINNoUsaPuedeCerrarParaBloquearReplay(t *testing.T) {
	s := &sesionPrueba{resultado: sesionADMINPrueba(t)}
	ref := "propuesta_admin:" + strings.Repeat("a", 32)
	huella := strings.Repeat("b", 64)
	l := &lecturasPrueba{propuesta: Propuesta{PropuestaRef: ref, HuellaSHA256: huella,
		ProponentePersonaRef: "per_" + strings.Repeat("f", 22), ObjetivoPersonaRef: "per_" + strings.Repeat("g", 22), PuedeCerrar: false}}
	a := &actosPrueba{err: ErrConflictoEstado}
	h, err := NuevoHandler("https://admin.example.test", s, l, &catalogoPrueba{}, a, &auditorPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"operacion_ref":"cierre_admin:` + strings.Repeat("c", 32) + `","propuesta_huella_sha256":"` + huella + `","decision":"aprobada","motivo":{"catalogo_id":"motivos_admin","catalogo_version":1,"catalogo_huella_sha256":"` + strings.Repeat("d", 64) + `","entrada_clave":"revision"}}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/propuestas/"+ref+"/cierre", cuerpo))
	if w.Code != http.StatusConflict || l.llamadas != 1 || a.llamadas != 1 {
		t.Fatalf("estado=%d lectura=%d cierre_durable=%d cuerpo=%s", w.Code, l.llamadas, a.llamadas, w.Body.String())
	}
}
