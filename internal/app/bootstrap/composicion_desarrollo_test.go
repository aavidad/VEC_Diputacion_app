package bootstrap

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
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
	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func generarMaterialDesarrolloPrueba(t *testing.T) (config.Config, config.DevelopmentMaterialPaths) {
	t.Helper()
	raizRepositorio, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	generador := filepath.Join(raizRepositorio, "scripts", "generar_credenciales_desarrollo.sh")
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl no disponible")
	}
	destino := filepath.Join(t.TempDir(), "credenciales")
	orden := exec.Command(generador, destino)
	if salida, err := orden.CombinedOutput(); err != nil {
		t.Fatalf("generar material: %v\n%s", err, salida)
	}
	cfg := config.Config{
		Address:                   "127.0.0.1:0",
		ExecutionProfile:          config.ExecutionProfileDevelopment,
		AuthMode:                  config.AuthModeDevelopment,
		DevelopmentGuard:          config.DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir:    destino,
		PersonalCatalogPath:       "memory",
		BolsaPublicSourcePath:     filepath.Join(raizRepositorio, config.DefaultBolsaPublicSourcePath),
		BolsaCategoriesSourcePath: filepath.Join(raizRepositorio, config.DefaultBolsaCategoriesSourcePath),
	}.Normalize()
	return cfg, cfg.DevelopmentPaths()
}

func TestProduccionRechazaTLSRealGeneradoPorT21(t *testing.T) {
	_, rutas := generarMaterialDesarrolloPrueba(t)
	servidor, err := NewHTTPServerWithConfig(config.Config{
		Address:             "127.0.0.1:0",
		PersonalCatalogPath: "memory",
		TLSCertFile:         rutas.ServerCertificate,
		TLSKeyFile:          rutas.ServerPrivateKey,
	})
	if servidor != nil || !errors.Is(err, ErrProveedorDesarrolloEnProduccion) {
		t.Fatalf("produccion acepto el TLS concreto de T21: servidor=%v error=%v", servidor, err)
	}
}

func TestServidorPublicoRechazaTLSYSelectoresRealesDeT21(t *testing.T) {
	cfgDesarrollo, rutas := generarMaterialDesarrolloPrueba(t)
	servidor, err := NewHTTPServerPublicoWithConfig(config.Config{
		Address:             "127.0.0.1:0",
		PersonalCatalogPath: "memory",
		TLSCertFile:         rutas.ServerCertificate,
		TLSKeyFile:          rutas.ServerPrivateKey,
	})
	if servidor != nil || !errors.Is(err, ErrProveedorDesarrolloEnProduccion) {
		t.Fatalf("servidor publico acepto TLS T21: servidor=%v error=%v", servidor, err)
	}

	cfgDesarrollo.TLSCertFile = ""
	cfgDesarrollo.TLSKeyFile = ""
	servidor, err = NewHTTPServerPublicoWithConfig(cfgDesarrollo)
	if servidor != nil || !errors.Is(err, ErrActivacionDesarrolloInvalida) {
		t.Fatalf("servidor publico degrado selectores T21: servidor=%v error=%v", servidor, err)
	}
}

func TestMaterialDesarrolloDetectaRepositorioDesdeLaPropiaRuta(t *testing.T) {
	repositorio := t.TempDir()
	if err := os.Mkdir(filepath.Join(repositorio, ".git"), 0o700); err != nil {
		t.Fatalf("crear marca Git: %v", err)
	}
	ruta := filepath.Join(repositorio, "estado", "credenciales")
	if !dentroDeRepositorioGit(ruta) {
		t.Fatal("la deteccion dependio del directorio de trabajo y acepto una ruta dentro de Git")
	}
	if dentroDeRepositorioGit(t.TempDir()) {
		t.Fatal("una ruta temporal ajena fue clasificada como repositorio")
	}
}

