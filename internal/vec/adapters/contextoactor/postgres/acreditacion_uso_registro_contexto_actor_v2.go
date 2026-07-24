package postgres

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaAcreditarUsoRegistroContextoActorV2 = `
	SELECT vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
		$1::text, $2::text, $3::text, $4::text, $5::text,
		$6::text, $7::numeric, $8::text, $9::numeric,
		$10::text, $11::numeric, $12::text, $13::numeric,
		$14::text, $15::text, $16::timestamptz, $17::timestamptz
	)`

type consultorFilaAcreditacionUsoRegistroContextoActorV2 interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// AcreditadorUsoRegistroContextoActorPostgreSQLV2 queda ligado a una
// transaccion ya abierta. No posee pool ni capacidad para Begin, Commit o
// Rollback. La composicion futura debe construirlo con el mismo pgx.Tx
// SERIALIZABLE de escritura que registra la decision o el efecto y puede usar
// la misma instancia para las dos acreditaciones exigidas por el contrato SQL.
type AcreditadorUsoRegistroContextoActorPostgreSQLV2 struct {
	transaccion consultorFilaAcreditacionUsoRegistroContextoActorV2
}

func NuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(
	transaccion pgx.Tx,
) (*AcreditadorUsoRegistroContextoActorPostgreSQLV2, error) {
	return nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(transaccion)
}

func nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(
	transaccion consultorFilaAcreditacionUsoRegistroContextoActorV2,
) (*AcreditadorUsoRegistroContextoActorPostgreSQLV2, error) {
	if valorNuloAcreditacionUsoRegistroContextoActorV2(transaccion) {
		return nil, ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible
	}
	return &AcreditadorUsoRegistroContextoActorPostgreSQLV2{transaccion: transaccion}, nil
}

func (a *AcreditadorUsoRegistroContextoActorPostgreSQLV2) AcreditarUsoRegistroContextoActorV2(
	ctx context.Context,
	orden ports.OrdenAcreditacionUsoRegistroContextoActorV2,
) (time.Time, error) {
	if a == nil || valorNuloAcreditacionUsoRegistroContextoActorV2(a.transaccion) ||
		valorNuloAcreditacionUsoRegistroContextoActorV2(ctx) {
		return time.Time{}, errors.Join(
			ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada,
			ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible,
		)
	}
	if err := ctx.Err(); err != nil {
		return time.Time{}, errors.Join(ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada, err)
	}
	datos, err := orden.Datos()
	if err != nil {
		return time.Time{}, ports.ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida
	}
	resultado := datos.Resultado
	actor := resultado.Contexto
	instantanea := actor.Instantanea

	fila := a.transaccion.QueryRow(
		ctx,
		consultaAcreditarUsoRegistroContextoActorV2,
		resultado.RegistroContextoRef,
		domain.EsquemaRepresentacionContextoActorV2,
		resultado.HuellaSHA256,
		resultado.ManifiestoProcedenciaHuellaSHA256,
		string(resultado.AutoridadEfectiva),
		instantanea.CuentaRef,
		strconv.FormatUint(instantanea.CuentaVersion, 10),
		actor.PersonaRef,
		strconv.FormatUint(instantanea.PersonaVersion, 10),
		actor.PerfilActivoRef,
		strconv.FormatUint(instantanea.PerfilVersion, 10),
		instantanea.VinculoRef,
		strconv.FormatUint(instantanea.VinculoVersion, 10),
		string(actor.Principal.AuthMethod),
		string(actor.Principal.AuthAssurance),
		datos.EmitidaEn,
		datos.ValidaHasta,
	)
	if valorNuloAcreditacionUsoRegistroContextoActorV2(fila) {
		return time.Time{}, errorConsultaAcreditacionUsoRegistroContextoActorV2(ctx, nil)
	}
	var salida pgtype.Timestamptz
	if err = fila.Scan(&salida); err != nil {
		return time.Time{}, errorConsultaAcreditacionUsoRegistroContextoActorV2(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return time.Time{}, errors.Join(ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada, err)
	}
	if !salida.Valid {
		return time.Time{}, ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada
	}
	acreditadaEn := salida.Time.UTC()
	if !instanteAcreditacionUsoRegistroContextoActorV2PostgreSQLValido(acreditadaEn) ||
		acreditadaEn.Before(datos.EmitidaEn) || !acreditadaEn.Before(datos.ValidaHasta) {
		return time.Time{}, errors.Join(
			ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada,
			ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible,
		)
	}
	return acreditadaEn, nil
}

func instanteAcreditacionUsoRegistroContextoActorV2PostgreSQLValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

func errorConsultaAcreditacionUsoRegistroContextoActorV2(ctx context.Context, causa error) error {
	if !valorNuloAcreditacionUsoRegistroContextoActorV2(ctx) {
		if err := ctx.Err(); err != nil {
			return errors.Join(ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada, err)
		}
	}
	var errContexto error
	switch {
	case errors.Is(causa, context.Canceled):
		errContexto = context.Canceled
	case errors.Is(causa, context.DeadlineExceeded):
		errContexto = context.DeadlineExceeded
	}
	return errors.Join(
		ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada,
		ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible,
		errContexto,
	)
}

func valorNuloAcreditacionUsoRegistroContextoActorV2(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

var _ ports.AcreditadorUsoRegistroContextoActorV2 = (*AcreditadorUsoRegistroContextoActorPostgreSQLV2)(nil)
