# Paquetes compartidos

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/shared/baremacion`

> Package baremacion contiene valores exactos y neutrales compartidos por los motores de baremacion.

Package baremacion contiene valores exactos y neutrales compartidos por los
motores de baremacion. No contiene reglas de una convocatoria concreta.

### Constantes

```go
const (
	// AnioCivilMinimo es el primer anio representable.
	AnioCivilMinimo = 1
	// AnioCivilMaximo mantiene la forma canonica de cuatro cifras.
	AnioCivilMaximo = 9999
)
const (
	// MicropuntosPorPunto fija seis decimales exactos por punto.
	MicropuntosPorPunto int64 = 1_000_000
	// MaximoMicropuntos es un limite tecnico defensivo, no un tope de baremo.
	MaximoMicropuntos int64 = 9_000_000_000_000_000
)
const MaximoComponenteRacional int64 = 1_000_000_000
```

MaximoComponenteRacional limita numerador y denominador ya reducidos.
Hace que los productos cruzados quepan en int64 sin depender de math/big.

### Variables

```go
var (
	ErrFueraDeLimites    = &ErrorValor{codigo: CodigoFueraDeLimites}
	ErrDenominadorCero   = &ErrorValor{codigo: CodigoDenominadorCero}
	ErrValorNoCanonico   = &ErrorValor{codigo: CodigoValorNoCanonico}
	ErrValorInvalido     = &ErrorValor{codigo: CodigoValorInvalido}
	ErrResultadoNegativo = &ErrorValor{codigo: CodigoResultadoNegativo}
	ErrDesbordamiento    = &ErrorValor{codigo: CodigoDesbordamiento}
	ErrDivisionPorCero   = &ErrorValor{codigo: CodigoDivisionPorCero}
	ErrResultadoNoExacto = &ErrorValor{codigo: CodigoResultadoNoExacto}
	ErrFechaInvalida     = &ErrorValor{codigo: CodigoFechaInvalida}
	ErrIntervaloVacio    = &ErrorValor{codigo: CodigoIntervaloVacio}
)
```

### Tipos

```go
type CodigoError string
```

CodigoError identifica de forma estable la causa de rechazo de un valor.

```go
const (
	CodigoFueraDeLimites    CodigoError = "fuera_de_limites"
	CodigoDenominadorCero   CodigoError = "denominador_cero"
	CodigoValorNoCanonico   CodigoError = "valor_no_canonico"
	CodigoValorInvalido     CodigoError = "valor_invalido"
	CodigoResultadoNegativo CodigoError = "resultado_negativo"
	CodigoDesbordamiento    CodigoError = "desbordamiento"
	CodigoDivisionPorCero   CodigoError = "division_por_cero"
	CodigoResultadoNoExacto CodigoError = "resultado_no_exacto"
	CodigoFechaInvalida     CodigoError = "fecha_invalida"
	CodigoIntervaloVacio    CodigoError = "intervalo_vacio"
)
type ErrorValor struct {
	// Has unexported fields.
}
```

ErrorValor es un error de dominio sin datos de entrada potencialmente
sensibles. Codigo permite clasificarlo sin depender del texto de Error.

```go
func (e *ErrorValor) Codigo() CodigoError
```

Codigo devuelve la clasificacion estable del error.

```go
func (e *ErrorValor) Error() string

func (e *ErrorValor) Is(objetivo error) bool
```

Is permite usar errors.Is con los errores centinela del paquete.

```go
func (e *ErrorValor) Tipo() string
```

Tipo devuelve el valor que no pudo construirse u operar.

```go
type FechaCivil struct {
	// Has unexported fields.
}
```

FechaCivil es una fecha del calendario gregoriano sin hora ni zona horaria.

```go
func NuevaFechaCivil(anio, mes, dia int) (FechaCivil, error)
```

NuevaFechaCivil valida los componentes sin asociarlos a una zona horaria.

```go
func (f FechaCivil) Anio() int
```

Anio devuelve el anio civil.

```go
func (f FechaCivil) Comparar(otra FechaCivil) (int, error)
```

Comparar devuelve -1, 0 o 1 segun el orden civil.

```go
func (f FechaCivil) Dia() int
```

Dia devuelve el dia del mes civil.

```go
func (f FechaCivil) DiasHasta(otra FechaCivil) (int64, error)
```

DiasHasta calcula otra-f. Puede devolver un numero negativo.

```go
func (f FechaCivil) EsValida() bool
```

EsValida indica si la fecha satisface el calendario gregoriano soportado.

```go
func (f FechaCivil) MarshalJSON() ([]byte, error)

func (f FechaCivil) Mes() int
```

Mes devuelve el mes civil, entre 1 y 12.

```go
func (f FechaCivil) Siguiente() (FechaCivil, error)
```

Siguiente devuelve el dia civil inmediato sin crear una hora intermedia.

```go
func (f FechaCivil) String() string

func (f *FechaCivil) UnmarshalJSON(datos []byte) error

type FraccionJornada struct {
	// Has unexported fields.
}
```

FraccionJornada representa de forma exacta una jornada mayor que cero y no
superior a la completa. Un tercio es 1/3, nunca una aproximacion decimal.

```go
func JornadaCompleta() FraccionJornada
```

JornadaCompleta devuelve la fraccion canonica 1/1.

```go
func NuevaFraccionJornada(numerador, denominador int64) (FraccionJornada, error)
```

NuevaFraccionJornada reduce la fraccion y exige 0 < valor <= 1.

```go
func (f FraccionJornada) Denominador() int64
```

Denominador devuelve el denominador canonico.

```go
func (f FraccionJornada) EsCompleta() bool
```

EsCompleta indica si la jornada es exactamente 1/1.

```go
func (f FraccionJornada) EsValida() bool
```

EsValida comprueba que la fraccion canonica esta en el intervalo (0,1].

```go
func (f FraccionJornada) MarshalJSON() ([]byte, error)

func (f FraccionJornada) Numerador() int64
```

Numerador devuelve el numerador canonico.

```go
func (f FraccionJornada) Racional() Racional
```

Racional devuelve el valor exacto para operaciones del motor.

```go
func (f FraccionJornada) String() string

func (f *FraccionJornada) UnmarshalJSON(datos []byte) error

type IntervaloCivil struct {
	// Has unexported fields.
}
```

IntervaloCivil es siempre semiabierto: Desde se incluye y Hasta se excluye.
Esta forma evita dobles conteos al unir periodos adyacentes.

```go
func IntervaloDeUnDia(dia FechaCivil) (IntervaloCivil, error)
```

IntervaloDeUnDia construye [dia, dia+1). El 9999-12-31 no es representable
como intervalo de un dia porque su extremo exclusivo quedaria fuera de
FechaCivil; en ese caso devuelve ErrFueraDeLimites.

```go
func NuevoIntervaloCivil(desde, hasta FechaCivil) (IntervaloCivil, error)
```

NuevoIntervaloCivil exige dos fechas validas y desde < hasta.

```go
func (i IntervaloCivil) Contiene(fecha FechaCivil) bool
```

Contiene aplica desde <= fecha < hasta.

```go
func (i IntervaloCivil) Desde() FechaCivil
```

Desde devuelve el extremo inclusivo.

```go
func (i IntervaloCivil) EsAdyacente(otro IntervaloCivil) bool
```

EsAdyacente detecta extremos coincidentes sin confundirlos con un solape.

```go
func (i IntervaloCivil) EsValido() bool
```

EsValido comprueba la forma semiabierta no vacia.

```go
func (i IntervaloCivil) Hasta() FechaCivil
```

Hasta devuelve el extremo exclusivo.

```go
func (i IntervaloCivil) MarshalJSON() ([]byte, error)

func (i IntervaloCivil) NumeroDias() (int64, error)
```

NumeroDias devuelve hasta-desde; siempre es positivo para un intervalo
valido.

```go
func (i IntervaloCivil) Solapa(otro IntervaloCivil) bool
```

Solapa detecta una interseccion de al menos un dia civil.

```go
func (i *IntervaloCivil) UnmarshalJSON(datos []byte) error

type ModoRedondeo string
```

ModoRedondeo fija como se convierte un resultado racional positivo a
micropuntos. La regla publicada debe elegirlo de forma expresa; el motor no
aplica una opcion implicita.

```go
const (
	RedondeoExacto      ModoRedondeo = "exacto"
	RedondeoTruncar     ModoRedondeo = "truncar"
	RedondeoHaciaArriba ModoRedondeo = "hacia_arriba"
	RedondeoMitadArriba ModoRedondeo = "mitad_arriba"
	RedondeoMitadAlPar  ModoRedondeo = "mitad_al_par"
)
func (m ModoRedondeo) EsValido() bool
```

EsValido impide que una cadena configurada desde administracion seleccione
un comportamiento no revisado por el dominio.

```go
type Puntos struct {
	// Has unexported fields.
}
```

Puntos representa una puntuacion no negativa en micropuntos. Su campo es
privado para que no pueda existir un valor fuera de limites.

```go
func PuntosDesdeMicropuntos(micropuntos int64) (Puntos, error)
```

PuntosDesdeMicropuntos construye una puntuacion exacta.

```go
func (p Puntos) Comparar(otros Puntos) (int, error)
```

Comparar devuelve -1, 0 o 1.

```go
func (p Puntos) EsValido() bool
```

EsValido comprueba tambien valores obtenidos por una decodificacion hostil.

```go
func (p Puntos) MarshalJSON() ([]byte, error)
```

MarshalJSON usa una cadena decimal: preserva enteros superiores al rango
exacto de Number en JavaScript y produce una unica representacion.

```go
func (p Puntos) Micropuntos() int64
```

Micropuntos devuelve la representacion entera exacta.

```go
func (p Puntos) MultiplicarExacto(factor Racional) (Puntos, error)
```

MultiplicarExacto aplica un factor racional sin introducir una politica de
redondeo. Si el resultado no cabe en un micropunto, la regla llamante debe
elegir de forma explicita su politica y no puede perder la fraccion aqui.

```go
func (p Puntos) MultiplicarRedondeado(factor Racional, modo ModoRedondeo) (Puntos, error)
```

MultiplicarRedondeado aplica un factor racional no negativo y redondea una
sola vez a micropuntos. La descomposicion evita multiplicar directamente un
valor de hasta 9e15 por el numerador del factor.

```go
func (p Puntos) Restar(otros Puntos) (Puntos, error)
```

Restar rechaza las puntuaciones negativas.

```go
func (p Puntos) String() string

func (p Puntos) Sumar(otros Puntos) (Puntos, error)
```

Sumar falla de forma cerrada antes de superar el limite del tipo.

```go
func (p *Puntos) UnmarshalJSON(datos []byte) error
```

UnmarshalJSON acepta exclusivamente la cadena decimal canonica.

```go
type Racional struct {
	// Has unexported fields.
}
```

Racional es una fraccion canonica: denominador positivo y componentes
coprimos. El cero se representa exclusivamente como 0/1.

```go
func NuevoRacional(numerador, denominador int64) (Racional, error)
```

NuevoRacional normaliza el signo y reduce la fraccion.

```go
func (r Racional) Comparar(otro Racional) (int, error)
```

Comparar devuelve -1, 0 o 1 mediante productos cruzados seguros.

```go
func (r Racional) Denominador() int64
```

Denominador devuelve el denominador canonico, siempre positivo.

```go
func (r Racional) Dividir(otro Racional) (Racional, error)
```

Dividir conserva exactitud y rechaza de forma tipada la division por cero.

```go
func (r Racional) EsValido() bool
```

EsValido verifica reduccion, signo y limites defensivos.

```go
func (r Racional) MarshalJSON() ([]byte, error)

func (r Racional) Multiplicar(otro Racional) (Racional, error)
```

Multiplicar cancela factores antes del producto para reducir el riesgo de
desbordamiento y aplica despues el limite canonico.

```go
func (r Racional) Numerador() int64
```

Numerador devuelve el numerador canonico con signo.

```go
func (r Racional) Restar(otro Racional) (Racional, error)
```

Restar conserva exactitud y rechaza resultados fuera del limite defensivo.

```go
func (r Racional) String() string

