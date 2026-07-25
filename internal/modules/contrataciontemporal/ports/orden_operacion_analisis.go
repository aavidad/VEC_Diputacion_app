package ports

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrOrdenOperacionAnalisisInvalida = errors.New(
		"contratacion temporal: orden de operacion de analisis invalida",
	)
	ErrPersistenciaOperacionAnalisisNoDisponible = errors.New(
		"contratacion temporal: persistencia de operacion de analisis no disponible",
	)
)

const (
	AtributoOperacionAnalisis       = "operacion"
	AtributoVersionAnalisis         = "version_expediente_esperada"
	AtributoPoliticaAnalisisRef     = "politica_ref"
	AtributoPoliticaAnalisisVersion = "politica_version"
	AtributoPoliticaAnalisisHuella  = "politica_huella_sha256"
	AtributoArtefactoAnalisisRef    = "artefacto_analisis_ref"
	AtributoArtefactoAnalisisHuella = "artefacto_analisis_huella_sha256"
	AtributoAnalisisDerivadoHuella  = "analisis_derivado_huella_sha256"
	AtributoConjuntoFuentesHuella   = "conjunto_fuentes_huella_sha256"
	AtributoUnidadPoliticaRef       = "unidad_politica_ref"
	AtributoMotivoRectificacion     = "motivo_rectificacion_clave"
	AtributoHuellaSemanticaAnalisis = "huella_semantica_hmac"
	AtributoSegregacionAnalisis     = "exige_actor_distinto"
	// ValorMotivoRectificacionNoAplica conserva la presencia exacta del
	// atributo sin debilitar VEC con valores vacíos.
	ValorMotivoRectificacionNoAplica = "no_aplica"
)

type DatosOrdenConfirmarOperacionAnalisis struct {
	bloqueoSerializacionOperacionAnalisis
	SolicitudContexto    SolicitudResolverContextoAutorizacionAltaV3
	ContextoAutorizacion ContextoAutorizacionAltaV3
	SolicitudArtefacto   SolicitudPrepararArtefactoAnalisis
	Artefacto            ArtefactoAnalisisPreparado
	OrdenConsumoFuentes  OrdenConsumoConjuntoFuentesAnalisisO3
	SolicitudPreparacion SolicitudPrepararOperacionAnalisis
	Preparacion          PreparacionOperacionAnalisis
	SolicitudPolitica    SolicitudResolverPoliticaOperacionAnalisis
	Politica             PoliticaOperacionAnalisis
	SolicitudV3          dominiovec.SolicitudAutorizacionLigadaV3
	DecisionV3           dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionV3       puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	InstanteEfecto       time.Time
	ExpedienteSiguiente  domain.Expediente
}

type OrdenConfirmarOperacionAnalisis struct {
	bloqueoSerializacionOperacionAnalisis
	datos *datosOrdenConfirmarOperacionAnalisis
}

type datosOrdenConfirmarOperacionAnalisis struct {
	solicitudContexto    SolicitudResolverContextoAutorizacionAltaV3
	contextoAutorizacion ContextoAutorizacionAltaV3
	solicitudArtefacto   SolicitudPrepararArtefactoAnalisis
	artefacto            ArtefactoAnalisisPreparado
	ordenConsumoFuentes  OrdenConsumoConjuntoFuentesAnalisisO3
	solicitudPreparacion SolicitudPrepararOperacionAnalisis
	preparacion          PreparacionOperacionAnalisis
	solicitudPolitica    SolicitudResolverPoliticaOperacionAnalisis
	politica             PoliticaOperacionAnalisis
	solicitudV3          dominiovec.SolicitudAutorizacionLigadaV3
	decisionV3           dominiovec.DecisionAutorizacionLigadaV3
	confirmacionV3       puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	instanteEfecto       time.Time
	expedienteAnterior   domain.Expediente
	expedienteSiguiente  domain.Expediente
}

