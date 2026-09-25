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

// PoliticaVerificacionFirmaV1 identifica la politica de verificacion de firma
// que aplica VEC. Cualquier cambio de sus reglas exige una version nueva.
//
// Reglas de la v1 (verificacion autonoma, sin depender de otros servicios):
//   - Confianza: solo anclas locales fijadas por despliegue; el almacen del
//     sistema no cuenta.
//   - Revocacion: solo `vigente` permite `valida`, con evidencia local (CRL
//     del directorio montado o embebida en la firma). OCSP y consultas
//     remotas son extensiones opcionales, nunca requisito.
//   - Sello de tiempo: opcional. `no_presente` y `no_comprobado` no bloquean
//     (la validez se evalua en el instante de la comprobacion y VEC no se
//     apoya en el sello como prueba de tiempo); un sello presente y
//     `no_valido` deja la verificacion indeterminada.
//   - Vinculo: la firma debe cubrir el original custodiado por VEC
//     (`acreditado`); no basta con que no se aportara.
//   - Un unico firmante identificado por la huella SHA-256 de su certificado.
const PoliticaVerificacionFirmaV1 = "politica:vec:firma:verificacion-autonoma:v1"

// Estados cerrados de revocacion y sello de tiempo del resultado.
const (
	RevocacionVigente      = "vigente"
	RevocacionRevocado     = "revocado"
	RevocacionNoComprobada = "no_comprobada"

	SelloTiempoNoPresente   = "no_presente"
	SelloTiempoValido       = "valido"
	SelloTiempoNoValido     = "no_valido"
	SelloTiempoNoComprobado = "no_comprobado"
)

// SelloTiempoAdmisible aplica la regla de sello de PoliticaVerificacionFirmaV1:
// el sello es opcional, pero uno presente y no valido impide acreditar.
func SelloTiempoAdmisible(estado string) bool {
	return estado == SelloTiempoNoPresente || estado == SelloTiempoValido || estado == SelloTiempoNoComprobado
}

// ValidarContra solo acepta un resultado positivo y ligado a ambos contenidos,
// conforme a PoliticaVerificacionFirmaV1. Un estado indeterminado, un eco de
// huella, una revocacion no vigente o un sello no valido no firman nada.
func (r ResultadoVerificacionFirma) ValidarContra(s SolicitudVerificacionFirma) error {
	if s.Validar() != nil || r.Estado != EstadoVerificacionValida || !r.VinculoOriginal ||
		r.HuellaOriginalSHA256 != s.HuellaOriginalSHA256 ||
		!domain.ReferenciaOpacaValida(r.FirmanteRef) ||
		!domain.HuellaValida(r.CertificadoHuellaSHA256) ||
		!SelloTiempoAdmisible(r.SelloTiempoEstado) || r.RevocacionEstado != RevocacionVigente {
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
