// Package destinocopias define almacenamiento de componentes cifrados acotados.
// Estos puertos no conceden autorización administrativa.
package destinocopias

import "context"

type Vinculo struct {
	ConjuntoRef      string `json:"conjunto_ref"`
	ComponenteRef    string `json:"componente_ref"`
	Posicion         uint64 `json:"posicion"`
	ManifiestoSHA256 string `json:"manifiesto_sha256"`
}
type Solicitud struct {
	Vinculo               Vinculo
	Manifiesto, Contenido []byte
}
type Referencia struct {
	Vinculo       Vinculo `json:"vinculo"`
	ObjetoRef     string  `json:"objeto_ref"`
	CifradoSHA256 string  `json:"cifrado_sha256"`
	TamanoCifrado int64   `json:"tamano_cifrado"`
}
type Recuperado struct{ Manifiesto, Contenido []byte }
type Protector interface {
	Proteger(context.Context, Vinculo, []byte) ([]byte, error)
	Recuperar(context.Context, Vinculo, []byte) ([]byte, error)
}
type Destino interface {
	Publicar(context.Context, Solicitud) (Referencia, error)
	Recuperar(context.Context, Referencia) (Recuperado, error)
	Comprobar(context.Context, Referencia) error
	Borrar(context.Context, Referencia) error
}
