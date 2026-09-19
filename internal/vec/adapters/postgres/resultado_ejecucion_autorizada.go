package postgres

import (
	"context"
	"errors"
	"log/slog"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/ports"
)

// RegistroResultadoEjecucion fija el único consumidor en construcción. El pool
// pertenece a un login auditor exclusivo, distinto del emisor y del ejecutor.
type RegistroResultadoEjecucion struct {
	pool   iniciadorTransacciones
	perfil ports.PerfilResultadoEjecucion
}

func NuevoRegistroResultadoRutasDietas(p *pgxpool.Pool) (*RegistroResultadoEjecucion, error) {
	return nuevoRegistroResultadoEjecucion(p, ports.ResultadoRutasDietas)
}
func NuevoRegistroResultadoBorradorDietas(p *pgxpool.Pool) (*RegistroResultadoEjecucion, error) {
	return nuevoRegistroResultadoEjecucion(p, ports.ResultadoBorradorDietas)
}
func NuevoRegistroResultadoMarcajeCronos(p *pgxpool.Pool) (*RegistroResultadoEjecucion, error) {
	return nuevoRegistroResultadoEjecucion(p, ports.ResultadoMarcajeCronos)
}
func nuevoRegistroResultadoEjecucion(p iniciadorTransacciones, perfil ports.PerfilResultadoEjecucion) (*RegistroResultadoEjecucion, error) {
	if valorNuloPostgreSQL(p) || consultaResultadoEjecucion(perfil) == "" {
		return nil, ports.ErrRegistroResultadoEjecucionNoDisponible
	}
	return &RegistroResultadoEjecucion{pool: p, perfil: perfil}, nil
}
func (*RegistroResultadoEjecucion) String() string         { return "[registro resultado ejecucion]" }
func (r *RegistroResultadoEjecucion) GoString() string     { return r.String() }
func (r *RegistroResultadoEjecucion) LogValue() slog.Value { return slog.StringValue(r.String()) }

func consultaResultadoEjecucion(p ports.PerfilResultadoEjecucion) string {
	// Nombres de funciones cerrados. Ninguna entrada de una petición forma SQL.
	const argumentos = `($1::text,$2::text,$3::text,$4::text,$5::text,$6::text,$7::text,$8::text,$9::text,$10::text,$11::text,$12::text,$13::text,$14::text)`
	switch p {
	case ports.ResultadoRutasDietas:
		return `SELECT informe_ref,recibo_ref,huella_sha256,observado_en,repeticion FROM vec_autorizacion_atestada_v3.registrar_resultado_rutas_dietas_v1` + argumentos
	case ports.ResultadoBorradorDietas:
		return `SELECT informe_ref,recibo_ref,huella_sha256,observado_en,repeticion FROM vec_autorizacion_atestada_v3.registrar_resultado_borrador_dietas_v1` + argumentos
	case ports.ResultadoMarcajeCronos:
		return `SELECT informe_ref,recibo_ref,huella_sha256,observado_en,repeticion FROM vec_autorizacion_atestada_v3.registrar_resultado_marcaje_cronos_v1` + argumentos
	}
	return ""
}

var reciboResultadoEjecucionValido = regexp.MustCompile(`^rec_ejec_[0-9a-f]{32}$`)
var huellaReciboResultadoEjecucionValida = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (r *RegistroResultadoEjecucion) RegistrarResultadoEjecucionAutorizada(ctx context.Context, informe ports.InformeResultadoEjecucionAutorizada) (ports.ReciboResultadoEjecucionAutorizada, error) {
	vacio := ports.ReciboResultadoEjecucionAutorizada{}
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil || ctx.Err() != nil {
		return vacio, ports.ErrRegistroResultadoEjecucionNoDisponible
	}
	d, err := informe.Datos()
	if err != nil || d.PerfilConsumidor != r.perfil || consultaResultadoEjecucion(r.perfil) == "" {
		return vacio, ports.ErrInformeResultadoEjecucionInvalido
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return vacio, ports.ErrRegistroResultadoEjecucionNoDisponible
	}
	defer revertirTransaccionPostgreSQL(tx)
	if _, err = tx.Exec(ctx, `SET LOCAL search_path = pg_catalog; SET LOCAL statement_timeout = '2500ms'; SET LOCAL lock_timeout = '2s'; SET LOCAL timezone = 'UTC'`); err != nil {
		return vacio, ports.ErrRegistroResultadoEjecucionNoDisponible
	}
	filas, err := tx.Query(ctx, consultaResultadoEjecucion(r.perfil), d.InformeRef, d.DecisionRef, d.DecisionHuellaSHA256, d.ContextoRef, d.ContextoHuellaSHA256, d.ActorRef, d.PerfilRef, d.CorrelacionRef, d.Accion, d.RecursoRef, d.RecursoHuellaSHA256, d.Resultado, d.Etapa, d.Causa)
	if err != nil {
		if !valorNuloPostgreSQL(filas) {
			filas.Close()
		}
		return vacio, errorResultadoEjecucion(err)
	}
	if valorNuloPostgreSQL(filas) {
		return vacio, ports.ErrRegistroResultadoEjecucionNoDisponible
	}
	var recibo ports.ReciboResultadoEjecucionAutorizada
	if !filas.Next() {
		err = filas.Err()
		filas.Close()
		return vacio, errorResultadoEjecucion(err)
	}
	err = filas.Scan(&recibo.InformeRef, &recibo.ReciboRef, &recibo.HuellaSHA256, &recibo.ObservadoEn, &recibo.Repeticion)
	if err != nil {
		filas.Close()
		return vacio, ports.ErrRegistroResultadoEjecucionNoDisponible
	}
	recibo.ObservadoEn = recibo.ObservadoEn.UTC()
	if filas.Next() {
		filas.Close()
		return vacio, ports.ErrRegistroResultadoEjecucionNoDisponible
	}
	err = filas.Err()
	filas.Close()
	if err != nil {
		return vacio, errorResultadoEjecucion(err)
	}
	if recibo.InformeRef != d.InformeRef || !reciboResultadoEjecucionValido.MatchString(recibo.ReciboRef) || !huellaReciboResultadoEjecucionValida.MatchString(recibo.HuellaSHA256) || recibo.ObservadoEn.IsZero() || recibo.ObservadoEn.Year() < 1 || recibo.ObservadoEn.Year() > 9999 || recibo.ObservadoEn.Nanosecond()%1000 != 0 || ctx.Err() != nil {
		return vacio, ports.ErrRegistroResultadoEjecucionNoDisponible
	}
	// No consultar ctx ni intentar deducir rollback después de un COMMIT ambiguo.
	if err = tx.Commit(ctx); err != nil {
		var pg *pgconn.PgError
		if errors.Is(err, pgx.ErrTxCommitRollback) || (errors.As(err, &pg) && pg.Severity == "ERROR") {
			return vacio, errorResultadoEjecucion(err)
		}
		return vacio, ports.ErrRegistroResultadoEjecucionIndeterminado
	}
	return recibo, nil
}
func errorResultadoEjecucion(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "P4201" {
		return ports.ErrInformeResultadoEjecucionEnConflicto
	}
	return ports.ErrRegistroResultadoEjecucionNoDisponible
}

var _ ports.RegistroResultadoEjecucionAutorizada = (*RegistroResultadoEjecucion)(nil)
