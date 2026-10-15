package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func vectorPreperfilBolsaRRHHPrueba(t *testing.T) ([]byte, DocumentoVerificacionMixta) {
	t.Helper()
	b, err := os.ReadFile("testdata/preperfil_bolsa_ad287.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err = json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return b, d
}

func TestPreperfilBolsaRRHHCotejaMaterialEslabonYExportacion(t *testing.T) {
	b, d := vectorPreperfilBolsaRRHHPrueba(t)
	resultado := VerificarCadenaMixtaV3(d, d.Manifiesto, 2)
	if resultado.Estado != "verificada" || !resultado.MaterialEventoRecalculado {
		t.Fatalf("vector AD287 válido rechazado: %+v", resultado.Fallo)
	}
	c := domain.CoberturaCheckpoint{CadenaID: d.Manifiesto.CadenaID,
		PrimeraSecuencia: d.Manifiesto.PrimeraSecuencia, UltimaSecuencia: d.Manifiesto.UltimaSecuencia,
		AnteriorSHA256: d.Manifiesto.AnteriorSHA256, CabezaSHA256: d.Manifiesto.CabezaSHA256,
		Registros: d.Manifiesto.Registros}
	esquema, verificado, err := VerificarDocumentoExportacionAuditoria(b, c, int64(len(b)+1), 2)
	if err != nil || esquema != EsquemaVerificacionPreperfil || verificado.Estado != "verificada" {
		t.Fatalf("exportación AD287 no verificable: esquema=%q error=%v fallo=%+v", esquema, err, verificado.Fallo)
	}
	mutado := d
	mutado.Registros = append([]RegistroMixtoV2(nil), d.Registros...)
	registro := *d.Registros[0].PreperfilBolsaRRHH
	registro.FuenteSHA256 = strings.Repeat("f", 64)
	mutado.Registros[0].PreperfilBolsaRRHH = &registro
	rechazo := VerificarCadenaMixtaV3(mutado, d.Manifiesto, 2)
	if rechazo.Estado == "verificada" || rechazo.Fallo == nil || rechazo.Fallo.Codigo != "material_distinto" {
		t.Fatalf("fuente alterada conservó sello: %+v", rechazo.Fallo)
	}
	mutado.Registros[0].PreperfilBolsaRRHH = nil
	if VerificarCadenaMixtaV3(mutado, d.Manifiesto, 2).Estado == "verificada" {
		t.Fatal("tipo AD287 sin proyección aceptado")
	}
}
