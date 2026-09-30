package httpapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteGobiernoRPTPrueba struct {
	cred       application.CredencialesGobiernoCategoriaRPT
	descriptor ports.DescriptorCatalogoRPT
	perfil     string
	err        error
	llamadas   int
}

func (f *fuenteGobiernoRPTPrueba) ResolverGobiernoCategoriaRPT(_ context.Context, _ *x509.Certificate) (application.CredencialesGobiernoCategoriaRPT, ports.DescriptorCatalogoRPT, string, error) {
	f.llamadas++
	return f.cred, f.descriptor, f.perfil, f.err
}

type auditorGobiernoRPTPrueba struct {
	codigos []string
	err     error
}

func (a *auditorGobiernoRPTPrueba) RegistrarDenegacionGobiernoCategoriaRPT(_ context.Context, ruta, codigo, correlacion string) error {
	if !rutaGobiernoRPTValida(ruta) || correlacion == "" {
		return errors.New("auditoria invalida")
	}
	a.codigos = append(a.codigos, codigo)
	return a.err
}

type operadorGobiernoRPTPrueba struct {
	llamadas  int
	respuesta ports.ResultadoGobiernoCategoriaRPT
}

func (o *operadorGobiernoRPTPrueba) Proponer(context.Context, application.OrdenProponerGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	o.llamadas++
	return o.respuesta, nil
}
func (o *operadorGobiernoRPTPrueba) Aprobar(context.Context, application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	o.llamadas++
	return o.respuesta, nil
}
func (o *operadorGobiernoRPTPrueba) Confirmar(context.Context, application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	o.llamadas++
	return o.respuesta, nil
}

func caGobiernoRPTPrueba(t *testing.T) (*x509.CertPool, tls.Certificate, tls.Certificate) {
	t.Helper()
	claveCA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now()
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA ADMIN sintética"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	derCA, err := x509.CreateCertificate(rand.Reader, ca, ca, &claveCA.PublicKey, claveCA)
	if err != nil {
		t.Fatal(err)
	}
	caParseada, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	claveCliente, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cliente := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "actor sintético ADMIN"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	derCliente, err := x509.CreateCertificate(rand.Reader, cliente, caParseada, &claveCliente.PublicKey, claveCA)
	if err != nil {
		t.Fatal(err)
	}
	clavePEM, err := x509.MarshalECPrivateKey(claveCliente)
	if err != nil {
		t.Fatal(err)
	}
	p := x509.NewCertPool()
	p.AddCert(caParseada)
	material, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derCliente}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: clavePEM}))
	if err != nil {
		t.Fatal(err)
	}
	claveServidor, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	servidor := &x509.Certificate{SerialNumber: big.NewInt(3), Subject: pkix.Name{CommonName: "admin.ejemplo.test"}, DNSNames: []string{"admin.ejemplo.test"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	derServidor, err := x509.CreateCertificate(rand.Reader, servidor, caParseada, &claveServidor.PublicKey, claveCA)
	if err != nil {
		t.Fatal(err)
	}
	claveServidorPEM, err := x509.MarshalECPrivateKey(claveServidor)
	if err != nil {
		t.Fatal(err)
	}
	materialServidor, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derServidor}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: claveServidorPEM}))
	if err != nil {
		t.Fatal(err)
	}
	return p, material, materialServidor
}

func TestGobiernoRPTConstructorDeniegaSinFronteraADMIN(t *testing.T) {
	op, fuente, audit := &operadorGobiernoRPTPrueba{}, &fuenteGobiernoRPTPrueba{}, &auditorGobiernoRPTPrueba{}
	d := ports.DescriptorCatalogoRPT{CatalogoID: "catalogo.rpt", ModuloID: "bolsa"}
	ca, _, _ := caGobiernoRPTPrueba(t)
	for _, tc := range []struct {
		name       string
		op         OperadorGobiernoCategoriaRPT
		fuente     FuenteCredencialesGobiernoCategoriaRPT
		audit      AuditorDenegacionGobiernoCategoriaRPT
		host       string
		ca         *x509.CertPool
		descriptor ports.DescriptorCatalogoRPT
		perfil     string
	}{
		{"operador", nil, fuente, audit, "admin.ejemplo.test", ca, d, "perfil-fijo"},
		{"fuente", op, nil, audit, "admin.ejemplo.test", ca, d, "perfil-fijo"},
		{"auditoria", op, fuente, nil, "admin.ejemplo.test", ca, d, "perfil-fijo"},
		{"CA", op, fuente, audit, "admin.ejemplo.test", nil, d, "perfil-fijo"},
		{"host", op, fuente, audit, "vec.ejemplo.test:8443", ca, d, "perfil-fijo"},
		{"descriptor", op, fuente, audit, "admin.ejemplo.test", ca, ports.DescriptorCatalogoRPT{}, "perfil-fijo"},
		{"perfil", op, fuente, audit, "admin.ejemplo.test", ca, d, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rutas, err := NuevasRutasGobiernoCategoriaRPT(tc.op, tc.fuente, tc.audit, tc.host, tc.ca, tc.descriptor, tc.perfil)
			if !errors.Is(err, ErrHandlerGobiernoCategoriaRPTInvalido) || rutas != nil {
				t.Fatalf("constructor: rutas=%v error=%v", rutas, err)
			}
		})
	}
}

