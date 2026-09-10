package postgres

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

const consultaLecturaIncorporacionV2 = `SELECT vec_personal.acreditar_alta_ejercicio_v2($1,$2,$3,$4::bigint,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::bigint,$16::bigint,$17,$18,$19,$20)::text`

// TransaccionLecturaIncorporacionV2PostgreSQL consume permiso lector y audita en
// la fachada propietaria. RW es necesario para esos dos efectos, nunca un alta.
// La composición del pool/rol y la instalación SQL se acreditan por separado.
type TransaccionLecturaIncorporacionV2PostgreSQL struct {
	pool  iniciadorTransaccionAlta
	reloj lector.Reloj
}

var _ lector.TransaccionLecturaV2 = (*TransaccionLecturaIncorporacionV2PostgreSQL)(nil)

func NuevaTransaccionLecturaIncorporacionV2PostgreSQL(pool *pgxpool.Pool, reloj lector.Reloj) (*TransaccionLecturaIncorporacionV2PostgreSQL, error) {
	return nuevaTransaccionLecturaIncorporacionV2PostgreSQL(pool, reloj)
}

func nuevaTransaccionLecturaIncorporacionV2PostgreSQL(pool iniciadorTransaccionAlta, reloj lector.Reloj) (*TransaccionLecturaIncorporacionV2PostgreSQL, error) {
	if dependenciaNulaAlta(pool) || dependenciaNulaAlta(reloj) {
		return nil, lector.ErrNoDisponible
	}
	return &TransaccionLecturaIncorporacionV2PostgreSQL{pool: pool, reloj: reloj}, nil
}

// LeerRegistroPersonalV2 no reintenta un commit incierto. Toda salida fallida es
// cero; un commit confirmado jamás se revierte, incluso si llega cancelación.
func (a *TransaccionLecturaIncorporacionV2PostgreSQL) LeerRegistroPersonalV2(ctx context.Context, s lector.Selector, o lector.OrdenV2) (lector.Resultado, error) {
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
		if err := ctx.Err(); err != nil {
			return err
		}
		if s != o.Selector() || ahora.Before(ultimo) || o.ValidarEn(ahora) != nil {
			return lector.ErrDenegada
		}
		ultimo = ahora
		return ctx.Err()
	}
	if err := validar(); err != nil {
		return cero, err
	}
	parametros, piezas, err := parametrosLecturaIncorporacionV2(s, o)
	if err != nil {
		return cero, lector.ErrDenegada
	}
	defer borrarPiezasAlta(piezas[:])
	if err = validar(); err != nil {
		return cero, err
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		if !dependenciaNulaAlta(tx) {
			revertirTransaccionAlta(tx)
		}
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
	err = tx.QueryRow(ctx, consultaLecturaIncorporacionV2, parametros...).Scan(&salida)
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
	if !wireLecturaV2ParaOrden(r, o, ultimo) {
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
func parametrosLecturaIncorporacionV2(s lector.Selector, o lector.OrdenV2) ([]any, [8][]byte, error) {
	x := o.Exportacion()
	var p [8][]byte
	if s != o.Selector() || s.VersionExpediente == 0 || s.VersionExpediente > 9_007_199_254_740_991 ||
		x.ValidarEstructura() != nil || x.PersonaVersion() > math.MaxInt64 || x.PerfilVersion() > math.MaxInt64 {
		return nil, p, lector.ErrDenegada
	}
	// Los getters V3 ya devuelven copias; no crear una segunda copia sensible
	// temporal que quedaría sin limpiar.
	p = [8][]byte{x.CapacidadCanonica(), x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI()}
	return []any{s.OrganizacionRef, s.SolicitudRef, s.ExpedienteRef, int64(s.VersionExpediente), s.ResultadoRef, s.ReciboRef, s.RelacionRef, s.OcupacionRef, s.MaterialSHA256, o.UnidadRef(),
		p[0], p[1], p[2], p[3], int64(x.PersonaVersion()), int64(x.PerfilVersion()), p[4], p[5], p[6], p[7]}, p, nil
}

// Guardas wire previas al commit; el consumidor conserva el único validador
// semántico del canon original de Personal, y SQL acredita el registro durable.
func wireLecturaV2ParaOrden(r lector.Resultado, o lector.OrdenV2, ahora time.Time) bool {
	s, x := o.Selector(), o.Exportacion().ResumenCapacidad()
	g := r.Registro
	return g.ValidarEstructuraPara(g.Solicitud, ahora) == nil &&
		g.Solicitud.SolicitudRef == s.SolicitudRef && g.Solicitud.ExpedienteRef == s.ExpedienteRef && g.Solicitud.VersionExpediente == s.VersionExpediente &&
		g.Resultado.ResultadoRef == s.ResultadoRef && g.Resultado.ReciboRef == s.ReciboRef && g.Resultado.RelacionRef == s.RelacionRef && g.Resultado.OcupacionRef == s.OcupacionRef && g.MaterialSHA256 == s.MaterialSHA256 &&
		r.DecisionLecturaRef == x.DecisionRef() && g.DecisionOriginalRef != r.DecisionLecturaRef && !g.RegistradoEn.After(o.Material().PreparadoEn()) &&
		!r.LeidaEn.Before(o.EvaluadaEn()) && !r.LeidaEn.Before(x.EmitidaEn()) && r.LeidaEn.Before(x.ExpiraEn()) && !r.LeidaEn.After(ahora)
}
