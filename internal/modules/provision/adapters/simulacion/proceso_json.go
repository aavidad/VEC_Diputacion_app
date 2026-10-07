package simulacion

import (
	"bytes"
	_ "embed"
	"io"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

//go:embed proceso_ejemplo.json
var ejemploProceso []byte

func DecodificarProceso(r io.Reader) (ports.PeticionProceso, error) {
	var p ports.PeticionProceso
	err := Decodificar(r, &p)
	return p, err
}

// EjemploProceso devuelve una nueva copia de datos enteramente sintéticos.
// El servidor puede fijar estos hechos sin aceptar hechos personales del cliente.
func EjemploProceso() (ports.PeticionProceso, error) {
	p, err := DecodificarProceso(bytes.NewReader(ejemploProceso))
	if err != nil {
		return ports.PeticionProceso{}, err
	}
	if err := domain.ValidarSolicitud(p.Proceso, p.Solicitud); err != nil {
		return ports.PeticionProceso{}, err
	}
	return p, nil
}
