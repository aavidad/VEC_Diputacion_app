package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

var ErrPlanIncorporacionPersonalB2 = errors.New("contratacion temporal: plan de incorporacion personal invalido")
var uuidPlanPersonalB2 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var claseOcupacionPlanPersonalB2 = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func ClaseOcupacionPlanPersonalB2Valida(s string) bool {
	return claseOcupacionPlanPersonalB2.MatchString(s)
}

// Referencias de fuentes propietarias; CT no copia sus agregados.
type FuentePlanPersonalB2 struct {
	Ref     string `json:"ref"`
	Version uint64 `json:"version"`
	SHA256  string `json:"sha256"`
}
type FuenteSinVersionPlanB2 struct {
	Ref    string `json:"ref"`
	SHA256 string `json:"sha256"`
}
type InstrumentoPlanPersonalB2 struct {
	Ref          string `json:"ref"`
	Revision     uint64 `json:"revision"`
	FuenteRef    string `json:"fuente_ref"`
	FuenteSHA256 string `json:"fuente_sha256"`
}

func (f FuenteSinVersionPlanB2) Valida() bool {
	return ReferenciaOpacaValida(f.Ref) && HuellaPlanPersonalB2Valida(f.SHA256)
}
func (f InstrumentoPlanPersonalB2) Valida() bool {
	return UUIDPlanPersonalB2Valido(f.Ref) && VersionPlanPersonalB2Valida(f.Revision) && ReferenciaOpacaValida(f.FuenteRef) && HuellaPlanPersonalB2Valida(f.FuenteSHA256)
}

type EntradaPlanPersonalB2 struct {
	Ref     string `json:"ref"`
	Version uint64 `json:"version"`
}
type SelectorBolsaPlanB2 struct {
	UnidadRef                string `json:"unidad_ref"`
	CategoriaRef             string `json:"categoria_ref"`
	NecesidadRef             string `json:"necesidad_ref"`
	AceptacionOperacionRef   string `json:"aceptacion_operacion_ref"`
	AceptacionRegistroSHA256 string `json:"aceptacion_registro_sha256"`
	AperturaOperacionRef     string `json:"apertura_operacion_ref"`
	AperturaRegistroSHA256   string `json:"apertura_registro_sha256"`
	LlamamientoRef           string `json:"llamamiento_ref"`
	PropuestaRef             string `json:"propuesta_ref"`
}

// PlanIncorporacionPersonalB2 es el acto prospectivo de RRHH: queda inmutable
// antes de cualquier efecto de Personal. Sus referencias proceden de fuentes
// autorizadas. Un conjunto sintético conserva el protocolo B2 independiente.
type PlanIncorporacionPersonalB2 struct {
	OrganizacionRef       string                    `json:"organizacion_ref"`
	UnidadCTRef           string                    `json:"unidad_ct_ref"`
	ExpedienteRef         string                    `json:"expediente_ref"`
	VersionExpediente     uint64                    `json:"version_expediente"`
	AnalisisVersion       uint64                    `json:"analisis_version"`
	AnalisisReciboRef     string                    `json:"analisis_recibo_ref"`
	AnalisisSHA256        string                    `json:"analisis_sha256"`
	PropuestaReciboRef    string                    `json:"propuesta_recibo_ref"`
	AceptacionRef         string                    `json:"aceptacion_ref"`
	AceptacionReciboRef   string                    `json:"aceptacion_recibo_ref"`
	Bolsa                 SelectorBolsaPlanB2       `json:"bolsa"`
	PersonaRef            string                    `json:"persona_ref"`
	PersonaVersion        uint64                    `json:"persona_version"`
	PersonaFuente         FuentePlanPersonalB2      `json:"persona_fuente"`
	PersonaReciboBolsaRef string                    `json:"persona_recibo_bolsa_ref"`
	OrganismoRef          string                    `json:"organismo_ref"`
	UnidadRef             string                    `json:"unidad_ref"`
	FuenteOrganizacion    FuenteSinVersionPlanB2    `json:"fuente_organizacion"`
	FuentePlantilla       InstrumentoPlanPersonalB2 `json:"fuente_plantilla"`
	FuenteRPT             InstrumentoPlanPersonalB2 `json:"fuente_rpt"`
	VersionPlazaRef       string                    `json:"version_plaza_ref"`
	VersionPuestoRef      string                    `json:"version_puesto_ref"`
	RevisionPlaza         uint64                    `json:"revision_plaza"`
	RevisionPuesto        uint64                    `json:"revision_puesto"`
	PuestoRef             string                    `json:"puesto_ref"`
	PlazaRef              string                    `json:"plaza_ref"`
	CatalogoRPTID         string                    `json:"catalogo_rpt_id"`
	CatalogoRPTModulo     string                    `json:"catalogo_rpt_modulo"`
	CatalogoRPTVersion    uint64                    `json:"catalogo_rpt_version"`
	CatalogoRPTSHA256     string                    `json:"catalogo_rpt_sha256"`
	CategoriaRef          string                    `json:"categoria_ref"`
	VinculoRevision       uint64                    `json:"vinculo_revision"`
	VinculoReciboRef      string                    `json:"vinculo_recibo_ref"`
	Regimen               EntradaPlanPersonalB2     `json:"regimen"`
	Modalidad             EntradaPlanPersonalB2     `json:"modalidad"`
	ClaseOcupacion        string                    `json:"clase_ocupacion"`
	Desde                 string                    `json:"desde"`
	Hasta                 string                    `json:"hasta"`
	MotivoClave           string                    `json:"motivo_clave"`
	DocumentoRef          string                    `json:"documento_ref"`
	DocumentoSHA256       string                    `json:"documento_sha256"`
	EjercicioSintetico    bool                      `json:"ejercicio_sintetico"`
}

