package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var errDefinicionCircuitoRRHHNoDisponible = errors.New("contratacion temporal: definicion del circuito RRHH no disponible")

// cargarDefinicionCircuitoRRHH lee la definicion publicada del circuito. La
// terna del flujo identifica el contenido exacto; el dominio comprueba su
// huella y las transiciones antes de que cualquier expediente pueda usarla.
func cargarDefinicionCircuitoRRHH(ruta string) (domain.DefinicionCircuitoRRHH, error) {
	vacia := domain.DefinicionCircuitoRRHH{}
	if strings.TrimSpace(ruta) == "" {
		return vacia, errDefinicionCircuitoRRHHNoDisponible
	}
	// #nosec G304 -- Ruta de configuración local del operador, ajena al cuerpo HTTP; lectura y JSON acotados.
	archivo, err := os.Open(ruta)
	if err != nil {
		return vacia, errDefinicionCircuitoRRHHNoDisponible
	}
	defer archivo.Close()

	const maximoCatalogo = 64 * 1024
	contenido, err := io.ReadAll(io.LimitReader(archivo, maximoCatalogo+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCatalogo {
		return vacia, errDefinicionCircuitoRRHHNoDisponible
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var definicion domain.DefinicionCircuitoRRHH
	if decodificador.Decode(&definicion) != nil ||
		decodificador.Decode(&struct{}{}) != io.EOF ||
		definicion.Validar() != nil {
		return vacia, errDefinicionCircuitoRRHHNoDisponible
	}
	return definicion, nil
}