func (r Racional) Sumar(otro Racional) (Racional, error)
```

Sumar conserva exactitud y rechaza resultados fuera del limite defensivo.

```go
func (r *Racional) UnmarshalJSON(datos []byte) error
```

## Paquete `internal/shared/i18n`

> Catalogo de internacionalizacion compartido con fallback espanol.

### Constantes

```go
const (
	DefaultLocale = "es"
)
```

### Variables

```go
var (
	ErrNoLocalesFound = errors.New("i18n: no locale files found")
	ErrLocaleRequired = errors.New("i18n: locale is required")
	ErrKeyRequired    = errors.New("i18n: key is required")
)
```

### Tipos

```go
type Catalog struct {
	// Has unexported fields.
}

func Load(opts ...Option) (*Catalog, error)

func LoadDir(dir string, opts ...Option) (*Catalog, error)

func LoadFS(fsys fs.FS, dir string, opts ...Option) (*Catalog, error)

func New(defaultLocale string, messages map[string]map[string]string) (*Catalog, error)

func (c *Catalog) DefaultLocale() string

func (c *Catalog) Locales() []string

func (c *Catalog) Lookup(locale, key string) (string, bool)

func (c *Catalog) Message(locale, key string) (string, bool)

func (c *Catalog) T(locale, key string) string

func (c *Catalog) Translate(locale, key string) string

type Option func(*config)

func WithDefaultLocale(locale string) Option
```

## Paquete `internal/vec/canonico/almacen`

> Package almacen concentra las reglas puras y deterministas del contrato de almacenamiento.

Package almacen concentra las reglas puras y deterministas del contrato de
almacenamiento. No conoce puertos, adaptadores ni errores de transporte.

### Constantes

```go
const (
	DuracionMaximaInstruccionesCargaDirecta = 10 * time.Minute
	LongitudMaximaDestinoCargaDirecta       = 8192
	MaximoCabecerasCargaDirecta             = 32
	MaximoOrigenesCargaDirecta              = 32
)
const (
	AccionEscribir               = "escribir"
	AccionLeer                   = "leer"
	AccionPrepararCargaDirecta   = "preparar_carga_directa"
	AccionConfirmarCargaDirecta  = "confirmar_carga_directa"
	AccionAbandonarCargaDirecta  = "abandonar_carga_directa"
	AccionPromover               = "promover"
	AccionAplicarRetencion       = "aplicar_retencion"
	AccionInmovilizar            = "inmovilizar"
	AccionLevantarInmovilizacion = "levantar_inmovilizacion"
	AccionEliminar               = "eliminar"
	AccionAnalizarContenido      = "analizar_contenido"
)
```

Las acciones son una lista positiva cerrada de operaciones tecnicas.

### Variables

```go
var (
	ErrSolicitudAlmacenInvalida              = errors.New("vec: solicitud de almacen invalida")
	ErrInstruccionesCargaDirectaNoValidas    = errors.New("vec: instrucciones de carga directa no validas")
	ErrReciboCargaDirectaNoValido            = errors.New("vec: recibo de carga directa no valido")
	ErrSerializacionReciboCargaProhibida     = errors.New("vec: serializacion accidental del recibo de carga directa prohibida")
	ErrSerializacionSeudonimizacionProhibida = errors.New("vec: serializacion accidental de seudonimizacion de almacen prohibida")
	ErrSeudonimizacionAlmacenNoDisponible    = errors.New("vec: seudonimizacion de sujeto para almacen no disponible")
	ErrSerializacionContextoAlmacenProhibida = errors.New("vec: serializacion de contexto de almacen prohibida")
)
```

### Funciones

```go
func AccionCreaObjeto(accion string) bool
func AccionIdempotente(accion string) bool
func AccionOperacionValida(accion string) bool
func AccionResultadoValida(accion string) bool
func CabecerasCargaDirectaValidas(cabeceras []CabeceraCargaDirecta) bool
func CapacidadesSatisfacen(capacidades Capacidades, requisitos Requisitos) bool
```

CapacidadesSatisfacen aplica la lista cerrada de requisitos de despliegue.

```go
func DestinoCargaDirectaValido(destino string) bool
func HMACSHA256Valido(valor string) bool
func HuellaPreparacionCargaDirecta(datos DatosHuellaPreparacionCargaDirecta) string
```

HuellaPreparacionCargaDirecta usa campos en orden fijo y prefijo decimal de
longitud para que ninguna concatenacion ambigua produzca la misma huella.

```go
func InstruccionesCargaDirectaValidas(datos DatosInstruccionesCargaDirecta) bool
```

InstruccionesCargaDirectaValidas valida la concesion completa sin revelar ni
transformar ninguno de sus valores.

```go
func LigaduraExacta(observada, esperada []string) bool
```

LigaduraExacta exige igual cardinalidad, orden y valor. Se usa con
proyecciones de campos declaradas en el puerto para impedir coincidencias
parciales o por subconjunto entre autorizacion, solicitud y evidencia.

```go
func NombreCabeceraCargaDirectaValido(nombre string) bool
func OrigenDestinoCargaDirecta(destino string) string
func OrigenesCargaDirectaValidos(origenes []string) bool
func ReferenciaOpacaValida(valor string, maximo int) bool
func SHA256HexadecimalValido(valor string) bool
func TextoSeguro(valor string, maximo int) bool
func ValidarDatosComprobanteConsumoReciboCargaDirecta(d DatosComprobanteConsumoReciboCargaDirecta) error
```

ValidarDatosComprobanteConsumoReciboCargaDirecta aplica las mismas reglas a
una proyeccion procedente de un contrato que mantiene su envoltorio opaco.

```go
func ValidarEvidenciaOperacion(datos DatosEvidenciaOperacionAlmacen) error
func ValidarResultadoMutacion(
	resultado DatosResultadoOperacionObjeto,
	anterior ObjetoAlmacenado,
	accion, fundamentoRef string,
	evidenciaLigada bool,
) error
func ValidarResultadoOperacion(datos DatosResultadoOperacionObjeto) error
func ValorCabeceraCargaDirectaValido(valor string) bool
```

### Tipos

```go
type CabeceraCargaDirecta struct {
	Nombre string
	Valor  string
}

type Capacidades struct {
	ConectorID                  string
	EscrituraEnFlujo            bool
	LecturaEnFlujo              bool
	ReferenciasOpacas           bool
	IntegridadSHA256            bool
	Versionado                  bool
	Retencion                   bool
	BloqueoLegal                bool
	PromocionAtomica            bool
	RetencionAtomicaEnPromocion bool
	CargaDirectaTemporal        bool
	CifradoEnTransito           bool
	CifradoEnReposo             bool
	CifradoPorObjeto            bool
	TamanoMaximoObjeto          int64
	PreservaObjetoOriginal      bool
	OrigenesCargaDirecta        []string
}

type ComprobanteConsumoReciboCargaDirecta struct {
	// Has unexported fields.
}
```

ComprobanteConsumoReciboCargaDirecta conserva de forma opaca la atestacion
del consumo durable; su forma valida no prueba por si sola la autenticidad.

```go
func NuevoComprobanteConsumoReciboCargaDirecta(
	resultado ResultadoConsumoReciboCargaDirecta,
	validaHasta time.Time,
	atestacionHMAC string,
) (ComprobanteConsumoReciboCargaDirecta, error)

func (c ComprobanteConsumoReciboCargaDirecta) DatosVerificados() (
	DatosComprobanteConsumoReciboCargaDirecta,
	error,
)

func (c ComprobanteConsumoReciboCargaDirecta) Format(estado fmt.State, _ rune)

func (ComprobanteConsumoReciboCargaDirecta) GoString() string

func (c ComprobanteConsumoReciboCargaDirecta) LogValue() slog.Value

func (ComprobanteConsumoReciboCargaDirecta) MarshalJSON() ([]byte, error)

func (ComprobanteConsumoReciboCargaDirecta) MarshalText() ([]byte, error)

func (c ComprobanteConsumoReciboCargaDirecta) RevelarParaVerificacion() (
	indiceHMAC, grupoHMAC, vinculoHMAC, evidenciaConsumoRef, intencionRef, huellaIntencionHMAC string,
	registradoEn, consumidoEn, expiraEn, validaHasta time.Time,
	atestacionHMAC string,
	err error,
)

func (ComprobanteConsumoReciboCargaDirecta) String() string

func (c ComprobanteConsumoReciboCargaDirecta) ValidarEstructura() error

type DatosComprobanteConsumoReciboCargaDirecta struct {
	IndiceReciboHMAC    string
	GrupoReciboHMAC     string
	VinculoReciboHMAC   string
	EvidenciaConsumoRef string
	IntencionRef        string
	HuellaIntencionHMAC string
	RegistradoEn        time.Time
	ConsumidoEn         time.Time
	ExpiraEn            time.Time
	ValidaHasta         time.Time
	AtestacionHMAC      string
}
```

DatosComprobanteConsumoReciboCargaDirecta es una copia validada sin recibo
ni sesion. No permite reconstruir o alterar el comprobante opaco.

```go
type DatosEvidenciaOperacionAlmacen struct {
	Referencia             string
	ConectorID             string
	EsquemaContexto        string
	EsquemaEsperado        string
	AccionNegocio          string
	Accion                 string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	HuellaManifiestoSHA256 string
	HuellaPasoSHA256       string
	PasoRef                string
	HuellaDecisionSHA256   string
	Objeto                 ReferenciaObjetoAlmacen
	OperacionRef           string
	CorrelacionRef         string
	AutorizacionRef        string
	Finalidad              string
	Clasificacion          string
	RealizadaEn            time.Time
	CargaRef               string
	SujetoSeudonimoHMAC    string
	RecursoRef             string
	ModuloID               string
	HuellaSolicitudHMAC    string
	FundamentoRef          string
	ReintentoIdempotente   bool
}
```

DatosEvidenciaOperacionAlmacen es la proyeccion escalar de una evidencia
tecnica. El puerto conserva el tipo de paso y aporta su representacion.

```go
type DatosHuellaPreparacionCargaDirecta struct {
	Esquema                string
	OperacionRef           string
	CorrelacionRef         string
	AutorizacionRef        string
	Finalidad              string
	Clasificacion          string
	AccionNegocio          string
	AccionTecnica          string
	CargaRef               string
	SujetoSeudonimoHMAC    string
	RecursoRef             string
	ModuloID               string
	HuellaSolicitudHMAC    string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	PasoRef                string
	HuellaDecisionSHA256   string
	ClaveIdempotencia      string
	MIME                   string
	Tamano                 int64
	HuellaSHA256           string
	ExpiraEn               time.Time
}

type DatosInstruccionesCargaDirecta struct {
	ConectorID   string
	SesionRef    string
	Metodo       MetodoCargaDirecta
	Destino      string
	Cabeceras    []CabeceraCargaDirecta
	EmitidaEn    time.Time
	ExpiraEn     time.Time
	TamanoMaximo int64
}

type DatosResultadoOperacionObjeto struct {
	Objeto    ObjetoAlmacenado
	Evidencia DatosEvidenciaOperacionAlmacen
}
```

DatosResultadoOperacionObjeto agrupa el estado material y su evidencia sin
introducir una dependencia desde el paquete canónico hacia los puertos.

```go
type InstruccionesCargaDirecta struct {
	// Has unexported fields.
}
```

InstruccionesCargaDirecta custodia una concesion temporal y su vinculo sin
exponer referencias mutables ni credenciales a serializadores genericos.

```go
func NuevasInstruccionesCargaDirecta(
	datos DatosInstruccionesCargaDirecta,
) (InstruccionesCargaDirecta, error)

func (i InstruccionesCargaDirecta) DatosVerificados() (
	DatosInstruccionesCargaDirecta,
	string,
	error,
)
```

DatosVerificados devuelve una copia defensiva y el vinculo inmutable solo
despues de validar la forma completa de la concesion.

```go
func (i InstruccionesCargaDirecta) Validar() error

func (i InstruccionesCargaDirecta) ValidarContra(capacidades Capacidades) error

