package internactproveedores

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
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
