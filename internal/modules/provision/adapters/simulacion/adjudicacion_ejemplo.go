package simulacion

import (
	"bytes"
	_ "embed"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

//go:embed adjudicacion_ejemplo.sintetico.json
var ejemploAdjudicacionJSON []byte

// EjemploAdjudicacion devuelve una copia nueva del ejercicio público. El
// llamante puede adaptar su política sintética sin modificar otros ensayos.
func EjemploAdjudicacion() (ports.PeticionAdjudicacion, error) {
	p, err := DecodificarAdjudicacion(bytes.NewReader(ejemploAdjudicacionJSON))
	if err != nil {
		return p, err
	}
	err = domain.ValidarAdjudicacion(p.Configuracion, p.Entrada)
	return p, err
}
