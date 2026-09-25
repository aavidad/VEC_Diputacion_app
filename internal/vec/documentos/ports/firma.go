package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"vec-diputacion-granada/internal/vec/documentos/domain"
)

var ErrVerificacionFirmaInvalida = errors.New("documentos: verificacion de firma invalida")

type EstadoVerificacionFirma string

const (
	EstadoVerificacionValida        EstadoVerificacionFirma = "valida"
	EstadoVerificacionNoValida      EstadoVerificacionFirma = "no_valida"
	EstadoVerificacionIndeterminada EstadoVerificacionFirma = "indeterminada"
)

type SolicitudVerificacionFirma struct {
	DocumentoID          string
	Version              uint64
	HuellaOriginalSHA256 string
	ContenidoOriginal    []byte
	ContenidoFirmado     []byte
}

func (s SolicitudVerificacionFirma) Validar() error {
	if !domain.ReferenciaOpacaValida(s.DocumentoID) || s.Version == 0 ||
		!domain.HuellaValida(s.HuellaOriginalSHA256) || len(s.ContenidoOriginal) == 0 ||
		len(s.ContenidoOriginal) > 16<<20 || len(s.ContenidoFirmado) == 0 ||
		len(s.ContenidoFirmado) > 16<<20 {
		return ErrVerificacionFirmaInvalida
	}
	suma := sha256.Sum256(s.ContenidoOriginal)
	if hex.EncodeToString(suma[:]) != s.HuellaOriginalSHA256 {
		return ErrVerificacionFirmaInvalida
	}
	return nil
}

type ResultadoVerificacionFirma struct {
	Estado                  EstadoVerificacionFirma
	VinculoOriginal         bool
	HuellaOriginalSHA256    string
	HuellaFirmadoSHA256     string
	FirmanteRef             string
	CertificadoHuellaSHA256 string
	SelloTiempoEstado       string
	RevocacionEstado        string
}

// ValidarContra solo acepta un resultado positivo y ligado a ambos contenidos.
// Un estado indeterminado, un eco de huella o revocacion dudosa no firma nada.
func (r ResultadoVerificacionFirma) ValidarContra(s SolicitudVerificacionFirma) error {
	if s.Validar() != nil || r.Estado != EstadoVerificacionValida || !r.VinculoOriginal ||
		r.HuellaOriginalSHA256 != s.HuellaOriginalSHA256 ||
		!domain.ReferenciaOpacaValida(r.FirmanteRef) ||
		!domain.HuellaValida(r.CertificadoHuellaSHA256) ||
		r.SelloTiempoEstado != "valido" || r.RevocacionEstado != "vigente" {
		return ErrVerificacionFirmaInvalida
	}
	suma := sha256.Sum256(s.ContenidoFirmado)
	if r.HuellaFirmadoSHA256 != hex.EncodeToString(suma[:]) {
		return ErrVerificacionFirmaInvalida
	}
	return nil
}

type VerificadorFirma interface {
	Verificar(context.Context, SolicitudVerificacionFirma) (ResultadoVerificacionFirma, error)
}

// ProveedorFirma solo declara el estado que acredita su propia fuente.
// Un proveedor ausente conserva el documento pendiente y nunca fabrica firma.
type ProveedorFirma interface {
	EstadoDeclarado(context.Context, string, uint64) (string, error)
}

type SinProveedorFirma struct{}

func (SinProveedorFirma) EstadoDeclarado(context.Context, string, uint64) (string, error) {
	return domain.EstadoFirmaPendienteProveedor, nil
}
