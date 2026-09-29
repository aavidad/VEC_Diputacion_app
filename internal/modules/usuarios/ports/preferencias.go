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
const AudienciaConsultarPreferenciasInterna = "vec_usuarios.preferencias.consultar.interna_corporativa.v1"
const AudienciaActualizarPreferenciasInterna = "vec_usuarios.preferencias.actualizar.interna_corporativa.v1"
const AudienciaConsultarPreferenciasExterna = "vec_usuarios.preferencias.consultar.externa_personal.v1"
const AudienciaActualizarPreferenciasExterna = "vec_usuarios.preferencias.actualizar.externa_personal.v1"
const TipoRecursoPreferencias = "preferencias_persona"

func AudienciaPreferencias(accion string, superficie vecdomain.SuperficieAutenticacionActorV1) (string, error) {
	switch {
	case accion == AccionConsultarPreferencias && superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		return AudienciaConsultarPreferenciasInterna, nil
	case accion == AccionActualizarPreferencias && superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		return AudienciaActualizarPreferenciasInterna, nil
	case accion == AccionConsultarPreferencias && superficie == vecdomain.SuperficieAutenticacionExternaPersonalV1:
		return AudienciaConsultarPreferenciasExterna, nil
	case accion == AccionActualizarPreferencias && superficie == vecdomain.SuperficieAutenticacionExternaPersonalV1:
		return AudienciaActualizarPreferenciasExterna, nil
	default:
		return "", ErrProhibido
	}
}

type MaterialPreferencias struct {
	Superficie         vecdomain.SuperficieAutenticacionActorV1 `json:"superficie"`
	PersonaRef         string                                   `json:"persona_ref"`
	PerfilRef          string                                   `json:"perfil_ref"`
	Accion             string                                   `json:"accion"`
	FinalidadRef       string                                   `json:"finalidad_ref"`
	CatalogoVersionRef string                                   `json:"catalogo_version_ref"`
	VersionEsperada    uint64                                   `json:"version_esperada"`
	ClaveOperacion     string                                   `json:"clave_operacion"`
	HuellaPeticion     string                                   `json:"huella_peticion"`
	Valores            domain.ValoresPreferencias               `json:"valores"`
}

type ProveedorMaterialPreferencias interface {
	ProveerMaterialPreferencias(context.Context, vecdomain.VinculoAutenticacionActorV2, MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenPreferencias sólo se construye desde la identidad V2 acreditada y la
// superficie exacta que el servidor fija por ruta; nunca desde un campo HTTP.
func metodoCertificado(m vecdomain.AuthMethod) bool {
	return m == vecdomain.AuthMethodCertificate || m == vecdomain.AuthMethodDNIe
}

type OrdenPreferencias struct {
	actor      vecdomain.ContextoActor
	vinculo    vecdomain.VinculoAutenticacionActorV2
	superficie vecdomain.SuperficieAutenticacionActorV1
	proveedor  ProveedorMaterialPreferencias
}

func cotejarIdentidadVinculo(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2) (vecdomain.DatosVinculoAutenticacionActorV2, bool) {
	if actor.Validar() != nil || !metodoCertificado(actor.Principal.AuthMethod) || actor.Instantanea.CuentaVersion == 0 {
		return vecdomain.DatosVinculoAutenticacionActorV2{}, false
	}
	datos, err := vinculo.Datos()
	//vec:silencio-justificado PREDICADO_VALIDACION un vínculo V2 sin datos acreditados no identifica al actor
	if err != nil {
		return vecdomain.DatosVinculoAutenticacionActorV2{}, false
	}
	if datos.CuentaPrivilegiada {
		return vecdomain.DatosVinculoAutenticacionActorV2{}, false
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	//vec:silencio-justificado PREDICADO_VALIDACION una huella de contexto no obtenible invalida la identidad ligada
	if err != nil {
		return vecdomain.DatosVinculoAutenticacionActorV2{}, false
	}
	coincide := datos.PrincipalID == actor.PersonaRef && datos.PerfilActivoRef == actor.PerfilActivoRef &&
		datos.CuentaRef == actor.Instantanea.CuentaRef && datos.CuentaOrdinariaRef == actor.Instantanea.CuentaRef &&
		datos.MetodoObservado == actor.Principal.AuthMethod && datos.GarantiaObservada == actor.Principal.AuthAssurance &&
		datos.ContextoActorRef == actor.Instantanea.VinculoRef && datos.ContextoActorVersion == actor.Instantanea.VinculoVersion &&
		datos.ContextoActorCuentaVersion == actor.Instantanea.CuentaVersion && datos.ContextoActorHuellaSHA256 == huella
	return datos, coincide
}
func cotejarVinculo(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1) bool {
	datos, ok := cotejarIdentidadVinculo(actor, vinculo)
	return ok && (superficieRuta == vecdomain.SuperficieAutenticacionInternaCorporativaV1 || superficieRuta == vecdomain.SuperficieAutenticacionExternaPersonalV1) && datos.Superficie == superficieRuta
}

func NuevaOrdenPreferencias(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1, proveedor ProveedorMaterialPreferencias) (OrdenPreferencias, error) {
	datos, identidadValida := cotejarIdentidadVinculo(actor, vinculo)
	if proveedor == nil || !identidadValida {
		return OrdenPreferencias{}, ErrNoAutenticado
	}
	if (superficieRuta != vecdomain.SuperficieAutenticacionInternaCorporativaV1 && superficieRuta != vecdomain.SuperficieAutenticacionExternaPersonalV1) || datos.Superficie != superficieRuta {
		return OrdenPreferencias{}, ErrProhibido
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenPreferencias{}, ErrNoAutenticado
	}
	return OrdenPreferencias{actor: copia, vinculo: vinculo, superficie: superficieRuta, proveedor: proveedor}, nil
}
func (o OrdenPreferencias) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || !cotejarVinculo(o.actor, o.vinculo, o.superficie) {
		return vecdomain.ContextoActor{}, ErrNoAutenticado
	}
	return o.actor.Clonar()
}
func (o OrdenPreferencias) Vinculo() (vecdomain.VinculoAutenticacionActorV2, error) {
	if o.proveedor == nil || !cotejarVinculo(o.actor, o.vinculo, o.superficie) {
		return vecdomain.VinculoAutenticacionActorV2{}, ErrNoAutenticado
	}
	return o.vinculo, nil
}
func (o OrdenPreferencias) Superficie() (vecdomain.SuperficieAutenticacionActorV1, error) {
	if o.proveedor == nil || !cotejarVinculo(o.actor, o.vinculo, o.superficie) {
		return "", ErrNoAutenticado
	}
	return o.superficie, nil
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
// vigente ni el recibo original. RecuperarOperacion consume V3 aun cuando no
// existe la clave; Guardar exige otra exportación fresca y consume su V3 propio.
type RegistroPreferencias interface {
	CatalogoVigente(context.Context, OrdenPreferencias) (domain.CatalogoPreferencias, error)
	ConsultarPropias(context.Context, OrdenPreferencias, MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (EstadoPreferencias, bool, error)
	RecuperarOperacion(context.Context, OrdenPreferencias, MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboPreferencias, bool, error)
	Guardar(context.Context, OrdenPreferencias, PeticionGuardarPreferencias, MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboPreferencias, error)
}
