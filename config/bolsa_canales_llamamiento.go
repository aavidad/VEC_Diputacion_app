package config

import (
	_ "embed"
	"errors"
	"io"
	"os"
	"strings"
)

// EnvCatalogoCanalesLlamamientoBolsa permite sustituir, sin recompilar, el
// catálogo de canales de aviso de los llamamientos de Bolsa (lista, modo,
// resultados y activación). Sin ella rige el catálogo versionado del binario:
// correo y teléfono activos; SMS y Telegram apagados. Activar un canal en el
// catálogo no basta: además hace falta su proveedor real en la composición.
const EnvCatalogoCanalesLlamamientoBolsa = "VEC_BOLSA_CANALES_LLAMAMIENTO_PATH"

const maximoCatalogoCanalesLlamamientoBolsa = 64 << 10

var ErrCatalogoCanalesLlamamientoBolsa = errors.New("config: catalogo de canales de llamamiento de Bolsa no legible")

//go:embed bolsa_canales_llamamiento_v1.json
var catalogoCanalesLlamamientoBolsaV1 []byte

// CatalogoCanalesLlamamientoBolsa devuelve los bytes del catálogo vigente. La
// validación semántica pertenece al dominio de Bolsa.
func CatalogoCanalesLlamamientoBolsa() ([]byte, error) {
	ruta := strings.TrimSpace(os.Getenv(EnvCatalogoCanalesLlamamientoBolsa))
	if ruta == "" {
		return append([]byte(nil), catalogoCanalesLlamamientoBolsaV1...), nil
	}
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, ErrCatalogoCanalesLlamamientoBolsa
	}
	defer fichero.Close()
	datos, err := io.ReadAll(io.LimitReader(fichero, maximoCatalogoCanalesLlamamientoBolsa+1))
	if err != nil || len(datos) == 0 || len(datos) > maximoCatalogoCanalesLlamamientoBolsa {
		return nil, ErrCatalogoCanalesLlamamientoBolsa
	}
	return datos, nil
}
