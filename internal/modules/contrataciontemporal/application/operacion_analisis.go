package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type SolicitudRegistrarAnalisis struct {
	AutenticacionRef  string
	SesionRef         string
	PerfilRef         string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	ArtefactoRef      string
	DatosFuncionales  ports.DatosFuncionalesOperacionAnalisis
}

type SolicitudRectificarAnalisis struct {
	AutenticacionRef         string
	SesionRef                string
	PerfilRef                string
	OrganizacionRef          string
	ExpedienteRef            string
	VersionEsperada          uint64
	ClaveIdempotencia        string
	ArtefactoRef             string
	DatosFuncionales         ports.DatosFuncionalesOperacionAnalisis
	MotivoRectificacionClave domain.ClaveCatalogo
}

type ServicioOperacionAnalisis struct {
	contextos     ports.ResolutorContextoAutorizacionAltaV3
	artefactos    ports.PreparadorArtefactoAnalisisO3
	sellador      ports.SelladorOperacionAnalisis
	preparaciones ports.PreparadorOperacionAnalisisIdempotente
	politicas     ports.ResolutorPoliticaOperacionAnalisis
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2
	autorizador   puertosvec.AutorizadorSolicitudLigadaV3
	reloj         ports.Reloj
	transaccion   ports.TransaccionOperacionesAnalisis
}

func NuevoServicioOperacionAnalisis(
	contextos ports.ResolutorContextoAutorizacionAltaV3,
	artefactos ports.PreparadorArtefactoAnalisisO3,
	sellador ports.SelladorOperacionAnalisis,
	preparaciones ports.PreparadorOperacionAnalisisIdempotente,
	politicas ports.ResolutorPoliticaOperacionAnalisis,
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	reloj ports.Reloj,
	transaccion ports.TransaccionOperacionesAnalisis,
) (*ServicioOperacionAnalisis, error) {
	if dependenciaNula(contextos) || dependenciaNula(artefactos) ||
		dependenciaNula(sellador) || dependenciaNula(preparaciones) ||
		dependenciaNula(politicas) || dependenciaNula(correlaciones) ||
		dependenciaNula(autorizador) || dependenciaNula(reloj) ||
		dependenciaNula(transaccion) {
		return nil, ErrServicioOperacionAnalisisInvalido
	}
	return &ServicioOperacionAnalisis{
		contextos:     contextos,
		artefactos:    artefactos,
		sellador:      sellador,
		preparaciones: preparaciones,
		politicas:     politicas,
		correlaciones: correlaciones,
		autorizador:   autorizador,
		reloj:         reloj,
		transaccion:   transaccion,
	}, nil
}

func (s *ServicioOperacionAnalisis) Registrar(
	ctx context.Context,
	solicitud SolicitudRegistrarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	return s.ejecutar(ctx, datosSolicitudOperacionAnalisis{
		operacion:         ports.OperacionRegistrarAnalisis,
		autenticacionRef:  solicitud.AutenticacionRef,
		sesionRef:         solicitud.SesionRef,
		perfilRef:         solicitud.PerfilRef,
		organizacionRef:   solicitud.OrganizacionRef,
		expedienteRef:     solicitud.ExpedienteRef,
		versionEsperada:   solicitud.VersionEsperada,
		claveIdempotencia: solicitud.ClaveIdempotencia,
		artefactoRef:      solicitud.ArtefactoRef,
		datosFuncionales:  solicitud.DatosFuncionales,
	})
}

func (s *ServicioOperacionAnalisis) Rectificar(
	ctx context.Context,
	solicitud SolicitudRectificarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	return s.ejecutar(ctx, datosSolicitudOperacionAnalisis{
		operacion:           ports.OperacionRectificarAnalisis,
		autenticacionRef:    solicitud.AutenticacionRef,
		sesionRef:           solicitud.SesionRef,
		perfilRef:           solicitud.PerfilRef,
		organizacionRef:     solicitud.OrganizacionRef,
		expedienteRef:       solicitud.ExpedienteRef,
		versionEsperada:     solicitud.VersionEsperada,
		claveIdempotencia:   solicitud.ClaveIdempotencia,
		artefactoRef:        solicitud.ArtefactoRef,
		datosFuncionales:    solicitud.DatosFuncionales,
		motivoRectificacion: solicitud.MotivoRectificacionClave,
	})
}

