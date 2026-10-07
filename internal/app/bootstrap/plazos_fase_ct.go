package bootstrap

import (
	"context"
	"errors"

	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// calculadoraPlazoFaseCT traduce el puerto de Contratación temporal al
// resolutor común de reglas. Un resolutor nulo significa «sin catálogo».
type calculadoraPlazoFaseCT struct {
	reglas *reglas.Resolutor
}

// nuevaCalculadoraPlazoFaseCT devuelve nil sin catálogo, para que el cuadro
// conserve su conducta sin plazos.
func nuevaCalculadoraPlazoFaseCT(resolutor *reglas.Resolutor) ports.CalculadoraPlazoFaseRRHH {
	if !resolutor.Disponible() {
		return nil
	}
	return calculadoraPlazoFaseCT{reglas: resolutor}
}

func (c calculadoraPlazoFaseCT) CalcularPlazoFase(
	ctx context.Context,
	solicitud ports.SolicitudPlazoFaseRRHH,
) (ports.PlazoFaseRRHH, bool, error) {
	if ctx == nil || !solicitud.Fase.Valida() || solicitud.Desde.IsZero() || solicitud.Ahora.IsZero() {
		return ports.PlazoFaseRRHH{}, false, reglas.ErrCalculoNoDisponible
	}
	if solicitud.Instantanea == nil || solicitud.Instantanea.Fase != string(solicitud.Fase) ||
		!solicitud.Instantanea.FaseDesde.Equal(solicitud.Desde) {
		return ports.PlazoFaseRRHH{}, false, reglas.ErrReglasNoDisponibles
	}
	// La versión base y los ajustes son los que guardó CT190 al abrir este
	// tramo. El resolutor actual solo aporta Calendarios y el municipio sede.
	regla, vencimiento, err := c.reglas.CalcularConInstantaneaPersistida(
		ctx, *solicitud.Instantanea, reglas.MunicipioSedeDiputacion, solicitud.Urgente,
	)
	if errors.Is(err, reglas.ErrReglaNoEncontrada) {
		return ports.PlazoFaseRRHH{}, false, nil
	}
	if err != nil {
		return ports.PlazoFaseRRHH{}, false, err
	}
	hoy, err := calendariosdomain.FechaCivilDe(solicitud.Ahora)
	if err != nil {
		return ports.PlazoFaseRRHH{}, false, reglas.ErrCalculoNoDisponible
	}
	estado := ports.PlazoFaseEnPlazo
	switch {
	case !solicitud.Ahora.Before(vencimiento.VenceAntesDe):
		estado = ports.PlazoFaseVencido
	case hoy.String() == vencimiento.UltimoDia:
		estado = ports.PlazoFaseVenceHoy
	}
	return ports.PlazoFaseRRHH{
		UltimoDia: vencimiento.UltimoDia, VenceAntesDe: vencimiento.VenceAntesDe.UTC(),
		Estado: estado, ReglaRef: regla.Referencia,
		ReglaEjemplo: regla.EsEjemplo() || regla.PaqueteEjemplo,
	}, true, nil
}

// Los contextos están en cada fila del resultado atestado. Preparar la
// calculadora no vuelve a consultar la cabeza vigente ni altera una captura.
func (c calculadoraPlazoFaseCT) PrepararPlazosFase(ctx context.Context) (ports.CalculadoraPlazoFaseRRHH, error) {
	if ctx == nil {
		return nil, reglas.ErrCalculoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c, nil
}
