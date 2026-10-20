package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const esquemaExclusionesOfrecimientoRRHH = "vec.bolsa.rrhh.exclusiones-ofrecimiento.v1"

var _ ports.LectorExclusionesOfrecimientoRRHH = (*RepositorioExclusionesOfrecimientoRRHHPostgreSQL)(nil)

type RepositorioExclusionesOfrecimientoRRHHPostgreSQL struct {
	pool *pgxpool.Pool
}

func NuevoRepositorioExclusionesOfrecimientoRRHHPostgreSQL(pool *pgxpool.Pool) (*RepositorioExclusionesOfrecimientoRRHHPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	return &RepositorioExclusionesOfrecimientoRRHHPostgreSQL{pool: pool}, nil
}

type sobreExclusionesOfrecimientoRRHH struct {
	Esquema string `json:"esquema"`
	ports.PaginaExclusionesOfrecimientoRRHH
}

// La función SQL acota el resultado al llamamiento histórico exacto y registra
// la lectura nominal en la misma transacción SERIALIZABLE.
func (r *RepositorioExclusionesOfrecimientoRRHHPostgreSQL) ConsultarExclusionesOfrecimientoRRHH(
	ctx context.Context, q ports.ConsultaExclusionesOfrecimientoRRHH,
) (ports.PaginaExclusionesOfrecimientoRRHH, error) {
	vacia := ports.PaginaExclusionesOfrecimientoRRHH{}
	if r == nil || r.pool == nil || ctx == nil || ctx.Err() != nil ||
		q.BolsaRef == "" || q.LlamamientoRef == "" || q.DesdeOrdenActa < 0 ||
		q.Limite < 1 || q.Limite > ports.LimitePaginaExclusionesOfrecimientoRRHH ||
		q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.Captura.BolsaRef != q.BolsaRef || q.Captura.LlamamientoRef != q.LlamamientoRef ||
		len(q.Captura.ContextoCanonico) == 0 || len(q.Captura.VinculoCanonico) == 0 ||
		len(q.Captura.Orden) == 0 || !q.Captura.ValidaHasta.After(time.Now().UTC()) {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	var orden struct {
		CorrelacionRef string `json:"correlacion_ref"`
	}
	if json.Unmarshal(q.Captura.Orden, &orden) != nil || orden.CorrelacionRef == "" {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true)`); err != nil {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.consultar_exclusiones_ofrecimiento_rrhh_v1($1,$2,$3,$4,$5,$6,$7::jsonb)`,
		q.BolsaRef, q.LlamamientoRef, q.DesdeOrdenActa, q.Limite,
		q.Captura.ContextoCanonico, q.Captura.VinculoCanonico, q.Captura.Orden).Scan(&contenido)
	if err != nil || len(contenido) == 0 || len(contenido) > 1024*1024 {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	var sobre sobreExclusionesOfrecimientoRRHH
	if decodificarInscripcionEstricta(contenido, &sobre) != nil || sobre.Esquema != esquemaExclusionesOfrecimientoRRHH ||
		sobre.CorrelacionRef != orden.CorrelacionRef ||
		!sobre.PaginaExclusionesOfrecimientoRRHH.ValidaPara(q) ||
		ctx.Err() != nil || !q.Captura.ValidaHasta.After(time.Now().UTC()) {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	if tx.Commit(ctx) != nil {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	return sobre.PaginaExclusionesOfrecimientoRRHH, nil
}
