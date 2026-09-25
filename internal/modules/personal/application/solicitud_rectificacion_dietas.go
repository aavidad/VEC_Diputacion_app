package application

import (
	"context"
	"regexp"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type ServicioRectificacionDietas struct {
	proveedor           personalports.ProveedorAutorizacionRectificacionDietas
	proveedorCorreccion personalports.ProveedorAutorizacionAsignacionDietas
	repositorio         personalports.RepositorioRectificacionDietas
}

func NuevoServicioRectificacionDietas(p personalports.ProveedorAutorizacionRectificacionDietas, correccion personalports.ProveedorAutorizacionAsignacionDietas, r personalports.RepositorioRectificacionDietas) (*ServicioRectificacionDietas, error) {
	if nuloAsignacion(p) || nuloAsignacion(r) {
		return nil, personalports.ErrRectificacionDietasNoDisponible
	}
	return &ServicioRectificacionDietas{proveedor: p, proveedorCorreccion: correccion, repositorio: r}, nil
}

func (s *ServicioRectificacionDietas) Ejecutar(ctx context.Context, solicitud personaldomain.SolicitudRectificacionDietas) (personalports.ResultadoRectificacionDietas, error) {
	var vacio personalports.ResultadoRectificacionDietas
	if s == nil || ctx == nil || nuloAsignacion(s.proveedor) || nuloAsignacion(s.repositorio) {
		return vacio, personalports.ErrRectificacionDietasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := personaldomain.NuevoMaterialRectificacionDietas(solicitud)
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasInvalida
	}
	solicitud = material.Solicitud()
	aut, err := s.proveedor.AutorizarRectificacionDietas(ctx, material)
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	audiencia, accion := contratoRectificacion(solicitud.Operacion)
	resumen := aut.ResumenCapacidad()
	huella, err := material.HuellaSHA256()
	if err != nil || aut.ValidarEstructura() != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia || resumen.EfectoRef() != material.Recurso().Referencia || resumen.EfectoHuellaSHA256() != huella || aut.PersonaVersion() != solicitud.Actor.Instantanea.PersonaVersion || aut.PerfilVersion() != solicitud.Actor.Instantanea.PerfilVersion {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	orden := personalports.OrdenRectificacionDietas{Material: material, Autorizacion: aut}
	if solicitud.Operacion == personaldomain.ConfirmarRectificacionDietas {
		if nuloAsignacion(s.proveedorCorreccion) || solicitud.Correccion == nil {
			return vacio, personalports.ErrRectificacionDietasNoDisponible
		}
		correccion, err := personaldomain.NuevoMaterialAsignacionDietas(*solicitud.Correccion)
		if err != nil {
			return vacio, personalports.ErrRectificacionDietasInvalida
		}
		autCorreccion, err := s.proveedorCorreccion.AutorizarAsignacionDietas(ctx, correccion)
		if err != nil || autCorreccion.ValidarEstructura() != nil {
			return vacio, personalports.ErrRectificacionDietasDenegada
		}
		resumenCorreccion := autCorreccion.ResumenCapacidad()
		huellaCorreccion, err := correccion.HuellaSHA256()
		if err != nil || resumenCorreccion.Operacion() != personalports.AccionCorregirAsignacionDietas ||
			resumenCorreccion.AudienciaConsumo() != personalports.AudienciaCorregirAsignacionDietas ||
			resumenCorreccion.EfectoRef() != correccion.Recurso().Referencia ||
			resumenCorreccion.EfectoHuellaSHA256() != huellaCorreccion ||
			autCorreccion.PersonaVersion() != solicitud.Actor.Instantanea.PersonaVersion ||
			autCorreccion.PerfilVersion() != solicitud.Actor.Instantanea.PerfilVersion {
			return vacio, personalports.ErrRectificacionDietasDenegada
		}
		orden.Correccion = correccion
		orden.AutorizacionCorreccion = autCorreccion
	}
	r, err := s.repositorio.EjecutarRectificacionDietas(ctx, orden)
	if err != nil {
		return vacio, err
	}
	if !resultadoRectificacionValido(solicitud, r, resumen.DecisionRef(), resumen.EfectoRef(), resumen.EmitidaEn(), resumen.ExpiraEn()) {
		return vacio, personalports.ErrRectificacionDietasNoDisponible
	}
	return r, nil
}

func contratoRectificacion(o personaldomain.OperacionRectificacionDietas) (audiencia, accion string) {
	switch o {
	case personaldomain.SolicitarRectificacionDietas:
		return personalports.AudienciaSolicitarRectificacionDietas, personalports.AccionSolicitarRectificacionDietas
	case personaldomain.ConsultarRectificacionDietas:
		return personalports.AudienciaConsultarRectificacionDietas, personalports.AccionConsultarRectificacionDietas
	case personaldomain.ConfirmarRectificacionDietas, personaldomain.RechazarRectificacionDietas:
		return personalports.AudienciaResolverRectificacionDietas, personalports.AccionResolverRectificacionDietas
	default:
		return "", ""
	}
}

var patronSolicitudRectificacionResultado = regexp.MustCompile(`^srd_[0-9a-f]{32}$`)
var patronReciboRectificacionResultado = regexp.MustCompile(`^rrd_[0-9a-f]{32}$`)

func resultadoRectificacionValido(s personaldomain.SolicitudRectificacionDietas, r personalports.ResultadoRectificacionDietas, decision, efecto string, emitida, expira time.Time) bool {
	if !patronSolicitudRectificacionResultado.MatchString(r.SolicitudRef) || !patronReciboRectificacionResultado.MatchString(r.ReciboRef) ||
		r.DecisionRef != decision || r.EfectoRef != efecto || !patronHuellaAsignacion.MatchString(r.ConsumoHuellaSHA256) || r.AuditoriaAD3Ref == "" ||
		r.RegistradaEn.IsZero() || r.RegistradaEn.Location() != time.UTC || r.RegistradaEn.Nanosecond()%1000 != 0 ||
		r.VersionOrigen < 1 || r.AsignacionRef == "" {
		return false
	}
	if s.Operacion == personaldomain.SolicitarRectificacionDietas && (r.AsignacionRef != s.AsignacionRef || r.VersionOrigen != s.VersionEsperada) {
		return false
	}
	if s.Operacion == personaldomain.ConfirmarRectificacionDietas || s.Operacion == personaldomain.RechazarRectificacionDietas {
		if r.SolicitudRef != s.SolicitudRef {
			return false
		}
	}
	if r.Estado != "pendiente" && r.Estado != "confirmada" && r.Estado != "rechazada" && r.Estado != "replay_confirmado" {
		return false
	}
	if r.Estado != "replay_confirmado" && (r.RegistradaEn.Before(emitida) || !r.RegistradaEn.Before(expira)) {
		return false
	}
	return true
}
