package simulacion

import (
	"io"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

// DecodificarAdjudicacion reutiliza el cierre común de JSON: tamaño, forma,
// campos exactos, duplicados, profundidad y documento único.
func DecodificarAdjudicacion(r io.Reader) (ports.PeticionAdjudicacion, error) {
	var p ports.PeticionAdjudicacion
	err := Decodificar(r, &p)
	return p, err
}
