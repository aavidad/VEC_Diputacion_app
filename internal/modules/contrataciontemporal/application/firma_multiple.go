package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

// PoliticaVerificacionFirmaMultipleV2 conserva las comprobaciones autónomas
// de v1 y exige revisiones incrementales de aprobación, sin cambios de
// contenido ni bytes posteriores sin firmar. No admite certificación DocMDP
// ni sellos de documento como pasos personales. Su persistencia requiere el
// contrato v2; no se puede registrar como evidencia de la política v1.
const PoliticaVerificacionFirmaMultipleV2 = "politica:vec:firma:verificacion-autonoma:v2"

// AntecedenteFirmaMultiple procede de la historia autorizada y de la custodia
// de Documentos. El consumidor conserva la autorización de ambas lecturas y
// selecciona exclusivamente los recibos activos de la misma ronda.
type AntecedenteFirmaMultiple struct {
	Firma      ports.FirmaRegistrada
	PDFFirmado []byte
}

// CanonicoEvidenciaFirmasMultiple produce el array estable de la evidencia
// validada. Conserva el orden y los campos de cada firma; ComprobadoEn no
// pertenece a este array ni a la idempotencia. El material v2 conserva aparte
// la política, el original y el resultado global del dictamen.
func CanonicoEvidenciaFirmasMultiple(
	s docports.SolicitudVerificacionFirma, v docports.VerificacionFirmasDocumento,
	documento string, pasoOrden int, anteriores []AntecedenteFirmaMultiple,
) ([]byte, error) {
	if _, err := ValidarFirmaMultipleParaPaso(s, v, documento, pasoOrden, anteriores); err != nil {
		return nil, err
	}
	return json.Marshal(v.Firmas)
}

// ValidarFirmaMultipleParaPaso verifica toda la cadena y devuelve únicamente
// la firma nueva del paso exacto. No acredita competencia ni registra efectos:
// el consumidor debe consultar la autoridad central y revalidar al confirmar.
func ValidarFirmaMultipleParaPaso(
	s docports.SolicitudVerificacionFirma, v docports.VerificacionFirmasDocumento,
	documento string, pasoOrden int, anteriores []AntecedenteFirmaMultiple,
) (docports.FirmaPDFVerificada, error) {
	var cero docports.FirmaPDFVerificada
	if s.Validar() != nil || s.FormatoEsperado != "PAdES" ||
		!domain.ClaveDocumentoFirmaValida(documento) || pasoOrden < 1 || pasoOrden > domain.MaximoPasosCircuitoFirma ||
		len(v.Firmas) != pasoOrden || len(anteriores) != pasoOrden-1 ||
		len(s.ContenidoOriginal) > ports.MaximoDocumentoFirmaBytes || len(s.ContenidoFirmado) > ports.MaximoDocumentoFirmaBytes ||
		v.Estado != docports.EstadoVerificacionValida || v.Motivo != docports.MotivoFirmaVerificada ||
		v.Formato != "PAdES" || !v.VinculoOriginal || v.ComprobadoEn.IsZero() ||
		v.HuellaOriginalSHA256 != s.HuellaOriginalSHA256 || v.HuellaFirmadoSHA256 != huella(s.ContenidoFirmado) ||
		len(s.ContenidoFirmado) <= len(s.ContenidoOriginal) || !bytes.HasPrefix(s.ContenidoFirmado, s.ContenidoOriginal) ||
		v.CambiosPosteriores.Estado != "ninguno" || len(v.CambiosPosteriores.Detalle) != 0 ||
		v.CambiosPosteriores.BytesNoFirmados == nil || *v.CambiosPosteriores.BytesNoFirmados != 0 {
		return cero, ErrFirmaNoVerificada
	}
	longitudAnterior := uint64(len(s.ContenidoOriginal))
	for i, firma := range v.Firmas {
		if !firmaMultipleValida(s.ContenidoFirmado, firma, i+1, longitudAnterior) {
			return cero, ErrFirmaNoVerificada
		}
		if i < len(anteriores) && !antecedenteFirmaMultipleValido(s, documento, firma, anteriores[i], anteriores[:i]) {
			return cero, ports.ErrCadenaFirmaDocumentoRota
		}
		longitudAnterior = firma.RevisionLongitud
	}
	if longitudAnterior != uint64(len(s.ContenidoFirmado)) {
		return cero, ErrFirmaNoVerificada
	}
	seleccionada := v.Firmas[pasoOrden-1]
	seleccionada.CambiosDesdeAnterior.Detalle = append([]string(nil), seleccionada.CambiosDesdeAnterior.Detalle...)
	return seleccionada, nil
}

