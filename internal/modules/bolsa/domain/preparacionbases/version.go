package preparacionbases

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var ErrVersionInvalida = errors.New("bolsa.preparacion_bases.version_invalida")

// Esperada es la preimagen CAS. Revision cero y huella vacia significan alta;
// no se permiten actualizaciones sin fijar ambos valores anteriores.
type Esperada struct {
	PreparacionRef       string `json:"preparacion_ref"`
	Revision             int    `json:"revision"`
	HuellaMaterialSHA256 string `json:"huella_material_sha256"`
}

func (e Esperada) Validar(permitirAlta bool) error {
	if !IdentificadorValido(e.PreparacionRef) || e.Revision < 0 || e.Revision > 1_000_000 ||
		(e.Revision == 0 && (!permitirAlta || e.HuellaMaterialSHA256 != "")) ||
		(e.Revision > 0 && !HuellaValida(e.HuellaMaterialSHA256)) {
		return ErrVersionInvalida
	}
	return nil
}

type Version struct {
	Ambito   bolsa.AmbitoOrganizativoConvocatoria `json:"ambito"`
	Estado   Esperada                             `json:"estado"`
	Material Material                             `json:"material"`
}

func (v Version) Validar() error {
	h, err := v.Material.HuellaSHA256()
	if v.Ambito.Validar() != nil || v.Estado.Validar(false) != nil || err != nil || h != v.Estado.HuellaMaterialSHA256 {
		return ErrVersionInvalida
	}
	return nil
}

// HuellaIntencion liga preimagen y material, sin fechas ni clave cliente.
// La barrera durable separa claves por actor y deniega reutilizacion semantica.
func HuellaIntencion(e Esperada, m Material, ambito bolsa.AmbitoOrganizativoConvocatoria) (string, error) {
	h, err := m.HuellaSHA256()
	if ambito.Validar() != nil || e.Validar(true) != nil || e.Revision == 1_000_000 || err != nil {
		return "", ErrVersionInvalida
	}
	b, err := json.Marshal(struct {
		Ambito         bolsa.AmbitoOrganizativoConvocatoria `json:"ambito"`
		Esquema        string                               `json:"esquema"`
		Esperada       Esperada                             `json:"esperada"`
		HuellaMaterial string                               `json:"huella_material"`
	}{ambito, "bolsa.preparacion_bases.intencion.v1", e, h})
	if err != nil {
		return "", ErrVersionInvalida
	}
	suma := sha256.Sum256(b)
	return hex.EncodeToString(suma[:]), nil
}
