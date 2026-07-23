package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const TiempoMaximoFuenteCobertura = 5 * time.Second

var (
	ErrServicioConsultaCoberturaInvalido = errors.New(
		"contratacion temporal: servicio de consulta de cobertura invalido",
	)
	ErrFuenteCoberturaNoDisponible = errors.New(
		"contratacion temporal: fuente de cobertura no disponible",
	)
	ErrVerificadorCoberturaNoDisponible = errors.New(
		"contratacion temporal: verificador de cobertura no disponible",
	)
	ErrPublicadorCatalogoCoberturaNoDisponible = errors.New(
		"contratacion temporal: publicador de catalogo de cobertura no disponible",
	)
	ErrConsumoCoberturaNoDisponible = errors.New(
		"contratacion temporal: consumo de cobertura no disponible",
	)
)

// ServicioConsultaCobertura coordina el caso de uso sin conocer HTTP,
// escritorio, CLI, MCP, SQL ni los proveedores concretos.
type ServicioConsultaCobertura struct {
	fuente       ports.FuenteComprobacionCobertura
	verificador  ports.VerificadorRespuestaCobertura
	publicador   ports.PublicadorCatalogoCobertura
	consumidor   ports.ConsumidorCobertura
	autenticador ports.AutenticadorAutoridadesFuenteAnalisis
	reloj        ports.Reloj
	tiempoMaximo time.Duration
}

func NuevoServicioConsultaCobertura(
	fuente ports.FuenteComprobacionCobertura,
	verificador ports.VerificadorRespuestaCobertura,
	publicador ports.PublicadorCatalogoCobertura,
	consumidor ports.ConsumidorCobertura,
	autenticador ports.AutenticadorAutoridadesFuenteAnalisis,
	reloj ports.Reloj,
	tiempoMaximo time.Duration,
) (*ServicioConsultaCobertura, error) {
	if dependenciaNula(fuente) || dependenciaNula(verificador) ||
		dependenciaNula(publicador) || dependenciaNula(consumidor) ||
		dependenciaNula(autenticador) || dependenciaNula(reloj) ||
		tiempoMaximo <= 0 || tiempoMaximo > TiempoMaximoFuenteCobertura {
		return nil, ErrServicioConsultaCoberturaInvalido
	}
	return &ServicioConsultaCobertura{
		fuente:       fuente,
		verificador:  verificador,
		publicador:   publicador,
		consumidor:   consumidor,
		autenticador: autenticador,
		reloj:        reloj,
		tiempoMaximo: tiempoMaximo,
	}, nil
}

