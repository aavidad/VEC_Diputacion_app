package application

import (
	"context"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type ServicioRectificacionesCompetentesDietas struct {
	proveedor   personalports.ProveedorAutorizacionRectificacionesCompetentesDietas
	repositorio personalports.RepositorioRectificacionesCompetentesDietas
}

func NuevoServicioRectificacionesCompetentesDietas(p personalports.ProveedorAutorizacionRectificacionesCompetentesDietas, r personalports.RepositorioRectificacionesCompetentesDietas) (*ServicioRectificacionesCompetentesDietas, error) {
	if nuloAsignacion(p) || nuloAsignacion(r) {
		return nil, personalports.ErrRectificacionDietasNoDisponible
	}
	return &ServicioRectificacionesCompetentesDietas{proveedor: p, repositorio: r}, nil
}

func (s *ServicioRectificacionesCompetentesDietas) Consultar(ctx context.Context, solicitud personaldomain.SolicitudRectificacionesCompetentesDietas) (personalports.ResultadoConsultaRectificacionesCompetentesDietas, error) {
	var vacio personalports.ResultadoConsultaRectificacionesCompetentesDietas
	if s == nil || ctx == nil || nuloAsignacion(s.proveedor) || nuloAsignacion(s.repositorio) {
		return vacio, personalports.ErrRectificacionDietasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := personaldomain.NuevoMaterialRectificacionesCompetentesDietas(solicitud)
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasInvalida
	}
	aut, err := s.proveedor.AutorizarRectificacionesCompetentesDietas(ctx, material)
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	resumen := aut.ResumenCapacidad()
	huella, err := material.HuellaSHA256()
	if err != nil || aut.ValidarEstructura() != nil ||
		resumen.Operacion() != personalports.AccionConsultarRectificacionesCompetentesDietas ||
		resumen.AudienciaConsumo() != personalports.AudienciaConsultarRectificacionesCompetentesDietas ||
		resumen.EfectoRef() != material.Recurso().Referencia || resumen.EfectoHuellaSHA256() != huella ||
		aut.PersonaVersion() != solicitud.Actor.Instantanea.PersonaVersion || aut.PerfilVersion() != solicitud.Actor.Instantanea.PerfilVersion {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	r, err := s.repositorio.ConsultarRectificacionesCompetentesDietas(ctx, personalports.OrdenConsultaRectificacionesCompetentesDietas{Material: material, Autorizacion: aut})
	if err != nil {
		return vacio, err
	}
	if !resultadoCompetenteRectificacionValido(r, solicitud.Actor.PersonaRef, resumen.DecisionRef(), resumen.EfectoRef(), resumen.EmitidaEn(), resumen.ExpiraEn()) {
		return vacio, personalports.ErrRectificacionDietasNoDisponible
	}
	return r, nil
}

func resultadoCompetenteRectificacionValido(r personalports.ResultadoConsultaRectificacionesCompetentesDietas, actor, decision, efecto string, emitida, expira time.Time) bool {
	if !patronReciboRectificacionResultado.MatchString(r.ReciboRef) || r.DecisionRef != decision || r.EfectoRef != efecto ||
		!patronHuellaAsignacion.MatchString(r.ConsumoHuellaSHA256) || r.AuditoriaAD3Ref == "" ||
		r.ConsultadaEn.IsZero() || r.ConsultadaEn.Location() != time.UTC || r.ConsultadaEn.Nanosecond()%1000 != 0 ||
		r.ConsultadaEn.Before(emitida) || !r.ConsultadaEn.Before(expira) ||
		r.Cardinalidad != len(r.Solicitudes) || r.Cardinalidad < 0 || r.Cardinalidad > 50 {
		return false
	}
	for _, v := range r.Solicitudes {
		if !patronSolicitudRectificacionResultado.MatchString(v.SolicitudRef) || v.Estado != "pendiente" ||
			!personaldomain.ReferenciaPersonaValida(v.PersonaRef) || !personaldomain.ReferenciaEmpleadoValida(v.EmpleadoRef) ||
			!personaldomain.ReferenciaRelacionValida(v.RelacionRef) || v.UnidadRef == "" || v.AsignacionRef == "" ||
			v.VersionOrigen < 1 || v.FechaReferencia.Validar() != nil || len(v.CamposARevisar) < 1 ||
			v.RegistradaEn.IsZero() || v.AsignacionActual.AdministrativoPersonaRef != actor ||
			v.AsignacionActual.Version < v.VersionOrigen || v.AsignacionActual.AsignacionRef == "" {
			return false
		}
	}
	return true
}
