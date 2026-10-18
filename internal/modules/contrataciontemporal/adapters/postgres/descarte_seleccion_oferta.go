package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// RegistroDescarteSeleccionOfertaPostgreSQL confirma el cierre Bolsa B121 en
// CT220 y conserva el terminal propio sin reabrir la selección.
type RegistroDescarteSeleccionOfertaPostgreSQL struct{ pool iniciadorTransacciones }

var _ ports.RegistroDescarteSeleccionOfertaCT = (*RegistroDescarteSeleccionOfertaPostgreSQL)(nil)

func NuevoRegistroDescarteSeleccionOfertaPostgreSQL(pool *pgxpool.Pool) (*RegistroDescarteSeleccionOfertaPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, errRegistroSeleccionOfertaPostgreSQL
	}
	return &RegistroDescarteSeleccionOfertaPostgreSQL{pool: pool}, nil
}

func (r *RegistroDescarteSeleccionOfertaPostgreSQL) Descartar(ctx context.Context,
	reserva ports.ReservaSeleccionOfertaCT, cierre ports.ReciboCierreOfertaCT,
	a ports.AutorizacionDescarteSeleccionOfertaCT,
) (ports.ReciboDescarteOfertaCT, error) {
	var vacio ports.ReciboDescarteOfertaCT
	i := reserva.Intencion
	intencion, err := codificarIntencionSeleccionOferta(i)
	if err != nil || reserva.ReservaRef == "" || reserva.Comando.ValidarPara(i) != nil ||
		cierre.ValidarPara(i, reserva.Comando) != nil || r == nil || dependenciaNula(r.pool) {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	canon, err := ports.CanonDescarteSeleccionOfertaCT(ports.SolicitudAutorizacionDescarteOfertaCT{
		Intencion: i, Comando: reserva.Comando, Cierre: cierre,
	})
	if err != nil || a.ActorRef != i.ActorRef || a.PerfilRef == i.PerfilRef ||
		a.PerfilRef == "" || a.MaterialCanonico != canon || a.Material.ValidarEstructura() != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	comando, err := json.Marshal(reserva.Comando)
	if err != nil {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	cierreJSON, err := json.Marshal(cierre)
	if err != nil {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	args := append([]any{intencion, reserva.ReservaRef, comando, cierreJSON, a.MaterialCanonico},
		argumentosV3SeleccionOferta(ports.AutorizacionSeleccionOfertaCT{Material: a.Material})...)
	registro := &RegistroSeleccionOfertaPostgreSQL{pool: r.pool}
	raw, err := registro.ejecutar(ctx, `SELECT vec_contratacion_temporal.descartar_seleccion_oferta_ct220(
		$1::jsonb,$2::text,$3::jsonb,$4::jsonb,$5::text,$6::bytea,$7::bytea,$8::bytea,
		$9::bytea,$10::numeric,$11::numeric,$12::bytea,$13::bytea,$14::bytea,$15::bytea)`, args...)
	if err != nil {
		return vacio, err
	}
	var recibo ports.ReciboDescarteOfertaCT
	if json.Unmarshal(raw, &recibo) != nil || recibo.ValidarPara(i, reserva.Comando, cierre) != nil {
		return vacio, errRegistroSeleccionOfertaPostgreSQL
	}
	return recibo, nil
}
