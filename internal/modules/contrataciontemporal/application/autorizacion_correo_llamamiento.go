package application

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionDespacharCorreoLlamamiento          = "contratacion_temporal.llamamiento.correo.despachar"
	AccionRegistrarResultadoCorreoLlamamiento = "contratacion_temporal.llamamiento.correo.registrar_resultado"
	AudienciaDespachoCorreoLlamamientoV3      = "vec_contratacion_temporal.despacho_correo_llamamiento.v1"
	TipoRecursoDespachoCorreoLlamamiento      = "despacho_correo_llamamiento_contratacion_temporal"
	AudienciaResultadoCorreoLlamamientoV3     = "vec_contratacion_temporal.resultado_correo_llamamiento.v1"
	TipoRecursoResultadoCorreoLlamamiento     = "resultado_correo_llamamiento_contratacion_temporal"
	FinalidadResultadoCorreoLlamamiento       = "gestionar_contratacion_temporal"
)

func RecursoDespachoCorreoLlamamiento(s ports.SolicitudDespacharCorreoLlamamiento) (vecdomain.RecursoAutorizable, error) {
	if s.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudCorreoLlamamientoInvalida
	}
	huella, err := s.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudCorreoLlamamientoInvalida
	}
	return vecdomain.RecursoAutorizable{Referencia: s.IntencionEnvioRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoDespachoCorreoLlamamiento, Ambitos: map[string]string{"organizacion_ref": s.OrganizacionRef}, Atributos: map[string]string{"material_sha256": huella}}, nil
}

func NuevaCapacidadDespachoCorreoLlamamiento(s ports.SolicitudDespacharCorreoLlamamiento, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) (ports.CapacidadDespachoCorreoLlamamiento, error) {
	if validarMaterialDespachoCorreoLlamamiento(s, material, ahora) != nil {
		return ports.CapacidadDespachoCorreoLlamamiento{}, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	return ports.TransportarMaterialDespachoCorreoLlamamiento(material), nil
}

func ValidarCapacidadDespachoCorreoLlamamiento(c ports.CapacidadDespachoCorreoLlamamiento, s ports.SolicitudDespacharCorreoLlamamiento, ahora time.Time) error {
	return validarMaterialDespachoCorreoLlamamiento(s, c.ExportarMaterialParaConsumidor(), ahora)
}

func validarMaterialDespachoCorreoLlamamiento(s ports.SolicitudDespacharCorreoLlamamiento, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) error {
	recurso, err := RecursoDespachoCorreoLlamamiento(s)
	if err != nil || material.ValidarEstructura() != nil {
		return ports.ErrDespachoCorreoLlamamientoDenegado
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if err != nil || resumen.Operacion() != AccionDespacharCorreoLlamamiento || resumen.EfectoRef() != s.IntencionEnvioRef || resumen.EfectoHuellaSHA256() != huella || resumen.AudienciaConsumo() != AudienciaDespachoCorreoLlamamientoV3 || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ports.ErrDespachoCorreoLlamamientoDenegado
	}
	return nil
}

func RecursoResultadoCorreoLlamamiento(s ports.SolicitudRegistrarResultadoCorreoLlamamiento, auditoria ports.AuditoriaResultadoCorreoLlamamiento) (vecdomain.RecursoAutorizable, error) {
	huella, err := s.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrResultadoCorreoLlamamientoNoConfiable
	}
	huellaAuditoria, err := auditoria.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrResultadoCorreoLlamamientoNoConfiable
	}
	return vecdomain.RecursoAutorizable{Referencia: s.IntentoRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoResultadoCorreoLlamamiento, Ambitos: map[string]string{"organizacion_ref": s.OrganizacionRef}, Atributos: map[string]string{"auditoria_sha256": huellaAuditoria, "material_sha256": huella}}, nil
}

func NuevaCapacidadResultadoCorreoLlamamiento(s ports.SolicitudRegistrarResultadoCorreoLlamamiento, auditoria ports.AuditoriaResultadoCorreoLlamamiento, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) (ports.CapacidadResultadoCorreoLlamamiento, error) {
	if validarMaterialResultadoCorreoLlamamiento(s, auditoria, material, ahora) != nil {
		return ports.CapacidadResultadoCorreoLlamamiento{}, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	return ports.TransportarMaterialResultadoCorreoLlamamiento(material), nil
}

func ValidarCapacidadResultadoCorreoLlamamiento(c ports.CapacidadResultadoCorreoLlamamiento, s ports.SolicitudRegistrarResultadoCorreoLlamamiento, auditoria ports.AuditoriaResultadoCorreoLlamamiento, ahora time.Time) error {
	return validarMaterialResultadoCorreoLlamamiento(s, auditoria, c.ExportarMaterialParaConsumidor(), ahora)
}

func validarMaterialResultadoCorreoLlamamiento(s ports.SolicitudRegistrarResultadoCorreoLlamamiento, auditoria ports.AuditoriaResultadoCorreoLlamamiento, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) error {
	recurso, err := RecursoResultadoCorreoLlamamiento(s, auditoria)
	if err != nil || material.ValidarEstructura() != nil {
		return ports.ErrDespachoCorreoLlamamientoDenegado
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if err != nil || resumen.Operacion() != AccionRegistrarResultadoCorreoLlamamiento || resumen.EfectoRef() != s.IntentoRef || resumen.EfectoHuellaSHA256() != huella || resumen.AudienciaConsumo() != AudienciaResultadoCorreoLlamamientoV3 || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ports.ErrDespachoCorreoLlamamientoDenegado
	}
	return nil
}
