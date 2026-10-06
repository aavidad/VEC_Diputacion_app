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
	PerfilActivoOperadorRef                                                          string
	CuentaFirmanteRef, VinculoCredencialFirmanteRef, VinculoCredencialFirmanteHuella string
	VinculoCredencialFirmanteRevision                                                uint64
	RolIDFirmante                                                                    string
	CatalogoVersion                                                                  uint64
	FirmaAnteriorRef, ReciboAnteriorRef                                              string
	EntradaDocumentoRef                                                              string
	EntradaDocumentoVersion, EntradaDocumentoLongitud                                uint64
	EntradaDocumentoHuella                                                           string
	OrdenFirmaPDF                                                                    int
	ByteRange                                                                        [4]uint64
	RevisionHuellaSHA256, ContenidoFirmadoHuellaSHA256                               string
	RevisionLongitud                                                                 uint64
	EvidenciaFirmasCanonica                                                          json.RawMessage
	EvidenciaFirmasHuellaSHA256                                                      string
	ComprobadaEn                                                                     time.Time
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
	if !domain.ReferenciaOpacaValida(m.CuentaFirmanteRef) || !domain.ReferenciaOpacaValida(m.VinculoCredencialFirmanteRef) || !domain.HuellaSHA256FirmaValida(m.VinculoCredencialFirmanteHuella) || m.VinculoCredencialFirmanteRevision < 1 || m.VinculoCredencialFirmanteRevision > 9007199254740991 {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	if !domain.ReferenciaOpacaValida(m.PerfilActivoOperadorRef) {
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
	op, _ := json.Marshal(m.PerfilActivoOperadorRef)
	fields["PerfilActivoOperadorRef"] = op
	for k, v := range map[string]any{"CuentaFirmanteRef": m.CuentaFirmanteRef, "VinculoCredencialFirmanteRef": m.VinculoCredencialFirmanteRef, "VinculoCredencialFirmanteRevision": m.VinculoCredencialFirmanteRevision, "VinculoCredencialFirmanteHuella": m.VinculoCredencialFirmanteHuella} {
		b, _ := json.Marshal(v)
		fields[k] = b
	}
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

type AutorizadorFirmaVerificadaV2 interface {
	AutorizarFirmaVerificadaV2(context.Context, MaterialFirmaVerificadaV2) (CapacidadFirmaVerificadaV2, error)
}
type RegistroFirmasVerificadasV2 interface {
	RegistrarFirmaVerificadaV2(context.Context, MaterialFirmaVerificadaV2, CapacidadFirmaVerificadaV2) (ReciboFirmaDocumento, error)
	ConsultarFirmasAutorizadasV2(context.Context, MaterialConsultaFirmasR5V2, CapacidadConsultaFirmasR5V2) (LecturaFirmasR5V2, error)
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

// La consulta usa el perfil operativo y su organización y, si su asignación
// tiene unidad, esa unidad en UnidadRef (sólo vía VEC; CT186 la liga a un
// paso del plan publicado). Sin unidad, UnidadRef vacía sale como null.
type MaterialConsultaFirmasR5V2 struct {
	MaterialConsultaFirmasR5
	Via       string
	UnidadRef string
}

func (m MaterialConsultaFirmasR5V2) Canonico() ([]byte, error) {
	c, e := m.MaterialConsultaFirmasR5.Canonico()
	if e != nil {
		return nil, e
	}
	// UnidadRef es la unidad de la asignación de quien consulta (CT186): sólo
	// en la vía VEC; CC10 la liga en SQL a un paso del plan publicado.
	if m.PasoOrden < 1 || m.PasoOrden > 2 || (m.Via != ViaFirmaCertificadoVEC && m.Via != ViaFirmaExternaPortafirmas) ||
		(m.UnidadRef != "" && (m.Via != ViaFirmaCertificadoVEC || !domain.ReferenciaOpacaValida(m.UnidadRef))) {
		return nil, ErrSolicitudFirmaDocumentoInvalida
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(c, &fields)
	b, _ := json.Marshal(m.Via)
	fields["Via"] = b
	b, _ = json.Marshal(textoFirma118Nullable(m.UnidadRef))
	fields["UnidadRef"] = b
	return json.Marshal(fields)
}
func (m MaterialConsultaFirmasR5V2) HuellaSHA256() (string, error) {
	b, e := m.Canonico()
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
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
	AutorizarConsultaFirmasR5V2(context.Context, MaterialConsultaFirmasR5V2) (CapacidadConsultaFirmasR5V2, error)
}

// CamposConsultaFirmasR5V2 es la proyección nominal acordada con AD162.
// Cada llamador obtiene una copia para conservar inmutable el contrato.
func CamposConsultaFirmasR5V2() []string {
	return []string{"ByteRange", "CatalogoHuella", "CatalogoRef", "CertificadoHuella", "ClaveIdempotencia", "CoincideFirmanteCandidato", "CoincideFirmanteEnOtroPaso", "ConMotivoDevolucion", "ContenidoFirmadoHuellaSHA256", "Documento", "DocumentoCustodiaRef", "DocumentoCustodiaVersion", "EntradaDocumentoHuella", "EntradaDocumentoLongitud", "EntradaDocumentoRef", "EntradaDocumentoVersion", "EvidenciaFirmasCanonica", "EvidenciaFirmasHuellaSHA256", "ExpedienteVersion", "FechaPortafirmasDeclarada", "FirmaAnteriorRef", "FirmaRef", "FirmadoHuella", "FirmantePrincipalAcreditado", "FirmanteRef", "HistoriaHuella", "HistoriaRevision", "HistoriaSeparacionAcreditada", "OrdenFirmaPDF", "OriginalHuella", "OriginalRef", "OriginalVersion", "PasoOrden", "PasoRef", "ReciboAnteriorRef", "ReciboRef", "ReferenciaPortafirmasDeclarada", "RegistradaEn", "Resultado", "RevisionHuellaSHA256", "RevisionLongitud", "Secuencia", "SelloTiempoEstado", "Via"}
}
