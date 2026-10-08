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
	iso := pgx.Serializable
	if c.InstanteServidor {
		// El cerrojo por clave e intento precede al reloj y a la lectura. En
		// READ COMMITTED, un replay concurrente ve el INSERT ya confirmado.
		iso = pgx.ReadCommitted
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: iso, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.RegistroContactoParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := c.Material
	out := ports.RegistroContactoParticipacion{Contacto: c.Contacto}
	if c.InstanteServidor {
		err = registrarContactoTelefonoServidorV4(ctx, tx, c, &out)
	} else if c.Contacto.OfertaRef != "" {
		err = registrarContactoOfertaV3(ctx, tx, c, &out)
	} else if requiereRegistroContactoV2(c) {
		err = registrarContactoV2(ctx, tx, c, &out)
	} else {
		err = tx.QueryRow(ctx, `SELECT reutilizado,recibo_ref,contacto_ref FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21)`, c.Contacto.ContactoRef, c.Contacto.BolsaRef, c.Contacto.ParticipacionRef, nuloTexto(c.Contacto.LlamamientoRef), c.Contacto.Canal, c.Contacto.Instante, c.Contacto.Actor, c.Contacto.Resultado, c.Contacto.Anotacion, c.ClaveIdempotencia, c.ReciboRef, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&out.Reutilizado, &out.ReciboRef, &out.Contacto.ContactoRef)
	}
	if err != nil {
		return ports.RegistroContactoParticipacion{}, errorContactoParticipacion(err)
	}
	if !c.InstanteServidor && out.Reutilizado {
		var verificador bool
		if err = tx.QueryRow(ctx, `SELECT pg_catalog.to_regprocedure('vec_bolsa_llamamientos.verificar_replay_contacto_legado_v1(text,text)') IS NOT NULL`).Scan(&verificador); err != nil {
			return ports.RegistroContactoParticipacion{}, errorContactoParticipacion(err)
		}
		if verificador {
			if err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.verificar_replay_contacto_legado_v1($1,$2)`, c.Contacto.ParticipacionRef, c.ClaveIdempotencia).Scan(&verificador); err != nil {
				return ports.RegistroContactoParticipacion{}, errorContactoParticipacion(err)
			}
		}
	}
	if out.ReciboRef != c.ReciboRef || (c.InstanteServidor && out.Contacto.Instante.IsZero()) {
		return ports.RegistroContactoParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.RegistroContactoParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	return out, nil
}
func registrarContactoOfertaV3(ctx context.Context, tx pgx.Tx, c ports.ComandoRegistrarContactoParticipacion, out *ports.RegistroContactoParticipacion) error {
	m := c.Material
	return tx.QueryRow(ctx, `SELECT reutilizado,recibo_ref,contacto_ref FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v3($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21,$22,$23,$24)`,
		c.Contacto.ContactoRef, c.Contacto.BolsaRef, c.Contacto.ParticipacionRef, nuloTexto(c.Contacto.LlamamientoRef), c.Contacto.Canal, c.Contacto.Instante, c.Contacto.Actor, c.Contacto.Resultado, c.Contacto.Anotacion, c.ClaveIdempotencia, c.ReciboRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(), c.Contacto.OfertaRef, nuloTexto(c.Contacto.EvidenciaRef), nuloTexto(c.Contacto.EvidenciaHuellaSHA256),
	).Scan(&out.Reutilizado, &out.ReciboRef, &out.Contacto.ContactoRef)
}
func nuloTexto(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func (r *RepositorioContactoParticipacionPostgreSQL) ListarContactosParticipacion(ctx context.Context, q ports.ConsultaContactosParticipacion) (ports.PaginaContactosParticipacion, error) {
	return r.listar(ctx, q.BolsaRef, q.ParticipacionRef, q.OfertaRef, q.Cursor, q.Limite, q.Material)
}
func (r *RepositorioContactoParticipacionPostgreSQL) ListarContactosBolsa(ctx context.Context, q ports.ConsultaContactosBolsa) (ports.PaginaContactosParticipacion, error) {
	return r.listar(ctx, q.BolsaRef, "", q.OfertaRef, q.Cursor, q.Limite, q.Material)
}
func (r *RepositorioContactoParticipacionPostgreSQL) listar(ctx context.Context, bolsa, participacion, oferta, cursor string, limite int, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.PaginaContactosParticipacion, error) {
	if r == nil || r.pool == nil || ctx == nil || m.ValidarEstructura() != nil {
		return ports.PaginaContactosParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		if ctx.Err() != nil {
			return ports.PaginaContactosParticipacion{}, ctx.Err()
		}
		return ports.PaginaContactosParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	filas, err := tx.Query(ctx, `SELECT contacto_ref,bolsa_ref,participacion_ref,coalesce(llamamiento_ref,''),canal,instante,actor,resultado,anotacion,coalesce(oferta_ref,''),coalesce(evidencia_ref,''),coalesce(evidencia_huella,'') FROM vec_bolsa_llamamientos.listar_contactos_participacion_v2($1,$2,$3,$4,$5,$6,$7,$8,$9::numeric,$10::numeric,$11,$12,$13,$14,$15)`, bolsa, nuloTexto(participacion), nuloTexto(cursor), limite, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(), nuloTexto(oferta))
	if err != nil {
		return ports.PaginaContactosParticipacion{}, errorContactoParticipacion(err)
	}
	defer filas.Close()
	p := ports.PaginaContactosParticipacion{}
	for filas.Next() {
		var c dominiobolsa.ContactoParticipacion
		if err = filas.Scan(&c.ContactoRef, &c.BolsaRef, &c.ParticipacionRef, &c.LlamamientoRef, &c.Canal, &c.Instante, &c.Actor, &c.Resultado, &c.Anotacion, &c.OfertaRef, &c.EvidenciaRef, &c.EvidenciaHuellaSHA256); err != nil {
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
	// La capacidad sólo se publica después de la lectura V3 autorizada y
	// cuando B87 está realmente instalada para el ejecutor vigente.
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1(text,text,text,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[],text,integer,integer,boolean,text,date,boolean)') AND pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE'))`).Scan(&p.RegistroTelefonoDisponible)
	if err != nil {
		return ports.PaginaContactosParticipacion{}, errorContactoParticipacion(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.PaginaContactosParticipacion{}, errorContactoParticipacion(err)
	}
	if len(p.Contactos) > 0 {
		p.CursorSiguiente = p.Contactos[len(p.Contactos)-1].ContactoRef
	}
	return p, nil
}

// errorContactoParticipacion traduce el error de PostgreSQL. Un plazo
// agotado o una cancelación se conservan (como errorConstitucion) para que
// la ruta responda 504 y no un «no disponible» genérico.
func errorContactoParticipacion(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "23503":
			return ports.ErrContactoParticipacionNoEncontrado
		case "VBC01", "22023", "23514":
			return dominiobolsa.ErrContactoParticipacionInvalido
		case "VBC02":
			return dominiobolsa.ErrIntentoAntesDeSeparacion
		case "VBC03":
			return dominiobolsa.ErrIntentosContactoAgotados
		case "VBC05":
			return dominiobolsa.ErrIntentoFueraDeFranja
		}
	}
	return ports.ErrContactoParticipacionNoDisponible
}
