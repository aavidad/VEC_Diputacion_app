package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func vectorMantenimientoFijo(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, e := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/mantenimiento_fijo_ad183.json")
	if e != nil {
		t.Fatal(e)
	}
	var d DocumentoVerificacionMixta
	if e = json.Unmarshal(b, &d); e != nil {
		t.Fatal(e)
	}
	return d
}
func TestAD183VectoresYRechazos(t *testing.T) {
	d := vectorMantenimientoFijo(t)
	r := VerificarCadenaMantenimientoFijoV1(d, d.Manifiesto, 5)
	if r.Estado != "verificada" || !r.MaterialMantenimientoRecalculado || !r.MaterialIntentosMantenimientoRecalculado || r.ActorPerfilContextoCotejados || r.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("vector: %+v fallo=%+v", r, r.Fallo)
	}
	for _, caso := range []string{"rol", "asignacion", "fecha", "cruce", "tipo"} {
		t.Run(caso, func(t *testing.T) {
			d := vectorMantenimientoFijo(t)
			switch caso {
			case "rol":
				d.Registros[0].MantenimientoFijo.RolDestinoSHA256 = strings.Repeat("a", 64)
			case "asignacion":
				d.Registros[0].MantenimientoFijo.Asignacion2DestinoRef = "otra:asignacion"
			case "fecha":
				d.Registros[0].MantenimientoFijo.RegistradaEn = "dato_privado:no_fecha"
			case "cruce":
				d.Registros[0].Bootstrap = &RegistroBootstrapV3{}
			case "tipo":
				d.Esquema = EsquemaVerificacionBootstrapCentral
			}
			r := VerificarCadenaMantenimientoFijoV1(d, d.Manifiesto, 5)
			b, _ := json.Marshal(r)
			if r.Estado != "rechazada" || strings.Contains(string(b), "dato_privado") {
				t.Fatalf("rechazo: %s", b)
			}
		})
	}
}
func TestAD183ConservaFamiliasAnteriores(t *testing.T) {
	ds := []DocumentoVerificacionMixta{vectorConsumosMixtosAD173(t), documentoFuentesPrueba(t), documentoUnidadInicialPrueba(t), documentoBootstrapIntentosPrueba(t)}
	for _, d := range ds {
		before, _ := json.Marshal(d.Registros)
		d.Esquema = EsquemaVerificacionMantenimientoFijo
		r := VerificarCadenaMantenimientoFijoV1(d, d.Manifiesto, uint64(len(d.Registros)))
		after, _ := json.Marshal(d.Registros)
		if r.Estado != "verificada" || string(before) != string(after) {
			t.Fatalf("historia: %+v fallo=%+v", r, r.Fallo)
		}
	}
}
