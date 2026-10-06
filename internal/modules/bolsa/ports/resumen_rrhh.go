package ports

import (
	"context"
	"errors"
	"time"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var ErrResumenBolsasNoDisponible = errors.New("bolsa: resumen de bolsas no disponible")

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
type LectorResumenBolsas interface {
	LeerResumenSituaciones(context.Context, time.Time) ([]SituacionResumenParticipacion, error)
	LeerPoliticasOrdenVigentes(context.Context, time.Time) (map[string]dominio.PoliticaOrdenBolsa, error)
}
