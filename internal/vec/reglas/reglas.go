// Package reglas resuelve reglas tipadas (plazos, límites y listas) publicadas
// como catálogos configurables. No conoce Bolsa ni Contratación temporal: cada
// módulo pide la regla por su clave y recibe valor, unidad, procedencia y la
// referencia exacta catálogo:versión:entrada con la huella del catálogo.
//
// En este corte solo existen paquetes de ejemplo (data/demo/reglas) que se
// componen en el perfil de desarrollo. Una regla de origen «ejemplo» no está
// aprobada por RRHH; una de origen «reglamento» cita un artículo publicado.
package reglas

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// MarcaPaqueteEjemplo es la referencia de fuente que comparten todos los
// paquetes de ejemplo; se retiran juntos antes de producción.
const MarcaPaqueteEjemplo = "paquete:ejemplo:vec:v1"

var (
	// ErrReglasNoConfiguradas: no hay catálogo compuesto. El consumidor
	// conserva su comportamiento sin reglas.
	ErrReglasNoConfiguradas = errors.New("reglas: sin catalogo de reglas configurado")
	// ErrReglasNoDisponibles: el catálogo no existe, no está vigente o excede
	// los límites. Nunca se interpreta como una regla por defecto.
	ErrReglasNoDisponibles = errors.New("reglas: catalogo de reglas no disponible")
	ErrReglaNoEncontrada   = errors.New("reglas: regla no encontrada o no vigente")
	ErrReglaInvalida       = errors.New("reglas: regla con atributos no validos")
	ErrReglaSinPlazo       = errors.New("reglas: la regla no define un plazo computable")
	ErrCalculoNoDisponible = errors.New("reglas: calculo de plazos no disponible")
	ErrConfiguracion       = errors.New("reglas: configuracion del resolutor no valida")
)

// Unidad de la cantidad de una regla. Las cuatro primeras son plazos que se
// calculan con Calendarios; el resto son valores sin vencimiento.
type Unidad string

const (
	UnidadDiasHabiles      Unidad = "dias_habiles"
	UnidadDiasNaturales    Unidad = "dias_naturales"
	UnidadMeses            Unidad = "meses"
	UnidadAnios            Unidad = "anios"
	UnidadHoras            Unidad = "horas"
	UnidadMinutosSemanales Unidad = "minutos_semanales"
	UnidadIntentos         Unidad = "intentos"
	UnidadProcesos         Unidad = "procesos"
	UnidadFranjaHoraria    Unidad = "franja_horaria"
	UnidadLista            Unidad = "lista"
	UnidadNinguna          Unidad = "ninguna"
)

const (
	maximoCantidadRegla       = 100_000
	maximoElementosListaRegla = 64
)

// EsPlazo indica si la unidad admite cálculo de vencimiento.
func (u Unidad) EsPlazo() bool {
	return u == UnidadDiasHabiles || u == UnidadDiasNaturales || u == UnidadMeses || u == UnidadAnios
}

func (u Unidad) conCantidad() bool {
	return u.EsPlazo() || u == UnidadHoras || u == UnidadMinutosSemanales || u == UnidadIntentos || u == UnidadProcesos
}

// Origen distingue lo que cita el Reglamento de bolsas de lo inventado para
// trabajar mientras RRHH responde.
type Origen string

const (
	OrigenReglamento Origen = "reglamento"
	OrigenEjemplo    Origen = "ejemplo"
)

// Computo fija la regla de cálculo: administrativo (art. 30 de la Ley
// 39/2015, con días inhábiles y prórroga al primer hábil) o civil (art. 5.1
// del Código Civil, de fecha a fecha y sin prórroga).
type Computo string

const (
	ComputoAdministrativo Computo = "administrativo"
	ComputoCivil          Computo = "civil"
)

// Regla es una entrada resuelta y tipada. Atributos conserva los parámetros
// adicionales (por ejemplo ventana_meses o cantidad_urgente) como copia.
type Regla struct {
	Clave       string
	Etiqueta    string
	Descripcion string
	Unidad      Unidad
	// Cantidad es cero cuando la unidad no la admite.
	Cantidad int
	// Valor contiene la franja («09:00-14:00») o la lista separada por comas.
	Valor   string
	Inicio  string
	Computo Computo
	Origen  Origen
	// Articulo solo existe para el origen reglamento.
	Articulo string
	Norma    string
	Duda     string
	// ParteEjemplo describe la parte inventada de una regla del Reglamento.
	ParteEjemplo string
	Atributos    map[string]string
	// Referencia es catalogo:version:entrada; ReferenciaEntrada añade la
	// huella SHA-256 del catálogo completo.
	Referencia        string
	ReferenciaEntrada domain.ReferenciaEntradaCatalogo
	HuellaCatalogo    string
	// PaqueteEjemplo es cierto si la fuente es de demostración o lleva la
	// marca compartida de los datos de ejemplo.
	PaqueteEjemplo bool
	Paquete        string
}

