package internactproveedores

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrAutoridadCTNoDisponible = errors.New("composicion interna: autoridad CT no disponible")

// La misma fuente F1 que recibe incorporacionejercicio resuelve la persona,
// el perfil y la sesión al consultar Detalle. Organización es una selección
// nominal, nunca una persona ni un permiso.
type AutoridadDetalle struct {
	Fuente *internagobierno.FuenteF1
	Reloj  ct.Reloj
}

func (a AutoridadDetalle) ResolverContextoConsultaRRHH(ctx context.Context) (ct.ContextoConsultaRRHH, error) {
	if a.Fuente == nil || a.Reloj == nil || ctx == nil || ctx.Err() != nil {
		return ct.ContextoConsultaRRHH{}, ErrAutoridadCTNoDisponible
	}
	p, err := a.Fuente.PeticionVerificada(ctx)
	if err != nil {
		return ct.ContextoConsultaRRHH{}, ErrAutoridadCTNoDisponible
	}
	autoridad, err := a.Fuente.ResolverContexto(ctx)
	if err != nil || ctx.Err() != nil {
		return ct.ContextoConsultaRRHH{}, ErrAutoridadCTNoDisponible
	}
	v, err := autoridad.Vinculo.Datos()
	if err != nil || v.PerfilActivoRef != p.Contexto.PerfilActivoRef ||
		v.CuentaRef != p.Contexto.Cuenta.CuentaRef ||
		autoridad.Resultado.Contexto.PersonaRef != v.PrincipalID {
		return ct.ContextoConsultaRRHH{}, ErrAutoridadCTNoDisponible
	}
	resultado, err := ct.NuevoContextoConsultaRRHH(autoridad, p.PreparacionCT.OrganizacionRef, a.Reloj.Ahora())
	if err != nil {
		return ct.ContextoConsultaRRHH{}, ErrAutoridadCTNoDisponible
	}
	return resultado, nil
}

var _ ct.AutoridadContextoConsultaRRHH = AutoridadDetalle{}

// AutoridadContextoRegistroEmpleadoB2 entrega el actor acreditado por F1 y
// la organización nominal del servidor. La ficha autoriza después el emp_ref
// objetivo en V3; el actor RRHH no necesita un vínculo de empleado propio.
type AutoridadContextoRegistroEmpleadoB2 struct {
	Fuente *internagobierno.FuenteF1
}

func (a AutoridadContextoRegistroEmpleadoB2) ResolverContextoRegistroEmpleadoB2(ctx context.Context) (vecdomain.ContextoActor, string, error) {
	if a.Fuente == nil {
		return vecdomain.ContextoActor{}, "", ErrAutoridadCTNoDisponible
	}
	actor, organismo, err := a.Fuente.ResolverContextoPersonalB2(ctx)
	if err != nil {
		return vecdomain.ContextoActor{}, "", ErrAutoridadCTNoDisponible
	}
	return actor, organismo, nil
}

var _ httpapi.AutoridadContextoRegistroEmpleadoB2 = AutoridadContextoRegistroEmpleadoB2{}

// AuditorDenegacionRegistroEmpleadoB2 usa la autoridad de auditoría común.
// La ruta patrón de ficha evita guardar referencias de empleado en la frontera.
type AuditorDenegacionRegistroEmpleadoB2 struct {
	Registrador vecports.RegistradorAuditoriaFronteraRutaExacta
}

func (a AuditorDenegacionRegistroEmpleadoB2) RegistrarDenegacionRegistroEmpleadoB2(ctx context.Context, d httpapi.DenegacionRegistroEmpleadoB2) error {
	if a.Registrador == nil || ctx == nil || ctx.Err() != nil {
		return ErrAutoridadCTNoDisponible
	}
	motivo := vecports.MotivoAuditoriaFronteraRutaExacta(d.Motivo)
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: d.CorrelacionRef, Motivo: motivo,
		Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaPersonal,
		Ruta:       d.Ruta, ActorRef: d.ActorRef,
	}
	if orden.Validar() != nil {
		return ErrAutoridadCTNoDisponible
	}
	return a.Registrador.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
}

var _ httpapi.AuditorDenegacionRegistroEmpleadoB2 = AuditorDenegacionRegistroEmpleadoB2{}
