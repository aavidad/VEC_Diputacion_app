package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente persiste una
// concesion V2 en un registro nominal separado de V1. PostgreSQL recibe los
// bytes canonicos de decision y motivo; nunca una proyeccion que pueda
// reinterpretarse como capacidad historica.
func (a *AlmacenAutorizacion) RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
	ctx context.Context,
	orden ports.OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
) error {
	if a == nil || valorNuloPostgreSQL(a.pool) {
		return ports.ErrRegistroDecisionNoDisponible
	}
	if ctx == nil {
		return ports.ErrRegistroDecisionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	datos, err := orden.Datos()
	if err != nil {
		return err
	}
	if !datos.Decision.Concedida || datos.Decision.Codigo != "concedida" {
		return ports.ErrRegistroDecisionNoDisponible
	}
	decisionCanonica, motivoCanonico, err := serializarDecisionSolicitudLigadaV2PostgreSQL(
		datos.Decision,
		datos.ReferenciaMotivo,
	)
	if err != nil {
		return ports.ErrRegistroDecisionNoDisponible
	}
	defer borrarBytesAutorizacionPostgreSQL(decisionCanonica, motivoCanonico)

	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return errorRegistroAutorizacion(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionAutorizacion(ctx, tx); err != nil {
		return errorRegistroAutorizacion(ctx, err)
	}
	var registrada bool
	err = tx.QueryRow(ctx, `
		SELECT vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
			$1::bytea, $2::bytea
		)`,
		decisionCanonica,
		motivoCanonico,
	).Scan(&registrada)
	if err != nil {
		return errorRegistroAutorizacion(ctx, err)
	}
	if !registrada {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	if err = tx.Commit(ctx); err != nil {
		return errorRegistroAutorizacion(ctx, err)
	}
	return nil
}

func serializarDecisionSolicitudLigadaV2PostgreSQL(
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
) ([]byte, []byte, error) {
	if decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil {
		return nil, nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if err != nil || huellaMotivo != decision.MotivoHuellaSHA256 {
		return nil, nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	decisionCanonica, err := ports.RepresentacionCanonicaDecisionAutorizacionReforzadaV2(decision)
	if err != nil {
		return nil, nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	motivoCanonico, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(referenciaMotivo)
	if err != nil {
		borrarBytesAutorizacionPostgreSQL(decisionCanonica)
		return nil, nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	sumaMotivo := sha256.Sum256(motivoCanonico)
	if hex.EncodeToString(sumaMotivo[:]) != decision.MotivoHuellaSHA256 {
		borrarBytesAutorizacionPostgreSQL(decisionCanonica, motivoCanonico)
		return nil, nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return decisionCanonica, motivoCanonico, nil
}

func borrarBytesAutorizacionPostgreSQL(contenidos ...[]byte) {
	for _, contenido := range contenidos {
		for indice := range contenido {
			contenido[indice] = 0
		}
	}
}

var _ ports.RegistroDecisionesAutorizacionSolicitudLigadaV2 = (*AlmacenAutorizacion)(nil)
