package telemetria

import (
	"sort"
	"time"
)

// Medida de la base de datos por petición. La alimenta el trazador de pgx
// (paquete medidorpg) sin que los módulos cambien: solo hace falta que la
// consulta use el contexto de la petición.

const (
	// maxOperacionesFicha acota el desglose por petición; las demás
	// operaciones se suman en "otras".
	maxOperacionesFicha = 16
	operacionOtras      = "otras"
	// maxDesglose es el número de operaciones que se escriben, de mayor a
	// menor tiempo.
	maxDesglose = 5
)

type medidaOperacion struct {
	consultas int
	total     time.Duration
	maxima    time.Duration
}

// medidasBD exige Ficha.mu tomado para leer o escribir.
type medidasBD struct {
	consultas   int
	total       time.Duration
	esperas     int
	espera      time.Duration
	ultimoError string
	operaciones map[string]*medidaOperacion
	// actividad describe lo que la petición está haciendo en la base de
	// datos ahora mismo, para la línea "en curso".
	actividad      string
	objetoActivo   string
	actividadDesde time.Time
	activas        int
}

// ConsultaIniciada anota que la petición espera la respuesta de una
// operación SQL (nombre cerrado que da medidorpg).
func (f *Ficha) ConsultaIniciada(operacion string, ahora time.Time) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.bd.activas++
	f.bd.actividad, f.bd.objetoActivo, f.bd.actividadDesde = "consulta", operacion, ahora
	f.mu.Unlock()
}

// ConsultaTerminada suma n consultas de la operación (n > 1 en un lote) y su
// duración. claseError es la de ClasificarError o vacía.
func (f *Ficha) ConsultaTerminada(operacion string, n int, d time.Duration, claseError string) {
	if f == nil {
		return
	}
	if n < 1 {
		n = 1
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.terminarActividadBloqueado()
	f.bd.consultas += n
	f.bd.total += d
	if claseError != "" {
		f.bd.ultimoError = claseError
	}
	if f.bd.operaciones == nil {
		f.bd.operaciones = make(map[string]*medidaOperacion, 4)
	}
	m := f.bd.operaciones[operacion]
	if m == nil {
		if len(f.bd.operaciones) >= maxOperacionesFicha {
			operacion = operacionOtras
			m = f.bd.operaciones[operacion]
		}
		if m == nil {
			m = &medidaOperacion{}
			f.bd.operaciones[operacion] = m
		}
	}
	m.consultas += n
	m.total += d
	if d > m.maxima {
		m.maxima = d
	}
}

// EsperaConexionIniciada anota que la petición espera una conexión libre
// del pool indicado (nombre del rol de base de datos, no de una persona).
func (f *Ficha) EsperaConexionIniciada(pool string, ahora time.Time) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.bd.activas++
	f.bd.actividad, f.bd.objetoActivo, f.bd.actividadDesde = "esperando_conexion", pool, ahora
	f.mu.Unlock()
}

// EsperaConexionTerminada suma la espera por una conexión del pool.
func (f *Ficha) EsperaConexionTerminada(d time.Duration, claseError string) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.terminarActividadBloqueado()
	f.bd.esperas++
	f.bd.espera += d
	if claseError != "" {
		f.bd.ultimoError = "conexion_" + claseError
	}
	f.mu.Unlock()
}

func (f *Ficha) terminarActividadBloqueado() {
	if f.bd.activas > 0 {
		f.bd.activas--
	}
	if f.bd.activas == 0 {
		f.bd.actividad, f.bd.objetoActivo = "", ""
	}
}

// OperacionDesglose es una fila del desglose de una petición lenta.
type OperacionDesglose struct {
	Operacion string  `json:"operacion"`
	Consultas int     `json:"consultas"`
	MS        float64 `json:"ms"`
	MaxMS     float64 `json:"max_ms"`
}

// desgloseBloqueado devuelve las operaciones con más tiempo. Exige f.mu.
func (f *Ficha) desgloseBloqueado() []OperacionDesglose {
	if len(f.bd.operaciones) == 0 {
		return nil
	}
	filas := make([]OperacionDesglose, 0, len(f.bd.operaciones))
	for op, m := range f.bd.operaciones {
		filas = append(filas, OperacionDesglose{Operacion: op, Consultas: m.consultas, MS: milisegundos(m.total), MaxMS: milisegundos(m.maxima)})
	}
	sort.Slice(filas, func(i, j int) bool {
		if filas[i].MS != filas[j].MS {
			return filas[i].MS > filas[j].MS
		}
		if filas[i].Consultas != filas[j].Consultas {
			return filas[i].Consultas > filas[j].Consultas
		}
		return filas[i].Operacion < filas[j].Operacion
	})
	if len(filas) > maxDesglose {
		filas = filas[:maxDesglose]
	}
	return filas
}

// ResumenBD es la medida acumulada de una ficha.
type ResumenBD struct {
	Consultas   int
	Total       time.Duration
	Esperas     int
	Espera      time.Duration
	Error       string
	Operaciones map[string]int
}

// ResumenBD devuelve una copia de la medida acumulada.
func (f *Ficha) ResumenBD() ResumenBD {
	if f == nil {
		return ResumenBD{}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	r := ResumenBD{Consultas: f.bd.consultas, Total: f.bd.total, Esperas: f.bd.esperas, Espera: f.bd.espera,
		Error: f.bd.ultimoError, Operaciones: make(map[string]int, len(f.bd.operaciones))}
	for op, m := range f.bd.operaciones {
		r.Operaciones[op] = m.consultas
	}
	return r
}
