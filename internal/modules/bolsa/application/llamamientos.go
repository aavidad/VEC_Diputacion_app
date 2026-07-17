// Package application contiene casos de uso del modulo de bolsa.
package application

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// ServicioLlamamientos tiene todas sus dependencias obligatorias y privadas.
// No existe constructor reducido: omitir PDP, fuentes, motor, reloj,
// generadores o transaccion deja el servicio deliberadamente inutilizable.
type ServicioLlamamientos struct {
	resolutor   puertosbolsa.ResolutorRecursoNecesidad
	vinculador  puertosbolsa.CreadorVinculoAutenticacionActor
	autorizador puertosvec.Autorizador
	fuente      puertosbolsa.FuenteDatosLlamamiento
	motor       puertosbolsa.MotorElegibilidadLlamamiento
	reloj       puertosbolsa.RelojLlamamientos
	generador   puertosbolsa.GeneradorReferenciasLlamamiento
	transaccion puertosbolsa.TransaccionPropuestasLlamamiento
}

func NuevoServicioLlamamientos(
	resolutor puertosbolsa.ResolutorRecursoNecesidad,
	vinculador puertosbolsa.CreadorVinculoAutenticacionActor,
	autorizador puertosvec.Autorizador,
	fuente puertosbolsa.FuenteDatosLlamamiento,
	motor puertosbolsa.MotorElegibilidadLlamamiento,
	reloj puertosbolsa.RelojLlamamientos,
	generador puertosbolsa.GeneradorReferenciasLlamamiento,
	transaccion puertosbolsa.TransaccionPropuestasLlamamiento,
) (*ServicioLlamamientos, error) {
	if dependenciaLlamamientoNula(resolutor) || dependenciaLlamamientoNula(vinculador) || dependenciaLlamamientoNula(autorizador) ||
		dependenciaLlamamientoNula(fuente) || dependenciaLlamamientoNula(motor) ||
		dependenciaLlamamientoNula(reloj) || dependenciaLlamamientoNula(generador) ||
		dependenciaLlamamientoNula(transaccion) {
		return nil, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida
	}
	return &ServicioLlamamientos{
		resolutor: resolutor, vinculador: vinculador, autorizador: autorizador, fuente: fuente, motor: motor,
		reloj: reloj, generador: generador, transaccion: transaccion,
	}, nil
}

