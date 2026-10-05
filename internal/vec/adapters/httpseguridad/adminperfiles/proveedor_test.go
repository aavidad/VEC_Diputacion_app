package adminperfiles

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

type relojPrueba struct{ ahora time.Time }

func (r relojPrueba) Ahora() time.Time { return r.ahora }

// Estos puertos deniegan siempre; ninguna prueba fabrica una sesión válida.
type cuentasDenegadas struct{ llamadas int }

func (c *cuentasDenegadas) ResolverCuentaADMIN(context.Context, ObservacionADMIN) (CuentaADMIN, error) {
	c.llamadas++
	return CuentaADMIN{}, api.ErrAccesoDenegado
}
func (*cuentasDenegadas) VincularSesionADMIN(context.Context, ObservacionADMIN, CuentaADMIN, ReferenciasSesionADMIN) error {
	return api.ErrAccesoDenegado
}

type registroDenegado struct{}

func (registroDenegado) ConsumirAsercionYRegistrar(context.Context, h.AltaSesionAtomica) (h.ConfirmacionAltaSesion, error) {
	return h.ConfirmacionAltaSesion{}, api.ErrAccesoDenegado
}
func (registroDenegado) ComprobarSesionYCuentaActivas(context.Context, h.ConsultaSesionActiva) error {
	return api.ErrAccesoDenegado
}

func TestResolverVerificaTLSAntesDeConsultarCuentaNominal(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	estado := estadoTLSReal(t, ahora)
	observada := observacionTLS(estado, ahora)
	config := configADMINPrueba(ahora)
	casos := []struct {
		nombre   string
		cambiar  func(*tls.ConnectionState, *ObservacionADMIN)
		llamadas int
		err      error
	}{
		{"real sin concesión nominal", func(*tls.ConnectionState, *ObservacionADMIN) {}, 1, api.ErrAccesoDenegado},
		{"fabricado", func(e *tls.ConnectionState, _ *ObservacionADMIN) {
			*e = tls.ConnectionState{HandshakeComplete: e.HandshakeComplete, Version: e.Version,
				CipherSuite: e.CipherSuite, PeerCertificates: e.PeerCertificates, VerifiedChains: e.VerifiedChains}
		}, 0, api.ErrAutenticacionRequerida},
		{"sin cadena verificada", func(e *tls.ConnectionState, _ *ObservacionADMIN) { e.VerifiedChains = nil }, 0, api.ErrAutenticacionRequerida},
		{"par distinto", func(e *tls.ConnectionState, _ *ObservacionADMIN) {
			e.PeerCertificates = []*x509.Certificate{e.VerifiedChains[0][1]}
		}, 0, api.ErrAutenticacionRequerida},
		{"sesión resumida", func(e *tls.ConnectionState, _ *ObservacionADMIN) { e.DidResume = true }, 0, api.ErrAutenticacionRequerida},
		{"huella distinta", func(_ *tls.ConnectionState, o *ObservacionADMIN) { o.CertificadoSHA256 = o.CASHA256 }, 0, api.ErrAutenticacionRequerida},
		{"CRL caducada", func(_ *tls.ConnectionState, o *ObservacionADMIN) { o.CRLVigenteHasta = ahora }, 0, api.ErrAutenticacionRequerida},
		{"vigencia del certificado ampliada", func(_ *tls.ConnectionState, o *ObservacionADMIN) {
			o.CertificadoVigenteHasta = o.CertificadoVigenteHasta.Add(time.Hour)
		}, 0, api.ErrAutenticacionRequerida},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e, o := estado, observada
			caso.cambiar(&e, &o)
			cuentas := &cuentasDenegadas{}
			p := &Proveedor{config: config, deps: Dependencias{Cuentas: cuentas, Registro: registroDenegado{}, Reloj: relojPrueba{ahora}}}
			sesion, err := p.Resolver(context.Background(), &http.Request{Host: o.Host, TLS: &e}, o)
			if !errors.Is(err, caso.err) || cuentas.llamadas != caso.llamadas || sesion.CorrelacionRef != "" {
				t.Fatalf("error=%v consultas=%d correlación=%q", err, cuentas.llamadas, sesion.CorrelacionRef)
			}
		})
	}
}

