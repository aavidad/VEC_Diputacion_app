package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
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
	// Recompone un asiento con encadenado histórico completo: sin la regla
	// AD287 sería verificable pese a carecer del eslabón obligatorio V5.
	historico := d
	historico.Registros = append([]RegistroMixtoV2(nil), d.Registros...)
	historico.Registros[0].Eslabon = nil
	previa := *d.Registros[0].PreperfilBolsaRRHH
	previa.AnteriorSHA256 = strings.Repeat("0", 64)
	h := sha256.Sum256(huellaEncuadradaIntento(
		"vec.auditoria.eslabon.preperfil-bolsa-rrhh.v1", "1", previa.AnteriorSHA256,
		previa.AuditoriaRef, previa.EventoMaterialSHA256, previa.RegistradaEn))
	previa.HuellaSHA256 = hex.EncodeToString(h[:])
	historico.Registros[0].PreperfilBolsaRRHH = &previa
	historico.Manifiesto.CabezaSHA256 = previa.HuellaSHA256
	sinEslabon := VerificarCadenaMixtaV3(historico, historico.Manifiesto, 2)
	if sinEslabon.Fallo == nil || sinEslabon.Fallo.Codigo != "eslabon_ausente" || sinEslabon.Fallo.Clave != "eslabon" {
		t.Fatalf("AD287 sin eslabón no tuvo rechazo específico: %+v", sinEslabon.Fallo)
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