func (s *ServicioConsultaCobertura) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (domain.ComprobacionCobertura, error) {
	if ctx == nil || s == nil || dependenciaNula(s.fuente) ||
		dependenciaNula(s.verificador) || dependenciaNula(s.publicador) ||
		dependenciaNula(s.consumidor) || dependenciaNula(s.autenticador) ||
		dependenciaNula(s.reloj) || s.tiempoMaximo <= 0 ||
		s.tiempoMaximo > TiempoMaximoFuenteCobertura ||
		solicitud.Validar() != nil ||
		solicitud.OrganizacionRef !=
			s.autenticador.OrganizacionAutoridadFuenteAnalisis() {
		return domain.ComprobacionCobertura{},
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	materialPeticion, err := solicitud.MaterialCanonico()
	if err != nil {
		return domain.ComprobacionCobertura{},
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	operacion, cancelar := context.WithTimeout(ctx, s.tiempoMaximo)
	defer cancelar()
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{},
			errorDisponibilidadCobertura(
				ErrFuenteCoberturaNoDisponible,
				err,
			)
	}

	identidadFuente, err := s.autenticar(
		operacion,
		s.fuente,
		materialPeticion,
		ports.RolFuenteCobertura,
		ErrFuenteCoberturaNoDisponible,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	identidadVerificador, err := s.autenticar(
		operacion,
		s.verificador,
		materialPeticion,
		ports.RolVerificadorCobertura,
		ErrVerificadorCoberturaNoDisponible,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	identidadPublicador, err := s.autenticar(
		operacion,
		s.publicador,
		materialPeticion,
		ports.RolPublicadorCatalogoCobertura,
		ErrPublicadorCatalogoCoberturaNoDisponible,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	if !ports.AutoridadesFuenteAnalisisSeparadas(
		identidadFuente,
		identidadVerificador,
		identidadPublicador,
	) || identidadFuente.BackendRef() !=
		solicitud.Comprobacion.Procedencia.DefinicionFuenteRef {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}

	confirmacionCatalogo, errPublicador :=
		s.publicador.ConsultarPublicacionCobertura(operacion, solicitud)
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadCobertura(
			ErrPublicadorCatalogoCoberturaNoDisponible,
			err,
		)
	}
	comprobadaEn := s.reloj.Ahora()
	datosCatalogo, errDatosCatalogo := confirmacionCatalogo.Datos()
	if errPublicador != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadCobertura(
			ErrPublicadorCatalogoCoberturaNoDisponible,
			errPublicador,
		)
	}
	if errDatosCatalogo != nil ||
		datosCatalogo.PublicadorRef != identidadPublicador.AutoridadRef() ||
		confirmacionCatalogo.ValidarPara(solicitud, comprobadaEn) != nil {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}

	resultado, errFuente := s.fuente.ConsultarCobertura(operacion, solicitud)
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{},
			errorDisponibilidadCobertura(
				ErrFuenteCoberturaNoDisponible,
				err,
			)
	}
	recibidaEn := s.reloj.Ahora()
	if errFuente != nil {
		return domain.ComprobacionCobertura{},
			errorDisponibilidadCobertura(
				ErrFuenteCoberturaNoDisponible,
				errFuente,
			)
	}
	datosResultado, errDatosResultado := resultado.Datos()
	atestacion, errAtestacion := resultado.Atestacion()
	if !domain.InstanteUTCCanonico(recibidaEn) ||
		errDatosResultado != nil || errAtestacion != nil ||
		resultado.ValidarPara(solicitud) != nil ||
		atestacion.Metadatos.AutoridadRef != identidadFuente.AutoridadRef() ||
		atestacion.Metadatos.EmitidaEn.Before(solicitud.SolicitadaEn) ||
		datosResultado.Comprobacion.EvaluadaEn.After(
			atestacion.Metadatos.EmitidaEn,
		) ||
		datosResultado.Comprobacion.EvaluadaEn.After(recibidaEn) ||
		recibidaEn.Before(atestacion.Metadatos.EmitidaEn) ||
		!recibidaEn.Before(atestacion.Metadatos.ValidaHasta) {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}

	solicitudVerificacion, err := resultado.SolicitudVerificacion()
	if err != nil {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	confirmacion, err := s.verificarRespuesta(
		operacion,
		identidadVerificador,
		solicitudVerificacion,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	claveVerificador := identidadVerificador.ClavePruebaEd25519()
	orden, err := ports.NuevaOrdenConsumoCobertura(
		solicitud,
		resultado,
		confirmacion,
		confirmacionCatalogo,
		claveVerificador,
	)
	if err != nil {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}

	antesConsumo := s.reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadCobertura(
			ErrConsumoCoberturaNoDisponible,
			err,
		)
	}
	if confirmacion.ValidarPara(
		solicitudVerificacion,
		antesConsumo,
		claveVerificador,
	) != nil ||
		confirmacionCatalogo.ValidarPara(solicitud, antesConsumo) != nil {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}

	recibo, errConsumo := s.consumidor.ConsumirCobertura(operacion, orden)
	errContextoFinal := operacion.Err()
	finalizadaEn := s.reloj.Ahora()
	if !domain.InstanteUTCCanonico(finalizadaEn) ||
		finalizadaEn.Before(antesConsumo) {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	reciboValido := recibo.ValidarPara(orden) == nil &&
		!recibo.ConsumidaEn.After(finalizadaEn)
	errorCompatibleConCommit := errConsumo == nil ||
		errors.Is(errConsumo, context.Canceled) ||
		errors.Is(errConsumo, context.DeadlineExceeded)
	if reciboValido && errorCompatibleConCommit {
		return datosResultado.Comprobacion, nil
	}
	if errContextoFinal != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadCobertura(
			ErrConsumoCoberturaNoDisponible,
			errContextoFinal,
		)
	}
	if confirmacion.ValidarPara(
		solicitudVerificacion,
		finalizadaEn,
		claveVerificador,
	) != nil ||
		confirmacionCatalogo.ValidarPara(solicitud, finalizadaEn) != nil {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	if errConsumo != nil {
		if errors.Is(
			errConsumo,
			ports.ErrRespuestaCoberturaYaConsumida,
		) {
			return domain.ComprobacionCobertura{},
				ports.ErrRespuestaCoberturaYaConsumida
		}
		return domain.ComprobacionCobertura{}, errorDisponibilidadCobertura(
			ErrConsumoCoberturaNoDisponible,
			errConsumo,
		)
	}
	if !reciboValido {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return datosResultado.Comprobacion, nil
}

func (s *ServicioConsultaCobertura) autenticar(
	ctx context.Context,
	presentador ports.PresentadorAutoridadFuenteAnalisis,
	materialPeticion []byte,
	rol ports.RolAutoridadFuenteAnalisis,
	errDisponibilidad error,
) (ports.IdentidadAutoridadFuenteAnalisis, error) {
	identidad, err := s.autenticador.AutenticarAutoridadFuenteAnalisis(
		ctx,
		presentador,
		materialPeticion,
		rol,
		s.reloj.Ahora(),
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			errorDisponibilidadCobertura(
				errDisponibilidad,
				errContexto,
			)
	}
	if err != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return identidad, nil
}

func (s *ServicioConsultaCobertura) verificarRespuesta(
	ctx context.Context,
	identidad ports.IdentidadAutoridadFuenteAnalisis,
	solicitud ports.SolicitudVerificarRespuestaCobertura,
) (ports.ConfirmacionRespuestaCobertura, error) {
	confirmacion, errVerificador :=
		s.verificador.VerificarRespuestaCobertura(ctx, solicitud)
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionRespuestaCobertura{},
			errorDisponibilidadCobertura(
				ErrVerificadorCoberturaNoDisponible,
				err,
			)
	}
	verificadaEn := s.reloj.Ahora()
	datos, errDatos := confirmacion.Datos()
	if errVerificador != nil {
		if errors.Is(
			errVerificador,
			ports.ErrResultadoFuenteCoberturaNoConfiable,
		) {
			return ports.ConfirmacionRespuestaCobertura{},
				ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		return ports.ConfirmacionRespuestaCobertura{},
			errorDisponibilidadCobertura(
				ErrVerificadorCoberturaNoDisponible,
				errVerificador,
			)
	}
	if errDatos != nil ||
		datos.VerificadorRef != identidad.AutoridadRef() ||
		confirmacion.ValidarPara(
			solicitud,
			verificadaEn,
			identidad.ClavePruebaEd25519(),
		) != nil {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return confirmacion, nil
}

func errorDisponibilidadCobertura(publico, causa error) error {
	var contexto error
	switch {
	case errors.Is(causa, context.Canceled):
		contexto = context.Canceled
	case errors.Is(causa, context.DeadlineExceeded):
		contexto = context.DeadlineExceeded
	}
	return errorConsultaCobertura{publico: publico, contexto: contexto}
}

type errorConsultaCobertura struct {
	publico  error
	contexto error
}

func (e errorConsultaCobertura) Error() string {
	return e.publico.Error()
}

func (e errorConsultaCobertura) Unwrap() []error {
	if e.contexto == nil {
		return []error{e.publico}
	}
	return []error{e.publico, e.contexto}
}
