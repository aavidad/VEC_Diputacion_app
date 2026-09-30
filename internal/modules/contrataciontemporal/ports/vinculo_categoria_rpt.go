package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	EsquemaConsultaVinculoCategoriaRPT        = "vec.ct.vinculo-categoria-rpt.consulta.v1"
	EsquemaRegistroVinculoCategoriaRPT        = "vec.ct.vinculo-categoria-rpt.registro.v1"
	AccionConsultarVinculoCategoriaRPT        = "contratacion_temporal.categoria_rpt.vinculo.consultar"
	AccionRegistrarVinculoCategoriaRPT        = "contratacion_temporal.categoria_rpt.vinculo.registrar"
	AudienciaConsultarVinculoCategoriaRPT     = "vec_contratacion_temporal.categoria_rpt.vinculo.consultar.v1"
	AudienciaRegistrarVinculoCategoriaRPT     = "vec_contratacion_temporal.categoria_rpt.vinculo.registrar.v1"
	AccionConsultarPublicacionCategoriaRPT    = "vec.catalogos.categorias.consultar_historica"
	AudienciaConsultarPublicacionCategoriaRPT = "vec_catalogos_configurables.lectura_categorias.v1"
)

var (
	ErrVinculoCategoriaRPTInvalido     = errors.New("contratacion temporal: vinculo de categoria RPT invalido")
	ErrVinculoCategoriaRPTConflicto    = errors.New("contratacion temporal: conflicto de vinculo de categoria RPT")
	ErrVinculoCategoriaRPTDenegado     = errors.New("contratacion temporal: vinculo de categoria RPT denegado")
	ErrVinculoCategoriaRPTNoEncontrado = errors.New("contratacion temporal: vinculo de categoria RPT no encontrado")
	ErrVinculoCategoriaRPTNoDisponible = errors.New("contratacion temporal: vinculo de categoria RPT no disponible")
)

var claveVinculoRPTUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var expedienteVinculoRPT = regexp.MustCompile(`^expediente:[A-Za-z0-9._:/#-]{2,149}$`)

// El orden de los campos es el contrato de material con V3 y SQL. La lectura
// devuelve la huella del analisis calculada por CT en PostgreSQL.
type ConsultaVinculoCategoriaRPT struct {
	Esquema         string `json:"esquema"`
	OrganizacionRef string `json:"organizacion_ref"`
	ExpedienteRef   string `json:"expediente_ref"`
}

func NuevaConsultaVinculoCategoriaRPT(organizacionRef, expedienteRef string) (ConsultaVinculoCategoriaRPT, error) {
	c := ConsultaVinculoCategoriaRPT{EsquemaConsultaVinculoCategoriaRPT, organizacionRef, expedienteRef}
	if c.Validar() != nil {
		return ConsultaVinculoCategoriaRPT{}, ErrVinculoCategoriaRPTInvalido
	}
	return c, nil
}

func (c ConsultaVinculoCategoriaRPT) Validar() error {
	if c.Esquema != EsquemaConsultaVinculoCategoriaRPT || !domain.ReferenciaOpacaValida(c.OrganizacionRef) || !expedienteVinculoRPT.MatchString(c.ExpedienteRef) {
		return ErrVinculoCategoriaRPTInvalido
	}
	return nil
}

func (c ConsultaVinculoCategoriaRPT) Canonico() ([]byte, error) {
	if c.Validar() != nil {
		return nil, ErrVinculoCategoriaRPTInvalido
	}
	return json.Marshal(c)
}

type AnclajeAnalisisCategoriaRPT struct {
	VersionExpediente    uint64 `json:"version_expediente"`
	AnalisisVersion      uint64 `json:"analisis_version"`
	AnalisisReciboRef    string `json:"analisis_recibo_ref"`
	AnalisisHuellaSHA256 string `json:"analisis_huella_sha256"`
	CategoriaRef         string `json:"categoria_ref"`
}

func (a AnclajeAnalisisCategoriaRPT) Validar() error {
	if a.VersionExpediente < 2 || a.VersionExpediente > MaximoEnteroSeguroOperacionAnalisis ||
		a.AnalisisVersion < 2 || a.AnalisisVersion > a.VersionExpediente ||
		!domain.ReferenciaOpacaValida(a.AnalisisReciboRef) ||
		!huellaVinculoRPT(a.AnalisisHuellaSHA256) || !domain.ReferenciaOpacaValida(a.CategoriaRef) {
		return ErrVinculoCategoriaRPTInvalido
	}
	return nil
}

type EstadoVinculoCategoriaRPT struct {
	Revision                     uint64 `json:"revision"`
	ReciboRef                    string `json:"recibo_ref"`
	CatalogoID                   string `json:"catalogo_id"`
	ModuloID                     string `json:"modulo_id"`
	CatalogoVersion              uint64 `json:"catalogo_version"`
	CatalogoHuellaSHA256         string `json:"catalogo_huella_sha256"`
	CategoriaID                  string `json:"categoria_id"`
	FuenteRef                    string `json:"fuente_ref"`
	MotivoRef                    string `json:"motivo_ref"`
	AprobacionRef                string `json:"aprobacion_ref"`
	Prospectivo                  bool   `json:"prospectivo"`
	AcreditaProcedenciaHistorica bool   `json:"acredita_procedencia_historica"`
}

