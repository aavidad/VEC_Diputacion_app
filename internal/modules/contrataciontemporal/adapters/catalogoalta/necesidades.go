// Package catalogoalta carga las opciones versionadas de necesidad para la
// petición de personal temporal. El fichero incluido es un ejemplo sustituible.
package catalogoalta

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

//go:embed necesidades_v1.ejemplo.json
var necesidadesEjemplo []byte

// CargarNecesidades carga la distribución incluida o una publicación local.
// Una ruta declarada que falta o es inválida produce error, sin fallback.
func CargarNecesidades(ruta string) (domain.CatalogoNecesidadesAlta, error) {
	contenido := necesidadesEjemplo
	if ruta != "" {
		// #nosec G304 -- La ruta la aporta la configuración local, nunca HTTP.
		f, err := os.Open(ruta)
		if err != nil {
			return domain.CatalogoNecesidadesAlta{}, err
		}
		defer func() { _ = f.Close() }()
		contenido, err = io.ReadAll(io.LimitReader(f, 32769))
		if err != nil || len(contenido) > 32768 {
			return domain.CatalogoNecesidadesAlta{}, domain.ErrNecesidadAltaInvalida
		}
	}
	var catalogo domain.CatalogoNecesidadesAlta
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	if d.Decode(&catalogo) != nil || d.Decode(&struct{}{}) != io.EOF {
		return domain.CatalogoNecesidadesAlta{}, domain.ErrNecesidadAltaInvalida
	}
	h := sha256.Sum256(contenido)
	catalogo.HuellaSHA256 = hex.EncodeToString(h[:])
	if catalogo.Validar() != nil {
		return domain.CatalogoNecesidadesAlta{}, domain.ErrNecesidadAltaInvalida
	}
	return catalogo, nil
}
