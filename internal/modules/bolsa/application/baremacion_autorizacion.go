package application

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func (s *ServicioBaremacion) autorizar(
	ctx context.Context,
	actor ActorBaremacion,
	accion puertosbolsa.AccionOperacionBaremacion,
	clase puertosbolsa.ClaseRecursoOperacionBaremacion,
	recursoRef, sujetoRef, finalidad, correlacionRef string,
) (puertosbolsa.ContextoOperacionBaremacion, error) {
	recurso := dominiovec.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: "bolsa", Tipo: string(clase),
		Ambitos: map[string]string{"sujeto_ref": sujetoRef},
	}
	return s.autorizarConRecursoBaremacion(
		ctx, actor, accion, clase, recursoRef, sujetoRef, finalidad, correlacionRef, recurso, false,
	)
}

func (s *ServicioBaremacion) autorizarAlmacen(
	ctx context.Context,
	actor ActorBaremacion,
	accion puertosbolsa.AccionOperacionBaremacion,
	clase puertosbolsa.ClaseRecursoOperacionBaremacion,
	recursoRef, sujetoRef, finalidad, correlacionRef string,
	recurso dominiovec.RecursoAutorizable,
) (puertosbolsa.ContextoOperacionBaremacion, error) {
	return s.autorizarConRecursoBaremacion(
		ctx, actor, accion, clase, recursoRef, sujetoRef, finalidad, correlacionRef, recurso, true,
	)
}

func (s *ServicioBaremacion) autorizarRevision(
	ctx context.Context,
	actor ActorBaremacion,
	revision RevisionBaremacionIniciada,
	accion puertosbolsa.AccionOperacionBaremacion,
	clase puertosbolsa.ClaseRecursoOperacionBaremacion,
	recursoRef, sujetoRef, finalidad, correlacionRef string,
) (puertosbolsa.ContextoOperacionBaremacion, error) {
	capacidad, err := s.autorizar(
		ctx, actor, accion, clase, recursoRef, sujetoRef, finalidad, correlacionRef,
	)
	if err != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, err
	}
	if !contextoAutorizadoCoincideConRevision(capacidad, revision) {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrResultadoBaremacionNoConfiable,
		)
	}
	return capacidad, nil
}

func (s *ServicioBaremacion) autorizarAlmacenRevision(
	ctx context.Context,
	actor ActorBaremacion,
	revision RevisionBaremacionIniciada,
	accion puertosbolsa.AccionOperacionBaremacion,
	clase puertosbolsa.ClaseRecursoOperacionBaremacion,
	recursoRef, sujetoRef, finalidad, correlacionRef string,
	recurso dominiovec.RecursoAutorizable,
) (puertosbolsa.ContextoOperacionBaremacion, error) {
	capacidad, err := s.autorizarAlmacen(
		ctx, actor, accion, clase, recursoRef, sujetoRef, finalidad, correlacionRef, recurso,
	)
	if err != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, err
	}
	if !contextoAutorizadoCoincideConRevision(capacidad, revision) {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrResultadoBaremacionNoConfiable,
		)
	}
	return capacidad, nil
}

func contextoAutorizadoCoincideConRevision(
	capacidad puertosbolsa.ContextoOperacionBaremacion,
	revision RevisionBaremacionIniciada,
) bool {
	if validarRevisionIniciada(revision) != nil || capacidad.Validar() != nil {
		return false
	}
	proyeccion := capacidad.Proyeccion()
	return proyeccion.PrincipalRef == revision.principalReservaRef &&
		proyeccion.PerfilActorClave == revision.perfilActorClave
}

