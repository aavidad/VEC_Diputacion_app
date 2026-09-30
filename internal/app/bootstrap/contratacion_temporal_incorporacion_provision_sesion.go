package bootstrap

import (
	"context"
	"errors"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Conserva el registro, las cápsulas y el revalidador comunes. Sólo separa
// una negativa de autorización de la indisponibilidad de una dependencia;
// la causa queda disponible internamente y el adaptador HTTP la redacta.
type sesionNominalIncorporacion struct {
	proveedor *proveedorSesionConsultaRRHHDesarrollo
}

// Captura por llamada: nunca comparte un error mutable entre peticiones.
type registroSesionNominalIncorporacion struct {
	httpseguridad.RegistroSesiones
	causa error
}

func (r *registroSesionNominalIncorporacion) ConsumirAsercionYRegistrar(ctx context.Context, alta httpseguridad.AltaSesionAtomica) (httpseguridad.ConfirmacionAltaSesion, error) {
	confirmacion, err := r.RegistroSesiones.ConsumirAsercionYRegistrar(ctx, alta)
	r.causa = err
	return confirmacion, err
}

func errorSesionNominalIncorporacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, denegada := range []error{ErrSeguridadComunDesarrolloDenegada, ct.ErrAutorizacionDenegada, core.ErrAutorizacionDenegada, core.ErrAutenticacionRevalidadaInvalida, core.ErrContextoActorNoResuelto, core.ErrContextoActorInvalido, core.ErrInstantaneaContextoActorInvalida, core.ErrSolicitudContextoActorInvalida} {
		if errors.Is(err, denegada) {
			return errors.Join(ct.ErrAutorizacionDenegada, err)
		}
	}
	return errors.Join(ct.ErrConsultaRRHHNoDisponible, err)
}

func (s *sesionNominalIncorporacion) ResolverContexto(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if s == nil || s.proveedor == nil || ctx == nil {
		return vacio, ct.ErrAutorizacionDenegada
	}
	p := *s.proveedor
	for _, d := range []any{p.registro, p.revalidador, p.resolutor, p.reloj} {
		if dependenciaEsNulaContratacionTemporalDesarrollo(d) {
			return vacio, ct.ErrConsultaRRHHNoDisponible
		}
	}
	registro := &registroSesionNominalIncorporacion{RegistroSesiones: p.registro}
	p.registro = registro
	perfil, seleccionado := p.perfilActivoSeleccionado(ctx)
	if !seleccionado {
		return vacio, ct.ErrAutorizacionDenegada
	}
	ctxCapsula, capsula, err := p.acreditarPeticion(ctx)
	if err != nil {
		return vacio, errorSesionNominalIncorporacion(ctx, err)
	}
	alta, confirmacion, err := p.registrarCapsula(ctxCapsula, capsula)
	if err != nil {
		return vacio, errorSesionNominalIncorporacion(ctx, errors.Join(err, registro.causa))
	}
	revalidador := revalidadorSesionConsultaRRHHDesarrollo{delegado: p.revalidador, alta: alta, confirmacion: confirmacion, reloj: p.reloj, superficie: p.superficie}
	vinculo, resultado, err := core.CrearVinculoAutenticacionActorV2ConResultado(ctxCapsula, revalidador,
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef}, p.resolutor,
		core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: p.base.Contexto.Instantanea.CuentaRef, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}, PerfilActivoRef: perfil}, p.reloj)
	if err != nil {
		return vacio, errorSesionNominalIncorporacion(ctx, err)
	}
	if !mismaIdentidadVersionadaSesionDesarrolloParaPerfil(p.base, resultado, perfil) || vinculo.ValidarPara(resultado) != nil {
		return vacio, ct.ErrAutorizacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	return contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado}, nil
}
