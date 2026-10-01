package simulacion

import (
	"io"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

func DecodificarCiclo(r io.Reader) (ports.PeticionCiclo, error) {
	var p ports.PeticionCiclo
	err := Decodificar(r, &p)
	return p, err
}
