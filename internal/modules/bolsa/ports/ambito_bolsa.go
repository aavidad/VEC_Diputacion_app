package ports

import "errors"

// ErrAmbitoBolsaNoDeclarado indica que la bolsa no tiene ámbito gobernado
// (B144): ni la constitución gobernada ni Administración lo han declarado.
// Los actos sobre esa bolsa se deniegan; nunca se deduce el ámbito.
var ErrAmbitoBolsaNoDeclarado = errors.New("bolsa: ambito de la bolsa no declarado")

// AmbitoBolsa es el par de ámbitos con que se gobierna una bolsa: el ámbito
// organizativo y la unidad gestora. Procede de la tabla ambito_bolsa (B144),
// nunca de la petición ni de la composición.
type AmbitoBolsa struct {
	AmbitoRef string
	UnidadRef string
}
