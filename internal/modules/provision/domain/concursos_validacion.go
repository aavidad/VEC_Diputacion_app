package domain

import (
	"fmt"
	"sort"
	"strings"
)

// Error contiene códigos de catálogo, nunca datos ni texto administrativo.
type Error struct {
	Codigo string `json:"codigo"`
	Campo  string `json:"campo"`
}

func (e *Error) Error() string         { return e.Codigo + ":" + e.Campo }
func fallo(codigo, campo string) error { return &Error{codigo, campo} }
func referencia(s string) bool         { return len(s) > 0 && len(s) <= 160 && strings.TrimSpace(s) == s }
func familiaValida(f Familia) bool {
	switch f {
	case Grado, Antiguedad, Permanencia, Cursos, Titulaciones, ValoracionTrabajo:
		return true
	}
	return false
}
func temporal(f Familia) bool { return f == Antiguedad || f == Permanencia || f == ValoracionTrabajo }

func ValidarConfiguracion(c Configuracion) error {
	if c.SchemaVersion != VersionMotor || !referencia(c.ConvocatoriaRef) || !referencia(c.Version) || !referencia(c.BasesRef) {
		return fallo("configuracion_invalida", "version_bases")
	}
	orden, err := c.VentanaDesde.Comparar(c.FechaCorte)
	if err != nil || orden >= 0 {
		return fallo("configuracion_invalida", "ventana")
	}
	if !c.MaximoTotal.EsValido() || len(c.Reglas) == 0 || len(c.Reglas) > 100 {
		return fallo("configuracion_invalida", "reglas")
	}
	ids := map[string]bool{}
	for i, r := range c.Reglas {
		campo := fmt.Sprintf("reglas.%d", i)
		if !referencia(r.ID) || ids[r.ID] || !familiaValida(r.Familia) || !referencia(r.ReferenciaBase) || !r.Redondeo.EsValido() || !r.Coeficiente.EsValido() || !r.Maximo.EsValido() {
			return fallo("regla_invalida", campo)
		}
		ids[r.ID] = true
		if r.MaximoElementos < 0 || r.MaximoElementos > 10000 {
			return fallo("regla_invalida", campo+".maximo_elementos")
		}
		if r.MaximoElementos > 0 {
			if (r.Familia == Cursos && r.SeleccionElementos != "mayor_unidades") || (r.Familia == Titulaciones && r.SeleccionElementos != "mismo_coeficiente") || (r.Familia != Cursos && r.Familia != Titulaciones) {
				return fallo("seleccion_elementos_invalida", campo)
			}
		} else if r.SeleccionElementos != "" {
			return fallo("regla_incompatible", campo)
		}
		if r.Familia == Grado && len(r.Tipos) > 0 {
			return fallo("regla_incompatible", campo)
		}
		tipos := map[string]bool{}
		for _, t := range r.Tipos {
			if !referencia(t) || tipos[t] {
				return fallo("catalogo_ambiguo", campo+".tipos")
			}
			tipos[t] = true
		}
		if len(r.Tipos) > 100 {
			return fallo("regla_invalida", campo+".tipos")
		}
		if temporal(r.Familia) {
			if r.Conversion == nil || (r.Agrupacion != "por_tramo" && r.Agrupacion != "por_periodo") || r.Solapes != "rechazar" {
				return fallo("politica_temporal_invalida", campo)
			}
			if r.Jornada != "integra" && r.Jornada != "proporcional" && r.Jornada != "protegida_integra" {
				return fallo("politica_jornada_invalida", campo)
			}
			v := r.Conversion
			if (v.Metodo == "meses_completos" || v.Metodo == "anos_desde_meses") && r.Agrupacion != "por_periodo" {
				return fallo("agrupacion_calendario_no_soportada", campo)
			}
			if v.Divisor <= 0 || v.Divisor > 1000000 || v.UmbralResto < 0 || v.UmbralResto >= v.Divisor {
				return fallo("conversion_invalida", campo)
			}
			switch v.Metodo {
			case "dias_racionales", "dias_completos", "meses_completos":
				if v.UmbralResto != 0 {
					return fallo("conversion_invalida", campo)
				}
			case "anos_desde_meses":
			default:
				return fallo("conversion_invalida", campo)
			}
		} else if r.Conversion != nil || r.Jornada != "" || r.Solapes != "" || r.Agrupacion != "" {
			return fallo("regla_incompatible", campo)
		}
		if r.Familia == Cursos {
			if r.HorasMinimas == nil || !r.HorasMinimas.EsValido() || r.HorasMinimas.Numerador() < 0 {
				return fallo("horas_minimas_invalidas", campo)
			}
		} else if r.HorasMinimas != nil {
			return fallo("regla_incompatible", campo)
		}
		if r.ExcluirRequisito && r.Familia != Titulaciones {
			return fallo("regla_incompatible", campo)
		}
		if r.Familia == Grado || r.Familia == ValoracionTrabajo {
			if err := validarTabla(r, campo); err != nil {
				return err
			}
		} else if len(r.Tramos) != 0 || r.Diferencia != "" {
			return fallo("regla_incompatible", campo)
		}
		for j := 0; j < i; j++ {
			a := c.Reglas[j]
			if a.Familia == r.Familia && filtrosSolapan(a.Tipos, r.Tipos) {
				return fallo("reglas_solapadas", campo)
			}
		}
	}
	return nil
}
func validarTabla(r Regla, campo string) error {
	if r.Diferencia != "puesto_menos_hecho" && r.Diferencia != "hecho_menos_puesto" {
		return fallo("diferencia_invalida", campo)
	}
	if r.MinDiferencia > r.MaxDiferencia || r.MinDiferencia < -1000 || r.MaxDiferencia > 1000 || len(r.Tramos) == 0 || len(r.Tramos) > 100 {
		return fallo("tabla_invalida", campo)
	}
	tramos := append([]Tramo(nil), r.Tramos...)
	sort.Slice(tramos, func(i, j int) bool { return tramos[i].MinDiferencia < tramos[j].MinDiferencia })
	inicio := r.MinDiferencia
	ids := map[string]bool{}
	for _, t := range tramos {
		if !referencia(t.ID) || ids[t.ID] || t.MinDiferencia != inicio || t.MaxDiferencia < t.MinDiferencia || t.MaxDiferencia > r.MaxDiferencia || !t.Coeficiente.EsValido() || !t.Maximo.EsValido() {
			return fallo("tabla_ambigua", campo)
		}
		ids[t.ID] = true
		inicio = t.MaxDiferencia + 1
	}
	if inicio != r.MaxDiferencia+1 {
		return fallo("tabla_incompleta", campo)
	}
	return nil
}
func filtrosSolapan(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return true
	}
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}
func acepta(tipo string, tipos []string) bool {
	if len(tipos) == 0 {
		return true
	}
	for _, t := range tipos {
		if t == tipo {
			return true
		}
	}
	return false
}
func contiene(familias []Familia, f Familia) bool {
	for _, v := range familias {
		if v == f {
			return true
		}
	}
	return false
}

