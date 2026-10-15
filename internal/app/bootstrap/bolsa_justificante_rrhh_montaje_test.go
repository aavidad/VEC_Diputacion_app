package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	httpseguridad "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestJustificanteRenunciaRRHHSinConfiguracionDejaRutaCerrada(t *testing.T) {
	ruta, cerrar := nuevaRutaJustificanteRenunciaRRHH(config.Config{}, nil, nil, nil)
	if ruta == nil || ruta.Ruta != bolsahttp.RutaJustificanteRenunciaRRHH || ruta.Manejador == nil || cerrar == nil {
		t.Fatal("la falta de configuración impidió montar la ruta cerrada")
	}
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		ruta.Ruta+"?bolsa=bolsa_1&participacion=part_1", nil))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") == "" {
		t.Fatalf("sin configuración: estado=%d cabeceras=%v", w.Code, w.Header())
	}
	cerrar()
}

func TestConfiguracionJustificanteRenunciaRRHHNoExponeDSN(t *testing.T) {
	c := configuracionJustificanteRenunciaRRHH{DSNDocumentos: "postgres://usuario:clave-ficticia@127.0.0.1/vec"}
	b, err := json.Marshal(c)
	if err != nil || strings.Contains(string(b), "clave-ficticia") ||
		strings.Contains(fmt.Sprintf("%+v %#v", c, c), "clave-ficticia") {
		t.Fatal("la configuración expuso el DSN")
	}
}

func TestConfiguracionJustificanteRRHHSoloAceptaDependenciasTecnicas(t *testing.T) {
	directorio := t.TempDir()
	ruta := filepath.Join(directorio, rutaConfiguracionJustificanteRenunciaRRHH)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	base := `{"version":1,"superficie":"interna_corporativa","dsn_documentos":"postgres://ejemplo"`
	for _, caso := range []struct {
		nombre, contenido string
		aceptar           bool
	}{
		{"tecnica", base + `}`, true},
		{"persona_fijada", base + `,"persona_ref":"per_fijada"}`, false},
		{"perfil_fijado", base + `,"perfil_ref":"prf_fijado"}`, false},
		{"certificado_fijado", base + `,"certificado_sha256":"fijado"}`, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := os.WriteFile(ruta, []byte(caso.contenido), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := leerConfiguracionJustificanteRenunciaRRHH(config.Config{DevelopmentMaterialDir: directorio})
			if (err == nil) != caso.aceptar {
				t.Fatalf("configuracion aceptada=%t", err == nil)
			}
		})
	}
}

type consultorJustificanteRRHHPermisoPrueba struct {
	actorRefs []string
	denegar   bool
}

type registroSesionJustificanteRRHHPrueba struct {
	httpseguridad.RegistroSesiones
	reloj *relojSesionConsultaPrueba
}

func (r registroSesionJustificanteRRHHPrueba) ConsumirAsercionYRegistrar(ctx context.Context,
	alta httpseguridad.AltaSesionAtomica) (httpseguridad.ConfirmacionAltaSesion, error) {
	r.reloj.ahora = alta.AutenticacionVerificadaEn
	return r.RegistroSesiones.ConsumirAsercionYRegistrar(ctx, alta)
}

func (c *consultorJustificanteRRHHPermisoPrueba) AbrirJustificanteRenunciaRRHH(_ context.Context,
	i bolsahttp.IdentidadJustificanteRenunciaRRHH, _, _ string) (io.ReadCloser, error) {
	c.actorRefs = append(c.actorRefs, i.Actor.Contexto.PersonaRef)
	if c.denegar {
		return nil, docports.ErrSubidaLocalDenegada
	}
	return io.NopCloser(strings.NewReader("%PDF-1.4\nrecibo\n%%EOF")), nil
}

