package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ValidadorMotivosUsuariosExterno solo llama a la fachada AUT-17 de motivos
// publicados para candidatos. No consulta el catálogo interno.
type ValidadorMotivosUsuariosExterno struct {
	consulta   consultorFilaMotivoAutorizacionV2
	catalogoID string
}

func NuevoValidadorMotivosUsuariosExterno(pool *pgxpool.Pool, catalogoID string) (*ValidadorMotivosUsuariosExterno, error) {
	return nuevoValidadorMotivosUsuariosExterno(pool, catalogoID)
}

func nuevoValidadorMotivosUsuariosExterno(consulta consultorFilaMotivoAutorizacionV2, catalogoID string) (*ValidadorMotivosUsuariosExterno, error) {
	centinela := domain.ReferenciaEntradaCatalogo{CatalogoID: catalogoID, CatalogoVersion: 1,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EntradaClave:         "motivo_00000000000000000000000000000001"}
	if valorNuloPostgreSQL(consulta) || !domain.ReferenciaMotivoAutorizacionV2Valida(centinela) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ValidadorMotivosUsuariosExterno{consulta: consulta, catalogoID: catalogoID}, nil
}

func (v *ValidadorMotivosUsuariosExterno) ValidarReferenciaMotivoAutorizacionV2(ctx context.Context, referencia domain.ReferenciaEntradaCatalogo, instante time.Time) error {
	if v == nil || valorNuloPostgreSQL(v.consulta) || ctx == nil || referencia.CatalogoID != v.catalogoID ||
		!domain.ReferenciaMotivoAutorizacionV2Valida(referencia) || !instanteHistoricoMotivoAutorizacionV2Valido(instante) {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return errorValidacionMotivoAutorizacionV2PostgreSQL(ctx, err, false)
	}
	var resuelta bool
	err := v.consulta.QueryRow(ctx, `SELECT vec_autorizacion.resolver_motivo_usuarios_externo_v1(
		$1::text,$2::integer,$3::text,$4::text,$5::timestamptz)`,
		referencia.CatalogoID, referencia.CatalogoVersion, referencia.CatalogoHuellaSHA256,
		referencia.EntradaClave, instante).Scan(&resuelta)
	if err != nil {
		return errorValidacionMotivoAutorizacionV2PostgreSQL(ctx, err, true)
	}
	if err := ctx.Err(); err != nil {
		return errorValidacionMotivoAutorizacionV2PostgreSQL(ctx, err, false)
	}
	if !resuelta {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

var _ ports.ValidadorReferenciaMotivoAutorizacionV2 = (*ValidadorMotivosUsuariosExterno)(nil)
