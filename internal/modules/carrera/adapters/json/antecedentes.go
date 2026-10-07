package json

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/carrera/domain"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

type instantanea struct {
	Alcance           string   `json:"alcance"`
	CasoRef           string   `json:"caso_ref"`
	Version           string   `json:"version"`
	PersonaRef        string   `json:"persona_ref"`
	EmpleadoRef       string   `json:"empleado_ref"`
	RelacionRef       string   `json:"relacion_ref"`
	CorteEfectivo     string   `json:"corte_efectivo"`
	CorteConocimiento string   `json:"corte_conocimiento"`
	Cobertura         string   `json:"cobertura"`
	Regimen           string   `json:"regimen"`
	GrupoSubgrupo     string   `json:"grupo_subgrupo"`
	GrupoProfesional  string   `json:"grupo_profesional"`
	Fuentes           []fuente `json:"fuentes"`
	Ocupaciones       []struct {
		Referencia string  `json:"referencia"`
		Periodo    periodo `json:"periodo"`
		Nivel      *int    `json:"nivel"`
	} `json:"ocupaciones"`
	Grado *struct {
		Valor     *int      `json:"valor"`
		Evidencia evidencia `json:"evidencia"`
	} `json:"grado"`
	Servicios []struct {
		Referencia string  `json:"referencia"`
		Estado     string  `json:"estado"`
		Periodo    periodo `json:"periodo"`
	} `json:"servicios"`
}

// LectorAntecedentes contiene solo el paquete de ensayo proporcionado por la
// CLI. No adapta ni consulta Personal.
type LectorAntecedentes struct {
	datos map[string]ports.InstantaneaAntecedentesSinteticos
}

func LeerAntecedentes(r io.Reader) (LectorAntecedentes, error) {
	if r == nil {
		return LectorAntecedentes{}, ErrEntrada
	}
	b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil || len(b) > MaxBytes {
		return LectorAntecedentes{}, ErrEntrada
	}
	tokens := stdjson.NewDecoder(bytes.NewReader(b))
	if unicos(tokens, 0) != nil {
		return LectorAntecedentes{}, ErrEntrada
	}
	if _, err := tokens.Token(); err != io.EOF {
		return LectorAntecedentes{}, ErrEntrada
	}
	d := stdjson.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var entrada struct {
		Instantaneas []instantanea `json:"instantaneas"`
	}
	if d.Decode(&entrada) != nil || len(entrada.Instantaneas) == 0 || len(entrada.Instantaneas) > 64 {
		return LectorAntecedentes{}, ErrEntrada
	}
	l := LectorAntecedentes{datos: make(map[string]ports.InstantaneaAntecedentesSinteticos)}
	for _, i := range entrada.Instantaneas {
		if i.Alcance != domain.AlcanceSintetico || i.CasoRef == "" || i.Version == "" {
			return LectorAntecedentes{}, ErrEntrada
		}
		if _, existe := l.datos[i.CasoRef]; existe {
			return LectorAntecedentes{}, ErrEntrada
		}
		a := ports.InstantaneaAntecedentesSinteticos{Alcance: i.Alcance, CasoRef: i.CasoRef, Version: i.Version, PersonaRef: i.PersonaRef, EmpleadoRef: i.EmpleadoRef, RelacionRef: i.RelacionRef, CorteEfectivo: i.CorteEfectivo, CorteConocimiento: i.CorteConocimiento, Cobertura: i.Cobertura, Regimen: i.Regimen, GrupoSubgrupo: i.GrupoSubgrupo, GrupoProfesional: i.GrupoProfesional}
		for _, f := range i.Fuentes {
			a.Fuentes = append(a.Fuentes, f.domain())
		}
		for _, o := range i.Ocupaciones {
			a.Ocupaciones = append(a.Ocupaciones, ports.OcupacionAntecedente{Referencia: o.Referencia, Periodo: o.Periodo.domain(), Nivel: o.Nivel})
		}
		if i.Grado != nil {
			if i.Grado.Valor == nil {
				return LectorAntecedentes{}, ErrEntrada
			}
			a.Grado = &ports.GradoAntecedente{Valor: *i.Grado.Valor, Evidencia: i.Grado.Evidencia.domain()}
		}
		for _, s := range i.Servicios {
			a.Servicios = append(a.Servicios, ports.ServicioAntecedente{Referencia: s.Referencia, Estado: s.Estado, Periodo: s.Periodo.domain()})
		}
		l.datos[i.CasoRef] = a
	}
	return l, nil
}

func (p periodo) domain() domain.Periodo {
	return domain.Periodo{Inicio: p.Inicio, Fin: p.Fin, Evidencia: p.Evidencia.domain()}
}

func (l LectorAntecedentes) ConsultarAntecedentesSinteticos(ctx context.Context, q ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
	a, ok := l.datos[q.CasoRef]
	if ctx == nil || ctx.Err() != nil || !ok || a.Version != q.VersionEsperada {
		return ports.InstantaneaAntecedentesSinteticos{}, ErrEntrada
	}
	a.Fuentes = append([]domain.Fuente(nil), a.Fuentes...)
	a.Servicios = append([]ports.ServicioAntecedente(nil), a.Servicios...)
	a.Ocupaciones = append([]ports.OcupacionAntecedente(nil), a.Ocupaciones...)
	for i, o := range a.Ocupaciones {
		if o.Nivel != nil {
			n := *o.Nivel
			a.Ocupaciones[i].Nivel = &n
		}
	}
	if a.Grado != nil {
		grado := *a.Grado
		a.Grado = &grado
	}
	return a, nil
}
