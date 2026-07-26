package cobertura

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	moduloRecursoOperacionDecisionCobertura = "contratacion_temporal"
	tipoRecursoOperacionDecisionCobertura   = "decision_cobertura_gobernada"
)

var ErrOrdenOperacionDecisionCoberturaInvalida = errors.New(
	"contratacion temporal: orden de operacion de decision de cobertura invalida",
)

// PreparacionOrdenOperacionDecisionCobertura es la frontera pura anterior a
// VEC. Solo puede nacer de capacidades servidor ya resueltas y permite
// construir la solicitud VEC sin aceptar atributos libres del canal.
//
// No ejecuta I/O, no consume evidencias y no confirma ningún efecto.
type PreparacionOrdenOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosPreparacionOrdenOperacionDecisionCobertura
}

type datosPreparacionOrdenOperacionDecisionCobertura struct {
	solicitudReserva  SolicitudReservarOperacionDecisionCobertura
	reserva           DatosReservaPropietariaOperacionDecisionCobertura
	solicitudGobierno SolicitudGobiernoOperacionCobertura
	gobierno          GobiernoOperacionCobertura
	datosGobierno     DatosGobiernoOperacionCobertura
	preparacionC1     PreparacionConjuntosViasCobertura
	propuesta         domain.PropuestaDecisionCobertura
	motivo            ResolucionMotivoDecisionCobertura
	recursoVEC        dominiovec.RecursoAutorizable
	preparadaEn       time.Time
	validaHasta       time.Time
}

// OrdenOperacionDecisionCobertura es la orden nominal final que O4-04 podrá
// confirmar. Contiene una candidata VEC, nunca una autorización durable.
// SERIALIZABLE, bloqueos, CAS, consumo y terminalidad pertenecen a O4-04.
type OrdenOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosOrdenOperacionDecisionCobertura
}

type datosOrdenOperacionDecisionCobertura struct {
	concedida  bool
	concesion  *datosOrdenConcedidaOperacionDecisionCobertura
	denegacion *datosOrdenDenegadaOperacionDecisionCobertura
}

type datosOrdenConcedidaOperacionDecisionCobertura struct {
	preparacion       *datosPreparacionOrdenOperacionDecisionCobertura
	candidata         puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3
	resumen           puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3
	agregadoSiguiente domain.Expediente
	efectoEn          time.Time
	validaHasta       time.Time
}

// La rama denegada no conserva preparación C1, órdenes de consumo, propuesta,
// motivo funcional ni agregado C2. Solo transporta lo imprescindible para
// cerrar y auditar la reserva propietaria en O4-04.
type datosOrdenDenegadaOperacionDecisionCobertura struct {
	prueba      pruebaDenegacionOperacionDecisionCobertura
	candidata   puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3
	resumen     puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3
	validaHasta time.Time
}

// reservaMinimaOperacionDecisionCobertura conserva la ligadura propietaria
// completa sin agregado ni material C1/C2. O4-04 revalidará la fila durable.
type reservaMinimaOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	solicitud               SolicitudReservarOperacionDecisionCobertura
	organizacionRef         string
	expedienteRef           string
	versionExpediente       uint64
	ambitoHMAC              string
	semanticaHMAC           string
	tokenSHA256             string
	reservaRef              string
	reciboRef               string
	actuacionRef            string
	auditoriaRef            string
	eventoRef               string
	correlacionVECRef       string
	decisionVECRef          string
	revisionCercadoAnterior uint64
	revisionCercado         uint64
	observadaEnDB           time.Time
	propiedadHasta          time.Time
	huellaSHA256            string
}

// pruebaDenegacionOperacionDecisionCobertura conserva una capacidad nominal
// mínima y autosellada. Liga la reserva con la identidad completa del recurso
// VEC y sus autoridades de actor, acción, finalidad, motivo y vigencia, pero
// nunca conserva C1, C2, propuesta, agregado ni órdenes de consumo.
type pruebaDenegacionOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	reserva           reservaMinimaOperacionDecisionCobertura
	recursoVEC        dominiovec.RecursoAutorizable
	actorRef          string
	perfilRef         string
	accionVEC         domain.ClaveCatalogo
	finalidadVEC      domain.ClaveCatalogo
	motivoVEC         dominiovec.ReferenciaEntradaCatalogo
	limitePreparacion time.Time
	huellaSHA256      string
}

