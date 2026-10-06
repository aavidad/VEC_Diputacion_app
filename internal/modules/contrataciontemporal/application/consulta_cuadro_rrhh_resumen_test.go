package application

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// calculadoraResumenPrueba da el plazo según la fase del grupo.
type calculadoraResumenPrueba struct {
	plazos   map[domain.ClaveFase]*ports.PlazoFaseRRHH
	llamadas int
}

func (c *calculadoraResumenPrueba) CalcularPlazoFase(_ context.Context, solicitud ports.SolicitudPlazoFaseRRHH) (ports.PlazoFaseRRHH, bool, error) {
	c.llamadas++
	plazo, existe := c.plazos[solicitud.Fase]
	if !existe {
		return ports.PlazoFaseRRHH{}, false, nil
	}
	return *plazo, true, nil
}

func plazoResumenPrueba(estado ports.EstadoPlazoFaseRRHH, ultimoDia string) *ports.PlazoFaseRRHH {
	if estado == ports.PlazoFaseNoCalculado {
		return &ports.PlazoFaseRRHH{Estado: estado}
	}
	dia, _ := time.Parse(time.DateOnly, ultimoDia)
	return &ports.PlazoFaseRRHH{UltimoDia: ultimoDia, VenceAntesDe: dia.Add(22 * time.Hour), Estado: estado,
		ReglaRef: "vec.contratacion_temporal.reglas:1:c03.plazo"}
}

// El resumen cuenta todo el corte: «en trámite» excluye completados y
// cancelados (como la portada), y los plazos se resuelven una vez por grupo.
func TestResumenCuadroRRHHCuentaComoLaPortada(t *testing.T) {
	// 6 de octubre de 2026, 10:00 en Madrid.
	ahora := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	desde := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	calculadora := &calculadoraResumenPrueba{plazos: map[domain.ClaveFase]*ports.PlazoFaseRRHH{
		"fiscalizacion":      plazoResumenPrueba(ports.PlazoFaseVencido, "2026-10-05"),
		"asignacion_unidad":  plazoResumenPrueba(ports.PlazoFaseVenceHoy, "2026-10-06"),
		"informe_juridico":   plazoResumenPrueba(ports.PlazoFaseEnPlazo, "2026-10-12"),
		"subsanacion_unidad": plazoResumenPrueba(ports.PlazoFaseEnPlazo, "2026-10-13"),
		"nombramiento":       plazoResumenPrueba(ports.PlazoFaseNoCalculado, ""),
	}}
	agregados := ports.AgregadosCuadroRRHH{
		Recuentos: []ports.RecuentoCuadroRRHH{
			{EstadoClave: domain.EstadoEnCurso, FaseClave: "fiscalizacion", Numero: 3},
			{EstadoClave: domain.EstadoIncidencia, FaseClave: "fiscalizacion", Numero: 1},
			{EstadoClave: domain.EstadoEnCurso, FaseClave: "solicitud", Numero: 5},
			{EstadoClave: domain.EstadoCompletado, FaseClave: "nombramiento", Numero: 7},
			{EstadoClave: domain.EstadoCancelado, FaseClave: "solicitud", Numero: 2},
		},
		GruposPlazo: []ports.GrupoPlazoCuadroRRHH{
			{FaseClave: "fiscalizacion", Desde: desde, Numero: 4},
			{FaseClave: "asignacion_unidad", Desde: desde, Numero: 2},
			{FaseClave: "informe_juridico", Desde: desde, Numero: 1},
			{FaseClave: "subsanacion_unidad", Desde: desde, Numero: 1},
			{FaseClave: "nombramiento", Desde: desde, Numero: 1},
			{FaseClave: "solicitud", Desde: desde, Numero: 5},
		},
	}
	resumen := resumirCuadroRRHH(context.Background(), calculadora, agregados, ahora)
	if resumen == nil || resumen.EnTramite != 9 || resumen.ConIncidencia != 1 ||
		resumen.PorFase["fiscalizacion"] != 4 || resumen.PorFase["solicitud"] != 5 || len(resumen.PorFase) != 2 ||
		resumen.Vencidos != 4 || resumen.VencenHoy != 2 || resumen.SinCalcular != 1 ||
		// Hoy (vence hoy) y el 12 entran en la semana; el 13 y el vencido, no.
		resumen.VencenSemana != 3 || calculadora.llamadas != len(agregados.GruposPlazo) {
		t.Fatalf("resumen = %+v, llamadas %d", resumen, calculadora.llamadas)
	}
	// Sin calculadora no se supone ningún plazo.
	sin := resumirCuadroRRHH(context.Background(), nil, agregados, ahora)
	if sin.EnTramite != 9 || sin.Vencidos != 0 || sin.VencenSemana != 0 || sin.SinCalcular != 0 {
		t.Fatalf("sin calculadora = %+v", sin)
	}
}
