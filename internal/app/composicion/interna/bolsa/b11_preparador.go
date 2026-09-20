package bolsa

import (
	"context"
	"errors"
	"reflect"

	httpinternobolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrPreparadorParticipacionesPropiasB11Invalido = errors.New(
	"composicion interna bolsa: preparador B11 invalido",
)

// extractorConfirmacionPeticionSesion conserva la frontera opaca de la
// petición ya consumida. La implementación productiva es ServicioPeticionSesion;
// el preparador no recibe afirmaciones, perfiles ni referencias de candidato.
type extractorConfirmacionPeticionSesion interface {
	ExtraerConfirmacionVinculada(context.Context) (httpseguridad.ConfirmacionPeticionSesion, error)
}

// PreparadorOrdenParticipacionesPropiasB11 forma la orden de aplicación a
// partir de la confirmación durable ligada a esta misma petición. No crea ni
// renueva una sesión: la fábrica revalida la autenticación una única vez y el
// resolutor gobernado decide el perfil y candidato desde sus fuentes.
type PreparadorOrdenParticipacionesPropiasB11 struct {
	peticion    extractorConfirmacionPeticionSesion
	revalidador domain.RevalidadorAutenticacionActorV1
	resolutor   domain.ResolutorContextoActorGobernadoV1
	reloj       domain.RelojVinculoAutenticacionActorV2
}

func NuevoPreparadorOrdenParticipacionesPropiasB11(
	peticion extractorConfirmacionPeticionSesion,
	revalidador domain.RevalidadorAutenticacionActorV1,
	resolutor domain.ResolutorContextoActorGobernadoV1,
	reloj domain.RelojVinculoAutenticacionActorV2,
) (*PreparadorOrdenParticipacionesPropiasB11, error) {
	if dependenciaPreparadorB11Nula(peticion) ||
		dependenciaPreparadorB11Nula(revalidador) ||
		dependenciaPreparadorB11Nula(resolutor) ||
		dependenciaPreparadorB11Nula(reloj) {
		return nil, ErrPreparadorParticipacionesPropiasB11Invalido
	}
	return &PreparadorOrdenParticipacionesPropiasB11{
		peticion: peticion, revalidador: revalidador, resolutor: resolutor, reloj: reloj,
	}, nil
}

// PrepararOrdenConsultaParticipacionesPropias extrae una sola vez la
// confirmación ligada al contexto. A partir de
// sus dos referencias canónicas crea la solicitud mínima de revalidación; ni el
// transporte ni el cliente pueden escoger identidad, perfil o candidatura.
func (p *PreparadorOrdenParticipacionesPropiasB11) PrepararOrdenConsultaParticipacionesPropias(
	ctx context.Context,
) (aplicacionbolsa.OrdenConsultaParticipacionesPropias, error) {
	vacia := aplicacionbolsa.OrdenConsultaParticipacionesPropias{}
	if ctx == nil || ctx.Err() != nil || p == nil ||
		dependenciaPreparadorB11Nula(p.peticion) ||
		dependenciaPreparadorB11Nula(p.revalidador) ||
		dependenciaPreparadorB11Nula(p.resolutor) ||
		dependenciaPreparadorB11Nula(p.reloj) {
		return vacia, ErrPreparadorParticipacionesPropiasB11Invalido
	}

	confirmacion, err := p.peticion.ExtraerConfirmacionVinculada(ctx)
	if err != nil || ctx.Err() != nil {
		return vacia, errors.Join(
			ErrPreparadorParticipacionesPropiasB11Invalido,
			httpinternobolsa.ErrAutenticacionInternaAusente,
			err,
			ctx.Err(),
		)
	}
	solicitud := domain.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: confirmacion.AutenticacionRef,
		SesionRef:        confirmacion.SesionRef,
	}
	// La cápsula ya verificó íntegramente la confirmación con el reloj de la
	// sesión. Aquí solo se admiten las dos referencias que el dominio autoriza
	// transportar; la fábrica vuelve a revalidarlas frente a su autoridad.
	if solicitud.Validar() != nil {
		return vacia, errors.Join(
			ErrPreparadorParticipacionesPropiasB11Invalido,
			httpinternobolsa.ErrAutenticacionInternaAusente,
		)
	}

	vinculo, resultado, err := domain.CrearVinculoAutenticacionActorGobernadoV1(
		ctx, p.revalidador, solicitud, p.resolutor, p.reloj,
	)
	if err != nil || ctx.Err() != nil || vinculo.ValidarPara(resultado) != nil {
		if errors.Is(err, puertosvec.ErrRevalidacionAutenticacionActorNoDisponible) ||
			errors.Is(err, puertosvec.ErrFuenteContextoActorNoDisponible) {
			return vacia, errors.Join(
				ErrPreparadorParticipacionesPropiasB11Invalido,
				err,
				ctx.Err(),
			)
		}
		return vacia, errors.Join(
			ErrPreparadorParticipacionesPropiasB11Invalido,
			httpinternobolsa.ErrAutenticacionInternaAusente,
			err,
			ctx.Err(),
		)
	}
	return aplicacionbolsa.OrdenConsultaParticipacionesPropias{
		Vinculo: vinculo, Resultado: resultado,
	}, nil
}

var _ httpinternobolsa.PreparadorOrdenConsultaParticipacionesPropias = (*PreparadorOrdenParticipacionesPropiasB11)(nil)

func dependenciaPreparadorB11Nula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
