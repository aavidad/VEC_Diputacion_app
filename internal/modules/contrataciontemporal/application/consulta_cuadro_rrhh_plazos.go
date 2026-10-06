package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ConfigurarPlazosFase compone la calculadora de plazos por fase. Se llama una
// sola vez al componer, antes de servir; nil deja el cuadro sin plazos.
func (s *ServicioConsultaCuadroRRHH) ConfigurarPlazosFase(
	calculadora ports.CalculadoraPlazoFaseRRHH,
) {
	if s == nil || dependenciaNula(calculadora) {
		return
	}
	s.plazos = calculadora
}

type clavePlazoFaseCuadro struct {
	fase    domain.ClaveFase
	desde   time.Time
	urgente bool
}

// completarPlazos devuelve el vencimiento de la fase actual de cada
// expediente de la página y, si la consulta trae agregados, el resumen de la
// portada, con una sola preparación de reglas. Sin calculadora o sin fecha de
// entrada en fase no hay plazos; un cálculo fallido queda como «no
// calculado» en lugar de suponer uno.
func (s *ServicioConsultaCuadroRRHH) completarPlazos(
	ctx context.Context,
	pagina ports.PaginaCuadroRRHH,
) ([]*ports.PlazoFaseRRHH, *ports.ResumenCuadroRRHH, error) {
	if s == nil || s.reloj == nil {
		return nil, nil, nil
	}
	conPlazos := s.plazos != nil && len(pagina.Expedientes) != 0 &&
		len(pagina.FasesDesde) == len(pagina.Expedientes)
	if !conPlazos && pagina.Agregados == nil {
		return nil, nil, nil
	}
	ahora := s.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) {
		return nil, nil, nil
	}
	calculadora := s.prepararPlazos(ctx)
	var plazos []*ports.PlazoFaseRRHH
	if conPlazos {
		plazos = calcularPlazosPagina(ctx, calculadora, pagina, ahora)
	}
	if pagina.Agregados == nil {
		return plazos, nil, nil
	}
	resumen, err := resumirCuadroRRHH(ctx, calculadora, *pagina.Agregados, ahora)
	if err != nil {
		return nil, nil, err
	}
	return plazos, resumen, nil
}

// prepararPlazos devuelve la calculadora con una sola lectura de reglas para
// toda la consulta, si la calculadora lo admite; si no, o si falla, la
// original, que calcula como siempre. Nil sin calculadora.
func (s *ServicioConsultaCuadroRRHH) prepararPlazos(ctx context.Context) ports.CalculadoraPlazoFaseRRHH {
	if s == nil || s.plazos == nil {
		return nil
	}
	calculadora := s.plazos
	if preparador, admite := calculadora.(ports.PreparadorPlazosFaseRRHH); admite {
		if preparada, err := preparador.PrepararPlazosFase(ctx); err == nil && !dependenciaNula(preparada) {
			calculadora = preparada
		}
	}
	return calculadora
}

func calcularPlazosPagina(
	ctx context.Context,
	calculadora ports.CalculadoraPlazoFaseRRHH,
	pagina ports.PaginaCuadroRRHH,
	ahora time.Time,
) []*ports.PlazoFaseRRHH {
	calculados := make(map[clavePlazoFaseCuadro]*ports.PlazoFaseRRHH)
	plazos := make([]*ports.PlazoFaseRRHH, len(pagina.Expedientes))
	alguno := false
	for indice, resumen := range pagina.Expedientes {
		if ctx.Err() != nil {
			return nil
		}
		clave := clavePlazoFaseCuadro{
			fase: resumen.FaseClave, desde: pagina.FasesDesde[indice],
			urgente: len(pagina.Urgentes) == len(pagina.Expedientes) && pagina.Urgentes[indice],
		}
		plazo, visto := calculados[clave]
		if !visto {
			plazo = calcularPlazoFase(ctx, calculadora, clave, ahora)
			calculados[clave] = plazo
		}
		if plazo != nil {
			copia := *plazo
			plazos[indice] = &copia
			alguno = true
		}
	}
	if !alguno {
		return nil
	}
	return plazos
}

func calcularPlazoFase(
	ctx context.Context,
	calculadora ports.CalculadoraPlazoFaseRRHH,
	clave clavePlazoFaseCuadro,
	ahora time.Time,
) *ports.PlazoFaseRRHH {
	plazo, aplicable, err := calculadora.CalcularPlazoFase(ctx, ports.SolicitudPlazoFaseRRHH{
		Fase: clave.fase, Desde: clave.desde, Ahora: ahora, Urgente: clave.urgente,
	})
	switch {
	case err != nil:
		// El fallo se publica en la fila como «no calculado»: no es un
		// fallo de la consulta ni se sustituye por una fecha supuesta.
		return &ports.PlazoFaseRRHH{Estado: ports.PlazoFaseNoCalculado}
	case !aplicable:
		return nil
	case !plazo.Valido() || plazo.Estado == ports.PlazoFaseNoCalculado:
		return &ports.PlazoFaseRRHH{Estado: ports.PlazoFaseNoCalculado}
	}
	return &plazo
}
