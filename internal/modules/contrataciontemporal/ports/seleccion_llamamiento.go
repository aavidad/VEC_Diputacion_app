package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrEjecucionSeleccionLlamamientoInvalida = errors.New(
	"contratacion temporal: ejecucion de seleccion y llamamiento invalida",
)

type SituacionEjecucionSeleccionLlamamiento string

const (
	EjecucionSeleccionLlamamientoPropietaria   SituacionEjecucionSeleccionLlamamiento = "propietaria"
	EjecucionSeleccionLlamamientoConfirmada    SituacionEjecucionSeleccionLlamamiento = "confirmada"
	EjecucionSeleccionLlamamientoOcupada       SituacionEjecucionSeleccionLlamamiento = "ocupada"
	EjecucionSeleccionLlamamientoColision      SituacionEjecucionSeleccionLlamamiento = "colision"
	EjecucionSeleccionLlamamientoIndeterminada SituacionEjecucionSeleccionLlamamiento = "indeterminada"
)

type EfectoSeleccionLlamamiento string

const (
	EfectoPrepararOrdenSeleccionLlamamiento EfectoSeleccionLlamamiento = "preparar_orden"
	EfectoSolicitarSeleccionLlamamiento     EfectoSeleccionLlamamiento = "solicitar_llamamiento"
)

// SolicitudReservaEjecucionSeleccionLlamamiento liga la UUID de intención a
// la orden gobernada y a la cantidad exacta observada antes del primer efecto.
// Los campos permiten cotejar un terminal durable sin repetir la consulta
// volátil; la huella no concede autoridad ni contiene posiciones o personas.
type SolicitudReservaEjecucionSeleccionLlamamiento struct {
	ClaveIdempotencia    string
	HuellaSemantica      string
	OrganizacionRef      string
	ExpedienteRef        string
	VersionExpediente    uint64
	CorrelacionRef       string
	AutoridadSolicitante string
	AutorizacionConsulta ReferenciaVersionadaIntegracionBolsa
	AccionConsulta       ReferenciaVersionadaIntegracionBolsa
	RecursoConsulta      ReferenciaVersionadaIntegracionBolsa
	AccionOrden          ReferenciaVersionadaIntegracionBolsa
	Finalidad            ReferenciaVersionadaIntegracionBolsa
	Necesidad            ReferenciaVersionadaIntegracionBolsa
	Bolsa                ReferenciaVersionadaIntegracionBolsa
	Politica             ReferenciaVersionadaIntegracionBolsa
	MaximoPosiciones     uint32
	CantidadDisponible   uint32
}

// DatosConsultaTerminalAutorizada es la proyeccion minima que un adaptador
// durable puede emplear para resolver un terminal dentro de la autoridad
// exacta. No es una capacidad y no puede construirse de vuelta como una.
type DatosConsultaTerminalAutorizada struct {
	ClaveIdempotencia    string
	OrganizacionRef      string
	ExpedienteRef        string
	VersionExpediente    uint64
	CorrelacionRef       string
	AutoridadSolicitante string
	Autorizacion         ReferenciaVersionadaIntegracionBolsa
	Accion               ReferenciaVersionadaIntegracionBolsa
	Recurso              ReferenciaVersionadaIntegracionBolsa
	Finalidad            ReferenciaVersionadaIntegracionBolsa
}

type datosConsultaTerminalAutorizada struct {
	clave    string
	contexto ContextoPeticionIntegracionBolsa
}

// ConsultaTerminalAutorizada es una capacidad nominal fresca: solo nace de
// un contexto confiable vigente y no admite serializacion ni reconstruccion.
type ConsultaTerminalAutorizada struct {
	datos *datosConsultaTerminalAutorizada
}

func (ConsultaTerminalAutorizada) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*ConsultaTerminalAutorizada) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

func NuevaConsultaTerminalAutorizada(
	clave string,
	contexto ContextoPeticionIntegracionBolsa,
	instante time.Time,
) (ConsultaTerminalAutorizada, error) {
	if !ClaveIdempotenciaValida(clave) {
		return ConsultaTerminalAutorizada{}, ErrEjecucionSeleccionLlamamientoInvalida
	}
	if _, err := contexto.DatosEn(instante); err != nil {
		return ConsultaTerminalAutorizada{}, ErrEjecucionSeleccionLlamamientoInvalida
	}
	return ConsultaTerminalAutorizada{datos: &datosConsultaTerminalAutorizada{
		clave: clave, contexto: contexto,
	}}, nil
}

