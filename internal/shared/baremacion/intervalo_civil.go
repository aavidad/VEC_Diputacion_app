package baremacion

import (
	"bytes"
	"encoding/json"
	"io"
)

// IntervaloCivil es siempre semiabierto: Desde se incluye y Hasta se excluye.
// Esta forma evita dobles conteos al unir periodos adyacentes.
type IntervaloCivil struct {
	desde FechaCivil
	hasta FechaCivil
}

// NuevoIntervaloCivil exige dos fechas validas y desde < hasta.
func NuevoIntervaloCivil(desde, hasta FechaCivil) (IntervaloCivil, error) {
	comparacion, err := desde.Comparar(hasta)
	if err != nil {
		return IntervaloCivil{}, nuevoError("intervalo_civil", CodigoFechaInvalida)
	}
	if comparacion >= 0 {
		return IntervaloCivil{}, nuevoError("intervalo_civil", CodigoIntervaloVacio)
	}
	return IntervaloCivil{desde: desde, hasta: hasta}, nil
}

// IntervaloDeUnDia construye [dia, dia+1). El 9999-12-31 no es representable
// como intervalo de un dia porque su extremo exclusivo quedaria fuera de
// FechaCivil; en ese caso devuelve ErrFueraDeLimites.
func IntervaloDeUnDia(dia FechaCivil) (IntervaloCivil, error) {
	siguiente, err := dia.Siguiente()
	if err != nil {
		return IntervaloCivil{}, err
	}
	return NuevoIntervaloCivil(dia, siguiente)
}

// Desde devuelve el extremo inclusivo.
func (i IntervaloCivil) Desde() FechaCivil { return i.desde }

// Hasta devuelve el extremo exclusivo.
func (i IntervaloCivil) Hasta() FechaCivil { return i.hasta }

// EsValido comprueba la forma semiabierta no vacia.
func (i IntervaloCivil) EsValido() bool {
	comparacion, err := i.desde.Comparar(i.hasta)
	return err == nil && comparacion < 0
}

// NumeroDias devuelve hasta-desde; siempre es positivo para un intervalo valido.
func (i IntervaloCivil) NumeroDias() (int64, error) {
	if !i.EsValido() {
		return 0, nuevoError("intervalo_civil", CodigoValorInvalido)
	}
	return i.desde.DiasHasta(i.hasta)
}

// Contiene aplica desde <= fecha < hasta.
func (i IntervaloCivil) Contiene(fecha FechaCivil) bool {
	if !i.EsValido() || !fecha.EsValida() {
		return false
	}
	desde, _ := i.desde.Comparar(fecha)
	hasta, _ := fecha.Comparar(i.hasta)
	return desde <= 0 && hasta < 0
}

// Solapa detecta una interseccion de al menos un dia civil.
func (i IntervaloCivil) Solapa(otro IntervaloCivil) bool {
	if !i.EsValido() || !otro.EsValido() {
		return false
	}
	inicioAntesDelFinAjeno, _ := i.desde.Comparar(otro.hasta)
	inicioAjenoAntesDelFin, _ := otro.desde.Comparar(i.hasta)
	return inicioAntesDelFinAjeno < 0 && inicioAjenoAntesDelFin < 0
}

// EsAdyacente detecta extremos coincidentes sin confundirlos con un solape.
func (i IntervaloCivil) EsAdyacente(otro IntervaloCivil) bool {
	if !i.EsValido() || !otro.EsValido() {
		return false
	}
	return i.hasta == otro.desde || otro.hasta == i.desde
}

func (i IntervaloCivil) MarshalJSON() ([]byte, error) {
	if !i.EsValido() {
		return nil, nuevoError("intervalo_civil", CodigoValorInvalido)
	}
	return json.Marshal(struct {
		Desde FechaCivil `json:"desde"`
		Hasta FechaCivil `json:"hasta"`
	}{Desde: i.desde, Hasta: i.hasta})
}

func (i *IntervaloCivil) UnmarshalJSON(datos []byte) error {
	if i == nil {
		return nuevoError("intervalo_civil", CodigoValorInvalido)
	}
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	apertura, err := decodificador.Token()
	if err != nil || apertura != json.Delim('{') {
		return nuevoError("intervalo_civil", CodigoValorNoCanonico)
	}

	var desde, hasta FechaCivil
	vistas := make(map[string]bool, 2)
	for decodificador.More() {
		tokenClave, err := decodificador.Token()
		if err != nil {
			return nuevoError("intervalo_civil", CodigoValorNoCanonico)
		}
		clave, esCadena := tokenClave.(string)
		if !esCadena || vistas[clave] || (clave != "desde" && clave != "hasta") {
			return nuevoError("intervalo_civil", CodigoValorNoCanonico)
		}
		vistas[clave] = true
		destino := &desde
		if clave == "hasta" {
			destino = &hasta
		}
		if err := decodificador.Decode(destino); err != nil {
			return nuevoError("intervalo_civil", CodigoValorNoCanonico)
		}
	}
	cierre, err := decodificador.Token()
	if err != nil || cierre != json.Delim('}') || !vistas["desde"] || !vistas["hasta"] {
		return nuevoError("intervalo_civil", CodigoValorNoCanonico)
	}
	if err := asegurarFinJSON(decodificador); err != nil {
		return nuevoError("intervalo_civil", CodigoValorNoCanonico)
	}
	construido, err := NuevoIntervaloCivil(desde, hasta)
	if err != nil {
		return err
	}
	*i = construido
	return nil
}

func asegurarFinJSON(decodificador *json.Decoder) error {
	var sobrante any
	err := decodificador.Decode(&sobrante)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return nuevoError("json", CodigoValorNoCanonico)
}