func (s *ServicioBaremacion) autorizarConRecursoBaremacion(
	ctx context.Context,
	actor ActorBaremacion,
	accion puertosbolsa.AccionOperacionBaremacion,
	clase puertosbolsa.ClaseRecursoOperacionBaremacion,
	recursoRef, sujetoRef, finalidad, correlacionRef string,
	recurso dominiovec.RecursoAutorizable,
	recursoAlmacen bool,
) (puertosbolsa.ContextoOperacionBaremacion, error) {
	if s == nil || dependenciaBaremacionNula(s.autorizador) {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrDependenciaBaremacionRequerida,
		)
	}
	contextoActor, vinculo, _, err := s.resolverSesionBaremacion(ctx, actor)
	if err != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, err
	}
	solicitud := dominiovec.SolicitudAutorizacion{
		Principal:                 contextoActor.Principal,
		PerfilActivoRef:           contextoActor.PerfilActivoRef,
		ContextoActor:             contextoActor,
		VinculoAutenticacionActor: vinculo,
		Accion:                    string(accion),
		Recurso:                   recurso,
		Finalidad:                 finalidad,
		CorrelacionRef:            correlacionRef,
		Motivo:                    actor.Motivo,
	}
	if solicitud.ValidarVinculoAutenticacionActor() != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			dominiovec.ErrSolicitudAutorizacionInvalida,
		)
	}
	huellaContextoRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	decision, err := s.autorizador.Exigir(ctx, solicitud)
	if err != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	instanteDecision, err := s.ahora()
	if err != nil || decision.ValidarEvidenciaInstantanea() != nil || !decision.Concedida ||
		!decision.VigenteEn(instanteDecision) || !vinculo.VigenteEn(instanteDecision, contextoActor) ||
		!decision.VinculoAutenticacionActor.CoincideExactamenteCon(vinculo) ||
		decision.PrincipalID != contextoActor.Principal.ID ||
		decision.PerfilActivoRef != contextoActor.PerfilActivoRef ||
		decision.Accion != solicitud.Accion || decision.RecursoRef != recurso.Referencia ||
		decision.ModuloID != recurso.ModuloID || decision.TipoRecurso != recurso.Tipo ||
		decision.ContextoRecursoHuellaSHA256 != huellaContextoRecurso ||
		decision.Finalidad != finalidad || decision.CorrelacionRef != correlacionRef ||
		!dominiovec.CumpleGarantiaAutenticacion(contextoActor.Principal.AuthAssurance, decision.GarantiaMinima) ||
		len(decision.Obligaciones) != 0 {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			dominiovec.ErrDecisionAutorizacionInvalida,
			err,
		)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	vinculoBaremacion := puertosbolsa.VinculoAutenticacionBaremacion{
		SujetoRef: sujetoRef, Metodo: datosVinculo.MetodoObservado, Garantia: datosVinculo.GarantiaObservada,
		AutenticacionRef: datosVinculo.AutenticacionRef, SesionRef: datosVinculo.SesionRef,
		SesionEmitidaEn: datosVinculo.SesionEmitidaEn, SesionValidaHasta: datosVinculo.SesionValidaHasta,
		VinculoAutenticacionActor: vinculo,
	}
	var capacidad puertosbolsa.ContextoOperacionBaremacion
	if recursoAlmacen {
		capacidad, err = puertosbolsa.NuevaAutorizacionOperacionAlmacenBaremacion(
			decision, recurso, vinculoBaremacion, instanteDecision,
		)
	} else {
		capacidad, err = puertosbolsa.NuevaAutorizacionOperacionBaremacion(
			decision, vinculoBaremacion, instanteDecision,
		)
	}
	if err != nil || capacidad.ValidarVigentePara(accion, clase, recursoRef, instanteDecision) != nil {
		return puertosbolsa.ContextoOperacionBaremacion{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	return capacidad, nil
}

func (s *ServicioBaremacion) resolverSesionBaremacion(
	ctx context.Context,
	actor ActorBaremacion,
) (dominiovec.ContextoActor, dominiovec.VinculoAutenticacionActorV1, time.Time, error) {
	if ctx == nil || s == nil || dependenciaBaremacionNula(s.sesiones) || dependenciaBaremacionNula(s.reloj) {
		return dominiovec.ContextoActor{}, dominiovec.VinculoAutenticacionActorV1{}, time.Time{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrDependenciaBaremacionRequerida,
		)
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.ContextoActor{}, dominiovec.VinculoAutenticacionActorV1{}, time.Time{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			err,
		)
	}
	ahora, err := s.ahora()
	if err != nil {
		return dominiovec.ContextoActor{}, dominiovec.VinculoAutenticacionActorV1{}, time.Time{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			err,
		)
	}
	sesiones, err := s.sesiones.BuscarSesionesAutenticadasBaremacion(ctx)
	if err != nil || len(sesiones) != 1 {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return dominiovec.ContextoActor{}, dominiovec.VinculoAutenticacionActorV1{}, time.Time{}, errors.Join(
				dominiovec.ErrAutorizacionDenegada,
				contextoErr,
			)
		}
		return dominiovec.ContextoActor{}, dominiovec.VinculoAutenticacionActorV1{}, time.Time{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrResultadoBaremacionNoConfiable,
		)
	}
	contextoActor, vinculo, err := sesiones[0].capacidades()
	if err != nil || validarActorBaremacion(actor) != nil || contextoActor.Validar() != nil ||
		!vinculo.VigenteEn(ahora, contextoActor) {
		return dominiovec.ContextoActor{}, dominiovec.VinculoAutenticacionActorV1{}, time.Time{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrResultadoBaremacionNoConfiable,
		)
	}
	return contextoActor, vinculo, ahora, nil
}

