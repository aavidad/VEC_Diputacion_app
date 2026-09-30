package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
)

// La preparación interna publica exclusivamente la huella de la CA interna.
// El servidor externo contrasta ese testigo con su CA propia sin leer ningún
// archivo del almacén interno durante el arranque o las peticiones.
func prepararManifiestoCAPropiaPortalExterno(interno, externo string) error {
	leerCA := func(raiz string) ([32]byte, error) {
		b, err := leerFicheroMaterialSeguro(filepath.Join(raiz, "ca", "ca.crt"), tamanoMaximoFicheroMaterialDesarrollo)
		if err != nil {
			return [32]byte{}, ErrPreparacionPortalExternoInvalida
		}
		ca, err := decodificarCertificadoUnico(b)
		if err != nil || !ca.IsCA {
			return [32]byte{}, ErrPreparacionPortalExternoInvalida
		}
		return sha256.Sum256(ca.Raw), nil
	}
	caInterna, err := leerCA(interno)
	if err != nil {
		return err
	}
	caExterna, err := leerCA(externo)
	if err != nil || caInterna == caExterna {
		return ErrPreparacionPortalExternoInvalida
	}
	ruta := filepath.Join(externo, "manifiesto.json")
	b, err := leerFicheroMaterialSeguro(ruta, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return ErrPreparacionPortalExternoInvalida
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil {
		return ErrPreparacionPortalExternoInvalida
	}
	var huellaPropia string
	if json.Unmarshal(campos["huella_ca_sha256"], &huellaPropia) != nil || !huellaHexIgual(strings.ToLower(huellaPropia), caExterna[:]) {
		return ErrPreparacionPortalExternoInvalida
	}
	campos["version"] = json.RawMessage("2")
	campos["huella_ca_interna_sha256"], err = json.Marshal(hex.EncodeToString(caInterna[:]))
	if err != nil {
		return ErrPreparacionPortalExternoInvalida
	}
	b, err = json.MarshalIndent(campos, "", "  ")
	if err != nil {
		return ErrPreparacionPortalExternoInvalida
	}
	return escribirFicheroPrivado(ruta, append(b, '\n'))
}
