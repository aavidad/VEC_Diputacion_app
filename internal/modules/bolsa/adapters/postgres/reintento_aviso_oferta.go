package postgres

import (
	"context"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func (r *RepositorioOfertasPublicadasPostgreSQL) ReintentarAvisosSinConfirmacion(
	ctx context.Context, c ports.ComandoReintentarAvisosOferta,
) (ports.OfertaPublicada, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil ||
		!c.ConfirmarDuplicado {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	tx, err := r.transaccionOferta(ctx)
	if err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	defer revertir(tx)
	m := c.Material
	var raw []byte
	var reutilizada bool
	err = tx.QueryRow(ctx, `SELECT oferta,reutilizada FROM vec_bolsa_llamamientos.reintentar_avisos_sin_confirmacion_v1(
	 $1,$2,$3,$4,$5::bigint,$6,$7,$8,$9,$10,$11::numeric,$12::numeric,$13,$14,$15,$16,$17,$18)`,
		c.OfertaRef, c.BolsaRef, c.ActorRef, c.ClaveIdempotencia,
		c.RevisionAvisos, c.ConfirmarDuplicado,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI(), c.UnidadRef, c.AmbitoRef,
	).Scan(&raw, &reutilizada)
	if err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	o, err := decodificarOferta(raw, reutilizada)
	if err != nil || o.OfertaRef != c.OfertaRef || o.BolsaRef != c.BolsaRef ||
		o.Aviso == nil || o.Aviso.Revision == nil || o.ReintentoAviso == nil ||
		o.ReintentoAviso.ReciboRef == "" || o.ReintentoAviso.Rearmados < 1 {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	return o, nil
}
