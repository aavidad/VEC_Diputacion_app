package ports

import (
	"context"
	"errors"
	"time"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrResumenBolsasNoDisponible = errors.New("bolsa: resumen de bolsas no disponible")
var ErrBolsaRRHHNoEncontrada = errors.New("bolsa: bolsa RRHH no encontrada")
var ErrCursorRRHHNoEncontrado = errors.New("bolsa: cursor RRHH no encontrado")

// SituacionResumenParticipacion es una participación de una constitución
// vigente (con su bolsa, categoría y fecha de constitución) con su situación
// vigente y, si existe, su estado de cese. Las filas llegan agrupadas por
// bolsa, en el orden de categoría de las constituciones y por orden de acta. Situacion
// es nil si la participación no tiene ninguna; Cese es nil si no consta cese.
type SituacionResumenParticipacion struct {
	BolsaRef           string
	CategoriaRef       string
	ConfirmadaEn       time.Time
	InstantaneaRef     string
	VersionInstantanea uint64
	Orden              uint64
	ParticipacionRef   string
	Situacion          *SituacionParticipacion
	Cese               *EstadoCese
}

// LectorResumenBolsas lee de una vez, para todas las bolsas constituidas, lo
// que el cuadro de RRHH necesita: situaciones, ceses y políticas de orden.
// LeerResumen hace las dos lecturas en una misma instantánea.
type LectorResumenBolsas interface {
	LeerResumen(context.Context, time.Time) ([]SituacionResumenParticipacion, map[string]dominio.PoliticaOrdenBolsa, error)
}

const (
	AccionRRHHBolsasConsultar          = "bolsa.rrhh.bolsas.consultar"
	AccionRRHHEstadisticasConsultar    = "bolsa.rrhh.estadisticas.consultar"
	AccionRRHHCandidatosConsultar      = "bolsa.rrhh.candidatos.consultar"
	AudienciaRRHHBolsasConsultar       = "vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1"
	AudienciaRRHHEstadisticasConsultar = "vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1"
	AudienciaRRHHCandidatosConsultar   = "vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1"
)

// ResumenBolsasNominal conserva la instantánea leída junto al consumo V3.
// El histórico de llamamientos se declara ausente mientras no exista su
// consulta; el contador de llamamientos en curso es otra magnitud.
type ResumenBolsasNominal struct {
	GeneradoEn   time.Time
	Bolsas       []BolsaResumenNominal
	Estadisticas *EstadisticasBolsasNominal
}

type BolsaResumenNominal struct {
	BolsaRef, CategoriaRef string
	ConfirmadaEn           time.Time
	VigenteHasta           *time.Time
	Politica               dominio.PoliticaOrdenBolsa
	PorEstado              map[string]int
	LlamamientosEnCurso    int
}

type EstadisticasBolsaNominal struct {
	BolsaRef, CategoriaRef, TipoLista string
	Vigente                           bool
	Total                             int
	PorEstado                         map[string]int
}

type EstadisticasBolsasNominal struct {
	BolsasTotal, BolsasVigentes, BolsasSustituidas int
	PersonasTotal                                  int
	PersonasPorEstado                              map[string]int
	LlamamientosEnCurso                            int
	HistoricoLlamamientosDisponible                bool
	PorBolsa                                       []EstadisticasBolsaNominal
}

type LectorResumenBolsasNominal interface {
	LeerResumenNominal(context.Context, string, bool, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ResumenBolsasNominal, error)
}
