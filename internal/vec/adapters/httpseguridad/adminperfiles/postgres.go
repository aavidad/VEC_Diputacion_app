package adminperfiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

const consultarCuenta = `SELECT resultado,acuse FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
const vincularSesion = `SELECT resultado,acuse FROM vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`

type transactor interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// PostgreSQL únicamente consume las fachadas nominales IS16. La
// composición entrega un LOGIN dedicado sin SELECT ni mutación de tablas.
type PostgreSQL struct {
	pool            transactor
	reloj           h.Reloj
	identificadores FuenteIdentificadoresADMIN
	seudonimizador  SeudonimizadorFuenteADMIN
}

// Sin el proveedor original de identificadores no se puede registrar una
// sesión mediante los HMAC persistidos. La composición debe usar ConFuente.
func NuevoPostgreSQL(ctx context.Context, pool *pgxpool.Pool, reloj h.Reloj) (*PostgreSQL, error) {
	return nil, api.ErrConfiguracionIncompleta
}

func (p *PostgreSQL) ResolverCuentaADMIN(ctx context.Context, o ObservacionADMIN) (CuentaADMIN, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || nulo(p.pool) || nulo(p.reloj) || !o.Valida(p.reloj.Ahora().UTC()) {
		return CuentaADMIN{}, api.ErrAutenticacionRequerida
	}
	if nulo(p.identificadores) || nulo(p.seudonimizador) {
		return CuentaADMIN{}, api.ErrConfiguracionIncompleta
	}
	var cuenta CuentaADMIN
	var decision error
	args, err := argumentosAuditadosADMIN(ctx, o)
	if err != nil {
		return CuentaADMIN{}, err
	}
	err = p.transaccion(ctx, func(tx pgx.Tx) error {
		sub, err := tx.Begin(ctx)
		if err != nil {
			return api.ErrConfiguracionIncompleta
		}
		defer sub.Rollback(ctx)
		var bruto, acuse []byte
		if err := sub.QueryRow(ctx, consultarCuenta, args...).Scan(&bruto, &acuse); err != nil {
			return errorConsultaIS16(err)
		}
		datos, denegacion, err := leerResultadoIS16(bruto, acuse, args[9].(string), args[10].(string), p.reloj.Ahora())
		if err != nil {
			return err
		}
		decision = denegacion
		if decision != nil {
			return confirmarSubtransaccionIS16(ctx, sub)
		} // conserva auditoría de denegación.
		cuenta, err = p.cuentaDesdeFuente(ctx, o, datos)
		if err == nil {
			return confirmarSubtransaccionIS16(ctx, sub)
		}
		if err = sub.Rollback(ctx); err != nil {
			return api.ErrConfiguracionIncompleta
		}
		decision, err = p.rechazarCotejoIS16(ctx, tx, args[9].(string), args[10].(string), "resolver_cuenta_admin")
		return err
	})
	if err != nil {
		return CuentaADMIN{}, errorAutoridad(err)
	}
	if decision != nil {
		return CuentaADMIN{}, decision
	}
	return cuenta, nil
}

func (p *PostgreSQL) VincularSesionADMIN(ctx context.Context, o ObservacionADMIN, esperada CuentaADMIN, refs ReferenciasSesionADMIN) error {
	_, err := p.VincularSesionADMINConAcuse(ctx, o, esperada, refs)
	return err
}

func argumentosCuenta(o ObservacionADMIN) []any {
	return []any{o.Entorno, o.Host, o.Audiencia, o.CertificadoSHA256, o.CASHA256,
		o.AutenticacionVerificadaEn, o.RevocacionVerificadaEn}
}

func argumentos(o ObservacionADMIN) []any {
	return []any{o.Entorno, o.Host, o.Audiencia, o.CertificadoSHA256, o.CASHA256,
		o.AutenticacionVerificadaEn, o.RevocacionVerificadaEn,
		o.CRLVigenteHasta, o.CertificadoVigenteHasta}
}

func (p *PostgreSQL) transaccion(ctx context.Context, fn func(pgx.Tx) error) error {
	return p.transaccionConAislamiento(ctx, pgx.Serializable, fn)
}

func (p *PostgreSQL) transaccionConAislamiento(ctx context.Context, aislamiento pgx.TxIsoLevel, fn func(pgx.Tx) error) error {
	// La política compartida de reintento pertenece al consumidor completo.
	// Una carrera en esta lectura/binder obliga a obtener evidencia F nueva.
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: aislamiento, AccessMode: pgx.ReadWrite})
	if err != nil {
		return api.ErrConfiguracionIncompleta
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(c)
	}()
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','4s',true),set_config('statement_timeout','8s',true)`); err != nil {
		return api.ErrConfiguracionIncompleta
	}
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		if postgresqlcomun.EsCarreraSerializable(err) {
			// Un 40001/40P01 en el COMMIT garantiza que no se aplicó nada. Se
			// conserva la clase sin el error del proveedor para que el
			// consumidor que lo admita repita la transacción entera.
			return errors.Join(api.ErrConfiguracionIncompleta, errCommitCarreraSerializable{})
		}
		return api.ErrConfiguracionIncompleta
	}
	return nil
}

// errCommitCarreraSerializable marca un COMMIT abortado por serialización o
// interbloqueo. Sigue siendo ErrConfiguracionIncompleta para errors.Is.
type errCommitCarreraSerializable struct{}

func (errCommitCarreraSerializable) Error() string             { return "commit_carrera_serializable" }
func (errCommitCarreraSerializable) CarreraSerializable() bool { return true }

var _ FuenteCuentasADMIN = (*PostgreSQL)(nil)