type EvidenciaOrdenConfirmarOperacionAnalisis struct {
	bloqueoSerializacionOperacionAnalisis
	SolicitudContexto    SolicitudResolverContextoAutorizacionAltaV3
	ContextoAutorizacion ContextoAutorizacionAltaV3
	SolicitudArtefacto   SolicitudPrepararArtefactoAnalisis
	Artefacto            ArtefactoAnalisisPreparado
	OrdenConsumoFuentes  OrdenConsumoConjuntoFuentesAnalisisO3
	SolicitudPreparacion SolicitudPrepararOperacionAnalisis
	Preparacion          PreparacionOperacionAnalisis
	SolicitudPolitica    SolicitudResolverPoliticaOperacionAnalisis
	Politica             PoliticaOperacionAnalisis
	SolicitudV3          dominiovec.SolicitudAutorizacionLigadaV3
	DecisionV3           dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionV3       puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	InstanteEfecto       time.Time
	ExpedienteAnterior   domain.Expediente
	ExpedienteSiguiente  domain.Expediente
}

func NuevaOrdenConfirmarOperacionAnalisis(
	datos DatosOrdenConfirmarOperacionAnalisis,
) (OrdenConfirmarOperacionAnalisis, error) {
	preparacion, err := datos.Preparacion.DatosPara(
		datos.SolicitudPreparacion,
	)
	artefacto, errArtefacto := datos.Artefacto.DatosPara(
		datos.SolicitudArtefacto,
	)
	pruebas, errPruebas := datos.Artefacto.PruebasParaO3(
		datos.SolicitudArtefacto,
	)
	ordenEsperada, errOrdenEsperada :=
		pruebas.OrdenConsumoConjunto.Datos()
	ordenRecibida, errOrdenRecibida :=
		datos.OrdenConsumoFuentes.Datos()
	if err != nil || errArtefacto != nil || errPruebas != nil ||
		errOrdenEsperada != nil || errOrdenRecibida != nil ||
		!reflect.DeepEqual(ordenEsperada, ordenRecibida) ||
		preparacion.Estado != PreparacionOperacionAnalisisReservada ||
		preparacion.ExpedienteAnterior == nil ||
		validarCoordenadasOrdenOperacionAnalisis(
			datos,
			preparacion,
			artefacto,
		) != nil {
		return OrdenConfirmarOperacionAnalisis{},
			ErrOrdenOperacionAnalisisInvalida
	}
	anterior := preparacion.ExpedienteAnterior.Clonar()
	return OrdenConfirmarOperacionAnalisis{
		datos: &datosOrdenConfirmarOperacionAnalisis{
			solicitudContexto:    datos.SolicitudContexto,
			contextoAutorizacion: datos.ContextoAutorizacion,
			solicitudArtefacto:   datos.SolicitudArtefacto,
			artefacto:            datos.Artefacto,
			ordenConsumoFuentes:  datos.OrdenConsumoFuentes,
			solicitudPreparacion: datos.SolicitudPreparacion,
			preparacion:          datos.Preparacion,
			solicitudPolitica:    datos.SolicitudPolitica,
			politica:             datos.Politica,
			solicitudV3:          datos.SolicitudV3,
			decisionV3:           datos.DecisionV3,
			confirmacionV3:       datos.ConfirmacionV3,
			instanteEfecto:       datos.InstanteEfecto,
			expedienteAnterior:   anterior,
			expedienteSiguiente:  datos.ExpedienteSiguiente.Clonar(),
		},
	}, nil
}

