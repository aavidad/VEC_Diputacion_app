package interna

import (
	"context"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vec "vec-diputacion-granada/internal/vec/domain"
)

// selectorPerfilActivoInstitucional recibe solo identidad ya autenticada. Su
// implementación deberá acreditar el perfil único seleccionado en la aserción
// institucional; ni una cabecera libre ni el cuerpo de la consulta lo suplen.
type selectorPerfilActivoInstitucional interface {
	SeleccionarPerfilActivo(context.Context, vec.CuentaAutenticadaContextoActor, httpseguridad.ContextoAuditoriaAutenticada) (string, error)
}

type contextoActorLecturaCT struct {
	identidad   *httpseguridad.ServicioIdentidad
	selector    selectorPerfilActivoInstitucional
	revalidador vec.RevalidadorAutenticacionActorV1
	resolutor   vec.ResolutorContextoActorRegistradoV2
	reloj       ctports.Reloj
}

// resolver solo funciona dentro del contexto ligado por la fachada C4/C5.
// ExtraerCapsulaIdentidadPeticion revalida la sesión durable; el vínculo F1
// vuelve a leer autenticación y ContextoActor registrados para esta petición.
func (a contextoActorLecturaCT) resolver(ctx context.Context) (ctports.ContextoAutorizacionAltaV3, error) {
	vacio := ctports.ContextoAutorizacionAltaV3{}
	if ctx == nil || ctx.Err() != nil || a.identidad == nil ||
		interfazNulaIdentidadOffline(a.selector) ||
		interfazNulaIdentidadOffline(a.revalidador) ||
		interfazNulaIdentidadOffline(a.resolutor) ||
		interfazNulaIdentidadOffline(a.reloj) {
		return vacio, ctports.ErrConsultaRRHHNoDisponible
	}
	cuenta, auditoria, err := a.identidad.ExtraerCapsulaIdentidadPeticion(ctx)
	if err != nil || cuenta.Validar() != nil ||
		cuenta.Garantia != vec.AuthAssuranceHigh ||
		auditoria.Superficie() != httpseguridad.SuperficieInternaCorporativa ||
		auditoria.ControlSesionEstado() != httpseguridad.EstadoControlSesionActiva ||
		auditoria.CuentaRef() != cuenta.CuentaRef ||
		auditoria.MetodoObservado() != cuenta.Metodo ||
		auditoria.Garantia() != cuenta.Garantia {
		return vacio, ctports.ErrConsultaRRHHNoDisponible
	}
	perfil, err := a.selector.SeleccionarPerfilActivo(ctx, cuenta, auditoria)
	solicitudContexto := vec.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: perfil}
	if err != nil || solicitudContexto.Validar() != nil {
		return vacio, ctports.ErrConsultaRRHHNoDisponible
	}
	vinculo, resultado, err := vec.CrearVinculoAutenticacionActorV2ConResultado(
		ctx,
		a.revalidador,
		vec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: auditoria.AutenticacionRef(),
			SesionRef:        auditoria.SesionRef(),
		},
		a.resolutor,
		solicitudContexto,
		a.reloj,
	)
	if err != nil || vinculo.ValidarPara(resultado) != nil || ctx.Err() != nil {
		return vacio, ctports.ErrConsultaRRHHNoDisponible
	}
	return ctports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}, nil
}

type autoridadContextoConsultaRRHH struct {
	actor           contextoActorLecturaCT
	organizacionRef string
	clase           ctports.ClaseAmbitoConsultaRRHH
	ambitoRef       string
}

func (a autoridadContextoConsultaRRHH) ResolverContextoConsultaRRHH(ctx context.Context) (ctports.ContextoConsultaRRHH, error) {
	contexto, err := a.actor.resolver(ctx)
	if err != nil || interfazNulaIdentidadOffline(a.actor.reloj) {
		return ctports.ContextoConsultaRRHH{}, ctports.ErrConsultaRRHHNoDisponible
	}
	consulta, err := ctports.NuevoContextoConsultaRRHHConAmbito(
		contexto, a.organizacionRef, a.clase, a.ambitoRef, a.actor.reloj.Ahora(),
	)
	if err != nil {
		return ctports.ContextoConsultaRRHH{}, ctports.ErrConsultaRRHHNoDisponible
	}
	return consulta, nil
}

var _ ctports.AutoridadContextoConsultaRRHH = autoridadContextoConsultaRRHH{}
