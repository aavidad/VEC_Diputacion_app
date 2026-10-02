// Prepara la huella de una propuesta C3 local. No publica ni adopta.
package main

import (
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
)

type resultado struct {
	PoliticaRef     string `json:"politica_ref"`
	PoliticaVersion int64  `json:"politica_version"`
	ContenidoSHA256 string `json:"contenido_sha256"`
	Reglas          int    `json:"reglas"`
	Estado          string `json:"estado"`
}

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout))
}

func ejecutar(argumentos []string, salida io.Writer) int {
	if len(argumentos) != 1 {
		return 2
	}
	// La ruta la elige quien ejecuta esta herramienta local; solo se emite el
	// resumen si su contenido supera el validador estricto de la propuesta C3.
	f, err := os.Open(argumentos[0]) // #nosec G703 -- argumento local explícito del operador
	if err != nil {
		return 1
	}
	contenido, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	errCierre := f.Close()
	if err != nil || errCierre != nil {
		return 1
	}
	politica, err := catalogoefectos.ValidarPropuestaC3(contenido)
	if err != nil {
		return 1
	}
	if json.NewEncoder(salida).Encode(resultado{politica.Referencia, politica.Version, politica.SHA256, len(politica.Reglas), "propuesta_sin_aprobar"}) != nil {
		return 1
	}
	return 0
}
