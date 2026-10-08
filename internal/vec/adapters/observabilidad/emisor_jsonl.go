// Package observabilidad contiene la emisión técnica de incidencias de VEC.
//
// EmisorJSONLines implementa ports.EmisorIncidenciasTecnicas escribiendo cada
// incidencia como una línea JSON UTF-8 en un io.Writer (la salida estándar en
// producción, recogida fuera del proceso). Cumple el requisito de que la
// supervisión sea una capa que nunca retrase la aplicación:
//
//   - Emitir solo sanea contra el catálogo, toma la hora y hace un envío no
//     bloqueante a una cola acotada; si está llena, descarta y cuenta.
//   - La serialización y la escritura ocurren en un único trabajador propio;
//     un destino lento o bloqueado solo llena la cola.
//   - Los descartes se declaran periódicamente como RECOLECCION_DEGRADADA
//     escrita directamente por el trabajador, sin reentrar en la cola.
//   - No hay estado global: cada emisor es independiente.
package observabilidad

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	// CapacidadPredeterminada de la cola de emisión.
	CapacidadPredeterminada = 1024
	// CapacidadMaxima acota la memoria retenida por la cola (cada elemento
	// ocupa un tamaño fijo y solo referencia constantes del catálogo).
	CapacidadMaxima = 8192
	// PeriodoInformeDescartesPredeterminado entre declaraciones de pérdidas.
	PeriodoInformeDescartesPredeterminado = 5 * time.Second
)

// ErrDestinoIncidenciasAusente indica que no se configuró un io.Writer.
var ErrDestinoIncidenciasAusente = errors.New("observabilidad: destino de incidencias ausente")

// OpcionesEmisor configura un EmisorJSONLines.
type OpcionesEmisor struct {
	// Catalogo es una instantánea validada. Nil usa el catálogo embebido
	// seleccionado por el índice común de idiomas.
	Catalogo *catalogoincidencias.Catalogo
	// Destino recibe líneas JSON completas, una llamada Write por línea.
	Destino io.Writer
	// Capacidad de la cola; 0 usa la predeterminada y se acota a CapacidadMaxima.
	Capacidad int
	// Entorno se normaliza a la lista cerrada del dominio.
	Entorno string
	// VersionBinario se normaliza; lo no conforme pasa a "desconocida".
	VersionBinario string
	// Reloj opcional; por defecto time.Now. Debe ser barato y no bloquear.
	Reloj func() time.Time
	// Aleatorio opcional para correlaciones; por defecto crypto/rand.
	Aleatorio io.Reader
	// PeriodoInformeDescartes opcional; por defecto cinco segundos.
	PeriodoInformeDescartes time.Duration
}

type elementoCola struct {
	clasificacion          domain.ClasificacionIncidenciaTecnica
	clasificacionResultado domain.ClasificacionResultadoTecnico
	esResultado            bool
	instante               time.Time
	correlacion            string
	referencia             domain.ReferenciaCorrelacionAutorizacionV2
}

// EmisorJSONLines es seguro para uso concurrente. Un puntero nil es un
// emisor inerte válido: Emitir no hace nada.
type EmisorJSONLines struct {
	catalogo  *catalogoincidencias.Catalogo
	destino   io.Writer
	entorno   domain.EntornoIncidenciaTecnica
	version   string
	reloj     func() time.Time
	aleatorio io.Reader
	periodo   time.Duration

	cola      chan elementoCola
	parar     chan struct{}
	terminado chan struct{}
	cierre    sync.Once

	cerrado atomic.Bool
	enVuelo atomic.Int64

	aceptadas                 atomic.Uint64
	descartadas               atomic.Uint64
	saneadas                  atomic.Uint64
	escritas                  atomic.Uint64
	fallosEscritura           atomic.Uint64
	fallosCorrelacion         atomic.Uint64
	resultadosAceptados       atomic.Uint64
	resultadosDescartados     atomic.Uint64
	resultadosInvalidos       atomic.Uint64
	resultadosSinCorrelacion  atomic.Uint64
	resultadosEscritos        atomic.Uint64
	resultadosFallosEscritura atomic.Uint64

	// Solo los usa el trabajador.
	descartesInformados uint64
	buffer              []byte
}

var (
	_ ports.EmisorIncidenciasTecnicasConContexto    = (*EmisorJSONLines)(nil)
	_ ports.AceptadorIncidenciasTecnicasConContexto = (*EmisorJSONLines)(nil)
	_ ports.ConsultaMetricasEmisionIncidencias      = (*EmisorJSONLines)(nil)
	_ ports.CierreEmisionIncidencias                = (*EmisorJSONLines)(nil)
)

