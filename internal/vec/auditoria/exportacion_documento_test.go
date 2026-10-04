package auditoria

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func fixtureExportacionPrueba(t *testing.T, nombre string) ([]byte, domain.CoberturaCheckpoint) {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/" + nombre)
	if err != nil {
		t.Fatal(err)
	}
	var cabecera struct {
		Manifiesto domain.CoberturaCheckpoint `json:"manifiesto"`
	}
	if err := json.Unmarshal(b, &cabecera); err != nil {
		t.Fatal(err)
	}
	return b, cabecera.Manifiesto
}

func TestDocumentoExportacionAdmiteEsquemasInstalados(t *testing.T) {
	for _, nombre := range []string{
		"cadena_mixta_v2.json", "preperfil_mixta_v3.json", "fuentes_iniciales_ad174.json",
		"unidad_inicial_ad176.json", "bootstrap_intentos_ad179.json",
	} {
		t.Run(nombre, func(t *testing.T) {
			b, cobertura := fixtureExportacionPrueba(t, nombre)
			esquema, informe, err := VerificarDocumentoExportacionAuditoria(b, cobertura, 1<<20, 100)
			if err != nil || esquema == "" || informe.Estado != "verificada" {
				t.Fatalf("fixture: esquema=%q estado=%q err=%v fallo=%+v", esquema, informe.Estado, err, informe.Fallo)
			}
		})
	}
	d := vectorCadena()
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	cobertura := domain.CoberturaCheckpoint(d.Manifiesto)
	if esquema, informe, err := VerificarDocumentoExportacionAuditoria(b, cobertura, 1<<20, 10); err != nil ||
		esquema != EsquemaVerificacion || informe.Estado != "verificada" {
		t.Fatalf("v1: esquema=%q estado=%q err=%v", esquema, informe.Estado, err)
	}
}

func TestDocumentoExportacionCierraJSONYManipulacion(t *testing.T) {
	b, cobertura := fixtureExportacionPrueba(t, "cadena_mixta_v2.json")
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		b      []byte
	}{
		{"duplicada", []byte(strings.Replace(string(b), `"esquema":`, `"esquema":"otro","esquema":`, 1))},
		{"alias", []byte(strings.Replace(string(b), `"esquema":`, `"Esquema":`, 1))},
		{"null", []byte(strings.Replace(string(b), `"esquema":"`+EsquemaVerificacionMixta+`"`, `"esquema":null`, 1))},
		{"esquema_futuro", []byte(strings.Replace(string(b), EsquemaVerificacionMixta, "vec.auditoria.verificacion.v4", 1))},
		{"limite", b},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			limite := int64(1 << 20)
			if caso.nombre == "limite" {
				limite = int64(len(b) - 1)
			}
			if _, _, err := VerificarDocumentoExportacionAuditoria(caso.b, cobertura, limite, 100); !errors.Is(err, ErrDocumentoExportacionAuditoriaInvalido) {
				t.Fatalf("JSON no cerrado: %v", err)
			}
		})
	}
	d.Registros[0].TipoRegistro = "desconocido"
	alterado, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	_, informe, err := VerificarDocumentoExportacionAuditoria(alterado, cobertura, 1<<20, 100)
	if err != nil || informe.Estado != "rechazada" {
		t.Fatalf("cadena manipulada: err=%v informe=%+v", err, informe)
	}
}
