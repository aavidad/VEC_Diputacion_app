// Package application orquesta los casos de uso del expediente de
// contratación temporal sin conocer HTTP, PostgreSQL ni el proveedor de
// identidad.
package application

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrServicioRegistroInvalido  = errors.New("contratacion temporal: servicio de registro invalido")
	ErrSolicitudRegistroInvalida = errors.New(
		"contratacion temporal: solicitud de registro invalida",
	)
	ErrResultadoRegistroNoConfiable = errors.New(
		"contratacion temporal: resultado de registro no confiable",
	)
)

// SolicitudRegistrarExpediente es neutral al canal. AutenticacionRef y
// SesionRef pueden proceder de web, escritorio, CLI o MCP, pero la autoridad
// común VEC las revalida y resuelve la cuenta, persona, perfil y garantía.
type SolicitudRegistrarExpediente struct {
	AutenticacionRef     string
	SesionRef            string
	PerfilRef            string
	OrganizacionRef      string
	ClaveIdempotencia    string
	NumeroExpedienteMOAD string
	Solicitud            domain.SolicitudCentro
}

// claveFinAltaConfirmada sólo existe dentro de aplicación: ni HTTP ni un
// llamador de otro paquete pueden insertar una confirmación en el contexto.
type claveFinAltaConfirmada struct{}

type finAltaConfirmada struct {
	organizacionRef string
	actorRef        string
	perfilRef       string
	motivoClave     domain.ClaveCatalogo
	ambitoHMAC      string
}

// PoliticaFinAltaConfirmadaPara permite a la composición comprobar un replay
// histórico sin transportar la política o la clave de idempotencia. El
// marcador nunca se serializa ni modifica la autorización V3.
func PoliticaFinAltaConfirmadaPara(
	ctx context.Context, organizacionRef, actorRef, perfilRef string,
	motivoClave domain.ClaveCatalogo,
) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	confirmacion, ok := ctx.Value(claveFinAltaConfirmada{}).(finAltaConfirmada)
	return ok && ports.SelloHMACSHA256Valido(confirmacion.ambitoHMAC) &&
		confirmacion.organizacionRef == organizacionRef &&
		confirmacion.actorRef == actorRef && confirmacion.perfilRef == perfilRef &&
		confirmacion.motivoClave == motivoClave
}

type ServicioRegistroSolicitud struct {
	contextosAutorizacion ports.ResolutorContextoAutorizacionAltaV3
	flujos                ports.ResolutorFlujoAlta
	huellas               ports.DerivadorHuellaAlta
	ambitos               ports.SelladorAmbitoIdempotencia
	motivos               ports.ResolutorMotivoAutorizacionAltaV3
	correlaciones         puertosvec.GeneradorReferenciasAutorizacionV2
	referencias           ports.GeneradorReferenciasAlta
	candidaturas          ports.ResolutorCandidaturaAlta
	huellasEfecto         ports.DerivadorHuellaEfectoAlta
	autorizador           puertosvec.AutorizadorSolicitudLigadaV3
	reloj                 ports.Reloj
	politicaNumero        *domain.PoliticaNumeroExpediente
	transaccion           ports.TransaccionAltasCandidata
	periodos              ports.PreparadorPeriodoModalidad
	recuperacion          ports.RecuperadorPoliticaFinConfirmada
}

func (s *ServicioRegistroSolicitud) ConfigurarPoliticaNumeroExpediente(
	politica domain.PoliticaNumeroExpediente,
) error {
	if s == nil || s.politicaNumero != nil || politica.Validar() != nil {
		return ErrServicioRegistroInvalido
	}
	s.politicaNumero = &politica
	return nil
}

func (s *ServicioRegistroSolicitud) ConfigurarRecuperacionPoliticaFin(
	recuperacion ports.RecuperadorPoliticaFinConfirmada,
) error {
	if s == nil || dependenciaNula(recuperacion) || s.recuperacion != nil {
		return ErrServicioRegistroInvalido
	}
	s.recuperacion = recuperacion
	return nil
}

