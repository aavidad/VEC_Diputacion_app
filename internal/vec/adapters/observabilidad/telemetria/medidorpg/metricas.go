package medidorpg

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria"
)

// Métricas de base de datos del proceso: consultas por operación, errores
// por clase y estado de cada pool. Cuentan también lo que no viene de una
// petición HTTP (trabajos en segundo plano).

const (
	maxSeriesOperacion = 1024
	maxPools           = 256
	maxClasesError     = 128
)

type serieOperacion struct {
	consultas atomic.Uint64
	nanos     atomic.Uint64
}

type metricasBD struct {
	operaciones sync.Map // string -> *serieOperacion
	totalOps    atomic.Int64
	errores     sync.Map // string -> *atomic.Uint64
	totalErr    atomic.Int64
	pools       sync.Map // *pgxpool.Pool -> string
	totalPools  atomic.Int64
}

var metricasComunes = &metricasBD{}

func (m *metricasBD) observar(operacion string, n int, d time.Duration, claseError string) {
	if m == nil {
		return
	}
	v, ok := m.operaciones.Load(operacion)
	if !ok {
		if m.totalOps.Load() >= maxSeriesOperacion {
			operacion = "otras"
		}
		var cargado bool
		v, cargado = m.operaciones.LoadOrStore(operacion, &serieOperacion{})
		if !cargado {
			m.totalOps.Add(1)
		}
	}
	s := v.(*serieOperacion)
	s.consultas.Add(uint64(max(n, 1)))
	s.nanos.Add(uint64(max(d, 0)))
	if claseError != "" {
		c, ok := m.errores.Load(claseError)
		if !ok {
			if m.totalErr.Load() >= maxClasesError {
				claseError = "otras"
			}
			var cargado bool
			c, cargado = m.errores.LoadOrStore(claseError, new(atomic.Uint64))
			if !cargado {
				m.totalErr.Add(1)
			}
		}
		c.(*atomic.Uint64).Add(1)
	}
}

func (m *metricasBD) registrarPool(pool *pgxpool.Pool, nombre string) {
	if m == nil || pool == nil {
		return
	}
	if _, ok := m.pools.Load(pool); ok || m.totalPools.Load() >= maxPools {
		return
	}
	if _, cargado := m.pools.LoadOrStore(pool, nombre); !cargado {
		m.totalPools.Add(1)
	}
}

// EscribirMetricas escribe las métricas de base de datos del proceso en
// formato de texto de Prometheus.
func EscribirMetricas(w io.Writer, servicio string) {
	metricasComunes.escribir(w, servicio)
}

func (m *metricasBD) escribir(w io.Writer, servicio string) {
	b := &strings.Builder{}
	base := `servicio="` + telemetria.EscaparEtiqueta(servicio) + `"`

	type fila struct {
		nombre string
		s      *serieOperacion
	}
	var ops []fila
	m.operaciones.Range(func(k, v any) bool {
		ops = append(ops, fila{k.(string), v.(*serieOperacion)})
		return true
	})
	sort.Slice(ops, func(i, j int) bool { return ops[i].nombre < ops[j].nombre })
	fmt.Fprintf(b, "# HELP vec_bd_consultas_total Consultas SQL por operación (esquema.función o tabla).\n# TYPE vec_bd_consultas_total counter\n")
	for _, o := range ops {
		fmt.Fprintf(b, "vec_bd_consultas_total{%s,operacion=\"%s\"} %d\n", base, telemetria.EscaparEtiqueta(o.nombre), o.s.consultas.Load())
	}
	fmt.Fprintf(b, "# HELP vec_bd_consultas_segundos_total Tiempo de respuesta acumulado por operación.\n# TYPE vec_bd_consultas_segundos_total counter\n")
	for _, o := range ops {
		fmt.Fprintf(b, "vec_bd_consultas_segundos_total{%s,operacion=\"%s\"} %g\n", base, telemetria.EscaparEtiqueta(o.nombre), float64(o.s.nanos.Load())/1e9)
	}

	var errores []string
	m.errores.Range(func(k, _ any) bool { errores = append(errores, k.(string)); return true })
	sort.Strings(errores)
	fmt.Fprintf(b, "# HELP vec_bd_errores_total Errores de consulta por clase (bd_SQLSTATE, plazo_vencido…).\n# TYPE vec_bd_errores_total counter\n")
	for _, clase := range errores {
		c, _ := m.errores.Load(clase)
		fmt.Fprintf(b, "vec_bd_errores_total{%s,clase=\"%s\"} %d\n", base, telemetria.EscaparEtiqueta(clase), c.(*atomic.Uint64).Load())
	}

	type pool struct {
		nombre string
		est    *pgxpool.Stat
	}
	var pools []pool
	m.pools.Range(func(k, v any) bool {
		pools = append(pools, pool{v.(string), k.(*pgxpool.Pool).Stat()})
		return true
	})
	sort.Slice(pools, func(i, j int) bool { return pools[i].nombre < pools[j].nombre })
	medidas := []struct {
		nombre, ayuda, tipo string
		valor               func(*pgxpool.Stat) string
	}{
		{"vec_pool_conexiones_maximas", "Tamaño máximo del pool.", "gauge", func(s *pgxpool.Stat) string { return fmt.Sprint(s.MaxConns()) }},
		{"vec_pool_conexiones_total", "Conexiones abiertas.", "gauge", func(s *pgxpool.Stat) string { return fmt.Sprint(s.TotalConns()) }},
		{"vec_pool_conexiones_en_uso", "Conexiones prestadas ahora.", "gauge", func(s *pgxpool.Stat) string { return fmt.Sprint(s.AcquiredConns()) }},
		{"vec_pool_conexiones_libres", "Conexiones libres.", "gauge", func(s *pgxpool.Stat) string { return fmt.Sprint(s.IdleConns()) }},
		{"vec_pool_conexiones_abriendo", "Conexiones que se están abriendo.", "gauge", func(s *pgxpool.Stat) string { return fmt.Sprint(s.ConstructingConns()) }},
		{"vec_pool_prestamos_total", "Préstamos de conexión atendidos.", "counter", func(s *pgxpool.Stat) string { return fmt.Sprint(s.AcquireCount()) }},
		{"vec_pool_prestamos_con_espera_total", "Préstamos que tuvieron que esperar porque no había conexión libre.", "counter", func(s *pgxpool.Stat) string { return fmt.Sprint(s.EmptyAcquireCount()) }},
		{"vec_pool_prestamos_cancelados_total", "Préstamos abandonados (plazo vencido o petición cancelada).", "counter", func(s *pgxpool.Stat) string { return fmt.Sprint(s.CanceledAcquireCount()) }},
		{"vec_pool_espera_segundos_total", "Tiempo total esperando conexión.", "counter", func(s *pgxpool.Stat) string { return fmt.Sprintf("%g", s.AcquireDuration().Seconds()) }},
		{"vec_pool_conexiones_nuevas_total", "Conexiones abiertas desde el arranque.", "counter", func(s *pgxpool.Stat) string { return fmt.Sprint(s.NewConnsCount()) }},
	}
	for _, md := range medidas {
		fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s %s\n", md.nombre, md.ayuda, md.nombre, md.tipo)
		for _, p := range pools {
			fmt.Fprintf(b, "%s{%s,pool=\"%s\"} %s\n", md.nombre, base, telemetria.EscaparEtiqueta(p.nombre), md.valor(p.est))
		}
	}
	_, _ = io.WriteString(w, b.String())
}
