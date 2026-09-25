package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	_ ports.RepositorioContratosParticipacion = (*RepositorioSituacionParticipacionPostgreSQL)(nil)
	_ ports.BuzonContratosParticipacion       = (*BuzonContratosParticipacionPostgreSQL)(nil)
)

// ListarContratos consume el material V3 en la misma transacción que lee.
func (r *RepositorioSituacionParticipacionPostgreSQL) ListarContratos(ctx context.Context, ref, actor string, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ports.ContratoParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || ref == "" || actor == "" || m.ValidarEstructura() != nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ports.ErrContratosParticipacionNoDisponible, err)
	}
	defer tx.Rollback(context.Background())
	filas, err := tx.Query(ctx, `SELECT evento_ref,tipo,inicio,fin_previsto,coalesce(modalidad_clave,''),coalesce(categoria_ref,''),coalesce(causa_clave,''),expediente_ref,llamamiento_ref,ocurrido_en FROM vec_bolsa_llamamientos.listar_contratos_participacion_v1($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`, ref, actor, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return nil, errorContratosParticipacion(err)
	}
	defer filas.Close()
	items := make([]ports.ContratoParticipacion, 0)
	for filas.Next() {
		var c ports.ContratoParticipacion
		if err := filas.Scan(&c.EventoRef, &c.Tipo, &c.Inicio, &c.FinPrevisto, &c.ModalidadClave, &c.CategoriaRef, &c.CausaClave, &c.ExpedienteRef, &c.LlamamientoRef, &c.OcurridoEn); err != nil {
			return nil, errorContratosParticipacion(err)
		}
		items = append(items, c)
	}
	if err := filas.Err(); err != nil {
		return nil, errorContratosParticipacion(err)
	}
	filas.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, errorContratosParticipacion(err)
	}
	return items, nil
}

// BuzonContratosParticipacionPostgreSQL es el inbox B13 con el rol ejecutor
// de Bolsa. No lee ni escribe tablas de Contratación temporal.
type BuzonContratosParticipacionPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoBuzonContratosParticipacionPostgreSQL(pool *pgxpool.Pool) (*BuzonContratosParticipacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	return &BuzonContratosParticipacionPostgreSQL{pool: pool}, nil
}

func (b *BuzonContratosParticipacionPostgreSQL) CursorContratos(ctx context.Context) (ports.CursorContratosParticipacion, bool, error) {
	var c ports.CursorContratosParticipacion
	if b == nil || b.pool == nil || ctx == nil {
		return c, false, ports.ErrContratosParticipacionNoDisponible
	}
	err := b.pool.QueryRow(ctx, `SELECT origen_posicion, origen_ref FROM vec_bolsa_llamamientos.cursor_contratos_participacion_v1()`).Scan(&c.Posicion, &c.OrigenRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.CursorContratosParticipacion{}, false, nil
	}
	if err != nil {
		return ports.CursorContratosParticipacion{}, false, errorContratosParticipacion(err)
	}
	return c, true, nil
}

func (b *BuzonContratosParticipacionPostgreSQL) RegistrarContrato(ctx context.Context, e ports.EventoContratoRecibido) (ports.ResultadoRegistroContrato, error) {
	var r ports.ResultadoRegistroContrato
	if b == nil || b.pool == nil || ctx == nil || e.Evento.Validar() != nil || len(e.Contenido) == 0 || e.HuellaSHA256 == "" || e.OrigenCreadaEn.IsZero() || e.OrigenPosicion < 0 {
		return r, ports.ErrContratosParticipacionNoDisponible
	}
	var participacion *string
	err := pgx.BeginTxFunc(ctx, b.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT reutilizado, participacion_ref FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1($1::text::jsonb,$2,$3,$4)`, string(e.Contenido), e.HuellaSHA256, e.OrigenCreadaEn.UTC().Truncate(time.Microsecond), e.OrigenPosicion).Scan(&r.Reutilizado, &participacion)
	})
	if err != nil {
		return ports.ResultadoRegistroContrato{}, errorContratosParticipacion(err)
	}
	if participacion != nil {
		r.ParticipacionRef = *participacion
	}
	return r, nil
}

func errorContratosParticipacion(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "VBC01":
			return fmt.Errorf("%w: %w", ports.ErrEventoContratoDivergente, err)
		case "42501":
			return fmt.Errorf("%w: %w", dominiovec.ErrAutorizacionDenegada, err)
		}
	}
	return fmt.Errorf("%w: %w", ports.ErrContratosParticipacionNoDisponible, err)
}