func (c ConsultaTerminalAutorizada) DatosEn(
	instante time.Time,
) (DatosConsultaTerminalAutorizada, error) {
	if c.datos == nil || !ClaveIdempotenciaValida(c.datos.clave) {
		return DatosConsultaTerminalAutorizada{}, ErrEjecucionSeleccionLlamamientoInvalida
	}
	datos, err := c.datos.contexto.DatosEn(instante)
	if err != nil {
		return DatosConsultaTerminalAutorizada{}, ErrEjecucionSeleccionLlamamientoInvalida
	}
	return DatosConsultaTerminalAutorizada{
		ClaveIdempotencia: c.datos.clave, OrganizacionRef: datos.OrganizacionRef,
		ExpedienteRef: datos.ExpedienteRef, VersionExpediente: datos.VersionExpediente,
		CorrelacionRef: datos.CorrelacionRef, AutoridadSolicitante: datos.AutoridadSolicitante,
		Autorizacion: datos.Autorizacion, Accion: datos.Accion,
		Recurso: datos.Recurso, Finalidad: datos.Finalidad,
	}, nil
}

func (c ConsultaTerminalAutorizada) ValidarSolicitudEn(
	s SolicitudReservaEjecucionSeleccionLlamamiento,
	instante time.Time,
) error {
	datos, err := c.DatosEn(instante)
	if err != nil || s.Validar() != nil ||
		datos.ClaveIdempotencia != s.ClaveIdempotencia ||
		datos.OrganizacionRef != s.OrganizacionRef || datos.ExpedienteRef != s.ExpedienteRef ||
		datos.VersionExpediente != s.VersionExpediente || datos.CorrelacionRef != s.CorrelacionRef ||
		datos.AutoridadSolicitante != s.AutoridadSolicitante ||
		datos.Autorizacion != s.AutorizacionConsulta || datos.Accion != s.AccionConsulta ||
		datos.Recurso != s.RecursoConsulta || datos.Finalidad != s.Finalidad {
		return ErrEjecucionSeleccionLlamamientoInvalida
	}
	return nil
}

func NuevaSolicitudReservaEjecucionSeleccionLlamamiento(
	consulta ConsultaTerminalAutorizada,
	comando ComandoPrepararOrdenBolsa,
	cantidadDisponible uint32,
	instante time.Time,
) (SolicitudReservaEjecucionSeleccionLlamamiento, error) {
	datosConsulta, errConsulta := consulta.DatosEn(instante)
	if errConsulta != nil || cantidadDisponible == 0 || comando.ValidarEn(instante) != nil ||
		comando.MaximoPosiciones < cantidadDisponible {
		return SolicitudReservaEjecucionSeleccionLlamamiento{},
			ErrEjecucionSeleccionLlamamientoInvalida
	}
	contexto, err := comando.Contexto.DatosEn(instante)
	if err != nil {
		return SolicitudReservaEjecucionSeleccionLlamamiento{},
			ErrEjecucionSeleccionLlamamientoInvalida
	}
	if datosConsulta.OrganizacionRef != contexto.OrganizacionRef ||
		datosConsulta.ExpedienteRef != contexto.ExpedienteRef ||
		datosConsulta.VersionExpediente != contexto.VersionExpediente ||
		datosConsulta.CorrelacionRef != contexto.CorrelacionRef ||
		datosConsulta.Finalidad != contexto.Finalidad || datosConsulta.Recurso != comando.Necesidad {
		return SolicitudReservaEjecucionSeleccionLlamamiento{},
			ErrEjecucionSeleccionLlamamientoInvalida
	}
	solicitud := SolicitudReservaEjecucionSeleccionLlamamiento{
		ClaveIdempotencia: datosConsulta.ClaveIdempotencia, OrganizacionRef: contexto.OrganizacionRef,
		ExpedienteRef: contexto.ExpedienteRef, VersionExpediente: contexto.VersionExpediente,
		CorrelacionRef: contexto.CorrelacionRef, AutoridadSolicitante: datosConsulta.AutoridadSolicitante,
		AutorizacionConsulta: datosConsulta.Autorizacion, AccionConsulta: datosConsulta.Accion,
		RecursoConsulta: datosConsulta.Recurso, AccionOrden: contexto.Accion,
		Finalidad: contexto.Finalidad, Necesidad: comando.Necesidad,
		Bolsa: comando.Bolsa, Politica: comando.Politica,
		MaximoPosiciones: comando.MaximoPosiciones, CantidadDisponible: cantidadDisponible,
	}
	solicitud.HuellaSemantica = solicitud.huellaEsperada()
	return solicitud, nil
}

