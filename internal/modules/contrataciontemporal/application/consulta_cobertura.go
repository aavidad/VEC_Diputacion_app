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
	autenticador ports.VerificadorPresentacionesAutoridadFuenteAnalisis
	reloj        ports.Reloj
	tiempoMaximo time.Duration
	crearPlazo   func(
		context.Context,
		time.Duration,
	) (context.Context, context.CancelFunc)
}

func NuevoServicioConsultaCobertura(
	fuente ports.FuenteComprobacionCobertura,
	verificador ports.VerificadorRespuestaCobertura,
	publicador ports.PublicadorCatalogoCobertura,
	consumidor ports.ConsumidorCobertura,
	autenticador ports.VerificadorPresentacionesAutoridadFuenteAnalisis,
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
		crearPlazo:   context.WithTimeout,
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
		s.crearPlazo == nil ||
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
	operacion, cancelar := s.crearPlazo(ctx, s.tiempoMaximo)
	if operacion == nil || cancelar == nil {
		return domain.ComprobacionCobertura{},
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	defer cancelar()
	relojOperacion := nuevoRelojMonotonoCobertura(s.reloj)
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{},
			errorDisponibilidadCobertura(
				ErrFuenteCoberturaNoDisponible,
				err,
			)
	}

	autoridadFuente, err := s.autenticar(
		operacion,
		s.fuente,
		materialPeticion,
		ports.RolFuenteCobertura,
		ErrFuenteCoberturaNoDisponible,
		&relojOperacion,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	autoridadVerificador, err := s.autenticar(
		operacion,
		s.verificador,
		materialPeticion,
		ports.RolVerificadorCobertura,
		ErrVerificadorCoberturaNoDisponible,
		&relojOperacion,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	autoridadPublicador, err := s.autenticar(
		operacion,
		s.publicador,
		materialPeticion,
		ports.RolPublicadorCatalogoCobertura,
		ErrPublicadorCatalogoCoberturaNoDisponible,
		&relojOperacion,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	if !ports.AutoridadesFuenteAnalisisSeparadas(
		autoridadFuente.identidad,
		autoridadVerificador.identidad,
		autoridadPublicador.identidad,
	) || autoridadFuente.identidad.BackendRef() !=
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
	comprobadaEn, errReloj := relojOperacion.ahora()
	datosCatalogo, errDatosCatalogo := confirmacionCatalogo.Datos()
	if errPublicador != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadCobertura(
			ErrPublicadorCatalogoCoberturaNoDisponible,
			errPublicador,
		)
	}
	if errReloj != nil || errDatosCatalogo != nil ||
		datosCatalogo.PublicadorRef !=
			autoridadPublicador.identidad.AutoridadRef() ||
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
	recibidaEn, errReloj := relojOperacion.ahora()
	if errFuente != nil {
		return domain.ComprobacionCobertura{},
			errorDisponibilidadCobertura(
				ErrFuenteCoberturaNoDisponible,
				errFuente,
			)
	}
	datosResultado, errDatosResultado := resultado.Datos()
	atestacion, errAtestacion := resultado.Atestacion()
	if errReloj != nil || errDatosResultado != nil || errAtestacion != nil ||
		resultado.ValidarPara(solicitud) != nil ||
		atestacion.Metadatos.AutoridadRef !=
			autoridadFuente.identidad.AutoridadRef() ||
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
		autoridadVerificador.identidad,
		solicitudVerificacion,
		&relojOperacion,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	claveVerificador :=
		autoridadVerificador.identidad.ClavePruebaEd25519()
	orden, err := ports.NuevaOrdenConsumoCobertura(
		solicitud,
		resultado,
		confirmacion,
		confirmacionCatalogo,
		autoridadVerificador.identidad,
		autoridadVerificador.evidencia,
	)
	if err != nil {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}

	antesConsumo, errReloj := relojOperacion.ahora()
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadCobertura(
			ErrConsumoCoberturaNoDisponible,
			err,
		)
	}
	if errReloj != nil || confirmacion.ValidarPara(
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
	finalizadaEn, errReloj := relojOperacion.ahora()
	if errReloj != nil {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	reciboValido := s.validarReciboConsumoCobertura(recibo, orden) == nil &&
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

type autoridadCoberturaAutenticada struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	evidencia ports.EvidenciaPublicaAutoridadFuenteAnalisis
}

func (s *ServicioConsultaCobertura) autenticar(
	ctx context.Context,
	presentador ports.PresentadorAutoridadFuenteAnalisis,
	materialPeticion []byte,
	rol ports.RolAutoridadFuenteAnalisis,
	errDisponibilidad error,
	reloj *relojMonotonoCobertura,
) (autoridadCoberturaAutenticada, error) {
	desafio, err := ports.NuevoDesafioAutoridadFuenteAnalisis(
		append([]byte(nil), materialPeticion...),
		s.autenticador.OrganizacionAutoridadFuenteAnalisis(),
		s.autenticador.AudienciaAutoridadFuenteAnalisis(),
		rol,
	)
	if err != nil {
		return autoridadCoberturaAutenticada{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	presentacion, errPresentacion :=
		presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
	if errContexto := ctx.Err(); errContexto != nil {
		return autoridadCoberturaAutenticada{},
			errorDisponibilidadCobertura(
				errDisponibilidad,
				errContexto,
			)
	}
	if errPresentacion != nil {
		return autoridadCoberturaAutenticada{},
			errorDisponibilidadCobertura(
				errDisponibilidad,
				errPresentacion,
			)
	}
	comprobadaEn, errReloj := reloj.ahora()
	if errReloj != nil {
		return autoridadCoberturaAutenticada{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	evidencia, err := ports.NuevaEvidenciaPublicaAutoridadFuenteAnalisis(
		desafio,
		presentacion,
		rol,
		comprobadaEn,
	)
	if err != nil {
		return autoridadCoberturaAutenticada{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	identidad, err := s.autenticador.
		VerificarEvidenciaPublicaAutoridadFuenteAnalisis(
			evidencia,
		)
	if err != nil {
		return autoridadCoberturaAutenticada{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return autoridadCoberturaAutenticada{
		identidad: identidad,
		evidencia: evidencia,
	}, nil
}

func (s *ServicioConsultaCobertura) validarReciboConsumoCobertura(
	recibo ports.ReciboConsumoCobertura,
	orden ports.OrdenConsumoCobertura,
) error {
	if recibo.ValidarPara(orden) != nil {
		return ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	evidencia, err := recibo.EvidenciaPublicaVerificador()
	if err != nil {
		return ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	identidad, err := s.autenticador.
		VerificarEvidenciaPublicaAutoridadFuenteAnalisis(evidencia)
	if err != nil ||
		recibo.ValidarIdentidadVerificadorOriginal(identidad) != nil {
		return ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (s *ServicioConsultaCobertura) verificarRespuesta(
	ctx context.Context,
	identidad ports.IdentidadAutoridadFuenteAnalisis,
	solicitud ports.SolicitudVerificarRespuestaCobertura,
	reloj *relojMonotonoCobertura,
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
	verificadaEn, errReloj := reloj.ahora()
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
	if errReloj != nil || errDatos != nil ||
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

// relojMonotonoCobertura impone un suelo por operación sobre el reloj
// autoritativo inyectado. Un retroceso temporal invalida la operación: nunca
// se corrige silenciosamente porque podría reabrir una evidencia ya caducada.
type relojMonotonoCobertura struct {
	reloj ports.Reloj
	suelo time.Time
}

func nuevoRelojMonotonoCobertura(reloj ports.Reloj) relojMonotonoCobertura {
	return relojMonotonoCobertura{reloj: reloj}
}

func (r *relojMonotonoCobertura) ahora() (time.Time, error) {
	if r == nil || dependenciaNula(r.reloj) {
		return time.Time{}, ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	actual := r.reloj.Ahora()
	if !domain.InstanteUTCCanonico(actual) ||
		(!r.suelo.IsZero() && actual.Before(r.suelo)) {
		return time.Time{}, ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	r.suelo = actual
	return actual, nil
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
