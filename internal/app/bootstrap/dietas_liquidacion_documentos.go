package bootstrap

import (
	"net/http"
	"os"

	dietasdocs "vec-diputacion-granada/internal/modules/dietas/adapters/documentos"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

// El catálogo se lee en el arranque. Si falta, sólo la descarga queda cerrada.
func componerDocumentoLiquidacion(c dietasports.CapturadorLecturaLiquidacionEconomica, l dietasports.LectorLiquidacionEconomica) http.Handler {
	textos := make(map[string]dietasdocs.TextosLiquidacionDocumento, 2)
	for _, idioma := range []string{"es", "en"} {
		archivo, err := abrirCatalogoDocumentoLiquidacion(idioma)
		if err != nil {
			return nil
		}
		t, err := dietasdocs.CargarTextosLiquidacionDocumento(archivo)
		errCierre := archivo.Close()
		if err != nil || errCierre != nil || t.Idioma != idioma {
			return nil
		}
		textos[idioma] = t
	}
	h, err := dietashttp.NuevoManejadorDocumentoLiquidacion(c, l, textos)
	if err != nil {
		return nil
	}
	return h
}

func abrirCatalogoDocumentoLiquidacion(idioma string) (*os.File, error) {
	if idioma == "es" {
		f, err := os.Open("web/static/textos/es/dietas-liquidacion-documento.json")
		if err == nil {
			return f, nil
		}
		return os.Open("../../../web/static/textos/es/dietas-liquidacion-documento.json")
	}
	if idioma == "en" {
		f, err := os.Open("web/static/textos/en/dietas-liquidacion-documento.json")
		if err == nil {
			return f, nil
		}
		return os.Open("../../../web/static/textos/en/dietas-liquidacion-documento.json")
	}
	return nil, os.ErrNotExist
}