func (i InstruccionesCargaDirecta) ValidarPara(
	tamano int64,
	expiraEn time.Time,
	huellaSolicitudSHA256 string,
	capacidades Capacidades,
) error

func (i InstruccionesCargaDirecta) VigenteEn(instante time.Time) bool

func (i InstruccionesCargaDirecta) VincularSolicitud(huellaSHA256 string) (InstruccionesCargaDirecta, error)
```

VincularSolicitud devuelve una nueva concesion ligada a la huella exacta de
la preparacion; nunca modifica el valor del que parte.

```go
type MetodoCargaDirecta string

const (
	MetodoCargaDirectaPUT  MetodoCargaDirecta = "PUT"
	MetodoCargaDirectaPOST MetodoCargaDirecta = "POST"
)
func (m MetodoCargaDirecta) Valido() bool

type ObjetoAlmacenado struct {
	Objeto               ReferenciaObjetoAlmacen
	ConectorID           string
	Zona                 Zona
	MIME                 string
	Tamano               int64
	HuellaSHA256         string
	EvidenciaCreacionRef string
	AlmacenadoEn         time.Time
	RetenidoHasta        time.Time
	Inmovilizado         bool
	Eliminado            bool
}
```

ObjetoAlmacenado es la proyeccion tecnica verificable de un objeto.

```go
func (o ObjetoAlmacenado) Validar() error

type OrdenConsumoReciboCargaDirecta struct {
	IndiceHMAC               string
	GrupoHMAC                string
	VinculoHMAC              string
	EvidenciaConsumoRef      string
	IntencionConfirmacionRef string
	HuellaIntencionHMAC      string
	RegistradoEn             time.Time
	ValidaHasta              time.Time
}
```

OrdenConsumoReciboCargaDirecta contiene las ligaduras exactas de la
escritura condicional que consume un recibo.

```go
func (o OrdenConsumoReciboCargaDirecta) Validar() error

type PasoOperacionAlmacen string
```

PasoOperacionAlmacen identifica un paso cerrado comprometido por el nucleo.

```go
const (
	PasoPrepararCargaDirecta  PasoOperacionAlmacen = "01_preparar_carga_directa"
	PasoAbandonarCargaDirecta PasoOperacionAlmacen = "02_abandonar_carga_directa"
	PasoConfirmarCargaDirecta PasoOperacionAlmacen = "01_confirmar_carga_directa"
	PasoLeerParaAnalisis      PasoOperacionAlmacen = "01_leer_para_analisis"
	PasoAnalizarContenido     PasoOperacionAlmacen = "02_analizar_contenido"
	PasoPromover              PasoOperacionAlmacen = "01_promover"
	PasoCustodiarDecision     PasoOperacionAlmacen = "01_custodiar_decision"
	PasoCustodiarFirmado      PasoOperacionAlmacen = "01_custodiar_documento_firmado"
	PasoRetenerFirmado        PasoOperacionAlmacen = "01_retener_documento_firmado"
)
type PredecesorReciboCargaDirecta struct {
	IndiceHMAC             string
	GrupoHMAC              string
	AutorizacionEmisionRef string
	SustituidoEn           time.Time
}
```

PredecesorReciboCargaDirecta liga el recibo sustituido en la misma
transaccion que registra el nuevo recibo activo del grupo.

```go
func (p PredecesorReciboCargaDirecta) Validar() error

type ProyeccionContextoOperacionAlmacen struct {
	Esquema                string
	OperacionRef           string
	CorrelacionRef         string
	AutorizacionRef        string
	Finalidad              string
	Clasificacion          string
	AccionNegocio          string
	AccionTecnica          string
	CargaRef               string
	SujetoSeudonimoHMAC    string
	RecursoRef             string
	ModuloID               string
	TipoRecurso            string
	HuellaRecursoSHA256    string
	HuellaSolicitudHMAC    string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	HuellaManifiestoSHA256 string
	HuellaPasoSHA256       string
	PasoRef                PasoOperacionAlmacen
	ObjetoVinculado        ReferenciaObjetoAlmacen
	HuellaDecisionSHA256   string
	VerificadaEn           time.Time
	ValidaHasta            time.Time
}
```

ProyeccionContextoOperacionAlmacen es una copia defensiva que no permite
reconstruir la capacidad opaca de la que procede.

```go
func (p ProyeccionContextoOperacionAlmacen) Format(estado fmt.State, _ rune)

func (p ProyeccionContextoOperacionAlmacen) GoString() string

func (p ProyeccionContextoOperacionAlmacen) LogValue() slog.Value

func (ProyeccionContextoOperacionAlmacen) MarshalJSON() ([]byte, error)

func (ProyeccionContextoOperacionAlmacen) MarshalText() ([]byte, error)

func (ProyeccionContextoOperacionAlmacen) String() string

func (*ProyeccionContextoOperacionAlmacen) UnmarshalJSON([]byte) error

func (*ProyeccionContextoOperacionAlmacen) UnmarshalText([]byte) error

type ReciboCargaDirecta struct {
	// Has unexported fields.
}
```

ReciboCargaDirecta es un secreto efimero opaco de un solo uso.

```go
func NuevoReciboCargaDirecta(valor string) (ReciboCargaDirecta, error)

func (r ReciboCargaDirecta) Format(estado fmt.State, _ rune)

func (ReciboCargaDirecta) GoString() string

func (r ReciboCargaDirecta) LogValue() slog.Value

func (ReciboCargaDirecta) MarshalJSON() ([]byte, error)

func (ReciboCargaDirecta) MarshalText() ([]byte, error)

func (r ReciboCargaDirecta) RevelarParaEntregaOConsumo() (string, error)

func (ReciboCargaDirecta) String() string

func (r ReciboCargaDirecta) Valido() bool

type ReferenciaObjetoAlmacen struct {
	Referencia string
	Version    string
}
```

ReferenciaObjetoAlmacen identifica de forma opaca una version inmutable.

```go
func (r ReferenciaObjetoAlmacen) Validar() error

type RegistroReciboCargaDirecta struct {
	IndiceHMAC             string
	GrupoHMAC              string
	VinculoHMAC            string
	EvidenciaAltaRef       string
	AutorizacionEmisionRef string
	ExpiraEn               time.Time
}
```

RegistroReciboCargaDirecta es la propuesta sin fecha de alta del proceso;
el repositorio fija RegistradoEn con su reloj transaccional.

```go
func (r RegistroReciboCargaDirecta) Validar() error

type Requisitos struct {
	EscrituraEnFlujo            bool
	LecturaEnFlujo              bool
	ReferenciasOpacas           bool
	IntegridadSHA256            bool
	Versionado                  bool
	Retencion                   bool
	BloqueoLegal                bool
	PromocionAtomica            bool
	RetencionAtomicaEnPromocion bool
	CargaDirectaTemporal        bool
	CifradoEnTransito           bool
	CifradoEnReposo             bool
	CifradoPorObjeto            bool
	TamanoMinimoObjeto          int64
	PreservaObjetoOriginal      bool
}

type ResultadoConsumoReciboCargaDirecta struct {
	IndiceHMAC               string
	GrupoHMAC                string
	VinculoHMAC              string
	EvidenciaConsumoRef      string
	IntencionConfirmacionRef string
	HuellaIntencionHMAC      string
	RegistradoEn             time.Time
	ConsumidoEn              time.Time
	ExpiraEn                 time.Time
}
```

ResultadoConsumoReciboCargaDirecta acredita las fechas autoritativas y la
ligadura exacta persistida por el repositorio.

```go
func (r ResultadoConsumoReciboCargaDirecta) Validar() error

func (r ResultadoConsumoReciboCargaDirecta) ValidarContra(o OrdenConsumoReciboCargaDirecta) error

type ResultadoRegistroReciboCargaDirecta struct {
	IndiceHMAC             string
	GrupoHMAC              string
	AutorizacionEmisionRef string
	RegistradoEn           time.Time
	Predecesor             *PredecesorReciboCargaDirecta
}
```

ResultadoRegistroReciboCargaDirecta acredita el alta durable y su posible
predecesor, ambos fechados por la misma transaccion.

```go
func (r ResultadoRegistroReciboCargaDirecta) ValidarContra(registro RegistroReciboCargaDirecta) error

type SolicitudSellarIdempotenciaCarga struct {
	OperacionRef     string
	PrincipalRef     string
	RecursoRef       string
	MIME             string
	Tamano           int64
	HuellaSHA256     string
	ClaveSolicitante string
}
```

SolicitudSellarIdempotenciaCarga contiene solo los campos exactos que el
sellador debe ligar mediante una clave exclusiva del servidor.

```go
func (s SolicitudSellarIdempotenciaCarga) Validar() error

type SolicitudSeudonimizarSujetoAlmacen struct {
	// Has unexported fields.
}
```

SolicitudSeudonimizarSujetoAlmacen mantiene las referencias internas fuera
de contextos y registros hasta su revelacion deliberada al sellador local.

```go
func NuevaSolicitudSeudonimizarSujetoAlmacen(
	sujetoRef, ambitoRef string,
) (SolicitudSeudonimizarSujetoAlmacen, error)

func (s SolicitudSeudonimizarSujetoAlmacen) Format(estado fmt.State, _ rune)

func (SolicitudSeudonimizarSujetoAlmacen) GoString() string

func (s SolicitudSeudonimizarSujetoAlmacen) LogValue() slog.Value

func (SolicitudSeudonimizarSujetoAlmacen) MarshalJSON() ([]byte, error)

func (SolicitudSeudonimizarSujetoAlmacen) MarshalText() ([]byte, error)

func (s SolicitudSeudonimizarSujetoAlmacen) RevelarParaSellado() (
	sujetoRef, ambitoRef string,
	err error,
)

func (SolicitudSeudonimizarSujetoAlmacen) String() string

type Zona string

