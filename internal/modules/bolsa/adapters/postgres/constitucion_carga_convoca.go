package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var _ ports.RepositorioConstitucionCargaConvoca = (*RepositorioConstitucionPostgreSQL)(nil)

// auditoriaRefConsumoCargaConvoca es la forma del asiento de consumo del núcleo AD3.
var auditoriaRefConsumoCargaConvoca = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)

type reciboCargaConvocaJSON struct {
	reciboConstitucionJSON
	DecisionRef  string `json:"decision_ref"`
	AuditoriaRef string `json:"auditoria_ref"`
	ConsumidaEn  string `json:"consumida_en"`
}

// ConstituirCargaConvocaAutorizada (B79) consume la decisión de la carga
// (AD203) y constituye la bolsa en una sola transacción SERIALIZABLE. El acta
// es el recurso de la decisión y el actor de la constitución es su titular.
func (r *RepositorioConstitucionPostgreSQL) ConstituirCargaConvocaAutorizada(ctx context.Context, c ports.Constitucion, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCargaConvoca, error) {
	if ctx == nil || r == nil || r.pool == nil || m.ValidarEstructura() != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	argumentos, err := argumentosConstitucion(c)
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	argumentos = append(argumentos, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	defer tx.Rollback(context.Background())
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(
		$1::text, $2::text, $3::text, $4::text, $5::bigint, $6::bytea, $7::timestamptz,
		$8::text, $9::bigint, $10::bytea, $11::timestamptz, $12::timestamptz, $13::jsonb, $14::timestamptz,
		$15::bytea, $16::bytea, $17::bytea, $18::bytea, $19::numeric, $20::numeric, $21::bytea, $22::bytea, $23::bytea, $24::bytea)`,
		argumentos...,
	).Scan(&contenido)
	if err != nil {
		return ports.ReciboCargaConvoca{}, errorConstitucionCargaConvoca(ctx, err)
	}
	var leido reciboCargaConvocaJSON
	if json.Unmarshal(contenido, &leido) != nil || leido.ActaRef != c.ActaRef || leido.DecisionRef == "" ||
		!auditoriaRefConsumoCargaConvoca.MatchString(leido.AuditoriaRef) {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	consumida, err := time.Parse("2006-01-02T15:04:05.000000Z", leido.ConsumidaEn)
	if err != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	recibo, err := leido.traducir()
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	return ports.ReciboCargaConvoca{ReciboConstitucion: recibo, DecisionRef: leido.DecisionRef, AuditoriaRef: leido.AuditoriaRef, ConsumidaEn: consumida.UTC()}, nil
}

// errorConstitucionCargaConvoca separa la denegación (42501 de AD203/B79) del
// resto de fallos de la constitución.
func errorConstitucionCargaConvoca(ctx context.Context, err error) error {
	var errorPG *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &errorPG) && errorPG.Code == "42501" {
		return dominiovec.ErrAutorizacionDenegada
	}
	return errorConstitucion(ctx, err)
}
