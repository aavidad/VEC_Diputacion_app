package analisiscontenido

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestServicioMTLSIdentidadAjena(t *testing.T) {
	buscar := func(nombre string) string {
		ruta, err := exec.LookPath(nombre)
		if err != nil {
			t.Skipf("%s no está instalado", nombre)
		}
		return ruta
	}
	const propia = "spiffe://vec.test/analizador-cliente"
	servicio, err := NuevoServicio(ConfiguracionServicio{
		ConectorID: "analizador_test", ClamdSocket: filepath.Join(t.TempDir(), "no-usado.sock"),
		Bwrap: buscar("bwrap"), QPDF: buscar("qpdf"), Prlimit: buscar("prlimit"),
		IdentidadesCliente: []string{propia}, MaxEdadFirmas: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	ca, claveCA, certCA := crearCertificadoCA(t)
	rutaCA := escribirPEMTemporal(t, "ca.pem", ca)
	rutaServidor, claveServidor, _ := crearCertificado(t, certCA, claveCA, true, "")
	configTLS, err := servicio.TLSConfig(rutaServidor, claveServidor, rutaCA)
	if err != nil {
		t.Fatal(err)
	}
	if configTLS.MinVersion != tls.VersionTLS13 || configTLS.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatalf("frontera TLS débil: versión=%d autenticación=%d", configTLS.MinVersion, configTLS.ClientAuth)
	}
	servidor := httptest.NewUnstartedServer(servicio.Handler())
	servidor.TLS = configTLS
	servidor.StartTLS()
	defer servidor.Close()
	casos := []struct {
		nombre, identidad string
		estado            int
	}{
		{"propia", propia, http.StatusNotFound},
		{"ajena", "spiffe://vec.test/otro-cliente", http.StatusForbidden},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rutaCert, rutaClave, _ := crearCertificado(t, certCA, claveCA, false, caso.identidad)
			par, err := tls.LoadX509KeyPair(rutaCert, rutaClave)
			if err != nil {
				t.Fatal(err)
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(ca)
			cliente := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
				RootCAs: pool, Certificates: []tls.Certificate{par}, ServerName: "localhost", MinVersion: tls.VersionTLS13,
			}}}
			defer cliente.CloseIdleConnections()
			peticion, err := http.NewRequestWithContext(context.Background(), http.MethodGet, servidor.URL+"/ruta-inexistente", nil)
			if err != nil {
				t.Fatal(err)
			}
			respuesta, err := cliente.Do(peticion)
			if err != nil {
				t.Fatal(err)
			}
			defer respuesta.Body.Close()
			if respuesta.StatusCode != caso.estado {
				t.Fatalf("estado=%d, esperado=%d", respuesta.StatusCode, caso.estado)
			}
		})
	}
}

func crearCertificadoCA(t *testing.T) ([]byte, *ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now()
	plantilla := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA sintética"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true, IsCA: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), clave, cert
}

func crearCertificado(t *testing.T, ca *x509.Certificate, claveCA *ecdsa.PrivateKey, servidor bool, identidad string) (string, string, *x509.Certificate) {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now()
	plantilla := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "sintético"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	if servidor {
		plantilla.DNSNames = []string{"localhost"}
		plantilla.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		uri, err := url.Parse(identidad)
		if err != nil {
			t.Fatal(err)
		}
		plantilla.URIs = []*url.URL{uri}
		plantilla.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, ca, &clave.PublicKey, claveCA)
	if err != nil {
		t.Fatal(err)
	}
	claveDER, err := x509.MarshalECPrivateKey(clave)
	if err != nil {
		t.Fatal(err)
	}
	rutaCert := escribirPEMTemporal(t, "cert.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	rutaClave := escribirPEMTemporal(t, "clave.pem", pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: claveDER}))
	return rutaCert, rutaClave, plantilla
}

func escribirPEMTemporal(t *testing.T, nombre string, datos []byte) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), nombre)
	if err := os.WriteFile(ruta, datos, 0600); err != nil {
		t.Fatal(err)
	}
	return ruta
}
