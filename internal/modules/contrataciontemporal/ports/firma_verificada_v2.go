package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const (
	AudienciaFirmaVecV2         = "vec_contratacion_temporal.firma_vec.v2"
	AudienciaFirmaExternaV2     = "vec_contratacion_temporal.firma_externa.v2"
	AccionConsultarFirmasR5V2   = "contratacion_temporal.documento.firmas_r5_v2.consultar"
	AudienciaConsultaFirmasR5V2 = "vec_contratacion_temporal.firmas_r5.consultar.v2"
)

// Cada acto V2 añade una única firma a la revisión de entrada. El original
// permanece como raíz y el antecedente identifica una firma V2 concreta.
type MaterialFirmaVerificadaV2 struct {
	MaterialFirmaExterna
	CatalogoVersion                                    uint64
	FirmaAnteriorRef, ReciboAnteriorRef                string
	EntradaDocumentoRef                                string
	EntradaDocumentoVersion, EntradaDocumentoLongitud  uint64
	EntradaDocumentoHuella                             string
	OrdenFirmaPDF                                      int
	ByteRange                                          [4]uint64
	RevisionHuellaSHA256, ContenidoFirmadoHuellaSHA256 string
	RevisionLongitud                                   uint64
	EvidenciaFirmasCanonica                            json.RawMessage
	EvidenciaFirmasHuellaSHA256                        string
	ComprobadaEn                                       time.Time
}

