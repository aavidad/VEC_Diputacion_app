package gobiernoconvocatorias

import (
	"context"
	"errors"
	"reflect"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const duracionArrendamientoBorrador = 2 * time.Minute

// ServicioBorradores gobierna exclusivamente alta y actualizacion. La
// idempotencia se resuelve antes de construir una ejecucion mutable.
type ServicioBorradores struct {
	reloj            puertosvec.Reloj
	preparadorAlta   PreparadorAltaBorrador
	motivos          ResolvedorMotivoBorrador
	lector           LectorBorradorExacto
	comprometedor    puertosbolsa.ComprometedorMotivoGobiernoConvocatoria
	derivador        DerivadorIdentidadOperacion
	autorizador      AutorizadorIntencionBorrador
	diario           DiarioOperacionesBorrador
	sellador         SelladorMotivoBorrador
	politicasCifrado ProveedorPoliticaGobernadaCifradoBorrador
	perfilesCifrado  ResolvedorPerfilCifradoBorrador
	cifrador         CifradorAEADKMSBorrador
	confirmador      ConfirmadorAtomicoBorrador
	verificador      VerificadorReciboBorrador
	procedencia      ProcedenciaActoBorrador
}

func NuevoServicioBorradores(
	reloj puertosvec.Reloj,
	preparadorAlta PreparadorAltaBorrador,
	motivos ResolvedorMotivoBorrador,
	lector LectorBorradorExacto,
	comprometedor puertosbolsa.ComprometedorMotivoGobiernoConvocatoria,
	derivador DerivadorIdentidadOperacion,
	autorizador AutorizadorIntencionBorrador,
	diario DiarioOperacionesBorrador,
	sellador SelladorMotivoBorrador,
	politicasCifrado ProveedorPoliticaGobernadaCifradoBorrador,
	perfilesCifrado ResolvedorPerfilCifradoBorrador,
	cifrador CifradorAEADKMSBorrador,
	confirmador ConfirmadorAtomicoBorrador,
	verificador VerificadorReciboBorrador,
	procedencia ProcedenciaActoBorrador,
) (*ServicioBorradores, error) {
	dependencias := []any{reloj, preparadorAlta, motivos, lector, comprometedor, derivador,
		autorizador, diario, sellador, politicasCifrado, perfilesCifrado, cifrador, confirmador,
		verificador}
	for _, dependencia := range dependencias {
		if dependenciaNulaBorrador(dependencia) {
			return nil, ErrServicioBorradoresInvalido
		}
	}
	if !procedencia.valida() || !autoridadesPoliticaBorradorSeparadas(
		politicasCifrado.IdentidadAutoridadBorrador(),
		perfilesCifrado.IdentidadAutoridadBorrador(),
	) || !autoridadesOperativasBorradorSeparadas(
		confirmador.IdentidadAutoridadBorrador(),
		verificador.IdentidadAutoridadBorrador(),
	) {
		return nil, ErrServicioBorradoresInvalido
	}
	return &ServicioBorradores{
		reloj: reloj, preparadorAlta: preparadorAlta, motivos: motivos, lector: lector,
		comprometedor: comprometedor, derivador: derivador, autorizador: autorizador,
		diario: diario, sellador: sellador, politicasCifrado: politicasCifrado,
		perfilesCifrado: perfilesCifrado,
		cifrador:        cifrador, confirmador: confirmador, verificador: verificador,
		procedencia: procedencia,
	}, nil
}

type baseOperacionBorrador struct {
	clave          ClaveClienteIdempotenciaConvocatoria
	actor          dominiovec.ContextoActor
	vinculo        dominiovec.VinculoAutenticacionActorV1
	correlacionRef string
	motivoCatalogo dominiovec.ReferenciaEntradaCatalogo
	intencion      IntencionBorradorCanonica
	preparar       func(context.Context) (ejecucionBorrador, error)
}

type ejecucionBorrador struct {
	version    dominiobolsa.VersionConvocatoriaGobernada
	material   puertosbolsa.MaterialIntencionGobiernoConvocatoria
	compromiso puertosbolsa.CompromisoMotivoGobiernoConvocatoria
	plantilla  *PlantillaBorradorResuelta
}

func (s *ServicioBorradores) Crear(
	ctx context.Context,
	orden OrdenCrearBorrador,
) (ProyeccionReciboBorrador, error) {
	if err := s.validarContexto(ctx); err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	actor, err := validarActorBorrador(
		orden.Actor, orden.VinculoAutenticacionActor, s.ahora(), orden.CorrelacionRef,
	)
	if err != nil || !orden.ClaveCliente.Valida() || orden.MotivoCatalogo.Validar() != nil ||
		orden.Plantilla.ID == "" || orden.Plantilla.Version < 1 ||
		!huellaHexValida(orden.Plantilla.HuellaContenidoSHA256) {
		return ProyeccionReciboBorrador{}, errors.Join(ErrOrdenBorradorInvalida, err)
	}
	motivo, err := s.resolverMotivo(ctx, orden.MotivoCatalogo)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	plantilla, err := s.preparadorAlta.ResolverPlantillaBorrador(ctx, orden.Plantilla, s.ahora())
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	if plantilla.Referencia.Validar() != nil || plantilla.Referencia.ID != orden.Plantilla.ID ||
		plantilla.Referencia.Version != orden.Plantilla.Version ||
		plantilla.Referencia.HuellaContenidoSHA256 != orden.Plantilla.HuellaContenidoSHA256 ||
		plantilla.Configuracion.ValidarPara(orden.Contenido) != nil {
		return ProyeccionReciboBorrador{}, ErrOrdenBorradorInvalida
	}
	intencion, err := nuevaIntencionAltaBorradorCanonica(
		plantilla.Referencia, orden.CodigoVersionPublica, orden.ExpedienteRef,
		orden.Contenido, motivo,
	)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	base := baseOperacionBorrador{
		clave: orden.ClaveCliente, actor: actor, vinculo: orden.VinculoAutenticacionActor,
		correlacionRef: orden.CorrelacionRef, motivoCatalogo: motivo, intencion: intencion,
	}
	base.preparar = func(ctx context.Context) (ejecucionBorrador, error) {
		return s.prepararAlta(ctx, base, orden, plantilla)
	}
	return s.ejecutar(ctx, base)
}

func (s *ServicioBorradores) Actualizar(
	ctx context.Context,
	orden OrdenActualizarBorrador,
) (ProyeccionReciboBorrador, error) {
	if err := s.validarContexto(ctx); err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	actor, err := validarActorBorrador(
		orden.Actor, orden.VinculoAutenticacionActor, s.ahora(), orden.CorrelacionRef,
	)
	if err != nil || !orden.ClaveCliente.Valida() || orden.MotivoCatalogo.Validar() != nil ||
		orden.Esperada.Validar() != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrOrdenBorradorInvalida, err)
	}
	motivo, err := s.resolverMotivo(ctx, orden.MotivoCatalogo)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	intencion, err := nuevaIntencionActualizacionBorradorCanonica(
		orden.Esperada, orden.Contenido, motivo,
	)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	base := baseOperacionBorrador{
		clave: orden.ClaveCliente, actor: actor, vinculo: orden.VinculoAutenticacionActor,
		correlacionRef: orden.CorrelacionRef, motivoCatalogo: motivo, intencion: intencion,
	}
	base.preparar = func(ctx context.Context) (ejecucionBorrador, error) {
		return s.prepararActualizacion(ctx, base, orden)
	}
	return s.ejecutar(ctx, base)
}

