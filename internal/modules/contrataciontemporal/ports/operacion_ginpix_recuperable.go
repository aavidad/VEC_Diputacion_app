package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrSolicitudOperacionGINPIXInvalida = errors.New(
		"contratacion temporal: solicitud de operacion ginpix invalida",
	)
	ErrReservaOperacionGINPIXInvalida = errors.New(
		"contratacion temporal: reserva de operacion ginpix invalida",
	)
	ErrColisionOperacionGINPIX = errors.New(
		"contratacion temporal: colision de operacion ginpix",
	)
	ErrOperacionGINPIXNoReservada = errors.New(
		"contratacion temporal: operacion ginpix no reservada",
	)
	ErrEmisionOperacionGINPIXNoIniciada = errors.New(
		"contratacion temporal: emision ginpix no iniciada",
	)
	ErrEmisionOperacionGINPIXIndeterminada = errors.New(
		"contratacion temporal: emision ginpix indeterminada",
	)
	ErrConsultaOperacionGINPIXNoDisponible = errors.New(
		"contratacion temporal: consulta ginpix no disponible",
	)
	ErrReciboExternoOperacionGINPIXInvalido = errors.New(
		"contratacion temporal: recibo externo ginpix invalido",
	)
	ErrResultadoOperacionGINPIXInvalido = errors.New(
		"contratacion temporal: resultado de operacion ginpix invalido",
	)
)

// DatosOperacionGINPIX contiene solo ligaduras opacas, versiones y huellas.
// La clave cubre la orden y el recibo O7-02, el modelo, el mapeo, la carga y
// la idempotencia; no concede por si sola permiso para emitir.
type DatosOperacionGINPIX struct {
	ClaveOperacionRef      string
	OrdenHuellaSHA256      string
	VersionExpediente      uint64
	ExpedienteRef          string
	IncorporacionRef       string
	ReciboIncorporacionRef string
	ResultadoPersonalRef   string
	ReciboPersonalRef      string
	CorrelacionRef         string
	IdempotenciaRef        string
	ProcedenciaModeloRef   string
	ModeloHuellaSHA256     string
	MapeoRef               string
	MapeoVersion           uint64
	ProcedenciaMapeoRef    string
	MapeoHuellaSHA256      string
	CargaHuellaSHA256      string
}

type datosSolicitudOperacionGINPIX struct {
	orden         OrdenConfirmarIncorporacion
	incorporacion ReciboConfirmacionIncorporacion
	mapeo         SolicitudMapeoGINPIX
	carga         domain.CargaMapeadaGINPIX
	datos         DatosOperacionGINPIX
}

// SolicitudOperacionGINPIX es un sobre inmutable y neutral. Prepararlo no
// reserva, no emite y no consulta ningun sistema externo.
type SolicitudOperacionGINPIX struct {
	datos *datosSolicitudOperacionGINPIX
}

