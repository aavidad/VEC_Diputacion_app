package ports

import "context"

// LectorPublicacionCesesBolsa entrega únicamente ceses CT115 posteriores al
// cursor. Usa el mismo evento y triple de verificación que CT113; Bolsa
// conserva un cursor propio sin avanzar por incorporaciones ajenas.
type LectorPublicacionCesesBolsa interface {
	LeerCesesBolsa(ctx context.Context, desde CursorPublicacionContratosBolsa, limite int) ([]EventoContratoBolsaPublicado, error)
}
