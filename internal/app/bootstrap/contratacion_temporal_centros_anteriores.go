package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

// rutaCentrosAnterioresCTEjemplo nombra los centros que usan expedientes dados
// de alta antes de publicar los centros de la RPT. Es un paquete de ejemplo
// retirable: sin el fichero no se añade ningún centro y el catálogo de alta
// ofrece solo los de la organización.
const rutaCentrosAnterioresCTEjemplo = "data/demo/contratacion/ct_centros_anteriores.demo.json"

const maximoCentrosAnterioresCT = 20

var (
	errCentrosAnterioresCTInvalidos = errors.New("bootstrap: catalogo de centros anteriores de contratacion temporal no valido")
	patronCentroAnteriorCT          = regexp.MustCompile(`^centro:[a-z0-9-]{1,40}:[a-z0-9-]{1,40}$`)
)

type archivoCentrosAnterioresCT struct {
	Version int    `json:"version"`
	Motivo  string `json:"motivo"`
	Centros []struct {
		Referencia string `json:"referencia"`
		Etiqueta   string `json:"etiqueta"`
	} `json:"centros"`
}

// cargarCentrosAnterioresCT busca el paquete en el directorio de trabajo o en
// la raíz del repositorio (pruebas). Un fichero presente pero inválido impide
// arrancar en lugar de mostrar referencias sin nombre.
func cargarCentrosAnterioresCT() ([]centroCatalogosAltaContratacionTemporalDesarrollo, error) {
	for _, ruta := range []string{rutaCentrosAnterioresCTEjemplo, "../../../" + rutaCentrosAnterioresCTEjemplo} {
		contenido, err := os.ReadFile(ruta)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || len(contenido) > 64<<10 {
			return nil, errCentrosAnterioresCTInvalidos
		}
		return centrosAnterioresCTDesdeJSON(contenido)
	}
	return nil, nil
}

func centrosAnterioresCTDesdeJSON(contenido []byte) ([]centroCatalogosAltaContratacionTemporalDesarrollo, error) {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var archivo archivoCentrosAnterioresCT
	if decodificador.Decode(&archivo) != nil {
		return nil, errCentrosAnterioresCTInvalidos
	}
	var sobra any
	if !errors.Is(decodificador.Decode(&sobra), io.EOF) || archivo.Version != 1 ||
		strings.TrimSpace(archivo.Motivo) == "" || len(archivo.Centros) > maximoCentrosAnterioresCT {
		return nil, errCentrosAnterioresCTInvalidos
	}
	centros := make([]centroCatalogosAltaContratacionTemporalDesarrollo, 0, len(archivo.Centros))
	vistos := make(map[string]bool, len(archivo.Centros))
	for _, c := range archivo.Centros {
		etiqueta := strings.TrimSpace(c.Etiqueta)
		if !patronCentroAnteriorCT.MatchString(c.Referencia) || vistos[c.Referencia] ||
			etiqueta == "" || etiqueta != c.Etiqueta || utf8.RuneCountInString(etiqueta) > 200 {
			return nil, errCentrosAnterioresCTInvalidos
		}
		vistos[c.Referencia] = true
		centros = append(centros, centroCatalogosAltaContratacionTemporalDesarrollo{
			Referencia: c.Referencia,
			Etiqueta:   etiqueta,
			Contactos: []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo{{
				Referencia: contactoAltaContratacionTemporalDesarrollo, Etiqueta: "Contacto del centro",
			}},
		})
	}
	return centros, nil
}