const (
	ZonaCuarentena Zona = "cuarentena"
	ZonaAdmitida   Zona = "admitida"
)
func (z Zona) Valida() bool
```

## Paquete `internal/vec/canonico/documental`

> Package documental concentra reglas puras y deterministas de la ejecucion documental.

Package documental concentra reglas puras y deterministas de la ejecucion
documental. No conoce puertos, adaptadores, persistencia ni transporte.

### Constantes

```go
const (
	AlgoritmoHMACSHA256V3                = "hmac-sha256"
	AudienciaTokenCercadoV3              = "vec.documentos.token-cercado.v3"
	AudienciaInicioEfectoV3              = "vec.documentos.inicio-efecto.v3"
	AudienciaReclamacionDespachoV3       = "vec.documentos.reclamacion-despacho.v3"
	AudienciaComprobacionOrdenDespachoV3 = "vec.documentos.comprobacion-orden-despacho.v3"
	ContextoTokenCercadoV3               = "cercado"
	ContextoInicioEfectoV3               = "inicio"
	ContextoReclamacionDespachoV3        = "reclamacion"
	ContextoComprobacionOrdenDespachoV3  = "inicio-reclamacion-cercada"
	TamanoFirmaHMACSHA256V3              = 32
)
const (
	DuracionMaximaReclamacionDespachoV3 = 10 * time.Minute
)
const (
	// EsquemaCanonizacionEntradaNeutralV1 identifica el codec cerrado de la
	// entrada neutral documental. Cambiarlo exige una nueva version del codec.
	EsquemaCanonizacionEntradaNeutralV1 = "vec.documentos.entrada-neutral.contenido-longitud-prefijada.v1"
)
const AudienciaSelloEvidenciaRenderizadoV3 = "vec.documentos.evidencia-renderizado.v3"
```

### Variables

```go
var (
	ErrOrdenDespachoDocumentalV3Invalida  = errors.New("vec: orden de despacho documental v3 invalida")
	ErrTokenCercadoDocumentalV3Invalido   = errors.New("vec: token de cercado documental v3 invalido")
	ErrSelloEvidenciaDocumentalV3Invalido = errors.New("vec: sello de evidencia documental v3 invalido")
	ErrReconciliacionDocumentalV3Invalida = errors.New("vec: reconciliacion documental v3 invalida")
	ErrSerializacionSecretoDocumentalV3   = errors.New("vec: serializacion de secreto documental v3 prohibida")
)
```

### Funciones

```go
func BytesIguales(primero, segundo []byte) bool
```

BytesIguales conserva la comparacion exacta de las preimagenes canonicas.

```go
func BytesNoNulos(valor []byte) bool
```

BytesNoNulos exige al menos un octeto distinto de cero.

```go
func CanonizarEntradaNeutralV1(titulo string, parrafos []string) ([]byte, bool)
```

CanonizarEntradaNeutralV1 fija titulo, cardinalidad y parrafos mediante
campos de longitud prefijada en bytes. Conserva el orden, no normaliza
Unicode y rechaza controles salvo tabulador y saltos de linea. El booleano
es falso cuando la entrada no pertenece al dominio cerrado del codec.

```go
func ClaveHMACSHA256V3(valor string) string
```

ClaveHMACSHA256V3 proyecta el identificador de clave de una huella con
forma hmac-sha256:<clave>:<digest>. La validacion del digest corresponde al
contrato que consume esta proyeccion.

```go
func ClavesHMACSHA256V3Distintas(valores ...string) bool
```

ClavesHMACSHA256V3Distintas impide reutilizar una clave entre dominios
criptograficos que deben permanecer independientes.

```go
func DNINIEASCIIMinusculoEvidente(valor string) bool
```

DNINIEASCIIMinusculoEvidente equivale exactamente a
^(?:[0-9]{8}[a-z]|[xyz][0-9]{7}[a-z])$.

```go
func HMACSHA256V3Valido(valor string) bool
```

HMACSHA256V3Valido comprueba algoritmo, referencia de clave y digest.

```go
func HuellaBytesSHA256(contenido []byte) string
```

HuellaBytesSHA256 devuelve la codificacion hexadecimal minuscula de SHA-256.

```go
func HuellaCamposSHA256V3(valores []string) string
```

HuellaCamposSHA256V3 deriva la huella hexadecimal canonica de una lista de
campos. Comparte exactamente la misma preimagen que SerializarCamposV3.

```go
func HuellaSolicitudVerificacionEvidenciaV3(
	huellaMensaje string,
	datos DatosSelloEvidenciaV3,
) string
```

HuellaSolicitudVerificacionEvidenciaV3 liga el mensaje firmado, el perfil,
la firma nominal y su evidencia operativa en el orden historico V3.

```go
func HuellaSolicitudVerificacionReconciliacionV3(
	mensaje []byte,
	huellaAtestacionSHA256 string,
) string
```

HuellaSolicitudVerificacionReconciliacionV3 liga la preimagen exacta con el
COSE declarado e impide reutilizar una comprobacion con otro sobre.

```go
func HuellaSolicitudVerificacionTokenV3(mensaje, mac []byte) string
func HuellaVinculoCercadoV3(secuencia uint64, huellaVinculoEstable string) string
func HuellasSHA256Distintas(huellas ...string) bool
```

HuellasSHA256Distintas exige forma hexadecimal canonica y ausencia de
reutilizacion entre compromisos que representan finalidades distintas.
La coleccion vacia satisface la condicion de forma vacua, igual que el
contrato historico de materializacion documental.

```go
func IDClaveHMACASCIIBasicoValido(valor string) bool
```

IDClaveHMACASCIIBasicoValido equivale exactamente a
^[a-z][a-z0-9._-]{0,127}$.

```go
func InstanteV3Valido(instante time.Time) bool
```

InstanteV3Valido exige UTC, rango representable y precision de microsegundo,
que es la precision contractual de persistencia de la ejecucion documental.

```go
func PerfilAtestacionDespachoV3Valido(algoritmo, audiencia, contexto string) bool
func PreimagenEntradaNeutralV1Valida(
	titulo string,
	parrafos []string,
	preimagen []byte,
	tamano uint64,
) bool
```

PreimagenEntradaNeutralV1Valida comprueba de nuevo el dominio y la
correspondencia byte a byte de una preimagen ya fijada. Mantiene privados
los limites del codec y permite que el puerto solo traduzca el resultado a
su error nominal, sin duplicar reglas de canonizacion.

```go
func ReferenciaASCIIBasicaValida(valor string) bool
```

ReferenciaASCIIBasicaValida equivale exactamente a
^[a-z][a-z0-9._:-]{0,255}$ sin construir un automata regular. Todos los
caracteres admitidos son ASCII de un byte, por lo que longitud de bytes y
numero de caracteres coinciden para cualquier entrada valida.

```go
func ReferenciaEjecucionV3Valida(valor string) bool
```

ReferenciaEjecucionV3Valida aplica la lista positiva de caracteres y excluye
esquemas que podrian convertir una referencia opaca en dato personal o URL.

```go
func ReferenciasEjecucionV3Distintas(valores ...string) bool
```

ReferenciasEjecucionV3Distintas exige forma opaca, cardinalidad exacta y
ausencia de reutilizacion entre referencias con finalidades distintas.

```go
func SHA256HexadecimalValido(valor string) bool
```

SHA256HexadecimalValido acepta exclusivamente 32 bytes expresados como 64
caracteres hexadecimales minusculos.

```go
func SerializarCamposV3(valores []string) []byte
```

SerializarCamposV3 fija la preimagen usada por los compromisos y
atestaciones documentales V3. La longitud se expresa en bytes, no en runas.
El formato es deliberadamente simple y no admite representaciones alternas:
<longitud-decimal>:<valor>\n para cada campo, incluido el campo vacio.

```go
func Uint64Decimal(valor uint64) string
```

Uint64Decimal fija la representacion decimal sin signo usada en preimagenes.

### Tipos

```go
type DatosAtestacionInicioEfectoV3 struct {
	InicioRef                  string
	ReservaRef                 string
	HuellaVinculoEstableSHA256 string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	OrdenConsumoDurableV4Ref   string
	VersionInicioCAS           uint64
	AuditoriaInicioRef         string
	OutboxInicioRef            string
	ClaveAtestacionRef         string
	RevisionClave              uint64
	EvidenciaOperacionRef      string
	IniciadoEn                 time.Time
}
```

DatosAtestacionInicioEfectoV3 fija todos los campos de la preimagen que
acredita el COMMIT de inicio. El puerto solo proyecta tipos ricos a este
DTO.

```go
func (d DatosAtestacionInicioEfectoV3) Bytes() []byte

func (d DatosAtestacionInicioEfectoV3) Validar() bool

type DatosAtestacionReclamacionV3 struct {
	Solicitud                  DatosSolicitudReclamacionV3
	HuellaReciboInicioSHA256   string
	InicioReciboRef            string
	OutboxInicioReciboRef      string
	IniciadoEn                 time.Time
	VersionReclamacionCAS      uint64
	AuditoriaReclamacionRef    string
	ClaveAtestacionRef         string
	RevisionClave              uint64
	EvidenciaOperacionRef      string
	SecuenciaCercado           uint64
	HuellaVinculoEstableSHA256 string
	HuellaVinculoCercadoSHA256 string
	OrdenConsumoDurableV4Ref   string
}
```

DatosAtestacionReclamacionV3 fija la preimagen del segundo CAS durable.

```go
func (d DatosAtestacionReclamacionV3) Bytes() []byte

func (d DatosAtestacionReclamacionV3) Validar() bool

type DatosMaterialDespachoV3 struct {
	Vinculos                        VinculosMaterialDespachoV3
	Cercado                         PerfilMaterialDespachoV3
	Inicio                          PerfilMaterialDespachoV3
	Reclamacion                     PerfilMaterialDespachoV3
	ClaveEsperadaRef                string
	RevisionEsperada                uint64
	HuellaOrdenEsperadaSHA256       string
	HuellaReciboEsperadaSHA256      string
	HuellaVinculoEsperadaSHA256     string
	HuellaCercadoEsperadaSHA256     string
	SecuenciaEsperada               uint64
	VersionInicioEsperada           uint64
	VersionReclamacionEsperada      uint64
	HuellaInicioEsperadaSHA256      string
	HuellaReclamacionEsperadaSHA256 string
	Mensaje                         []byte
	HuellaMensajeSHA256             string
}
```

DatosMaterialDespachoV3 reune perfiles, ligaduras esperadas y preimagen.

```go
func (d DatosMaterialDespachoV3) Bytes() []byte

func (d DatosMaterialDespachoV3) Validar() bool

type DatosMetadatosComprobacionV3 struct {
	HuellaSolicitud         string
	HuellaSolicitudEsperada string
	VerificacionRef         string
	VerificadaEn            time.Time
}
```

DatosMetadatosComprobacionV3 concentra el cotejo nominal de una
comprobacion.

```go
func (d DatosMetadatosComprobacionV3) Validar() bool

type DatosOrdenDespachoV3 struct {
	Solicitud                   DatosSolicitudReclamacionV3
	HuellaReciboInicioSHA256    string
	HuellaReciboCalculadaSHA256 string
	VersionReclamacionCAS       uint64
	AuditoriaReclamacionRef     string
	EvidenciaOperacionRef       string
	AtestacionValida            bool
	HuellaAtestacionSHA256      string
	MensajeAtestacion           []byte
	MensajeEsperado             []byte
	IniciadoEn                  time.Time
}
```

DatosOrdenDespachoV3 contiene los cotejos primitivos de la orden nominal.

```go
func (d DatosOrdenDespachoV3) HuellaSHA256() string

func (d DatosOrdenDespachoV3) Validar() bool

type DatosPruebaAtestacionDespachoV3 struct {
	Algoritmo               string
	Audiencia               string
	Contexto                string
	ClaveGestionadaRef      string
	RevisionClaveGestionada uint64
	EvidenciaOperacionRef   string
	MensajeCanonico         []byte
	SobreCriptografico      []byte
	HuellaMensajeSHA256     string
	HuellaSobreSHA256       string
}
```

DatosPruebaAtestacionDespachoV3 representa material nominal restaurable.

```go
func (d DatosPruebaAtestacionDespachoV3) Format(estado fmt.State, _ rune)

func (d DatosPruebaAtestacionDespachoV3) GoString() string

func (d DatosPruebaAtestacionDespachoV3) HuellaSHA256() string

func (d DatosPruebaAtestacionDespachoV3) LogValue() slog.Value

func (DatosPruebaAtestacionDespachoV3) MarshalBinary() ([]byte, error)

func (DatosPruebaAtestacionDespachoV3) MarshalJSON() ([]byte, error)

func (DatosPruebaAtestacionDespachoV3) MarshalText() ([]byte, error)

func (DatosPruebaAtestacionDespachoV3) String() string

func (*DatosPruebaAtestacionDespachoV3) UnmarshalBinary([]byte) error

func (*DatosPruebaAtestacionDespachoV3) UnmarshalJSON([]byte) error

func (*DatosPruebaAtestacionDespachoV3) UnmarshalText([]byte) error

func (d DatosPruebaAtestacionDespachoV3) Validar() bool

type DatosReciboInicioEfectoV3 struct {
	InicioRef                  string
	ReservaRef                 string
	HuellaVinculoEstableSHA256 string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	OrdenConsumoDurableV4Ref   string
	VersionInicioCAS           uint64
	AuditoriaInicioRef         string
	OutboxInicioRef            string
	EvidenciaOperacionRef      string
	AtestacionValida           bool
	HuellaAtestacionSHA256     string
	IniciadoEn                 time.Time
}
```

DatosReciboInicioEfectoV3 concentra forma, unicidad y huella del recibo
durable nominal sin conocer la representacion opaca del puerto.

```go
func (d DatosReciboInicioEfectoV3) HuellaSHA256() string

func (d DatosReciboInicioEfectoV3) Validar() bool

