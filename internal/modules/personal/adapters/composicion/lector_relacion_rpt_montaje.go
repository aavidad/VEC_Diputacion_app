package composicion

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	personalpostgres "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rpt"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type DependenciasLectorRelacionRPT struct {
	Identidad         ResolutorIdentidadLectorRelacionRPT
	Emisor            EmisorMaterialLectorRelacionRPTV3
	Motivo            vecdomain.ReferenciaEntradaCatalogo
	Lectura, Intentos *pgxpool.Pool
	Ahora             func() time.Time
}

// ComponerLectorRelacionSeleccionadaRPT prepara el consumidor de la selección
// 356 con autoridades reales y pools segregados. No monta HTTP ni provisión.
// La ausencia de SQL, concesión o registrador mantiene cada lectura cerrada.
func ComponerLectorRelacionSeleccionadaRPT(d DependenciasLectorRelacionRPT) (*personalrpt.LectorRelacionSeleccionadaRPT, error) {
	if d.Lectura == nil || d.Intentos == nil || d.Lectura == d.Intentos || d.Ahora == nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	a, err := NuevoProveedorAutorizacionLectorRelacionRPT(d.Identidad, d.Emisor, d.Motivo)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	r, err := personalpostgres.NuevoRepositorioLectorRelacionRPTPostgreSQL(d.Lectura)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	destino, err := personalpostgres.NuevoRegistroIntentosLectorRPTPostgreSQL(d.Intentos)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	i, err := NuevoRegistroIntentosLectorRelacionRPT(d.Identidad, destino)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	s, err := personalapp.NuevoServicioLectorRelacionRPT(a, r, i, d.Ahora)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return personalrpt.NuevoLectorRelacionSeleccionadaRPT(s, i)
}
