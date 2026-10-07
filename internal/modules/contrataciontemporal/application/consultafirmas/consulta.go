package consultafirmas

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// CamposConsultaFirmasDocumento devuelve la proyección cerrada de CT145.
// No incluye identidad del firmante, certificado, actor, perfil ni texto libre.
func CamposConsultaFirmasDocumento() []string {
	return []string{"CatalogoHuella", "CatalogoRef", "ClaveIdempotencia", "ConMotivoDevolucion", "Documento",
		"DocumentoCustodiaRef", "DocumentoCustodiaVersion", "ExpedienteVersion", "FirmaRef", "FirmadoHuella",
		"OriginalHuella", "PasoOrden", "PasoRef", "ReciboRef", "RegistradaEn", "Resultado", "Secuencia", "SelloTiempoEstado"}
}

func CanonicoConsultaFirmasDocumento(m ports.MaterialConsultaFirmasDocumento) ([]byte, error) {
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) || !domain.ReferenciaOpacaValida(m.ExpedienteRef) {
		return nil, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	return json.Marshal(m)
}

func RecursoConsultaFirmasDocumento(m ports.MaterialConsultaFirmasDocumento) (vecdomain.RecursoAutorizable, error) {
	b, err := CanonicoConsultaFirmasDocumento(m)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	return vecdomain.RecursoAutorizable{
		Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoConsultaFirmasDocumento,
		Ambitos:   map[string]string{"organizacion_ref": m.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}

func ValidarCapacidadConsultaFirmasDocumento(c ports.CapacidadConsultaFirmasDocumento, m ports.MaterialConsultaFirmasDocumento) error {
	r, err := RecursoConsultaFirmasDocumento(m)
	h, errHuella := r.HuellaContextoAutorizacionSHA256()
	a := c.ExportarMaterialParaConsumidor()
	resumen := a.ResumenCapacidad()
	if err != nil || errHuella != nil || a.ValidarEstructura() != nil ||
		resumen.Operacion() != ports.AccionConsultarFirmasDocumento || resumen.EfectoRef() != m.ExpedienteRef ||
		resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != ports.AudienciaConsultaFirmasDocumentoV3 {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}

func RecursoConsultaFirmasR5(m ports.MaterialConsultaFirmasR5) (vecdomain.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	return vecdomain.RecursoAutorizable{
		Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoConsultaFirmasR5,
		Ambitos:   map[string]string{"organizacion_ref": m.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": h},
	}, nil
}

func ValidarCapacidadConsultaFirmasR5(c ports.CapacidadConsultaFirmasR5, m ports.MaterialConsultaFirmasR5) error {
	r, err := RecursoConsultaFirmasR5(m)
	if err != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	a := c.ExportarMaterialParaConsumidor()
	resumen := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil ||
		resumen.Operacion() != ports.AccionConsultarFirmasR5 || resumen.EfectoRef() != m.ExpedienteRef ||
		resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != ports.AudienciaConsultaFirmasR5V3 {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