func (s *ServicioBorradores) resolverMotivo(
	ctx context.Context,
	referencia dominiovec.ReferenciaEntradaCatalogo,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	resuelta, err := s.motivos.ResolverMotivoBorrador(ctx, referencia, s.ahora())
	if err != nil {
		return dominiovec.ReferenciaEntradaCatalogo{}, err
	}
	if resuelta.Validar() != nil || resuelta != referencia {
		return dominiovec.ReferenciaEntradaCatalogo{}, ErrOrdenBorradorInvalida
	}
	return resuelta, nil
}

func (s *ServicioBorradores) prepararAlta(
	ctx context.Context,
	base baseOperacionBorrador,
	orden OrdenCrearBorrador,
	plantilla PlantillaBorradorResuelta,
) (ejecucionBorrador, error) {
	instanteVersion := s.ahora()
	preparacion, err := s.preparadorAlta.PrepararAltaBorrador(
		ctx, plantilla, orden.CodigoVersionPublica, orden.ExpedienteRef, instanteVersion,
	)
	if err != nil {
		return ejecucionBorrador{}, err
	}
	if preparacion.Plantilla.Referencia != plantilla.Referencia ||
		!reflect.DeepEqual(preparacion.Plantilla.Configuracion, plantilla.Configuracion) ||
		preparacion.AmbitoOrganizativo.Validar() != nil {
		return ejecucionBorrador{}, ErrOrdenBorradorInvalida
	}
	version, err := dominiobolsa.NuevaVersionConvocatoriaGobernada(
		dominiobolsa.DatosNuevaVersionConvocatoriaGobernada{
			ID: preparacion.ID, CodigoVersionPublica: orden.CodigoVersionPublica,
			InstanciaFlujoRef:  preparacion.InstanciaFlujoRef,
			AmbitoOrganizativo: preparacion.AmbitoOrganizativo,
			Contenido:          orden.Contenido, Configuracion: plantilla.Configuracion,
			ExpedienteRef: orden.ExpedienteRef, Motivo: base.motivoCatalogo.Referencia(),
			ActorID: base.actor.PersonaRef, Instante: instanteVersion,
		},
	)
	if err != nil {
		return ejecucionBorrador{}, errors.Join(ErrOrdenBorradorInvalida, err)
	}
	compromiso, material, err := s.materializarMotivo(ctx, base, version)
	if err != nil {
		return ejecucionBorrador{}, err
	}
	ejecutar := ejecucionBorrador{version: version, material: material, compromiso: compromiso, plantilla: &plantilla}
	if !base.intencion.coincideEjecucion(version, material, &plantilla) {
		return ejecucionBorrador{}, ErrIntencionBorradorInvalida
	}
	return ejecutar, nil
}

