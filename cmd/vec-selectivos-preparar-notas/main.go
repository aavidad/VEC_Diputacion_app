package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

// El CLI consume stdin y emite JSON; no recibe rutas ni abre servicios.
func ejecutar(args []string, entrada io.Reader, salida, diagnostico io.Writer) int {
	if len(args) != 0 || entrada == nil || salida == nil {
		return informar(diagnostico, "solicitud_invalida")
	}
	datos, err := io.ReadAll(io.LimitReader(entrada, simulacion.MaximoBytes+1))
	if err != nil || len(datos) > simulacion.MaximoBytes {
		return informar(diagnostico, "solicitud_invalida")
	}
	solicitud, err := simulacion.DecodificarRevisionNotas(datos)
	if err != nil {
		return informar(diagnostico, "solicitud_invalida")
	}
	revision, err := simulacion.PrepararRevisionNotas(solicitud)
	if err != nil {
		codigoError := "preparacion_no_disponible"
		if errors.Is(err, simulacion.ErrSolicitud) || errors.Is(err, simulacion.ErrNotas) || errors.Is(err, simulacion.ErrEjemplo) || errors.Is(err, domain.ErrConfiguracion) {
			codigoError = "solicitud_invalida"
		}
		return informar(diagnostico, codigoError)
	}
	if json.NewEncoder(salida).Encode(revision) != nil {
		return informar(diagnostico, "salida_no_disponible")
	}
	return 0
}

func informar(w io.Writer, clave string) int {
	if w == nil || json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{clave}) != nil {
		return 2
	}
	return 1
}

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
