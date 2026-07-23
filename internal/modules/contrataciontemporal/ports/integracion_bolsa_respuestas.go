package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// SolicitudDisponibilidadBolsa es una consulta volátil y acotada. Su resultado
// sirve para decidir el siguiente paso mientras esté fresco, pero no es un
// recibo durable ni acredita por sí solo una actuación administrativa.
type SolicitudDisponibilidadBolsa struct {
	Contexto         ContextoPeticionIntegracionBolsa     `json:"contexto"`
	Necesidad        ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	CategoriaRef     string                               `json:"categoria_ref"`
	MaximoResultados uint32                               `json:"maximo_resultados"`
}

func (s SolicitudDisponibilidadBolsa) ValidarEn(instante time.Time) error {
	contexto, err := s.Contexto.DatosEn(instante)
	if err != nil || s.Necesidad.Validar() != nil ||
		contexto.Recurso != s.Necesidad ||
		!domain.ReferenciaOpacaValida(s.CategoriaRef) || s.MaximoResultados == 0 {
		return ErrPeticionIntegracionBolsaInvalida
	}
	if s.MaximoResultados > MaximoElementosIntegracionBolsa {
		return ErrLimiteIntegracionBolsaExcedido
	}
	return nil
}

type ResultadoDisponibilidadBolsa struct {
	OperacionRef       string                               `json:"operacion_ref"`
	OrganizacionRef    string                               `json:"organizacion_ref"`
	ExpedienteRef      string                               `json:"expediente_ref"`
	VersionExpediente  uint64                               `json:"version_expediente"`
	CorrelacionRef     string                               `json:"correlacion_ref"`
	Necesidad          ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	CategoriaRef       string                               `json:"categoria_ref"`
	Resultado          ReferenciaVersionadaIntegracionBolsa `json:"resultado"`
	BolsaEncontrada    bool                                 `json:"bolsa_encontrada"`
	Bolsa              ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Disponible         bool                                 `json:"disponible"`
	CantidadDisponible uint32                               `json:"cantidad_disponible"`
	CantidadExacta     bool                                 `json:"cantidad_exacta"`
	Procedencia        ProcedenciaIntegracionBolsa          `json:"procedencia"`
}

