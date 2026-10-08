package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	portsDietas "vec-diputacion-granada/internal/modules/dietas/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func TestConfiguracionSesionFirmanteV2CerradaYPlantilla(t *testing.T) {
	contenido, err := os.ReadFile("contratacion_temporal_firma_v2_sesion_firmante.ejemplo.json")
	if err != nil {
		t.Fatal(err)
	}
	c, retirada, err := decodificarConfiguracionSesionFirmanteV2(contenido)
	if err != nil || c.Autoridad != AutoridadNoAutoritativa || retirada.IsZero() {
		t.Fatalf("plantilla: %v", err)
	}
	if strings.Contains(fmt.Sprintf("%#v", c), c.DSNRegistroIdentidad) {
		t.Fatal("configuración expuesta al formato Go")
	}
	for nombre, dato := range map[string]string{
		"sin_fecha":      strings.Replace(string(contenido), `"retirada_en": "2026-12-31T00:00:00Z"`, `"retirada_en": ""`, 1),
		"sin_dsn":        strings.Replace(string(contenido), `"COMPLETAR_EN_CONFIGURACION_PRIVADA"`, `""`, 1),
		"campo_ajeno":    strings.Replace(string(contenido), `"version": 1,`, `"version": 1, "cuentas": [],`, 1),
		"clave_repetida": strings.Replace(string(contenido), `"version": 1,`, `"version": 1, "version": 1,`, 1),
		"otra_autoridad": strings.Replace(string(contenido), `"no_autoritativo"`, `"productivo"`, 1),
		"texto_extra":    string(contenido) + `{}`,
	} {
		if _, _, err := decodificarConfiguracionSesionFirmanteV2([]byte(dato)); err == nil {
			t.Errorf("%s admitida", nombre)
		}
	}
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	identidadRuta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "firma-vec.json")
	if a, cerrar, err := nuevaAutoridadSesionFirmanteV2(cfg, nil, nil); err != nil || a != nil || cerrar == nil {
		t.Fatalf("ausencia: %v", err)
	}
	if err := os.WriteFile(identidadRuta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.DevelopmentGuard = ""
	if _, _, err := nuevaAutoridadSesionFirmanteV2(cfg, nil, nil); !errors.Is(err, errSesionFirmanteV2NoDisponible) {
		t.Fatalf("sin doble llave: %v", err)
	}
}

type entornoFirmanteV2Prueba struct {
	a         *autoridadSesionFirmanteV2
	fuente    *fuenteSinteticaSesionFirmanteV2
	r         *http.Request
	seleccion ports.SeleccionFirmanteV2
	huella    string
	persona   string
	reloj     *relojSesionConsultaPrueba
	registro  *registroSesionConsultaPrueba
	contextos *resolutorContextoCronosPrueba
}

