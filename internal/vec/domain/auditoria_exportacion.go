package domain

import (
	"encoding/json"
	"strings"
	"time"
)

const EsquemaExportacionAuditoriaDesarrollo = "vec.auditoria.exportacion.desarrollo.v1"
const DominioFirmaExportacionAuditoriaDesarrollo = "vec.kms.desarrollo.auditoria.exportacion.ed25519.v1"

// CapturaExportacionAuditoria identifica el recibo confirmado por la autoridad
// común. Estos campos no autentican por sí mismos la procedencia de un archivo.
type CapturaExportacionAuditoria struct {
	Referencia      string `json:"referencia"`
	AuditoriaRef    string `json:"auditoria_ref"`
	AuditoriaSHA256 string `json:"auditoria_sha256"`
	CapturadaEn     string `json:"capturada_en"`
}

func (c CapturaExportacionAuditoria) Validar() error {
	t, err := time.Parse(time.RFC3339Nano, c.CapturadaEn)
	if !codigoCheckpoint.MatchString(c.Referencia) || !codigoCheckpoint.MatchString(c.AuditoriaRef) || !SHA256CheckpointValido(c.AuditoriaSHA256) || err != nil || t.Year() < 1 || t.Year() > 9999 || t.UTC().Format("2006-01-02T15:04:05.000000Z") != c.CapturadaEn {
		return ErrCheckpointInvalido
	}
	return nil
}

type DocumentoExportacionAuditoria struct {
	Esquema string `json:"esquema"`
	Bytes   int64  `json:"bytes"`
	SHA256  string `json:"sha256"`
}

type ManifiestoExportacionAuditoriaDesarrollo struct {
	Esquema                  string                        `json:"esquema"`
	Politica                 PoliticaCheckpoint            `json:"politica"`
	Captura                  CapturaExportacionAuditoria   `json:"captura"`
	Documento                DocumentoExportacionAuditoria `json:"documento"`
	Cobertura                CoberturaCheckpoint           `json:"cobertura"`
	HistoricosSinFechaLigada bool                          `json:"historicos_sin_fecha_ligada"`
}

func (m ManifiestoExportacionAuditoriaDesarrollo) Canonico() ([]byte, error) {
	if m.Esquema != EsquemaExportacionAuditoriaDesarrollo || m.Politica.Validar() != nil || m.Captura.Validar() != nil || m.Cobertura.Validar() != nil || !codigoCheckpoint.MatchString(m.Documento.Esquema) || m.Documento.Bytes < 1 || m.Documento.Bytes > 64*1024*1024 || !SHA256CheckpointValido(m.Documento.SHA256) {
		return nil, ErrCheckpointInvalido
	}
	return json.Marshal(m)
}

type ReciboExportacionAuditoriaDesarrollo struct {
	Manifiesto    ManifiestoExportacionAuditoriaDesarrollo `json:"manifiesto"`
	TSA           ReciboTSACheckpoint                      `json:"tsa"`
	PinSPKISHA256 string                                   `json:"pin_spki_sha256"`
	FirmaBase64   string                                   `json:"firma_base64"`
}

func (r ReciboExportacionAuditoriaDesarrollo) CanonicoParaFirma() ([]byte, error) {
	b, err := r.Manifiesto.Canonico()
	if err != nil {
		return nil, err
	}
	if !SHA256CheckpointValido(r.PinSPKISHA256) || r.TSA.HuellaCheckpointSHA256 != HuellaCheckpoint(b) || !SHA256CheckpointValido(r.TSA.HuellaPreimagenSHA256) || r.TSA.Autoridad != "no_autoritativo" || r.TSA.Esquema != "vec.tsa.desarrollo.v1" || !strings.HasPrefix(r.TSA.Referencia, "tsa-desarrollo:hmac-sha256:") || !SHA256CheckpointValido(strings.TrimPrefix(r.TSA.Referencia, "tsa-desarrollo:hmac-sha256:")) {
		return nil, ErrCheckpointInvalido
	}
	return json.Marshal(struct {
		Dominio    string                                   `json:"dominio"`
		Manifiesto ManifiestoExportacionAuditoriaDesarrollo `json:"manifiesto"`
		TSA        ReciboTSACheckpoint                      `json:"tsa"`
		Pin        string                                   `json:"pin_spki_sha256"`
	}{DominioFirmaExportacionAuditoriaDesarrollo, r.Manifiesto, r.TSA, r.PinSPKISHA256})
}
