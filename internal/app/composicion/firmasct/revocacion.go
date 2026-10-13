package firmasct

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"time"
)

var ErrCRLFirmaNoDisponible = errors.New("composicion CT: CRL no disponible")

// LeerCRLFirmaR5 conserva el formato estricto del comprobador de certificado
// personal. El adaptador lee el fichero privado con sus guardas originales.
func LeerCRLFirmaR5(contenido []byte) (*x509.RevocationList, error) {
	bloque, resto := pem.Decode(contenido)
	if bloque == nil || bloque.Type != "X509 CRL" || len(bloque.Headers) != 0 || len(bytes.TrimSpace(resto)) != 0 {
		return nil, ErrCRLFirmaNoDisponible
	}
	lista, err := x509.ParseRevocationList(bloque.Bytes)
	if err != nil || lista == nil || lista.ThisUpdate.IsZero() || lista.NextUpdate.IsZero() {
		return nil, ErrCRLFirmaNoDisponible
	}
	return lista, nil
}

// ComprobarCRLFirmaR5 es el mismo cotejo de vigencia, emisor, firma y revocación
// de la composición interna; no interpreta protección de clave ni un AAL.
func ComprobarCRLFirmaR5(lista *x509.RevocationList, certificado, autoridad *x509.Certificate, ahora time.Time) error {
	if certificado == nil || autoridad == nil || ahora.Before(certificado.NotBefore) || !ahora.Before(certificado.NotAfter) ||
		ahora.Before(autoridad.NotBefore) || !ahora.Before(autoridad.NotAfter) {
		return ErrCRLFirmaNoDisponible
	}
	if lista == nil || !bytes.Equal(lista.RawIssuer, autoridad.RawSubject) || lista.CheckSignatureFrom(autoridad) != nil ||
		ahora.Before(lista.ThisUpdate) || !ahora.Before(lista.NextUpdate) {
		return ErrCRLFirmaNoDisponible
	}
	for _, entrada := range lista.RevokedCertificateEntries {
		if entrada.SerialNumber != nil && entrada.SerialNumber.Cmp(certificado.SerialNumber) == 0 {
			return ErrCRLFirmaNoDisponible
		}
	}
	return nil
}
