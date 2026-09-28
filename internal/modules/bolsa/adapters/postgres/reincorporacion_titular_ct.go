package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	_ ports.FuenteReincorporacionesTitularCT      = (*FuenteReincorporacionesTitularCTPostgreSQL)(nil)
	_ ports.BuzonReincorporacionesTitularCT       = (*BuzonReincorporacionesTitularCTPostgreSQL)(nil)
	_ ports.RepositorioReincorporacionesTitularCT = (*RepositorioSituacionParticipacionPostgreSQL)(nil)
)

// FuenteReincorporacionesTitularCTPostgreSQL usa el pool de ejecución CT.
// Lee solo la fachada de publicación CT130, nunca tablas ajenas a Bolsa.
type FuenteReincorporacionesTitularCTPostgreSQL struct{ pool *pgxpool.Pool }

func NuevaFuenteReincorporacionesTitularCTPostgreSQL(pool *pgxpool.Pool) (*FuenteReincorporacionesTitularCTPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	return &FuenteReincorporacionesTitularCTPostgreSQL{pool: pool}, nil
}

func (f *FuenteReincorporacionesTitularCTPostgreSQL) LeerReincorporacionesTitular(ctx context.Context, desde *ports.CursorContratosParticipacion, limite int) ([]ports.PublicacionReincorporacionTitularCT, error) {
	if f == nil || f.pool == nil || ctx == nil || limite < 1 || limite > 100 ||
		(desde != nil && (desde.Posicion < 0 || desde.OrigenRef == "")) {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	var posicion *int64
	var referencia *string
	if desde != nil {
		posicion, referencia = &desde.Posicion, &desde.OrigenRef
	}
	filas, err := f.pool.Query(ctx, `SELECT evento_ref,huella_sha256,origen_ref,origen_posicion
		FROM vec_contratacion_temporal.leer_reincorporaciones_bolsa_v1($1,$2,$3)`, posicion, referencia, limite)
	if err != nil {
		return nil, errorReincorporacionTitular(err)
	}
	defer filas.Close()
	salida := make([]ports.PublicacionReincorporacionTitularCT, 0, limite)
	for filas.Next() {
		var e ports.PublicacionReincorporacionTitularCT
		if err := filas.Scan(&e.EventoRef, &e.HuellaSHA256, &e.OrigenRef, &e.OrigenPosicion); err != nil {
			return nil, errorReincorporacionTitular(err)
		}
		salida = append(salida, e)
	}
	if err := filas.Err(); err != nil {
		return nil, errorReincorporacionTitular(err)
	}
	return salida, nil
}

// BuzonReincorporacionesTitularCTPostgreSQL usa el pool ejecutor de Bolsa.
// La propia función SQL comprueba origen y enlace al cese; este adaptador no
// traduce el retorno del titular en una segunda restricción.
type BuzonReincorporacionesTitularCTPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoBuzonReincorporacionesTitularCTPostgreSQL(pool *pgxpool.Pool) (*BuzonReincorporacionesTitularCTPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	return &BuzonReincorporacionesTitularCTPostgreSQL{pool: pool}, nil
}

func (b *BuzonReincorporacionesTitularCTPostgreSQL) CursorReincorporacionesTitular(ctx context.Context) (ports.CursorContratosParticipacion, bool, error) {
	var c ports.CursorContratosParticipacion
	if b == nil || b.pool == nil || ctx == nil {
		return c, false, ports.ErrReincorporacionTitularNoDisponible
	}
	err := b.pool.QueryRow(ctx, `SELECT origen_posicion,origen_ref FROM vec_bolsa_llamamientos.cursor_reincorporaciones_titular_ct_v1()`).Scan(&c.Posicion, &c.OrigenRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.CursorContratosParticipacion{}, false, nil
	}
	if err != nil {
		return c, false, errorReincorporacionTitular(err)
	}
	return c, true, nil
}

func (b *BuzonReincorporacionesTitularCTPostgreSQL) RegistrarReincorporacionTitular(ctx context.Context, e ports.PublicacionReincorporacionTitularCT) (ports.ResultadoReincorporacionTitularCT, error) {
	var r ports.ResultadoReincorporacionTitularCT
	if b == nil || b.pool == nil || ctx == nil || e.OrigenRef == "" || e.EventoRef != e.OrigenRef ||
		len(e.HuellaSHA256) != 64 || e.OrigenPosicion < 0 {
		return r, ports.ErrReincorporacionTitularNoDisponible
	}
	err := pgx.BeginTxFunc(ctx, b.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT reutilizada,estado,cese_evento_ref,disponible_desde
			FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1($1,$2,$3)`,
			e.OrigenRef, e.HuellaSHA256, e.OrigenPosicion).
			Scan(&r.Reutilizada, &r.Estado, &r.CeseEventoRef, &r.DisponibleDesde)
	})
	if err != nil {
		return ports.ResultadoReincorporacionTitularCT{}, errorReincorporacionTitular(err)
	}
	r.CeseAplicado = r.Estado == "cese_aplicado"
	return r, nil
}

func (r *RepositorioSituacionParticipacionPostgreSQL) ListarReincorporacionesTitular(ctx context.Context, ref, actor string, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ports.ReincorporacionTitularFicha, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" || actor == "" || m.ValidarEstructura() != nil {
		return nil, ports.ErrReincorporacionTitularNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorReincorporacionTitular(err)
	}
	defer tx.Rollback(context.Background())
	filas, err := tx.Query(ctx, `SELECT evento_ref,expediente_ref,relacion_ref,fecha_efectiva,recibo_ct_ref,cese_evento_ref,
		estado,disponible_desde,regla_version,regla_huella_sha256
		FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`,
		ref, actor, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return nil, errorReincorporacionTitular(err)
	}
	defer filas.Close()
	salida := make([]ports.ReincorporacionTitularFicha, 0)
	for filas.Next() {
		var item ports.ReincorporacionTitularFicha
		var huella *string
		if err := filas.Scan(&item.EventoRef, &item.ExpedienteRef, &item.RelacionRef, &item.FechaEfectiva,
			&item.ReciboCTRef, &item.CeseEventoRef, &item.Estado, &item.DisponibleDesde, &item.ReglaVersion, &huella); err != nil {
			return nil, errorReincorporacionTitular(err)
		}
		if huella != nil {
			item.ReglaHuellaSHA256 = *huella
		}
		salida = append(salida, item)
	}
	if err := filas.Err(); err != nil {
		return nil, errorReincorporacionTitular(err)
	}
	filas.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, errorReincorporacionTitular(err)
	}
	return salida, nil
}

func errorReincorporacionTitular(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return dominiovec.ErrAutorizacionDenegada
	}
	return ports.ErrReincorporacionTitularNoDisponible
}
