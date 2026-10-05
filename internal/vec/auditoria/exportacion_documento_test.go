package auditoria

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestDocumentoExportacionAdminConservaHistoriaPreperfil(t *testing.T) {
	for _, esquema := range []string{EsquemaVerificacionGobiernoUsuarios, EsquemaVerificacionFronteraAdminTecnicaV1} {
		for _, registro := range vectoresAD171Prueba(t) {
			d := documentoAD171Prueba(registro)
			d.Esquema = esquema
			antes, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			recibido, informe, err := VerificarDocumentoExportacionAuditoria(antes, domain.CoberturaCheckpoint(d.Manifiesto), 1<<20, 10)
			if err != nil || recibido != esquema || informe.Estado != "verificada" {
				t.Fatalf("historia AD171: esquema=%q tipo=%q err=%v fallo=%+v", esquema, registro.TipoRegistro, err, informe.Fallo)
			}
			despues, err := json.Marshal(d)
			if err != nil || !bytes.Equal(antes, despues) {
				t.Fatal("historia modificada al verificar")
			}
		}
	}
}

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

func TestDocumentoExportacionConservaGobiernoYFronteraAdmin(t *testing.T) {
	for _, nombre := range []string{"exportacion_ad188.json", "exportacion_ad189.json"} {
		t.Run(nombre, func(t *testing.T) {
			// Vectores sintéticos encuadrados con Python/SHA256, independientes
			// del despacho y reutilizados por la CLI con firma y TSA DEV reales.
			b, err := os.ReadFile("../../../cmd/vec-auditoria-checkpoint/testdata/" + nombre)
			if err != nil {
				t.Fatal(err)
			}
			var d DocumentoVerificacionMixta
			if err := json.Unmarshal(b, &d); err != nil {
				t.Fatal(err)
			}
			cobertura := domain.CoberturaCheckpoint(d.Manifiesto)
			esquema, informe, err := VerificarDocumentoExportacionAuditoria(b, cobertura, 1<<20, 10)
			if err != nil || esquema != d.Esquema || informe.Estado != "verificada" || informe.ActorPerfilContextoCotejados {
				t.Fatalf("acto técnico: esquema=%q informe=%+v err=%v", esquema, informe, err)
			}
			for _, caso := range []string{"fecha", "codigo", "esquema_anterior", "campo_extra", "campo_omitido"} {
				t.Run(caso, func(t *testing.T) {
					alterado := string(b)
					switch caso {
					case "fecha":
						alterado = strings.Replace(alterado, "T08:30:00", "T08:30:01", 1)
						alterado = strings.Replace(alterado, "T12:00:00", "T12:00:01", 1)
					case "codigo":
						alterado = strings.Replace(alterado, "gobierno_usuarios_registrado", "contenido_privado_sintetico", 1)
						alterado = strings.Replace(alterado, "autenticacion_requerida", "contenido_privado_sintetico", 1)
					case "esquema_anterior":
						alterado = strings.Replace(alterado, d.Esquema, EsquemaVerificacionMixta, 1)
					case "campo_extra":
						alterado = strings.Replace(alterado, `"operador_login":`, `"actor_ref":"contenido_privado_sintetico","operador_login":`, 1)
					case "campo_omitido":
						var arbol map[string]any
						if err := json.Unmarshal(b, &arbol); err != nil {
							t.Fatal(err)
						}
						registro := arbol["registros"].([]any)[0].(map[string]any)
						for clave, valor := range registro {
							if clave != "tipo_registro" {
								delete(valor.(map[string]any), "modulo_id")
							}
						}
						omitido, err := json.Marshal(arbol)
						if err != nil {
							t.Fatal(err)
						}
						alterado = string(omitido)
					}
					_, rechazado, err := VerificarDocumentoExportacionAuditoria([]byte(alterado), cobertura, 1<<20, 10)
					if rechazado.Estado != "rechazada" || err != nil && !errors.Is(err, ErrDocumentoExportacionAuditoriaInvalido) {
						t.Fatalf("alteración admitida: err=%v informe=%+v", err, rechazado)
					}
					raw, err := json.Marshal(rechazado)
					if err != nil || strings.Contains(string(raw), "contenido_privado_sintetico") {
						t.Fatal("informe expone contenido rechazado")
					}
				})
			}
		})
	}
}