func validarCoordenadasOrdenOperacionAnalisis(
	datos DatosOrdenConfirmarOperacionAnalisis,
	preparacion DatosPreparacionOperacionAnalisis,
	artefacto DatosArtefactoAnalisis,
) error {
	anterior := preparacion.ExpedienteAnterior
	if anterior == nil ||
		!expedienteOperacionAnalisisSeguro(*anterior) ||
		anterior.ViaCobertura != nil || anterior.Asignacion != nil ||
		!instanteSeguroOperacionAnalisis(datos.InstanteEfecto) ||
		datos.InstanteEfecto.Before(anterior.ActualizadoEn) ||
		artefacto.PreparadoEn.After(datos.InstanteEfecto) ||
		datos.ContextoAutorizacion.ValidarPara(
			datos.SolicitudContexto,
			datos.InstanteEfecto,
		) != nil ||
		datos.Artefacto.ValidarVigenciaEn(
			datos.SolicitudArtefacto,
			datos.InstanteEfecto,
		) != nil ||
		datos.Politica.ValidarPara(datos.SolicitudPolitica) != nil ||
		preparacion.ArtefactoRef != artefacto.ArtefactoRef ||
		preparacion.ArtefactoHuellaSHA256 !=
			artefacto.ArtefactoHuellaSHA256 ||
		datos.SolicitudPolitica.FasePrevia != anterior.FaseActual ||
		datos.SolicitudPolitica.EstadoPrevio != anterior.EstadoActual ||
		datos.SolicitudPolitica.ArtefactoRef != artefacto.ArtefactoRef ||
		datos.SolicitudPolitica.ArtefactoHuellaSHA256 !=
			artefacto.ArtefactoHuellaSHA256 {
		return ErrOrdenOperacionAnalisisInvalida
	}
	actorAnterior, err := actorAnalisisAnteriorParaOperacion(
		*anterior,
		preparacion.Operacion,
	)
	if err != nil ||
		datos.SolicitudPolitica.ActorAnalisisAnteriorRef != actorAnterior ||
		(preparacion.Operacion == OperacionRectificarAnalisis &&
			datos.Politica.ActorRef == actorAnterior) {
		return ErrOrdenOperacionAnalisisInvalida
	}
	if validarAutorizacionOrdenOperacionAnalisisEn(
		datos,
		preparacion,
		artefacto,
		datos.InstanteEfecto,
	) != nil {
		return ErrOrdenOperacionAnalisisInvalida
	}
	esperado, err := reproducirOperacionAnalisis(
		datos,
		*anterior,
	)
	if err != nil || !expedienteOperacionAnalisisSeguro(esperado) ||
		!reflect.DeepEqual(esperado, datos.ExpedienteSiguiente) {
		return ErrOrdenOperacionAnalisisInvalida
	}
	return nil
}

func reproducirOperacionAnalisis(
	datos DatosOrdenConfirmarOperacionAnalisis,
	anterior domain.Expediente,
) (domain.Expediente, error) {
	analisis, err := DerivarAnalisisDesdeArtefacto(
		datos.SolicitudArtefacto,
		datos.Artefacto,
	)
	if err != nil {
		return domain.Expediente{}, ErrOrdenOperacionAnalisisInvalida
	}
	preparacion, err := datos.Preparacion.DatosPara(
		datos.SolicitudPreparacion,
	)
	if err != nil {
		return domain.Expediente{}, ErrOrdenOperacionAnalisisInvalida
	}
	actuacion := domain.DatosActuacion{
		AccionClave:   datos.Politica.Accion,
		ActorRef:      datos.Politica.ActorRef,
		UnidadRef:     datos.Politica.UnidadRef,
		ReciboRef:     preparacion.ReciboRef,
		RealizadaEn:   datos.InstanteEfecto,
		FaseDestino:   anterior.FaseActual,
		EstadoDestino: anterior.EstadoActual,
	}
	if preparacion.Operacion == OperacionRegistrarAnalisis {
		return anterior.RegistrarAnalisis(
			preparacion.VersionExpediente,
			analisis,
			actuacion,
		)
	}
	actuacion.Observaciones =
		string(datos.Politica.MotivoRectificacion.ClaveMensajeI18N)
	return anterior.RectificarAnalisis(
		preparacion.VersionExpediente,
		analisis,
		actuacion,
	)
}

