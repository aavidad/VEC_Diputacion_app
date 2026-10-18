package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func (r *RepositorioOfertasPublicadasPostgreSQL) EstablecerTokensAvisoOferta(d ports.DerivadorTokenAvisoOferta) {
	r.tokensAviso = d
}

func (r *RepositorioOfertasPublicadasPostgreSQL) tokenAviso(ctx context.Context, o ports.OfertaPublicada) ([]byte, error) {
	if r == nil || r.tokensAviso == nil || o.Aviso == nil || o.VersionTokenAviso == "" {
		return nil, ports.ErrOfertaNoDisponible
	}
	token, version, err := r.tokensAviso.DerivarTokenAvisoOferta(ctx, o.OfertaRef, o.VersionTokenAviso)
	if err != nil || len(token) != 32 || version != o.VersionTokenAviso {
		clear(token)
		return nil, ports.ErrOfertaNoDisponible
	}
	return token, nil
}

func (r *RepositorioOfertasPublicadasPostgreSQL) ReclamarAvisoOferta(ctx context.Context, o ports.OfertaPublicada) (ports.IntentoAvisoOferta, bool, error) {
	token, err := r.tokenAviso(ctx, o)
	if err != nil {
		return ports.IntentoAvisoOferta{}, false, err
	}
	defer clear(token)
	tx, err := r.transaccionOferta(ctx)
	if err != nil {
		return ports.IntentoAvisoOferta{}, false, errorOferta(err)
	}
	defer revertir(tx)
	var i ports.IntentoAvisoOferta
	err = tx.QueryRow(ctx, `SELECT participacion_ref,message_id,intento FROM vec_bolsa_llamamientos.reclamar_aviso_oferta_v1($1,$2,1)`,
		o.OfertaRef, token).Scan(&i.ParticipacionRef, &i.MessageID, &i.Intento)
	if errors.Is(err, pgx.ErrNoRows) {
		return i, false, nil
	}
	if err != nil {
		return i, false, errorOferta(err)
	}
	if i.ParticipacionRef == "" || i.MessageID == "" || i.Intento < 1 {
		return i, false, ports.ErrOfertaNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.IntentoAvisoOferta{}, false, errorOferta(err)
	}
	return i, true, nil
}

func (r *RepositorioOfertasPublicadasPostgreSQL) FinalizarAvisoOferta(ctx context.Context, o ports.OfertaPublicada, i ports.IntentoAvisoOferta, estado string) (ports.OfertaPublicada, error) {
	token, err := r.tokenAviso(ctx, o)
	if err != nil {
		return ports.OfertaPublicada{}, err
	}
	defer clear(token)
	tx, err := r.transaccionOferta(ctx)
	if err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	defer revertir(tx)
	var raw []byte
	if err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.finalizar_aviso_oferta_v1($1,$2,$3,$4,$5)`,
		o.OfertaRef, i.ParticipacionRef, token, i.Intento, estado).Scan(&raw); err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	actual, err := decodificarOferta(raw, o.Reutilizada)
	if err != nil || actual.OfertaRef != o.OfertaRef || actual.BolsaRef != o.BolsaRef || actual.ReciboRef != o.ReciboRef ||
		!actual.PublicadaEn.Equal(o.PublicadaEn) || !actual.VenceAntesDe.Equal(o.VenceAntesDe) || actual.Datos != o.Datos {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.OfertaPublicada{}, errorOferta(err)
	}
	return actual, nil
}
