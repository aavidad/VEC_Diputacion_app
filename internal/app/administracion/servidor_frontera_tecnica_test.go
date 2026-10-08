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
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

func materialServidorFronteraPrueba(t *testing.T) (Configuracion, tls.Certificate, *x509.CertPool, func(bool)) {
	t.Helper()
	ahora := time.Now()
	caClave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caModelo := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA ADMIN prueba"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, SubjectKeyId: []byte{1, 2, 3},
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
	crearHoja := func(serie int64, uso x509.ExtKeyUsage) (tls.Certificate, []byte, []byte) {
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
	_, servidorPEM, clavePEM := crearHoja(2, x509.ExtKeyUsageServerAuth)
	cliente, _, _ := crearHoja(3, x509.ExtKeyUsageClientAuth)
	hoja, err := x509.ParseCertificate(cliente.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if !cadenaDirectaVigente([]*x509.Certificate{hoja, ca}, ca, ahora) {
		t.Fatal("cadena directa valida rechazada")
	}
	if cadenaDirectaVigente([]*x509.Certificate{hoja, ca, ca}, ca, ahora) {
		t.Fatal("CA intermedia aceptada")
	}
	if cadenaDirectaVigente([]*x509.Certificate{hoja, ca}, ca, ahora.Add(2*time.Hour)) {
		t.Fatal("conexion antigua conserva hoja caducada")
	}
	dir := t.TempDir()
	escribir := func(nombre string, datos []byte) string {
		ruta := filepath.Join(dir, nombre)
		if err := os.WriteFile(ruta, datos, 0o600); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	rutaCRL := filepath.Join(dir, "admin.crl")
	actualizarCRL := func(revocar bool) {
		lista := &x509.RevocationList{Number: big.NewInt(1), ThisUpdate: ahora.Add(-time.Minute), NextUpdate: ahora.Add(time.Hour)}
		if revocar {
			lista.RevokedCertificateEntries = []x509.RevocationListEntry{{SerialNumber: big.NewInt(3), RevocationTime: ahora}}
		}
		der, err := x509.CreateRevocationList(rand.Reader, lista, ca, caClave)
		if err != nil {
			t.Fatal(err)
		}
		escribir("admin.crl", pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: der}))
	}
	actualizarCRL(false)
	cfg := Configuracion{
		Entorno: "cidonia", Escucha: "127.0.0.1:19443", Host: "admin.example.test",
		Audiencia: "vec-admin-prueba", EmisorIdentidad: "https://identidad.example.test",
		CertificadoServidor: escribir("servidor.crt", servidorPEM),
		ClaveServidor:       escribir("servidor.key", clavePEM),
		CAAdministracion:    escribir("ca.crt", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})),
		CRLAdministracion:   rutaCRL, RedesPermitidas: []string{"0.0.0.0/0", "::/0"},
		RetiradaEn: ahora.Add(time.Hour).UTC().Truncate(time.Second),
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	return cfg, cliente, raices, actualizarCRL
}

type auditorServidorFronteraPrueba struct {
	eventos []api.DenegacionADMIN
	refs    []string
	err     error
}

func (a *auditorServidorFronteraPrueba) RegistrarDenegacionADMIN(ctx context.Context, d api.DenegacionADMIN) error {
	ref, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return err
	}
	r, err := ref.ValorCanonico()
	if err != nil {
		return err
	}
	a.eventos = append(a.eventos, d)
	a.refs = append(a.refs, r)
	return a.err
}
func TestServidorAuditaRechazosHTTPAntesDeNegocio(t *testing.T) {
	cfg, cliente, raices, revocar := materialServidorFronteraPrueba(t)
	for _, caso := range []string{"host", "red", "revocado", "crl_no_disponible", "fallo_auditoria"} {
		t.Run(caso, func(t *testing.T) {
			aud := &auditorServidorFronteraPrueba{}
			c := cfg
			revocar(false)
			if caso == "red" {
				c.RedesPermitidas = []string{"192.0.2.0/24"}
			}
			if caso == "revocado" {
				revocar(true)
			}
			if caso == "crl_no_disponible" {
				c.CRLAdministracion = filepath.Join(t.TempDir(), "ausente.crl")
			}
			if caso == "fallo_auditoria" {
				aud.err = errors.New("sin_acuse")
			}
			llamadas := 0
			h := &handlerPerfilesADMIN{auditor: aud, api: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { llamadas++; w.WriteHeader(200) }), rutas: map[string]string{}}
			s, err := nuevoServidor(c, h)
			if err != nil {
				t.Fatal(err)
			}
			web := httptest.NewUnstartedServer(s.Handler)
			web.TLS = s.TLSConfig.Clone()
			web.StartTLS()
			defer web.Close()
			transporte := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cliente}}}
			defer transporte.CloseIdleConnections()
			req, err := http.NewRequest(http.MethodGet, web.URL+"/api/admin/perfiles/v1/personas?SECRET_query", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Host = c.Host
			if caso == "host" || caso == "fallo_auditoria" {
				req.Host = "ajeno.example.test"
			}
			req.Header.Set("X-Correlation-ID", strings.Repeat("f", 32))
			res, err := (&http.Client{Transport: transporte}).Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			esperado := 403
			if caso == "fallo_auditoria" {
				esperado = 503
			}
			if res.StatusCode != esperado || llamadas != 0 || len(aud.eventos) != 1 || aud.eventos[0].SesionResuelta || aud.eventos[0].Actor.PersonaRef != "" || aud.eventos[0].RecursoRef != "" || strings.Contains(string(body), "SECRET") || res.Header.Get("Set-Cookie") != "" || res.Header.Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("frontera_sin_auditoria_o_datos_expuestos")
			}
			if aud.refs[0] == "correlacion_"+strings.Repeat("f", 32) {
				t.Fatal("correlacion_de_cabecera")
			}
		})
	}
}
