// Package application orquesta los casos de uso del expediente de
// contratación temporal sin conocer HTTP, PostgreSQL ni el proveedor de
// identidad.
package application

import (
	"context"
	"crypto/hmac"
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
	AutenticacionRef  string
	SesionRef         string
	PerfilRef         string
	CorrelacionRef    string
	OrganizacionRef   string
	ClaveIdempotencia string
	Solicitud         domain.SolicitudCentro
}

type ServicioRegistroSolicitud struct {
	contextosAutorizacion ports.ResolutorContextoAutorizacionAltaV3
	flujos                ports.ResolutorFlujoAlta
	huellas               ports.DerivadorHuellaAlta
	ambitos               ports.SelladorAmbitoIdempotencia
	motivos               ports.ResolutorMotivoAutorizacionAltaV3
	correlaciones         puertosvec.GeneradorReferenciasAutorizacionV2
	preparaciones         ports.PreparadorAltaIdempotente
	autorizador           puertosvec.AutorizadorSolicitudLigadaV3
	reloj                 ports.Reloj
	transaccion           ports.TransaccionAltas
}

func NuevoServicioRegistroSolicitud(
	contextosAutorizacion ports.ResolutorContextoAutorizacionAltaV3,
	flujos ports.ResolutorFlujoAlta,
	huellas ports.DerivadorHuellaAlta,
	ambitos ports.SelladorAmbitoIdempotencia,
	motivos ports.ResolutorMotivoAutorizacionAltaV3,
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2,
	preparaciones ports.PreparadorAltaIdempotente,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	reloj ports.Reloj,
	transaccion ports.TransaccionAltas,
) (*ServicioRegistroSolicitud, error) {
	if dependenciaNula(contextosAutorizacion) || dependenciaNula(flujos) ||
		dependenciaNula(huellas) || dependenciaNula(ambitos) ||
		dependenciaNula(motivos) || dependenciaNula(correlaciones) ||
		dependenciaNula(preparaciones) || dependenciaNula(autorizador) ||
		dependenciaNula(reloj) || dependenciaNula(transaccion) {
		return nil, ErrServicioRegistroInvalido
	}
	return &ServicioRegistroSolicitud{
		contextosAutorizacion: contextosAutorizacion,
		flujos:                flujos,
		huellas:               huellas,
		ambitos:               ambitos,
		motivos:               motivos,
		correlaciones:         correlaciones,
		preparaciones:         preparaciones,
		autorizador:           autorizador,
		reloj:                 reloj,
		transaccion:           transaccion,
	}, nil
}

