package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

func archivosComparacionConcursos(t *testing.T) (string, string, string, string) {
	t.Helper()
	rutas := [2]string{}
	shas := [2]string{}
	for i, nombre := range []string{"comparacion_concursos_v1", "comparacion_concursos_v2"} {
		rutas[i] = filepath.Join("testdata", nombre+".json")
		datos, err := os.ReadFile(rutas[i])
		if err != nil {
			t.Fatal(err)
		}
		suma := sha256.Sum256(datos)
		shas[i] = hex.EncodeToString(suma[:])
	}
	return rutas[0], shas[0], rutas[1], shas[1]
}
func escribirComparacionConcursos(t *testing.T, datos []byte) (string, string) {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "reglas_privadas.json")
	if err := os.WriteFile(ruta, datos, 0600); err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(datos)
	return ruta, hex.EncodeToString(suma[:])
}

func TestCLIComparacionConcursosReproducible(t *testing.T) {
	a, ha, n, hn := archivosComparacionConcursos(t)
	var original []byte
	for i := 0; i < 2; i++ {
		var salida, diagnostico bytes.Buffer
		if rc := ejecutarDiferenciaConcursos(&salida, &diagnostico, a, ha, n, hn, maximoBytesArchivo); rc != 0 || diagnostico.Len() != 0 {
			t.Fatalf("%d %s", rc, diagnostico.String())
		}
		var diff domain.DiferenciaConfiguracion
		if err := json.Unmarshal(salida.Bytes(), &diff); err != nil {
			t.Fatal(err)
		}
		if diff.Esquema != "vec.provision.diferencia_configuracion.v1" || diff.Alcance != "comparacion_reglas" || len(diff.Cambios) != 10 || diff.Anterior.Version != "revision:zeta" || diff.Nuevo.Version != "revision:alfa" {
			t.Fatalf("salida inesperada: %+v", diff)
		}
		if i == 1 && !bytes.Equal(original, salida.Bytes()) {
			t.Fatal("salida no reproducible")
		}
		original = append([]byte(nil), salida.Bytes()...)
	}
	var salida, diagnostico bytes.Buffer
	if rc := ejecutarDiferenciaConcursos(&salida, &diagnostico, a, ha, a, ha, maximoBytesArchivo); rc != 0 || !bytes.Contains(salida.Bytes(), []byte(`"cambios":[]`)) {
		t.Fatalf("identidad no vacía: %d %s", rc, diagnostico.String())
	}
}

func TestCLIComparacionConcursosContratosHuellasYLimites(t *testing.T) {
	for _, caso := range []string{"sha_anterior", "sha_nueva", "archivo", "limite", "exceso_2mib", "desconocido", "duplicado", "documentos", "entrada_personal", "convocatoria", "version_contradictoria", "configuracion_invalida"} {
		t.Run(caso, func(t *testing.T) {
			a, ha, n, hn := archivosComparacionConcursos(t)
			limite := int64(maximoBytesArchivo)
			datos, err := os.ReadFile(n)
			if err != nil {
				t.Fatal(err)
			}
			switch caso {
			case "sha_anterior":
				ha = strings.Repeat("0", 64)
			case "sha_nueva":
				hn = strings.Repeat("0", 64)
			case "archivo":
				n = filepath.Join(t.TempDir(), "ruta_privada_ausente")
			case "limite":
				limite = 10
			case "exceso_2mib":
				n, hn = escribirComparacionConcursos(t, bytes.Repeat([]byte(" "), 2*1024*1024+1))
			case "desconocido":
				n, hn = escribirComparacionConcursos(t, bytes.Replace(datos, []byte(`"schema_version"`), []byte(`"schema_VERSION"`), 1))
			case "duplicado":
				n, hn = escribirComparacionConcursos(t, append([]byte(`{"version":"repetida",`), datos[1:]...))
			case "documentos":
				n, hn = escribirComparacionConcursos(t, append(datos, []byte(`{}`)...))
			case "entrada_personal":
				n, hn = escribirComparacionConcursos(t, append([]byte(`{"entrada":{},`), datos[1:]...))
			case "convocatoria":
				n, hn = escribirComparacionConcursos(t, bytes.Replace(datos, []byte(`"convocatoria_ref": "convocatoria:sintetica:concursos:v1"`), []byte(`"convocatoria_ref": "convocatoria:otra"`), 1))
			case "version_contradictoria":
				n, hn = escribirComparacionConcursos(t, bytes.Replace(datos, []byte(`revision:alfa`), []byte(`revision:zeta`), 1))
			case "configuracion_invalida":
				n, hn = escribirComparacionConcursos(t, bytes.Replace(datos, []byte(`"divisor": 365`), []byte(`"divisor": 0`), 1))
			}
			var salida, diagnostico bytes.Buffer
			if rc := ejecutarDiferenciaConcursos(&salida, &diagnostico, a, ha, n, hn, limite); rc != 2 || salida.Len() != 0 || !json.Valid(diagnostico.Bytes()) {
				t.Fatalf("rechazo incorrecto: %d %s", rc, diagnostico.String())
			}
			if strings.Contains(diagnostico.String(), "ruta_privada") || strings.Contains(diagnostico.String(), "reglas_privadas") {
				t.Fatal("ruta filtrada")
			}
		})
	}
}

