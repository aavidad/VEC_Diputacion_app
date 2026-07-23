package ports

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const TiempoMaximoFuenteCobertura = 5 * time.Second

var (
	ErrPeticionFuenteCoberturaInvalida = errors.New(
		"contratacion temporal: peticion a fuente de cobertura invalida",
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
	ErrResultadoFuenteCoberturaNoConfiable = errors.New(
		"contratacion temporal: resultado de fuente de cobertura no confiable",
	)
	ErrRespuestaCoberturaYaConsumida = errors.New(
		"contratacion temporal: respuesta de cobertura ya consumida con otros datos",
	)
)

// SolicitudConsultarCobertura contiene la información mínima para comprobar
// una vía. No transporta nombres, DNI, candidatos, posiciones ni credenciales.
// El catálogo publicado decide la procedencia; añadir Bolsa, SAE u otra fuente
// no exige modificar este contrato.
type SolicitudConsultarCobertura struct {
	PeticionRef       string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	Catalogo          domain.IdentidadCatalogoViasCobertura
	ViaClave          domain.ClaveCatalogo
	Comprobacion      domain.ComprobacionExigibleCobertura
	CategoriaRef      string
	Periodo           domain.PeriodoPrevisto
	SolicitadaEn      time.Time
}

func (s SolicitudConsultarCobertura) Validar() error {
	if !domain.ReferenciaOpacaValida(s.PeticionRef) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		s.VersionExpediente == 0 ||
		s.VersionExpediente > maximoEnteroSeguroFuenteAnalisis ||
		s.Catalogo.Validar() != nil ||
		!s.ViaClave.Valida() ||
		s.Comprobacion.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.CategoriaRef) ||
		!periodoFuenteAnalisisValido(s.Periodo) ||
		!instanteFuenteAnalisisCanonico(s.SolicitadaEn) {
		return ErrPeticionFuenteCoberturaInvalida
	}
	return nil
}

// FuenteComprobacionCobertura es un puerto de salida genérico. El adaptador
// puede despachar por la definición gobernada a Bolsa, SAE, convocatorias u
// otros conectores, pero nunca consultar sus tablas desde este módulo.
type FuenteComprobacionCobertura interface {
	PresentadorAutoridadFuenteAnalisis
	ConsultarCobertura(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error)
}

