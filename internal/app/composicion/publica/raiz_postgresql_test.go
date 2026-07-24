package publica

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestIntegracionRaizPublicaSoloArrancaConPostgreSQLAutoritativo(t *testing.T) {
	dsn := os.Getenv("VEC_PRUEBA_BOLSA_PUBLICA_DSN")
	ancla := os.Getenv("VEC_PRUEBA_BOLSA_PUBLICA_MANIFIESTO_SHA256")
	if dsn == "" || ancla == "" {
		t.Skip("integracion PostgreSQL no solicitada")
	}
	postgresql, err := config.NuevaConfiguracionPostgreSQLPublica(dsn)
	if err != nil {
		t.Fatal(err)
	}
	servidor, err := NuevoServidor(Configuracion{
		Direccion:                  "127.0.0.1:0",
		RedesPermitidas:            []string{"0.0.0.0/0", "::/0"},
		PerfilEjecucion:            config.ExecutionProfileProduction,
		AutenticacionSolicitada:    config.AuthModeDisabled,
		CatalogoCategorias:         "categorias-profesionales",
		VersionCategorias:          2,
		HuellaCategorias:           strings.Repeat("b", 64),
		HuellaProyeccionCategorias: "b661b37ca7323fa168734899038f8fa99cb77ff07d114e4a7d787d62b5d36593",
		HuellaManifiesto:           ancla,
		PostgreSQL:                 postgresql,
	})
	if err != nil {
		t.Fatalf("componer raiz publica: %v", err)
	}
	defer func() { _ = servidor.Shutdown(context.Background()) }()

	for _, ruta := range []string{
		"/api/publico/bolsa/convocatorias",
		"/api/publico/bolsa/convocatorias/auxiliares-2026",
		"/api/publico/bolsa/categorias",
	} {
		peticion := httptest.NewRequest(http.MethodGet, ruta, nil)
		peticion.RemoteAddr = "192.0.2.10:43210"
		respuesta := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(respuesta, peticion)
		cuerpo := respuesta.Body.String()
		if respuesta.Code != http.StatusOK ||
			!strings.Contains(cuerpo, `"demostracion":false`) {
			t.Fatalf("raiz productiva %s = %d %s", ruta, respuesta.Code, cuerpo)
		}
		for _, prohibido := range []string{
			`"referencia_agregado"`, `"id":`, "proceso:bolsa:auxiliares:2026",
		} {
			if strings.Contains(cuerpo, prohibido) {
				t.Fatalf("la respuesta %s expuso identidad interna %q: %s", ruta, prohibido, cuerpo)
			}
		}
	}
}
