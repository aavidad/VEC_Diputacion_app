package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	RutaPlanB2                               = "/api/interno/contratacion-temporal/incorporacion-personal-b2/plan/v1"
	RutaConfirmacionB2                       = "/api/interno/contratacion-temporal/incorporacion-personal-b2/confirmar/v1"
	EsquemaConsultaIncorporacionPersonalB2   = "vec.contratacion-temporal.incorporacion-personal-b2.consulta.v1"
	EsquemaReciboIncorporacionPersonalB2     = "vec.contratacion-temporal.incorporacion-personal-b2.recibo.v1"
	MaximoCuerpoIncorporacionPersonalB2Bytes = 8192
)

var (
	ErrPeticionIncorporacionPersonalB2             = errors.New("contratacion temporal http: peticion B2 invalida")
	ErrManejadorIncorporacionPersonalB2            = errors.New("contratacion temporal http: servicio B2 no disponible")
	ErrDenegadaIncorporacionPersonalB2             = errors.New("contratacion temporal http: acceso B2 denegado")
	ErrConflictoIncorporacionPersonalB2            = errors.New("contratacion temporal http: conflicto B2")
	ErrPreparacionPendienteIncorporacionPersonalB2 = errors.New("contratacion temporal http: preparacion B2 pendiente")
	patronUUIDHTTPB2                               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	patronSHAHTTPB2                                = regexp.MustCompile(`^[0-9a-f]{64}$`)
	patronClaveHTTPB2                              = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{1,159}$`)
	patronClaseOcupacionHTTPB2                     = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

// Estos DTO son el canal público minimizado. El ensamblado obtiene organización,
// persona, contexto, fuentes y recibos propietarios mediante sus autoridades.
type EntradaCatalogoB2 struct {
	Ref     string `json:"ref"`
	Version uint64 `json:"version"`
}
type EntradaPlanB2 struct {
	ExpedienteRef       string            `json:"expediente_ref"`
	VersionExpediente   uint64            `json:"version_expediente"`
	PuestoRef           string            `json:"puesto_ref"`
	PlazaRef            string            `json:"plaza_ref"`
	VersionPlantillaRef string            `json:"version_plantilla_ref"`
	VersionRPTRef       string            `json:"version_rpt_ref"`
	Regimen             EntradaCatalogoB2 `json:"regimen"`
	Modalidad           EntradaCatalogoB2 `json:"modalidad"`
	ClaseOcupacion      string            `json:"clase_ocupacion"`
	Desde               string            `json:"desde"`
	Hasta               string            `json:"hasta"`
	MotivoClave         string            `json:"motivo_clave"`
	DocumentoRef        string            `json:"documento_ref"`
	DocumentoSHA256     string            `json:"documento_sha256"`
	ClaveIdempotencia   string            `json:"clave_idempotencia"`
}
type EntradaConfirmacionB2 struct {
	ExpedienteRef     string `json:"expediente_ref"`
	PlanRef           string `json:"plan_ref"`
	VersionPlan       uint64 `json:"version_plan"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
}
type PrerrequisitoB2 struct {
	ClaveI18n string `json:"clave_i18n"`
	Cumplido  bool   `json:"cumplido"`
}
type OpcionVacanteB2 struct {
	PlazaRef            string `json:"plaza_ref"`
	PuestoRef           string `json:"puesto_ref"`
	VersionPlantillaRef string `json:"version_plantilla_ref"`
	VersionRPTRef       string `json:"version_rpt_ref"`
	UnidadRef           string `json:"unidad_ref"`
	CategoriaRef        string `json:"categoria_ref"`
	PlazaEtiqueta       string `json:"plaza_etiqueta"`
	PuestoEtiqueta      string `json:"puesto_etiqueta"`
}
type OpcionCatalogoB2 struct {
	Ref          string `json:"ref"`
	Version      uint64 `json:"version"`
	Denominacion string `json:"denominacion"`
}
type OpcionDocumentoB2 struct {
	DocumentoRef      string `json:"documento_ref"`
	DocumentoSHA256   string `json:"documento_sha256"`
	EtiquetaClaveI18n string `json:"etiqueta_clave_i18n"`
}
type OpcionClaseOcupacionB2 struct {
	Valor      string `json:"valor"`
	TextoClave string `json:"texto_clave"`
}
type CatalogoClasesOcupacionB2 struct {
	Ref          string `json:"ref"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}
type PeriodoOpcionesB2 struct {
	Desde     string `json:"desde"`
	Hasta     string `json:"hasta"`
	FuenteRef string `json:"fuente_ref"`
}
type OpcionesIncorporacionPersonalB2 struct {
	Vacantes                []OpcionVacanteB2         `json:"vacantes"`
	Regimenes               []OpcionCatalogoB2        `json:"regimenes"`
	Modalidades             []OpcionCatalogoB2        `json:"modalidades"`
	ClasesOcupacion         []OpcionClaseOcupacionB2  `json:"clases_ocupacion"`
	CatalogoClasesOcupacion CatalogoClasesOcupacionB2 `json:"catalogo_clases_ocupacion"`
	Motivos                 []string                  `json:"motivos"`
	Documentos              []OpcionDocumentoB2       `json:"documentos"`
	Periodo                 PeriodoOpcionesB2         `json:"periodo"`
}
type PlanIncorporacionPersonalB2HTTP struct {
	PlanRef   string        `json:"plan_ref"`
	Version   uint64        `json:"version"`
	SHA256    string        `json:"sha256"`
	Intencion EntradaPlanB2 `json:"intencion"`
}
type ReciboIncorporacionPersonalB2HTTP struct {
	Esquema                string `json:"esquema"`
	ExpedienteRef          string `json:"expediente_ref"`
	PlanRef                string `json:"plan_ref"`
	PlanVersion            uint64 `json:"plan_version"`
	ReciboRef              string `json:"recibo_ref"`
	RegistradaEn           string `json:"registrada_en"`
	EmpleadoRef            string `json:"empleado_ref"`
	RelacionRef            string `json:"relacion_ref"`
	OcupacionRef           string `json:"ocupacion_ref"`
	FirmaOficial           bool   `json:"firma_oficial"`
	EficaciaAdministrativa bool   `json:"eficacia_administrativa"`
}
type ProyeccionIncorporacionPersonalB2HTTP struct {
	Esquema                 string                             `json:"esquema"`
	ExpedienteRef           string                             `json:"expediente_ref"`
	VersionExpedienteActual uint64                             `json:"version_expediente_actual"`
	Estado                  string                             `json:"estado"`
	Prerrequisitos          []PrerrequisitoB2                  `json:"prerrequisitos"`
	Opciones                OpcionesIncorporacionPersonalB2    `json:"opciones"`
	Plan                    *PlanIncorporacionPersonalB2HTTP   `json:"plan"`
	Recibo                  *ReciboIncorporacionPersonalB2HTTP `json:"recibo"`
}

// Consultar es exclusivamente lectura. Preparar guarda intención prospectiva;
// Confirmar recupera ese plan y coteja su clave y versión antes de efectos.
// Cada operación exige autorización nominal vigente desde el contexto sellado.
type EjecutorIncorporacionPersonalB2 interface {
	Consultar(context.Context, string) (ProyeccionIncorporacionPersonalB2HTTP, error)
	Preparar(context.Context, EntradaPlanB2) (ProyeccionIncorporacionPersonalB2HTTP, error)
	Confirmar(context.Context, EntradaConfirmacionB2) (ReciboIncorporacionPersonalB2HTTP, error)
}

func (e EntradaPlanB2) Validar() error {
	if !patronClaseOcupacionHTTPB2.MatchString(e.ClaseOcupacion) {
		return ErrPeticionIncorporacionPersonalB2
	}
	for _, ref := range []string{e.ExpedienteRef, e.VersionPlantillaRef, e.VersionRPTRef, e.Regimen.Ref, e.Modalidad.Ref, e.DocumentoRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrPeticionIncorporacionPersonalB2
		}
	}
	if !referenciaOrganizativaHTTPB2(e.PlazaRef) || !referenciaOrganizativaHTTPB2(e.PuestoRef) || !versionHTTPB2(e.VersionExpediente) || !versionHTTPB2(e.Regimen.Version) || !versionHTTPB2(e.Modalidad.Version) || !periodoHTTPB2(e.Desde, e.Hasta, false) || !patronClaveHTTPB2.MatchString(e.MotivoClave) || !patronSHAHTTPB2.MatchString(e.DocumentoSHA256) || !patronUUIDHTTPB2.MatchString(e.ClaveIdempotencia) {
		return ErrPeticionIncorporacionPersonalB2
	}
	return nil
}
func (e EntradaConfirmacionB2) Validar() error {
	if !domain.ReferenciaOpacaValida(e.ExpedienteRef) || !domain.ReferenciaOpacaValida(e.PlanRef) || !versionHTTPB2(e.VersionPlan) || !patronUUIDHTTPB2.MatchString(e.ClaveIdempotencia) {
		return ErrPeticionIncorporacionPersonalB2
	}
	return nil
}
func versionHTTPB2(v uint64) bool { return v > 0 && v <= ports.MaximoEnteroSeguroOperacionAnalisis }
func referenciaOrganizativaHTTPB2(s string) bool {
	return domain.ReferenciaOpacaValida(s) || patronUUIDHTTPB2.MatchString(s)
}
func fechaHTTPB2(s string) bool {
	t, e := time.Parse("2006-01-02", s)
	return e == nil && t.Year() > 0 && t.Format("2006-01-02") == s
}
func periodoHTTPB2(desde, hasta string, admiteVacio bool) bool {
	return (admiteVacio && desde == "" && hasta == "") || (fechaHTTPB2(desde) && (hasta == "" || (fechaHTTPB2(hasta) && desde < hasta)))
}

func leerCuerpoHTTPB2(w http.ResponseWriter, r *http.Request, destino any, claves []string) error {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 || r.ContentLength > MaximoCuerpoIncorporacionPersonalB2Bytes || len(r.Trailer) != 0 || !transferenciaAltaPermitida(r.TransferEncoding) || !tipoContenidoJSON(r.Header) {
		return ErrPeticionIncorporacionPersonalB2
	}
	b, e := io.ReadAll(http.MaxBytesReader(w, r.Body, MaximoCuerpoIncorporacionPersonalB2Bytes))
	if r.Context().Err() != nil {
		return r.Context().Err()
	}
	if e != nil || len(b) == 0 || !utf8.Valid(b) || (r.ContentLength >= 0 && r.ContentLength != int64(len(b))) || validarJSONPropuestaFormalizacionSinDuplicados(b) != nil {
		return ErrPeticionIncorporacionPersonalB2
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != len(claves) {
		return ErrPeticionIncorporacionPersonalB2
	}
	for _, c := range claves {
		if _, ok := campos[c]; !ok {
			return ErrPeticionIncorporacionPersonalB2
		}
	}
	// Las selecciones de catálogo también tienen conjunto exacto de campos.
	for _, c := range []string{"regimen", "modalidad"} {
		if v, ok := campos[c]; ok {
			var m map[string]json.RawMessage
			if json.Unmarshal(v, &m) != nil || len(m) != 2 || m["ref"] == nil || m["version"] == nil {
				return ErrPeticionIncorporacionPersonalB2
			}
		}
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(&struct{}{}) != io.EOF {
		return ErrPeticionIncorporacionPersonalB2
	}
	return nil
}

func instanteHTTPB2(s string) bool {
	t, err := time.Parse(time.RFC3339Nano, s)
	return err == nil && strings.HasSuffix(s, "Z") && domain.InstanteUTCCanonico(t)
}

func reciboHTTPB2Valido(v ReciboIncorporacionPersonalB2HTTP, exp string) bool {
	if v.Esquema != EsquemaReciboIncorporacionPersonalB2 || v.ExpedienteRef != exp || !versionHTTPB2(v.PlanVersion) || v.FirmaOficial || v.EficaciaAdministrativa || !instanteHTTPB2(v.RegistradaEn) {
		return false
	}
	for _, ref := range []string{v.ExpedienteRef, v.PlanRef, v.ReciboRef, v.EmpleadoRef, v.RelacionRef, v.OcupacionRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return false
		}
	}
	return true
}
func proyeccionHTTPB2Valida(v ProyeccionIncorporacionPersonalB2HTTP, exp string) bool {
	if v.Esquema != EsquemaConsultaIncorporacionPersonalB2 || v.ExpedienteRef != exp || !domain.ReferenciaOpacaValida(exp) || !versionHTTPB2(v.VersionExpedienteActual) || len(v.Prerrequisitos) > 32 {
		return false
	}
	switch v.Estado {
	case "sin_plan":
		if v.Plan != nil || v.Recibo != nil {
			return false
		}
	case "plan_preparado":
		if v.Plan == nil || v.Recibo != nil {
			return false
		}
	case "incorporacion_confirmada":
		if v.Plan == nil || v.Recibo == nil {
			return false
		}
	default:
		return false
	}
	if v.Plan != nil && (!domain.ReferenciaOpacaValida(v.Plan.PlanRef) || !versionHTTPB2(v.Plan.Version) || !patronSHAHTTPB2.MatchString(v.Plan.SHA256) || v.Plan.Intencion.Validar() != nil || v.Plan.Intencion.ExpedienteRef != exp || v.Plan.Intencion.VersionExpediente > v.VersionExpedienteActual) {
		return false
	}
	if v.Recibo != nil && (!reciboHTTPB2Valido(*v.Recibo, exp) || v.Recibo.PlanRef != v.Plan.PlanRef || v.Recibo.PlanVersion != v.Plan.Version) {
		return false
	}
	for _, p := range v.Prerrequisitos {
		if !patronClaveHTTPB2.MatchString(p.ClaveI18n) {
			return false
		}
	}
	o := v.Opciones
	if len(o.ClasesOcupacion) > 100 {
		return false
	}
	if len(o.ClasesOcupacion) > 0 && (!domain.ReferenciaOpacaValida(o.CatalogoClasesOcupacion.Ref) || !versionHTTPB2(o.CatalogoClasesOcupacion.Version) || !domain.HuellaPlanPersonalB2Valida(o.CatalogoClasesOcupacion.HuellaSHA256)) {
		return false
	}
	clases := make(map[string]bool, len(o.ClasesOcupacion))
	for _, clase := range o.ClasesOcupacion {
		if !patronClaseOcupacionHTTPB2.MatchString(clase.Valor) || !patronClaveHTTPB2.MatchString(clase.TextoClave) || clases[clase.Valor] {
			return false
		}
		clases[clase.Valor] = true
	}
	if len(o.Vacantes) > 100 || len(o.Regimenes) > 100 || len(o.Modalidades) > 100 || len(o.Motivos) > 32 || len(o.Documentos) > 32 || !periodoHTTPB2(o.Periodo.Desde, o.Periodo.Hasta, true) || (o.Periodo.FuenteRef != "" && !domain.ReferenciaOpacaValida(o.Periodo.FuenteRef)) {
		return false
	}
	for _, x := range o.Vacantes {
		if !referenciaOrganizativaHTTPB2(x.PlazaRef) || !referenciaOrganizativaHTTPB2(x.PuestoRef) || !domain.ReferenciaOpacaValida(x.VersionPlantillaRef) || !domain.ReferenciaOpacaValida(x.VersionRPTRef) || !domain.ReferenciaOpacaValida(x.UnidadRef) || !domain.ReferenciaOpacaValida(x.CategoriaRef) || !etiquetaHTTPB2(x.PlazaEtiqueta) || !etiquetaHTTPB2(x.PuestoEtiqueta) {
			return false
		}
	}
	for _, lista := range [][]OpcionCatalogoB2{o.Regimenes, o.Modalidades} {
		for _, x := range lista {
			if !domain.ReferenciaOpacaValida(x.Ref) || !versionHTTPB2(x.Version) || !etiquetaHTTPB2(x.Denominacion) {
				return false
			}
		}
	}
	for _, x := range o.Motivos {
		if !patronClaveHTTPB2.MatchString(x) {
			return false
		}
	}
	for _, x := range o.Documentos {
		if !domain.ReferenciaOpacaValida(x.DocumentoRef) || !patronSHAHTTPB2.MatchString(x.DocumentoSHA256) || !patronClaveHTTPB2.MatchString(x.EtiquetaClaveI18n) {
			return false
		}
	}
	return true
}
func etiquetaHTTPB2(s string) bool {
	return len(s) > 0 && len(s) <= 320 && utf8.ValidString(s) && !strings.ContainsAny(s, "\x00\r\n")
}
