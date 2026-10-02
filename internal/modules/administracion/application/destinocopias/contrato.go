package destinocopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

var (
	ErrConfiguracion = errors.New("copias_destino.configuracion")
	ErrMaterial      = errors.New("copias_destino.material")
	ErrNoDisponible  = errors.New("copias_destino.no_disponible")
	ErrExiste        = errors.New("copias_destino.existe")
	opaca            = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:_-]{0,127}$`)
	huella           = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

func ReferenciaOpaca(v string) bool { return opaca.MatchString(v) }
func Huella(v []byte) string        { h := sha256.Sum256(v); return hex.EncodeToString(h[:]) }
func AAD(v ports.Vinculo) ([]byte, error) {
	if !ReferenciaOpaca(v.ConjuntoRef) || !ReferenciaOpaca(v.ComponenteRef) || v.Posicion == 0 || !huella.MatchString(v.ManifiestoSHA256) {
		return nil, ErrMaterial
	}
	return json.Marshal(struct {
		Esquema string        `json:"esquema"`
		Vinculo ports.Vinculo `json:"vinculo"`
	}{"vec.copias.componente.v1", v})
}
func Objeto(v ports.Vinculo) (string, error) {
	aad, err := AAD(v)
	if err != nil {
		return "", err
	}
	return Huella(aad), nil
}
func ValidarReferencia(r ports.Referencia, limite int64) error {
	id, err := Objeto(r.Vinculo)
	if err != nil || r.ObjetoRef != id || !huella.MatchString(r.CifradoSHA256) || r.TamanoCifrado < 1 || r.TamanoCifrado > limite {
		return ErrMaterial
	}
	return nil
}
