package application

import (
	"strings"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Este plan es un doble de coordinación; no concede competencia nominal.
func descriptorMultiplePrueba(m ports.MaterialFirmaVerificadaV2) ports.DescriptorConstructorFirmaV2 {
	d := ports.DescriptorConstructorFirmaV2{Esquema: "vec.competencia-firmante.constructor-ct.v1", CertificadoDERSHA256: m.CertificadoHuella,
		Seleccion: ports.SeleccionConstructorFirmaV2{PerfilEsperadoRef: m.PerfilFirmanteRef, PerfilActivoRef: m.PerfilActivoFirmanteRef, RolID: m.RolIDFirmante, CargoRef: "cargo:prueba", EnlaceEjercicioRef: "enlace:prueba"},
		Recurso: vd.RecursoFirmaHistoricaV1{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef, ExpedienteRef: m.ExpedienteRef, DocumentoRef: m.OriginalRef, RecursoAutorizableRef: m.OriginalRef,
			ModuloID: ports.ModuloContratacion, TipoRecurso: "documento_contratacion_temporal", RecursoContextoSHA256: strings.Repeat("a", 64),
			Original: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.OriginalRef, Version: m.OriginalVersion, HuellaSHA256: m.OriginalHuella}, PDFRaizSHA256: m.OriginalHuella,
			Firmado: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.DocumentoCustodiaRef, Version: m.DocumentoCustodiaVersion, HuellaSHA256: m.FirmadoHuella}, PDFFirmadoSHA256: m.FirmadoHuella, NumeroFirmas: uint64(m.OrdenFirmaPDF)},
		Accion: "contratacion_temporal.documento.firmar", Finalidad: "formalizacion", Motivo: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_firma", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "firma"},
		Circuito: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.CatalogoRef, Version: m.CatalogoVersion, HuellaSHA256: m.CatalogoHuella}, PasoRef: m.PasoRef, PasoOrden: uint64(m.PasoOrden)}
	if m.OrdenFirmaPDF > 1 {
		d.Recurso.EntradaRevision = &vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.EntradaDocumentoRef, Version: m.EntradaDocumentoVersion, HuellaSHA256: m.EntradaDocumentoHuella}
	}
	return d
}
