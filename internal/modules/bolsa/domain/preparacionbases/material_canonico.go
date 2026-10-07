package preparacionbases

import (
	"bytes"
	"encoding/json"
	"io"
)

// RepresentacionCanonica exporta exactamente la preimagen ya usada por la
// huella v1. No modifica el esquema ni convierte material en bases aprobadas.
func (m Material) RepresentacionCanonica() ([]byte, error) {
	c, err := m.Canonico()
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(struct {
		Esquema  string   `json:"esquema"`
		Material Material `json:"material"`
	}{"bolsa.preparacion_bases.material.v1", c})
	if err != nil {
		return nil, ErrMaterialInvalido
	}
	return b, nil
}

// DecodificarMaterialCanonico rechaza duplicados, campos ajenos y cualquier
// representacion distinta de los bytes canonicos conservados por Bolsa.
func DecodificarMaterialCanonico(b []byte) (Material, error) {
	if len(b) == 0 || len(b) > MaximoBytesMaterial+128 {
		return Material{}, ErrMaterialInvalido
	}
	var envoltura struct {
		Esquema  string   `json:"esquema"`
		Material Material `json:"material"`
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&envoltura) != nil || envoltura.Esquema != "bolsa.preparacion_bases.material.v1" {
		return Material{}, ErrMaterialInvalido
	}
	if d.Decode(new(any)) != io.EOF {
		return Material{}, ErrMaterialInvalido
	}
	canonico, err := envoltura.Material.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, b) {
		return Material{}, ErrMaterialInvalido
	}
	return envoltura.Material.Canonico()
}
