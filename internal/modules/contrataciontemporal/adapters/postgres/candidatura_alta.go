package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionResolverCandidaturaAlta = "vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1"
	maximoIntentosCandidaturaAlta  = 3
)

var _ ports.ResolutorCandidaturaAlta = (*ResolutorCandidaturaAltaPostgreSQL)(nil)

// ResolutorCandidaturaAltaPostgreSQL conserva solo coordenadas técnicas. La
// función SQL que invoca carece de autoridad para reservar o confirmar.
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

type entradaResolverCandidaturaAlta struct {
	ambitos         []string
	huellas         []string
	organizacionRef string
	actorRef        string
	perfilRef       string
	propuesta       ports.DatosCandidaturaAlta
}

func nuevaEntradaResolverCandidaturaAlta(
	solicitud ports.SolicitudResolverCandidaturaAlta,
) (entradaResolverCandidaturaAlta, error) {
	datos, err := solicitud.Datos()
	if err != nil {
		return entradaResolverCandidaturaAlta{}, ports.ErrPreparacionAltaInvalida
	}
	ambitos, errAmbitos := datos.AmbitosIdempotenciaHMAC.Datos()
	huellas, errHuellas := datos.HuellasPeticionHMAC.Datos()
	propuesta, errPropuesta := datos.Propuesta.Datos()
	if errAmbitos != nil || errHuellas != nil || errPropuesta != nil ||
		ambitos.Activo.Generacion != huellas.Activo.Generacion ||
		len(ambitos.Retenidos) != len(huellas.Retenidos) {
		return entradaResolverCandidaturaAlta{}, ports.ErrPreparacionAltaInvalida
	}
	entrada := entradaResolverCandidaturaAlta{
		ambitos:         make([]string, 1, 1+len(ambitos.Retenidos)),
		huellas:         make([]string, 1, 1+len(huellas.Retenidos)),
		organizacionRef: datos.OrganizacionRef,
		actorRef:        datos.ActorRef,
		perfilRef:       datos.PerfilRef,
		propuesta:       propuesta,
	}
	entrada.ambitos[0] = ambitos.Activo.Valor
	entrada.huellas[0] = huellas.Activo.Valor
	for indice := range ambitos.Retenidos {
		if ambitos.Retenidos[indice].Generacion != huellas.Retenidos[indice].Generacion {
			return entradaResolverCandidaturaAlta{}, ports.ErrPreparacionAltaInvalida
		}
		entrada.ambitos = append(entrada.ambitos, ambitos.Retenidos[indice].Valor)
		entrada.huellas = append(entrada.huellas, huellas.Retenidos[indice].Valor)
	}
	return entrada, nil
}