func (s *ServicioBaremacion) validarSesionRevision(
	ctx context.Context,
	actor ActorBaremacion,
	revision RevisionBaremacionIniciada,
) error {
	if validarRevisionIniciada(revision) != nil {
		return errors.Join(dominiovec.ErrAutorizacionDenegada, ErrResultadoBaremacionNoConfiable)
	}
	contextoActor, _, _, err := s.resolverSesionBaremacion(ctx, actor)
	if err != nil {
		return err
	}
	if contextoActor.Principal.ID != revision.principalReservaRef ||
		contextoActor.PerfilActivoRef != revision.perfilActorClave {
		return errors.Join(dominiovec.ErrAutorizacionDenegada, ErrResultadoBaremacionNoConfiable)
	}
	return nil
}

func (s *ServicioBaremacion) verificarFuentesDecision(
	ctx context.Context,
	actor ActorBaremacion,
	revision RevisionBaremacionIniciada,
	calculo dominiobolsa.CalculoOficialBaremacion,
) ([]string, error) {
	autorizaciones := make([]string, 0, 1+2*len(calculo.Evidencias))
	criterio := calculo.Criterio
	autorizacionCriterio, err := s.autorizarRevision(ctx, actor, revision, puertosbolsa.AccionConsultarCriterioBaremacion,
		puertosbolsa.ClaseRecursoProceso, criterio.ProcesoRef, calculo.SujetoRef,
		revision.finalidadClave, revision.correlacionRef)
	if err != nil {
		return nil, err
	}
	autorizaciones, err = incorporarAutorizacionesBaremacion(autorizaciones, autorizacionCriterio)
	if err != nil {
		return nil, err
	}
	solicitudCriterio := puertosbolsa.SolicitudObtenerCriterioBaremacion{
		Contexto: autorizacionCriterio, ProcesoRef: criterio.ProcesoRef, Clave: criterio.Clave,
		Version: criterio.Version, HuellaEsperadaSHA256: criterio.HuellaSHA256,
	}
	criterioConfiable, err := s.fuenteDatos.ObtenerCriterio(ctx, solicitudCriterio)
	if err != nil || criterioConfiable.ValidarPara(solicitudCriterio) != nil || criterioConfiable.Referencia != criterio {
		return nil, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	for _, evidencia := range calculo.Evidencias {
		autorizacionEvidencia, err := s.autorizarRevision(ctx, actor, revision, puertosbolsa.AccionConsultarEvidenciaBaremacion,
			puertosbolsa.ClaseRecursoEvidencia, evidencia.Referencia.DocumentoRef, calculo.SujetoRef,
			revision.finalidadClave, revision.correlacionRef)
		if err != nil {
			return nil, err
		}
		autorizaciones, err = incorporarAutorizacionesBaremacion(autorizaciones, autorizacionEvidencia)
		if err != nil {
			return nil, err
		}
		solicitudEvidencia := puertosbolsa.SolicitudObtenerEvidenciaBaremacion{
			Contexto: autorizacionEvidencia, ProcesoRef: calculo.ProcesoRef,
			SolicitudRef: calculo.SolicitudRef, Evidencia: evidencia,
		}
		confiable, err := s.fuenteDatos.ObtenerEvidencia(ctx, solicitudEvidencia)
		if err != nil || confiable.ValidarPara(solicitudEvidencia) != nil {
			return nil, errors.Join(ErrResultadoBaremacionNoConfiable, err)
		}
		autorizacionRepresentacion, err := s.autorizarRevision(ctx, actor, revision,
			puertosbolsa.AccionConsultarRepresentacionBaremacion, puertosbolsa.ClaseRecursoRepresentacion,
			evidencia.Referencia.RepresentacionRef, calculo.SujetoRef,
			revision.finalidadClave, revision.correlacionRef)
		if err != nil {
			return nil, err
		}
		autorizaciones, err = incorporarAutorizacionesBaremacion(autorizaciones, autorizacionRepresentacion)
		if err != nil {
			return nil, err
		}
		solicitudRepresentacion := puertosbolsa.SolicitudObtenerRepresentacionBaremacion{
			Contexto: autorizacionRepresentacion, Referencia: evidencia.Referencia,
		}
		representacion, err := s.fuenteDatos.ObtenerRepresentacion(ctx, solicitudRepresentacion)
		if err != nil || representacion.ValidarPara(solicitudRepresentacion) != nil ||
			representacion.Representacion.ValidarPertenencia(confiable.Documento) != nil {
			return nil, errors.Join(ErrResultadoBaremacionNoConfiable, err)
		}
	}
	return autorizaciones, nil
}

func (s *ServicioBaremacion) sellarReserva(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) (puertosbolsa.SolicitudReservarCambioBaremacion, error) {
	representacion, err := puertosbolsa.RepresentacionCanonicaReservaBaremacion(solicitud)
	if err != nil {
		return puertosbolsa.SolicitudReservarCambioBaremacion{}, err
	}
	sello, err := s.selladorSolicitud.SellarSelloBaremacion(ctx, puertosbolsa.SolicitudSellarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloReservaBaremacion,
		RepresentacionCanonica: representacion,
	})
	if err != nil || !selloGeneradoBaremacionValido(sello) {
		return puertosbolsa.SolicitudReservarCambioBaremacion{},
			errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	solicitud.HuellaSolicitudHMAC = sello
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.SolicitudReservarCambioBaremacion{}, err
	}
	return solicitud, nil
}