func validarAutorizacionOrdenOperacionAnalisisEn(
	datos DatosOrdenConfirmarOperacionAnalisis,
	preparacion DatosPreparacionOperacionAnalisis,
	artefacto DatosArtefactoAnalisis,
	comprobadaEn time.Time,
) error {
	solicitudV3, err := datos.SolicitudV3.Datos()
	vinculo, errVinculo := solicitudV3.VinculoAutenticacionActor.Datos()
	vinculoContexto, errVinculoContexto :=
		datos.ContextoAutorizacion.Vinculo.Datos()
	concedida, _, errDecision := datos.DecisionV3.Resultado()
	huellaDecision, errHuella :=
		dominiovec.HuellaSHA256DecisionAutorizacionV3(datos.DecisionV3)
	confirmacion, errConfirmacion := datos.ConfirmacionV3.Datos()
	if err != nil || errVinculo != nil || errVinculoContexto != nil ||
		errDecision != nil ||
		errHuella != nil || errConfirmacion != nil || !concedida ||
		!instanteSeguroOperacionAnalisis(comprobadaEn) ||
		datos.ContextoAutorizacion.ValidarPara(
			datos.SolicitudContexto,
			comprobadaEn,
		) != nil ||
		datos.DecisionV3.ValidarPara(datos.SolicitudV3) != nil ||
		!reflect.DeepEqual(vinculo, vinculoContexto) ||
		vinculo.PrincipalID != preparacion.ActorRef ||
		vinculo.PerfilActivoRef != preparacion.PerfilRef ||
		solicitudV3.ReferenciaMotivo !=
			datos.Politica.MotivoAutorizacion ||
		solicitudV3.Accion != string(datos.Politica.Accion) ||
		solicitudV3.Finalidad != string(datos.Politica.Finalidad) ||
		!recursoAutorizacionOperacionAnalisisValido(
			solicitudV3.Recurso,
			preparacion,
			artefacto,
			datos.OrdenConsumoFuentes,
			datos.Politica,
			datos.SolicitudPolitica.MotivoRectificacionClave,
		) ||
		confirmacion.DecisionHuellaSHA256 != huellaDecision ||
		!datos.ConfirmacionV3.DentroDeVentanaEn(comprobadaEn) {
		return ErrOrdenOperacionAnalisisInvalida
	}
	return nil
}

func recursoAutorizacionOperacionAnalisisValido(
	recurso dominiovec.RecursoAutorizable,
	preparacion DatosPreparacionOperacionAnalisis,
	artefacto DatosArtefactoAnalisis,
	ordenFuentes OrdenConsumoConjuntoFuentesAnalisisO3,
	politica PoliticaOperacionAnalisis,
	motivoRectificacion domain.ClaveCatalogo,
) bool {
	huellaAnalisis, err := huellaAnalisisDerivadoDesdeDatosO3(artefacto)
	fuentes, errFuentes := ordenFuentes.Datos()
	motivoEsperado := string(motivoRectificacion)
	if motivoEsperado == "" {
		motivoEsperado = ValorMotivoRectificacionNoAplica
	}
	ambitosEsperados := map[string]string{
		"organizacion_ref": preparacion.OrganizacionRef,
		"expediente_ref":   preparacion.ExpedienteRef,
		"fase_previa":      string(politica.FasePrevia),
		"estado_previo":    string(politica.EstadoPrevio),
	}
	atributosEsperados := map[string]string{
		AtributoOperacionAnalisis:       string(preparacion.Operacion),
		AtributoVersionAnalisis:         strconv.FormatUint(preparacion.VersionExpediente, 10),
		AtributoPoliticaAnalisisRef:     politica.DefinicionRef,
		AtributoPoliticaAnalisisVersion: strconv.FormatUint(politica.Version, 10),
		AtributoPoliticaAnalisisHuella:  politica.HuellaSHA256,
		AtributoArtefactoAnalisisRef:    artefacto.ArtefactoRef,
		AtributoArtefactoAnalisisHuella: artefacto.ArtefactoHuellaSHA256,
		AtributoAnalisisDerivadoHuella:  huellaAnalisis,
		AtributoConjuntoFuentesHuella:   fuentes.HuellaSHA256,
		AtributoUnidadPoliticaRef:       politica.UnidadRef,
		AtributoMotivoRectificacion:     motivoEsperado,
		AtributoHuellaSemanticaAnalisis: preparacion.HuellaSemanticaHMAC,
		AtributoSegregacionAnalisis:     strconv.FormatBool(politica.ExigeActorDistinto),
	}
	return recurso.Referencia == preparacion.ExpedienteRef &&
		err == nil && errFuentes == nil &&
		recurso.ModuloID == ModuloContratacion &&
		recurso.Tipo == TipoRecursoAnalisis &&
		reflect.DeepEqual(recurso.Ambitos, ambitosEsperados) &&
		reflect.DeepEqual(recurso.Atributos, atributosEsperados)
}

