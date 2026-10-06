package telemetria

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Medida de base de datos de una petición. La rellena el trazador de pgx
// que se instala en cada pool con Instrumentar; la lógica de los módulos no
// cambia, solo hace falta que la consulta use el contexto de la petición.
type medida struct {
	consultas atomic.Int64
	bd        atomic.Int64 // nanosegundos en consultas
	espera    atomic.Int64 // nanosegundos esperando conexión libre

	mu        sync.Mutex
	maxima    time.Duration
	sqlMaxima string // solo se analiza si la petición resulta lenta
	ultimoErr string
}

type claveMedida struct{}

func medidaDe(ctx context.Context) *medida {
	m, _ := ctx.Value(claveMedida{}).(*medida)
	return m
}

// Instrumentar instala el trazador en la configuración de un pool antes de
// crearlo. No sustituye otro trazador: un pool acreditado que los rechaza
// queda como estaba.
func Instrumentar(cfg *pgxpool.Config) {
	if cfg != nil && cfg.ConnConfig != nil && cfg.ConnConfig.Tracer == nil {
		cfg.ConnConfig.Tracer = trazador{}
	}
}

// trazador implementa los trazadores de consulta y lote de pgx y el de
// préstamo de conexión de pgxpool.
type trazador struct{}

type inicio struct {
	m   *medida
	t   time.Time
	sql string
	n   int64
}

type claveInicio struct{}

func empezar(ctx context.Context, sql string, n int64) context.Context {
	m := medidaDe(ctx)
	if m == nil {
		return ctx
	}
	return context.WithValue(ctx, claveInicio{}, &inicio{m: m, t: time.Now(), sql: sql, n: n})
}

func terminar(ctx context.Context, err error) {
	i, _ := ctx.Value(claveInicio{}).(*inicio)
	if i == nil {
		return
	}
	d := time.Since(i.t)
	i.m.consultas.Add(i.n)
	i.m.bd.Add(int64(d))
	i.m.mu.Lock()
	if d > i.m.maxima {
		i.m.maxima, i.m.sqlMaxima = d, i.sql
	}
	if err != nil {
		i.m.ultimoErr = claseError(err)
	}
	i.m.mu.Unlock()
}

func (trazador) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	return empezar(ctx, d.SQL, 1)
}

func (trazador) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryEndData) {
	terminar(ctx, d.Err)
}

func (trazador) TraceBatchStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceBatchStartData) context.Context {
	if d.Batch == nil || len(d.Batch.QueuedQueries) == 0 || d.Batch.QueuedQueries[0] == nil {
		return ctx
	}
	return empezar(ctx, d.Batch.QueuedQueries[0].SQL, int64(len(d.Batch.QueuedQueries)))
}

func (trazador) TraceBatchQuery(context.Context, *pgx.Conn, pgx.TraceBatchQueryData) {}

func (trazador) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, d pgx.TraceBatchEndData) {
	terminar(ctx, d.Err)
}

type claveEspera struct{}

func (trazador) TraceAcquireStart(ctx context.Context, pool *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	registrarPool(pool)
	if medidaDe(ctx) == nil {
		return ctx
	}
	return context.WithValue(ctx, claveEspera{}, time.Now())
}

func (trazador) TraceAcquireEnd(ctx context.Context, _ *pgxpool.Pool, d pgxpool.TraceAcquireEndData) {
	t, ok := ctx.Value(claveEspera{}).(time.Time)
	m := medidaDe(ctx)
	if !ok || m == nil {
		return
	}
	m.espera.Add(int64(time.Since(t)))
	if d.Err != nil {
		m.mu.Lock()
		m.ultimoErr = "conexion_" + claseError(d.Err)
		m.mu.Unlock()
	}
}

// claseError reduce el error a una clase cerrada; nunca su texto, que puede
// llevar datos o rutas.
func claseError(err error) string {
	var pg interface{ SQLState() string }
	var red net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelada"
	case errors.Is(err, context.DeadlineExceeded):
		return "plazo_vencido"
	case errors.As(err, &pg) && len(pg.SQLState()) == 5 && strings.Trim(pg.SQLState(), "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ") == "":
		return "bd_" + pg.SQLState()
	case errors.As(err, &red):
		return "red"
	}
	return "otro"
}

var (
	literalSQL   = regexp.MustCompile(`'(?:[^']|'')*'|"(?:[^"]|"")*"|--[^\n]*|/\*(?s:.*?)\*/`)
	funcionSQL   = regexp.MustCompile(`(?i)\b([a-z_][a-z0-9_]*\.[a-z_][a-z0-9_]*)\s*\(`)
	objetoSQL    = regexp.MustCompile(`(?i)\b(?:from|into|update|join|call)\s+([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?)`)
	primeraOrden = regexp.MustCompile(`(?i)^\s*([a-z]+)`)
)

// operacion da un nombre sin valores para una consulta: la primera función
// cualificada (esquema.funcion), el objeto tras FROM/INTO/UPDATE/JOIN/CALL o
// la orden (begin, commit…). Antes quita literales y comentarios.
func operacion(sql string) string {
	if len(sql) > 2048 {
		sql = sql[:2048]
	}
	sql = literalSQL.ReplaceAllString(sql, " ")
	for _, re := range []*regexp.Regexp{funcionSQL, objetoSQL, primeraOrden} {
		if m := re.FindStringSubmatch(sql); m != nil && len(m[1]) <= 127 {
			return strings.ToLower(m[1])
		}
	}
	return "sql"
}
