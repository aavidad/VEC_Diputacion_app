package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/cronos/adapters/informepermisos"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func ejecutarCSV(ctx context.Context, args []string, salida io.Writer) error {
	if ctx == nil || len(args) != 2 || salida == nil {
		return ports.ErrExportacionPermisosInvalida
	}
	// #nosec G304 G703 -- catálogo local elegido por el operador; sin entrada HTTP.
	catalogo, err := os.Open(args[0])
	if err != nil {
		return ports.ErrExportacionPermisosInvalida
	}
	defer catalogo.Close()
	// #nosec G304 G703 -- fixture local; demo obligatorio y sin servicio productivo.
	fichero, err := os.Open(args[1])
	if err != nil {
		return ports.ErrExportacionPermisosInvalida
	}
	defer fichero.Close()
	dec := json.NewDecoder(io.LimitReader(fichero, 65537))
	dec.DisallowUnknownFields()
	var e escenarioSintetico
	if dec.Decode(&e) != nil || dec.Decode(new(any)) != io.EOF || !e.Demo || len(e.Filas) > 2 {
		return ports.ErrExportacionPermisosInvalida
	}
	resumen, err := resumenEjemplo(e)
	if err != nil {
		return err
	}
	contenido, err := informepermisos.PrepararCSVEjemploSintetico(ctx, catalogo, informepermisos.EjemploSinteticoCSV{Demo: e.Demo, Nombre: e.Nombre, Resumen: resumen})
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = salida.Write(contenido)
	return err
}
