package domain

import "time"

type DatosActuacion struct {
	AccionClave   ClaveCatalogo   `json:"accion_clave"`
	ActorRef      string          `json:"actor_ref"`
	UnidadRef     string          `json:"unidad_ref"`
	ReciboRef     string          `json:"recibo_ref"`
	RealizadaEn   time.Time       `json:"realizada_en"`
	FaseDestino   ClaveFase       `json:"fase_destino"`
	EstadoDestino EstadoOperativo `json:"estado_destino"`
	Observaciones string          `json:"observaciones,omitempty"`
	DocumentosRef []string        `json:"documentos_ref,omitempty"`
}

func (d DatosActuacion) validar() error {
	if !d.AccionClave.Valida() || !referenciaValida(d.ActorRef) ||
		!referenciaValida(d.UnidadRef) || !referenciaValida(d.ReciboRef) ||
		!instanteCanonico(d.RealizadaEn) || !d.FaseDestino.Valida() ||
		!d.EstadoDestino.Valido() || !textoValido(d.Observaciones, 2000, true) ||
		!referenciasUnicasValidas(d.DocumentosRef, 64) {
		return ErrDatoInvalido
	}
	return nil
}

type Actuacion struct {
	Secuencia         uint64          `json:"secuencia"`
	VersionExpediente uint64          `json:"version_expediente"`
	AccionClave       ClaveCatalogo   `json:"accion_clave"`
	ActorRef          string          `json:"actor_ref"`
	UnidadRef         string          `json:"unidad_ref"`
	ReciboRef         string          `json:"recibo_ref"`
	RealizadaEn       time.Time       `json:"realizada_en"`
	FaseOrigen        ClaveFase       `json:"fase_origen"`
	FaseDestino       ClaveFase       `json:"fase_destino"`
	EstadoOrigen      EstadoOperativo `json:"estado_origen"`
	EstadoDestino     EstadoOperativo `json:"estado_destino"`
	Observaciones     string          `json:"observaciones,omitempty"`
	DocumentosRef     []string        `json:"documentos_ref,omitempty"`
}

type Expediente struct {
	Referencia      string                `json:"referencia"`
	OrganizacionRef string                `json:"organizacion_ref"`
	NumeroVisible   string                `json:"numero_visible"`
	Version         uint64                `json:"version"`
	Flujo           ReferenciaFlujo       `json:"flujo"`
	FaseActual      ClaveFase             `json:"fase_actual"`
	EstadoActual    EstadoOperativo       `json:"estado_actual"`
	Solicitud       SolicitudCentro       `json:"solicitud"`
	Analisis        *AnalisisRRHH         `json:"analisis,omitempty"`
	ViaCobertura    *DecisionViaCobertura `json:"via_cobertura,omitempty"`
	// DecisionesCobertura conserva la cadena gobernada de solo adición. La
	// última publicación coincide con la proyección ViaCobertura.
	DecisionesCobertura []PublicacionDecisionCoberturaGobernada `json:"decisiones_cobertura,omitempty"`
	Asignacion          *AsignacionUnidad                       `json:"asignacion,omitempty"`
	CreadoEn            time.Time                               `json:"creado_en"`
	ActualizadoEn       time.Time                               `json:"actualizado_en"`
	Actuaciones         []Actuacion                             `json:"actuaciones"`
}

type AltaExpediente struct {
	Referencia      string
	OrganizacionRef string
	NumeroVisible   string
	Flujo           ReferenciaFlujo
	FaseInicial     ClaveFase
	Solicitud       SolicitudCentro
	Actuacion       DatosActuacion
}

