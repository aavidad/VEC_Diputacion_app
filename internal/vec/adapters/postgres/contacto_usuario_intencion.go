package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// RegistroContactoRecuperablePostgreSQL confirma una intención de contacto
// protegida. La función SQL conserva el recibo original y sólo añade la
// auditoría y el consumo propios del reintento cuando la intención coincide.
type RegistroContactoRecuperablePostgreSQL struct{ pool iniciadorTransacciones }

func NuevoRegistroContactoRecuperablePostgreSQL(pool *pgxpool.Pool) (*RegistroContactoRecuperablePostgreSQL, error) {
	return nuevoRegistroContactoRecuperablePostgreSQL(pool)
}

func nuevoRegistroContactoRecuperablePostgreSQL(pool iniciadorTransacciones) (*RegistroContactoRecuperablePostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, vecapp.ErrContactoUsuarioNoDisponible
	}
	return &RegistroContactoRecuperablePostgreSQL{pool: pool}, nil
}

func (r *RegistroContactoRecuperablePostgreSQL) GuardarContactoRecuperable(ctx context.Context, o ports.OrdenRegistroContactoRecuperable) (ports.ResultadoRegistroContactoRecuperable, error) {
	vacio := ports.ResultadoRegistroContactoRecuperable{}
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil || ctx.Err() != nil || vecapp.ValidarOrdenRegistroContactoRecuperable(o, time.Now().UTC().Truncate(time.Microsecond)) != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}

	a := o.Autorizacion
	recurso, auditoria, err := contactoRecursoAuditoria(a.Recurso, a.Auditoria)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return vacio, contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return vacio, contactoError(ctx, err)
	}

	p := o.Preparacion.Registro
	var recuperado bool
	var sujeto string
	var version uint64
	var original, auditoriaIntento []byte
	var consumoOriginalRef, consumoOriginalHuella string
	var consumoIntentoRef, consumoIntentoHuella string
	err = tx.QueryRow(ctx, `SELECT recuperado, sujeto_ref, version, recibo_original, consumo_original_ref, consumo_original_huella_sha256, auditoria_intento, consumo_intento_ref, consumo_intento_huella_sha256 FROM vec_contacto_usuario_v1.registrar_contacto_v2($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(accionRegistroContacto(p.VersionEsperada), a.Material, p.PayloadNegocio, recurso, auditoria)...).Scan(
		&recuperado, &sujeto, &version, &original, &consumoOriginalRef, &consumoOriginalHuella,
		&auditoriaIntento, &consumoIntentoRef, &consumoIntentoHuella,
	)
	if err != nil {
		return vacio, contactoRecuperableError(ctx, err)
	}

	resultado, err := vecapp.DecodificarResultadoRegistroContactoRecuperable(o, recuperado, sujeto, version, original, consumoOriginalRef, consumoOriginalHuella, auditoriaIntento, consumoIntentoRef, consumoIntentoHuella)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, contactoError(ctx, err)
	}
	// Tras confirmar, una respuesta perdida se recupera mediante la intención;
	// este adaptador no vuelve a ejecutar SQL ni fabrica otro recibo.
	return resultado, nil
}

func contactoRecuperableError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "P1103":
			return vecapp.ErrVersionContactoDivergente
		case "P1104":
			return vecapp.ErrIntencionContactoDivergente
		}
	}
	return contactoError(ctx, err)
}

var _ ports.RegistroContactoRecuperable = (*RegistroContactoRecuperablePostgreSQL)(nil)
