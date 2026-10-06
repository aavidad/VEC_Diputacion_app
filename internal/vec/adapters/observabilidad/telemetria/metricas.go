package telemetria

import (
	"fmt"
	"io"
	"runtime/metrics"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Métricas por ruta en formato de texto de Prometheus. Solo se publican en
// la superficie de diagnóstico (paquete diagnostico), nunca en un portal.

const (
	// maxSeriesRuta acota la cardinalidad: a partir de ahí las rutas nuevas
	// se suman en "{otras}".
	maxSeriesRuta = 512
	rutaOtras     = "{otras}"
)

// limitesHistograma en segundos, de 5 ms a 30 s.
var limitesHistograma = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

type serieRuta struct {
	metodo, ruta string
	porClase     [6]atomic.Uint64 // índice = estado/100 (1xx…5xx)
	cubetas      []atomic.Uint64  // sin acumular: una por límite y la última para +Inf
	sumaNanos    atomic.Uint64
	bdConsultas  atomic.Uint64
	bdNanos      atomic.Uint64
	esperaNanos  atomic.Uint64
	lentas       atomic.Uint64
}

type metricasRutas struct {
	series sync.Map // clave "METODO ruta" -> *serieRuta
	total  atomic.Int64
	altaMu sync.Mutex
}

func (m *metricasRutas) serie(metodo, ruta string) *serieRuta {
	if s, ok := m.series.Load(metodo + " " + ruta); ok {
		return s.(*serieRuta)
	}
	if m.total.Load() >= maxSeriesRuta {
		ruta = rutaOtras
	}
	clave := metodo + " " + ruta
	m.altaMu.Lock()
	defer m.altaMu.Unlock()
	if s, ok := m.series.Load(clave); ok {
		return s.(*serieRuta)
	}
	s := &serieRuta{metodo: metodo, ruta: ruta, cubetas: make([]atomic.Uint64, len(limitesHistograma)+1)}
	m.series.Store(clave, s)
	m.total.Add(1)
	return s
}

func (m *metricasRutas) observar(l *lineaAcceso, duracion, bd, espera time.Duration) {
	s := m.serie(l.Metodo, l.Ruta)
	if clase := l.Estado / 100; clase >= 1 && clase <= 5 {
		s.porClase[clase].Add(1)
	}
	segundos := duracion.Seconds()
	i := sort.SearchFloat64s(limitesHistograma, segundos)
	s.cubetas[i].Add(1)
	s.sumaNanos.Add(uint64(max(duracion, 0)))
	s.bdConsultas.Add(uint64(max(l.BDConsultas, 0)))
	s.bdNanos.Add(uint64(max(bd, 0)))
	s.esperaNanos.Add(uint64(max(espera, 0)))
	if l.Lenta {
		s.lentas.Add(1)
	}
}

// EscribirMetricas escribe las métricas del registro de acceso y del
// proceso en formato de texto de Prometheus 0.0.4.
func (reg *Registro) EscribirMetricas(w io.Writer) {
	if reg == nil {
		return
	}
	var series []*serieRuta
	reg.metricas.series.Range(func(_, v any) bool {
		series = append(series, v.(*serieRuta))
		return true
	})
	sort.Slice(series, func(i, j int) bool {
		if series[i].ruta != series[j].ruta {
			return series[i].ruta < series[j].ruta
		}
		return series[i].metodo < series[j].metodo
	})
	b := &strings.Builder{}
	base := fmt.Sprintf(`servicio="%s",superficie="%s"`, reg.servicio, reg.superficie)

	fmt.Fprintf(b, "# HELP vec_http_peticiones_total Peticiones atendidas por ruta, método y clase de estado.\n# TYPE vec_http_peticiones_total counter\n")
	for _, s := range series {
		for clase := 1; clase <= 5; clase++ {
			if n := s.porClase[clase].Load(); n > 0 {
				fmt.Fprintf(b, "vec_http_peticiones_total{%s,%s,clase=\"%dxx\"} %d\n", base, etiquetasRuta(s), clase, n)
			}
		}
	}
	fmt.Fprintf(b, "# HELP vec_http_duracion_segundos Duración de las peticiones en el servidor.\n# TYPE vec_http_duracion_segundos histogram\n")
	for _, s := range series {
		var acumulado, total uint64
		for i, limite := range limitesHistograma {
			acumulado += s.cubetas[i].Load()
			fmt.Fprintf(b, "vec_http_duracion_segundos_bucket{%s,%s,le=\"%g\"} %d\n", base, etiquetasRuta(s), limite, acumulado)
		}
		total = acumulado + s.cubetas[len(limitesHistograma)].Load()
		fmt.Fprintf(b, "vec_http_duracion_segundos_bucket{%s,%s,le=\"+Inf\"} %d\n", base, etiquetasRuta(s), total)
		fmt.Fprintf(b, "vec_http_duracion_segundos_sum{%s,%s} %g\n", base, etiquetasRuta(s), float64(s.sumaNanos.Load())/1e9)
		fmt.Fprintf(b, "vec_http_duracion_segundos_count{%s,%s} %d\n", base, etiquetasRuta(s), total)
	}
	escribirContadorRuta(b, series, base, "vec_http_bd_consultas_total", "Consultas SQL hechas por las peticiones de cada ruta.",
		func(s *serieRuta) string { return fmt.Sprint(s.bdConsultas.Load()) })
	escribirContadorRuta(b, series, base, "vec_http_bd_segundos_total", "Tiempo en base de datos de las peticiones de cada ruta.",
		func(s *serieRuta) string { return fmt.Sprintf("%g", float64(s.bdNanos.Load())/1e9) })
	escribirContadorRuta(b, series, base, "vec_http_bd_espera_conexion_segundos_total", "Tiempo esperando una conexión libre del pool.",
		func(s *serieRuta) string { return fmt.Sprintf("%g", float64(s.esperaNanos.Load())/1e9) })
	escribirContadorRuta(b, series, base, "vec_http_lentas_total", "Peticiones marcadas como lentas.",
		func(s *serieRuta) string { return fmt.Sprint(s.lentas.Load()) })

	fmt.Fprintf(b, "# HELP vec_http_en_curso Peticiones que se están atendiendo ahora.\n# TYPE vec_http_en_curso gauge\nvec_http_en_curso{%s} %d\n", base, reg.EnCurso())
	fmt.Fprintf(b, "# HELP vec_registro_acceso_descartadas_total Líneas de acceso perdidas por cola llena.\n# TYPE vec_registro_acceso_descartadas_total counter\nvec_registro_acceso_descartadas_total{%s} %d\n", base, reg.Descartadas())
	escribirMetricasProceso(b, base)
	_, _ = io.WriteString(w, b.String())
}

func escribirContadorRuta(b *strings.Builder, series []*serieRuta, base, nombre, ayuda string, valor func(*serieRuta) string) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s counter\n", nombre, ayuda, nombre)
	for _, s := range series {
		fmt.Fprintf(b, "%s{%s,%s} %s\n", nombre, base, etiquetasRuta(s), valor(s))
	}
}