func NuevoExpediente(alta AltaExpediente) (Expediente, error) {
	if !referenciaValida(alta.Referencia) ||
		!referenciaValida(alta.OrganizacionRef) ||
		!patronNumero.MatchString(alta.NumeroVisible) ||
		alta.Flujo.Validar() != nil || !alta.FaseInicial.Valida() ||
		alta.Solicitud.Validar() != nil || alta.Actuacion.validar() != nil ||
		alta.Actuacion.FaseDestino != alta.FaseInicial ||
		alta.Actuacion.EstadoDestino != EstadoEnCurso {
		return Expediente{}, ErrExpedienteInvalido
	}
	expediente := Expediente{
		Referencia: alta.Referencia, OrganizacionRef: alta.OrganizacionRef,
		NumeroVisible: alta.NumeroVisible, Version: 1,
		Flujo: alta.Flujo, FaseActual: alta.FaseInicial, EstadoActual: EstadoEnCurso,
		Solicitud: alta.Solicitud.clonar(), CreadoEn: alta.Actuacion.RealizadaEn,
		ActualizadoEn: alta.Actuacion.RealizadaEn,
	}
	expediente.Actuaciones = []Actuacion{expediente.nuevaActuacion(
		"", EstadoPendiente, alta.Actuacion, 1,
	)}
	if expediente.Validar() != nil {
		return Expediente{}, ErrExpedienteInvalido
	}
	return expediente, nil
}

