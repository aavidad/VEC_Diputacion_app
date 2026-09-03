package bootstrap

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestAltaContratacionTemporalDesarrolloApareceEnCuadroYDetalleConMTLSReal(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
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
	cuerpoAlta := cuerpoAltaContratacionTemporalDesarrolloPrueba()

	t.Run("sin certificado no alcanza el manejador", func(t *testing.T) {
		cliente := nuevoClienteSinCertificadoContratacionTemporalDesarrollo(t, rutas)
		peticion := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
			t, baseURL+httpinterno.RutaAltaSolicitudes, cuerpoAlta,
		)
		respuesta, err := cliente.Do(peticion)
		if err == nil {
			contenido, _ := io.ReadAll(respuesta.Body)
			respuesta.Body.Close()
			t.Fatalf("el listener acepto alta sin certificado: %d %s", respuesta.StatusCode, contenido)
		}
	})

	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	t.Cleanup(cliente.CloseIdleConnections)

	for _, caso := range []struct{ nombre, cabecera, valor string }{
		{"bearer", "Authorization", "Bearer autoridad-forjada"},
		{"principal cliente", "X-Vec-Principal", "administrador"},
	} {
		t.Run(caso.nombre+" no aporta autoridad ni crea expediente", func(t *testing.T) {
			peticion := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
				t, baseURL+httpinterno.RutaAltaSolicitudes, cuerpoAlta,
			)
			peticion.Header.Set(caso.cabecera, caso.valor)
			respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
				t, cliente, peticion,
			)
			if respuesta.StatusCode != http.StatusBadRequest ||
				bytes.Contains(contenido, []byte(`"expediente_ref"`)) {
				t.Fatalf("cabecera cliente alcanzo el alta: %d %s", respuesta.StatusCode, contenido)
			}
		})
	}

	peticionAlta := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
		t, baseURL+httpinterno.RutaAltaSolicitudes, cuerpoAlta,
	)
	respuestaAlta, contenidoAlta := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticionAlta,
	)
	if respuestaAlta.StatusCode != http.StatusCreated {
		t.Fatalf("alta=%d %s", respuestaAlta.StatusCode, contenidoAlta)
	}
	var alta struct {
		Data struct {
			ExpedienteRef string `json:"expediente_ref"`
			NumeroVisible string `json:"numero_visible"`
			Version       uint64 `json:"version"`
			ReciboRef     string `json:"recibo_ref"`
		} `json:"data"`
	}
	if err := json.Unmarshal(contenidoAlta, &alta); err != nil {
		t.Fatalf("decodificar recibo: %v: %s", err, contenidoAlta)
	}
	if alta.Data.ExpedienteRef == "" || alta.Data.NumeroVisible == "" ||
		alta.Data.Version != 1 || alta.Data.ReciboRef == "" {
		t.Fatalf("recibo incompleto: %+v", alta.Data)
	}

	cuerpoCuadro := fmt.Sprintf(
		`{"filtros":{"texto":%q,"estado_clave":"en_curso","fase_clave":"solicitud"},`+
			`"paginacion":{"limite":50,"cursor":""}}`,
		alta.Data.NumeroVisible,
	)
	peticionCuadro := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
		t, baseURL+httpinterno.RutaConsultaCuadroRRHH, cuerpoCuadro,
	)
	respuestaCuadro, contenidoCuadro := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticionCuadro,
	)
	if respuestaCuadro.StatusCode != http.StatusOK ||
		!bytes.Contains(contenidoCuadro, []byte(alta.Data.ExpedienteRef)) ||
		!bytes.Contains(contenidoCuadro, []byte(alta.Data.NumeroVisible)) {
		t.Fatalf("cuadro no contiene el alta: %d %s", respuestaCuadro.StatusCode, contenidoCuadro)
	}

	cuerpoDetalle := fmt.Sprintf(
		`{"expediente_ref":%q,"version_observada":%d}`,
		alta.Data.ExpedienteRef,
		alta.Data.Version,
	)
	peticionDetalle := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
		t, baseURL+httpinterno.RutaConsultaDetalleRRHH, cuerpoDetalle,
	)
	respuestaDetalle, contenidoDetalle := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticionDetalle,
	)
	if respuestaDetalle.StatusCode != http.StatusOK ||
		!bytes.Contains(contenidoDetalle, []byte(alta.Data.ExpedienteRef)) ||
		!bytes.Contains(contenidoDetalle, []byte(`"motivo_clave":"sustitucion"`)) {
		t.Fatalf("detalle no contiene el alta: %d %s", respuestaDetalle.StatusCode, contenidoDetalle)
	}
}

func cuerpoAltaContratacionTemporalDesarrolloPrueba() string {
	ahora := time.Now().UTC()
	inicio := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, 1)
	fin := inicio.AddDate(0, 3, 0)
	return fmt.Sprintf(`{
		"clave_idempotencia":"4d36e96e-e325-4f9b-bebc-291d91d6f732",
		"solicitud":{
			"centro_ref":"centro:desarrollo:001",
			"contacto_ref":"contacto:desarrollo:001",
			"categoria_ref":"categoria:desarrollo:c2",
			"grupo_subgrupo":"C2",
			"motivo_clave":"sustitucion",
			"detalle":"Cobertura temporal para demostracion local.",
			"periodo":{"inicio":%q,"fin":%q},
			"rc":{"existe":false},
			"documentos_adjuntos":[],
			"observaciones":""
		}
	}`,
		inicio.Format("2006-01-02T15:04:05Z"),
		fin.Format("2006-01-02T15:04:05Z"),
	)
}

func nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
	t *testing.T,
	ruta string,
	cuerpo string,
) *http.Request {
	t.Helper()
	peticion, err := http.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	if err != nil {
		t.Fatal(err)
	}
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func ejecutarPeticionContratacionTemporalDesarrolloPrueba(
	t *testing.T,
	cliente *http.Client,
	peticion *http.Request,
) (*http.Response, []byte) {
	t.Helper()
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		t.Fatalf("peticion mTLS: %v", err)
	}
	contenido, err := io.ReadAll(respuesta.Body)
	respuesta.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return respuesta, contenido
}

func nuevoClienteSinCertificadoContratacionTemporalDesarrollo(
	t *testing.T,
	rutas config.DevelopmentMaterialPaths,
) *http.Client {
	t.Helper()
	caPEM, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA local no cargada")
	}
	cliente := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13,
	}}}
	t.Cleanup(cliente.CloseIdleConnections)
	return cliente
}
