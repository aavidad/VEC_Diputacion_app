package application

import (
	"context"
	"errors"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const tiempoMaximoInformeJuridico = 15 * time.Second

var (
	ErrServicioInformeJuridicoInvalido = errors.New(
		"contratacion temporal: servicio de informe juridico invalido",
	)
	ErrSolicitudInformeJuridicoInvalida = errors.New(
		"contratacion temporal: solicitud de informe juridico invalida",
	)
	ErrInformeJuridicoDenegado = errors.New(
		"contratacion temporal: informe juridico denegado",
	)
	ErrGeneracionInformeJuridicoNoDisponible = errors.New(
		"contratacion temporal: generacion de informe juridico no disponible",
	)
	ErrResultadoInformeJuridicoNoConfiable = errors.New(
		"contratacion temporal: resultado de informe juridico no confiable",
	)
)

type SolicitudEmitirInformeJuridico struct {
	AutenticacionRef  string
	SesionRef         string
	PerfilRef         string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
}

func (s SolicitudEmitirInformeJuridico) validar() error {
	if (ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: s.AutenticacionRef,
		SesionRef:        s.SesionRef,
		PerfilRef:        s.PerfilRef,
	}).Validar() != nil || !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!ports.VersionOperacionAnalisisConIncrementoValida(s.VersionEsperada) ||
		!ports.ClaveIdempotenciaValida(s.ClaveIdempotencia) {
		return ErrSolicitudInformeJuridicoInvalida
	}
	return nil
}

type ServicioInformesJuridicos struct {
	contextos     ports.ResolutorContextoAutorizacionAltaV3
	ambitos       ports.SelladorAmbitoInformeJuridico
	huellas       ports.DerivadorHuellaInformeJuridico
	preparaciones ports.PreparadorInformeJuridicoIdempotente
	configuracion ports.ResolutorConfiguracionInformeJuridico
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2
	autorizador   puertosvec.AutorizadorSolicitudLigadaV3
	generador     ports.GeneradorDocumentoInformeJuridico
	reloj         ports.Reloj
	transaccion   ports.TransaccionInformesJuridicos
}

func NuevoServicioInformesJuridicos(
	contextos ports.ResolutorContextoAutorizacionAltaV3,
	ambitos ports.SelladorAmbitoInformeJuridico,
	huellas ports.DerivadorHuellaInformeJuridico,
	preparaciones ports.PreparadorInformeJuridicoIdempotente,
	configuracion ports.ResolutorConfiguracionInformeJuridico,
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	generador ports.GeneradorDocumentoInformeJuridico,
	reloj ports.Reloj,
	transaccion ports.TransaccionInformesJuridicos,
) (*ServicioInformesJuridicos, error) {
	dependencias := []any{
		contextos, ambitos, huellas, preparaciones, configuracion,
		correlaciones, autorizador, generador, reloj, transaccion,
	}
	for _, dependencia := range dependencias {
		if dependenciaNula(dependencia) {
			return nil, ErrServicioInformeJuridicoInvalido
		}
	}
	return &ServicioInformesJuridicos{
		contextos: contextos, ambitos: ambitos, huellas: huellas,
		preparaciones: preparaciones, configuracion: configuracion,
		correlaciones: correlaciones, autorizador: autorizador,
		generador: generador, reloj: reloj, transaccion: transaccion,
	}, nil
}

