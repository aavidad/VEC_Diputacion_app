package telemetria

import (
	"context"
	"errors"
	"net"
	"regexp"
	"sort"
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
	consultas      atomic.Int64
	bd             atomic.Int64 // nanosegundos en consultas
	espera         atomic.Int64 // nanosegundos esperando conexión libre
	lotes          atomic.Int64
	desconocidas   atomic.Int64 // SQL sin alias positivo; solo se emite en diagnóstico
	resultadosLote atomic.Int64 // callbacks observados, no consultas enviadas
	erroresLote    atomic.Int64
	duracionLote   atomic.Int64 // hasta cerrar el lote; incluye consumo del cliente

	mu          sync.Mutex
	maxima      time.Duration
	opMaxima    string
	operaciones map[string]*resumenConsulta
	// ultimoErr es la clase del último error con la base de datos. Una
	// consulta de datos correcta posterior lo vacía, así que un error ya
	// manejado no se atribuye a un fallo posterior; las órdenes de control
	// (ROLLBACK, COMMIT…) y los préstamos correctos no lo tocan.
	ultimoErr string
}

const maxOperacionesPeticion = 16
const maxClasesErrorOperacion = 4

type resumenConsulta struct {
	n       int64
	total   time.Duration
	maxima  time.Duration
	errores map[string]int64
}

type operacionMedida struct {
	Nombre  string           `json:"nombre"`
	N       int64            `json:"n"`
	Total   float64          `json:"total"`
	Maxima  float64          `json:"maxima"`
	Errores map[string]int64 `json:"errores,omitempty"`
}

