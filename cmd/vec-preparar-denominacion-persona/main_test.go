package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const metadataPrueba = `{"version":1,"entorno":"desarrollo","norma":{"referencia":"norma:denominacion:prueba:v1","case":"fold","forma_unicode":"NFC","separadores":" -'","max_bytes":512,"max_tokens":12},"cifrado":{"referencia":"clave:denominacion:cifrado:prueba","version":1,"revocada":false,"retener_hasta":"0001-01-01T00:00:00Z"},"busqueda":{"referencia":"clave:denominacion:busqueda:prueba","version":1,"revocada":false,"retener_hasta":"0001-01-01T00:00:00Z"},"cifrado_retenidas":[]}`

func fixtureCLI(t *testing.T) ([]string, string, string) {
	t.Helper()
	dir := t.TempDir()
	if os.Chmod(dir, 0700) != nil {
		t.Fatal("directorio_privado")
	}
	write := func(n string, b []byte) string {
		p := filepath.Join(dir, n)
		if os.WriteFile(p, b, 0600) != nil {
			t.Fatal("fixture")
		}
		return p
	}
	cfg := write("config.json", []byte(metadataPrueba))
	nombre := write("nombre.privado", []byte("Elena Márquez"))
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	maestra := write("maestra.privada", key)
	textos, e := filepath.Abs("../../web/static/textos/es/persona-denominacion-preparar.json")
	if e != nil {
		t.Fatal("catalogo")
	}
	salida := filepath.Join(dir, "sobre.json")
	return []string{"--configuracion", cfg, "--nombre-fichero", nombre, "--maestra-fichero", maestra, "--salida", salida, "--persona-ref", "per_aaaaaaaaaaaaaaaaaaaaaaaa", "--version-esperada", "0", "--procedencia-ref", "prc_bbbbbbbbbbbbbbbbbbbbbbbb", "--ambito-ref", "ambito:admin", "--sintetico", "--textos", textos, "--idioma", "es"}, salida, nombre
}
func TestCLIProtegidaReintentoSinNuevoNonce(t *testing.T) {
	args, salida, nombre := fixtureCLI(t)
	var out, errout bytes.Buffer
	if ejecutar(args, &out, &errout) != 0 {
		t.Fatalf("preparacion:%s", errout.String())
	}
	b, e := os.ReadFile(salida)
	if e != nil {
		t.Fatal("artefacto")
	}
	var d documentoPreparado
	if decodificarDocumento(b, &d) != nil {
		t.Fatal("documento")
	}
	if d.Estado != "preparada_pendiente_cotejo_y_autorizacion" || huellaSobre(d.SobreCanonico) != d.Preparacion.SobreSHA256 || bytes.Contains(b, []byte("Elena Márquez")) {
		t.Fatal("salida_clara_o_falsa_autoridad")
	}
	canon, e := json.Marshal(d.Preparacion.Sobre)
	if e != nil || !bytes.Equal(canon, d.SobreCanonico) {
		t.Fatal("canon_pascal_case")
	}
	if strings.Contains(out.String(), nombre) || strings.Contains(out.String(), "Elena") {
		t.Fatal("stdout_privado")
	}
	out.Reset()
	errout.Reset()
	if ejecutar(args, &out, &errout) != 0 {
		t.Fatalf("reintento:%s", errout.String())
	}
	replay, e := os.ReadFile(salida)
	if e != nil || !bytes.Equal(b, replay) || !strings.Contains(out.String(), `"reutilizada":true`) {
		t.Fatal("reintento_regenero_nonce")
	}
	if os.WriteFile(nombre, []byte("Irene Navarro"), 0600) != nil {
		t.Fatal("fixture")
	}
	out.Reset()
	errout.Reset()
	if ejecutar(args, &out, &errout) == 0 {
		t.Fatal("nombre_cambiado_reutilizado")
	}
	replay, e = os.ReadFile(salida)
	if e != nil || !bytes.Equal(b, replay) {
		t.Fatal("reintento_modifico_original")
	}
}
func TestCLIRechazaNombrePublicoYVersionImplicita(t *testing.T) {
	args, salida, nombre := fixtureCLI(t)
	if os.Chmod(nombre, 0644) != nil {
		t.Fatal("chmod")
	}
	var out, errores bytes.Buffer
	if ejecutar(args, &out, &errores) == 0 {
		t.Fatal("nombre_publico")
	}
	if _, e := os.Stat(salida); !os.IsNotExist(e) {
		t.Fatal("salida_parcial")
	}
	if os.Chmod(nombre, 0600) != nil {
		t.Fatal("chmod")
	}
	var sinVersion []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--version-esperada" {
			i++
			continue
		}
		sinVersion = append(sinVersion, args[i])
	}
	if ejecutar(sinVersion, &out, &errores) == 0 {
		t.Fatal("version_por_defecto")
	}
}

