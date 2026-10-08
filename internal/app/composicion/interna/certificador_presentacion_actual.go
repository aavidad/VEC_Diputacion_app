package interna

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

// El certificador C4 queda fijado al mismo registro revocable que el emisor
// de la aserción. El servicio común lo llama después de emitirla y le pasa
// únicamente la cadena verificada del handshake TLS actual.
type certificadorPresentacionPersonal struct {
	registro *registroCertificadosPersonales
}

var _ httpseguridad.CertificadorPresentacionActual = (*certificadorPresentacionPersonal)(nil)

func (c *certificadorPresentacionPersonal) AcreditarCertificadoActual(ctx context.Context,
	cadena []*x509.Certificate, ahora time.Time,
) (httpseguridad.AcreditacionCertificadoActual, error) {
	var vacia httpseguridad.AcreditacionCertificadoActual
	if c == nil || c.registro == nil || ctx == nil || ctx.Err() != nil || len(cadena) < 2 ||
		ahora.IsZero() || ahora.Location() != time.UTC {
		return vacia, ErrCertificadoPersonalNoDisponible
	}
	certificado, ca := cadena[0], cadena[1]
	if !certificadoPersonalAdmisible(certificado) || certificado.SerialNumber == nil || ca == nil ||
		len(certificado.Raw) == 0 || len(ca.Raw) == 0 {
		return vacia, ErrCertificadoPersonalNoDisponible
	}
	caHasta := ca.NotAfter.UTC().Truncate(time.Microsecond)
	for i, actual := range cadena {
		if actual == nil || len(actual.Raw) == 0 || ahora.Before(actual.NotBefore) ||
			!ahora.Before(actual.NotAfter) {
			return vacia, ErrCertificadoPersonalNoDisponible
		}
		if i > 0 && cadena[i-1].CheckSignatureFrom(actual) != nil {
			return vacia, ErrCertificadoPersonalNoDisponible
		}
		if i > 0 && actual.NotAfter.Before(caHasta) {
			caHasta = actual.NotAfter.UTC().Truncate(time.Microsecond)
		}
	}
	lista, err := c.registro.leerCRL()
	if err != nil || !bytes.Equal(lista.RawIssuer, ca.RawSubject) ||
		lista.CheckSignatureFrom(ca) != nil || ahora.Before(lista.ThisUpdate) ||
		!ahora.Before(lista.NextUpdate) {
		return vacia, ErrCertificadoPersonalNoDisponible
	}
	for _, entrada := range lista.RevokedCertificateEntries {
		if entrada.SerialNumber != nil && certificado.SerialNumber != nil &&
			entrada.SerialNumber.Cmp(certificado.SerialNumber) == 0 {
			return vacia, ErrCertificadoPersonalNoDisponible
		}
	}
	huellaCertificado := sha256.Sum256(certificado.Raw)
	registro, err := c.registro.resolver(ctx, "sha256:"+hex.EncodeToString(huellaCertificado[:]))
	if err != nil || ctx.Err() != nil {
		return vacia, ErrCertificadoPersonalNoDisponible
	}
	huellaCA := sha256.Sum256(ca.Raw)
	return httpseguridad.AcreditacionCertificadoActual{
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
