package config

import (
	_ "embed"
	"errors"
	"io"
	"os"
	"strings"
)

// EnvCatalogoCorreoLlamamientoBolsa permite sustituir, sin recompilar, el
// catálogo del correo personalizado de los llamamientos de Bolsa (plantillas,
// marcadores, textos por defecto y límite). Sin ella rige el catálogo
// versionado que acompaña al binario.
const EnvCatalogoCorreoLlamamientoBolsa = "VEC_BOLSA_CORREO_LLAMAMIENTO_PATH"

const maximoCatalogoCorreoLlamamientoBolsa = 64 << 10

var ErrCatalogoCorreoLlamamientoBolsa = errors.New("config: catalogo de correo de llamamiento de Bolsa no legible")

//go:embed bolsa_correo_llamamiento_v1.json
var catalogoCorreoLlamamientoBolsaV1 []byte

// CatalogoCorreoLlamamientoBolsa devuelve los bytes del catálogo vigente. La
// validación semántica pertenece al dominio de Bolsa.
func CatalogoCorreoLlamamientoBolsa() ([]byte, error) {
	ruta := strings.TrimSpace(os.Getenv(EnvCatalogoCorreoLlamamientoBolsa))
	if ruta == "" {
		return append([]byte(nil), catalogoCorreoLlamamientoBolsaV1...), nil
	}
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, ErrCatalogoCorreoLlamamientoBolsa
	}
	defer fichero.Close()
	datos, err := io.ReadAll(io.LimitReader(fichero, maximoCatalogoCorreoLlamamientoBolsa+1))
	if err != nil || len(datos) == 0 || len(datos) > maximoCatalogoCorreoLlamamientoBolsa {
		return nil, ErrCatalogoCorreoLlamamientoBolsa
	}
	return datos, nil
}