func NuevaSolicitudOperacionGINPIX(
	mapeo SolicitudMapeoGINPIX,
	orden OrdenConfirmarIncorporacion,
	incorporacion ReciboConfirmacionIncorporacion,
) (SolicitudOperacionGINPIX, error) {
	evidencia, errOrden := orden.Datos()
	modelo, errModelo := mapeo.Modelo()
	mapa, errMapeo := mapeo.Mapeo()
	if errOrden != nil || incorporacion.ValidarPara(orden) != nil ||
		errModelo != nil || errMapeo != nil {
		return SolicitudOperacionGINPIX{}, ErrSolicitudOperacionGINPIXInvalida
	}
	carga, err := domain.AplicarMapeoGINPIX(modelo, mapa)
	if err != nil || carga.Validar() != nil {
		return SolicitudOperacionGINPIX{}, ErrSolicitudOperacionGINPIXInvalida
	}
	datosCarga := carga.Datos()
	if datosCarga.VersionExpediente != incorporacion.VersionExpediente ||
		datosCarga.ExpedienteRef != incorporacion.ExpedienteRef ||
		datosCarga.IncorporacionRef != incorporacion.ActuacionRef {
		return SolicitudOperacionGINPIX{}, ErrSolicitudOperacionGINPIXInvalida
	}
	huellaOrden := huellaOrdenOperacionGINPIX(evidencia, incorporacion)
	datos := DatosOperacionGINPIX{
		OrdenHuellaSHA256:      huellaOrden,
		VersionExpediente:      datosCarga.VersionExpediente,
		ExpedienteRef:          datosCarga.ExpedienteRef,
		IncorporacionRef:       datosCarga.IncorporacionRef,
		ReciboIncorporacionRef: incorporacion.ReciboRef,
		ResultadoPersonalRef:   incorporacion.ResultadoPersonalRef,
		ReciboPersonalRef:      incorporacion.ReciboPersonalRef,
		CorrelacionRef:         datosCarga.CorrelacionRef,
		IdempotenciaRef:        datosCarga.IdempotenciaRef,
		ProcedenciaModeloRef:   datosCarga.ProcedenciaModeloRef,
		ModeloHuellaSHA256:     datosCarga.ModeloHuellaSHA256,
		MapeoRef:               datosCarga.MapeoRef,
		MapeoVersion:           datosCarga.MapeoVersion,
		ProcedenciaMapeoRef:    datosCarga.ProcedenciaMapeoRef,
		MapeoHuellaSHA256:      datosCarga.MapeoHuellaSHA256,
		CargaHuellaSHA256:      datosCarga.HuellaSHA256,
	}
	datos.ClaveOperacionRef = referenciaOperacionGINPIX(datos)
	incorporacion = clonarReciboOperacionGINPIX(incorporacion)
	solicitud := SolicitudOperacionGINPIX{datos: &datosSolicitudOperacionGINPIX{
		orden: orden, incorporacion: incorporacion, mapeo: mapeo, carga: carga, datos: datos,
	}}
	if solicitud.Validar() != nil {
		return SolicitudOperacionGINPIX{}, ErrSolicitudOperacionGINPIXInvalida
	}
	return solicitud, nil
}

func (s SolicitudOperacionGINPIX) Validar() error {
	if s.datos == nil || s.datos.incorporacion.ValidarPara(s.datos.orden) != nil ||
		s.datos.mapeo.Validar() != nil || s.datos.carga.Validar() != nil ||
		!datosOperacionGINPIXValidos(s.datos.datos) {
		return ErrSolicitudOperacionGINPIXInvalida
	}
	evidencia, err := s.datos.orden.Datos()
	if err != nil || s.datos.datos.OrdenHuellaSHA256 !=
		huellaOrdenOperacionGINPIX(evidencia, s.datos.incorporacion) ||
		s.datos.datos.ClaveOperacionRef != referenciaOperacionGINPIX(s.datos.datos) {
		return ErrSolicitudOperacionGINPIXInvalida
	}
	datosCarga := s.datos.carga.Datos()
	if s.datos.datos.VersionExpediente != datosCarga.VersionExpediente ||
		s.datos.datos.ExpedienteRef != datosCarga.ExpedienteRef ||
		s.datos.datos.IncorporacionRef != datosCarga.IncorporacionRef ||
		s.datos.datos.ReciboIncorporacionRef != s.datos.incorporacion.ReciboRef ||
		s.datos.datos.ResultadoPersonalRef != s.datos.incorporacion.ResultadoPersonalRef ||
		s.datos.datos.ReciboPersonalRef != s.datos.incorporacion.ReciboPersonalRef ||
		s.datos.datos.CorrelacionRef != datosCarga.CorrelacionRef ||
		s.datos.datos.IdempotenciaRef != datosCarga.IdempotenciaRef ||
		s.datos.datos.ProcedenciaModeloRef != datosCarga.ProcedenciaModeloRef ||
		s.datos.datos.ModeloHuellaSHA256 != datosCarga.ModeloHuellaSHA256 ||
		s.datos.datos.MapeoRef != datosCarga.MapeoRef ||
		s.datos.datos.MapeoVersion != datosCarga.MapeoVersion ||
		s.datos.datos.ProcedenciaMapeoRef != datosCarga.ProcedenciaMapeoRef ||
		s.datos.datos.MapeoHuellaSHA256 != datosCarga.MapeoHuellaSHA256 ||
		s.datos.datos.CargaHuellaSHA256 != datosCarga.HuellaSHA256 {
		return ErrSolicitudOperacionGINPIXInvalida
	}
	return nil
}