func (e EstadoVinculoCategoriaRPT) Publicacion() domain.PublicacionCategoriaRPT {
	return domain.PublicacionCategoriaRPT{CatalogoID: e.CatalogoID, ModuloID: e.ModuloID,
		CatalogoVersion: e.CatalogoVersion, CatalogoHuella: e.CatalogoHuellaSHA256,
		CategoriaID: e.CategoriaID, CategoriaClave: e.CategoriaID}
}

func (e EstadoVinculoCategoriaRPT) Validar() error {
	if e.Revision == 0 || e.Revision > MaximoEnteroSeguroOperacionAnalisis ||
		!domain.ReferenciaOpacaValida(e.ReciboRef) || e.Publicacion().Validar() != nil ||
		!domain.ReferenciaOpacaValida(e.FuenteRef) || !domain.ReferenciaOpacaValida(e.MotivoRef) || !domain.ReferenciaOpacaValida(e.AprobacionRef) ||
		!e.Prospectivo || e.AcreditaProcedenciaHistorica {
		return ErrVinculoCategoriaRPTInvalido
	}
	return nil
}

type LecturaVinculoCategoriaRPT struct {
	OrganizacionRef string                      `json:"organizacion_ref"`
	ExpedienteRef   string                      `json:"expediente_ref"`
	Analisis        AnclajeAnalisisCategoriaRPT `json:"analisis"`
	Vinculo         *EstadoVinculoCategoriaRPT  `json:"vinculo"`
}

func (l LecturaVinculoCategoriaRPT) ValidarPara(c ConsultaVinculoCategoriaRPT) error {
	if c.Validar() != nil || l.OrganizacionRef != c.OrganizacionRef || l.ExpedienteRef != c.ExpedienteRef || l.Analisis.Validar() != nil {
		return ErrVinculoCategoriaRPTInvalido
	}
	if l.Vinculo != nil && l.Vinculo.Validar() != nil {
		return ErrVinculoCategoriaRPTInvalido
	}
	return nil
}

// RegistroVinculoCategoriaRPT contiene solo la intencion: SQL aporta actor,
// perfil, fecha y recibo y revalida el anclaje bloqueado antes de escribir.
type RegistroVinculoCategoriaRPT struct {
	Esquema                   string  `json:"esquema"`
	OrganizacionRef           string  `json:"organizacion_ref"`
	ExpedienteRef             string  `json:"expediente_ref"`
	VersionExpedienteEsperada uint64  `json:"version_expediente_esperada"`
	AnalisisVersion           uint64  `json:"analisis_version"`
	AnalisisReciboRef         string  `json:"analisis_recibo_ref"`
	AnalisisHuellaSHA256      string  `json:"analisis_huella_sha256"`
	CategoriaRef              string  `json:"categoria_ref"`
	CatalogoID                string  `json:"catalogo_id"`
	ModuloID                  string  `json:"modulo_id"`
	CatalogoVersion           uint64  `json:"catalogo_version"`
	CatalogoHuellaSHA256      string  `json:"catalogo_huella_sha256"`
	CategoriaID               string  `json:"categoria_id"`
	FuenteRef                 string  `json:"fuente_ref"`
	MotivoRef                 string  `json:"motivo_ref"`
	AprobacionRef             string  `json:"aprobacion_ref"`
	RevisionEsperada          uint64  `json:"revision_esperada"`
	AnteriorReciboRef         *string `json:"anterior_recibo_ref"`
	ClaveIdempotencia         string  `json:"clave_idempotencia"`
}

func (r RegistroVinculoCategoriaRPT) Publicacion() domain.PublicacionCategoriaRPT {
	return domain.PublicacionCategoriaRPT{CatalogoID: r.CatalogoID, ModuloID: r.ModuloID, CatalogoVersion: r.CatalogoVersion, CatalogoHuella: r.CatalogoHuellaSHA256, CategoriaID: r.CategoriaID, CategoriaClave: r.CategoriaID}
}

func (r RegistroVinculoCategoriaRPT) Validar() error {
	a := AnclajeAnalisisCategoriaRPT{VersionExpediente: r.VersionExpedienteEsperada, AnalisisVersion: r.AnalisisVersion, AnalisisReciboRef: r.AnalisisReciboRef, AnalisisHuellaSHA256: r.AnalisisHuellaSHA256, CategoriaRef: r.CategoriaRef}
	if r.Esquema != EsquemaRegistroVinculoCategoriaRPT || !domain.ReferenciaOpacaValida(r.OrganizacionRef) || !expedienteVinculoRPT.MatchString(r.ExpedienteRef) ||
		a.Validar() != nil || !r.Publicacion().CorrespondeA(r.CategoriaRef) || !domain.ReferenciaOpacaValida(r.FuenteRef) ||
		!domain.ReferenciaOpacaValida(r.MotivoRef) || !domain.ReferenciaOpacaValida(r.AprobacionRef) ||
		!claveVinculoRPTUUID.MatchString(r.ClaveIdempotencia) || r.RevisionEsperada > 999999 ||
		(r.RevisionEsperada == 0 && r.AnteriorReciboRef != nil) ||
		(r.RevisionEsperada > 0 && (r.AnteriorReciboRef == nil || !domain.ReferenciaOpacaValida(*r.AnteriorReciboRef))) {
		return ErrVinculoCategoriaRPTInvalido
	}
	return nil
}

