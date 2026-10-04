// vec-cronos-informe-permisos emite sólo un ejemplo sintético por stdout.
// No compone fuentes de datos, registros de autorización ni rutas HTTP.
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

	"vec-diputacion-granada/internal/modules/cronos/adapters/informepermisos"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
)

type filaEjemplo struct {
	TipoRef string                    `json:"tipo_ref"`
	Resumen ports.FilaInformePermisos `json:"resumen"`
}
type escenarioSintetico struct {
	Demo             bool          `json:"demo"`
	Nombre           string        `json:"nombre"`
	Ejercicio        int           `json:"ejercicio"`
	CorteUTC         time.Time     `json:"corte_utc"`
	CamposPermitidos []string      `json:"campos_permitidos"`
	Filas            []filaEjemplo `json:"filas"`
}

func ejecutar(ctx context.Context, args []string, salida io.Writer) error {
	if len(args) > 0 && strings.HasPrefix(args[0], "--formato=") {
		if args[0] != "--formato=csv" {
			return ports.ErrExportacionPermisosInvalida
		}
		return ejecutarCSV(ctx, args[1:], salida)
	}
	if len(args) != 2 || salida == nil {
		return ports.ErrExportacionPermisosInvalida
	}
	// #nosec G304 G703 -- catálogo explícito del operador en CLI sintética, sin red ni servidor.
	catalogo, err := os.Open(args[0])
	if err != nil {
		return ports.ErrExportacionPermisosInvalida
	}
	defer catalogo.Close()
	raw, err := io.ReadAll(io.LimitReader(catalogo, 65537))
	if err != nil || len(raw) > 65536 {
		return ports.ErrExportacionPermisosInvalida
	}
	var textos informepermisos.Catalogo
	if json.Unmarshal(raw, &textos) != nil {
		return ports.ErrExportacionPermisosInvalida
	}
	// El idioma y el preparador consumen los mismos bytes del catálogo.
	preparador, err := informepermisos.Nuevo(pdf.Renderizador{Idioma: textos.Idioma}, bytes.NewReader(raw))
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
	resumen, err := resumenEjemplo(e)
	if err != nil {
		return err
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

func resumenEjemplo(e escenarioSintetico) (ports.ResumenPermisosInforme, error) {
	// Política interna exclusiva del ejemplo. No implementa un puerto de autoridad
	// ni se transmite al caso de uso productivo. Estos dos tipos NO están aprobados.
	tiposEjemplo := map[string]bool{"vacaciones_ejemplo": true, "asuntos_propios_ejemplo": true}
	vistos := map[string]bool{}
	resumen := ports.ResumenPermisosInforme{Ejercicio: e.Ejercicio, CorteUTC: e.CorteUTC, CamposPermitidos: append([]string(nil), e.CamposPermitidos...), Filas: make([]ports.FilaInformePermisos, 0, len(e.Filas))}
	for _, f := range e.Filas {
		if !tiposEjemplo[f.TipoRef] || vistos[f.TipoRef] {
			return ports.ResumenPermisosInforme{}, ports.ErrExportacionPermisosInvalida
		}
		vistos[f.TipoRef] = true
		resumen.Filas = append(resumen.Filas, f.Resumen)
	}
	return resumen, nil
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