func (s SolicitudOperacionGINPIX) Datos() (DatosOperacionGINPIX, error) {
	if s.Validar() != nil {
		return DatosOperacionGINPIX{}, ErrSolicitudOperacionGINPIXInvalida
	}
	return s.datos.datos, nil
}

func (s SolicitudOperacionGINPIX) Orden() (OrdenConfirmarIncorporacion, error) {
	if s.Validar() != nil {
		return OrdenConfirmarIncorporacion{}, ErrSolicitudOperacionGINPIXInvalida
	}
	return s.datos.orden, nil
}

func (s SolicitudOperacionGINPIX) ReciboIncorporacion() (
	ReciboConfirmacionIncorporacion,
	error,
) {
	if s.Validar() != nil {
		return ReciboConfirmacionIncorporacion{}, ErrSolicitudOperacionGINPIXInvalida
	}
	return clonarReciboOperacionGINPIX(s.datos.incorporacion), nil
}

func (s SolicitudOperacionGINPIX) Carga() (domain.CargaMapeadaGINPIX, error) {
	if s.Validar() != nil {
		return domain.CargaMapeadaGINPIX{}, ErrSolicitudOperacionGINPIXInvalida
	}
	return s.datos.carga, nil
}

type SituacionReservaOperacionGINPIX uint8

const (
	ReservaOperacionGINPIXEmisionAutorizada SituacionReservaOperacionGINPIX = iota + 1
	ReservaOperacionGINPIXPendienteConciliacion
	ReservaOperacionGINPIXConfirmada
)

// ReservaOperacionGINPIX es la decision durable del registro local. Solo el
// estado EmisionAutorizada permite una unica llamada al emisor.
type ReservaOperacionGINPIX struct {
	ReservaRef        string
	ClaveOperacionRef string
	Intento           uint64
	Situacion         SituacionReservaOperacionGINPIX
	Resultado         ResultadoOperacionGINPIX
}

func (r ReservaOperacionGINPIX) ValidarPara(s SolicitudOperacionGINPIX) error {
	datos, err := s.Datos()
	if err != nil || !domain.ReferenciaOpacaValida(r.ReservaRef) || r.Intento == 0 ||
		r.ClaveOperacionRef != datos.ClaveOperacionRef {
		return ErrReservaOperacionGINPIXInvalida
	}
	switch r.Situacion {
	case ReservaOperacionGINPIXEmisionAutorizada,
		ReservaOperacionGINPIXPendienteConciliacion:
		if r.Resultado != (ResultadoOperacionGINPIX{}) {
			return ErrReservaOperacionGINPIXInvalida
		}
	case ReservaOperacionGINPIXConfirmada:
		if r.Resultado.ValidarPara(s) != nil {
			return ErrReservaOperacionGINPIXInvalida
		}
	default:
		return ErrReservaOperacionGINPIXInvalida
	}
	return nil
}

// ReciboExternoOperacionGINPIX es neutral al transporte. Un adaptador solo
// puede construir exito si aporta todas las ligaduras y evidencia externa.
type ReciboExternoOperacionGINPIX struct {
	ReciboExternoRef             string
	EvidenciaExternaRef          string
	EvidenciaExternaHuellaSHA256 string
	ClaveOperacionRef            string
	VersionExpediente            uint64
	ExpedienteRef                string
	IncorporacionRef             string
	ReciboIncorporacionRef       string
	ResultadoPersonalRef         string
	ReciboPersonalRef            string
	CorrelacionRef               string
	IdempotenciaRef              string
	ModeloHuellaSHA256           string
	MapeoRef                     string
	MapeoVersion                 uint64
	MapeoHuellaSHA256            string
	CargaHuellaSHA256            string
}