func (s *ServicioInformesJuridicos) Emitir(
	ctx context.Context,
	solicitud SolicitudEmitirInformeJuridico,
) (ports.ReciboInformeJuridico, error) {
	if s == nil || ctx == nil || solicitud.validar() != nil {
		return ports.ReciboInformeJuridico{}, ErrSolicitudInformeJuridicoInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboInformeJuridico{}, err
	}
	ctxOperacion, cancelar := context.WithTimeout(ctx, tiempoMaximoInformeJuridico)
	defer cancelar()

	contextoSolicitud := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: solicitud.AutenticacionRef,
		SesionRef:        solicitud.SesionRef,
		PerfilRef:        solicitud.PerfilRef,
	}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(
		ctxOperacion, contextoSolicitud,
	)
	if err != nil || contexto.ValidarPara(
		contextoSolicitud, instanteCanonico(s.reloj.Ahora()),
	) != nil {
		return ports.ReciboInformeJuridico{}, ErrInformeJuridicoDenegado
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return ports.ReciboInformeJuridico{}, ErrInformeJuridicoDenegado
	}
	material := ports.MaterialHuellaInformeJuridico{
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		VersionExpediente: solicitud.VersionEsperada,
		ActorRef:          vinculo.PrincipalID,
		PerfilRef:         vinculo.PerfilActivoRef,
	}
	ambitos, err := s.ambitos.SellarAmbitoInformeJuridico(
		ctxOperacion,
		ports.SolicitudSellarAmbitoIdempotencia{
			ClaveIdempotencia: solicitud.ClaveIdempotencia,
			OrganizacionRef:   solicitud.OrganizacionRef,
			ActorRef:          material.ActorRef,
			PerfilRef:         material.PerfilRef,
		},
	)
	if err != nil {
		return ports.ReciboInformeJuridico{}, clasificarFalloInformeJuridico(ctxOperacion, err)
	}
	huellas, err := s.huellas.DerivarHuellaInformeJuridico(ctxOperacion, material)
	if err != nil {
		return ports.ReciboInformeJuridico{}, clasificarFalloInformeJuridico(ctxOperacion, err)
	}
	preparar := ports.SolicitudPrepararInformeJuridico{
		ClaveIdempotencia:   solicitud.ClaveIdempotencia,
		AmbitosHMAC:         ambitos,
		HuellasPeticionHMAC: huellas,
		Material:            material,
	}
	preparacion, err := s.preparaciones.PrepararInformeJuridico(
		ctxOperacion, preparar,
	)
	if err != nil {
		return ports.ReciboInformeJuridico{}, clasificarFalloInformeJuridico(ctxOperacion, err)
	}
	if preparacion.ValidarPara(preparar) != nil {
		return ports.ReciboInformeJuridico{}, ErrResultadoInformeJuridicoNoConfiable
	}
	if preparacion.Estado == ports.PreparacionInformeJuridicoConfirmada {
		return *preparacion.ReciboConfirmado, nil
	}

	instanteConfiguracion := instanteCanonico(s.reloj.Ahora())
	solicitudConfiguracion := ports.SolicitudResolverConfiguracionInformeJuridico{
		OrganizacionRef:   material.OrganizacionRef,
		ExpedienteRef:     material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente,
		ActorRef:          material.ActorRef,
		PerfilRef:         material.PerfilRef,
		FaseActual:        preparacion.Expediente.FaseActual,
		EstadoActual:      preparacion.Expediente.EstadoActual,
		UnidadAsignadaRef: preparacion.Expediente.Asignacion.UnidadRef,
		Instante:          instanteConfiguracion,
	}
	configuracion, err := s.configuracion.ResolverConfiguracionInformeJuridico(
		ctxOperacion, solicitudConfiguracion,
	)
	if err != nil || configuracion.ValidarPara(
		solicitudConfiguracion, instanteConfiguracion,
	) != nil {
		return ports.ReciboInformeJuridico{}, ErrInformeJuridicoDenegado
	}
	borrador, err := domain.NuevoBorradorInformeJuridico(
		domain.DatosBorradorInformeJuridico{
			Canon:                     domain.CanonBorradorInformeJuridicoV1(),
			ExpedienteRef:             material.ExpedienteRef,
			VersionEsperadaExpediente: material.VersionExpediente,
			Plantilla:                 configuracion.Plantilla,
			ReferenciasNormativas:     configuracion.ReferenciasNormativas,
			Anexos:                    configuracion.Anexos,
		},
	)
	if err != nil {
		return ports.ReciboInformeJuridico{}, ErrGeneracionInformeJuridicoNoDisponible
	}
	ambitoActivo, huellaActiva, err := ports.ParActivoColeccionesHMAC(
		ambitos, ports.DominioAmbitoIdempotenciaInformeJuridico,
		huellas, ports.DominioHuellaPeticionInformeJuridico,
	)
	if err != nil {
		return ports.ReciboInformeJuridico{}, ErrResultadoInformeJuridicoNoConfiable
	}
	solicitudV3, err := s.nuevaSolicitudAutorizacion(
		ctxOperacion, contexto, material, preparacion, configuracion,
		borrador, ambitoActivo, huellaActiva,
	)
	if err != nil {
		return ports.ReciboInformeJuridico{}, ErrInformeJuridicoDenegado
	}
	decisionV3, confirmacionV3, err := s.autorizador.ExigirSolicitudLigadaV3(
		ctxOperacion, solicitudV3, contexto.Resultado,
	)
	instanteEfecto := instanteCanonico(s.reloj.Ahora())
	if err != nil || configuracion.ValidarPara(
		solicitudConfiguracion, instanteEfecto,
	) != nil || !autorizacionV3ValidaEn(
		solicitudV3, decisionV3, confirmacionV3, instanteEfecto,
	) {
		return ports.ReciboInformeJuridico{}, ErrInformeJuridicoDenegado
	}
	datosConfirmacion, err := confirmacionV3.Datos()
	if err != nil {
		return ports.ReciboInformeJuridico{}, ErrInformeJuridicoDenegado
	}

	solicitudDocumento := ports.SolicitudGenerarDocumentoInformeJuridico{
		DocumentoRef: preparacion.Referencias.DocumentoRef,
		Borrador:     borrador,
	}
	documento, err := s.generador.GenerarDocumentoInformeJuridico(
		ctxOperacion, solicitudDocumento,
	)
	if err != nil || documento.ValidarPara(solicitudDocumento) != nil {
		return ports.ReciboInformeJuridico{}, ErrGeneracionInformeJuridicoNoDisponible
	}
	informe := domain.InformeJuridicoEmitido{
		Borrador:              borrador.Estado(),
		InformeRef:            preparacion.Referencias.InformeRef,
		DocumentoRef:          documento.DocumentoRef,
		VersionDocumento:      documento.VersionDocumento,
		HuellaDocumentoSHA256: documento.HuellaDocumentoSHA256,
		EmitidoEn:             instanteEfecto,
	}
	expedienteSiguiente, err := preparacion.Expediente.RegistrarInformeJuridico(
		material.VersionExpediente,
		informe,
		domain.DatosActuacion{
			AccionClave:   domain.AccionEmitirInformeJuridico,
			ActorRef:      material.ActorRef,
			UnidadRef:     configuracion.UnidadEjecutoraRef,
			ReciboRef:     preparacion.Referencias.ReciboRef,
			RealizadaEn:   instanteEfecto,
			FaseDestino:   domain.FaseInformeJuridico,
			EstadoDestino: domain.EstadoEnCurso,
			DocumentosRef: []string{documento.DocumentoRef},
		},
	)
	if err != nil {
		return ports.ReciboInformeJuridico{}, ErrResultadoInformeJuridicoNoConfiable
	}
	recibo, err := s.transaccion.ConfirmarInformeJuridico(
		ctxOperacion,
		ports.OrdenConfirmarInformeJuridico{
			Preparacion:         preparacion,
			Configuracion:       configuracion,
			Borrador:            borrador,
			Documento:           documento,
			ExpedienteSiguiente: expedienteSiguiente,
			Evidencia: ports.EvidenciaAutorizacionInformeJuridico{
				Contexto: contexto, SolicitudV3: solicitudV3,
				DecisionV3: decisionV3, ConfirmacionV3: confirmacionV3,
			},
			InstanteEfecto: instanteEfecto,
		},
	)
	if err != nil {
		return ports.ReciboInformeJuridico{}, clasificarFalloInformeJuridico(ctxOperacion, err)
	}
	if recibo.ValidarParaPreparacion(preparacion) != nil ||
		recibo.HuellaDocumentoSHA256 != documento.HuellaDocumentoSHA256 ||
		recibo.HuellaBorradorSHA256 != borrador.HuellaSHA256() ||
		recibo.ContenidoDesarrollo != documento.ContenidoDesarrollo ||
		recibo.ConcesionV3DecisionRef != datosConfirmacion.DecisionRef {
		return ports.ReciboInformeJuridico{}, ErrResultadoInformeJuridicoNoConfiable
	}
	return recibo, nil
}