// resumenOperaciones solo se serializa al terminar una petición lenta o fallida.
// El número de nombres y clases por petición es fijo, también ante SQL dinámico.
func (m *medida) resumenOperaciones() []operacionMedida {
	m.mu.Lock()
	defer m.mu.Unlock()
	nombres := make([]string, 0, len(m.operaciones))
	for nombre := range m.operaciones {
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)
	salida := make([]operacionMedida, 0, len(nombres))
	for _, nombre := range nombres {
		r := m.operaciones[nombre]
		var errores map[string]int64
		if len(r.errores) > 0 {
			errores = make(map[string]int64, len(r.errores))
			for clase, n := range r.errores {
				errores[clase] = n
			}
		}
		salida = append(salida, operacionMedida{Nombre: nombre, N: r.n,
			Total: r.total.Seconds(), Maxima: r.maxima.Seconds(), Errores: errores})
	}
	return salida
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
	m            *medida
	t            time.Time
	sql          string
	n            int64
	lote         bool
	terminado    atomic.Bool
	errResultado atomic.Bool
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
	if i == nil || i.lote {
		return
	}
	d := time.Since(i.t)
	i.m.consultas.Add(i.n)
	i.m.bd.Add(int64(d))
	var nombre, clase string
	if i.n > 0 {
		var conocida bool
		nombre, conocida = clasificarOperacion(i.sql)
		if !conocida {
			i.m.desconocidas.Add(1)
		}
	}
	if err != nil {
		clase = claseError(err)
	}
	i.m.mu.Lock()
	if i.n > 0 {
		if d > i.m.maxima {
			i.m.maxima, i.m.opMaxima = d, nombre
		}
		if i.m.operaciones == nil {
			i.m.operaciones = make(map[string]*resumenConsulta)
		}
		r := i.m.operaciones[nombre]
		if r == nil {
			if len(i.m.operaciones) >= maxOperacionesPeticion {
				nombre = "otras"
			}
			r = i.m.operaciones[nombre]
			if r == nil {
				r = &resumenConsulta{}
				i.m.operaciones[nombre] = r
			}
		}
		r.n++
		r.total += d
		if d > r.maxima {
			r.maxima = d
		}
		if err != nil {
			if r.errores == nil {
				r.errores = make(map[string]int64)
			}
			if _, ok := r.errores[clase]; !ok && len(r.errores) >= maxClasesErrorOperacion {
				r.errores["otras"]++
			} else {
				r.errores[clase]++
			}
		}
	}
	if err != nil {
		i.m.ultimoErr = clase
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
	if d.Batch == nil || len(d.Batch.QueuedQueries) == 0 {
		return ctx
	}
	m := medidaDe(ctx)
	if m == nil {
		return ctx
	}
	return context.WithValue(ctx, claveInicio{}, &inicio{m: m, t: time.Now(), lote: true})
}

func (trazador) TraceBatchQuery(ctx context.Context, _ *pgx.Conn, d pgx.TraceBatchQueryData) {
	i, _ := ctx.Value(claveInicio{}).(*inicio)
	if i == nil || !i.lote || i.terminado.Load() {
		return
	}
	i.m.resultadosLote.Add(1)
	n := consultasDe(d.SQL)
	i.m.consultas.Add(n)
	i.m.mu.Lock()
	if d.Err != nil {
		i.errResultado.Store(true)
		i.m.erroresLote.Add(1)
		i.m.ultimoErr = claseError(d.Err)
	} else if n > 0 {
		i.m.ultimoErr = ""
	}
	i.m.mu.Unlock()
}

func (trazador) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, d pgx.TraceBatchEndData) {
	i, _ := ctx.Value(claveInicio{}).(*inicio)
	if i == nil || !i.lote || !i.terminado.CompareAndSwap(false, true) {
		return
	}
	duracion := time.Since(i.t)
	i.m.lotes.Add(1)
	i.m.bd.Add(int64(duracion))
	i.m.duracionLote.Add(int64(duracion))
	if d.Err != nil {
		if !i.errResultado.Load() {
			i.m.erroresLote.Add(1)
		}
		i.m.mu.Lock()
		i.m.ultimoErr = claseError(d.Err)
		i.m.mu.Unlock()
	}
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

// Alias de SQL literal identificado en los adaptadores. Un identificador no
// registrado puede contener material dinámico; solo se emite su verbo cerrado.
var aliasesOperacion = map[string]struct{}{
	"vec_usuarios.catalogo_vigente_preferencias_v1":               {},
	"vec_usuarios.consultar_preferencias_propias_v1":              {},
	"vec_usuarios.recuperar_preferencias_operacion_v1":            {},
	"vec_usuarios.guardar_preferencias_propias_v1":                {},
	"vec_usuarios.registrar_denegacion_preferencias_v1":           {},
	"vec_identidad_sesiones_v1.registrar_sesion_v1":               {},
	"vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1":    {},
	"vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1":     {},
	"vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1": {},
	"vec_identidad_externa_v1.registrar_sesion_v1":                {},
	"vec_identidad_externa_v1.reconciliar_registro_sesion_v1":     {},
	"vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1":      {},
	"vec_autorizacion_atestada_v3.leer_configuracion_interna_v2":  {},
	"vec_autorizacion_atestada_v3.leer_configuracion_externa_v1":  {},
}

var verbosOperacion = map[string]struct{}{
	"select": {}, "insert": {}, "update": {}, "delete": {}, "with": {}, "call": {},
	"begin": {}, "commit": {}, "rollback": {}, "set": {}, "copy": {},
	"create": {}, "alter": {}, "drop": {}, "grant": {}, "revoke": {},
	"do": {}, "explain": {}, "values": {},
}

// clasificarOperacion limita el análisis a 2048 bytes, elimina literales y
// comentarios, y solo acepta alias exactos aprobados en código. El indicador
// distingue las consultas cuyo nombre se ha reducido al verbo.
func clasificarOperacion(sql string) (string, bool) {
	if len(sql) > 2048 {
		sql = sql[:2048]
	}
	sql = literalSQL.ReplaceAllString(sql, " ")
	for _, re := range []*regexp.Regexp{funcionSQL, objetoSQL} {
		for _, m := range re.FindAllStringSubmatch(sql, -1) {
			alias := strings.ToLower(m[1])
			if _, ok := aliasesOperacion[alias]; ok {
				return alias, true
			}
		}
	}
	if m := primeraOrden.FindStringSubmatch(sql); m != nil {
		verbo := strings.ToLower(m[1])
		if _, ok := verbosOperacion[verbo]; ok {
			return verbo, false
		}
	}
	return "sql", false
}

func operacion(sql string) string {
	nombre, _ := clasificarOperacion(sql)
	return nombre
}