func (s *ServicioBorradores) prepararActualizacion(
	ctx context.Context,
	base baseOperacionBorrador,
	orden OrdenActualizarBorrador,
) (ejecucionBorrador, error) {
	anterior, err := s.lector.ObtenerBorradorExacto(ctx, orden.Esperada)
	if err != nil {
		return ejecucionBorrador{}, err
	}
	estadoReal, err := puertosbolsa.EstadoVersionConvocatoria(anterior)
	if err != nil || estadoReal != orden.Esperada ||
		anterior.EstadoGobierno != dominiobolsa.EstadoGobiernoConvocatoriaBorrador {
		return ejecucionBorrador{}, puertosbolsa.ErrCASVersionConvocatoriaEnConflicto
	}
	version, err := anterior.ActualizarBorrador(
		orden.Esperada.Revision, orden.Contenido, anterior.Configuracion,
		base.actor.PersonaRef, base.motivoCatalogo.Referencia(), s.ahora(),
	)
	if err != nil {
		return ejecucionBorrador{}, errors.Join(ErrOrdenBorradorInvalida, err)
	}
	compromiso, material, err := s.materializarMotivo(ctx, base, version)
	if err != nil {
		return ejecucionBorrador{}, err
	}
	ejecutar := ejecucionBorrador{version: version, material: material, compromiso: compromiso}
	if !base.intencion.coincideEjecucion(version, material, nil) {
		return ejecucionBorrador{}, ErrIntencionBorradorInvalida
	}
	return ejecutar, nil
}

func (s *ServicioBorradores) materializarMotivo(
	ctx context.Context,
	base baseOperacionBorrador,
	version dominiobolsa.VersionConvocatoriaGobernada,
) (puertosbolsa.CompromisoMotivoGobiernoConvocatoria, puertosbolsa.MaterialIntencionGobiernoConvocatoria, error) {
	accion := base.intencion.accion()
	compromiso, err := s.comprometerMotivo(
		ctx, accion, version, base.actor.PersonaRef, base.correlacionRef,
		base.motivoCatalogo.Referencia(), s.ahora(),
	)
	if err != nil {
		return puertosbolsa.CompromisoMotivoGobiernoConvocatoria{}, puertosbolsa.MaterialIntencionGobiernoConvocatoria{}, err
	}
	var material puertosbolsa.MaterialIntencionGobiernoConvocatoria
	if accion == puertosbolsa.AccionCrearBorradorConvocatoria {
		material, err = puertosbolsa.MaterialAltaBorradorConvocatoria(version, nil, nil, compromiso)
	} else if base.intencion.datos != nil && base.intencion.datos.Esperada != nil {
		material, err = puertosbolsa.MaterialActualizacionBorradorConvocatoria(
			*base.intencion.datos.Esperada, version, compromiso,
		)
	} else {
		err = ErrIntencionBorradorInvalida
	}
	return compromiso, material, err
}

