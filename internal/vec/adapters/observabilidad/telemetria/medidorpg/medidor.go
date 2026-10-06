// Package medidorpg mide, por petición, cuántas consultas SQL hace VEC, cuánto
// tardan y cuánto se espera por una conexión libre. Es un trazador de pgx
// que se instala en la configuración del pool al componer la aplicación; la
// lógica de los módulos no cambia.
//
// Solo anota en la ficha de la petición (telemetria.Ficha) un nombre de
// operación sacado del texto SQL (esquema.función o tabla), el nombre del rol
// del pool, duraciones y la clase del error. Nunca guarda parámetros,
// literales, filas ni el texto del error.
package medidorpg

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria"
)

// Trazador implementa los trazadores de consulta, lote y copia de pgx y los
// de adquisición de conexión de pgxpool. No tiene estado por petición: lo
// que mide va a la ficha del contexto.
type Trazador struct {
	reloj    func() time.Time
	nombres  sync.Map // *pgxpool.Pool -> string
	metricas *metricasBD
}

// trazadorComun es el que instalan Instrumentar e InstrumentarConexion.
var trazadorComun = &Trazador{reloj: time.Now, metricas: metricasComunes}

// Instrumentar instala el trazador en la configuración de un pool antes de
// crearlo. No sustituye un trazador ya configurado: en ese caso devuelve
// false y el pool queda como estaba.
func Instrumentar(cfg *pgxpool.Config) bool {
	if cfg == nil || cfg.ConnConfig == nil || cfg.ConnConfig.Tracer != nil {
		return false
	}
	cfg.ConnConfig.Tracer = trazadorComun
	return true
}

// InstrumentarConexion hace lo mismo para una conexión suelta.
func InstrumentarConexion(cfg *pgx.ConnConfig) bool {
	if cfg == nil || cfg.Tracer != nil {
		return false
	}
	cfg.Tracer = trazadorComun
	return true
}

type claveInicio struct{}

type inicioMedida struct {
	ficha     *telemetria.Ficha
	inicio    time.Time
	operacion string
	consultas int
}

// empezar abre la medida de una consulta. Se mide siempre, para las
// métricas del proceso; la ficha solo existe dentro de una petición.
func (t *Trazador) empezar(ctx context.Context, sql string) context.Context {
	return t.empezarOperacion(ctx, operacionSQL(sql), 1)
}

func (t *Trazador) empezarOperacion(ctx context.Context, op string, n int) context.Context {
	f := telemetria.FichaDe(ctx)
	ahora := t.reloj()
	f.ConsultaIniciada(op, ahora)
	return context.WithValue(ctx, claveInicio{}, &inicioMedida{ficha: f, inicio: ahora, operacion: op, consultas: n})
}

func (t *Trazador) terminar(ctx context.Context, err error) {
	m, _ := ctx.Value(claveInicio{}).(*inicioMedida)
	if m == nil {
		return
	}
	d := t.reloj().Sub(m.inicio)
	clase := telemetria.ClasificarError(err)
	m.ficha.ConsultaTerminada(m.operacion, m.consultas, d, clase)
	t.metricas.observar(m.operacion, m.consultas, d, clase)
}

// TraceQueryStart cubre Query, QueryRow y Exec, también dentro de una
// transacción (BEGIN y COMMIT cuentan como consultas: son viajes a la base).
func (t *Trazador) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return t.empezar(ctx, data.SQL)
}

// TraceQueryEnd suma la duración y la clase de error.
func (t *Trazador) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	t.terminar(ctx, data.Err)
}

// TraceBatchStart abre la medida de un lote; cuenta como un viaje con tantas
// consultas como lleve.
func (t *Trazador) TraceBatchStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	op := "lote"
	n := 0
	if data.Batch != nil {
		n = data.Batch.Len()
		if n > 0 && len(data.Batch.QueuedQueries) > 0 && data.Batch.QueuedQueries[0] != nil {
			op = operacionSQL(data.Batch.QueuedQueries[0].SQL)
		}
	}
	return t.empezarOperacion(ctx, op, n)
}

// TraceBatchQuery no mide cada consulta del lote por separado.
func (t *Trazador) TraceBatchQuery(context.Context, *pgx.Conn, pgx.TraceBatchQueryData) {}

// TraceBatchEnd cierra la medida del lote.
func (t *Trazador) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	t.terminar(ctx, data.Err)
}

// TraceCopyFromStart mide una copia masiva con el nombre de la tabla.
func (t *Trazador) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	return t.empezarOperacion(ctx, "copy."+nombreCualificado(data.TableName), 1)
}

// TraceCopyFromEnd cierra la medida de la copia.
func (t *Trazador) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	t.terminar(ctx, data.Err)
}

type claveEspera struct{}

type inicioEspera struct {
	ficha  *telemetria.Ficha
	inicio time.Time
}

// TraceAcquireStart anota que la petición espera una conexión del pool. Si el
// pool está agotado, aquí es donde se va el tiempo.
func (t *Trazador) TraceAcquireStart(ctx context.Context, pool *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	nombre := t.nombrePool(pool)
	f := telemetria.FichaDe(ctx)
	if f == nil {
		return ctx
	}
	ahora := t.reloj()
	f.EsperaConexionIniciada(nombre, ahora)
	return context.WithValue(ctx, claveEspera{}, &inicioEspera{ficha: f, inicio: ahora})
}

// TraceAcquireEnd suma la espera.
func (t *Trazador) TraceAcquireEnd(ctx context.Context, _ *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	m, _ := ctx.Value(claveEspera{}).(*inicioEspera)
	if m == nil {
		return
	}
	m.ficha.EsperaConexionTerminada(t.reloj().Sub(m.inicio), telemetria.ClasificarError(data.Err))
}

// nombrePool es el rol de base de datos del pool (por ejemplo
// vec_bolsa_ejecutor). Se calcula una vez por pool: Config() copia la
// configuración entera.
func (t *Trazador) nombrePool(pool *pgxpool.Pool) string {
	if pool == nil {
		return "pool"
	}
	if v, ok := t.nombres.Load(pool); ok {
		return v.(string)
	}
	nombre := "pool"
	if c := pool.Config(); c != nil && c.ConnConfig != nil {
		if n := identificadorSQL(c.ConnConfig.User); n != "" {
			nombre = n
		}
	}
	t.nombres.Store(pool, nombre)
	t.metricas.registrarPool(pool, nombre)
	return nombre
}

var (
	_ pgx.QueryTracer       = (*Trazador)(nil)
	_ pgx.BatchTracer       = (*Trazador)(nil)
	_ pgx.CopyFromTracer    = (*Trazador)(nil)
	_ pgxpool.AcquireTracer = (*Trazador)(nil)
)
