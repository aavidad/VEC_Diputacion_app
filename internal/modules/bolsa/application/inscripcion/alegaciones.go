package inscripcion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const AccionPresentarAlegacion = "bolsa.inscripcion.alegacion.presentar"
const EsquemaMaterialAlegacion = "vec.bolsa.inscripcion.alegacion.presentar.v1"

var referenciaActoBolsaAlegacion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._-]{2,511}$`)

const (
	AccionObjetosAlegacion     = "bolsa.alegacion.propias.consultar"
	AccionAlegacionesPropias   = AccionObjetosAlegacion
	AccionAlegacionesRRHH      = "bolsa.inscripcion.alegacion.rrhh.consultar"
	EsquemaObjetosAlegacion    = "vec.bolsa.alegaciones.objetos.v1"
	EsquemaAlegacionPresentada = "vec.bolsa.alegacion.presentada.v1"
	EsquemaAlegacionesPropias  = "vec.bolsa.alegaciones.propias.v1"
	EsquemaAlegacionesRRHH     = "vec.bolsa.alegaciones.rrhh.v1"
	EsquemaDetalleAlegacion    = "vec.bolsa.alegacion.detalle.v1"
	MaximoVersionActoAlegacion = 9007199254740991
)

// La solicitud y el acto se cotejan en el repositorio propietario. La persona
// y la participación se resuelven allí desde el actor certificado; el cliente
// nunca decide a quién pertenece la alegación.
type ObjetoAlegacion struct {
	Tipo             string `json:"tipo"`
	SolicitudRef     string `json:"solicitud_ref,omitempty"`
	ParticipacionRef string `json:"participacion_ref,omitempty"`
	ActoRef          string `json:"acto_ref"`
	VersionActo      uint64 `json:"version_acto"`
}

func (o ObjetoAlegacion) Validar() error {
	if !referenciaActoBolsaAlegacion.MatchString(o.ActoRef) || o.VersionActo == 0 || o.VersionActo > MaximoVersionActoAlegacion {
		return ErrSolicitudInvalida
	}
	switch o.Tipo {
	case "decision_inscripcion":
		if !solicitudRefValida(o.SolicitudRef) || o.ParticipacionRef != "" {
			return ErrSolicitudInvalida
		}
	case "acto_participacion":
		if o.SolicitudRef != "" || !referenciaActoBolsaAlegacion.MatchString(o.ParticipacionRef) {
			return ErrSolicitudInvalida
		}
	default:
		return ErrSolicitudInvalida
	}
	return nil
}

// La operación se liga a la persona acreditada y a la clave, no a un dato del
// navegador. La misma clave recupera una única operación tras un corte HTTP.
func ReferenciaAlegacion(personaRef, clave string) (string, error) {
	if !referenciaOpaca.MatchString(personaRef) || !claveIdempotencia.MatchString(clave) {
		return "", ErrSolicitudInvalida
	}
	h := sha256.Sum256([]byte("vec.bolsa.alegacion.v1\x1f" + personaRef + "\x1f" + clave))
	return "alegacion_" + hex.EncodeToString(h[:]), nil
}

type ObjetoSeleccionAlegacion struct {
	ObjetoAlegacion
	Titulo          string    `json:"titulo"`
	Estado          string    `json:"estado"`
	OcurridoEn      time.Time `json:"ocurrido_en"`
	ConvocatoriaRef string    `json:"convocatoria_ref,omitempty"`
	BolsaRef        string    `json:"bolsa_ref,omitempty"`
}

func (o ObjetoSeleccionAlegacion) Validar() error {
	if o.ObjetoAlegacion.Validar() != nil || strings.TrimSpace(o.Titulo) == "" ||
		utf8.RuneCountInString(o.Titulo) > 180 || o.OcurridoEn.IsZero() {
		return ErrNoDisponible
	}
	if o.Tipo == "decision_inscripcion" {
		if o.VersionActo < 2 || (o.Estado != EstadoAdmitidaAConvocatoria && o.Estado != EstadoRechazada) ||
			!convocatoriaRefValida(o.ConvocatoriaRef) || o.BolsaRef != "" {
			return ErrNoDisponible
		}
	} else if o.Estado != "constituida" || !referenciaActoBolsaAlegacion.MatchString(o.BolsaRef) || o.ConvocatoriaRef != "" {
		return ErrNoDisponible
	}
	return nil
}

type PaginaObjetosAlegacion struct {
	Objetos         []ObjetoSeleccionAlegacion `json:"objetos"`
	Total           uint64                     `json:"total"`
	CursorSiguiente *string                    `json:"cursor_siguiente"`
}

type AmbitoAlegacionRRHH struct {
	Tipo       string `json:"tipo"`
	Referencia string `json:"referencia"`
	Titulo     string `json:"titulo"`
}

type PaginaAmbitosAlegacionRRHH struct {
	Ambitos         []AmbitoAlegacionRRHH `json:"ambitos"`
	Total           uint64                `json:"total"`
	CursorSiguiente *string               `json:"cursor_siguiente"`
}

type PresentacionAlegacion struct {
	Objeto            ObjetoAlegacion `json:"objeto"`
	Fundamento        string          `json:"fundamento"`
	ClaveIdempotencia string          `json:"clave_idempotencia"`
}

func (p PresentacionAlegacion) Validar() error {
	if p.Objeto.Validar() != nil || !claveIdempotencia.MatchString(p.ClaveIdempotencia) ||
		!utf8.ValidString(p.Fundamento) || strings.TrimSpace(p.Fundamento) == "" ||
		utf8.RuneCountInString(p.Fundamento) > 4000 || strings.ContainsRune(p.Fundamento, '\x00') {
		return ErrSolicitudInvalida
	}
	return nil
}

func MaterialAlegacion(p PresentacionAlegacion) ([]byte, string, error) {
	if err := p.Validar(); err != nil {
		return nil, "", err
	}
	contenido, err := json.Marshal(struct {
		Esquema string `json:"esquema"`
		PresentacionAlegacion
	}{EsquemaMaterialAlegacion, p})
	if err != nil {
		return nil, "", ErrSolicitudInvalida
	}
	huella := sha256.Sum256(contenido)
	return contenido, hex.EncodeToString(huella[:]), nil
}

func RecursoPresentacionAlegacion(p PresentacionAlegacion, materialSHA256 string, ambitos map[string]string) ([]byte, error) {
	if p.Validar() != nil || len(materialSHA256) != 64 || len(ambitos) != 1 || ambitos["candidato_ref"] == "" {
		return nil, ErrSolicitudInvalida
	}
	if _, err := hex.DecodeString(materialSHA256); err != nil {
		return nil, ErrSolicitudInvalida
	}
	atributos := map[string]string{"tipo_objeto": p.Objeto.Tipo, "acto_ref": p.Objeto.ActoRef,
		"version_acto": strconv.FormatUint(p.Objeto.VersionActo, 10), "material_sha256": materialSHA256}
	if p.Objeto.SolicitudRef != "" {
		atributos["solicitud_ref"] = p.Objeto.SolicitudRef
	} else {
		atributos["participacion_ref"] = p.Objeto.ParticipacionRef
	}
	return recursoEscrituraInscripcion(ambitos, atributos)
}

func recursoLecturaAlegacion(accion, personaRef, idioma string, filtro Filtro, ref string) (string, error) {
	if !referenciaOpaca.MatchString(personaRef) || (idioma != "es" && idioma != "en") ||
		filtro.Estado != "" || filtro.ConvocatoriaRef != "" || filtro.Limite < 1 || filtro.Limite > 20 ||
		(filtro.Cursor != "" && !referenciaOpaca.MatchString(filtro.Cursor)) {
		return "", ErrSolicitudInvalida
	}
	if accion == AccionObjetosAlegacion {
		if ref != "objetos" && ref != "propias" {
			return "", ErrSolicitudInvalida
		}
	} else if accion != AccionAlegacionesRRHH ||
		(ref != "ambitos" && !referenciaActoBolsaAlegacion.MatchString(ref) &&
			!(strings.HasPrefix(ref, "bolsa:") && referenciaActoBolsaAlegacion.MatchString(strings.TrimPrefix(ref, "bolsa:"))) &&
			!(strings.HasPrefix(ref, "convocatoria:") && referenciaActoBolsaAlegacion.MatchString(strings.TrimPrefix(ref, "convocatoria:")))) {
		return "", ErrSolicitudInvalida
	}
	b, err := json.Marshal(struct {
		Accion, PersonaRef, Idioma, Ref, Cursor string
		Limite                                  int
	}{accion, personaRef, idioma, ref, filtro.Cursor, filtro.Limite})
	if err != nil {
		return "", ErrSolicitudInvalida
	}
	h := sha256.Sum256(b)
	return "alegaciones_lectura_" + hex.EncodeToString(h[:]), nil
}

type Alegacion struct {
	AlegacionRef   string          `json:"alegacion_ref"`
	Objeto         ObjetoAlegacion `json:"objeto"`
	Fundamento     string          `json:"fundamento"`
	Estado         string          `json:"estado"`
	PresentadaEn   time.Time       `json:"presentada_en"`
	ReciboRef      string          `json:"recibo_ref"`
	AuditoriaRef   string          `json:"auditoria_ref"`
	PersonaResumen string          `json:"persona_resumen,omitempty"`
	// Solo lo devuelve la lectura SQL RRHH de una participación B7 vigente.
	// Nunca forma parte del recibo ni del JSON enviado al navegador.
	MetadatoNombreB7 *MetadatoNombreAlegacionB7 `json:"-"`
}

type MetadatoNombreAlegacionB7 struct {
	ActaRef             string `json:"acta_ref"`
	ParticipacionRef    string `json:"participacion_ref"`
	HuellaListadoSHA256 string `json:"huella_listado_sha256"`
	CategoriaRef        string `json:"categoria_ref"`
	FilaNumero          int    `json:"fila_numero"`
}

func (a Alegacion) Validar() error {
	if !referenciaOpaca.MatchString(a.AlegacionRef) || a.Objeto.Validar() != nil ||
		a.Estado != "presentada" || a.PresentadaEn.IsZero() || !referenciaOpaca.MatchString(a.ReciboRef) ||
		!referenciaOpaca.MatchString(a.AuditoriaRef) || !utf8.ValidString(a.Fundamento) ||
		strings.TrimSpace(a.Fundamento) == "" || utf8.RuneCountInString(a.Fundamento) > 4000 ||
		strings.ContainsRune(a.Fundamento, '\x00') {
		return ErrNoDisponible
	}
	return nil
}

type ReciboAlegacion struct {
	Alegacion
	Repetida bool `json:"repetida"`
}

type PaginaAlegaciones struct {
	Alegaciones     []Alegacion `json:"alegaciones"`
	Total           uint64      `json:"total"`
	CursorSiguiente *string     `json:"cursor_siguiente"`
}

// Es una extensión del repositorio de inscripción, no otra autoridad de la
// solicitud. PostgreSQL une consumo V3, titularidad, efecto, historia y recibo.
type RepositorioAlegaciones interface {
	PresentarAlegacion(context.Context, Actor, PresentacionAlegacion) (ReciboAlegacion, error)
	ObjetosAlegables(context.Context, Actor, int, string) (PaginaObjetosAlegacion, error)
	AlegacionesPropias(context.Context, Actor, int, string) (PaginaAlegaciones, error)
	AlegacionesRRHH(context.Context, Actor, string, string, int, string) (PaginaAlegaciones, error)
	AlegacionRRHH(context.Context, Actor, string) (Alegacion, error)
	AmbitosAlegacionesRRHH(context.Context, Actor, int, string) (PaginaAmbitosAlegacionRRHH, error)
}

func repositorioAlegaciones(s *Servicio) (RepositorioAlegaciones, error) {
	if s == nil || s.repositorio == nil {
		return nil, ErrNoDisponible
	}
	r, ok := s.repositorio.(RepositorioAlegaciones)
	if !ok {
		return nil, ErrNoDisponible
	}
	return r, nil
}

func lecturaAlegacionValida(actor Actor, accion, ref string, limite int, cursor string) bool {
	if limite < 1 || limite > 20 || (cursor != "" && !referenciaOpaca.MatchString(cursor)) {
		return false
	}
	id := actor.Idioma
	if id == "" {
		id = "es"
	}
	filtro := Filtro{Limite: limite, Cursor: cursor}
	recurso, err := recursoLecturaAlegacion(accion, actor.PersonaRef, id, filtro, ref)
	return err == nil && actor.LecturaValida(accion, recurso, filtro)
}

func (s *Servicio) ObjetosAlegables(ctx context.Context, actor Actor, limite int, cursor string) (PaginaObjetosAlegacion, error) {
	r, err := repositorioAlegaciones(s)
	if err != nil {
		return PaginaObjetosAlegacion{}, err
	}
	if !lecturaAlegacionValida(actor, AccionObjetosAlegacion, "objetos", limite, cursor) || actor.Canal != "externa_personal" {
		return PaginaObjetosAlegacion{}, ErrAccesoDenegado
	}
	p, err := r.ObjetosAlegables(ctx, actor, limite, cursor)
	if err != nil {
		return PaginaObjetosAlegacion{}, err
	}
	if len(p.Objetos) > limite || uint64(len(p.Objetos)) > p.Total {
		return PaginaObjetosAlegacion{}, ErrNoDisponible
	}
	for _, o := range p.Objetos {
		if o.Validar() != nil {
			return PaginaObjetosAlegacion{}, ErrNoDisponible
		}
	}
	return p, nil
}

func (s *Servicio) AlegacionesPropias(ctx context.Context, actor Actor, limite int, cursor string) (PaginaAlegaciones, error) {
	r, err := repositorioAlegaciones(s)
	if err != nil {
		return PaginaAlegaciones{}, err
	}
	if !lecturaAlegacionValida(actor, AccionAlegacionesPropias, "propias", limite, cursor) || actor.Canal != "externa_personal" {
		return PaginaAlegaciones{}, ErrAccesoDenegado
	}
	return r.AlegacionesPropias(ctx, actor, limite, cursor)
}

func (s *Servicio) AlegacionesRRHH(ctx context.Context, actor Actor, tipo, ref string, limite int, cursor string) (PaginaAlegaciones, error) {
	r, err := repositorioAlegaciones(s)
	if err != nil {
		return PaginaAlegaciones{}, err
	}
	if (tipo != "convocatoria" && tipo != "bolsa") || !referenciaActoBolsaAlegacion.MatchString(ref) ||
		!lecturaAlegacionValida(actor, AccionAlegacionesRRHH, tipo+":"+ref, limite, cursor) || actor.Canal != "interna_corporativa" {
		return PaginaAlegaciones{}, ErrAccesoDenegado
	}
	return r.AlegacionesRRHH(ctx, actor, tipo, ref, limite, cursor)
}

func (s *Servicio) AmbitosAlegacionesRRHH(ctx context.Context, actor Actor, limite int, cursor string) (PaginaAmbitosAlegacionRRHH, error) {
	r, err := repositorioAlegaciones(s)
	if err != nil {
		return PaginaAmbitosAlegacionRRHH{}, err
	}
	if !lecturaAlegacionValida(actor, AccionAlegacionesRRHH, "ambitos", limite, cursor) || actor.Canal != "interna_corporativa" {
		return PaginaAmbitosAlegacionRRHH{}, ErrAccesoDenegado
	}
	return r.AmbitosAlegacionesRRHH(ctx, actor, limite, cursor)
}

func (s *Servicio) AlegacionRRHH(ctx context.Context, actor Actor, ref string) (Alegacion, error) {
	r, err := repositorioAlegaciones(s)
	if err != nil {
		return Alegacion{}, err
	}
	if !referenciaOpaca.MatchString(ref) || !lecturaAlegacionValida(actor, AccionAlegacionesRRHH, ref, 1, "") || actor.Canal != "interna_corporativa" {
		return Alegacion{}, ErrAccesoDenegado
	}
	return r.AlegacionRRHH(ctx, actor, ref)
}

func (s *Servicio) PresentarAlegacion(ctx context.Context, actor Actor, p PresentacionAlegacion) (ReciboAlegacion, error) {
	if s == nil || s.repositorio == nil {
		return ReciboAlegacion{}, ErrNoDisponible
	}
	r, ok := s.repositorio.(RepositorioAlegaciones)
	if !ok {
		return ReciboAlegacion{}, ErrNoDisponible
	}
	if !actor.EscrituraValida() || actor.Canal != "externa_personal" || p.Validar() != nil {
		return ReciboAlegacion{}, ErrSolicitudInvalida
	}
	recibo, err := r.PresentarAlegacion(ctx, actor, p)
	if err != nil {
		return ReciboAlegacion{}, err
	}
	if recibo.Alegacion.Validar() != nil || recibo.Objeto != p.Objeto || recibo.Fundamento != p.Fundamento {
		return ReciboAlegacion{}, ErrNoDisponible
	}
	return recibo, nil
}
