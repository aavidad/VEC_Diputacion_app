// vec-cronos-informe-permisos emite sólo un ejemplo sintético por stdout.
// No compone fuentes de datos, registros de autorización ni rutas HTTP.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/adapters/informepermisos"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
)

type filaEjemplo struct {
	TipoRef string                    `json:"tipo_ref"`
	Resumen ports.FilaInformePermisos `json:"resumen"`
}
type escenarioSintetico struct {
	Demo      bool          `json:"demo"`
	Nombre    string        `json:"nombre"`
	Ejercicio int           `json:"ejercicio"`
	CorteUTC  time.Time     `json:"corte_utc"`
	Filas     []filaEjemplo `json:"filas"`
}

func ejecutar(ctx context.Context, args []string, salida io.Writer) error {
	if len(args) != 2 || salida == nil {
		return ports.ErrExportacionPermisosInvalida
	}
	// #nosec G304 G703 -- catálogo explícito del operador en CLI sintética, sin red ni servidor.
	catalogo, err := os.Open(args[0])
	if err != nil {
		return ports.ErrExportacionPermisosInvalida
	}
	defer catalogo.Close()
	preparador, err := informepermisos.Nuevo(pdf.Renderizador{}, catalogo)
	if err != nil {
		return err
	}
	// #nosec G304 G703 -- fixture sintético seleccionado por el operador; demo obligatorio y sin servicios productivos.
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
	// Política interna exclusiva del ejemplo. No implementa un puerto de autoridad
	// ni se transmite al caso de uso productivo. Estos dos tipos NO están aprobados.
	tiposEjemplo := map[string]bool{"vacaciones_ejemplo": true, "asuntos_propios_ejemplo": true}
	vistos := map[string]bool{}
	resumen := ports.ResumenPermisosInforme{Ejercicio: e.Ejercicio, CorteUTC: e.CorteUTC, Filas: make([]ports.FilaInformePermisos, 0, len(e.Filas))}
	for _, f := range e.Filas {
		if !tiposEjemplo[f.TipoRef] || vistos[f.TipoRef] {
			return ports.ErrExportacionPermisosInvalida
		}
		vistos[f.TipoRef] = true
		resumen.Filas = append(resumen.Filas, f.Resumen)
	}
	documento, err := preparador.PrepararEjemploSintetico(ctx, resumen, e.Nombre)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = salida.Write(documento.Contenido)
	return err
}
func informarError(salida io.Writer, fallo error) error {
	codigo := ports.ErrExportacionPermisosNoDisponible.Error()
	switch {
	case errors.Is(fallo, ports.ErrExportacionPermisosInvalida):
		codigo = ports.ErrExportacionPermisosInvalida.Error()
	case errors.Is(fallo, context.Canceled):
		codigo = "cronos_exportacion_permisos_cancelada"
	case errors.Is(fallo, context.DeadlineExceeded):
		codigo = "cronos_exportacion_permisos_tiempo_agotado"
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