// EsEjemplo es cierto si la regla, o una parte de ella, no procede del
// Reglamento. La interfaz debe rotularla como «regla de ejemplo».
func (r Regla) EsEjemplo() bool { return r.Origen == OrigenEjemplo || r.ParteEjemplo != "" }

// Elementos devuelve la lista de una regla de unidad lista.
func (r Regla) Elementos() []string {
	if r.Unidad != UnidadLista || r.Valor == "" {
		return nil
	}
	return strings.Split(r.Valor, ",")
}

// reglaDesdeEntrada valida los atributos del contrato y construye la regla.
func reglaDesdeEntrada(catalogo domain.CatalogoConfigurable, huella string, ejemplo bool, entrada domain.EntradaCatalogoConfigurable) (Regla, error) {
	a := entrada.Atributos
	regla := Regla{
		Clave: entrada.Clave, Etiqueta: entrada.Etiqueta, Descripcion: entrada.Descripcion,
		Unidad: Unidad(a["unidad"]), Valor: a["valor"], Inicio: a["inicio"],
		Computo: Computo(a["computo"]), Origen: Origen(a["origen"]),
		Articulo: a["articulo"], Norma: a["norma"], Duda: a["duda"],
		ParteEjemplo: a["ejemplo_parcial"], Atributos: make(map[string]string, len(a)),
		ReferenciaEntrada: domain.ReferenciaEntradaCatalogo{
			CatalogoID: catalogo.ID, CatalogoVersion: catalogo.Version,
			CatalogoHuellaSHA256: huella, EntradaClave: entrada.Clave,
		},
		HuellaCatalogo: huella, PaqueteEjemplo: ejemplo, Paquete: catalogo.FuenteRef,
	}
	for clave, valor := range a {
		regla.Atributos[clave] = valor
	}
	regla.Referencia = regla.ReferenciaEntrada.Referencia()
	if regla.ReferenciaEntrada.Validar() != nil || regla.Norma == "" || regla.Duda == "" {
		return Regla{}, ErrReglaInvalida
	}
	switch regla.Origen {
	case OrigenReglamento:
		if regla.Articulo == "" {
			return Regla{}, ErrReglaInvalida
		}
	case OrigenEjemplo:
		if regla.Articulo != "" || regla.ParteEjemplo != "" {
			return Regla{}, ErrReglaInvalida
		}
	default:
		return Regla{}, ErrReglaInvalida
	}
	if err := validarUnidadRegla(&regla, a); err != nil {
		return Regla{}, err
	}
	return regla, nil
}

func validarUnidadRegla(regla *Regla, a map[string]string) error {
	texto, conCantidad := a["cantidad"]
	switch {
	case regla.Unidad.conCantidad():
		cantidad, err := strconv.Atoi(texto)
		if err != nil || cantidad < 1 || cantidad > maximoCantidadRegla || strconv.Itoa(cantidad) != texto {
			return ErrReglaInvalida
		}
		regla.Cantidad = cantidad
	case regla.Unidad == UnidadFranjaHoraria, regla.Unidad == UnidadLista, regla.Unidad == UnidadNinguna:
		if conCantidad {
			return ErrReglaInvalida
		}
	default:
		return ErrReglaInvalida
	}
	if (regla.Unidad == UnidadFranjaHoraria || regla.Unidad == UnidadLista) && regla.Valor == "" {
		return ErrReglaInvalida
	}
	if regla.Unidad == UnidadFranjaHoraria && !franjaValida(regla.Valor) {
		return ErrReglaInvalida
	}
	if regla.Unidad == UnidadLista && len(strings.Split(regla.Valor, ",")) > maximoElementosListaRegla {
		return ErrReglaInvalida
	}
	conComputo := regla.Computo == ComputoAdministrativo || regla.Computo == ComputoCivil
	if (regla.Computo != "" && !conComputo) || conComputo != (regla.Inicio != "") ||
		(conComputo && !regla.Unidad.EsPlazo()) {
		return ErrReglaInvalida
	}
	return nil
}

// franjaValida admite «HH:MM-HH:MM» con inicio anterior al fin.
func franjaValida(valor string) bool {
	desde, hasta, ok := strings.Cut(valor, "-")
	if !ok {
		return false
	}
	inicio, errInicio := time.Parse("15:04", desde)
	fin, errFin := time.Parse("15:04", hasta)
	return errInicio == nil && errFin == nil && inicio.Before(fin)
}
