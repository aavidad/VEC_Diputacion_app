// Package importacionconvoca contiene el modelo neutral de infraestructura
// para ensayar y validar exportaciones enmascaradas de Convoca.
//
// Una fila importada nunca es autoridad de identidad ni habilita por si sola
// llamamientos, contratos u otros actos con efectos.
package importacionconvoca

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
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

// EsquemaExportacion identifica el contenido lógico de una hoja (resumen por
// persona o detalle de méritos), no una heurística por número de columnas.
// Es el valor durable del acta; la variante literal de cabeceras con la que
// llegó el fichero se deriva de él (FormatoCabeceras) y se puede reproducir a
// partir del fichero custodiado y su huella.
type EsquemaExportacion string

func (e EsquemaExportacion) Validar() error {
	switch e {
	case EsquemaResumenPersona, EsquemaDetalleMerito:
		return nil
	default:
		return ErrEsquemaExportacionDesconocido
	}
}

// FormatoCabeceras identifica una variante literal y cerrada de la fila de
// títulos. Ambas variantes describen las mismas columnas en el mismo orden y
// se validan con las mismas reglas por posición; solo cambian los literales.
type FormatoCabeceras string

const (
	// FormatoConvocaV1 son las cabeceras transliteradas acreditadas en T17
	// (sin tildes, «DNI/NIE»). Se conserva para no invalidar ficheros ni
	// bolsas ya importados. No exige nombre de hoja: T17 no lo fijó.
	FormatoConvocaV1 FormatoCabeceras = "convoca:v1"
	// FormatoConvocaV2 son las cabeceras literales de una exportación real de
	// CONVOCA («DNI/NIE enmascarado», con tildes) y exige además el nombre de
	// hoja literal que CONVOCA asigna a cada exportación.
	FormatoConvocaV2 FormatoCabeceras = "convoca:v2"
)

var formatosCabeceras = []FormatoCabeceras{FormatoConvocaV1, FormatoConvocaV2}

// Cabeceras devuelve una copia de las cabeceras literales acreditadas en T17
// (formato convoca:v1).
func (e EsquemaExportacion) Cabeceras() []string {
	return e.CabecerasFormato(FormatoConvocaV1)
}

// CabecerasFormato devuelve una copia de las cabeceras literales del esquema
// en el formato indicado, o nil si la combinación no existe.
func (e EsquemaExportacion) CabecerasFormato(formato FormatoCabeceras) []string {
	origen := cabecerasPorFormato[formato][e]
	if origen == nil {
		return nil
	}
	return append([]string(nil), origen...)
}

func (e EsquemaExportacion) NumeroColumnas() int { return len(e.Cabeceras()) }

// NombreHoja devuelve el nombre de hoja exigido por el formato, o "" si el
// formato no fija ninguno.
func (e EsquemaExportacion) NombreHoja(formato FormatoCabeceras) string {
	return nombresHojaPorFormato[formato][e]
}

// Todos los literales se escriben en Unicode NFC; una prueba lo comprueba.
var cabecerasPorFormato = map[FormatoCabeceras]map[EsquemaExportacion][]string{
	FormatoConvocaV1: {
		EsquemaResumenPersona: {
			"DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
			"Experiencia", "Formacion", "Total",
		},
		EsquemaDetalleMerito: {
			"DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
			"Grupo", "Descripcion del grupo", "Orden grupo",
			"Descripcion del merito", "Puntos autobaremacion", "Puntos tribunal",
			"Motivo",
		},
	},
	// «DNI/NIE enmascarado» ocupa la misma primera columna que «DNI/NIE» y
	// alimenta el mismo campo IdentidadEnmascarada.Documento.
	FormatoConvocaV2: {
		EsquemaResumenPersona: {
			"DNI/NIE enmascarado", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
			"Experiencia", "Formación", "Total",
		},
		EsquemaDetalleMerito: {
			"DNI/NIE enmascarado", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
			"Grupo", "Descripción del grupo", "Orden grupo",
			"Descripción del mérito", "Puntos autobaremación", "Puntos tribunal",
			"Motivo",
		},
	},
}

// Nombres de hoja literales de la exportación real. El sufijo « (1)» es el
// único observado. No se acepta « (2)» ni ningún otro: no hay constancia de
// que CONVOCA lo genere ni de qué significaría (otra hoja, otro tribunal u
// otra copia), así que se rechaza hasta que RRHH lo confirme y se añada aquí
// como literal explícito.
var nombresHojaPorFormato = map[FormatoCabeceras]map[EsquemaExportacion]string{
	FormatoConvocaV2: {
		EsquemaResumenPersona: "grupo de méritos (Tribunal) (1)",
		EsquemaDetalleMerito:  "méritos (1)",
	},
}

