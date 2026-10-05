package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/cronos/adapters/informesaldo"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func ejecutarCSV(ctx context.Context, args []string, salida io.Writer) error {
	if ctx == nil || len(args) != 2 || salida == nil {
		return ports.ErrExportacionSaldoInvalida
	}
	// #nosec G304 G703 -- catálogo elegido por el operador local; no existe entrada HTTP ni cuenta de servicio.
	catalogo, err := os.Open(args[0])
	if err != nil {
		return ports.ErrExportacionSaldoInvalida
	}
	defer catalogo.Close()
	// #nosec G304 G703 -- fixture local, con demo obligatorio; sin conexión a autoridades productivas.
	fichero, err := os.Open(args[1])
	if err != nil {
		return ports.ErrExportacionSaldoInvalida
	}
	defer fichero.Close()
	dec := json.NewDecoder(io.LimitReader(fichero, 65537))
	dec.DisallowUnknownFields()
	var escenario escenarioSintetico
	if dec.Decode(&escenario) != nil || dec.Decode(new(any)) != io.EOF || !escenario.Demo {
		return ports.ErrExportacionSaldoInvalida
	}
	contenido, err := informesaldo.PrepararCSVEjemploSintetico(ctx, catalogo, escenario.Saldo, escenario.Nombre)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = salida.Write(contenido)
	return err
}
