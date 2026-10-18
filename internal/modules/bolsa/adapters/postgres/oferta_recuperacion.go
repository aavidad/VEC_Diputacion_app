package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var _ ports.RecuperadorOfertaPublicada = (*RepositorioOfertasPublicadasPostgreSQL)(nil)

// La lectura exacta consume autoridad nueva antes de devolver el original. No
// consulta la política vigente ni interpreta una página vacía como ausencia.
func (r *RepositorioOfertasPublicadasPostgreSQL) RecuperarOferta(ctx context.Context, c ports.ComandoRecuperarOferta) (ports.OfertaPublicada, bool, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil {
		return ports.OfertaPublicada{}, false, ports.ErrOfertaNoDisponible
	}
	datos, err := json.Marshal(c.Datos)
	if err != nil {
		return ports.OfertaPublicada{}, false, ports.ErrOfertaInvalida
	}
	origen, err := serializarOrigenOfertaCT(c.OrigenCT)
	if err != nil {
		return ports.OfertaPublicada{}, false, ports.ErrOfertaInvalida
	}
	notificacion, err := serializarNotificacionOfertaSQL(c.Notificacion)
	if err != nil {
		return ports.OfertaPublicada{}, false, ports.ErrOfertaInvalida
	}
	m := c.Material
	var salida []byte
	var encontrada bool
	tx, err := r.transaccionOferta(ctx)
	if err != nil {
		return ports.OfertaPublicada{}, false, errorOferta(err)
	}
	defer revertir(tx)
	err = tx.QueryRow(ctx, `SELECT oferta,encontrada FROM vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(
		$1,$2,$3,$4::jsonb,$5::integer,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12::numeric,$13::numeric,$14,$15,$16,$17,$18,$19)`,
		c.BolsaRef, c.ActorRef, c.ClaveIdempotencia, datos, c.NumeroPlazas, origen, notificacion,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
		c.UnidadRef, c.AmbitoRef,
	).Scan(&salida, &encontrada)
	if err != nil {
		return ports.OfertaPublicada{}, false, errorOferta(err)
	}
	if !encontrada {
		if len(salida) != 0 && string(salida) != "null" {
			return ports.OfertaPublicada{}, false, ports.ErrOfertaNoDisponible
		}
		if err = tx.Commit(ctx); err != nil {
			return ports.OfertaPublicada{}, false, errorOferta(err)
		}
		return ports.OfertaPublicada{}, false, nil
	}
	oferta, err := decodificarOferta(salida, true)
	if err != nil {
		return ports.OfertaPublicada{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.OfertaPublicada{}, false, errorOferta(err)
	}
	return oferta, true, nil
}

func (r *RepositorioOfertasPublicadasPostgreSQL) transaccionOferta(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		revertir(tx)
		return nil, err
	}
	return tx, nil
}
