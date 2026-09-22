package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type RepositorioEmisionLlamamientoPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoRepositorioEmisionLlamamientoPostgreSQL(pool *pgxpool.Pool) (*RepositorioEmisionLlamamientoPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	return &RepositorioEmisionLlamamientoPostgreSQL{pool}, nil
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) Reservar(ctx context.Context, c ports.ComandoEmitirLlamamiento) (ports.EmisionLlamamiento, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	participaciones, _ := json.Marshal(c.Participaciones)
	configuracion, _ := json.Marshal(c.Configuracion)
	m := c.Material
	huellaFinalizacion := sha256.Sum256(c.TokenFinalizacion)
	if len(c.TokenFinalizacion) != 32 {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	var salidaJSON []byte
	var reutilizada bool
	err := r.pool.QueryRow(ctx, `SELECT emision,reutilizada FROM vec_bolsa_llamamientos.reservar_llamamiento_v1($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12,$13,$14::numeric,$15::numeric,$16,$17,$18,$19)`, c.LlamamientoRef, c.ReciboRef, c.BolsaRef, c.ActorRef, c.ClaveIdempotencia, participaciones, configuracion, c.EmitidoEn, huellaFinalizacion[:], m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&salidaJSON, &reutilizada)
	if err != nil {
		return ports.EmisionLlamamiento{}, errorEmision(err)
	}
	var out ports.EmisionLlamamiento
	if json.Unmarshal(salidaJSON, &out) != nil {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	out.Reutilizada = reutilizada
	return out, nil
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) RegistrarContactos(ctx context.Context, bolsa, clave, actor string, token []byte, contactos []ports.ResultadoContactoEmision) (ports.EmisionLlamamiento, error) {
	if r == nil || r.pool == nil || ctx == nil || bolsa == "" || clave == "" || actor == "" || len(token) != 32 || len(contactos) == 0 {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	rawContactos, _ := json.Marshal(contactos)
	var salidaJSON []byte
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1($1,$2,$3,$4,$5::jsonb)`, bolsa, clave, actor, token, rawContactos).Scan(&salidaJSON); err != nil {
		return ports.EmisionLlamamiento{}, errorEmision(err)
	}
	var out ports.EmisionLlamamiento
	if json.Unmarshal(salidaJSON, &out) != nil {
		return out, ports.ErrEmisionLlamamientoNoDisponible
	}
	return out, nil
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) Recuperar(ctx context.Context, bolsa, clave string) (ports.EmisionLlamamiento, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	var raw []byte
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1($1,$2)`, bolsa, clave).Scan(&raw); err != nil {
		return ports.EmisionLlamamiento{}, errorEmision(err)
	}
	var out ports.EmisionLlamamiento
	if json.Unmarshal(raw, &out) != nil {
		return out, ports.ErrEmisionLlamamientoNoDisponible
	}
	out.Reutilizada = true
	return out, nil
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) ContarEnCurso(ctx context.Context, bolsa string) (int, error) {
	if r == nil || r.pool == nil || ctx == nil || bolsa == "" {
		return 0, ports.ErrEmisionLlamamientoNoDisponible
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1($1)`, bolsa).Scan(&total); err != nil {
		return 0, errorEmision(err)
	}
	return total, nil
}

func errorEmision(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "VBE01":
			return ports.ErrEmisionLlamamientoConflicto
		case "22023", "23503":
			return ports.ErrEmisionLlamamientoInvalida
		}
	}
	return ports.ErrEmisionLlamamientoNoDisponible
}
