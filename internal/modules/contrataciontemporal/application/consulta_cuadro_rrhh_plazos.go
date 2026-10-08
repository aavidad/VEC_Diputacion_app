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
	captura string
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
// toda la consulta, si la calculadora lo admite. Si esa lectura falla, los
// plazos quedan sin calcular: volver a la calculadora original repetiría la
// misma lectura por cada fila y por cada grupo del resumen.
func (s *ServicioConsultaCuadroRRHH) prepararPlazos(ctx context.Context) ports.CalculadoraPlazoFaseRRHH {
	if s == nil || s.plazos == nil {
		return nil
	}
	calculadora := s.plazos
	if preparador, admite := calculadora.(ports.PreparadorPlazosFaseRRHH); admite {
		preparada, err := preparador.PrepararPlazosFase(ctx)
		if err != nil {
			return calculadoraPlazosNoDisponibles{causa: err}
		}
		if dependenciaNula(preparada) {
			return calculadoraPlazosNoDisponibles{causa: ErrConsultaRRHHNoDisponible}
		}
		calculadora = preparada
	}
	return calculadora
}

// La indisponibilidad se representa por grupo o fila, sin otra lectura de
// catálogo. El resumen puede contar esos grupos como «sin calcular».
type calculadoraPlazosNoDisponibles struct{ causa error }

func (c calculadoraPlazosNoDisponibles) CalcularPlazoFase(
	context.Context, ports.SolicitudPlazoFaseRRHH,
) (ports.PlazoFaseRRHH, bool, error) {
	return ports.PlazoFaseRRHH{}, false, c.causa
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
		var captura *ports.CapturaPlazoFaseRRHH
		if len(pagina.CapturasPlazo) == len(pagina.Expedientes) {
			captura = &pagina.CapturasPlazo[indice]
		}
		clave := clavePlazoFaseCuadro{
			fase: resumen.FaseClave, desde: pagina.FasesDesde[indice],
			urgente: len(pagina.Urgentes) == len(pagina.Expedientes) && pagina.Urgentes[indice],
		}
		if captura != nil {
			clave.captura = captura.Estado + ":" + captura.BaseHuella + ":" + captura.AjustesHuella + ":" + captura.CapturadaEn.Format(time.RFC3339Nano)
		}
		plazo, visto := calculados[clave]
		if !visto {
			plazo = calcularPlazoFase(ctx, calculadora, clave, ahora, captura)
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
	captura *ports.CapturaPlazoFaseRRHH,
) *ports.PlazoFaseRRHH {
	if captura != nil && captura.Estado == "legado_sin_instantanea" {
		return &ports.PlazoFaseRRHH{Estado: ports.PlazoFaseNoCalculado}
	}
	if calculadora == nil {
		return &ports.PlazoFaseRRHH{Estado: ports.PlazoFaseNoCalculado}
	}
	solicitud := ports.SolicitudPlazoFaseRRHH{
		Fase: clave.fase, Desde: clave.desde, Ahora: ahora, Urgente: clave.urgente,
	}
	var plazo ports.PlazoFaseRRHH
	var aplicable bool
	var err error
	if captura != nil {
		conCaptura, ok := calculadora.(ports.CalculadoraConCapturaPlazoFaseRRHH)
		if !ok || (captura.Estado != "capturada" && captura.Estado != "legado_base_transicion") {
			return &ports.PlazoFaseRRHH{Estado: ports.PlazoFaseNoCalculado}
		}
		plazo, aplicable, err = conCaptura.CalcularPlazoConCaptura(ctx, solicitud, *captura)
	} else {
		plazo, aplicable, err = calculadora.CalcularPlazoFase(ctx, solicitud)
	}
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
