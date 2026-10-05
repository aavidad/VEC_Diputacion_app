package auditoria

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func documentoContextoPreV2Prueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("testdata/contexto_admin_prev2.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// Las huellas del vector fueron encuadradas fuera de Go con Python UTF8/SHA256.
func TestContextoPreV2CadenaYBundleParcialAcreditado(t *testing.T) {
	d := documentoContextoPreV2Prueba(t)
	r := VerificarCadenaContextoAdminPreV2(d, d.Manifiesto, 3)
	if r.Estado != "verificada" || r.ActorPerfilContextoCotejados {
		t.Fatalf("vector:%+v", r.Fallo)
	}
	for _, caso := range []string{"perfil_sin_actor", "actor_sin_fuente", "success_parcial", "tipo_cruzado", "evento_colision", "accion_renombrada", "motivo_abierto", "V2_fingido"} {
		t.Run(caso, func(t *testing.T) {
			d := documentoContextoPreV2Prueba(t)
			switch caso {
			case "perfil_sin_actor":
				d.Registros[0].ContextoAdminPreV2.ActorRef = nil
			case "actor_sin_fuente":
				d.Registros[1].ContextoAdminPreV2.FuenteRef = nil
				d.Registros[1].ContextoAdminPreV2.FuenteSHA256 = nil
			case "success_parcial":
				d.Registros[0].ContextoAdminPreV2.PerfilActivoRef = nil
			case "tipo_cruzado":
				d.Registros[0].TipoRegistro = "frontera_admin_tecnica"
			case "evento_colision":
				d.Registros[1].ContextoAdminPreV2.EventoRef = d.Registros[0].ContextoAdminPreV2.EventoRef
			case "accion_renombrada":
				d.Registros[0].ContextoAdminPreV2.Accion = "reconciliar_contexto_admin"
			case "motivo_abierto":
				d.Registros[0].ContextoAdminPreV2.MotivoRef = "SECRET_motivo"
			case "V2_fingido":
				d.Registros[0].Intento = &RegistroIntentoV2{}
			}
			r := VerificarCadenaContextoAdminPreV2(d, d.Manifiesto, 3)
			b, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			if r.Estado != "rechazada" || bytes.Contains(b, []byte("SECRET")) {
				t.Fatal("alteracion_aceptada_o_publicada")
			}
		})
	}
}
func TestEsquemaContextoPreV2ConservaFamiliasAnteriores(t *testing.T) {
	for _, d := range []DocumentoVerificacionMixta{vectorConsumosMixtosAD173(t), vectorGobiernoUsuariosPrueba(t), documentoFuentesPrueba(t), documentoUnidadInicialPrueba(t), documentoBootstrapIntentosPrueba(t), vectorMantenimientoFijo(t)} {
		d.Esquema = EsquemaVerificacionContextoAdminPreV2
		r := VerificarCadenaContextoAdminPreV2(d, d.Manifiesto, uint64(len(d.Registros)))
		if r.Estado != "verificada" {
			t.Fatalf("historia:%+v", r.Fallo)
		}
	}
}
