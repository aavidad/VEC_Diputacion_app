package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaRegistrarDecisionContextoActorV3 = `
	SELECT concedida, codigo, decision_huella_sha256, registrada_en
	FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
		$1::bytea, $2::bytea, $3::numeric, $4::numeric
	)`

// RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente registra
// exclusivamente una concesion V3. La confirmacion nominal se construye en el
// puerto, despues de que este metodo haya confirmado la transaccion durable.
func (a *AlmacenAutorizacion) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context,
	orden ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if a == nil || valorNuloPostgreSQL(a.pool) || ctx == nil {
		return time.Time{}, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	datos, err := orden.Datos()
	if err != nil {
		return time.Time{}, err
	}
	return a.registrarDecisionContextoActorV3(
		ctx, datos, true, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
	)
}

// RegistrarDenegacionAutorizacionLigadaV3 usa la misma frontera SQL cerrada,
// pero conserva el puerto nominal de auditoria: nunca produce confirmacion ni
// convierte la denegacion en capacidad ejecutable.
func (a *AlmacenAutorizacion) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context,
	orden ports.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if a == nil || valorNuloPostgreSQL(a.pool) || ctx == nil {
		return ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	datos, err := orden.Datos()
	if err != nil {
		return err
	}
	_, err = a.registrarDecisionContextoActorV3(
		ctx, datos, false, ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible,
	)
	return err
}

func (a *AlmacenAutorizacion) registrarDecisionContextoActorV3(
	ctx context.Context,
	datos ports.DatosOrdenRegistroAutorizacionLigadaV3,
	concedidaEsperada bool,
	errorNoDisponible error,
) (time.Time, error) {
	decisionCanonica, motivoCanonico, huellaEsperada, emitidaEn, validaHasta, codigoEsperado, err :=
		serializarDecisionContextoActorV3PostgreSQL(datos, concedidaEsperada)
	if err != nil {
		return time.Time{}, errorNoDisponible
	}
	defer borrarBytesAutorizacionPostgreSQL(decisionCanonica, motivoCanonico)

	// Los uint64 viajan como decimal y PostgreSQL hace la conversion explicita
	// a numeric. No existe paso intermedio por int64 ni perdida sobre MaxUint64.
	personaVersion := strconv.FormatUint(datos.ResultadoContexto.Contexto.Instantanea.PersonaVersion, 10)
	perfilVersion := strconv.FormatUint(datos.ResultadoContexto.Contexto.Instantanea.PerfilVersion, 10)

	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil || valorNuloPostgreSQL(tx) {
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, err, errorNoDisponible)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionAutorizacion(ctx, tx); err != nil {
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, err, errorNoDisponible)
	}

	filas, err := tx.Query(
		ctx, consultaRegistrarDecisionContextoActorV3,
		decisionCanonica, motivoCanonico, personaVersion, perfilVersion,
	)
	if err != nil {
		if !valorNuloPostgreSQL(filas) {
			filas.Close()
		}
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, err, errorNoDisponible)
	}
	if valorNuloPostgreSQL(filas) {
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, err, errorNoDisponible)
	}
	var concedida bool
	var codigo, huella string
	var registradaEn time.Time
	if !filas.Next() {
		errFilas := filas.Err()
		filas.Close()
		if errFilas != nil {
			return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, errFilas, errorNoDisponible)
		}
		return time.Time{}, ports.ErrInstantaneaAutorizacionObsoleta
	}
	if err = filas.Scan(&concedida, &codigo, &huella, &registradaEn); err != nil {
		filas.Close()
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, err, errorNoDisponible)
	}
	// pgx puede materializar timestamptz en time.Local incluso usando el codec
	// binario. UTC conserva el instante y restablece la forma canonica del puerto.
	registradaEn = registradaEn.UTC()
	if filas.Next() {
		filas.Close()
		return time.Time{}, errorNoDisponible
	}
	errFilas := filas.Err()
	filas.Close()
	if errFilas != nil {
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, errFilas, errorNoDisponible)
	}
	if concedida != concedidaEsperada || codigo != codigoEsperado || huella != huellaEsperada ||
		!instanteRegistroContextoActorV3PostgreSQLValido(registradaEn) ||
		registradaEn.Before(emitidaEn) || !registradaEn.Before(validaHasta) {
		return time.Time{}, errorNoDisponible
	}

	// Ultima consulta del contexto antes de COMMIT. Tras intentar COMMIT no se
	// consulta ctx ni reloj: un error puede representar resultado ambiguo.
	if err = ctx.Err(); err != nil {
		return time.Time{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return time.Time{}, errorCommitRegistroAutorizacionLigadaV3(err, errorNoDisponible)
	}
	return registradaEn, nil
}

