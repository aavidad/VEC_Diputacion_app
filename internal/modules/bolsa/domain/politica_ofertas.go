package domain

import (
	"errors"
	"regexp"
)

// PoliticaOfertas es deliberadamente de ejemplo mientras RRHH no ratifique
// plazo, inicio y efectos (dudas 1-3, 14, 33 y 43). La edición crea una nueva
// versión; las ofertas anteriores conservan la versión que las gobernó.
type PoliticaOfertas struct {
	Plazo        PlazoPoliticaOfertas        `json:"plazo"`
	Adjudicacion AdjudicacionPoliticaOfertas `json:"adjudicacion"`
	NoCubierta   NoCubiertaPoliticaOfertas   `json:"no_cubierta"`
}

type PlazoPoliticaOfertas struct {
	Unidad        string `json:"unidad"`
	Cantidad      int    `json:"cantidad"`
	Computo       string `json:"computo"`
	MunicipioSede string `json:"municipio_sede"`
}

type AdjudicacionPoliticaOfertas struct {
	Criterio     string `json:"criterio"`
	Elegibilidad string `json:"elegibilidad"`
	// Ausente en versiones anteriores: el valor seguro es exigir otra persona.
	RequiereSegundaValidacion *bool `json:"requiere_segunda_validacion,omitempty"`
}

func (p AdjudicacionPoliticaOfertas) SegundaValidacionRequerida() bool {
	return p.RequiereSegundaValidacion == nil || *p.RequiereSegundaValidacion
}

type NoCubiertaPoliticaOfertas struct {
	Accion    string `json:"accion"`
	Condicion string `json:"condicion"`
}

var (
	ErrPoliticaOfertasInvalida = errors.New("bolsa: politica de ofertas invalida")
	municipioINE               = regexp.MustCompile(`^[0-9]{5}$`)
)

// Validar limita el catálogo a reglas que ejecuta el evaluador de Bolsa.
// No acepta una etiqueta libre que pudiera prometer un efecto no implementado.
func (p PoliticaOfertas) Validar() error {
	unidad, cantidad, computo := p.Plazo.Unidad, p.Plazo.Cantidad, p.Plazo.Computo
	plazoValido := ((unidad == "dias_habiles" || unidad == "dias_naturales") &&
		cantidad >= 1 && cantidad <= 30 && computo == "administrativo") ||
		(unidad == "horas_naturales" && cantidad >= 1 && cantidad <= 720 && computo == "continuo_utc")
	if !plazoValido || !municipioINE.MatchString(p.Plazo.MunicipioSede) ||
		p.Adjudicacion.Criterio != "orden_vigente" || p.Adjudicacion.Elegibilidad != "disposicion_en_plazo" ||
		!p.Adjudicacion.SegundaValidacionRequerida() ||
		p.NoCubierta.Accion != "llamamiento_directo" || p.NoCubierta.Condicion != "sin_disposiciones_elegibles" {
		return ErrPoliticaOfertasInvalida
	}
	return nil
}
