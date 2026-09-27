package postgres

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var _ ports.LectorPublicacionCesesBolsa = (*LectorPublicacionContratosBolsaPostgreSQL)(nil)

// LeerCesesBolsa usa la fachada CT129 que pagina solo ceses. Reutiliza la
// transacción de lectura y la comprobación de huella del adaptador CT113.
func (l *LectorPublicacionContratosBolsaPostgreSQL) LeerCesesBolsa(
	ctx context.Context, desde ports.CursorPublicacionContratosBolsa, limite int,
) ([]ports.EventoContratoBolsaPublicado, error) {
	return l.leer(ctx, `SELECT evento_ref, evento::text, huella_sha256, origen_ref, origen_posicion, origen_creada_en FROM vec_contratacion_temporal.leer_ceses_bolsa_v1($1,$2,$3)`, desde, limite)
}
