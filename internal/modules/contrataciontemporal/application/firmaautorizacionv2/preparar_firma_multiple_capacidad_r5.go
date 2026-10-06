package firmaautorizacionv2

import (
	"crypto/sha256"
	"encoding/hex"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// RecursoFirmaVerificadaV2 es el recurso de quien actúa con una asignación
// sólo de organización.
func RecursoFirmaVerificadaV2(m ports.MaterialFirmaVerificadaV2, descriptor []byte) (vecdomain.RecursoAutorizable, error) {
	return RecursoFirmaVerificadaV2ConAmbitos(m, descriptor, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef})
}

// RecursoFirmaVerificadaV2ConAmbitos lleva exactamente los ámbitos de la
// asignación de quien actúa. La organización es la del material y, en la vía
// VEC, la unidad (si la asignación la tiene) es la del paso del plan. La vía
// externa sigue sólo con organización hasta su propio corte.
func RecursoFirmaVerificadaV2ConAmbitos(m ports.MaterialFirmaVerificadaV2, descriptor []byte, a ports.AmbitosOperadorFirmaV2) (vecdomain.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil || ValidarDescriptorFirmaVerificadaV2(m, descriptor) != nil || a.OrganizacionRef != m.OrganizacionRef ||
		(a.UnidadRef != "" && (m.Via != ports.ViaFirmaCertificadoVEC || a.UnidadRef != m.UnidadFirmanteRef)) {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	tipo := ports.TipoRecursoFirmaExterna
	if m.Via == ports.ViaFirmaCertificadoVEC {
		tipo = ports.TipoRecursoFirmaVec
	}
	huellaDescriptor := sha256.Sum256(descriptor)
	return vecdomain.RecursoAutorizable{Referencia: m.RecursoRef(), ModuloID: ports.ModuloContratacion, Tipo: tipo, Ambitos: a.Mapa(), Atributos: map[string]string{"material_sha256": h, "descriptor_firma_sha256": hex.EncodeToString(huellaDescriptor[:])}}, nil
}

func ValidarCapacidadFirmaVerificadaV2(c ports.CapacidadFirmaVerificadaV2, m ports.MaterialFirmaVerificadaV2) error {
	descriptor, huella := c.ExportarDescriptorParaConsumidor()
	defer clear(descriptor)
	calculada := sha256.Sum256(descriptor)
	if huella != hex.EncodeToString(calculada[:]) {
		return ports.ErrFirmaDocumentoDenegada
	}
	a := c.ExportarMaterialParaConsumidor()
	resumen := a.ResumenCapacidad()
	accion, audiencia := ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV2
	if m.Via == ports.ViaFirmaCertificadoVEC {
		accion, audiencia = ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV2
	}
	if a.ValidarEstructura() != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia {
		return ports.ErrFirmaDocumentoDenegada
	}
	// Quien registra no conoce la asignación del firmante: sólo admite los dos
	// recursos que el material permite. AD206 fija el exacto en el consumo.
	for _, ambitos := range ambitosPosiblesFirmaVerificadaV2(m) {
		r, err := RecursoFirmaVerificadaV2ConAmbitos(m, descriptor, ambitos)
		if err != nil {
			return ports.ErrFirmaDocumentoDenegada
		}
		h, err := r.HuellaContextoAutorizacionSHA256()
		if err == nil && resumen.EfectoRef() == r.Referencia && resumen.EfectoHuellaSHA256() == h {
			return nil
		}
	}
	return ports.ErrFirmaDocumentoDenegada
}

func ambitosPosiblesFirmaVerificadaV2(m ports.MaterialFirmaVerificadaV2) []ports.AmbitosOperadorFirmaV2 {
	posibles := []ports.AmbitosOperadorFirmaV2{{OrganizacionRef: m.OrganizacionRef}}
	if m.Via == ports.ViaFirmaCertificadoVEC {
		posibles = append(posibles, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef})
	}
	return posibles
}

func ValidarCapacidadConsultaFirmasR5V2(c ports.CapacidadConsultaFirmasR5V2, m ports.MaterialConsultaFirmasR5V2) error {
	r, err := RecursoConsultaFirmasR5V2(m)
	if err != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	a := c.ExportarMaterialParaConsumidor()
	resumen := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || resumen.Operacion() != ports.AccionConsultarFirmasR5V2 || resumen.EfectoRef() != r.Referencia || resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != ports.AudienciaConsultaFirmasR5V2 {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}

func RecursoConsultaFirmasR5V2(m ports.MaterialConsultaFirmasR5V2) (vecdomain.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	// Los ámbitos de la asignación de quien consulta: organización y, si el
	// material trae UnidadRef, esa unidad (AD210 la relee en el consumo).
	ambitos := ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadRef}.Mapa()
	return vecdomain.RecursoAutorizable{Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoConsultaFirmasR5, Ambitos: ambitos, Atributos: map[string]string{"material_sha256": h}}, nil
}