type escritorCortoConcursos struct{}

func (escritorCortoConcursos) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestCLIComparacionConcursosSalidaCorta(t *testing.T) {
	a, ha, n, hn := archivosComparacionConcursos(t)
	var diagnostico bytes.Buffer
	if rc := ejecutarDiferenciaConcursos(escritorCortoConcursos{}, &diagnostico, a, ha, n, hn, maximoBytesArchivo); rc != 2 {
		t.Fatal("escritura corta aceptada")
	}
	if !bytes.Contains(diagnostico.Bytes(), []byte(`salida_fallida`)) {
		t.Fatal("diagnóstico incorrecto")
	}
}

func TestCLIComparacionConcursosDespachoYArgumentos(t *testing.T) {
	a, ha, n, hn := archivosComparacionConcursos(t)
	args := []string{"--modo", "concursos", "--comparar-reglas", "--reglas", a, "--reglas-sha256", ha, "--reglas-nuevas", n, "--reglas-nuevas-sha256", hn}
	var directa, cli, diagnostico bytes.Buffer
	if ejecutarDiferenciaConcursos(&directa, &diagnostico, a, ha, n, hn, maximoBytesArchivo) != 0 || ejecutar(args, &cli, &diagnostico, simulacionbaremo.Servicio{}) != 0 || !bytes.Equal(directa.Bytes(), cli.Bytes()) {
		t.Fatal("despacho no conserva el comparador")
	}
	for _, caso := range []string{"entrada", "entrada-sha256", "ejemplo", "listar-ejemplos", "sin_comparar", "sin_reglas", "posicional"} {
		t.Run(caso, func(t *testing.T) {
			invalida := append([]string{}, args...)
			switch caso {
			case "entrada", "entrada-sha256", "ejemplo":
				invalida = append(invalida, "--"+caso, "dato_privado_no_leer")
			case "listar-ejemplos":
				invalida = append(invalida, "--listar-ejemplos")
			case "sin_comparar":
				invalida = append(invalida[:2], invalida[3:]...)
			case "sin_reglas":
				invalida[4] = ""
			case "posicional":
				invalida = append(invalida, "dato_privado_no_leer")
			}
			var salida, diagnostico bytes.Buffer
			if rc := ejecutar(invalida, &salida, &diagnostico, simulacionbaremo.Servicio{}); rc != 2 || salida.Len() != 0 || !bytes.Contains(diagnostico.Bytes(), []byte(`"codigo":"argumentos_invalidos"`)) {
				t.Fatalf("argumentos aceptados: %d %s", rc, diagnostico.String())
			}
		})
	}
}
