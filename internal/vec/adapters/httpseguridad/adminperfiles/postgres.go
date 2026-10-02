package adminperfiles

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

const consultarCuenta = `SELECT sujeto_id,cuenta_id,cuenta_ordinaria_id,persona_ref,cuenta_ref,
 cuenta_ordinaria_ref,perfil_activo_ref,rol_id,vinculo_ref,vinculo_version::text,
 politica_garantia_ref,politica_garantia_huella_sha256,garantia_observada,vigente_hasta,seleccion_revision::text
 FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1($1,$2,$3,$4,$5,$6,$7)`
const vincularSesion = `SELECT vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

type transactor interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// PostgreSQL únicamente consume las fachadas nominales IS12/CA23. La
// composición entrega un LOGIN dedicado sin SELECT ni mutación de tablas.
type PostgreSQL struct {
	pool  transactor
	reloj h.Reloj
}

func NuevoPostgreSQL(ctx context.Context, pool *pgxpool.Pool, reloj h.Reloj) (*PostgreSQL, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil || nulo(reloj) {
		return nil, api.ErrConfiguracionIncompleta
	}
	var login string
	var acreditada bool
	if err := pool.QueryRow(ctx, `SELECT identidad_login,acreditada FROM vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1()`).Scan(&login, &acreditada); err != nil || !acreditada || login == "" {
		return nil, api.ErrConfiguracionIncompleta
	}
	return &PostgreSQL{pool: pool, reloj: reloj}, nil
}

func (p *PostgreSQL) ResolverCuentaADMIN(ctx context.Context, o ObservacionADMIN) (CuentaADMIN, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || nulo(p.pool) || !o.Valida(p.reloj.Ahora().UTC()) {
		return CuentaADMIN{}, api.ErrAutenticacionRequerida
	}
	var cuenta CuentaADMIN
	err := p.transaccion(ctx, func(tx pgx.Tx) error {
		var err error
		cuenta, err = p.leerCuenta(ctx, tx, o)
		return err
	})
	if err != nil {
		return CuentaADMIN{}, errorAutoridad(err)
	}
	return cuenta, nil
}

func (p *PostgreSQL) VincularSesionADMIN(ctx context.Context, o ObservacionADMIN, esperada CuentaADMIN, refs ReferenciasSesionADMIN) error {
	if p == nil || ctx == nil || ctx.Err() != nil || nulo(p.pool) || !o.Valida(p.reloj.Ahora().UTC()) ||
		!referencia(refs.AutenticacionRef, "aut_") || !referencia(refs.SesionRef, "ses_") {
		return api.ErrAutenticacionRequerida
	}
	return p.transaccion(ctx, func(tx pgx.Tx) error {
		actual, err := p.leerCuenta(ctx, tx, o)
		if err != nil {
			return err
		}
		if actual != esperada {
			return api.ErrAccesoDenegado
		}
		var ligada bool
		args := append(argumentos(o), refs.AutenticacionRef, refs.SesionRef)
		if err := tx.QueryRow(ctx, vincularSesion, args...).Scan(&ligada); err != nil {
			return err
		}
		if !ligada {
			return api.ErrAccesoDenegado
		}
		return nil
	})
}

func (p *PostgreSQL) leerCuenta(ctx context.Context, tx pgx.Tx, o ObservacionADMIN) (CuentaADMIN, error) {
	var c CuentaADMIN
	var version, seleccionRevision, garantia string
	err := tx.QueryRow(ctx, consultarCuenta, argumentosCuenta(o)...).Scan(
		&c.SujetoID, &c.CuentaID, &c.CuentaOrdinariaID, &c.PersonaRef, &c.CuentaRef,
		&c.CuentaOrdinariaRef, &c.PerfilActivoRef, &c.RolID, &c.VinculoRef, &version,
		&c.PoliticaGarantiaRef, &c.PoliticaGarantiaHuellaSHA256, &garantia, &c.VigenteHasta, &seleccionRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		return CuentaADMIN{}, api.ErrAccesoDenegado
	}
	if err != nil {
		return CuentaADMIN{}, err
	}
	c.VinculoVersion, err = strconv.ParseUint(version, 10, 64)
	if err == nil {
		c.SeleccionRevision, err = strconv.ParseUint(seleccionRevision, 10, 64)
	}
	c.GarantiaObservada = domain.AuthAssurance(garantia)
	c.VigenteHasta = c.VigenteHasta.UTC().Truncate(time.Microsecond)
	if err != nil || !c.Valida(p.reloj.Ahora().UTC().Truncate(time.Microsecond)) {
		return CuentaADMIN{}, api.ErrAccesoDenegado
	}
	return c, nil
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
	// La política compartida de reintento pertenece al consumidor completo.
	// Una carrera en esta lectura/binder obliga a obtener evidencia F nueva.
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(c)
	}()
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','4s',true),set_config('statement_timeout','8s',true)`); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ FuenteCuentasADMIN = (*PostgreSQL)(nil)