func (s *ServicioBorradores) ejecutar(
	ctx context.Context,
	base baseOperacionBorrador,
) (ProyeccionReciboBorrador, error) {
	solicitudDerivacion, err := nuevaSolicitudDerivacionIdempotencia(
		base.clave, base.intencion, base.actor,
	)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	conjunto, err := s.derivador.Derivar(ctx, solicitudDerivacion)
	if err != nil || !conjunto.valido() {
		return ProyeccionReciboBorrador{}, errors.Join(ErrRotacionIdempotenciaInvalida, err)
	}
	consulta, err := nuevaSolicitudConsultaIdentidadesBorrador(conjunto, s.ahora())
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	resultadoConsulta, err := s.diario.ConsultarIdentidades(ctx, consulta)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	if err := resultadoConsulta.ValidarPara(consulta); err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	if len(resultadoConsulta.Coincidencias) == 1 {
		coincidencia := resultadoConsulta.Coincidencias[0]
		if coincidencia.Resultado.Estado == ResultadoDiarioConflicto {
			return ProyeccionReciboBorrador{}, puertosbolsa.ErrClaveIdempotenciaConvocatoriaReusada
		}
		if coincidencia.Resultado.Estado == ResultadoDiarioConfirmado {
			return s.resolverResultadoDiario(ctx,
				coincidencia.Resultado, coincidencia.Resolucion.IdentidadPrimaria,
			)
		}
		return s.recuperar(ctx, base, consulta, coincidencia)
	}
	ejecutar, err := base.preparar(ctx)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	return s.autorizarYReservar(ctx, base, ejecutar, consulta, consulta.Identidades[0], nil, nil)
}

func (s *ServicioBorradores) recuperar(
	ctx context.Context,
	base baseOperacionBorrador,
	consulta SolicitudConsultaIdentidadesBorrador,
	coincidencia CoincidenciaIdentidadBorrador,
) (ProyeccionReciboBorrador, error) {
	instante := s.ahora()
	if _, err := validarActorBorrador(base.actor, base.vinculo, instante, base.correlacionRef); err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	solicitud := SolicitudReconciliacionBorrador{
		IdentidadPrimaria: coincidencia.Resolucion.IdentidadPrimaria,
		Control:           coincidencia.Resultado, SolicitadaEn: instante,
	}
	if solicitud.Validar() != nil {
		return ProyeccionReciboBorrador{}, ErrReconciliacionBorradorInvalida
	}
	reconciliacion, err := s.diario.Reconciliar(ctx, solicitud)
	if err != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
	}
	if err := reconciliacion.ValidarPara(solicitud); err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	switch reconciliacion.Resultado.Estado {
	case ResultadoDiarioConfirmado:
		return s.resolverResultadoDiario(ctx,
			reconciliacion.Resultado, coincidencia.Resolucion.IdentidadPrimaria,
		)
	case ResultadoDiarioNoAplicado:
		if reconciliacion.ComprobadaEn.Before(reconciliacion.Resultado.ArrendamientoVenceEn) {
			return ProyeccionReciboBorrador{}, ErrOperacionBorradorEnCurso
		}
		ejecutar, err := base.preparar(ctx)
		if err != nil {
			return ProyeccionReciboBorrador{}, err
		}
		return s.autorizarYReservar(
			ctx, base, ejecutar, consulta, coincidencia.Resolucion.IdentidadPrimaria,
			&coincidencia.Resolucion, &reconciliacion,
		)
	case ResultadoDiarioReservado, ResultadoDiarioEnCurso:
		if !reconciliacion.ComprobadaEn.Before(reconciliacion.Resultado.ArrendamientoVenceEn) {
			return ProyeccionReciboBorrador{}, ErrOperacionBorradorIndeterminada
		}
		return ProyeccionReciboBorrador{}, ErrOperacionBorradorEnCurso
	default:
		return ProyeccionReciboBorrador{}, ErrOperacionBorradorIndeterminada
	}
}

