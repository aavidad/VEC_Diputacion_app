package bootstrap

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
)

func escenarioSesionExternaInscripcionPrueba(t *testing.T) (*SesionExternaInscripcion, *autoridadPreferenciasUsuariosDesarrollo, *entornoSesionConsultaPrueba, *http.Request, *x509.Certificate) {
	t.Helper()
	e := nuevaSesionConsultaPrueba(t)
	ahora := time.Now().UTC()
	certificado := &x509.Certificate{Raw: []byte("certificado-persona-sin-bolsa"), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	huella := sha256.Sum256(certificado.Raw)
	sha := hex.EncodeToString(huella[:])
	principal := clonarPrincipalDesarrollo(e.principal)
	principal.Attributes["certificate_sha256"] = sha
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(identidadCertificadoDesarrollo{huella: huella, principal: principal})
	if err != nil || resolvedor.candidatoBolsa != nil {
		t.Fatal("la identidad de ensayo exige candidato")
	}
	cuenta := cuentaUsuariosPreferenciasDesarrollo{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
		CertificadoSHA256: sha, Sujeto: principal.ID,
		CuentaRef: e.soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef,
		PerfilRef: e.soporte.contexto.Resultado.Contexto.PerfilActivoRef,
	}}
	e.revalidador.alterar = func(a *core.AutenticacionRevalidadaV1) {
		a.Superficie = core.SuperficieAutenticacionExternaPersonalV1
	}
	e.registro.despues = func() {
		e.reloj.ahora = e.registro.confirmacion.AltaConfirmada.SesionEmitidaEn
		e.registro.confirmacion.SesionRevalidadaEn = e.reloj.ahora
	}
	autoridad := &autoridadPreferenciasUsuariosDesarrollo{
		base: &autoridadRutasDietasDesarrollo{resolvedor: resolvedor, registro: e.registro,
			revalidador: e.revalidador, contextos: e.resolutor, instancia: "sesion-inscripcion-prueba"},
		cuentas: map[string]cuentaUsuariosPreferenciasDesarrollo{sha: cuenta}, reloj: relojRutasDietas{},
		ruta: usuarioshttp.RutaMisPreferenciasAreaPersonal, superficie: core.SuperficieAutenticacionExternaPersonalV1,
	}
	sesion, err := NuevaSesionExternaInscripcion(autoridad)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/inscripciones/convocatorias-abiertas", nil)
	r.RemoteAddr = "127.0.0.1:12345"
	r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: []*x509.Certificate{certificado},
		VerifiedChains:   [][]*x509.Certificate{{certificado, {Raw: []byte("ca-sintetica")}}}}
	return sesion, autoridad, e, r, certificado
}

func TestSesionExternaInscripcionResuelvePersonaSinCandidatoUnaVez(t *testing.T) {
	sesion, _, e, r, certificado := escenarioSesionExternaInscripcionPrueba(t)
	ctx, a, err := sesion.ResolverSesionExterna(r)
	if err != nil {
		t.Fatalf("%v (registro=%d, revalidacion=%d, contexto=%d)", err, len(e.registro.altas), e.revalidador.llamadas, e.resolutor.llamadas)
	}
	if len(ctx.Resultado.Contexto.Instantanea.Vinculos) != 0 ||
		ctx.Resultado.Contexto.PersonaRef == e.principal.ID ||
		len(ctx.Resultado.Contexto.Principal.Permissions) != 0 ||
		a.PersonaRef != ctx.Resultado.Contexto.PersonaRef ||
		a.Canal != string(core.SuperficieAutenticacionExternaPersonalV1) ||
		a.PerfilRef != ctx.Resultado.Contexto.PerfilActivoRef ||
		a.CuentaRef != ctx.Resultado.Contexto.Instantanea.CuentaRef ||
		!a.ValidaHasta.After(time.Now()) || a.ValidaHasta.After(certificado.NotAfter) ||
		len(e.registro.altas) != 1 || e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 {
		t.Fatal("resolución duplicada, persona inferida del principal o permiso heredado")
	}
	serializada, err := json.Marshal(a)
	if err != nil || strings.Contains(string(serializada), a.PersonaRef) ||
		strings.Contains(fmt.Sprintf("%+v", a), a.CertificadoHuellaSHA256) ||
		strings.Contains(a.LogValue().String(), a.PersonaRef) {
		t.Fatal("la acreditación expone persona o certificado al formatear")
	}
	// No se compuso proveedor de material ni PDP de Preferencias: esta salida
	// acredita identidad y deja las acciones de inscripción a su autoridad.
}