// ConsultarCoberturaConFuente ejecuta una operación completa fail-closed. La
// confianza y sus raíces pertenecen exclusivamente a la composición del
// servidor; ningún cliente web, escritorio, CLI o MCP puede aportarlas.
func ConsultarCoberturaConFuente(
	ctx context.Context,
	fuente FuenteComprobacionCobertura,
	verificador VerificadorRespuestaCobertura,
	publicador PublicadorCatalogoCobertura,
	consumidor ConsumidorCobertura,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj Reloj,
	solicitud SolicitudConsultarCobertura,
	tiempoMaximo time.Duration,
) (domain.ComprobacionCobertura, error) {
	if ctx == nil || dependenciaNulaFuenteCobertura(fuente) ||
		dependenciaNulaFuenteCobertura(verificador) ||
		dependenciaNulaFuenteCobertura(publicador) ||
		dependenciaNulaFuenteCobertura(consumidor) ||
		dependenciaNulaFuenteCobertura(reloj) ||
		solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != confianza.organizacionRef ||
		tiempoMaximo <= 0 || tiempoMaximo > TiempoMaximoFuenteCobertura {
		return domain.ComprobacionCobertura{},
			ErrPeticionFuenteCoberturaInvalida
	}
	materialPeticion, err := canonPeticionCobertura(solicitud)
	if err != nil {
		return domain.ComprobacionCobertura{},
			ErrPeticionFuenteCoberturaInvalida
	}
	operacion, cancelar := context.WithTimeout(ctx, tiempoMaximo)
	defer cancelar()
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{},
			errorDisponibilidadFuente(ErrFuenteCoberturaNoDisponible, err)
	}

	identidadFuente, err := autenticarAutoridadCobertura(
		operacion,
		fuente,
		confianza,
		materialPeticion,
		RolFuenteCobertura,
		reloj,
		ErrFuenteCoberturaNoDisponible,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	identidadVerificador, err := autenticarAutoridadCobertura(
		operacion,
		verificador,
		confianza,
		materialPeticion,
		RolVerificadorCobertura,
		reloj,
		ErrVerificadorCoberturaNoDisponible,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	identidadPublicador, err := autenticarAutoridadCobertura(
		operacion,
		publicador,
		confianza,
		materialPeticion,
		RolPublicadorCatalogoCobertura,
		reloj,
		ErrPublicadorCatalogoCoberturaNoDisponible,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	if !autoridadesFuenteAnalisisSeparadas(
		identidadFuente,
		identidadVerificador,
		identidadPublicador,
	) || identidadFuente.backendRef !=
		solicitud.Comprobacion.Procedencia.DefinicionFuenteRef {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}

	confirmacionCatalogo, errPublicador :=
		publicador.ConsultarPublicacionCobertura(operacion, solicitud)
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadFuente(
			ErrPublicadorCatalogoCoberturaNoDisponible,
			err,
		)
	}
	comprobadaEn := reloj.Ahora()
	datosCatalogo, errDatosCatalogo := confirmacionCatalogo.Datos()
	if errPublicador != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadFuente(
			ErrPublicadorCatalogoCoberturaNoDisponible,
			errPublicador,
		)
	}
	if errDatosCatalogo != nil ||
		datosCatalogo.PublicadorRef != identidadPublicador.autoridadRef ||
		confirmacionCatalogo.ValidarPara(solicitud, comprobadaEn) != nil {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}

	resultado, errFuente := fuente.ConsultarCobertura(operacion, solicitud)
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{},
			errorDisponibilidadFuente(ErrFuenteCoberturaNoDisponible, err)
	}
	recibidaEn := reloj.Ahora()
	if errFuente != nil {
		return domain.ComprobacionCobertura{},
			errorDisponibilidadFuente(
				ErrFuenteCoberturaNoDisponible,
				errFuente,
			)
	}
	datosResultado, errDatosResultado := resultado.Datos()
	if !instanteFuenteAnalisisCanonico(recibidaEn) ||
		errDatosResultado != nil ||
		resultado.ValidarPara(solicitud) != nil ||
		resultado.atestacion.Metadatos.AutoridadRef !=
			identidadFuente.autoridadRef ||
		resultado.atestacion.Metadatos.EmitidaEn.Before(
			solicitud.SolicitadaEn,
		) ||
		datosResultado.Comprobacion.EvaluadaEn.After(
			resultado.atestacion.Metadatos.EmitidaEn,
		) ||
		datosResultado.Comprobacion.EvaluadaEn.After(recibidaEn) ||
		recibidaEn.Before(resultado.atestacion.Metadatos.EmitidaEn) ||
		!recibidaEn.Before(resultado.atestacion.Metadatos.ValidaHasta) {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}

	solicitudVerificacion, err := nuevaSolicitudVerificarRespuestaCobertura(
		datosResultado.HuellaPeticionSHA256,
		resultado.preimagen,
		resultado.atestacion,
	)
	if err != nil {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	confirmacion, err := verificarRespuestaCobertura(
		operacion,
		verificador,
		identidadVerificador,
		solicitudVerificacion,
		reloj,
	)
	if err != nil {
		return domain.ComprobacionCobertura{}, err
	}
	orden, err := nuevaOrdenConsumoCobertura(
		solicitud,
		resultado,
		confirmacion,
		confirmacionCatalogo,
		identidadVerificador.clavePrueba,
	)
	if err != nil {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}

	antesConsumo := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadFuente(
			ErrConsumoCoberturaNoDisponible,
			err,
		)
	}
	if confirmacion.ValidarPara(
		solicitudVerificacion,
		antesConsumo,
		identidadVerificador.clavePrueba,
	) != nil ||
		confirmacionCatalogo.ValidarPara(solicitud, antesConsumo) != nil {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	recibo, errConsumo := consumidor.ConsumirCobertura(operacion, orden)
	errContextoFinal := operacion.Err()
	finalizadaEn := reloj.Ahora()
	if errContextoFinal != nil {
		return domain.ComprobacionCobertura{}, errorDisponibilidadFuente(
			ErrConsumoCoberturaNoDisponible,
			errContextoFinal,
		)
	}
	if !instanteFuenteAnalisisCanonico(finalizadaEn) ||
		finalizadaEn.Before(antesConsumo) ||
		confirmacion.ValidarPara(
			solicitudVerificacion,
			finalizadaEn,
			identidadVerificador.clavePrueba,
		) != nil ||
		confirmacionCatalogo.ValidarPara(solicitud, finalizadaEn) != nil {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	if errConsumo != nil {
		if errors.Is(errConsumo, ErrRespuestaCoberturaYaConsumida) {
			return domain.ComprobacionCobertura{},
				ErrRespuestaCoberturaYaConsumida
		}
		return domain.ComprobacionCobertura{}, errorDisponibilidadFuente(
			ErrConsumoCoberturaNoDisponible,
			errConsumo,
		)
	}
	if recibo.ValidarPara(orden) != nil ||
		recibo.ConsumidaEn.After(finalizadaEn) {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return datosResultado.Comprobacion, nil
}

func autenticarAutoridadCobertura(
	ctx context.Context,
	presentador PresentadorAutoridadFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	materialPeticion []byte,
	rol RolAutoridadFuenteAnalisis,
	reloj Reloj,
	errDisponibilidad error,
) (identidadAutoridadFuenteAnalisis, error) {
	identidad, err := presentarYVerificarAutoridadFuenteAnalisis(
		ctx,
		presentador,
		confianza,
		materialPeticion,
		rol,
		reloj.Ahora(),
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return identidadAutoridadFuenteAnalisis{},
			errorDisponibilidadFuente(errDisponibilidad, errContexto)
	}
	if err != nil {
		return identidadAutoridadFuenteAnalisis{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return identidad, nil
}

func verificarRespuestaCobertura(
	ctx context.Context,
	verificador VerificadorRespuestaCobertura,
	identidad identidadAutoridadFuenteAnalisis,
	solicitud SolicitudVerificarRespuestaCobertura,
	reloj Reloj,
) (ConfirmacionRespuestaCobertura, error) {
	confirmacion, errVerificador :=
		verificador.VerificarRespuestaCobertura(ctx, solicitud)
	if err := ctx.Err(); err != nil {
		return ConfirmacionRespuestaCobertura{},
			errorDisponibilidadFuente(
				ErrVerificadorCoberturaNoDisponible,
				err,
			)
	}
	verificadaEn := reloj.Ahora()
	datos, errDatos := confirmacion.Datos()
	if errVerificador != nil {
		if errors.Is(
			errVerificador,
			ErrResultadoFuenteCoberturaNoConfiable,
		) {
			return ConfirmacionRespuestaCobertura{},
				ErrResultadoFuenteCoberturaNoConfiable
		}
		return ConfirmacionRespuestaCobertura{},
			errorDisponibilidadFuente(
				ErrVerificadorCoberturaNoDisponible,
				errVerificador,
			)
	}
	if errDatos != nil ||
		datos.VerificadorRef != identidad.autoridadRef ||
		confirmacion.ValidarPara(
			solicitud,
			verificadaEn,
			identidad.clavePrueba,
		) != nil {
		return ConfirmacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return confirmacion, nil
}

func dependenciaNulaFuenteCobertura(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
