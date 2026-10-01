// vec-cronos-informe-saldo emite exclusivamente un PDF sintético por stdout.
// No compone el caso de uso de exportación ni una autoridad de lectura/auditoría.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/adapters/informesaldo"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
)

type escenarioSintetico struct {
	Demo   bool                  `json:"demo"`
	Nombre string                `json:"nombre"`
	Saldo  ports.SaldoExportable `json:"saldo"`
}

func ejecutar(ctx context.Context, args []string, salida io.Writer) error {
	if len(args) != 2 || salida == nil {
		return ports.ErrExportacionSaldoInvalida
	}
	// #nosec G304 G703 -- rutas explícitas del operador en una CLI local sintética; sin servidor, secretos ni entrada HTTP.
	catalogo, err := os.Open(args[0])
	if err != nil {
		return ports.ErrExportacionSaldoInvalida
	}
	defer catalogo.Close()
	preparador, err := informesaldo.Nuevo(pdf.Renderizador{}, catalogo)
	if err != nil {
		return err
	}
	// #nosec G304 G703 -- fixture elegido por el operador; se exige demo y nunca se conecta a servicios productivos.
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
	documento, err := preparador.PrepararEjemploSintetico(ctx, escenario.Saldo, escenario.Nombre)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = salida.Write(documento.Contenido)
	return err
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ejecutar(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
