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
	AtributoHuellaSemanticaAnalisis = "huella_semantica_hmac"
	AtributoSegregacionAnalisis     = "exige_actor_distinto"
)

type DatosOrdenConfirmarOperacionAnalisis struct {
	bloqueoSerializacionOperacionAnalisis
	SolicitudArtefacto   SolicitudPrepararArtefactoAnalisis
	Artefacto            ArtefactoAnalisisPreparado
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
	solicitudArtefacto   SolicitudPrepararArtefactoAnalisis
	artefacto            ArtefactoAnalisisPreparado
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
	SolicitudArtefacto   SolicitudPrepararArtefactoAnalisis
	Artefacto            ArtefactoAnalisisPreparado
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
	_, errPruebas := datos.Artefacto.PruebasParaO3(
		datos.SolicitudArtefacto,
	)
	if err != nil || errArtefacto != nil || errPruebas != nil ||
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
			solicitudArtefacto:   datos.SolicitudArtefacto,
			artefacto:            datos.Artefacto,
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
	if validarAutorizacionOrdenOperacionAnalisis(
		datos,
		preparacion,
		artefacto,
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

func validarAutorizacionOrdenOperacionAnalisis(
	datos DatosOrdenConfirmarOperacionAnalisis,
	preparacion DatosPreparacionOperacionAnalisis,
	artefacto DatosArtefactoAnalisis,
) error {
	solicitudV3, err := datos.SolicitudV3.Datos()
	vinculo, errVinculo := solicitudV3.VinculoAutenticacionActor.Datos()
	concedida, _, errDecision := datos.DecisionV3.Resultado()
	huellaDecision, errHuella :=
		dominiovec.HuellaSHA256DecisionAutorizacionV3(datos.DecisionV3)
	confirmacion, errConfirmacion := datos.ConfirmacionV3.Datos()
	if err != nil || errVinculo != nil || errDecision != nil ||
		errHuella != nil || errConfirmacion != nil || !concedida ||
		datos.DecisionV3.ValidarPara(datos.SolicitudV3) != nil ||
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
			datos.Politica,
		) ||
		confirmacion.DecisionHuellaSHA256 != huellaDecision ||
		!datos.ConfirmacionV3.DentroDeVentanaEn(datos.InstanteEfecto) {
		return ErrOrdenOperacionAnalisisInvalida
	}
	return nil
}

func recursoAutorizacionOperacionAnalisisValido(
	recurso dominiovec.RecursoAutorizable,
	preparacion DatosPreparacionOperacionAnalisis,
	artefacto DatosArtefactoAnalisis,
	politica PoliticaOperacionAnalisis,
) bool {
	return recurso.Referencia == preparacion.ExpedienteRef &&
		recurso.ModuloID == ModuloContratacion &&
		recurso.Tipo == TipoRecursoAnalisis &&
		len(recurso.Ambitos) == 4 && len(recurso.Atributos) == 9 &&
		recurso.Ambitos["organizacion_ref"] == preparacion.OrganizacionRef &&
		recurso.Ambitos["expediente_ref"] == preparacion.ExpedienteRef &&
		recurso.Ambitos["fase_previa"] == string(politica.FasePrevia) &&
		recurso.Ambitos["estado_previo"] == string(politica.EstadoPrevio) &&
		recurso.Atributos[AtributoOperacionAnalisis] ==
			string(preparacion.Operacion) &&
		recurso.Atributos[AtributoVersionAnalisis] ==
			strconv.FormatUint(preparacion.VersionExpediente, 10) &&
		recurso.Atributos[AtributoPoliticaAnalisisRef] ==
			politica.DefinicionRef &&
		recurso.Atributos[AtributoPoliticaAnalisisVersion] ==
			strconv.FormatUint(politica.Version, 10) &&
		recurso.Atributos[AtributoPoliticaAnalisisHuella] ==
			politica.HuellaSHA256 &&
		recurso.Atributos[AtributoArtefactoAnalisisRef] ==
			artefacto.ArtefactoRef &&
		recurso.Atributos[AtributoArtefactoAnalisisHuella] ==
			artefacto.ArtefactoHuellaSHA256 &&
		recurso.Atributos[AtributoHuellaSemanticaAnalisis] ==
			preparacion.HuellaSemanticaHMAC &&
		recurso.Atributos[AtributoSegregacionAnalisis] ==
			strconv.FormatBool(politica.ExigeActorDistinto)
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
		SolicitudArtefacto:   o.datos.solicitudArtefacto,
		Artefacto:            o.datos.artefacto,
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
		SolicitudArtefacto:   entrada.SolicitudArtefacto,
		Artefacto:            entrada.Artefacto,
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

// El adaptador productivo debe ejecutar en una sola transacción: CAS del
// agregado y política, consumo único del artefacto O3-03 y de la concesión V3,
// confirmación idempotente, agregado, historial append-only, auditoría, recibo
// y outbox. El puerto no prueba por sí mismo durabilidad.
type TransaccionOperacionesAnalisis interface {
	ConfirmarOperacionAnalisis(
		context.Context,
		OrdenConfirmarOperacionAnalisis,
	) (ReciboOperacionAnalisis, error)
}
