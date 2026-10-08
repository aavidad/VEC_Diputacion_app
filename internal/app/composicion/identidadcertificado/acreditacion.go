package identidadcertificado

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"time"
)

func CertificadoAdmisible(c *x509.Certificate) bool {
	if c == nil || c.IsCA || c.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return false
	}
	for _, uso := range c.ExtKeyUsage {
		if uso == x509.ExtKeyUsageClientAuth {
			return true
		}
	}
	return false
}

// AcreditarCadenaActual relee la CRL y el registro una sola vez después de la
// aserción. La cadena debe proceder de VerifiedChains del TLS de esta petición.
func (r *Registro) AcreditarCadenaActual(ctx context.Context,
	cadena []*x509.Certificate, ahora time.Time,
) (AcreditacionActual, error) {
	var vacia AcreditacionActual
	if r == nil || ctx == nil || ctx.Err() != nil || len(cadena) < 2 ||
		ahora.IsZero() || ahora.Location() != time.UTC {
		return vacia, ErrNoDisponible
	}
	certificado, ca := cadena[0], cadena[1]
	if !CertificadoAdmisible(certificado) || certificado.SerialNumber == nil || ca == nil ||
		len(certificado.Raw) == 0 || len(ca.Raw) == 0 {
		return vacia, ErrNoDisponible
	}
	caHasta := ca.NotAfter.UTC().Truncate(time.Microsecond)
	for i, actual := range cadena {
		if actual == nil || len(actual.Raw) == 0 || ahora.Before(actual.NotBefore) ||
			!ahora.Before(actual.NotAfter) {
			return vacia, ErrNoDisponible
		}
		if i > 0 && cadena[i-1].CheckSignatureFrom(actual) != nil {
			return vacia, ErrNoDisponible
		}
		if i > 0 && actual.NotAfter.Before(caHasta) {
			caHasta = actual.NotAfter.UTC().Truncate(time.Microsecond)
		}
	}
	lista, err := r.LeerCRLActual()
	if err != nil || !bytes.Equal(lista.RawIssuer, ca.RawSubject) ||
		lista.CheckSignatureFrom(ca) != nil || ahora.Before(lista.ThisUpdate) ||
		!ahora.Before(lista.NextUpdate) {
		return vacia, ErrNoDisponible
	}
	for _, entrada := range lista.RevokedCertificateEntries {
		if entrada.SerialNumber != nil && entrada.SerialNumber.Cmp(certificado.SerialNumber) == 0 {
			return vacia, ErrNoDisponible
		}
	}
	huellaCertificado := sha256.Sum256(certificado.Raw)
	huella := "sha256:" + hex.EncodeToString(huellaCertificado[:])
	// Una sola lectura del registro dentro de esta acreditación. Resolver hace
	// su propia lectura para otros consumidores; aquí se usa el mapa ya leído.
	todos, err := r.leer()
	if err != nil || ctx.Err() != nil {
		return vacia, ErrNoDisponible
	}
	registro, existe := todos[huella]
	if !existe || !registro.Activo {
		return vacia, ErrNoDisponible
	}
	huellaCA := sha256.Sum256(ca.Raw)
	return AcreditacionActual{
		SujetoID:               registro.SujetoID,
		CuentaID:               registro.CuentaID,
		CertificadoSHA256:      registro.HuellaSHA256,
		CASHA256:               "sha256:" + hex.EncodeToString(huellaCA[:]),
		CertificadoValidoHasta: certificado.NotAfter.UTC().Truncate(time.Microsecond),
		CAValidaHasta:          caHasta,
		CRLSiguienteEn:         lista.NextUpdate.UTC().Truncate(time.Microsecond),
		RevocacionVerificadaEn: ahora,
	}, nil
}
