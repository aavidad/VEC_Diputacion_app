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
	// ultimoErr es la clase del último error con la base de datos. Una
	// consulta de datos correcta posterior lo vacía, así que un error ya
	// manejado no se atribuye a un fallo posterior; las órdenes de control
	// (ROLLBACK, COMMIT…) y los préstamos correctos no lo tocan.
	ultimoErr string
}

type claveMedida struct{}

func medidaDe(ctx context.Context) *medida {
	m, _ := ctx.Value(claveMedida{}).(*medida)
	return m
}

// Instrumentar instala el trazador en la configuración de un pool antes de
// crearlo. No sustituye un trazador ya configurado. No debe llamarse en los
// pools acreditados que exigen no tener trazador (las fábricas O4-05 de
// Contratación temporal vuelven a comprobarlo en cada préstamo y dejarían de
// prestar conexiones).
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

// esControl reconoce BEGIN, COMMIT, ROLLBACK, SAVEPOINT, SET y similares, y
// SELECT set_config(…): son viajes a la base, pero no consultas de datos.
func esControl(sql string) bool {
	sql = strings.TrimLeft(sql, " \t\r\n")
	for _, orden := range []string{"begin", "start transaction", "commit", "end", "rollback", "abort",
		"savepoint", "release", "set", "reset", "discard"} {
		if len(sql) >= len(orden) && strings.EqualFold(sql[:len(orden)], orden) &&
			(len(sql) == len(orden) || !esLetraSQL(sql[len(orden)])) {
			return true
		}
	}
	if len(sql) < 6 || !strings.EqualFold(sql[:6], "select") {
		return false
	}
	resto := strings.TrimLeft(sql[6:], " \t\r\n")
	for _, funcion := range []string{"set_config(", "pg_catalog.set_config("} {
		if len(resto) >= len(funcion) && strings.EqualFold(resto[:len(funcion)], funcion) {
			return true
		}
	}
	return false
}

func esLetraSQL(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func consultasDe(sql string) int64 {
	if esControl(sql) {
		return 0
	}
	return 1
}

// empezar abre la medida de una operación con n consultas de datos (0 para
// las de control de transacción o sesión, que no cuentan).
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
	if i.n > 0 && d > i.m.maxima {
		i.m.maxima, i.m.sqlMaxima = d, i.sql
	}
	if err != nil {
		i.m.ultimoErr = claseError(err)
	} else if i.n > 0 {
		// Solo una consulta de datos correcta lo vacía: el ROLLBACK que sigue
		// a un fallo no debe borrar su causa.
		i.m.ultimoErr = ""
	}
	i.m.mu.Unlock()
}

func (trazador) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	return empezar(ctx, d.SQL, consultasDe(d.SQL))
}

func (trazador) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryEndData) {
	terminar(ctx, d.Err)
}

func (trazador) TraceBatchStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceBatchStartData) context.Context {
	if d.Batch == nil || len(d.Batch.QueuedQueries) == 0 || d.Batch.QueuedQueries[0] == nil {
		return ctx
	}
	var n int64
	for _, q := range d.Batch.QueuedQueries {
		if q != nil {
			n += consultasDe(q.SQL)
		}
	}
	return empezar(ctx, d.Batch.QueuedQueries[0].SQL, n)
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
