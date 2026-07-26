package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// PreparadorConsultaCobertura ofrece únicamente la operación sin efecto
// durable. No contiene un consumidor que pueda activarse accidentalmente.
type PreparadorConsultaCobertura struct {
	servicio *ServicioConsultaCobertura
}

func NuevoPreparadorConsultaCobertura(
	fuente ports.FuenteComprobacionCobertura,
	verificador ports.VerificadorRespuestaCobertura,
	publicador ports.PublicadorCatalogoCobertura,
	autenticador ports.VerificadorPresentacionesAutoridadFuenteAnalisis,
	reloj ports.Reloj,
	tiempoMaximo time.Duration,
) (*PreparadorConsultaCobertura, error) {
	if dependenciaNula(fuente) || dependenciaNula(verificador) ||
		dependenciaNula(publicador) || dependenciaNula(autenticador) ||
		dependenciaNula(reloj) || tiempoMaximo <= 0 ||
		tiempoMaximo > TiempoMaximoFuenteCobertura {
		return nil, ErrServicioConsultaCoberturaInvalido
	}
	return &PreparadorConsultaCobertura{
		servicio: &ServicioConsultaCobertura{
			fuente: fuente, verificador: verificador,
			publicador: publicador, autenticador: autenticador,
			reloj: reloj, tiempoMaximo: tiempoMaximo,
			crearPlazo: context.WithTimeout,
		},
	}, nil
}

