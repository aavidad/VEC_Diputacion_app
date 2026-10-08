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

func TestCLI189RutaRealJSONCerradoSinActorPrestado(t *testing.T) {
	var r auditoria.RegistroFronteraAdminTecnicaV1
	if err := json.Unmarshal([]byte(`{"tipo_registro":"frontera_admin_tecnica","evento_ref":"evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","operador_login":"login_admin_ensayo","accion":"controlar_frontera_admin_v1","recurso_ref":"solicitud_admin:b6d725ae832695adb3d6c948d1104d70","resultado":"denegado","codigo_ref":"autenticacion_requerida","proceso":"vec_admin","canal":"administracion_privilegiada","finalidad_ref":"control_frontera_admin","correlacion_ref":"correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","auditoria_ref":"aud_v3_fat_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","secuencia":1,"anterior_sha256":"0000000000000000000000000000000000000000000000000000000000000000","huella_sha256":"4e26709f855509ab7749ae4c88d07dc174cdad84299d374a38fa3802b0858a79","registrada_en":"2026-10-04T12:00:00.000001Z","evento_material_sha256":"91fde8874afd229ab8070c208082a94013f67f079db2211f581cc3952dffb1d6","modulo_id":"administracion"}`), &r); err != nil {
		t.Fatal(err)
	}
	c := auditoria.CoberturaCadena{CadenaID: "cadena:comun:interna", PrimeraSecuencia: 1, UltimaSecuencia: 1, AnteriorSHA256: r.AnteriorSHA256, CabezaSHA256: r.HuellaSHA256, Registros: 1}
	d := auditoria.DocumentoVerificacionMixta{Esquema: auditoria.EsquemaVerificacionFronteraAdminTecnicaV1, Manifiesto: c, Registros: []auditoria.RegistroMixtoV2{{TipoRegistro: r.TipoRegistro, FronteraAdminTecnica: &r}}}
	cp, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(ruta, cp, 0600); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", ruta, "--max-bytes", "8192", "--max-registros", "2"}
	for _, caso := range []string{"valido", "campo_extra", "codigo_desconocido", "codigo_omitido", "codigo_null", "perfil_prestado", "duplicado", "hash", "esquema_anterior"} {
		t.Run(caso, func(t *testing.T) {
			entrada := string(b)
			esperado := 2
			switch caso {
			case "valido":
				esperado = 0
			case "campo_extra":
				entrada = strings.Replace(entrada, `"codigo_ref":`, `"certificado":"SECRET","codigo_ref":`, 1)
			case "codigo_desconocido":
				entrada = strings.Replace(entrada, `"codigo_ref":"autenticacion_requerida"`, `"codigo_ref":"SECRET"`, 1)
				esperado = 1
			case "codigo_omitido":
				entrada = strings.Replace(entrada, `"codigo_ref":"autenticacion_requerida",`, ``, 1)
			case "codigo_null":
				entrada = strings.Replace(entrada, `"codigo_ref":"autenticacion_requerida"`, `"codigo_ref":null`, 1)
			case "perfil_prestado":
				entrada = strings.Replace(entrada, `"frontera_admin_tecnica":`, `"intento":{},"frontera_admin_tecnica":`, 1)
			case "duplicado":
				entrada = strings.Replace(entrada, `"codigo_ref":`, `"codigo_ref":"SECRET","codigo_ref":`, 1)
			case "hash":
				entrada = strings.Replace(entrada, r.EventoMaterialSHA256, strings.Repeat("f", 64), 1)
				esperado = 1
			case "esquema_anterior":
				entrada = strings.Replace(entrada, d.Esquema, auditoria.EsquemaVerificacionMixta, 1)
			}
			var salida bytes.Buffer
			code := ejecutar(args, strings.NewReader(entrada), &salida)
			if code != esperado || strings.Contains(salida.String(), "SECRET") {
				t.Fatalf("CLI189 caso%s exit%d esperado%d salida%s", caso, code, esperado, salida.String())
			}
		})
	}
}
