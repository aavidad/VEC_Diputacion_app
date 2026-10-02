// Prepara la huella de una propuesta C3 local. No publica ni adopta.
package main

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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
		return rechazar(err, "entrada")
	}
	contenido, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	errCierre := f.Close()
	if err != nil || errCierre != nil {
		return rechazar(errors.Join(err, errCierre), "lectura")
	}
	politica, err := catalogoefectos.ValidarPropuestaC3(contenido)
	if err != nil {
		return rechazar(err, "validacion")
	}
	if err := json.NewEncoder(salida).Encode(resultado{politica.Referencia, politica.Version, politica.SHA256, len(politica.Reglas), "propuesta_sin_aprobar"}); err != nil {
		return rechazar(err, "salida")
	}
	return 0
}

// Registrar sólo la etapa cerrada: el error original puede contener rutas o
// contenido del archivo. El código de salida mantiene el rechazo observable.
func rechazar(err error, etapa string) int {
	if err == nil {
		return 0
	}
	slog.Error("cronos_catalogo_rechazado", "etapa", etapa)
	return 1
}
