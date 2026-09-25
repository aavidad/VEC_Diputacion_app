package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type RepositorioContactoParticipacionPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioContactoParticipacion = (*RepositorioContactoParticipacionPostgreSQL)(nil)

func NuevoRepositorioContactoParticipacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioContactoParticipacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrContactoParticipacionNoDisponible
	}
	return &RepositorioContactoParticipacionPostgreSQL{pool}, nil
}
func (r *RepositorioContactoParticipacionPostgreSQL) ParticipacionPerteneceABolsa(ctx context.Context, bolsa, participacion string) (bool, error) {
	s := RepositorioSituacionParticipacionPostgreSQL{r.pool}
	return s.ParticipacionPerteneceABolsa(ctx, bolsa, participacion)
}
func (r *RepositorioContactoParticipacionPostgreSQL) RegistrarContacto(ctx context.Context, c ports.ComandoRegistrarContactoParticipacion) (ports.RegistroContactoParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Contacto.Validar() != nil || c.ClaveIdempotencia == "" || c.ReciboRef == "" || c.Material.ValidarEstructura() != nil {
		return ports.RegistroContactoParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.RegistroContactoParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := c.Material
	out := ports.RegistroContactoParticipacion{Contacto: c.Contacto}
	if requiereRegistroContactoV2(c) {
		err = registrarContactoV2(ctx, tx, c, &out)
	} else {
		err = tx.QueryRow(ctx, `SELECT reutilizado,recibo_ref,contacto_ref FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21)`, c.Contacto.ContactoRef, c.Contacto.BolsaRef, c.Contacto.ParticipacionRef, nuloTexto(c.Contacto.LlamamientoRef), c.Contacto.Canal, c.Contacto.Instante, c.Contacto.Actor, c.Contacto.Resultado, c.Contacto.Anotacion, c.ClaveIdempotencia, c.ReciboRef, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&out.Reutilizado, &out.ReciboRef, &out.Contacto.ContactoRef)
	}
	if err != nil {
		return ports.RegistroContactoParticipacion{}, errorContactoParticipacion(err)
	}
	if out.ReciboRef != c.ReciboRef {
		return ports.RegistroContactoParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.RegistroContactoParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	return out, nil
}
func nuloTexto(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func (r *RepositorioContactoParticipacionPostgreSQL) ListarContactosParticipacion(ctx context.Context, q ports.ConsultaContactosParticipacion) (ports.PaginaContactosParticipacion, error) {
	return r.listar(ctx, q.BolsaRef, q.ParticipacionRef, q.Cursor, q.Limite, q.Material)
}
func (r *RepositorioContactoParticipacionPostgreSQL) ListarContactosBolsa(ctx context.Context, q ports.ConsultaContactosBolsa) (ports.PaginaContactosParticipacion, error) {
	return r.listar(ctx, q.BolsaRef, "", q.Cursor, q.Limite, q.Material)
}
func (r *RepositorioContactoParticipacionPostgreSQL) listar(ctx context.Context, bolsa, participacion, cursor string, limite int, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.PaginaContactosParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || m.ValidarEstructura() != nil {
		return ports.PaginaContactosParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.PaginaContactosParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	filas, err := tx.Query(ctx, `SELECT contacto_ref,bolsa_ref,participacion_ref,coalesce(llamamiento_ref,''),canal,instante,actor,resultado,anotacion FROM vec_bolsa_llamamientos.listar_contactos_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9::numeric,$10::numeric,$11,$12,$13,$14)`, bolsa, nuloTexto(participacion), nuloTexto(cursor), limite, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return ports.PaginaContactosParticipacion{}, errorContactoParticipacion(err)
	}
	defer filas.Close()
	p := ports.PaginaContactosParticipacion{}
	for filas.Next() {
		var c dominiobolsa.ContactoParticipacion
		if err = filas.Scan(&c.ContactoRef, &c.BolsaRef, &c.ParticipacionRef, &c.LlamamientoRef, &c.Canal, &c.Instante, &c.Actor, &c.Resultado, &c.Anotacion); err != nil {
			return ports.PaginaContactosParticipacion{}, ports.ErrContactoParticipacionNoDisponible
		}
		c.Instante = c.Instante.UTC()
		if c.Validar() != nil {
			return ports.PaginaContactosParticipacion{}, ports.ErrContactoParticipacionNoDisponible
		}
		p.Contactos = append(p.Contactos, c)
	}
	if err = filas.Err(); err != nil {
		return ports.PaginaContactosParticipacion{}, errorContactoParticipacion(err)
	}
	filas.Close()
	if err = tx.Commit(ctx); err != nil {
		return ports.PaginaContactosParticipacion{}, errorContactoParticipacion(err)
	}
	if len(p.Contactos) > 0 {
		p.CursorSiguiente = p.Contactos[len(p.Contactos)-1].ContactoRef
	}
	return p, nil
}
func errorContactoParticipacion(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "23503":
			return ports.ErrContactoParticipacionNoEncontrado
		case "VBC01", "22023":
			return dominiobolsa.ErrContactoParticipacionInvalido
		case "VBC02":
			return dominiobolsa.ErrIntentoAntesDeSeparacion
		case "VBC03":
			return dominiobolsa.ErrIntentosContactoAgotados
		}
	}
	return ports.ErrContactoParticipacionNoDisponible
}
