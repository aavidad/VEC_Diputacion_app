package application

import (
	"errors"
	"slices"
	"strings"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const maxBolsasPreferidas = 256

// ValidarOrdenPreferencias exige una permutacion exacta del conjunto propio
// actual. Nunca acepta posiciones, puntos ni una bolsa extra del cliente.
func ValidarOrdenPreferencias(actuales, orden []string) error {
	if len(actuales) == 0 || len(actuales) > maxBolsasPreferidas || len(orden) != len(actuales) {
		return bolsa.ErrOrdenPreferenciasInvalido
	}
	propias := make(map[string]bool, len(actuales))
	for _, ref := range actuales {
		if !referenciaBolsaPreferenciaValida(ref) || propias[ref] {
			return bolsa.ErrOrdenPreferenciasInvalido
		}
		propias[ref] = true
	}
	visitadas := make(map[string]bool, len(orden))
	for _, ref := range orden {
		if !propias[ref] || visitadas[ref] {
			return bolsa.ErrOrdenPreferenciasInvalido
		}
		visitadas[ref] = true
	}
	return nil
}

func referenciaBolsaPreferenciaValida(ref string) bool {
	return len(ref) > 0 && len(ref) <= 512 && strings.TrimSpace(ref) == ref && !strings.ContainsAny(ref, "\x00\r\n")
}

// ProyectarEstadoPreferencias deja la version escrita intacta. Una bolsa nueva
// o retirada queda visible hasta que la persona confirme un orden completo.
func ProyectarEstadoPreferencias(version uint64, ordenGuardado, actuales []string) (bolsa.EstadoPreferenciasBolsas, error) {
	resultado := bolsa.EstadoPreferenciasBolsas{
		Version: version, OrdenGuardado: slices.Clone(ordenGuardado), BolsasActuales: slices.Clone(actuales),
	}
	if len(actuales) > maxBolsasPreferidas || len(ordenGuardado) > maxBolsasPreferidas ||
		version == 0 && len(ordenGuardado) != 0 || version > 0 && len(ordenGuardado) == 0 {
		return bolsa.EstadoPreferenciasBolsas{}, bolsa.ErrOrdenPreferenciasInvalido
	}
	actual := make(map[string]bool, len(actuales))
	for _, ref := range actuales {
		if !referenciaBolsaPreferenciaValida(ref) || actual[ref] {
			return bolsa.EstadoPreferenciasBolsas{}, bolsa.ErrOrdenPreferenciasInvalido
		}
		actual[ref] = true
	}
	guardado := make(map[string]bool, len(ordenGuardado))
	for _, ref := range ordenGuardado {
		if !referenciaBolsaPreferenciaValida(ref) || guardado[ref] {
			return bolsa.EstadoPreferenciasBolsas{}, bolsa.ErrOrdenPreferenciasInvalido
		}
		guardado[ref] = true
		if !actual[ref] {
			resultado.BolsasRetiradas = append(resultado.BolsasRetiradas, ref)
		}
	}
	for _, ref := range actuales {
		if !guardado[ref] {
			resultado.BolsasNuevas = append(resultado.BolsasNuevas, ref)
		}
	}
	resultado.RevisionRequerida = version == 0 || len(resultado.BolsasNuevas) != 0 || len(resultado.BolsasRetiradas) != 0
	return resultado, nil
}

// RecomendarSeleccionSimultanea solo proyecta la primera bolsa elegible de
// una preferencia vigente. No produce efectos sobre ningun llamamiento.
func RecomendarSeleccionSimultanea(estado bolsa.EstadoPreferenciasBolsas, elegibles []string) (bolsa.PreferenciaSeleccionRRHH, error) {
	proyectado, err := ProyectarEstadoPreferencias(estado.Version, estado.OrdenGuardado, estado.BolsasActuales)
	if err != nil || estado.RevisionRequerida != proyectado.RevisionRequerida || len(elegibles) > maxBolsasPreferidas {
		return bolsa.PreferenciaSeleccionRRHH{}, bolsa.ErrOrdenPreferenciasInvalido
	}
	resultado := bolsa.PreferenciaSeleccionRRHH{Version: estado.Version,
		RevisionRequerida: proyectado.RevisionRequerida, BolsasElegibles: slices.Clone(elegibles)}
	actuales := make(map[string]bool, len(estado.BolsasActuales))
	for _, ref := range estado.BolsasActuales {
		actuales[ref] = true
	}
	conjunto := make(map[string]bool, len(elegibles))
	for _, ref := range elegibles {
		if !actuales[ref] || conjunto[ref] {
			return bolsa.PreferenciaSeleccionRRHH{}, errors.Join(bolsa.ErrOrdenPreferenciasInvalido, bolsa.ErrPreferenciasDesactualizadas)
		}
		conjunto[ref] = true
	}
	if len(elegibles) < 2 {
		resultado.MotivoCodigo = "seleccion_no_simultanea"
		return resultado, nil
	}
	if proyectado.RevisionRequerida {
		resultado.MotivoCodigo = "revision_requerida"
		return resultado, nil
	}
	for _, ref := range estado.OrdenGuardado {
		if conjunto[ref] {
			resultado.BolsaPreferida = ref
			resultado.MotivoCodigo = "primera_preferencia_elegible"
			return resultado, nil
		}
	}
	return bolsa.PreferenciaSeleccionRRHH{}, bolsa.ErrOrdenPreferenciasInvalido
}
