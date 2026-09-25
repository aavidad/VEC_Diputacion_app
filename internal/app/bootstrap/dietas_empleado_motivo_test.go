package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// resolutorEmpleadoDenegadoPrueba reproduce la respuesta del servicio de
// contexto con alcance {empleado}: conserva el motivo cerrado de Personal.
type resolutorEmpleadoDenegadoPrueba struct{ motivo error }

func (r resolutorEmpleadoDenegadoPrueba) ResolverContextoActorRegistradoV2(context.Context, dominiovec.SolicitudContextoActor) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return dominiovec.ResultadoContextoActorRegistradoV2{}, errors.Join(dominiovec.ErrContextoActorNoResuelto, r.motivo)
}

func TestDietasSinEmpleadoCanonicoDeniegaConMotivoSinServir(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	identidad := composicion.identidad.(*resolvedorIdentidadDesarrollo)
	clienteCert, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(clienteCert.Certificate[0])
	huella := hex.EncodeToString(digest[:])
	principal := identidad.porHuella[digest]
	fixture := nuevoEscenarioMaterialRutasDietasPrueba(t, dietasports.AccionConsultarCatalogoRutasDietas, time.Now().UTC().Truncate(time.Microsecond))
	reloj := &relojSesionConsultaPrueba{ahora: fixture.ahora}
	cuenta := cuentaRutasDietasDesarrollo{CertificadoSHA256: huella, Sujeto: principal.ID, CuentaRef: fixture.resultado.Contexto.Instantanea.CuentaRef, PerfilRef: fixture.resultado.Contexto.PerfilActivoRef}
	ca, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(ca) {
		t.Fatal("CA de prueba inválida")
	}
	for _, caso := range []struct {
		motivo error
		codigo string
		ruta   string
	}{
		{vecports.ErrProyeccionEmpleadoContextoActorAusente, "empleado_no_disponible", personalhttp.RutaRelacionesDietas},
		{vecports.ErrProyeccionEmpleadoContextoActorAmbigua, "empleado_ambiguo", personalhttp.RutaRelacionesDietas},
		{vecports.ErrProyeccionEmpleadoContextoActorAusente, "empleado_no_disponible", dietashttp.RutaBorradores},
	} {
		t.Run(caso.codigo+caso.ruta, func(t *testing.T) {
			registro := &registroSesionConsultaPrueba{reloj: reloj, cuenta: cuenta.CuentaRef}
			revalidador := &revalidadorSesionConsultaPrueba{registro: registro}
			base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, registro: registro, revalidador: revalidador, contextos: resolutorEmpleadoDenegadoPrueba{motivo: caso.motivo}, reloj: reloj, instancia: strings.Repeat("a", 64)}
			auditoria := &registradorFronteraComisionPrueba{}
			auditoriaPersonal := &registradorFronteraPersonalDietasPrueba{}
			autoridad := &autoridadComisionesDietasDesarrollo{base: base, reloj: reloj, cuentas: map[string]cuentaRutasDietasDesarrollo{huella: cuenta}, registrador: auditoria, registradorPersonal: auditoriaPersonal}
			servidos := 0
			servidor := httptest.NewUnstartedServer(autoridad.proteger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				servidos++
				w.WriteHeader(http.StatusNoContent)
			})))
			servidor.TLS = composicion.tls.Clone()
			servidor.StartTLS()
			t.Cleanup(servidor.Close)
			transporte := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{clienteCert}, RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13}}
			t.Cleanup(transporte.CloseIdleConnections)
			respuesta, err := (&http.Client{Transport: transporte}).Get(servidor.URL + caso.ruta)
			if err != nil {
				t.Fatal(err)
			}
			defer respuesta.Body.Close()
			var cuerpo struct {
				Codigo string `json:"codigo"`
			}
			if err := json.NewDecoder(respuesta.Body).Decode(&cuerpo); err != nil {
				t.Fatalf("respuesta sin motivo legible: %v", err)
			}
			if respuesta.StatusCode != http.StatusForbidden || cuerpo.Codigo != caso.codigo || servidos != 0 ||
				respuesta.Header.Get("Content-Type") != "application/json; charset=utf-8" || respuesta.Header.Get("Cache-Control") != "no-store" {
				t.Fatalf("denegación sin motivo: estado=%d codigo=%q servidos=%d cabeceras=%v", respuesta.StatusCode, cuerpo.Codigo, servidos, respuesta.Header)
			}
			if caso.ruta == personalhttp.RutaRelacionesDietas {
				if auditoriaPersonal.llamadas != 1 || auditoriaPersonal.orden.EstadoHTTP != http.StatusForbidden || auditoriaPersonal.orden.ActorRef != "" || auditoriaPersonal.orden.Motivo != personalports.MotivoFronteraPersonalDenegado {
					t.Fatalf("denegación Personal no auditada: %+v", auditoriaPersonal.orden)
				}
			} else if auditoria.orden.Motivo != dietasports.MotivoFronteraAccesoDenegado || auditoria.orden.ActorRef != "" {
				t.Fatalf("denegación Dietas no auditada: %+v", auditoria.orden)
			}
		})
	}
}
