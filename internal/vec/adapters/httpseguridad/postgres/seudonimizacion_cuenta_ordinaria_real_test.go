package postgres_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/app/bootstrap"
	postgres "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

// Usa el proveedor DEV publicado y cuatro claves sintéticas aisladas. No
// implementa HMAC en la prueba ni lee el material privado de un entorno.
func TestAliasOrdinarioCoincideConProveedorDEVYFinalidadCuenta(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "idempotencia")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	type generacion struct {
		Generacion                int    `json:"generacion"`
		ReferenciaLocalizador     string `json:"referencia_localizador"`
		ReferenciaHuellaSolicitud string `json:"referencia_huella_solicitud"`
	}
	config := struct {
		Version            int          `json:"version"`
		Esquema            string       `json:"esquema"`
		Autoridad          string       `json:"autoridad"`
		VersionEsquemaHMAC int          `json:"version_esquema_hmac"`
		Generaciones       []generacion `json:"generaciones"`
	}{Version: 1, Esquema: "vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1",
		Autoridad: "no_autoritativo", VersionEsquemaHMAC: 2}
	for _, numero := range []int{2, 1} {
		config.Generaciones = append(config.Generaciones, generacion{
			Generacion:                numero,
			ReferenciaLocalizador:     fmt.Sprintf("clave:hmac:convocatorias:localizador:desarrollo:v%d", numero),
			ReferenciaHuellaSolicitud: fmt.Sprintf("clave:hmac:convocatorias:huella:desarrollo:v%d", numero),
		})
		for indice, dominio := range []string{"localizador", "huella-solicitud"} {
			clave := bytes.Repeat([]byte{byte(numero*2 + indice)}, 32)
			ruta := filepath.Join(dir, fmt.Sprintf("g%d-%s.bin", numero, dominio))
			if err := os.WriteFile(ruta, clave, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	b, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	rutaConfig := filepath.Join(dir, "configuracion.json")
	if err := os.WriteFile(rutaConfig, b, 0600); err != nil {
		t.Fatal(err)
	}
	configProveedor := bootstrap.ConfiguracionSeudonimosSesionPrivada{
		DirectorioMaterial: raiz, RutaConfiguracionHMAC: rutaConfig,
		EspacioIdentidad:       "https://admin.example.invalid/identidad",
		DominioRef:             "idh_0123456789abcdefghijklmn",
		EspacioClave:           "vec.identidad.admin.prueba",
		DominioHMAC:            "vec.identidad.admin.hmac.v1",
		IncluirCuentaOrdinaria: true,
	}
	proveedor, cerrar, err := bootstrap.NuevoSeudonimizadorSesionDesdeArchivo(configProveedor)
	if err != nil {
		t.Fatal(err)
	}
	defer cerrar()
	ids := postgres.IdentificadoresAlta{
		EspacioIdentidad: configProveedor.EspacioIdentidad,
		AsercionID:       "asercion-sintetica", SesionID: "sesion-sintetica",
		SujetoID: "persona-sintetica", CuentaID: "cuenta-admin-sintetica",
		CuentaOrdinariaID: "cuenta-ordinaria-sintetica",
	}
	original, alias, err := postgres.SeudonimizarAltaConAliasCuentaOrdinaria(
		context.Background(), proveedor, ids, configProveedor.EspacioIdentidad, configProveedor.DominioRef,
	)
	if err != nil || len(alias) != 32 {
		t.Fatal("el proveedor DEV no entregó un alias ordinario válido")
	}
	consultaOrdinaria := ids
	consultaOrdinaria.CuentaID = ids.CuentaOrdinariaID
	ordinaria, err := proveedor.SeudonimizarAlta(context.Background(), consultaOrdinaria)
	if err != nil || !bytes.Equal(alias, ordinaria.CuentaIDHMAC[:]) ||
		bytes.Equal(alias, original.CuentaOrdinariaIDHMAC[:]) ||
		bytes.Equal(alias, original.CuentaIDHMAC[:]) {
		t.Fatal("el alias no coincide con la finalidad cuenta del proveedor DEV")
	}
}