func (r ReciboExternoOperacionGINPIX) ValidarPara(s SolicitudOperacionGINPIX) error {
	datos, err := s.Datos()
	if err != nil || !domain.ReferenciaOpacaValida(r.ReciboExternoRef) ||
		!domain.ReferenciaOpacaValida(r.EvidenciaExternaRef) ||
		!huellaOperacionGINPIXValida(r.EvidenciaExternaHuellaSHA256) ||
		r.ClaveOperacionRef != datos.ClaveOperacionRef ||
		r.VersionExpediente != datos.VersionExpediente ||
		r.ExpedienteRef != datos.ExpedienteRef || r.IncorporacionRef != datos.IncorporacionRef ||
		r.ReciboIncorporacionRef != datos.ReciboIncorporacionRef ||
		r.ResultadoPersonalRef != datos.ResultadoPersonalRef ||
		r.ReciboPersonalRef != datos.ReciboPersonalRef ||
		r.CorrelacionRef != datos.CorrelacionRef || r.IdempotenciaRef != datos.IdempotenciaRef ||
		r.ModeloHuellaSHA256 != datos.ModeloHuellaSHA256 || r.MapeoRef != datos.MapeoRef ||
		r.MapeoVersion != datos.MapeoVersion || r.MapeoHuellaSHA256 != datos.MapeoHuellaSHA256 ||
		r.CargaHuellaSHA256 != datos.CargaHuellaSHA256 {
		return ErrReciboExternoOperacionGINPIXInvalido
	}
	return nil
}

type ResultadoOperacionGINPIX struct {
	ConfirmacionLocalRef string
	ClaveOperacionRef    string
	ReciboExterno        ReciboExternoOperacionGINPIX
}

func (r ResultadoOperacionGINPIX) ValidarPara(s SolicitudOperacionGINPIX) error {
	datos, err := s.Datos()
	if err != nil || !domain.ReferenciaOpacaValida(r.ConfirmacionLocalRef) ||
		r.ClaveOperacionRef != datos.ClaveOperacionRef ||
		r.ReciboExterno.ValidarPara(s) != nil {
		return ErrResultadoOperacionGINPIXInvalido
	}
	return nil
}

// RegistroOperacionGINPIX posee la maquina durable local. Reservar debe
// detectar colisiones por idempotencia y conceder EmisionAutorizada a un solo
// llamador. Confirmar persiste recibo externo y resultado en una sola unidad
// atomica.
type RegistroOperacionGINPIX interface {
	ReservarOperacionGINPIX(context.Context, SolicitudOperacionGINPIX) (ReservaOperacionGINPIX, error)
	ConsultarReservaOperacionGINPIX(context.Context, SolicitudOperacionGINPIX) (ReservaOperacionGINPIX, error)
	RegistrarFalloPreemisionGINPIX(context.Context, ReservaOperacionGINPIX) error
	MarcarOperacionGINPIXIndeterminada(context.Context, ReservaOperacionGINPIX) error
	ConfirmarOperacionGINPIX(
		context.Context,
		ReservaOperacionGINPIX,
		ReciboExternoOperacionGINPIX,
	) (ResultadoOperacionGINPIX, error)
}

// EmisorOperacionGINPIX consume una autorizacion local una sola vez. Solo
// puede devolver ErrEmisionOperacionGINPIXNoIniciada cuando acredita que no
// alcanzo la frontera de efecto; desde esa frontera, la ausencia de un recibo
// completo se clasifica como ErrEmisionOperacionGINPIXIndeterminada.
type EmisorOperacionGINPIX interface {
	EmitirOperacionGINPIX(
		context.Context,
		SolicitudOperacionGINPIX,
		ReservaOperacionGINPIX,
	) (ReciboExternoOperacionGINPIX, error)
}

// ConsultorOperacionGINPIX observa por la clave ya emitida. Su implementacion
// no puede reenviar la carga ni activar una nueva operacion externa.
type ConsultorOperacionGINPIX interface {
	ConsultarOperacionGINPIX(
		context.Context,
		SolicitudOperacionGINPIX,
		ReservaOperacionGINPIX,
	) (ReciboExternoOperacionGINPIX, error)
}

