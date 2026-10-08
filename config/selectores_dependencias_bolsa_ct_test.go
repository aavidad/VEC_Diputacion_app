package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// El inventario debe nombrar exactamente las variables VEC_CT_* y VEC_BOLSA_*
// que lee el código de composición, y cada variable suya debe existir en él.
func TestInventarioSelectoresBolsaCTCubreElCodigo(t *testing.T) {
	inv, err := CargarInventarioSelectoresBolsaCT()
	if err != nil {
		t.Fatalf("inventario: %v", err)
	}
	enCodigo := map[string]bool{}
	literal := regexp.MustCompile(`"(VEC_[A-Z0-9_]+)"`)
	for _, raiz := range []string{".", filepath.Join("..", "internal", "app", "bootstrap"), filepath.Join("..", "cmd", "vec-server")} {
		err := filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
				return err
			}
			datos, err := os.ReadFile(ruta)
			if err != nil {
				return err
			}
			for _, m := range literal.FindAllStringSubmatch(string(datos), -1) {
				enCodigo[m[1]] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("recorrer %s: %v", raiz, err)
		}
	}
	enInventario := map[string]bool{}
	for _, v := range inv.Variables {
		enInventario[v.Variable] = true
		if !enCodigo[v.Variable] {
			t.Errorf("%s figura en el inventario pero no la lee el código", v.Variable)
		}
	}
	for nombre := range enCodigo {
		if (strings.HasPrefix(nombre, "VEC_CT_") || strings.HasPrefix(nombre, "VEC_BOLSA_")) && !enInventario[nombre] {
			t.Errorf("%s la lee el código y falta en el inventario", nombre)
		}
	}
}

func TestDependenciasAusentesSelectoresBolsaCT(t *testing.T) {
	inv, err := CargarInventarioSelectoresBolsaCT()
	if err != nil {
		t.Fatalf("inventario: %v", err)
	}
	entorno := func(valores map[string]string) func(string) string {
		return func(n string) string { return valores[n] }
	}
	// Selector apagado o con otro valor: no se comprueba nada.
	for _, valor := range []string{"", "false", "si"} {
		if a := inv.DependenciasAusentes(entorno(map[string]string{EnvCTCancelacionEnabled: valor})); len(a) != 0 {
			t.Fatalf("valor %q: avisos inesperados %v", valor, a)
		}
	}
	// Selector encendido sin sus catálogos ni su conexión.
	a := inv.DependenciasAusentes(entorno(map[string]string{EnvCTCancelacionEnabled: " true "}))
	faltan := map[string]bool{}
	for _, x := range a {
		if x.Variable != EnvCTCancelacionEnabled {
			t.Fatalf("aviso de otra variable: %v", x)
		}
		faltan[x.Falta] = true
	}
	for _, r := range []string{EnvContratacionTemporalDatabaseURL, EnvCTReglasSourcePath, EnvCTMotivosCancelacionSourcePath} {
		if !faltan[r] {
			t.Fatalf("falta el aviso de %s en %v", r, a)
		}
	}
	// Un selector requerido solo cuenta con "true".
	a = inv.DependenciasAusentes(entorno(map[string]string{
		"VEC_BOLSA_POLITICA_OFERTAS_ENABLED":         "true",
		EnvBolsaBorradoresEnabled:                    "1",
		EnvBolsaPoliticaOfertasCalculadorDatabaseURL: "x",
	}))
	encontrado := false
	for _, x := range a {
		if x.Variable == "VEC_BOLSA_POLITICA_OFERTAS_ENABLED" {
			if x.Falta != EnvBolsaBorradoresEnabled {
				t.Fatalf("aviso inesperado %v", x)
			}
			encontrado = true
		}
	}
	if !encontrado {
		t.Fatalf("debía avisar de %s con valor distinto de true: %v", EnvBolsaBorradoresEnabled, a)
	}
	if inv.DependenciasAusentes(nil) != nil {
		t.Fatal("sin lector de entorno no debe avisar")
	}
}

func TestInventarioSelectoresBolsaCTRechazaDatosInvalidos(t *testing.T) {
	for nombre, datos := range map[string]string{
		"vacio":              `{}`,
		"version":            `{"version":2,"variables":[{"variable":"VEC_A","tipo":"ajuste"}]}`,
		"campo desconocido":  `{"version":1,"variables":[{"variable":"VEC_A","tipo":"ajuste","x":1}]}`,
		"tipo":               `{"version":1,"variables":[{"variable":"VEC_A","tipo":"otro"}]}`,
		"prefijo":            `{"version":1,"variables":[{"variable":"OTRA","tipo":"ajuste"}]}`,
		"duplicada":          `{"version":1,"variables":[{"variable":"VEC_A","tipo":"ajuste"},{"variable":"VEC_A","tipo":"ajuste"}]}`,
		"requisito ausente":  `{"version":1,"variables":[{"variable":"VEC_A","tipo":"ajuste","requiere":["VEC_B"]}]}`,
		"requisito circular": `{"version":1,"variables":[{"variable":"VEC_A","tipo":"ajuste","requiere":["VEC_A"]}]}`,
	} {
		if _, err := decodificarInventarioSelectoresBolsaCT([]byte(datos)); !errors.Is(err, ErrInventarioSelectoresBolsaCTInvalido) {
			t.Errorf("%s: se esperaba rechazo, error %v", nombre, err)
		}
	}
}
