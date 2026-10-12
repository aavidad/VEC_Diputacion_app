package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const esquemaRespuestasPortalRRHH = "vec.bolsa.rrhh.respuestas-portal.v1"

var _ ports.LectorRespuestasPortalRRHH = (*RepositorioRespuestasPortalRRHHPostgreSQL)(nil)

// RepositorioRespuestasPortalRRHHPostgreSQL solo invoca la función B103. El
// propietario de Bolsa une proyección y asiento central AD237 en esta TX.
type RepositorioRespuestasPortalRRHHPostgreSQL struct {
	pool *pgxpool.Pool
}

func NuevoRepositorioRespuestasPortalRRHHPostgreSQL(pool *pgxpool.Pool) (*RepositorioRespuestasPortalRRHHPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	return &RepositorioRespuestasPortalRRHHPostgreSQL{pool: pool}, nil
}

type sobreRespuestasPortalRRHH struct {
	Esquema        string                      `json:"esquema"`
	LlamamientoRef string                      `json:"llamamiento_ref"`
	Respuestas     []ports.RespuestaPortalRRHH `json:"respuestas"`
	AuditoriaRef   string                      `json:"auditoria_ref"`
	CorrelacionRef string                      `json:"correlacion_ref"`
	ConsultadaEn   time.Time                   `json:"consultada_en"`
}

func (r *RepositorioRespuestasPortalRRHHPostgreSQL) ConsultarRespuestasPortalLlamamiento(
	ctx context.Context, q ports.ConsultaRespuestasPortalRRHH,
) (ports.RespuestasPortalLlamamientoRRHH, error) {
	vacia := ports.RespuestasPortalLlamamientoRRHH{}
	if r == nil || r.pool == nil || ctx == nil || ctx.Err() != nil ||
		q.BolsaRef == "" || q.LlamamientoRef == "" || q.ResultadoContexto.Validar() != nil ||
		q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.Captura.BolsaRef != q.BolsaRef || q.Captura.LlamamientoRef != q.LlamamientoRef ||
		len(q.Captura.ContextoCanonico) == 0 || len(q.Captura.VinculoCanonico) == 0 ||
		len(q.Captura.Orden) == 0 || !q.Captura.ValidaHasta.After(time.Now().UTC()) {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	var orden struct {
		CorrelacionRef string `json:"correlacion_ref"`
	}
	if json.Unmarshal(q.Captura.Orden, &orden) != nil || orden.CorrelacionRef == "" {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true)`); err != nil {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.consultar_respuestas_portal_seguimiento_rrhh_v1($1,$2,$3,$4,$5::jsonb)`,
		q.BolsaRef, q.LlamamientoRef, q.Captura.ContextoCanonico, q.Captura.VinculoCanonico,
		q.Captura.Orden).Scan(&contenido)
	if err != nil || len(contenido) == 0 || len(contenido) > 32*1024 {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	var sobre sobreRespuestasPortalRRHH
	if decodificarInscripcionEstricta(contenido, &sobre) != nil || sobre.Esquema != esquemaRespuestasPortalRRHH ||
		sobre.LlamamientoRef != q.LlamamientoRef || sobre.AuditoriaRef == "" ||
		sobre.CorrelacionRef != orden.CorrelacionRef || sobre.ConsultadaEn.IsZero() ||
		sobre.Respuestas == nil || len(sobre.Respuestas) > 100 || ctx.Err() != nil ||
		!q.Captura.ValidaHasta.After(time.Now().UTC()) {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	vistas := make(map[string]bool, len(sobre.Respuestas))
	for _, respuesta := range sobre.Respuestas {
		if respuesta.ParticipacionRef == "" || vistas[respuesta.ParticipacionRef] ||
			(respuesta.Respuesta != "acepta" && respuesta.Respuesta != "renuncia" && respuesta.Respuesta != "renuncia_justificada") {
			return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
		}
		vistas[respuesta.ParticipacionRef] = true
	}
	if tx.Commit(ctx) != nil {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	return ports.RespuestasPortalLlamamientoRRHH{LlamamientoRef: sobre.LlamamientoRef, Respuestas: sobre.Respuestas}, nil
}
