package datos

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	"vec-diputacion-granada/internal/modules/meritos/domain"
)

//go:embed roles_firmantes.v1.json
var rolesFirmantesV1 []byte

var ErrCatalogoRolesFirmantes = errors.New("meritos.error.catalogo_roles_firmantes")

type etiqueta struct {
	ES string `json:"es"`
	EN string `json:"en"`
}

type rolDatos struct {
	Clave    string   `json:"clave"`
	Clase    string   `json:"clase"`
	Efecto   string   `json:"efecto"`
	Etiqueta etiqueta `json:"etiqueta"`
}

type catalogoDatos struct {
	Referencia string     `json:"referencia"`
	Version    uint64     `json:"version"`
	Roles      []rolDatos `json:"roles"`
}

// CatalogoRolesFirmantesV1 ofrece clases de firma, no designaciones ni
// competencias. Las etiquetas permanecen en datos y fuera del dominio.
func CatalogoRolesFirmantesV1() (domain.CatalogoRolesFirmantes, error) {
	dec := json.NewDecoder(bytes.NewReader(rolesFirmantesV1))
	dec.DisallowUnknownFields()
	var entrada catalogoDatos
	var extra any
	if dec.Decode(&entrada) != nil || !errors.Is(dec.Decode(&extra), io.EOF) {
		return domain.CatalogoRolesFirmantes{}, ErrCatalogoRolesFirmantes
	}
	roles := make([]domain.DefinicionRolFirmante, 0, len(entrada.Roles))
	for _, r := range entrada.Roles {
		if r.Etiqueta.ES == "" || r.Etiqueta.EN == "" {
			return domain.CatalogoRolesFirmantes{}, ErrCatalogoRolesFirmantes
		}
		roles = append(roles, domain.DefinicionRolFirmante{Clave: r.Clave, Clase: r.Clase, Efecto: r.Efecto})
	}
	h := sha256.Sum256(rolesFirmantesV1)
	c := domain.CatalogoRolesFirmantes{Referencia: entrada.Referencia, Version: entrada.Version,
		HuellaSHA256: hex.EncodeToString(h[:]), Roles: roles}
	if c.Validar() != nil {
		return domain.CatalogoRolesFirmantes{}, ErrCatalogoRolesFirmantes
	}
	return c, nil
}
