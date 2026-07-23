package ports

import (
	"bytes"
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// EnlaceEventoLlamamientoBolsa nace de una petición y un recibo autenticados.
// Evita que referencias y digests declarados por el evento se validen entre sí
// sin cotejarlos con los artefactos locales.
type EnlaceEventoLlamamientoBolsa struct {
	datos *datosEnlaceEventoLlamamientoBolsa
}

type datosEnlaceEventoLlamamientoBolsa struct {
	organizacionRef, expedienteRef, correlacionRef string
	versionExpediente                              uint64
	finalidad, accion, recurso                     ReferenciaVersionadaIntegracionBolsa
	peticionRef, huellaPeticion                    string
	reciboRef, huellaRecibo                        string
}

type PreparacionEnlaceEventoLlamamientoBolsa struct {
	Comando     ComandoSolicitarLlamamientoBolsa
	Recibo      ReciboSolicitudLlamamientoBolsa
	Comprobante ComprobanteEvidenciaIntegracionBolsa
}

func NuevoEnlaceEventoLlamamientoBolsa(
	preparacion PreparacionEnlaceEventoLlamamientoBolsa,
) (EnlaceEventoLlamamientoBolsa, error) {
	datosComando, err := preparacion.Comando.datosCanonicos()
	if err != nil ||
		preparacion.Recibo.ValidarDurablePara(preparacion.Comando) != nil ||
		!preparacion.Recibo.PropuestaGenerada {
		return EnlaceEventoLlamamientoBolsa{}, ErrEventoBolsaInvalido
	}
	contexto, err := datosComando.Contexto.datosDurables()
	if err != nil {
		return EnlaceEventoLlamamientoBolsa{}, ErrEventoBolsaInvalido
	}
	materialPeticion := materialComandoLlamamientoBolsa(preparacion.Comando)
	materialRecibo := materialReciboLlamamientoBolsa(
		preparacion.Comando,
		preparacion.Recibo,
	)
	evidencia := nuevaEvidenciaDurableBolsa(
		"recibo_llamamiento",
		contexto.OperacionRef,
		materialPeticion,
		materialRecibo,
		preparacion.Recibo.Procedencia,
	)
	if !preparacion.Comprobante.coincide(evidencia) {
		return EnlaceEventoLlamamientoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return EnlaceEventoLlamamientoBolsa{
		datos: &datosEnlaceEventoLlamamientoBolsa{
			organizacionRef:   contexto.OrganizacionRef,
			expedienteRef:     contexto.ExpedienteRef,
			correlacionRef:    contexto.CorrelacionRef,
			versionExpediente: contexto.VersionExpediente,
			finalidad:         contexto.Finalidad,
			accion:            preparacion.Recibo.AccionEvento,
			recurso:           preparacion.Recibo.Propuesta,
			peticionRef:       contexto.OperacionRef,
			huellaPeticion:    huellaBytesBolsa(materialPeticion),
			reciboRef:         preparacion.Recibo.ReciboRef,
			huellaRecibo:      huellaBytesBolsa(materialRecibo),
		},
	}, nil
}

func (e EnlaceEventoLlamamientoBolsa) valido() bool {
	d := e.datos
	return d != nil &&
		domain.ReferenciaOpacaValida(d.organizacionRef) &&
		domain.ReferenciaOpacaValida(d.expedienteRef) &&
		domain.ReferenciaOpacaValida(d.correlacionRef) &&
		enteroSeguroBolsa(d.versionExpediente) &&
		d.finalidad.Validar() == nil && d.accion.Validar() == nil &&
		d.recurso.Validar() == nil &&
		domain.ReferenciaOpacaValida(d.peticionRef) &&
		huellaSHA256Valida(d.huellaPeticion) &&
		domain.ReferenciaOpacaValida(d.reciboRef) &&
		huellaSHA256Valida(d.huellaRecibo)
}

// EventoLlamamientoBolsa transporta una transición durable. SeleccionRef es
// dato personal seudonimizado, excluido de logs y sujeto a RetencionSeleccion.
type EventoLlamamientoBolsa struct {
	EventoRef                 string                               `json:"evento_ref"`
	OrganizacionRef           string                               `json:"organizacion_ref"`
	ExpedienteRef             string                               `json:"expediente_ref"`
	VersionExpedienteEsperada uint64                               `json:"version_expediente_esperada"`
	CorrelacionRef            string                               `json:"correlacion_ref"`
	Finalidad                 ReferenciaVersionadaIntegracionBolsa `json:"finalidad"`
	Accion                    ReferenciaVersionadaIntegracionBolsa `json:"accion"`
	Recurso                   ReferenciaVersionadaIntegracionBolsa `json:"recurso"`
	PeticionRef               string                               `json:"peticion_ref"`
	HuellaPeticionSHA256      string                               `json:"huella_peticion_sha256"`
	ReciboRef                 string                               `json:"recibo_ref"`
	HuellaReciboSHA256        string                               `json:"huella_recibo_sha256"`
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

func (e EventoLlamamientoBolsa) ValidarParaEn(
	enlace EnlaceEventoLlamamientoBolsa,
	instante time.Time,
) error {
	if e.ValidarEn(instante) != nil || !enlace.valido() {
		return ErrEventoBolsaInvalido
	}
	d := enlace.datos
	if e.OrganizacionRef != d.organizacionRef ||
		e.ExpedienteRef != d.expedienteRef ||
		e.VersionExpedienteEsperada != d.versionExpediente ||
		e.CorrelacionRef != d.correlacionRef ||
		e.Finalidad != d.finalidad || e.Accion != d.accion ||
		e.Recurso != d.recurso ||
		e.PeticionRef != d.peticionRef ||
		!huellasBolsaIguales(e.HuellaPeticionSHA256, d.huellaPeticion) ||
		e.ReciboRef != d.reciboRef ||
		!huellasBolsaIguales(e.HuellaReciboSHA256, d.huellaRecibo) ||
		e.Propuesta != d.recurso {
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
		e.Finalidad.Validar() != nil || e.Accion.Validar() != nil ||
		e.Recurso.Validar() != nil ||
		!domain.ReferenciaOpacaValida(e.PeticionRef) ||
		!huellaSHA256Valida(e.HuellaPeticionSHA256) ||
		!domain.ReferenciaOpacaValida(e.ReciboRef) ||
		!huellaSHA256Valida(e.HuellaReciboSHA256) ||
		e.Necesidad.Validar() != nil || e.Bolsa.Validar() != nil ||
		e.Orden.Validar() != nil || e.Politica.Validar() != nil ||
		e.Propuesta.Validar() != nil || e.Recurso != e.Propuesta ||
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
	enlace EnlaceEventoLlamamientoBolsa,
	instante time.Time,
) (
	ComprobanteEvidenciaIntegracionBolsa,
	EvidenciaDurableIntegracionBolsa,
	error,
) {
	if evento.ValidarParaEn(enlace, instante) != nil {
		return ComprobanteEvidenciaIntegracionBolsa{},
			EvidenciaDurableIntegracionBolsa{},
			ErrEventoBolsaInvalido
	}
	return v.verificarFresco(
		ctx,
		"evento_llamamiento",
		evento.PeticionRef,
		materialEnlaceEventoBolsa(enlace),
		materialEventoBolsa(evento),
		evento.Procedencia,
		instante,
	)
}

// ReautenticarEvento comprueba un evento conservado tras un reinicio. La
// ventana de transporte puede haber expirado, pero la evidencia, el enlace y
// los bytes canónicos deben seguir coincidiendo exactamente.
func (v *VerificadorEvidenciaIntegracionBolsa) ReautenticarEvento(
	ctx context.Context,
	evento EventoLlamamientoBolsa,
	enlace EnlaceEventoLlamamientoBolsa,
	evidencia EvidenciaDurableIntegracionBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if evento.validarEstructuraDurable() != nil || !enlace.valido() ||
		evidencia.TipoMaterial != "evento_llamamiento" {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEventoBolsaInvalido
	}
	d := enlace.datos
	if evento.OrganizacionRef != d.organizacionRef ||
		evento.ExpedienteRef != d.expedienteRef ||
		evento.VersionExpedienteEsperada != d.versionExpediente ||
		evento.CorrelacionRef != d.correlacionRef ||
		evento.Finalidad != d.finalidad || evento.Accion != d.accion ||
		evento.Recurso != d.recurso ||
		evento.PeticionRef != d.peticionRef ||
		!huellasBolsaIguales(evento.HuellaPeticionSHA256, d.huellaPeticion) ||
		evento.ReciboRef != d.reciboRef ||
		!huellasBolsaIguales(evento.HuellaReciboSHA256, d.huellaRecibo) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEventoBolsaInvalido
	}
	esperada := nuevaEvidenciaDurableBolsa(
		"evento_llamamiento",
		evento.PeticionRef,
		materialEnlaceEventoBolsa(enlace),
		materialEventoBolsa(evento),
		evento.Procedencia,
	)
	if !evidenciasDurablesBolsaIguales(evidencia, esperada) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return v.reautenticar(
		ctx,
		evidencia,
		materialEnlaceEventoBolsa(enlace),
		materialEventoBolsa(evento),
		instante,
	)
}

type ComandoRegistrarEventoBolsa struct {
	evento      *EventoLlamamientoBolsa
	enlace      EnlaceEventoLlamamientoBolsa
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
	enlace EnlaceEventoLlamamientoBolsa,
	comprobante ComprobanteEvidenciaIntegracionBolsa,
) (ComandoRegistrarEventoBolsa, error) {
	evidencia := nuevaEvidenciaDurableBolsa(
		"evento_llamamiento",
		evento.PeticionRef,
		materialEnlaceEventoBolsa(enlace),
		materialEventoBolsa(evento),
		evento.Procedencia,
	)
	if evento.ValidarParaEn(enlace, comprobante.instanteVerificacion()) != nil ||
		!comprobante.coincide(evidencia) {
		return ComandoRegistrarEventoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	clon := evento
	return ComandoRegistrarEventoBolsa{
		evento: &clon, enlace: enlace, comprobante: comprobante,
	}, nil
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
	reconstruido, err := NuevoComandoRegistrarEventoBolsa(
		evento,
		c.enlace,
		c.comprobante,
	)
	if err != nil || reconstruido.evento == nil {
		return EventoLlamamientoBolsa{}, ComprobanteEvidenciaIntegracionBolsa{},
			ErrEventoBolsaInvalido
	}
	return evento, c.comprobante, nil
}

type AcuseEventoLlamamientoBolsa struct {
	AutoridadRef         string    `json:"autoridad_ref"`
	EventoRef            string    `json:"evento_ref"`
	OrganizacionRef      string    `json:"organizacion_ref"`
	ExpedienteRef        string    `json:"expediente_ref"`
	CorrelacionRef       string    `json:"correlacion_ref"`
	PeticionRef          string    `json:"peticion_ref"`
	HuellaPeticionSHA256 string    `json:"huella_peticion_sha256"`
	ReciboRef            string    `json:"recibo_ref"`
	HuellaReciboSHA256   string    `json:"huella_recibo_sha256"`
	HuellaEventoSHA256   string    `json:"huella_evento_sha256"`
	SecuenciaAnterior    uint64    `json:"secuencia_anterior"`
	Secuencia            uint64    `json:"secuencia"`
	VersionAnterior      uint64    `json:"version_anterior"`
	VersionResultante    uint64    `json:"version_resultante"`
	ActuacionRef         string    `json:"actuacion_ref"`
	AuditoriaRef         string    `json:"auditoria_ref"`
	InboxRef             string    `json:"inbox_ref"`
	RegistradoEn         time.Time `json:"registrado_en"`
}

func (a AcuseEventoLlamamientoBolsa) ValidarPara(evento EventoLlamamientoBolsa) error {
	if evento.validarEstructuraDurable() != nil ||
		a.AutoridadRef != evento.Procedencia.AutoridadRef ||
		a.EventoRef != evento.EventoRef ||
		a.OrganizacionRef != evento.OrganizacionRef ||
		a.ExpedienteRef != evento.ExpedienteRef ||
		a.CorrelacionRef != evento.CorrelacionRef ||
		a.PeticionRef != evento.PeticionRef ||
		!huellasBolsaIguales(a.HuellaPeticionSHA256, evento.HuellaPeticionSHA256) ||
		a.ReciboRef != evento.ReciboRef ||
		!huellasBolsaIguales(a.HuellaReciboSHA256, evento.HuellaReciboSHA256) ||
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
	if primero.HuellaCargaSHA256 != repetido.HuellaCargaSHA256 ||
		!bytes.Equal(materialEventoBolsa(primero), materialEventoBolsa(repetido)) {
		return ErrColisionEventoBolsa
	}
	if primero.validarEstructuraDurable() != nil ||
		repetido.validarEstructuraDurable() != nil {
		return ErrEventoBolsaInvalido
	}
	return nil
}

type BandejaEventosLlamamientoBolsa interface {
	RegistrarEventoLlamamiento(
		context.Context,
		ComandoRegistrarEventoBolsa,
	) (AcuseEventoLlamamientoBolsa, error)
}

func materialEnlaceEventoBolsa(enlace EnlaceEventoLlamamientoBolsa) []byte {
	c := nuevoCanonicoBolsa("enlace-evento-llamamiento")
	if !enlace.valido() {
		c.campo("enlace_invalido", "1")
		return c.bytes()
	}
	d := enlace.datos
	c.campo("organizacion_ref", d.organizacionRef)
	c.campo("expediente_ref", d.expedienteRef)
	c.entero("version_expediente", d.versionExpediente)
	c.campo("correlacion_ref", d.correlacionRef)
	c.referencia("finalidad", d.finalidad)
	c.referencia("accion", d.accion)
	c.referencia("recurso", d.recurso)
	c.campo("peticion_ref", d.peticionRef)
	c.campo("huella_peticion_sha256", d.huellaPeticion)
	c.campo("recibo_ref", d.reciboRef)
	c.campo("huella_recibo_sha256", d.huellaRecibo)
	return c.bytes()
}

func materialEventoBolsa(evento EventoLlamamientoBolsa) []byte {
	c := nuevoCanonicoBolsa("evento-llamamiento-durable")
	c.campo("evento_ref", evento.EventoRef)
	c.campo("organizacion_ref", evento.OrganizacionRef)
	c.campo("expediente_ref", evento.ExpedienteRef)
	c.entero("version_expediente_esperada", evento.VersionExpedienteEsperada)
	c.campo("correlacion_ref", evento.CorrelacionRef)
	c.referencia("finalidad", evento.Finalidad)
	c.referencia("accion", evento.Accion)
	c.referencia("recurso", evento.Recurso)
	c.campo("peticion_ref", evento.PeticionRef)
	c.campo("huella_peticion_sha256", evento.HuellaPeticionSHA256)
	c.campo("recibo_ref", evento.ReciboRef)
	c.campo("huella_recibo_sha256", evento.HuellaReciboSHA256)
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