func TestSesionExternaInscripcionDeniegaCertificadoCuentaOVigenciaDivergentes(t *testing.T) {
	casos := []struct {
		nombre      string
		mutar       func(*autoridadPreferenciasUsuariosDesarrollo, *entornoSesionConsultaPrueba, *http.Request, *x509.Certificate)
		conRegistro bool
	}{
		{"huella ajena", func(_ *autoridadPreferenciasUsuariosDesarrollo, _ *entornoSesionConsultaPrueba, r *http.Request, _ *x509.Certificate) {
			r.TLS.PeerCertificates[0] = &x509.Certificate{Raw: []byte("otro")}
		}, false},
		{"cuenta ajena", func(a *autoridadPreferenciasUsuariosDesarrollo, _ *entornoSesionConsultaPrueba, _ *http.Request, _ *x509.Certificate) {
			for k, c := range a.cuentas {
				c.Sujeto = "otro"
				a.cuentas[k] = c
			}
		}, false},
		{"huella de cuenta distinta", func(a *autoridadPreferenciasUsuariosDesarrollo, _ *entornoSesionConsultaPrueba, _ *http.Request, _ *x509.Certificate) {
			for k, c := range a.cuentas {
				c.CertificadoSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				a.cuentas[k] = c
			}
		}, false},
		{"certificado vencido", func(_ *autoridadPreferenciasUsuariosDesarrollo, _ *entornoSesionConsultaPrueba, _ *http.Request, c *x509.Certificate) {
			c.NotAfter = time.Now().Add(-time.Second)
		}, false},
		{"persona de contexto divergente", func(_ *autoridadPreferenciasUsuariosDesarrollo, e *entornoSesionConsultaPrueba, _ *http.Request, _ *x509.Certificate) {
			e.resolutor.base.Contexto.Instantanea.PersonaRef = "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}, true},
		{"sesion revocada", func(_ *autoridadPreferenciasUsuariosDesarrollo, e *entornoSesionConsultaPrueba, _ *http.Request, _ *x509.Certificate) {
			e.revalidador.err = errors.New("revocada")
		}, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			sesion, autoridad, e, r, certificado := escenarioSesionExternaInscripcionPrueba(t)
			caso.mutar(autoridad, e, r, certificado)
			ctx, a, err := sesion.ResolverSesionExterna(r)
			if err == nil || ctx.Resultado.Validar() == nil || a.PersonaRef != "" ||
				(len(e.registro.altas) != 0) != caso.conRegistro {
				t.Fatalf("se emitió identidad con frontera divergente: %v", err)
			}
		})
	}
}

func TestSesionExternaInscripcionSinPreferenciasNiRutaCierra(t *testing.T) {
	if _, err := NuevaSesionExternaInscripcion(nil); !errors.Is(err, ErrSesionExternaInscripcionNoDisponible) {
		t.Fatal("sin autoridad de Preferencias se compuso sesión paralela")
	}
	sesion, _, e, r, _ := escenarioSesionExternaInscripcionPrueba(t)
	r.URL.Path = "/api/vec/session"
	if _, _, err := sesion.ResolverSesionExterna(r); !errors.Is(err, ErrSesionExternaInscripcionNoAutenticada) || len(e.registro.altas) != 0 {
		t.Fatal("ruta ajena obtuvo una sesión")
	}
}

func TestPreferenciasExternaConservaResolucionTrasCompartirCotejo(t *testing.T) {
	_, autoridad, e, r, _ := escenarioSesionExternaInscripcionPrueba(t)
	r.URL.Path = usuarioshttp.RutaMisPreferenciasAreaPersonal
	autoridad.manejador = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	visto := false
	siguiente := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := r.Context().Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
		visto = ok && c.autoridad == autoridad && c.resultado.Validar() == nil &&
			c.vinculo.ValidarPara(c.resultado) == nil && len(c.resultado.Contexto.Instantanea.Vinculos) == 0
		w.WriteHeader(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	autoridad.proteger(siguiente).ServeHTTP(w, r)
	if w.Code != http.StatusNoContent || !visto || len(e.registro.altas) != 1 ||
		e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 {
		t.Fatal("la ruta de Preferencias perdió la identidad certificada")
	}
}
