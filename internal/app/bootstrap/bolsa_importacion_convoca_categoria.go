package bootstrap

import (
	"errors"
	"strings"
	"vec-diputacion-granada/config"
)

var ErrCategoriaImportacionConvocaInvalida = errors.New("bootstrap: categoria RPT de importacion Convoca invalida")

func validarCategoriaImportacionConvoca(cfg config.Config, clave string) (string, error) {
	clave = strings.TrimSpace(clave)
	if clave == "" || strings.TrimSpace(cfg.RPTCatalogoPath) == "" {
		return "", ErrCategoriaImportacionConvocaInvalida
	}
	categorias, err := cargarCategoriasRPTDesarrollo(cfg.RPTCatalogoPath)
	if err != nil {
		return "", ErrCategoriaImportacionConvocaInvalida
	}
	ref := "categoria:rpt:" + clave
	for _, c := range categorias {
		if c.Referencia == ref {
			return ref, nil
		}
	}
	return "", ErrCategoriaImportacionConvocaInvalida
}
