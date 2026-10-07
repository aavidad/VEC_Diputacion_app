package domain

import (
	"fmt"
	"sort"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

type grupoTemporal struct {
	id        string
	unidades  b.Racional
	coef, max b.Puntos
}

func calcularPeriodos(c Configuracion, r Regla, e Entrada) ([]Detalle, b.Puntos, error) {
	if r.PermanenciaPolitica == "resto_provisional_primero_v1" {
		return calcularPermanenciaMixta(c, r, e)
	}
	detalles := []Detalle{}
	grupos := map[string]grupoTemporal{}
	for _, p := range e.Periodos {
		if !contiene(p.Familias, r.Familia) || !acepta(p.Tipo, r.Tipos) {
			continue
		}
		desde, hasta, ok := recortarRegla(c, r, p)
		diasBrutos := int64(0)
		finOriginal := c.FechaCorte
		if p.Hasta != nil {
			finOriginal = *p.Hasta
		}
		if p.Desde.EsValida() && finOriginal.EsValida() {
			diasBrutos, _ = p.Desde.DiasHasta(finOriginal)
			if diasBrutos < 0 {
				diasBrutos = 0
			}
		}
		detalle := Detalle{HechoID: p.ID, EvidenciaRef: p.EvidenciaRef, Motivo: "fuera_ventana", DiasBrutos: diasBrutos, Unidades: entero(0), FactorJornada: entero(1), Coeficiente: r.Coeficiente}
		if !ok {
			detalles = append(detalles, detalle)
			continue
		}
		detalle.Motivo = "computado"
		detalle.DiasElegibles, _ = desde.DiasHasta(hasta)
		unidades := entero(detalle.DiasElegibles)
		if r.Conversion.Metodo == "meses_completos" || r.Conversion.Metodo == "anos_desde_meses" {
			if r.Agrupacion == "por_nivel" && desde.Dia() != hasta.Dia() {
				return nil, b.Puntos{}, fallo("meses_no_acreditados", "periodos")
			}
			unidades = entero(mesesCompletos(desde, hasta))
		}
		factor := entero(1)
		if r.Jornada == "proporcional" || (r.Jornada == "protegida_integra" && p.AtestacionProtegidaRef == "") {
			factor = p.Jornada.Racional()
		}
		detalle.FactorJornada = factor
		var err error
		unidades, err = unidades.Multiplicar(factor)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		detalle.Unidades = unidades
		coef, max, id := r.Coeficiente, r.Maximo, "general"
		if r.Familia == ValoracionTrabajo {
			t, err := tramoPara(r, p.Nivel, e.NivelPuesto)
			if err != nil {
				return nil, b.Puntos{}, err
			}
			coef, max, id = t.Coeficiente, t.Maximo, t.ID
			detalle.TramoID = t.ID
			detalle.Coeficiente = t.Coeficiente
		}
		clave := id
		if r.Agrupacion == "por_periodo" {
			clave = p.ID + ":" + id
		} else if r.Agrupacion == "por_nivel" {
			// El tramo superior agrupa varios niveles; las fracciones sólo se
			// pueden sumar entre puestos con el mismo nivel de origen.
			clave = fmt.Sprintf("nivel:%d:%s", p.Nivel, id)
		}
		g, existe := grupos[clave]
		if !existe {
			g = grupoTemporal{id: id, unidades: entero(0), coef: coef, max: max}
		}
		g.unidades, err = g.unidades.Sumar(unidades)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		grupos[clave] = g
		detalles = append(detalles, detalle)
	}
	claves := make([]string, 0, len(grupos))
	for k := range grupos {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	bruto := b.Puntos{}
	// Tope por tramo, incluso si la conversión/redondeo se pide por periodo.
	sumasTramo := map[string]b.Puntos{}
	maximos := map[string]b.Puntos{}
	for _, k := range claves {
		g := grupos[k]
		unidades, err := convertirUnidades(g.unidades, *r.Conversion)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		puntos, err := g.coef.MultiplicarRedondeado(unidades, r.Redondeo)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		sumasTramo[g.id], err = sumasTramo[g.id].Sumar(puntos)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		maximos[g.id] = g.max
		detalles = append(detalles, Detalle{HechoID: "grupo:" + k, TramoID: g.id, Motivo: "conversion_y_coeficiente", Unidades: unidades, FactorJornada: entero(1), Coeficiente: g.coef, Bruto: puntos, Maximo: g.max, Resultado: tope(puntos, g.max)})
	}
	ids := make([]string, 0, len(sumasTramo))
	for k := range sumasTramo {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	for _, id := range ids {
		neto := tope(sumasTramo[id], maximos[id])
		var err error
		bruto, err = bruto.Sumar(neto)
		if err != nil {
			return nil, b.Puntos{}, err
		}
		detalles = append(detalles, Detalle{HechoID: "tope:" + id, TramoID: id, Motivo: "tope_tramo", Unidades: entero(0), FactorJornada: entero(1), Coeficiente: b.Puntos{}, Bruto: sumasTramo[id], Maximo: maximos[id], Resultado: neto})
	}
	return detalles, bruto, nil
}

// La política de ensayo detrae los meses no computables de la modalidad
// provisional y aplica su factor a la proporción de tiempo computable. Sólo
// acepta periodos de meses civiles completos: no convierte días sueltos en
// meses mediante un divisor supuesto.
func calcularPermanenciaMixta(c Configuracion, r Regla, e Entrada) ([]Detalle, b.Puntos, error) {
	detalles := []Detalle{}
	var definitivos, provisionales int64
	factor, _ := b.NuevoRacional(r.FactorProvisionalNumerador, r.FactorProvisionalDenominador)
	for _, p := range e.Periodos {
		if !contiene(p.Familias, Permanencia) || !acepta(p.Tipo, r.Tipos) {
			continue
		}
		desde, hasta, ok := recortarRegla(c, r, p)
		if !ok {
			detalles = append(detalles, Detalle{HechoID: p.ID, EvidenciaRef: p.EvidenciaRef, Motivo: "fuera_ventana", Unidades: entero(0), FactorJornada: entero(1), Coeficiente: r.Coeficiente})
			continue
		}
		if desde.Dia() != hasta.Dia() {
			return nil, b.Puntos{}, fallo("meses_no_acreditados", "periodos")
		}
		meses := mesesCompletos(desde, hasta)
		dias, _ := desde.DiasHasta(hasta)
		if p.Tipo == r.TipoProvisional {
			provisionales += meses
		} else {
			definitivos += meses
		}
		detalles = append(detalles, Detalle{HechoID: p.ID, EvidenciaRef: p.EvidenciaRef, Motivo: "meses_civiles_completos", DiasElegibles: dias, Unidades: entero(meses), FactorJornada: entero(1), Coeficiente: r.Coeficiente})
	}
	total := definitivos + provisionales
	if total == 0 {
		return detalles, b.Puntos{}, nil
	}
	divisor := r.Conversion.Divisor
	completos, resto := total/divisor, total%divisor
	if resto > r.Conversion.UmbralResto {
		completos++
	} else {
		descontarProvisional := min(resto, provisionales)
		provisionales -= descontarProvisional
		definitivos -= resto - descontarProvisional
	}
	computables := definitivos + provisionales
	if computables == 0 || completos == 0 {
		return detalles, b.Puntos{}, nil
	}
	if definitivos > 0 && provisionales > 0 {
		detalles = append(detalles, Detalle{HechoID: "grupo:permanencia", Motivo: "reparto_mixto_pendiente", Unidades: entero(computables), FactorJornada: entero(1), Coeficiente: r.Coeficiente})
		return detalles, b.Puntos{}, fallo("politica_pendiente", "permanencia")
	}
	ponderados, err := entero(provisionales).Multiplicar(factor)
	if err != nil {
		return nil, b.Puntos{}, err
	}
	ponderados, err = ponderados.Sumar(entero(definitivos))
	if err != nil {
		return nil, b.Puntos{}, err
	}
	unidades, err := ponderados.Multiplicar(entero(completos))
	if err != nil {
		return nil, b.Puntos{}, err
	}
	unidades, err = unidades.Dividir(entero(computables))
	if err != nil {
		return nil, b.Puntos{}, err
	}
	puntos, err := r.Coeficiente.MultiplicarRedondeado(unidades, r.Redondeo)
	if err != nil {
		return nil, b.Puntos{}, err
	}
	motivo, corrector := "resto_sin_corrector", ""
	if provisionales > 0 {
		motivo, corrector = "corrector_provisional_y_resto", factor.String()
	}
	detalles = append(detalles, Detalle{HechoID: "grupo:permanencia", Motivo: motivo, Unidades: unidades, FactorJornada: entero(1), CorrectorProvisional: corrector, Coeficiente: r.Coeficiente, Bruto: puntos, Maximo: r.Maximo, Resultado: tope(puntos, r.Maximo)})
	return detalles, puntos, nil
}

func recortar(c Configuracion, p Periodo) (b.FechaCivil, b.FechaCivil, bool) {
	desde, hasta := p.Desde, c.FechaCorte
	if p.Hasta != nil {
		cmp, _ := p.Hasta.Comparar(hasta)
		if cmp < 0 {
			hasta = *p.Hasta
		}
	}
	cmp, _ := desde.Comparar(c.VentanaDesde)
	if cmp < 0 {
		desde = c.VentanaDesde
	}
	cmp, _ = desde.Comparar(hasta)
	return desde, hasta, cmp < 0
}

func recortarRegla(c Configuracion, r Regla, p Periodo) (b.FechaCivil, b.FechaCivil, bool) {
	desde, hasta, ok := recortar(c, p)
	if r.VentanaDesde == "" {
		return desde, hasta, ok
	}
	ventana, _ := fechaVentanaRegla(r.VentanaDesde) // validada al abrir el cálculo
	if anterior, _ := desde.Comparar(ventana); anterior < 0 {
		desde = ventana
	}
	orden, _ := desde.Comparar(hasta)
	return desde, hasta, orden < 0
}
func mesesCompletos(desde, hasta b.FechaCivil) int64 {
	meses := int64((hasta.Anio()-desde.Anio())*12 + hasta.Mes() - desde.Mes())
	if hasta.Dia() < desde.Dia() {
		meses--
	}
	if meses < 0 {
		return 0
	}
	return meses
}
func convertirUnidades(unidades b.Racional, c Conversion) (b.Racional, error) {
	switch c.Metodo {
	case "dias_racionales", "meses_completos":
		return unidades.Dividir(entero(c.Divisor))
	case "dias_completos":
		valor, err := unidades.Dividir(entero(c.Divisor))
		if err != nil {
			return b.Racional{}, err
		}
		return entero(valor.Numerador() / valor.Denominador()), nil
	case "anos_desde_meses":
		valor, err := unidades.Dividir(entero(c.Divisor))
		if err != nil {
			return b.Racional{}, err
		}
		anos := valor.Numerador() / valor.Denominador()
		resto, err := unidades.Restar(entero(anos * c.Divisor))
		if err != nil {
			return b.Racional{}, err
		}
		cmp, err := resto.Comparar(entero(c.UmbralResto))
		if err != nil {
			return b.Racional{}, err
		}
		if cmp > 0 {
			anos++
		}
		return entero(anos), nil
	}
	return b.Racional{}, fallo("conversion_invalida", "metodo")
}

func validarSolapes(c Configuracion, e Entrada) error {
	for _, familia := range []Familia{Antiguedad, Permanencia, ValoracionTrabajo} {
		intervalos := []b.IntervaloCivil{}
		for _, p := range e.Periodos {
			if !contiene(p.Familias, familia) {
				continue
			}
			seleccionado := false
			for _, r := range c.Reglas {
				if r.Familia == familia && acepta(p.Tipo, r.Tipos) {
					seleccionado = true
					break
				}
			}
			if !seleccionado {
				continue
			}
			desde, hasta, ok := recortar(c, p)
			if !ok {
				continue
			}
			i, _ := b.NuevoIntervaloCivil(desde, hasta)
			intervalos = append(intervalos, i)
		}
		sort.Slice(intervalos, func(i, j int) bool { cmp, _ := intervalos[i].Desde().Comparar(intervalos[j].Desde()); return cmp < 0 })
		for i := 1; i < len(intervalos); i++ {
			if intervalos[i-1].Solapa(intervalos[i]) {
				return fallo("periodos_solapados", string(familia))
			}
		}
	}
	return nil
}