type DatosResultadoReconciliacionV3 struct {
	ReservaRef             string
	EfectoRef              string
	SecuenciaCercado       uint64
	HuellaVinculoSHA256    string
	HuellaPlanSHA256       string
	Estado                 EstadoResultadoReconciliacionV3
	Resultado              DatosResultadoRenderizadoV3
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	SobreAtestacion        SobreAtestacionReconciliacionV3
	ConsultadaEn           time.Time
}
```

DatosResultadoReconciliacionV3 concentra la forma estable que se valida y
se serializa. El sobre permanece opaco y su huella debe coincidir con la
atestacion declarada.

```go
func (d DatosResultadoReconciliacionV3) Bytes() []byte
```

Bytes fija, sin alterar su orden historico, la preimagen firmada del
resultado de reconciliacion.

```go
func (d DatosResultadoReconciliacionV3) ValidarContra(
	esperado ExpectativasResultadoReconciliacionV3,
) error

type DatosResultadoRenderizadoV3 struct {
	BorradorRef           string
	EfectoRef             string
	ContenidoRef          string
	ContenidoVersion      string
	ConectorRef           string
	MIME                  string
	HuellaSalidaSHA256    string
	TamanoSalida          uint64
	EvidenciaOperacionRef string
}
```

DatosResultadoRenderizadoV3 es la proyeccion pura comprometida por la
atestacion de reconciliacion. No contiene el documento ni una URL temporal.

```go
func (d DatosResultadoRenderizadoV3) EsCero() bool

type DatosSelloEvidenciaV3 struct {
	Algoritmo             string
	ClaveID               string
	Audiencia             string
	HuellaMensajeSHA256   string
	Firma                 []byte
	EvidenciaOperacionRef string
	FirmadoEn             time.Time
}
```

DatosSelloEvidenciaV3 transporta una firma nominal y sus ligaduras.

```go
func (d DatosSelloEvidenciaV3) Format(estado fmt.State, _ rune)

func (d DatosSelloEvidenciaV3) GoString() string

func (d DatosSelloEvidenciaV3) LogValue() slog.Value

func (DatosSelloEvidenciaV3) MarshalBinary() ([]byte, error)

func (DatosSelloEvidenciaV3) MarshalJSON() ([]byte, error)

func (DatosSelloEvidenciaV3) MarshalText() ([]byte, error)

func (DatosSelloEvidenciaV3) String() string

func (*DatosSelloEvidenciaV3) UnmarshalBinary([]byte) error

func (*DatosSelloEvidenciaV3) UnmarshalJSON([]byte) error

func (*DatosSelloEvidenciaV3) UnmarshalText([]byte) error

func (d DatosSelloEvidenciaV3) ValidarPara(
	perfil PerfilSelloEvidenciaV3,
	huellaMensaje string,
) bool

type DatosSolicitudReclamacionV3 struct {
	ReclamacionRef string
	InicioRef      string
	OutboxRef      string
	ConsumidorRef  string
	ReclamadaEn    time.Time
	ExpiraEn       time.Time
}
```

DatosSolicitudReclamacionV3 contiene la ventana CAS aportada por el
consumidor de outbox.

```go
func (d DatosSolicitudReclamacionV3) EsValida() bool

func (d DatosSolicitudReclamacionV3) Format(estado fmt.State, _ rune)

func (d DatosSolicitudReclamacionV3) GoString() string

func (d DatosSolicitudReclamacionV3) LogValue() slog.Value

func (DatosSolicitudReclamacionV3) MarshalBinary() ([]byte, error)

func (DatosSolicitudReclamacionV3) MarshalJSON() ([]byte, error)

func (DatosSolicitudReclamacionV3) MarshalText() ([]byte, error)

func (DatosSolicitudReclamacionV3) String() string

func (*DatosSolicitudReclamacionV3) UnmarshalBinary([]byte) error

func (*DatosSolicitudReclamacionV3) UnmarshalJSON([]byte) error

func (*DatosSolicitudReclamacionV3) UnmarshalText([]byte) error

func (d DatosSolicitudReclamacionV3) Validar() error

type DatosTokenCercadoV3 struct {
	Valor                       string
	Secuencia                   uint64
	HuellaVinculoEstableSHA256  string
	HuellaVinculoEsperadoSHA256 string
	HuellaVinculoInternoSHA256  string
	HuellaVinculoCercadoSHA256  string
	ClaveAtestacionRef          string
	RevisionClave               uint64
	MACAtestacion               []byte
	EvidenciaOperacionRef       string
	ClaveHuellaEntradaHMAC      string
}
```

DatosTokenCercadoV3 contiene solo primitivas y cotejos ya calculados por el
puerto. La MAC se valida estructuralmente; su autenticidad corresponde al
KMS.

```go
func (d DatosTokenCercadoV3) MensajeAtestacion() []byte

func (d DatosTokenCercadoV3) Validar() bool

type DatosVinculoActivacionV3 struct {
	ReservaRef                    string
	IndiceIdempotenciaHMAC        string
	HuellaSolicitudHMAC           string
	HuellaEntradaHMAC             string
	HuellaManifiestoSHA256        string
	EfectoManifiestoRef           string
	HuellaPlanManifiestoSHA256    string
	OrdenConsumoDurableV4Ref      string
	DecisionRef                   string
	EfectoDecisionRef             string
	EsquemaHuellaDecision         string
	EsquemaHuellaDecisionEsperado string
	HuellaDecisionSHA256          string
	HuellaPlanDecisionSHA256      string
}
```

DatosVinculoActivacionV3 es la proyeccion pura del vinculo nominal.
Duplica los valores comprometidos por el manifiesto para poder cotejarlos
sin importar tipos del puerto.

```go
func (d DatosVinculoActivacionV3) HuellaSHA256() string

func (d DatosVinculoActivacionV3) Validar() bool

type EstadoResultadoReconciliacionV3 string
```

EstadoResultadoReconciliacionV3 expresa el resultado cerrado comunicado por
el conector. Ninguno de sus valores constituye por si solo una comprobacion.

```go
const (
	ResultadoReconciliacionV3AplicadoExacto EstadoResultadoReconciliacionV3 = "aplicado_exacto"
	ResultadoReconciliacionV3NoAplicado     EstadoResultadoReconciliacionV3 = "no_aplicado_atestado"
	ResultadoReconciliacionV3Desconocido    EstadoResultadoReconciliacionV3 = "desconocido"
	ResultadoReconciliacionV3Conflictivo    EstadoResultadoReconciliacionV3 = "conflictivo"
)
func (e EstadoResultadoReconciliacionV3) Valido() bool

type ExpectativasResultadoReconciliacionV3 struct {
	ReservaRef              string
	EfectoRef               string
	SecuenciaCercado        uint64
	HuellaVinculoSHA256     string
	HuellaPlanSHA256        string
	ResultadoAplicadoValido bool
}
```

ExpectativasResultadoReconciliacionV3 contiene exclusivamente los vinculos
ya revalidados por el puerto. ResultadoAplicadoValido representa la
validacion rica contra el manifiesto, que permanece fuera del canonico puro.

```go
type PerfilMaterialDespachoV3 struct {
	Valido                  bool
	Audiencia               string
	ClaveGestionadaRef      string
	RevisionClaveGestionada uint64
	HuellaSHA256            string
}

type PerfilSelloEvidenciaV3 struct {
	Algoritmo string
	ClaveID   string
	Audiencia string
}
```

PerfilSelloEvidenciaV3 es una seleccion nominal de clave y audiencia.
No contiene la clave ni concede capacidad para firmar.

```go
func NuevoPerfilSelloEvidenciaHMACSHA256V3(claveID string) (PerfilSelloEvidenciaV3, error)

func (p PerfilSelloEvidenciaV3) Format(estado fmt.State, _ rune)

func (p PerfilSelloEvidenciaV3) GoString() string

func (p PerfilSelloEvidenciaV3) LogValue() slog.Value

func (PerfilSelloEvidenciaV3) MarshalBinary() ([]byte, error)

func (PerfilSelloEvidenciaV3) MarshalJSON() ([]byte, error)

func (PerfilSelloEvidenciaV3) MarshalText() ([]byte, error)

func (PerfilSelloEvidenciaV3) String() string

func (*PerfilSelloEvidenciaV3) UnmarshalBinary([]byte) error

func (*PerfilSelloEvidenciaV3) UnmarshalJSON([]byte) error

func (*PerfilSelloEvidenciaV3) UnmarshalText([]byte) error

func (p PerfilSelloEvidenciaV3) Validar() error

type SelloEvidenciaV3 struct {
	// Has unexported fields.
}
```

SelloEvidenciaV3 es opaco y autocontenible, pero siempre nominal: solo una
comprobacion privada con relectura durable puede promover su resultado.

```go
func NuevoSelloEvidenciaV3(
	perfil PerfilSelloEvidenciaV3,
	huellaMensaje string,
	firma []byte,
	evidenciaOperacionRef string,
	firmadoEn time.Time,
) (SelloEvidenciaV3, error)

func RestaurarSelloEvidenciaV3(datos DatosSelloEvidenciaV3) SelloEvidenciaV3

func (s SelloEvidenciaV3) Datos() (DatosSelloEvidenciaV3, error)

func (s SelloEvidenciaV3) EsCero() bool

func (s SelloEvidenciaV3) Format(estado fmt.State, _ rune)

func (s SelloEvidenciaV3) GoString() string

func (s SelloEvidenciaV3) LogValue() slog.Value

func (SelloEvidenciaV3) MarshalBinary() ([]byte, error)

func (SelloEvidenciaV3) MarshalJSON() ([]byte, error)

func (SelloEvidenciaV3) MarshalText() ([]byte, error)

func (SelloEvidenciaV3) String() string

func (*SelloEvidenciaV3) UnmarshalBinary([]byte) error

func (*SelloEvidenciaV3) UnmarshalJSON([]byte) error

func (*SelloEvidenciaV3) UnmarshalText([]byte) error

func (s SelloEvidenciaV3) ValidarPara(perfil PerfilSelloEvidenciaV3, huellaMensaje string) error

type SobreAtestacionReconciliacionV3 struct {
	// Has unexported fields.
}
```

SobreAtestacionReconciliacionV3 conserva el COSE_Sign1 y su compromiso sin
exponer representacion mutable. Sigue siendo material nominal hasta que una
dependencia criptografica privada comprueba la atestacion.

```go
func NuevoSobreAtestacionReconciliacionV3(
	coseSign1 []byte,
) (SobreAtestacionReconciliacionV3, error)

func RestaurarSobreAtestacionReconciliacionV3(
	coseSign1 []byte,
	huella string,
) SobreAtestacionReconciliacionV3
```

RestaurarSobreAtestacionReconciliacionV3 permite verificar una forma
persistida, incluida su huella original. Copia siempre los octetos
recibidos.

```go
func (s SobreAtestacionReconciliacionV3) COSESign1() ([]byte, error)

func (s SobreAtestacionReconciliacionV3) Format(estado fmt.State, _ rune)

func (s SobreAtestacionReconciliacionV3) GoString() string

func (s SobreAtestacionReconciliacionV3) HuellaSHA256() (string, error)

func (s SobreAtestacionReconciliacionV3) LogValue() slog.Value

func (SobreAtestacionReconciliacionV3) MarshalBinary() ([]byte, error)

func (SobreAtestacionReconciliacionV3) MarshalJSON() ([]byte, error)

func (SobreAtestacionReconciliacionV3) MarshalText() ([]byte, error)

func (SobreAtestacionReconciliacionV3) String() string

func (*SobreAtestacionReconciliacionV3) UnmarshalBinary([]byte) error

func (*SobreAtestacionReconciliacionV3) UnmarshalJSON([]byte) error

func (*SobreAtestacionReconciliacionV3) UnmarshalText([]byte) error

func (s SobreAtestacionReconciliacionV3) Validar() error

type SobrePruebaAtestacionDespachoV3 struct {
	// Has unexported fields.
}
```

SobrePruebaAtestacionDespachoV3 mantiene opacos y copiados defensivamente el
mensaje y su firma. Es material nominal: no representa una comprobacion KMS.

```go
func NuevoSobrePruebaAtestacionDespachoV3(
	algoritmo, audiencia, contexto, claveGestionadaRef string,
	revisionClaveGestionada uint64,
	evidenciaOperacionRef string,
	mensajeCanonico, sobreCriptografico []byte,
) (SobrePruebaAtestacionDespachoV3, error)