func nuevoEntornoFirmanteV2Prueba(t *testing.T) entornoFirmanteV2Prueba {
	t.Helper()
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	identidad := composicion.identidad.(*resolvedorIdentidadDesarrollo)
	par, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	hoja, err := x509.ParseCertificate(par.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	bloque, _ := pem.Decode(caPEM)
	if bloque == nil {
		t.Fatal("CA ilegible")
	}
	ca, err := x509.ParseCertificate(bloque.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	huellaBytes := sha256.Sum256(hoja.Raw)
	huella := hex.EncodeToString(huellaBytes[:])
	if _, ok := identidad.porHuella[huellaBytes]; !ok {
		t.Fatal("certificado no provisionado")
	}
	r := httptest.NewRequest(http.MethodPost, httpinterno.RutaRegistroFirmaVec, nil)
	r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: []*x509.Certificate{hoja}, VerifiedChains: [][]*x509.Certificate{{hoja, ca}}}
	r.RemoteAddr = "127.0.0.1:4321"
	fixture := nuevoEscenarioMaterialRutasDietasPrueba(t, portsDietas.AccionConsultarCatalogoRutasDietas, time.Now().UTC().Truncate(time.Microsecond))
	reloj := &relojSesionConsultaPrueba{ahora: fixture.ahora}
	cuenta := fixture.resultado.Contexto.Instantanea.CuentaRef
	registro := &registroSesionConsultaPrueba{reloj: reloj, cuenta: cuenta}
	contextos := &resolutorContextoCronosPrueba{base: &resolutorSesionConsultaPrueba{base: fixture.resultado, reloj: reloj}}
	a, err := nuevaAutoridadSesionFirmanteV2ConPuertos(identidad, registro,
		&revalidadorSesionConsultaPrueba{registro: registro}, contextos, reloj, reloj.Ahora().Add(time.Hour), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	seleccion := ports.SeleccionFirmanteV2{PersonaRef: fixture.resultado.Contexto.PersonaRef, RolID: "rol_firma_vec_prueba",
		CuentaRef: cuenta, PerfilActivoRef: fixture.resultado.Contexto.PerfilActivoRef,
		VinculoCertificado: ports.ReferenciaVersionadaFirmanteV2{Referencia: "vcr_prueba", Version: 1, HuellaSHA256: strings.Repeat("b", 64)}}
	return entornoFirmanteV2Prueba{a: a, fuente: a.fuente.(*fuenteSinteticaSesionFirmanteV2), r: r, seleccion: seleccion, huella: huella, persona: seleccion.PersonaRef,
		reloj: reloj, registro: registro, contextos: contextos}
}

func TestSesionFirmanteV2LigaCertificadoSeleccionYPeticion(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	if _, err := e.a.acreditar(e.r, e.seleccion, e.persona, strings.Repeat("c", 64)); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("huella AUT56 ajena admitida", err)
	}
	if _, err := e.a.acreditar(e.r, e.seleccion, "per_otra_persona_000000000", e.huella); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("persona CA25 ajena admitida", err)
	}
	conCabecera := e.r.Clone(e.r.Context())
	conCabecera.Header.Set("X-Vec-User", "otro")
	if _, err := e.a.acreditar(conCabecera, e.seleccion, e.persona, e.huella); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("cabecera ambiental admitida", err)
	}
	conCookieVacia := e.r.Clone(e.r.Context())
	conCookieVacia.Header["Cookie"] = []string{""}
	if _, err := e.a.acreditar(conCookieVacia, e.seleccion, e.persona, e.huella); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("cabecera Cookie vacía admitida", err)
	}
	capsula, err := e.a.acreditar(e.r, e.seleccion, e.persona, e.huella)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.a.abrir(e.r.Clone(e.r.Context()), capsula); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("cápsula de otra petición admitida", err)
	}
	if len(e.registro.altas) != 0 {
		t.Fatal("otra petición produjo alta")
	}
	if _, _, err := e.a.abrir(e.r, capsula); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("empleado ausente admitido", err)
	}
	if len(e.registro.altas) != 1 || e.registro.altas[0].CuentaID != "desarrollo:"+e.seleccion.CuentaRef ||
		e.registro.altas[0].AsercionExpiraEn.After(e.fuente.retirada) {
		t.Fatal("alta sintética incorrecta")
	}
	if _, _, err := e.a.abrir(e.r, capsula); !errors.Is(err, errSesionFirmanteV2Denegada) || len(e.registro.altas) != 1 {
		t.Fatal("cápsula repetida")
	}
	proyectarEmpleadoFirmanteV2Prueba(t, &e)
	if _, _, err := e.a.revalidar(e.r, capsula); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("cápsula fallida recuperada después de cambiar el contexto")
	}
}

