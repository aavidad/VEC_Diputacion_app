package composicion

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	personalcronos "vec-diputacion-granada/internal/modules/personal/adapters/cronos"
	personalpostgres "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// DependenciasLectorVinculoCRN11 procede de la composición interna. El pool
// pertenece al ejecutor de Personal y el emisor a la autoridad V3 común.
// Ninguno se sustituye por la autorización de recuperación de Cronos.
type DependenciasLectorVinculoCRN11 struct {
	Identidad ResolutorIdentidadVinculoCRN11
	Emisor    EmisorMaterialVinculoCRN11V3
	Motivo    vecdomain.ReferenciaEntradaCatalogo
	Personal  *pgxpool.Pool
	Ahora     func() time.Time
}

// ComponerLectorVinculoPropioCRN11 prepara el proveedor real para el constructor
// explícito de correcciones de Cronos. No registra rutas ni declara instalada
// la consulta nominal: si falta SQL, política o fuente, la lectura queda cerrada.
func ComponerLectorVinculoPropioCRN11(d DependenciasLectorVinculoCRN11) (cronosports.LectorVinculoPropioHistoricoCRN11, error) {
	if d.Personal == nil || d.Ahora == nil {
		return nil, personaldomain.ErrVinculoCRN11NoDisponible
	}
	autorizador, err := NuevoProveedorAutorizacionVinculoCRN11(d.Identidad, d.Emisor, d.Motivo)
	if err != nil {
		return nil, personaldomain.ErrVinculoCRN11NoDisponible
	}
	repositorio, err := personalpostgres.NuevoRepositorioVinculoPropioCRN11PostgreSQL(d.Personal)
	if err != nil {
		return nil, personaldomain.ErrVinculoCRN11NoDisponible
	}
	servicio, err := personalapp.NuevoServicioVinculoPropioCRN11(autorizador, repositorio, d.Ahora)
	if err != nil {
		return nil, personaldomain.ErrVinculoCRN11NoDisponible
	}
	lector, err := personalcronos.NuevoLectorVinculoPropioCRN11(servicio)
	if err != nil {
		return nil, personaldomain.ErrVinculoCRN11NoDisponible
	}
	return lector, nil
}
