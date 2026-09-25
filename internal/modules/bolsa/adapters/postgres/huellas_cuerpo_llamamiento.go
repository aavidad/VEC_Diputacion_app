package postgres

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// RegistrarHuellasCuerpo fija las huellas del asunto y del cuerpo que recibirá
// cada destinatario, ligadas a la reserva por el token de finalización
// (migración 000025). Repetir las mismas huellas es idempotente; otras distintas
// para la misma reserva se rechazan como conflicto.
func (r *RepositorioEmisionLlamamientoPostgreSQL) RegistrarHuellasCuerpo(ctx context.Context, bolsa, clave, actor string, token []byte, huellas []ports.HuellaCuerpoContacto) error {
	if r == nil || r.pool == nil || ctx == nil || bolsa == "" || clave == "" || actor == "" || len(token) != 32 || len(huellas) == 0 || len(huellas) > 100 {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	raw, err := json.Marshal(huellas)
	if err != nil {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	if _, err := r.pool.Exec(ctx, `SELECT vec_bolsa_llamamientos.registrar_cuerpos_llamamiento_v1($1,$2,$3,$4,$5::jsonb)`, bolsa, clave, actor, token, raw); err != nil {
		return errorEmision(err)
	}
	return nil
}
