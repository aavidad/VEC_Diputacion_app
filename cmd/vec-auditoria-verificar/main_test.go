package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func fixtureCLI(t *testing.T) ([]string, string) {
	t.Helper()
	checkpoint := `{"cadena_id":"cadena:sintetica:vacia","primera_secuencia":0,"ultima_secuencia":0,"anterior_sha256":"` + strings.Repeat("0", 64) + `","cabeza_sha256":"` + strings.Repeat("0", 64) + `","registros":0}`
	ruta := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(ruta, []byte(checkpoint), 0600); err != nil {
		t.Fatal(err)
	}
	return []string{"--checkpoint", ruta, "--max-bytes", "4096", "--max-registros", "10"},
		`{"esquema":"vec.auditoria.verificacion.v1","manifiesto":` + checkpoint + `,"registros":[]}`
}

func TestCLIComparaCheckpointSeparado(t *testing.T) {
	args, documento := fixtureCLI(t)
	var salida bytes.Buffer
	if codigo := ejecutar(args, strings.NewReader(documento), &salida); codigo != 0 {
		t.Fatalf("exit=%d output=%s", codigo, salida.String())
	}
	var informe auditoria.InformeVerificacion
	if json.Unmarshal(salida.Bytes(), &informe) != nil || !informe.CheckpointCotejado || informe.AutenticidadCheckpoint != "no_comprobada" {
		t.Fatalf("unexpected output=%s", salida.String())
	}
}

func TestCLIJSONEstrictoYLimites(t *testing.T) {
	args, documento := fixtureCLI(t)
	for _, entrada := range []string{
		documento + `{}`, documento + `dato_secreto`, `null`,
		strings.Replace(documento, `"registros":[]`, `"registros":null`, 1),
		strings.Replace(documento, `"primera_secuencia":0,`, ``, 1),
		strings.Replace(documento, `"registros":0`, `"registros":0,"REGISTROS":1`, 1),
		strings.Replace(documento, `"registros":0`, `"REGISTROS":0`, 1),
		strings.Replace(documento, `"esquema":`, `"esquema":"otro","esquema":`, 1),
		strings.Replace(documento, `"manifiesto":`, `"desconocido":"dato_secreto","manifiesto":`, 1),
		strings.Replace(documento, `"primera_secuencia":0`, `"primera_secuencia":0,"primera_secuencia":1`, 1),
		strings.Repeat(" ", 4097) + documento,
	} {
		var salida bytes.Buffer
		if codigo := ejecutar(args, strings.NewReader(entrada), &salida); codigo != 2 || strings.Contains(salida.String(), "dato_secreto") {
			t.Fatalf("untrusted input leaked or accepted: exit=%d output=%s", codigo, salida.String())
		}
	}
}

func TestCLIRechazaArgumentosYCheckpoint(t *testing.T) {
	args, documento := fixtureCLI(t)
	for _, argumentos := range [][]string{nil, {"--checkpoint", args[1]}, append(args, "extra"),
		{"--checkpoint", args[1], "--max-bytes", "1073741825", "--max-registros", "1"},
		{"--checkpoint", filepath.Join(t.TempDir(), "ausente"), "--max-bytes", "4096", "--max-registros", "1"}} {
		var salida bytes.Buffer
		if codigo := ejecutar(argumentos, strings.NewReader(documento), &salida); codigo != 2 {
			t.Fatalf("invalid arguments accepted: exit=%d", codigo)
		}
	}
	if err := os.WriteFile(args[1], []byte(`{"dato_secreto":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if codigo := ejecutar(args, strings.NewReader(documento), &salida); codigo != 2 || strings.Contains(salida.String(), "dato_secreto") {
		t.Fatalf("invalid checkpoint accepted or leaked: exit=%d output=%s", codigo, salida.String())
	}
}

func TestCLICadenaMixtaV2YJSONAmbiguo(t *testing.T) {
	contenido, err := os.ReadFile("testdata/cadena_mixta_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", "testdata/checkpoint_mixto_v2.json", "--max-bytes", "8192", "--max-registros", "2"}
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(contenido), &salida); codigo != 0 {
		t.Fatalf("mixta rechazada: codigo=%d salida=%s", codigo, salida.String())
	}
	var informe auditoria.InformeVerificacion
	if json.Unmarshal(salida.Bytes(), &informe) != nil || informe.Estado != "verificada" ||
		!informe.MaterialIntentoRecalculado || !informe.ActorPerfilContextoCotejados ||
		informe.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("alcance de CLI mixto incorrecto: %s", salida.String())
	}
	for _, alterado := range [][]byte{
		bytes.Replace(contenido, []byte(`"accion":"administracion.perfiles.consultar"`),
			[]byte(`"accion":"otra","accion":"administracion.perfiles.consultar"`), 1),
		bytes.Replace(contenido, []byte(`"tipo_registro":"intento_nominal"`),
			[]byte(`"tipo_registro":"intento_nominal","consumo":{}`), 1),
		bytes.Replace(contenido, []byte(`"contexto_canonico_base64":`),
			[]byte(`"campo_ajeno":"x","contexto_canonico_base64":`), 1),
		bytes.Replace(contenido, []byte(`"registros":[`), []byte(`"registros":null,"registros":[`), 1),
	} {
		salida.Reset()
		if codigo := ejecutar(args, bytes.NewReader(alterado), &salida); codigo != 2 {
			t.Fatalf("JSON mixto ambiguo aceptado: codigo=%d", codigo)
		}
	}
}

func TestCLIIntentoConAnteriorInvalidoNoReflejaDatos(t *testing.T) {
	contenido, err := os.ReadFile("testdata/cadena_mixta_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var documento auditoria.DocumentoVerificacionMixta
	if err := json.Unmarshal(contenido, &documento); err != nil {
		t.Fatal(err)
	}
	const datoPrivadoSintetico = "DNI-sintetico-12345678"
	documento.Registros[0].Intento.AnteriorSHA256 = datoPrivadoSintetico
	alterado, err := json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", "testdata/checkpoint_mixto_v2.json", "--max-bytes", "8192", "--max-registros", "2"}
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(alterado), &salida); codigo != 1 ||
		strings.Contains(salida.String(), datoPrivadoSintetico) {
		t.Fatalf("anterior invalido filtrado: codigo=%d salida=%s", codigo, salida.String())
	}
	var informe auditoria.InformeVerificacion
	if json.Unmarshal(salida.Bytes(), &informe) != nil || informe.Fallo == nil ||
		informe.Fallo.Clave != "anterior_sha256" || informe.Fallo.Obtenido != "no_admitido" {
		t.Fatalf("rechazo no acotado: %s", salida.String())
	}
}
