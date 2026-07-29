package ports

import (
	"reflect"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type clasePreparacionAutorizacionConsultaRRHH uint8

const (
	clasePreparacionAutorizacionCuadroRRHH clasePreparacionAutorizacionConsultaRRHH = iota + 1
	clasePreparacionAutorizacionDetalleRRHH
)

// preparacionAutorizacionConsultaRRHH es el nominal privado que un futuro
// guardián entregará a la fachada VEC. No autoriza ni emite material.
type preparacionAutorizacionConsultaRRHH struct {
	bloqueoSerializacionConsultaRRHH
	clase            clasePreparacionAutorizacionConsultaRRHH
	contexto         ContextoConsultaRRHH
	solicitudCuadro  SolicitudCuadroRRHH
	solicitudDetalle SolicitudDetalleRRHH
	recursos         RecursosConsultaRRHH
	motivo           dominiovec.ReferenciaEntradaCatalogo
	correlacion      dominiovec.ReferenciaCorrelacionAutorizacionV2
	solicitudVEC     dominiovec.SolicitudAutorizacionLigadaV3
	resultado        dominiovec.ResultadoContextoActorRegistradoV2
}

func prepararAutorizacionCuadroRRHH(
	contexto ContextoConsultaRRHH,
	solicitud SolicitudCuadroRRHH,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	instante time.Time,
) (preparacionAutorizacionConsultaRRHH, error) {
	recursos, err := NuevosRecursosConsultaCuadroRRHH(
		contexto, solicitud, instante,
	)
	if err != nil {
		return preparacionAutorizacionConsultaRRHH{},
			ErrCapacidadConsultaRRHHInvalida
	}
	return nuevaPreparacionAutorizacionConsultaRRHH(
		clasePreparacionAutorizacionCuadroRRHH,
		contexto,
		solicitud,
		SolicitudDetalleRRHH{},
		recursos,
		motivo,
		correlacion,
		instante,
	)
}

func prepararAutorizacionDetalleRRHH(
	contexto ContextoConsultaRRHH,
	solicitud SolicitudDetalleRRHH,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	instante time.Time,
) (preparacionAutorizacionConsultaRRHH, error) {
	recursos, err := NuevosRecursosConsultaDetalleRRHH(
		contexto, solicitud, instante,
	)
	if err != nil {
		return preparacionAutorizacionConsultaRRHH{},
			ErrCapacidadConsultaRRHHInvalida
	}
	return nuevaPreparacionAutorizacionConsultaRRHH(
		clasePreparacionAutorizacionDetalleRRHH,
		contexto,
		SolicitudCuadroRRHH{},
		solicitud,
		recursos,
		motivo,
		correlacion,
		instante,
	)
}

func nuevaPreparacionAutorizacionConsultaRRHH(
	clase clasePreparacionAutorizacionConsultaRRHH,
	contexto ContextoConsultaRRHH,
	solicitudCuadro SolicitudCuadroRRHH,
	solicitudDetalle SolicitudDetalleRRHH,
	recursos RecursosConsultaRRHH,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	instante time.Time,
) (preparacionAutorizacionConsultaRRHH, error) {
	vacia := preparacionAutorizacionConsultaRRHH{}
	if contexto.validarEn(instante) != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) ||
		correlacion.Validar() != nil {
		return vacia, ErrCapacidadConsultaRRHHInvalida
	}
	resultado, err := contexto.autoridad.Resultado.Clonar()
	if err != nil || contexto.autoridad.Vinculo.ValidarPara(resultado) != nil {
		return vacia, ErrCapacidadConsultaRRHHInvalida
	}
	contextoRetenido := contexto
	contextoRetenido.autoridad = &ContextoAutorizacionAltaV3{
		Vinculo: contexto.autoridad.Vinculo, Resultado: resultado,
	}
	accion, finalidad, err := parametrosCerradosPreparacionConsultaRRHH(
		clase, contextoRetenido, solicitudCuadro, solicitudDetalle,
		recursos, instante,
	)
	if err != nil {
		return vacia, ErrCapacidadConsultaRRHHInvalida
	}
	solicitudVEC, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contextoRetenido.autoridad.Vinculo,
			ReferenciaMotivo:          motivo,
			Accion:                    accion,
			Recurso:                   recursos.recurso,
			Finalidad:                 finalidad,
			Correlacion:               correlacion,
		},
	)
	if err != nil {
		return vacia, ErrCapacidadConsultaRRHHInvalida
	}
	preparacion := preparacionAutorizacionConsultaRRHH{
		clase: clase, contexto: contextoRetenido,
		solicitudCuadro: solicitudCuadro, solicitudDetalle: solicitudDetalle,
		recursos: recursos, motivo: motivo, correlacion: correlacion,
		solicitudVEC: solicitudVEC, resultado: resultado,
	}
	if preparacion.validarEn(instante) != nil {
		return vacia, ErrCapacidadConsultaRRHHInvalida
	}
	return preparacion, nil
}

