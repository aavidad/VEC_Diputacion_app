package bootstrap

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestConsultasContratacionTemporalDesarrolloRespondenConMTLS(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)

	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	casos := []struct {
		nombre    string
		ruta      string
		cuerpo    string
		esquema   string
		contenido string
	}{
		{
			nombre: "cuadro",
			ruta:   httpinterno.RutaConsultaCuadroRRHH,
			cuerpo: "{\"filtros\":{\"texto\":\"2026/CT\",\"estado_clave\":\"en_curso\"," +
				"\"fase_clave\":\"analisis\"},\"paginacion\":{\"limite\":50,\"cursor\":\"\"}}",
			esquema:   "vec.contratacion-temporal.cuadro-rrhh.v1",
			contenido: expedienteContratacionTemporalDesarrolloRef,
		},
		{
			nombre: "detalle",
			ruta:   httpinterno.RutaConsultaDetalleRRHH,
			cuerpo: "{\"expediente_ref\":\"" + expedienteContratacionTemporalDesarrolloRef +
				"\",\"version_observada\":3}",
			esquema:   "vec.contratacion-temporal.detalle-rrhh.v1",
			contenido: "\"grupo_subgrupo\":\"C2\"",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion, err := http.NewRequest(
				http.MethodPost,
				prueba.URL+caso.ruta,
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
			esquema := []byte("\"esquema\":\"" + caso.esquema + "\"")
			if respuesta.StatusCode != http.StatusOK ||
				!bytes.Contains(contenido, esquema) ||
				!bytes.Contains(contenido, []byte(caso.contenido)) {
				t.Fatalf("respuesta=%d %s", respuesta.StatusCode, contenido)
			}
		})
	}
}

func TestConsultasContratacionTemporalDesarrolloNoAceptanAutoridadCliente(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
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
	if respuesta.StatusCode != http.StatusBadRequest &&
		respuesta.StatusCode != http.StatusUnauthorized ||
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
