package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrReincorporacionTitularNoDisponible = errors.New("bolsa: reincorporacion del titular no disponible")

const (
	AccionConsultarReincorporacionTitular    = "bolsa.reincorporacion_titular.consultar"
	FinalidadConsultarReincorporacionTitular = "consulta_reincorporacion_titular"
	AudienciaConsultarReincorporacionTitular = "vec_bolsa_llamamientos.reincorporacion_titular.consultar.v1"
	CampoConsultarReincorporacionTitular     = "reincorporaciones_titular"
)

// SolicitudConsultarReincorporacionesTitular contiene sólo las referencias de
// lectura; no acepta destino, motivo de cambio ni clave de escritura del HTTP.
type SolicitudConsultarReincorporacionesTitular struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef           string
	ParticipacionRef   string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

func (s SolicitudConsultarReincorporacionesTitular) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.BolsaRef == "" || s.ParticipacionRef == "" || s.Correlacion.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return ErrReincorporacionTitularNoDisponible
	}
	return nil
}

// PublicacionReincorporacionTitularCT contiene solo la triple necesaria para
// que la bandeja compruebe el hecho original en CT. La fecha y el cese los
// obtiene Bolsa exclusivamente del verificador CT130.
type PublicacionReincorporacionTitularCT struct {
	EventoRef, OrigenRef, HuellaSHA256 string
	OrigenPosicion                     int64
}

type ResultadoReincorporacionTitularCT struct {
	Reutilizada, CeseAplicado bool
	Estado                    string
	CeseEventoRef             string
	DisponibleDesde           *time.Time
}

type ReincorporacionTitularFicha struct {
	EventoRef, ExpedienteRef, RelacionRef, ReciboCTRef, CeseEventoRef string
	FechaEfectiva                                                     time.Time
	Estado                                                            string
	DisponibleDesde                                                   *time.Time
	ReglaVersion                                                      *int64
	ReglaHuellaSHA256                                                 string
}

type FuenteReincorporacionesTitularCT interface {
	LeerReincorporacionesTitular(context.Context, *CursorContratosParticipacion, int) ([]PublicacionReincorporacionTitularCT, error)
}

type BuzonReincorporacionesTitularCT interface {
	CursorReincorporacionesTitular(context.Context) (CursorContratosParticipacion, bool, error)
	RegistrarReincorporacionTitular(context.Context, PublicacionReincorporacionTitularCT) (ResultadoReincorporacionTitularCT, error)
}

type RepositorioReincorporacionesTitularCT interface {
	ListarReincorporacionesTitular(context.Context, string, string, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ReincorporacionTitularFicha, error)
}