func (r *ResolutorCandidaturaAltaPostgreSQL) ResolverCandidaturaAlta(
	ctx context.Context,
	solicitud ports.SolicitudResolverCandidaturaAlta,
) (ports.CandidaturaAlta, error) {
	if ctx == nil || r == nil || dependenciaNula(r.pool) {
		return ports.CandidaturaAlta{}, ports.ErrPreparacionAltaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.CandidaturaAlta{}, err
	}
	entrada, err := nuevaEntradaResolverCandidaturaAlta(solicitud)
	if err != nil {
		return ports.CandidaturaAlta{}, err
	}
	for intento := 1; intento <= maximoIntentosCandidaturaAlta; intento++ {
		candidatura, causa := r.resolverEnTransaccion(ctx, solicitud, entrada)
		if causa == nil {
			return candidatura, nil
		}
		if ctx.Err() != nil {
			return ports.CandidaturaAlta{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(causa) || intento == maximoIntentosCandidaturaAlta {
			return ports.CandidaturaAlta{}, normalizarErrorCandidatura(causa)
		}
	}
	return ports.CandidaturaAlta{}, ports.ErrPersistenciaNoDisponible
}

type filaCandidaturaAlta struct {
	resultado       string
	reservaRef      string
	expedienteRef   string
	numeroVisible   string
	reciboRef       string
	ambitoHMAC      string
	huellaHMAC      string
	organizacionRef string
	actorRef        string
	perfilRef       string
	instanteEfecto  time.Time
}

func (r *ResolutorCandidaturaAltaPostgreSQL) resolverEnTransaccion(
	ctx context.Context,
	solicitud ports.SolicitudResolverCandidaturaAlta,
	entrada entradaResolverCandidaturaAlta,
) (ports.CandidaturaAlta, error) {
	tx, err := iniciarTransaccionAltaCandidata(ctx, r.pool)
	if err != nil {
		return ports.CandidaturaAlta{}, err
	}
	defer revertirTransaccion(tx)
	fila := filaCandidaturaAlta{}
	err = tx.QueryRow(ctx, `
		SELECT resultado, reserva_ref, expediente_ref, numero_visible,
		       recibo_ref, ambito_hmac, huella_peticion_hmac,
		       organizacion_ref, actor_ref, perfil_ref, instante_efecto
		  FROM `+funcionResolverCandidaturaAlta+`(
		       $1::text[], $2::text[], $3, $4, $5, $6, $7, $8, $9,
		       $10::timestamptz)`,
		entrada.ambitos, entrada.huellas, entrada.organizacionRef,
		entrada.actorRef, entrada.perfilRef, entrada.propuesta.ReservaRef,
		entrada.propuesta.Referencias.ExpedienteRef,
		entrada.propuesta.Referencias.NumeroVisible,
		entrada.propuesta.Referencias.ReciboRef,
		entrada.propuesta.InstanteEfecto,
	).Scan(
		&fila.resultado, &fila.reservaRef, &fila.expedienteRef,
		&fila.numeroVisible, &fila.reciboRef, &fila.ambitoHMAC,
		&fila.huellaHMAC, &fila.organizacionRef, &fila.actorRef,
		&fila.perfilRef, &fila.instanteEfecto,
	)
	if err != nil {
		return ports.CandidaturaAlta{}, err
	}
	candidatura, err := fila.restaurar(solicitud, entrada.propuesta)
	if err != nil {
		return ports.CandidaturaAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.CandidaturaAlta{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.CandidaturaAlta{}, err
	}
	return candidatura, nil
}

func (f filaCandidaturaAlta) restaurar(
	solicitud ports.SolicitudResolverCandidaturaAlta,
	propuesta ports.DatosCandidaturaAlta,
) (ports.CandidaturaAlta, error) {
	if f.resultado != "estabilizada" && f.resultado != "recuperada" {
		return ports.CandidaturaAlta{}, ports.ErrPersistenciaNoDisponible
	}
	datos := ports.DatosCandidaturaAlta{
		ReservaRef: f.reservaRef,
		Referencias: ports.ReferenciasAlta{
			ExpedienteRef: f.expedienteRef,
			NumeroVisible: f.numeroVisible,
			ReciboRef:     f.reciboRef,
		},
		AmbitoIdempotenciaHMAC: f.ambitoHMAC,
		HuellaPeticionHMAC:     f.huellaHMAC,
		OrganizacionRef:        f.organizacionRef,
		ActorRef:               f.actorRef,
		PerfilRef:              f.perfilRef,
		InstanteEfecto:         f.instanteEfecto.UTC(),
	}
	candidatura, err := ports.NuevaCandidaturaAlta(datos)
	if err != nil || solicitud.ValidarResultado(candidatura) != nil {
		return ports.CandidaturaAlta{}, ports.ErrPersistenciaNoDisponible
	}
	if f.resultado == "estabilizada" && datos != propuesta {
		return ports.CandidaturaAlta{}, ports.ErrPersistenciaNoDisponible
	}
	return candidatura, nil
}

func iniciarTransaccionAltaCandidata(
	ctx context.Context,
	pool iniciadorTransacciones,
) (pgx.Tx, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		revertirTransaccion(tx)
		return nil, err
	}
	return tx, nil
}

func normalizarErrorCandidatura(causa error) error {
	if errors.Is(causa, ports.ErrPreparacionAltaInvalida) {
		return ports.ErrPreparacionAltaInvalida
	}
	var postgres *pgconn.PgError
	if errors.As(causa, &postgres) && postgres.Code == "23505" {
		// Los conflictos semánticos los emite el resolutor sin nombre de
		// constraint. Una UNIQUE aleatoria nunca acredita idempotencia.
		if postgres.ConstraintName == "" {
			return ports.ErrClaveIdempotenciaUsada
		}
		return ports.ErrPersistenciaNoDisponible
	}
	return ports.ErrPersistenciaNoDisponible
}
