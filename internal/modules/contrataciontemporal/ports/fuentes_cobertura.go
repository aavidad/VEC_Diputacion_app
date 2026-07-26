package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const redaccionResumenOrdenConsumoCobertura = "[RESUMEN-ORDEN-CONSUMO-COBERTURA-REDACTADO]"

var (
	ErrPeticionFuenteCoberturaInvalida = errors.New(
		"contratacion temporal: peticion a fuente de cobertura invalida",
	)
	ErrResultadoFuenteCoberturaNoConfiable = errors.New(
		"contratacion temporal: resultado de fuente de cobertura no confiable",
	)
	ErrRespuestaCoberturaYaConsumida = errors.New(
		"contratacion temporal: respuesta de cobertura ya consumida con otros datos",
	)
)

func (ResumenOrdenConsumoCobertura) String() string {
	return redaccionResumenOrdenConsumoCobertura
}

func (r ResumenOrdenConsumoCobertura) GoString() string { return r.String() }
func (r ResumenOrdenConsumoCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r ResumenOrdenConsumoCobertura) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (r ResumenOrdenConsumoCobertura) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}
func (r ResumenOrdenConsumoCobertura) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

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
