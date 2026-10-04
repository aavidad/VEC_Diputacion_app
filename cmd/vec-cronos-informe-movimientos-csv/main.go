// vec-cronos-informe-movimientos-csv exporta un ejemplo sintético local.
// No consulta registros ni autoriza descargas de Cronos.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/adapters/informemovimientos"
)

func ejecutar(ctx context.Context, args []string, salida io.Writer) error {
	if len(args) != 2 || salida == nil {
		return informemovimientos.ErrEjemploInvalido
	}
	// #nosec G304 G703 -- rutas explícitas del operador de esta CLI local de ensayo.
	catalogo, err := os.Open(args[0])
	if err != nil {
		return informemovimientos.ErrEjemploInvalido
	}
	defer catalogo.Close()
	// #nosec G304 G703 -- el lector exige un fixture demo limitado a 64 KiB.
	entrada, err := os.Open(args[1])
	if err != nil {
		return informemovimientos.ErrEjemploInvalido
	}
	defer entrada.Close()
	ejemplo, err := informemovimientos.LeerEjemploSintetico(entrada)
	if err != nil {
		return err
	}
	contenido, err := informemovimientos.PrepararCSV(ctx, catalogo, ejemplo)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = salida.Write(contenido)
	return err
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ejecutar(ctx, os.Args[1:], os.Stdout); err != nil {
		codigo := informemovimientos.ErrEjemploNoDisponible.Error()
		if errors.Is(err, informemovimientos.ErrEjemploInvalido) {
			codigo = informemovimientos.ErrEjemploInvalido.Error()
		}
		_, _ = fmt.Fprintln(os.Stderr, codigo)
		os.Exit(1)
	}
}
