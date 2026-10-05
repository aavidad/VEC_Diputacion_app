// vec-cronos-informe-saldo emite exclusivamente un PDF o CSV sintético por stdout.
// No compone el caso de uso de exportación ni una autoridad de lectura/auditoría.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
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
	if len(args) > 0 && strings.HasPrefix(args[0], "--formato=") {
		if args[0] != "--formato=csv" {
			return ports.ErrExportacionSaldoInvalida
		}
		return ejecutarCSV(ctx, args[1:], salida)
	}
	if len(args) > 0 && strings.HasPrefix(args[0], "--vista=") {
		if args[0] != "--vista=movimientos" {
			return ports.ErrExportacionSaldoInvalida
		}
		return ejecutarMovimientos(ctx, args[1:], salida)
	}
	if len(args) != 2 || salida == nil {
		return ports.ErrExportacionSaldoInvalida
	}
	// #nosec G304 G703 -- rutas explícitas del operador en una CLI local sintética; sin servidor, secretos ni entrada HTTP.
	catalogo, err := os.Open(args[0])
	if err != nil {
		return ports.ErrExportacionSaldoInvalida
	}
	defer catalogo.Close()
	raw, err := io.ReadAll(io.LimitReader(catalogo, 65537))
	if err != nil || len(raw) > 65536 {
		return ports.ErrExportacionSaldoInvalida
	}
	var textos informesaldo.Catalogo
	if json.Unmarshal(raw, &textos) != nil {
		return ports.ErrExportacionSaldoInvalida
	}
	// El idioma y el preparador consumen los mismos bytes del catálogo.
	preparador, err := informesaldo.Nuevo(pdf.Renderizador{Idioma: textos.Idioma}, bytes.NewReader(raw))
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

// informarError emite sólo un código cerrado. Nunca imprime el error original,
// que puede contener texto del renderer, rutas o datos del documento.
func informarError(salida io.Writer, fallo error) error {
	codigo := ports.ErrExportacionSaldoNoDisponible.Error()
	switch {
	case errors.Is(fallo, ports.ErrExportacionSaldoInvalida):
		codigo = ports.ErrExportacionSaldoInvalida.Error()
	case errors.Is(fallo, context.Canceled):
		codigo = "cronos_exportacion_saldo_cancelada"
	case errors.Is(fallo, context.DeadlineExceeded):
		codigo = "cronos_exportacion_saldo_tiempo_agotado"
	}
	_, err := fmt.Fprintln(salida, codigo)
	return err
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ejecutar(ctx, os.Args[1:], os.Stdout); err != nil {
		_ = informarError(os.Stderr, err)
		os.Exit(1)
	}
}