// ProponerPrimerLlamamiento ejecuta el recorrido autoritativo del orden. Solo
// persiste si existe una primera persona elegible y la concesion sigue vigente
// en el instante final. Cualquier dato desconocido, ambiguedad o capacidad
// parcial falla cerrado antes de la transaccion de negocio.
func (s *ServicioLlamamientos) ProponerPrimerLlamamiento(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudProponerLlamamiento,
) (dominiobolsa.PropuestaLlamamiento, error) {
	if ctx == nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida)
	}
	if err := ctx.Err(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if s == nil || dependenciaLlamamientoNula(s.resolutor) || dependenciaLlamamientoNula(s.vinculador) || dependenciaLlamamientoNula(s.autorizador) ||
		dependenciaLlamamientoNula(s.fuente) || dependenciaLlamamientoNula(s.motor) ||
		dependenciaLlamamientoNula(s.reloj) || dependenciaLlamamientoNula(s.generador) ||
		dependenciaLlamamientoNula(s.transaccion) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida)
	}
	solicitudCanonica, err := solicitud.Clonar()
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}

	// El recurso se resuelve antes de solicitar cualquier decision. No se
	// autoriza sobre ambitos aportados por el cliente ni sobre un recurso vacio.
	recursos, err := s.resolutor.ResolverRecursosNecesidad(ctx, solicitudCanonica.NecesidadRef)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(contextoErr)
		}
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrRecursoNecesidadNoConfiable)
	}
	if err := ctx.Err(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if len(recursos) == 0 {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrRecursoNecesidadNoEncontrado)
	}
	if len(recursos) != 1 {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrRecursoNecesidadAmbiguo)
	}
	recurso := clonarRecursoAutorizable(recursos[0])
	if !recursoNecesidadMinimoValido(recurso, solicitudCanonica.NecesidadRef) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrRecursoNecesidadNoConfiable)
	}
	actorParaVinculo, err := solicitudCanonica.Actor.Clonar()
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	vinculo, err := s.vinculador.Crear(ctx, dominiovec.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: solicitudCanonica.AutenticacionRef,
		SesionRef:        solicitudCanonica.SesionRef,
	}, actorParaVinculo)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(contextoErr)
		}
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if err := ctx.Err(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if !puertosbolsa.VinculoAptoParaGestionLlamamientos(
		vinculo,
		solicitudCanonica.Actor,
		solicitudCanonica.PerfilActivoRef,
	) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(dominiovec.ErrVinculoAutenticacionActorInvalido)
	}

	instanteAutorizacion, err := s.ahoraCanonico()
	if err != nil || !actorVigenteEn(solicitudCanonica, instanteAutorizacion) ||
		!vinculo.VigenteEn(instanteAutorizacion, solicitudCanonica.Actor) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	actorParaPDP, err := solicitudCanonica.Actor.Clonar()
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	ordenAutorizacion := dominiovec.SolicitudAutorizacion{
		Principal:                 actorParaPDP.Principal,
		PerfilActivoRef:           solicitudCanonica.PerfilActivoRef,
		ContextoActor:             actorParaPDP,
		VinculoAutenticacionActor: vinculo,
		Accion:                    puertosbolsa.AccionProponerLlamamiento,
		Recurso:                   clonarRecursoAutorizable(recurso),
		Finalidad:                 puertosbolsa.FinalidadProponerLlamamiento,
		CorrelacionRef:            solicitudCanonica.CorrelacionRef,
		Motivo:                    "proponer primer llamamiento para necesidad autorizada",
	}
	if err := ordenAutorizacion.Validar(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if err := ordenAutorizacion.ValidarVinculoAutenticacionActor(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	decision, err := s.autorizador.Exigir(ctx, ordenAutorizacion)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(contextoErr)
		}
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if err := ctx.Err(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if !recursosAutorizablesExactos(recurso, ordenAutorizacion.Recurso) ||
		!decisionLlamamientoExacta(decision, ordenAutorizacion, instanteAutorizacion) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(dominiovec.ErrDecisionAutorizacionInvalida)
	}

	coincidencias, err := s.fuente.CargarDatosAutoritativosLlamamiento(ctx, solicitudCanonica.NecesidadRef)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(contextoErr)
		}
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrDatosLlamamientoNoConfiables)
	}
	if err := ctx.Err(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if len(coincidencias) == 0 {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrDatosLlamamientoNoEncontrados)
	}
	if len(coincidencias) != 1 {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrDatosLlamamientoAmbiguos)
	}
	datos, err := coincidencias[0].Clonar()
	if err != nil || !datosCoincidenConRecurso(datos, recurso, solicitudCanonica.NecesidadRef) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrDatosLlamamientoNoConfiables)
	}

	referenciaInstantanea, err := s.generador.NuevaReferenciaInstantaneaOrdenBolsa()
	if err != nil || !puertosbolsa.ReferenciaOpacaLlamamientoValida(referenciaInstantanea) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrGeneracionReferenciaLlamamiento)
	}
	instanteInstantanea, err := s.ahoraCanonico()
	if err != nil || !actorVigenteEn(solicitudCanonica, instanteInstantanea) ||
		!vinculo.VigenteEn(instanteInstantanea, solicitudCanonica.Actor) || !decision.VigenteEn(instanteInstantanea) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	instantanea, err := dominiobolsa.NuevaInstantaneaOrdenBolsa(dominiobolsa.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: referenciaInstantanea,
		Version:        1,
		Bolsa:          datos.Bolsa,
		ReferidaEn:     instanteInstantanea,
		GeneradaEn:     instanteInstantanea,
		Entradas:       datos.Entradas,
	})
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrDatosLlamamientoNoConfiables)
	}

	evaluaciones := make([]dominiobolsa.EvaluacionParticipacionLlamamiento, 0, len(instantanea.Entradas))
	recibos := make(map[string]struct{}, len(instantanea.Entradas)*2)
	encontradaElegible := false
	for indice := range instantanea.Entradas {
		if err := ctx.Err(); err != nil {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
		}
		instanteEvaluacion, err := s.ahoraCanonico()
		if err != nil || instanteEvaluacion.Before(instantanea.GeneradaEn) ||
			!actorVigenteEn(solicitudCanonica, instanteEvaluacion) ||
			!vinculo.VigenteEn(instanteEvaluacion, solicitudCanonica.Actor) || !decision.VigenteEn(instanteEvaluacion) {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrEvaluacionMotorNoConfiable)
		}
		peticionMotor, err := nuevaSolicitudMotor(datos.Necesidad, instantanea, datos.Politica, instantanea.Entradas[indice], instanteEvaluacion)
		if err != nil {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
		}
		evaluacion, err := s.motor.EvaluarParticipacion(ctx, peticionMotor)
		if err != nil {
			if contextoErr := ctx.Err(); contextoErr != nil {
				return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(contextoErr)
			}
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrMotorElegibilidadNoDisponible)
		}
		if err := ctx.Err(); err != nil {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
		}
		if !evaluacionMotorExacta(evaluacion, peticionMotor) ||
			referenciaRepetida(recibos, evaluacion.EntradaEvaluacionRef) ||
			referenciaRepetida(recibos, evaluacion.ResultadoEvaluacionRef) {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrEvaluacionMotorNoConfiable)
		}
		recibos[evaluacion.EntradaEvaluacionRef] = struct{}{}
		recibos[evaluacion.ResultadoEvaluacionRef] = struct{}{}
		evaluaciones = append(evaluaciones, clonarEvaluacion(evaluacion))
		if evaluacion.Resultado == dominiobolsa.ResultadoElegible {
			encontradaElegible = true
			break
		}
		if evaluacion.Resultado != dominiobolsa.ResultadoNoElegible {
			return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrEvaluacionMotorNoConfiable)
		}
	}
	if !encontradaElegible {
		return dominiobolsa.PropuestaLlamamiento{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			dominiobolsa.ErrSinParticipacionElegible,
		)
	}

	referenciaPropuesta, err := s.generador.NuevaReferenciaPropuestaLlamamiento()
	if err != nil || !puertosbolsa.ReferenciaOpacaLlamamientoValida(referenciaPropuesta) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(puertosbolsa.ErrGeneracionReferenciaLlamamiento)
	}
	instanteFinal, err := s.ahoraCanonico()
	if err != nil || !actorVigenteEn(solicitudCanonica, instanteFinal) ||
		!vinculo.VigenteEn(instanteFinal, solicitudCanonica.Actor) || !decision.VigenteEn(instanteFinal) {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	propuesta, err := dominiobolsa.ProponerPrimerLlamamiento(dominiobolsa.OrdenProponerPrimerLlamamiento{
		PropuestaRef: referenciaPropuesta,
		Bolsa:        datos.Bolsa,
		Necesidad:    datos.Necesidad,
		Instantanea:  instantanea,
		Politica:     datos.Politica,
		Evaluaciones: evaluaciones,
		GeneradaEn:   instanteFinal,
	})
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, instanteFinal)
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if err := ctx.Err(); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errorPropuestaDenegada(err)
	}
	if err := s.transaccion.GuardarPropuestaLlamamiento(ctx, propuesta, evidencia); err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errors.Join(puertosbolsa.ErrPersistenciaPropuestaNoDisponible, err)
	}
	clon, err := propuesta.ClonarCanonica()
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, errors.Join(puertosbolsa.ErrPersistenciaPropuestaNoDisponible, err)
	}
	return clon, nil
}