func NuevoServicioRegistroSolicitud(
	contextosAutorizacion ports.ResolutorContextoAutorizacionAltaV3,
	flujos ports.ResolutorFlujoAlta,
	huellas ports.DerivadorHuellaAlta,
	ambitos ports.SelladorAmbitoIdempotencia,
	motivos ports.ResolutorMotivoAutorizacionAltaV3,
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2,
	referencias ports.GeneradorReferenciasAlta,
	candidaturas ports.ResolutorCandidaturaAlta,
	huellasEfecto ports.DerivadorHuellaEfectoAlta,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	reloj ports.Reloj,
	transaccion ports.TransaccionAltasCandidata,
	periodos ...ports.PreparadorPeriodoModalidad,
) (*ServicioRegistroSolicitud, error) {
	if dependenciaNula(contextosAutorizacion) || dependenciaNula(flujos) ||
		dependenciaNula(huellas) || dependenciaNula(ambitos) ||
		dependenciaNula(motivos) || dependenciaNula(correlaciones) ||
		dependenciaNula(referencias) || dependenciaNula(candidaturas) ||
		dependenciaNula(huellasEfecto) || dependenciaNula(autorizador) ||
		dependenciaNula(reloj) || dependenciaNula(transaccion) || len(periodos) > 1 {
		return nil, ErrServicioRegistroInvalido
	}
	servicio := &ServicioRegistroSolicitud{
		contextosAutorizacion: contextosAutorizacion,
		flujos:                flujos,
		huellas:               huellas,
		ambitos:               ambitos,
		motivos:               motivos,
		correlaciones:         correlaciones,
		referencias:           referencias,
		candidaturas:          candidaturas,
		huellasEfecto:         huellasEfecto,
		autorizador:           autorizador,
		reloj:                 reloj,
		transaccion:           transaccion,
	}
	if len(periodos) == 1 {
		servicio.periodos = periodos[0]
	}
	return servicio, nil
}

