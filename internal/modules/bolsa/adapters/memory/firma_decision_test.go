package memory

import (
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func firmaMemoriaPrueba(
	contenido dominiobolsa.ContenidoDecisionTecnica,
	huella string,
	firmadaEn time.Time,
) dominiobolsa.FirmaDecisionTecnica {
	return dominiobolsa.FirmaDecisionTecnica{
		FirmanteRef: contenido.DecisorRef, PerfilFirmanteClave: contenido.PerfilDecisorClave,
		PoliticaFirmaRef: "politica-firma-baremacion", PoliticaFirmaVersion: 4,
		HuellaPoliticaFirmaSHA256: huellaMemoria("1"), PerfilFirmaAlcanzadoClave: "pades_baseline_lta",
		RequiereFirmaInteractiva:   true,
		RequiereValidacionServidor: true, RequiereSelloTiempo: true, RequiereAumentoLongevidad: true,
		SesionFirmaInteractivaRef:             "sesion-firma-decision-001",
		HuellaEvidenciaFirmaInteractivaSHA256: huellaMemoria("2"),
		DocumentoFirmableRef:                  "objeto-firmable-decision-001", VersionDocumentoFirmable: "version-1",
		HuellaDocumentoFirmableSHA256: huellaMemoria("8"), EvidenciaCustodiaRef: "custodia-decision-001",
		FirmaRef: "firma-decision-001", HuellaFirmaSHA256: huellaMemoria("f"),
		DocumentoFirmadoRef: "documento-firmado-decision-001", HuellaDocumentoSHA256: huellaMemoria("d"),
		DocumentoFirmadoCustodiadoRef: "objeto-firmado-decision-001", VersionDocumentoFirmadoCustodiado: "version-firmada-1",
		EvidenciaRecuperacionFirmadoRef:       "evidencia-recuperacion-firmado-decision-001",
		HuellaEvidenciaRecuperacionSHA256:     huellaMemoria("b"),
		EvidenciaCustodiaDocumentoFirmadoRef:  "evidencia-custodia-firmado-decision-001",
		EvidenciaRetencionDocumentoFirmadoRef: "evidencia-retencion-firmado-decision-001",
		PoliticaRetencionDocumentoFirmadoRef:  "politica-retencion-firmado-v1",
		DocumentoFirmadoRetenidoHasta:         firmadaEn.Add(365 * 24 * time.Hour),
		ManifiestoProbatorioRef:               "manifiesto-probatorio-decision-001",
		HuellaManifiestoProbatorioSHA256:      huellaMemoria("a"),
		SelloManifiestoProbatorioHMACSHA256:   "hmac-sha256:manifiesto_1:" + huellaMemoria("c"),
		HuellaContenidoSHA256:                 huella, ValidacionInicialFirmaRef: "validacion-inicial-decision-001",
		HuellaValidacionInicialSHA256: huellaMemoria("9"), ValidadaInicialEn: firmadaEn.Add(30 * time.Second),
		ValidacionFirmaRef: "validacion-final-decision-001", HuellaValidacionSHA256: huellaMemoria("e"),
		ValidadaEn: firmadaEn.Add(4 * time.Minute), SelloTiempoRef: "sello-tiempo-decision-001",
		HuellaSelloTiempoSHA256: huellaMemoria("c"), PoliticaSelloTiempoRef: "politica-sello-tsa",
		PoliticaSelloTiempoVersion: 2, HuellaPoliticaSelloTiempoSHA256: huellaMemoria("3"),
		ValidacionSelloTiempoRef:          "validacion-sello-decision-001",
		HuellaValidacionSelloTiempoSHA256: huellaMemoria("4"), SelladaEn: firmadaEn.Add(time.Minute),
		ValidacionDocumentoSelladoRef:          "validacion-documento-sellado-decision-001",
		HuellaValidacionDocumentoSelladoSHA256: huellaMemoria("a"),
		ValidadoDocumentoSelladoEn:             firmadaEn.Add(2 * time.Minute),
		NivelLongevidadClave:                   "pades_lta",
		AumentoLongevidadRef:                   "aumento-longevidad-decision-001",
		HuellaAumentoLongevidadSHA256:          huellaMemoria("5"),
		PoliticaLongevidadRef:                  "politica-longevidad-lta",
		PoliticaLongevidadVersion:              3,
		HuellaPoliticaLongevidadSHA256:         huellaMemoria("6"),
		ValidacionLongevidadRef:                "validacion-longevidad-decision-001",
		HuellaValidacionLongevidadSHA256:       huellaMemoria("7"),
		AumentadaEn:                            firmadaEn.Add(3 * time.Minute),
		FirmadaEn:                              firmadaEn,
	}
}
