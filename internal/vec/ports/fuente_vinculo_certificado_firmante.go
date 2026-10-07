package ports

import (
	"context"
	"errors"
)

var (
	ErrVinculoCertificadoFirmanteNoAcreditado = errors.New("vec: vinculo de certificado firmante no acreditado")
	ErrVinculoCertificadoFirmanteNoDisponible = errors.New("vec: fuente de certificado firmante no disponible")
)

// VinculoCertificadoFirmante procede exclusivamente de identidad central.
// No concede perfiles ni competencia: el consumidor revalida esa autoridad.
type VinculoCertificadoFirmante struct {
	CertificadoHuella    string
	PrincipalRef         string
	CuentaRef            string
	VinculoCredencialRef string
	Revision             uint64
	Huella               string
	Vigente              bool
}

// FuenteVinculoCertificadoFirmante identifica un certificado DER exacto.
// No admite nombres civiles, DN ni huellas SPKI como sustitutos.
type FuenteVinculoCertificadoFirmante interface {
	ResolverFirmantePorCertificado(context.Context, string) (VinculoCertificadoFirmante, error)
}
