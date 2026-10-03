package postgres

import (
	"context"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// FuenteDescriptorFirmaV2 selecciona el paso y cargo/enlace desde el plan
// gobernado de CT. No consulta SQL, transporta Acreditacion ni acepta un canon
// nominal autónomo. El fabricante AUT35 consulta las fuentes en la TX final.
type FuenteDescriptorFirmaV2 interface {
	DescriptorFirmaV2(context.Context, ports.MaterialFirmaVerificadaV2) (DescriptorConstructorFirmaV2, error)
}

type SeleccionConstructorFirmaV2 struct {
	PerfilEsperadoRef  string `json:"perfil_esperado_ref"`
	PerfilActivoRef    string `json:"perfil_activo_ref"`
	RolID              string `json:"rol_id"`
	CargoRef           string `json:"cargo_ref"`
	EnlaceEjercicioRef string `json:"enlace_ejercicio_ref"`
}

type DescriptorConstructorFirmaV2 struct {
	Esquema              string                              `json:"esquema"`
	CertificadoDERSHA256 string                              `json:"certificado_der_sha256"`
	Seleccion            SeleccionConstructorFirmaV2         `json:"seleccion"`
	Recurso              vd.RecursoFirmaHistoricaV1          `json:"recurso"`
	Accion               string                              `json:"accion"`
	Finalidad            string                              `json:"finalidad"`
	Motivo               vd.ReferenciaEntradaCatalogo        `json:"motivo"`
	Circuito             vd.ReferenciaHistoricaCompetenciaV1 `json:"circuito"`
	PasoRef              string                              `json:"paso_ref"`
	PasoOrden            uint64                              `json:"paso_orden"`
	// CT fija el instante de acreditación al primer registro. En replay lo
	// recupera del acto original; el plan nunca aporta la hora del navegador.
	FechaHistorica *time.Time `json:"fecha_historica"`
}

func (d DescriptorConstructorFirmaV2) canonicoPara(m ports.MaterialFirmaVerificadaV2) ([]byte, error) {
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
