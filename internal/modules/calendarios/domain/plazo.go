package domain

import (
	"errors"
	"time"
)

var ErrPlazoInvalido = errors.New("calendarios: solicitud de plazo invalida")

// UnidadPlazo es la semántica jurídica soportada. Los plazos por horas no se
// admiten en este corte: exigen hora de inicio y régimen propio (art. 30.1).
type UnidadPlazo string

const (
	UnidadDiasHabiles   UnidadPlazo = "dias_habiles"
	UnidadDiasNaturales UnidadPlazo = "dias_naturales"
	UnidadMeses         UnidadPlazo = "meses"
	UnidadAnios         UnidadPlazo = "anios"
)

// Límites de cardinalidad: acotan el trabajo antes de consultar calendarios.
var maximoPorUnidad = map[UnidadPlazo]int{
	UnidadDiasHabiles: 250, UnidadDiasNaturales: 730, UnidadMeses: 60, UnidadAnios: 5,
}

func (u UnidadPlazo) Valida() bool { _, ok := maximoPorUnidad[u]; return ok }

// CalendarioComputo responde si una fecha es inhábil para el cómputo. Debe
// fallar con ErrCalendarioNoCubre si carece del calendario de ese año.
type CalendarioComputo interface {
	EsInhabil(FechaCivil) (bool, []Motivo, error)
}

type SolicitudPlazo struct {
	// Inicio es la fecha de notificación o publicación del acto.
	Inicio   FechaCivil
	Unidad   UnidadPlazo
	Cantidad int
}

func (s SolicitudPlazo) Validar() error {
	maximo, ok := maximoPorUnidad[s.Unidad]
	if !s.Inicio.EsValida() || !ok || s.Cantidad < 1 || s.Cantidad > maximo {
		return ErrPlazoInvalido
	}
	return nil
}

type DiaExcluido struct {
	Fecha       FechaCivil `json:"fecha"`
	FinDeSemana bool       `json:"fin_de_semana"`
	Motivos     []Motivo   `json:"motivos"`
}

type ResultadoPlazo struct {
	Inicio            FechaCivil    `json:"inicio"`
	PrimerDia         FechaCivil    `json:"primer_dia"`
	FinNominal        FechaCivil    `json:"fin_nominal"`
	Vencimiento       FechaCivil    `json:"vencimiento"`
	Prorrogado        bool          `json:"prorrogado"`
	VenceAntesDe      time.Time     `json:"vence_antes_de"`
	DuracionUltimoDia time.Duration `json:"-"`
	DiasExcluidos     []DiaExcluido `json:"dias_excluidos"`
}

// maximoIteraciones impide bucles si un calendario declarase años enteros
// inhábiles: nunca se busca más allá de este número de días.
const maximoIteraciones = 2000

// CalcularPlazo aplica el artículo 30 de la Ley 39/2015:
//   - 30.2: los plazos en días son hábiles salvo que se declaren naturales;
//     se excluyen sábados, domingos y festivos.
//   - 30.3 y 30.4: el cómputo empieza el día siguiente a la notificación o
//     publicación; en meses o años concluye el mismo día del mes de
//     vencimiento o, si no existe, el último día de ese mes.
//   - 30.5: si el último día es inhábil se prorroga al primer hábil siguiente.
func CalcularPlazo(s SolicitudPlazo, cal CalendarioComputo) (ResultadoPlazo, error) {
	if err := s.Validar(); err != nil {
		return ResultadoPlazo{}, err
	}
	if cal == nil {
		return ResultadoPlazo{}, ErrCalculoNoDeterminado
	}
	primer, err := s.Inicio.SumarDias(1)
	if err != nil {
		return ResultadoPlazo{}, ErrPlazoInvalido
	}
	r := ResultadoPlazo{Inicio: s.Inicio, PrimerDia: primer}
	var fin FechaCivil
	switch s.Unidad {
	case UnidadDiasHabiles:
		fin, err = contarHabiles(primer, s.Cantidad, cal, &r)
	case UnidadDiasNaturales:
		fin, err = s.Inicio.SumarDias(s.Cantidad)
	case UnidadMeses:
		fin, err = s.Inicio.SumarMesesMismoDia(s.Cantidad)
	case UnidadAnios:
		fin, err = s.Inicio.SumarMesesMismoDia(12 * s.Cantidad)
	}
	if err != nil {
		return ResultadoPlazo{}, err
	}
	r.FinNominal = fin
	vencimiento, err := prorrogar(fin, cal, &r)
	if err != nil {
		return ResultadoPlazo{}, err
	}
	r.Vencimiento = vencimiento
	r.Prorrogado = vencimiento != fin
	if r.VenceAntesDe, err = vencimiento.FinEnMadrid(); err != nil {
		return ResultadoPlazo{}, err
	}
	if r.DuracionUltimoDia, err = vencimiento.DuracionEnMadrid(); err != nil {
		return ResultadoPlazo{}, err
	}
	return r, nil
}

func contarHabiles(desde FechaCivil, cantidad int, cal CalendarioComputo, r *ResultadoPlazo) (FechaCivil, error) {
	actual, contados := desde, 0
	for i := 0; i < maximoIteraciones; i++ {
		inhabil, motivos, err := cal.EsInhabil(actual)
		if err != nil {
			return FechaCivil{}, err
		}
		if inhabil {
			r.DiasExcluidos = append(r.DiasExcluidos, DiaExcluido{Fecha: actual, FinDeSemana: actual.EsFinDeSemana(), Motivos: motivos})
		} else if contados++; contados == cantidad {
			return actual, nil
		}
		if actual, err = actual.SumarDias(1); err != nil {
			return FechaCivil{}, err
		}
	}
	return FechaCivil{}, ErrCalculoNoDeterminado
}

func prorrogar(fin FechaCivil, cal CalendarioComputo, r *ResultadoPlazo) (FechaCivil, error) {
	actual := fin
	for i := 0; i < maximoIteraciones; i++ {
		inhabil, motivos, err := cal.EsInhabil(actual)
		if err != nil {
			return FechaCivil{}, err
		}
		if !inhabil {
			return actual, nil
		}
		r.DiasExcluidos = append(r.DiasExcluidos, DiaExcluido{Fecha: actual, FinDeSemana: actual.EsFinDeSemana(), Motivos: motivos})
		if actual, err = actual.SumarDias(1); err != nil {
			return FechaCivil{}, err
		}
	}
	return FechaCivil{}, ErrCalculoNoDeterminado
}
