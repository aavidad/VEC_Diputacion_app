package ports

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	// TiempoMaximoFuenteCobertura limita técnicamente una consulta individual.
	// Cada despliegue puede escoger un valor menor sin modificar el núcleo.
	TiempoMaximoFuenteCobertura = 30 * time.Second
)

var (
	ErrPeticionFuenteCoberturaInvalida = errors.New(
		"contratacion temporal: peticion a fuente de cobertura invalida",
	)
	ErrFuenteCoberturaNoDisponible = errors.New(
		"contratacion temporal: fuente de cobertura no disponible",
	)
	ErrResultadoFuenteCoberturaNoConfiable = errors.New(
		"contratacion temporal: resultado de fuente de cobertura no confiable",
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
		s.VersionExpediente == 0 || s.Catalogo.Validar() != nil ||
		!s.ViaClave.Valida() || s.Comprobacion.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.CategoriaRef) ||
		s.Periodo.Validar() != nil ||
		!domain.InstanteUTCCanonico(s.SolicitadaEn) {
		return ErrPeticionFuenteCoberturaInvalida
	}
	return nil
}

// ResultadoConsultaCobertura liga la respuesta a todas las coordenadas que
// pueden cambiar su significado. FuenteRef y ReciboRef identifican la
// procedencia efectiva sin copiar datos personales del sistema consultado.
type ResultadoConsultaCobertura struct {
	PeticionRef         string
	OrganizacionRef     string
	ExpedienteRef       string
	VersionExpediente   uint64
	Catalogo            domain.IdentidadCatalogoViasCobertura
	ViaClave            domain.ClaveCatalogo
	ProcedenciaClave    domain.ClaveCatalogo
	CategoriaRef        string
	Periodo             domain.PeriodoPrevisto
	Comprobacion        domain.ComprobacionCobertura
	DefinicionFuenteRef string
}

func (r ResultadoConsultaCobertura) ValidarPara(
	solicitud SolicitudConsultarCobertura,
) error {
	if solicitud.Validar() != nil ||
		r.PeticionRef != solicitud.PeticionRef ||
		r.OrganizacionRef != solicitud.OrganizacionRef ||
		r.ExpedienteRef != solicitud.ExpedienteRef ||
		r.VersionExpediente != solicitud.VersionExpediente ||
		!r.Catalogo.CoincideExactamente(solicitud.Catalogo) ||
		r.ViaClave != solicitud.ViaClave ||
		r.ProcedenciaClave != solicitud.Comprobacion.Procedencia.Clave ||
		r.CategoriaRef != solicitud.CategoriaRef ||
		!r.Periodo.Inicio.Equal(solicitud.Periodo.Inicio) ||
		!r.Periodo.Fin.Equal(solicitud.Periodo.Fin) ||
		r.Comprobacion.Validar() != nil ||
		r.Comprobacion.Detalle != "" ||
		r.Comprobacion.Clave != solicitud.Comprobacion.Clave ||
		r.DefinicionFuenteRef !=
			solicitud.Comprobacion.Procedencia.DefinicionFuenteRef ||
		r.Comprobacion.EvaluadaEn.Before(solicitud.SolicitadaEn) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

// FuenteComprobacionCobertura es un puerto de salida genérico. El adaptador
// puede despachar por la definición gobernada a Bolsa, SAE, convocatorias u
// otros conectores, pero nunca consultar sus tablas desde este módulo.
type FuenteComprobacionCobertura interface {
	ConsultarCobertura(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error)
}

// ConsultarCoberturaConFuente aplica el fallo cerrado, el límite temporal y la
// validación de ligaduras de forma uniforme para todos los conectores.
func ConsultarCoberturaConFuente(
	ctx context.Context,
	fuente FuenteComprobacionCobertura,
	reloj Reloj,
	solicitud SolicitudConsultarCobertura,
	tiempoMaximo time.Duration,
) (domain.ComprobacionCobertura, error) {
	if ctx == nil || dependenciaNulaFuenteCobertura(fuente) ||
		dependenciaNulaFuenteCobertura(reloj) ||
		solicitud.Validar() != nil || tiempoMaximo <= 0 ||
		tiempoMaximo > TiempoMaximoFuenteCobertura {
		return domain.ComprobacionCobertura{}, ErrPeticionFuenteCoberturaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.ComprobacionCobertura{}, nuevoErrorFuenteCobertura(err)
	}

	ctxConsulta, cancelar := context.WithTimeout(ctx, tiempoMaximo)
	defer cancelar()
	resultado, err := fuente.ConsultarCobertura(ctxConsulta, solicitud)
	if err != nil {
		return domain.ComprobacionCobertura{}, nuevoErrorFuenteCobertura(err)
	}
	if err := ctxConsulta.Err(); err != nil {
		return domain.ComprobacionCobertura{}, nuevoErrorFuenteCobertura(err)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	finalizadaEn := reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !domain.InstanteUTCCanonico(finalizadaEn) ||
		finalizadaEn.Before(solicitud.SolicitadaEn) ||
		resultado.Comprobacion.EvaluadaEn.After(finalizadaEn) {
		return domain.ComprobacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return resultado.Comprobacion, nil
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

type errorFuenteCobertura struct {
	causa error
}

func (e errorFuenteCobertura) Error() string {
	return ErrFuenteCoberturaNoDisponible.Error()
}

func (e errorFuenteCobertura) Unwrap() []error {
	return []error{ErrFuenteCoberturaNoDisponible, e.causa}
}

func nuevoErrorFuenteCobertura(causa error) error {
	return errorFuenteCobertura{causa: causa}
}