// NuevoEmisorJSONLines crea el emisor y arranca su trabajador. Debe cerrarse
// con Cerrar para vaciar lo pendiente.
func NuevoEmisorJSONLines(o OpcionesEmisor) (*EmisorJSONLines, error) {
	if o.Destino == nil {
		return nil, ErrDestinoIncidenciasAusente
	}
	catalogo := o.Catalogo
	if catalogo == nil {
		var err error
		catalogo, err = catalogoincidencias.Predeterminado()
		if err != nil {
			return nil, err
		}
	}
	if !catalogo.Valido() {
		return nil, os.ErrInvalid
	}
	capacidad := o.Capacidad
	if capacidad <= 0 {
		capacidad = CapacidadPredeterminada
	}
	if capacidad > CapacidadMaxima {
		capacidad = CapacidadMaxima
	}
	reloj := o.Reloj
	if reloj == nil {
		reloj = time.Now
	}
	aleatorio := o.Aleatorio
	if aleatorio == nil {
		aleatorio = rand.Reader
	}
	periodo := o.PeriodoInformeDescartes
	if periodo <= 0 {
		periodo = PeriodoInformeDescartesPredeterminado
	}
	e := &EmisorJSONLines{
		catalogo:  catalogo,
		destino:   o.Destino,
		entorno:   domain.NormalizarEntornoIncidenciaTecnica(o.Entorno),
		version:   domain.NormalizarVersionBinario(o.VersionBinario),
		reloj:     reloj,
		aleatorio: aleatorio,
		periodo:   periodo,
		cola:      make(chan elementoCola, capacidad),
		parar:     make(chan struct{}),
		terminado: make(chan struct{}),
		buffer:    make([]byte, 0, 512),
	}
	go e.trabajar()
	return e, nil
}

// Emitir sanea la solicitud y la encola sin bloquear. Nunca hace E/S.
func (e *EmisorJSONLines) Emitir(s domain.SolicitudIncidenciaTecnica) {
	e.EmitirConContexto(context.Background(), s)
}

// EmitirConContexto copia únicamente la correlación interna antes de encolar.
// No retiene contexto ni genera aleatoriedad o hace E/S en el llamante.
func (e *EmisorJSONLines) EmitirConContexto(ctx context.Context, s domain.SolicitudIncidenciaTecnica) {
	e.AceptarConContexto(ctx, s)
}

// AceptarConContexto confirma solo la entrada en la cola no bloqueante.
// El destino se escribe más tarde; una escritura fallida queda en métricas.
func (e *EmisorJSONLines) AceptarConContexto(ctx context.Context, s domain.SolicitudIncidenciaTecnica) bool {
	if e == nil {
		return false
	}
	// Invariante: incrementar enVuelo ANTES de leer cerrado. Cerrar marca la
	// bandera y después espera a enVuelo == 0; si el orden se invirtiera, una
	// incidencia aceptada podría quedar en la cola sin escribir tras el cierre.
	e.enVuelo.Add(1)
	defer e.enVuelo.Add(-1)
	if e.cerrado.Load() {
		e.descartadas.Add(1)
		return false
	}
	clasificacion, saneada := domain.ClasificarIncidenciaTecnica(s)
	correlacion, _ := ports.CorrelacionIncidenciasPeticion(ctx)
	select {
	case e.cola <- elementoCola{clasificacion: clasificacion, instante: e.reloj(), correlacion: correlacion}:
		e.aceptadas.Add(1)
		if saneada {
			e.saneadas.Add(1)
		}
		return true
	default:
		e.descartadas.Add(1)
		return false
	}
}

// MetricasEmision devuelve una instantánea de los contadores internos.
func (e *EmisorJSONLines) MetricasEmision() ports.MetricasEmisionIncidencias {
	if e == nil {
		return ports.MetricasEmisionIncidencias{}
	}
	return ports.MetricasEmisionIncidencias{
		Aceptadas:         e.aceptadas.Load(),
		Descartadas:       e.descartadas.Load(),
		Saneadas:          e.saneadas.Load(),
		Escritas:          e.escritas.Load(),
		FallosEscritura:   e.fallosEscritura.Load(),
		FallosCorrelacion: e.fallosCorrelacion.Load(),
		PendientesEnCola:  uint64(len(e.cola)),
	}
}

