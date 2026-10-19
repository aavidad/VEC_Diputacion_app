package bootstrap

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// El suplemento sólo identifica a una segunda persona sintética. Sus
// referencias de contexto no son una asignación ni una concesión de V3.
const nombreSegundoTecnicoRRHHDesarrollo = "tecnico-rrhh-adicional.json"
const certificadoSegundoTecnicoRRHHDesarrollo = "mtls/tecnico-rrhh-adicional.crt"
const identidadSegundoTecnicoRRHHDesarrollo = "identidad/tecnico-rrhh-adicional-identidad.json"

type archivoSegundoTecnicoRRHHDesarrollo struct {
	Version     int    `json:"version"`
	Autoridad   string `json:"autoridad"`
	Certificado string `json:"certificado"`
	Identidad   string `json:"identidad"`
	Sujeto      string `json:"sujeto"`
	CuentaRef   string `json:"cuenta_ref"`
	PersonaRef  string `json:"persona_ref"`
	PerfilRef   string `json:"perfil_ref"`
}

func cargarSegundoTecnicoRRHHDesarrollo(
	directorio string, ca *x509.Certificate, original identidadCertificadoDesarrollo,
) (*identidadCertificadoDesarrollo, error) {
	ruta := filepath.Join(directorio, "identidad", nombreSegundoTecnicoRRHHDesarrollo)
	if _, err := os.Lstat(ruta); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil || ca == nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, 16<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	var archivo archivoSegundoTecnicoRRHHDesarrollo
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if decodificador.Decode(&archivo) != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	var sobra any
	if !errors.Is(decodificador.Decode(&sobra), io.EOF) || archivo.Version != 1 ||
		archivo.Autoridad != AutoridadNoAutoritativa || archivo.Sujeto == "" ||
		archivo.Sujeto == original.principal.ID ||
		archivo.Certificado != certificadoSegundoTecnicoRRHHDesarrollo ||
		archivo.Identidad != identidadSegundoTecnicoRRHHDesarrollo {
		return nil, ErrMaterialDesarrolloInvalido
	}
	rutaCert, okCert := rutaMaterialConsultaRRHHDesarrollo(directorio, archivo.Certificado)
	rutaIdentidad, okIdentidad := rutaMaterialConsultaRRHHDesarrollo(directorio, archivo.Identidad)
	if !okCert || !okIdentidad || rutaCert == rutaIdentidad {
		return nil, ErrMaterialDesarrolloInvalido
	}
	pem, err := leerFicheroMaterialSeguro(rutaCert, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	certificado, err := decodificarCertificadoUnico(pem)
	if err != nil || certificado == nil || certificado.IsCA ||
		certificado.NotAfter.Sub(certificado.NotBefore) > 90*24*time.Hour ||
		certificado.NotAfter.After(time.Now().UTC().Add(90*24*time.Hour)) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	if _, err := certificado.Verify(x509.VerifyOptions{
		Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	identidad, err := cargarIdentidadDesarrollo(rutaIdentidad, certificado, "tecnico_rrhh")
	if err != nil || identidad.principal.ID != archivo.Sujeto || identidad.huella == original.huella {
		return nil, ErrMaterialDesarrolloInvalido
	}
	base := archivo.Sujeto + "\x00" + identidad.principal.Attributes["certificate_sha256"]
	if archivo.CuentaRef != referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta") ||
		archivo.PersonaRef != referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona") ||
		archivo.PerfilRef != referenciaAltaContratacionTemporalDesarrollo("prf_", base+"\x00perfil") ||
		archivo.CuentaRef == referenciaAltaContratacionTemporalDesarrollo("cta_", original.principal.ID+"\x00"+original.principal.Attributes["certificate_sha256"]+"\x00cuenta") ||
		archivo.PersonaRef == referenciaAltaContratacionTemporalDesarrollo("per_", original.principal.ID+"\x00"+original.principal.Attributes["certificate_sha256"]+"\x00persona") {
		return nil, ErrMaterialDesarrolloInvalido
	}
	return &identidad, nil
}

func segundoTecnicoRRHHDistintoDeIdentidades(identidad identidadCertificadoDesarrollo, existentes []identidadCertificadoDesarrollo) bool {
	for _, existente := range existentes {
		if identidad.huella == existente.huella || identidad.principal.ID == existente.principal.ID {
			return false
		}
	}
	return true
}
