package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
	"time"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaAutenticacionOriginalV1 = `SELECT autenticacion_ref,autenticacion_huella_sha256,asercion_ref,sesion_ref,control_sesion_ref,control_sesion_revision,control_sesion_huella_sha256,cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,superficie,metodo_observado,garantia_observada,politica_garantia_ref,politica_garantia_huella_sha256,autenticacion_verificada_en,sesion_emitida_en,sesion_valida_hasta,sesion_revalidada_en FROM vec_identidad_sesiones_v1.leer_autenticacion_original_v1($1,$2,$3)`
const ajustesAutenticacionOriginalV1 = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

type RelojLecturaAutenticacionOriginalV1 interface{ Ahora() time.Time }
type LectorAutenticacionOriginalPostgreSQLV1 struct {
	pool  iniciadorTransacciones
	reloj RelojLecturaAutenticacionOriginalV1
}

var _ ports.LectorAutenticacionOriginalV1 = (*LectorAutenticacionOriginalPostgreSQLV1)(nil)

// Pool histórico separado de la revalidación viva. Su LOGIN/ACL y codec
// Timestamptz ScanLocation=time.UTC se gobiernan en composición, no aquí.
func NuevoLectorAutenticacionOriginalPostgreSQLV1(p *pgxpool.Pool, r RelojLecturaAutenticacionOriginalV1) (*LectorAutenticacionOriginalPostgreSQLV1, error) {
	return nuevoLectorAutenticacionOriginalPostgreSQLV1(p, r)
}
func nuevoLectorAutenticacionOriginalPostgreSQLV1(p iniciadorTransacciones, r RelojLecturaAutenticacionOriginalV1) (*LectorAutenticacionOriginalPostgreSQLV1, error) {
	if valorNulo(p) || valorNulo(r) {
		return nil, ports.ErrLecturaAutenticacionOriginalV1
	}
	return &LectorAutenticacionOriginalPostgreSQLV1{p, r}, nil
}
func instanteAutenticacionOriginal(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Year() >= 1 && t.Year() <= 9999 && t.Nanosecond()%1000 == 0
}
func errorAutenticacionOriginal(ctx context.Context) error {
	if !valorNulo(ctx) && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrLecturaAutenticacionOriginalV1
}

func (l *LectorAutenticacionOriginalPostgreSQLV1) LeerAutenticacionOriginalV1(ctx context.Context, s ports.SolicitudLecturaAutenticacionOriginalV1) (domain.AutenticacionRevalidadaV1, error) {
	var cero domain.AutenticacionRevalidadaV1
	if valorNulo(ctx) || l == nil || valorNulo(l.pool) || valorNulo(l.reloj) || s.Validar() != nil {
		return cero, errorAutenticacionOriginal(ctx)
	}
	var ultimo time.Time
	validar := func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		t := l.reloj.Ahora()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !instanteAutenticacionOriginal(t) || t.Before(ultimo) {
			return ports.ErrLecturaAutenticacionOriginalV1
		}
		ultimo = t
		return ctx.Err()
	}
	if e := validar(); e != nil {
		return cero, e
	}
	tx, e := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	confirmado := false
	if !valorNulo(tx) {
		defer func() {
			if !confirmado {
				c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = tx.Rollback(c)
			}
		}()
	}
	if e != nil || valorNulo(tx) {
		return cero, errorAutenticacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	if _, e = tx.Exec(ctx, ajustesAutenticacionOriginalV1); e != nil {
		return cero, errorAutenticacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	rows, e := tx.Query(ctx, consultaAutenticacionOriginalV1, s.AutenticacionRef, s.SesionRef, s.HuellaSHA256)
	if !valorNulo(rows) {
		defer rows.Close()
	}
	if e != nil || valorNulo(rows) {
		return cero, errorAutenticacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	if !rows.Next() {
		if rows.Err() != nil {
			return cero, errorAutenticacionOriginal(ctx)
		}
		return cero, errorAutenticacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	var r domain.AutenticacionRevalidadaV1
	var revision, sup, metodo, garantia string
	e = rows.Scan(&r.AutenticacionRef, &r.AutenticacionHuellaSHA256, &r.AsercionRef, &r.SesionRef, &r.ControlSesionRef, &revision, &r.ControlSesionHuellaSHA256, &r.CuentaRef, &r.CuentaOrdinariaRef, &r.CuentaPrivilegiada, &sup, &metodo, &garantia, &r.PoliticaGarantiaRef, &r.PoliticaGarantiaHuellaSHA256, &r.AutenticacionVerificadaEn, &r.SesionEmitidaEn, &r.SesionValidaHasta, &r.SesionRevalidadaEn)
	if e != nil {
		return cero, errorAutenticacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	// Cotas previas al parseo; no se normalizan ni reparan columnas recibidas.
	if len(revision) < 1 || len(revision) > 20 || revision[0] < '1' || revision[0] > '9' {
		return cero, errorAutenticacionOriginal(ctx)
	}
	for _, c := range revision {
		if c < '0' || c > '9' {
			return cero, errorAutenticacionOriginal(ctx)
		}
	}
	for _, v := range []string{r.AutenticacionRef, r.SesionRef, r.AsercionRef, r.ControlSesionRef, r.CuentaRef, r.CuentaOrdinariaRef, r.PoliticaGarantiaRef, r.AutenticacionHuellaSHA256, r.ControlSesionHuellaSHA256, r.PoliticaGarantiaHuellaSHA256, sup, metodo, garantia} {
		if len(v) == 0 || len(v) > 256 {
			return cero, errorAutenticacionOriginal(ctx)
		}
	}
	r.ControlSesionRevision, e = strconv.ParseUint(revision, 10, 64)
	if e != nil {
		return cero, errorAutenticacionOriginal(ctx)
	}
	r.Superficie = domain.SuperficieAutenticacionActorV1(sup)
	r.MetodoObservado = domain.AuthMethod(metodo)
	r.GarantiaObservada = domain.AuthAssurance(garantia)
	for _, t := range []time.Time{r.AutenticacionVerificadaEn, r.SesionEmitidaEn, r.SesionValidaHasta, r.SesionRevalidadaEn} {
		if !instanteAutenticacionOriginal(t) {
			return cero, errorAutenticacionOriginal(ctx)
		}
	}
	if s.ValidarResultado(r) != nil || r.SesionRevalidadaEn.After(ultimo) {
		return cero, errorAutenticacionOriginal(ctx)
	}
	if rows.Next() || rows.Err() != nil {
		return cero, errorAutenticacionOriginal(ctx)
	}
	rows.Close()
	if e = validar(); e != nil {
		return cero, e
	}
	if e = tx.Commit(ctx); e != nil {
		return cero, errorAutenticacionOriginal(ctx)
	}
	confirmado = true
	if e = validar(); e != nil {
		return cero, e
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	// Todos los campos son valores (strings, bool, uint64, time.Time), sin slices.
	return r, nil
}