func (m MaterialFirmaVerificadaV2) Validar() error {
	// Las guardas compartidas validan los hechos comunes, nunca registran V1.
	base := m.MaterialFirmaExterna
	if base.PoliticaVerificacion != "politica:vec:firma:verificacion-autonoma:v2" {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	base.PoliticaVerificacion = PoliticaVerificacionFirma
	if (base.Via == ViaFirmaCertificadoVEC && MaterialFirmaVec(base).Validar() != nil) ||
		(base.Via == ViaFirmaExternaPortafirmas && base.Validar() != nil) ||
		(base.Via != ViaFirmaCertificadoVEC && base.Via != ViaFirmaExternaPortafirmas) ||
		m.PasoOrden < 1 || m.PasoOrden > 2 || m.OrdenFirmaPDF != m.PasoOrden ||
		!domain.ReferenciaOpacaValida(m.EntradaDocumentoRef) || m.EntradaDocumentoVersion < 1 || m.EntradaDocumentoVersion > 9007199254740991 ||
		m.EntradaDocumentoLongitud < 1 || m.EntradaDocumentoLongitud >= m.RevisionLongitud || m.RevisionLongitud > 32*1024*1024 ||
		!domain.HuellaSHA256FirmaValida(m.EntradaDocumentoHuella) ||
		m.RevisionHuellaSHA256 != m.FirmadoHuella || !domain.HuellaSHA256FirmaValida(m.ContenidoFirmadoHuellaSHA256) ||
		m.ByteRange[0] != 0 || m.ByteRange[1] < m.EntradaDocumentoLongitud || m.ByteRange[2] <= m.ByteRange[1] ||
		m.ByteRange[2] > m.RevisionLongitud || m.ByteRange[3] != m.RevisionLongitud-m.ByteRange[2] || m.ByteRange[3] == 0 ||
		len(m.EvidenciaFirmasCanonica) < 2 || len(m.EvidenciaFirmasCanonica) > 32768 {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	if m.RolIDFirmante == "" || m.RolIDFirmante != m.CargoFirmante {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	if m.CatalogoVersion < 1 || m.CatalogoVersion > 9007199254740991 {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	if m.PasoOrden == 1 {
		if m.FirmaAnteriorRef != "" || m.ReciboAnteriorRef != "" || m.EntradaDocumentoRef != m.OriginalRef || m.EntradaDocumentoVersion != m.OriginalVersion || m.EntradaDocumentoHuella != m.OriginalHuella {
			return ErrSolicitudFirmaDocumentoInvalida
		}
	} else if !domain.ReferenciaOpacaValida(m.FirmaAnteriorRef) || !domain.ReferenciaOpacaValida(m.ReciboAnteriorRef) {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	var evidence []json.RawMessage
	if json.Unmarshal(m.EvidenciaFirmasCanonica, &evidence) != nil || len(evidence) != m.OrdenFirmaPDF {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	var compact bytes.Buffer
	if json.Compact(&compact, m.EvidenciaFirmasCanonica) != nil || !bytes.Equal(compact.Bytes(), m.EvidenciaFirmasCanonica) {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	h := sha256.Sum256(m.EvidenciaFirmasCanonica)
	if hex.EncodeToString(h[:]) != m.EvidenciaFirmasHuellaSHA256 {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	return nil
}

// ComprobadaEn es observación del verificador; no cambia la identidad del acto.
func (m MaterialFirmaVerificadaV2) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrSolicitudFirmaDocumentoInvalida
	}
	c, err := canonicoFirmaVerificada(m.MaterialFirmaExterna, m.Via == ViaFirmaExternaPortafirmas)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(c, &fields) != nil {
		return nil, ErrSolicitudFirmaDocumentoInvalida
	}
	extra := struct {
		FirmaAnteriorRef, ReciboAnteriorRef                *string
		EntradaDocumentoRef                                string
		EntradaDocumentoVersion, EntradaDocumentoLongitud  uint64
		EntradaDocumentoHuella                             string
		OrdenFirmaPDF                                      int
		ByteRange                                          [4]uint64
		RevisionHuellaSHA256, ContenidoFirmadoHuellaSHA256 string
		RevisionLongitud                                   uint64
		EvidenciaFirmasCanonica                            json.RawMessage
		EvidenciaFirmasHuellaSHA256                        string
	}{textoFirma118Nullable(m.FirmaAnteriorRef), textoFirma118Nullable(m.ReciboAnteriorRef), m.EntradaDocumentoRef, m.EntradaDocumentoVersion, m.EntradaDocumentoLongitud, m.EntradaDocumentoHuella, m.OrdenFirmaPDF, m.ByteRange, m.RevisionHuellaSHA256, m.ContenidoFirmadoHuellaSHA256, m.RevisionLongitud, m.EvidenciaFirmasCanonica, m.EvidenciaFirmasHuellaSHA256}
	b, _ := json.Marshal(extra)
	var more map[string]json.RawMessage
	_ = json.Unmarshal(b, &more)
	cv, _ := json.Marshal(m.CatalogoVersion)
	fields["CatalogoVersion"] = cv
	rid, _ := json.Marshal(m.RolIDFirmante)
	fields["RolIDFirmante"] = rid
	for k, v := range more {
		fields[k] = v
	}
	return json.Marshal(fields)
}
func textoFirma118Nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func (m MaterialFirmaVerificadaV2) HuellaSHA256() (string, error) {
	b, e := m.Canonico()
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func (m MaterialFirmaVerificadaV2) RecursoRef() string {
	if m.Via == ViaFirmaCertificadoVEC {
		return PrefijoRecursoFirmaVec + m.ClaveIdempotencia
	}
	return PrefijoRecursoFirmaExterna + m.ClaveIdempotencia
}

type CapacidadFirmaVerificadaV2 struct {
	material vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialFirmaVerificadaV2(m vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadFirmaVerificadaV2 {
	return CapacidadFirmaVerificadaV2{m}
}
func (c CapacidadFirmaVerificadaV2) ExportarMaterialParaConsumidor() vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

type AutorizadorFirmaVerificadaV2 interface {
	AutorizarFirmaVerificadaV2(context.Context, MaterialFirmaVerificadaV2) (CapacidadFirmaVerificadaV2, error)
}
type RegistroFirmasVerificadasV2 interface {
	RegistrarFirmaVerificadaV2(context.Context, MaterialFirmaVerificadaV2, CapacidadFirmaVerificadaV2) (ReciboFirmaDocumento, error)
	ConsultarFirmasAutorizadasV2(context.Context, MaterialConsultaFirmasR5, CapacidadConsultaFirmasR5V2) (LecturaFirmasR5V2, error)
}
type FirmaRegistradaRevisionPDFV2 struct {
	FirmaRegistrada
	FirmaAnteriorRef, ReciboAnteriorRef, EntradaDocumentoRef string
	EntradaDocumentoVersion, EntradaDocumentoLongitud        uint64
	EntradaDocumentoHuella                                   string
	OrdenFirmaPDF                                            int
	ByteRange                                                [4]uint64
	RevisionHuellaSHA256, ContenidoFirmadoHuellaSHA256       string
	RevisionLongitud                                         uint64
	EvidenciaFirmasCanonica                                  json.RawMessage
	EvidenciaFirmasHuellaSHA256                              string
}
type LecturaFirmasR5V2 struct {
	LecturaFirmasR5
	RevisionesPDF []FirmaRegistradaRevisionPDFV2
}
type CapacidadConsultaFirmasR5V2 struct {
	material vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialConsultaFirmasR5V2(m vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadConsultaFirmasR5V2 {
	return CapacidadConsultaFirmasR5V2{m}
}
func (c CapacidadConsultaFirmasR5V2) ExportarMaterialParaConsumidor() vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

type AutorizadorConsultaFirmasR5V2 interface {
	AutorizarConsultaFirmasR5V2(context.Context, MaterialConsultaFirmasR5) (CapacidadConsultaFirmasR5V2, error)
}