// PrepararOrdenOperacionDecisionCobertura cruza reserva propietaria, gobierno
// servidor, preparación C1 y propuesta exacta. La transición C2 se calcula en
// memoria sobre un clon del agregado reservado.
func PrepararOrdenOperacionDecisionCobertura(
	ctx context.Context,
	reloj RelojGobiernoOperacionCobertura,
	solicitudReserva SolicitudReservarOperacionDecisionCobertura,
	preparacionReserva PreparacionOperacionDecisionCobertura,
	solicitudGobierno SolicitudGobiernoOperacionCobertura,
	gobierno GobiernoOperacionCobertura,
	preparacionC1 PreparacionConjuntosViasCobertura,
	propuesta domain.PropuestaDecisionCobertura,
	motivo ResolucionMotivoDecisionCobertura,
) (PreparacionOrdenOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(reloj) ||
		solicitudReserva.Validar() != nil {
		return PreparacionOrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	identidad, err := solicitudReserva.consulta.identidadInterna()
	reserva, errReserva := preparacionReserva.DatosPropietariaPara(
		solicitudReserva,
	)
	datosGobierno, errGobierno := gobierno.DesplegarPara(
		ctx,
		reloj,
		solicitudGobierno,
	)
	instantePreparacion, errReloj := ahoraGobiernoOperacionCobertura(ctx, reloj)
	if err != nil || errReserva != nil || errGobierno != nil ||
		errReloj != nil ||
		!coordenadasOrdenOperacionDecisionCoberturaCoinciden(
			identidad,
			reserva,
			solicitudGobierno,
			datosGobierno,
		) {
		return PreparacionOrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}

	publicacionPropuesta := propuesta.Publicacion()
	propuestaEsperada, err := propuestaExactaOperacionDecisionCobertura(
		preparacionC1,
		publicacionPropuesta.GeneradaEn,
	)
	identidadSemantica, errSemantica := propuestaEsperada.IdentidadSemantica()
	validaHastaC1, errVigenciaC1 := preparacionC1.ValidaHasta()
	if err != nil || errSemantica != nil || errVigenciaC1 != nil ||
		!propuestasOperacionDecisionCoberturaIguales(
			propuesta,
			propuestaEsperada,
		) ||
		!identidadSemantica.CoincideExactamente(
			identidad.identidadSemantica,
		) ||
		!ventanaPreparacionOrdenOperacionDecisionCoberturaValida(
			instantePreparacion,
			publicacionPropuesta.GeneradaEn,
			preparacionC1.preparadaEn,
			reserva,
			datosGobierno,
			publicacionPropuesta.ValidaHasta,
			validaHastaC1,
		) {
		return PreparacionOrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	if !gobiernoC1OperacionDecisionCoberturaCoincide(
		datosGobierno,
		publicacionPropuesta,
		identidad,
		reserva,
	) {
		return PreparacionOrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}

	motivoFuncional, err := motivoFuncionalOperacionDecisionCobertura(
		identidad,
		motivo,
		publicacionPropuesta.GeneradaEn,
		instantePreparacion,
	)
	if err != nil {
		return PreparacionOrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	_, err = transicionPuraOperacionDecisionCobertura(
		identidad,
		reserva,
		datosGobierno,
		propuesta,
		motivoFuncional,
		instantePreparacion,
	)
	if err != nil {
		return PreparacionOrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	recurso, err := recursoVECOperacionDecisionCobertura(
		identidad,
		reserva,
		datosGobierno,
		identidadSemantica,
		publicacionPropuesta,
	)
	if err != nil {
		return PreparacionOrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	validaHasta := minimoInstanteOperacionDecisionCobertura(
		reserva.PropiedadHasta,
		datosGobierno.ValidaHasta,
		publicacionPropuesta.ValidaHasta,
		validaHastaC1,
	)
	datos := &datosPreparacionOrdenOperacionDecisionCobertura{
		solicitudReserva:  solicitudReserva,
		reserva:           clonarReservaPropietariaOperacionDecisionCobertura(reserva),
		solicitudGobierno: solicitudGobierno,
		gobierno:          gobierno,
		datosGobierno:     datosGobierno,
		preparacionC1:     preparacionC1,
		propuesta:         propuesta,
		motivo:            motivo,
		recursoVEC:        clonarRecursoOperacionDecisionCobertura(recurso),
		preparadaEn:       instantePreparacion,
		validaHasta:       validaHasta,
	}
	preparacion := PreparacionOrdenOperacionDecisionCobertura{datos: datos}
	if preparacion.validar() != nil {
		return PreparacionOrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return preparacion, nil
}

// RecursoAutorizableVEC devuelve exclusivamente el recurso técnico minimizado.
// No revela actor, perfil, roles, motivos, órdenes de consumo ni candidata.
func (p PreparacionOrdenOperacionDecisionCobertura) RecursoAutorizableVEC() (
	dominiovec.RecursoAutorizable,
	error,
) {
	if p.validar() != nil {
		return dominiovec.RecursoAutorizable{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return clonarRecursoOperacionDecisionCobertura(p.datos.recursoVEC), nil
}

// NuevaOrdenOperacionDecisionCobertura compone la preparación exacta con una
// candidata VEC de concesión o denegación. No registra ninguna de las ramas.
func NuevaOrdenOperacionDecisionCobertura(
	ctx context.Context,
	reloj RelojGobiernoOperacionCobertura,
	preparacion PreparacionOrdenOperacionDecisionCobertura,
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
) (OrdenOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(reloj) ||
		preparacion.validar() != nil {
		return OrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	resumen, err := candidata.Resumen()
	if err != nil || resumen.ValidarPara(candidata) != nil ||
		validarCandidataVECOperacionDecisionCobertura(
			preparacion.datos,
			candidata,
			resumen,
		) != nil {
		return OrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	candidata, resumen, err = clonarCandidataOperacionDecisionCobertura(
		candidata,
	)
	if err != nil {
		return OrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	datosResumen, _ := resumen.Datos()
	datosGobierno, err := preparacion.datos.gobierno.DesplegarPara(
		ctx,
		reloj,
		preparacion.datos.solicitudGobierno,
	)
	instanteEfecto, errReloj := ahoraGobiernoOperacionCobertura(ctx, reloj)
	limite := minimoInstanteOperacionDecisionCobertura(
		preparacion.datos.validaHasta,
		datosResumen.ValidaHasta,
	)
	if err != nil || errReloj != nil ||
		!reflect.DeepEqual(datosGobierno, preparacion.datos.datosGobierno) ||
		instanteEfecto.Before(datosResumen.EmitidaEn) ||
		instanteEfecto.Before(preparacion.datos.preparadaEn) ||
		!instanteEfecto.Before(limite) {
		return OrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	if !datosResumen.Concedida {
		prueba, errPrueba := nuevaPruebaDenegacionOperacionDecisionCobertura(
			preparacion.datos,
		)
		if errPrueba != nil {
			return OrdenOperacionDecisionCobertura{},
				ErrOrdenOperacionDecisionCoberturaInvalida
		}
		orden := OrdenOperacionDecisionCobertura{
			datos: &datosOrdenOperacionDecisionCobertura{
				denegacion: &datosOrdenDenegadaOperacionDecisionCobertura{
					prueba:      prueba,
					candidata:   candidata,
					resumen:     resumen,
					validaHasta: limite,
				},
			},
		}
		if orden.validar() != nil {
			return OrdenOperacionDecisionCobertura{},
				ErrOrdenOperacionDecisionCoberturaInvalida
		}
		return orden, nil
	}
	identidad, _ := preparacion.datos.solicitudReserva.consulta.identidadInterna()
	motivoFuncional, err := motivoFuncionalOperacionDecisionCobertura(
		identidad,
		preparacion.datos.motivo,
		preparacion.datos.propuesta.Publicacion().GeneradaEn,
		instanteEfecto,
	)
	if err != nil {
		return OrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	if _, err = preparacion.datos.preparacionC1.validarEn(
		instanteEfecto,
	); err != nil {
		return OrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	agregadoSiguiente, err := transicionPuraOperacionDecisionCobertura(
		identidad,
		preparacion.datos.reserva,
		preparacion.datos.datosGobierno,
		preparacion.datos.propuesta,
		motivoFuncional,
		instanteEfecto,
	)
	if err != nil {
		return OrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	clonPreparacion := clonarDatosPreparacionOrdenOperacionDecisionCobertura(
		preparacion.datos,
	)
	orden := OrdenOperacionDecisionCobertura{
		datos: &datosOrdenOperacionDecisionCobertura{
			concedida: true,
			concesion: &datosOrdenConcedidaOperacionDecisionCobertura{
				preparacion:       clonPreparacion,
				candidata:         candidata,
				resumen:           resumen,
				agregadoSiguiente: agregadoSiguiente.Clonar(),
				efectoEn:          instanteEfecto,
				validaHasta:       limite,
			},
		},
	}
	if orden.validar() != nil {
		return OrdenOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return orden, nil
}

func (p PreparacionOrdenOperacionDecisionCobertura) validar() error {
	if p.datos == nil || p.datos.solicitudReserva.Validar() != nil ||
		p.datos.reserva.validarPara(p.datos.solicitudReserva) != nil ||
		p.datos.recursoVEC.Validar() != nil ||
		!instanteOperacionDecisionCoberturaValido(p.datos.preparadaEn) ||
		!instanteOperacionDecisionCoberturaValido(p.datos.validaHasta) ||
		!p.datos.validaHasta.After(p.datos.preparadaEn) {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return nil
}

func (o OrdenOperacionDecisionCobertura) validar() error {
	if o.datos == nil ||
		(o.datos.concesion == nil) == (o.datos.denegacion == nil) ||
		o.datos.concedida != (o.datos.concesion != nil) {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	if o.datos.concesion != nil {
		datos := o.datos.concesion
		if (PreparacionOrdenOperacionDecisionCobertura{
			datos: datos.preparacion,
		}).validar() != nil ||
			datos.resumen.ValidarPara(datos.candidata) != nil ||
			datos.agregadoSiguiente.Validar() != nil ||
			!instanteOperacionDecisionCoberturaValido(datos.efectoEn) ||
			!instanteOperacionDecisionCoberturaValido(datos.validaHasta) ||
			validarCandidataVECOperacionDecisionCobertura(
				datos.preparacion,
				datos.candidata,
				datos.resumen,
			) != nil ||
			validarTransicionFinalOperacionDecisionCobertura(datos) != nil {
			return ErrOrdenOperacionDecisionCoberturaInvalida
		}
		return nil
	}
	datos := o.datos.denegacion
	resumen, err := datos.resumen.Datos()
	limite := minimoInstanteOperacionDecisionCobertura(
		datos.prueba.limitePreparacion,
		resumen.ValidaHasta,
	)
	if err != nil || resumen.Concedida ||
		datos.resumen.ValidarPara(datos.candidata) != nil ||
		datos.prueba.validar() != nil ||
		!referenciasOperacionDecisionCoberturaIguales(
			datos.prueba.reserva.correlacionVECRef,
			resumenCorrelacionCandidataOperacionDecisionCobertura(
				datos.candidata,
			),
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			datos.prueba.reserva.decisionVECRef,
			resumen.DecisionRef,
		) ||
		!candidataLigaPruebaDenegacionOperacionDecisionCobertura(
			datos.candidata,
			datos.prueba,
		) ||
		!instanteOperacionDecisionCoberturaValido(datos.validaHasta) ||
		!datos.validaHasta.Equal(limite) ||
		resumen.EmitidaEn.Before(datos.prueba.reserva.observadaEnDB) ||
		!datos.validaHasta.After(resumen.EmitidaEn) {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return nil
}

func validarTransicionFinalOperacionDecisionCobertura(
	datos *datosOrdenConcedidaOperacionDecisionCobertura,
) error {
	if datos == nil || datos.preparacion == nil {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	resumen, err := datos.resumen.Datos()
	identidad, errIdentidad :=
		datos.preparacion.solicitudReserva.consulta.identidadInterna()
	motivo, errMotivo := motivoFuncionalOperacionDecisionCobertura(
		identidad,
		datos.preparacion.motivo,
		datos.preparacion.propuesta.Publicacion().GeneradaEn,
		datos.efectoEn,
	)
	if err != nil || errIdentidad != nil || errMotivo != nil {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	if _, err = datos.preparacion.preparacionC1.validarEn(
		datos.efectoEn,
	); err != nil {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	esperado, err := transicionPuraOperacionDecisionCobertura(
		identidad,
		datos.preparacion.reserva,
		datos.preparacion.datosGobierno,
		datos.preparacion.propuesta,
		motivo,
		datos.efectoEn,
	)
	validaHasta := minimoInstanteOperacionDecisionCobertura(
		datos.preparacion.validaHasta,
		resumen.ValidaHasta,
	)
	if err != nil ||
		!validaHasta.Equal(datos.validaHasta) ||
		!datos.efectoEn.Before(datos.validaHasta) ||
		datos.efectoEn.Before(resumen.EmitidaEn) ||
		!reflect.DeepEqual(
			esperado,
			datos.agregadoSiguiente,
		) {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return nil
}