func firmaMultipleValida(pdf []byte, f docports.FirmaPDFVerificada, orden int, longitudAnterior uint64) bool {
	br := f.ByteRange
	longitud := uint64(len(pdf))
	if f.Orden != orden || !f.CubreDocumentoCompletoHastaAqui || f.TipoFirma != "aprobacion" || f.NivelDocMDP != nil ||
		f.IntegridadEstado != "valida" || f.CadenaEstado != "valida" || f.CertificadoEstado != "vigente" ||
		f.RevocacionEstado != docports.RevocacionVigente || !docports.SelloTiempoAdmisible(f.SelloTiempoEstado) ||
		!domain.ReferenciaOpacaValida(f.FirmanteRef) || !domain.HuellaSHA256FirmaValida(f.CertificadoHuellaSHA256) ||
		!domain.HuellaSHA256FirmaValida(f.RevisionHuellaSHA256) || !domain.HuellaSHA256FirmaValida(f.ContenidoFirmadoHuellaSHA256) ||
		f.CambiosDesdeAnterior.Estado != "permitidos" || len(f.CambiosDesdeAnterior.Detalle) != 1 ||
		f.CambiosDesdeAnterior.Detalle[0] != "firma_anadida" ||
		br[0] != 0 || br[1] < longitudAnterior || br[1] >= br[2] || br[2] > longitud ||
		br[3] == 0 || br[3] > longitud-br[2] || f.RevisionLongitud != br[2]+br[3] ||
		f.RevisionLongitud <= longitudAnterior {
		return false
	}
	if huella(pdf[:f.RevisionLongitud]) != f.RevisionHuellaSHA256 {
		return false
	}
	h := sha256.New()
	_, _ = h.Write(pdf[:br[1]])
	_, _ = h.Write(pdf[br[2]:f.RevisionLongitud])
	return hex.EncodeToString(h.Sum(nil)) == f.ContenidoFirmadoHuellaSHA256
}

func antecedenteFirmaMultipleValido(
	s docports.SolicitudVerificacionFirma, documento string, firma docports.FirmaPDFVerificada,
	a AntecedenteFirmaMultiple, previos []AntecedenteFirmaMultiple,
) bool {
	r := a.Firma
	if r.Documento != documento || r.PasoOrden != firma.Orden || r.Resultado != domain.ResultadoFirmaFirmado ||
		(r.Via != ports.ViaFirmaCertificadoVEC && r.Via != ports.ViaFirmaExternaPortafirmas) ||
		!domain.ReferenciaOpacaValida(r.FirmaRef) || !domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!r.FirmantePrincipalAcreditado || !domain.ReferenciaOpacaValida(r.DocumentoCustodiaRef) || r.DocumentoCustodiaVersion == 0 ||
		r.OriginalRef != s.DocumentoID || r.OriginalVersion != s.Version || r.OriginalHuella != s.HuellaOriginalSHA256 ||
		r.FirmanteRef != firma.FirmanteRef || r.CertificadoHuella != firma.CertificadoHuellaSHA256 ||
		r.FirmadoHuella != firma.RevisionHuellaSHA256 || uint64(len(a.PDFFirmado)) != firma.RevisionLongitud ||
		!bytes.HasPrefix(s.ContenidoFirmado, a.PDFFirmado) || huella(a.PDFFirmado) != r.FirmadoHuella {
		return false
	}
	for _, previo := range previos {
		if previo.Firma.FirmaRef == r.FirmaRef || previo.Firma.ReciboRef == r.ReciboRef ||
			(previo.Firma.DocumentoCustodiaRef == r.DocumentoCustodiaRef && previo.Firma.DocumentoCustodiaVersion == r.DocumentoCustodiaVersion) {
			return false
		}
	}
	return true
}
