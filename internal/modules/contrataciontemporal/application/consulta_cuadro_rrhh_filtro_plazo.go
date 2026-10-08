package application

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ResultadoFiltroPlazoCuadroRRHH struct {
	Referencias  []string
	Vencidos     uint64
	VencenHoy    uint64
	VencenSemana uint64
	SinCalcular  uint64
}

func (ResultadoFiltroPlazoCuadroRRHH) String() string {
	return "[resultado-filtro-plazo-rrhh-redactado]"
}
func (r ResultadoFiltroPlazoCuadroRRHH) GoString() string { return r.String() }
func (r ResultadoFiltroPlazoCuadroRRHH) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ResultadoFiltroPlazoCuadroRRHH) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (ResultadoFiltroPlazoCuadroRRHH) MarshalJSON() ([]byte, error) {
	return nil, ports.ErrMaterialConsultaRRHHSensible
}

// SeleccionarPlazosCuadroRRHH recorre el corte completo, con una calculadora
// preparada y un instante UTC comunes. El llamador obtiene los contextos
// tras autorizar y aplica Referencias a COUNT y página en la misma TX.
func SeleccionarPlazosCuadroRRHH(
	ctx context.Context,
	calculadora ports.CalculadoraPlazoFaseRRHH,
	ahora time.Time,
	filtro ports.FiltroPlazoCuadroRRHH,
	contextos []ports.ContextoPlazoCuadroRRHH,
) (ResultadoFiltroPlazoCuadroRRHH, error) {
	var resultado ResultadoFiltroPlazoCuadroRRHH
	if ctx == nil || calculadora == nil || !domain.InstanteUTCCanonico(ahora) ||
		len(contextos) > ports.MaximoContextosPlazoCuadroRRHH ||
		(filtro != "" && filtro != ports.FiltroPlazoVencido &&
			filtro != ports.FiltroPlazoVenceHoy && filtro != ports.FiltroPlazoVenceSemana) {
		return resultado, ErrSolicitudConsultaRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return resultado, err
	}
	hoy, err := diaCivilMadrid(ahora)
	if err != nil {
		return resultado, err
	}
	calculados := make(map[clavePlazoFaseCuadro]clasificacionPlazoCuadroRRHH)
	var anterior string
	for _, contexto := range contextos {
		if err := ctx.Err(); err != nil {
			return ResultadoFiltroPlazoCuadroRRHH{}, err
		}
		if !domain.ReferenciaOpacaValida(contexto.ExpedienteRef) ||
			(anterior != "" && contexto.ExpedienteRef <= anterior) ||
			contexto.VersionExpediente < 1 ||
			contexto.VersionExpediente > 9_007_199_254_740_991 ||
			!contexto.FaseClave.Valida() ||
			!domain.InstanteUTCCanonico(contexto.FaseDesde) ||
			contexto.FaseDesde.After(ahora) {
			return ResultadoFiltroPlazoCuadroRRHH{}, ErrResultadoConsultaRRHHNoConfiable
		}
		anterior = contexto.ExpedienteRef
		clave := clavePlazoFaseCuadro{
			fase: contexto.FaseClave, desde: contexto.FaseDesde, urgente: contexto.Urgente,
		}
		clase, vista := calculados[clave]
		if !vista {
			plazo := calcularPlazoFase(ctx, calculadora, clave, ahora)
			if err := ctx.Err(); err != nil {
				return ResultadoFiltroPlazoCuadroRRHH{}, err
			}
			clase, err = clasificarPlazoCuadroRRHH(plazo, hoy)
			if err != nil {
				return ResultadoFiltroPlazoCuadroRRHH{}, err
			}
			calculados[clave] = clase
		}
		if clase.vencido {
			resultado.Vencidos++
		}
		if clase.venceHoy {
			resultado.VencenHoy++
		}
		if clase.venceSemana {
			resultado.VencenSemana++
		}
		if clase.sinCalcular {
			resultado.SinCalcular++
		}
		if (filtro == ports.FiltroPlazoVencido && clase.vencido) ||
			(filtro == ports.FiltroPlazoVenceHoy && clase.venceHoy) ||
			(filtro == ports.FiltroPlazoVenceSemana && clase.venceSemana) {
			resultado.Referencias = append(resultado.Referencias, contexto.ExpedienteRef)
		}
	}
	return resultado, nil
}