func (s *ServicioLlamamientos) ahoraCanonico() (time.Time, error) {
	if s == nil || dependenciaLlamamientoNula(s.reloj) {
		return time.Time{}, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida
	}
	instante := s.reloj.Ahora()
	if instante.IsZero() || instante.Year() < 1 || instante.Year() > 9999 {
		return time.Time{}, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida
	}
	return instante.UTC().Truncate(time.Microsecond), nil
}

func actorVigenteEn(s puertosbolsa.SolicitudProponerLlamamiento, instante time.Time) bool {
	return s.Validar() == nil && !instante.Before(s.Actor.ResueltoEn) &&
		s.Actor.Instantanea.VigenteEn(instante) && s.PerfilActivoRef == s.Actor.PerfilActivoRef
}

func recursoNecesidadMinimoValido(recurso dominiovec.RecursoAutorizable, necesidadRef string) bool {
	if recurso.Validar() != nil || recurso.Referencia != necesidadRef ||
		recurso.ModuloID != puertosbolsa.ModuloLlamamientos || recurso.Tipo != puertosbolsa.TipoRecursoNecesidad ||
		len(recurso.Atributos) != 0 || len(recurso.Ambitos) != 2 {
		return false
	}
	return puertosbolsa.ReferenciaOpacaLlamamientoValida(recurso.Ambitos["categoria_ref"]) &&
		puertosbolsa.ReferenciaOpacaLlamamientoValida(recurso.Ambitos["unidad_ref"])
}

