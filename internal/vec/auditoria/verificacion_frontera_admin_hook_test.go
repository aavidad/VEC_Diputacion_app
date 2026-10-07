package auditoria

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCadenaComunIncluyeFronteraTecnicaSinActorNominal(t *testing.T) {
	var r RegistroFronteraAdminTecnicaV1
	if err := json.Unmarshal([]byte(`{"tipo_registro":"frontera_admin_tecnica","evento_ref":"evento_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","operador_login":"login_admin_ensayo","accion":"controlar_frontera_admin_v1","recurso_ref":"solicitud_admin:b6d725ae832695adb3d6c948d1104d70","resultado":"denegado","codigo_ref":"autenticacion_requerida","proceso":"vec_admin","canal":"administracion_privilegiada","finalidad_ref":"control_frontera_admin","correlacion_ref":"correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","auditoria_ref":"aud_v3_fat_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","secuencia":1,"anterior_sha256":"0000000000000000000000000000000000000000000000000000000000000000","huella_sha256":"4e26709f855509ab7749ae4c88d07dc174cdad84299d374a38fa3802b0858a79","registrada_en":"2026-10-04T12:00:00.000001Z","evento_material_sha256":"91fde8874afd229ab8070c208082a94013f67f079db2211f581cc3952dffb1d6","modulo_id":"administracion"}`), &r); err != nil {
		t.Fatal(err)
	}
	d := DocumentoVerificacionMixta{Esquema: EsquemaVerificacionFronteraAdminTecnicaV1, Manifiesto: CoberturaCadena{CadenaID: "cadena:comun:interna", PrimeraSecuencia: 1, UltimaSecuencia: 1, AnteriorSHA256: r.AnteriorSHA256, CabezaSHA256: r.HuellaSHA256, Registros: 1}, Registros: []RegistroMixtoV2{{TipoRegistro: r.TipoRegistro, FronteraAdminTecnica: &r}}}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var rehidratado DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &rehidratado); err != nil {
		t.Fatal(err)
	}
	i := VerificarCadenaFronteraAdminTecnicaV1(rehidratado, d.Manifiesto, 2)
	if i.Estado != "verificada" || i.ActorPerfilContextoCotejados {
		t.Fatalf("frontera:%+v", i.Fallo)
	}
	d.Registros[0].Intento = &RegistroIntentoV2{}
	if VerificarCadenaFronteraAdminTecnicaV1(d, d.Manifiesto, 2).Estado != "rechazada" {
		t.Fatal("actor_prestado")
	}
	d.Registros[0].Intento = nil
	d.Esquema = EsquemaVerificacionMixta
	if VerificarCadenaMixtaV2(d, d.Manifiesto, 2).Estado != "rechazada" {
		t.Fatal("esquema_anterior_ampliado")
	}
}
func TestEsquemaFronteraConservaHistoriaPrevia(t *testing.T) {
	for _, d := range []DocumentoVerificacionMixta{vectorConsumosMixtosAD173(t), vectorGobiernoUsuariosPrueba(t), documentoFuentesPrueba(t), documentoUnidadInicialPrueba(t), documentoBootstrapIntentosPrueba(t), vectorMantenimientoFijo(t)} {
		b, err := json.Marshal(d.Registros)
		if err != nil {
			t.Fatal(err)
		}
		d.Esquema = EsquemaVerificacionFronteraAdminTecnicaV1
		i := VerificarCadenaFronteraAdminTecnicaV1(d, d.Manifiesto, uint64(len(d.Registros)))
		a, err := json.Marshal(d.Registros)
		if err != nil {
			t.Fatal(err)
		}
		if i.Estado != "verificada" || !bytes.Equal(a, b) {
			t.Fatalf("historia_previa:%+v", i.Fallo)
		}
	}
}
