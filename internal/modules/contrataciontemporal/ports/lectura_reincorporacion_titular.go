package ports

import (
	"context"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarAntecedenteReincorporacionTitular domain.ClaveCatalogo = "contratacion_temporal.seguimiento.consultar_reincorporacion_titular"
	TipoRecursoLecturaReincorporacionTitular                              = "lectura_reincorporacion_titular_contratacion_temporal"
	FinalidadLecturaReincorporacionTitular                                = "verificar_antecedente_reincorporacion_titular"
	AudienciaLecturaReincorporacionTitularV1                              = "vec_contratacion_temporal.lectura_reincorporacion_titular.v1"
	ResultadoAntecedenteCoincide                                          = "coincide"
	ResultadoAntecedenteNoCoincide                                        = "no_coincide"
)

// AntecedenteReincorporacionTitular contiene solo el cotejo necesario para la
// operación CT130. La lectura deja recibo y auditoría incluso si no coincide.
type AntecedenteReincorporacionTitular struct {
	Resultado, ExpedienteRef                      string
	CeseEventoRef, CeseReciboRef                  string
	LecturaRef, AuditoriaRef, ConsumoHuellaSHA256 string
	RegistradaEn                                  time.Time
}

func (a AntecedenteReincorporacionTitular) ValidoPara(m MaterialReincorporacionTitular) bool {
	if !m.Valido() || a.ExpedienteRef != m.ExpedienteRef ||
		!referenciaLecturaReincorporacionValida(a.LecturaRef) || !auditoriaLecturaReincorporacionValida(a.AuditoriaRef) ||
		!huellaSHA256OperacionAnalisisValida(a.ConsumoHuellaSHA256) || !domain.InstanteUTCCanonico(a.RegistradaEn) {
		return false
	}
	switch a.Resultado {
	case ResultadoAntecedenteCoincide:
		return domain.ReferenciaOpacaValida(a.CeseEventoRef) && domain.ReferenciaOpacaValida(a.CeseReciboRef)
	case ResultadoAntecedenteNoCoincide:
		return a.CeseEventoRef == "" && a.CeseReciboRef == ""
	default:
		return false
	}
}

func referenciaLecturaReincorporacionValida(ref string) bool {
	return strings.HasPrefix(ref, "lectura:") && hexMinusculasLecturaValido(strings.TrimPrefix(ref, "lectura:"), 64)
}

func auditoriaLecturaReincorporacionValida(ref string) bool {
	return strings.HasPrefix(ref, "aud_v3_") && hexMinusculasLecturaValido(strings.TrimPrefix(ref, "aud_v3_"), 32)
}

func hexMinusculasLecturaValido(valor string, longitud int) bool {
	if len(valor) != longitud {
		return false
	}
	for _, c := range valor {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// LectorAntecedenteReincorporacionTitular consume la autorización V3 dentro
// de la misma transacción que el cotejo, la auditoría y el recibo de lectura.
type LectorAntecedenteReincorporacionTitular interface {
	LeerAntecedenteReincorporacionTitular(context.Context, MaterialReincorporacionTitular, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (AntecedenteReincorporacionTitular, error)
}

type FuentePoliticaLecturaReincorporacionTitular interface {
	PoliticaLecturaReincorporacionTitular(context.Context, time.Time) (PoliticaOperacionSeguimiento, error)
}
