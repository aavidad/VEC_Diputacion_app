package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	puertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type RepositorioCatalogoCausasParticipacionPostgreSQL struct{ pool *pgxpool.Pool }

const maximoCausasParticipacionConsultables = 500

var _ puertos.RepositorioCatalogoCausasParticipacion = (*RepositorioCatalogoCausasParticipacionPostgreSQL)(nil)

func NuevoRepositorioCatalogoCausasParticipacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioCatalogoCausasParticipacionPostgreSQL, error) {
	if pool == nil {
		return nil, puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	return &RepositorioCatalogoCausasParticipacionPostgreSQL{pool}, nil
}

func (r *RepositorioCatalogoCausasParticipacionPostgreSQL) ProponerCausaParticipacion(ctx context.Context, c puertos.ComandoProponerCausaParticipacion) (puertos.PropuestaCausaParticipacion, string, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Actor == "" || c.PropuestaRef == "" || c.ReciboRef == "" || c.Material.ValidarEstructura() != nil {
		return puertos.PropuestaCausaParticipacion{}, "", puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return puertos.PropuestaCausaParticipacion{}, "", puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := c.Material
	var propuesta puertos.PropuestaCausaParticipacion
	var recibo string
	err = tx.QueryRow(ctx, `SELECT propuesta_ref,codigo,version,huella_sha256,recibo_ref FROM vec_bolsa_llamamientos.proponer_causa_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21)`,
		c.Causa.Codigo, c.Causa.Version, c.Causa.Etiqueta, c.Causa.AplicaSituacion, c.Causa.AplicaContacto, c.Causa.Publicable, c.Causa.Activa, c.Causa.HuellaSHA256, c.Actor, c.PropuestaRef, c.ReciboRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(
		&propuesta.PropuestaRef, &propuesta.Codigo, &propuesta.Version, &propuesta.HuellaSHA256, &recibo)
	if err != nil {
		return puertos.PropuestaCausaParticipacion{}, "", errorCatalogoPG(err)
	}
	if propuesta.PropuestaRef != c.PropuestaRef || recibo != c.ReciboRef || propuesta.Codigo != c.Causa.Codigo || propuesta.Version != c.Causa.Version || propuesta.HuellaSHA256 != c.Causa.HuellaSHA256 {
		return puertos.PropuestaCausaParticipacion{}, "", puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return puertos.PropuestaCausaParticipacion{}, "", puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	propuesta.CausaParticipacionCatalogada = c.Causa
	return propuesta, recibo, nil
}
func (r *RepositorioCatalogoCausasParticipacionPostgreSQL) PublicarCausaParticipacion(ctx context.Context, c puertos.ComandoPublicarCausaParticipacion) (puertos.CausaParticipacionCatalogada, string, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Actor == "" || c.PropuestaRef == "" || c.ReciboRef == "" || c.Material.ValidarEstructura() != nil {
		return puertos.CausaParticipacionCatalogada{}, "", puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if e != nil {
		return puertos.CausaParticipacionCatalogada{}, "", puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := c.Material
	var out puertos.CausaParticipacionCatalogada
	var recibo string
	e = tx.QueryRow(ctx, `SELECT codigo,version,huella_sha256,recibo_ref FROM vec_bolsa_llamamientos.publicar_causa_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::numeric,$16::numeric,$17,$18,$19,$20,$21)`, c.Causa.Codigo, c.Causa.Version, c.Causa.Etiqueta, c.Causa.AplicaSituacion, c.Causa.AplicaContacto, c.Causa.Publicable, c.Causa.Activa, c.Causa.HuellaSHA256, c.Actor, c.ReciboRef, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(), c.PropuestaRef).Scan(&out.Codigo, &out.Version, &out.HuellaSHA256, &recibo)
	if e != nil {
		return out, "", errorCatalogoPG(e)
	}
	if recibo != c.ReciboRef || out.Codigo != c.Causa.Codigo || out.Version != c.Causa.Version || out.HuellaSHA256 != c.Causa.HuellaSHA256 {
		return out, "", puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	if e = tx.Commit(ctx); e != nil {
		return out, "", puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	out.Etiqueta, out.AplicaSituacion, out.AplicaContacto, out.Publicable, out.Activa = c.Causa.Etiqueta, c.Causa.AplicaSituacion, c.Causa.AplicaContacto, c.Causa.Publicable, c.Causa.Activa
	return out, recibo, nil
}
func (r *RepositorioCatalogoCausasParticipacionPostgreSQL) ListarCausasParticipacion(ctx context.Context, q puertos.ConsultaCausasParticipacionAutorizada) ([]puertos.CausaParticipacionCatalogada, error) {
	if r == nil || r.pool == nil || ctx == nil || q.Actor == "" || q.Material.ValidarEstructura() != nil {
		return nil, puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if e != nil {
		return nil, puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := q.Material
	rows, e := tx.Query(ctx, `SELECT codigo,version,huella_sha256,etiqueta,aplica_situacion,aplica_contacto FROM vec_bolsa_llamamientos.listar_causas_participacion_v2($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`, q.Actor, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if e != nil {
		return nil, errorCatalogoPG(e)
	}
	defer rows.Close()
	out := []puertos.CausaParticipacionCatalogada{}
	for rows.Next() {
		if len(out) == maximoCausasParticipacionConsultables {
			return nil, puertos.ErrCatalogoCausasParticipacionNoDisponible
		}
		var x puertos.CausaParticipacionCatalogada
		if e = rows.Scan(&x.Codigo, &x.Version, &x.HuellaSHA256, &x.Etiqueta, &x.AplicaSituacion, &x.AplicaContacto); e != nil {
			return nil, puertos.ErrCatalogoCausasParticipacionNoDisponible
		}
		out = append(out, x)
	}
	if e = rows.Err(); e != nil {
		return nil, errorCatalogoPG(e)
	}
	rows.Close()
	if e = tx.Commit(ctx); e != nil {
		return nil, puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	return out, nil
}

func (r *RepositorioCatalogoCausasParticipacionPostgreSQL) ConsultarPropuestaCausaParticipacion(ctx context.Context, q puertos.ConsultaPropuestaCausaParticipacionAutorizada) (puertos.PropuestaCausaParticipacionLeida, error) {
	if r == nil || r.pool == nil || ctx == nil || q.PropuestaRef == "" || q.Actor == "" || q.Material.ValidarEstructura() != nil {
		return puertos.PropuestaCausaParticipacionLeida{}, puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return puertos.PropuestaCausaParticipacionLeida{}, puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	defer tx.Rollback(context.Background())
	m := q.Material
	var leida puertos.PropuestaCausaParticipacionLeida
	err = tx.QueryRow(ctx, `SELECT propuesta_ref,codigo,version,huella_sha256,etiqueta,aplica_situacion,aplica_contacto,publicable,activa,recibo_ref,estado FROM vec_bolsa_llamamientos.consultar_propuesta_causa_participacion_v1($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`,
		q.PropuestaRef, q.Actor, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(
		&leida.Propuesta.PropuestaRef, &leida.Propuesta.Codigo, &leida.Propuesta.Version, &leida.Propuesta.HuellaSHA256, &leida.Propuesta.Etiqueta,
		&leida.Propuesta.AplicaSituacion, &leida.Propuesta.AplicaContacto, &leida.Propuesta.Publicable, &leida.Propuesta.Activa, &leida.Recibo, &leida.Estado)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return puertos.PropuestaCausaParticipacionLeida{}, puertos.ErrCatalogoCausasParticipacionNoDisponible
		}
		return puertos.PropuestaCausaParticipacionLeida{}, puertos.ErrPropuestaCausaParticipacionNoEncontrada
	}
	if err != nil {
		return puertos.PropuestaCausaParticipacionLeida{}, errorCatalogoPG(err)
	}
	selector := puertos.SelectorCausaParticipacion{Codigo: leida.Propuesta.Codigo, Version: leida.Propuesta.Version, HuellaSHA256: leida.Propuesta.HuellaSHA256}
	if leida.Propuesta.PropuestaRef != q.PropuestaRef || selector.Validar() != nil ||
		(leida.Estado != "pendiente" && leida.Estado != "publicada" && leida.Estado != "superada") || !strings.HasPrefix(leida.Recibo, "recibo:propuesta:causa:") {
		return puertos.PropuestaCausaParticipacionLeida{}, puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return puertos.PropuestaCausaParticipacionLeida{}, puertos.ErrCatalogoCausasParticipacionNoDisponible
	}
	return leida, nil
}

func errorCatalogoPG(e error) error {
	var p *pgconn.PgError
	if errors.As(e, &p) {
		switch p.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "VBS01", "VBS57", "23505":
			return puertos.ErrCatalogoCausasParticipacionEnConflicto
		}
	}
	return puertos.ErrCatalogoCausasParticipacionNoDisponible
}
