package ports

import (
	"strconv"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func artefactoFirmaValidoPrueba(t *testing.T, contenido dominiobolsa.ContenidoDecisionTecnica, politica PoliticaFirmaBaremacion, sufijo string) ArtefactoFirma {
	t.Helper()
	huella, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return ArtefactoFirma{
		ProcesoRef: contenido.ProcesoRef, SolicitudRef: contenido.SolicitudRef, SujetoRef: contenido.SujetoRef,
		BaremacionMeritoRef: contenido.BaremacionMeritoRef, DecisionRef: contenido.ID,
		VersionBaremacion: contenido.VersionBaremacion, SesionFirmaRef: "sesion-firma-1",
		EvidenciaFirmaInteractivaRef:     "evidencia-firma-interactiva-" + sufijo,
		HuellaEvidenciaInteractivaSHA256: huellaPruebaPuertos("7"),
		DocumentoFirmable:                puertosvec.ReferenciaObjetoAlmacen{Referencia: "objeto-firmable-001", Version: "version-001"},
		HuellaDocumentoFirmableSHA256:    huellaPruebaPuertos("5"), EvidenciaCustodiaRef: "evidencia-custodia-001",
		FirmaRef:          "firma-decision-001",
		HuellaFirmaSHA256: huellaPruebaPuertos("8"), DocumentoFirmadoRef: "documento-firmado-" + sufijo,
		HuellaDocumentoSHA256: huellaPruebaPuertos(map[bool]string{true: "9", false: "a"}[sufijo == "inicial"]),
		HuellaContenidoSHA256: huella, PoliticaFirmaRef: politica.Referencia, PoliticaFirmaVersion: politica.Version,
		HuellaPoliticaFirmaSHA256: politica.HuellaSHA256, FirmanteRef: principalBaremacionPuertosPrueba,
		PerfilFirmanteClave: perfilBaremacionPuertosPrueba, FirmadaEn: instantePuertosPrueba.Add(5 * time.Minute),
	}
}

func validacionFirmaValidaPrueba(
	artefacto ArtefactoFirma,
	instante time.Time,
	sufijo string,
	perfilesFirma ...string,
) ValidacionFirmaServidor {
	perfilFirma := PerfilFirmaPAdESBaselineB
	if len(perfilesFirma) == 1 {
		perfilFirma = perfilesFirma[0]
	}
	comprobaciones := make([]ComprobacionFirma, 0, len(comprobacionesFirmaObligatorias))
	for indice, clave := range ComprobacionesFirmaObligatorias() {
		comprobaciones = append(comprobaciones, ComprobacionFirma{
			Clave: clave, Estado: EstadoComprobacionSuperada,
			EvidenciaRef:          "evidencia-validacion-" + sufijo + "-" + strconv.Itoa(indice),
			HuellaEvidenciaSHA256: huellaPruebaPuertos(strconv.FormatInt(int64(10+indice%6), 16)),
		})
	}
	return ValidacionFirmaServidor{
		Estado: EstadoValidacionFirmaValida, Artefacto: artefacto, ValidacionRef: "validacion-firma-" + sufijo,
		HuellaValidacionSHA256: huellaPruebaPuertos("b"), FirmanteVerificadoRef: artefacto.FirmanteRef,
		PerfilVerificadoClave: artefacto.PerfilFirmanteClave, PerfilFirmaVerificadoClave: perfilFirma,
		Comprobaciones: comprobaciones, ValidadaEn: instante,
	}
}

func documentoFirmadoCustodiadoPrueba(artefacto ArtefactoFirma) DocumentoFirmadoCustodiado {
	objetoRef := puertosvec.ReferenciaObjetoAlmacen{Referencia: "objeto-firmado-1", Version: "version-firmada-1"}
	retenidoHasta := instantePuertosPrueba.Add(365 * 24 * time.Hour)
	return DocumentoFirmadoCustodiado{
		DocumentoFirmadoRef: artefacto.DocumentoFirmadoRef, FirmaRef: artefacto.FirmaRef,
		HuellaDocumentoSHA256: artefacto.HuellaDocumentoSHA256,
		Objeto: puertosvec.ObjetoAlmacenado{
			Objeto: objetoRef, ConectorID: "almacen-prueba", Zona: puertosvec.ZonaAlmacenAdmitida,
			MIME: "application/pdf", Tamano: 1024, HuellaSHA256: artefacto.HuellaDocumentoSHA256,
			EvidenciaCreacionRef: "evidencia-escritura-firmado-1", AlmacenadoEn: instantePuertosPrueba.Add(9 * time.Minute),
			RetenidoHasta: retenidoHasta,
		},
		EvidenciaEscritura:                puertosvec.EvidenciaOperacionAlmacen{Referencia: "evidencia-escritura-firmado-1"},
		EvidenciaRetencion:                puertosvec.EvidenciaOperacionAlmacen{Referencia: "evidencia-retencion-firmado-1"},
		EvidenciaRecuperacionRef:          "evidencia-recuperacion-firmado-1",
		HuellaEvidenciaRecuperacionSHA256: huellaPruebaPuertos("e"),
		PoliticaRetencionRef:              "politica-retencion-firmado-v1", RetenidoHasta: retenidoHasta,
	}
}
