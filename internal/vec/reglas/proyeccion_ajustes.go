package reglas

import (
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// ProyectarReglasConAjustes resuelve una base y la versión efectiva que ya
// seleccionó por fecha una consulta autorizada. La cabeza de escritura puede
// tener efecto futuro y no debe pasarse como vigente antes de esa fecha.
// Esta proyección es pura y no consulta el almacén.
func ProyectarReglasConAjustes(
	catalogo domain.CatalogoConfigurable, instante time.Time, vigente *VersionAjustes,
) ([]Regla, error) {
	if instante.IsZero() {
		return nil, ErrReglasNoDisponibles
	}
	base, err := catalogo.ClonarCanonico()
	if err != nil || !catalogoVigenteEn(base, instante.UTC()) ||
		base.ID != CatalogoContratacionTemporal {
		return nil, ErrReglasNoDisponibles
	}
	huella, err := base.HuellaSHA256()
	if err != nil {
		return nil, ErrReglasNoDisponibles
	}
	if vigente != nil {
		if err := validarVersionAjustes(*vigente, CatalogoAjustesDe(base.ID), instante.UTC()); err != nil {
			return nil, err
		}
	}
	ejemplo := base.FuenteRef == MarcaPaqueteEjemplo
	proyectadas := make([]Regla, 0, len(base.Entradas))
	for _, entrada := range base.Entradas {
		if !entrada.VigenteEn(instante.UTC()) {
			continue
		}
		regla, err := reglaDesdeEntrada(base, huella, ejemplo, entrada)
		if err != nil {
			return nil, err
		}
		if vigente != nil {
			if campos, ajustada := vigente.Ajustes[regla.Clave]; ajustada {
				aplicada, err := aplicarAjuste(base, huella, ejemplo, entrada, regla, *vigente, campos)
				if err != nil {
					regla.AjusteNoAplicable = true
				} else {
					regla = aplicada
				}
			}
		}
		proyectadas = append(proyectadas, regla)
	}
	return proyectadas, nil
}
