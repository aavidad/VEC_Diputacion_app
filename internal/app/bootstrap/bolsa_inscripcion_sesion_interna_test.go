package bootstrap

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

func escenarioSesionInternaInscripcionPrueba(t *testing.T) (SesionInscripcionBolsa, *entornoSesionConsultaPrueba, *http.Request) {
	t.Helper()
	e := nuevaSesionConsultaPrueba(t)
	ahora := e.reloj.Ahora()
	certificado := &x509.Certificate{Raw: []byte("certificado-rrhh-marta-rios"), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	huella := sha256.Sum256(certificado.Raw)
	sha := hex.EncodeToString(huella[:])
	principal := clonarPrincipalDesarrollo(e.principal)
	principal.Attributes["certificate_sha256"] = sha
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(identidadCertificadoDesarrollo{huella: huella, principal: principal})
	if err != nil {
		t.Fatal(err)
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: resolvedor, registro: e.registro,
		revalidador: e.revalidador, contextos: e.resolutor, reloj: e.reloj, instancia: "inscripcion-interna-prueba"}
	cuenta := cuentaRutasDietasDesarrollo{CertificadoSHA256: sha, Sujeto: principal.ID,
		CuentaRef: e.soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef,
		PerfilRef: e.soporte.contexto.Resultado.Contexto.PerfilActivoRef}
	sesion, err := nuevaSesionInternaInscripcionBolsa(base, []cuentaRutasDietasDesarrollo{cuenta})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/rrhh/inscripciones/convocatorias?limite=20", nil)
	r.RemoteAddr = "127.0.0.1:12345"
	r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: []*x509.Certificate{certificado},
		VerifiedChains:   [][]*x509.Certificate{{certificado, {Raw: []byte("ca-sintetica")}}}}
	return sesion, e, r
}

func TestSesionInternaInscripcionReusaRegistroNominalSinPermisoPrestado(t *testing.T) {
	sesion, e, r := escenarioSesionInternaInscripcionPrueba(t)
	ctx, acreditacion, err := sesion.ResolverInscripcion(r)
	if err != nil {
		t.Fatalf("resolución nominal RRHH: %v", err)
	}
	if acreditacion.PersonaRef != ctx.Resultado.Contexto.PersonaRef ||
		acreditacion.PerfilRef != ctx.Resultado.Contexto.PerfilActivoRef ||
		acreditacion.CuentaRef != ctx.Resultado.Contexto.Instantanea.CuentaRef ||
		acreditacion.Canal != "interna_corporativa" || !acreditacion.ValidaHasta.After(time.Now()) ||
		len(e.registro.altas) != 1 || e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 ||
		len(ctx.Resultado.Contexto.Principal.Permissions) != 0 {
		t.Fatal("la sesión duplica el registro, infiere Persona o hereda permisos de otro módulo")
	}
}

func TestSesionInternaInscripcionRechazaCabeceraSuplantadaYRutaAjena(t *testing.T) {
	sesion, e, r := escenarioSesionInternaInscripcionPrueba(t)
	r.Header.Set("X-Vec-Persona", "per_ajena")
	if _, _, err := sesion.ResolverInscripcion(r); !errors.Is(err, inscripcion.ErrSesionAusente) || len(e.registro.altas) != 0 {
		t.Fatalf("cabecera del navegador abrió sesión: %v", err)
	}
	sesion, e, r = escenarioSesionInternaInscripcionPrueba(t)
	r.URL.Path = "/api/vec/bolsa/inscripciones/propias"
	if _, _, err := sesion.ResolverInscripcion(r); !errors.Is(err, inscripcion.ErrSesionAusente) || len(e.registro.altas) != 0 {
		t.Fatalf("la sesión interna atendió una ruta de aspirante: %v", err)
	}
	if _, err := nuevaSesionInternaInscripcionBolsa(nil, nil); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatal("fuente común ausente admitida")
	}
	if !rutaInscripcionInterna("/api/vec/bolsa/rrhh/inscripciones") ||
		rutaInscripcionInterna("/api/vec/bolsa/inscripciones/convocatorias-abiertas") ||
		strings.Contains(acreditacionInscripcionRedactada, "per_") {
		t.Fatal("se mezclaron canales internos")
	}
}
