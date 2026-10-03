package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/meritos/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/seleccion/application"
)

type entrada struct {
	Solicitud application.SolicitudHechosPreparacion `json:"solicitud"`
	Paquete   domain.Paquete                         `json:"paquete"`
}

func run(in io.Reader, out, errOut io.Writer) int {
	err := ejecutar(in, out)
	if err == nil {
		return 0
	}
	_ = json.NewEncoder(errOut).Encode(struct {
		Clave string `json:"error_clave"`
	}{err.Error()})
	return 1
}

func ejecutar(in io.Reader, out io.Writer) error {
	var e entrada
	if err := leerJSON(in, &e); err != nil {
		return err
	}
	lector, err := simulacion.NuevoLectorHechos(e.Paquete)
	if err != nil {
		return err
	}
	resultado, err := application.ConsultarHechosPreparacion(context.Background(), e.Solicitud, lector)
	if err != nil {
		return err
	}
	if out == nil {
		return errSalida
	}
	if err := json.NewEncoder(out).Encode(resultado); err != nil {
		return errSalida
	}
	return nil
}

func main() { os.Exit(run(os.Stdin, os.Stdout, os.Stderr)) }
