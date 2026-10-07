package firmavec

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func estadoTLSRealFirmanteV2Prueba(t *testing.T) tls.ConnectionState {
	t.Helper()
	ahora := time.Now()
	_, claveCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA test"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true,
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	derCA, err := x509.CreateCertificate(rand.Reader, ca, ca, claveCA.Public(), claveCA)
	if err != nil {
		t.Fatal(err)
	}
	certCA, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	crear := func(serial int64, nombre string, uso x509.ExtKeyUsage) tls.Certificate {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre}, DNSNames: []string{nombre},
			NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso}}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, certCA, pub, claveCA)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: priv}
	}
	raices := x509.NewCertPool()
	raices.AddCert(certCA)
	ladoServidor, ladoCliente := net.Pipe()
	servidor := tls.Server(ladoServidor, &tls.Config{Certificates: []tls.Certificate{crear(2, "server.test", x509.ExtKeyUsageServerAuth)},
		ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: raices, MinVersion: tls.VersionTLS13})
	cliente := tls.Client(ladoCliente, &tls.Config{Certificates: []tls.Certificate{crear(3, "proxy.test", x509.ExtKeyUsageClientAuth)},
		RootCAs: raices, ServerName: "server.test", MinVersion: tls.VersionTLS13})
	defer servidor.Close()
	defer cliente.Close()
	errores := make(chan error, 2)
	go func() { errores <- servidor.Handshake() }()
	go func() { errores <- cliente.Handshake() }()
	for i := 0; i < 2; i++ {
		if err := <-errores; err != nil {
			t.Fatal(err)
		}
	}
	return servidor.ConnectionState()
}

type relojCertificadoFirmaVecPOSTPrueba struct{ ahora time.Time }

func (r *relojCertificadoFirmaVecPOSTPrueba) Ahora() time.Time { return r.ahora }

type acreditadorCertificadoFirmaVecPOSTPrueba struct {
	respuesta AcreditacionCertificadoFirmaVecV2
	llamadas  int
	err       error
}

func (a *acreditadorCertificadoFirmaVecPOSTPrueba) AcreditarCertificadoFirmaVecV2(
	_ context.Context, hoja, ca *x509.Certificate, _ time.Time,
) (AcreditacionCertificadoFirmaVecV2, error) {
	a.llamadas++
	if hoja == nil || ca == nil {
		return AcreditacionCertificadoFirmaVecV2{}, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return a.respuesta, a.err
}

type emisorCertificadoFirmaVecPOSTPrueba struct {
	identidad httpseguridad.AsercionProxyIdentidad
	llamadas  int
}

func (e *emisorCertificadoFirmaVecPOSTPrueba) EmitirRegistroFirmaVecPreparada(_ context.Context,
	a httpseguridad.AsercionProxyIdentidad,
) ([]byte, error) {
	e.llamadas++
	e.identidad = a
	return []byte("asercion-protegida-de-prueba"), nil
}

func TestExtractorCertificadoFirmaVecPOSTUsaCanalYCuerpoPreparado(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	reloj := &relojCertificadoFirmaVecPOSTPrueba{ahora: ahora}
	acreditador := &acreditadorCertificadoFirmaVecPOSTPrueba{respuesta: AcreditacionCertificadoFirmaVecV2{
		PersonaRef: "per_0123456789abcdefghijkl", CuentaRef: "cta_0123456789abcdefghijkl"}}
	emisor := &emisorCertificadoFirmaVecPOSTPrueba{}
	e := &extractorCertificadoFirmaVecV2{emisor: emisor, acreditador: acreditador,
		emisorID: "https://idp.example.invalid", audiencia: "vec-interna",
		retirada: ahora.Add(time.Hour), reloj: reloj}
	cuerpo := []byte(`{"firmado_base64":"JVBERi0xLjcKJSVFT0Y="}`)
	original := httptest.NewRequest(http.MethodPost, httpinterno.RutaRegistroFirmaVec, strings.NewReader(string(cuerpo)))
	estado := estadoTLSRealFirmanteV2Prueba(t)
	original.TLS = &estado
	r, err := httpseguridad.PrepararPeticionAsercionPasarela(original, httpseguridad.LimiteCuerpoRegistroFirmaVecPasarela)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	resultado, err := e.Extraer(r)
	if err != nil || string(resultado) != "asercion-protegida-de-prueba" ||
		acreditador.llamadas != 1 || emisor.llamadas != 1 {
		t.Fatalf("extracción POST: %v", err)
	}
	if emisor.identidad.SujetoID != acreditador.respuesta.PersonaRef ||
		emisor.identidad.Cuenta.ID != acreditador.respuesta.CuentaRef ||
		emisor.identidad.MetodoPrimario != httpseguridad.MetodoCertificado ||
		emisor.identidad.ACRVerificado != httpseguridad.ACRCertificadoPersonalDesarrolloProtegido ||
		len(emisor.identidad.Factores) != 1 || emisor.identidad.CanalVinculadoRef == "" {
		t.Fatal("identidad de certificado incompleta")
	}
	contenido, err := io.ReadAll(r.Body)
	if err != nil || string(contenido) != string(cuerpo) {
		t.Fatal("el extractor modificó el cuerpo firmado", err)
	}
	r.Header.Set("Authorization", "Bearer inyectado")
	if _, err := e.Extraer(r); !errors.Is(err, errIdentidadCertificadoFirmaVecNoDisponible) ||
		acreditador.llamadas != 1 || emisor.llamadas != 1 {
		t.Fatal("cabecera ambiental llegó a autoridades", err)
	}
	r.Header.Del("Authorization")
	r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: estado.PeerCertificates, VerifiedChains: estado.VerifiedChains}
	if _, err := e.Extraer(r); !errors.Is(err, errIdentidadCertificadoFirmaVecNoDisponible) ||
		acreditador.llamadas != 2 || emisor.llamadas != 1 {
		t.Fatal("certificado sin canal exportable produjo aserción", err)
	}
}

