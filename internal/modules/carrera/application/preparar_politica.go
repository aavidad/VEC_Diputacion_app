package application

import "vec-diputacion-granada/internal/modules/carrera/domain"

type PreparacionConPoliticaGrado struct {
	Preparacion domain.Preparacion
	Politica    domain.RevisionPoliticaGrado
}

// PrepararConPoliticaGradoSintetica reúne integridad del catálogo y preparación
// del expediente. No usa sus condiciones para calcular ni reconocer derechos.
func (s Servicio) PrepararConPoliticaGradoSintetica(e domain.Escenario, c domain.CatalogoPoliticaGrado) (PreparacionConPoliticaGrado, error) {
	r, err := domain.RevisarPoliticaGrado(c)
	if err != nil {
		return PreparacionConPoliticaGrado{}, err
	}
	e.Casos = append([]domain.Caso(nil), e.Casos...)
	for i := range e.Casos {
		if e.Casos[i].Via != "grado" {
			continue
		}
		e.Casos[i].Politica = r.Datos.Politica
		// El dato aportado se conserva en la revisión; no se presta como
		// aprobación a la comprobación del preparador existente.
		e.Casos[i].Politica.AprobacionReferencia = ""
		e.Casos[i].Fuentes = fuentesPreparacion(e.Casos[i].Fuentes, r.Datos.Fuentes)
	}
	p, err := s.Preparar(e)
	if err != nil {
		return PreparacionConPoliticaGrado{}, err
	}
	return PreparacionConPoliticaGrado{Preparacion: p, Politica: r}, nil
}
