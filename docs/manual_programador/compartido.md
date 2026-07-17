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
