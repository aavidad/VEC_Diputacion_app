package domain

import (
	"sort"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

func calcularCursos(c Configuracion, r Regla, e Entrada) ([]Detalle, b.Puntos, error) {
	detalles := []Detalle{}
	horas := entero(0)
	cuenta := 0
	cursos := append([]Curso{}, e.Cursos...)
	if r.MaximoElementos > 0 {
		sort.SliceStable(cursos, func(i, j int) bool { cmp, _ := cursos[i].Horas.Comparar(cursos[j].Horas); return cmp > 0 })
	}
	for _, x := range cursos {
		motivo := "computado"
		unidades := entero(0)
		cmp, _ := x.Fecha.Comparar(c.FechaCorte)
		minimo, _ := x.Horas.Comparar(*r.HorasMinimas)
		switch {
		case !acepta(x.Tipo, r.Tipos):
			motivo = "tipo_no_admitido"
		case !x.Acreditado:
			motivo = "no_acreditado"
		case !x.Relacionado:
			motivo = "no_relacionado"
		case cmp >= 0:
			motivo = "posterior_corte"
		case caducado(x.VigenteHasta, c.FechaCorte):
			motivo = "caducado"
		case minimo < 0:
			motivo = "horas_insuficientes"
		case r.MaximoElementos > 0 && cuenta >= r.MaximoElementos:
			motivo = "limite_elementos"
		default:
			unidades = x.Horas
			cuenta++
		}
		var err error
		horas, err = horas.Sumar(unidades)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		detalles = append(detalles, Detalle{HechoID: x.ID, EvidenciaRef: x.EvidenciaRef, Motivo: motivo, Unidades: unidades, FactorJornada: entero(1), Coeficiente: r.Coeficiente})
	}
	bruto, err := r.Coeficiente.MultiplicarRedondeado(horas, r.Redondeo)
	if err != nil {
		return nil, b.Puntos{}, err
	}
	detalles = append(detalles, Detalle{HechoID: "grupo:cursos", Motivo: "suma_horas", Unidades: horas, FactorJornada: entero(1), Coeficiente: r.Coeficiente, Bruto: bruto, Maximo: r.Maximo, Resultado: tope(bruto, r.Maximo)})
	return detalles, bruto, nil
}
func caducado(hasta *b.FechaCivil, corte b.FechaCivil) bool {
	if hasta == nil {
		return false
	}
	cmp, _ := hasta.Comparar(corte)
	return cmp < 0
}
func calcularTitulos(c Configuracion, r Regla, e Entrada) ([]Detalle, b.Puntos, error) {
	detalles := []Detalle{}
	total := b.Puntos{}
	cuenta := 0
	for _, x := range e.Titulaciones {
		motivo := "computado"
		unidades := entero(0)
		cmp, _ := x.Fecha.Comparar(c.FechaCorte)
		switch {
		case !acepta(x.Tipo, r.Tipos):
			motivo = "tipo_no_admitido"
		case !x.Acreditado:
			motivo = "no_acreditado"
		case cmp >= 0:
			motivo = "posterior_corte"
		case x.UsadoRequisito && r.ExcluirRequisito:
			motivo = "usado_requisito"
		case r.MaximoElementos > 0 && cuenta >= r.MaximoElementos:
			motivo = "limite_elementos"
		default:
			unidades = entero(1)
			cuenta++
		}
		puntos, err := r.Coeficiente.MultiplicarRedondeado(unidades, r.Redondeo)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		total, err = total.Sumar(puntos)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		detalles = append(detalles, Detalle{HechoID: x.ID, EvidenciaRef: x.EvidenciaRef, Motivo: motivo, Unidades: unidades, FactorJornada: entero(1), Coeficiente: r.Coeficiente, Bruto: puntos, Maximo: r.Maximo, Resultado: puntos})
	}
	return detalles, total, nil
}