func parametrosCerradosPreparacionConsultaRRHH(
	clase clasePreparacionAutorizacionConsultaRRHH,
	contexto ContextoConsultaRRHH,
	solicitudCuadro SolicitudCuadroRRHH,
	solicitudDetalle SolicitudDetalleRRHH,
	recursos RecursosConsultaRRHH,
	instante time.Time,
) (string, string, error) {
	switch clase {
	case clasePreparacionAutorizacionCuadroRRHH:
		if solicitudDetalle != (SolicitudDetalleRRHH{}) ||
			recursos.validarParaCuadro(contexto, solicitudCuadro, instante) != nil {
			return "", "", ErrCapacidadConsultaRRHHInvalida
		}
		return AccionConsultarCuadroRRHH, FinalidadConsultarCuadroRRHH, nil
	case clasePreparacionAutorizacionDetalleRRHH:
		if solicitudCuadro != (SolicitudCuadroRRHH{}) ||
			recursos.validarParaDetalle(contexto, solicitudDetalle, instante) != nil {
			return "", "", ErrCapacidadConsultaRRHHInvalida
		}
		return AccionConsultarDetalleRRHH, FinalidadConsultarDetalleRRHH, nil
	default:
		return "", "", ErrCapacidadConsultaRRHHInvalida
	}
}

func (p preparacionAutorizacionConsultaRRHH) validarEn(instante time.Time) error {
	if p.contexto.validarEn(instante) != nil ||
		p.resultado.Validar() != nil ||
		p.contexto.autoridad.Vinculo.ValidarPara(p.resultado) != nil ||
		!reflect.DeepEqual(p.contexto.autoridad.Resultado, p.resultado) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(p.motivo) ||
		p.correlacion.Validar() != nil {
		return ErrCapacidadConsultaRRHHInvalida
	}
	accion, finalidad, err := parametrosCerradosPreparacionConsultaRRHH(
		p.clase, p.contexto, p.solicitudCuadro, p.solicitudDetalle,
		p.recursos, instante,
	)
	datos, errSolicitud := p.solicitudVEC.Datos()
	correlacion, errCorrelacion := p.correlacion.ValorCanonico()
	correlacionVEC, errCorrelacionVEC := datos.Correlacion.ValorCanonico()
	if err != nil || errSolicitud != nil || errCorrelacion != nil ||
		errCorrelacionVEC != nil ||
		!datos.VinculoAutenticacionActor.CoincideExactamenteCon(
			p.contexto.autoridad.Vinculo,
		) ||
		datos.VinculoAutenticacionActor.ValidarPara(p.resultado) != nil ||
		datos.ReferenciaMotivo != p.motivo ||
		datos.Accion != accion || datos.Finalidad != finalidad ||
		correlacionVEC != correlacion ||
		!reflect.DeepEqual(datos.Recurso, p.recursos.recurso) {
		return ErrCapacidadConsultaRRHHInvalida
	}
	return nil
}
