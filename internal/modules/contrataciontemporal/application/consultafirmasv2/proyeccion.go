package consultafirmasv2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// No se trunca una historia ni se presenta como completa una lectura excesiva.
const MaximoFilas = 128

func proyectar(m ports.MaterialConsultaFirmasR5V2, l ports.LecturaFirmasR5V2) (Resultado, error) {
	var cero Resultado
	if l.HistoriaRevision > 9007199254740991 || !domain.HuellaSHA256FirmaValida(l.HistoriaHuella) ||
		len(l.Firmas) > MaximoFilas || len(l.RevisionesPDF) > MaximoFilas {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	porFirma := make(map[string]ports.FirmaRegistrada, len(l.Firmas))
	porRecibo := make(map[string]bool, len(l.Firmas))
	for _, f := range l.Firmas {
		if !firmaValida(m, f) || porFirma[f.FirmaRef].FirmaRef != "" || porRecibo[f.ReciboRef] {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		porFirma[f.FirmaRef], porRecibo[f.ReciboRef] = f, true
	}
	porRevision := make(map[string]ports.FirmaRegistradaRevisionPDFV2, len(l.RevisionesPDF))
	for _, r := range l.RevisionesPDF {
		base, ok := porFirma[r.FirmaRef]
		if !ok || porRevision[r.FirmaRef].FirmaRef != "" || !revisionValida(base, r) {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		porRevision[r.FirmaRef] = r
	}
	for _, r := range l.RevisionesPDF {
		if r.OrdenFirmaPDF == 2 && !antecedenteValido(r, porRevision) {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
	}
	salida := Resultado{ExpedienteRef: m.ExpedienteRef, Documento: m.Documento, VersionExpediente: m.VersionExpediente,
		ExpedienteDocumentalRef: ports.ExpedienteDocumentalRef(m.OrganizacionRef, m.ExpedienteRef),
		HistoriaRevision:        l.HistoriaRevision, HistoriaHuella: l.HistoriaHuella, Firmas: make([]Firma, 0, len(l.Firmas))}
	for _, f := range l.Firmas {
		fila := Firma{FirmaRef: f.FirmaRef, ReciboRef: f.ReciboRef, PasoRef: f.PasoRef, Documento: f.Documento,
			RegistradaEn: f.RegistradaEn, Secuencia: f.Secuencia, PasoOrden: f.PasoOrden, VersionExpediente: f.ExpedienteVersion,
			Via: f.Via, Resultado: f.Resultado, CatalogoRef: f.CatalogoRef, CatalogoHuella: f.CatalogoHuella,
			Original: Documento{f.OriginalRef, f.OriginalVersion, f.OriginalHuella}}
		if f.DocumentoCustodiaRef != "" {
			fila.Custodiado = &Documento{f.DocumentoCustodiaRef, f.DocumentoCustodiaVersion, f.FirmadoHuella}
		}
		if r, ok := porRevision[f.FirmaRef]; ok {
			fila.RevisionPDF = &RevisionPDF{OrdenFirma: r.OrdenFirmaPDF, FirmaAnteriorRef: r.FirmaAnteriorRef,
				ReciboAnteriorRef: r.ReciboAnteriorRef, Entrada: Documento{r.EntradaDocumentoRef, r.EntradaDocumentoVersion, r.EntradaDocumentoHuella},
				EntradaLongitud: r.EntradaDocumentoLongitud, ByteRange: r.ByteRange, RevisionSHA256: r.RevisionHuellaSHA256,
				ContenidoFirmadoSHA256: r.ContenidoFirmadoHuellaSHA256, RevisionLongitud: r.RevisionLongitud,
				EvidenciaFirmasSHA256: r.EvidenciaFirmasHuellaSHA256}
		}
		salida.Firmas = append(salida.Firmas, fila)
	}
	// Ni la colección vacía ni la cabeza implican una causa de ausencia.
	return salida, nil
}

func firmaValida(m ports.MaterialConsultaFirmasR5V2, f ports.FirmaRegistrada) bool {
	if f.Documento != m.Documento || f.ExpedienteVersion < 1 || f.ExpedienteVersion > m.VersionExpediente ||
		!domain.ReferenciaOpacaValida(f.FirmaRef) || !domain.ReferenciaOpacaValida(f.ReciboRef) ||
		!domain.ReferenciaOpacaValida(f.PasoRef) || !domain.ReferenciaOpacaValida(f.CatalogoRef) ||
		!domain.HuellaSHA256FirmaValida(f.CatalogoHuella) ||
		f.PasoOrden < 1 || f.PasoOrden > domain.MaximoPasosCircuitoFirma || f.Secuencia < 1 || f.Secuencia > 100000 ||
		!domain.InstanteUTCCanonico(f.RegistradaEn) ||
		(f.Via != "" && f.Via != ports.ViaFirmaCertificadoVEC && f.Via != ports.ViaFirmaExternaPortafirmas) ||
		(f.Resultado != domain.ResultadoFirmaFirmado && f.Resultado != domain.ResultadoFirmaDevuelto) ||
		(f.DocumentoCustodiaRef == "") != (f.DocumentoCustodiaVersion == 0) {
		return false
	}
	if f.DocumentoCustodiaRef != "" && (!domain.ReferenciaOpacaValida(f.DocumentoCustodiaRef) ||
		f.DocumentoCustodiaVersion > 9007199254740991 || f.Resultado != domain.ResultadoFirmaFirmado ||
		!domain.HuellaSHA256FirmaValida(f.FirmadoHuella)) {
		return false
	}
	if f.Via != "" && (f.Resultado != domain.ResultadoFirmaFirmado || f.PasoOrden > 2 ||
		!domain.ReferenciaOpacaValida(f.OriginalRef) || f.OriginalVersion < 1 || f.OriginalVersion > 9007199254740991 ||
		!domain.HuellaSHA256FirmaValida(f.OriginalHuella) || !domain.HuellaSHA256FirmaValida(f.FirmadoHuella)) {
		return false
	}
	return true
}

func revisionValida(f ports.FirmaRegistrada, r ports.FirmaRegistradaRevisionPDFV2) bool {
	if r.ReciboRef != f.ReciboRef || r.Documento != f.Documento || r.Secuencia != f.Secuencia ||
		r.ExpedienteVersion != f.ExpedienteVersion || r.PasoOrden != f.PasoOrden || r.PasoRef != f.PasoRef ||
		r.CatalogoRef != f.CatalogoRef || r.CatalogoHuella != f.CatalogoHuella || !r.RegistradaEn.Equal(f.RegistradaEn) ||
		r.OriginalRef != f.OriginalRef || r.OriginalVersion != f.OriginalVersion || r.OriginalHuella != f.OriginalHuella ||
		r.DocumentoCustodiaRef != f.DocumentoCustodiaRef || r.DocumentoCustodiaVersion != f.DocumentoCustodiaVersion ||
		r.FirmadoHuella != f.FirmadoHuella || r.Via != f.Via || f.Via == "" ||
		f.Resultado != domain.ResultadoFirmaFirmado || f.DocumentoCustodiaRef == "" ||
		r.OrdenFirmaPDF != f.PasoOrden || r.OrdenFirmaPDF < 1 || r.OrdenFirmaPDF > 2 ||
		!domain.ReferenciaOpacaValida(r.EntradaDocumentoRef) || r.EntradaDocumentoVersion < 1 || r.EntradaDocumentoVersion > 9007199254740991 ||
		!domain.HuellaSHA256FirmaValida(r.EntradaDocumentoHuella) || r.EntradaDocumentoLongitud < 1 ||
		r.EntradaDocumentoLongitud >= r.RevisionLongitud || r.RevisionLongitud > ports.MaximoDocumentoFirmaBytes ||
		r.ByteRange[0] != 0 || r.ByteRange[1] < r.EntradaDocumentoLongitud || r.ByteRange[2] <= r.ByteRange[1] ||
		r.ByteRange[2] > r.RevisionLongitud || r.ByteRange[3] == 0 || r.ByteRange[3] != r.RevisionLongitud-r.ByteRange[2] ||
		r.RevisionHuellaSHA256 != f.FirmadoHuella || !domain.HuellaSHA256FirmaValida(r.ContenidoFirmadoHuellaSHA256) ||
		!domain.HuellaSHA256FirmaValida(r.EvidenciaFirmasHuellaSHA256) || len(r.EvidenciaFirmasCanonica) < 2 || len(r.EvidenciaFirmasCanonica) > 32768 {
		return false
	}
	if r.OrdenFirmaPDF == 1 && (r.FirmaAnteriorRef != "" || r.ReciboAnteriorRef != "" ||
		r.EntradaDocumentoRef != f.OriginalRef || r.EntradaDocumentoVersion != f.OriginalVersion || r.EntradaDocumentoHuella != f.OriginalHuella) {
		return false
	}
	var compacto bytes.Buffer
	var evidencia []json.RawMessage
	h := sha256.Sum256(r.EvidenciaFirmasCanonica)
	return json.Compact(&compacto, r.EvidenciaFirmasCanonica) == nil && bytes.Equal(compacto.Bytes(), r.EvidenciaFirmasCanonica) &&
		json.Unmarshal(r.EvidenciaFirmasCanonica, &evidencia) == nil && len(evidencia) == r.OrdenFirmaPDF &&
		hex.EncodeToString(h[:]) == r.EvidenciaFirmasHuellaSHA256
}

func antecedenteValido(r ports.FirmaRegistradaRevisionPDFV2, revisiones map[string]ports.FirmaRegistradaRevisionPDFV2) bool {
	previa, ok := revisiones[r.FirmaAnteriorRef]
	return ok && previa.OrdenFirmaPDF == 1 && r.ReciboAnteriorRef == previa.ReciboRef && r.Secuencia == previa.Secuencia+1 &&
		r.Documento == previa.Documento && r.CatalogoRef == previa.CatalogoRef && r.CatalogoHuella == previa.CatalogoHuella &&
		r.OriginalRef == previa.OriginalRef && r.OriginalVersion == previa.OriginalVersion && r.OriginalHuella == previa.OriginalHuella &&
		r.EntradaDocumentoRef == previa.DocumentoCustodiaRef && r.EntradaDocumentoVersion == previa.DocumentoCustodiaVersion &&
		r.EntradaDocumentoHuella == previa.FirmadoHuella && r.EntradaDocumentoLongitud == previa.RevisionLongitud
}
