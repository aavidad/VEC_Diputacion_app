package bootstrap

import "vec-diputacion-granada/internal/modules/bolsa/ports"

// mapearSituacionesConjuntoBolsa ata el conjunto B82 a la instantánea y las
// entradas que el lector RRHH va a presentar. Una fila ausente, duplicada o
// de otra versión hace fallar la lectura completa.
func mapearSituacionesConjuntoBolsa(vigente ports.ConstitucionVigente, entradas []ports.EntradaConstitucion, filas []ports.SituacionResumenParticipacion) (map[string]ports.SituacionParticipacion, map[string]ports.EstadoCese, error) {
	if len(filas) != len(entradas) {
		return nil, nil, ErrComposicionDesarrolloIncompleta
	}
	esperadas := make(map[string]ports.EntradaConstitucion, len(entradas))
	for _, entrada := range entradas {
		if entrada.ParticipacionRef == "" || entrada.Orden == 0 {
			return nil, nil, ErrComposicionDesarrolloIncompleta
		}
		if _, repetida := esperadas[entrada.ParticipacionRef]; repetida {
			return nil, nil, ErrComposicionDesarrolloIncompleta
		}
		esperadas[entrada.ParticipacionRef] = entrada
	}
	situaciones := make(map[string]ports.SituacionParticipacion, len(filas))
	ceses := make(map[string]ports.EstadoCese)
	for _, fila := range filas {
		entrada, existe := esperadas[fila.ParticipacionRef]
		if !existe || fila.BolsaRef != vigente.Bolsa.BolsaRef || fila.CategoriaRef != vigente.CategoriaRef ||
			!fila.ConfirmadaEn.Equal(vigente.ConfirmadaEn) || fila.InstantaneaRef != vigente.Instantanea.InstantaneaRef ||
			fila.VersionInstantanea != vigente.Instantanea.Version || fila.Orden != entrada.Orden || fila.Situacion == nil {
			return nil, nil, ErrComposicionDesarrolloIncompleta
		}
		if _, repetida := situaciones[fila.ParticipacionRef]; repetida {
			return nil, nil, ErrComposicionDesarrolloIncompleta
		}
		situaciones[fila.ParticipacionRef] = *fila.Situacion
		if fila.Cese != nil {
			ceses[fila.ParticipacionRef] = *fila.Cese
		}
	}
	return situaciones, ceses, nil
}