func (p *PreparadorConsultaCobertura) ConsultarConEvidencia(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (cobertura.EvidenciaConsultaCobertura, error) {
	if p == nil || p.servicio == nil {
		return cobertura.EvidenciaConsultaCobertura{},
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	return p.servicio.ConsultarConEvidencia(ctx, solicitud)
}

// ConsultarConEvidencia obtiene y verifica una comprobación, pero no la
// consume. La orden opaca resultante debe entregarse a una transacción final.
func (s *ServicioConsultaCobertura) ConsultarConEvidencia(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (cobertura.EvidenciaConsultaCobertura, error) {
	operacion, cancelar, reloj, material, err :=
		s.iniciarConsultaCobertura(ctx, solicitud, false)
	if err != nil {
		return cobertura.EvidenciaConsultaCobertura{}, err
	}
	defer cancelar()
	return s.prepararConsultaCoberturaEnOperacion(
		operacion,
		solicitud,
		material,
		reloj,
	)
}

func (s *ServicioConsultaCobertura) iniciarConsultaCobertura(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
	exigirConsumidor bool,
) (
	context.Context,
	context.CancelFunc,
	*relojMonotonoCobertura,
	[]byte,
	error,
) {
	if ctx == nil || s == nil || dependenciaNula(s.fuente) ||
		dependenciaNula(s.verificador) || dependenciaNula(s.publicador) ||
		dependenciaNula(s.autenticador) || dependenciaNula(s.reloj) ||
		(exigirConsumidor && dependenciaNula(s.consumidor)) ||
		s.tiempoMaximo <= 0 ||
		s.tiempoMaximo > TiempoMaximoFuenteCobertura ||
		s.crearPlazo == nil || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef !=
			s.autenticador.OrganizacionAutoridadFuenteAnalisis() {
		return nil, nil, nil, nil,
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	material, err := solicitud.MaterialCanonico()
	if err != nil {
		return nil, nil, nil, nil,
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	operacion, cancelar := s.crearPlazo(ctx, s.tiempoMaximo)
	if operacion == nil || cancelar == nil {
		return nil, nil, nil, nil,
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	reloj := nuevoRelojMonotonoCobertura(s.reloj)
	if err := operacion.Err(); err != nil {
		cancelar()
		return nil, nil, nil, nil, errorDisponibilidadCobertura(
			ErrFuenteCoberturaNoDisponible,
			err,
		)
	}
	return operacion, cancelar, &reloj, material, nil
}

func (s *ServicioConsultaCobertura) prepararConsultaCoberturaEnOperacion(
	operacion context.Context,
	solicitud ports.SolicitudConsultarCobertura,
	materialPeticion []byte,
	reloj *relojMonotonoCobertura,
) (cobertura.EvidenciaConsultaCobertura, error) {
	autoridadFuente, err := s.autenticar(
		operacion,
		s.fuente,
		materialPeticion,
		ports.RolFuenteCobertura,
		ErrFuenteCoberturaNoDisponible,
		reloj,
	)
	if err != nil {
		return cobertura.EvidenciaConsultaCobertura{}, err
	}
	autoridadVerificador, err := s.autenticar(
		operacion,
		s.verificador,
		materialPeticion,
		ports.RolVerificadorCobertura,
		ErrVerificadorCoberturaNoDisponible,
		reloj,
	)
	if err != nil {
		return cobertura.EvidenciaConsultaCobertura{}, err
	}
	autoridadPublicador, err := s.autenticar(
		operacion,
		s.publicador,
		materialPeticion,
		ports.RolPublicadorCatalogoCobertura,
		ErrPublicadorCatalogoCoberturaNoDisponible,
		reloj,
	)
	if err != nil {
		return cobertura.EvidenciaConsultaCobertura{}, err
	}
	if !ports.AutoridadesFuenteAnalisisSeparadas(
		autoridadFuente.identidad,
		autoridadVerificador.identidad,
		autoridadPublicador.identidad,
	) || autoridadFuente.identidad.BackendRef() !=
		solicitud.Comprobacion.Procedencia.DefinicionFuenteRef {
		return cobertura.EvidenciaConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	confirmacionCatalogo, errPublicador :=
		s.publicador.ConsultarPublicacionCobertura(operacion, solicitud)
	if err := operacion.Err(); err != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			errorDisponibilidadCobertura(
				ErrPublicadorCatalogoCoberturaNoDisponible,
				err,
			)
	}
	comprobadaEn, errReloj := reloj.ahora()
	datosCatalogo, errDatosCatalogo := confirmacionCatalogo.Datos()
	if errPublicador != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			errorDisponibilidadCobertura(
				ErrPublicadorCatalogoCoberturaNoDisponible,
				errPublicador,
			)
	}
	if errReloj != nil || errDatosCatalogo != nil ||
		datosCatalogo.PublicadorRef !=
			autoridadPublicador.identidad.AutoridadRef() ||
		confirmacionCatalogo.ValidarPara(solicitud, comprobadaEn) != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	resultado, errFuente := s.fuente.ConsultarCobertura(operacion, solicitud)
	if err := operacion.Err(); err != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			errorDisponibilidadCobertura(
				ErrFuenteCoberturaNoDisponible,
				err,
			)
	}
	recibidaEn, errReloj := reloj.ahora()
	if errFuente != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			errorDisponibilidadCobertura(
				ErrFuenteCoberturaNoDisponible,
				errFuente,
			)
	}
	datosResultado, errDatosResultado := resultado.Datos()
	atestacion, errAtestacion := resultado.Atestacion()
	if errReloj != nil || errDatosResultado != nil ||
		errAtestacion != nil || resultado.ValidarPara(solicitud) != nil ||
		atestacion.Metadatos.AutoridadRef !=
			autoridadFuente.identidad.AutoridadRef() ||
		atestacion.Metadatos.EmitidaEn.Before(solicitud.SolicitadaEn) ||
		datosResultado.Comprobacion.EvaluadaEn.After(
			atestacion.Metadatos.EmitidaEn,
		) ||
		datosResultado.Comprobacion.EvaluadaEn.After(recibidaEn) ||
		recibidaEn.Before(atestacion.Metadatos.EmitidaEn) ||
		!recibidaEn.Before(atestacion.Metadatos.ValidaHasta) {
		return cobertura.EvidenciaConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	solicitudVerificacion, err := resultado.SolicitudVerificacion()
	if err != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	confirmacion, err := s.verificarRespuesta(
		operacion,
		autoridadVerificador.identidad,
		solicitudVerificacion,
		reloj,
	)
	if err != nil {
		return cobertura.EvidenciaConsultaCobertura{}, err
	}
	orden, err := ports.NuevaOrdenConsumoCobertura(
		solicitud,
		resultado,
		confirmacion,
		confirmacionCatalogo,
		autoridadVerificador.identidad,
		autoridadVerificador.evidencia,
	)
	if err != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	preparadaEn, errReloj := reloj.ahora()
	if errContexto := operacion.Err(); errContexto != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			errorDisponibilidadCobertura(
				ErrFuenteCoberturaNoDisponible,
				errContexto,
			)
	}
	if errReloj != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	evidencia, err := cobertura.NuevaEvidenciaConsultaCobertura(
		orden,
		preparadaEn,
	)
	if err != nil {
		return cobertura.EvidenciaConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return evidencia, nil
}
