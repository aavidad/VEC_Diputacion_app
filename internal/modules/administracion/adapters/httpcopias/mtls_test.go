package httpcopias

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	app "vec-diputacion-granada/internal/modules/administracion/application/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

type autoridadFija struct{ permitir atomic.Bool }

func (a *autoridadFija) AutorizarCopias(ctx context.Context, s p.Sesion, op p.Operacion, ref string) error {
	if ctx.Err() != nil {
		return p.ErrNoDisponible
	}
	if !a.permitir.Load() || s.Actor.PersonaRef != "per_"+strings.Repeat("a", 22) || s.Actor.PerfilActivoRef != "prf_"+strings.Repeat("b", 22) || op != p.Lanzar || ref != "copias" {
		return p.ErrDenegado
	}
	return nil
}

// Real TLS and HTTP exercise the boundary; these callbacks are test fixtures,
// not a production resolver, central authority or installed runtime.
func TestHTTPRealCertificadoNoConcedeOperacion(t *testing.T) {
	now := time.Now()
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	caModel := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "ADMIN isolated test"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, e := x509.CreateCertificate(rand.Reader, caModel, caModel, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	ca, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	clientKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	leafModel := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "synthetic client"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	leafDER, e := x509.CreateCertificate(rand.Reader, leafModel, ca, &clientKey.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	cert := tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: clientKey}
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	ses := sesionPrueba(t)
	a := &autoridadFija{}
	b := &backendTest{}
	var identity atomic.Bool
	identity.Store(true)
	h, e := Nuevo("https://admin.example.test", func(_ context.Context, r *http.Request) (p.Sesion, error) {
		if !identity.Load() || r.TLS == nil || len(r.TLS.PeerCertificates) == 0 || len(r.TLS.VerifiedChains) == 0 || !r.TLS.VerifiedChains[0][0].Equal(r.TLS.PeerCertificates[0]) {
			return p.Sesion{}, p.ErrAutenticacion
		}
		return ses, nil
	},
		func(context.Context, Denegacion) error { return nil }, &app.Servicio{Autoridad: a, Lecturas: b, Cambios: b})
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewUnstartedServer(h)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: roots}
	server.StartTLS()
	defer server.Close()
	serverRoots := x509.NewCertPool()
	serverRoots.AddCert(server.Certificate())
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: serverRoots, Certificates: []tls.Certificate{cert}}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	request := func(body string, forged bool) int {
		t.Helper()
		r, e := http.NewRequest("POST", server.URL+PrefijoV1+"/lanzamientos", strings.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		r.Host = "admin.example.test"
		for k, v := range map[string]string{"Origin": "https://admin.example.test", "Sec-Fetch-Site": "same-origin", "Sec-Fetch-Mode": "cors", "Sec-Fetch-Dest": "empty", "Content-Type": "application/json"} {
			r.Header.Set(k, v)
		}
		if forged {
			r.Header.Set("Authorization", "Bearer forged")
		}
		response, e := client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	if got := request(lanzamiento, false); got != 403 || b.llamadas != 0 {
		t.Fatalf("certificate implicitly authorized: %d", got)
	}
	a.permitir.Store(true)
	identity.Store(false)
	if got := request(lanzamiento, false); got != 401 || b.llamadas != 0 {
		t.Fatalf("no session allowed: %d", got)
	}
	identity.Store(true)
	if got := request(lanzamiento, true); got != 401 || b.llamadas != 0 {
		t.Fatalf("forged header allowed: %d", got)
	}
	if got := request(strings.Replace(lanzamiento, `"tipo":"completa"`, `"tipo":"completa","perfil":"admin"`, 1), false); got != 400 || b.llamadas != 0 {
		t.Fatalf("forged input allowed: %d", got)
	}
	if got := request(lanzamiento, false); got != 200 || b.llamadas != 1 {
		t.Fatalf("fixed callback denied: %d", got)
	}
	a.permitir.Store(false)
	if got := request(lanzamiento, false); got != 403 || b.llamadas != 1 {
		t.Fatalf("revoked callback reused: %d", got)
	}
}
