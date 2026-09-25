package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// RepositorioOfertasPublicadasPostgreSQL invoca las funciones de la migración
// 000028. La autorización se consume dentro de la misma transacción que
// escribe la oferta o su resolución.
type RepositorioOfertasPublicadasPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoRepositorioOfertasPublicadasPostgreSQL(pool *pgxpool.Pool) (*RepositorioOfertasPublicadasPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrOfertaNoDisponible
	}
	return &RepositorioOfertasPublicadasPostgreSQL{pool}, nil
}

func (r *RepositorioOfertasPublicadasPostgreSQL) Publicar(ctx context.Context, c ports.ComandoPublicarOferta) (ports.OfertaPublicada, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	datos, errDatos := json.Marshal(c.Datos)
	plazo, errPlazo := json.Marshal(c.Plazo)
	if errDatos != nil || errPlazo != nil {
		return ports.OfertaPublicada{}, ports.ErrOfertaInvalida
	}
	m := c.Material
	var salida []byte
	var reutilizada bool
	err := r.pool.QueryRow(ctx, `SELECT oferta,reutilizada FROM vec_bolsa_llamamientos.publicar_oferta_v1($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12,$13,$14::numeric,$15::numeric,$16,$17,$18,$19)`,
		c.OfertaRef, c.ReciboRef, c.BolsaRef, c.ActorRef, c.ClaveIdempotencia, datos, plazo, c.PublicadaEn, c.VenceAntesDe,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&salida, &reutilizada)
	if err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	return decodificarOferta(salida, reutilizada)
}

func (r *RepositorioOfertasPublicadasPostgreSQL) Resolver(ctx context.Context, c ports.ComandoResolverOferta) (ports.OfertaPublicada, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	var participacion any
	if c.ParticipacionRef != "" {
		participacion = c.ParticipacionRef
	}
	m := c.Material
	var salida []byte
	var reutilizada bool
	err := r.pool.QueryRow(ctx, `SELECT oferta,reutilizada FROM vec_bolsa_llamamientos.resolver_oferta_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::numeric,$12::numeric,$13,$14,$15,$16)`,
		c.OfertaRef, c.ReciboRef, c.BolsaRef, participacion, c.ActorRef, c.ClaveIdempotencia,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&salida, &reutilizada)
	if err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	return decodificarOferta(salida, reutilizada)
}

func (r *RepositorioOfertasPublicadasPostgreSQL) Listar(ctx context.Context, bolsa string, corte time.Time, limite int) ([]ports.OfertaPublicada, error) {
	if r == nil || r.pool == nil || ctx == nil || bolsa == "" || corte.IsZero() || limite < 1 || limite > 100 {
		return nil, ports.ErrOfertaNoDisponible
	}
	var salida []byte
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.listar_ofertas_bolsa_v1($1,$2,$3)`, bolsa, corte, limite).Scan(&salida); err != nil {
		return nil, errorOferta(err)
	}
	return decodificarListaOfertas(salida)
}

func decodificarOferta(salida []byte, reutilizada bool) (ports.OfertaPublicada, error) {
	var oferta ports.OfertaPublicada
	if json.Unmarshal(salida, &oferta) != nil || oferta.OfertaRef == "" || oferta.Estado == "" {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	oferta.Reutilizada = reutilizada
	return oferta, nil
}

func decodificarListaOfertas(salida []byte) ([]ports.OfertaPublicada, error) {
	var ofertas []ports.OfertaPublicada
	if json.Unmarshal(salida, &ofertas) != nil {
		return nil, ports.ErrOfertaNoDisponible
	}
	for _, oferta := range ofertas {
		if oferta.OfertaRef == "" || oferta.Estado == "" {
			return nil, ports.ErrOfertaNoDisponible
		}
	}
	if ofertas == nil {
		ofertas = []ports.OfertaPublicada{}
	}
	return ofertas, nil
}

// errorOferta traduce los códigos de la migración 000028 sin exponer su texto.
func errorOferta(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "VBO01":
			return ports.ErrOfertaConflicto
		case "VBO02":
			return ports.ErrOfertaYaResuelta
		case "VBO03":
			return ports.ErrOfertaPlazoAbierto
		case "VBO04":
			return ports.ErrOfertaPropuestaCambiada
		case "22023", "23503":
			return ports.ErrOfertaInvalida
		}
	}
	return ports.ErrOfertaNoDisponible
}