func (s *ServicioRegistroSolicitud) Registrar(
	ctx context.Context,
	solicitud SolicitudRegistrarExpediente,
) (ports.ReciboAlta, error) {
	if ctx == nil || s == nil || dependenciaNula(s.contextosAutorizacion) ||
		dependenciaNula(s.flujos) || dependenciaNula(s.huellas) ||
		dependenciaNula(s.ambitos) || dependenciaNula(s.motivos) ||
		dependenciaNula(s.correlaciones) || dependenciaNula(s.preparaciones) ||
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
	configuracion, err := s.flujos.ResolverFlujoAlta(ctx, resolverFlujo)
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
		OrganizacionRef: solicitud.OrganizacionRef,
		ActorRef:        vinculo.PrincipalID,
		PerfilRef:       vinculo.PerfilActivoRef,
		Flujo:           configuracion.Flujo,
		Solicitud:       solicitudParaHuella,
	}
	if materialHuella.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrPreparacionAltaInvalida
	}
	huella, err := s.huellas.DerivarHuellaAlta(ctx, materialHuella)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
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
	ambitoHMAC, err := s.ambitos.SellarAmbitoIdempotencia(ctx, solicitudAmbito)
	if err != nil || !ports.SelloHMACSHA256Valido(ambitoHMAC) {
		return ports.ReciboAlta{}, errors.Join(ports.ErrPreparacionAltaInvalida, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
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
	motivo, err := s.motivos.ResolverMotivoAutorizacionAltaV3(ctx, resolverMotivo)
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
					"flujo_ref":           configuracion.Flujo.DefinicionRef,
					"flujo_version":       strconv.FormatUint(configuracion.Flujo.Version, 10),
					"flujo_huella_sha256": configuracion.Flujo.HuellaSHA256,
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
	instantePreparacion := instanteCanonico(s.reloj.Ahora())
	if !autorizacionV3ValidaEn(
		solicitudAutorizacionV3,
		decisionV3,
		confirmacionV3,
		instantePreparacion,
	) {
		return ports.ReciboAlta{}, ports.ErrAutorizacionDenegada
	}

	// Preparar es una operación interna y solo se invoca después de obtener una
	// concesión V3 durable y vigente. O2-05 deberá incorporarla físicamente a la
	// misma transacción que consume la concesión y confirma el efecto.
	preparar := ports.SolicitudPrepararAlta{
		ClaveIdempotencia:  solicitud.ClaveIdempotencia,
		HuellaPeticionHMAC: huella,
		OrganizacionRef:    solicitud.OrganizacionRef,
		ActorRef:           vinculo.PrincipalID,
		PerfilRef:          vinculo.PerfilActivoRef,
	}
	if preparar.Validar() != nil {
		return ports.ReciboAlta{}, errors.Join(
			ErrSolicitudRegistroInvalida,
			ports.ErrPreparacionAltaInvalida,
		)
	}
	preparacion, err := s.preparaciones.PrepararAlta(ctx, preparar)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	if preparacion.ValidarPara(preparar) != nil ||
		!hmac.Equal([]byte(preparacion.AmbitoIdempotenciaHMAC), []byte(ambitoHMAC)) {
		return ports.ReciboAlta{}, ports.ErrPreparacionAltaInvalida
	}
	instanteEfecto := instanteCanonico(s.reloj.Ahora())
	if contextoAutorizacion.ValidarPara(resolverContexto, instanteEfecto) != nil ||
		!autorizacionV3ValidaEn(
			solicitudAutorizacionV3,
			decisionV3,
			confirmacionV3,
			instanteEfecto,
		) {
		return ports.ReciboAlta{}, ports.ErrAutorizacionDenegada
	}
	if preparacion.Estado == ports.PreparacionConfirmada {
		return *preparacion.ReciboConfirmado, nil
	}

	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      preparacion.Referencias.ExpedienteRef,
		OrganizacionRef: solicitud.OrganizacionRef,
		NumeroVisible:   preparacion.Referencias.NumeroVisible,
		Flujo:           configuracion.Flujo,
		FaseInicial:     configuracion.FaseInicial,
		Solicitud:       solicitudCentro,
		Actuacion: domain.DatosActuacion{
			AccionClave:   configuracion.AccionInicial,
			ActorRef:      vinculo.PrincipalID,
			UnidadRef:     configuracion.UnidadInicialRef,
			ReciboRef:     preparacion.Referencias.ReciboRef,
			RealizadaEn:   instanteEfecto,
			FaseDestino:   configuracion.FaseInicial,
			EstadoDestino: domain.EstadoEnCurso,
		},
	})
	if err != nil {
		return ports.ReciboAlta{}, errors.Join(ErrSolicitudRegistroInvalida, err)
	}
	orden, err := ports.NuevaOrdenConfirmarAlta(ports.DatosOrdenConfirmarAlta{
		Expediente:              expediente,
		SolicitudAutorizacionV3: solicitudAutorizacionV3,
		DecisionAutorizacionV3:  decisionV3,
		ConfirmacionRegistroV3:  confirmacionV3,
		Preparacion:             preparacion,
		CorrelacionRef:          solicitud.CorrelacionRef,
	})
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	recibo, err := s.transaccion.ConfirmarAlta(ctx, orden)
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
		!domain.ReferenciaOpacaValida(solicitud.CorrelacionRef) ||
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
