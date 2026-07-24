package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionResolverCandidaturaAltaV1 = "vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1"
	maximoIntentosCandidaturaAlta    = 3
)

// ResolutorCandidaturaAltaPostgreSQL estabiliza referencias opacas sin crear
// expediente, reserva administrativa, auditoría ni evento.
type ResolutorCandidaturaAltaPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoResolutorCandidaturaAltaPostgreSQL(
	pool *pgxpool.Pool,
) (*ResolutorCandidaturaAltaPostgreSQL, error) {
	return nuevoResolutorCandidaturaAltaPostgreSQL(pool)
}

func nuevoResolutorCandidaturaAltaPostgreSQL(
	pool iniciadorTransacciones,
) (*ResolutorCandidaturaAltaPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrPersistenciaNoDisponible
	}
	return &ResolutorCandidaturaAltaPostgreSQL{pool: pool}, nil
}

func (r *ResolutorCandidaturaAltaPostgreSQL) ResolverCandidaturaAlta(
	ctx context.Context,
	solicitud ports.SolicitudResolverCandidaturaAlta,
) (ports.CandidaturaAlta, error) {
	if ctx == nil || r == nil || dependenciaNula(r.pool) ||
		solicitud.Validar() != nil {
		return ports.CandidaturaAlta{}, ports.ErrPreparacionAltaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.CandidaturaAlta{}, err
	}
	ambitos, huellas, err := paresCandidaturaAlta(solicitud)
	if err != nil {
		return ports.CandidaturaAlta{}, err
	}
	for intento := 1; intento <= maximoIntentosCandidaturaAlta; intento++ {
		candidatura, errIntento := r.resolverEnTransaccion(
			ctx,
			solicitud,
			ambitos,
			huellas,
		)
		if errIntento == nil {
			return candidatura, nil
		}
		if err := ctx.Err(); err != nil {
			return ports.CandidaturaAlta{}, err
		}
		if errors.Is(errIntento, ports.ErrClaveIdempotenciaUsada) {
			return ports.CandidaturaAlta{}, ports.ErrClaveIdempotenciaUsada
		}
		if !errorPostgreSQLReintentable(errIntento) ||
			intento == maximoIntentosCandidaturaAlta {
			return ports.CandidaturaAlta{}, ports.ErrPersistenciaNoDisponible
		}
	}
	return ports.CandidaturaAlta{}, ports.ErrPersistenciaNoDisponible
}

func paresCandidaturaAlta(
	solicitud ports.SolicitudResolverCandidaturaAlta,
) ([]string, []string, error) {
	if solicitud.Validar() != nil {
		return nil, nil, ports.ErrPreparacionAltaInvalida
	}
	ambitos, errAmbitos := solicitud.AmbitosIdempotenciaHMAC.Datos()
	huellas, errHuellas := solicitud.HuellasPeticionHMAC.Datos()
	if errAmbitos != nil || errHuellas != nil ||
		len(ambitos.Retenidos) != len(huellas.Retenidos) {
		return nil, nil, ports.ErrPreparacionAltaInvalida
	}
	valoresAmbito := make([]string, 1, len(ambitos.Retenidos)+1)
	valoresHuella := make([]string, 1, len(huellas.Retenidos)+1)
	valoresAmbito[0] = ambitos.Activo.Valor
	valoresHuella[0] = huellas.Activo.Valor
	for indice := range ambitos.Retenidos {
		if ambitos.Retenidos[indice].Generacion !=
			huellas.Retenidos[indice].Generacion {
			return nil, nil, ports.ErrPreparacionAltaInvalida
		}
		valoresAmbito = append(
			valoresAmbito,
			ambitos.Retenidos[indice].Valor,
		)
		valoresHuella = append(
			valoresHuella,
			huellas.Retenidos[indice].Valor,
		)
	}
	return valoresAmbito, valoresHuella, nil
}

func (r *ResolutorCandidaturaAltaPostgreSQL) resolverEnTransaccion(
	ctx context.Context,
	solicitud ports.SolicitudResolverCandidaturaAlta,
	ambitos []string,
	huellas []string,
) (ports.CandidaturaAlta, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return ports.CandidaturaAlta{}, err
	}
	defer revertirTransaccion(tx)
	if err := configurarTransaccionConfirmacion(ctx, tx); err != nil {
		return ports.CandidaturaAlta{}, err
	}
	propuesta := solicitud.Propuesta
	var resultado string
	var candidatura ports.CandidaturaAlta
	err = tx.QueryRow(ctx, `
		SELECT resultado, ambito_hmac, huella_peticion_hmac,
		       reserva_ref, expediente_ref, numero_visible, recibo_ref,
		       organizacion_ref, actor_ref, perfil_ref
		  FROM `+funcionResolverCandidaturaAltaV1+`(
		       $1::text[], $2::text[], $3, $4, $5, $6, $7, $8, $9, $10
		  )`,
		ambitos,
		huellas,
		solicitud.OrganizacionRef,
		solicitud.ActorRef,
		solicitud.PerfilRef,
		propuesta.ReservaRef,
		propuesta.Referencias.ExpedienteRef,
		propuesta.Referencias.NumeroVisible,
		propuesta.Referencias.ReciboRef,
		propuesta.AmbitoIdempotenciaHMAC,
	).Scan(
		&resultado,
		&candidatura.AmbitoIdempotenciaHMAC,
		&candidatura.HuellaPeticionHMAC,
		&candidatura.ReservaRef,
		&candidatura.Referencias.ExpedienteRef,
		&candidatura.Referencias.NumeroVisible,
		&candidatura.Referencias.ReciboRef,
		&candidatura.OrganizacionRef,
		&candidatura.ActorRef,
		&candidatura.PerfilRef,
	)
	if err != nil {
		return ports.CandidaturaAlta{}, err
	}
	switch resultado {
	case "idempotencia_reutilizada":
		return ports.CandidaturaAlta{}, ports.ErrClaveIdempotenciaUsada
	case "estabilizada":
		if candidatura != propuesta {
			return ports.CandidaturaAlta{}, ports.ErrResultadoAltaNoConfiable
		}
	case "recuperada":
	default:
		return ports.CandidaturaAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	if solicitud.ValidarResultado(candidatura) != nil {
		return ports.CandidaturaAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return ports.CandidaturaAlta{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.CandidaturaAlta{}, err
	}
	return candidatura, nil
}

var _ ports.ResolutorCandidaturaAlta = (*ResolutorCandidaturaAltaPostgreSQL)(nil)
