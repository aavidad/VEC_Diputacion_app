package bootstrap

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestConsultasContratacionTemporalDesarrolloDenieganComoNoCompuestasEnListenerTLSReal(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	escucha, err := net.Listen("tcp", servidor.Addr)
	if err != nil {
		t.Fatalf("abrir listener: %v", err)
	}
	t.Cleanup(func() {
		_ = servidor.Close()
		_ = escucha.Close()
	})
	go func() {
		_ = servidor.ServeTLS(escucha, cfg.TLSCertFile, cfg.TLSKeyFile)
	}()
	baseURL := fmt.Sprintf("https://localhost:%d", escucha.Addr().(*net.TCPAddr).Port)

	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	casos := []struct {
		nombre string
		ruta   string
		cuerpo string
	}{
		{
			nombre: "cuadro",
			ruta:   httpinterno.RutaConsultaCuadroRRHH,
			cuerpo: "{\"filtros\":{\"texto\":\"2026/CT\",\"estado_clave\":\"en_curso\"," +
				"\"fase_clave\":\"analisis\"},\"paginacion\":{\"limite\":50,\"cursor\":\"\"}}",
		},
		{
			nombre: "detalle",
			ruta:   httpinterno.RutaConsultaDetalleRRHH,
			cuerpo: "{\"expediente_ref\":\"" + expedienteContratacionTemporalDesarrolloRef +
				"\",\"version_observada\":3}",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion, err := http.NewRequest(
				http.MethodPost,
				baseURL+caso.ruta,
				strings.NewReader(caso.cuerpo),
			)
			if err != nil {
				t.Fatal(err)
			}
			peticion.Header.Set("Content-Type", "application/json")
			peticion.Header.Set("Accept", "application/json")
			respuesta, err := cliente.Do(peticion)
			if err != nil {
				t.Fatalf("consulta mTLS: %v", err)
			}
			contenido, err := io.ReadAll(respuesta.Body)
			respuesta.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if respuesta.StatusCode != http.StatusServiceUnavailable ||
				!bytes.Contains(contenido, []byte(`"codigo":"servicio_no_disponible"`)) ||
				bytes.Contains(contenido, []byte(expedienteContratacionTemporalDesarrolloRef)) {
				t.Fatalf("respuesta=%d %s", respuesta.StatusCode, contenido)
			}
		})
	}
}

func anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(
	t *testing.T,
	cliente *http.Client,
	rutas config.DevelopmentMaterialPaths,
) {
	t.Helper()
	caPEM, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	ca, resto := pem.Decode(caPEM)
	if ca == nil || ca.Type != "CERTIFICATE" || len(bytes.TrimSpace(resto)) != 0 {
		t.Fatal("CA de prueba invalida")
	}
	transporte, ok := cliente.Transport.(*http.Transport)
	if !ok || transporte.TLSClientConfig == nil ||
		len(transporte.TLSClientConfig.Certificates) != 1 {
		t.Fatal("cliente mTLS de prueba invalido")
	}
	transporte.TLSClientConfig.Certificates[0].Certificate = append(
		transporte.TLSClientConfig.Certificates[0].Certificate,
		ca.Bytes,
	)
}

func TestPeticionIdentidadConsultasContratacionTemporalDesarrolloSoloNormalizaCadenaVerificada(
	t *testing.T,
) {
	hoja := &x509.Certificate{Raw: []byte("hoja")}
	raiz := &x509.Certificate{Raw: []byte("raiz")}
	peticion := &http.Request{TLS: &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{hoja, raiz},
		VerifiedChains:   [][]*x509.Certificate{{hoja, raiz}},
	}}
	normalizada := peticionIdentidadConsultasContratacionTemporalDesarrollo(peticion)
	if normalizada == peticion || normalizada.TLS == peticion.TLS ||
		len(normalizada.TLS.PeerCertificates) != 1 ||
		normalizada.TLS.PeerCertificates[0] != hoja ||
		len(peticion.TLS.PeerCertificates) != 2 {
		t.Fatal("la cadena verificada no se normalizo mediante una copia aislada")
	}

	adulterada := &http.Request{TLS: &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{hoja, {Raw: []byte("otra")}},
		VerifiedChains:   [][]*x509.Certificate{{hoja, raiz}},
	}}
	if obtenida := peticionIdentidadConsultasContratacionTemporalDesarrollo(adulterada); obtenida != adulterada {
		t.Fatal("una cadena presentada ajena a la verificada fue normalizada")
	}
}

func TestConsultasContratacionTemporalDesarrolloNoAceptanAutoridadCliente(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)
	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	peticion, err := http.NewRequest(
		http.MethodPost,
		prueba.URL+httpinterno.RutaConsultaCuadroRRHH,
		strings.NewReader(
			"{\"filtros\":{\"texto\":\"\",\"estado_clave\":\"\",\"fase_clave\":\"\"},"+
				"\"paginacion\":{\"limite\":50,\"cursor\":\"\"}}",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("X-Vec-Principal", "administrador")
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		t.Fatal(err)
	}
	contenido, _ := io.ReadAll(respuesta.Body)
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusServiceUnavailable ||
		!bytes.Contains(contenido, []byte(`"codigo":"servicio_no_disponible"`)) ||
		bytes.Contains(contenido, []byte(expedienteContratacionTemporalDesarrolloRef)) {
		t.Fatalf("cabecera cliente autorizo la consulta: %d %s", respuesta.StatusCode, contenido)
	}
}

func nuevoClienteMTLSContratacionTemporalDesarrollo(
	t *testing.T,
	rutas config.DevelopmentMaterialPaths,
) *http.Client {
	t.Helper()
	certificado, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
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
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		Certificates: []tls.Certificate{certificado},
		RootCAs:      raices,
		ServerName:   "localhost",
		MinVersion:   tls.VersionTLS13,
	}}}
}
