package bootstrap

import (
	"context"
	"errors"
	"strings"

	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// atributoFasesReglaCT es el atributo del catálogo de reglas de Contratación
// temporal que lista, separadas por comas, las fases del expediente a las que
// la regla da plazo. Qué fases tienen plazo, y con qué regla, se cambia en el
// catálogo; aquí no hay ningún mapa fijo.
const atributoFasesReglaCT = "fases"

var errPlazoFaseAmbiguo = errors.New("bootstrap: dos reglas vigentes dan plazo a la misma fase")

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
	vigentes, err := c.reglas.Reglas(ctx)
	if errors.Is(err, reglas.ErrReglasNoConfiguradas) {
		return ports.PlazoFaseRRHH{}, false, nil
	}
	if err != nil {
		return ports.PlazoFaseRRHH{}, false, err
	}
	clave := ""
	for _, regla := range vigentes {
		if !reglaCubreFaseCT(regla, string(solicitud.Fase)) {
			continue
		}
		if clave != "" {
			return ports.PlazoFaseRRHH{}, false, errPlazoFaseAmbiguo
		}
		clave = regla.Clave
	}
	if clave == "" {
		return ports.PlazoFaseRRHH{}, false, nil
	}
	regla, vencimiento, err := c.reglas.Vencimiento(ctx, clave, solicitud.Desde, "")
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

func reglaCubreFaseCT(regla reglas.Regla, fase string) bool {
	if !regla.Unidad.EsPlazo() || regla.Computo == "" {
		return false
	}
	for _, candidata := range strings.Split(regla.Atributos[atributoFasesReglaCT], ",") {
		if strings.TrimSpace(candidata) == fase {
			return true
		}
	}
	return false
}