func (s *ServicioRegistroSolicitud) Registrar(
	ctx context.Context,
	solicitud SolicitudRegistrarExpediente,
) (ports.ReciboAlta, error) {
	if ctx == nil || s == nil || dependenciaNula(s.contextosAutorizacion) ||
		dependenciaNula(s.flujos) || dependenciaNula(s.huellas) ||
		dependenciaNula(s.ambitos) ||
		dependenciaNula(s.motivos) ||
		dependenciaNula(s.correlaciones) || dependenciaNula(s.referencias) ||
		dependenciaNula(s.candidaturas) || dependenciaNula(s.huellasEfecto) ||
		dependenciaNula(s.autorizador) || dependenciaNula(s.reloj) ||
		dependenciaNula(s.transaccion) {
		return ports.ReciboAlta{}, ErrServicioRegistroInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	instante := instanteCanonico(s.reloj.Ahora())
	if validarSolicitudRegistro(solicitud, instante) != nil {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ErrSolicitudRegistroInvalida,
		)
	}
	if solicitud.NumeroExpedienteMOAD != "" && !domain.NumeroExpedienteValido(solicitud.NumeroExpedienteMOAD) {
		return ports.ReciboAlta{}, ErrSolicitudRegistroInvalida
	}
	solicitudCentro, err := solicitud.Solicitud.Clonar()
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ErrSolicitudRegistroInvalida,
			err,
		)
	}
	resolverContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: solicitud.AutenticacionRef,
		SesionRef:        solicitud.SesionRef,
		PerfilRef:        solicitud.PerfilRef,
	}
	contextoAutorizacion, err := s.contextosAutorizacion.
		ResolverContextoAutorizacionAltaV3(ctx, resolverContexto)
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	instanteContexto := instanteCanonico(s.reloj.Ahora())
	if contextoAutorizacion.ValidarPara(resolverContexto, instanteContexto) != nil {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ports.ErrContextoAutorizacionV3Invalido,
		)
	}
	vinculo, err := contextoAutorizacion.Vinculo.Datos()
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ports.ErrContextoAutorizacionV3Invalido,
		)
	}
	solicitudAmbito := ports.SolicitudSellarAmbitoIdempotencia{
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ActorRef:          vinculo.PrincipalID,
		PerfilRef:         vinculo.PerfilActivoRef,
	}
	if solicitudAmbito.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrPreparacionAltaInvalida
	}
	ambitosHMAC, err := s.ambitos.SellarAmbitoIdempotencia(ctx, solicitudAmbito)
	if err != nil || ambitosHMAC.ValidarDominio("vec.contratacion-temporal.ambito-idempotencia") != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrPreparacionAltaInvalida, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	sinNumeroMOAD := solicitud.NumeroExpedienteMOAD == ""
	formatoRetirado := !sinNumeroMOAD && s.politicaNumero != nil &&
		s.politicaNumero.ValidarNumero(solicitud.NumeroExpedienteMOAD) != nil
	soloRecuperacion := sinNumeroMOAD || formatoRetirado
	consultarConfirmacion := soloRecuperacion || solicitudCentro.Periodo.Fin.IsZero()
	var politicaAnterior domain.PoliticaFin
	var confirmada bool
	if consultarConfirmacion {
		if dependenciaNula(s.recuperacion) {
			if soloRecuperacion {
				return ports.ReciboAlta{}, ErrSolicitudRegistroInvalida
			}
			return ports.ReciboAlta{}, ErrServicioRegistroInvalido
		}
		politicaAnterior, confirmada, err = s.recuperacion.ConsultarPoliticaFinAltaConfirmada(
			ctx, ports.ConsultaPoliticaFinAltaConfirmada{
				AmbitosHMAC: ambitosHMAC, OrganizacionRef: solicitud.OrganizacionRef,
				ActorRef: vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef,
			},
		)
		if err != nil {
			return ports.ReciboAlta{}, errors.Join(ports.ErrPersistenciaNoDisponible, err)
		}
		if soloRecuperacion && !confirmada {
			return ports.ReciboAlta{}, ErrSolicitudRegistroInvalida
		}
	}
	ctxFlujo := ctx
	if solicitudCentro.Periodo.Fin.IsZero() {
		if confirmada {
			if politicaAnterior != (domain.PoliticaFin{}) && politicaAnterior.Validar() != nil {
				return ports.ReciboAlta{}, ErrResultadoRegistroNoConfiable
			}
			if solicitudCentro.Periodo.PoliticaFin != (domain.PoliticaFin{}) &&
				solicitudCentro.Periodo.PoliticaFin != politicaAnterior {
				return ports.ReciboAlta{}, ports.ErrClaveIdempotenciaUsada
			}
			solicitudCentro.Periodo.PoliticaFin = politicaAnterior
			ambitos, err := ambitosHMAC.Datos()
			if err != nil {
				return ports.ReciboAlta{}, ErrResultadoRegistroNoConfiable
			}
			ctxFlujo = context.WithValue(ctx, claveFinAltaConfirmada{}, finAltaConfirmada{
				organizacionRef: solicitud.OrganizacionRef,
				actorRef:        vinculo.PrincipalID, perfilRef: vinculo.PerfilActivoRef,
				motivoClave: solicitudCentro.MotivoClave, ambitoHMAC: ambitos.Activo.Valor,
			})
		} else if solicitudCentro.Periodo.PoliticaFin == (domain.PoliticaFin{}) {
			if dependenciaNula(s.periodos) {
				return ports.ReciboAlta{}, ErrServicioRegistroInvalido
			}
			periodo, err := s.periodos.PrepararPeriodoModalidad(ctx, solicitudCentro.MotivoClave, solicitudCentro.Periodo)
			if err != nil {
				return ports.ReciboAlta{}, ErrSolicitudRegistroInvalida
			}
			solicitudCentro.Periodo = periodo
		}
		if solicitudCentro.Periodo.Validar() != nil {
			return ports.ReciboAlta{}, ErrSolicitudRegistroInvalida
		}
	}

	resolverFlujo := ports.SolicitudResolverFlujo{
		OrganizacionRef: solicitud.OrganizacionRef,
		CentroRef:       solicitudCentro.CentroRef,
		CategoriaRef:    solicitudCentro.CategoriaRef,
		MotivoClave:     solicitudCentro.MotivoClave,
		Instante:        instanteContexto,
	}
	if resolverFlujo.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrFlujoNoDisponible
	}
	configuracion, err := s.flujos.ResolverFlujoAlta(ctxFlujo, resolverFlujo)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	if configuracion.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrFlujoNoDisponible
	}

	solicitudParaHuella, err := solicitudCentro.Clonar()
	if err != nil {
		return ports.ReciboAlta{}, ErrSolicitudRegistroInvalida
	}
	materialHuella := ports.MaterialHuellaAlta{
		NumeroExpedienteMOAD: solicitud.NumeroExpedienteMOAD,
		OrganizacionRef:      solicitud.OrganizacionRef,
		ActorRef:             vinculo.PrincipalID,
		PerfilRef:            vinculo.PerfilActivoRef,
		Flujo:                configuracion.Flujo,
		Solicitud:            solicitudParaHuella,
	}
	if materialHuella.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrPreparacionAltaInvalida
	}
	huellasHMAC, err := s.huellas.DerivarHuellaAlta(
		ctx,
		materialHuella,
	)
	if err != nil || huellasHMAC.ValidarDominio(
		"vec.contratacion-temporal.huella-peticion",
	) != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrPreparacionAltaInvalida, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	ambitoHMAC, huellaHMAC, err := ports.ParActivoColeccionesHMACAlta(
		ambitosHMAC,
		huellasHMAC,
	)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	numeroPropuesto := solicitud.NumeroExpedienteMOAD
	if sinNumeroMOAD {
		if s.politicaNumero == nil || s.politicaNumero.Validar() != nil {
			return ports.ReciboAlta{}, ErrServicioRegistroInvalido
		}
		// CT47 exige una propuesta técnica válida incluso al recuperar. Esta
		// muestra del catálogo nunca puede confirmarse ni presentarse como MOAD.
		numeroPropuesto = s.politicaNumero.Ejemplo
	}
	referencias, err := s.referencias.GenerarReferenciasAlta(ctx, numeroPropuesto)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if referencias.NumeroVisible != numeroPropuesto {
		return ports.ReciboAlta{}, ErrResultadoRegistroNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	reservaRef, err := s.referencias.NuevaReferenciaReservaAlta(ctx)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	instanteCandidatura := instanteCanonico(s.reloj.Ahora())
	propuesta, err := ports.NuevaCandidaturaAlta(ports.DatosCandidaturaAlta{
		ReservaRef:             reservaRef,
		Referencias:            referencias,
		AmbitoIdempotenciaHMAC: ambitoHMAC,
		HuellaPeticionHMAC:     huellaHMAC,
		OrganizacionRef:        solicitud.OrganizacionRef,
		ActorRef:               vinculo.PrincipalID,
		PerfilRef:              vinculo.PerfilActivoRef,
		InstanteEfecto:         instanteCandidatura,
	})
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrPreparacionAltaInvalida,
			err,
		)
	}
	solicitudCandidatura, err := ports.NuevaSolicitudResolverCandidaturaAlta(
		ports.DatosSolicitudResolverCandidaturaAlta{
			AmbitosIdempotenciaHMAC: ambitosHMAC,
			HuellasPeticionHMAC:     huellasHMAC,
			OrganizacionRef:         solicitud.OrganizacionRef,
			ActorRef:                vinculo.PrincipalID,
			PerfilRef:               vinculo.PerfilActivoRef,
			Propuesta:               propuesta,
		},
	)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	var candidatura ports.CandidaturaAlta
	if soloRecuperacion {
		recuperador, ok := s.candidaturas.(ports.RecuperadorCandidaturaAlta)
		if !ok || dependenciaNula(recuperador) {
			return ports.ReciboAlta{}, ErrServicioRegistroInvalido
		}
		candidatura, err = recuperador.RecuperarCandidaturaAlta(ctx, solicitudCandidatura)
	} else {
		candidatura, err = s.candidaturas.ResolverCandidaturaAlta(ctx, solicitudCandidatura)
	}
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	if solicitudCandidatura.ValidarResultado(candidatura) != nil {
		return ports.ReciboAlta{}, ports.ErrPreparacionAltaInvalida
	}
	datosCandidatura, err := candidatura.Datos()
	if err != nil {
		return ports.ReciboAlta{}, ports.ErrPreparacionAltaInvalida
	}
	if soloRecuperacion && !datosCandidatura.Recuperada {
		return ports.ReciboAlta{}, ErrResultadoRegistroNoConfiable
	}
	if !datosCandidatura.Recuperada && !solicitudCentro.Periodo.Fin.IsZero() && s.periodos != nil {
		if _, err := s.periodos.PrepararPeriodoModalidad(ctx, solicitudCentro.MotivoClave, solicitudCentro.Periodo); err != nil {
			return ports.ReciboAlta{}, ErrSolicitudRegistroInvalida
		}
	}

	if !sinNumeroMOAD && datosCandidatura.Referencias.NumeroVisible != solicitud.NumeroExpedienteMOAD {
		return ports.ReciboAlta{}, ports.ErrClaveIdempotenciaUsada
	}
	var circuito *domain.CircuitoAdministrativo
	if configuracion.DefinicionCircuito != nil {
		inicial, err := domain.NuevoCircuitoAdministrativo(*configuracion.DefinicionCircuito)
		if err != nil {
			return ports.ReciboAlta{}, ports.ErrFlujoNoDisponible
		}
		circuito = &inicial
	}
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      datosCandidatura.Referencias.ExpedienteRef,
		OrganizacionRef: solicitud.OrganizacionRef,
		NumeroVisible:   datosCandidatura.Referencias.NumeroVisible,
		Flujo:           configuracion.Flujo,
		Circuito:        circuito,
		FaseInicial:     configuracion.FaseInicial,
		Solicitud:       solicitudCentro,
		Actuacion: domain.DatosActuacion{
			AccionClave:   configuracion.AccionInicial,
			ActorRef:      vinculo.PrincipalID,
			UnidadRef:     configuracion.UnidadInicialRef,
			ReciboRef:     datosCandidatura.Referencias.ReciboRef,
			RealizadaEn:   datosCandidatura.InstanteEfecto,
			FaseDestino:   configuracion.FaseInicial,
			EstadoDestino: domain.EstadoEnCurso,
		},
	})
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ErrSolicitudRegistroInvalida, err)
	}
	huellaEfecto, err := s.huellasEfecto.DerivarHuellaEfectoAlta(
		expediente,
		candidatura,
	)
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrOrdenAltaInvalida, err)
	}

	instanteAutorizacion := instanteCanonico(s.reloj.Ahora())
	resolverMotivo := ports.SolicitudResolverMotivoAutorizacionAltaV3{
		OrganizacionRef: solicitud.OrganizacionRef,
		Flujo:           configuracion.Flujo,
		MotivoClave:     solicitudCentro.MotivoClave,
		Instante:        instanteAutorizacion,
	}
	if resolverMotivo.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrAutorizacionDenegada
	}
	motivo, err := s.motivos.ResolverMotivoAutorizacionAltaV3(ctxFlujo, resolverMotivo)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ReciboAlta{}, errContexto
	}
	if err != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return ports.ReciboAlta{}, errors.Join(
			ports.ErrAutorizacionDenegada,
			ports.ErrMotivoAutorizacionNoDisponible,
			err,
		)
	}
	correlacionV3, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx,
		s.correlaciones,
	)
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	solicitudAutorizacionV3, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contextoAutorizacion.Vinculo,
			ReferenciaMotivo:          motivo,
			Accion:                    ports.AccionCrearSolicitud,
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: ambitoHMAC,
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoExpediente,
				Ambitos: map[string]string{
					"organizacion_ref": solicitud.OrganizacionRef,
					"centro_ref":       solicitudCentro.CentroRef,
					"categoria_ref":    solicitudCentro.CategoriaRef,
				},
				Atributos: map[string]string{
					"flujo_ref": configuracion.Flujo.DefinicionRef,
					"flujo_version": strconv.FormatUint(
						configuracion.Flujo.Version,
						10,
					),
					"flujo_huella_sha256":                  configuracion.Flujo.HuellaSHA256,
					ports.AtributoHuellaPeticionHMACActiva: huellaHMAC,
					ports.AtributoHuellaEfectoAltaSHA256:   huellaEfecto,
				},
			},
			Finalidad:   ports.FinalidadCrearSolicitud,
			Correlacion: correlacionV3,
		},
	)
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrAutorizacionDenegada, err)
	}
	decisionV3, confirmacionV3, err := s.autorizador.ExigirSolicitudLigadaV3(
		ctx,
		solicitudAutorizacionV3,
		contextoAutorizacion.Resultado,
	)
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ports.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	instanteConfirmacion := instanteCanonico(s.reloj.Ahora())
	if contextoAutorizacion.ValidarPara(
		resolverContexto,
		instanteConfirmacion,
	) != nil || !autorizacionV3ValidaEn(
		solicitudAutorizacionV3,
		decisionV3,
		confirmacionV3,
		instanteConfirmacion,
	) {
		return ports.ReciboAlta{}, ports.ErrAutorizacionDenegada
	}
	orden, err := ports.NuevaOrdenConfirmarAltaCandidata(
		ports.DatosOrdenConfirmarAltaCandidata{
			Expediente:              expediente,
			SolicitudAutorizacionV3: solicitudAutorizacionV3,
			DecisionAutorizacionV3:  decisionV3,
			ConfirmacionRegistroV3:  confirmacionV3,
			AmbitosIdempotenciaHMAC: ambitosHMAC,
			HuellasPeticionHMAC:     huellasHMAC,
			Candidatura:             candidatura,
		})
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	recibo, err := s.transaccion.ConfirmarAltaCandidata(ctx, orden)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	// La transacción es la frontera de efecto. Una cancelación observada tras
	// un COMMIT confirmado no puede convertir el éxito durable en un fallo
	// ambiguo y provocar que el cliente repita la operación.
	if recibo.ValidarPara(expediente) != nil {
		return ports.ReciboAlta{}, ErrResultadoRegistroNoConfiable
	}
	return recibo, nil
}

