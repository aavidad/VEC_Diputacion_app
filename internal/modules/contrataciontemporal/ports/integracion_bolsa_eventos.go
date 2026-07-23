package ports

import (
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

func (EnlaceEventoLlamamientoBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*EnlaceEventoLlamamientoBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

type datosEnlaceEventoLlamamientoBolsa struct {
	organizacionRef, expedienteRef, correlacionRef string
	versionExpediente                              uint64
	finalidad, accion, recurso                     ReferenciaVersionadaIntegracionBolsa
	necesidad, bolsa, orden, politica              ReferenciaVersionadaIntegracionBolsa
	llamamientoRef                                 string
	seleccionRef                                   SeudonimoSeleccionBolsa
	retencionSeleccion                             ReferenciaVersionadaIntegracionBolsa
	peticionRef, huellaPeticion                    string
	reciboRef, huellaRecibo                        string
	peticionSolicitadaEn, peticionValidaHasta      time.Time
	reciboConfirmadaEn, reciboEvidenciaEmitidaEn   time.Time
	reciboEvidenciaValidaHasta, reciboRetenerHasta time.Time
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
			organizacionRef:            contexto.OrganizacionRef,
			expedienteRef:              contexto.ExpedienteRef,
			correlacionRef:             contexto.CorrelacionRef,
			versionExpediente:          contexto.VersionExpediente,
			finalidad:                  contexto.Finalidad,
			accion:                     preparacion.Recibo.AccionEvento,
			recurso:                    preparacion.Recibo.Propuesta,
			necesidad:                  preparacion.Recibo.Necesidad,
			bolsa:                      preparacion.Recibo.Bolsa,
			orden:                      preparacion.Recibo.Orden,
			politica:                   preparacion.Recibo.Politica,
			llamamientoRef:             preparacion.Recibo.LlamamientoRef,
			seleccionRef:               preparacion.Recibo.SeleccionRef,
			retencionSeleccion:         preparacion.Recibo.RetencionSeleccion,
			peticionRef:                contexto.OperacionRef,
			huellaPeticion:             huellaBytesBolsa(materialPeticion),
			reciboRef:                  preparacion.Recibo.ReciboRef,
			huellaRecibo:               huellaBytesBolsa(materialRecibo),
			peticionSolicitadaEn:       contexto.SolicitadaEn,
			peticionValidaHasta:        contexto.ValidaHasta,
			reciboConfirmadaEn:         preparacion.Recibo.ConfirmadaEn,
			reciboEvidenciaEmitidaEn:   preparacion.Recibo.Procedencia.Evidencia.EmitidaEn,
			reciboEvidenciaValidaHasta: preparacion.Recibo.Procedencia.Evidencia.ValidaHasta,
			reciboRetenerHasta:         preparacion.Recibo.Procedencia.Evidencia.RetenerHasta,
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
		d.necesidad.Validar() == nil && d.bolsa.Validar() == nil &&
		d.orden.Validar() == nil && d.politica.Validar() == nil &&
		domain.ReferenciaOpacaValida(d.llamamientoRef) &&
		d.seleccionRef.Validar() == nil &&
		d.retencionSeleccion.Validar() == nil &&
		domain.ReferenciaOpacaValida(d.peticionRef) &&
		huellaSHA256Valida(d.huellaPeticion) &&
		domain.ReferenciaOpacaValida(d.reciboRef) &&
		huellaSHA256Valida(d.huellaRecibo) &&
		instanteBolsaCanonico(d.peticionSolicitadaEn) &&
		instanteBolsaCanonico(d.peticionValidaHasta) &&
		instanteBolsaCanonico(d.reciboConfirmadaEn) &&
		instanteBolsaCanonico(d.reciboEvidenciaEmitidaEn) &&
		instanteBolsaCanonico(d.reciboEvidenciaValidaHasta) &&
		instanteBolsaCanonico(d.reciboRetenerHasta) &&
		d.peticionValidaHasta.After(d.peticionSolicitadaEn) &&
		!d.reciboConfirmadaEn.Before(d.peticionSolicitadaEn) &&
		!d.reciboEvidenciaEmitidaEn.Before(d.reciboConfirmadaEn) &&
		d.reciboEvidenciaEmitidaEn.Before(d.peticionValidaHasta) &&
		d.reciboEvidenciaValidaHasta.After(d.reciboEvidenciaEmitidaEn) &&
		!d.reciboEvidenciaValidaHasta.After(d.peticionValidaHasta) &&
		d.reciboRetenerHasta.After(d.reciboEvidenciaValidaHasta)
}

// EventoLlamamientoBolsa transporta una transición durable. SeleccionRef es
// dato personal seudonimizado, excluido de logs y sujeto a RetencionSeleccion.
type EventoLlamamientoBolsa struct {
	EventoRef                  string                               `json:"evento_ref"`
	OrganizacionRef            string                               `json:"organizacion_ref"`
	ExpedienteRef              string                               `json:"expediente_ref"`
	VersionExpedienteEsperada  uint64                               `json:"version_expediente_esperada"`
	CorrelacionRef             string                               `json:"correlacion_ref"`
	Finalidad                  ReferenciaVersionadaIntegracionBolsa `json:"finalidad"`
	Accion                     ReferenciaVersionadaIntegracionBolsa `json:"accion"`
	Recurso                    ReferenciaVersionadaIntegracionBolsa `json:"recurso"`
	PeticionRef                string                               `json:"peticion_ref"`
	HuellaPeticionSHA256       string                               `json:"huella_peticion_sha256"`
	ReciboRef                  string                               `json:"recibo_ref"`
	HuellaReciboSHA256         string                               `json:"huella_recibo_sha256"`
	PeticionSolicitadaEn       time.Time                            `json:"peticion_solicitada_en"`
	PeticionValidaHasta        time.Time                            `json:"peticion_valida_hasta"`
	ReciboConfirmadaEn         time.Time                            `json:"recibo_confirmada_en"`
	ReciboEvidenciaEmitidaEn   time.Time                            `json:"recibo_evidencia_emitida_en"`
	ReciboEvidenciaValidaHasta time.Time                            `json:"recibo_evidencia_valida_hasta"`
	ReciboRetenerHasta         time.Time                            `json:"recibo_retener_hasta"`
	Necesidad                  ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa                      ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Orden                      ReferenciaVersionadaIntegracionBolsa `json:"orden"`
	Politica                   ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	Propuesta                  ReferenciaVersionadaIntegracionBolsa `json:"propuesta"`
	LlamamientoRef             string                               `json:"llamamiento_ref"`
	SeleccionRef               SeudonimoSeleccionBolsa              `json:"seleccion_ref"`
	RetencionSeleccion         ReferenciaVersionadaIntegracionBolsa `json:"retencion_seleccion"`
	Tipo                       ReferenciaVersionadaIntegracionBolsa `json:"tipo"`
	Estado                     ReferenciaVersionadaIntegracionBolsa `json:"estado"`
	SecuenciaAnterior          uint64                               `json:"secuencia_anterior"`
	Secuencia                  uint64                               `json:"secuencia"`
	HuellaCargaSHA256          string                               `json:"huella_carga_sha256"`
	OcurridoEn                 time.Time                            `json:"ocurrido_en"`
	PublicadoEn                time.Time                            `json:"publicado_en"`
	Procedencia                ProcedenciaIntegracionBolsa          `json:"procedencia"`
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
	if e.ValidarEn(instante) != nil || e.validarVinculo(enlace) != nil {
		return ErrEventoBolsaInvalido
	}
	return nil
}

// ValidarDurableParaEn coteja un evento histórico sin reabrir la ventana de
// transporte. Solo es válido mientras la retención firmada siga vigente.
func (e EventoLlamamientoBolsa) ValidarDurableParaEn(
	enlace EnlaceEventoLlamamientoBolsa,
	instante time.Time,
) error {
	if e.validarEstructuraDurable() != nil ||
		!instanteBolsaCanonico(instante) ||
		instante.Before(e.Procedencia.Evidencia.EmitidaEn) ||
		!instante.Before(e.Procedencia.Evidencia.RetenerHasta) ||
		e.validarVinculo(enlace) != nil {
		return ErrEventoBolsaInvalido
	}
	return nil
}

func (e EventoLlamamientoBolsa) validarVinculo(
	enlace EnlaceEventoLlamamientoBolsa,
) error {
	if !enlace.valido() {
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
		e.PeticionSolicitadaEn != d.peticionSolicitadaEn ||
		e.PeticionValidaHasta != d.peticionValidaHasta ||
		e.ReciboConfirmadaEn != d.reciboConfirmadaEn ||
		e.ReciboEvidenciaEmitidaEn != d.reciboEvidenciaEmitidaEn ||
		e.ReciboEvidenciaValidaHasta != d.reciboEvidenciaValidaHasta ||
		e.ReciboRetenerHasta != d.reciboRetenerHasta ||
		e.Necesidad != d.necesidad || e.Bolsa != d.bolsa ||
		e.Orden != d.orden || e.Politica != d.politica ||
		e.Propuesta != d.recurso ||
		e.LlamamientoRef != d.llamamientoRef ||
		e.SeleccionRef != d.seleccionRef ||
		e.RetencionSeleccion != d.retencionSeleccion {
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
		!instanteBolsaCanonico(e.PeticionSolicitadaEn) ||
		!instanteBolsaCanonico(e.PeticionValidaHasta) ||
		!instanteBolsaCanonico(e.ReciboConfirmadaEn) ||
		!instanteBolsaCanonico(e.ReciboEvidenciaEmitidaEn) ||
		!instanteBolsaCanonico(e.ReciboEvidenciaValidaHasta) ||
		!instanteBolsaCanonico(e.ReciboRetenerHasta) ||
		!e.PeticionValidaHasta.After(e.PeticionSolicitadaEn) ||
		e.ReciboConfirmadaEn.Before(e.PeticionSolicitadaEn) ||
		e.ReciboConfirmadaEn.After(e.PeticionValidaHasta) ||
		e.ReciboEvidenciaEmitidaEn.Before(e.ReciboConfirmadaEn) ||
		!e.ReciboEvidenciaValidaHasta.After(e.ReciboEvidenciaEmitidaEn) ||
		e.ReciboEvidenciaValidaHasta.After(e.PeticionValidaHasta) ||
		!e.ReciboRetenerHasta.After(e.ReciboEvidenciaValidaHasta) ||
		e.Necesidad.Validar() != nil || e.Bolsa.Validar() != nil ||
		e.Orden.Validar() != nil || e.Politica.Validar() != nil ||
		e.Propuesta.Validar() != nil || e.Recurso != e.Propuesta ||
		!domain.ReferenciaOpacaValida(e.LlamamientoRef) ||
		e.SeleccionRef.Validar() != nil ||
		e.RetencionSeleccion.Validar() != nil ||
		e.Tipo.Validar() != nil || e.Estado.Validar() != nil ||
		!enteroSeguroBolsa(e.Secuencia) ||
		e.SecuenciaAnterior > MaximoEnteroSeguroIntegracionBolsa ||
		e.SecuenciaAnterior+1 != e.Secuencia ||
		!huellaSHA256Valida(e.HuellaCargaSHA256) ||
		!instanteBolsaCanonico(e.OcurridoEn) ||
		!instanteBolsaCanonico(e.PublicadoEn) ||
		e.OcurridoEn.Before(e.ReciboConfirmadaEn) ||
		e.PublicadoEn.Before(e.OcurridoEn) ||
		!e.Procedencia.validarNominal() ||
		e.Procedencia.Evidencia.EmitidaEn.Before(e.PublicadoEn) ||
		e.Procedencia.Evidencia.RetenerHasta != e.ReciboRetenerHasta ||
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

// reautenticarEvento comprueba un evento conservado tras un reinicio. La
// ventana de transporte puede haber expirado, pero la evidencia, el enlace y
// los bytes canónicos deben seguir coincidiendo exactamente.
func (v *VerificadorEvidenciaIntegracionBolsa) reautenticarEvento(
	ctx context.Context,
	evento EventoLlamamientoBolsa,
	enlace EnlaceEventoLlamamientoBolsa,
	evidencia EvidenciaDurableIntegracionBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if evento.ValidarDurableParaEn(enlace, instante) != nil ||
		evidencia.TipoMaterial != "evento_llamamiento" {
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
	preparadoEn time.Time
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
	instante time.Time,
) (ComandoRegistrarEventoBolsa, error) {
	evidencia := nuevaEvidenciaDurableBolsa(
		"evento_llamamiento",
		evento.PeticionRef,
		materialEnlaceEventoBolsa(enlace),
		materialEventoBolsa(evento),
		evento.Procedencia,
	)
	if !instanteBolsaCanonico(instante) ||
		!comprobante.instanteVerificacion().Equal(instante) ||
		evento.ValidarDurableParaEn(enlace, instante) != nil ||
		!comprobante.coincide(evidencia) {
		return ComandoRegistrarEventoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	clon := evento
	return ComandoRegistrarEventoBolsa{
		evento: &clon, enlace: enlace, comprobante: comprobante,
		preparadoEn: instante,
	}, nil
}

func nuevoComandoRegistrarEventoHistoricoBolsa(
	evento EventoLlamamientoBolsa,
	enlace EnlaceEventoLlamamientoBolsa,
	comprobante ComprobanteEvidenciaIntegracionBolsa,
	instanteActual time.Time,
) (ComandoRegistrarEventoBolsa, error) {
	evidencia := nuevaEvidenciaDurableBolsa(
		"evento_llamamiento",
		evento.PeticionRef,
		materialEnlaceEventoBolsa(enlace),
		materialEventoBolsa(evento),
		evento.Procedencia,
	)
	verificadaEn := comprobante.instanteVerificacion()
	if !instanteBolsaCanonico(instanteActual) ||
		!instanteBolsaCanonico(verificadaEn) ||
		instanteActual.Before(verificadaEn) ||
		evento.ValidarDurableParaEn(enlace, instanteActual) != nil ||
		!comprobante.coincide(evidencia) {
		return ComandoRegistrarEventoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	clon := evento
	return ComandoRegistrarEventoBolsa{
		evento: &clon, enlace: enlace, comprobante: comprobante,
		preparadoEn: instanteActual,
	}, nil
}

// DatosParaEfectoEn es la única apertura de la capacidad de registro. El
// adaptador debe invocarla con su reloj confiable dentro de la misma
// transacción CAS que escribe inbox, estado, auditoría y outbox.
func (c ComandoRegistrarEventoBolsa) DatosParaEfectoEn(
	instanteActual time.Time,
) (
	EventoLlamamientoBolsa,
	ComprobanteEvidenciaIntegracionBolsa,
	error,
) {
	if c.evento == nil ||
		!instanteBolsaCanonico(instanteActual) ||
		!instanteBolsaCanonico(c.preparadoEn) ||
		instanteActual.Before(c.preparadoEn) {
		return EventoLlamamientoBolsa{}, ComprobanteEvidenciaIntegracionBolsa{},
			ErrEventoBolsaInvalido
	}
	evento := *c.evento
	reconstruido, err := nuevoComandoRegistrarEventoHistoricoBolsa(
		evento,
		c.enlace,
		c.comprobante,
		instanteActual,
	)
	if err != nil || reconstruido.evento == nil {
		return EventoLlamamientoBolsa{}, ComprobanteEvidenciaIntegracionBolsa{},
			ErrEventoBolsaInvalido
	}
	return evento, c.comprobante, nil
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
	c.referencia("necesidad", d.necesidad)
	c.referencia("bolsa", d.bolsa)
	c.referencia("orden", d.orden)
	c.referencia("politica", d.politica)
	c.campo("llamamiento_ref", d.llamamientoRef)
	c.campo("seleccion_ref_seudonimizada", d.seleccionRef.valorCanonico())
	c.referencia("retencion_seleccion", d.retencionSeleccion)
	c.instante("peticion_solicitada_en", d.peticionSolicitadaEn)
	c.instante("peticion_valida_hasta", d.peticionValidaHasta)
	c.instante("recibo_confirmada_en", d.reciboConfirmadaEn)
	c.instante("recibo_evidencia_emitida_en", d.reciboEvidenciaEmitidaEn)
	c.instante("recibo_evidencia_valida_hasta", d.reciboEvidenciaValidaHasta)
	c.instante("recibo_retener_hasta", d.reciboRetenerHasta)
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
	c.instante("peticion_solicitada_en", evento.PeticionSolicitadaEn)
	c.instante("peticion_valida_hasta", evento.PeticionValidaHasta)
	c.instante("recibo_confirmada_en", evento.ReciboConfirmadaEn)
	c.instante("recibo_evidencia_emitida_en", evento.ReciboEvidenciaEmitidaEn)
	c.instante("recibo_evidencia_valida_hasta", evento.ReciboEvidenciaValidaHasta)
	c.instante("recibo_retener_hasta", evento.ReciboRetenerHasta)
	c.referencia("necesidad", evento.Necesidad)
	c.referencia("bolsa", evento.Bolsa)
	c.referencia("orden", evento.Orden)
	c.referencia("politica", evento.Politica)
	c.referencia("propuesta", evento.Propuesta)
	c.campo("llamamiento_ref", evento.LlamamientoRef)
	c.campo(
		"seleccion_ref_seudonimizada",
		evento.SeleccionRef.valorCanonico(),
	)
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
