package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// diasSemanaPortada: «vencen esta semana» es del día de la consulta al sexto
// siguiente, como en la portada.
const diasSemanaPortada = 6

// resumirCuadroRRHH calcula el resumen de la portada a partir de los
// agregados de todo el corte filtrado. Los plazos se resuelven una vez por
// grupo con la misma calculadora que la página; un grupo cuyo plazo no se
// pudo calcular cuenta como «sin calcular», nunca como en plazo.
func resumirCuadroRRHH(
	ctx context.Context,
	calculadora ports.CalculadoraPlazoFaseRRHH,
	agregados ports.AgregadosCuadroRRHH,
	ahora time.Time,
) *ports.ResumenCuadroRRHH {
	resumen := &ports.ResumenCuadroRRHH{PorFase: make(map[domain.ClaveFase]uint64)}
	for _, recuento := range agregados.Recuentos {
		if ports.EstadoTerminado(recuento.EstadoClave) {
			continue
		}
		resumen.EnTramite += recuento.Numero
		resumen.PorFase[recuento.FaseClave] += recuento.Numero
		if recuento.EstadoClave == domain.EstadoIncidencia {
			resumen.ConIncidencia += recuento.Numero
		}
	}
	hoy := diaCivilMadrid(ahora)
	if hoy == "" {
		// Sin la zona de Madrid no se sabe qué vence esta semana: no se
		// publica un resumen con ceros que no lo son.
		return nil
	}
	for _, grupo := range agregados.GruposPlazo {
		if ctx.Err() != nil {
			return nil
		}
		if calculadora == nil {
			continue
		}
		plazo := calcularPlazoFase(ctx, calculadora, clavePlazoFaseCuadro{
			fase: grupo.FaseClave, desde: grupo.Desde, urgente: grupo.Urgente,
		}, ahora)
		if plazo == nil {
			// La fase no tiene plazo: no cuenta en ningún recuento de plazos.
			continue
		}
		switch plazo.Estado {
		case ports.PlazoFaseNoCalculado:
			resumen.SinCalcular += grupo.Numero
			continue
		case ports.PlazoFaseVencido:
			resumen.Vencidos += grupo.Numero
		case ports.PlazoFaseVenceHoy:
			resumen.VencenHoy += grupo.Numero
		}
		// Como la portada: por el último día del plazo, de hoy a seis días.
		if dias, valido := diasEntreDiasCiviles(hoy, plazo.UltimoDia); valido && dias >= 0 && dias <= diasSemanaPortada {
			resumen.VencenSemana += grupo.Numero
		}
	}
	return resumen
}

func diaCivilMadrid(instante time.Time) string {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return ""
	}
	return instante.In(zona).Format(time.DateOnly)
}

func diasEntreDiasCiviles(desde, hasta string) (int, bool) {
	a, errA := time.Parse(time.DateOnly, desde)
	b, errB := time.Parse(time.DateOnly, hasta)
	if errA != nil || errB != nil {
		return 0, false
	}
	return int(b.Sub(a).Hours() / 24), true
}
