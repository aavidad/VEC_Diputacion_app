package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/vec/domain"
)

// Vector producido por el director en PostgreSQL 18 real POST186+AD187.
func TestPreservacionVectorPostgreSQLReal(t *testing.T) {
	raw, err := os.ReadFile("testdata/preservacion_ad187.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := decodificarDocumentoExportacionEstricto(raw, &d); err != nil {
		t.Fatal(err)
	}
	r := VerificarCadenaPreservacionAuditoriaV1(d, d.Manifiesto, 6)
	if r.Estado != "verificada" || !r.MaterialPreservacionRecalculado || r.AutenticidadCheckpoint != "no_comprobada" {
		t.Fatalf("vectorPG: %+v fallo=%+v", r, r.Fallo)
	}
	c := d.Manifiesto
	_, informe, err := VerificarDocumentoExportacionAuditoria(raw, domain.CoberturaCheckpoint{CadenaID: c.CadenaID, PrimeraSecuencia: c.PrimeraSecuencia, UltimaSecuencia: c.UltimaSecuencia, AnteriorSHA256: c.AnteriorSHA256, CabezaSHA256: c.CabezaSHA256, Registros: c.Registros}, 16384, 6)
	if err != nil || informe.Estado != "verificada" {
		t.Fatalf("parser: %v %+v", err, informe)
	}
	for _, caso := range []string{"operador", "detalle", "fecha", "cruce"} {
		t.Run(caso, func(t *testing.T) {
			var d DocumentoVerificacionMixta
			if json.Unmarshal(raw, &d) != nil {
				t.Fatal("fixture")
			}
			p := d.Registros[0].Preservacion
			switch caso {
			case "operador":
				p.OperadorLogin = "otro_login"
			case "detalle":
				p.DetalleCanonicoBase64 = "AA=="
			case "fecha":
				p.RegistradaEn = "dato_privado"
			case "cruce":
				d.Registros[0].Periodica = &RegistroOperacionPeriodicaV1{}
			}
			r := VerificarCadenaPreservacionAuditoriaV1(d, d.Manifiesto, 6)
			out, _ := json.Marshal(r)
			if r.Estado != "rechazada" || strings.Contains(string(out), "dato_privado") {
				t.Fatal("alteración aceptada o expuesta")
			}
		})
	}
}

func TestPreservacionConservaFamiliasPrevias(t *testing.T) {
	for _, d := range []DocumentoVerificacionMixta{vectorConsumosMixtosAD173(t), documentoFuentesPrueba(t), documentoUnidadInicialPrueba(t), documentoBootstrapIntentosPrueba(t), vectorMantenimientoFijo(t), documentoPeriodicaPrueba()} {
		antes, _ := json.Marshal(d.Registros)
		d.Esquema = EsquemaVerificacionPreservacionAuditoria
		r := VerificarCadenaPreservacionAuditoriaV1(d, d.Manifiesto, uint64(len(d.Registros)))
		despues, _ := json.Marshal(d.Registros)
		if r.Estado != "verificada" || r.MaterialPreservacionRecalculado || string(antes) != string(despues) {
			t.Fatalf("historia: %+v", r)
		}
	}
}