func autorizacionV3ValidaEn(
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	instante time.Time,
) bool {
	concedida, _, err := decision.Resultado()
	emitidaEn, validaHasta, errVentana := decision.VentanaValidez()
	huellaDecision, errHuella := dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	return domain.InstanteUTCCanonico(instante) &&
		err == nil && errVentana == nil && errHuella == nil &&
		errConfirmacion == nil && concedida &&
		decision.ValidarPara(solicitud) == nil &&
		datosConfirmacion.DecisionHuellaSHA256 == huellaDecision &&
		datosConfirmacion.EmitidaEn.Equal(emitidaEn) &&
		datosConfirmacion.ValidaHasta.Equal(validaHasta) &&
		confirmacion.DentroDeVentanaEn(instante)
}

func validarSolicitudRegistro(
	solicitud SolicitudRegistrarExpediente,
	instante time.Time,
) error {
	resolverContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: solicitud.AutenticacionRef,
		SesionRef:        solicitud.SesionRef,
		PerfilRef:        solicitud.PerfilRef,
	}
	if !domain.InstanteUTCCanonico(instante) ||
		resolverContexto.Validar() != nil ||
		!domain.ReferenciaOpacaValida(solicitud.OrganizacionRef) ||
		!ports.ClaveIdempotenciaValida(solicitud.ClaveIdempotencia) ||
		solicitud.Solicitud.Validar() != nil {
		return ErrSolicitudRegistroInvalida
	}
	return nil
}

func instanteCanonico(valor time.Time) time.Time {
	if valor.IsZero() {
		return time.Time{}
	}
	return valor.UTC().Truncate(time.Microsecond)
}

func dependenciaNula(dependencia any) bool {
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
