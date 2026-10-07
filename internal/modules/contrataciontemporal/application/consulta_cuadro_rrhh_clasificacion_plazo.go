package application

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type clasificacionPlazoCuadroRRHH struct {
	vencido     bool
	venceHoy    bool
	venceSemana bool
	sinCalcular bool
}

// clasificarPlazoCuadroRRHH es la única comparación temporal de la portada
// y del filtro que abre su lista. La calculadora aporta regla y calendario.
func clasificarPlazoCuadroRRHH(
	plazo *ports.PlazoFaseRRHH,
	hoy string,
) (clasificacionPlazoCuadroRRHH, error) {
	var clase clasificacionPlazoCuadroRRHH
	if plazo == nil {
		return clase, nil
	}
	if plazo.Estado == ports.PlazoFaseNoCalculado {
		clase.sinCalcular = true
		return clase, nil
	}
	clase.vencido = plazo.Estado == ports.PlazoFaseVencido
	clase.venceHoy = plazo.Estado == ports.PlazoFaseVenceHoy
	dias, err := diasEntreDiasCiviles(hoy, plazo.UltimoDia)
	if err != nil {
		return clasificacionPlazoCuadroRRHH{}, err
	}
	clase.venceSemana = dias >= 0 && dias <= diasSemanaPortada
	return clase, nil
}
