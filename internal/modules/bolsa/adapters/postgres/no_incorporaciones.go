package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var _ ports.BuzonNoIncorporaciones = (*BuzonNoIncorporacionesPostgreSQL)(nil)

// BuzonNoIncorporacionesPostgreSQL es la bandeja de Bolsa 000042 con el rol
// ejecutor de Bolsa. No lee ni escribe tablas de Contratación temporal.
type BuzonNoIncorporacionesPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoBuzonNoIncorporacionesPostgreSQL(pool *pgxpool.Pool) (*BuzonNoIncorporacionesPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrContratosParticipacionNoDisponible
	}
	return &BuzonNoIncorporacionesPostgreSQL{pool: pool}, nil
}

func (b *BuzonNoIncorporacionesPostgreSQL) CursorNoIncorporaciones(ctx context.Context) (ports.CursorContratosParticipacion, bool, error) {
	var c ports.CursorContratosParticipacion
	if b == nil || b.pool == nil || ctx == nil {
		return c, false, ports.ErrContratosParticipacionNoDisponible
	}
	err := b.pool.QueryRow(ctx, `SELECT origen_posicion, origen_ref FROM vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1()`).Scan(&c.Posicion, &c.OrigenRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.CursorContratosParticipacion{}, false, nil
	}
	if err != nil {
		return ports.CursorContratosParticipacion{}, false, errorContratosParticipacion(err)
	}
	return c, true, nil
}

func (b *BuzonNoIncorporacionesPostgreSQL) RegistrarNoIncorporacion(ctx context.Context, e ports.EventoNoIncorporacionRecibido) (ports.ResultadoRegistroNoIncorporacion, error) {
	var r ports.ResultadoRegistroNoIncorporacion
	if b == nil || b.pool == nil || ctx == nil || e.Evento.Validar() != nil || len(e.Contenido) == 0 || e.HuellaSHA256 == "" ||
		e.OrigenCreadaEn.IsZero() || e.OrigenPosicion < 0 {
		return r, ports.ErrContratosParticipacionNoDisponible
	}
	var consecuencia *string
	if e.Consecuencia != nil {
		j, err := json.Marshal(e.Consecuencia)
		if err != nil {
			return r, ports.ErrContratosParticipacionNoDisponible
		}
		texto := string(j)
		consecuencia = &texto
	}
	var estado, participacion *string
	var enCuarentena bool
	err := pgx.BeginTxFunc(ctx, b.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT reutilizado, estado, participacion_ref, en_cuarentena FROM vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1($1::text::jsonb,$2,$3,$4,$5::text::jsonb)`,
			string(e.Contenido), e.HuellaSHA256, e.OrigenCreadaEn.UTC().Truncate(time.Microsecond), e.OrigenPosicion, consecuencia).
			Scan(&r.Reutilizado, &estado, &participacion, &enCuarentena)
	})
	if err != nil {
		return ports.ResultadoRegistroNoIncorporacion{}, errorContratosParticipacion(err)
	}
	// Una entrega divergente ya quedó en la cuarentena de Bolsa: se informa
	// como divergente para que el relevo la registre y continúe.
	if enCuarentena {
		return ports.ResultadoRegistroNoIncorporacion{}, ports.ErrEventoContratoDivergente
	}
	if estado == nil {
		return ports.ResultadoRegistroNoIncorporacion{}, fmt.Errorf("%w: estado ausente", ports.ErrContratosParticipacionNoDisponible)
	}
	r.Estado = *estado
	if participacion != nil {
		r.ParticipacionRef = *participacion
	}
	return r, nil
}
