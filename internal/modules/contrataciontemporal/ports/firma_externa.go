package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El registro externo describe un PDF ya firmado y verificado por VEC. La
// referencia y fecha de Portafirmas son declaraciones de RRHH, no acreditan
// envío, recepción ni eficacia jurídica.
const (
	ViaFirmaExternaPortafirmas  = "portafirmas_registro_rrhh"
	AccionRegistrarFirmaExterna = "contratacion_temporal.documento.firma_externa.registrar"
	AudienciaFirmaExternaV3     = "vec_contratacion_temporal.firma_externa.v1"
	TipoRecursoFirmaExterna     = "firma_externa_documento_contratacion_temporal"
	PrefijoRecursoFirmaExterna  = "operacion-firma-externa-ct:"
)

var (
	ErrFuenteOriginalFirmaNoDisponible  = errors.New("contratacion temporal: original autorizado no disponible")
	ErrOriginalFirmaNoAutorizado        = errors.New("contratacion temporal: original no corresponde al expediente y version")
	ErrCompetenciaFirmanteNoDisponible  = errors.New("contratacion temporal: competencia del firmante no disponible")
	ErrCompetenciaFirmanteNoAcreditada  = errors.New("contratacion temporal: competencia del firmante no acreditada")
	ErrRegistroFirmaExternaNoDisponible = errors.New("contratacion temporal: registro de firma externa no disponible")
)

// SolicitudOriginalFirma identifica una revisión concreta. La fuente es la
// autoridad del original y devuelve sus bytes; nunca los toma del navegador.
type SolicitudOriginalFirma struct {
	OrganizacionRef, ExpedienteRef, Documento, OriginalRef string
	OriginalVersion                                        uint64
}

type OriginalFirmaAutorizado struct {
	Solicitud    SolicitudOriginalFirma
	Contenido    []byte
	HuellaSHA256 string
}

type FuenteOriginalFirmaAutorizado interface {
	ObtenerOriginalFirma(context.Context, SolicitudOriginalFirma) (OriginalFirmaAutorizado, error)
}

// La competencia corresponde al firmante del dictamen criptográfico, no al
// registrador RRHH. La fuente debe resolver una única asignación vigente.
type SolicitudCompetenciaFirmante struct {
	OrganizacionRef, ExpedienteRef, Documento string
	CatalogoRef, CatalogoHuella, PasoRef      string
	PasoOrden                                 int
	CargoFirmante, PerfilFirmanteRef          string
	FirmanteRef, CertificadoHuella            string
}

type EvidenciaCompetenciaFirmante struct {
	Solicitud                                                               SolicitudCompetenciaFirmante
	FirmantePrincipalRef                                                    string
	PerfilFirmanteRef, CargoFirmante, UnidadFirmanteRef                     string
	PuestoFirmanteRef, AmbitoFirmanteRef                                    string
	AsignacionFirmanteRef                                                   string
	AsignacionFirmanteVersion                                               uint64
	AsignacionFirmanteHuella                                                string
	AsignacionVigenteDesde, AsignacionVigenteHasta, CompetenciaComprobadaEn string
	ActoCompetenciaRef, DelegacionRef                                       string
	Vigente                                                                 bool
}

type FuenteCompetenciaFirmante interface {
	AcreditarCompetenciaFirmante(context.Context, SolicitudCompetenciaFirmante) (EvidenciaCompetenciaFirmante, error)
}

// MaterialFirmaExterna es el contrato canónico de CT170. La identidad del
// registrador procede exclusivamente de la capacidad V3 consumida en SQL.
type MaterialFirmaExterna struct {
	Via                                                                       string
	OrganizacionRef, ExpedienteRef                                            string
	VersionExpediente                                                         uint64
	Documento, CatalogoRef, CatalogoHuella, PasoRef                           string
	PasoOrden, Secuencia                                                      int
	OriginalRef                                                               string
	OriginalVersion                                                           uint64
	OriginalHuella, FirmadoHuella, CertificadoHuella, FirmanteRef             string
	FirmantePrincipalRef, PerfilFirmanteRef, CargoFirmante, UnidadFirmanteRef string
	PuestoFirmanteRef, AmbitoFirmanteRef                                      string
	AsignacionFirmanteRef                                                     string
	AsignacionFirmanteVersion                                                 uint64
	AsignacionFirmanteHuella                                                  string
	AsignacionVigenteDesde, AsignacionVigenteHasta, CompetenciaComprobadaEn   string
	ActoCompetenciaRef, DelegacionRef                                         string
	PoliticaVerificacion, RevocacionEstado, SelloTiempoEstado                 string
	ReferenciaPortafirmasDeclarada, FechaPortafirmasDeclarada                 string
	ClaveIdempotencia, DocumentoCustodiaRef                                   string
	DocumentoCustodiaVersion                                                  uint64
}

