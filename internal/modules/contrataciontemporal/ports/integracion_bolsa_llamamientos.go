package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// ComandoSolicitarLlamamientoBolsa ordena a Bolsa proponer el siguiente
// llamamiento conforme a una instantánea y política exactas. No incorpora
// identidad, posición, disponibilidad ni datos de contacto declarados por un
// cliente.
type ComandoSolicitarLlamamientoBolsa struct {
	Contexto  ContextoPeticionIntegracionBolsa     `json:"contexto"`
	Necesidad ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa     ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Orden     ReferenciaVersionadaIntegracionBolsa `json:"orden"`
	Politica  ReferenciaVersionadaIntegracionBolsa `json:"politica"`
}

func (c ComandoSolicitarLlamamientoBolsa) ValidarEn(instante time.Time) error {
	if c.Contexto.ValidarEn(instante) != nil || c.Necesidad.Validar() != nil ||
		c.Bolsa.Validar() != nil || c.Orden.Validar() != nil ||
		c.Politica.Validar() != nil {
		return ErrPeticionIntegracionBolsaInvalida
	}
	return nil
}

// ResultadoSolicitudLlamamientoBolsa no expone quién ocupa la posición. Si se
// genera una propuesta, SeleccionRef es una referencia opaca de entrega que
// solo puede resolverse mediante un caso de uso autorizado de Bolsa.
type ResultadoSolicitudLlamamientoBolsa struct {
	OperacionRef      string                               `json:"operacion_ref"`
	OrganizacionRef   string                               `json:"organizacion_ref"`
	ExpedienteRef     string                               `json:"expediente_ref"`
	VersionExpediente uint64                               `json:"version_expediente"`
	CorrelacionRef    string                               `json:"correlacion_ref"`
	Necesidad         ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa             ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Orden             ReferenciaVersionadaIntegracionBolsa `json:"orden"`
	Politica          ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	Resultado         ReferenciaVersionadaIntegracionBolsa `json:"resultado"`
	PropuestaGenerada bool                                 `json:"propuesta_generada"`
	Propuesta         ReferenciaVersionadaIntegracionBolsa `json:"propuesta"`
	LlamamientoRef    string                               `json:"llamamiento_ref"`
	SeleccionRef      string                               `json:"seleccion_ref"`
	OrdenSeleccionado uint32                               `json:"orden_seleccionado"`
	Procedencia       ProcedenciaIntegracionBolsa          `json:"procedencia"`
}