// Cerrar deja de aceptar incidencias, espera a que el trabajador escriba lo
// pendiente y declare los descartes, y devuelve nil. Si el contexto vence
// antes (por ejemplo, destino bloqueado) devuelve su error; el trabajador
// seguirá hasta que el destino desbloquee. Es idempotente.
func (e *EmisorJSONLines) Cerrar(ctx context.Context) error {
	if e == nil {
		return nil
	}
	e.cierre.Do(func() {
		e.cerrado.Store(true)
		go func() {
			// Espera a los Emitir que ya comprobaron la bandera, para que
			// ninguna incidencia aceptada quede sin escribir tras el cierre.
			// Espera con pausa corta, sin ocupar CPU, aunque siga habiendo
			// emisiones concurrentes (éstas ven la bandera y descartan).
			for e.enVuelo.Load() > 0 {
				time.Sleep(time.Millisecond)
			}
			close(e.parar)
		}()
	})
	select {
	case <-e.terminado:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *EmisorJSONLines) trabajar() {
	defer close(e.terminado)
	temporizador := time.NewTicker(e.periodo)
	defer temporizador.Stop()
	for {
		select {
		case elemento := <-e.cola:
			e.escribirElemento(elemento)
		case <-temporizador.C:
			e.informarDescartes()
		case <-e.parar:
			for {
				select {
				case elemento := <-e.cola:
					e.escribirElemento(elemento)
				default:
					e.informarDescartes()
					return
				}
			}
		}
	}
}

func (e *EmisorJSONLines) escribirElemento(elemento elementoCola) {
	if elemento.esResultado {
		e.escribirResultado(elemento.clasificacionResultado, elemento.instante, elemento.correlacion, elemento.referencia)
		return
	}
	e.escribir(elemento.clasificacion, elemento.instante, elemento.correlacion)
}

// informarDescartes declara las pérdidas nuevas como una única incidencia
// escrita directamente, sin pasar por la cola (evita recursión y pérdida).
func (e *EmisorJSONLines) informarDescartes() {
	total := e.descartadas.Load()
	nuevos := total - e.descartesInformados
	if nuevos == 0 {
		return
	}
	e.descartesInformados = total
	recuento := uint32(domain.RecuentoMaximoIncidenciaTecnica)
	if nuevos < uint64(recuento) {
		recuento = uint32(nuevos)
	}
	e.escribir(domain.ClasificacionRecoleccionDegradada(domain.EtapaIncidenciaEmision, recuento), e.reloj(), "")
}

// lineaIncidencia es la lista blanca serializada; no admite otros campos.
type lineaIncidencia struct {
	Esquema        string `json:"esquema"`
	Instante       string `json:"instante"`
	Codigo         string `json:"codigo"`
	Severidad      string `json:"severidad"`
	Componente     string `json:"componente"`
	Etapa          string `json:"etapa"`
	Entorno        string `json:"entorno"`
	VersionBinario string `json:"version_binario"`
	Correlacion    string `json:"correlacion"`
	Recuento       uint32 `json:"recuento"`
	Mensaje        string `json:"mensaje"`
}

const formatoInstante = "2006-01-02T15:04:05.000Z"

func (e *EmisorJSONLines) escribir(c domain.ClasificacionIncidenciaTecnica, instante time.Time, correlacion string) {
	if correlacion == "" {
		correlacion = e.nuevaCorrelacion()
		if correlacion == "" {
			e.fallosCorrelacion.Add(1)
			return
		}
	}
	incidencia := domain.NuevaIncidenciaTecnica(c, instante, e.entorno, e.version, correlacion)
	plantilla, ok := e.catalogo.Plantilla(incidencia.Codigo)
	if !ok {
		e.fallosEscritura.Add(1)
		return
	}
	linea := lineaIncidencia{
		Esquema:        incidencia.Esquema,
		Instante:       incidencia.Instante.Format(formatoInstante),
		Codigo:         string(incidencia.Codigo),
		Severidad:      string(incidencia.Severidad),
		Componente:     string(incidencia.Componente),
		Etapa:          string(incidencia.Etapa),
		Entorno:        string(incidencia.Entorno),
		VersionBinario: incidencia.VersionBinario,
		Correlacion:    incidencia.Correlacion,
		Recuento:       incidencia.Recuento,
		Mensaje:        plantilla,
	}
	datos, err := json.Marshal(linea)
	if err != nil {
		e.fallosEscritura.Add(1)
		return
	}
	if e.escribirLinea(datos) {
		e.escritas.Add(1)
	} else {
		e.fallosEscritura.Add(1)
	}
}

// escribirLinea es el único punto de escritura del trabajador para ambas
// familias técnicas. Una escritura corta nunca se cuenta como confirmada.
func (e *EmisorJSONLines) escribirLinea(datos []byte) bool {
	e.buffer = append(append(e.buffer[:0], datos...), '\n')
	n, err := e.destino.Write(e.buffer)
	return err == nil && n == len(e.buffer)
}

// nuevaCorrelacion genera 128 bits aleatorios sin relación con ningún dato
// de la petición o de la persona. Si la fuente falla, no se escribe la
// incidencia y se contabiliza: nunca se inventa una correlación compartida.
func (e *EmisorJSONLines) nuevaCorrelacion() string {
	var bruto [16]byte
	if _, err := io.ReadFull(e.aleatorio, bruto[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(bruto[:])
}