func proyectarEmpleadoFirmanteV2Prueba(t *testing.T, e *entornoFirmanteV2Prueba) {
	t.Helper()
	base := e.contextos.base.base
	actor := base.Contexto
	vinculo := core.VinculoReferenciaContextoActor{VinculoRef: "pep_0123456789abcdefghijkl", Version: 1,
		Tipo: core.TipoReferenciaContextoActorEmpleado, Referencia: "emp_0123456789abcdefghijkl",
		Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: e.reloj.Ahora().Add(-time.Hour),
		VigenteHasta: e.reloj.Ahora().Add(time.Hour)}
	actor.Instantanea.Vinculos = append(actor.Instantanea.Vinculos, vinculo)
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := core.RehidratarManifiestoProcedenciaContextoActorV1(base.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto.Vinculos = append(manifiesto.Vinculos, core.ProcedenciaVinculoReferenciaContextoActorV1{
		VinculoRef: vinculo.VinculoRef, Version: vinculo.Version, Tipo: vinculo.Tipo, Referencia: vinculo.Referencia,
		AcreditacionProcedenciaComponenteContextoActorV1: manifiesto.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1})
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	base.Contexto = actor
	base.RepresentacionCanonica, base.HuellaSHA256 = canon, huella
	base.ManifiestoProcedenciaCanonico, base.ManifiestoProcedenciaHuellaSHA256 = canonManifiesto, huellaManifiesto
	if base.Validar() != nil {
		t.Fatal("fixture de empleado incoherente")
	}
	e.contextos.base.base = base
}

func TestSesionFirmanteV2EntregaContextoEmpleadoRegistrado(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	proyectarEmpleadoFirmanteV2Prueba(t, &e)
	capsula, err := e.a.acreditar(e.r, e.seleccion, e.persona, e.huella)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, resultado, err := e.a.abrir(e.r, capsula)
	if err != nil || vinculo.ValidarPara(resultado) != nil || !resultado.Contexto.AlcanceProyecciones().IncluyeEmpleado() ||
		resultado.Contexto.PersonaRef != e.persona || resultado.Contexto.PerfilActivoRef != e.seleccion.PerfilActivoRef || len(e.registro.altas) != 1 {
		t.Fatalf("contexto de firmante no acreditado: %v", err)
	}
}

type fuenteSesionFirmanteInyectadaPrueba struct {
	sesion         *sesionFirmanteInyectadaPrueba
	abiertas       int
	ultimoContexto context.Context
}

type sesionFirmanteInyectadaPrueba struct {
	evidencia      ports.EvidenciaSesionFirmanteV2
	revalidaciones int
	ultimoContexto context.Context
}

func (f *fuenteSesionFirmanteInyectadaPrueba) AbrirSesionFirmanteV2(ctx context.Context,
	_ ports.SolicitudSesionFirmanteV2) (ports.SesionFirmanteV2, error) {
	f.abiertas++
	f.ultimoContexto = ctx
	return f.sesion, nil
}

func (s *sesionFirmanteInyectadaPrueba) RevalidarSesionFirmanteV2(ctx context.Context) (ports.EvidenciaSesionFirmanteV2, error) {
	s.revalidaciones++
	s.ultimoContexto = ctx
	return s.evidencia, nil
}

func TestSesionFirmanteV2AceptaAutoridadComunInyectadaYRevalida(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	proyectarEmpleadoFirmanteV2Prueba(t, &e)
	cSintetica, err := e.a.acreditar(e.r, e.seleccion, e.persona, e.huella)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, resultado, err := e.a.abrir(e.r, cSintetica)
	if err != nil {
		t.Fatal(err)
	}
	sesion := &sesionFirmanteInyectadaPrueba{evidencia: ports.EvidenciaSesionFirmanteV2{
		Vinculo: vinculo, Resultado: resultado, CertificadoCanalSHA256: e.huella,
		CertificadoValidoHasta: e.r.TLS.VerifiedChains[0][0].NotAfter.UTC().Truncate(time.Microsecond)}}
	fuente := &fuenteSesionFirmanteInyectadaPrueba{sesion: sesion}
	a, err := nuevaAutoridadSesionFirmanteV2ConFuente(fuente, e.reloj, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := a.acreditar(e.r, e.seleccion, e.persona, e.huella)
	if err != nil {
		t.Fatal(err)
	}
	ctxOperativo := context.WithValue(e.r.Context(), struct{ Correlacion string }{"firma"}, "correlacion-de-prueba")
	if _, _, err := a.abrirConContexto(ctxOperativo, e.r, c); err != nil {
		t.Fatal("fuente común rechazada", err)
	}
	if _, _, err := a.revalidarConContexto(ctxOperativo, e.r, c); err != nil || fuente.abiertas != 1 || sesion.revalidaciones != 2 ||
		fuente.ultimoContexto != ctxOperativo || sesion.ultimoContexto != ctxOperativo {
		t.Fatalf("se abrió otra sesión o se omitió revalidación: %v", err)
	}
	sesion.evidencia.CertificadoValidoHasta = e.reloj.Ahora()
	if _, _, err := a.revalidar(e.r, c); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("vigencia de autoridad común ignorada", err)
	}
}

func TestSesionFirmanteV2DeniegaPerfilPersonaEmpleadoYRetirada(t *testing.T) {
	for caso, preparar := range map[string]func(*entornoFirmanteV2Prueba){
		"perfil_ajeno": func(e *entornoFirmanteV2Prueba) { e.seleccion.PerfilActivoRef = "prf_otra_persona_00000000" },
		"persona_ajena": func(e *entornoFirmanteV2Prueba) {
			e.seleccion.PersonaRef = "per_otra_persona_00000000"
			e.persona = e.seleccion.PersonaRef
		},
		"empleado_ausente": func(e *entornoFirmanteV2Prueba) { e.contextos.motivo = vp.ErrProyeccionEmpleadoContextoActorAusente },
		"empleado_ambiguo": func(e *entornoFirmanteV2Prueba) { e.contextos.motivo = vp.ErrProyeccionEmpleadoContextoActorAmbigua },
	} {
		t.Run(caso, func(t *testing.T) {
			e := nuevoEntornoFirmanteV2Prueba(t)
			preparar(&e)
			capsula, err := e.a.acreditar(e.r, e.seleccion, e.persona, e.huella)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := e.a.abrir(e.r, capsula); !errors.Is(err, errSesionFirmanteV2Denegada) {
				t.Fatal("contexto ajeno admitido", err)
			}
		})
	}
	e := nuevoEntornoFirmanteV2Prueba(t)
	e.reloj.ahora = e.fuente.retirada
	if _, err := e.a.acreditar(e.r, e.seleccion, e.persona, e.huella); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatal("política vencida admitida", err)
	}
	if _, err := nuevaAutoridadSesionFirmanteV2ConPuertos(e.fuente.resolvedor, e.fuente.registro, e.fuente.revalidador,
		e.fuente.contextos, e.reloj, e.fuente.retirada, e.fuente.instancia); !errors.Is(err, errSesionFirmanteV2NoDisponible) {
		t.Fatal("arranque vencido", err)
	}
}
