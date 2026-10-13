package firmavec

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// La vía VEC exige organización y unidad del firmante en la asignación real.
// Nunca admite el perfil fijo de RRHH de la vía externa ni otra acción.
func SolicitudAutorizacionFirmaVecV2CTValida(datos core.DatosSolicitudAutorizacionLigadaV3,
	motivo core.ReferenciaEntradaCatalogo) bool {
	r := datos.Recurso
	if !core.ReferenciaMotivoAutorizacionV2Valida(motivo) || datos.ReferenciaMotivo != motivo ||
		datos.Finalidad != ports.FinalidadFirmaDocumento ||
		r.Validar() != nil || r.ModuloID != ports.ModuloContratacion || len(r.Ambitos) != 2 ||
		r.Ambitos["organizacion_ref"] == "" || r.Ambitos["unidad_ref"] == "" ||
		!domain.HuellaSHA256FirmaValida(r.Atributos["material_sha256"]) {
		return false
	}
	switch datos.Accion {
	case ports.AccionRegistrarFirmaVec:
		clave := r.Referencia
		if len(clave) <= len(ports.PrefijoRecursoFirmaVec) || clave[:len(ports.PrefijoRecursoFirmaVec)] != ports.PrefijoRecursoFirmaVec {
			return false
		}
		descriptor := domain.HuellaSHA256FirmaValida(r.Atributos["descriptor_firma_sha256"])
		plan := domain.HuellaSHA256FirmaValida(r.Atributos["plan_firma_sha256"])
		return r.Tipo == ports.TipoRecursoFirmaVec && ports.ClaveIdempotenciaFirmaValida(clave[len(ports.PrefijoRecursoFirmaVec):]) &&
			len(r.Atributos) == 2 && descriptor != plan
	case ports.AccionConsultarFirmasR5V2:
		return r.Tipo == ports.TipoRecursoConsultaFirmasR5 && domain.ReferenciaOpacaValida(r.Referencia) && len(r.Atributos) == 1
	}
	return false
}