func datosOperacionGINPIXValidos(d DatosOperacionGINPIX) bool {
	return referenciaClaveOperacionGINPIXValida(d.ClaveOperacionRef) &&
		huellaOperacionGINPIXValida(d.OrdenHuellaSHA256) && d.VersionExpediente > 0 &&
		d.MapeoVersion > 0 && domain.ReferenciaOpacaValida(d.ExpedienteRef) &&
		domain.ReferenciaOpacaValida(d.IncorporacionRef) &&
		domain.ReferenciaOpacaValida(d.ReciboIncorporacionRef) &&
		domain.ReferenciaOpacaValida(d.ResultadoPersonalRef) &&
		domain.ReferenciaOpacaValida(d.ReciboPersonalRef) &&
		domain.ReferenciaOpacaValida(d.CorrelacionRef) &&
		domain.ReferenciaOpacaValida(d.IdempotenciaRef) &&
		domain.ReferenciaOpacaValida(d.ProcedenciaModeloRef) &&
		domain.ReferenciaOpacaValida(d.MapeoRef) &&
		domain.ReferenciaOpacaValida(d.ProcedenciaMapeoRef) &&
		huellaOperacionGINPIXValida(d.ModeloHuellaSHA256) &&
		huellaOperacionGINPIXValida(d.MapeoHuellaSHA256) &&
		huellaOperacionGINPIXValida(d.CargaHuellaSHA256)
}

func huellaOrdenOperacionGINPIX(
	e EvidenciaOrdenConfirmarIncorporacion,
	r ReciboConfirmacionIncorporacion,
) string {
	datosConfirmacion := e.Confirmacion
	solicitudPersonal := datosConfirmacion.SolicitudPersonal
	resultadoPersonal := datosConfirmacion.ResultadoPersonal
	huellaSolicitudPersonal, _ := solicitudPersonal.HuellaSHA256()
	huellaDecision, _ := dominiovec.HuellaSHA256DecisionAutorizacionV3(
		e.Contexto.DecisionAutorizacionV3,
	)
	confirmacionV3, _ := e.Contexto.ConfirmacionRegistroV3.Datos()
	c := &canonOperacionGINPIX{}
	c.campo("dominio", "vec.dipgra.contratacion-temporal.ginpix.orden.v1")
	c.instante("evaluada_en", e.EvaluadaEn)
	c.campo("solicitud_personal_huella", huellaSolicitudPersonal)
	c.campo("resultado_personal_esquema", resultadoPersonal.Esquema)
	c.entero("resultado_personal_contrato_version", resultadoPersonal.ContratoVersion)
	c.campo("resultado_personal_ref_orden", resultadoPersonal.ResultadoRef)
	c.campo("resultado_personal_recibo_ref_orden", resultadoPersonal.ReciboRef)
	c.campo("resultado_personal_solicitud_ref", resultadoPersonal.SolicitudRef)
	c.campo("resultado_personal_correlacion_ref", resultadoPersonal.CorrelacionRef)
	c.campo("resultado_personal_idempotencia_ref", resultadoPersonal.IdempotenciaRef)
	c.campo("resultado_personal_huella_solicitud", resultadoPersonal.HuellaSolicitudSHA256)
	c.campo("resultado_personal_estado", string(resultadoPersonal.Estado))
	c.campo("resultado_personal_relacion_ref", resultadoPersonal.RelacionRef)
	c.campo("resultado_personal_ocupacion_ref", resultadoPersonal.OcupacionRef)
	c.campo("decision_autorizacion_huella", huellaDecision)
	c.campo("confirmacion_v3_decision_ref", confirmacionV3.DecisionRef)
	c.campo("confirmacion_v3_decision_huella", confirmacionV3.DecisionHuellaSHA256)
	c.instante("confirmacion_v3_emitida_en", confirmacionV3.EmitidaEn)
	c.instante("confirmacion_v3_valida_hasta", confirmacionV3.ValidaHasta)
	c.campo("recibo_ref", r.ReciboRef)
	c.campo("actuacion_ref", r.ActuacionRef)
	c.campo("correlacion_ref", r.CorrelacionRef)
	c.campo("actor_ref", r.ActorRef)
	c.campo("organizacion_ref", r.OrganizacionRef)
	c.campo("unidad_ref", r.UnidadRef)
	c.campo("expediente_ref", r.ExpedienteRef)
	c.campo("solicitud_personal_ref", r.SolicitudPersonalRef)
	c.campo("decision_autorizacion_ref", r.DecisionAutorizacionRef)
	c.campo("resultado_personal_ref", r.ResultadoPersonalRef)
	c.campo("recibo_personal_ref", r.ReciboPersonalRef)
	c.campo("relacion_ref", r.RelacionRef)
	c.campo("ocupacion_ref", r.OcupacionRef)
	c.campo("transicion", string(r.TransicionClave))
	c.campo("motivo", string(r.MotivoClave))
	c.entero("version_expediente", r.VersionExpediente)
	c.entero("version_anterior", r.VersionAnterior)
	c.entero("version_resultante", r.VersionResultante)
	c.instante("fecha_incorporacion", r.FechaIncorporacion)
	c.instante("fecha_fin_prevista", r.FechaFinPrevista)
	c.instante("confirmada_en", r.ConfirmadaEn)
	c.entero("numero_documentos", uint64(len(r.Documentos)))
	for _, documento := range r.Documentos {
		c.campo("documento_tipo", string(documento.TipoClave))
		c.campo("documento_ref", documento.Referencia)
	}
	return c.huella()
}