func (e Expediente) RegistrarAnalisis(
	versionEsperada uint64,
	analisis AnalisisRRHH,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || analisis.Validar() != nil || actuacion.validar() != nil ||
		analisis.ActuacionRegistro != nil ||
		analisis.ValidacionRC.ValidadaEn.After(actuacion.RealizadaEn) ||
		e.Analisis != nil || e.ViaCobertura != nil || e.Asignacion != nil {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	clon := analisis.clonar()
	vinculo := nuevoVinculoActuacionAnalisis(
		e.Version+1,
		uint64(len(e.Actuaciones)+1),
		actuacion,
	)
	clon.ActuacionRegistro = &vinculo
	siguiente.Analisis = &clon
	return siguiente.confirmarTransicion(actuacion)
}

// RectificarAnalisis sustituye únicamente la proyección vigente del análisis.
// La actuación anterior permanece en la cronología y el adaptador durable debe
// conservar ambos contenidos como versiones de solo adición. No permite
// rectificar bajo una decisión de cobertura ya materializada: ese supuesto
// necesita un flujo expreso de retroacción para no dejar decisiones huérfanas.
func (e Expediente) RectificarAnalisis(
	versionEsperada uint64,
	analisis AnalisisRRHH,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || analisis.Validar() != nil || actuacion.validar() != nil ||
		analisis.ActuacionRegistro != nil ||
		analisis.ValidacionRC.ValidadaEn.After(actuacion.RealizadaEn) ||
		e.Analisis == nil || e.ViaCobertura != nil || e.Asignacion != nil ||
		actuacion.FaseDestino != e.FaseActual ||
		actuacion.EstadoDestino != e.EstadoActual ||
		!textoValido(actuacion.Observaciones, 2000, false) {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	clon := analisis.clonar()
	vinculo := nuevoVinculoActuacionAnalisis(
		e.Version+1,
		uint64(len(e.Actuaciones)+1),
		actuacion,
	)
	clon.ActuacionRegistro = &vinculo
	siguiente.Analisis = &clon
	return siguiente.confirmarTransicion(actuacion)
}

// RegistrarViaCobertura restaura el contrato histórico O1. No acepta las
// acciones reservadas de O4-03: toda decisión nueva usa el método gobernado.
func (e Expediente) RegistrarViaCobertura(
	versionEsperada uint64,
	decision DecisionViaCobertura,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || decision.Validar() != nil || actuacion.validar() != nil ||
		e.Analisis == nil || !e.Analisis.HabilitaAvance() ||
		e.ViaCobertura != nil || e.Asignacion != nil ||
		actuacion.AccionClave == AccionDecidirCoberturaGobernada ||
		actuacion.AccionClave == AccionRectificarCoberturaGobernada {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	clon := decision.clonar()
	siguiente.ViaCobertura = &clon
	return siguiente.confirmarTransicion(actuacion)
}

// RegistrarDecisionCoberturaGobernada materializa una decisión humana ligada
// a propuesta, evidencias, análisis, catálogo, política y actuación exactos.
// No persiste ni concede autoridad; esas garantías corresponden a O4-04.
func (e Expediente) RegistrarDecisionCoberturaGobernada(
	versionEsperada uint64,
	datos DatosAdoptarDecisionCobertura,
	propuesta PropuestaDecisionCobertura,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || actuacion.validar() != nil ||
		e.Analisis == nil || !e.Analisis.HabilitaAvance() ||
		e.ViaCobertura != nil || len(e.DecisionesCobertura) != 0 ||
		e.Asignacion != nil {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	decision, err := crearDecisionCoberturaGobernada(
		datosCrearDecisionCobertura{
			Tipo: DecisionCoberturaInicial, Expediente: e,
			PerfilRef: datos.PerfilRef, ViaElegida: datos.ViaElegida,
			Motivo: datos.Motivo, Propuesta: propuesta, Actuacion: actuacion,
		},
	)
	if err != nil {
		return Expediente{}, err
	}
	publicacion := decision.Publicacion()
	siguiente.ViaCobertura = &DecisionViaCobertura{
		ViaClave:          publicacion.ViaElegida,
		DecisionGobernada: &publicacion,
	}
	siguiente.DecisionesCobertura = append(
		siguiente.DecisionesCobertura,
		publicacion,
	)
	return siguiente.confirmarTransicion(actuacion)
}

// RectificarDecisionCoberturaGobernada añade una decisión y sustituye solo la
// proyección actual. La predecesora debe ser la última decisión y su actuación
// debe seguir siendo la última del expediente.
func (e Expediente) RectificarDecisionCoberturaGobernada(
	versionEsperada uint64,
	datos DatosRectificarDecisionCobertura,
	propuesta PropuestaDecisionCobertura,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || actuacion.validar() != nil ||
		e.Analisis == nil || e.ViaCobertura == nil ||
		e.ViaCobertura.DecisionGobernada == nil ||
		e.Asignacion != nil || len(e.DecisionesCobertura) == 0 ||
		len(e.DecisionesCobertura) >= maximoDecisionesCoberturaGobernadas ||
		actuacion.FaseDestino != e.FaseActual ||
		actuacion.EstadoDestino != e.EstadoActual {
		return Expediente{}, ErrTransicionInvalida
	}
	anterior := e.DecisionesCobertura[len(e.DecisionesCobertura)-1]
	if datos.PredecesoraRef != anterior.Referencia ||
		datos.PredecesoraHuellaSHA256 != anterior.HuellaSHA256 ||
		actuacion.ActorRef == anterior.ActorRef ||
		!actuacion.RealizadaEn.After(anterior.DecididaEn) ||
		anterior.Actuacion.Secuencia != uint64(len(e.Actuaciones)) ||
		anterior.Actuacion.VersionExpediente != e.Version {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	decision, err := crearDecisionCoberturaGobernada(
		datosCrearDecisionCobertura{
			Tipo: DecisionCoberturaRectificacion, Expediente: e,
			PerfilRef: datos.PerfilRef, ViaElegida: datos.ViaElegida,
			Motivo: datos.Motivo, Predecesora: &anterior,
			Propuesta: propuesta, Actuacion: actuacion,
		},
	)
	if err != nil {
		return Expediente{}, err
	}
	publicacion := decision.Publicacion()
	if publicacion.ViaElegida == anterior.ViaElegida {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente.ViaCobertura = &DecisionViaCobertura{
		ViaClave:          publicacion.ViaElegida,
		DecisionGobernada: &publicacion,
	}
	siguiente.DecisionesCobertura = append(
		siguiente.DecisionesCobertura,
		publicacion,
	)
	return siguiente.confirmarTransicion(actuacion)
}

func (e Expediente) RegistrarAsignacion(
	versionEsperada uint64,
	asignacion AsignacionUnidad,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || asignacion.validarEntrada() != nil ||
		actuacion.validar() != nil ||
		e.Analisis == nil || e.ViaCobertura == nil || e.Asignacion != nil ||
		!asignacion.AsignadaEn.Equal(actuacion.RealizadaEn) ||
		asignacion.Observaciones != actuacion.Observaciones {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	clon := asignacion
	vinculo := nuevoVinculoActuacionAsignacion(
		e.Version+1,
		uint64(len(e.Actuaciones)+1),
		actuacion,
		asignacion,
	)
	clon.ActuacionRegistro = &vinculo
	siguiente.Asignacion = &clon
	return siguiente.confirmarTransicion(actuacion)
}

// ReasignarUnidad conserva la asignación anterior en la historia append-only
// del expediente y sustituye únicamente su proyección vigente. La frontera
// durable debe persistir también la instantánea anterior; este método no
// autoriza una reescritura de actuaciones ni un cambio implícito de fase.
func (e Expediente) ReasignarUnidad(
	versionEsperada uint64,
	asignacion AsignacionUnidad,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || asignacion.validarReasignacion() != nil ||
		actuacion.validar() != nil || e.Analisis == nil ||
		e.ViaCobertura == nil || e.Asignacion == nil ||
		!asignacion.AsignadaEn.Equal(actuacion.RealizadaEn) ||
		!asignacion.AsignadaEn.After(e.Asignacion.AsignadaEn) ||
		asignacion.NotificacionRef == e.Asignacion.NotificacionRef ||
		(asignacion.UnidadRef == e.Asignacion.UnidadRef &&
			asignacion.ResponsableRef == e.Asignacion.ResponsableRef) ||
		!textoValido(asignacion.Observaciones, 1000, false) ||
		asignacion.Observaciones != actuacion.Observaciones ||
		actuacion.FaseDestino != e.FaseActual ||
		actuacion.EstadoDestino != e.EstadoActual {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	clon := asignacion
	vinculo := nuevoVinculoActuacionAsignacion(
		e.Version+1,
		uint64(len(e.Actuaciones)+1),
		actuacion,
		asignacion,
	)
	clon.ActuacionRegistro = &vinculo
	siguiente.Asignacion = &clon
	return siguiente.confirmarTransicion(actuacion)
}

func (e Expediente) prepararTransicion(
	versionEsperada uint64,
	actuacion DatosActuacion,
) (Expediente, error) {
	if versionEsperada != e.Version {
		return Expediente{}, ErrVersionEnConflicto
	}
	if actuacion.RealizadaEn.Before(e.ActualizadoEn) ||
		e.EstadoActual == EstadoCompletado || e.EstadoActual == EstadoCancelado {
		return Expediente{}, ErrTransicionInvalida
	}
	return e.Clonar(), nil
}

func (e Expediente) confirmarTransicion(actuacion DatosActuacion) (Expediente, error) {
	origenFase, origenEstado := e.FaseActual, e.EstadoActual
	e.Version++
	e.FaseActual = actuacion.FaseDestino
	e.EstadoActual = actuacion.EstadoDestino
	e.ActualizadoEn = actuacion.RealizadaEn
	e.Actuaciones = append(e.Actuaciones, e.nuevaActuacion(
		origenFase, origenEstado, actuacion, e.Version,
	))
	if e.Validar() != nil {
		return Expediente{}, ErrTransicionInvalida
	}
	return e, nil
}

func (e Expediente) nuevaActuacion(
	faseOrigen ClaveFase,
	estadoOrigen EstadoOperativo,
	datos DatosActuacion,
	version uint64,
) Actuacion {
	return Actuacion{
		Secuencia: uint64(len(e.Actuaciones) + 1), VersionExpediente: version,
		AccionClave: datos.AccionClave, ActorRef: datos.ActorRef,
		UnidadRef: datos.UnidadRef, ReciboRef: datos.ReciboRef,
		RealizadaEn: datos.RealizadaEn, FaseOrigen: faseOrigen,
		FaseDestino: datos.FaseDestino, EstadoOrigen: estadoOrigen,
		EstadoDestino: datos.EstadoDestino, Observaciones: datos.Observaciones,
		DocumentosRef: append([]string(nil), datos.DocumentosRef...),
	}
}