func (s *ServicioInformesJuridicos) nuevaSolicitudAutorizacion(
	ctx context.Context,
	contexto ports.ContextoAutorizacionAltaV3,
	material ports.MaterialHuellaInformeJuridico,
	preparacion ports.PreparacionInformeJuridico,
	configuracion ports.ConfiguracionInformeJuridico,
	borrador domain.BorradorInformeJuridico,
	ambitoActivo string,
	huellaActiva string,
) (dominiovec.SolicitudAutorizacionLigadaV3, error) {
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx, s.correlaciones,
	)
	if err != nil {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	return dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo:          configuracion.MotivoAutorizacion,
			Accion:                    ports.AccionEmitirInformeJuridico,
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: material.ExpedienteRef,
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoInformeJuridico,
				Ambitos: map[string]string{
					"organizacion_ref": material.OrganizacionRef,
					"expediente_ref":   material.ExpedienteRef,
					"fase_previa":      string(preparacion.Expediente.FaseActual),
					"estado_previo":    string(preparacion.Expediente.EstadoActual),
				},
				Atributos: map[string]string{
					"version_expediente":          strconv.FormatUint(material.VersionExpediente, 10),
					"configuracion_ref":           configuracion.DefinicionRef,
					"configuracion_version":       strconv.FormatUint(configuracion.DefinicionVersion, 10),
					"configuracion_huella_sha256": configuracion.DefinicionHuella,
					"plantilla_ref":               configuracion.Plantilla.PlantillaRef,
					"plantilla_version":           strconv.FormatUint(configuracion.Plantilla.Version, 10),
					"plantilla_huella_sha256":     configuracion.Plantilla.HuellaSHA256,
					"borrador_huella_sha256":      borrador.HuellaSHA256(),
					"ambito_idempotencia_hmac":    ambitoActivo,
					"huella_peticion_hmac":        huellaActiva,
				},
			},
			Finalidad:   string(configuracion.Finalidad),
			Correlacion: correlacion,
		},
	)
}

func clasificarFalloInformeJuridico(ctx context.Context, causa error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if errors.Is(causa, ports.ErrClaveIdempotenciaUsada) {
		return ports.ErrClaveIdempotenciaUsada
	}
	if errors.Is(causa, ErrInformeJuridicoDenegado) {
		return ErrInformeJuridicoDenegado
	}
	if errors.Is(causa, ports.ErrResultadoInformeJuridicoNoConfiable) {
		return ErrResultadoInformeJuridicoNoConfiable
	}
	return ports.ErrPersistenciaInformeJuridicoNoDisponible
}