func TestComposicionDesarrolloOperaConTLSMutuoEIdentidadAlta(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	var registro bytes.Buffer
	servidor, composicion, err := NewHTTPServerDesarrolloWithConfig(cfg, &registro)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	if servidor.TLSConfig == nil || servidor.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert ||
		servidor.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatalf("TLS de desarrollo no exige mTLS 1.3: %+v", servidor.TLSConfig)
	}
	metadatos, err := composicion.MetadatosComposicion()
	if err != nil || metadatos.Datos().Autoridad != AutoridadNoAutoritativa {
		t.Fatalf("marca del acto: %+v, %v", metadatos.Datos(), err)
	}
	procedencia, err := composicion.ProcedenciaActosBorrador()
	if err != nil || procedencia.Esquema == "" ||
		procedencia.Autoridad != gobiernoconvocatorias.AutoridadActoNoAutoritativa || procedencia.MigrableProduccion {
		t.Fatalf("procedencia durable: %+v, %v", procedencia, err)
	}
	if !strings.Contains(registro.String(), "credenciales_no_autoritativas") {
		t.Fatalf("arranque no fue ruidoso: %s", registro.String())
	}
	if _, err := composicion.CifradorBorradores(); err != nil {
		t.Fatalf("emisor KMS no compuesto: %v", err)
	}
	if _, err := composicion.RevalidadorKMSBorradores(); err != nil {
		t.Fatalf("revalidador KMS no compuesto: %v", err)
	}
	if _, err := composicion.VerificadorFirmasKMSBorradores(); err != nil {
		t.Fatalf("verificador publico KMS no compuesto: %v", err)
	}
	if _, err := composicion.DerivadorIdentidadesBorrador(); err != nil {
		t.Fatalf("derivador HMAC de idempotencia no compuesto: %v", err)
	}

	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)

	certificadoCliente, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA local no cargada")
	}
	clienteSinCertificado := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13,
	}}}
	if respuestaSinCertificado, err := clienteSinCertificado.Get(prueba.URL + "/api/vec/session"); err == nil {
		respuestaSinCertificado.Body.Close()
		t.Fatal("el listener mTLS acepto un cliente sin certificado")
	}
	cliente := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		Certificates: []tls.Certificate{certificadoCliente}, RootCAs: raices,
		ServerName: "localhost", MinVersion: tls.VersionTLS13,
	}}}
	respuesta, err := cliente.Get(prueba.URL + "/api/vec/session")
	if err != nil {
		t.Fatalf("peticion mTLS: %v", err)
	}
	defer respuesta.Body.Close()
	contenido, _ := io.ReadAll(respuesta.Body)
	if respuesta.StatusCode != http.StatusOK || !bytes.Contains(contenido, []byte(`"autoridad":"no_autoritativo"`)) ||
		!bytes.Contains(contenido, []byte(`"auth_assurance":"alto"`)) {
		t.Fatalf("sesion mTLS = %d %s", respuesta.StatusCode, contenido)
	}
	respuestaPublica, err := cliente.Get(prueba.URL + "/api/publico/bolsa/convocatorias")
	if err != nil {
		t.Fatalf("consulta publica en servidor de desarrollo: %v", err)
	}
	contenidoPublico, err := io.ReadAll(respuestaPublica.Body)
	respuestaPublica.Body.Close()
	if err != nil || respuestaPublica.StatusCode != http.StatusOK ||
		!bytes.Contains(contenidoPublico, []byte(`"esquema":"vec.bolsa.publico.convocatorias.v1"`)) {
		t.Fatalf("consulta publica en desarrollo = %d %s, %v", respuestaPublica.StatusCode, contenidoPublico, err)
	}
	peticionSuplantada, err := http.NewRequest(http.MethodGet, prueba.URL+"/api/vec/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	peticionSuplantada.Header.Set("X-Vec-Principal", "administrador")
	respuestaSuplantada, err := cliente.Do(peticionSuplantada)
	if err != nil {
		t.Fatal(err)
	}
	defer respuestaSuplantada.Body.Close()
	if respuestaSuplantada.StatusCode == http.StatusOK {
		t.Fatal("una cabecera de identidad suplanto al certificado mTLS")
	}
}

func TestRaizHTTPComponePerfilDesarrolloSoloConDobleLlaveCompleta(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("raiz HTTP desarrollo: %v", err)
	}
	if servidor == nil || servidor.TLSConfig == nil ||
		servidor.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert ||
		servidor.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatalf("raiz HTTP no selecciono mTLS de desarrollo: %+v", servidor)
	}
}

func TestComposicionDesarrolloRechazaPermisosAmplios(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	if err := os.Chmod(rutas.KMSSecret, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard); err == nil {
		t.Fatal("se acepto un secreto legible por el grupo")
	}
}

func TestComposicionDesarrolloRechazaReutilizarClaveDeAtestacion(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	for origen, destino := range map[string]string{
		rutas.KMSAttestationKey:    rutas.KMSRevalidationKey,
		rutas.KMSAttestationPublic: rutas.KMSRevalidationPublic,
	} {
		contenido, err := os.ReadFile(origen)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destino, contenido, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard); err == nil {
		t.Fatal("se acepto la misma pareja para atestacion y revalidacion")
	}
}

func TestTSADesarrolloEsDeterministaYQuedaMarcada(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	tsa, err := composicion.SelladorTiempo()
	if err != nil {
		t.Fatal(err)
	}
	solicitud := vecports.InteropRequest{
		Operation: "sellar-huella", Subject: "documento:123",
		Payload: map[string]string{"huella_sha256": strings.Repeat("a", 64), "perfil": "PAdES-B-T"},
	}
	primero, err := tsa.Timestamp(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := tsa.Timestamp(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if primero.Reference != segundo.Reference || primero.Status != AutoridadNoAutoritativa ||
		primero.Payload["migrable_a_produccion"] != "false" {
		t.Fatalf("sello local no determinista o sin marca: %+v / %+v", primero, segundo)
	}
	solicitud.Payload["perfil"] = "PAdES-B-LTA"
	tercero, err := tsa.Timestamp(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if tercero.Reference == primero.Reference {
		t.Fatal("una carga distinta produjo el mismo sello")
	}
}