func RestaurarSobrePruebaAtestacionDespachoV3(
	datos DatosPruebaAtestacionDespachoV3,
) SobrePruebaAtestacionDespachoV3
```

RestaurarSobrePruebaAtestacionDespachoV3 no promueve autoridad. Permite que
el puerto restaure datos y que Validar detecte cualquier alteracion.

```go
func (s SobrePruebaAtestacionDespachoV3) Datos() (DatosPruebaAtestacionDespachoV3, error)

func (s SobrePruebaAtestacionDespachoV3) EvidenciaOperacionRef() (string, error)

func (s SobrePruebaAtestacionDespachoV3) Format(estado fmt.State, _ rune)

func (s SobrePruebaAtestacionDespachoV3) GoString() string

func (s SobrePruebaAtestacionDespachoV3) HuellaSHA256() string

func (s SobrePruebaAtestacionDespachoV3) HuellasSHA256() (mensaje, sobre string, err error)

func (s SobrePruebaAtestacionDespachoV3) LogValue() slog.Value

func (SobrePruebaAtestacionDespachoV3) MarshalBinary() ([]byte, error)

func (SobrePruebaAtestacionDespachoV3) MarshalJSON() ([]byte, error)

func (SobrePruebaAtestacionDespachoV3) MarshalText() ([]byte, error)

func (s SobrePruebaAtestacionDespachoV3) MensajeCanonico() ([]byte, error)

func (s SobrePruebaAtestacionDespachoV3) Perfil() (
	algoritmo, audiencia, contexto, claveGestionadaRef string,
	revisionClaveGestionada uint64,
	err error,
)

func (s SobrePruebaAtestacionDespachoV3) SobreCriptografico() ([]byte, error)

func (SobrePruebaAtestacionDespachoV3) String() string

func (*SobrePruebaAtestacionDespachoV3) UnmarshalBinary([]byte) error

func (*SobrePruebaAtestacionDespachoV3) UnmarshalJSON([]byte) error

func (*SobrePruebaAtestacionDespachoV3) UnmarshalText([]byte) error

func (s SobrePruebaAtestacionDespachoV3) Validar() error

type VinculosMaterialDespachoV3 struct {
	InicioRef                  string
	AtestacionInicioRef        string
	ReclamacionRef             string
	AtestacionReclamacionRef   string
	OrdenConsumoDurableV4Ref   string
	HuellaOrdenDespachoSHA256  string
	HuellaReciboInicioSHA256   string
	HuellaVinculoEstableSHA256 string
	HuellaVinculoCercadoSHA256 string
	SecuenciaCercado           uint64
	VersionInicioCAS           uint64
	VersionReclamacionCAS      uint64
}
```

VinculosMaterialDespachoV3 son los identificadores durables comprometidos
por la comprobacion conjunta de cercado, inicio y reclamacion.

```go
func (v VinculosMaterialDespachoV3) EsValido() bool

func (v VinculosMaterialDespachoV3) Format(estado fmt.State, _ rune)

func (v VinculosMaterialDespachoV3) GoString() string

func (v VinculosMaterialDespachoV3) LogValue() slog.Value

func (VinculosMaterialDespachoV3) MarshalBinary() ([]byte, error)

func (VinculosMaterialDespachoV3) MarshalJSON() ([]byte, error)

func (VinculosMaterialDespachoV3) MarshalText() ([]byte, error)

func (VinculosMaterialDespachoV3) String() string

func (*VinculosMaterialDespachoV3) UnmarshalBinary([]byte) error

func (*VinculosMaterialDespachoV3) UnmarshalJSON([]byte) error

func (*VinculosMaterialDespachoV3) UnmarshalText([]byte) error
```

## Paquete `internal/vec/canonico/pagos`

> Package pagos concentra las reglas puras y deterministas del contrato de cobros.

Package pagos concentra las reglas puras y deterministas del contrato de cobros.
No contiene puertos ni conoce adaptadores, redes o persistencia.

### Variables

```go
var (
	ErrCapacidadPasarelaCobroNoDisponible = errors.New("vec: capacidad de pasarela de cobro no disponible")
	ErrSolicitudOperacionCobroInvalida    = errors.New("vec: solicitud de operacion de cobro invalida")
	ErrReferenciaOperacionCobroInvalida   = errors.New("vec: referencia de operacion de cobro invalida")
	ErrNotificacionCobroInvalida          = errors.New("vec: notificacion de cobro invalida")
	ErrSolicitudDevolucionCobroInvalida   = errors.New("vec: solicitud de devolucion de cobro invalida")
	ErrSolicitudConciliacionCobroInvalida = errors.New("vec: solicitud de conciliacion de cobro invalida")
	ErrResultadoPasarelaCobroInvalido     = errors.New("vec: resultado de pasarela de cobro invalido")
)
var ErrInicioOperacionCobroInvalido = errors.New("vec: inicio de operacion de cobro invalido")
```

ErrInicioOperacionCobroInvalido es el error contractual compartido por la
representacion canonica y su reexportacion desde los puertos.

```go
var ErrMutacionOrdenCobroInvalida = errors.New("vec: mutacion de orden de cobro invalida")
```

ErrMutacionOrdenCobroInvalida es el error contractual compartido por los
registros canonicos y su reexportacion desde los puertos.

### Funciones

```go
func AccionAuditoriaPermitida(accion domain.AccionCobro) bool
```

AccionAuditoriaPermitida mantiene cerradas las acciones auditables de cobro.

```go
func BytesConfiguracionOrigen(origen OrigenPasarelaCobroPublicado) ([]byte, error)
```

BytesConfiguracionOrigen fija una representacion estable. Las listas se
tratan como conjuntos y se ordenan sobre copias defensivas.

```go
func ClaveValida(valor string) bool
```

ClaveValida comprueba una clave tecnica de lista cerrada.

```go
func ConfiguracionOrigenValidaSinHuella(origen OrigenPasarelaCobroPublicado) bool
```

ConfiguracionOrigenValidaSinHuella comprueba la forma previa al sellado.

```go
func ConfiguracionesOrigenIguales(a, b OrigenPasarelaCobroPublicado) bool
```

ConfiguracionesOrigenIguales compara tambien la representacion canonica de
las listas, por lo que su orden accidental no afecta al resultado.

```go
func ContieneCadenaExacta(valores []string, buscado string) bool
```

ContieneCadenaExacta busca sin normalizaciones ambiguas.

```go
func ContieneDatoTarjeta(valor string) bool
```

ContieneDatoTarjeta detecta etiquetas sensibles y numeros que superan Luhn,
incluidos digitos arabigos y de ancho completo y separadores invisibles.

```go
func GarantiaAutenticacionPermitida(garantia domain.AuthAssurance) bool
```

GarantiaAutenticacionPermitida mantiene cerrados los niveles aceptados.

```go
func HuellaConfiguracionOrigen(origen OrigenPasarelaCobroPublicado) (string, error)
```

HuellaConfiguracionOrigen deriva la huella SHA-256 de los bytes canonicos.

```go
func HuellaHMACDeDominioValida(valor, dominio string) bool
```

HuellaHMACDeDominioValida exige el dominio criptografico explicito.

```go
func HuellaSHA256Valida(valor string) bool
```

HuellaSHA256Valida acepta solamente la representacion hexadecimal canonica.

```go
func IDAuditoria(
	ordenRef string,
	version int,
	huellaPosterior string,
	hecho domain.TipoHechoCobro,
	accion domain.AccionCobro,
) string
```

IDAuditoria deriva el identificador de auditoria ligado a la mutacion.

```go
func IDEvento(
	ordenRef string,
	version int,
	secuencia int64,
	huellaHecho string,
	hecho domain.TipoHechoCobro,
	estado domain.EstadoCobro,
	accion domain.AccionCobro,
) string
```

IDEvento deriva el identificador de outbox ligado al ultimo hecho.

```go
func ListaCerradaValida(valores []string, rutas bool) bool
```

ListaCerradaValida comprueba unicidad y sintaxis sin modificar la entrada.

```go
func MatrizEvidenciaAuditoriaValida(r RegistroAuditoriaCobro) bool
```

MatrizEvidenciaAuditoriaValida liga procedencia, hecho y prueba remota.

```go
func MetodoAutenticacionPermitido(metodo domain.AuthMethod) bool
```

MetodoAutenticacionPermitido mantiene cerrada la matriz de autenticacion.

```go
func ReferenciaOpacaValida(valor, prefijo string) bool
```

ReferenciaOpacaValida comprueba una referencia opaca ligada a un prefijo.

```go
func RutaHandoffValida(valor string) bool
```

RutaHandoffValida solo admite rutas relativas canonicas sin escapes ni
datos.

```go
func TextoValido(valor string, maximo int) bool
```

TextoValido rechaza controles, espacios exteriores y posibles datos de
tarjeta.

```go
func TipoContenidoNotificacionPermitido(valor string) bool
```

TipoContenidoNotificacionPermitido mantiene cerrados los formatos remotos.

### Tipos

```go
type AtributosEventoSalidaCobro struct {
	Hecho  domain.TipoHechoCobro
	Estado domain.EstadoCobro
	Accion domain.AccionCobro
}
```

AtributosEventoSalidaCobro sustituye el mapa abierto del outbox.

```go
type CampoHandoffCobro struct {
	Nombre string
	Valor  string
}
```

CampoHandoffCobro es un par permitido de la carga opaca de handoff.

```go
type CanalAuditoriaCobro string
```

CanalAuditoriaCobro es informativo y nunca concede acceso.

```go
const (
	CanalAuditoriaCobroInterno           CanalAuditoriaCobro = "interno"
	CanalAuditoriaCobroPasarela          CanalAuditoriaCobro = "pasarela"
	CanalAuditoriaCobroProcesoAutomatico CanalAuditoriaCobro = "proceso_automatico"
)
func CanalAuditoriaParaHecho(hecho domain.TipoHechoCobro, accion domain.AccionCobro) (CanalAuditoriaCobro, bool)
```

CanalAuditoriaParaHecho deriva el canal sin aceptar metadatos del llamador.

```go
type CapacidadesPasarelaCobro struct {
	ConectorID              string `json:"conector_id"`
	VersionConector         int    `json:"version_conector"`
	RedireccionAlojada      bool   `json:"redireccion_alojada"`
	NotificacionAutenticada bool   `json:"notificacion_autenticada"`
	ConsultaOperacion       bool   `json:"consulta_operacion"`
	Devolucion              bool   `json:"devolucion"`
	Conciliacion            bool   `json:"conciliacion"`
	Justificante            bool   `json:"justificante"`
	TLSMutuo                bool   `json:"tls_mutuo"`
	IdempotenciaProveedor   bool   `json:"idempotencia_proveedor"`
}
```

CapacidadesPasarelaCobro evita suponer funciones de un proveedor concreto.

```go
func (c CapacidadesPasarelaCobro) Validar() error

type CargaHandoffCobro struct {
	// Has unexported fields.
}
```

CargaHandoffCobro oculta la carga para impedir serializarla o construirla
sin pasar por NuevaCargaHandoffCobro.

```go
func NuevaCargaHandoffCobro(campos []CampoHandoffCobro, permitidos []string) (CargaHandoffCobro, error)
```

NuevaCargaHandoffCobro copia y valida una carga contra una lista cerrada.

```go
func (c CargaHandoffCobro) GoString() string

func (CargaHandoffCobro) MarshalJSON() ([]byte, error)

func (CargaHandoffCobro) String() string

type ContenidoNotificacionCobroUnico struct {
	Metadatos SolicitudCustodiarNotificacionCobro
	Contenido io.ReadCloser
}

func (c ContenidoNotificacionCobroUnico) Validar() error

