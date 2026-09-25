package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionConsultarCatalogoEmpleadoB2    = "personal.registro_empleado.catalogo.consultar"
	AccionPublicarCatalogoEmpleadoB2     = "personal.registro_empleado.catalogo.publicar"
	AccionRetirarCatalogoEmpleadoB2      = "personal.registro_empleado.catalogo.retirar"
	AudienciaConsultarCatalogoEmpleadoB2 = "vec_personal.registro_empleado.catalogo.consultar.v1"
	AudienciaPublicarCatalogoEmpleadoB2  = "vec_personal.registro_empleado.catalogo.publicar.v1"
	AudienciaRetirarCatalogoEmpleadoB2   = "vec_personal.registro_empleado.catalogo.retirar.v1"
)

type EntradaCatalogoRegistroEmpleadoB2 struct {
	OrganismoRef string     `json:"organismo_ref"`
	Tipo         string     `json:"tipo"`
	Ref          string     `json:"ref"`
	Version      int64      `json:"version"`
	Revision     int64      `json:"revision"`
	Denominacion string     `json:"denominacion"`
	HuellaSHA256 string     `json:"huella_sha256"`
	VigenteDesde FechaCivil `json:"vigente_desde"`
	VigenteHasta FechaCivil `json:"vigente_hasta"`
	Estado       string     `json:"estado"`
}

func (e EntradaCatalogoRegistroEmpleadoB2) Validar() error {
	if !organismoCatalogoEmpleadoB2Valido(e.OrganismoRef) || !tipoCatalogoEmpleadoB2Valido(e.Tipo) ||
		!patronReferenciaB2.MatchString(e.Ref) || e.Version < 1 || e.Version > 2147483647 ||
		e.Revision < 1 || e.Revision > 2147483647 || !denominacionCatalogoEmpleadoB2Valida(e.Denominacion) ||
		!huellaRegistroDominioB2Valida(e.HuellaSHA256) || !intervaloActoB2Valido(e.VigenteDesde, e.VigenteHasta) ||
		(e.Estado != "publicada" && e.Estado != "retirada") {
		return ErrRegistroEmpleadoB2Invalido
	}
	return nil
}

type SolicitudConsultaCatalogoEmpleadoB2 struct {
	OrganismoRef  string
	Tipo          string
	Estado        string
	CursorRef     string
	CursorVersion int64
	Limite        int
	Actor         core.ContextoActor
}

type SolicitudCambioCatalogoEmpleadoB2 struct {
	Operacion       string
	OrganismoRef    string
	Tipo            string
	Ref             string
	Version         int64
	Revision        int64
	Denominacion    string
	HuellaSHA256    string
	VigenteDesde    FechaCivil
	VigenteHasta    FechaCivil
	IdempotenciaRef string
	Actor           core.ContextoActor
}

// MaterialCatalogoEmpleadoB2 conserva exactamente los bytes presentados al
// punto de decisión y a PostgreSQL. El actor procede del contexto servidor.
type MaterialCatalogoEmpleadoB2 struct {
	operacion     string
	organismoRef  string
	tipo          string
	ref           string
	version       int64
	revision      int64
	estado        string
	cursorRef     string
	cursorVersion int64
	limite        int
	actor         core.ContextoActor
	canonico      []byte
	recurso       core.RecursoAutorizable
}

