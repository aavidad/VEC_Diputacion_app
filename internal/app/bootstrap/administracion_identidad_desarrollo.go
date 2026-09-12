package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// El material opcional lo prepara el integrador fuera de Git. No se crean
// certificados, cuentas ni permisos al arrancar y su ausencia cierra ADMIN.
type identidadAdministracionDesarrollo struct {
	identidad          identidadCertificadoDesarrollo
	cuentaRef          string
	cuentaOrdinariaRef string
	perfilRef          string
	personaRef         string
}

type archivoAdministracionDesarrollo struct {
	Version            int    `json:"version"`
	Autoridad          string `json:"autoridad"`
	Certificado        string `json:"certificado"`
	Identidad          string `json:"identidad"`
	CuentaRef          string `json:"cuenta_ref"`
	CuentaOrdinariaRef string `json:"cuenta_ordinaria_ref"`
	PerfilRef          string `json:"perfil_ref"`
	PersonaRef         string `json:"persona_ref"`
}

// El catálogo administrativo pertenece únicamente a su listener. No incorpora
// identidades RRHH ni comparte mapas mutables con el material que lo origina.
func nuevoResolvedorAdministracionDesarrollo(identidad *identidadAdministracionDesarrollo) (*resolvedorIdentidadDesarrollo, error) {
	if identidad == nil {
		return nil, nil
	}
	if !referenciasIdentidadAdministracionDesarrolloValidas(identidad) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	copia := *identidad
	copia.identidad.principal = clonarPrincipalDesarrollo(identidad.identidad.principal)
	principal := copia.identidad.principal
	if principal.AuthMethod != vecdomain.AuthMethodCertificate || principal.AuthAssurance != vecdomain.AuthAssuranceHigh ||
		len(principal.Roles) != 1 || principal.Roles[0] != "administrador" || len(principal.Permissions) != 0 {
		return nil, ErrMaterialDesarrolloInvalido
	}
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(copia.identidad)
	if err != nil {
		return nil, err
	}
	resolvedor.administracion = &copia
	return resolvedor, nil
}

func cargarIdentidadAdministracionDesarrollo(root string, ca *x509.Certificate, existentes ...identidadCertificadoDesarrollo) (*identidadAdministracionDesarrollo, error) {
	ruta := filepath.Join(root, "identidad", "administracion.json")
	if _, err := os.Lstat(ruta); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil || validarClavesJSONUnicas(contenido) != nil || ca == nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	var archivo archivoAdministracionDesarrollo
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&archivo) != nil || !errors.Is(dec.Decode(&struct{}{}), io.EOF) ||
		archivo.Version != 1 || archivo.Autoridad != AutoridadNoAutoritativa ||
		!rutaMaterialCentroSegura(root, archivo.Certificado) || !rutaMaterialCentroSegura(root, archivo.Identidad) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: archivo.CuentaRef, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	ordinaria := cuenta
	ordinaria.CuentaRef = archivo.CuentaOrdinariaRef
	if (vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: archivo.PerfilRef}).Validar() != nil ||
		ordinaria.Validar() != nil || archivo.CuentaRef == archivo.CuentaOrdinariaRef ||
		!referenciaPersonaAdministracionDesarrolloValida(archivo.PersonaRef) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	certPEM, err := leerFicheroMaterialSeguro(filepath.Join(root, archivo.Certificado), tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	cert, err := decodificarCertificadoUnico(certPEM)
	if err != nil || cert.IsCA {
		return nil, ErrMaterialDesarrolloInvalido
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		return nil, ErrMaterialDesarrolloInvalido
	}
	identidad, err := cargarIdentidadDesarrollo(filepath.Join(root, archivo.Identidad), cert, "administrador")
	if err != nil || identidad.huella == ([sha256.Size]byte{}) {
		return nil, ErrMaterialDesarrolloInvalido
	}
	for _, existente := range existentes {
		if existente.huella == identidad.huella || existente.principal.ID == identidad.principal.ID {
			return nil, ErrMaterialDesarrolloInvalido
		}
	}
	return &identidadAdministracionDesarrollo{identidad: identidad, cuentaRef: archivo.CuentaRef, cuentaOrdinariaRef: archivo.CuentaOrdinariaRef, perfilRef: archivo.PerfilRef, personaRef: archivo.PersonaRef}, nil
}

func referenciaPersonaAdministracionDesarrolloValida(ref string) bool {
	// Reutiliza el validador opaco de cuenta: mismo alfabeto y longitud,
	// cambiando únicamente el prefijo nominal esperado.
	if !strings.HasPrefix(ref, "per_") {
		return false
	}
	return (vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + strings.TrimPrefix(ref, "per_"), Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}).Validar() == nil
}

func referenciasIdentidadAdministracionDesarrolloValidas(identidad *identidadAdministracionDesarrollo) bool {
	if identidad == nil || identidad.cuentaRef == identidad.cuentaOrdinariaRef || !referenciaPersonaAdministracionDesarrolloValida(identidad.personaRef) {
		return false
	}
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: identidad.cuentaRef, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	ordinaria := cuenta
	ordinaria.CuentaRef = identidad.cuentaOrdinariaRef
	return ordinaria.Validar() == nil && (vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: identidad.perfilRef}).Validar() == nil
}