func datosCoincidenConRecurso(
	datos puertosbolsa.DatosAutoritativosLlamamiento,
	recurso dominiovec.RecursoAutorizable,
	necesidadRef string,
) bool {
	return datos.Bolsa.Validar() == nil && datos.Necesidad.Validar() == nil && datos.Politica.Validar() == nil &&
		datos.Necesidad.NecesidadRef == necesidadRef && datos.Necesidad.BolsaRef == datos.Bolsa.BolsaRef &&
		datos.Necesidad.CategoriaRef == datos.Bolsa.CategoriaRef &&
		recurso.Ambitos["categoria_ref"] == datos.Necesidad.CategoriaRef &&
		recurso.Ambitos["unidad_ref"] == datos.Necesidad.UnidadRef
}

func decisionLlamamientoExacta(
	decision dominiovec.DecisionAutorizacion,
	solicitud dominiovec.SolicitudAutorizacion,
	instante time.Time,
) bool {
	huellaContexto, err := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	return err == nil && decision.ValidarEvidenciaInstantanea() == nil && decision.Concedida && decision.VigenteEn(instante) &&
		puertosbolsa.VinculoAptoParaGestionLlamamientos(
			decision.VinculoAutenticacionActor,
			solicitud.ContextoActor,
			solicitud.PerfilActivoRef,
		) &&
		vinculosAutenticacionActorExactos(decision.VinculoAutenticacionActor, solicitud.VinculoAutenticacionActor) &&
		decision.VinculoAutenticacionActor.VigenteEn(instante, solicitud.ContextoActor) &&
		decision.PrincipalID == solicitud.Principal.ID && decision.PerfilActivoRef == solicitud.PerfilActivoRef &&
		decision.Accion == puertosbolsa.AccionProponerLlamamiento && decision.Accion == solicitud.Accion &&
		decision.RecursoRef == solicitud.Recurso.Referencia && decision.ModuloID == puertosbolsa.ModuloLlamamientos &&
		decision.TipoRecurso == puertosbolsa.TipoRecursoNecesidad && decision.Finalidad == puertosbolsa.FinalidadProponerLlamamiento &&
		decision.Finalidad == solicitud.Finalidad && decision.CorrelacionRef == solicitud.CorrelacionRef &&
		decision.ContextoRecursoHuellaSHA256 == huellaContexto && len(decision.CamposPermitidos) == 0 &&
		len(decision.Obligaciones) == 0 && decision.GarantiaMinima == dominiovec.AuthAssuranceHigh
}

func vinculosAutenticacionActorExactos(
	primero dominiovec.VinculoAutenticacionActorV1,
	segundo dominiovec.VinculoAutenticacionActorV1,
) bool {
	datosPrimero, errPrimero := primero.Datos()
	datosSegundo, errSegundo := segundo.Datos()
	return errPrimero == nil && errSegundo == nil && datosPrimero == datosSegundo
}

func nuevaSolicitudMotor(
	necesidad dominiobolsa.NecesidadCobertura,
	instantanea dominiobolsa.InstantaneaOrdenBolsa,
	politica dominiobolsa.ReferenciaPoliticaLlamamiento,
	entrada dominiobolsa.EntradaOrdenBolsa,
	evaluadaEn time.Time,
) (puertosbolsa.SolicitudEvaluarParticipacionLlamamiento, error) {
	necesidad, err := necesidad.ClonarCanonica()
	if err != nil {
		return puertosbolsa.SolicitudEvaluarParticipacionLlamamiento{}, puertosbolsa.ErrDatosLlamamientoNoConfiables
	}
	politica, err = politica.ClonarCanonica()
	if err != nil {
		return puertosbolsa.SolicitudEvaluarParticipacionLlamamiento{}, puertosbolsa.ErrDatosLlamamientoNoConfiables
	}
	participacion, err := entrada.Participacion.ClonarCanonica()
	if err != nil {
		return puertosbolsa.SolicitudEvaluarParticipacionLlamamiento{}, puertosbolsa.ErrDatosLlamamientoNoConfiables
	}
	solicitud := puertosbolsa.SolicitudEvaluarParticipacionLlamamiento{
		Necesidad: necesidad, InstantaneaRef: instantanea.InstantaneaRef, VersionInstantanea: instantanea.Version,
		HuellaInstantaneaSHA256: instantanea.HuellaContenidoSHA256, InstanteReferencia: instantanea.ReferidaEn,
		InstantaneaGeneradaEn: instantanea.GeneradaEn, Politica: politica,
		Entrada: dominiobolsa.EntradaOrdenBolsa{Orden: entrada.Orden, Participacion: participacion}, EvaluadaEn: evaluadaEn,
	}
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.SolicitudEvaluarParticipacionLlamamiento{}, err
	}
	return solicitud, nil
}