func NuevoMaterialConsultaCatalogoEmpleadoB2(s SolicitudConsultaCatalogoEmpleadoB2) (MaterialCatalogoEmpleadoB2, error) {
	if !organismoCatalogoEmpleadoB2Valido(s.OrganismoRef) || !tipoCatalogoEmpleadoB2Valido(s.Tipo) ||
		(s.Estado != "" && s.Estado != "publicada" && s.Estado != "retirada") ||
		((s.CursorRef == "") != (s.CursorVersion == 0)) ||
		(s.CursorRef != "" && !patronReferenciaB2.MatchString(s.CursorRef)) || s.CursorVersion < 0 ||
		s.Limite < 1 || s.Limite > 100 || s.Actor.Validar() != nil {
		return MaterialCatalogoEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	var estado any
	if s.Estado != "" {
		estado = s.Estado
	}
	var cursorRef any
	var cursorVersion any
	if s.CursorRef != "" {
		cursorRef, cursorVersion = s.CursorRef, s.CursorVersion
	}
	obj := struct {
		Esquema       string `json:"esquema"`
		OrganismoRef  string `json:"organismo_ref"`
		Tipo          string `json:"tipo"`
		Estado        any    `json:"estado"`
		CursorRef     any    `json:"cursor_ref"`
		CursorVersion any    `json:"cursor_version"`
		Limite        int    `json:"limite"`
	}{"vec.personal.catalogo-registro-empleado.consulta.v1", s.OrganismoRef, s.Tipo, estado, cursorRef, cursorVersion, s.Limite}
	m, err := nuevoMaterialCatalogoEmpleadoB2("consultar", s.OrganismoRef, s.Tipo, "", 0, 0, s.Actor, obj)
	if err != nil {
		return MaterialCatalogoEmpleadoB2{}, err
	}
	m.estado, m.cursorRef, m.cursorVersion, m.limite = s.Estado, s.CursorRef, s.CursorVersion, s.Limite
	return m, nil
}

func NuevoMaterialCambioCatalogoEmpleadoB2(s SolicitudCambioCatalogoEmpleadoB2) (MaterialCatalogoEmpleadoB2, error) {
	if (s.Operacion != "publicar" && s.Operacion != "retirar") || !organismoCatalogoEmpleadoB2Valido(s.OrganismoRef) ||
		!tipoCatalogoEmpleadoB2Valido(s.Tipo) || !patronReferenciaB2.MatchString(s.Ref) ||
		s.Version < 1 || s.Version > 2147483647 || s.Revision < 1 || s.Revision > 2147483647 ||
		(s.Operacion == "publicar" && s.Revision != 1) || (s.Operacion == "retirar" && s.Revision < 2) ||
		!denominacionCatalogoEmpleadoB2Valida(s.Denominacion) || !huellaRegistroDominioB2Valida(s.HuellaSHA256) ||
		!intervaloActoB2Valido(s.VigenteDesde, s.VigenteHasta) ||
		!patronUUIDRegistroB2.MatchString(s.IdempotenciaRef) || s.Actor.Validar() != nil {
		return MaterialCatalogoEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	if s.Operacion == "publicar" && s.HuellaSHA256 != HuellaPublicacionCatalogoEmpleadoB2(s) {
		return MaterialCatalogoEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	obj := struct {
		Esquema         string `json:"esquema"`
		Operacion       string `json:"operacion"`
		OrganismoRef    string `json:"organismo_ref"`
		Tipo            string `json:"tipo"`
		Ref             string `json:"ref"`
		Version         int64  `json:"version"`
		Revision        int64  `json:"revision"`
		Denominacion    string `json:"denominacion"`
		HuellaSHA256    string `json:"huella_sha256"`
		VigenteDesde    string `json:"vigente_desde"`
		VigenteHasta    string `json:"vigente_hasta"`
		ActorRef        string `json:"actor_ref"`
		IdempotenciaRef string `json:"idempotencia_ref"`
	}{"vec.personal.catalogo-registro-empleado.v1", s.Operacion, s.OrganismoRef, s.Tipo, s.Ref,
		s.Version, s.Revision, s.Denominacion, s.HuellaSHA256, s.VigenteDesde.Texto(),
		s.VigenteHasta.Texto(), s.Actor.Principal.ID, s.IdempotenciaRef}
	return nuevoMaterialCatalogoEmpleadoB2(s.Operacion, s.OrganismoRef, s.Tipo, s.Ref, s.Version, s.Revision, s.Actor, obj)
}

func nuevoMaterialCatalogoEmpleadoB2(operacion, organismo, tipo, ref string, version, revision int64, actor core.ContextoActor, obj any) (MaterialCatalogoEmpleadoB2, error) {
	actor, err := actor.Clonar()
	if err != nil {
		return MaterialCatalogoEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	canonico, err := json.Marshal(obj)
	if err != nil {
		return MaterialCatalogoEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	suma := sha256.Sum256(canonico)
	efecto := organismo + ":" + tipo
	recursoTipo := "catalogo_empleado_rrhh"
	ambitos := map[string]string{"organismo_ref": organismo, "objetivo_ref": efecto}
	if operacion != "consultar" {
		efecto = organismo + ":" + tipo + ":" + ref + ":" + strconv.FormatInt(version, 10)
		recursoTipo = "entrada_catalogo_empleado_rrhh"
		ambitos["objetivo_ref"] = efecto
	}
	recurso := core.RecursoAutorizable{Referencia: efecto, ModuloID: "personal", Tipo: recursoTipo,
		Ambitos: ambitos, Atributos: map[string]string{"operacion": operacion, "material_sha256": hex.EncodeToString(suma[:])}}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return MaterialCatalogoEmpleadoB2{}, ErrRegistroEmpleadoB2Invalido
	}
	return MaterialCatalogoEmpleadoB2{operacion: operacion, organismoRef: organismo, tipo: tipo, ref: ref, version: version, revision: revision, actor: actor, canonico: canonico, recurso: recurso}, nil
}

func (m MaterialCatalogoEmpleadoB2) Operacion() string         { return m.operacion }
func (m MaterialCatalogoEmpleadoB2) OrganismoRef() string      { return m.organismoRef }
func (m MaterialCatalogoEmpleadoB2) Tipo() string              { return m.tipo }
func (m MaterialCatalogoEmpleadoB2) Ref() string               { return m.ref }
func (m MaterialCatalogoEmpleadoB2) Version() int64            { return m.version }
func (m MaterialCatalogoEmpleadoB2) Revision() int64           { return m.revision }
func (m MaterialCatalogoEmpleadoB2) Estado() string            { return m.estado }
func (m MaterialCatalogoEmpleadoB2) CursorRef() string         { return m.cursorRef }
func (m MaterialCatalogoEmpleadoB2) CursorVersion() int64      { return m.cursorVersion }
func (m MaterialCatalogoEmpleadoB2) Limite() int               { return m.limite }
func (m MaterialCatalogoEmpleadoB2) Canonico() []byte          { return append([]byte(nil), m.canonico...) }
func (m MaterialCatalogoEmpleadoB2) Actor() core.ContextoActor { a, _ := m.actor.Clonar(); return a }
func (m MaterialCatalogoEmpleadoB2) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialCatalogoEmpleadoB2) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}

func tipoCatalogoEmpleadoB2Valido(tipo string) bool {
	return tipo == "regimen" || tipo == "modalidad" || tipo == "situacion" || tipo == "clase_servicio"
}
func organismoCatalogoEmpleadoB2Valido(ref string) bool {
	return len(ref) <= 127 && patronReferenciaB2.MatchString(ref)
}

// HuellaPublicacionCatalogoEmpleadoB2 fija el contenido de una versión.
// La referencia de acto procede de la decisión V3 dentro de PostgreSQL.
// La retirada conserva la huella original y no calcula otra publicación.
func HuellaPublicacionCatalogoEmpleadoB2(s SolicitudCambioCatalogoEmpleadoB2) string {
	partes := []string{"vec.personal.catalogo-registro-empleado.entrada.v1", s.OrganismoRef,
		s.Tipo, s.Ref, strconv.FormatInt(s.Version, 10), strconv.FormatInt(s.Revision, 10),
		s.Denominacion, s.VigenteDesde.Texto(), s.VigenteHasta.Texto()}
	suma := sha256.Sum256([]byte(strings.Join(partes, "\n")))
	return hex.EncodeToString(suma[:])
}
func denominacionCatalogoEmpleadoB2Valida(s string) bool {
	if s == "" || s != strings.TrimSpace(s) || len(s) > 256 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
