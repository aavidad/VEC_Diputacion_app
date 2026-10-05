package administracion

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
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type relojFijoReanudacion struct{ t time.Time }

func (r relojFijoReanudacion) Ahora() time.Time { return r.t }

// montarFronteraReanudacion crea CA, hojas y CRL sintéticas y devuelve la
// configuración ADMIN, las raíces y el certificado de cliente.
func montarFronteraReanudacion(t *testing.T) (Configuracion, *x509.CertPool, tls.Certificate) {
	t.Helper()
	ahora := time.Now()
	caClave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caModelo := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA ADMIN reanudacion"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, SubjectKeyId: []byte{4, 5, 6},
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caModelo, caModelo, &caClave.PublicKey, caClave)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	hoja := func(serie int64, uso x509.ExtKeyUsage) (tls.Certificate, []byte, []byte) {
		clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		modelo := &x509.Certificate{
			SerialNumber: big.NewInt(serie), Subject: pkix.Name{CommonName: "identidad sintetica"},
			NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso},
			DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		}
		der, err := x509.CreateCertificate(rand.Reader, modelo, ca, &clave.PublicKey, caClave)
		if err != nil {
			t.Fatal(err)
		}
		claveDER, err := x509.MarshalECPrivateKey(clave)
		if err != nil {
			t.Fatal(err)
		}
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		clavePEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: claveDER})
		par, err := tls.X509KeyPair(certPEM, clavePEM)
		if err != nil {
			t.Fatal(err)
		}
		return par, certPEM, clavePEM
	}
	_, servidorPEM, servidorClave := hoja(2, x509.ExtKeyUsageServerAuth)
	cliente, _, _ := hoja(3, x509.ExtKeyUsageClientAuth)
	dir := t.TempDir()
	escribir := func(nombre string, datos []byte) string {
		ruta := filepath.Join(dir, nombre)
		if err := os.WriteFile(ruta, datos, 0o600); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	crl, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number: big.NewInt(1), ThisUpdate: ahora.Add(-time.Minute), NextUpdate: ahora.Add(time.Hour),
	}, ca, caClave)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	return Configuracion{
		Entorno: "cidonia", Escucha: "127.0.0.1:19444", Host: "admin.example.test",
		Audiencia: "vec-admin-prueba", EmisorIdentidad: "https://identidad.example.test",
		CertificadoServidor: escribir("servidor.crt", servidorPEM),
		ClaveServidor:       escribir("servidor.key", servidorClave),
		CAAdministracion:    escribir("ca.crt", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})),
		CRLAdministracion:   escribir("admin.crl", pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: crl})),
		RedesPermitidas:     []string{"0.0.0.0/0", "::/0"},
		RetiradaEn:          ahora.Add(time.Hour).UTC().Truncate(time.Second),
	}, raices, cliente
}

func TestServidorAdminSinTicketsYConexionOciosaAcotada(t *testing.T) {
	cfg, _, _ := montarFronteraReanudacion(t)
	servidor, err := NuevoServidor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !servidor.TLSConfig.SessionTicketsDisabled {
		t.Fatal("la frontera ADMIN emite tickets de sesión que luego rechaza")
	}
	if servidor.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert || servidor.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatal("guarda mTLS relajada")
	}
	if servidor.IdleTimeout <= 0 || servidor.IdleTimeout >= vidaAutenticacionConexionPerfiles {
		t.Fatalf("IdleTimeout=%v no queda por debajo de la vida autenticada %v", servidor.IdleTimeout, vidaAutenticacionConexionPerfiles)
	}
	if servidor.ReadTimeout <= 0 || servidor.WriteTimeout <= 0 {
		t.Fatal("petición sin límite de lectura o escritura")
	}
	// Una petición que entra justo antes del umbral de renovación debe
	// terminar y dejar paso a la siguiente dentro de la vida autenticada.
	if renovacionConexionPerfiles+servidor.WriteTimeout+servidor.IdleTimeout+servidor.ReadHeaderTimeout >= vidaAutenticacionConexionPerfiles {
		t.Fatal("sin margen entre renovación, inactividad y caducidad")
	}
}

