// Package importacionconvoca contiene el modelo neutral de infraestructura
// para ensayar y validar exportaciones enmascaradas de Convoca.
//
// Una fila importada nunca es autoridad de identidad ni habilita por si sola
// llamamientos, contratos u otros actos con efectos.
package importacionconvoca

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	EsquemaResumenPersona EsquemaExportacion = "convoca_resumen_persona_v1"
	EsquemaDetalleMerito  EsquemaExportacion = "convoca_detalle_merito_v1"

	EsquemaProcedenciaV1    = "vec.bolsa.importacion-convoca.procedencia.v1"
	FuenteConvoca           = "Convoca (exportacion enmascarada)"
	AutoridadNoAutoritativa = "no_autoritativa"
	UsoAutobaremoHistorico  = "historico_contraste"
)

var (
	ErrEsquemaExportacionDesconocido = errors.New("bolsa: esquema de exportacion Convoca desconocido")
	ErrHojaStagingInvalida           = errors.New("bolsa: hoja staging Convoca invalida")
	ErrLoteImportacionInvalido       = errors.New("bolsa: lote de importacion Convoca invalido")
)

var huellaSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)
var codigoIncidencia = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)
var referenciaCustodia = regexp.MustCompile(`^[a-z][a-z0-9_.:/-]{2,511}$`)
var actorOpaco = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)

// EsquemaExportacion identifica una cabecera exacta, no una heuristica por
// numero de columnas.
type EsquemaExportacion string

func (e EsquemaExportacion) Validar() error {
	switch e {
	case EsquemaResumenPersona, EsquemaDetalleMerito:
		return nil
	default:
		return ErrEsquemaExportacionDesconocido
	}
}

// Cabeceras devuelve una copia de las cabeceras literales acreditadas en T17.
func (e EsquemaExportacion) Cabeceras() []string {
	var origen []string
	switch e {
	case EsquemaResumenPersona:
		origen = cabecerasResumen
	case EsquemaDetalleMerito:
		origen = cabecerasDetalle
	default:
		return nil
	}
	return append([]string(nil), origen...)
}

func (e EsquemaExportacion) NumeroColumnas() int { return len(e.Cabeceras()) }

var cabecerasResumen = []string{
	"DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
	"Experiencia", "Formacion", "Total",
}

var cabecerasDetalle = []string{
	"DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
	"Grupo", "Descripcion del grupo", "Orden grupo",
	"Descripcion del merito", "Puntos autobaremacion", "Puntos tribunal",
	"Motivo",
}

// DetectarEsquema exige coincidencia literal y ordenada. No corrige tildes,
// mayusculas ni espacios porque hacerlo podria aceptar una exportacion ajena.
func DetectarEsquema(cabeceras []string) (EsquemaExportacion, error) {
	for _, esquema := range []EsquemaExportacion{EsquemaResumenPersona, EsquemaDetalleMerito} {
		esperadas := esquema.Cabeceras()
		if len(cabeceras) != len(esperadas) {
			continue
		}
		coinciden := true
		for i := range esperadas {
			if cabeceras[i] != esperadas[i] {
				coinciden = false
				break
			}
		}
		if coinciden {
			return esquema, nil
		}
	}
	return "", ErrEsquemaExportacionDesconocido
}

// TipoCelda conserva solo la clase necesaria para rechazar formulas y tipos
// inesperados. El adaptador no entrega expresiones ni otros metadatos XLS.
type TipoCelda string

const (
	CeldaVacia   TipoCelda = "vacia"
	CeldaTexto   TipoCelda = "texto"
	CeldaNumero  TipoCelda = "numero"
	CeldaFormula TipoCelda = "formula"
	CeldaError   TipoCelda = "error"
	CeldaLogica  TipoCelda = "logica"
	CeldaFecha   TipoCelda = "fecha"
)

type CeldaStaging struct {
	Tipo  TipoCelda
	Valor string
}