type auditoriaCertificadoFirmaVecPOSTPrueba struct{ llamadas int }

func (a *auditoriaCertificadoFirmaVecPOSTPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context,
	orden vp.OrdenAuditoriaFronteraRutaExacta,
) error {
	a.llamadas++
	if orden.Validar() != nil || orden.Ruta != httpinterno.RutaRegistroFirmaVec {
		return errors.New("orden inesperada")
	}
	return nil
}

func TestPuenteCertificadoFirmaVecPOSTCierraRutaMetodoYOrigen(t *testing.T) {
	auditoria := &auditoriaCertificadoFirmaVecPOSTPrueba{}
	llamadas := 0
	p := &puenteCertificadoFirmaVecV2{entorno: &EntornoIdentidadCertificadoFirmaVecV2{
		origen: "https://vec.example.invalid", host: "vec.example.invalid", auditoria: auditoria},
		siguiente: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadas++ })}
	casos := []struct {
		nombre, metodo, ruta, origen string
		estado                       int
	}{
		{"ruta_ajena", http.MethodPost, httpinterno.RutaRegistroFirmaExterna, "https://vec.example.invalid", 404},
		{"metodo_ajeno", http.MethodGet, httpinterno.RutaRegistroFirmaVec, "https://vec.example.invalid", 405},
		{"origen_ajeno", http.MethodPost, httpinterno.RutaRegistroFirmaVec, "https://otro.example.invalid", 403},
		{"sin_origen", http.MethodPost, httpinterno.RutaRegistroFirmaVec, "", 403},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			r := httptest.NewRequest(c.metodo, c.ruta, nil)
			r.Host = "vec.example.invalid"
			if c.origen != "" {
				r.Header.Set("Origin", c.origen)
			}
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)
			if w.Code != c.estado || llamadas != 0 || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("frontera %d, llamadas %d", w.Code, llamadas)
			}
		})
	}
	if auditoria.llamadas != 2 {
		t.Fatalf("denegaciones de origen sin auditoría: %d", auditoria.llamadas)
	}
}
