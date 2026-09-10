package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"time"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

type ResolverRaizIncorporacionV2 interface {
	ResolverSeguimientoIncorporacionV2(context.Context, ct.OrdenConfirmacionIncorporacionV2) (string, error)
}
type RestauradorOriginalIncorporacionV2 interface {
	Restaurar(context.Context, hist.Selector) (hist.Restauracion, error)
}
type iniciadorRegistroIncorporacionV2 interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type TransaccionRegistroIncorporacionV2PostgreSQL struct {
	pool        iniciadorRegistroIncorporacionV2
	proveedor   lector.ProveedorV2
	raices      ResolverRaizIncorporacionV2
	restaurador RestauradorOriginalIncorporacionV2
	reloj       ct.Reloj
}

var _ ct.TransaccionRegistroIncorporacionV2 = (*TransaccionRegistroIncorporacionV2PostgreSQL)(nil)

func NuevaTransaccionRegistroIncorporacionV2PostgreSQL(p *pgxpool.Pool, l lector.ProveedorV2, r ResolverRaizIncorporacionV2, h RestauradorOriginalIncorporacionV2, c ct.Reloj) (*TransaccionRegistroIncorporacionV2PostgreSQL, error) {
	return nuevaTransaccionRegistroIncorporacionV2PostgreSQL(p, l, r, h, c)
}
func nuloRegistroTX(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}
func nuevaTransaccionRegistroIncorporacionV2PostgreSQL(p iniciadorRegistroIncorporacionV2, l lector.ProveedorV2, r ResolverRaizIncorporacionV2, h RestauradorOriginalIncorporacionV2, c ct.Reloj) (*TransaccionRegistroIncorporacionV2PostgreSQL, error) {
	if nuloRegistroTX(p) || nuloRegistroTX(l) || nuloRegistroTX(r) || nuloRegistroTX(h) || nuloRegistroTX(c) {
		return nil, ct.ErrRegistroIncorporacionV2
	}
	return &TransaccionRegistroIncorporacionV2PostgreSQL{p, l, r, h, c}, nil
}
func errorRegistroTX(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ct.ErrRegistroIncorporacionV2
}

// Cada operación retiene su propia TX/resultado. El consumidor privado valida
// Personal ANTES del commit; ningún DTO intermedio sale del método exterior.
type operacionRegistroTX struct {
	adapter     *TransaccionRegistroIncorporacionV2PostgreSQL
	ctx         context.Context
	actual      ct.OrdenConfirmacionIncorporacionV2
	lectura     *lector.OrdenV2
	ultimo      time.Time
	relojErr    error
	raiz        string
	raizInicial PreparacionRaizIncorporacionV2
	tx          pgx.Tx
	confirmado  bool
	resultado   ct.ResultadoRegistroIncorporacionV2
	invocada    bool
}