type FilaStaging struct {
	Numero int
	Celdas []CeldaStaging
}

type HojaStaging struct {
	Esquema   EsquemaExportacion
	Cabeceras []string
	Filas     []FilaStaging
}

func (h HojaStaging) ValidarEstructura() error {
	if h.Esquema.Validar() != nil || len(h.Cabeceras) != h.Esquema.NumeroColumnas() {
		return ErrHojaStagingInvalida
	}
	detectado, err := DetectarEsquema(h.Cabeceras)
	if err != nil || detectado != h.Esquema {
		return ErrHojaStagingInvalida
	}
	anterior := 1
	for _, fila := range h.Filas {
		if fila.Numero <= anterior || fila.Numero < 2 {
			return ErrHojaStagingInvalida
		}
		anterior = fila.Numero
	}
	return nil
}

type IdentidadEnmascarada struct {
	Documento       string
	PrimerApellido  string
	SegundoApellido string
	Nombre          string
}

type ResumenPersona struct {
	Experiencia string
	Formacion   string
	Total       string
}

type DetalleMerito struct {
	Grupo                          string
	DescripcionGrupo               string
	OrdenGrupo                     uint32
	DescripcionMerito              string
	PuntosAutobaremacionHistoricos string
	PuntosTribunal                 string
	Motivo                         string
}

type FilaAceptada struct {
	Numero    int
	Esquema   EsquemaExportacion
	Identidad IdentidadEnmascarada
	Turno     string
	Resumen   *ResumenPersona
	Detalle   *DetalleMerito
}

type Incidencia struct {
	Fila   int
	Campo  string
	Codigo string
}

type ResultadoStaging struct {
	FilasLeidas int
	Aceptadas   []FilaAceptada
	Rechazadas  int
	Incidencias []Incidencia
}

// Procedencia es una marca durable y estructural. Sus valores cerrados evitan
// que una exportacion enmascarada se promueva accidentalmente a autoridad.
type Procedencia struct {
	Esquema                      string
	Fuente                       string
	Autoridad                    string
	HabilitaActosConEfectos      bool
	RequiereConfirmacionRegistro bool
	UsoPuntosAutobaremacion      string
}

func NuevaProcedenciaNoAutoritativa() Procedencia {
	return Procedencia{
		Esquema: EsquemaProcedenciaV1, Fuente: FuenteConvoca,
		Autoridad: AutoridadNoAutoritativa, HabilitaActosConEfectos: false,
		RequiereConfirmacionRegistro: true,
		UsoPuntosAutobaremacion:      UsoAutobaremoHistorico,
	}
}

func (p Procedencia) Validar() error {
	if p != NuevaProcedenciaNoAutoritativa() {
		return ErrLoteImportacionInvalido
	}
	return nil
}

type ActaImportacion struct {
	ActaRef              string
	ImportacionRef       string
	HuellaFicheroSHA256  string
	FicheroCustodiadoRef string
	NombreFichero        string
	ActorRef             string
	RegistradaEn         time.Time
	Esquema              EsquemaExportacion
	FilasLeidas          int
	FilasAceptadas       int
	FilasRechazadas      int
	Incidencias          []Incidencia
	Procedencia          Procedencia
}

type LoteValidado struct {
	Acta      ActaImportacion
	Aceptadas []FilaAceptada
}

