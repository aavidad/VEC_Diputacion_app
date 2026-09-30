package bootstrap

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
)

type huellasCAPortalExterno struct {
	der  [sha256.Size]byte
	spki [sha256.Size]byte
}

// La preparación interna publica exclusivamente la huella de la CA interna.
// El servidor externo contrasta ese testigo con su CA propia sin leer ningún
// archivo del almacén interno durante el arranque o las peticiones.
func prepararManifiestoCAPropiaPortalExterno(interno, externo string) error {
	leerCA := func(raiz string) (huellasCAPortalExterno, error) {
		b, err := leerFicheroMaterialSeguro(filepath.Join(raiz, "ca", "ca.crt"), tamanoMaximoFicheroMaterialDesarrollo)
		if err != nil {
			return huellasCAPortalExterno{}, ErrPreparacionPortalExternoInvalida
		}
		ca, err := decodificarCertificadoUnico(b)
		if err != nil || !ca.IsCA || ca.KeyUsage&x509.KeyUsageCertSign == 0 {
			return huellasCAPortalExterno{}, ErrPreparacionPortalExternoInvalida
		}
		return huellasCAPortalExterno{
			der: sha256.Sum256(ca.Raw), spki: sha256.Sum256(ca.RawSubjectPublicKeyInfo),
		}, nil
	}
	caInterna, err := leerCA(interno)
	if err != nil {
		return err
	}
	caExterna, err := leerCA(externo)
	if err != nil || caInterna.der == caExterna.der || caInterna.spki == caExterna.spki {
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
	if json.Unmarshal(campos["huella_ca_sha256"], &huellaPropia) != nil || !huellaHexIgual(strings.ToLower(huellaPropia), caExterna.der[:]) {
		return ErrPreparacionPortalExternoInvalida
	}
	campos["version"] = json.RawMessage("2")
	campos["huella_ca_interna_sha256"], err = json.Marshal(hex.EncodeToString(caInterna.der[:]))
	if err != nil {
		return ErrPreparacionPortalExternoInvalida
	}
	b, err = json.MarshalIndent(campos, "", "  ")
	if err != nil {
		return ErrPreparacionPortalExternoInvalida
	}
	return escribirFicheroPrivado(ruta, append(b, '\n'))
}