func TestJustificanteRRHHResuelveDosTecnicasYDeniegaSinPermiso(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	identidades := make([]identidadCertificadoDesarrollo, 0, 2)
	cuentas := make(map[string]cuentaUsuariosPreferenciasDesarrollo, 2)
	peticiones := make([]*http.Request, 0, 2)
	contextos := make([]core.ResultadoContextoActorRegistradoV2, 0, 2)
	for _, nombre := range []string{"tecnica-uno", "tecnica-dos"} {
		certificado := &x509.Certificate{Raw: []byte("certificado-" + nombre),
			NotBefore: e.reloj.Ahora().Add(-time.Hour), NotAfter: e.reloj.Ahora().Add(90 * 24 * time.Hour)}
		huella := sha256.Sum256(certificado.Raw)
		sha := hex.EncodeToString(huella[:])
		principal := clonarPrincipalDesarrollo(e.principal)
		principal.ID = nombre
		principal.Attributes["certificate_sha256"] = sha
		contexto, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, e.reloj.Ahora())
		if err != nil {
			t.Fatal(err)
		}
		contextos = append(contextos, contexto.Resultado)
		identidades = append(identidades, identidadCertificadoDesarrollo{huella: huella, principal: principal})
		cuentas[sha] = cuentaUsuariosPreferenciasDesarrollo{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
			CertificadoSHA256: sha, Sujeto: principal.ID,
			CuentaRef: contexto.Resultado.Contexto.Instantanea.CuentaRef,
			PerfilRef: contexto.Resultado.Contexto.PerfilActivoRef}}
		r := httptest.NewRequest(http.MethodGet,
			bolsahttp.RutaJustificanteRenunciaRRHH+"?bolsa=bolsa_1&participacion=part_1", nil)
		r.RemoteAddr = "127.0.0.1:12345"
		r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
			PeerCertificates: []*x509.Certificate{certificado},
			VerifiedChains:   [][]*x509.Certificate{{certificado, {Raw: []byte("ca-sintetica")}}}}
		peticiones = append(peticiones, r)
	}
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(identidades...)
	if err != nil {
		t.Fatal(err)
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: resolvedor,
		registro: registroSesionJustificanteRRHHPrueba{RegistroSesiones: e.registro, reloj: e.reloj},
		revalidador: e.revalidador, contextos: e.resolutor, reloj: e.reloj, instancia: "justificante-rrhh-prueba"}
	autoridad := &autoridadPreferenciasUsuariosDesarrollo{base: base, cuentas: cuentas, reloj: relojRutasDietas{},
		superficie: core.SuperficieAutenticacionInternaCorporativaV1}
	consulta := &consultorJustificanteRRHHPermisoPrueba{}
	h, err := bolsahttp.NuevoHandlerJustificanteRenunciaRRHH(
		proveedorIdentidadJustificanteRenunciaRRHH{autoridad: autoridad}, consulta)
	if err != nil {
		t.Fatal(err)
	}
	for indice, r := range peticiones {
		e.resolutor.base = contextos[indice]
		e.registro.cuenta = contextos[indice].Contexto.Instantanea.CuentaRef
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK || len(consulta.actorRefs) != indice+1 ||
			consulta.actorRefs[indice] != contextos[indice].Contexto.PersonaRef {
			t.Fatalf("tecnica %d: estado=%d actor=%v", indice, w.Code, consulta.actorRefs)
		}
	}
	if consulta.actorRefs[0] == consulta.actorRefs[1] || len(e.registro.altas) != 2 ||
		e.revalidador.llamadas != 2 || e.resolutor.llamadas != 2 {
		t.Fatal("las tecnicas compartieron identidad o se omitio una revalidacion")
	}
	consulta.denegar = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticiones[1])
	if w.Code != http.StatusForbidden || len(consulta.actorRefs) != 3 ||
		consulta.actorRefs[2] != contextos[1].Contexto.PersonaRef || len(e.registro.altas) != 3 {
		t.Fatalf("permiso retirado: estado=%d actor=%v", w.Code, consulta.actorRefs)
	}
	peticiones[0].Header.Set("X-Vec-Perfil", "prf_suplantado")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticiones[0])
	if w.Code != http.StatusForbidden || len(consulta.actorRefs) != 3 {
		t.Fatalf("cabecera de perfil aceptada: estado=%d", w.Code)
	}
}
