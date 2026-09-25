package composicion

import (
	"context"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

// ProveedorAsignacionParaEnvioPersonal consulta Personal con su autorización
// nominal. Dietas recibe una proyección sellada y SQL revalida su versión.
type ProveedorAsignacionParaEnvioPersonal struct {
	servicio interface {
		Ejecutar(context.Context, personaldomain.SolicitudAsignacionDietas) (personalports.ResultadoAsignacionDietas, error)
	}
}

func NuevoProveedorAsignacionParaEnvioPersonal(servicio interface {
	Ejecutar(context.Context, personaldomain.SolicitudAsignacionDietas) (personalports.ResultadoAsignacionDietas, error)
}) (*ProveedorAsignacionParaEnvioPersonal, error) {
	if nulo(servicio) {
		return nil, dietasports.ErrRelacionNoDisponible
	}
	return &ProveedorAsignacionParaEnvioPersonal{servicio: servicio}, nil
}

func (p *ProveedorAsignacionParaEnvioPersonal) ConsultarAsignacionParaEnvio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador) (dietasports.AsignacionDietasAcreditada, error) {
	var cero dietasports.AsignacionDietasAcreditada
	if p == nil || nulo(p.servicio) || ctx == nil || ctx.Err() != nil || identidad.ContextoRegistrado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.ContextoRegistrado) != nil {
		return cero, dietasports.ErrRelacionNoDisponible
	}
	fecha, err := personaldomain.NuevaFechaCivil(identidad.Autorizacion.Revalidacion.FechaReferencia)
	if err != nil || identidad.Relacion.RelacionRef == "" || identidad.Relacion.PersonaRef != identidad.ContextoRegistrado.Contexto.PersonaRef {
		return cero, dietasports.ErrRelacionNoDisponible
	}
	r, err := p.servicio.Ejecutar(ctx, personaldomain.SolicitudAsignacionDietas{Actor: identidad.ContextoRegistrado.Contexto, RelacionRef: identidad.Relacion.RelacionRef, UnidadRef: identidad.Relacion.UnidadRef, FechaReferencia: fecha, Operacion: personaldomain.ConsultarAsignacionDietas})
	if err != nil || r.EstadoLocal != "consultada" || r.Asignacion.Validar() != nil || r.Asignacion.RelacionRef != identidad.Relacion.RelacionRef || r.Asignacion.PersonaRef != identidad.Relacion.PersonaRef || r.Asignacion.UnidadRef != identidad.Relacion.UnidadRef || !r.Asignacion.VigenteEn(fecha) || r.ReciboRef == "" || r.DecisionRef == "" || r.EfectoRef == "" || r.ConsumoHuellaSHA256 == "" || r.AuditoriaRef == "" || r.RegistradaEn.IsZero() {
		return cero, dietasports.ErrRelacionNoDisponible
	}
	a := r.Asignacion
	return dietasports.AsignacionDietasAcreditada{AsignacionRef: a.AsignacionRef, RelacionRef: a.RelacionRef, PersonaRef: a.PersonaRef, UnidadRef: a.UnidadRef, CentroRef: a.CentroRef, AdministrativoPersonaRef: a.AdministrativoPersonaRef, ResponsablePersonaRef: a.ResponsablePersonaRef, GrupoDieta: a.GrupoDieta, VigenteDesde: a.VigenteDesde.Texto(), Version: a.Version, ReciboRef: r.ReciboRef, DecisionRef: r.DecisionRef, EfectoRef: r.EfectoRef, ConsumoHuellaSHA256: r.ConsumoHuellaSHA256, AuditoriaRef: r.AuditoriaRef, RegistradaEn: r.RegistradaEn}, nil
}

var _ dietasports.ProveedorAsignacionParaEnvio = (*ProveedorAsignacionParaEnvioPersonal)(nil)
