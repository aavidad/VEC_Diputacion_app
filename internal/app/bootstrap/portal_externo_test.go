package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
)

// materialPortalExternoPrueba prepara, con datos sintéticos, el material de un
// proceso externo: CA pública, TLS propio e identidad de la persona
// candidata. Las claves privadas de clientes quedan fuera del material, en
// clientes, solo para que la prueba pueda conectarse.
type materialPortalExternoPrueba struct {
	cfg      config.Config
	ca       string
	clientes string
}

func generarMaterialPortalExternoPrueba(t *testing.T) materialPortalExternoPrueba {
	t.Helper()
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl no disponible")
	}
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	raiz := cfg.DevelopmentMaterialDir
	clientes := t.TempDir()
	certificado := filepath.Join(raiz, "mtls", "candidato.crt")
	clave := filepath.Join(clientes, "candidato.key")
	csr := filepath.Join(clientes, "candidato.csr")
	extension := filepath.Join(clientes, "candidato.ext")
	if err := os.WriteFile(extension, []byte("basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=clientAuth\nsubjectAltName=URI:urn:vec:desarrollo:candidato-bolsa\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, argumentos := range [][]string{
		{"genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256", "-out", clave},
		{"req", "-new", "-sha256", "-key", clave, "-subj", "/CN=candidato-bolsa-desarrollo/O=VEC Desarrollo/OU=NO AUTORITATIVO", "-out", csr},
		{"x509", "-req", "-sha256", "-days", "30", "-in", csr, "-CA", rutas.CACertificate, "-CAkey", rutas.CAPrivateKey, "-CAserial", filepath.Join(raiz, "ca", "serie"), "-extfile", extension, "-out", certificado},
	} {
		if salida, err := exec.Command("openssl", argumentos...).CombinedOutput(); err != nil {
			t.Fatalf("certificado sintetico: %v: %s", err, salida)
		}
	}
	if err := os.Chmod(certificado, 0o600); err != nil {
		t.Fatal(err)
	}
	pemCandidato, err := os.ReadFile(certificado)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := decodificarCertificadoUnico(pemCandidato)
	if err != nil {
		t.Fatal(err)
	}
	huella := hexHuellaMiBolsaPrueba(sha256.Sum256(cert.Raw))
	sujeto := "per_candidato_sintetico_1234567890123456"
	escribirJSONExternoPrueba(t, filepath.Join(raiz, "identidad", "candidato.json"), map[string]any{
		"version": 1, "autoridad": AutoridadNoAutoritativa, "certificate_sha256": huella,
		"subject": sujeto, "display_name": "Lucia Moreno Castillo", "roles": []string{"candidato_bolsa"}})
	base := sujeto + "\x00" + huella
	escribirJSONExternoPrueba(t, filepath.Join(raiz, "identidad", "bolsa-candidato.json"), map[string]any{
		"version": 1, "autoridad": AutoridadNoAutoritativa,
		"certificado": "mtls/candidato.crt", "identidad": "identidad/candidato.json", "sujeto": sujeto,
		"cuenta_ref":    referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta"),
		"persona_ref":   referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona"),
		"perfil_ref":    "prf_candidato_sintetico_1234567890123456",
		"candidato_ref": "can_candidato_sintetico_1234567890123456"})
	// Cliente de RRHH: se conserva fuera del material para comprobar que el
	// proceso externo no lo reconoce como persona candidata.
	for _, nombre := range []string{"cliente.crt", "cliente.key"} {
		contenido, err := os.ReadFile(filepath.Join(raiz, "mtls", nombre))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(clientes, nombre), contenido, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	separarMaterial(t, raiz, separacionportales.PortalExterno)
	for _, relativa := range []string{"identidad/identidad.json", "identidad/intervencion.json", "mtls/cliente.crt", "mtls/intervencion.crt"} {
		if err := os.Remove(filepath.Join(raiz, filepath.FromSlash(relativa))); err != nil {
			t.Fatal(err)
		}
	}
	cfg.PortalProceso = "externo"
	return materialPortalExternoPrueba{cfg: cfg, ca: rutas.CACertificate, clientes: clientes}
}

func escribirJSONExternoPrueba(t *testing.T, ruta string, valor any) {
	t.Helper()
	contenido, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (m materialPortalExternoPrueba) cliente(t *testing.T, nombre string) *http.Client {
	t.Helper()
	caPEM, err := os.ReadFile(m.ca)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA no cargada")
	}
	configuracion := &tls.Config{RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13}
	if nombre != "" {
		certificado := filepath.Join(m.clientes, nombre+".crt")
		if nombre == "candidato" {
			certificado = filepath.Join(m.cfg.DevelopmentMaterialDir, "mtls", "candidato.crt")
		}
		par, err := tls.LoadX509KeyPair(certificado, filepath.Join(m.clientes, nombre+".key"))
		if err != nil {
			t.Fatal(err)
		}
		configuracion.Certificates = []tls.Certificate{par}
	}
	return &http.Client{Transport: &http.Transport{TLSClientConfig: configuracion}}
}

func TestProcesoExternoArrancaConSuPropioMaterial(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	m := generarMaterialPortalExternoPrueba(t)
	m.cfg.PersonalCatalogPath = "memory"
	var registro bytes.Buffer
	servidor, _, err := NewHTTPServerDesarrolloWithConfig(m.cfg, &registro)
	if err != nil {
		t.Fatalf("el proceso externo debe arrancar: %v", err)
	}
	if !strings.Contains(registro.String(), `"portal":"externo"`) {
		t.Fatalf("el arranque del externo no quedo registrado: %s", registro.String())
	}
	if servidor.TLSConfig == nil || servidor.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert ||
		servidor.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatalf("el externo no exige mTLS 1.3: %+v", servidor.TLSConfig)
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)

	if respuesta, err := m.cliente(t, "").Get(prueba.URL + "/api/publico/bolsa/convocatorias"); err == nil {
		respuesta.Body.Close()
		t.Fatal("el externo acepto un cliente sin certificado")
	}
	candidato := m.cliente(t, "candidato")
	codigo := func(ruta string) (int, []byte) {
		t.Helper()
		respuesta, err := candidato.Get(prueba.URL + ruta)
		if err != nil {
			t.Fatalf("%s: %v", ruta, err)
		}
		defer respuesta.Body.Close()
		cuerpo, _ := io.ReadAll(respuesta.Body)
		return respuesta.StatusCode, cuerpo
	}
	if estado, cuerpo := codigo("/api/publico/bolsa/convocatorias"); estado != http.StatusOK ||
		!bytes.Contains(cuerpo, []byte(`"esquema":"vec.bolsa.publico.convocatorias.v1"`)) {
		t.Fatalf("consulta publica en el externo = %d %s", estado, cuerpo)
	}
	if estado, cuerpo := codigo("/area-personal/"); estado != http.StatusOK || !bytes.Contains(cuerpo, []byte("<html")) {
		t.Fatalf("el Area personal debe servirse en el externo: %d", estado)
	}
	for _, ruta := range []string{
		"/api/vec/session", "/api/vec/contratacion-temporal/expedientes", "/api/vec/bolsa/bolsas",
		"/api/vec/usuarios/mis-preferencias", "/portal-empleado/", "/api/vec/personal/categories",
	} {
		if estado, _ := codigo(ruta); estado != http.StatusNotFound {
			t.Fatalf("el externo no debe servir %s: %d", ruta, estado)
		}
	}
}

func TestProcesoExternoNoArrancaSinPersonaCandidataNiConMaterialAjeno(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	m := generarMaterialPortalExternoPrueba(t)
	m.cfg.PersonalCatalogPath = "memory"
	if err := os.Remove(filepath.Join(m.cfg.DevelopmentMaterialDir, "identidad", "bolsa-candidato.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewHTTPServerWithConfig(m.cfg); !errors.Is(err, ErrMaterialPortalExternoInvalido) {
		t.Fatalf("sin persona candidata el externo no debe arrancar: %v", err)
	}
	m = generarMaterialPortalExternoPrueba(t)
	m.cfg.TLSKeyFile = filepath.Join(m.clientes, "cliente.key")
	if _, err := NewHTTPServerWithConfig(m.cfg); err == nil {
		t.Fatal("una clave TLS fuera del material del externo no debe aceptarse")
	}
}
