package bootstrap

import (
	"errors"
	"net"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

var ErrComposicionFirmaVerificacionNoDisponible = errors.New("bootstrap: verificacion de firma no disponible")

// nuevoVerificadorFirmaDocumentos compone exclusivamente el puerto interno.
// La presencia del cliente no publica rutas ni habilita una transicion a firmado.
func nuevoVerificadorFirmaDocumentos(cfg config.Config) (docports.VerificadorFirmaMotivado, error) {
	cfg = cfg.Normalize()
	switch cfg.FirmaVerificacionEnabled {
	case "", "false":
		return nil, nil
	case "true":
		activo, err := cfg.DocumentosDesarrolloActivo()
		if err != nil || !activo {
			return nil, ErrComposicionFirmaVerificacionNoDisponible
		}
	default:
		return nil, ErrComposicionFirmaVerificacionNoDisponible
	}
	if cfg.FirmaVerificacionURL == "" || cfg.FirmaVerificacionCAFile == "" || cfg.FirmaVerificacionTimeout == "" ||
		(cfg.FirmaVerificacionTokenFile == "" && (cfg.FirmaVerificacionCertFile == "" || cfg.FirmaVerificacionKeyFile == "")) ||
		(cfg.FirmaVerificacionCertFile == "") != (cfg.FirmaVerificacionKeyFile == "") {
		return nil, ErrComposicionFirmaVerificacionNoDisponible
	}
	if !nombreServidorTLSValido(cfg.FirmaVerificacionNombreServidorTLS) {
		return nil, ErrComposicionFirmaVerificacionNoDisponible
	}
	plazo, err := time.ParseDuration(cfg.FirmaVerificacionTimeout)
	if err != nil || plazo < time.Millisecond || plazo > 60*time.Second {
		return nil, ErrComposicionFirmaVerificacionNoDisponible
	}
	ca, err := leerFicheroMaterialSeguro(cfg.FirmaVerificacionCAFile, 64<<10)
	if err != nil {
		return nil, ErrComposicionFirmaVerificacionNoDisponible
	}
	defer borrarBytes(ca)
	var token, certificado, clave []byte
	if cfg.FirmaVerificacionTokenFile != "" {
		token, err = leerFicheroMaterialSeguro(cfg.FirmaVerificacionTokenFile, 512)
		if err != nil {
			return nil, ErrComposicionFirmaVerificacionNoDisponible
		}
		defer borrarBytes(token)
	}
	if cfg.FirmaVerificacionCertFile != "" {
		certificado, err = leerFicheroMaterialSeguro(cfg.FirmaVerificacionCertFile, 64<<10)
		if err != nil {
			return nil, ErrComposicionFirmaVerificacionNoDisponible
		}
		defer borrarBytes(certificado)
		clave, err = leerFicheroMaterialSeguro(cfg.FirmaVerificacionKeyFile, 64<<10)
		if err != nil {
			return nil, ErrComposicionFirmaVerificacionNoDisponible
		}
		defer borrarBytes(clave)
	}
	cliente, err := validadorautofirma.Nuevo(validadorautofirma.Configuracion{
		URL: cfg.FirmaVerificacionURL, CAPEM: ca, NombreServidorTLS: cfg.FirmaVerificacionNombreServidorTLS, Token: token,
		CertificadoClientePEM: certificado, ClaveClientePEM: clave, Timeout: plazo,
	})
	if err != nil {
		return nil, ErrComposicionFirmaVerificacionNoDisponible
	}
	return cliente, nil
}

// nombreServidorTLSValido admite vacio (se verifica el host de la URL), un
// nombre DNS sin punto final ni puerto o una direccion IP literal.
func nombreServidorTLSValido(nombre string) bool {
	if nombre == "" {
		return true
	}
	if len(nombre) > 253 {
		return false
	}
	if net.ParseIP(nombre) != nil {
		return true
	}
	for _, etiqueta := range strings.Split(nombre, ".") {
		if etiqueta == "" || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for i := 0; i < len(etiqueta); i++ {
			c := etiqueta[i]
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
