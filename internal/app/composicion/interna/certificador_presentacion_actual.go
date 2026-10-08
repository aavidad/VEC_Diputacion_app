package interna

import (
	"context"
	"crypto/x509"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadcertificado"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

// El certificador C4 usa el mismo Registro que el extractor TLS, después de
// la aserción y con la cadena verificada de esta petición.
type certificadorPresentacionPersonal struct {
	registro *registroCertificadosPersonales
}

var _ httpseguridad.CertificadorPresentacionActual = (*certificadorPresentacionPersonal)(nil)

// NuevoCertificadorPresentacionPersonal fija para C4 el Registro que también
// recibió el extractor TLS; el llamador no aporta identidad ni garantía.
func NuevoCertificadorPresentacionPersonal(
	registro *identidadcertificado.Registro,
) (httpseguridad.CertificadorPresentacionActual, error) {
	if registro == nil {
		return nil, ErrCertificadoPersonalNoDisponible
	}
	return &certificadorPresentacionPersonal{registro: registro}, nil
}

func (c *certificadorPresentacionPersonal) AcreditarCertificadoActual(ctx context.Context,
	cadena []*x509.Certificate, ahora time.Time,
) (httpseguridad.AcreditacionCertificadoActual, error) {
	if c == nil || c.registro == nil {
		return httpseguridad.AcreditacionCertificadoActual{}, ErrCertificadoPersonalNoDisponible
	}
	actual, err := c.registro.AcreditarCadenaActual(ctx, cadena, ahora)
	if err != nil {
		return httpseguridad.AcreditacionCertificadoActual{}, ErrCertificadoPersonalNoDisponible
	}
	return httpseguridad.AcreditacionCertificadoActual{
		SujetoID:               actual.SujetoID,
		CuentaID:               actual.CuentaID,
		CertificadoSHA256:      actual.CertificadoSHA256,
		CASHA256:               actual.CASHA256,
		CertificadoValidoHasta: actual.CertificadoValidoHasta,
		CAValidaHasta:          actual.CAValidaHasta,
		CRLSiguienteEn:         actual.CRLSiguienteEn,
		RevocacionVerificadaEn: actual.RevocacionVerificadaEn,
	}, nil
}