func ValidarEntrada(e Entrada) error {
	if !referencia(e.InstantaneaRef) || !referencia(e.PuestoRef) || e.NivelPuesto <= 0 || e.NivelPuesto > 1000 {
		return fallo("entrada_invalida", "puesto")
	}
	if len(e.Periodos) > 10000 || len(e.Cursos) > 10000 || len(e.Titulaciones) > 10000 {
		return fallo("entrada_excesiva", "meritos")
	}
	vistas := map[Familia]bool{}
	for _, f := range e.Disponibles {
		if !familiaValida(f) || vistas[f] {
			return fallo("entrada_ambigua", "disponibles")
		}
		vistas[f] = true
	}
	if e.GradoPersonal != nil && (*e.GradoPersonal <= 0 || *e.GradoPersonal > 1000 || !referencia(e.GradoEvidenciaRef) || !vistas[Grado]) {
		return fallo("entrada_invalida", "grado")
	}
	if vistas[Cursos] && e.Cursos == nil {
		return fallo("fuente_sin_instantanea", "cursos")
	}
	if vistas[Titulaciones] && e.Titulaciones == nil {
		return fallo("fuente_sin_instantanea", "titulaciones")
	}
	if (vistas[Antiguedad] || vistas[Permanencia] || vistas[ValoracionTrabajo]) && e.Periodos == nil {
		return fallo("fuente_sin_instantanea", "periodos")
	}
	ids := map[string]bool{}
	for _, p := range e.Periodos {
		if !referencia(p.ID) || ids[p.ID] || !referencia(p.EvidenciaRef) || !p.Desde.EsValida() || !p.Jornada.EsValida() || !referencia(p.Tipo) || len(p.Familias) == 0 {
			return fallo("periodo_invalido", "periodos")
		}
		ids[p.ID] = true
		if p.AtestacionProtegidaRef != "" && !referencia(p.AtestacionProtegidaRef) {
			return fallo("atestacion_invalida", "periodos")
		}
		if p.Hasta != nil {
			cmp, err := p.Desde.Comparar(*p.Hasta)
			if err != nil || cmp >= 0 {
				return fallo("periodo_invalido", "hasta")
			}
		}
		fs := map[Familia]bool{}
		for _, f := range p.Familias {
			if !temporal(f) || fs[f] || !vistas[f] {
				return fallo("periodo_invalido", "familias")
			}
			fs[f] = true
		}
		if contiene(p.Familias, ValoracionTrabajo) && (p.Nivel <= 0 || p.Nivel > 1000) {
			return fallo("nivel_ausente", "periodos")
		}
	}
	return validarMeritos(e, vistas)
}