func HuellaPlanPersonalB2Valida(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == sha256.Size && hex.EncodeToString(b) == s && s != string(bytes.Repeat([]byte{'0'}, 64))
}
func VersionPlanPersonalB2Valida(v uint64) bool { return v > 0 && v <= 9007199254740991 }
func (f FuentePlanPersonalB2) Valida() bool {
	return ReferenciaOpacaValida(f.Ref) && VersionPlanPersonalB2Valida(f.Version) && HuellaPlanPersonalB2Valida(f.SHA256)
}
func (p PlanIncorporacionPersonalB2) Validar() error {
	for _, r := range []string{p.OrganizacionRef, p.UnidadCTRef, p.ExpedienteRef, p.AnalisisReciboRef, p.PropuestaReciboRef, p.AceptacionRef, p.AceptacionReciboRef, p.PersonaRef, p.PersonaReciboBolsaRef, p.OrganismoRef, p.UnidadRef, p.VersionPlazaRef, p.VersionPuestoRef, p.PuestoRef, p.PlazaRef, p.CatalogoRPTID, p.CatalogoRPTModulo, p.CategoriaRef, p.VinculoReciboRef, p.DocumentoRef, p.Regimen.Ref, p.Modalidad.Ref, p.Bolsa.UnidadRef, p.Bolsa.CategoriaRef, p.Bolsa.NecesidadRef, p.Bolsa.AceptacionOperacionRef, p.Bolsa.AperturaOperacionRef, p.Bolsa.LlamamientoRef, p.Bolsa.PropuestaRef} {
		if !ReferenciaOpacaValida(r) {
			return ErrPlanIncorporacionPersonalB2
		}
	}
	for _, v := range []uint64{p.VersionExpediente, p.AnalisisVersion, p.PersonaVersion, p.RevisionPlaza, p.RevisionPuesto, p.CatalogoRPTVersion, p.VinculoRevision, p.Regimen.Version, p.Modalidad.Version} {
		if !VersionPlanPersonalB2Valida(v) {
			return ErrPlanIncorporacionPersonalB2
		}
	}
	for _, h := range []string{p.AnalisisSHA256, p.CatalogoRPTSHA256, p.DocumentoSHA256, p.Bolsa.AceptacionRegistroSHA256, p.Bolsa.AperturaRegistroSHA256} {
		if !HuellaPlanPersonalB2Valida(h) {
			return ErrPlanIncorporacionPersonalB2
		}
	}
	d, e := time.Parse(time.DateOnly, p.Desde)
	h, eh := time.Parse(time.DateOnly, p.Hasta)
	if !ClaseOcupacionPlanPersonalB2Valida(p.ClaseOcupacion) || e != nil || d.Year() <= 0 || d.Format(time.DateOnly) != p.Desde || p.Hasta != "" && (eh != nil || h.Year() <= 0 || h.Format(time.DateOnly) != p.Hasta || !h.After(d)) || !ClaveCatalogo(p.MotivoClave).Valida() || p.AnalisisVersion > p.VersionExpediente || p.CategoriaRef != p.Bolsa.CategoriaRef || !p.PersonaFuente.Valida() || !p.FuenteOrganizacion.Valida() || !p.FuentePlantilla.Valida() || !p.FuenteRPT.Valida() {
		return ErrPlanIncorporacionPersonalB2
	}
	return nil
}

// CanonicoPlanPersonalB2 ordena las claves recursivamente; no convierte números
// a float64 ni normaliza fechas o referencias aportadas por la fuente.
func CanonicoPlanPersonalB2(v any) ([]byte, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	var m any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if e = dec.Decode(&m); e != nil {
		return nil, e
	}
	return json.Marshal(m)
}
func SHA256PlanPersonalB2(v any) (string, error) {
	b, e := CanonicoPlanPersonalB2(v)
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func UUIDPlanPersonalB2Valido(s string) bool {
	return uuidPlanPersonalB2.MatchString(s) && s != "00000000-0000-0000-0000-000000000000"
}