func serializarDecisionContextoActorV3PostgreSQL(
	datos ports.DatosOrdenRegistroAutorizacionLigadaV3,
	concedidaEsperada bool,
) (
	decisionCanonica, motivoCanonico []byte,
	huella string,
	emitidaEn, validaHasta time.Time,
	codigo string,
	err error,
) {
	if datos.ResultadoContexto.Validar() != nil ||
		datos.Decision.ValidarPara(datos.Solicitud) != nil {
		return nil, nil, "", time.Time{}, time.Time{}, "", ports.ErrOrdenRegistroAutorizacionLigadaV3Invalida
	}
	concedida, codigo, err := datos.Decision.Resultado()
	if err != nil || concedida != concedidaEsperada {
		return nil, nil, "", time.Time{}, time.Time{}, "", ports.ErrOrdenRegistroAutorizacionLigadaV3Invalida
	}
	emitidaEn, validaHasta, err = datos.Decision.VentanaValidez()
	if err != nil {
		return nil, nil, "", time.Time{}, time.Time{}, "", ports.ErrOrdenRegistroAutorizacionLigadaV3Invalida
	}
	decisionCanonica, err = domain.RepresentacionCanonicaDecisionAutorizacionV3(datos.Decision)
	if err != nil {
		return nil, nil, "", time.Time{}, time.Time{}, "", err
	}
	motivoCanonico, err = domain.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		borrarBytesAutorizacionPostgreSQL(decisionCanonica)
		return nil, nil, "", time.Time{}, time.Time{}, "", err
	}
	huella, err = domain.HuellaSHA256DecisionAutorizacionV3(datos.Decision)
	if err != nil {
		borrarBytesAutorizacionPostgreSQL(decisionCanonica, motivoCanonico)
		return nil, nil, "", time.Time{}, time.Time{}, "", err
	}
	return decisionCanonica, motivoCanonico, huella, emitidaEn, validaHasta, codigo, nil
}

func instanteRegistroContextoActorV3PostgreSQLValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

func errorRegistroAutorizacionLigadaV3(
	ctx context.Context,
	err error,
	errorNoDisponible error,
) error {
	traducido := errorRegistroAutorizacion(ctx, err)
	if errors.Is(traducido, ports.ErrRegistroDecisionNoDisponible) {
		return errorNoDisponible
	}
	return traducido
}

// No consulta ctx: despues de intentar COMMIT no puede distinguirse una
// cancelacion local de un COMMIT aplicado cuya respuesta se perdio.
func errorCommitRegistroAutorizacionLigadaV3(err, errorNoDisponible error) error {
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) {
		switch errorPG.Code {
		case "40001", "40P01":
			return ports.ErrInstantaneaAutorizacionObsoleta
		}
	}
	return errorNoDisponible
}

var _ ports.RegistroConcesionesCandidatasAutorizacionLigadaV3 = (*AlmacenAutorizacion)(nil)
var _ ports.RegistroDenegacionesAutorizacionLigadaV3 = (*AlmacenAutorizacion)(nil)
