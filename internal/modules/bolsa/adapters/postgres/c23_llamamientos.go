package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type RepositorioLlamamientoOperativoPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioLlamamientoOperativo = (*RepositorioLlamamientoOperativoPostgreSQL)(nil)

func NuevoRepositorioLlamamientoOperativoPostgreSQL(pool *pgxpool.Pool) (*RepositorioLlamamientoOperativoPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrLlamamientoOperativoNoDisponible
	}
	return &RepositorioLlamamientoOperativoPostgreSQL{pool}, nil
}

func (r *RepositorioLlamamientoOperativoPostgreSQL) ContactosParticipacion(ctx context.Context, participacion string) ([]ports.ContactoLlamamientoOperativo, error) {
	if ctx == nil || r == nil || r.pool == nil || !ports.ReferenciaOpacaLlamamientoValida(participacion) {
		return nil, ports.ErrLlamamientoOperativoInvalido
	}
	rows, err := r.pool.Query(ctx, `SELECT candidato_ref, canal, disponible FROM vec_bolsa_llamamientos.contactos_participacion_v1($1::text)`, participacion)
	if err != nil {
		return nil, errorC23(ctx, err)
	}
	defer rows.Close()
	out := []ports.ContactoLlamamientoOperativo{}
	for rows.Next() {
		var c ports.ContactoLlamamientoOperativo
		if err := rows.Scan(&c.CandidatoRef, &c.Canal, &c.Disponible); err != nil {
			return nil, ports.ErrLlamamientoOperativoNoDisponible
		}
		out = append(out, c)
	}
	if rows.Err() != nil {
		return nil, ports.ErrLlamamientoOperativoNoDisponible
	}
	return out, nil
}
func (r *RepositorioLlamamientoOperativoPostgreSQL) AbrirLlamamiento(ctx context.Context, a ports.AperturaLlamamientoOperativo) (ports.ReciboLlamamientoOperativo, error) {
	if ctx == nil || r == nil || r.pool == nil || a.Validar() != nil {
		return ports.ReciboLlamamientoOperativo{}, ports.ErrLlamamientoOperativoInvalido
	}
	var b []byte
	err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.abrir_llamamiento_operativo_v1($1,$2,$3,$4,$5,$6,$7,$8)`, a.OperacionRef, a.LlamamientoRef, a.ParticipacionRef, a.ActorRef, a.Canal, a.ComunicadoEn, a.PlazoRespuestaHasta, a.Anotacion).Scan(&b)
	if err != nil {
		return ports.ReciboLlamamientoOperativo{}, errorC23(ctx, err)
	}
	return decodificarReciboC23(b)
}
func (r *RepositorioLlamamientoOperativoPostgreSQL) RegistrarResultadoLlamamiento(ctx context.Context, a ports.ResultadoLlamamientoOperativo) (ports.ReciboLlamamientoOperativo, error) {
	if ctx == nil || r == nil || r.pool == nil || a.Validar() != nil {
		return ports.ReciboLlamamientoOperativo{}, ports.ErrLlamamientoOperativoInvalido
	}
	var b []byte
	err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.registrar_resultado_llamamiento_operativo_v1($1,$2,$3,$4,$5)`, a.OperacionRef, a.LlamamientoRef, a.ActorRef, a.Resultado, a.RegistradoEn).Scan(&b)
	if err != nil {
		return ports.ReciboLlamamientoOperativo{}, errorC23(ctx, err)
	}
	return decodificarReciboC23(b)
}
func decodificarReciboC23(b []byte) (ports.ReciboLlamamientoOperativo, error) {
	var x struct {
		Reutilizado      bool      `json:"reutilizado"`
		LlamamientoRef   string    `json:"llamamiento_ref"`
		ParticipacionRef string    `json:"participacion_ref"`
		Estado           string    `json:"estado"`
		ReciboRef        string    `json:"recibo_ref"`
		ConfirmadoEn     time.Time `json:"confirmado_en"`
	}
	if err := jsonC23(b, &x); err != nil || !ports.ReferenciaOpacaLlamamientoValida(x.LlamamientoRef) || !ports.ReferenciaOpacaLlamamientoValida(x.ParticipacionRef) || !ports.ReferenciaOpacaLlamamientoValida(x.ReciboRef) || x.ConfirmadoEn.IsZero() {
		return ports.ReciboLlamamientoOperativo{}, ports.ErrLlamamientoOperativoNoDisponible
	}
	return ports.ReciboLlamamientoOperativo{Reutilizado: x.Reutilizado, LlamamientoRef: x.LlamamientoRef, ParticipacionRef: x.ParticipacionRef, Estado: x.Estado, ReciboRef: x.ReciboRef, ConfirmadoEn: x.ConfirmadoEn.UTC()}, nil
}
func jsonC23(b []byte, v any) error { return json.Unmarshal(b, v) }
func errorC23(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "22023", "23502", "23503", "23514":
			return ports.ErrLlamamientoOperativoInvalido
		case "23505", "PBL23":
			return ports.ErrLlamamientoOperativoConflicto
		}
	}
	return ports.ErrLlamamientoOperativoNoDisponible
}
