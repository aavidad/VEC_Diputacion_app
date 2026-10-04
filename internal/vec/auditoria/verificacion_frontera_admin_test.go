package auditoria

import (
	"encoding/json"
	"testing"
)

// Vector encuadrado con Python bytes/SHA256, independiente del verificador Go.
func TestFronteraAdminTecnicaVectorIndependienteYRechazoActorPrestado(t *testing.T) {
	raw := []byte(`{"tipo_registro":"frontera_admin_tecnica","evento_ref":"evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","operador_login":"login_admin_ensayo","accion":"controlar_frontera_admin_v1","recurso_ref":"solicitud_admin:b6d725ae832695adb3d6c948d1104d70","resultado":"denegado","codigo_ref":"autenticacion_requerida","proceso":"vec_admin","canal":"administracion_privilegiada","finalidad_ref":"control_frontera_admin","correlacion_ref":"correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","auditoria_ref":"aud_v3_fat_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","secuencia":1,"anterior_sha256":"0000000000000000000000000000000000000000000000000000000000000000","huella_sha256":"4e26709f855509ab7749ae4c88d07dc174cdad84299d374a38fa3802b0858a79","registrada_en":"2026-10-04T12:00:00.000001Z","evento_material_sha256":"91fde8874afd229ab8070c208082a94013f67f079db2211f581cc3952dffb1d6","modulo_id":"administracion"}`)
	var r RegistroFronteraAdminTecnicaV1
	if json.Unmarshal(raw, &r) != nil {
		t.Fatal("vector_no_json")
	}
	if _, motivo, campo := CotejarRegistroFronteraAdminTecnicaV1(r, 1); motivo != "" {
		t.Fatalf("vector:%s/%s", motivo, campo)
	}
	r.CodigoRef = "codigo_inventado"
	if _, motivo, _ := CotejarRegistroFronteraAdminTecnicaV1(r, 1); motivo == "" {
		t.Fatal("codigo_abierto")
	}
	if json.Unmarshal(raw, &r) != nil {
		t.Fatal("vector_no_json")
	}
	r.Resultado = "permitido"
	if _, motivo, _ := CotejarRegistroFronteraAdminTecnicaV1(r, 1); motivo == "" {
		t.Fatal("permiso_humano_ficticio")
	}
	if json.Unmarshal(raw, &r) != nil {
		t.Fatal("vector_no_json")
	}
	r.RecursoRef = "per_persona_inventada"
	if _, motivo, _ := CotejarRegistroFronteraAdminTecnicaV1(r, 1); motivo == "" {
		t.Fatal("recurso_prestado")
	}
}
