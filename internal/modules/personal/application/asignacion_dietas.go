package application

import (
	"context"
	"reflect"
	"regexp"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type ServicioAsignacionDietas struct {
	proveedor   personalports.ProveedorAutorizacionAsignacionDietas
	repositorio personalports.RepositorioAsignacionDietas
}

func NuevoServicioAsignacionDietas(p personalports.ProveedorAutorizacionAsignacionDietas, r personalports.RepositorioAsignacionDietas) (*ServicioAsignacionDietas, error) {
	if nuloAsignacion(p) || nuloAsignacion(r) {
		return nil, personalports.ErrAsignacionDietasNoDisponible
	}
	return &ServicioAsignacionDietas{proveedor: p, repositorio: r}, nil
}

func (s *ServicioAsignacionDietas) Ejecutar(ctx context.Context, solicitud personaldomain.SolicitudAsignacionDietas) (personalports.ResultadoAsignacionDietas, error) {
	var vacio personalports.ResultadoAsignacionDietas
	if s == nil || ctx == nil || nuloAsignacion(s.proveedor) || nuloAsignacion(s.repositorio) {
		return vacio, personalports.ErrAsignacionDietasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := personaldomain.NuevoMaterialAsignacionDietas(solicitud)
	if err != nil {
		return vacio, personalports.ErrSolicitudAsignacionDietasInvalida
	}
	solicitud = material.Solicitud()
	aut, err := s.proveedor.AutorizarAsignacionDietas(ctx, material)
	if err != nil {
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	audiencia, accion := contratoAsignacion(solicitud.Operacion)
	resumen := aut.ResumenCapacidad()
	huella, err := material.HuellaSHA256()
	if err != nil || aut.ValidarEstructura() != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia || resumen.EfectoRef() != material.Recurso().Referencia || resumen.EfectoHuellaSHA256() != huella || aut.PersonaVersion() != solicitud.Actor.Instantanea.PersonaVersion || aut.PerfilVersion() != solicitud.Actor.Instantanea.PerfilVersion {
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	r, err := s.repositorio.EjecutarAsignacionDietas(ctx, personalports.OrdenAsignacionDietas{Material: material, Autorizacion: aut})
	if err != nil {
		return vacio, err
	}
	if !resultadoAsignacionValido(solicitud, r, aut.ResumenCapacidad().DecisionRef(), aut.ResumenCapacidad().EfectoRef(), resumen.EmitidaEn(), resumen.ExpiraEn()) {
		return vacio, personalports.ErrAsignacionDietasNoDisponible
	}
	return r, nil
}

func contratoAsignacion(o personaldomain.OperacionAsignacionDietas) (audiencia, accion string) {
	switch o {
	case personaldomain.ConsultarAsignacionDietas:
		return personalports.AudienciaConsultarAsignacionDietas, personalports.AccionConsultarAsignacionDietas
	case personaldomain.RegistrarInicialAsignacionDietas:
		return personalports.AudienciaRegistrarInicialAsignacionDietas, personalports.AccionRegistrarInicialAsignacionDietas
	case personaldomain.CorregirAsignacionDietas:
		return personalports.AudienciaCorregirAsignacionDietas, personalports.AccionCorregirAsignacionDietas
	case personaldomain.CorregirGrupoAsignacionDietas:
		return personalports.AudienciaCorregirGrupoDietas, personalports.AccionCorregirGrupoDietas
	default:
		return "", ""
	}
}

var patronReciboAsignacion = regexp.MustCompile(`^rad_[0-9a-f]{32}$`)
var patronHuellaAsignacion = regexp.MustCompile(`^[0-9a-f]{64}$`)

func resultadoAsignacionValido(s personaldomain.SolicitudAsignacionDietas, r personalports.ResultadoAsignacionDietas, decision, efecto string, emitida, expira time.Time) bool {
	a := r.Asignacion
	if a.Validar() != nil || a.RelacionRef != s.RelacionRef || a.PersonaRef != s.PersonaRef || a.UnidadRef != s.UnidadRef || !a.VigenteEn(s.FechaReferencia) || !patronReciboAsignacion.MatchString(r.ReciboRef) || !patronHuellaAsignacion.MatchString(r.ConsumoHuellaSHA256) || r.DecisionRef != decision || r.EfectoRef != efecto || r.AuditoriaRef == "" || r.RegistradaEn.IsZero() || r.RegistradaEn.Location() != time.UTC || r.RegistradaEn.Nanosecond()%1000 != 0 {
		return false
	}
	if s.Operacion == personaldomain.ConsultarAsignacionDietas {
		return r.EstadoLocal == "consultada" && !r.RegistradaEn.Before(emitida) && r.RegistradaEn.Before(expira)
	}
	if r.EstadoLocal != "registrada" && r.EstadoLocal != "replay_confirmado" {
		return false
	}
	if r.EstadoLocal == "registrada" && (r.RegistradaEn.Before(emitida) || !r.RegistradaEn.Before(expira)) {
		return false
	}
	if a.Version != s.VersionEsperada+1 || a.CentroRef != s.CentroRef || a.AdministrativoPersonaRef != s.AdministrativoPersonaRef || a.ResponsablePersonaRef != s.ResponsablePersonaRef || a.GrupoDieta != s.GrupoDieta || a.VigenteDesde != s.VigenteDesde {
		return false
	}
	return true
}

func nuloAsignacion(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}
