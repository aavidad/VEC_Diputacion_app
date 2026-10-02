package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"
)

var ErrJustificacionInvalida = errors.New("cronos: justificacion invalida")
var ErrJustificacionConflicto = errors.New("cronos: justificacion en conflicto")

type EstadoJustificacion string

const (
	JustificacionPendiente EstadoJustificacion = "pendiente_revision"
	JustificacionAceptada  EstadoJustificacion = "aceptada"
	JustificacionRechazada EstadoJustificacion = "rechazada"
)

// Referencias y metadatos; nunca contenido ni motivos libres.
type DocumentoJustificacion struct {
	ID          string `json:"id"`
	Version     uint64 `json:"version"`
	SHA256      string `json:"sha256"`
	CustodioID  string `json:"custodio_id"`
	CustodiaRef string `json:"custodia_ref"`
}
type VinculoJustificacion struct {
	SolicitudRef            string                 `json:"solicitud_ref"`
	EmpleadoRef             string                 `json:"empleado_ref"`
	CatalogoVersionRef      string                 `json:"catalogo_version_ref"`
	PermisoRef              string                 `json:"permiso_ref"`
	ExpedienteDocumentalRef string                 `json:"expediente_documental_ref"`
	Documento               DocumentoJustificacion `json:"documento"`
}

// Esta proyección procede de una lectura autorizada, no de la petición.
type SolicitudJustificable struct {
	SolicitudRef            string                 `json:"solicitud_ref"`
	EmpleadoRef             string                 `json:"empleado_ref"`
	CatalogoVersionRef      string                 `json:"catalogo_version_ref"`
	PermisoRef              string                 `json:"permiso_ref"`
	ExpedienteDocumentalRef string                 `json:"expediente_documental_ref"`
	Version                 int64                  `json:"version"`
	Estado                  EstadoSolicitudPermiso `json:"estado"`
	JustificanteExigido     bool                   `json:"justificante_exigido"`
}
type PoliticaJustificacion struct {
	Referencia         string   `json:"referencia"`
	Version            uint64   `json:"version"`
	SHA256             string   `json:"sha256"`
	CatalogoVersionRef string   `json:"catalogo_version_ref"`
	PermisoRef         string   `json:"permiso_ref"`
	TipoDocumentalRef  string   `json:"tipo_documental_ref"`
	CustodioID         string   `json:"custodio_id"`
	MotivosRef         []string `json:"motivos_ref"`
}
type Justificacion struct {
	Vinculo   VinculoJustificacion `json:"vinculo"`
	Version   int64                `json:"version"`
	Estado    EstadoJustificacion  `json:"estado"`
	MotivoRef string               `json:"motivo_ref,omitempty"`
}

var opacaJustificacion = regexp.MustCompile(`^(ref:[0-9a-f]{64}|[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$`)
var custodiaJustificacion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:#-]{2,159}$`)
var tecnicoJustificacion = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)

func RefDocumentoJustificacionValida(s string) bool {
	return opacaJustificacion.MatchString(s) && s != "ref:0000000000000000000000000000000000000000000000000000000000000000"
}
func (d DocumentoJustificacion) Validar() error {
	if !RefDocumentoJustificacionValida(d.ID) || d.Version == 0 || !HuellaEfectosValida(d.SHA256) || !tecnicoJustificacion.MatchString(d.CustodioID) || !custodiaJustificacion.MatchString(d.CustodiaRef) || strings.Contains(d.CustodiaRef, "..") {
		return ErrJustificacionInvalida
	}
	return nil
}
func (p PoliticaJustificacion) Validar() error {
	if !referenciaMarcaje(p.Referencia) || p.Version == 0 || !HuellaEfectosValida(p.SHA256) || !referenciaMarcaje(p.CatalogoVersionRef) || !referenciaMarcaje(p.PermisoRef) || !RefDocumentoJustificacionValida(p.TipoDocumentalRef) || !tecnicoJustificacion.MatchString(p.CustodioID) || len(p.MotivosRef) == 0 || len(p.MotivosRef) > 32 {
		return ErrJustificacionInvalida
	}
	vistos := map[string]bool{}
	for _, m := range p.MotivosRef {
		if !referenciaMarcaje(m) || vistos[m] {
			return ErrJustificacionInvalida
		}
		vistos[m] = true
	}
	return nil
}
func (s SolicitudJustificable) Validar(p PoliticaJustificacion) error {
	if p.Validar() != nil || !SolicitudPermisoRefValida(s.SolicitudRef) || !referenciaIdentidadMarcaje(s.EmpleadoRef, "emp_") || !RefDocumentoJustificacionValida(s.ExpedienteDocumentalRef) || s.CatalogoVersionRef != p.CatalogoVersionRef || s.PermisoRef != p.PermisoRef || s.Version < 1 || s.Estado != EstadoPermisoConcedido || !s.JustificanteExigido {
		return ErrJustificacionInvalida
	}
	return nil
}
func (v VinculoJustificacion) Validar(s SolicitudJustificable, p PoliticaJustificacion) error {
	if s.Validar(p) != nil || v.Documento.Validar() != nil || v.SolicitudRef != s.SolicitudRef || v.EmpleadoRef != s.EmpleadoRef || v.CatalogoVersionRef != s.CatalogoVersionRef || v.PermisoRef != s.PermisoRef || v.ExpedienteDocumentalRef != s.ExpedienteDocumentalRef || v.Documento.CustodioID != p.CustodioID {
		return ErrJustificacionInvalida
	}
	return nil
}
func (j Justificacion) Validar(s SolicitudJustificable, p PoliticaJustificacion) error {
	if j.Vinculo.Validar(s, p) != nil || j.Version < 1 {
		return ErrJustificacionInvalida
	}
	if j.Estado == JustificacionPendiente && j.MotivoRef == "" {
		return nil
	}
	if j.Estado != JustificacionAceptada && j.Estado != JustificacionRechazada {
		return ErrJustificacionInvalida
	}
	for _, m := range p.MotivosRef {
		if j.MotivoRef == m {
			return nil
		}
	}
	return ErrJustificacionInvalida
}