// DetectarEsquema devuelve el esquema lógico de una fila de títulos conocida.
func DetectarEsquema(cabeceras []string) (EsquemaExportacion, error) {
	esquema, _, err := DetectarFormato(cabeceras)
	return esquema, err
}

// DetectarFormato exige que la fila de títulos coincida, columna a columna y
// en orden, con uno de los conjuntos cerrados de cabeceras. La única
// transformación admitida es la normalización Unicode NFC: une formas
// canónicamente equivalentes de la misma letra («ó» precompuesta frente a
// «o» + tilde combinante), que son el mismo texto y que distintas
// herramientas pueden guardar de forma distinta. No se pliegan mayúsculas, ni
// se quitan tildes, ni se recortan o colapsan espacios, ni se aplica NFKC:
// cualquier otra diferencia deja la exportación como desconocida. Los
// conjuntos son disjuntos, así que la detección no es ambigua y una fila que
// mezcle literales de dos formatos no coincide con ninguno.
func DetectarFormato(cabeceras []string) (EsquemaExportacion, FormatoCabeceras, error) {
	for _, formato := range formatosCabeceras {
		for _, esquema := range []EsquemaExportacion{EsquemaResumenPersona, EsquemaDetalleMerito} {
			if cabecerasCoinciden(cabeceras, cabecerasPorFormato[formato][esquema]) {
				return esquema, formato, nil
			}
		}
	}
	return "", "", ErrEsquemaExportacionDesconocido
}

func cabecerasCoinciden(recibidas, esperadas []string) bool {
	if len(recibidas) != len(esperadas) {
		return false
	}
	for i := range esperadas {
		if !utf8.ValidString(recibidas[i]) || norm.NFC.String(recibidas[i]) != esperadas[i] {
			return false
		}
	}
	return true
}

// ValidarNombreHoja comprueba el nombre de la hoja con el formato detectado.
// En convoca:v2 el nombre es parte del esquema y debe coincidir literalmente
// (tras NFC) con el de su tipo. En convoca:v1 no se fijó nombre y se admite
// cualquiera, salvo el nombre real de la otra exportación: una hoja llamada
// como el detalle con cabeceras de resumen, o al revés, se rechaza.
func ValidarNombreHoja(esquema EsquemaExportacion, formato FormatoCabeceras, nombre string) error {
	if esquema.Validar() != nil || !utf8.ValidString(nombre) {
		return ErrEsquemaExportacionDesconocido
	}
	nombre = norm.NFC.String(nombre)
	switch formato {
	case FormatoConvocaV2:
		if nombre != esquema.NombreHoja(FormatoConvocaV2) {
			return ErrEsquemaExportacionDesconocido
		}
		return nil
	case FormatoConvocaV1:
		for otro, real := range nombresHojaPorFormato[FormatoConvocaV2] {
			if otro != esquema && nombre == real {
				return ErrEsquemaExportacionDesconocido
			}
		}
		return nil
	default:
		return ErrEsquemaExportacionDesconocido
	}
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
	Esquema    EsquemaExportacion
	Cabeceras  []string
	NombreHoja string
	Filas      []FilaStaging
}

func (h HojaStaging) ValidarEstructura() error {
	if h.Esquema.Validar() != nil || len(h.Cabeceras) != h.Esquema.NumeroColumnas() {
		return ErrHojaStagingInvalida
	}
	detectado, formato, err := DetectarFormato(h.Cabeceras)
	if err != nil || detectado != h.Esquema ||
		ValidarNombreHoja(detectado, formato, h.NombreHoja) != nil {
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
	CategoriaRef         string
	BolsaRef             string
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
		!referenciaCustodia.MatchString(a.CategoriaRef) || (a.BolsaRef != "" && !referenciaCustodia.MatchString(a.BolsaRef)) ||
		a.ActaRef != "acta:importacion-convoca:"+ReferenciaContexto(a.HuellaFicheroSHA256, a.CategoriaRef) ||
		a.ImportacionRef != "importacion:convoca:"+ReferenciaContexto(a.HuellaFicheroSHA256, a.CategoriaRef) ||
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
	if a.CategoriaRef != otra.CategoriaRef || a.BolsaRef != otra.BolsaRef || a.ActaRef != otra.ActaRef ||
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

func ReferenciaContexto(huella, categoria string) string {
	s := sha256.Sum256([]byte(huella + "\x1f" + categoria))
	return hex.EncodeToString(s[:])
}
