package ports

import (
	"context"
	"errors"
	"time"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// Constitución de una bolsa (B1→B2): RRHH confirma un acta de importación
// Convoca y la lista definitiva pasa a ser una bolsa con su orden. El
// repositorio persiste los canónicos del dominio y el vínculo acta → bolsa;
// nunca datos personales en claro (los nombres siguen en el staging protegido
// del acta y se recuperan al leer).

var (
	ErrConstitucionBolsaInvalida     = errors.New("bolsa constitucion: solicitud invalida")
	ErrConstitucionBolsaNoDisponible = errors.New("bolsa constitucion: repositorio no disponible")
	ErrConstitucionBolsaEnConflicto  = errors.New("bolsa constitucion: acta ya constituida en otra bolsa")
)

// EntradaConstitucion vincula una posición de la instantánea con la fila del
// acta de la que procede.
type EntradaConstitucion struct {
	Orden            uint64
	ParticipacionRef string
	FilaNumero       int
}

// Constitucion es lo que se persiste: bolsa e instantánea canónicas más el
// vínculo con el acta y el actor de RRHH que la confirma.
type Constitucion struct {
	ActaRef      string
	ActorRef     string
	CategoriaRef string
	Bolsa        dominio.BolsaConstituida
	Instantanea  dominio.InstantaneaOrdenBolsa
	Entradas     []EntradaConstitucion
	ConfirmadaEn time.Time
}

// ReciboConstitucion identifica la constitución registrada (o la ya existente
// para la misma acta: la operación es idempotente por acta).
type ReciboConstitucion struct {
	Reutilizada        bool
	ActaRef            string
	BolsaRef           string
	VersionBolsa       uint64
	InstantaneaRef     string
	VersionInstantanea uint64
	ConfirmadaEn       time.Time
}

// ConstitucionVigente es la última constitución de cada categoría.
type ConstitucionVigente struct {
	ActaRef              string
	CategoriaRef         string
	Bolsa                dominio.BolsaConstituida
	Instantanea          dominio.InstantaneaOrdenBolsa
	Estado               string
	TotalParticipaciones uint64
	ConfirmadaEn         time.Time
}

type RepositorioConstitucion interface {
	Constituir(context.Context, Constitucion) (ReciboConstitucion, error)
	ListarVigentes(context.Context) ([]ConstitucionVigente, error)
	Entradas(context.Context, string, uint64) ([]EntradaConstitucion, error)
}
