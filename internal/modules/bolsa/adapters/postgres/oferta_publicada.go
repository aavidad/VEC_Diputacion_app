package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// RepositorioOfertasPublicadasPostgreSQL invoca B58 para publicar con número de
// plazas y registrar actos por plaza, y B28/B58 para consultar. La
// autorización se consume dentro de la transacción que escribe la oferta o el
// acto, junto con su auditoría y outbox.
type RepositorioOfertasPublicadasPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoRepositorioOfertasPublicadasPostgreSQL(pool *pgxpool.Pool) (*RepositorioOfertasPublicadasPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrOfertaNoDisponible
	}
	return &RepositorioOfertasPublicadasPostgreSQL{pool}, nil
}

func (r *RepositorioOfertasPublicadasPostgreSQL) Publicar(ctx context.Context, c ports.ComandoPublicarOferta) (ports.OfertaPublicada, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil || c.NumeroPlazas < 1 {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	datos, errDatos := json.Marshal(c.Datos)
	plazo, errPlazo := serializarPlazoOfertaPostgreSQL(c.Plazo)
	if errDatos != nil || errPlazo != nil {
		return ports.OfertaPublicada{}, ports.ErrOfertaInvalida
	}
	m := c.Material
	var salida []byte
	var reutilizada bool
	err := r.pool.QueryRow(ctx, `SELECT oferta,reutilizada FROM vec_bolsa_llamamientos.publicar_oferta_v4($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12,$13,$14::numeric,$15::numeric,$16,$17,$18,$19,$20,$21,$22::integer)`,
		c.OfertaRef, c.ReciboRef, c.BolsaRef, c.ActorRef, c.ClaveIdempotencia, datos, plazo, c.PublicadaEn, c.VenceAntesDe,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(), c.UnidadRef, c.AmbitoRef, c.NumeroPlazas).Scan(&salida, &reutilizada)
	if err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	return decodificarOferta(salida, reutilizada)
}

func (r *RepositorioOfertasPublicadasPostgreSQL) Resolver(ctx context.Context, c ports.ComandoResolverOferta) (ports.OfertaPublicada, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil || c.NumeroDePlaza < 1 || c.Tipo == "" {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	var participacion any
	if c.ParticipacionRef != "" {
		participacion = c.ParticipacionRef
	}
	m := c.Material
	var salida []byte
	var reutilizada bool
	err := r.pool.QueryRow(ctx, `SELECT oferta,reutilizada FROM vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1($1,$2,$3,$4::integer,$5,$6,$7::integer,$8,$9,$10,$11,$12,$13,$14::numeric,$15::numeric,$16,$17,$18,$19)`,
		c.OfertaRef, c.ReciboRef, c.BolsaRef, c.NumeroDePlaza, c.Tipo, participacion, c.SecuenciaEsperada, c.ActorRef, c.ClaveIdempotencia,
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
	if json.Unmarshal(salida, &oferta) != nil || !ofertaCompleta(oferta) {
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
		if !ofertaCompleta(oferta) {
			return nil, ports.ErrOfertaNoDisponible
		}
	}
	if ofertas == nil {
		ofertas = []ports.OfertaPublicada{}
	}
	return ofertas, nil
}

// ofertaCompleta exige la proyección B58: una entrada por plaza, numeradas
// desde 1 y con estado.
func ofertaCompleta(o ports.OfertaPublicada) bool {
	if o.OfertaRef == "" || o.Estado == "" || o.NumeroPlazas < 1 || len(o.Plazas) != o.NumeroPlazas ||
		!dominiobolsa.ConfirmacionAdjudicacionValida(o.ConfirmacionAdjudicacion) {
		return false
	}
	for i, p := range o.Plazas {
		if p.NumeroDePlaza != i+1 || p.Estado == "" || p.Secuencia < 0 {
			return false
		}
	}
	return true
}

// errorOferta traduce los códigos de las migraciones 000028 y 000058 sin
// exponer su texto.
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
		case "VBO07":
			return ports.ErrOfertaRespuestaAbierta
		case "VBO08":
			return ports.ErrOfertaPoliticaSinPlazas
		case "22023", "23503":
			return ports.ErrOfertaInvalida
		}
	}
	return ports.ErrOfertaNoDisponible
}

// serializarPlazoOfertaPostgreSQL conserva el formato UTC de seis decimales
// que liga la evidencia al material V3 y a la validación de PostgreSQL.
// El dominio mantiene time.Time; el formato corresponde al adaptador.
func serializarPlazoOfertaPostgreSQL(p ports.PlazoOferta) ([]byte, error) {
	type notificacionSQL struct {
		dominiobolsa.NotificacionOferta
		NotificadaEn string `json:"notificada_en"`
	}
	type plazoSQL struct {
		ports.PlazoOferta
		Notificacion *notificacionSQL `json:"notificacion,omitempty"`
	}
	salida := plazoSQL{PlazoOferta: p}
	if n := p.Notificacion; n != nil {
		salida.Notificacion = &notificacionSQL{NotificacionOferta: *n, NotificadaEn: n.NotificadaEn.UTC().Format("2006-01-02T15:04:05.000000Z")}
	}
	return json.Marshal(salida)
}