type datosSolicitudOperacionAnalisis struct {
	operacion           ports.TipoOperacionAnalisis
	autenticacionRef    string
	sesionRef           string
	perfilRef           string
	organizacionRef     string
	expedienteRef       string
	versionEsperada     uint64
	claveIdempotencia   string
	artefactoRef        string
	datosFuncionales    ports.DatosFuncionalesOperacionAnalisis
	motivoRectificacion domain.ClaveCatalogo
}

func (s *ServicioOperacionAnalisis) ejecutar(
	ctx context.Context,
	solicitud datosSolicitudOperacionAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	if s == nil || ctx == nil || !s.dependenciasValidas() {
		return ports.ReciboOperacionAnalisis{},
			ErrServicioOperacionAnalisisInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	ctxOperacion, cancelar := context.WithTimeout(
		ctx,
		ports.TiempoMaximoOperacionAnalisis,
	)
	defer cancelar()
	instanteInicial := instanteCanonico(s.reloj.Ahora())
	if solicitud.validar(instanteInicial) != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorSolicitud, nil)
	}

	resolverContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: solicitud.autenticacionRef,
		SesionRef:        solicitud.sesionRef,
		PerfilRef:        solicitud.perfilRef,
	}
	contextoAutorizacion, err := s.contextos.
		ResolverContextoAutorizacionAltaV3(ctxOperacion, resolverContexto)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			clasificarFalloAutorizacion(ctxOperacion, err)
	}
	if err := ctxOperacion.Err(); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	instanteContexto := instanteCanonico(s.reloj.Ahora())
	if contextoAutorizacion.ValidarPara(
		resolverContexto,
		instanteContexto,
	) != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorDenegacion, nil)
	}
	vinculo, err := contextoAutorizacion.Vinculo.Datos()
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorDenegacion, nil)
	}
	datosConsulta := ports.DatosPreimagenesConsultaOperacionAnalisis{
		Operacion:           solicitud.operacion,
		OrganizacionRef:     solicitud.organizacionRef,
		ExpedienteRef:       solicitud.expedienteRef,
		VersionExpediente:   solicitud.versionEsperada,
		ActorRef:            vinculo.PrincipalID,
		PerfilRef:           vinculo.PerfilActivoRef,
		ClaveIdempotencia:   solicitud.claveIdempotencia,
		ArtefactoRef:        solicitud.artefactoRef,
		DatosFuncionales:    solicitud.datosFuncionales,
		MotivoRectificacion: solicitud.motivoRectificacion,
	}
	preimagenesConsulta, err :=
		ports.NuevasPreimagenesConsultaOperacionAnalisis(datosConsulta)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorSolicitud, nil)
	}
	sellosConsulta, err := s.sellador.SellarOperacionAnalisis(
		ctxOperacion,
		preimagenesConsulta,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			errorDependenciaOperacionAnalisis(ctxOperacion)
	}
	solicitudConsulta, err :=
		ports.NuevaSolicitudConsultarOperacionAnalisisConfirmada(
			datosConsulta,
			sellosConsulta,
		)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	reciboConfirmado, encontrado, err := s.preparaciones.
		ConsultarOperacionAnalisisConfirmada(
			ctxOperacion,
			solicitudConsulta,
		)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			clasificarFalloPersistencia(ctxOperacion, err)
	}
	if encontrado {
		if reciboConfirmado.ValidarParaConsulta(solicitudConsulta) != nil {
			return ports.ReciboOperacionAnalisis{},
				nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
		}
		return reciboConfirmado, nil
	}

	solicitudArtefacto := ports.SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      solicitud.artefactoRef,
		OrganizacionRef:   solicitud.organizacionRef,
		ExpedienteRef:     solicitud.expedienteRef,
		VersionExpediente: solicitud.versionEsperada,
		DatosFuncionales:  solicitud.datosFuncionales,
		SolicitadaEn:      instanteContexto,
	}
	artefacto, err := s.artefactos.PrepararArtefactoAnalisis(
		ctxOperacion,
		solicitudArtefacto,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			errorDependenciaOperacionAnalisis(ctxOperacion)
	}
	if err := ctxOperacion.Err(); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	datosArtefacto, err := artefacto.DatosPara(solicitudArtefacto)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}

	preimagenes, err := ports.NuevasPreimagenesOperacionAnalisis(
		ports.DatosPreimagenesOperacionAnalisis{
			ClaveIdempotencia:   solicitud.claveIdempotencia,
			Operacion:           solicitud.operacion,
			ActorRef:            vinculo.PrincipalID,
			PerfilRef:           vinculo.PerfilActivoRef,
			MotivoRectificacion: solicitud.motivoRectificacion,
			SolicitudArtefacto:  solicitudArtefacto,
			Artefacto:           artefacto,
		},
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorSolicitud, nil)
	}
	sellos, err := s.sellador.SellarOperacionAnalisis(
		ctxOperacion,
		preimagenes,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			errorDependenciaOperacionAnalisis(ctxOperacion)
	}
	if err := ctxOperacion.Err(); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	if sellos.Validar() != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}

	solicitudPreparacion := ports.SolicitudPrepararOperacionAnalisis{
		Operacion:             solicitud.operacion,
		OrganizacionRef:       solicitud.organizacionRef,
		ExpedienteRef:         solicitud.expedienteRef,
		VersionExpediente:     solicitud.versionEsperada,
		ActorRef:              vinculo.PrincipalID,
		PerfilRef:             vinculo.PerfilActivoRef,
		ArtefactoRef:          datosArtefacto.ArtefactoRef,
		ArtefactoHuellaSHA256: datosArtefacto.ArtefactoHuellaSHA256,
		Sellos:                sellos,
		IdentidadConsulta:     solicitudConsulta,
	}
	preparacion, err := s.preparaciones.PrepararOperacionAnalisis(
		ctxOperacion,
		solicitudPreparacion,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			clasificarFalloPersistencia(ctxOperacion, err)
	}
	if err := ctxOperacion.Err(); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	datosPreparacion, err := preparacion.DatosPara(solicitudPreparacion)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	if datosPreparacion.Estado ==
		ports.PreparacionOperacionAnalisisConfirmada {
		return *datosPreparacion.ReciboConfirmado, nil
	}

	anterior := datosPreparacion.ExpedienteAnterior.Clonar()
	actorAnterior, err := actorAnalisisAnterior(
		anterior,
		solicitud.operacion,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorSolicitud, nil)
	}
	instantePolitica := instanteCanonico(s.reloj.Ahora())
	solicitudPolitica := ports.SolicitudResolverPoliticaOperacionAnalisis{
		Operacion:                solicitud.operacion,
		OrganizacionRef:          solicitud.organizacionRef,
		ExpedienteRef:            solicitud.expedienteRef,
		VersionExpediente:        solicitud.versionEsperada,
		Flujo:                    anterior.Flujo,
		FasePrevia:               anterior.FaseActual,
		EstadoPrevio:             anterior.EstadoActual,
		ActorRef:                 vinculo.PrincipalID,
		PerfilRef:                vinculo.PerfilActivoRef,
		ActorAnalisisAnteriorRef: actorAnterior,
		ArtefactoRef:             datosArtefacto.ArtefactoRef,
		ArtefactoHuellaSHA256:    datosArtefacto.ArtefactoHuellaSHA256,
		MotivoRectificacionClave: solicitud.motivoRectificacion,
		Instante:                 instantePolitica,
	}
	politica, err := s.politicas.ResolverPoliticaOperacionAnalisis(
		ctxOperacion,
		solicitudPolitica,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			errorDependenciaOperacionAnalisis(ctxOperacion)
	}
	if err := ctxOperacion.Err(); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	if politica.ValidarPara(solicitudPolitica) != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	if solicitud.operacion == ports.OperacionRectificarAnalisis &&
		actorAnterior == vinculo.PrincipalID {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorDenegacion, nil)
	}
	pruebasArtefacto, err := artefacto.PruebasParaO3(
		solicitudArtefacto,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	ordenConsumoFuentes := pruebasArtefacto.OrdenConsumoConjunto
	datosOrdenConsumo, err := ordenConsumoFuentes.Datos()
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	huellaAnalisisDerivado, err := ports.HuellaAnalisisDerivadoO3(
		solicitudArtefacto,
		artefacto,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}

	solicitudV3, err := s.nuevaSolicitudAutorizacion(
		ctxOperacion,
		contextoAutorizacion,
		datosPreparacion,
		politica,
		huellaAnalisisDerivado,
		datosOrdenConsumo.HuellaSHA256,
		solicitud.motivoRectificacion,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			errorDependenciaOperacionAnalisis(ctxOperacion)
	}
	decisionV3, confirmacionV3, err := s.autorizador.
		ExigirSolicitudLigadaV3(
			ctxOperacion,
			solicitudV3,
			contextoAutorizacion.Resultado,
		)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			clasificarResultadoAutorizador(
				ctxOperacion,
				solicitudV3,
				decisionV3,
			)
	}
	if err := ctxOperacion.Err(); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	concedida, _, errDecision := decisionV3.Resultado()
	if errDecision != nil || decisionV3.ValidarPara(solicitudV3) != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	if !concedida {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorDenegacion, nil)
	}
	instanteEfecto := instanteCanonico(s.reloj.Ahora())
	if contextoAutorizacion.ValidarPara(
		resolverContexto,
		instanteEfecto,
	) != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorDenegacion, nil)
	}
	if !autorizacionV3ValidaEn(
		solicitudV3,
		decisionV3,
		confirmacionV3,
		instanteEfecto,
	) {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	if artefacto.ValidarVigenciaEn(
		solicitudArtefacto,
		instanteEfecto,
	) != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	siguiente, err := aplicarOperacionAnalisis(
		solicitud,
		anterior,
		solicitudArtefacto,
		artefacto,
		politica,
		datosPreparacion.ReciboRef,
		instanteEfecto,
	)
	if err != nil {
		tipo := tipoErrorSolicitud
		if errors.Is(err, domain.ErrVersionEnConflicto) {
			tipo = tipoErrorConflicto
		}
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipo, nil)
	}
	orden, err := ports.NuevaOrdenConfirmarOperacionAnalisis(
		ports.DatosOrdenConfirmarOperacionAnalisis{
			SolicitudContexto:    resolverContexto,
			ContextoAutorizacion: contextoAutorizacion,
			SolicitudArtefacto:   solicitudArtefacto,
			Artefacto:            artefacto,
			OrdenConsumoFuentes:  ordenConsumoFuentes,
			SolicitudPreparacion: solicitudPreparacion,
			Preparacion:          preparacion,
			SolicitudPolitica:    solicitudPolitica,
			Politica:             politica,
			SolicitudV3:          solicitudV3,
			DecisionV3:           decisionV3,
			ConfirmacionV3:       confirmacionV3,
			InstanteEfecto:       instanteEfecto,
			ExpedienteSiguiente:  siguiente,
		},
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	if err := ctxOperacion.Err(); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	instantePrecommit := instanteCanonico(s.reloj.Ahora())
	if contextoAutorizacion.ValidarPara(
		resolverContexto,
		instantePrecommit,
	) != nil ||
		artefacto.ValidarVigenciaEn(
			solicitudArtefacto,
			instantePrecommit,
		) != nil ||
		!autorizacionV3ValidaEn(
			solicitudV3,
			decisionV3,
			confirmacionV3,
			instantePrecommit,
		) ||
		orden.ValidarConfirmacionDentroDeTransaccion(
			instantePrecommit,
		) != nil {
		return ports.ReciboOperacionAnalisis{},
			nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
	}
	recibo, err := s.transaccion.ConfirmarOperacionAnalisis(
		ctxOperacion,
		orden,
	)
	if recibo.ValidarParaOrdenDentroDeTransaccion(orden) == nil {
		return recibo, nil
	}
	if err != nil {
		return ports.ReciboOperacionAnalisis{},
			clasificarFalloPersistencia(ctxOperacion, err)
	}
	return ports.ReciboOperacionAnalisis{},
		nuevoErrorOperacionAnalisis(tipoErrorResultado, nil)
}