func actorAnalisisAnteriorParaOperacion(
	expediente domain.Expediente,
	operacion TipoOperacionAnalisis,
) (string, error) {
	if operacion == OperacionRegistrarAnalisis {
		if expediente.Analisis != nil {
			return "", ErrOrdenOperacionAnalisisInvalida
		}
		return "", nil
	}
	if expediente.Analisis == nil ||
		expediente.Analisis.ActuacionRegistro == nil {
		return "", ErrOrdenOperacionAnalisisInvalida
	}
	secuencia := expediente.Analisis.ActuacionRegistro.Secuencia
	if secuencia == 0 || secuencia > uint64(len(expediente.Actuaciones)) {
		return "", ErrOrdenOperacionAnalisisInvalida
	}
	actuacion := expediente.Actuaciones[secuencia-1]
	if actuacion.Secuencia != secuencia ||
		actuacion.VersionExpediente !=
			expediente.Analisis.ActuacionRegistro.VersionExpediente ||
		!domain.ReferenciaOpacaValida(actuacion.ActorRef) {
		return "", ErrOrdenOperacionAnalisisInvalida
	}
	return actuacion.ActorRef, nil
}

func expedienteOperacionAnalisisSeguro(expediente domain.Expediente) bool {
	if expediente.Validar() != nil ||
		!VersionOperacionAnalisisValida(expediente.Version) ||
		!VersionOperacionAnalisisValida(expediente.Flujo.Version) {
		return false
	}
	for _, actuacion := range expediente.Actuaciones {
		if !VersionOperacionAnalisisValida(actuacion.Secuencia) ||
			!VersionOperacionAnalisisValida(actuacion.VersionExpediente) {
			return false
		}
	}
	return true
}

func (o OrdenConfirmarOperacionAnalisis) Datos() (
	EvidenciaOrdenConfirmarOperacionAnalisis,
	error,
) {
	if o.datos == nil {
		return EvidenciaOrdenConfirmarOperacionAnalisis{},
			ErrOrdenOperacionAnalisisInvalida
	}
	entrada := DatosOrdenConfirmarOperacionAnalisis{
		SolicitudContexto:    o.datos.solicitudContexto,
		ContextoAutorizacion: o.datos.contextoAutorizacion,
		SolicitudArtefacto:   o.datos.solicitudArtefacto,
		Artefacto:            o.datos.artefacto,
		OrdenConsumoFuentes:  o.datos.ordenConsumoFuentes,
		SolicitudPreparacion: o.datos.solicitudPreparacion,
		Preparacion:          o.datos.preparacion,
		SolicitudPolitica:    o.datos.solicitudPolitica,
		Politica:             o.datos.politica,
		SolicitudV3:          o.datos.solicitudV3,
		DecisionV3:           o.datos.decisionV3,
		ConfirmacionV3:       o.datos.confirmacionV3,
		InstanteEfecto:       o.datos.instanteEfecto,
		ExpedienteSiguiente:  o.datos.expedienteSiguiente.Clonar(),
	}
	if _, err := NuevaOrdenConfirmarOperacionAnalisis(entrada); err != nil {
		return EvidenciaOrdenConfirmarOperacionAnalisis{},
			ErrOrdenOperacionAnalisisInvalida
	}
	return EvidenciaOrdenConfirmarOperacionAnalisis{
		SolicitudContexto:    entrada.SolicitudContexto,
		ContextoAutorizacion: entrada.ContextoAutorizacion,
		SolicitudArtefacto:   entrada.SolicitudArtefacto,
		Artefacto:            entrada.Artefacto,
		OrdenConsumoFuentes:  entrada.OrdenConsumoFuentes,
		SolicitudPreparacion: entrada.SolicitudPreparacion,
		Preparacion:          entrada.Preparacion,
		SolicitudPolitica:    entrada.SolicitudPolitica,
		Politica:             entrada.Politica,
		SolicitudV3:          entrada.SolicitudV3,
		DecisionV3:           entrada.DecisionV3,
		ConfirmacionV3:       entrada.ConfirmacionV3,
		InstanteEfecto:       entrada.InstanteEfecto,
		ExpedienteAnterior:   o.datos.expedienteAnterior.Clonar(),
		ExpedienteSiguiente:  entrada.ExpedienteSiguiente,
	}, nil
}