func (s SolicitudReservaEjecucionSeleccionLlamamiento) huellaEsperada() string {
	canon := nuevoCanonicoBolsa("ejecucion-seleccion-llamamiento")
	canon.campo("clave_idempotencia", s.ClaveIdempotencia)
	canon.campo("organizacion_ref", s.OrganizacionRef)
	canon.campo("expediente_ref", s.ExpedienteRef)
	canon.entero("version_expediente", s.VersionExpediente)
	canon.campo("correlacion_ref", s.CorrelacionRef)
	canon.campo("autoridad_solicitante", s.AutoridadSolicitante)
	canon.referencia("autorizacion_consulta", s.AutorizacionConsulta)
	canon.referencia("accion_consulta", s.AccionConsulta)
	canon.referencia("recurso_consulta", s.RecursoConsulta)
	canon.referencia("accion_orden", s.AccionOrden)
	canon.referencia("recurso", s.Bolsa)
	canon.referencia("finalidad", s.Finalidad)
	canon.referencia("necesidad", s.Necesidad)
	canon.referencia("bolsa", s.Bolsa)
	canon.referencia("politica", s.Politica)
	canon.entero("maximo_posiciones", uint64(s.MaximoPosiciones))
	canon.entero("cantidad_disponible", uint64(s.CantidadDisponible))
	return huellaBytesBolsa(canon.bytes())
}

// Validar aplica el unico canon semantico de la solicitud durable. Los
// adaptadores deben invocarlo antes de cruzar su frontera de persistencia.
func (s SolicitudReservaEjecucionSeleccionLlamamiento) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) || !enteroSeguroBolsa(s.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(s.CorrelacionRef) ||
		!domain.ReferenciaOpacaValida(s.AutoridadSolicitante) ||
		s.AutorizacionConsulta.Validar() != nil || s.AccionConsulta.Validar() != nil ||
		s.RecursoConsulta.Validar() != nil || s.RecursoConsulta != s.Necesidad ||
		s.AccionOrden.Validar() != nil ||
		s.Finalidad.Validar() != nil || s.Necesidad.Validar() != nil ||
		s.Bolsa.Validar() != nil || s.Politica.Validar() != nil ||
		s.MaximoPosiciones == 0 || s.MaximoPosiciones > MaximoElementosIntegracionBolsa ||
		s.CantidadDisponible == 0 || s.CantidadDisponible > s.MaximoPosiciones ||
		!huellasBolsaIguales(s.HuellaSemantica, s.huellaEsperada()) {
		return ErrEjecucionSeleccionLlamamientoInvalida
	}
	return nil
}

type ReservaEjecucionSeleccionLlamamiento struct {
	Solicitud  SolicitudReservaEjecucionSeleccionLlamamiento
	ReservaRef string
}

type EstadoEjecucionSeleccionLlamamiento struct {
	Solicitud           SolicitudReservaEjecucionSeleccionLlamamiento
	Situacion           SituacionEjecucionSeleccionLlamamiento
	ReservaRef          string
	EfectoPosible       EfectoSeleccionLlamamiento
	ReciboConfirmado    ReciboSolicitudLlamamientoBolsa
	ArtefactoConfirmado ArtefactoProbatorioLlamamientoBolsa
}