func etiquetasRuta(s *serieRuta) string {
	return `metodo="` + s.metodo + `",ruta="` + EscaparEtiqueta(s.ruta) + `"`
}

// EscaparEtiqueta escapa un valor de etiqueta de Prometheus.
func EscaparEtiqueta(v string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(v)
}

var muestrasProceso = []struct{ nombre, metrica, ayuda, tipo string }{
	{"/sched/goroutines:goroutines", "vec_proceso_gorrutinas", "Gorrutinas vivas.", "gauge"},
	{"/memory/classes/heap/objects:bytes", "vec_proceso_memoria_objetos_bytes", "Memoria ocupada por objetos vivos y aún no recogidos.", "gauge"},
	{"/memory/classes/total:bytes", "vec_proceso_memoria_total_bytes", "Memoria total reservada por el entorno de Go.", "gauge"},
	{"/gc/cycles/total:gc-cycles", "vec_proceso_gc_ciclos_total", "Ciclos de recolección de memoria.", "counter"},
}

// escribirMetricasProceso usa runtime/metrics, que no detiene el proceso.
func escribirMetricasProceso(b *strings.Builder, base string) {
	muestras := make([]metrics.Sample, len(muestrasProceso))
	for i, m := range muestrasProceso {
		muestras[i].Name = m.nombre
	}
	metrics.Read(muestras)
	for i, m := range muestrasProceso {
		if muestras[i].Value.Kind() != metrics.KindUint64 {
			continue
		}
		fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s %s\n%s{%s} %d\n", m.metrica, m.ayuda, m.metrica, m.tipo, m.metrica, base, muestras[i].Value.Uint64())
	}
}
