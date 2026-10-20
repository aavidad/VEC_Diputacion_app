package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// AmbitoBolsa lee el ámbito gobernado de la bolsa con la lectura B145, la única
// vía del ejecutor sobre ambito_bolsa. Una bolsa sin fila devuelve
// ErrAmbitoBolsaNoDeclarado; cualquier otro fallo, no disponible.
func (r *RepositorioConstitucionPostgreSQL) AmbitoBolsa(ctx context.Context, bolsaRef string) (ports.AmbitoBolsa, error) {
	if ctx == nil || r == nil || r.pool == nil || bolsaRef == "" || len(bolsaRef) > 512 {
		return ports.AmbitoBolsa{}, ports.ErrConstitucionBolsaNoDisponible
	}
	var a ports.AmbitoBolsa
	err := r.pool.QueryRow(ctx, `SELECT ambito_ref, unidad_ref
		FROM vec_bolsa_llamamientos.leer_ambito_bolsa_v1($1::text)`, bolsaRef).Scan(&a.AmbitoRef, &a.UnidadRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.AmbitoBolsa{}, ports.ErrAmbitoBolsaNoDeclarado
	}
	if err != nil {
		return ports.AmbitoBolsa{}, errorConstitucion(ctx, err)
	}
	if a.AmbitoRef == "" || a.UnidadRef == "" {
		return ports.AmbitoBolsa{}, ports.ErrConstitucionBolsaNoDisponible
	}
	return a, nil
}
