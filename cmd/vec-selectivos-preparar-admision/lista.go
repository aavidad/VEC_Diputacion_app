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
// solicitudes de domain.MaximoSolicitudesLista con material de tamaño normal,
// también cuando la revisión de la provisional añade la lista anterior.
const maximoEntradaLista = 16 * 1024 * 1024

// ejecutarLista atiende las salidas de listas con el catálogo configurado en
// fichero. Sin catálogo válido no se produce ninguna salida.
//   - lista-provisional: borrador de la provisional.
//   - antecedente-lista: huella de esa provisional, para citarla en la definitiva.
//   - lista-definitiva: borrador de la definitiva desde la provisional y las resoluciones.
//   - revision-provisional: nueva revisión de la provisional con solicitudes omitidas.
func ejecutarLista(ctx context.Context, formato, dir, nombre string, entrada io.Reader, salida, errores io.Writer, catalogo *i18n.Catalog, idioma string) int {
	invalida := domain.ErrListaAdmision.Error()
	switch formato {
	case "lista-definitiva":
		invalida = domain.ErrListaDefinitiva.Error()
	case "revision-provisional":
		invalida = domain.ErrRevisionLista.Error()
	}
	catalogos, err := catalogoadmision.Cargar(dir, nombre)
	if err != nil {
		return informarError(errores, catalogo, idioma, application.ErrCatalogoAdmisionNoDisponible.Error())
	}
	var resultado any
	switch formato {
	case "revision-provisional":
		var material ports.MaterialRevisionProvisional
		if leerJSONHasta(entrada, &material, maximoEntradaLista) != nil {
			return informarError(errores, catalogo, idioma, invalida)
		}
		resultado, err = application.PrepararRevisionProvisional(ctx, material, catalogos)
	case "lista-definitiva":
		var material ports.MaterialListaDefinitiva
		if leerJSONHasta(entrada, &material, maximoEntradaLista) != nil {
			return informarError(errores, catalogo, idioma, invalida)
		}
		resultado, err = application.PrepararListaAdmisionDefinitiva(ctx, material, catalogos)
	default:
		var material ports.MaterialListaAdmision
		if leerJSONHasta(entrada, &material, maximoEntradaLista) != nil {
			return informarError(errores, catalogo, idioma, invalida)
		}
		if formato == "antecedente-lista" {
			resultado, err = application.IdentificarListaProvisional(ctx, material, catalogos)
		} else {
			resultado, err = application.PrepararListaAdmisionProvisional(ctx, material, catalogos)
		}
	}
	if errors.Is(err, application.ErrCatalogoAdmisionNoDisponible) {
		return informarError(errores, catalogo, idioma, application.ErrCatalogoAdmisionNoDisponible.Error())
	}
	if err != nil {
		return informarError(errores, catalogo, idioma, invalida)
	}
	enc := json.NewEncoder(salida)
	enc.SetIndent("", "  ")
	if enc.Encode(resultado) != nil {
		return informarError(errores, catalogo, idioma, "seleccion.admision.salida_no_disponible")
	}
	return 0
}
