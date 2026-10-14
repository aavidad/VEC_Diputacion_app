package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestConsultaFirmaV2TercerPasoConservaCadenaCompleta(t *testing.T) {
	for _, caso := range []string{"cadena", "salta_segundo", "reescribe_primera_evidencia", "cuarto_paso"} {
		t.Run(caso, func(t *testing.T) {
			m, w := fixtureConsultaFirmaV2()
			m.PasoOrden = 3
			custodiaInicial, versionInicial := "custodia:primera", uint64(1)
			w.Firmas[0].DocumentoCustodia, w.Firmas[0].VersionCustodia = &custodiaInicial, &versionInicial
			w.RevisionesPDF[0].firmaExternaSQL170 = w.Firmas[0]
			for orden := 2; orden <= 3; orden++ {
				previa := w.Firmas[orden-2]
				anterior := w.RevisionesPDF[orden-2]
				base, v := previa, anterior
				ref, recibo := "firma:segunda", "recibo:segunda"
				if orden == 3 {
					ref, recibo = "firma:tercera", "recibo:tercera"
				}
				base.FirmaRef, base.ReciboRef, base.Secuencia, base.PasoOrden = ref, recibo, orden, orden
				huella := strings.Repeat(string(rune('c'+orden)), 64)
				custodia := "custodia:" + ref
				version := uint64(orden)
				base.FirmadoHuella, base.DocumentoCustodia, base.VersionCustodia = &huella, &custodia, &version
				v.firmaExternaSQL170 = base
				v.FirmaAnteriorRef, v.ReciboAnteriorRef = &previa.FirmaRef, &previa.ReciboRef
				v.OrdenFirmaPDF = orden
				v.EntradaDocumentoRef, v.EntradaDocumentoVersion = *previa.DocumentoCustodia, *previa.VersionCustodia
				v.EntradaDocumentoHuella, v.EntradaDocumentoLongitud = *previa.FirmadoHuella, anterior.RevisionLongitud
				v.RevisionLongitud = anterior.RevisionLongitud + 100
				v.ByteRange = []uint64{0, anterior.RevisionLongitud + 20, anterior.RevisionLongitud + 80, 20}
				v.RevisionHuellaSHA256 = huella
				v.EvidenciaFirmasCanonica = strings.TrimSuffix(anterior.EvidenciaFirmasCanonica, "]") + `,{"orden":` + string(rune('0'+orden)) + `}]`
				h := sha256.Sum256([]byte(v.EvidenciaFirmasCanonica))
				v.EvidenciaFirmasHuellaSHA256 = hex.EncodeToString(h[:])
				w.Firmas = append(w.Firmas, base)
				w.RevisionesPDF = append(w.RevisionesPDF, v)
			}
			switch caso {
			case "salta_segundo":
				w.RevisionesPDF[2].FirmaAnteriorRef = &w.Firmas[0].FirmaRef
			case "reescribe_primera_evidencia":
				w.RevisionesPDF[2].EvidenciaFirmasCanonica = `[{"orden":9},{"orden":2},{"orden":3}]`
				h := sha256.Sum256([]byte(w.RevisionesPDF[2].EvidenciaFirmasCanonica))
				w.RevisionesPDF[2].EvidenciaFirmasHuellaSHA256 = hex.EncodeToString(h[:])
			case "cuarto_paso":
				w.RevisionesPDF[2].OrdenFirmaPDF = 4
			}
			lectura, err := proyectarLecturaFirmasSQL172(m, w)
			if caso == "cadena" {
				if err != nil || len(lectura.RevisionesPDF) != 3 || lectura.RevisionesPDF[2].FirmaAnteriorRef != w.Firmas[1].FirmaRef {
					t.Fatalf("la tercera revisión perdió su antecedente: %v", err)
				}
			} else if err == nil {
				t.Fatal("se admitió una cadena alterada")
			}
		})
	}
}
