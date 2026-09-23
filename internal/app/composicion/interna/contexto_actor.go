package interna

import (
	"context"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
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
	actor   contextoActorLecturaCT
	ambitos resolutorAmbitoConsultaRRHHRegistrado
}

// El dueño de organización resuelve el ámbito exacto y vigente para el mismo
// actor/perfil ya revalidado. Ningún campo procede de parámetros HTTP.
type resolutorAmbitoConsultaRRHHRegistrado interface {
	ResolverAmbitoConsultaRRHH(context.Context, vec.ResultadoContextoActorRegistradoV2) (ambitoConsultaRRHHRegistrado, error)
}

type ambitoConsultaRRHHRegistrado struct {
	RegistroRef     string
	ActorRef        string
	PerfilRef       string
	PerfilVersion   uint64
	ContextoRef     string
	ContextoHuella  string
	OrganizacionRef string
	Clase           ctports.ClaseAmbitoConsultaRRHH
	AmbitoRef       string
	VigenteDesde    time.Time
	VigenteHasta    time.Time
}

func (a ambitoConsultaRRHHRegistrado) coincideCon(resultado vec.ResultadoContextoActorRegistradoV2, instante time.Time) bool {
	return ctdomain.ReferenciaOpacaValida(a.RegistroRef) &&
		ctdomain.ReferenciaOpacaValida(a.OrganizacionRef) &&
		ctdomain.ReferenciaOpacaValida(a.AmbitoRef) &&
		a.ActorRef == resultado.Contexto.Principal.ID &&
		a.PerfilRef == resultado.Contexto.PerfilActivoRef &&
		a.PerfilVersion == resultado.Contexto.Instantanea.PerfilVersion &&
		a.ContextoRef == resultado.RegistroContextoRef &&
		a.ContextoHuella == resultado.HuellaSHA256 &&
		ctdomain.InstanteUTCCanonico(a.VigenteDesde) &&
		ctdomain.InstanteUTCCanonico(a.VigenteHasta) &&
		ctdomain.InstanteUTCCanonico(instante) &&
		!instante.Before(a.VigenteDesde) && instante.Before(a.VigenteHasta)
}

func (a autoridadContextoConsultaRRHH) ResolverContextoConsultaRRHH(ctx context.Context) (ctports.ContextoConsultaRRHH, error) {
	contexto, err := a.actor.resolver(ctx)
	if err != nil || interfazNulaIdentidadOffline(a.ambitos) ||
		interfazNulaIdentidadOffline(a.actor.reloj) {
		return ctports.ContextoConsultaRRHH{}, ctports.ErrConsultaRRHHNoDisponible
	}
	ambito, err := a.ambitos.ResolverAmbitoConsultaRRHH(ctx, contexto.Resultado)
	instante := a.actor.reloj.Ahora()
	if err != nil || !ambito.coincideCon(contexto.Resultado, instante) || ctx.Err() != nil {
		return ctports.ContextoConsultaRRHH{}, ctports.ErrConsultaRRHHNoDisponible
	}
	consulta, err := ctports.NuevoContextoConsultaRRHHConAmbito(
		contexto, ambito.OrganizacionRef, ambito.Clase, ambito.AmbitoRef, instante,
	)
	if err != nil {
		return ctports.ContextoConsultaRRHH{}, ctports.ErrConsultaRRHHNoDisponible
	}
	return consulta, nil
}

var _ ctports.AutoridadContextoConsultaRRHH = autoridadContextoConsultaRRHH{}
