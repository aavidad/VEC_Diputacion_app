package reglas

import (
	"context"
	"time"
)

// ReglasLeidas son las reglas vigentes leídas una sola vez. Sirven para
// calcular varios vencimientos dentro de una misma petición (p. ej. los plazos
// de las filas de un cuadro) sin volver a leer, clonar y resumir el catálogo
// por cada uno. Cada vencimiento es el mismo que daría Vencimiento o
// VencimientoUrgente con esa lectura. No se guarda entre peticiones: una
// lectura nueva ve el catálogo vigente en ese momento.
type ReglasLeidas struct {
	resolutor *Resolutor
	reglas    []Regla
}

// LeerReglas lee las reglas vigentes una vez, con los mismos errores que
// Reglas.
func (r *Resolutor) LeerReglas(ctx context.Context) (ReglasLeidas, error) {
	vigentes, err := r.Reglas(ctx)
	if err != nil {
		return ReglasLeidas{}, err
	}
	return ReglasLeidas{resolutor: r, reglas: vigentes}, nil
}

// Reglas devuelve una copia de las reglas leídas, en el orden del catálogo.
func (l ReglasLeidas) Reglas() []Regla {
	copia := make([]Regla, len(l.reglas))
	for indice, regla := range l.reglas {
		copia[indice] = copiarRegla(regla)
	}
	return copia
}

// Vencimiento calcula el vencimiento de la regla clave con la lectura, con
// las mismas comprobaciones y errores que Resolutor.Vencimiento (o
// VencimientoUrgente si urgente).
func (l ReglasLeidas) Vencimiento(ctx context.Context, clave string, inicio time.Time, municipioSede string, urgente bool) (Regla, Vencimiento, error) {
	if inicio.IsZero() {
		return Regla{}, Vencimiento{}, ErrReglaSinPlazo
	}
	if l.resolutor == nil {
		return Regla{}, Vencimiento{}, ErrReglasNoConfiguradas
	}
	if l.resolutor.cfg.Ajustes != nil {
		return Regla{}, Vencimiento{}, ErrAjustesNoDisponibles
	}
	if ctx == nil {
		return Regla{}, Vencimiento{}, ErrReglasNoDisponibles
	}
	for _, regla := range l.reglas {
		if regla.Clave != clave {
			continue
		}
		if regla.AjusteNoAplicable {
			return Regla{}, Vencimiento{}, ErrAjusteInvalido
		}
		return l.resolutor.calcularVencimiento(ctx, copiarRegla(regla), inicio, municipioSede, urgente)
	}
	return Regla{}, Vencimiento{}, ErrReglaNoEncontrada
}

// CalculadoraPlazosPorConsulta la implementa una calculadora que puede dar
// una copia para una sola consulta, que lee cada dato externo (por ejemplo, el
// calendario de un año) una sola vez para todos sus vencimientos.
type CalculadoraPlazosPorConsulta interface {
	CalculadoraParaConsulta() CalculadoraPlazos
}

// ParaConsulta devuelve un resolutor para una sola consulta con la
// calculadora de esa consulta. Sin calculadora que lo admita devuelve el
// mismo resolutor. Los vencimientos son los mismos; no se guarda entre
// consultas.
func (r *Resolutor) ParaConsulta() *Resolutor {
	if r == nil {
		return nil
	}
	porConsulta, admite := r.cfg.Calculadora.(CalculadoraPlazosPorConsulta)
	if !admite {
		return r
	}
	calculadora := porConsulta.CalculadoraParaConsulta()
	if nulo(calculadora) {
		return r
	}
	copia := *r
	copia.cfg.Calculadora = calculadora
	return &copia
}
