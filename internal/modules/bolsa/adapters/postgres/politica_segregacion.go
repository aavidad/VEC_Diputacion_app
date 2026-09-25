package postgres

import (
	"context"
	"errors"
	"regexp"
	"slices"

	"github.com/jackc/pgx/v5/pgconn"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var (
	_ ports.ConsultaPoliticaSegregacion   = (*RepositorioSituacionParticipacionPostgreSQL)(nil)
	_ ports.PublicadorPoliticaSegregacion = (*RepositorioSituacionParticipacionPostgreSQL)(nil)

	patronSHA256PoliticaSegregacion = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// PoliticaSegregacion lee la política que la base aplicará a la próxima
// operación. Sin la migración 000033 la base conserva el literal de 000019,
// que coincide con la política mínima; cualquier otro fallo se propaga.
func (r *RepositorioSituacionParticipacionPostgreSQL) PoliticaSegregacion(ctx context.Context) (ports.PoliticaSegregacionVigente, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ports.PoliticaSegregacionVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	var vigente ports.PoliticaSegregacionVigente
	var operaciones []string
	err := r.pool.QueryRow(ctx, `SELECT version,catalogo_ref,operaciones FROM vec_bolsa_llamamientos.consultar_politica_segregacion_v1()`).Scan(&vigente.Version, &vigente.CatalogoRef, &operaciones)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42883" {
			return ports.PoliticaSegregacionVigente{Politica: dominiobolsa.PoliticaSegregacionMinima()}, nil
		}
		return ports.PoliticaSegregacionVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	if vigente.Politica, err = dominiobolsa.NuevaPoliticaSegregacion(operaciones); err != nil || vigente.Version < 1 {
		return ports.PoliticaSegregacionVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return vigente, nil
}

// PublicarPoliticaSegregacion registra la entrada del catálogo como nueva
// versión si difiere de la vigente. La base vuelve a exigir la exclusión.
func (r *RepositorioSituacionParticipacionPostgreSQL) PublicarPoliticaSegregacion(ctx context.Context, p ports.PublicacionPoliticaSegregacion) (ports.PoliticaSegregacionVigente, error) {
	if r == nil || r.pool == nil || ctx == nil || p.CatalogoRef == "" || len(p.CatalogoRef) > 512 ||
		!patronSHA256PoliticaSegregacion.MatchString(p.CatalogoSHA256) {
		return ports.PoliticaSegregacionVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	var vigente ports.PoliticaSegregacionVigente
	var reutilizada bool
	var operaciones []string
	err := r.pool.QueryRow(ctx, `SELECT version,reutilizada,catalogo_ref,operaciones FROM vec_bolsa_llamamientos.publicar_politica_segregacion_v1($1,$2,$3)`,
		p.CatalogoRef, p.CatalogoSHA256, p.Politica.Operaciones()).Scan(&vigente.Version, &reutilizada, &vigente.CatalogoRef, &operaciones)
	if err != nil {
		return ports.PoliticaSegregacionVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	if vigente.Politica, err = dominiobolsa.NuevaPoliticaSegregacion(operaciones); err != nil ||
		vigente.CatalogoRef != p.CatalogoRef || !slices.Equal(operaciones, p.Politica.Operaciones()) {
		return ports.PoliticaSegregacionVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return vigente, nil
}
