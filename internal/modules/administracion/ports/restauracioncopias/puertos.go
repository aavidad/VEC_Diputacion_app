package restauracioncopias

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	d "vec-diputacion-granada/internal/modules/administracion/domain/restauracioncopias"
)

type Accion string

const (
	Proponer  Accion = "copias_restauracion_proponer"
	Revisar   Accion = "copias_restauracion_revisar"
	Sustituir Accion = "copias_restauracion_sustituir"
)

// Acceso usa solo una referencia de sesión de la frontera confiable. PersonaRef
// la devuelve la autoridad canónica; nunca se toma de un formulario.
type Acceso struct {
	SesionRef       string
	Accion          Accion
	DestinoRef      string
	PropuestaSHA256 string
}
type Concesion struct {
	PersonaRef string
	Ref        string
	Acceso     Acceso
	Caduca     time.Time
}
type Autoridad interface {
	AutorizarActual(context.Context, Acceso) (Concesion, error)
	RevalidarPersona(context.Context, string, Accion, string, string) error
}

// Observador debe verificar bytes/autenticidad/cobertura fuera del destino de
// rollback y devolver preimagen de TODO el estado bajo la exclusión indicada.
type Observacion struct {
	Manifiesto          copias.Manifiesto
	Destino             copias.Inventario
	Politica            copias.Politica
	PreimagenSHA256     string
	ConjuntoAutenticado bool
	PoliticaAutenticada bool
	ExclusionRef        string
	VentanaRef          string
	CopiaPrevia         *CopiaPrevia
}
type CopiaPrevia struct {
	Manifiesto      copias.Manifiesto
	Autenticada     bool
	DestinoRef      string
	PreimagenSHA256 string
	ExclusionRef    string
}
type Observador interface {
	ObservarActual(context.Context, string, string, string) (Observacion, error)
}
type Reloj interface{ Ahora() time.Time }
type Registro struct {
	Propuesta d.Sellada   `json:"propuesta"`
	Revision  *d.Revision `json:"revision"`
	Version   uint64      `json:"version"`
}

// La implementación debe estar fuera del rollback. Crear/CAS son atómicos con
// auditoría y consumo de concesión vigente. Un fallo de consumo no escribe.
// Crear es idempotente solo para igual ref y contenido; CAS comprueba versión y
// sello guardados, añade historia y conserva recibo. No se admite last-write-wins.
type RegistroPropuestas interface {
	Crear(context.Context, d.Sellada, Concesion) (Registro, error)
	Leer(context.Context, string, Concesion) (Registro, error)
	RevisarCAS(context.Context, string, uint64, string, d.Revision, Concesion) (Registro, error)
	// Cercar valida CAS y exclusión y consume permiso de sustitución en la misma
	// transacción del registro externo. No ejecuta ni concede sustitución física.
	Cercar(context.Context, string, uint64, string, string, Concesion) error
}
