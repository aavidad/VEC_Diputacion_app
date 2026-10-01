package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/carrera/domain"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

var ErrAntecedentes = errors.New("carrera.error.antecedentes_no_disponibles")

type AntecedentesCaso struct {
	CasoRef     string
	Instantanea ports.InstantaneaAntecedentesSinteticos
	Faltantes   []string
}

type PreparacionAntecedentes struct {
	Preparacion domain.Preparacion
	Casos       []AntecedentesCaso
}

// PrepararConAntecedentesSinteticos conecta el puerto con la preparación
// existente. No sirve datos de Personal ni habilita reconocimiento o registro.
func (s Servicio) PrepararConAntecedentesSinteticos(ctx context.Context, e domain.Escenario, lector ports.LectorAntecedentesSinteticos) (PreparacionAntecedentes, error) {
	if ctx == nil || lector == nil || e.Alcance != domain.AlcanceSintetico {
		return PreparacionAntecedentes{}, ErrAntecedentes
	}
	if _, err := s.Preparar(e); err != nil {
		return PreparacionAntecedentes{}, err
	}
	// Las colecciones del llamante siguen siendo suyas.
	e.Casos = append([]domain.Caso(nil), e.Casos...)
	out := PreparacionAntecedentes{Casos: make([]AntecedentesCaso, 0, len(e.Casos))}
	for i, c := range e.Casos {
		if ctx.Err() != nil {
			return PreparacionAntecedentes{}, ErrAntecedentes
		}
		a, err := lector.ConsultarAntecedentesSinteticos(ctx, ports.ConsultaAntecedentesSinteticos{CasoRef: c.Referencia, VersionEsperada: e.Version})
		if err != nil || ctx.Err() != nil || a.Alcance != domain.AlcanceSintetico || a.CasoRef != c.Referencia || a.Version != e.Version || !limitesInstantanea(a) {
			return PreparacionAntecedentes{}, ErrAntecedentes
		}
		faltantes := []string{"contrato_nominal_personal", "lector_autorizado_personal", "autorizacion_carrera_h08"}
		for _, campo := range []struct{ nombre, valor string }{{"persona_ref", a.PersonaRef}, {"empleado_ref", a.EmpleadoRef}, {"relacion_ref", a.RelacionRef}, {"cobertura", a.Cobertura}} {
			if strings.TrimSpace(campo.valor) == "" {
				faltantes = append(faltantes, campo.nombre)
			}
		}
		if _, err := time.Parse(time.DateOnly, a.CorteEfectivo); err != nil {
			faltantes = append(faltantes, "corte_efectivo")
		}
		if _, err := time.Parse(time.RFC3339Nano, a.CorteConocimiento); err != nil {
			faltantes = append(faltantes, "corte_conocimiento")
		}
		c.Regimen, c.GrupoSubgrupo, c.GrupoProfesional = a.Regimen, a.GrupoSubgrupo, a.GrupoProfesional
		c.Fuentes = fuentesPreparacion(c.Fuentes, a.Fuentes)
		c.NivelPuesto, c.GradoPersonal = nil, nil
		c.Periodos = nil
		c.Evidencias = append([]domain.Evidencia(nil), c.Evidencias...)
		if len(a.Ocupaciones) == 1 && strings.TrimSpace(a.Ocupaciones[0].Referencia) != "" && a.Ocupaciones[0].Nivel != nil && *a.Ocupaciones[0].Nivel >= 0 && ocupacionVigente(a.Ocupaciones[0], a.CorteEfectivo) && evidenciaAportada(a.Ocupaciones[0].Periodo.Evidencia, a.Fuentes) {
			v := *a.Ocupaciones[0].Nivel
			c.NivelPuesto = &v
		} else {
			faltantes = append(faltantes, "nivel_puesto")
		}
		if a.Grado != nil && a.Grado.Valor >= 0 && evidenciaAportada(a.Grado.Evidencia, a.Fuentes) {
			v := a.Grado.Valor
			c.GradoPersonal = &v
			c.Evidencias = append(c.Evidencias, a.Grado.Evidencia)
		} else {
			faltantes = append(faltantes, "grado_personal")
		}
		for _, servicio := range a.Servicios {
			if servicio.Estado == "reconocido" && strings.TrimSpace(servicio.Referencia) != "" && evidenciaAportada(servicio.Periodo.Evidencia, a.Fuentes) {
				c.Periodos = append(c.Periodos, servicio.Periodo)
				c.Evidencias = append(c.Evidencias, servicio.Periodo.Evidencia)
			} else {
				faltantes = append(faltantes, "servicio:"+servicio.Referencia)
			}
		}
		e.Casos[i] = c
		out.Casos = append(out.Casos, AntecedentesCaso{c.Referencia, copiarInstantanea(a), faltantes})
	}
	p, err := s.Preparar(e)
	if err != nil {
		return PreparacionAntecedentes{}, err
	}
	out.Preparacion = p
	return out, nil
}

