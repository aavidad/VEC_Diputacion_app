package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
)

const registrarIntentoCartera = `SELECT auditoria_ref,secuencia::text,huella_sha256,correlacion_ref,registrada_en
FROM vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1($1::bytea,$2::bytea,$3::jsonb)`
const registrarIntentoRetiradaCartera = `SELECT auditoria_ref,secuencia::text,huella_sha256,correlacion_ref,registrada_en
FROM vec_autorizacion_atestada_v3.registrar_intento_retirada_cartera_meritos_v1($1::bytea,$2::bytea,$3::jsonb)`

type AuditoriaIntentosCartera struct{ pool iniciador }

var _ ports.AuditorIntentosCartera = (*AuditoriaIntentosCartera)(nil)
var _ ports.AuditorIntentosRetiradaCartera = (*AuditoriaIntentosCartera)(nil)

func NuevaAuditoriaIntentosCartera(pool *pgxpool.Pool) (*AuditoriaIntentosCartera, error) {
	if nulo(pool) {
		return nil, ports.ErrCarteraNoDisponible
	}
	return &AuditoriaIntentosCartera{pool: pool}, nil
}

func (r *AuditoriaIntentosCartera) RegistrarIntentoCartera(ctx context.Context,
	c ports.CapturaLecturaCartera) error {
	return r.registrar(ctx, c, registrarIntentoCartera)
}

func (r *AuditoriaIntentosCartera) RegistrarIntentoRetiradaCartera(ctx context.Context,
	c ports.CapturaLecturaCartera) error {
	return r.registrar(ctx, c, registrarIntentoRetiradaCartera)
}

func (r *AuditoriaIntentosCartera) registrar(ctx context.Context,
	c ports.CapturaLecturaCartera, consulta string) error {
	if ctx == nil || r == nil || nulo(r.pool) || c.PersonaRef() == "" ||
		c.CorrelacionRef() == "" || len(c.ContextoCanonico()) == 0 ||
		len(c.VinculoCanonico()) == 0 || len(c.OrdenJSON()) == 0 {
		return ports.ErrCarteraNoDisponible
	}
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		return r.registrarIntento(ctx, c, consulta)
	})
	if err != nil {
		return errorCarteraSQL(ctx, err)
	}
	return nil
}

func (r *AuditoriaIntentosCartera) registrarIntento(ctx context.Context,
	c ports.CapturaLecturaCartera, consulta string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	if nulo(tx) {
		return ports.ErrCarteraNoDisponible
	}
	defer revertir(tx)
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
set_config('row_security','on',true),set_config('timezone','UTC',true),
set_config('lock_timeout','2s',true),set_config('statement_timeout','10s',true),
set_config('idle_in_transaction_session_timeout','15s',true)`); err != nil {
		return err
	}
	var ref, secuencia, huella, correlacion string
	var registrada time.Time
	err = tx.QueryRow(ctx, consulta,
		c.ContextoCanonico(), c.VinculoCanonico(), c.OrdenJSON()).
		Scan(&ref, &secuencia, &huella, &correlacion, &registrada)
	if err != nil {
		return err
	}
	numero, err := strconv.ParseUint(secuencia, 10, 64)
	if err != nil || numero == 0 || !domain.ReferenciaValida(ref) ||
		!huellaSHA256CarteraSQL(huella) || correlacion != c.CorrelacionRef() ||
		registrada.IsZero() || registrada.After(time.Now().UTC().Add(time.Second)) {
		return ports.ErrCarteraNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func huellaSHA256CarteraSQL(h string) bool {
	if len(h) != 64 {
		return false
	}
	for _, c := range h {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}
