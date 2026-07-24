package interna

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func construirServidorInternoPrueba(
	t *testing.T,
	cfg Configuracion,
	api http.Handler,
) (*ServidorInterno, error) {
	t.Helper()
	certPEM, err := os.ReadFile(cfg.CertificadoServidorTLS)
	if err != nil {
		return nil, ErrTLSMutuoNoVerificado
	}
	clavePEM, err := os.ReadFile(cfg.ClaveServidorTLS)
	if err != nil {
		return nil, ErrTLSMutuoNoVerificado
	}
	caPEM, err := os.ReadFile(cfg.AutoridadClientesTLS)
	if err != nil {
		return nil, ErrTLSMutuoNoVerificado
	}
	material, err := materializarTLS(cfg, certPEM, clavePEM, caPEM)
	if err != nil {
		return nil, err
	}
	return construirServidorInternoConMaterial(cfg, api, material)
}

type opcionesCertificadoServidor struct {
	expirado, futuro, sinServerAuth, sinSAN, sanDNSAjeno, sanIP, caComoHoja, sinRaiz, anclaIntermedia bool
}

type materialTLSMutuo struct {
	cfg            Configuracion
	cliente        tls.Certificate
	raicesServidor *x509.CertPool
	raicesClientes *x509.CertPool
	caClientes     *x509.Certificate
	configServidor *tls.Config
}

func materialTLSMutuoPrueba(t *testing.T, opciones opcionesCertificadoServidor) materialTLSMutuo {
	t.Helper()
	ahora := time.Now()
	crearCA := func(serial int64, nombre string) (*x509.Certificate, ed25519.PrivateKey, []byte) {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, publica, privada)
		if err != nil {
			t.Fatal(err)
		}
		certificado, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		return certificado, privada, der
	}
	caServidor, claveCAServidor, derCAServidor := crearCA(1, "CA servidor interna")
	caClientes, claveCAClientes, derCAClientes := crearCA(2, "CA clientes interna")
	emisorServidor, claveEmisorServidor, derEmisorServidor := caServidor, claveCAServidor, derCAServidor
	if opciones.anclaIntermedia {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(20), Subject: pkix.Name{CommonName: "CA intermedia servidor"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(12 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, caServidor, publica, claveCAServidor)
		if err != nil {
			t.Fatal(err)
		}
		emisorServidor, err = x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		claveEmisorServidor, derEmisorServidor = privada, der
	}

	publicaServidor, claveServidor, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	noAntes, noDespues := ahora.Add(-time.Hour), ahora.Add(time.Hour)
	if opciones.expirado {
		noAntes, noDespues = ahora.Add(-2*time.Hour), ahora.Add(-time.Hour)
	}
	if opciones.futuro {
		noAntes, noDespues = ahora.Add(time.Hour), ahora.Add(2*time.Hour)
	}
	plantillaServidor := &x509.Certificate{SerialNumber: big.NewInt(3), Subject: pkix.Name{CommonName: "servidor interno"}, NotBefore: noAntes, NotAfter: noDespues, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, DNSNames: []string{"servidor.interna.test"}}
	if opciones.sinServerAuth {
		plantillaServidor.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	if opciones.sinSAN {
		plantillaServidor.DNSNames = nil
	}
	if opciones.sanDNSAjeno {
		plantillaServidor.DNSNames = []string{"ajeno.test"}
	}
	if opciones.sanIP {
		plantillaServidor.DNSNames = nil
		plantillaServidor.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	}
	padre, clavePadre := emisorServidor, claveEmisorServidor
	if opciones.caComoHoja {
		plantillaServidor.IsCA = true
		plantillaServidor.BasicConstraintsValid = true
		plantillaServidor.KeyUsage |= x509.KeyUsageCertSign
	}
	derServidor, err := x509.CreateCertificate(rand.Reader, plantillaServidor, padre, publicaServidor, clavePadre)
	if err != nil {
		t.Fatal(err)
	}

	crearFinal := func(serial int64, nombre string, ca *x509.Certificate, claveCA ed25519.PrivateKey, derCA []byte) tls.Certificate {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, ca, publica, claveCA)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: privada}
	}
	cliente := crearFinal(4, "cliente interno", caClientes, claveCAClientes, derCAClientes)

	directorio := t.TempDir()
	cfg := configuracionInternaValidaPrueba()
	cfg.DireccionEscucha = "127.0.0.1:8443"
	cfg.RedesPermitidas = []string{"127.0.0.0/8"}
	cfg.NombreServidorTLS = "servidor.interna.test"
	if opciones.sanIP {
		cfg.NombreServidorTLS = "127.0.0.1"
	}
	cfg.CertificadoServidorTLS = filepath.Join(directorio, "servidor.crt")
	cfg.ClaveServidorTLS = filepath.Join(directorio, "servidor.key")
	cfg.AutoridadClientesTLS = filepath.Join(directorio, "clientes-ca.crt")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derServidor})
	if !opciones.sinRaiz {
		certPEM = append(certPEM, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derEmisorServidor})...)
	}
	claveDER, err := x509.MarshalPKCS8PrivateKey(claveServidor)
	if err != nil {
		t.Fatal(err)
	}
	clavePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: claveDER})
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derCAClientes})
	if err := os.WriteFile(cfg.CertificadoServidorTLS, certPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.ClaveServidorTLS, clavePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.AutoridadClientesTLS, caPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	raicesServidor := x509.NewCertPool()
	raicesServidor.AddCert(caServidor)
	raicesClientes := x509.NewCertPool()
	raicesClientes.AddCert(caClientes)
	parServidor, err := tls.X509KeyPair(certPEM, clavePEM)
	if err != nil && !opciones.sinRaiz {
		t.Fatal(err)
	}
	return materialTLSMutuo{
		cfg: cfg, cliente: cliente, raicesServidor: raicesServidor, raicesClientes: raicesClientes, caClientes: caClientes,
		configServidor: &tls.Config{Certificates: []tls.Certificate{parServidor}},
	}
}

func (m materialTLSMutuo) configCliente(protocolos []string, conCertificado bool) *tls.Config {
	cfg := &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: m.raicesServidor, ServerName: m.cfg.NombreServidorTLS, NextProtos: protocolos}
	if conCertificado {
		cfg.Certificates = []tls.Certificate{m.cliente}
	}
	return cfg
}

func estadoTLSMutuoValidoPrueba(t *testing.T, material materialTLSMutuo) *tls.ConnectionState {
	t.Helper()
	pares := make([]*x509.Certificate, len(material.cliente.Certificate))
	for indice, der := range material.cliente.Certificate {
		certificado, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		pares[indice] = certificado
	}
	return &tls.ConnectionState{
		Version:                    tls.VersionTLS13,
		HandshakeComplete:          true,
		CipherSuite:                tls.TLS_AES_128_GCM_SHA256,
		CurveID:                    tls.X25519,
		NegotiatedProtocol:         protocoloALPNHTTPUno,
		NegotiatedProtocolIsMutual: true,
		ServerName:                 material.cfg.NombreServidorTLS,
		PeerCertificates:           pares,
		VerifiedChains:             [][]*x509.Certificate{pares},
	}
}

type manejadorPunteroPrueba struct{}

func (*manejadorPunteroPrueba) ServeHTTP(http.ResponseWriter, *http.Request) {}
