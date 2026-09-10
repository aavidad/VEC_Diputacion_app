package postgres

import (
	"bytes"
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaLecturaConcesionHistoricaV3 = `
 SELECT concedida, codigo, decision_huella_sha256, registrada_en
 FROM vec_autorizacion.leer_concesion_historica_contexto_actor_v3(
  $1::bytea, $2::bytea, $3::numeric, $4::numeric
 )`

// LeerConcesionHistoricaAutorizacionLigadaV3 relee una concesión original por
// la fachada propietaria. La composición usa el pool de registro autorizado,
// nunca un pool CT ni acceso directo a tablas. No registra ni renueva permisos.
// La fecha se valida en la ventana ORIGINAL: esta lectura no acredita vigencia
// actual. Sólo devuelve datos tras confirmar su transacción de lectura.
func (a *AlmacenAutorizacion) LeerConcesionHistoricaAutorizacionLigadaV3(ctx context.Context, orden ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (fecha time.Time, err error) {
	defer func() {
		if ctx != nil && ctx.Err() != nil {
			fecha, err = time.Time{}, ctx.Err()
		} else if err != nil {
			fecha, err = time.Time{}, ports.ErrLecturaConcesionHistoricaV3
		}
	}()
	if ctx == nil || a == nil || valorNuloPostgreSQL(a.pool) {
		return time.Time{}, ports.ErrLecturaConcesionHistoricaV3
	}
	if err = ctx.Err(); err != nil {
		return time.Time{}, err
	}
	datos, err := orden.Datos()
	if err != nil {
		return time.Time{}, err
	}
	decision, motivo, huella, desde, hasta, codigo, err := serializarDecisionContextoActorV3PostgreSQL(datos, true)
	if err != nil {
		return time.Time{}, err
	}
	defer borrarBytesAutorizacionPostgreSQL(decision, motivo)
	pv := strconv.FormatUint(datos.ResultadoContexto.Contexto.Instantanea.PersonaVersion, 10)
	fv := strconv.FormatUint(datos.ResultadoContexto.Contexto.Instantanea.PerfilVersion, 10)
	// Volver a obtener datos revalida la candidata nominal; comparar la preimagen
	// evita mezclar material entre fronteras. Las copias no se comparten con pgx.
	validar := func() bool {
		d, e := orden.Datos()
		if e != nil {
			return false
		}
		dc, mc, h, di, df, c, e := serializarDecisionContextoActorV3PostgreSQL(d, true)
		defer borrarBytesAutorizacionPostgreSQL(dc, mc)
		return e == nil && bytes.Equal(dc, decision) && bytes.Equal(mc, motivo) && h == huella && di.Equal(desde) && df.Equal(hasta) && c == codigo &&
			strconv.FormatUint(d.ResultadoContexto.Contexto.Instantanea.PersonaVersion, 10) == pv && strconv.FormatUint(d.ResultadoContexto.Contexto.Instantanea.PerfilVersion, 10) == fv
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if !valorNuloPostgreSQL(tx) {
		confirmado := false
		defer func() {
			if !confirmado {
				revertirTransaccionPostgreSQL(tx)
			}
		}()
		if err != nil {
			return time.Time{}, err
		}
		if err = configurarTransaccionAutorizacion(ctx, tx); err != nil {
			return time.Time{}, err
		}
		if ctx.Err() != nil || !validar() {
			return time.Time{}, ports.ErrLecturaConcesionHistoricaV3
		}
		dc, mc := bytes.Clone(decision), bytes.Clone(motivo)
		defer borrarBytesAutorizacionPostgreSQL(dc, mc)
		filas, e := tx.Query(ctx, consultaLecturaConcesionHistoricaV3, dc, mc, pv, fv)
		if !valorNuloPostgreSQL(filas) {
			defer filas.Close()
		}
		if e != nil {
			return time.Time{}, e
		}
		if valorNuloPostgreSQL(filas) || !filas.Next() {
			return time.Time{}, ports.ErrLecturaConcesionHistoricaV3
		}
		var concedida bool
		var c, h string
		if e = filas.Scan(&concedida, &c, &h, &fecha); e != nil {
			return time.Time{}, e
		}
		if filas.Next() || filas.Err() != nil {
			return time.Time{}, ports.ErrLecturaConcesionHistoricaV3
		}
		filas.Close()
		// El codec pgx puede entregar Location local: UTC conserva el instante, no
		// redondea precisión ni refecha historia.
		fecha = fecha.UTC()
		if !concedida || c != codigo || h != huella || !instanteRegistroContextoActorV3PostgreSQLValido(fecha) || fecha.Before(desde) || !fecha.Before(hasta) || !validar() || ctx.Err() != nil {
			return time.Time{}, ports.ErrLecturaConcesionHistoricaV3
		}
		if err = tx.Commit(ctx); err != nil {
			return time.Time{}, err
		}
		confirmado = true
		if !validar() {
			return time.Time{}, ports.ErrLecturaConcesionHistoricaV3
		}
		return fecha, nil
	}
	return time.Time{}, ports.ErrLecturaConcesionHistoricaV3
}

var _ ports.LectorConcesionHistoricaAutorizacionLigadaV3 = (*AlmacenAutorizacion)(nil)
