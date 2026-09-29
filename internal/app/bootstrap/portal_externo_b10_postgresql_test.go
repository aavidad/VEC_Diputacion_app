package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	pgpublico "vec-diputacion-granada/internal/modules/bolsa/adapters/postgrespublico"

	"vec-diputacion-granada/config"
)

// La fuente se publica una vez en una base dedicada de un clon sintético.
// Esta prueba sólo lee: no instala SQL ni altera la proyección conservada.
func TestProcesoExternoB10ActivadoDesdePostgreSQLPublico(t *testing.T) {
	if os.Getenv("VEC_PRUEBA_PORTAL_EXTERNO_B10") != "1" {
		t.Skip("recorrido PostgreSQL B10 no solicitado")
	}
	publica := config.Load()
	if _, err := publica.ExternoBolsaPublicaPostgreSQL.DSN(); err != nil {
		t.Fatal("falta conexion publica externa de prueba")
	}

	dsn, _ := publica.ExternoBolsaPublicaPostgreSQL.DSN()
	ctx, cancelar := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancelar()
	fuente, err := pgpublico.Abrir(ctx, dsn, publica.BolsaCategoriesCatalogID, publica.BolsaCategoriesVersion, publica.BolsaCategoriesSHA256, publica.BolsaCategoriesPublicProjectionSHA256, publica.BolsaPublicaManifiestoSHA256)
	if err != nil {
		t.Fatalf("conexion publica nominal de prueba: %v", err)
	}
	err = fuente.ValidarConfiguracionPublica(ctx, time.Now().UTC())
	fuente.Cerrar()
	if err != nil {
		t.Fatalf("manifiesto publico de prueba: %v", err)
	}
	vaciarConexionesDelEntorno(t)
	m := generarMaterialPortalExternoPrueba(t)
	m.cfg.PersonalCatalogPath = "memory"
	m.cfg.ExternoBolsaPublicaPostgreSQL = publica.ExternoBolsaPublicaPostgreSQL
	m.cfg.BolsaPublicaManifiestoSHA256 = publica.BolsaPublicaManifiestoSHA256
	m.cfg.BolsaCategoriesSourcePath = publica.BolsaCategoriesSourcePath
	m.cfg.BolsaCategoriesCatalogID = publica.BolsaCategoriesCatalogID
	m.cfg.BolsaCategoriesVersion = publica.BolsaCategoriesVersion
	m.cfg.BolsaCategoriesSHA256 = publica.BolsaCategoriesSHA256
	m.cfg.BolsaCategoriesPublicProjectionSHA256 = publica.BolsaCategoriesPublicProjectionSHA256
	var registro bytes.Buffer
	servidor, _, err := NewHTTPServerDesarrolloWithConfig(m.cfg, &registro)
	if err != nil {
		t.Fatalf("arranque B10: %v", err)
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(func() {
		prueba.Close()
		if err := servidor.Shutdown(context.Background()); err != nil {
			t.Errorf("cierre servidor B10: %v", err)
		}
	})
	candidato := m.cliente(t, "candidato")
	solicitar := func(ruta string, destino any) int {
		t.Helper()
		respuesta, err := candidato.Get(prueba.URL + ruta)
		if err != nil {
			t.Fatal(err)
		}
		defer respuesta.Body.Close()
		if destino != nil {
			if err := json.NewDecoder(respuesta.Body).Decode(destino); err != nil {
				t.Fatal(err)
			}
		}
		if respuesta.Header.Get("Set-Cookie") != "" {
			t.Fatal("cookie inesperada")
		}
		return respuesta.StatusCode
	}
	var lista struct {
		Data struct {
			Bolsas []struct {
				BolsaRef string `json:"bolsa_ref"`
				Total    int    `json:"total"`
			} `json:"bolsas"`
		} `json:"data"`
	}
	if estado := solicitar("/api/publico/bolsa/bolsas", &lista); estado != http.StatusOK || len(lista.Data.Bolsas) != 1 || lista.Data.Bolsas[0].Total != 1 {
		t.Fatalf("lista B10: estado=%d bolsas=%d", estado, len(lista.Data.Bolsas))
	}
	var detalle struct {
		Data struct {
			Posiciones []struct {
				Orden     int    `json:"orden"`
				Documento string `json:"documento_enmascarado"`
			} `json:"posiciones"`
		} `json:"data"`
	}
	if estado := solicitar("/api/publico/bolsa/bolsas/"+lista.Data.Bolsas[0].BolsaRef+"/lista", &detalle); estado != http.StatusOK || len(detalle.Data.Posiciones) != 1 || detalle.Data.Posiciones[0].Orden != 1 || !strings.HasPrefix(detalle.Data.Posiciones[0].Documento, "***") {
		t.Fatalf("detalle B10: estado=%d posiciones=%d", estado, len(detalle.Data.Posiciones))
	}
	for _, ruta := range []string{"/api/vec/contratacion-temporal/expedientes", "/api/vec/bolsa/bolsas"} {
		if estado := solicitar(ruta, nil); estado != http.StatusNotFound {
			t.Fatalf("ruta interna disponible: %s = %d", ruta, estado)
		}
	}
	for _, cliente := range []string{"", "cliente"} {
		respuesta, err := m.cliente(t, cliente).Get(prueba.URL + "/api/publico/bolsa/bolsas")
		if err == nil {
			respuesta.Body.Close()
			t.Fatalf("TLS acepto identidad ajena o anonima: %q", cliente)
		}
	}
}
