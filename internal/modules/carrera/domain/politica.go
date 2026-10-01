package domain

import (
	"errors"
	"time"
)

var ErrPoliticaGrado = errors.New("carrera.error.politica_grado_invalida")

// Cada elemento identifica una condición declarada y su procedencia. No
// expresa fórmulas ni permite decidir qué periodos personales son computables.
type CatalogoPoliticaGrado struct {
	Alcance                             string
	Politica                            Politica
	Procedencia                         Evidencia
	Fuentes                             []Fuente
	Vigencia                            Periodo
	Vias, Periodos, Limites, Evidencias []Evidencia
}

type RevisionPoliticaGrado struct {
	Datos          CatalogoPoliticaGrado
	Comprobaciones []Comprobacion
	Pendientes     []string
}

// RevisarPoliticaGrado coteja integridad de metadatos sintéticos. Una
// referencia de aprobación aportada nunca acredita aprobación competente.
func RevisarPoliticaGrado(c CatalogoPoliticaGrado) (RevisionPoliticaGrado, error) {
	colecciones := [][]Evidencia{c.Vias, c.Periodos, c.Limites, c.Evidencias}
	var elementos []Evidencia
	for _, grupo := range colecciones {
		if len(grupo) > 32 {
			return RevisionPoliticaGrado{}, ErrPoliticaGrado
		}
		elementos = append(elementos, grupo...)
	}
	if c.Alcance != AlcanceSintetico || !limites(Caso{Politica: c.Politica, Fuentes: c.Fuentes, Periodos: []Periodo{c.Vigencia}, Evidencias: elementos, Convenio: c.Procedencia}) {
		return RevisionPoliticaGrado{}, ErrPoliticaGrado
	}
	r := RevisionPoliticaGrado{Datos: copiarCatalogoPolitica(c), Comprobaciones: []Comprobacion{}, Pendientes: []string{}}
	add := func(campo, motivo string, disponible bool) {
		estado := "pendiente"
		if disponible {
			estado, motivo = "disponible", "disponible"
		} else {
			r.Pendientes = append(r.Pendientes, "carrera.politica.pendiente."+campo)
		}
		r.Comprobaciones = append(r.Comprobaciones, Comprobacion{"carrera.politica.comprobacion." + campo, estado, "carrera.politica.motivo." + motivo})
	}
	add("identificacion", "dato_ausente", texto(c.Politica.Referencia) && texto(c.Politica.Version))
	fs := len(c.Fuentes) > 0
	vistas := make(map[string]bool)
	for _, f := range c.Fuentes {
		fs = fs && fuenteCompleta(f) && !vistas[f.Referencia]
		vistas[f.Referencia] = true
	}
	add("fuentes", "fuente_incompleta_o_repetida", fs)
	add("procedencia", "fuente_version_ausente", c.Procedencia.Referencia == c.Politica.Referencia && c.Procedencia.Fuente == c.Politica.Fuente && evidenciaCompleta(c.Procedencia, c.Fuentes))
	regimenes := len(c.Politica.Regimenes) > 0
	vistos := make(map[string]bool)
	for _, regimen := range c.Politica.Regimenes {
		regimenes = regimenes && texto(regimen) && !vistos[regimen]
		vistos[regimen] = true
	}
	add("regimenes", "dato_ausente_o_repetido", regimenes)
	motivoVigencia := integridadVigenciaPolitica(c.Vigencia, c.Fuentes)
	add("vigencia", motivoVigencia, motivoVigencia == "disponible")
	for i, nombre := range []string{"vias", "periodos", "limites", "evidencias"} {
		add(nombre, "elemento_ausente_incompleto_o_repetido", integridadElementosPolitica(colecciones[i], c.Fuentes))
	}
	add("fuente_admitida", "fuente_no_acreditada", false)
	add("aprobacion_competente", "aprobacion_no_acreditada", false)
	add("circuito_nominal", "circuito_no_disponible", false)
	return r, nil
}

func integridadVigenciaPolitica(p Periodo, fs []Fuente) string {
	ini, err := time.Parse(time.DateOnly, p.Inicio)
	inicioValido := err == nil && ini.Year() >= 1
	if !inicioValido {
		return "fecha_invalida"
	}
	if p.Fin != "" {
		fin, err := time.Parse(time.DateOnly, p.Fin)
		finValido := err == nil && fin.Year() >= 1 && !fin.Before(ini)
		if !finValido {
			return "fecha_invalida"
		}
	}
	if !evidenciaCompleta(p.Evidencia, fs) {
		return "evidencia_incompleta"
	}
	return "disponible"
}

func integridadElementosPolitica(es []Evidencia, fs []Fuente) bool {
	integros := len(es) > 0
	vistos := make(map[string]bool)
	for _, e := range es {
		integros = integros && evidenciaCompleta(e, fs) && !vistos[e.Referencia]
		vistos[e.Referencia] = true
	}
	return integros
}

func copiarCatalogoPolitica(c CatalogoPoliticaGrado) CatalogoPoliticaGrado {
	c.Politica.Regimenes = append([]string{}, c.Politica.Regimenes...)
	c.Fuentes = append([]Fuente{}, c.Fuentes...)
	c.Vias = append([]Evidencia{}, c.Vias...)
	c.Periodos = append([]Evidencia{}, c.Periodos...)
	c.Limites = append([]Evidencia{}, c.Limites...)
	c.Evidencias = append([]Evidencia{}, c.Evidencias...)
	return c
}