func TestGobiernoRPTMTLSADMINDeniegaPortalNormalYFuenteSinPerfil(t *testing.T) {
	ca, cert, certServidor := caGobiernoRPTPrueba(t)
	op, fuente, audit := &operadorGobiernoRPTPrueba{}, &fuenteGobiernoRPTPrueba{err: ErrAccesoRutaExactaDenegado}, &auditorGobiernoRPTPrueba{}
	d := ports.DescriptorCatalogoRPT{CatalogoID: "catalogo.rpt", ModuloID: "bolsa"}
	actor := actorOrganizacionHistoricaPrueba(t)
	rutas, err := NuevasRutasGobiernoCategoriaRPT(op, fuente, audit, "admin.ejemplo.test", ca, d, actor.PerfilActivoRef)
	if err != nil || len(rutas) != 3 {
		t.Fatalf("rutas=%d error=%v", len(rutas), err)
	}
	mux := http.NewServeMux()
	for _, ruta := range rutas {
		mux.Handle(ruta.Ruta, ruta.Manejador)
	}
	servidor := httptest.NewUnstartedServer(mux)
	servidor.TLS = &tls.Config{Certificates: []tls.Certificate{certServidor}, ClientAuth: tls.VerifyClientCertIfGiven, ClientCAs: ca, MinVersion: tls.VersionTLS12}
	servidor.StartTLS()
	defer servidor.Close()
	cliente := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: ca, Certificates: []tls.Certificate{cert}, ServerName: "admin.ejemplo.test", MinVersion: tls.VersionTLS12}}}
	sinCert := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: ca, ServerName: "admin.ejemplo.test", MinVersion: tls.VersionTLS12}}}
	clave := "12345678-1234-4234-8234-123456789abc"
	cuerpo := `{"propuesta_ref":"propuesta:ejemplo","catalogo_id":"catalogo.rpt","modulo_id":"bolsa","revision_esperada":1,"huella_sha256":"` + strings.Repeat("a", 64) + `"}`
	enviar := func(host, cuerpoEnvio string, c *http.Client) int {
		t.Helper()
		r, err := http.NewRequest(http.MethodPost, servidor.URL+RutaAprobarGobiernoCategoriaRPT, strings.NewReader(cuerpoEnvio))
		if err != nil {
			t.Fatal(err)
		}
		r.Host = host
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", clave)
		resp, err := c.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if estado := enviar("admin.ejemplo.test", cuerpo, sinCert); estado != http.StatusUnauthorized {
		t.Fatalf("sin certificado=%d", estado)
	}
	if estado := enviar("vec.ejemplo.test", cuerpo, cliente); estado != http.StatusUnauthorized {
		t.Fatalf("portal ordinario=%d", estado)
	}
	if estado := enviar("admin.ejemplo.test", cuerpo, cliente); estado != http.StatusForbidden {
		t.Fatalf("fuente sin perfil=%d", estado)
	}
	if estado := enviar("admin.ejemplo.test", `{"actor":"falso"}`, cliente); estado != http.StatusForbidden {
		t.Fatalf("actor del cuerpo con fuente denegada=%d", estado)
	}
	fuente.err = nil
	fuente.descriptor = d
	fuente.perfil = actor.PerfilActivoRef
	fuente.cred.Actor = actor
	fuente.cred.Actor.Principal.AuthAssurance = domain.AuthAssuranceSubstantial
	if estado := enviar("admin.ejemplo.test", cuerpo, cliente); estado != http.StatusForbidden {
		t.Fatalf("perfil substantial=%d", estado)
	}
	if op.llamadas != 0 || fuente.llamadas != 3 || len(audit.codigos) != 5 {
		t.Fatalf("efecto=%d fuente=%d auditoria=%v", op.llamadas, fuente.llamadas, audit.codigos)
	}
}

func TestGobiernoRPTDTORechazaAutoridadYHuellasCliente(t *testing.T) {
	for _, tc := range []struct{ ruta, campo string }{
		{RutaProponerGobiernoCategoriaRPT, "actor"}, {RutaProponerGobiernoCategoriaRPT, "perfil_activo_ref"},
		{RutaProponerGobiernoCategoriaRPT, "auth_assurance"}, {RutaProponerGobiernoCategoriaRPT, "preimagenes_huella_sha256"},
		{RutaProponerGobiernoCategoriaRPT, "documento_huella_sha256"}, {RutaProponerGobiernoCategoriaRPT, "huella_sha256"},
		{RutaAprobarGobiernoCategoriaRPT, "fuente_ref"}, {RutaConfirmarGobiernoCategoriaRPT, "motivo_ref"},
	} {
		t.Run(tc.campo, func(t *testing.T) {
			if camposGobiernoRPTAdmitidos(tc.ruta, map[string]json.RawMessage{tc.campo: json.RawMessage(`null`)}) {
				t.Fatal("campo de autoridad admitido")
			}
		})
	}
	for _, cuerpo := range []string{
		`{"propuesta_ref":"propuesta:ejemplo","actor":"falso"}`,
		`{"propuesta_ref":"propuesta:ejemplo","perfil_activo_ref":"falso"}`,
		`{"propuesta_ref":"propuesta:ejemplo","huella_sha256":null}`,
		`{"propuesta_ref":"propuesta:ejemplo","propuesta_ref":"otra"}`,
	} {
		r := httptest.NewRequest(http.MethodPost, RutaProponerGobiernoCategoriaRPT, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "12345678-1234-4234-8234-123456789abc")
		if _, _, err := leerEntradaGobiernoRPT(httptest.NewRecorder(), r); !errors.Is(err, ErrHandlerGobiernoCategoriaRPTInvalido) {
			t.Fatalf("entrada de autoridad admitida: %s", cuerpo)
		}
	}
}