func (o *operacionRegistroTX) Ahora() time.Time {
	if o.ctx.Err() != nil {
		o.relojErr = o.ctx.Err()
		return time.Time{}
	}
	t := o.adapter.reloj.Ahora()
	if o.ctx.Err() != nil {
		o.relojErr = o.ctx.Err()
		return time.Time{}
	}
	if !dom.InstanteUTCCanonico(t) || t.Before(o.ultimo) {
		o.relojErr = ct.ErrRegistroIncorporacionV2
		return time.Time{}
	}
	o.ultimo = t
	return t
}
func (o *operacionRegistroTX) validar() error {
	if o.relojErr != nil {
		return errorRegistroTX(o.ctx)
	}
	t := o.Ahora()
	if o.relojErr != nil || o.actual.ValidarEn(t) != nil {
		return errorRegistroTX(o.ctx)
	}
	if o.lectura != nil && o.lectura.ValidarEn(t) != nil {
		return errorRegistroTX(o.ctx)
	}
	if o.ctx.Err() != nil {
		return o.ctx.Err()
	}
	return nil
}
func (o *operacionRegistroTX) cerrar() {
	if !nuloRegistroTX(o.tx) && !o.confirmado {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = o.tx.Rollback(c)
	}
}
func (a *TransaccionRegistroIncorporacionV2PostgreSQL) RegistrarORecuperarIncorporacion(ctx context.Context, actual ct.OrdenConfirmacionIncorporacionV2, previa ct.AcreditacionPersonalIncorporacion) (ct.ResultadoRegistroIncorporacionV2, error) {
	var cero ct.ResultadoRegistroIncorporacionV2
	if ctx == nil || a == nil || nuloRegistroTX(a.pool) || nuloRegistroTX(a.proveedor) || nuloRegistroTX(a.raices) || nuloRegistroTX(a.restaurador) || nuloRegistroTX(a.reloj) {
		return cero, errorRegistroTX(ctx)
	}
	op := &operacionRegistroTX{adapter: a, ctx: ctx, actual: actual}
	defer op.cerrar()
	if e := op.validar(); e != nil {
		return cero, e
	}
	personal, e := previa.RegistroPara(actual, op.ultimo)
	if e != nil {
		return cero, errorRegistroTX(ctx)
	}
	if e = op.validar(); e != nil {
		return cero, e
	}
	if inicial, ok := a.raices.(ResolverRaizInicialIncorporacionV2); ok {
		op.raizInicial, e = inicial.ResolverRaizInicialIncorporacionV2(ctx, actual)
		op.raiz = op.raizInicial.seguimiento
	} else {
		op.raiz, e = a.raices.ResolverSeguimientoIncorporacionV2(ctx, actual)
	}
	if e != nil {
		return cero, errorRegistroTX(ctx)
	}
	if e = op.validar(); e != nil {
		return cero, e
	}
	if !refSeguimientoRegistroV2(op.raiz) {
		return cero, errorRegistroTX(ctx)
	}
	d, e := actual.Material().Datos()
	if e != nil {
		return cero, errorRegistroTX(ctx)
	}
	s := lector.Selector{OrganizacionRef: d.Preparacion.OrganizacionRef, SolicitudRef: personal.Solicitud.SolicitudRef, ExpedienteRef: personal.Solicitud.ExpedienteRef, VersionExpediente: personal.Solicitud.VersionExpediente, ResultadoRef: personal.Resultado.ResultadoRef, ReciboRef: personal.Resultado.ReciboRef, RelacionRef: personal.Resultado.RelacionRef, OcupacionRef: personal.Resultado.OcupacionRef, MaterialSHA256: personal.MaterialSHA256}
	c, e := lector.NuevoV2(a.proveedor, op, op)
	if e != nil {
		return cero, errorRegistroTX(ctx)
	}
	_, e = c.Leer(ctx, s, d.Preparacion.UnidadRef, d.Contexto)
	if e != nil {
		return cero, errorRegistroTX(ctx)
	}
	if e = op.validar(); e != nil {
		return cero, e
	}
	if !op.invocada || nuloRegistroTX(op.tx) || op.resultado.ValidarPara(actual, op.ultimo) != nil {
		return cero, errorRegistroTX(ctx)
	}
	salida := op.resultado.Copia()
	if e = op.validar(); e != nil {
		return cero, e
	}
	if e = op.tx.Commit(ctx); e != nil {
		return cero, errorRegistroTX(ctx)
	}
	op.confirmado = true
	if e = op.validar(); e != nil {
		return cero, e
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	return salida.Copia(), nil
}

const consultaRegistroIncorporacionTXV2 = `SELECT vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2($1::jsonb,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12,$13,$14,$15,$16,$17::bigint,$18::bigint,$19,$20,$21,$22,$23::jsonb)::text`
const ajustesRegistroIncorporacionTXV2 = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

func (o *operacionRegistroTX) LeerRegistroPersonalV2(ctx context.Context, s lector.Selector, l lector.OrdenV2) (lector.Resultado, error) {
	var cero lector.Resultado
	if o.invocada {
		return cero, errorRegistroTX(ctx)
	}
	o.invocada = true
	o.lectura = &l
	if e := o.validar(); e != nil {
		return cero, e
	}
	if s != l.Selector() {
		return cero, errorRegistroTX(ctx)
	}
	llamada, e := PrepararLlamadaRegistroIncorporacionV2(ctx, o.actual, l, o.raiz, o.ultimo)
	if e != nil {
		return cero, errorRegistroTX(ctx)
	}
	params := llamada.Parametros()
	defer func() {
		for _, v := range params {
			if b, ok := v.([]byte); ok {
				clear(b)
			}
		}
	}()
	if len(o.raizInicial.estado) != 0 {
		original, ok := params[22].([]byte)
		if !ok {
			return cero, errorRegistroTX(ctx)
		}
		params[22], e = o.raizInicial.envolverEvidencia(original)
		clear(original)
		if e != nil {
			return cero, errorRegistroTX(ctx)
		}
	}
	o.tx, e = o.adapter.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if e != nil || nuloRegistroTX(o.tx) {
		return cero, errorRegistroTX(ctx)
	}
	if e = o.validar(); e != nil {
		return cero, e
	}
	if _, e = o.tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); e != nil {
		return cero, errorRegistroTX(ctx)
	}
	if e = o.validar(); e != nil {
		return cero, e
	}
	var b []byte
	defer func() { clear(b) }()
	if e = o.tx.QueryRow(ctx, consultaRegistroIncorporacionTXV2, params...).Scan(&b); e != nil {
		return cero, errorRegistroTX(ctx)
	}
	if e = o.validar(); e != nil {
		return cero, e
	}
	var wire salidaTransporteRegistroV2
	if decodificarJSONRegistroV2(ctx, b, &wire) != nil {
		return cero, errorRegistroTX(ctx)
	}
	original := o.actual
	if wire.Recuperado {
		selector := hist.Selector{ReciboRef: wire.Recibo.Transicion.ReciboRef, MaterialSHA256: wire.Recibo.MaterialOriginalSHA256, IntencionSHA256: wire.Recibo.IntencionSHA256}
		restaurada, err := o.adapter.restaurador.Restaurar(ctx, selector)
		if err != nil {
			return cero, errorRegistroTX(ctx)
		}
		if e = o.validar(); e != nil {
			return cero, e
		}
		original = restaurada.OrdenOriginal
		pub, anterior, posterior := restaurada.Historia.EvidenciaSeguimiento()
		if !igualRegistroTX(restaurada.Historia.ReciboOriginal(), wire.Recibo) || !igualRegistroTX(pub, wire.Historia.Publicacion) || !igualRegistroTX(anterior, wire.Historia.Anterior) || !igualRegistroTX(posterior, wire.Historia.Posterior) {
			return cero, errorRegistroTX(ctx)
		}
	}
	r, e := DecodificarRegistroIncorporacionV2(ctx, b, o.actual, original, o.ultimo)
	if e != nil {
		return cero, errorRegistroTX(ctx)
	}
	c := r.ConsumosActuales
	x := l.Exportacion().ResumenCapacidad()
	if r.Recibo.SeguimientoRef != o.raiz || c.DecisionLecturaRef != x.DecisionRef() || c.LeidaPersonalEn.Before(l.EvaluadaEn()) || c.LeidaPersonalEn.Before(x.EmitidaEn()) || !c.LeidaPersonalEn.Before(x.ExpiraEn()) || l.ValidarEn(c.LeidaPersonalEn) != nil || o.actual.ValidarEn(c.ConsumidaCTEn) != nil {
		return cero, errorRegistroTX(ctx)
	}
	if e = o.validar(); e != nil {
		return cero, e
	}
	d, e := o.actual.Material().Datos()
	if e != nil {
		return cero, errorRegistroTX(ctx)
	}
	personal := d.Personal
	personal.MaterialCanonico = bytes.Clone(personal.MaterialCanonico)
	o.resultado = r.Copia()
	return lector.Resultado{Registro: personal, DecisionLecturaRef: c.DecisionLecturaRef, ConsumoHuellaSHA256: c.ConsumoLecturaSHA256, AuditoriaLecturaRef: c.AuditoriaPersonalLecturaRef, LeidaEn: c.LeidaPersonalEn}, nil
}
func igualRegistroTX(a, b any) bool {
	x, e := json.Marshal(a)
	y, f := json.Marshal(b)
	return e == nil && f == nil && bytes.Equal(x, y)
}
