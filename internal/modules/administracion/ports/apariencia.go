package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain"
)

var (
	ErrRevisionGlobalObsoleta      = errors.New("administracion: revisión global obsoleta")
	ErrClavePublicacionEnConflicto = errors.New("administracion: clave de publicación en conflicto")
)

const (
	AccionConsultarApariencia = "vec.apariencia.global.consultar"
	AccionPublicarApariencia  = "vec.apariencia.global.publicar"
	RecursoAparienciaGlobal   = "vec.apariencia.global"
	AmbitoAparienciaGlobal    = "global"
	FinalidadConsultarTema    = "consultar_tema_global"
	FinalidadPublicarTema     = "publicar_tema_global"
)

// SolicitudAutorizacionApariencia fija la acción y el efecto exactos. No recibe
// actor, rol ni permiso del cliente: la autoridad existente los resuelve desde
// la frontera autenticada y registra tanto concesiones como denegaciones.
type SolicitudAutorizacionApariencia struct {
	Accion                 string
	RecursoRef             string
	AmbitoRef              string
	Finalidad              string
	Tema                   domain.Tema
	RevisionGlobalEsperada uint64
	ClaveIdempotencia      string
}

// ConcesionApariencia procede únicamente de la autoridad central. La
// aprobación de publicación corresponde a otra identidad, acreditada allí;
// este módulo no define roles ni decide permisos por su cuenta.
type ConcesionApariencia struct {
	Permitida              bool
	Accion                 string
	RecursoRef             string
	AmbitoRef              string
	Finalidad              string
	Tema                   domain.Tema
	RevisionGlobalEsperada uint64
	ClaveIdempotencia      string
	ActorRef               string
	PerfilRef              string
	DecisionRef            string
	AprobacionRef          string
	AprobadorRef           string
	EmitidaEn              time.Time
	ExpiraEn               time.Time
}

type AutorizadorApariencia interface {
	ExigirApariencia(context.Context, SolicitudAutorizacionApariencia) (ConcesionApariencia, error)
}

type OrdenConsultaApariencia struct {
	Concesion ConcesionApariencia
}

type ResultadoConsultaApariencia struct {
	Estado       domain.EstadoApariencia
	DecisionRef  string
	AuditoriaRef string
	ConsultadaEn time.Time
}

type OrdenConfirmarPublicacion struct {
	Publicacion domain.OrdenPublicacion
	Concesion   ConcesionApariencia
}

type ReciboPublicacion struct {
	Estado            domain.EstadoApariencia
	RevisionAnterior  uint64
	ClaveIdempotencia string
	ActorRef          string
	DecisionRef       string
	AprobacionRef     string
	AuditoriaRef      string
	EventoRef         string
	ConfirmadaEn      time.Time
	Replay            bool
}

// RegistroApariencia es el único dueño durable del estado global. Consultar
// exige auditar la lectura antes de devolverla. PublicarAtomico debe revalidar
// y consumir la concesión central dentro de la misma transacción que compara
// RevisionGlobalEsperada, escribe estado e historia de solo adición, auditoría
// segregada, evento y recibo. La clave idempotente queda ligada a actor,
// acción, recurso, tema y revisión esperada, y recupera exactamente el mismo
// recibo tras un fallo o reinicio. Sin esas garantías, el adaptador falla
// cerrado y no confirma la publicación. Una revisión global obsoleta devuelve
// ErrRevisionGlobalObsoleta; la misma clave con otro efecto devuelve
// ErrClavePublicacionEnConflicto. Una preview en memoria no usa este puerto.
type RegistroApariencia interface {
	ConsultarAuditable(context.Context, OrdenConsultaApariencia) (ResultadoConsultaApariencia, error)
	PublicarAtomico(context.Context, OrdenConfirmarPublicacion) (ReciboPublicacion, error)
}