func referenciaOperacionGINPIX(d DatosOperacionGINPIX) string {
	c := &canonOperacionGINPIX{}
	c.campo("dominio", "vec.dipgra.contratacion-temporal.ginpix.operacion.v1")
	c.campo("orden_huella", d.OrdenHuellaSHA256)
	c.entero("version_expediente", d.VersionExpediente)
	c.campo("expediente_ref", d.ExpedienteRef)
	c.campo("incorporacion_ref", d.IncorporacionRef)
	c.campo("recibo_incorporacion_ref", d.ReciboIncorporacionRef)
	c.campo("resultado_personal_ref", d.ResultadoPersonalRef)
	c.campo("recibo_personal_ref", d.ReciboPersonalRef)
	c.campo("correlacion_ref", d.CorrelacionRef)
	c.campo("idempotencia_ref", d.IdempotenciaRef)
	c.campo("procedencia_modelo_ref", d.ProcedenciaModeloRef)
	c.campo("modelo_huella", d.ModeloHuellaSHA256)
	c.campo("mapeo_ref", d.MapeoRef)
	c.entero("mapeo_version", d.MapeoVersion)
	c.campo("procedencia_mapeo_ref", d.ProcedenciaMapeoRef)
	c.campo("mapeo_huella", d.MapeoHuellaSHA256)
	c.campo("carga_huella", d.CargaHuellaSHA256)
	return "operacion-ginpix:" + c.huella()
}

type canonOperacionGINPIX struct{ bytes.Buffer }

func (c *canonOperacionGINPIX) campo(nombre, valor string) {
	c.WriteString(strconv.Itoa(len(nombre)))
	c.WriteByte(':')
	c.WriteString(nombre)
	c.WriteString(strconv.Itoa(len(valor)))
	c.WriteByte(':')
	c.WriteString(valor)
}

func (c *canonOperacionGINPIX) entero(nombre string, valor uint64) {
	c.campo(nombre, strconv.FormatUint(valor, 10))
}

func (c *canonOperacionGINPIX) instante(nombre string, valor time.Time) {
	c.campo(nombre, valor.UTC().Format(time.RFC3339Nano))
}

func (c *canonOperacionGINPIX) huella() string {
	suma := sha256.Sum256(c.Bytes())
	return hex.EncodeToString(suma[:])
}

func referenciaClaveOperacionGINPIXValida(valor string) bool {
	const prefijo = "operacion-ginpix:"
	return len(valor) == len(prefijo)+sha256.Size*2 &&
		valor[:len(prefijo)] == prefijo && huellaOperacionGINPIXValida(valor[len(prefijo):])
}

func huellaOperacionGINPIXValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor == string(bytes.Repeat([]byte{'0'}, sha256.Size*2)) {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil && valor == string(bytes.ToLower([]byte(valor)))
}

func clonarReciboOperacionGINPIX(r ReciboConfirmacionIncorporacion) ReciboConfirmacionIncorporacion {
	r.Documentos = append([]domain.DocumentoSeguimiento(nil), r.Documentos...)
	return r
}