type EventoSalidaCobro struct {
	ID                string
	Tipo              TipoEventoSalidaCobro
	OrdenRef          string
	VersionOrden      int
	SecuenciaHecho    int64
	HuellaHechoSHA256 string
	HuellaOrdenSHA256 string
	CorrelacionRef    string
	OcurridoEn        time.Time
	Atributos         AtributosEventoSalidaCobro
}
```

EventoSalidaCobro es el evento derivado del ultimo hecho confirmado.

```go
func NuevoEventoSalidaCobro(orden domain.OrdenCobro) (EventoSalidaCobro, error)
```

NuevoEventoSalidaCobro deriva el evento completo sin campos semanticos
libres.

```go
func (e EventoSalidaCobro) Validar() error
```

Validar comprueba la tupla cerrada y el identificador derivado.

```go
type InicioOperacionCobro struct {
	Evidencia                 domain.EvidenciaInicioOperacionCobro
	Origen                    OrigenPasarelaCobroPublicado
	VersionOrden              int
	HuellaOrdenSHA256         string
	HuellaConfiguracionSHA256 string
	Ruta                      string
	Metodo                    MetodoHandoffCobro
	Carga                     CargaHandoffCobro
	GeneradaEn                time.Time
	ExpiraEn                  time.Time
}
```

InicioOperacionCobro separa el origen publicado de la carga de handoff.
No acepta una URL devuelta libremente por el proveedor ni secretos en query.

```go
func (i InicioOperacionCobro) CamposRespuestaPOSTContra(
	comando domain.ComandoInicioOperacionCobro,
	origen OrigenPasarelaCobroPublicado,
	ahora time.Time,
) ([]CampoHandoffCobro, error)
```

CamposRespuestaPOSTContra devuelve una copia solo tras la validacion
completa. El consumo unico corresponde a una custodia durable externa.

```go
func (i InicioOperacionCobro) Validar() error
```

Validar comprueba la estructura. La entrega exige ademas ValidarContra.

```go
func (i InicioOperacionCobro) ValidarContra(
	comando domain.ComandoInicioOperacionCobro,
	origen OrigenPasarelaCobroPublicado,
	ahora time.Time,
) error
```

ValidarContra liga la respuesta al comando sellado, al origen publicado y al
reloj confiable. Es la unica validacion suficiente para entregar handoff.

```go
type MetadatosAuditoriaCobro struct {
	Canal CanalAuditoriaCobro
}
```

MetadatosAuditoriaCobro mantiene cerrado el unico metadato informativo.

```go
type MetodoHandoffCobro string
```

MetodoHandoffCobro es una lista cerrada de mecanismos de entrega al cliente.

```go
const MetodoHandoffCobroPOSTFormulario MetodoHandoffCobro = "post_formulario"
type NotificacionCobro struct {
	ConectorID      string    `json:"conector_id"`
	VersionConector int       `json:"version_conector"`
	RecepcionRef    string    `json:"recepcion_ref"`
	Audiencia       string    `json:"audiencia"`
	RecibidaEn      time.Time `json:"recibida_en"`
}
```

NotificacionCobro solo transporta una referencia opaca a una recepcion
temporal controlada por el adaptador.

```go
func (n NotificacionCobro) Validar() error

type OrigenPasarelaCobroPublicado struct {
	ID                        string
	Version                   int
	BaseHTTPS                 string
	RutasPermitidas           []string
	CamposHandoffPermitidos   []string
	HuellaConfiguracionSHA256 string
	PublicadaEn               time.Time
}
```

OrigenPasarelaCobroPublicado contiene los datos necesarios para validar y
sellar un origen publicado de pasarela.

```go
func (o OrigenPasarelaCobroPublicado) Validar() error
```

Validar comprueba tambien que la huella corresponda a la configuracion.

```go
type ReferenciaDevolucionCobro struct {
	ConectorID            string
	VersionConector       int
	OrdenRef              string
	DevolucionRef         string
	OperacionProveedorRef string
	CorrelacionRef        string
}

func (r ReferenciaDevolucionCobro) Validar() error

type ReferenciaOperacionCobro struct {
	ConectorID            string `json:"conector_id"`
	VersionConector       int    `json:"version_conector"`
	OrdenRef              string `json:"orden_ref"`
	OperacionProveedorRef string `json:"operacion_proveedor_ref"`
	CorrelacionRef        string `json:"correlacion_ref"`
}

func (r ReferenciaOperacionCobro) Validar() error

type RegistroAuditoriaCobro struct {
	ID                          string
	ActorRef                    string
	PerfilActivoRef             string
	DecisionAutorizacionRef     string
	HuellaDecisionSHA256        string
	AutorizacionEmitidaEn       time.Time
	AutorizacionValidaHasta     time.Time
	AutorizacionEvaluadaEn      time.Time
	AtestacionAutenticacionRef  string
	AtestacionEmitidaEn         time.Time
	AtestacionValidaHasta       time.Time
	AutenticacionVerificadaEn   time.Time
	SesionRef                   string
	HuellaSesionHMAC            string
	MetodoAutenticacion         domain.AuthMethod
	GarantiaAutenticacion       domain.AuthAssurance
	Accion                      domain.AccionCobro
	Hecho                       domain.TipoHechoCobro
	OrdenRef                    string
	ExpedienteRef               string
	VersionAnterior             int
	VersionPosterior            int
	HuellaAnteriorSHA256        string
	HuellaPosteriorSHA256       string
	EvidenciaRef                string
	HuellaEvidenciaSHA256       string
	VerificacionEvidenciaRef    string
	HuellaVerificacionSHA256    string
	MetodoVerificacionEvidencia domain.MetodoAutenticacionEvidenciaCobro
	AudienciaEvidencia          string
	EvidenciaEmitidaEn          time.Time
	EvidenciaRecibidaEn         time.Time
	EvidenciaVerificadaEn       time.Time
	Resultado                   string
	Motivo                      string
	CorrelacionRef              string
	OcurridoEn                  time.Time
	Metadatos                   MetadatosAuditoriaCobro
}
```

RegistroAuditoriaCobro es la proyeccion inmutable del ultimo hecho de cobro.

```go
func (r RegistroAuditoriaCobro) Validar() error
```

Validar comprueba la integridad completa y el identificador derivado.

```go
type ResultadoConciliacionCobro struct {
	Evidencia domain.EvidenciaConciliacionCobro `json:"-"`
}

func (ResultadoConciliacionCobro) MarshalJSON() ([]byte, error)

func (r ResultadoConciliacionCobro) Validar() error

type ResultadoDevolucionCobro struct {
	Evidencia domain.EvidenciaResultadoDevolucionCobro `json:"-"`
}

func (ResultadoDevolucionCobro) MarshalJSON() ([]byte, error)

func (r ResultadoDevolucionCobro) Validar() error

type ResultadoOperacionCobro struct {
	Evidencia domain.EvidenciaResultadoCobro `json:"-"`
}

func (ResultadoOperacionCobro) MarshalJSON() ([]byte, error)

func (r ResultadoOperacionCobro) Validar() error

type SolicitudConciliacionCobro struct {
	Comando domain.ComandoConciliacionCobro
}

func (s SolicitudConciliacionCobro) Validar() error

type SolicitudCustodiarNotificacionCobro struct {
	ConectorID      string
	VersionConector int
	RecepcionRef    string
	Audiencia       string
	TipoContenido   string
	Tamano          int64
	HuellaSHA256    string
	RecibidaEn      time.Time
	ExpiraEn        time.Time
}

func (s SolicitudCustodiarNotificacionCobro) Validar() error

type SolicitudDevolucionCobro struct {
	Comando domain.ComandoDevolucionCobro
}

func (s SolicitudDevolucionCobro) Validar() error

type SolicitudOperacionCobro struct {
	Comando domain.ComandoInicioOperacionCobro
}

func (s SolicitudOperacionCobro) Validar() error

