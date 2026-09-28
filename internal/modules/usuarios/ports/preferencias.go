package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Errores nominales para una futura API: 401, 403, 409, 422 y 503.
var (
	ErrNoAutenticado    = errors.New("usuarios preferencias: no autenticado")
	ErrProhibido        = errors.New("usuarios preferencias: prohibido")
	ErrConflicto        = errors.New("usuarios preferencias: conflicto")
	ErrPeticionInvalida = errors.New("usuarios preferencias: peticion invalida")
	ErrNoDisponible     = errors.New("usuarios preferencias: no disponible")
)

const FinalidadPreferenciasPropias = "finalidad:usuarios:preferencias-propias:v1"
const AccionConsultarPreferencias = "vec.preferencias.consultar"
const AccionActualizarPreferencias = "vec.preferencias.actualizar"

type MaterialPreferencias struct {
	PersonaRef         string                     `json:"persona_ref"`
	PerfilRef          string                     `json:"perfil_ref"`
	Accion             string                     `json:"accion"`
	FinalidadRef       string                     `json:"finalidad_ref"`
	CatalogoVersionRef string                     `json:"catalogo_version_ref"`
	VersionEsperada    uint64                     `json:"version_esperada"`
	ClaveOperacion     string                     `json:"clave_operacion"`
	HuellaPeticion     string                     `json:"huella_peticion"`
	Valores            domain.ValoresPreferencias `json:"valores"`
}

type ProveedorMaterialPreferencias interface {
	ProveerMaterialPreferencias(context.Context, MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenPreferencias sólo se construye con un contexto canónico ya resuelto.
func metodoCertificado(m vecdomain.AuthMethod) bool {
	return m == vecdomain.AuthMethodCertificate || m == vecdomain.AuthMethodDNIe
}

type OrdenPreferencias struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorMaterialPreferencias
}

func NuevaOrdenPreferencias(actor vecdomain.ContextoActor, proveedor ProveedorMaterialPreferencias) (OrdenPreferencias, error) {
	if actor.Validar() != nil || !metodoCertificado(actor.Principal.AuthMethod) || proveedor == nil {
		return OrdenPreferencias{}, ErrNoAutenticado
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenPreferencias{}, ErrNoAutenticado
	}
	return OrdenPreferencias{actor: copia, proveedor: proveedor}, nil
}
func (o OrdenPreferencias) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || o.actor.Validar() != nil || !metodoCertificado(o.actor.Principal.AuthMethod) {
		return vecdomain.ContextoActor{}, ErrNoAutenticado
	}
	return o.actor.Clonar()
}
func (o OrdenPreferencias) Proveedor() ProveedorMaterialPreferencias { return o.proveedor }

type PeticionGuardarPreferencias struct {
	VersionEsperada    uint64                     `json:"version_esperada"`
	CatalogoVersionRef string                     `json:"catalogo_version_ref"`
	ClaveOperacion     string                     `json:"clave_operacion"`
	Valores            domain.ValoresPreferencias `json:"valores"`
}

type EstadoPreferencias struct {
	PersonaRef         string                     `json:"persona_ref"`
	Version            uint64                     `json:"version"`
	CatalogoVersionRef string                     `json:"catalogo_version_ref"`
	Valores            domain.ValoresPreferencias `json:"valores"`
}

type VistaPreferencias struct {
	Catalogo domain.CatalogoPreferencias `json:"catalogo"`
	Estado   EstadoPreferencias          `json:"estado"`
}

type ReciboPreferencias struct {
	ReciboRef          string                     `json:"recibo_ref"`
	PersonaRef         string                     `json:"persona_ref"`
	Version            uint64                     `json:"version"`
	CatalogoVersionRef string                     `json:"catalogo_version_ref"`
	Valores            domain.ValoresPreferencias `json:"valores"`
	FechaUTC           time.Time                  `json:"fecha_utc"`
	Replay             bool                       `json:"replay"`
}

// Implementar en una sola transacción PostgreSQL la autorización V3 fresca,
// el CAS, estado, historia, auditoría y recibo. En replay no alterar el estado
// vigente ni el recibo original; reautorizar la lectura de la operación.
type RegistroPreferencias interface {
	CatalogoVigente(context.Context) (domain.CatalogoPreferencias, error)
	ConsultarPropias(context.Context, OrdenPreferencias, MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (EstadoPreferencias, bool, error)
	RecuperarOperacion(context.Context, OrdenPreferencias, MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboPreferencias, bool, error)
	Guardar(context.Context, OrdenPreferencias, PeticionGuardarPreferencias, MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboPreferencias, error)
}