// El repositorio añadirá este hecho a la historia; no reemplazará el anterior.
// Estas funciones puras no conceden permisos ni computan saldos.
func PrepararAnexoJustificacion(s SolicitudJustificable, p PoliticaJustificacion, actual *Justificacion, v VinculoJustificacion, version int64) (Justificacion, error) {
	if v.Validar(s, p) != nil || version < 0 || version == math.MaxInt64 {
		return Justificacion{}, ErrJustificacionInvalida
	}
	if actual == nil {
		if version != 0 {
			return Justificacion{}, ErrJustificacionConflicto
		}
	} else if actual.Validar(s, p) != nil || actual.Version != version || actual.Vinculo.Documento == v.Documento {
		return Justificacion{}, ErrJustificacionConflicto
	}
	return Justificacion{Vinculo: v, Version: version + 1, Estado: JustificacionPendiente}, nil
}
func PrepararRevisionJustificacion(s SolicitudJustificable, p PoliticaJustificacion, actual Justificacion, v VinculoJustificacion, version int64, decision EstadoJustificacion, motivo string) (Justificacion, error) {
	if actual.Validar(s, p) != nil || v.Validar(s, p) != nil || version < 1 || version == math.MaxInt64 {
		return Justificacion{}, ErrJustificacionInvalida
	}
	if actual.Version != version || actual.Vinculo != v || actual.Estado != JustificacionPendiente {
		return Justificacion{}, ErrJustificacionConflicto
	}
	siguiente := Justificacion{Vinculo: v, Version: version + 1, Estado: decision, MotivoRef: motivo}
	if (decision != JustificacionAceptada && decision != JustificacionRechazada) || siguiente.Validar(s, p) != nil {
		return Justificacion{}, ErrJustificacionInvalida
	}
	return siguiente, nil
}

type MaterialJustificacion struct {
	ActorRef         string               `json:"actor_ref"`
	PerfilRef        string               `json:"perfil_ref"`
	ClaveOperacion   string               `json:"clave_operacion"`
	Accion           string               `json:"accion"`
	SolicitudVersion int64                `json:"solicitud_version"`
	VersionEsperada  int64                `json:"version_esperada"`
	PoliticaRef      string               `json:"politica_ref"`
	PoliticaVersion  uint64               `json:"politica_version"`
	PoliticaSHA256   string               `json:"politica_sha256"`
	Vinculo          VinculoJustificacion `json:"vinculo"`
	Decision         EstadoJustificacion  `json:"decision,omitempty"`
	MotivoRef        string               `json:"motivo_ref,omitempty"`
}

const AccionAnexarJustificacion = "cronos.justificacion.anexar"
const AccionRevisarJustificacion = "cronos.justificacion.revisar"

func (m MaterialJustificacion) Canonico() ([]byte, error) {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") || !RefDocumentoJustificacionValida(m.ClaveOperacion) || m.SolicitudVersion < 1 || m.VersionEsperada < 0 || m.VersionEsperada == math.MaxInt64 || !referenciaMarcaje(m.PoliticaRef) || m.PoliticaVersion == 0 || !HuellaEfectosValida(m.PoliticaSHA256) ||
		!SolicitudPermisoRefValida(m.Vinculo.SolicitudRef) || !referenciaIdentidadMarcaje(m.Vinculo.EmpleadoRef, "emp_") || !referenciaMarcaje(m.Vinculo.CatalogoVersionRef) || !referenciaMarcaje(m.Vinculo.PermisoRef) || !RefDocumentoJustificacionValida(m.Vinculo.ExpedienteDocumentalRef) || m.Vinculo.Documento.Validar() != nil {
		return nil, ErrJustificacionInvalida
	}
	switch m.Accion {
	case AccionAnexarJustificacion:
		if m.Decision != "" || m.MotivoRef != "" {
			return nil, ErrJustificacionInvalida
		}
	case AccionRevisarJustificacion:
		if (m.Decision != JustificacionAceptada && m.Decision != JustificacionRechazada) || !referenciaMarcaje(m.MotivoRef) || m.VersionEsperada < 1 {
			return nil, ErrJustificacionInvalida
		}
	default:
		return nil, ErrJustificacionInvalida
	}
	return json.Marshal(m)
}
func (m MaterialJustificacion) Huella() (string, error) {
	b, e := m.Canonico()
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
