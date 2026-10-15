package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/app/administracion"
)

// El arranque dice en qué etapa falló sin dejar de ser ErrConfiguracion y
// sin copiar rutas ni valores de la configuración privada.
func TestArranqueIndicaEtapaSinDatosPrivados(t *testing.T) {
	ruta := "/ruta/privada/no-debe-salir.json"
	base := configuracionPerfilesPrivada{ActivosDirectorio: ruta}
	for nombre, caso := range map[string]struct {
		componer func() error
		etapa    string
	}{
		"perfiles": {func() error { _, _, err := componerProcesoADMIN(administracion.Configuracion{}, base); return err }, "usuarios_config"},
		"perfiles_runtime": {func() error {
			_, _, err := componerProcesoADMINConRuntime(administracion.Configuracion{}, base, configuracionRuntimeADMIN{})
			return err
		}, "usuarios_config"},
		"usuarios": {func() error {
			_, _, err := componerProcesoUsuariosMetadatosADMIN(administracion.Configuracion{}, base, configuracionUsuariosMetadatosPrivada{PoolLector: ruta})
			return err
		}, "configuracion"},
	} {
		err := caso.componer()
		if !errors.Is(err, administracion.ErrConfiguracion) || !strings.HasSuffix(err.Error(), ": etapa="+caso.etapa) || strings.Contains(err.Error(), ruta) {
			t.Fatalf("%s: error de arranque sin etapa o con datos privados: %v", nombre, err)
		}
	}
	base.Identidad.EspacioIdentidad = "espacio:admin"
	_, _, err := componerProcesoUsuariosMetadatosADMIN(administracion.Configuracion{EmisorIdentidad: "emisor:otro"}, base, configuracionUsuariosMetadatosPrivada{})
	if !errors.Is(err, administracion.ErrConfiguracion) || !strings.HasSuffix(err.Error(), ": etapa=emisor_identidad") {
		t.Fatalf("emisor distinto del espacio de identidad aceptado: %v", err)
	}
	if err := errorArranque("escucha"); !errors.Is(err, administracion.ErrConfiguracion) || !strings.HasSuffix(err.Error(), ": etapa=escucha") {
		t.Fatalf("etapa perdida: %v", err)
	}
}

// Ejecuta el main real en un hijo: log.Fatal termina el proceso y deja un
// diagnóstico sin abrir archivos de DSN ni iniciar el servidor.
func TestArranqueRechazaUsuariosAntesDePools(t *testing.T) {
	if os.Getenv("VEC_ADMIN_ARRANQUE_PRUEBA") == "1" {
		main()
		t.Fatal("arranque inesperado")
	}
	base := configuracionPrivadaPrueba(t)
	baseJSON, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(base.Pools.CuentasADMIN)
	runtime := configuracionRuntimeADMIN{Version: 1, PoolContexto: filepath.Join(dir, "contexto.json"),
		FuenteIdentificadoresArchivo: filepath.Join(dir, "identificadores.json"),
		FuenteIdentificadoresSHA256:  strings.Repeat("a", 64), ProcesoContexto: "vec_admin"}
	runtimeJSON, err := json.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	baseRuta := archivoPrivadoPrueba(t, string(baseJSON))
	runtimeRuta := archivoPrivadoPrueba(t, string(runtimeJSON))
	usuariosInvalidos := archivoPrivadoPrueba(t, `{}`)
	for _, caso := range []struct {
		nombre, ruta string
	}{
		{"ausente", ""},
		{"invalida", usuariosInvalidos},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestArranqueRechazaUsuariosAntesDePools$")
			cmd.Env = []string{
				"VEC_ADMIN_ARRANQUE_PRUEBA=1",
				"VEC_ADMIN_USUARIOS_CONFIG_FILE=" + caso.ruta,
				"VEC_ADMIN_RETIRADA_EN=2030-01-01T00:00:00Z",
				"VEC_ADMIN_PERFILES_CONFIG_FILE=" + baseRuta,
				"VEC_ADMIN_RUNTIME_CONFIG_FILE=" + runtimeRuta,
			}
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "etapa=usuarios_config") ||
				strings.Contains(string(out), "pool_") || strings.Contains(string(out), baseRuta) || strings.Contains(string(out), usuariosInvalidos) {
				t.Fatalf("arranque no cerrado antes de conexiones: salida=%q, error=%v", out, err)
			}
		})
	}
}