func (r ResultadoSolicitudLlamamientoBolsa) ValidarPara(
	comando ComandoSolicitarLlamamientoBolsa,
) error {
	if validarVinculoRespuestaBolsa(
		r.OperacionRef, r.OrganizacionRef, r.ExpedienteRef, r.VersionExpediente, r.CorrelacionRef,
		r.Necesidad, r.Resultado, r.Procedencia, comando.Contexto, comando.Necesidad,
	) != nil || r.Bolsa != comando.Bolsa || r.Orden != comando.Orden ||
		r.Politica != comando.Politica {
		return ErrRespuestaBolsaNoConfiable
	}
	if !r.PropuestaGenerada {
		if r.Propuesta != (ReferenciaVersionadaIntegracionBolsa{}) ||
			r.LlamamientoRef != "" || r.SeleccionRef != "" || r.OrdenSeleccionado != 0 {
			return ErrRespuestaBolsaNoConfiable
		}
		return nil
	}
	if r.Propuesta.Validar() != nil ||
		!domain.ReferenciaOpacaValida(r.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(r.SeleccionRef) ||
		r.OrdenSeleccionado == 0 ||
		r.OrdenSeleccionado > MaximoElementosIntegracionBolsa {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

// GestorLlamamientosBolsa es sustituible por adaptadores locales, de red o
// asíncronos. El contexto debe conservar deadline/cancelación en toda llamada.
type GestorLlamamientosBolsa interface {
	SolicitarLlamamiento(
		context.Context,
		ComandoSolicitarLlamamientoBolsa,
	) (ResultadoSolicitudLlamamientoBolsa, error)
}

// EventoLlamamientoBolsa transporta únicamente referencias y evidencias. El
// tipo y el estado son recursos gobernados y versionados para admitir nuevas
// transiciones sin recompilar contratación temporal.
type EventoLlamamientoBolsa struct {
	EventoRef                 string                               `json:"evento_ref"`
	Secuencia                 uint64                               `json:"secuencia"`
	OrganizacionRef           string                               `json:"organizacion_ref"`
	ExpedienteRef             string                               `json:"expediente_ref"`
	VersionExpedienteEsperada uint64                               `json:"version_expediente_esperada"`
	CorrelacionRef            string                               `json:"correlacion_ref"`
	Necesidad                 ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa                     ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Orden                     ReferenciaVersionadaIntegracionBolsa `json:"orden"`
	Politica                  ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	Propuesta                 ReferenciaVersionadaIntegracionBolsa `json:"propuesta"`
	LlamamientoRef            string                               `json:"llamamiento_ref"`
	SeleccionRef              string                               `json:"seleccion_ref"`
	Tipo                      ReferenciaVersionadaIntegracionBolsa `json:"tipo"`
	Estado                    ReferenciaVersionadaIntegracionBolsa `json:"estado"`
	HuellaCargaSHA256         string                               `json:"huella_carga_sha256"`
	OcurridoEn                time.Time                            `json:"ocurrido_en"`
	PublicadoEn               time.Time                            `json:"publicado_en"`
	Procedencia               ProcedenciaIntegracionBolsa          `json:"procedencia"`
}

func (e EventoLlamamientoBolsa) Validar() error {
	if !domain.ReferenciaOpacaValida(e.EventoRef) || e.Secuencia == 0 ||
		!domain.ReferenciaOpacaValida(e.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(e.ExpedienteRef) ||
		e.VersionExpedienteEsperada == 0 ||
		!domain.ReferenciaOpacaValida(e.CorrelacionRef) ||
		e.Necesidad.Validar() != nil || e.Bolsa.Validar() != nil ||
		e.Orden.Validar() != nil || e.Politica.Validar() != nil ||
		e.Propuesta.Validar() != nil ||
		!domain.ReferenciaOpacaValida(e.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(e.SeleccionRef) ||
		e.Tipo.Validar() != nil || e.Estado.Validar() != nil ||
		!huellaIntegracionBolsaValida(e.HuellaCargaSHA256) ||
		e.HuellaCargaSHA256 != e.Procedencia.Fuente.HuellaSHA256 ||
		!domain.InstanteUTCCanonico(e.OcurridoEn) ||
		!domain.InstanteUTCCanonico(e.PublicadoEn) ||
		e.PublicadoEn.Before(e.OcurridoEn) ||
		e.Procedencia.Validar() != nil ||
		e.Procedencia.EmitidaEn.Before(e.OcurridoEn) ||
		e.Procedencia.EmitidaEn.After(e.PublicadoEn) {
		return ErrEventoBolsaInvalido
	}
	return nil
}

// AcuseEventoLlamamientoBolsa hace explícito si el evento se aplicó o ya
// estaba aplicado. Cualquier otra combinación es inválida; un error del inbox
// no puede convertirse en duplicado ni éxito.
type AcuseEventoLlamamientoBolsa struct {
	EventoRef         string    `json:"evento_ref"`
	Secuencia         uint64    `json:"secuencia"`
	Aplicado          bool      `json:"aplicado"`
	YaRegistrado      bool      `json:"ya_registrado"`
	VersionResultante uint64    `json:"version_resultante"`
	ActuacionRef      string    `json:"actuacion_ref"`
	AuditoriaRef      string    `json:"auditoria_ref"`
	RegistradoEn      time.Time `json:"registrado_en"`
}

func (a AcuseEventoLlamamientoBolsa) ValidarPara(evento EventoLlamamientoBolsa) error {
	if evento.Validar() != nil || a.EventoRef != evento.EventoRef ||
		a.Secuencia != evento.Secuencia || a.Aplicado == a.YaRegistrado ||
		a.VersionResultante <= evento.VersionExpedienteEsperada ||
		!domain.ReferenciaOpacaValida(a.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(a.AuditoriaRef) ||
		!domain.InstanteUTCCanonico(a.RegistradoEn) ||
		a.RegistradoEn.Before(evento.PublicadoEn) {
		return ErrAcuseEventoBolsaNoConfiable
	}
	return nil
}

// BandejaEventosLlamamientoBolsa representa un inbox durable. Su adaptador
// debe aplicar evento, cronología, auditoría y marca de consumo en una sola
// transacción, con unicidad por autoridad, evento y secuencia.
type BandejaEventosLlamamientoBolsa interface {
	RegistrarEventoLlamamiento(
		context.Context,
		EventoLlamamientoBolsa,
	) (AcuseEventoLlamamientoBolsa, error)
}