func evidenciaAportada(e domain.Evidencia, fs []domain.Fuente) bool {
	if strings.TrimSpace(e.Referencia) == "" || strings.TrimSpace(e.Fuente) == "" || strings.TrimSpace(e.Version) == "" {
		return false
	}
	for _, f := range fs {
		if f.Referencia == e.Fuente && f.Version == e.Version {
			return true
		}
	}
	return false
}

func fuentesPreparacion(propias, personales []domain.Fuente) []domain.Fuente {
	out := append([]domain.Fuente(nil), propias...)
	for _, f := range personales {
		encontrada := false
		for _, anterior := range out {
			if anterior == f {
				encontrada = true
				break
			}
		}
		if !encontrada {
			out = append(out, f)
		}
	}
	return out
}

func limitesInstantanea(a ports.InstantaneaAntecedentesSinteticos) bool {
	if len(a.Fuentes) > 32 || len(a.Ocupaciones) > 128 || len(a.Servicios) > 128 {
		return false
	}
	for _, texto := range []string{a.CasoRef, a.Version, a.PersonaRef, a.EmpleadoRef, a.RelacionRef, a.CorteEfectivo, a.CorteConocimiento, a.Cobertura, a.Regimen, a.GrupoSubgrupo, a.GrupoProfesional} {
		if len(texto) > 1024 {
			return false
		}
	}
	for _, f := range a.Fuentes {
		if len(f.Referencia) > 1024 || len(f.Version) > 1024 {
			return false
		}
	}
	if a.Grado != nil && !limitesEvidencia(a.Grado.Evidencia) {
		return false
	}
	for _, o := range a.Ocupaciones {
		if len(o.Referencia) > 1024 || !limitesPeriodo(o.Periodo) {
			return false
		}
	}
	for _, s := range a.Servicios {
		if len(s.Referencia) > 1024 || !limitesPeriodo(s.Periodo) || (s.Estado != "declarado" && s.Estado != "comprobado" && s.Estado != "reconocido") {
			return false
		}
	}
	return true
}

func limitesEvidencia(e domain.Evidencia) bool {
	return len(e.Referencia) <= 1024 && len(e.Fuente) <= 1024 && len(e.Version) <= 1024
}

func limitesPeriodo(p domain.Periodo) bool {
	return len(p.Inicio) <= 1024 && len(p.Fin) <= 1024 && limitesEvidencia(p.Evidencia)
}

func ocupacionVigente(o ports.OcupacionAntecedente, corte string) bool {
	dia, err := time.Parse(time.DateOnly, corte)
	ini, errIni := time.Parse(time.DateOnly, o.Periodo.Inicio)
	vigente := err == nil && errIni == nil && !dia.Before(ini)
	if !vigente || o.Periodo.Fin == "" {
		return vigente
	}
	fin, errFin := time.Parse(time.DateOnly, o.Periodo.Fin)
	return errFin == nil && !fin.Before(ini) && !dia.After(fin)
}

func copiarInstantanea(a ports.InstantaneaAntecedentesSinteticos) ports.InstantaneaAntecedentesSinteticos {
	a.Fuentes = append([]domain.Fuente(nil), a.Fuentes...)
	a.Servicios = append([]ports.ServicioAntecedente(nil), a.Servicios...)
	a.Ocupaciones = append([]ports.OcupacionAntecedente(nil), a.Ocupaciones...)
	for i, o := range a.Ocupaciones {
		if o.Nivel != nil {
			v := *o.Nivel
			a.Ocupaciones[i].Nivel = &v
		}
	}
	if a.Grado != nil {
		grado := *a.Grado
		a.Grado = &grado
	}
	return a
}
