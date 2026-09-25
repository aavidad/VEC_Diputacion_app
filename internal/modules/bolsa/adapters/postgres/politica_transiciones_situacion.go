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
	_ ports.ConsultaPoliticaTransicionesSituacion   = (*RepositorioSituacionParticipacionPostgreSQL)(nil)
	_ ports.PublicadorPoliticaTransicionesSituacion = (*RepositorioSituacionParticipacionPostgreSQL)(nil)

	patronSHA256PoliticaTransiciones = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// funcionInexistente: la base no tiene la migración 000032.
func funcionInexistente(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42883"
}

// PoliticaTransicionesSituacion lee la política que la base aplicará al
// próximo cambio. Sin la migración 000032 la base conserva el literal de
// 000012, que es la tabla compilada; cualquier otro fallo se propaga.
func (r *RepositorioSituacionParticipacionPostgreSQL) PoliticaTransicionesSituacion(ctx context.Context) (ports.PoliticaTransicionesVigente, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ports.PoliticaTransicionesVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	var vigente ports.PoliticaTransicionesVigente
	var pares []string
	err := r.pool.QueryRow(ctx, `SELECT version,catalogo_ref,transiciones FROM vec_bolsa_llamamientos.consultar_politica_transiciones_situacion_v1()`).Scan(&vigente.Version, &vigente.CatalogoRef, &pares)
	if err != nil {
		if funcionInexistente(err) {
			return ports.PoliticaTransicionesVigente{Politica: dominiobolsa.PoliticaTransicionesSituacionCompilada()}, nil
		}
		return ports.PoliticaTransicionesVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	if vigente.Politica, err = dominiobolsa.PoliticaTransicionesDesdePares(pares); err != nil || vigente.Version < 1 {
		return ports.PoliticaTransicionesVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return vigente, nil
}

// PublicarPoliticaTransicionesSituacion registra la política del catálogo
// como nueva versión si difiere de la vigente.
func (r *RepositorioSituacionParticipacionPostgreSQL) PublicarPoliticaTransicionesSituacion(ctx context.Context, p ports.PublicacionPoliticaTransicionesSituacion) (ports.PoliticaTransicionesVigente, error) {
	if r == nil || r.pool == nil || ctx == nil || p.CatalogoRef == "" || len(p.CatalogoRef) > 512 ||
		!patronSHA256PoliticaTransiciones.MatchString(p.CatalogoSHA256) {
		return ports.PoliticaTransicionesVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	var vigente ports.PoliticaTransicionesVigente
	var reutilizada bool
	var pares []string
	err := r.pool.QueryRow(ctx, `SELECT version,reutilizada,catalogo_ref,transiciones FROM vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1($1,$2,$3)`,
		p.CatalogoRef, p.CatalogoSHA256, p.Politica.Pares()).Scan(&vigente.Version, &reutilizada, &vigente.CatalogoRef, &pares)
	if err != nil {
		if funcionInexistente(err) {
			return ports.PoliticaTransicionesVigente{}, ports.ErrPoliticaTransicionesNoInstalada
		}
		return ports.PoliticaTransicionesVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	if vigente.Politica, err = dominiobolsa.PoliticaTransicionesDesdePares(pares); err != nil || vigente.Version < 1 ||
		vigente.CatalogoRef != p.CatalogoRef || !slices.Equal(pares, p.Politica.Pares()) {
		return ports.PoliticaTransicionesVigente{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return vigente, nil
}
