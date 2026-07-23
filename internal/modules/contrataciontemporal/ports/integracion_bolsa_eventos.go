package ports

import (
	"bytes"
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// EventoLlamamientoBolsa transporta una transición durable. SeleccionRef es
// dato personal seudonimizado: queda limitado a esta integración, excluido de
// logs/telemetría y sujeto a la política RetencionSeleccion.
type EventoLlamamientoBolsa struct {
	EventoRef                 string                               `json:"evento_ref"`
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
	RetencionSeleccion        ReferenciaVersionadaIntegracionBolsa `json:"retencion_seleccion"`
	Tipo                      ReferenciaVersionadaIntegracionBolsa `json:"tipo"`
	Estado                    ReferenciaVersionadaIntegracionBolsa `json:"estado"`
	SecuenciaAnterior         uint64                               `json:"secuencia_anterior"`
	Secuencia                 uint64                               `json:"secuencia"`
	HuellaCargaSHA256         string                               `json:"huella_carga_sha256"`
	OcurridoEn                time.Time                            `json:"ocurrido_en"`
	PublicadoEn               time.Time                            `json:"publicado_en"`
	Procedencia               ProcedenciaIntegracionBolsa          `json:"procedencia"`
}

func (e EventoLlamamientoBolsa) ValidarEn(instante time.Time) error {
	if e.validarEstructuraDurable() != nil ||
		!e.Procedencia.validarNominalEn(instante) {
		return ErrEventoBolsaInvalido
	}
	return nil
}

func (e EventoLlamamientoBolsa) validarEstructuraDurable() error {
	if !domain.ReferenciaOpacaValida(e.EventoRef) ||
		!domain.ReferenciaOpacaValida(e.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(e.ExpedienteRef) ||
		!enteroSeguroBolsa(e.VersionExpedienteEsperada) ||
		e.VersionExpedienteEsperada >= MaximoEnteroSeguroIntegracionBolsa ||
		!domain.ReferenciaOpacaValida(e.CorrelacionRef) ||
		e.Necesidad.Validar() != nil || e.Bolsa.Validar() != nil ||
		e.Orden.Validar() != nil || e.Politica.Validar() != nil ||
		e.Propuesta.Validar() != nil ||
		!domain.ReferenciaOpacaValida(e.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(e.SeleccionRef) ||
		e.RetencionSeleccion.Validar() != nil ||
		e.Tipo.Validar() != nil || e.Estado.Validar() != nil ||
		!enteroSeguroBolsa(e.Secuencia) ||
		e.SecuenciaAnterior > MaximoEnteroSeguroIntegracionBolsa ||
		e.SecuenciaAnterior+1 != e.Secuencia ||
		!huellaSHA256Valida(e.HuellaCargaSHA256) ||
		!instanteBolsaCanonico(e.OcurridoEn) ||
		!instanteBolsaCanonico(e.PublicadoEn) ||
		e.PublicadoEn.Before(e.OcurridoEn) ||
		!e.Procedencia.validarNominal() ||
		e.Procedencia.Evidencia.EmitidaEn.Before(e.PublicadoEn) ||
		huellaBytesBolsa(materialEventoBolsa(e)) != e.HuellaCargaSHA256 {
		return ErrEventoBolsaInvalido
	}
	return nil
}

func (v *VerificadorEvidenciaIntegracionBolsa) VerificarEvento(
	ctx context.Context,
	evento EventoLlamamientoBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if evento.ValidarEn(instante) != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEventoBolsaInvalido
	}
	material := materialEventoBolsa(evento)
	peticion := solicitudVerificacionEvidenciaBolsa{
		material: material, evidencia: evento.Procedencia.Evidencia,
		autoridadRef: evento.Procedencia.AutoridadRef, organizacionRef: evento.OrganizacionRef,
		expedienteRef: evento.ExpedienteRef, correlacionRef: evento.CorrelacionRef,
		respuestaRef: evento.Procedencia.RespuestaRef, huellaMaterial: huellaBytesBolsa(material),
	}
	comprobante, err := v.verificar(ctx, peticion, instante)
	if err != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, err
	}
	if !comprobante.coincide(peticion) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return comprobante, nil
}

// ComandoRegistrarEventoBolsa solo se construye con el comprobante TCB ligado
// a los mismos bytes. El adaptador de entrada no puede promover el evento por
// sí solo ni reemplazarlo después de la verificación.
type ComandoRegistrarEventoBolsa struct {
	evento      *EventoLlamamientoBolsa
	comprobante ComprobanteEvidenciaIntegracionBolsa
}

func (ComandoRegistrarEventoBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*ComandoRegistrarEventoBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

func NuevoComandoRegistrarEventoBolsa(
	evento EventoLlamamientoBolsa,
	comprobante ComprobanteEvidenciaIntegracionBolsa,
) (ComandoRegistrarEventoBolsa, error) {
	material := materialEventoBolsa(evento)
	peticion := solicitudVerificacionEvidenciaBolsa{
		material: material, evidencia: evento.Procedencia.Evidencia,
		autoridadRef: evento.Procedencia.AutoridadRef, organizacionRef: evento.OrganizacionRef,
		expedienteRef: evento.ExpedienteRef, correlacionRef: evento.CorrelacionRef,
		respuestaRef: evento.Procedencia.RespuestaRef, huellaMaterial: huellaBytesBolsa(material),
	}
	if evento.ValidarEn(comprobante.instanteVerificacion()) != nil ||
		!comprobante.coincide(peticion) {
		return ComandoRegistrarEventoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	clon := evento
	return ComandoRegistrarEventoBolsa{evento: &clon, comprobante: comprobante}, nil
}

func (c ComandoRegistrarEventoBolsa) Datos() (
	EventoLlamamientoBolsa,
	ComprobanteEvidenciaIntegracionBolsa,
	error,
) {
	if c.evento == nil {
		return EventoLlamamientoBolsa{}, ComprobanteEvidenciaIntegracionBolsa{},
			ErrEventoBolsaInvalido
	}
	evento := *c.evento
	reconstruido, err := NuevoComandoRegistrarEventoBolsa(evento, c.comprobante)
	if err != nil || reconstruido.evento == nil {
		return EventoLlamamientoBolsa{}, ComprobanteEvidenciaIntegracionBolsa{},
			ErrEventoBolsaInvalido
	}
	return evento, c.comprobante, nil
}

func (c ComprobanteEvidenciaIntegracionBolsa) instanteVerificacion() time.Time {
	if c.datos == nil {
		return time.Time{}
	}
	return c.datos.verificadaEn
}

// AcuseEventoLlamamientoBolsa no cambia entre la primera aplicación y un
// replay idéntico. Por eso no incluye banderas "nuevo/duplicado".
type AcuseEventoLlamamientoBolsa struct {
	AutoridadRef       string    `json:"autoridad_ref"`
	EventoRef          string    `json:"evento_ref"`
	OrganizacionRef    string    `json:"organizacion_ref"`
	ExpedienteRef      string    `json:"expediente_ref"`
	CorrelacionRef     string    `json:"correlacion_ref"`
	HuellaEventoSHA256 string    `json:"huella_evento_sha256"`
	SecuenciaAnterior  uint64    `json:"secuencia_anterior"`
	Secuencia          uint64    `json:"secuencia"`
	VersionAnterior    uint64    `json:"version_anterior"`
	VersionResultante  uint64    `json:"version_resultante"`
	ActuacionRef       string    `json:"actuacion_ref"`
	AuditoriaRef       string    `json:"auditoria_ref"`
	InboxRef           string    `json:"inbox_ref"`
	RegistradoEn       time.Time `json:"registrado_en"`
}

func (a AcuseEventoLlamamientoBolsa) ValidarPara(evento EventoLlamamientoBolsa) error {
	if evento.validarEstructuraDurable() != nil ||
		a.AutoridadRef != evento.Procedencia.AutoridadRef ||
		a.EventoRef != evento.EventoRef ||
		a.OrganizacionRef != evento.OrganizacionRef ||
		a.ExpedienteRef != evento.ExpedienteRef ||
		a.CorrelacionRef != evento.CorrelacionRef ||
		a.HuellaEventoSHA256 != evento.HuellaCargaSHA256 ||
		a.SecuenciaAnterior != evento.SecuenciaAnterior ||
		a.Secuencia != evento.Secuencia ||
		a.VersionAnterior != evento.VersionExpedienteEsperada ||
		evento.VersionExpedienteEsperada >= MaximoEnteroSeguroIntegracionBolsa ||
		a.VersionResultante != evento.VersionExpedienteEsperada+1 ||
		!domain.ReferenciaOpacaValida(a.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(a.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(a.InboxRef) ||
		!instanteBolsaCanonico(a.RegistradoEn) ||
		a.RegistradoEn.Before(evento.PublicadoEn) {
		return ErrAcuseEventoBolsaNoConfiable
	}
	return nil
}

// ValidarReplayAcuseEventoBolsa exige igualdad completa. Un adaptador debe
// recuperar el acuse original, no fabricar otro con nuevas referencias.
func ValidarReplayAcuseEventoBolsa(
	primero AcuseEventoLlamamientoBolsa,
	repetido AcuseEventoLlamamientoBolsa,
	evento EventoLlamamientoBolsa,
) error {
	if primero.ValidarPara(evento) != nil || repetido.ValidarPara(evento) != nil ||
		primero != repetido {
		return ErrAcuseEventoBolsaNoConfiable
	}
	return nil
}

// ValidarIdentidadEventoBolsa detecta la colisión
// (autoridad_ref, evento_ref) con otra carga.
func ValidarIdentidadEventoBolsa(
	primero EventoLlamamientoBolsa,
	repetido EventoLlamamientoBolsa,
) error {
	mismaIdentidad := primero.Procedencia.AutoridadRef == repetido.Procedencia.AutoridadRef &&
		primero.EventoRef == repetido.EventoRef
	if !mismaIdentidad ||
		!domain.ReferenciaOpacaValida(primero.Procedencia.AutoridadRef) ||
		!domain.ReferenciaOpacaValida(primero.EventoRef) {
		return ErrEventoBolsaInvalido
	}
	if primero.HuellaCargaSHA256 != repetido.HuellaCargaSHA256 {
		return ErrColisionEventoBolsa
	}
	if materialPrimero, materialRepetido := materialEventoBolsa(primero), materialEventoBolsa(repetido); !materialesCanonicosBolsaIguales(materialPrimero, materialRepetido) {
		return ErrColisionEventoBolsa
	}
	if primero.validarEstructuraDurable() != nil ||
		repetido.validarEstructuraDurable() != nil {
		return ErrEventoBolsaInvalido
	}
	return nil
}

// BandejaEventosLlamamientoBolsa aplica por transacción el CAS de versión, la
// secuencia exacta del flujo, actuación, auditoría e inbox. Replay idéntico
// devuelve el mismo acuse; misma identidad con otra carga devuelve
// ErrColisionEventoBolsa; salto/retroceso devuelve ErrSecuenciaEventoBolsaConflicto.
type BandejaEventosLlamamientoBolsa interface {
	RegistrarEventoLlamamiento(
		context.Context,
		ComandoRegistrarEventoBolsa,
	) (AcuseEventoLlamamientoBolsa, error)
}

func materialEventoBolsa(evento EventoLlamamientoBolsa) []byte {
	c := nuevoCanonicoBolsa("evento-llamamiento-durable")
	c.campo("evento_ref", evento.EventoRef)
	c.campo("organizacion_ref", evento.OrganizacionRef)
	c.campo("expediente_ref", evento.ExpedienteRef)
	c.entero("version_expediente_esperada", evento.VersionExpedienteEsperada)
	c.campo("correlacion_ref", evento.CorrelacionRef)
	c.referencia("necesidad", evento.Necesidad)
	c.referencia("bolsa", evento.Bolsa)
	c.referencia("orden", evento.Orden)
	c.referencia("politica", evento.Politica)
	c.referencia("propuesta", evento.Propuesta)
	c.campo("llamamiento_ref", evento.LlamamientoRef)
	c.campo("seleccion_ref_seudonimizada", evento.SeleccionRef)
	c.referencia("retencion_seleccion", evento.RetencionSeleccion)
	c.referencia("tipo", evento.Tipo)
	c.referencia("estado", evento.Estado)
	c.entero("secuencia_anterior", evento.SecuenciaAnterior)
	c.entero("secuencia", evento.Secuencia)
	c.instante("ocurrido_en", evento.OcurridoEn)
	c.instante("publicado_en", evento.PublicadoEn)
	c.procedencia(evento.Procedencia)
	return c.bytes()
}

func materialesCanonicosBolsaIguales(primero, segundo []byte) bool {
	return bytes.Equal(primero, segundo)
}
