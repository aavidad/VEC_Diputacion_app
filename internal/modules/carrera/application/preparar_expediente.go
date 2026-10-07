package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/carrera/domain"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

type RevisionExpedienteCaso struct {
	Declaracion     domain.PreparacionCaso
	Antecedentes    AntecedentesCaso
	Contradicciones []string
}

type PreparacionExpediente struct {
	Preparacion domain.Preparacion
	Politica    domain.RevisionPoliticaGrado
	Casos       []RevisionExpedienteCaso
}

// PrepararExpedienteSintetico reúne H05 y H06 sin prestar una aprobación ni
// consumir datos personales. Conserva la declaración y la instantánea separadas
// para revisar diferencias; no decide derechos ni suma periodos.
func (s Servicio) PrepararExpedienteSintetico(ctx context.Context, e domain.Escenario, lector ports.LectorAntecedentesSinteticos, c domain.CatalogoPoliticaGrado) (PreparacionExpediente, error) {
	declaracion, err := s.Preparar(e)
	if err != nil {
		return PreparacionExpediente{}, err
	}
	politica, err := s.PrepararConPoliticaGradoSintetica(e, c)
	if err != nil {
		return PreparacionExpediente{}, err
	}
	e.Casos = append([]domain.Caso(nil), e.Casos...)
	for i := range e.Casos {
		if e.Casos[i].Via == "grado" {
			e.Casos[i].Politica = politica.Preparacion.Casos[i].Antecedentes.Politica
			e.Casos[i].Fuentes = politica.Preparacion.Casos[i].Fuentes
		}
	}
	a, err := s.PrepararConAntecedentesSinteticos(ctx, e, lector)
	if err != nil {
		return PreparacionExpediente{}, err
	}
	out := PreparacionExpediente{Preparacion: a.Preparacion, Politica: politica.Politica, Casos: make([]RevisionExpedienteCaso, 0, len(e.Casos))}
	for i, preparado := range a.Preparacion.Casos {
		out.Casos = append(out.Casos, RevisionExpedienteCaso{
			Declaracion: declaracion.Casos[i], Antecedentes: a.Casos[i],
			Contradicciones: contradiccionesExpediente(declaracion.Casos[i], preparado),
		})
	}
	return out, nil
}

func contradiccionesExpediente(d, a domain.PreparacionCaso) []string {
	out := []string{}
	for _, campo := range []struct{ clave, declarado, antecedente string }{
		{"regimen", d.Regimen, a.Regimen},
		{"grupo_subgrupo", d.Antecedentes.GrupoSubgrupo, a.Antecedentes.GrupoSubgrupo},
		{"grupo_profesional", d.Antecedentes.GrupoProfesional, a.Antecedentes.GrupoProfesional},
	} {
		if campo.declarado != "" && campo.antecedente != "" && campo.declarado != campo.antecedente {
			out = append(out, campo.clave)
		}
	}
	for _, campo := range []struct {
		clave                  string
		declarado, antecedente *int
	}{
		{"nivel_puesto", d.NivelPuesto, a.NivelPuesto},
		{"grado_personal", d.GradoPersonal, a.GradoPersonal},
	} {
		if campo.declarado != nil && campo.antecedente != nil && *campo.declarado != *campo.antecedente {
			out = append(out, campo.clave)
		}
	}
	return out
}
