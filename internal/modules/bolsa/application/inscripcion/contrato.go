// Package inscripcion coordina la presentación y revisión de solicitudes de
// entrada en Bolsa. La identidad procede de la frontera confiable del servidor.
package inscripcion

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrSolicitudInvalida   = errors.New("bolsa inscripcion: solicitud invalida")
	ErrNoDisponible        = errors.New("bolsa inscripcion: servicio no disponible")
	ErrNoEncontrada        = errors.New("bolsa inscripcion: solicitud no encontrada")
	ErrConflicto           = errors.New("bolsa inscripcion: conflicto")
	ErrPlazoCerrado        = errors.New("bolsa inscripcion: plazo cerrado")
	ErrAccesoDenegado      = errors.New("bolsa inscripcion: acceso denegado")
	ErrSesionAusente       = errors.New("bolsa inscripcion: sesion ausente")
	ErrActaNoDisponible    = errors.New("bolsa inscripcion: acta no disponible")
	ErrCatalogoCambiado    = errors.New("bolsa inscripcion: catalogo cambiado")
	ErrRequisitoInvalido   = errors.New("bolsa inscripcion: requisito invalido")
	ErrDeclaracionInvalida = errors.New("bolsa inscripcion: declaracion invalida")
	ErrSolicitudExistente  = errors.New("bolsa inscripcion: solicitud existente")
	ErrClaveConflicto      = errors.New("bolsa inscripcion: clave en conflicto")
)

const (
	EstadoPendiente             = "pendiente"
	EstadoAdmitidaAConvocatoria = "admitida_a_convocatoria"
	EstadoIncorporada           = "incorporada"
	EstadoRechazada             = "rechazada"
)

const (
	AccionListarAbiertas = "bolsa.inscripcion.convocatorias.listar"
	AccionDetalleAbierta = "bolsa.inscripcion.convocatoria.consultar"
	AccionPresentar      = "bolsa.inscripcion.presentar"
	AccionListarPropias  = "bolsa.inscripcion.propias.listar"
	AccionDetallePropia  = "bolsa.inscripcion.propia.consultar"
	AccionListarRRHH     = "bolsa.inscripcion.rrhh.listar"
	AccionDetalleRRHH    = "bolsa.inscripcion.rrhh.consultar"
	AccionMotivosRRHH    = "bolsa.inscripcion.rrhh.motivos"
	AccionDecidir        = "bolsa.inscripcion.rrhh.decidir"
	AccionIncorporar     = "bolsa.inscripcion.rrhh.incorporar"
)