func (s *ServicioBaremacion) sellarConfirmacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (puertosbolsa.SolicitudConfirmarCambioBaremacion, error) {
	representacion, err := puertosbolsa.RepresentacionCanonicaConfirmacionBaremacion(solicitud)
	if err != nil {
		return puertosbolsa.SolicitudConfirmarCambioBaremacion{}, err
	}
	sello, err := s.selladorSolicitud.SellarSelloBaremacion(ctx, puertosbolsa.SolicitudSellarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloConfirmacionBaremacionV2,
		RepresentacionCanonica: representacion,
	})
	if err != nil || !selloGeneradoBaremacionValido(sello) {
		return puertosbolsa.SolicitudConfirmarCambioBaremacion{},
			errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	solicitud.HuellaSolicitudHMAC = sello
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.SolicitudConfirmarCambioBaremacion{}, err
	}
	return solicitud, nil
}

func (s *ServicioBaremacion) sellarCustodia(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudCustodiarDocumentoFirmable,
) (puertosbolsa.SolicitudCustodiarDocumentoFirmable, error) {
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.SolicitudCustodiarDocumentoFirmable{}, err
	}
	partes := []string{
		"custodia_decision_baremacion_v1", solicitud.OperacionRef, solicitud.ClaveIdempotencia,
		solicitud.CargaRef, solicitud.SujetoSeudonimoHMAC, solicitud.ProcesoRef, solicitud.SolicitudRef,
		solicitud.BaremacionMeritoRef, solicitud.DecisionRef, solicitud.ClasificacionClave,
		solicitud.Codificacion.HuellaDocumentoSHA256, solicitud.Contexto.Proyeccion().AutorizacionRef,
		solicitud.Contexto.Proyeccion().CorrelacionRef, solicitud.Contexto.Proyeccion().FinalidadClave,
	}
	var canonica bytes.Buffer
	for _, parte := range partes {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(parte)))
		_, _ = canonica.Write(longitud[:])
		_, _ = canonica.WriteString(parte)
	}
	carga, err := puertosbolsa.NuevaCargaProtegida(canonica.Bytes())
	if err != nil {
		return puertosbolsa.SolicitudCustodiarDocumentoFirmable{}, err
	}
	sello, err := s.selladorSolicitud.SellarSolicitudBaremacion(ctx, carga)
	if err != nil || !selloGeneradoBaremacionValido(sello) {
		return puertosbolsa.SolicitudCustodiarDocumentoFirmable{},
			errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	solicitud.HuellaAlmacenHMAC = sello
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.SolicitudCustodiarDocumentoFirmable{}, err
	}
	return solicitud, nil
}

func (s *ServicioBaremacion) sellarPartesBaremacion(ctx context.Context, partes []string) (string, error) {
	if len(partes) == 0 || len(partes) > 128 {
		return "", ErrOrdenBaremacionInvalida
	}
	var canonica bytes.Buffer
	for _, parte := range partes {
		if parte == "" {
			return "", ErrOrdenBaremacionInvalida
		}
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(parte)))
		_, _ = canonica.Write(longitud[:])
		_, _ = canonica.WriteString(parte)
	}
	carga, err := puertosbolsa.NuevaCargaProtegida(canonica.Bytes())
	if err != nil {
		return "", err
	}
	sello, err := s.selladorSolicitud.SellarSolicitudBaremacion(ctx, carga)
	if err != nil || !selloGeneradoBaremacionValido(sello) {
		return "", errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	return sello, nil
}
