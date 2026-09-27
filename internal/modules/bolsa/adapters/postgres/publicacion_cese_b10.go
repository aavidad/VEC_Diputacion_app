package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	publicacionb10 "vec-diputacion-granada/internal/modules/bolsa/publico/aplicacion"
)

var ErrFeedPublicacionCeseB10NoDisponible = errors.New("bolsa: feed de publicacion B10 no disponible")

// RepositorioPublicacionCeseB10PostgreSQL consume únicamente el feed B49 de
// Bolsa. La credencial del pool debe ser un LOGIN miembro exclusivo del rol
// vec_bolsa_llamamientos_publicador_cese; las funciones SQL lo revalidan.
type RepositorioPublicacionCeseB10PostgreSQL struct {
	pool *pgxpool.Pool
}

func NuevoRepositorioPublicacionCeseB10PostgreSQL(pool *pgxpool.Pool) (*RepositorioPublicacionCeseB10PostgreSQL, error) {
	if pool == nil {
		return nil, ErrFeedPublicacionCeseB10NoDisponible
	}
	return &RepositorioPublicacionCeseB10PostgreSQL{pool: pool}, nil
}

// PublicarSiguienteCeseB10 mantiene un cerrojo de sesión durante feed,
// publicación en la base separada y confirmación. Hijack retira la conexión
// del pool: cerrarla libera el cerrojo incluso si el proceso falla antes de
// confirmar el cursor B49. Un segundo trabajador no puede publicar un corte
// anterior mientras el primero avanza.
func (r *RepositorioPublicacionCeseB10PostgreSQL) PublicarSiguienteCeseB10(
	ctx context.Context,
	instantanea publicacionb10.FuenteInstantaneaCeseB10,
	publicar publicacionb10.DestinoPublicacionCeseB10,
) (publicacionb10.ReciboPublicacionCeseB10, bool, error) {
	if ctx == nil || r == nil || r.pool == nil || instantanea == nil || publicar == nil {
		return publicacionb10.ReciboPublicacionCeseB10{}, false, ErrFeedPublicacionCeseB10NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return publicacionb10.ReciboPublicacionCeseB10{}, false, err
	}
	adquirida, err := r.pool.Acquire(ctx)
	if err != nil {
		return publicacionb10.ReciboPublicacionCeseB10{}, false, ErrFeedPublicacionCeseB10NoDisponible
	}
	conexion := adquirida.Hijack()
	defer conexion.Close(context.Background())
	var libre bool
	if err := conexion.QueryRow(ctx, `SELECT pg_catalog.pg_try_advisory_lock(pg_catalog.hashtextextended('vec:bolsa:publicacion:cese:b10', 0))`).Scan(&libre); err != nil {
		return publicacionb10.ReciboPublicacionCeseB10{}, false, ErrFeedPublicacionCeseB10NoDisponible
	}
	if !libre {
		return publicacionb10.ReciboPublicacionCeseB10{}, false, nil
	}
	return publicacionb10.PublicarSiguienteCeseB10(ctx, feedCeseB10Conexion{conexion: conexion}, instantanea, publicar)
}

type feedCeseB10Conexion struct {
	conexion *pgx.Conn
}

var _ publicacionb10.FeedPublicacionCeseB10 = feedCeseB10Conexion{}

func (f feedCeseB10Conexion) SiguientePublicacionCeseB10(ctx context.Context) (publicacionb10.CesePendienteB10, bool, error) {
	if ctx == nil || f.conexion == nil {
		return publicacionb10.CesePendienteB10{}, false, ErrFeedPublicacionCeseB10NoDisponible
	}
	var evento publicacionb10.CesePendienteB10
	err := f.conexion.QueryRow(ctx, `
		SELECT origen_posicion, origen_ref, evento_ref, bolsa_ref, fase
		  FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1()`).Scan(
		&evento.OrigenPosicion, &evento.OrigenRef, &evento.EventoRef, &evento.BolsaRef, &evento.Fase,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return publicacionb10.CesePendienteB10{}, false, nil
	}
	if err != nil {
		return publicacionb10.CesePendienteB10{}, false, ErrFeedPublicacionCeseB10NoDisponible
	}
	return evento, true, nil
}

func (f feedCeseB10Conexion) ConfirmarPublicacionCeseB10(ctx context.Context, evento publicacionb10.CesePendienteB10, ancla string) (bool, error) {
	if ctx == nil || f.conexion == nil {
		return false, ErrFeedPublicacionCeseB10NoDisponible
	}
	var reutilizada bool
	err := f.conexion.QueryRow(ctx, `
		SELECT vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1($1::bigint,$2::text,$3::text,$4::text)`,
		evento.OrigenPosicion, evento.OrigenRef, evento.Fase, ancla,
	).Scan(&reutilizada)
	if err != nil {
		return false, ErrFeedPublicacionCeseB10NoDisponible
	}
	return reutilizada, nil
}