func FechaFirmaExternaCanonica(v string) (time.Time, bool) {
	if !strings.HasSuffix(v, "Z") || len(v) > len("2006-01-02T15:04:05.000000Z") {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil || t.UTC().Format(time.RFC3339Nano) != v {
		return time.Time{}, false
	}
	return t, true
}

func ReferenciaPortafirmasDeclaradaValida(v string) bool {
	if v == "" || len(v) > 256 || strings.TrimSpace(v) != v || !utf8.ValidString(v) {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (m MaterialFirmaExterna) Validar() error {
	if m.Via != ViaFirmaExternaPortafirmas ||
		!ReferenciaPortafirmasDeclaradaValida(m.ReferenciaPortafirmasDeclarada) {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	if _, ok := FechaFirmaExternaCanonica(m.FechaPortafirmasDeclarada); !ok {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	return m.validarComun()
}

func (m MaterialFirmaExterna) validarComun() error {
	desde, okDesde := FechaFirmaExternaCanonica(m.AsignacionVigenteDesde)
	hasta, okHasta := FechaFirmaExternaCanonica(m.AsignacionVigenteHasta)
	comprobada, okComprobada := FechaFirmaExternaCanonica(m.CompetenciaComprobadaEn)
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) || !domain.ReferenciaOpacaValida(m.ExpedienteRef) ||
		m.VersionExpediente == 0 || m.VersionExpediente > 9007199254740991 ||
		!domain.ClaveDocumentoFirmaValida(m.Documento) || !domain.ReferenciaOpacaValida(m.CatalogoRef) ||
		!domain.HuellaSHA256FirmaValida(m.CatalogoHuella) || m.PasoRef == "" || len(m.PasoRef) > 256 ||
		m.PasoOrden < 1 || m.PasoOrden > domain.MaximoPasosCircuitoFirma ||
		m.Secuencia < 1 || m.Secuencia > 100000 ||
		!domain.ReferenciaOpacaValida(m.OriginalRef) || m.OriginalVersion == 0 || m.OriginalVersion > 9007199254740991 ||
		!domain.HuellaSHA256FirmaValida(m.OriginalHuella) || !domain.HuellaSHA256FirmaValida(m.FirmadoHuella) ||
		m.OriginalHuella == m.FirmadoHuella || !domain.HuellaSHA256FirmaValida(m.CertificadoHuella) ||
		m.FirmanteRef != "ref:"+m.CertificadoHuella ||
		!domain.ReferenciaOpacaValida(m.FirmantePrincipalRef) || !strings.HasPrefix(m.FirmantePrincipalRef, "per_") ||
		!domain.ReferenciaOpacaValida(m.PerfilFirmanteRef) || !ReferenciaPortafirmasDeclaradaValida(m.CargoFirmante) ||
		!domain.ReferenciaOpacaValida(m.UnidadFirmanteRef) ||
		(m.PuestoFirmanteRef != "" && !domain.ReferenciaOpacaValida(m.PuestoFirmanteRef)) ||
		(m.AmbitoFirmanteRef != "" && !domain.ReferenciaOpacaValida(m.AmbitoFirmanteRef)) ||
		!domain.ReferenciaOpacaValida(m.AsignacionFirmanteRef) ||
		m.AsignacionFirmanteVersion == 0 || m.AsignacionFirmanteVersion > 9007199254740991 ||
		!domain.HuellaSHA256FirmaValida(m.AsignacionFirmanteHuella) ||
		!okDesde || !okHasta || !okComprobada || comprobada.Before(desde) || !comprobada.Before(hasta) ||
		(m.ActoCompetenciaRef != "" && !domain.ReferenciaOpacaValida(m.ActoCompetenciaRef)) ||
		(m.DelegacionRef != "" && !domain.ReferenciaOpacaValida(m.DelegacionRef)) ||
		m.PoliticaVerificacion != PoliticaVerificacionFirma || m.RevocacionEstado != "vigente" ||
		(m.SelloTiempoEstado != "no_presente" && m.SelloTiempoEstado != "valido" && m.SelloTiempoEstado != "no_comprobado") ||
		!ClaveIdempotenciaFirmaValida(m.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(m.DocumentoCustodiaRef) || m.DocumentoCustodiaVersion != VersionDocumentoCustodiado {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	return nil
}

// Canonico conserva orden y nulabilidad exactos del JSON que CT170 valida.
func (m MaterialFirmaExterna) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrSolicitudFirmaDocumentoInvalida
	}
	return canonicoFirmaVerificada(m, true)
}

// canonicoFirmaVerificada comparte el contrato de hechos comprobados. Solo
// Portafirmas añade las dos declaraciones; VEC las omite del JSON.
func canonicoFirmaVerificada(m MaterialFirmaExterna, declarada bool) ([]byte, error) {
	var referencia, fecha *string
	if declarada {
		referencia, fecha = &m.ReferenciaPortafirmasDeclarada, &m.FechaPortafirmasDeclarada
	}
	return json.Marshal(struct {
		Via                                                                       string
		OrganizacionRef, ExpedienteRef                                            string
		VersionExpediente                                                         uint64
		Documento, CatalogoRef, CatalogoHuella, PasoRef                           string
		PasoOrden, Secuencia                                                      int
		OriginalRef                                                               string
		OriginalVersion                                                           uint64
		OriginalHuella, FirmadoHuella, CertificadoHuella, FirmanteRef             string
		FirmantePrincipalRef, PerfilFirmanteRef, CargoFirmante, UnidadFirmanteRef string
		PuestoFirmanteRef, AmbitoFirmanteRef                                      *string
		AsignacionFirmanteRef                                                     string
		AsignacionFirmanteVersion                                                 uint64
		AsignacionFirmanteHuella                                                  string
		AsignacionVigenteDesde, AsignacionVigenteHasta, CompetenciaComprobadaEn   string
		ActoCompetenciaRef, DelegacionRef                                         *string
		PoliticaVerificacion, RevocacionEstado, SelloTiempoEstado                 string
		ReferenciaPortafirmasDeclarada                                            *string `json:"ReferenciaPortafirmasDeclarada,omitempty"`
		FechaPortafirmasDeclarada                                                 *string `json:"FechaPortafirmasDeclarada,omitempty"`
		ClaveIdempotencia, DocumentoCustodiaRef                                   string
		DocumentoCustodiaVersion                                                  uint64
	}{
		m.Via, m.OrganizacionRef, m.ExpedienteRef, m.VersionExpediente,
		m.Documento, m.CatalogoRef, m.CatalogoHuella, m.PasoRef, m.PasoOrden, m.Secuencia,
		m.OriginalRef, m.OriginalVersion, m.OriginalHuella, m.FirmadoHuella, m.CertificadoHuella, m.FirmanteRef,
		m.FirmantePrincipalRef, m.PerfilFirmanteRef, m.CargoFirmante, m.UnidadFirmanteRef,
		nulo(m.PuestoFirmanteRef), nulo(m.AmbitoFirmanteRef), m.AsignacionFirmanteRef,
		m.AsignacionFirmanteVersion, m.AsignacionFirmanteHuella, m.AsignacionVigenteDesde,
		m.AsignacionVigenteHasta, m.CompetenciaComprobadaEn, nulo(m.ActoCompetenciaRef),
		nulo(m.DelegacionRef), m.PoliticaVerificacion, m.RevocacionEstado, m.SelloTiempoEstado,
		referencia, fecha, m.ClaveIdempotencia,
		m.DocumentoCustodiaRef, m.DocumentoCustodiaVersion,
	})
}

func (m MaterialFirmaExterna) HuellaSHA256() (string, error) {
	c, err := m.Canonico()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(c)
	return hex.EncodeToString(h[:]), nil
}

func (m MaterialFirmaExterna) RecursoRef() string {
	return PrefijoRecursoFirmaExterna + m.ClaveIdempotencia
}

type CapacidadFirmaExterna struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialFirmaExterna(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadFirmaExterna {
	return CapacidadFirmaExterna{material: m}
}

func (c CapacidadFirmaExterna) ExportarMaterialParaConsumidor() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

type AutorizadorRegistroFirmaExterna interface {
	AutorizarRegistroFirmaExterna(context.Context, MaterialFirmaExterna) (CapacidadFirmaExterna, error)
}

// ConsultarFirmas debe unir CT118 y CT170 en una sola secuencia por documento.
type RegistroFirmasExternas interface {
	RegistrarFirmaExterna(context.Context, MaterialFirmaExterna, CapacidadFirmaExterna) (ReciboFirmaDocumento, error)
	ConsultarFirmas(context.Context, string, string) ([]FirmaRegistrada, error)
}