// Validar comprueba el acta minimizada sin exigir que el staging siga
// disponible. Esta separación permite conservar la prueba de una importación
// después del expurgo gobernado de sus filas personales.
func (a ActaImportacion) Validar() error {
	if !huellaSHA256.MatchString(a.HuellaFicheroSHA256) ||
		a.ActaRef != "acta:importacion-convoca:"+a.HuellaFicheroSHA256 ||
		a.ImportacionRef != "importacion:convoca:"+a.HuellaFicheroSHA256 ||
		!referenciaCustodia.MatchString(a.FicheroCustodiadoRef) ||
		a.Esquema.Validar() != nil || a.Procedencia.Validar() != nil ||
		a.FilasLeidas < 0 || a.FilasAceptadas < 0 || a.FilasRechazadas < 0 ||
		a.FilasLeidas > 100_001 ||
		a.FilasAceptadas+a.FilasRechazadas != a.FilasLeidas ||
		strings.TrimSpace(a.NombreFichero) != a.NombreFichero ||
		strings.TrimSpace(a.ActorRef) != a.ActorRef ||
		len(a.NombreFichero) < 5 || len(a.NombreFichero) > 255 ||
		!utf8.ValidString(a.NombreFichero) ||
		strings.ContainsAny(a.NombreFichero, `/\`) ||
		!strings.HasSuffix(strings.ToLower(a.NombreFichero), ".xls") ||
		!actorOpaco.MatchString(a.ActorRef) || a.RegistradaEn.IsZero() ||
		a.RegistradaEn.Location() != time.UTC || a.RegistradaEn.Nanosecond()%1000 != 0 {
		return ErrLoteImportacionInvalido
	}
	for _, caracter := range a.NombreFichero {
		if unicode.IsControl(caracter) {
			return ErrLoteImportacionInvalido
		}
	}
	if len(a.Incidencias) > a.FilasRechazadas*a.Esquema.NumeroColumnas() {
		return ErrLoteImportacionInvalido
	}
	filasConIncidencia := make(map[int]struct{}, a.FilasRechazadas)
	for _, incidencia := range a.Incidencias {
		if incidencia.Fila < 2 || !textoIncidenciaValido(incidencia.Campo, 120) ||
			!codigoIncidencia.MatchString(incidencia.Codigo) {
			return ErrLoteImportacionInvalido
		}
		filasConIncidencia[incidencia.Fila] = struct{}{}
	}
	if len(filasConIncidencia) != a.FilasRechazadas {
		return ErrLoteImportacionInvalido
	}
	return nil
}

func textoIncidenciaValido(valor string, maximo int) bool {
	if !utf8.ValidString(valor) || len(valor) < 1 || len(valor) > maximo ||
		strings.TrimSpace(valor) != valor {
		return false
	}
	for _, r := range valor {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// CoincideExactamente impide que la idempotencia por contenido reutilice un
// acta perteneciente a otro actor, nombre de fichero o resultado de validación.
func (a ActaImportacion) CoincideExactamente(otra ActaImportacion) bool {
	if a.ActaRef != otra.ActaRef ||
		a.ImportacionRef != otra.ImportacionRef ||
		a.HuellaFicheroSHA256 != otra.HuellaFicheroSHA256 ||
		a.FicheroCustodiadoRef != otra.FicheroCustodiadoRef ||
		a.NombreFichero != otra.NombreFichero ||
		a.ActorRef != otra.ActorRef ||
		a.Esquema != otra.Esquema ||
		a.FilasLeidas != otra.FilasLeidas ||
		a.FilasAceptadas != otra.FilasAceptadas ||
		a.FilasRechazadas != otra.FilasRechazadas ||
		a.Procedencia != otra.Procedencia ||
		len(a.Incidencias) != len(otra.Incidencias) {
		return false
	}
	for i := range a.Incidencias {
		if a.Incidencias[i] != otra.Incidencias[i] {
			return false
		}
	}
	return true
}

func (l LoteValidado) Validar() error {
	a := l.Acta
	if a.Validar() != nil || len(l.Aceptadas) != a.FilasAceptadas {
		return ErrLoteImportacionInvalido
	}
	numeroAnterior := 1
	for _, fila := range l.Aceptadas {
		if fila.Numero <= numeroAnterior || fila.Esquema != a.Esquema ||
			!filaAceptadaValida(fila) {
			return ErrLoteImportacionInvalido
		}
		numeroAnterior = fila.Numero
	}
	return nil
}