var referenciaOpaca = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._-]{2,255}$`)
var referenciaConvocatoria = regexp.MustCompile(`^cv1_[A-Za-z0-9_-]+_v[1-9][0-9]*$`)
var referenciaSolicitud = regexp.MustCompile(`^solicitud_inscripcion_[0-9a-f]{64}$`)
var claveIdempotencia = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)
var huellaCertificado = regexp.MustCompile(`^[0-9a-f]{64}$`)

func convocatoriaRefValida(ref string) bool {
	return len(ref) <= 200 && referenciaConvocatoria.MatchString(ref)
}
func solicitudRefValida(ref string) bool { return referenciaSolicitud.MatchString(ref) }

// Actor sólo se construye a partir de la sesión vinculada al certificado.
// PersonaRef no forma parte de ningún DTO de petición.
type Actor struct {
	PersonaRef        string
	PerfilRef         string
	SesionRef         string
	Idioma            string
	Canal             string
	ResultadoContexto dominiovec.ResultadoContextoActorRegistradoV2
	Vinculo           dominiovec.VinculoAutenticacionActorV2
	Lectura           *CapturaLectura
	MaterialEscritura *puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (a Actor) Valido() bool {
	return referenciaOpaca.MatchString(a.PersonaRef) &&
		referenciaOpaca.MatchString(a.PerfilRef) &&
		referenciaOpaca.MatchString(a.SesionRef) && (a.Idioma == "" || a.Idioma == "es" || a.Idioma == "en") &&
		(a.Canal == "externa_personal" || a.Canal == "interna_corporativa") &&
		a.ResultadoContexto.Validar() == nil && a.Vinculo.ValidarPara(a.ResultadoContexto) == nil &&
		a.ResultadoContexto.Contexto.PersonaRef == a.PersonaRef &&
		a.ResultadoContexto.Contexto.PerfilActivoRef == a.PerfilRef
}

// CapturaLectura es una decisión cerrada de la sesión en memoria, ligada a la
// autenticación certificada y al recurso/filtro exacto. SQL recibe los datos
// ya resueltos, restringe por ellos y asienta un acceso nominal en la misma TX.
type CapturaLectura struct {
	PersonaRef              string
	PerfilRef               string
	CuentaRef               string
	SesionRef               string
	AutenticacionRef        string
	CertificadoHuellaSHA256 string
	Canal                   string
	Accion                  string
	RecursoRef              string
	Finalidad               string
	CorrelacionRef          string
	RevisionPermisos        uint64
	Filtro                  Filtro
	EmitidaEn               time.Time
	ValidaHasta             time.Time
}

func (a Actor) LecturaValida(accion, recurso string, filtro Filtro) bool {
	c := a.Lectura
	ahora := time.Now().UTC()
	if c == nil || (strings.Contains(accion, ".rrhh.") && c.Canal != "interna_corporativa") {
		return false
	}
	return a.Valido() && c != nil && c.PersonaRef == a.PersonaRef &&
		c.PerfilRef == a.PerfilRef && c.SesionRef == a.SesionRef && c.Canal == a.Canal &&
		c.CuentaRef == a.ResultadoContexto.Contexto.Instantanea.CuentaRef &&
		c.Accion == accion && c.RecursoRef == recurso && c.Filtro == filtro &&
		c.Finalidad != "" && (c.Canal == "externa_personal" || c.Canal == "interna_corporativa") &&
		referenciaOpaca.MatchString(c.CorrelacionRef) &&
		referenciaOpaca.MatchString(c.AutenticacionRef) && huellaCertificado.MatchString(c.CertificadoHuellaSHA256) &&
		c.RevisionPermisos > 0 && !c.EmitidaEn.IsZero() && !c.EmitidaEn.After(ahora) &&
		c.ValidaHasta.After(ahora)
}

func (a Actor) EscrituraValida() bool {
	return a.Valido() && a.MaterialEscritura != nil && a.MaterialEscritura.ValidarEstructura() == nil
}

type Presentacion struct {
	ConvocatoriaRef   string
	CategoriaRef      string
	CatalogoVersion   uint64
	ClaveIdempotencia string
	Declaraciones     []Declaracion
}

// Declaracion transporta sólo una clave de requisito y una referencia a
// evidencia ya custodiada. Declarar un dato no acredita su cumplimiento.
type Declaracion struct {
	RequisitoCodigo string `json:"requisito_codigo"`
	EvidenciaRef    string `json:"evidencia_ref,omitempty"`
}

func (p Presentacion) Validar() error {
	if !convocatoriaRefValida(p.ConvocatoriaRef) || !referenciaOpaca.MatchString(p.CategoriaRef) || len(p.CategoriaRef) > 200 || p.CatalogoVersion == 0 ||
		!claveIdempotencia.MatchString(p.ClaveIdempotencia) || len(p.Declaraciones) > 32 {
		return ErrSolicitudInvalida
	}
	vistos := make(map[string]struct{}, len(p.Declaraciones))
	for _, d := range p.Declaraciones {
		if !referenciaOpaca.MatchString(d.RequisitoCodigo) ||
			(d.EvidenciaRef != "" && !referenciaOpaca.MatchString(d.EvidenciaRef)) {
			return ErrSolicitudInvalida
		}
		if _, repetido := vistos[d.RequisitoCodigo]; repetido {
			return ErrSolicitudInvalida
		}
		vistos[d.RequisitoCodigo] = struct{}{}
	}
	return nil
}

type Decision struct {
	SolicitudRef      string
	Tipo              string
	MotivoCodigo      string
	VersionEsperada   uint64
	ClaveIdempotencia string
}

// Incorporacion solicita enlazar la admisión con una participación de acta
// aprobada. El cliente sólo indica la evidencia; Bolsa resuelve y comprueba
// acta, persona y posición desde sus fuentes autoritativas.
type Incorporacion struct {
	SolicitudRef      string
	EvidenciaRef      string
	VersionEsperada   uint64
	ClaveIdempotencia string
}

func (i Incorporacion) Validar() error {
	if !solicitudRefValida(i.SolicitudRef) ||
		!referenciaOpaca.MatchString(i.EvidenciaRef) || i.VersionEsperada == 0 ||
		!claveIdempotencia.MatchString(i.ClaveIdempotencia) {
		return ErrSolicitudInvalida
	}
	return nil
}

func (d Decision) Validar() error {
	if !solicitudRefValida(d.SolicitudRef) || d.VersionEsperada == 0 ||
		!claveIdempotencia.MatchString(d.ClaveIdempotencia) ||
		(d.Tipo != "admitir" && d.Tipo != "rechazar") ||
		(d.Tipo == "rechazar" && !referenciaOpaca.MatchString(d.MotivoCodigo)) ||
		(d.MotivoCodigo != "" && !referenciaOpaca.MatchString(d.MotivoCodigo)) {
		return ErrSolicitudInvalida
	}
	return nil
}

type Solicitud struct {
	SolicitudRef    string      `json:"solicitud_ref"`
	ReciboRef       string      `json:"recibo_ref"`
	ConvocatoriaRef string      `json:"convocatoria_ref"`
	CategoriaRef    string      `json:"categoria_ref"`
	BolsaRef        *string     `json:"bolsa_ref,omitempty"`
	Categoria       string      `json:"categoria"`
	DeclaracionRef  string      `json:"declaracion_ref,omitempty"`
	BasesRef        string      `json:"bases_ref,omitempty"`
	CatalogoVersion uint64      `json:"catalogo_version,omitempty"`
	PlazoInicio     *time.Time  `json:"plazo_inicio,omitempty"`
	PlazoFin        *time.Time  `json:"plazo_fin,omitempty"`
	Requisitos      []Requisito `json:"requisitos,omitempty"`
	PersonaResumen  string      `json:"persona_resumen,omitempty"`
	DecisionRef     string      `json:"decision_ref,omitempty"`
	Estado          string      `json:"estado"`
	Version         uint64      `json:"version"`
	RegistradaEn    time.Time   `json:"registrada_en"`
	DecididaEn      *time.Time  `json:"decidida_en,omitempty"`
	MotivoCodigo    string      `json:"motivo_codigo,omitempty"`
	MotivoEtiqueta  string      `json:"motivo_etiqueta,omitempty"`
	// ParticipacionRef sólo existe cuando la autoridad de Bolsa ha creado un
	// vínculo a una participación válida del orden vigente.
	ParticipacionRef string `json:"participacion_ref,omitempty"`
}

func (s Solicitud) Validar() error {
	if !solicitudRefValida(s.SolicitudRef) || !referenciaOpaca.MatchString(s.ReciboRef) ||
		!convocatoriaRefValida(s.ConvocatoriaRef) || !referenciaOpaca.MatchString(s.CategoriaRef) || len(s.CategoriaRef) > 200 ||
		s.Categoria == "" || len(s.Categoria) > 200 ||
		s.Version == 0 || s.RegistradaEn.IsZero() {
		return ErrSolicitudInvalida
	}
	switch s.Estado {
	case EstadoPendiente:
		if s.DecididaEn != nil || s.MotivoCodigo != "" || s.ParticipacionRef != "" {
			return ErrSolicitudInvalida
		}
	case EstadoAdmitidaAConvocatoria:
		if s.DecididaEn == nil || s.ParticipacionRef != "" {
			return ErrSolicitudInvalida
		}
	case EstadoIncorporada:
		if s.DecididaEn == nil || !referenciaOpaca.MatchString(s.ParticipacionRef) {
			return ErrSolicitudInvalida
		}
	case EstadoRechazada:
		if s.DecididaEn == nil || s.MotivoCodigo == "" || s.ParticipacionRef != "" {
			return ErrSolicitudInvalida
		}
	default:
		return ErrSolicitudInvalida
	}
	return nil
}

type Recibo struct {
	Solicitud
	Repetida bool `json:"repetida"`
}

type Filtro struct {
	Estado          string
	ConvocatoriaRef string
	Limite          int
	Cursor          string
}

func (f Filtro) Validar() error {
	if f.Limite < 1 || f.Limite > 100 ||
		(f.Estado != "" && f.Estado != EstadoPendiente && f.Estado != EstadoAdmitidaAConvocatoria && f.Estado != EstadoIncorporada && f.Estado != EstadoRechazada) ||
		(f.ConvocatoriaRef != "" && !convocatoriaRefValida(f.ConvocatoriaRef)) || len(f.Cursor) > 512 {
		return ErrSolicitudInvalida
	}
	return nil
}

type Pagina struct {
	Solicitudes     []Solicitud `json:"solicitudes"`
	Total           uint64      `json:"total"`
	CursorSiguiente *string     `json:"cursor_siguiente"`
}

type Requisito struct {
	Codigo           string     `json:"codigo"`
	Descripcion      string     `json:"descripcion"`
	Obligatorio      bool       `json:"obligatorio"`
	Estado           string     `json:"estado"`
	MotivoCodigo     string     `json:"motivo_codigo"`
	MotivoEtiqueta   string     `json:"motivo_etiqueta"`
	ProcedenciaRef   string     `json:"procedencia_ref,omitempty"`
	HitoCumplimiento *string    `json:"hito_cumplimiento"`
	HitoEtiqueta     *string    `json:"hito_etiqueta"`
	HitoFecha        *time.Time `json:"hito_fecha,omitempty"`
}

type Categoria struct {
	CategoriaRef string `json:"categoria_ref"`
	Categoria    string `json:"categoria"`
}

type BolsaAbierta struct {
	ConvocatoriaRef     string      `json:"convocatoria_ref"`
	Titulo              string      `json:"titulo"`
	CategoriasResumen   string      `json:"categorias_resumen"`
	Categorias          []Categoria `json:"categorias"`
	PlazoInicio         time.Time   `json:"plazo_inicio"`
	PlazoFin            time.Time   `json:"plazo_fin"`
	CatalogoVersion     uint64      `json:"catalogo_version"`
	RequisitosResumen   string      `json:"requisitos_resumen"`
	Requisitos          []Requisito `json:"requisitos,omitempty"`
	PuedeIniciar        bool        `json:"puede_iniciar"`
	ImpedimentoEtiqueta string      `json:"impedimento_etiqueta,omitempty"`
	EstadoPropio        *string     `json:"estado_solicitud_propia"`
	SolicitudRef        *string     `json:"solicitud_ref"`
}

type PaginaAbiertas struct {
	Bolsas          []BolsaAbierta `json:"convocatorias"`
	Total           uint64         `json:"total"`
	CursorSiguiente *string        `json:"cursor_siguiente"`
}

type Motivo struct {
	Codigo      string `json:"codigo"`
	Etiqueta    string `json:"etiqueta"`
	Obligatorio bool   `json:"obligatorio"`
}

type CatalogoMotivos struct {
	Version uint64   `json:"catalogo_version"`
	Motivos []Motivo `json:"motivos"`
}

// Repositorio conserva en la misma transacción la decisión V3, el efecto,
// la historia y la auditoría común. Las lecturas consumen la sesión vigente
// y registran una auditoría nominal por petición.
type Repositorio interface {
	Abiertas(context.Context, Actor, int, string) (PaginaAbiertas, error)
	DetalleAbierta(context.Context, Actor, string) (BolsaAbierta, error)
	Presentar(context.Context, Actor, Presentacion) (Recibo, error)
	Propias(context.Context, Actor, Filtro) (Pagina, error)
	Propia(context.Context, Actor, string) (Solicitud, error)
	PendientesRRHH(context.Context, Actor, Filtro) (Pagina, error)
	DetalleRRHH(context.Context, Actor, string) (Solicitud, error)
	MotivosRRHH(context.Context, Actor, string) (CatalogoMotivos, error)
	Decidir(context.Context, Actor, Decision) (Recibo, error)
	Incorporar(context.Context, Actor, Incorporacion) (Recibo, error)
}