func evaluacionMotorExacta(
	evaluacion dominiobolsa.EvaluacionParticipacionLlamamiento,
	solicitud puertosbolsa.SolicitudEvaluarParticipacionLlamamiento,
) bool {
	if evaluacion.Validar() != nil || solicitud.Validar() != nil || !evaluacion.EvaluadaEn.Equal(solicitud.EvaluadaEn) {
		return false
	}
	situacion, vigente := solicitud.Entrada.Participacion.SituacionVigenteEn(solicitud.InstanteReferencia)
	huellaNecesidad, err := solicitud.Necesidad.HuellaCanonicaSHA256()
	return err == nil && vigente && evaluacion.ParticipacionRef == solicitud.Entrada.Participacion.ParticipacionRef &&
		evaluacion.SujetoRef == solicitud.Entrada.Participacion.SujetoRef && evaluacion.Orden == solicitud.Entrada.Orden &&
		evaluacion.SituacionSecuencia == situacion.Secuencia && evaluacion.EstadoClave == situacion.EstadoClave &&
		evaluacion.EstadoVersion == situacion.EstadoVersion && evaluacion.HuellaEstadoSHA256 == situacion.HuellaEstadoSHA256 &&
		evaluacion.NecesidadRef == solicitud.Necesidad.NecesidadRef && evaluacion.VersionNecesidad == solicitud.Necesidad.Version &&
		evaluacion.HuellaNecesidadSHA256 == huellaNecesidad && evaluacion.InstantaneaRef == solicitud.InstantaneaRef &&
		evaluacion.VersionInstantanea == solicitud.VersionInstantanea &&
		evaluacion.HuellaInstantaneaSHA256 == solicitud.HuellaInstantaneaSHA256 &&
		evaluacion.PoliticaRef == solicitud.Politica.PoliticaRef && evaluacion.VersionPolitica == solicitud.Politica.Version &&
		evaluacion.HuellaPoliticaSHA256 == solicitud.Politica.HuellaSHA256
}

func referenciaRepetida(vistas map[string]struct{}, referencia string) bool {
	_, existe := vistas[referencia]
	return existe
}

func clonarEvaluacion(e dominiobolsa.EvaluacionParticipacionLlamamiento) dominiobolsa.EvaluacionParticipacionLlamamiento {
	e.Motivos = append([]dominiobolsa.MotivoEvaluacionLlamamiento(nil), e.Motivos...)
	sort.Slice(e.Motivos, func(i, j int) bool {
		if e.Motivos[i].ReglaRef == e.Motivos[j].ReglaRef {
			return e.Motivos[i].Clave < e.Motivos[j].Clave
		}
		return e.Motivos[i].ReglaRef < e.Motivos[j].ReglaRef
	})
	return e
}

func clonarRecursoAutorizable(recurso dominiovec.RecursoAutorizable) dominiovec.RecursoAutorizable {
	clon := recurso
	clon.Ambitos = clonarMapa(recurso.Ambitos)
	clon.Atributos = clonarMapa(recurso.Atributos)
	return clon
}

func recursosAutorizablesExactos(primero, segundo dominiovec.RecursoAutorizable) bool {
	return primero.Referencia == segundo.Referencia && primero.ModuloID == segundo.ModuloID &&
		primero.Tipo == segundo.Tipo && reflect.DeepEqual(primero.Ambitos, segundo.Ambitos) &&
		reflect.DeepEqual(primero.Atributos, segundo.Atributos)
}

func clonarMapa(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	clon := make(map[string]string, len(origen))
	for clave, valor := range origen {
		clon[clave] = valor
	}
	return clon
}

func errorPropuestaDenegada(causa error) error {
	if causa == nil {
		return dominiovec.ErrAutorizacionDenegada
	}
	return errors.Join(dominiovec.ErrAutorizacionDenegada, causa)
}

func dependenciaLlamamientoNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
