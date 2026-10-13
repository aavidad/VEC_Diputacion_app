package interna

import (
	"crypto/x509"
	"path/filepath"
	"time"
)

// ComprobarCRLFirmaR5 reutiliza el comprobador de revocación del certificado
// personal. No interpreta registros de protección de clave ni atribuye un AAL.
func ComprobarCRLFirmaR5(ruta string, hoja, autoridad *x509.Certificate, ahora time.Time) error {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return ErrCertificadoPersonalNoDisponible
	}
	registro := registroCertificadosPersonales{rutaCRL: ruta}
	return registro.comprobarCRL(hoja, autoridad, ahora)
}
