package baremacion

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const (
	// AnioCivilMinimo es el primer anio representable.
	AnioCivilMinimo = 1
	// AnioCivilMaximo mantiene la forma canonica de cuatro cifras.
	AnioCivilMaximo = 9999
)

// FechaCivil es una fecha del calendario gregoriano sin hora ni zona horaria.
type FechaCivil struct {
	anio int
	mes  int
	dia  int
}

// NuevaFechaCivil valida los componentes sin asociarlos a una zona horaria.
func NuevaFechaCivil(anio, mes, dia int) (FechaCivil, error) {
	if anio < AnioCivilMinimo || anio > AnioCivilMaximo || mes < 1 || mes > 12 ||
		dia < 1 || dia > diasDelMes(anio, mes) {
		return FechaCivil{}, nuevoError("fecha_civil", CodigoFechaInvalida)
	}
	return FechaCivil{anio: anio, mes: mes, dia: dia}, nil
}

// Anio devuelve el anio civil.
func (f FechaCivil) Anio() int { return f.anio }

// Mes devuelve el mes civil, entre 1 y 12.
func (f FechaCivil) Mes() int { return f.mes }

// Dia devuelve el dia del mes civil.
func (f FechaCivil) Dia() int { return f.dia }

// EsValida indica si la fecha satisface el calendario gregoriano soportado.
func (f FechaCivil) EsValida() bool {
	_, err := NuevaFechaCivil(f.anio, f.mes, f.dia)
	return err == nil
}

// Comparar devuelve -1, 0 o 1 segun el orden civil.
func (f FechaCivil) Comparar(otra FechaCivil) (int, error) {
	if !f.EsValida() || !otra.EsValida() {
		return 0, nuevoError("fecha_civil", CodigoFechaInvalida)
	}
	izquierda, derecha := f.numeroOrdinal(), otra.numeroOrdinal()
	if izquierda < derecha {
		return -1, nil
	}
	if izquierda > derecha {
		return 1, nil
	}
	return 0, nil
}

// DiasHasta calcula otra-f. Puede devolver un numero negativo.
func (f FechaCivil) DiasHasta(otra FechaCivil) (int64, error) {
	if !f.EsValida() || !otra.EsValida() {
		return 0, nuevoError("fecha_civil", CodigoFechaInvalida)
	}
	return otra.numeroOrdinal() - f.numeroOrdinal(), nil
}

// Siguiente devuelve el dia civil inmediato sin crear una hora intermedia.
func (f FechaCivil) Siguiente() (FechaCivil, error) {
	if !f.EsValida() {
		return FechaCivil{}, nuevoError("fecha_civil", CodigoFechaInvalida)
	}
	if f.dia < diasDelMes(f.anio, f.mes) {
		return NuevaFechaCivil(f.anio, f.mes, f.dia+1)
	}
	if f.mes < 12 {
		return NuevaFechaCivil(f.anio, f.mes+1, 1)
	}
	if f.anio == AnioCivilMaximo {
		return FechaCivil{}, nuevoError("fecha_civil", CodigoFueraDeLimites)
	}
	return NuevaFechaCivil(f.anio+1, 1, 1)
}

func (f FechaCivil) String() string {
	if !f.EsValida() {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", f.anio, f.mes, f.dia)
}

func (f FechaCivil) MarshalJSON() ([]byte, error) {
	if !f.EsValida() {
		return nil, nuevoError("fecha_civil", CodigoFechaInvalida)
	}
	return json.Marshal(f.String())
}

func (f *FechaCivil) UnmarshalJSON(datos []byte) error {
	if f == nil {
		return nuevoError("fecha_civil", CodigoFechaInvalida)
	}
	var texto string
	if err := json.Unmarshal(datos, &texto); err != nil || len(texto) != 10 || texto[4] != '-' || texto[7] != '-' {
		return nuevoError("fecha_civil", CodigoValorNoCanonico)
	}
	anio, errAnio := strconv.Atoi(texto[0:4])
	mes, errMes := strconv.Atoi(texto[5:7])
	dia, errDia := strconv.Atoi(texto[8:10])
	if errAnio != nil || errMes != nil || errDia != nil {
		return nuevoError("fecha_civil", CodigoValorNoCanonico)
	}
	construida, err := NuevaFechaCivil(anio, mes, dia)
	if err != nil {
		return err
	}
	if construida.String() != texto {
		return nuevoError("fecha_civil", CodigoValorNoCanonico)
	}
	*f = construida
	return nil
}

func (f FechaCivil) numeroOrdinal() int64 {
	anioAnterior := int64(f.anio - 1)
	dias := 365*anioAnterior + anioAnterior/4 - anioAnterior/100 + anioAnterior/400
	acumulados := [...]int{0, 0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	dias += int64(acumulados[f.mes] + f.dia - 1)
	if f.mes > 2 && esBisiesto(f.anio) {
		dias++
	}
	return dias
}

func esBisiesto(anio int) bool {
	return anio%4 == 0 && (anio%100 != 0 || anio%400 == 0)
}

func diasDelMes(anio, mes int) int {
	dias := [...]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if mes < 1 || mes > 12 {
		return 0
	}
	if mes == 2 && esBisiesto(anio) {
		return 29
	}
	return dias[mes]
}
