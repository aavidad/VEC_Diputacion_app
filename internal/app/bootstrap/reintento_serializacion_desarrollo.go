package bootstrap

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
)

// La composición de desarrollo prepara y publica la instantánea de
// autorización en cada petición, y lee la configuración de confianza V3 en
// una transacción de gobierno. Varias peticiones simultáneas del mismo perfil
// (la página del centro pide a la vez contexto, bandeja e incorporaciones) se
// ordenan en el bloqueo consultivo del perfil; en SERIALIZABLE la instantánea
// se tomó antes de esperar y PostgreSQL aborta a las que llegan tarde con
// 40001. Esas transacciones se repiten enteras con la política única de
// internal/shared/postgresql; el aborto las dejó sin efecto.
const intentosPublicacionSerializableCTDesarrollo = postgresqlcomun.IntentosMaximosCarreraSerializable

// falloSerializacionPostgreSQLCTDesarrollo es el centinela de siempre marcado
// como pérdida de carrera (40001 o 40P01). errors.Is con el centinela sigue
// funcionando; solo el reintento distingue la marca.
type falloSerializacionPostgreSQLCTDesarrollo struct{}

func (falloSerializacionPostgreSQLCTDesarrollo) Error() string {
	return errPostgreSQLContratacionTemporalDesarrolloNoDisponible.Error()
}

func (falloSerializacionPostgreSQLCTDesarrollo) Unwrap() error {
	return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
}

// CarreraSerializable marca el fallo como repetible para la política común.
func (falloSerializacionPostgreSQLCTDesarrollo) CarreraSerializable() bool { return true }

func causaSerializacionPostgreSQLCTDesarrollo(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && (pg.Code == "40001" || pg.Code == "40P01")
}

func falloReintentableSerializacionCTDesarrollo(err error) bool {
	return postgresqlcomun.EsCarreraSerializable(err)
}

// reintentarSerializacionCTDesarrollo repite intento mientras pierda una
// carrera de serialización o un interbloqueo. Cada intento abre y cierra su
// propia transacción. Cualquier otro error se devuelve sin repetir.
func reintentarSerializacionCTDesarrollo(ctx context.Context, intento func() error) error {
	return postgresqlcomun.RepetirTrasCarreraSerializable(ctx, intento)
}
