package main

import (
	"context"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/i18n"
)

func ejecutarAportacion(ctx context.Context, formato string, entrada io.Reader, salida, errores io.Writer, catalogo *i18n.Catalog, idioma string) int {
	var resultado any
	if formato == "antecedente" {
		var material ports.MaterialAdmisionPreparacion
		if leerJSON(entrada, &material) != nil {
			return informarError(errores, catalogo, idioma, domain.ErrAportacionAdmision.Error())
		}
		antecedente, err := application.IdentificarAntecedenteAdmision(ctx, material)
		if err != nil {
			return informarError(errores, catalogo, idioma, domain.ErrAportacionAdmision.Error())
		}
		resultado = antecedente
	} else {
		var material application.MaterialAportacionAdmision
		if leerJSON(entrada, &material) != nil {
			return informarError(errores, catalogo, idioma, domain.ErrAportacionAdmision.Error())
		}
		aportacion, err := application.PrepararAportacionAdmision(ctx, material)
		if err != nil {
			return informarError(errores, catalogo, idioma, domain.ErrAportacionAdmision.Error())
		}
		resultado = aportacion
	}
	enc := json.NewEncoder(salida)
	enc.SetIndent("", "  ")
	if err := enc.Encode(resultado); err != nil {
		return informarError(errores, catalogo, idioma, "seleccion.admision.salida_no_disponible")
	}
	return 0
}