// ValidarConfirmacionDentroDeTransaccion debe invocarse con el reloj
// autoritativo de la base de datos después de adquirir los bloqueos y antes de
// escribir ningún efecto. No sustituye el COMMIT: cerca contexto, fuentes y
// concesión V3 en el instante exacto que aparecerá en el recibo.
func (o OrdenConfirmarOperacionAnalisis) ValidarConfirmacionDentroDeTransaccion(
	confirmadaEn time.Time,
) error {
	evidencia, err := o.Datos()
	if err != nil || !instanteSeguroOperacionAnalisis(confirmadaEn) ||
		confirmadaEn.Before(evidencia.InstanteEfecto) ||
		evidencia.ContextoAutorizacion.ValidarPara(
			evidencia.SolicitudContexto,
			confirmadaEn,
		) != nil ||
		evidencia.Artefacto.ValidarVigenciaEn(
			evidencia.SolicitudArtefacto,
			confirmadaEn,
		) != nil {
		return ErrOrdenOperacionAnalisisInvalida
	}
	preparacion, err := evidencia.Preparacion.DatosPara(
		evidencia.SolicitudPreparacion,
	)
	artefacto, errArtefacto := evidencia.Artefacto.DatosPara(
		evidencia.SolicitudArtefacto,
	)
	entrada := DatosOrdenConfirmarOperacionAnalisis{
		SolicitudContexto:    evidencia.SolicitudContexto,
		ContextoAutorizacion: evidencia.ContextoAutorizacion,
		SolicitudArtefacto:   evidencia.SolicitudArtefacto,
		Artefacto:            evidencia.Artefacto,
		OrdenConsumoFuentes:  evidencia.OrdenConsumoFuentes,
		SolicitudPreparacion: evidencia.SolicitudPreparacion,
		Preparacion:          evidencia.Preparacion,
		SolicitudPolitica:    evidencia.SolicitudPolitica,
		Politica:             evidencia.Politica,
		SolicitudV3:          evidencia.SolicitudV3,
		DecisionV3:           evidencia.DecisionV3,
		ConfirmacionV3:       evidencia.ConfirmacionV3,
		InstanteEfecto:       evidencia.InstanteEfecto,
		ExpedienteSiguiente:  evidencia.ExpedienteSiguiente,
	}
	if err != nil || errArtefacto != nil ||
		validarAutorizacionOrdenOperacionAnalisisEn(
			entrada,
			preparacion,
			artefacto,
			confirmadaEn,
		) != nil {
		return ErrOrdenOperacionAnalisisInvalida
	}
	return nil
}

// El adaptador O3-04 productivo posee la única frontera de efectos. En una
// sola transacción debe bloquear y revalidar reserva, idempotencia, CAS,
// política y tiempo; consumir conjuntamente RC+coste y la concesión V3;
// persistir agregado, historia append-only, auditoría, recibo y outbox; validar
// el recibo antes del COMMIT; y hacer rollback total ante cualquier fallo. Un
// replay exacto devuelve el mismo recibo. No existe un puerto de consumo
// previo ni una compensación entre puertos.
type TransaccionOperacionesAnalisis interface {
	ConfirmarOperacionAnalisis(
		context.Context,
		OrdenConfirmarOperacionAnalisis,
	) (ReciboOperacionAnalisis, error)
}