// El proveedor exige de nuevo la autoridad exacta de la frontera (con puerto
// público) y que su nombre sin puerto sea el de la observación (host_admin).
func TestResolverExigeAutoridadExactaYNombreSinPuerto(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	estado := estadoTLSReal(t, ahora)
	for _, caso := range []struct {
		autoridad, cabecera string
		llamadas            int
	}{
		{"admin.example.invalid", "admin.example.invalid", 1},
		{"admin.example.invalid:8444", "admin.example.invalid:8444", 1},
		{"admin.example.invalid:8444", "admin.example.invalid", 0},
		{"admin.example.invalid:8444", "admin.example.invalid:8443", 0},
		{"admin.example.invalid:8444", "admin.example.invalid:443", 0},
		{"admin.example.invalid", "admin.example.invalid:8444", 0},
		{"", "admin.example.invalid", 0},
		{"otro.example.invalid:8444", "otro.example.invalid:8444", 0},
		{"admin.example.invalid:", "admin.example.invalid:", 0},
		{"admin.example.invalid:84a4", "admin.example.invalid:84a4", 0},
		{"admin.example.invalid:8444:1", "admin.example.invalid:8444:1", 0},
		{"[admin.example.invalid]:8444", "[admin.example.invalid]:8444", 0},
	} {
		o := observacionTLS(estado, ahora)
		o.Autoridad = caso.autoridad
		cuentas := &cuentasDenegadas{}
		e := estado
		p := &Proveedor{config: configADMINPrueba(ahora), deps: Dependencias{Cuentas: cuentas, Registro: registroDenegado{}, Reloj: relojPrueba{ahora}}}
		_, err := p.Resolver(context.Background(), &http.Request{Host: caso.cabecera, TLS: &e}, o)
		if cuentas.llamadas != caso.llamadas || (caso.llamadas == 0) != errors.Is(err, api.ErrAutenticacionRequerida) {
			t.Fatalf("autoridad %q Host %q: consultas=%d error=%v", caso.autoridad, caso.cabecera, cuentas.llamadas, err)
		}
	}
}

func configADMINPrueba(ahora time.Time) h.ConfiguracionSuperficie {
	return h.ConfiguracionSuperficie{
		Superficie: h.SuperficieAdministracionPrivilegiada, ZonaRed: h.ZonaRedAdministracion,
		DireccionEscucha: "127.0.0.1:9443", Audiencia: "vec.admin.perfiles.v1", EmisorIdentidad: "https://admin.example.invalid",
		RedesPermitidas: []string{"127.0.0.0/8"}, DuracionMaximaAsercion: time.Minute,
		EdadMaximaAutenticacion: time.Minute, MetodosAdmitidos: []h.MetodoAutenticacion{h.MetodoCertificado},
		FactoresRequeridos: []h.MetodoAutenticacion{h.MetodoCertificado}, MinimoFactoresVerificados: 1,
		MinimoGruposCriptograficosDistintos: 1, GarantiaMinima: domain.AuthAssuranceHigh,
		RequiereCuentaPrivilegiada: true, CertificadoClienteDirecto: true,
		PoliticaAdministracion:           h.PoliticaAdministracionCertificadoTemporal,
		RetiradaPoliticaAdministracionEn: ahora.Add(time.Hour).Truncate(time.Second),
	}
}

func observacionTLS(e tls.ConnectionState, ahora time.Time) ObservacionADMIN {
	certHash, caHash := sha256.Sum256(e.VerifiedChains[0][0].Raw), sha256.Sum256(e.VerifiedChains[0][1].Raw)
	return ObservacionADMIN{Entorno: "desarrollo", Host: "admin.example.invalid", Autoridad: "admin.example.invalid", Audiencia: "vec.admin.perfiles.v1",
		CertificadoSHA256: hex.EncodeToString(certHash[:]), CASHA256: hex.EncodeToString(caHash[:]),
		AutenticacionVerificadaEn: ahora, RevocacionVerificadaEn: ahora,
		CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: e.VerifiedChains[0][0].NotAfter.UTC()}
}

func estadoTLSReal(t *testing.T, ahora time.Time) tls.ConnectionState {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA sintética"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, key)
	if err != nil {
		t.Fatal(err)
	}
	ca, err = x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	crear := func(serial int64, uso x509.ExtKeyUsage) tls.Certificate {
		t.Helper()
		p, k, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), DNSNames: []string{"admin.example.invalid"},
			NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature,
			ExtKeyUsage: []x509.ExtKeyUsage{uso}}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, ca, p, key)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: k}
	}
	servidorCert, clienteCert := crear(2, x509.ExtKeyUsageServerAuth), crear(3, x509.ExtKeyUsageClientAuth)
	srv, cli := net.Pipe()
	t.Cleanup(func() { _ = srv.Close(); _ = cli.Close() })
	_ = srv.SetDeadline(time.Now().Add(5 * time.Second))
	_ = cli.SetDeadline(time.Now().Add(5 * time.Second))
	servidor := tls.Server(srv, &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{servidorCert},
		ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool})
	cliente := tls.Client(cli, &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{clienteCert},
		RootCAs: pool, ServerName: "admin.example.invalid"})
	fin := make(chan error, 1)
	go func() { fin <- cliente.Handshake() }()
	if err := servidor.Handshake(); err != nil {
		t.Fatal(err)
	}
	if err := <-fin; err != nil {
		t.Fatal(err)
	}
	return servidor.ConnectionState()
}
