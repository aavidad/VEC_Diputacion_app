package ports

import (
	"context"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const EsquemaConsultaEstadoSeleccionOfertaCT = "vec.ct.seleccion-oferta.consulta-expediente.v1"

// SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT nace de la cabecera CT
// y la sesión revalidada. La clave original se obtiene después, de O6.
type SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT struct {
	OrganizacionRef    string
	ExpedienteRef      string
	CategoriaRef       string
	CentroRef          string
	UnidadGestoraRef   string
	VersionObservada   uint64
	ActorRef           string
	PerfilRef          string
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

func (q SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT) Validar() error {
	if !domain.ReferenciaOpacaValida(q.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(q.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(q.CategoriaRef) ||
		!domain.ReferenciaOpacaValida(q.CentroRef) ||
		!domain.ReferenciaOpacaValida(q.UnidadGestoraRef) ||
		q.VersionObservada == 0 || q.VersionObservada > 1<<53-1 ||
		!domain.ReferenciaOpacaValida(q.ActorRef) ||
		!domain.ReferenciaOpacaValida(q.PerfilRef) ||
		q.ResultadoContexto.Validar() != nil ||
		q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.ResultadoContexto.Contexto.PersonaRef != q.ActorRef ||
		q.ResultadoContexto.Contexto.PerfilActivoRef != q.PerfilRef ||
		q.Correlacion.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return ErrSeleccionOfertaCTInvalida
	}
	return nil
}

// CanonConsultaEstadoSeleccionOfertaCT liga la lectura sin UUID al expediente
// y a la identidad actual. SQL recibe exactamente estos bytes para consumir V3.
func CanonConsultaEstadoSeleccionOfertaCT(q SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT) (string, error) {
	if q.Validar() != nil {
		return "", ErrSeleccionOfertaCTInvalida
	}
	b, err := json.Marshal(struct {
		Esquema          string `json:"esquema"`
		OrganizacionRef  string `json:"organizacion_ref"`
		ExpedienteRef    string `json:"expediente_ref"`
		VersionObservada uint64 `json:"version_observada"`
		CategoriaRef     string `json:"categoria_ref"`
		CentroRef        string `json:"centro_ref"`
		ActorRef         string `json:"actor_ref"`
		PerfilRef        string `json:"perfil_ref"`
	}{EsquemaConsultaEstadoSeleccionOfertaCT, q.OrganizacionRef, q.ExpedienteRef,
		q.VersionObservada, q.CategoriaRef, q.CentroRef, q.ActorRef, q.PerfilRef})
	if err != nil || len(b) > 65536 {
		return "", ErrSeleccionOfertaCTInvalida
	}
	return string(b), nil
}

type AutorizadorConsultaEstadoSeleccionOfertaCT interface {
	AutorizarConsultaEstadoSeleccionOfertaCT(context.Context, SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT) (AutorizacionSeleccionOfertaCT, error)
}

type PreparadorConsultaEstadoSeleccionOfertaCT interface {
	PrepararConsultaEstadoSeleccionOfertaCT(context.Context, string) (SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT, error)
}

type ContextoExpedienteSeleccionOfertaCT struct {
	VersionActual    uint64
	CategoriaRef     string
	CentroRef        string
	UnidadGestoraRef string
}

// El lector preparatorio devuelve solo el ámbito CT vigente. La lectura de
// O6 y su resultado ocurren después, con una decisión V3 propia.
type LectorContextoExpedienteSeleccionOfertaCT interface {
	LeerContextoExpedienteSeleccionOfertaCT(context.Context, string, string) (ContextoExpedienteSeleccionOfertaCT, error)
}

// EstadoSeleccionOfertaPorExpedienteCT devuelve sólo la ejecución original
// vigente o, en su ausencia, el último descarte. La aplicación comprueba el
// comando y el recibo antes de exponer una proyección minimizada.
type EstadoSeleccionOfertaPorExpedienteCT struct {
	Encontrada bool
	Intencion  IntencionSeleccionOfertaCT
	Estado     EstadoSeleccionOfertaCT
}

type RegistroConsultaEstadoSeleccionOfertaCT interface {
	ConsultarEstadoPorExpediente(context.Context, SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT, AutorizacionSeleccionOfertaCT) (EstadoSeleccionOfertaPorExpedienteCT, error)
}
