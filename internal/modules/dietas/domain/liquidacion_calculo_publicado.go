package domain

import (
	"math"
	"time"
)

// CalcularBasesLiquidacionPublicada coteja los hechos declarados con una
// publicación exacta. No reconoce gasto: devuelve el importe calculado y la
// diferencia para que la persona competente tome una decisión motivada.
func CalcularBasesLiquidacionPublicada(fechaInicio, horaInicio, fechaFin, horaFin string,
	d DocumentoComision, c CatalogoLiquidacionPublicado) ([]LineaLiquidacionPropuesta, TotalesLiquidacionPropuesta, error) {
	var vacio TotalesLiquidacionPropuesta
	if c.ValidarForma() != nil || !fechaCivilValida(fechaInicio) || !fechaCivilValida(fechaFin) ||
		!horaCivil(horaInicio) || !horaCivil(horaFin) || fechaInicio > fechaFin ||
		int(d.GrupoDieta) != c.Grupo || d.VehiculoPropio != (c.Vehiculo == "propio") ||
		d.AlojamientoTopeCentimos != 0 || len(d.Lineas) < 1 || len(d.Lineas) > MaximoLineasLiquidacionPublicada ||
		fechaInicio < c.VigenteDesde || fechaFin > c.VigenteHasta {
		return nil, vacio, ErrLiquidacionPublicada
	}
	zona, err := time.LoadLocation(c.ReglaDevengo.Configuracion.Zona)
	if err != nil {
		return nil, vacio, ErrLiquidacionPublicada
	}
	inicio, err := ResolverInstanteCivil(fechaInicio, horaInicio, zona)
	if err != nil {
		return nil, vacio, ErrLiquidacionPublicada
	}
	fin, err := ResolverInstanteCivil(fechaFin, horaFin, zona)
	if err != nil || !fin.After(inicio) {
		return nil, vacio, ErrLiquidacionPublicada
	}
	tramos, err := CalcularTramosNacionalesPublicados(inicio, fin, c)
	if err != nil {
		return nil, vacio, ErrLiquidacionPublicada
	}
	reglas := make(map[string]ReglaLiquidacionPublicada, len(c.Reglas))
	for _, r := range c.Reglas {
		reglas[r.Tipo+":"+r.Concepto] = r
	}
	lineas := make([]LineaLiquidacionPropuesta, 0, len(d.Lineas))
	var totales TotalesLiquidacionPropuesta
	var manutencion, kilometraje, otros, kmEscalados int64
	tramosLeidos, rutasLeidas := 0, 0
	for i, l := range d.Lineas {
		if l.ImporteCentimos < 0 || (l.Fecha != "" && (l.Fecha < fechaInicio || l.Fecha > fechaFin)) {
			return nil, vacio, ErrLiquidacionPublicada
		}
		concepto := l.Concepto
		if l.Tipo == "kilometraje" {
			concepto = "vehiculo_propio"
		} else if l.Tipo == ClaseOtroMedio || l.Tipo == ClaseOtroGasto {
			concepto = l.TipoGasto
		}
		r, ok := reglas[l.Tipo+":"+concepto]
		if !ok || (l.Tipo == "dieta" && r.Grupo != int(d.GrupoDieta)) {
			return nil, vacio, ErrLiquidacionPublicada
		}
		var calculado int64
		switch l.Tipo {
		case "dieta":
			if l.Concepto != "manutencion" || l.VersionTarifaRef != d.VersionTarifaAceptada || l.Rotulo != c.RotuloClave ||
				l.IndiceTramo == nil || *l.IndiceTramo < 0 ||
				*l.IndiceTramo >= len(tramos.Tramos) || tramosLeidos >= len(d.TramosAceptados) ||
				d.TramosAceptados[tramosLeidos] != *l.IndiceTramo || l.Fecha != tramos.Tramos[*l.IndiceTramo].Fecha ||
				tramos.Tramos[*l.IndiceTramo].Tipo != "manutencion" {
				return nil, vacio, ErrLiquidacionPublicada
			}
			calculado = tramos.Tramos[*l.IndiceTramo].ImporteCentimos
			tramosLeidos++
			if !sumarLiquidacion(&manutencion, l.ImporteCentimos) {
				return nil, vacio, ErrLiquidacionPublicada
			}
		case "kilometraje":
			if !d.VehiculoPropio || l.VersionTarifaRef != d.VersionTarifaAceptada || l.Rotulo != c.RotuloClave ||
				l.RutaIndice != rutasLeidas+1 || l.RutaIndice > 8 ||
				!referenciaJustificante.MatchString(l.VersionGrafo) || !codigoRuta.MatchString(l.OrigenCodigo) ||
				!codigoRuta.MatchString(l.DestinoCodigo) || l.OrigenCodigo == l.DestinoCodigo {
				return nil, vacio, ErrLiquidacionPublicada
			}
			cantidad, e := decimal4(l.Kilometros)
			if e != nil || cantidad < 1 || cantidad > 10000*10000-kmEscalados ||
				cantidad > (math.MaxInt64-5000)/r.CentimosPorKM {
				return nil, vacio, ErrLiquidacionPublicada
			}
			base, e := decimal4(l.KilometrosBase)
			ajuste, ok := ajusteEscalado(l.AjusteKilometros)
			if e != nil || base < 1 || !ok || !ajusteTextoValido(l.AjusteKilometros, l.MotivoAjuste) ||
				base+ajuste != cantidad {
				return nil, vacio, ErrLiquidacionPublicada
			}
			kmEscalados += cantidad
			calculado = (cantidad*r.CentimosPorKM + 5000) / 10000
			rutasLeidas++
			if !sumarLiquidacion(&kilometraje, l.ImporteCentimos) {
				return nil, vacio, ErrLiquidacionPublicada
			}
		case ClaseOtroMedio, ClaseOtroGasto:
			if l.JustificanteRef == nil || l.JustificanteSHA256 == nil ||
				validarOtroGastoPublicado(OtroGastoDeclarado{Tipo: l.Tipo, Concepto: l.Concepto,
					ImporteCentimos: l.ImporteCentimos, JustificanteRef: *l.JustificanteRef,
					JustificanteSHA256: *l.JustificanteSHA256, TipoGasto: l.TipoGasto,
					CatalogoVersion: l.CatalogoVersion, Fecha: l.Fecha}) != nil ||
				l.CatalogoVersion != c.CatalogoOtrosGastosVersionRef {
				return nil, vacio, ErrLiquidacionPublicada
			}
			calculado = l.ImporteCentimos
			if calculado > r.TopeCentimos {
				calculado = r.TopeCentimos
			}
			if !sumarLiquidacion(&otros, l.ImporteCentimos) {
				return nil, vacio, ErrLiquidacionPublicada
			}
		default:
			return nil, vacio, ErrLiquidacionPublicada
		}
		if !sumarLiquidacion(&totales.OriginalCentimos, l.ImporteCentimos) ||
			!sumarLiquidacion(&totales.CalculadoPublicadoCentimos, calculado) {
			return nil, vacio, ErrLiquidacionPublicada
		}
		lineas = append(lineas, LineaLiquidacionPropuesta{Indice: i, ReglaRef: r.Referencia,
			OriginalCentimos: l.ImporteCentimos, CalculadoPublicadoCentimos: calculado,
			DiferenciaCentimos: calculado - l.ImporteCentimos})
	}
	if tramosLeidos != len(d.TramosAceptados) || manutencion != d.ManutencionCentimos ||
		kilometraje != d.KilometrajeCentimos || otros != d.OtrosCentimos ||
		totales.OriginalCentimos != d.TotalOrientativoCentimos ||
		(d.VehiculoPropio && rutasLeidas == 0) || (!d.VehiculoPropio && rutasLeidas != 0) {
		return nil, vacio, ErrLiquidacionPublicada
	}
	totales.DiferenciaCentimos = totales.CalculadoPublicadoCentimos - totales.OriginalCentimos
	return lineas, totales, nil
}
