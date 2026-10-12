package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var errEtiquetasHistoricasPeticionCentro = errors.New("contratacion temporal: catalogo de etiquetas historicas invalido")

type archivoEtiquetasHistoricasPeticionCentro struct {
	Version   int    `json:"version"`
	FuenteRef string `json:"fuente_ref"`
	Entradas  []struct {
		ActorRef string `json:"actor_ref"`
		Nombre   string `json:"nombre"`
	} `json:"entradas"`
}

// Solo contiene etiquetas: no certificados, roles, perfiles ni adscripciones.
// Cargar una persona aquí nunca la incorpora al resolvedor de identidad.
func cargarEtiquetasHistoricasPeticionCentro(root string) (map[string]string, error) {
	ruta := filepath.Join(root, "identidad", "centros-etiquetas.json")
	if _, err := os.Lstat(ruta); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	b, err := leerFicheroMaterialSeguro(ruta, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return nil, errEtiquetasHistoricasPeticionCentro
	}
	var archivo archivoEtiquetasHistoricasPeticionCentro
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&archivo) != nil || !errors.Is(d.Decode(&struct{}{}), io.EOF) || archivo.Version != 1 ||
		!domain.ReferenciaOpacaValida(archivo.FuenteRef) || len(archivo.Entradas) > 2048 {
		return nil, errEtiquetasHistoricasPeticionCentro
	}
	nombres := make(map[string]string, len(archivo.Entradas))
	for _, entrada := range archivo.Entradas {
		if !domain.ReferenciaOpacaValida(entrada.ActorRef) || !strings.HasPrefix(entrada.ActorRef, "per_") ||
			entrada.Nombre == "" || !utf8.ValidString(entrada.Nombre) || strings.TrimSpace(entrada.Nombre) != entrada.Nombre ||
			utf8.RuneCountInString(entrada.Nombre) > 250 || strings.IndexFunc(entrada.Nombre, unicode.IsControl) >= 0 {
			return nil, errEtiquetasHistoricasPeticionCentro
		}
		if anterior, existe := nombres[entrada.ActorRef]; existe && anterior != entrada.Nombre {
			return nil, errEtiquetasHistoricasPeticionCentro
		}
		nombres[entrada.ActorRef] = entrada.Nombre
	}
	return nombres, nil
}
