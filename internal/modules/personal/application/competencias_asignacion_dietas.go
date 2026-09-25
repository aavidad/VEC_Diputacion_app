package application

import (
	"context"
	"regexp"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

// ServicioCompetenciasAsignacionDietas entrega a D6/D8 el sello mínimo de
// Personal. La autorización y la lectura consumida son una sola operación V3.
type ServicioCompetenciasAsignacionDietas struct {
	proveedor   personalports.ProveedorAutorizacionCompetenciasAsignacionDietas
	repositorio personalports.RepositorioCompetenciasAsignacionDietas
}

var patronReciboCompetenciasAsignacion = regexp.MustCompile(`^rca_[0-9a-f]{32}$`)

func NuevoServicioCompetenciasAsignacionDietas(p personalports.ProveedorAutorizacionCompetenciasAsignacionDietas, r personalports.RepositorioCompetenciasAsignacionDietas) (*ServicioCompetenciasAsignacionDietas, error) {
	if nuloAsignacion(p) || nuloAsignacion(r) {
		return nil, personalports.ErrCompetenciasAsignacionDietasNoDisponibles
	}
	return &ServicioCompetenciasAsignacionDietas{proveedor: p, repositorio: r}, nil
}

func (s *ServicioCompetenciasAsignacionDietas) Consultar(ctx context.Context, solicitud personaldomain.SolicitudCompetenciasAsignacionDietas) (personalports.ResultadoConsultaCompetenciasAsignacionDietas, error) {
	var vacio personalports.ResultadoConsultaCompetenciasAsignacionDietas
	if s == nil || ctx == nil || nuloAsignacion(s.proveedor) || nuloAsignacion(s.repositorio) {
		return vacio, personalports.ErrCompetenciasAsignacionDietasNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := personaldomain.NuevoMaterialCompetenciasAsignacionDietas(solicitud)
	if err != nil {
		return vacio, personalports.ErrConsultaCompetenciasAsignacionDietasInvalida
	}
	solicitud = material.Solicitud()
	aut, err := s.proveedor.AutorizarCompetenciasAsignacionDietas(ctx, material)
	if err != nil {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	resumen := aut.ResumenCapacidad()
	huella, err := material.HuellaSHA256()
	if err != nil || aut.ValidarEstructura() != nil || resumen.Operacion() != personalports.AccionConsultarCompetenciasAsignacionDietas ||
		resumen.AudienciaConsumo() != personalports.AudienciaConsultarCompetenciasAsignacionDietas ||
		resumen.EfectoRef() != material.Recurso().Referencia || resumen.EfectoHuellaSHA256() != huella ||
		aut.PersonaVersion() != solicitud.Actor.Instantanea.PersonaVersion || aut.PerfilVersion() != solicitud.Actor.Instantanea.PerfilVersion {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	r, err := s.repositorio.ConsultarCompetenciasAsignacionDietas(ctx, personalports.OrdenConsultaCompetenciasAsignacionDietas{Material: material, Autorizacion: aut})
	if err != nil {
		return vacio, err
	}
	if !resultadoCompetenciasAsignacionValido(solicitud, r, resumen.DecisionRef(), resumen.EfectoRef(), resumen.EmitidaEn(), resumen.ExpiraEn()) {
		return vacio, personalports.ErrCompetenciasAsignacionDietasNoDisponibles
	}
	return r, nil
}

func resultadoCompetenciasAsignacionValido(s personaldomain.SolicitudCompetenciasAsignacionDietas, r personalports.ResultadoConsultaCompetenciasAsignacionDietas, decision, efecto string, emitida, expira time.Time) bool {
	if len(r.Competencias) > 100 || !patronReciboCompetenciasAsignacion.MatchString(r.Evidencia.ReciboRef) ||
		!patronHuellaAsignacion.MatchString(r.Evidencia.ConsumoHuellaSHA256) ||
		r.Evidencia.DecisionRef != decision || r.Evidencia.EfectoRef != efecto || r.Evidencia.AuditoriaRef == "" ||
		r.Evidencia.ConsultadaEn.IsZero() || r.Evidencia.ConsultadaEn.Location() != time.UTC ||
		r.Evidencia.ConsultadaEn.Nanosecond()%1000 != 0 || r.Evidencia.ConsultadaEn.Before(emitida) || !r.Evidencia.ConsultadaEn.Before(expira) {
		return false
	}
	vistas := make(map[string]bool, len(r.Competencias))
	for _, c := range r.Competencias {
		if c.Validar(s.FechaReferencia) != nil || vistas[c.RelacionRef] {
			return false
		}
		vistas[c.RelacionRef] = true
	}
	return true
}
