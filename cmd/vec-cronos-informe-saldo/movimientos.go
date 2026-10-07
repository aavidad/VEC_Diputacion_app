package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/cronos/adapters/informemovimientos"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
)

func ejecutarMovimientos(ctx context.Context, args []string, salida io.Writer) error {
	if len(args) != 2 || salida == nil {
		return ports.ErrExportacionSaldoInvalida
	}
	// #nosec G304 G703 -- catálogo elegido por el operador local; sin HTTP, red ni servicios productivos.
	catalogo, err := os.Open(args[0])
	if err != nil {
		return ports.ErrExportacionSaldoInvalida
	}
	defer catalogo.Close()
	raw, err := io.ReadAll(io.LimitReader(catalogo, 65537))
	if err != nil || len(raw) > 65536 {
		return ports.ErrExportacionSaldoInvalida
	}
	var c informemovimientos.Catalogo
	if json.Unmarshal(raw, &c) != nil {
		return ports.ErrExportacionSaldoInvalida
	}
	p, err := informemovimientos.Nuevo(pdf.Renderizador{Idioma: c.Idioma}, bytes.NewReader(raw))
	if err != nil {
		return errorMovimientos(err)
	}
	// #nosec G304 G703 -- fixture explícito del operador; la entrada exige demo y excluye campos ajenos.
	fichero, err := os.Open(args[1])
	if err != nil {
		return ports.ErrExportacionSaldoInvalida
	}
	defer fichero.Close()
	e, err := informemovimientos.LeerEjemploSintetico(fichero)
	if err != nil {
		return errorMovimientos(err)
	}
	documento, err := p.PrepararEjemploSintetico(ctx, e)
	if err != nil {
		return errorMovimientos(err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = salida.Write(documento.Contenido)
	return err
}
func errorMovimientos(err error) error {
	if errors.Is(err, informemovimientos.ErrEjemploInvalido) {
		return ports.ErrExportacionSaldoInvalida
	}
	return err
}
