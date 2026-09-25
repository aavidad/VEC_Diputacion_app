// Package domain contiene las reglas neutrales del contexto Calendarios:
// fechas civiles, ámbitos, versiones de calendario y cómputo de plazos. No
// conoce HTTP, SQL ni proveedores.
package domain

import (
	"errors"
	"time"
	_ "time/tzdata" // La zona oficial no depende del sistema anfitrión.
)

// ZonaOficial es la zona de la hora legal peninsular (Real Decreto 236/2002).
const ZonaOficial = "Europe/Madrid"

var ErrFechaInvalida = errors.New("calendarios: fecha civil invalida")

var zonaMadrid = func() *time.Location {
	zona, err := time.LoadLocation(ZonaOficial)
	if err != nil {
		panic("calendarios: zona oficial no disponible")
	}
	return zona
}()

// ZonaMadrid devuelve la zona oficial usada para convertir instantes en
// fechas civiles y para expresar el final de un plazo como instante.
func ZonaMadrid() *time.Location { return zonaMadrid }

// FechaCivil es un día del calendario gregoriano, sin hora ni zona. Solo es
// válida si procede de NuevaFechaCivil, ParsearFechaCivil o FechaCivilDe.
type FechaCivil struct {
	anio, mes, dia int
}

const (
	anioMinimo = 1900
	anioMaximo = 2200
)

func NuevaFechaCivil(anio, mes, dia int) (FechaCivil, error) {
	if anio < anioMinimo || anio > anioMaximo || mes < 1 || mes > 12 || dia < 1 || dia > DiasDelMes(anio, mes) {
		return FechaCivil{}, ErrFechaInvalida
	}
	return FechaCivil{anio: anio, mes: mes, dia: dia}, nil
}

// ParsearFechaCivil admite únicamente la forma canónica AAAA-MM-DD.
func ParsearFechaCivil(texto string) (FechaCivil, error) {
	if len(texto) != 10 || texto[4] != '-' || texto[7] != '-' {
		return FechaCivil{}, ErrFechaInvalida
	}
	numero := func(s string) (int, bool) {
		n := 0
		for _, c := range s {
			if c < '0' || c > '9' {
				return 0, false
			}
			n = n*10 + int(c-'0')
		}
		return n, true
	}
	a, okA := numero(texto[0:4])
	m, okM := numero(texto[5:7])
	d, okD := numero(texto[8:10])
	if !okA || !okM || !okD {
		return FechaCivil{}, ErrFechaInvalida
	}
	return NuevaFechaCivil(a, m, d)
}

// FechaCivilDe convierte un instante en la fecha civil de Europe/Madrid.
func FechaCivilDe(instante time.Time) (FechaCivil, error) {
	if instante.IsZero() {
		return FechaCivil{}, ErrFechaInvalida
	}
	local := instante.In(zonaMadrid)
	return NuevaFechaCivil(local.Year(), int(local.Month()), local.Day())
}

func DiasDelMes(anio, mes int) int {
	switch mes {
	case 2:
		if EsBisiesto(anio) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	default:
		return 31
	}
}

func EsBisiesto(anio int) bool {
	return anio%4 == 0 && (anio%100 != 0 || anio%400 == 0)
}

func (f FechaCivil) Anio() int      { return f.anio }
func (f FechaCivil) Mes() int       { return f.mes }
func (f FechaCivil) Dia() int       { return f.dia }
func (f FechaCivil) EsValida() bool { return f.anio != 0 }

func (f FechaCivil) String() string {
	if !f.EsValida() {
		return ""
	}
	return f.medianoche().Format(time.DateOnly)
}

func (f FechaCivil) MarshalText() ([]byte, error) {
	if !f.EsValida() {
		return nil, ErrFechaInvalida
	}
	return []byte(f.String()), nil
}

func (f *FechaCivil) UnmarshalText(texto []byte) error {
	v, err := ParsearFechaCivil(string(texto))
	if err != nil {
		return err
	}
	*f = v
	return nil
}

// medianoche es una representación técnica en UTC: la aritmética de días
// civiles no se ve afectada por cambios horarios.
func (f FechaCivil) medianoche() time.Time {
	return time.Date(f.anio, time.Month(f.mes), f.dia, 0, 0, 0, 0, time.UTC)
}

func desdeMedianoche(t time.Time) (FechaCivil, error) {
	return NuevaFechaCivil(t.Year(), int(t.Month()), t.Day())
}

// SumarDias desplaza la fecha n días civiles.
func (f FechaCivil) SumarDias(n int) (FechaCivil, error) {
	if !f.EsValida() {
		return FechaCivil{}, ErrFechaInvalida
	}
	return desdeMedianoche(f.medianoche().AddDate(0, 0, n))
}

// SumarMesesMismoDia aplica la regla del artículo 30.4 de la Ley 39/2015: el
// plazo concluye el mismo día del mes de vencimiento y, si ese mes no tiene
// día equivalente, el último día del mes.
func (f FechaCivil) SumarMesesMismoDia(meses int) (FechaCivil, error) {
	if !f.EsValida() || meses < 0 {
		return FechaCivil{}, ErrFechaInvalida
	}
	total := f.anio*12 + (f.mes - 1) + meses
	anio, mes := total/12, total%12+1
	dia := f.dia
	if ultimo := DiasDelMes(anio, mes); dia > ultimo {
		dia = ultimo
	}
	return NuevaFechaCivil(anio, mes, dia)
}

// DiaSemana usa la convención de Go: domingo es 0 y sábado 6.
func (f FechaCivil) DiaSemana() time.Weekday { return f.medianoche().Weekday() }

func (f FechaCivil) EsFinDeSemana() bool {
	d := f.DiaSemana()
	return d == time.Saturday || d == time.Sunday
}

func (f FechaCivil) Antes(otra FechaCivil) bool { return f.medianoche().Before(otra.medianoche()) }
func (f FechaCivil) Igual(otra FechaCivil) bool { return f == otra }

// InicioEnMadrid es el primer instante de la fecha en la hora oficial.
// Los días de cambio horario duran 23 o 25 horas; por eso el final de un día
// se obtiene como el inicio del siguiente y nunca sumando 24 horas.
func (f FechaCivil) InicioEnMadrid() time.Time {
	return time.Date(f.anio, time.Month(f.mes), f.dia, 0, 0, 0, 0, zonaMadrid).UTC()
}

// FinEnMadrid es el instante exclusivo en que termina la fecha.
func (f FechaCivil) FinEnMadrid() (time.Time, error) {
	siguiente, err := f.SumarDias(1)
	if err != nil {
		return time.Time{}, err
	}
	return siguiente.InicioEnMadrid(), nil
}

// DuracionEnMadrid devuelve la duración real del día civil (23, 24 o 25 h).
func (f FechaCivil) DuracionEnMadrid() (time.Duration, error) {
	fin, err := f.FinEnMadrid()
	if err != nil {
		return 0, err
	}
	return fin.Sub(f.InicioEnMadrid()), nil
}