// VerificarTerminalConfirmado comprueba forma, ligadura y autenticidad del
// terminal durable. La reautenticación es local: no consulta disponibilidad
// ni permite repetir los efectos que originaron el recibo.
func (e EstadoEjecucionSeleccionLlamamiento) VerificarTerminalConfirmado(
	ctx context.Context,
	consulta ConsultaTerminalAutorizada,
	verificador *VerificadorEvidenciaIntegracionBolsa,
	instante time.Time,
) (ReciboSolicitudLlamamientoBolsa, error) {
	vacio := ReciboSolicitudLlamamientoBolsa{}
	a := e.ArtefactoConfirmado
	if ctx == nil || verificador == nil || !instanteBolsaCanonico(instante) ||
		e.Situacion != EjecucionSeleccionLlamamientoConfirmada || e.ReservaRef != "" ||
		e.EfectoPosible != "" ||
		consulta.ValidarSolicitudEn(e.Solicitud, instante) != nil ||
		e.ReciboConfirmado == vacio || !e.ReciboConfirmado.PropuestaGenerada ||
		a.Validar() != nil || a.Recibo != e.ReciboConfirmado {
		return vacio, ErrEjecucionSeleccionLlamamientoInvalida
	}
	datos := a.Comando.Contexto.Datos
	if datos.OrganizacionRef != e.Solicitud.OrganizacionRef ||
		datos.ExpedienteRef != e.Solicitud.ExpedienteRef ||
		datos.VersionExpediente != e.Solicitud.VersionExpediente ||
		datos.CorrelacionRef != e.Solicitud.CorrelacionRef ||
		datos.Finalidad != e.Solicitud.Finalidad ||
		a.Comando.Necesidad != e.Solicitud.Necesidad ||
		a.Comando.Bolsa != e.Solicitud.Bolsa || a.Comando.Politica != e.Solicitud.Politica ||
		a.Comando.TotalPosicionesOrden < e.Solicitud.CantidadDisponible ||
		a.Comando.TotalPosicionesOrden > e.Solicitud.MaximoPosiciones ||
		a.Comando.MaximaPosicionEvaluable != a.Comando.TotalPosicionesOrden {
		return vacio, ErrEjecucionSeleccionLlamamientoInvalida
	}
	contexto := contextoDesdeRegistroVerificado(a.Comando.Contexto)
	comando := ComandoSolicitarLlamamientoBolsa{datos: &datosComandoSolicitarLlamamientoBolsa{
		contexto: contexto, necesidad: a.Comando.Necesidad, bolsa: a.Comando.Bolsa,
		orden: a.Comando.Orden, politica: a.Comando.Politica,
		totalPosicionesOrden:    a.Comando.TotalPosicionesOrden,
		maximaPosicionEvaluable: a.Comando.MaximaPosicionEvaluable,
		huellaReciboOrden:       a.Comando.HuellaReciboOrden,
	}}
	if _, err := verificador.reautenticarReciboLlamamiento(
		ctx, comando, e.ReciboConfirmado, a.Evidencia, instante,
	); err != nil {
		return vacio, ErrEjecucionSeleccionLlamamientoInvalida
	}
	return e.ReciboConfirmado, nil
}

// EjecucionesSeleccionLlamamiento es el límite atómico de idempotencia. Una
// implementación durable debe comparar UUID y huella en la misma operación de
// reserva, mantener una reserva viva como ocupada y convertir una reserva
// abandonada después de AbrirVentanaEfecto en indeterminada, nunca liberarla
// para repetir a ciegas. LiberarAntesDeEfectos solo admite una ejecución que
// todavía no cruzó ninguna ventana. ResolverTerminal exige una capacidad
// nominal fresca y solo devuelve terminales dentro de su autoridad exacta;
// Confirmar conserva atómicamente recibo y prueba.
// Este contrato no aporta persistencia: la composición real requiere un
// adaptador durable y recuperación explícita de estados indeterminados.
type EjecucionesSeleccionLlamamiento interface {
	ResolverTerminal(
		context.Context,
		ConsultaTerminalAutorizada,
		time.Time,
	) (EstadoEjecucionSeleccionLlamamiento, bool, error)
	Reservar(
		context.Context,
		SolicitudReservaEjecucionSeleccionLlamamiento,
	) (EstadoEjecucionSeleccionLlamamiento, error)
	AbrirVentanaEfecto(
		context.Context,
		ReservaEjecucionSeleccionLlamamiento,
		EfectoSeleccionLlamamiento,
	) error
	MarcarIndeterminada(
		context.Context,
		ReservaEjecucionSeleccionLlamamiento,
		EfectoSeleccionLlamamiento,
	) error
	LiberarAntesDeEfectos(
		context.Context,
		ReservaEjecucionSeleccionLlamamiento,
	) error
	Confirmar(
		context.Context,
		ReservaEjecucionSeleccionLlamamiento,
		ReciboSolicitudLlamamientoBolsa,
		ArtefactoProbatorioLlamamientoBolsa,
	) error
	ConsultarEstado(
		context.Context,
		SolicitudReservaEjecucionSeleccionLlamamiento,
	) (EstadoEjecucionSeleccionLlamamiento, error)
}

// PreparadorSeleccionLlamamiento resuelve desde una referencia de intención
// no autoritativa los contextos, la política y los límites gobernados de cada
// paso. No selecciona una posición ni ejecuta efectos en Bolsa.
type PreparadorSeleccionLlamamiento interface {
	PrepararConsultaDisponibilidad(
		context.Context,
		string,
	) (SolicitudDisponibilidadBolsa, error)
	PrepararOrdenCompleto(
		context.Context,
		string,
		ResultadoDisponibilidadBolsa,
	) (ComandoPrepararOrdenBolsa, error)
	PrepararContextoLlamamiento(
		context.Context,
		string,
		ReciboOrdenBolsa,
	) (ContextoPeticionIntegracionBolsa, error)
}
