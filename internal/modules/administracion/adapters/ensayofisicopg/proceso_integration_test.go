package ensayofisicopg

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
	"os"
	"path/filepath"
	"testing"
	"time"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

func TestMain(m *testing.M) {
	if c, ok := ManejarModoInterno(os.Args[1:], os.Stdin, os.Stdout); ok {
		os.Exit(c)
	}
	if len(os.Args) == 2 && os.Args[1] == "--testigo-http-sintetico" {
		os.Exit(servirTestigo())
	}
	os.Exit(m.Run())
}
func servirTestigo() int {
	if comprobarNSSDentro() != nil {
		return 1
	}
	certificado := os.Getenv("VEC_TESTIGO_CERTIFICADO")
	clave := os.Getenv("VEC_TESTIGO_CLAVE")
	ca, err := os.ReadFile(certificado)
	if err != nil {
		return 1
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return 1
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/consulta", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer testigo-sintetico" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"principal":"testigo_sintetico","permissions":["consulta_sintetica"]}`))
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.invalid/", http.StatusFound)
	})
	servidor := &http.Server{Addr: "127.0.0.1:18443", Handler: mux, ReadHeaderTimeout: time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool}}
	if servidor.ListenAndServeTLS(certificado, clave) != nil {
		return 1
	}
	return 0
}
func archivosTLS(t *testing.T) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "testigo-sintetico"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, BasicConstraintsValid: true, IsCA: true, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privada, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cert, priv := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600) != nil || os.WriteFile(priv, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privada}), 0600) != nil {
		t.Fatal("crear TLS sintético")
	}
	return cert, priv
}
func comprobarProceso(ctx context.Context, entorno puertos.Entorno) error {
	var bin, cert, key string
	for _, c := range entorno.Componentes {
		switch c.ID {
		case "fisica:servidor":
			bin = c.ID
		case "fisica:certificado":
			cert = c.RutaInterna
		case "fisica:clave":
			key = c.RutaInterna
		}
	}
	proceso, err := entorno.Archivado.IniciarArchivado(ctx, bin, []string{"--testigo-http-sintetico"}, map[string]string{"VEC_TESTIGO_CERTIFICADO": cert, "VEC_TESTIGO_CLAVE": key})
	if err != nil {
		return err
	}
	s := puertos.SondaHTTP{Puerto: 18443, Ruta: "/livez", TLS: true, CA: cert, Certificado: cert, Clave: key, LimiteBytes: 4096}
	var respuesta puertos.RespuestaHTTP
	for ctx.Err() == nil {
		respuesta, err = proceso.SondarHTTP(ctx, s)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	if respuesta.EstadoHTTP != 204 {
		return errRuntime
	}
	sinCert := s
	sinCert.Certificado, sinCert.Clave = "", ""
	if _, err = proceso.SondarHTTP(ctx, sinCert); err == nil {
		return errRuntime
	}
	s.Ruta = "/consulta"
	respuesta, err = proceso.SondarHTTP(ctx, s)
	if err != nil || respuesta.EstadoHTTP != 403 {
		return errRuntime
	}
	s.Authorization = "Bearer testigo-sintetico"
	respuesta, err = proceso.SondarHTTP(ctx, s)
	if err != nil || respuesta.EstadoHTTP != 200 || len(respuesta.Contenido) == 0 {
		return errRuntime
	}
	s.Ruta = "/redirect"
	respuesta, err = proceso.SondarHTTP(ctx, s)
	if err != nil || respuesta.EstadoHTTP != 302 {
		return errRuntime
	}
	return proceso.Detener(ctx)
}
