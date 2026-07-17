package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrServicioPanelInternoInvalido  = errors.New("bolsa: servicio de panel interno invalido")
	ErrOrdenPanelInternoInvalida     = errors.New("bolsa: orden de panel interno invalida")
	ErrDatosPanelInternoNoConfiables = errors.New("bolsa: datos de panel interno no confiables")
)

// OrdenConsultaPanelInterno recibe capacidades resueltas por la frontera
// interna. No admite roles, permisos ni garantia declarados por el cliente.
type OrdenConsultaPanelInterno struct {
	ContextoActor             dominiovec.ContextoActor
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
	Selector                  puertosbolsa.SelectorPanelInterno
	MotivoCatalogo            dominiovec.ReferenciaEntradaCatalogo
	Correlacion               dominiovec.ReferenciaCorrelacionAutorizacionV2
}

// ServicioConsultaPanelInterno es el PEP del cuadro operativo de Bolsa. Solo
// acepta sesion interna de garantia alta y una concesion PDP ligada V2.
type ServicioConsultaPanelInterno struct {
	consulta puertosbolsa.ConsultaPanelInterno
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	reloj    puertosvec.Reloj
	politica aplicacionvec.PoliticaUsoDecisionAutorizacion
}

func NuevoServicioConsultaPanelInterno(
	consulta puertosbolsa.ConsultaPanelInterno,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	reloj puertosvec.Reloj,
) (*ServicioConsultaPanelInterno, error) {
	if dependenciaPanelInternoNula(consulta) || dependenciaPanelInternoNula(exigidor) ||
		dependenciaPanelInternoNula(reloj) {
		return nil, ErrServicioPanelInternoInvalido
	}
	politica, err := aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		puertosbolsa.AccionConsultarPanelInterno,
		puertosbolsa.ModuloPanelInternoBolsa,
		puertosbolsa.TipoRecursoPanelInternoBolsa,
		puertosbolsa.FinalidadPanelInternoBolsa,
		[]string{puertosbolsa.CampoPanelInternoAgregado},
		aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto,
	)
	if err != nil {
		return nil, errors.Join(ErrServicioPanelInternoInvalido, err)
	}
	return &ServicioConsultaPanelInterno{
		consulta: consulta, exigidor: exigidor, reloj: reloj, politica: politica,
	}, nil
}

func (s *ServicioConsultaPanelInterno) Consultar(
	ctx context.Context,
	orden OrdenConsultaPanelInterno,
) (puertosbolsa.InstantaneaPanelInterno, error) {
	if ctx == nil || s == nil || dependenciaPanelInternoNula(s.consulta) ||
		dependenciaPanelInternoNula(s.exigidor) || dependenciaPanelInternoNula(s.reloj) {
		return puertosbolsa.InstantaneaPanelInterno{}, ErrServicioPanelInternoInvalido
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	consultadaEn := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	actor, err := validarOrdenPanelInterno(orden, consultadaEn)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	recurso, err := puertosbolsa.RecursoAutorizablePanelInterno(
		orden.Selector,
		orden.MotivoCatalogo,
	)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrOrdenPanelInternoInvalida,
			err,
		)
	}

	evidencia, err := s.exigidor.ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		ctx,
		actor,
		orden.VinculoAutenticacionActor,
		recurso,
		orden.Correlacion,
		orden.MotivoCatalogo,
		s.politica,
	)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	consultadaEn = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if consultadaEn.IsZero() || evidencia.ValidarEn(consultadaEn) != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida,
		)
	}
	solicitud, err := puertosbolsa.NuevaSolicitudConsultaPanelInterno(
		orden.Selector,
		evidencia,
		orden.MotivoCatalogo,
		orden.Correlacion,
		consultadaEn,
	)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrOrdenPanelInternoInvalida,
			err,
		)
	}
	resultado, err := s.consulta.ConsultarPanel(ctx, solicitud)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	resultado, err = resultado.ClonarValidadaPara(solicitud)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, errors.Join(
			ErrDatosPanelInternoNoConfiables,
			err,
		)
	}
	return resultado, nil
}

func validarOrdenPanelInterno(
	orden OrdenConsultaPanelInterno,
	instante time.Time,
) (dominiovec.ContextoActor, error) {
	actor, errActor := orden.ContextoActor.Clonar()
	datosVinculo, errVinculo := orden.VinculoAutenticacionActor.Datos()
	if instante.IsZero() || errActor != nil || errVinculo != nil ||
		actor.Principal.AuthMethod == dominiovec.AuthMethodDemo ||
		actor.Principal.AuthAssurance != dominiovec.AuthAssuranceHigh ||
		datosVinculo.MetodoObservado == dominiovec.AuthMethodDemo ||
		datosVinculo.GarantiaObservada != dominiovec.AuthAssuranceHigh ||
		!superficieInternaPanelValida(datosVinculo.Superficie) ||
		orden.VinculoAutenticacionActor.ValidarPara(actor) != nil ||
		!orden.VinculoAutenticacionActor.VigenteEn(instante, actor) ||
		orden.Selector.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(orden.MotivoCatalogo) ||
		orden.Correlacion.Validar() != nil {
		return dominiovec.ContextoActor{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrOrdenPanelInternoInvalida,
			errActor,
			errVinculo,
		)
	}
	return actor, nil
}

func superficieInternaPanelValida(superficie dominiovec.SuperficieAutenticacionActorV1) bool {
	return superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 ||
		superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1
}

func dependenciaPanelInternoNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
