package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/catalogoadmision"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/i18n"
)

// La lista reúne una revisión S4 por solicitud; 16 MiB cubren el máximo de
// solicitudes de domain.MaximoSolicitudesLista con material de tamaño normal.
const maximoEntradaLista = 16 * 1024 * 1024

// ejecutarListaProvisional compone el borrador con el catálogo configurado en
// fichero. Sin catálogo válido no se produce ninguna salida.
func ejecutarListaProvisional(ctx context.Context, dir, nombre string, entrada io.Reader, salida, errores io.Writer, catalogo *i18n.Catalog, idioma string) int {
	catalogos, err := catalogoadmision.Cargar(dir, nombre)
	if err != nil {
		return informarError(errores, catalogo, idioma, application.ErrCatalogoAdmisionNoDisponible.Error())
	}
	var material ports.MaterialListaAdmision
	if leerJSONHasta(entrada, &material, maximoEntradaLista) != nil {
		return informarError(errores, catalogo, idioma, domain.ErrListaAdmision.Error())
	}
	lista, err := application.PrepararListaAdmisionProvisional(ctx, material, catalogos)
	if errors.Is(err, application.ErrCatalogoAdmisionNoDisponible) {
		return informarError(errores, catalogo, idioma, application.ErrCatalogoAdmisionNoDisponible.Error())
	}
	if err != nil {
		return informarError(errores, catalogo, idioma, domain.ErrListaAdmision.Error())
	}
	enc := json.NewEncoder(salida)
	enc.SetIndent("", "  ")
	if enc.Encode(lista) != nil {
		return informarError(errores, catalogo, idioma, "seleccion.admision.salida_no_disponible")
	}
	return 0
}