func TestServidorAdminSegundaConexionNoSeReanuda(t *testing.T) {
	cfg, raices, cliente := montarFronteraReanudacion(t)
	servidor, err := NuevoServidor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	pedir := func(url string, cache tls.ClientSessionCache) (int, bool) {
		t.Helper()
		transporte := &http.Transport{TLSClientConfig: &tls.Config{
			RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13,
			Certificates: []tls.Certificate{cliente}, ClientSessionCache: cache,
		}}
		defer transporte.CloseIdleConnections()
		req, err := http.NewRequest(http.MethodGet, url+"/livez", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = cfg.Host
		resp, err := (&http.Client{Transport: transporte}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode, resp.TLS != nil && resp.TLS.DidResume
	}
	arrancar := func(config *tls.Config) *httptest.Server {
		prueba := httptest.NewUnstartedServer(servidor.Handler)
		prueba.TLS = config
		prueba.StartTLS()
		t.Cleanup(prueba.Close)
		return prueba
	}

	prueba := arrancar(servidor.TLSConfig.Clone())
	cache := tls.NewLRUClientSessionCache(4)
	for i := 0; i < 2; i++ {
		codigo, reanudada := pedir(prueba.URL, cache)
		if codigo != http.StatusNoContent || reanudada {
			t.Fatalf("conexión %d: código=%d reanudada=%v", i+1, codigo, reanudada)
		}
	}

	// Control: con tickets la segunda conexión se reanuda y la guarda la
	// rechaza. Demuestra que la prueba anterior distingue ambos casos.
	conTickets := servidor.TLSConfig.Clone()
	conTickets.SessionTicketsDisabled = false
	control := arrancar(conTickets)
	cacheControl := tls.NewLRUClientSessionCache(4)
	if codigo, _ := pedir(control.URL, cacheControl); codigo != http.StatusNoContent {
		t.Fatalf("control primera conexión: %d", codigo)
	}
	if codigo, reanudada := pedir(control.URL, cacheControl); !reanudada || codigo != http.StatusForbidden {
		t.Fatalf("control con tickets: código=%d reanudada=%v", codigo, reanudada)
	}
}

func TestServidorAdminCierraKeepAliveCercaDeCaducar(t *testing.T) {
	cfg, _, _ := montarFronteraReanudacion(t)
	ahora := time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	perfiles := &handlerPerfilesADMIN{auditor: &auditorActivosPrueba{}, reloj: relojFijoReanudacion{ahora}, rutas: map[string]string{}}
	servidor, err := nuevoServidor(cfg, perfiles)
	if err != nil {
		t.Fatal(err)
	}
	conEdad := func(edad time.Duration) string {
		ctx := context.WithValue(context.Background(), claveConexionPerfiles{}, conexionPerfiles{aceptadaEn: ahora.Add(-edad)})
		req := httptest.NewRequest(http.MethodGet, "https://"+cfg.Host+"/livez", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("sin TLS la frontera debe denegar: %d", rec.Code)
		}
		return rec.Header().Get("Connection")
	}
	if got := conEdad(time.Minute); got != "" {
		t.Fatalf("conexión joven cerrada: %q", got)
	}
	if got := conEdad(renovacionConexionPerfiles); got != "close" {
		t.Fatalf("conexión al límite conserva keep-alive: %q", got)
	}
	if got := conEdad(vidaAutenticacionConexionPerfiles); got != "close" {
		t.Fatalf("conexión caducada conserva keep-alive: %q", got)
	}
	if conexionPerfilesPorRenovar(context.Background(), ahora) || conexionPerfilesPorRenovar(nil, ahora) { //nolint:staticcheck // nil se prueba a propósito
		t.Fatal("sin conexión registrada no hay nada que renovar")
	}
}
