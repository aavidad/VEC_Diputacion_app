package application

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func RecursoFirmaVerificadaV2(m ports.MaterialFirmaVerificadaV2) (vecdomain.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	tipo := ports.TipoRecursoFirmaExterna
	if m.Via == ports.ViaFirmaCertificadoVEC {
		tipo = ports.TipoRecursoFirmaVec
	}
	return vecdomain.RecursoAutorizable{Referencia: m.RecursoRef(), ModuloID: ports.ModuloContratacion, Tipo: tipo, Ambitos: map[string]string{"organizacion_ref": m.OrganizacionRef}, Atributos: map[string]string{"material_sha256": h}}, nil
}

func ValidarCapacidadFirmaVerificadaV2(c ports.CapacidadFirmaVerificadaV2, m ports.MaterialFirmaVerificadaV2) error {
	r, err := RecursoFirmaVerificadaV2(m)
	if err != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	a := c.ExportarMaterialParaConsumidor()
	resumen := a.ResumenCapacidad()
	accion, audiencia := ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV2
	if m.Via == ports.ViaFirmaCertificadoVEC {
		accion, audiencia = ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV2
	}
	if err != nil || a.ValidarEstructura() != nil || resumen.Operacion() != accion || resumen.EfectoRef() != r.Referencia || resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != audiencia {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
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
	ambitos := map[string]string{"organizacion_ref": m.OrganizacionRef}
	return vecdomain.RecursoAutorizable{Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoConsultaFirmasR5, Ambitos: ambitos, Atributos: map[string]string{"material_sha256": h}}, nil
}