type TipoEventoSalidaCobro string
```

TipoEventoSalidaCobro es la lista cerrada de tipos del outbox.

```go
const (
	EventoCobroOrdenCreada                    TipoEventoSalidaCobro = "cobro.orden.creada"
	EventoCobroOperacionEnviada               TipoEventoSalidaCobro = "cobro.operacion.enviada"
	EventoCobroResultadoPendiente             TipoEventoSalidaCobro = "cobro.resultado.pendiente"
	EventoCobroResultadoDesconocido           TipoEventoSalidaCobro = "cobro.resultado.desconocido"
	EventoCobroConfirmado                     TipoEventoSalidaCobro = "cobro.confirmado"
	EventoCobroRechazado                      TipoEventoSalidaCobro = "cobro.rechazado"
	EventoCobroCancelado                      TipoEventoSalidaCobro = "cobro.cancelado"
	EventoCobroCaducado                       TipoEventoSalidaCobro = "cobro.caducado"
	EventoCobroConciliado                     TipoEventoSalidaCobro = "cobro.conciliado"
	EventoCobroDevolucionSolicitada           TipoEventoSalidaCobro = "cobro.devolucion.solicitada"
	EventoCobroDevolucionResultadoPendiente   TipoEventoSalidaCobro = "cobro.devolucion.resultado_pendiente"
	EventoCobroDevolucionResultadoDesconocido TipoEventoSalidaCobro = "cobro.devolucion.resultado_desconocido"
	EventoCobroDevolucionRechazada            TipoEventoSalidaCobro = "cobro.devolucion.rechazada"
	EventoCobroDevuelto                       TipoEventoSalidaCobro = "cobro.devuelto"
	EventoCobroDevolucionConciliada           TipoEventoSalidaCobro = "cobro.devolucion.conciliada"
	EventoCobroIncidenciaDetectada            TipoEventoSalidaCobro = "cobro.incidencia.detectada"
	EventoCobroEvidenciaAdicional             TipoEventoSalidaCobro = "cobro.evidencia.adicional"
)
func TipoEventoParaHecho(hecho domain.TipoHechoCobro) (TipoEventoSalidaCobro, bool)
```

TipoEventoParaHecho mantiene el mapeo uno a uno entre hecho y outbox.

```go
func (t TipoEventoSalidaCobro) Valido() bool
```

Valido comprueba la lista cerrada del outbox.

## Paquete `internal/vec/canonico/recibomaterial`

> Package recibomaterial concentra la validacion y la representacion canonica del recibo material de escritura.

Package recibomaterial concentra la validacion y la representacion canonica del
recibo material de escritura. No conoce puertos, adaptadores ni estado.

### Constantes

```go
const (
	EsquemaPerfil      = "vec.almacen.perfil-capacidades-material.v2"
	EsquemaInstantanea = "vec.almacen.instantanea-objeto-material.v2"
	EsquemaRecibo      = "vec.almacen.recibo-escritura-material.v2"
	EsquemaVersion     = uint16(2)

	DominioPerfil = "perfil-capacidades-almacen-material-v2"
	DominioRecibo = "recibo-escritura-objeto-material-v2"

	EstadoNoInmovilizado = "no_inmovilizado"
	EstadoInmovilizado   = "inmovilizado"
	EstadoObjetoActivo   = "activo"

	AlgoritmoHMACSHA256 = "hmac-sha-256"
	AlgoritmoCOSESign1  = "cose-sign1"
	AccionEscribir      = "escribir"

	TextoRedactado = "[MATERIAL-ALMACEN-V2-REDACTADO]"
)
```

### Variables

```go
var (
	// ErrReciboNoValido mantiene deliberadamente opacos todos los rechazos.
	ErrReciboNoValido = errors.New("vec: recibo material v2 de escritura no valido")
	// ErrAtestacionNoValida evita distinguir fallos de forma y autenticidad.
	ErrAtestacionNoValida = errors.New("vec: atestacion material v2 de almacen no valida")
	// ErrSerializacionProhibida impide volcar capacidades opacas por accidente.
	ErrSerializacionProhibida = errors.New("vec: serializacion generica de material v2 de almacen prohibida")
)
```

### Funciones

```go
func AliasLogicoValido(valor string, maximo int) bool
```

AliasLogicoValido acepta solo referencias logicas ASCII y rechaza indicios
de ubicacion fisica o identificadores personales.

```go
func AnexarTLV(destino []byte, etiqueta uint16, valor []byte) []byte
```

AnexarTLV compone etiqueta y longitud en orden de red.

```go
func AtestacionValida(
	dominio string,
	mensaje []byte,
	huella [sha256.Size]byte,
	algoritmo, claveRef string,
	claveVersion uint32,
	dominioAtestado string,
	huellaAtestada [sha256.Size]byte,
	codigo []byte,
) bool
```

AtestacionValida comprueba que la respuesta corresponde a la solicitud
exacta.

```go
func Bool(valor bool) []byte
func CanonicoIdentidadDurable(r Recibo) ([]byte, error)
func CanonicoInstantanea(i Instantanea) ([]byte, error)
func CanonicoPerfil(p Perfil) ([]byte, error)
func CanonicoRecibo(r Recibo) ([]byte, error)
func CanonicoVinculoPlan(v VinculoPlan) ([]byte, error)
func CodigoAtestacionValido(algoritmo string, codigo []byte) bool
```

CodigoAtestacionValido aplica limites cerrados por algoritmo.

```go
func DecodificarSHA256(valor string) ([sha256.Size]byte, error)
```

DecodificarSHA256 exige hexadecimal minusculo y rechaza el valor cero.

```go
func DependenciaNula(dependencia any) bool
```

DependenciaNula detecta interfaces que esconden un puntero tipado nulo.

```go
func DeserializacionProhibida() error
func DominioAtestacionValido(dominio string) bool
```

DominioAtestacionValido mantiene separadas las dos finalidades admitidas.

```go
func FormatoRedactado(estado fmt.State)
func HechosContextoValidos(h HechosContexto) bool
func HechosMaterialesReciboValidos(r Recibo) bool
func HuellaInstantanea(i Instantanea) ([sha256.Size]byte, error)
func HuellaPerfil(p Perfil) ([sha256.Size]byte, error)
func HuellaPlanValida(h HuellaPlan) bool
func HuellaRecibo(r Recibo) ([sha256.Size]byte, error)
```

HuellaRecibo calcula el compromiso sobre los bytes canonicos completos.

```go
func HuellaVinculoPlan(v VinculoPlan) ([sha256.Size]byte, error)
func HuellasIguales(primera, segunda [sha256.Size]byte) bool
```

HuellasIguales compara sin filtrar informacion temporal sobre el prefijo.

```go
func InstantaneaValida(i Instantanea) bool
func InstanteValido(instante time.Time) bool
```

InstanteValido exige UTC exacto a microsegundos.

```go
func Int64(valor int64) []byte
func MIMEValido(valor string) bool
```

MIMEValido acepta un tipo canonico sin parametros ni rutas ambiguas.

```go
func NuevaHuellaIdentidad(canonico []byte) ([sha256.Size]byte, error)
func PareceIdentificadorPersonal(valor string) bool
```

PareceIdentificadorPersonal reconoce las formas basicas de DNI y NIE.

```go
func PerfilCotejaCapacidades(p Perfil, c almacencanonico.Capacidades) bool
func PerfilPublicadoValido(p DatosPerfilPublicado) bool
func PerfilSelladoNominalValido(p Perfil, huella [sha256.Size]byte, a DatosAtestacion) bool
```

PerfilSelladoNominalValido coteja forma, huella y ligadura nominal.
No sustituye la verificacion criptografica del conector homologado.

```go
func PerfilValido(p Perfil) bool
func ReciboSelladoNominalValido(r Recibo, huella [sha256.Size]byte, a DatosAtestacion) bool
```

ReciboSelladoNominalValido coteja el recibo, sus dominios, huellas y
ligadura nominal. No promueve la atestacion a autoridad criptografica.

```go
func ReciboValido(r Recibo) bool
func ResultadoLigado(huellaPlan, huellaResultado, vinculoResultado [sha256.Size]byte) bool
```

ResultadoLigado valida un resultado criptografico contra su vinculo exacto.

```go
func ResultadoPlanValido(
	v VinculoPlan,
	huellaVinculoSolicitud, huellaPlan, huellaVinculoResultado [sha256.Size]byte,
) bool
```

ResultadoPlanValido impide sustituir o reutilizar el vinculo como plan.

```go
func ResultadoReferenciaValido(huella [sha256.Size]byte, r DatosResultadoReferencia) bool
func RevelarVerificacionAtestacion(
	s DatosSolicitudAtestacion,
	a DatosAtestacion,
) (DatosSolicitudAtestacion, DatosAtestacion, error)
```

RevelarVerificacionAtestacion valida ambas capacidades y copia sus bytes.

```go
func SeleccionPlanValida(s SeleccionPlan) bool
func SerializacionProhibida() ([]byte, error)
func SolicitudAtestacionValida(dominio string, mensaje []byte, huella [sha256.Size]byte) bool
```

SolicitudAtestacionValida comprueba dominio, copia logica y compromiso.

```go
func SolicitudPlanValida(v VinculoPlan, huella [sha256.Size]byte) bool
```

SolicitudPlanValida coteja el compromiso almacenado con el vinculo exacto.

```go
func SumaSHA256(contenido []byte) [sha256.Size]byte
func TextoASCIICanonico(valor string) bool
```

TextoASCIICanonico limita la entrada al repertorio imprimible no ambiguo.

```go
func Uint16(valor uint16) []byte
func Uint32(valor uint32) []byte
```

### Tipos

```go
type DatosAtestacion struct {
	Algoritmo    string
	ClaveRef     string
	ClaveVersion uint32
	Dominio      string
	Huella       [sha256.Size]byte
	Codigo       []byte
}
```

DatosAtestacion representa material criptografico nominal ligado al mensaje.
Su forma valida no acredita la firma: la autoridad corresponde al conector
criptografico homologado y a la relectura durable del flujo de aplicacion.

```go
func NuevaAtestacionNominal(s DatosSolicitudAtestacion, a DatosAtestacion) (DatosAtestacion, error)

func (d DatosAtestacion) Format(estado fmt.State, _ rune)

func (d DatosAtestacion) GoString() string

func (d DatosAtestacion) LogValue() slog.Value

func (DatosAtestacion) MarshalBinary() ([]byte, error)

func (DatosAtestacion) MarshalJSON() ([]byte, error)

func (DatosAtestacion) MarshalText() ([]byte, error)

func (DatosAtestacion) String() string

func (*DatosAtestacion) UnmarshalBinary([]byte) error

func (*DatosAtestacion) UnmarshalJSON([]byte) error

func (*DatosAtestacion) UnmarshalText([]byte) error

type DatosPerfilPublicado struct {
	Referencia       string
	Version          uint32
	ConectorLogicoID string
	Huella           [sha256.Size]byte
	Canonico         []byte
}
```

DatosPerfilPublicado es la apertura minima hacia el catalogo homologado.

```go
func RevelarPerfilPublicado(p DatosPerfilPublicado) (DatosPerfilPublicado, error)

func (d DatosPerfilPublicado) Format(estado fmt.State, _ rune)

func (d DatosPerfilPublicado) GoString() string

func (d DatosPerfilPublicado) LogValue() slog.Value

func (DatosPerfilPublicado) MarshalBinary() ([]byte, error)

func (DatosPerfilPublicado) MarshalJSON() ([]byte, error)

func (DatosPerfilPublicado) MarshalText() ([]byte, error)

func (DatosPerfilPublicado) String() string

func (*DatosPerfilPublicado) UnmarshalBinary([]byte) error

func (*DatosPerfilPublicado) UnmarshalJSON([]byte) error

func (*DatosPerfilPublicado) UnmarshalText([]byte) error

type DatosResultadoReferencia struct {
	Referencia      string
	HuellaIdentidad [sha256.Size]byte
}
```

DatosResultadoReferencia liga la referencia durable a la identidad exacta.

```go
func (d DatosResultadoReferencia) Format(estado fmt.State, _ rune)

func (d DatosResultadoReferencia) GoString() string

func (d DatosResultadoReferencia) LogValue() slog.Value

func (DatosResultadoReferencia) MarshalBinary() ([]byte, error)

func (DatosResultadoReferencia) MarshalJSON() ([]byte, error)

func (DatosResultadoReferencia) MarshalText() ([]byte, error)

func (DatosResultadoReferencia) String() string

func (*DatosResultadoReferencia) UnmarshalBinary([]byte) error

func (*DatosResultadoReferencia) UnmarshalJSON([]byte) error

func (*DatosResultadoReferencia) UnmarshalText([]byte) error

type DatosSolicitudAtestacion struct {
	Dominio string
	Mensaje []byte
	Huella  [sha256.Size]byte
}
```

DatosSolicitudAtestacion conserva el mensaje exacto y su compromiso.

```go
func PrepararSolicitudAtestacion(dominio string, mensaje []byte) (DatosSolicitudAtestacion, error)
```

PrepararSolicitudAtestacion copia el mensaje antes de calcular su huella.

```go
func RevelarSolicitudAtestacion(s DatosSolicitudAtestacion) (DatosSolicitudAtestacion, error)
```

RevelarSolicitudAtestacion valida y devuelve siempre una copia defensiva.

```go
func (d DatosSolicitudAtestacion) Format(estado fmt.State, _ rune)

func (d DatosSolicitudAtestacion) GoString() string

func (d DatosSolicitudAtestacion) LogValue() slog.Value

func (DatosSolicitudAtestacion) MarshalBinary() ([]byte, error)

func (DatosSolicitudAtestacion) MarshalJSON() ([]byte, error)

func (DatosSolicitudAtestacion) MarshalText() ([]byte, error)

func (DatosSolicitudAtestacion) String() string

func (*DatosSolicitudAtestacion) UnmarshalBinary([]byte) error

func (*DatosSolicitudAtestacion) UnmarshalJSON([]byte) error

func (*DatosSolicitudAtestacion) UnmarshalText([]byte) error

type HechosContexto struct {
	ModuloID, AccionNegocio, AccionTecnica string
	RecursoRef, OperacionRef, CargaRef     string
	EfectoRef, Clasificacion               string
}
```

HechosContexto conserva solo los hechos estables necesarios para ligar un
plan.

```go
type HuellaPlan struct {
	Referencia    string
	Version       uint32
	Suma          [sha256.Size]byte
	HuellaVinculo [sha256.Size]byte
}
```

HuellaPlan contiene el compromiso publicado y el vinculo que lo
contextualiza.

```go
type Instantanea struct {
	Esquema              string
	VersionEsquema       uint16
	ConectorLogicoID     string
	ObjetoRef            string
	ObjetoVersion        string
	Zona                 almacencanonico.Zona
	MIME                 string
	Tamano               int64
	HuellaContenido      [sha256.Size]byte
	EvidenciaCreacionRef string
	AlmacenadoEn         time.Time
	TieneRetencion       bool
	RetenidoHasta        time.Time
	EstadoInmovilizacion string
	EstadoObjeto         string
}
```

Instantanea conserva los hechos originales del objeto, no los del reintento.

```go
type Perfil struct {
	Esquema                string
	VersionEsquema         uint16
	Referencia             string
	Version                uint32
	ConectorLogicoID       string
	EscrituraEnFlujo       bool
	ReferenciasOpacas      bool
	IntegridadSHA256       bool
	Versionado             bool
	Retencion              bool
	BloqueoLegal           bool
	CifradoEnTransito      bool
	CifradoEnReposo        bool
	CifradoPorObjeto       bool
	PreservaObjetoOriginal bool
	TamanoMaximoObjeto     int64
}
```

Perfil describe exclusivamente capacidades materiales, sin topologia fisica.

```go
type Recibo struct {
	Esquema                   string
	VersionEsquema            uint16
	ReferenciaDurableOriginal string
	PerfilReferencia          string
	PerfilVersion             uint32
	HuellaPerfil              [sha256.Size]byte
	Hechos                    HechosContexto
	HuellaPlan                HuellaPlan
	Instantanea               Instantanea
}
```

Recibo contiene todos los hechos deterministas cubiertos por la atestacion.

```go
type SeleccionPlan struct {
	Referencia string
	Version    uint32
}
```

SeleccionPlan contiene la identidad publica de un plan material versionado.

```go
type VinculoPlan struct {
	Seleccion        SeleccionPlan
	ConectorLogicoID string
	Hechos           HechosContexto
}
```

VinculoPlan liga la seleccion al conector y al contexto de negocio exacto.
