package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/publica"
)

// El entorno de prueba aporta una proyección sintética conservada y su login
// lector público. No instala SQL ni utiliza la composición del Área personal.
func TestBolsasPublicasTLSAnonimoDesdePostgreSQL(t *testing.T) {
	if os.Getenv("VEC_PRUEBA_BOLSA_PUBLICA_ANONIMA") != "1" {
		t.Skip("recorrido publico PostgreSQL no solicitado")
	}
	// El manifiesto de producción se resuelve desde la raíz del repositorio.
	t.Chdir("../..")
	cfg := publica.CargarConfiguracion()
	cfg.Direccion = "127.0.0.1:0"
	servidor, err := publica.NuevoServidor(cfg)
	if err != nil {
		t.Fatal("no se pudo componer la entrada publica de prueba")
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	if servidor.TLSConfig != nil {
		prueba.TLS = servidor.TLSConfig.Clone()
	}
	prueba.StartTLS()
	t.Cleanup(func() {
		prueba.Close()
		ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		if err := servidor.Shutdown(ctx); err != nil {
			t.Errorf("cierre de la entrada publica: %v", err)
		}
	})
	cliente := prueba.Client()
	cliente.Timeout = 15 * time.Second
	cliente.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	transporte, ok := cliente.Transport.(*http.Transport)
	if !ok || transporte.TLSClientConfig == nil || len(transporte.TLSClientConfig.Certificates) != 0 || transporte.TLSClientConfig.GetClientCertificate != nil {
		t.Fatal("el cliente de prueba debe visitar sin certificado")
	}
	if prueba.TLS.ClientAuth != tls.NoClientCert {
		t.Fatal("la entrada publica requiere certificado de cliente")
	}
	solicitar := func(ruta string, esperado int, destino any) {
		t.Helper()
		respuesta, err := cliente.Get(prueba.URL + ruta)
		if err != nil {
			t.Fatal("fallo la consulta publica sin certificado")
		}
		defer respuesta.Body.Close()
		if respuesta.StatusCode != esperado || respuesta.Header.Get("Location") != "" || respuesta.Header.Get("Set-Cookie") != "" {
			t.Fatalf("GET %s = %d; esperado %d sin redireccion ni cookie", ruta, respuesta.StatusCode, esperado)
		}
		if respuesta.TLS == nil || len(respuesta.TLS.VerifiedChains) == 0 {
			t.Fatal("la consulta no verifico TLS del servidor")
		}
		if destino != nil {
			if err := json.NewDecoder(io.LimitReader(respuesta.Body, 1<<20)).Decode(destino); err != nil {
				t.Fatal("respuesta publica invalida")
			}
		}
	}
	var lista struct {
		Data struct {
			Bolsas []struct {
				BolsaRef string `json:"bolsa_ref"`
			} `json:"bolsas"`
		} `json:"data"`
	}
	solicitar("/api/publico/bolsa/bolsas", http.StatusOK, &lista)
	if len(lista.Data.Bolsas) == 0 || lista.Data.Bolsas[0].BolsaRef == "" {
		t.Fatal("el escenario sintetico debe conservar al menos una bolsa publicada")
	}
	var detalle struct {
		Data struct {
			Posiciones []struct {
				Orden int `json:"orden"`
			} `json:"posiciones"`
		} `json:"data"`
	}
	solicitar("/api/publico/bolsa/bolsas/"+url.PathEscape(lista.Data.Bolsas[0].BolsaRef)+"/lista", http.StatusOK, &detalle)
	if len(detalle.Data.Posiciones) == 0 || detalle.Data.Posiciones[0].Orden < 1 {
		t.Fatal("el escenario sintetico debe conservar una posicion publicada")
	}
	solicitar("/bolsa/", http.StatusOK, nil)
	for _, ruta := range []string{
		"/portal-empleado", "/portal-empleado/portal.js", "/area-personal", "/area-personal/",
		"/api/vec", "/api/vec/contratacion-temporal/expedientes", "/api/vec/bolsa/mi-bolsa",
		"/api/vec/personas/mis-preferencias/", "/api/vec/usuarios/contacto-propio", "/api/demo/bolsa",
	} {
		solicitar(ruta, http.StatusNotFound, nil)
	}
}
