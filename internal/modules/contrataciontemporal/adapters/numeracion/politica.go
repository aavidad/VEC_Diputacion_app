// Package numeracion carga el formato versionado del número externo MOAD.
package numeracion

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

//go:embed numero_expediente.json
var catalogo []byte

// Cargar usa el catálogo distribuido o una publicación configurada. Nunca
// genera ni reserva números de expediente.
func Cargar(ruta string) (domain.PoliticaNumeroExpediente, error) {
	contenido := catalogo
	if ruta != "" {
		// #nosec G304 -- La ruta procede de configuración local de despliegue, nunca de una petición HTTP.
		f, err := os.Open(ruta)
		if err != nil {
			return domain.PoliticaNumeroExpediente{}, err
		}
		defer func() { _ = f.Close() }()
		contenido, err = io.ReadAll(io.LimitReader(f, 8193))
		if err != nil || len(contenido) > 8192 {
			return domain.PoliticaNumeroExpediente{}, domain.ErrDatoInvalido
		}
	}
	var politica domain.PoliticaNumeroExpediente
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	if d.Decode(&politica) != nil || d.Decode(&struct{}{}) != io.EOF || politica.Validar() != nil {
		return domain.PoliticaNumeroExpediente{}, domain.ErrDatoInvalido
	}
	return politica, nil
}