func (r RegistroVinculoCategoriaRPT) Canonico() ([]byte, error) {
	if r.Validar() != nil {
		return nil, ErrVinculoCategoriaRPTInvalido
	}
	return json.Marshal(r)
}

func (r RegistroVinculoCategoriaRPT) SHA256() (string, error) {
	b, e := r.Canonico()
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

type ReciboVinculoCategoriaRPT struct {
	ReciboRef                    string                    `json:"recibo_ref"`
	RegistradoEn                 time.Time                 `json:"registrado_en"`
	Revision                     uint64                    `json:"revision"`
	MaterialSHA256               string                    `json:"material_huella_sha256"`
	Prospectivo                  bool                      `json:"prospectivo"`
	AcreditaProcedenciaHistorica bool                      `json:"acredita_procedencia_historica"`
	DecisionRef                  string                    `json:"decision_ref"`
	AuditoriaRef                 string                    `json:"auditoria_ref"`
	ConsumoHuellaSHA256          string                    `json:"consumo_huella_sha256"`
	RPTDecisionRef               string                    `json:"rpt_decision_ref"`
	RPTPerfilRef                 string                    `json:"rpt_perfil_ref"`
	RPTAuditoriaRef              string                    `json:"rpt_auditoria_ref"`
	RPTConsumoHuellaSHA256       string                    `json:"rpt_consumo_huella_sha256"`
	Vinculo                      EstadoVinculoCategoriaRPT `json:"vinculo"`
}

func (r ReciboVinculoCategoriaRPT) ValidarPara(m RegistroVinculoCategoriaRPT) error {
	h, e := m.SHA256()
	if e != nil || !domain.ReferenciaOpacaValida(r.ReciboRef) || !domain.InstanteUTCCanonico(r.RegistradoEn) ||
		r.Revision != m.RevisionEsperada+1 || r.MaterialSHA256 != h || !r.Prospectivo || r.AcreditaProcedenciaHistorica ||
		!domain.ReferenciaOpacaValida(r.DecisionRef) || !domain.ReferenciaOpacaValida(r.AuditoriaRef) || !huellaVinculoRPT(r.ConsumoHuellaSHA256) ||
		!domain.ReferenciaOpacaValida(r.RPTDecisionRef) || !domain.ReferenciaOpacaValida(r.RPTPerfilRef) || !domain.ReferenciaOpacaValida(r.RPTAuditoriaRef) || !huellaVinculoRPT(r.RPTConsumoHuellaSHA256) ||
		r.DecisionRef == r.RPTDecisionRef || r.AuditoriaRef == r.RPTAuditoriaRef || r.ConsumoHuellaSHA256 == r.RPTConsumoHuellaSHA256 ||
		r.Vinculo.Validar() != nil || r.Vinculo.Revision != r.Revision || r.Vinculo.ReciboRef != r.ReciboRef ||
		r.Vinculo.CatalogoID != m.CatalogoID || r.Vinculo.ModuloID != m.ModuloID || r.Vinculo.CatalogoVersion != m.CatalogoVersion ||
		r.Vinculo.CatalogoHuellaSHA256 != m.CatalogoHuellaSHA256 || r.Vinculo.CategoriaID != m.CategoriaID ||
		r.Vinculo.FuenteRef != m.FuenteRef || r.Vinculo.MotivoRef != m.MotivoRef || r.Vinculo.AprobacionRef != m.AprobacionRef {
		return ErrVinculoCategoriaRPTInvalido
	}
	return nil
}

func huellaVinculoRPT(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == sha256.Size && hex.EncodeToString(b) == s
}

// Las exportaciones son estructuras de transporte; el efecto consume y vuelve
// a verificar ambos permisos y la publicacion en una sola transaccion SQL.
type AutoridadVinculoCategoriaRPT interface {
	ConsultaCT(context.Context, ConsultaVinculoCategoriaRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	RegistroCT(context.Context, RegistroVinculoCategoriaRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	LecturaRPT(context.Context, domain.PublicacionCategoriaRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type FuenteVinculoCategoriaRPT interface {
	Consultar(context.Context, ConsultaVinculoCategoriaRPT, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (LecturaVinculoCategoriaRPT, error)
	Registrar(context.Context, RegistroVinculoCategoriaRPT, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboVinculoCategoriaRPT, error)
}

type FuentePublicacionCategoriaRPT interface {
	ConsultarPublicacionCategoriaRPT(context.Context, domain.PublicacionCategoriaRPT, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.PublicacionCategoriaRPT, error)
}