func (r ResultadoDisponibilidadBolsa) ValidarParaEn(
	solicitud SolicitudDisponibilidadBolsa,
	instante time.Time,
) error {
	if solicitud.ValidarEn(instante) != nil ||
		!respuestaComunBolsaValidaEn(
			r.OperacionRef, r.OrganizacionRef, r.ExpedienteRef, r.VersionExpediente,
			r.CorrelacionRef, r.Necesidad, r.Resultado, r.Procedencia,
			solicitud.Contexto, solicitud.Necesidad, instante,
		) || r.CategoriaRef != solicitud.CategoriaRef ||
		r.CantidadDisponible > solicitud.MaximoResultados {
		return ErrRespuestaBolsaNoConfiable
	}
	if !r.BolsaEncontrada {
		if r.Bolsa != (ReferenciaVersionadaIntegracionBolsa{}) ||
			r.Disponible || r.CantidadDisponible != 0 || !r.CantidadExacta {
			return ErrRespuestaBolsaNoConfiable
		}
		return nil
	}
	if r.Bolsa.Validar() != nil || r.Disponible != (r.CantidadDisponible > 0) ||
		(!r.Disponible && !r.CantidadExacta) {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

type ConsultaDisponibilidadBolsa interface {
	ConsultarDisponibilidad(
		context.Context,
		SolicitudDisponibilidadBolsa,
	) (ResultadoDisponibilidadBolsa, error)
}

// ComandoPrepararOrdenBolsa solicita a Bolsa una fotografía durable y
// completa del orden. Las posiciones y participantes no cruzan la frontera.
type ComandoPrepararOrdenBolsa struct {
	Contexto         ContextoPeticionIntegracionBolsa     `json:"contexto"`
	Necesidad        ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa            ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Politica         ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	MaximoPosiciones uint32                               `json:"maximo_posiciones"`
}

func (c ComandoPrepararOrdenBolsa) ValidarEn(instante time.Time) error {
	contexto, err := c.Contexto.DatosEn(instante)
	if err != nil || c.Necesidad.Validar() != nil ||
		c.Bolsa.Validar() != nil || c.Politica.Validar() != nil ||
		contexto.Recurso != c.Bolsa ||
		c.MaximoPosiciones == 0 {
		return ErrPeticionIntegracionBolsaInvalida
	}
	if c.MaximoPosiciones > MaximoElementosIntegracionBolsa {
		return ErrLimiteIntegracionBolsaExcedido
	}
	return nil
}

// ReciboOrdenBolsa es un recibo durable nominal. El comprobante opaco autoriza
// su persistencia inicial; se conserva el recibo, su material canónico y la
// evidencia HMAC, nunca se serializa el comprobante efímero.
type ReciboOrdenBolsa struct {
	OperacionRef      string                               `json:"operacion_ref"`
	OrganizacionRef   string                               `json:"organizacion_ref"`
	ExpedienteRef     string                               `json:"expediente_ref"`
	VersionExpediente uint64                               `json:"version_expediente"`
	CorrelacionRef    string                               `json:"correlacion_ref"`
	Necesidad         ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa             ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Politica          ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	Resultado         ReferenciaVersionadaIntegracionBolsa `json:"resultado"`
	OrdenGenerada     bool                                 `json:"orden_generada"`
	OrdenCompleta     bool                                 `json:"orden_completa"`
	Orden             ReferenciaVersionadaIntegracionBolsa `json:"orden"`
	AccionLlamamiento ReferenciaVersionadaIntegracionBolsa `json:"accion_llamamiento"`
	TotalPosiciones   uint32                               `json:"total_posiciones"`
	ReciboRef         string                               `json:"recibo_ref"`
	AuditoriaRef      string                               `json:"auditoria_ref"`
	EventoRef         string                               `json:"evento_ref"`
	ConfirmadaEn      time.Time                            `json:"confirmada_en"`
	Procedencia       ProcedenciaIntegracionBolsa          `json:"procedencia"`
}

func (r ReciboOrdenBolsa) ValidarParaEn(
	comando ComandoPrepararOrdenBolsa,
	instante time.Time,
) error {
	if comando.ValidarEn(instante) != nil ||
		!r.Procedencia.validarNominalEn(instante) ||
		r.ValidarDurablePara(comando) != nil {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

// ValidarDurablePara revalida el recibo conservado sin exigir que su ventana
// de transporte siga abierta. No autentica el HMAC: el almacenamiento solo
// puede llamarlo sobre un recibo promovido previamente por el verificador TCB.
func (r ReciboOrdenBolsa) ValidarDurablePara(
	comando ComandoPrepararOrdenBolsa,
) error {
	contexto, err := comando.Contexto.datosDurables()
	if err != nil || comando.ValidarEn(contexto.SolicitadaEn) != nil ||
		!respuestaComunBolsaValida(
			r.OperacionRef, r.OrganizacionRef, r.ExpedienteRef, r.VersionExpediente,
			r.CorrelacionRef, r.Necesidad, r.Resultado, r.Procedencia,
			contexto, comando.Necesidad,
		) || r.Bolsa != comando.Bolsa || r.Politica != comando.Politica ||
		r.TotalPosiciones > comando.MaximoPosiciones ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!instanteBolsaCanonico(r.ConfirmadaEn) ||
		r.ConfirmadaEn.Before(contexto.SolicitadaEn) ||
		r.ConfirmadaEn.After(r.Procedencia.Evidencia.EmitidaEn) {
		return ErrRespuestaBolsaNoConfiable
	}
	if !r.OrdenGenerada {
		if r.OrdenCompleta || r.Orden != (ReferenciaVersionadaIntegracionBolsa{}) ||
			r.AccionLlamamiento != (ReferenciaVersionadaIntegracionBolsa{}) ||
			r.TotalPosiciones != 0 {
			return ErrRespuestaBolsaNoConfiable
		}
		return nil
	}
	if !r.OrdenCompleta || r.Orden.Validar() != nil ||
		r.AccionLlamamiento.Validar() != nil || r.TotalPosiciones == 0 {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

type PreparadorOrdenBolsa interface {
	PrepararOrden(context.Context, ComandoPrepararOrdenBolsa) (ReciboOrdenBolsa, error)
}

// PreparacionComandoSolicitarLlamamientoBolsa exige el recibo y comprobante
// de la orden, además del nuevo contexto de operación. El total nunca procede
// de un formulario ni de un DTO libre.
type PreparacionComandoSolicitarLlamamientoBolsa struct {
	Contexto                ContextoPeticionIntegracionBolsa
	ComandoOrden            ComandoPrepararOrdenBolsa
	ReciboOrden             ReciboOrdenBolsa
	ComprobanteOrden        ComprobanteEvidenciaIntegracionBolsa
	MaximaPosicionEvaluable uint32
}

type ComandoSolicitarLlamamientoBolsa struct {
	datos *datosComandoSolicitarLlamamientoBolsa
}

func (ComandoSolicitarLlamamientoBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*ComandoSolicitarLlamamientoBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

type datosComandoSolicitarLlamamientoBolsa struct {
	contexto                ContextoPeticionIntegracionBolsa
	necesidad               ReferenciaVersionadaIntegracionBolsa
	bolsa                   ReferenciaVersionadaIntegracionBolsa
	orden                   ReferenciaVersionadaIntegracionBolsa
	politica                ReferenciaVersionadaIntegracionBolsa
	totalPosicionesOrden    uint32
	maximaPosicionEvaluable uint32
	huellaReciboOrden       string
}

type DatosComandoSolicitarLlamamientoBolsa struct {
	Contexto                ContextoPeticionIntegracionBolsa
	Necesidad               ReferenciaVersionadaIntegracionBolsa
	Bolsa                   ReferenciaVersionadaIntegracionBolsa
	Orden                   ReferenciaVersionadaIntegracionBolsa
	Politica                ReferenciaVersionadaIntegracionBolsa
	TotalPosicionesOrden    uint32
	MaximaPosicionEvaluable uint32
	HuellaReciboOrden       string
}

func NuevoComandoSolicitarLlamamientoBolsa(
	preparacion PreparacionComandoSolicitarLlamamientoBolsa,
	instante time.Time,
) (ComandoSolicitarLlamamientoBolsa, error) {
	contextoNuevo, errNuevo := preparacion.Contexto.DatosEn(instante)
	contextoOrden, errOrden := preparacion.ComandoOrden.Contexto.datosDurables()
	if errNuevo != nil || errOrden != nil ||
		preparacion.ReciboOrden.ValidarDurablePara(preparacion.ComandoOrden) != nil ||
		!preparacion.ReciboOrden.OrdenGenerada ||
		preparacion.ReciboOrden.TotalPosiciones == 0 ||
		preparacion.MaximaPosicionEvaluable == 0 ||
		preparacion.MaximaPosicionEvaluable > preparacion.ReciboOrden.TotalPosiciones ||
		contextoNuevo.OrganizacionRef != contextoOrden.OrganizacionRef ||
		contextoNuevo.ExpedienteRef != contextoOrden.ExpedienteRef ||
		contextoNuevo.VersionExpediente != contextoOrden.VersionExpediente ||
		contextoNuevo.CorrelacionRef != contextoOrden.CorrelacionRef ||
		contextoNuevo.Finalidad != contextoOrden.Finalidad ||
		contextoNuevo.Accion != preparacion.ReciboOrden.AccionLlamamiento ||
		contextoNuevo.Recurso != preparacion.ReciboOrden.Orden {
		return ComandoSolicitarLlamamientoBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	if !preparacion.ComprobanteOrden.instanteVerificacion().Equal(instante) ||
		!instante.Before(preparacion.ReciboOrden.Procedencia.Evidencia.RetenerHasta) {
		return ComandoSolicitarLlamamientoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	material := materialReciboOrdenBolsa(preparacion.ComandoOrden, preparacion.ReciboOrden)
	evidencia := nuevaEvidenciaDurableBolsa(
		"recibo_orden",
		contextoOrden.OperacionRef,
		materialComandoOrdenBolsa(preparacion.ComandoOrden),
		material,
		preparacion.ReciboOrden.Procedencia,
	)
	if !preparacion.ComprobanteOrden.coincide(evidencia) {
		return ComandoSolicitarLlamamientoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return ComandoSolicitarLlamamientoBolsa{
		datos: &datosComandoSolicitarLlamamientoBolsa{
			contexto: preparacion.Contexto, necesidad: preparacion.ReciboOrden.Necesidad,
			bolsa: preparacion.ReciboOrden.Bolsa, orden: preparacion.ReciboOrden.Orden,
			politica:                preparacion.ReciboOrden.Politica,
			totalPosicionesOrden:    preparacion.ReciboOrden.TotalPosiciones,
			maximaPosicionEvaluable: preparacion.MaximaPosicionEvaluable,
			huellaReciboOrden:       huellaBytesBolsa(material),
		},
	}, nil
}

func (c ComandoSolicitarLlamamientoBolsa) DatosEn(
	instante time.Time,
) (DatosComandoSolicitarLlamamientoBolsa, error) {
	if c.datos == nil {
		return DatosComandoSolicitarLlamamientoBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	datos := DatosComandoSolicitarLlamamientoBolsa{
		Contexto: c.datos.contexto, Necesidad: c.datos.necesidad, Bolsa: c.datos.bolsa,
		Orden: c.datos.orden, Politica: c.datos.politica,
		TotalPosicionesOrden:    c.datos.totalPosicionesOrden,
		MaximaPosicionEvaluable: c.datos.maximaPosicionEvaluable,
		HuellaReciboOrden:       c.datos.huellaReciboOrden,
	}
	contexto, err := datos.Contexto.DatosEn(instante)
	if err != nil || datos.Necesidad.Validar() != nil ||
		datos.Bolsa.Validar() != nil || datos.Orden.Validar() != nil ||
		datos.Politica.Validar() != nil || datos.TotalPosicionesOrden == 0 ||
		datos.TotalPosicionesOrden > MaximoElementosIntegracionBolsa ||
		datos.MaximaPosicionEvaluable == 0 ||
		datos.MaximaPosicionEvaluable > datos.TotalPosicionesOrden ||
		!huellaSHA256Valida(datos.HuellaReciboOrden) ||
		contexto.Recurso != datos.Orden {
		return DatosComandoSolicitarLlamamientoBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	return datos, nil
}

func (c ComandoSolicitarLlamamientoBolsa) datosCanonicos() (
	DatosComandoSolicitarLlamamientoBolsa,
	error,
) {
	if c.datos == nil {
		return DatosComandoSolicitarLlamamientoBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	contexto, err := c.datos.contexto.datosDurables()
	if err != nil {
		return DatosComandoSolicitarLlamamientoBolsa{}, err
	}
	return c.DatosEn(contexto.SolicitadaEn)
}

// ReciboSolicitudLlamamientoBolsa no expone identidad ni contacto. SeleccionRef
// es, aun siendo opaca, dato personal seudonimizado: se minimiza a esta entrega,
// no se registra en logs/telemetría y se conserva según RetencionSeleccion.
type ReciboSolicitudLlamamientoBolsa struct {
	OperacionRef       string                               `json:"operacion_ref"`
	OrganizacionRef    string                               `json:"organizacion_ref"`
	ExpedienteRef      string                               `json:"expediente_ref"`
	VersionExpediente  uint64                               `json:"version_expediente"`
	CorrelacionRef     string                               `json:"correlacion_ref"`
	Necesidad          ReferenciaVersionadaIntegracionBolsa `json:"necesidad"`
	Bolsa              ReferenciaVersionadaIntegracionBolsa `json:"bolsa"`
	Orden              ReferenciaVersionadaIntegracionBolsa `json:"orden"`
	Politica           ReferenciaVersionadaIntegracionBolsa `json:"politica"`
	Resultado          ReferenciaVersionadaIntegracionBolsa `json:"resultado"`
	PropuestaGenerada  bool                                 `json:"propuesta_generada"`
	Propuesta          ReferenciaVersionadaIntegracionBolsa `json:"propuesta"`
	AccionEvento       ReferenciaVersionadaIntegracionBolsa `json:"accion_evento"`
	LlamamientoRef     string                               `json:"llamamiento_ref"`
	SeleccionRef       string                               `json:"seleccion_ref"`
	RetencionSeleccion ReferenciaVersionadaIntegracionBolsa `json:"retencion_seleccion"`
	OrdenSeleccionado  uint32                               `json:"orden_seleccionado"`
	ReciboRef          string                               `json:"recibo_ref"`
	AuditoriaRef       string                               `json:"auditoria_ref"`
	EventoRef          string                               `json:"evento_ref"`
	ConfirmadaEn       time.Time                            `json:"confirmada_en"`
	Procedencia        ProcedenciaIntegracionBolsa          `json:"procedencia"`
}

func (r ReciboSolicitudLlamamientoBolsa) ValidarParaEn(
	comando ComandoSolicitarLlamamientoBolsa,
	instante time.Time,
) error {
	datosComando, err := comando.DatosEn(instante)
	if err != nil || !r.Procedencia.validarNominalEn(instante) ||
		r.validarDurableParaDatos(datosComando) != nil {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

func (r ReciboSolicitudLlamamientoBolsa) validarDurableParaDatos(
	datosComando DatosComandoSolicitarLlamamientoBolsa,
) error {
	contexto, err := datosComando.Contexto.datosDurables()
	_, errFresco := datosComando.Contexto.DatosEn(contexto.SolicitadaEn)
	if err != nil || errFresco != nil ||
		!respuestaComunBolsaValida(
			r.OperacionRef, r.OrganizacionRef, r.ExpedienteRef, r.VersionExpediente,
			r.CorrelacionRef, r.Necesidad, r.Resultado, r.Procedencia,
			contexto, datosComando.Necesidad,
		) || r.Bolsa != datosComando.Bolsa || r.Orden != datosComando.Orden ||
		r.Politica != datosComando.Politica ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!instanteBolsaCanonico(r.ConfirmadaEn) ||
		r.ConfirmadaEn.Before(contexto.SolicitadaEn) ||
		r.ConfirmadaEn.After(r.Procedencia.Evidencia.EmitidaEn) {
		return ErrRespuestaBolsaNoConfiable
	}
	if !r.PropuestaGenerada {
		if r.Propuesta != (ReferenciaVersionadaIntegracionBolsa{}) ||
			r.AccionEvento != (ReferenciaVersionadaIntegracionBolsa{}) ||
			r.LlamamientoRef != "" || r.SeleccionRef != "" ||
			r.RetencionSeleccion != (ReferenciaVersionadaIntegracionBolsa{}) ||
			r.OrdenSeleccionado != 0 {
			return ErrRespuestaBolsaNoConfiable
		}
		return nil
	}
	if r.Propuesta.Validar() != nil || r.AccionEvento.Validar() != nil ||
		!domain.ReferenciaOpacaValida(r.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(r.SeleccionRef) ||
		r.RetencionSeleccion.Validar() != nil ||
		r.OrdenSeleccionado == 0 ||
		r.OrdenSeleccionado > datosComando.TotalPosicionesOrden ||
		r.OrdenSeleccionado > datosComando.MaximaPosicionEvaluable {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}

// ValidarDurablePara revalida un recibo ya autenticado y conservado; no
// convierte por sí sola una respuesta nominal en evidencia confiable.
func (r ReciboSolicitudLlamamientoBolsa) ValidarDurablePara(
	comando ComandoSolicitarLlamamientoBolsa,
) error {
	datosComando, err := comando.datosCanonicos()
	if err != nil {
		return ErrRespuestaBolsaNoConfiable
	}
	return r.validarDurableParaDatos(datosComando)
}

type GestorLlamamientosBolsa interface {
	SolicitarLlamamiento(
		context.Context,
		ComandoSolicitarLlamamientoBolsa,
	) (ReciboSolicitudLlamamientoBolsa, error)
}

func respuestaComunBolsaValidaEn(
	operacionRef string,
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
	correlacionRef string,
	necesidad ReferenciaVersionadaIntegracionBolsa,
	resultado ReferenciaVersionadaIntegracionBolsa,
	procedencia ProcedenciaIntegracionBolsa,
	contexto ContextoPeticionIntegracionBolsa,
	necesidadEsperada ReferenciaVersionadaIntegracionBolsa,
	instante time.Time,
) bool {
	datosContexto, err := contexto.DatosEn(instante)
	return err == nil &&
		procedencia.validarNominalEn(instante) &&
		respuestaComunBolsaValida(
			operacionRef, organizacionRef, expedienteRef, versionExpediente,
			correlacionRef, necesidad, resultado, procedencia, datosContexto, necesidadEsperada,
		)
}

func respuestaComunBolsaValida(
	operacionRef string,
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
	correlacionRef string,
	necesidad ReferenciaVersionadaIntegracionBolsa,
	resultado ReferenciaVersionadaIntegracionBolsa,
	procedencia ProcedenciaIntegracionBolsa,
	contexto DatosContextoPeticionIntegracionBolsa,
	necesidadEsperada ReferenciaVersionadaIntegracionBolsa,
) bool {
	return operacionRef == contexto.OperacionRef &&
		organizacionRef == contexto.OrganizacionRef &&
		expedienteRef == contexto.ExpedienteRef &&
		versionExpediente == contexto.VersionExpediente &&
		correlacionRef == contexto.CorrelacionRef &&
		necesidad == necesidadEsperada && resultado.Validar() == nil &&
		procedencia.validarNominal() &&
		!procedencia.Evidencia.EmitidaEn.Before(contexto.SolicitadaEn) &&
		procedencia.Evidencia.EmitidaEn.Before(contexto.ValidaHasta) &&
		!procedencia.Evidencia.ValidaHasta.After(contexto.ValidaHasta)
}