func TestCLIReferenciasCA32AntesDePrepararYReutilizar(t *testing.T) {
	args, salida, _ := fixtureCLI(t)
	modificar := func(flag, valor string) []string {
		x := append([]string(nil), args...)
		for i := range x {
			if x[i] == flag {
				x[i+1] = valor
				break
			}
		}
		return x
	}
	casos := []struct{ flag, valor string }{
		{"--procedencia-ref", "procedencia:sintetica"},
		{"--procedencia-ref", "prc_" + strings.Repeat("a", 21)},
		{"--procedencia-ref", "prc_" + strings.Repeat("a", 125)},
		{"--procedencia-ref", "prc_" + strings.Repeat("a", 22) + ":"},
		{"--persona-ref", "per_" + strings.Repeat("a", 125)},
	}
	for _, c := range casos {
		var out, errout bytes.Buffer
		if ejecutar(modificar(c.flag, c.valor), &out, &errout) == 0 {
			t.Fatal("referencia_no_publicable_preparada")
		}
		if _, e := os.Stat(salida); !os.IsNotExist(e) {
			t.Fatal("referencia_invalida_creo_salida")
		}
	}
	var out, errout bytes.Buffer
	if ejecutar(args, &out, &errout) != 0 {
		t.Fatal("preparacion_valida")
	}
	original, e := os.ReadFile(salida)
	if e != nil {
		t.Fatal("artefacto_original")
	}
	for _, c := range casos {
		out.Reset()
		errout.Reset()
		if ejecutar(modificar(c.flag, c.valor), &out, &errout) == 0 {
			t.Fatal("referencia_no_publicable_reutilizada")
		}
		actual, e := os.ReadFile(salida)
		if e != nil || !bytes.Equal(actual, original) {
			t.Fatal("referencia_invalida_modifico_original")
		}
	}
	if !refProcedenciaCA32("prc_" + strings.Repeat("a", 124)) {
		t.Fatal("limite_CA32_rechazado")
	}
}

func TestCLIRechazaMaterialMaestroComoNombrePorContenido(t *testing.T) {
	for _, modo := range []string{"enlace_duro", "copia"} {
		t.Run(modo, func(t *testing.T) {
			args, salida, nombre := fixtureCLI(t)
			var maestra string
			for i := range args {
				if args[i] == "--maestra-fichero" {
					maestra = args[i+1]
					break
				}
			}
			// Material sintético de 32 bytes que también sería un nombre UTF8 válido.
			material := []byte("Elena Marquez Navarro Lopez Luna")
			if len(material) != 32 {
				t.Fatal("fixture_material")
			}
			if os.WriteFile(maestra, material, 0600) != nil {
				t.Fatal("fixture_maestra")
			}
			alias := filepath.Join(filepath.Dir(nombre), "nombre_equivocado.privado")
			if modo == "enlace_duro" {
				if os.Link(maestra, alias) != nil {
					t.Fatal("fixture_enlace")
				}
			} else {
				if os.WriteFile(alias, material, 0600) != nil {
					t.Fatal("fixture_copia")
				}
			}
			equivocados := append([]string(nil), args...)
			for i := range equivocados {
				if equivocados[i] == "--nombre-fichero" {
					equivocados[i+1] = alias
					break
				}
			}
			var out, errores bytes.Buffer
			if ejecutar(equivocados, &out, &errores) == 0 {
				t.Fatal("material_preparado_como_nombre")
			}
			if _, e := os.Stat(salida); !os.IsNotExist(e) {
				t.Fatal("rechazo_creo_salida")
			}
			if bytes.Contains(out.Bytes(), material) || bytes.Contains(errores.Bytes(), material) {
				t.Fatal("material_en_diagnostico")
			}
			out.Reset()
			errores.Reset()
			if ejecutar(args, &out, &errores) != 0 {
				t.Fatal("nombre_legitimo_diferente_rechazado")
			}
			original, e := os.ReadFile(salida)
			if e != nil {
				t.Fatal("preparacion_original")
			}
			out.Reset()
			errores.Reset()
			if ejecutar(equivocados, &out, &errores) == 0 {
				t.Fatal("material_reutilizado_como_nombre")
			}
			actual, e := os.ReadFile(salida)
			if e != nil || !bytes.Equal(original, actual) {
				t.Fatal("rechazo_modifico_sobre_original")
			}
			if bytes.Contains(out.Bytes(), material) || bytes.Contains(errores.Bytes(), material) {
				t.Fatal("material_en_diagnostico_reintento")
			}
		})
	}
}