func (s *ServicioBorradores) autorizarYReservar(
	ctx context.Context,
	base baseOperacionBorrador,
	ejecutar ejecucionBorrador,
	consulta SolicitudConsultaIdentidadesBorrador,
	identidad ProyeccionIdentidadOperacion,
	resolucionAnterior *ResolucionIdentidadBorrador,
	reclamacion *ResultadoReconciliacionBorrador,
) (ProyeccionReciboBorrador, error) {
	recurso, err := puertosbolsa.RecursoAutorizableMutacionConvocatoria(ejecutar.material, ejecutar.version)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	instantePDP := s.ahora()
	actor, err := validarActorBorrador(base.actor, base.vinculo, instantePDP, base.correlacionRef)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	evaluacion, err := s.autorizador.EvaluarDecisionBorrador(
		ctx, actor, base.vinculo, recurso, base.correlacionRef,
		base.motivoCatalogo, base.intencion, instantePDP,
	)
	if err != nil {
		// Un error tecnico del PDP nunca se etiqueta como denegacion.
		return ProyeccionReciboBorrador{}, err
	}
	if evaluacion.Estado == EvaluacionPDPDenegada {
		if !referenciaProyeccionValida(evaluacion.DenegacionRef) {
			return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
		}
		return ProyeccionReciboBorrador{}, dominiovec.ErrAutorizacionDenegada
	}
	if evaluacion.Estado != EvaluacionPDPConcedida || evaluacion.DenegacionRef != "" {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	instanteReserva := s.ahora()
	actor, err = validarActorBorrador(base.actor, base.vinculo, instanteReserva, base.correlacionRef)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	decision, err := nuevaProyeccionDecisionDiario(
		evaluacion.Concesion.Evidencia, ejecutar.material, ejecutar.version, actor,
		base.correlacionRef, instanteReserva, evaluacion.Concesion.Atestacion,
	)
	if err != nil {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	venceEn := instanteReserva.Add(duracionArrendamientoBorrador)
	if decision.ValidaHasta.Before(venceEn) {
		venceEn = decision.ValidaHasta
	}
	arrendamiento, err := NuevoArrendamientoDiario(instanteReserva, venceEn)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	proyeccion, err := nuevaProyeccionReservaDecision(
		identidad, ejecutar.material.Accion, decision, arrendamiento,
	)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	solicitudReserva := SolicitudReservaDecisionBorrador{
		Proyeccion: proyeccion, IdentidadesConsulta: append([]ProyeccionIdentidadOperacion(nil), consulta.Identidades...),
		Intencion: base.intencion, Plantilla: ejecutar.plantilla, Material: ejecutar.material,
		Version: ejecutar.version, Recurso: recurso, Actor: actor, CorrelacionRef: base.correlacionRef,
		Concesion: evaluacion.Concesion, SolicitadaEn: instanteReserva,
	}
	if reclamacion == nil {
		reservada, err := s.diario.ReservarDecision(ctx, solicitudReserva)
		if err != nil {
			return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
		}
		if err := reservada.ValidarPara(solicitudReserva); err != nil {
			return ProyeccionReciboBorrador{}, err
		}
		if reservada.Resultado.Estado != ResultadoDiarioReservado {
			return s.resolverResultadoDiario(ctx,
				reservada.Resultado, reservada.Resolucion.IdentidadPrimaria,
			)
		}
		return s.confirmar(ctx, base, ejecutar, evaluacion.Concesion, proyeccion, reservada.Resultado)
	}
	if resolucionAnterior == nil {
		return ProyeccionReciboBorrador{}, ErrReclamacionBorradorInvalida
	}
	solicitudReclamacion := SolicitudReclamacionDecisionBorrador{
		ResolucionAnterior: *resolucionAnterior, Reconciliacion: *reclamacion,
		Nueva: solicitudReserva, SolicitadaEn: instanteReserva,
	}
	if solicitudReclamacion.Validar() != nil {
		return ProyeccionReciboBorrador{}, ErrReclamacionBorradorInvalida
	}
	reservada, err := s.diario.ReclamarDecision(ctx, solicitudReclamacion)
	if err != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
	}
	if err := comprobarReclamacionCreciente(
		reclamacion.Resultado, reservada, proyeccion,
	); err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	return s.confirmar(ctx, base, ejecutar, evaluacion.Concesion, proyeccion, reservada)
}

func (s *ServicioBorradores) confirmar(
	ctx context.Context,
	base baseOperacionBorrador,
	ejecutar ejecucionBorrador,
	concesion ConcesionBorradorDurable,
	proyeccion ProyeccionReservaDecision,
	reserva ResultadoOperacionDiario,
) (ProyeccionReciboBorrador, error) {
	instanteSellado := s.ahora()
	actor, err := validarActorBorrador(base.actor, base.vinculo, instanteSellado, base.correlacionRef)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	solicitudSellado := SolicitudSelladoMotivoBorrador{
		Reserva: proyeccion, Control: reserva, Version: ejecutar.version, Material: ejecutar.material,
		Compromiso: ejecutar.compromiso, Actor: actor, CorrelacionRef: base.correlacionRef,
		Concesion: concesion, SolicitadaEn: instanteSellado,
	}
	if solicitudSellado.Validar() != nil {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	sellado, err := s.sellador.VerificarYSellarMotivo(ctx, solicitudSellado)
	if err != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
	}
	instantePolitica := s.ahora()
	solicitudPolitica := SolicitudSeleccionPoliticaCifradoBorrador{
		Reserva: proyeccion, Control: reserva, Material: ejecutar.material,
		SelladoMotivo: sellado, SolicitadaEn: instantePolitica,
	}
	if solicitudPolitica.Validar() != nil {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	politicaCifrado, err := s.politicasCifrado.SeleccionarPoliticaCifradoBorrador(
		ctx, solicitudPolitica,
	)
	if err != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
	}
	if !politicaCifrado.validaPara(solicitudPolitica) {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	instantePerfil := s.ahora()
	solicitudPerfil := SolicitudResolucionPerfilCifradoBorrador{
		Reserva: proyeccion, Control: reserva, Material: ejecutar.material,
		SelladoMotivo: sellado, PoliticaEsperada: politicaCifrado,
		SolicitadaEn: instantePerfil,
	}
	if solicitudPerfil.Validar() != nil {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	resolucionPerfil, err := s.perfilesCifrado.ResolverPerfilCifradoBorrador(ctx, solicitudPerfil)
	if err != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
	}
	if resolucionPerfil.ValidarPara(solicitudPerfil) != nil {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	perfilCifrado := resolucionPerfil.Perfil
	instanteCifrado := s.ahora()
	solicitudCifrado, err := nuevaSolicitudCifradoBorrador(
		ejecutar.version, proyeccion, reserva, ejecutar.material, sellado,
		resolucionPerfil, s.procedencia, base.correlacionRef, instanteCifrado,
	)
	if err != nil {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	cifrado, err := s.cifrador.CifrarBorrador(ctx, solicitudCifrado)
	if err != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
	}
	if !cifrado.validaPara(solicitudCifrado) {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	instanteConfirmacion := s.ahora()
	actor, err = validarActorBorrador(base.actor, base.vinculo, instanteConfirmacion, base.correlacionRef)
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	solicitud := SolicitudConfirmacionBorrador{
		Reserva: proyeccion, Control: reserva, Version: ejecutar.version, Material: ejecutar.material,
		Actor: actor, CorrelacionRef: base.correlacionRef, Concesion: concesion,
		SelladoMotivo: sellado, PerfilCifrado: perfilCifrado,
		ResolucionPerfilCifrado: resolucionPerfil,
		Cifrado:                 cifrado, Procedencia: s.procedencia, SolicitadaEn: instanteConfirmacion,
	}
	if solicitud.Validar() != nil {
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	resultado, err := s.confirmador.ConfirmarBorrador(ctx, solicitud)
	if err != nil && resultado == (ResultadoConfirmacionAtomica{}) {
		// Un fallo de transporte sin veredicto no demuestra rollback. El COMMIT
		// pudo consumarse y la unica salida segura es reconciliar por el diario.
		return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
	}
	if validacion := resultado.ValidarPara(solicitud); validacion != nil {
		if resultado.Estado == ResultadoDiarioConfirmado {
			return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, validacion)
		}
		return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
	}
	switch resultado.Estado {
	case ResultadoDiarioConfirmado:
		if err != nil || !reciboProyectadoValido(resultado.Recibo, proyeccion.IdentidadPrimaria) ||
			!reciboCoincideReserva(resultado.Recibo, proyeccion, reserva, sellado) ||
			!procedenciasActoCoinciden(resultado.Recibo.Procedencia, s.procedencia) {
			return ProyeccionReciboBorrador{}, ErrOperacionBorradorIndeterminada
		}
		if err := s.verificador.VerificarReciboBorrador(ctx, resultado.Recibo); err != nil {
			return ProyeccionReciboBorrador{}, errors.Join(
				ErrOperacionBorradorIndeterminada, ErrResultadoBorradorInseguro, err,
			)
		}
		return resultado.Recibo, nil
	case ResultadoDiarioNoAplicado:
		return ProyeccionReciboBorrador{}, errors.Join(ErrConfirmacionBorradorNoAplicada, err)
	default:
		return ProyeccionReciboBorrador{}, errors.Join(ErrOperacionBorradorIndeterminada, err)
	}
}

func (s *ServicioBorradores) resolverResultadoDiario(
	ctx context.Context,
	r ResultadoOperacionDiario,
	identidad ProyeccionIdentidadOperacion,
) (ProyeccionReciboBorrador, error) {
	if !resultadoDiarioValido(r) {
		return ProyeccionReciboBorrador{}, ErrReservaBorradorInvalida
	}
	switch r.Estado {
	case ResultadoDiarioConflicto:
		return ProyeccionReciboBorrador{}, puertosbolsa.ErrClaveIdempotenciaConvocatoriaReusada
	case ResultadoDiarioConfirmado:
		if r.Recibo == nil || !reciboProyectadoValido(*r.Recibo, identidad) ||
			!procedenciasActoCoinciden(r.Recibo.Procedencia, s.procedencia) {
			return ProyeccionReciboBorrador{}, ErrResultadoBorradorInseguro
		}
		if err := s.verificador.VerificarReciboBorrador(ctx, *r.Recibo); err != nil {
			return ProyeccionReciboBorrador{}, errors.Join(
				ErrOperacionBorradorIndeterminada, ErrResultadoBorradorInseguro, err,
			)
		}
		return *r.Recibo, nil
	case ResultadoDiarioNoAplicado:
		return ProyeccionReciboBorrador{}, ErrConfirmacionBorradorNoAplicada
	case ResultadoDiarioReservado, ResultadoDiarioEnCurso:
		return ProyeccionReciboBorrador{}, ErrOperacionBorradorEnCurso
	default:
		return ProyeccionReciboBorrador{}, ErrOperacionBorradorIndeterminada
	}
}

func resultadoDiarioValido(r ResultadoOperacionDiario) bool {
	switch r.Estado {
	case ResultadoDiarioAusente, ResultadoDiarioConflicto:
		return r.Revision == 0 && r.Cercado == 0 && r.ArrendamientoIniciaEn.IsZero() &&
			r.ArrendamientoVenceEn.IsZero() && r.Recibo == nil
	case ResultadoDiarioReservado, ResultadoDiarioEnCurso, ResultadoDiarioIndeterminado,
		ResultadoDiarioConfirmado, ResultadoDiarioNoAplicado:
		if r.Revision == 0 || r.Cercado == 0 || !instanteOperacionCanonico(r.ArrendamientoIniciaEn) ||
			!instanteOperacionCanonico(r.ArrendamientoVenceEn) ||
			!r.ArrendamientoVenceEn.After(r.ArrendamientoIniciaEn) ||
			r.ArrendamientoVenceEn.Sub(r.ArrendamientoIniciaEn) > DuracionMaximaArrendamientoDiario ||
			(r.Estado == ResultadoDiarioConfirmado) != (r.Recibo != nil) {
			return false
		}
		return r.Recibo == nil || r.Recibo.RevisionConfirmada == r.Revision &&
			r.Recibo.CercadoConfirmado == r.Cercado &&
			r.Recibo.ArrendamientoIniciaEn.Equal(r.ArrendamientoIniciaEn) &&
			r.Recibo.ArrendamientoVenceEn.Equal(r.ArrendamientoVenceEn) &&
			reciboProyectadoValido(*r.Recibo, r.Recibo.IdentidadPrimaria)
	default:
		return false
	}
}

func reciboCoincideReserva(
	r ProyeccionReciboBorrador,
	p ProyeccionReservaDecision,
	control ResultadoOperacionDiario,
	sellado ProyeccionSelladoMotivoBorrador,
) bool {
	return identidadesProyectadasCoinciden(r.IdentidadPrimaria, p.IdentidadPrimaria) &&
		r.Decision == p.Decision && r.SelladoMotivo == sellado &&
		r.RevisionConfirmada > control.Revision && r.CercadoConfirmado == control.Cercado &&
		r.ArrendamientoIniciaEn.Equal(control.ArrendamientoIniciaEn) &&
		r.ArrendamientoVenceEn.Equal(control.ArrendamientoVenceEn)
}

func (s *ServicioBorradores) comprometerMotivo(
	ctx context.Context,
	accion string,
	version dominiobolsa.VersionConvocatoriaGobernada,
	principal, correlacion, motivo string,
	instante time.Time,
) (puertosbolsa.CompromisoMotivoGobiernoConvocatoria, error) {
	semantica := puertosbolsa.SolicitudSemanticaMotivoGobiernoConvocatoria{
		DominioCriptografico: puertosbolsa.DominioCriptograficoMotivoGobiernoConvocatoriaV1,
		Accion:               accion, ConvocatoriaRef: version.Referencia(), PrincipalRef: principal,
		CorrelacionRef: correlacion, Motivo: motivo, SolicitadaEn: instante,
	}
	solicitud, err := puertosbolsa.NuevaSolicitudComprometerMotivoGobiernoConvocatoria(semantica)
	if err != nil {
		return puertosbolsa.CompromisoMotivoGobiernoConvocatoria{}, err
	}
	hmac, err := s.comprometedor.ComprometerMotivo(ctx, solicitud)
	if err != nil {
		return puertosbolsa.CompromisoMotivoGobiernoConvocatoria{}, err
	}
	return puertosbolsa.NuevoCompromisoMotivoGobiernoConvocatoria(semantica, hmac)
}

func validarActorBorrador(
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	instante time.Time,
	correlacion string,
) (dominiovec.ContextoActor, error) {
	copia, errActor := actor.Clonar()
	datos, errVinculo := vinculo.Datos()
	superficieInterna := errVinculo == nil &&
		datos.Superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 && !datos.CuentaPrivilegiada
	superficiePrivilegiada := errVinculo == nil &&
		datos.Superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 && datos.CuentaPrivilegiada
	if errActor != nil || errVinculo != nil || !instanteOperacionCanonico(instante) ||
		actor.Principal.AuthMethod == dominiovec.AuthMethodDemo ||
		actor.Principal.AuthAssurance != dominiovec.AuthAssuranceHigh ||
		datos.GarantiaObservada != dominiovec.AuthAssuranceHigh || datos.MetodoObservado == dominiovec.AuthMethodDemo ||
		(!superficieInterna && !superficiePrivilegiada) || vinculo.ValidarPara(copia) != nil ||
		!vinculo.VigenteEn(instante, copia) || correlacion == "" {
		return dominiovec.ContextoActor{}, errors.Join(dominiovec.ErrAutorizacionDenegada, ErrOrdenBorradorInvalida)
	}
	return copia, nil
}

func (s *ServicioBorradores) ahora() time.Time {
	return s.reloj.Ahora().UTC().Truncate(time.Microsecond)
}

func (s *ServicioBorradores) validarContexto(ctx context.Context) error {
	if s == nil || ctx == nil || dependenciaNulaBorrador(s.reloj) ||
		dependenciaNulaBorrador(s.diario) || dependenciaNulaBorrador(s.perfilesCifrado) ||
		dependenciaNulaBorrador(s.cifrador) ||
		dependenciaNulaBorrador(s.confirmador) {
		return ErrServicioBorradoresInvalido
	}
	return ctx.Err()
}

func dependenciaNulaBorrador(valor any) bool {
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
