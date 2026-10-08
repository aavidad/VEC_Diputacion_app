package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
)

// contextoRecursoCargaConvocaValido verifica la preimagen exacta que B95
// consume: dos ámbitos resueltos y ningún atributo de vista previa. La
// capacidad debe estar ligada a esos mismos bytes antes de abrir la TX.
func contextoRecursoCargaConvocaValido(raw []byte, huellaCapacidad string) bool {
	if len(raw) == 0 || len(raw) > 2048 {
		return false
	}
	var c struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || dec.Decode(new(any)) != io.EOF || len(c.Ambitos) != 2 ||
		c.Ambitos["ambito_ref"] == "" || c.Ambitos["unidad_ref"] == "" || c.Atributos == nil || len(c.Atributos) != 0 {
		return false
	}
	canonico, err := json.Marshal(c)
	if err != nil || !bytes.Equal(raw, canonico) {
		return false
	}
	suma := sha256.Sum256(raw)
	return hex.EncodeToString(suma[:]) == huellaCapacidad
}
