package firmaautorizacionv2

import (
	"bytes"
	"encoding/json"
	"io"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func CanonicoDescriptorFirmaVerificadaV2(m ports.MaterialFirmaVerificadaV2, d ports.DescriptorConstructorFirmaV2) ([]byte, error) {
	orden, paso := m.OrdenFirmaPDF, m.PasoOrden
	if orden < 1 || orden > 2 || paso < 1 || paso > 2 {
		return nil, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	r, s := d.Recurso, d.Seleccion
	if d.Esquema != "vec.competencia-firmante.constructor-ct.v1" || d.CertificadoDERSHA256 != m.CertificadoHuella || d.FechaHistorica != nil ||
		s.PerfilEsperadoRef != m.PerfilFirmanteRef || s.PerfilActivoRef != m.PerfilActivoFirmanteRef || s.RolID != m.RolIDFirmante ||
		!domain.ReferenciaOpacaValida(s.CargoRef) || !domain.ReferenciaOpacaValida(s.EnlaceEjercicioRef) ||
		r.OrganizacionRef != m.OrganizacionRef || r.UnidadRef != m.UnidadFirmanteRef || r.ExpedienteRef != m.ExpedienteRef ||
		r.DocumentoRef != m.OriginalRef || r.RecursoAutorizableRef != r.DocumentoRef || r.ModuloID != ports.ModuloContratacion ||
		!domain.ReferenciaOpacaValida(r.TipoRecurso) || !domain.HuellaSHA256FirmaValida(r.RecursoContextoSHA256) ||
		r.Original.Referencia != m.OriginalRef || r.Original.Version != m.OriginalVersion || r.Original.HuellaSHA256 != m.OriginalHuella || r.PDFRaizSHA256 != m.OriginalHuella ||
		r.Firmado.Referencia != m.DocumentoCustodiaRef || r.Firmado.Version != m.DocumentoCustodiaVersion || r.Firmado.HuellaSHA256 != m.FirmadoHuella || r.PDFFirmadoSHA256 != m.FirmadoHuella ||
		r.NumeroFirmas != uint64(orden) || d.Circuito.Referencia != m.CatalogoRef || d.Circuito.Version != m.CatalogoVersion || d.Circuito.HuellaSHA256 != m.CatalogoHuella ||
		d.PasoRef != m.PasoRef || d.PasoOrden != uint64(paso) || !domain.ReferenciaOpacaValida(d.Accion) || !domain.ReferenciaOpacaValida(d.Finalidad) || d.Motivo.Validar() != nil {
		return nil, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	if (m.OrdenFirmaPDF == 1 && r.EntradaRevision != nil) || (m.OrdenFirmaPDF == 2 && (r.EntradaRevision == nil ||
		r.EntradaRevision.Referencia != m.EntradaDocumentoRef || r.EntradaRevision.Version != m.EntradaDocumentoVersion || r.EntradaRevision.HuellaSHA256 != m.EntradaDocumentoHuella)) {
		return nil, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	return json.Marshal(d)
}

// ValidarDescriptorFirmaVerificadaV2 exige el mismo JSON canónico que quedó
// ligado al PDP; rechaza claves repetidas, desconocidas y otra representación.
func ValidarDescriptorFirmaVerificadaV2(m ports.MaterialFirmaVerificadaV2, datos []byte) error {
	if len(datos) < 2 || len(datos) > 32768 {
		return ports.ErrFirmaDocumentoDenegada
	}
	var d ports.DescriptorConstructorFirmaV2
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if dec.Decode(&d) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return ports.ErrFirmaDocumentoDenegada
	}
	canon, err := CanonicoDescriptorFirmaVerificadaV2(m, d)
	if err != nil || !bytes.Equal(canon, datos) {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
