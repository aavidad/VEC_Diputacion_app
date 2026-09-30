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
