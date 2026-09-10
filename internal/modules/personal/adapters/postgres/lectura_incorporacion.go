package postgres

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

const consultaLecturaIncorporacion = `SELECT vec_personal.acreditar_alta_ejercicio_v1($1,$2,$3,$4::bigint,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::bigint,$15::bigint,$16,$17,$18,$19)::text`
const ajustesLecturaIncorporacion = `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`

// TransaccionLecturaIncorporacionPostgreSQL consume permiso lector y audita en
// la fachada propietaria. RW es necesario para esos dos efectos, nunca un alta.
// La composición del pool/rol y la instalación SQL se acreditan por separado.
type TransaccionLecturaIncorporacionPostgreSQL struct {
	pool  iniciadorTransaccionAlta
	reloj lector.Reloj
}

var _ lector.TransaccionLectura = (*TransaccionLecturaIncorporacionPostgreSQL)(nil)

func NuevaTransaccionLecturaIncorporacionPostgreSQL(pool *pgxpool.Pool, reloj lector.Reloj) (*TransaccionLecturaIncorporacionPostgreSQL, error) {
	return nuevaTransaccionLecturaIncorporacionPostgreSQL(pool, reloj)
}

func nuevaTransaccionLecturaIncorporacionPostgreSQL(pool iniciadorTransaccionAlta, reloj lector.Reloj) (*TransaccionLecturaIncorporacionPostgreSQL, error) {
	if dependenciaNulaAlta(pool) || dependenciaNulaAlta(reloj) {
		return nil, lector.ErrNoDisponible
	}
	return &TransaccionLecturaIncorporacionPostgreSQL{pool: pool, reloj: reloj}, nil
}

// LeerRegistroPersonal no reintenta un commit incierto. Toda salida fallida es
// cero; un commit confirmado jamás se revierte, incluso si llega cancelación.
func (a *TransaccionLecturaIncorporacionPostgreSQL) LeerRegistroPersonal(ctx context.Context, s lector.Selector, o lector.Orden) (lector.Resultado, error) {
	var cero lector.Resultado
	if a == nil || ctx == nil || dependenciaNulaAlta(a.pool) || dependenciaNulaAlta(a.reloj) {
		return cero, lector.ErrNoDisponible
	}
	ultimo := o.EvaluadaEn()
	validar := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ahora := a.reloj.Ahora()
		if s != o.Selector() || ahora.Before(ultimo) || o.ValidarEn(ahora) != nil {
			return lector.ErrDenegada
		}
		ultimo = ahora
		return nil
	}
	if err := validar(); err != nil {
		return cero, err
	}
	parametros, piezas, err := parametrosLecturaIncorporacion(s, o)
	if err != nil {
		return cero, lector.ErrDenegada
	}
	defer borrarPiezasAlta(piezas[:])
	if err = validar(); err != nil {
		return cero, err
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, errorLecturaIncorporacion(ctx, err)
	}
	if dependenciaNulaAlta(tx) {
		return cero, lector.ErrNoDisponible
	}
	confirmada := false
	defer func() {
		if !confirmada {
			revertirTransaccionAlta(tx)
		}
	}()
	if err = validar(); err != nil {
		return cero, err
	}
	if _, err = tx.Exec(ctx, ajustesLecturaIncorporacion); err != nil {
		return cero, errorLecturaIncorporacion(ctx, err)
	}
	if err = validar(); err != nil {
		return cero, err
	}
	var salida []byte
	defer func() { borrarBytesAlta(salida) }()
	err = tx.QueryRow(ctx, consultaLecturaIncorporacion, parametros...).Scan(&salida)
	if err != nil {
		return cero, errorLecturaIncorporacion(ctx, err)
	}
	if err = validar(); err != nil {
		return cero, err
	}
	r, err := decodificarLecturaIncorporacion(salida)
	if err != nil {
		return cero, errorLecturaIncorporacion(ctx, lector.ErrNoDisponible)
	}
	entregada := false
	defer func() {
		if !entregada {
			borrarBytesAlta(r.Registro.MaterialCanonico)
		}
	}()
	if err = validar(); err != nil {
		return cero, err
	}
	if !wireLecturaParaOrden(r, o, ultimo) {
		return cero, errorLecturaIncorporacion(ctx, lector.ErrNoDisponible)
	}
	if err = validar(); err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		e := errorLecturaIncorporacion(ctx, err)
		if e == lector.ErrDenegada {
			e = lector.ErrNoDisponible
		}
		return cero, e
	}
	confirmada = true
	if err = validar(); err != nil {
		return cero, err
	}
	entregada = true
	return r, nil
}

// ValidarEstructura impone los límites INDIVIDUALES del contrato V3, no una
// cota genérica de 64KiB. Las copias de transporte se limpian al salir.
func parametrosLecturaIncorporacion(s lector.Selector, o lector.Orden) ([]any, [8][]byte, error) {
	x := o.Exportacion()
	var p [8][]byte
	if s != o.Selector() || s.VersionExpediente == 0 || s.VersionExpediente > 9_007_199_254_740_991 ||
		x.ValidarEstructura() != nil || x.PersonaVersion() > math.MaxInt64 || x.PerfilVersion() > math.MaxInt64 {
		return nil, p, lector.ErrDenegada
	}
	// Los getters V3 ya devuelven copias; no crear una segunda copia sensible
	// temporal que quedaría sin limpiar.
	p = [8][]byte{x.CapacidadCanonica(), x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI()}
	return []any{s.OrganizacionRef, s.SolicitudRef, s.ExpedienteRef, int64(s.VersionExpediente), s.ResultadoRef, s.ReciboRef, s.RelacionRef, s.OcupacionRef, s.MaterialSHA256,
		p[0], p[1], p[2], p[3], int64(x.PersonaVersion()), int64(x.PerfilVersion()), p[4], p[5], p[6], p[7]}, p, nil
}

func errorLecturaIncorporacion(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var pg *pgconn.PgError
	// El puerto no expone conflicto: selector discrepante y acceso rechazado
	// cierran con Denegada. Ningún mensaje/Detail del servidor sale del adaptador.
	if errors.As(err, &pg) && (pg.Code == "42501" || pg.Code == "P1102") {
		return lector.ErrDenegada
	}
	return lector.ErrNoDisponible
}

// Guardas wire previas al commit; el consumidor conserva el único validador
// semántico del canon original de Personal, y SQL acredita el registro durable.
func wireLecturaParaOrden(r lector.Resultado, o lector.Orden, ahora time.Time) bool {
	s, x := o.Selector(), o.Exportacion().ResumenCapacidad()
	g := r.Registro
	return g.ValidarEstructuraPara(g.Solicitud, ahora) == nil &&
		g.Solicitud.SolicitudRef == s.SolicitudRef && g.Solicitud.ExpedienteRef == s.ExpedienteRef && g.Solicitud.VersionExpediente == s.VersionExpediente &&
		g.Resultado.ResultadoRef == s.ResultadoRef && g.Resultado.ReciboRef == s.ReciboRef && g.Resultado.RelacionRef == s.RelacionRef && g.Resultado.OcupacionRef == s.OcupacionRef && g.MaterialSHA256 == s.MaterialSHA256 &&
		r.DecisionLecturaRef == x.DecisionRef() && g.DecisionOriginalRef != r.DecisionLecturaRef && !g.RegistradoEn.After(o.Material().PreparadoEn()) &&
		!r.LeidaEn.Before(o.EvaluadaEn()) && !r.LeidaEn.Before(x.EmitidaEn()) && r.LeidaEn.Before(x.ExpiraEn()) && !r.LeidaEn.After(ahora)
}
