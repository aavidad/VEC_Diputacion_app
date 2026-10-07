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
) (*ports.ResumenCuadroRRHH, error) {
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
	hoy, err := diaCivilMadrid(ahora)
	if err != nil {
		// Sin la zona de Madrid no se sabe qué vence esta semana: la consulta
		// falla en lugar de publicar ceros que no lo son.
		return nil, err
	}
	for _, grupo := range agregados.GruposPlazo {
		if err := ctx.Err(); err != nil {
			return nil, err
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
		clase, err := clasificarPlazoCuadroRRHH(plazo, hoy)
		if err != nil {
			return nil, err
		}
		if clase.sinCalcular {
			resumen.SinCalcular += grupo.Numero
			continue
		}
		if clase.vencido {
			resumen.Vencidos += grupo.Numero
		}
		if clase.venceHoy {
			resumen.VencenHoy += grupo.Numero
		}
		if clase.venceSemana {
			resumen.VencenSemana += grupo.Numero
		}
	}
	return resumen, nil
}

func diaCivilMadrid(instante time.Time) (string, error) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return "", err
	}
	return instante.In(zona).Format(time.DateOnly), nil
}

func diasEntreDiasCiviles(desde, hasta string) (int, error) {
	a, err := time.Parse(time.DateOnly, desde)
	if err != nil {
		return 0, err
	}
	b, err := time.Parse(time.DateOnly, hasta)
	if err != nil {
		return 0, err
	}
	return int(b.Sub(a).Hours() / 24), nil
}
