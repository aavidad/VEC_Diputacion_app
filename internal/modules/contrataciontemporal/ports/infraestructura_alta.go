package ports

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	DominioAmbitoIdempotenciaAsignacion = "vec.contratacion-temporal.asignacion.ambito"
	DominioHuellaPeticionAsignacion     = "vec.contratacion-temporal.asignacion.peticion"
	estadoCandidatoAsignacionRedactado  = "[ESTADO-CANDIDATO-ASIGNACION-IDEMPOTENTE-OPACO]"
)

var (
	ErrPreparacionAsignacionInvalida = errors.New(
		"contratacion temporal: preparacion de asignacion invalida",
	)
	ErrResultadoAsignacionNoConfiable = errors.New(
		"contratacion temporal: resultado de asignacion no confiable",
	)
)

type SolicitudSellarAmbitoIdempotencia struct {
	ClaveIdempotencia string
	OrganizacionRef   string
	ActorRef          string
	PerfilRef         string
}

func (s SolicitudSellarAmbitoIdempotencia) Validar() error {
	if !claveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// SelladorAmbitoIdempotencia deriva un identificador HMAC sin persistir ni
// exponer la clave aportada. La clave criptográfica procede del gestor de
// secretos y su identificador/versionado pertenece al adaptador concreto.
type SelladorAmbitoIdempotencia interface {
	SellarAmbitoIdempotencia(
		context.Context,
		SolicitudSellarAmbitoIdempotencia,
	) (ColeccionSellosHMAC, error)
}

// GeneradorReferenciasAlta acuña candidatos opacos. PostgreSQL decide cuáles
// prevalecen ante dos preparaciones concurrentes del mismo ámbito.
type GeneradorReferenciasAlta interface {
	GenerarReferenciasAlta(context.Context) (ReferenciasAlta, error)
	NuevaReferenciaReservaAlta(context.Context) (string, error)
}

type MaterialHuellaAsignacion struct {
	Operacion               TipoOperacionAsignacion
	OrganizacionRef         string
	ExpedienteRef           string
	VersionExpediente       uint64
	ActorRef                string
	PerfilRef               string
	UnidadRef               string
	ResponsableRef          string
	MotivoReasignacionClave domain.ClaveCatalogo
	Observaciones           string
}

func (m MaterialHuellaAsignacion) Validar() error {
	if !m.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(m.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(m.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(m.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(m.ActorRef) ||
		!domain.ReferenciaOpacaValida(m.PerfilRef) ||
		!domain.ReferenciaOpacaValida(m.UnidadRef) ||
		!domain.ReferenciaOpacaValida(m.ResponsableRef) ||
		!textoAsignacionValido(m.Observaciones, 1000, true) {
		return ErrPreparacionAsignacionInvalida
	}
	if m.Operacion == OperacionRegistrarAsignacion {
		if m.MotivoReasignacionClave != "" || m.Observaciones != "" {
			return ErrPreparacionAsignacionInvalida
		}
		return nil
	}
	if !m.MotivoReasignacionClave.Valida() ||
		!textoAsignacionValido(m.Observaciones, 1000, false) {
		return ErrPreparacionAsignacionInvalida
	}
	return nil
}

type DerivadorHuellaAsignacion interface {
	DerivarHuellaAsignacion(
		context.Context,
		MaterialHuellaAsignacion,
	) (ColeccionSellosHMAC, error)
}

type SelladorAmbitoAsignacion interface {
	SellarAmbitoAsignacion(
		context.Context,
		SolicitudSellarAmbitoIdempotencia,
	) (ColeccionSellosHMAC, error)
}

type SolicitudPrepararAsignacion struct {
	ClaveIdempotencia   string
	AmbitosHMAC         ColeccionSellosHMAC
	HuellasPeticionHMAC ColeccionSellosHMAC
	Operacion           TipoOperacionAsignacion
	OrganizacionRef     string
	ExpedienteRef       string
	VersionExpediente   uint64
	ActorRef            string
	PerfilRef           string
	UnidadRef           string
	ResponsableRef      string
}

func (s SolicitudPrepararAsignacion) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		s.AmbitosHMAC.ValidarDominio(
			DominioAmbitoIdempotenciaAsignacion,
		) != nil ||
		s.HuellasPeticionHMAC.ValidarDominio(
			DominioHuellaPeticionAsignacion,
		) != nil ||
		!s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!domain.ReferenciaOpacaValida(s.UnidadRef) ||
		!domain.ReferenciaOpacaValida(s.ResponsableRef) {
		return ErrPreparacionAsignacionInvalida
	}
	return nil
}

type ReferenciasEfectoAsignacion struct {
	ReservaRef      string
	ReciboRef       string
	NotificacionRef string
	BandejaRef      string
	AuditoriaRef    string
	EventoRef       string
}

func (r ReferenciasEfectoAsignacion) Validar() error {
	if !domain.ReferenciaOpacaValida(r.ReservaRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.NotificacionRef) ||
		!domain.ReferenciaOpacaValida(r.BandejaRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) {
		return ErrPreparacionAsignacionInvalida
	}
	return nil
}

type EstadoPreparacionAsignacion string

const (
	PreparacionAsignacionReservada  EstadoPreparacionAsignacion = "reservada"
	PreparacionAsignacionConfirmada EstadoPreparacionAsignacion = "confirmada"
)

type PreparacionAsignacion struct {
	Expediente             domain.Expediente
	Referencias            ReferenciasEfectoAsignacion
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
	Operacion              TipoOperacionAsignacion
	OrganizacionRef        string
	ActorRef               string
	PerfilRef              string
	UnidadRef              string
	ResponsableRef         string
	Estado                 EstadoPreparacionAsignacion
	ReciboConfirmado       *ReciboAsignacion
}

func (p PreparacionAsignacion) ValidarPara(
	solicitud SolicitudPrepararAsignacion,
) error {
	if solicitud.Validar() != nil || p.Expediente.Validar() != nil ||
		p.Referencias.Validar() != nil ||
		!ColeccionesHMACContienenPar(
			solicitud.AmbitosHMAC,
			DominioAmbitoIdempotenciaAsignacion,
			solicitud.HuellasPeticionHMAC,
			DominioHuellaPeticionAsignacion,
			p.AmbitoIdempotenciaHMAC,
			p.HuellaPeticionHMAC,
		) ||
		p.Operacion != solicitud.Operacion ||
		p.OrganizacionRef != solicitud.OrganizacionRef ||
		p.Expediente.Referencia != solicitud.ExpedienteRef ||
		p.Expediente.OrganizacionRef != solicitud.OrganizacionRef ||
		p.Expediente.Version != solicitud.VersionExpediente ||
		p.ActorRef != solicitud.ActorRef ||
		p.PerfilRef != solicitud.PerfilRef ||
		p.UnidadRef != solicitud.UnidadRef ||
		p.ResponsableRef != solicitud.ResponsableRef ||
		(p.Estado != PreparacionAsignacionReservada &&
			p.Estado != PreparacionAsignacionConfirmada) {
		return ErrPreparacionAsignacionInvalida
	}
	if p.Operacion == OperacionRegistrarAsignacion &&
		p.Expediente.Asignacion != nil {
		return ErrPreparacionAsignacionInvalida
	}
	if p.Operacion == OperacionRegistrarReasignacion &&
		p.Expediente.Asignacion == nil {
		return ErrPreparacionAsignacionInvalida
	}
	if p.Estado == PreparacionAsignacionReservada &&
		p.ReciboConfirmado != nil {
		return ErrPreparacionAsignacionInvalida
	}
	if p.Estado == PreparacionAsignacionConfirmada &&
		(p.ReciboConfirmado == nil ||
			p.ReciboConfirmado.ValidarParaPreparacion(p) != nil) {
		return ErrPreparacionAsignacionInvalida
	}
	return nil
}

type PreparadorAsignacionIdempotente interface {
	PrepararAsignacion(
		context.Context,
		SolicitudPrepararAsignacion,
	) (PreparacionAsignacion, error)
}

// SolicitudConsultarAsignacionIdempotente identifica un posible terminal por
// el par HMAC activo y por sus coordenadas exactas. No contiene la UUID en
// claro, no concede autoridad y no permite recuperar otra intención.
type SolicitudConsultarAsignacionIdempotente struct {
	AmbitoIdempotenciaHMACActivo string
	HuellaPeticionHMACActiva     string
	Operacion                    TipoOperacionAsignacion
	OrganizacionRef              string
	ExpedienteRef                string
	VersionExpediente            uint64
	ActorRef                     string
	PerfilRef                    string
	UnidadRef                    string
	ResponsableRef               string
}

func NuevaSolicitudConsultarAsignacionIdempotente(
	solicitud SolicitudPrepararAsignacion,
) (SolicitudConsultarAsignacionIdempotente, error) {
	if solicitud.Validar() != nil {
		return SolicitudConsultarAsignacionIdempotente{},
			ErrPreparacionAsignacionInvalida
	}
	ambito, huella, err := ParActivoColeccionesHMAC(
		solicitud.AmbitosHMAC,
		DominioAmbitoIdempotenciaAsignacion,
		solicitud.HuellasPeticionHMAC,
		DominioHuellaPeticionAsignacion,
	)
	if err != nil {
		return SolicitudConsultarAsignacionIdempotente{},
			ErrPreparacionAsignacionInvalida
	}
	consulta := SolicitudConsultarAsignacionIdempotente{
		AmbitoIdempotenciaHMACActivo: ambito,
		HuellaPeticionHMACActiva:     huella,
		Operacion:                    solicitud.Operacion,
		OrganizacionRef:              solicitud.OrganizacionRef,
		ExpedienteRef:                solicitud.ExpedienteRef,
		VersionExpediente:            solicitud.VersionExpediente,
		ActorRef:                     solicitud.ActorRef,
		PerfilRef:                    solicitud.PerfilRef,
		UnidadRef:                    solicitud.UnidadRef,
		ResponsableRef:               solicitud.ResponsableRef,
	}
	if consulta.Validar() != nil {
		return SolicitudConsultarAsignacionIdempotente{},
			ErrPreparacionAsignacionInvalida
	}
	return consulta, nil
}

func (s SolicitudConsultarAsignacionIdempotente) Validar() error {
	dominioAmbito, generacionAmbito, ambitoValido := descomponerSelloHMAC(
		s.AmbitoIdempotenciaHMACActivo,
	)
	dominioHuella, generacionHuella, huellaValida := descomponerSelloHMAC(
		s.HuellaPeticionHMACActiva,
	)
	if !ambitoValido || !huellaValida ||
		dominioAmbito != DominioAmbitoIdempotenciaAsignacion ||
		dominioHuella != DominioHuellaPeticionAsignacion ||
		generacionAmbito != generacionHuella || !s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!domain.ReferenciaOpacaValida(s.UnidadRef) ||
		!domain.ReferenciaOpacaValida(s.ResponsableRef) {
		return ErrPreparacionAsignacionInvalida
	}
	return nil
}

// DatosEstadoCandidatoAsignacionIdempotente contiene los compromisos durables
// mínimos que permiten detectar cambios de destino o política sin convertir
// el recibo anterior en autoridad.
type DatosEstadoCandidatoAsignacionIdempotente struct {
	Consulta                     SolicitudConsultarAsignacionIdempotente
	Preparacion                  PreparacionAsignacion
	DestinoEvidenciaRef          string
	DestinoEvidenciaHuellaSHA256 string
	PoliticaRef                  string
	PoliticaVersion              uint64
	PoliticaHuellaSHA256         string
	Finalidad                    domain.ClaveCatalogo
}

type datosEstadoCandidatoAsignacionIdempotente struct {
	datos        DatosEstadoCandidatoAsignacionIdempotente
	reconciliada atomic.Bool
}

// EstadoCandidatoAsignacionIdempotente es una capacidad opaca y de un solo
// uso. Un adaptador durable solo la construye tras una consulta de lectura
// exacta; la aplicación no puede alterar ni serializar sus compromisos.
type EstadoCandidatoAsignacionIdempotente struct {
	datos *datosEstadoCandidatoAsignacionIdempotente
}

func NuevoEstadoCandidatoAsignacionIdempotente(
	datos DatosEstadoCandidatoAsignacionIdempotente,
) (EstadoCandidatoAsignacionIdempotente, error) {
	if validarEstadoCandidatoAsignacion(datos) != nil {
		return EstadoCandidatoAsignacionIdempotente{},
			ErrResultadoAsignacionNoConfiable
	}
	copia := datos
	copia.Preparacion = clonarPreparacionAsignacion(datos.Preparacion)
	return EstadoCandidatoAsignacionIdempotente{
		datos: &datosEstadoCandidatoAsignacionIdempotente{datos: copia},
	}, nil
}

func (e EstadoCandidatoAsignacionIdempotente) PreparacionPara(
	consulta SolicitudConsultarAsignacionIdempotente,
) (PreparacionAsignacion, error) {
	if e.datos == nil || consulta.Validar() != nil ||
		e.datos.datos.Consulta != consulta ||
		validarEstadoCandidatoAsignacion(e.datos.datos) != nil {
		return PreparacionAsignacion{}, ErrResultadoAsignacionNoConfiable
	}
	return clonarPreparacionAsignacion(e.datos.datos.Preparacion), nil
}

func (e EstadoCandidatoAsignacionIdempotente) Reconciliar(
	evidencia EvidenciaReconciliacionAsignacion,
) (ReciboAsignacion, error) {
	if e.datos == nil || !e.datos.reconciliada.CompareAndSwap(false, true) ||
		validarReconciliacionAsignacion(e.datos.datos, evidencia) != nil {
		return ReciboAsignacion{}, ErrResultadoAsignacionNoConfiable
	}
	return *e.datos.datos.Preparacion.ReciboConfirmado, nil
}

func (e EstadoCandidatoAsignacionIdempotente) EsCero() bool {
	return e.datos == nil
}

func (EstadoCandidatoAsignacionIdempotente) String() string {
	return estadoCandidatoAsignacionRedactado
}

func (e EstadoCandidatoAsignacionIdempotente) GoString() string {
	return e.String()
}

func (e EstadoCandidatoAsignacionIdempotente) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, e.String())
}

func (e EstadoCandidatoAsignacionIdempotente) LogValue() slog.Value {
	return slog.StringValue(e.String())
}

func (EstadoCandidatoAsignacionIdempotente) MarshalJSON() ([]byte, error) {
	return nil, ErrResultadoAsignacionNoConfiable
}

func (*EstadoCandidatoAsignacionIdempotente) UnmarshalJSON([]byte) error {
	return ErrResultadoAsignacionNoConfiable
}

func (EstadoCandidatoAsignacionIdempotente) MarshalText() ([]byte, error) {
	return nil, ErrResultadoAsignacionNoConfiable
}

func (*EstadoCandidatoAsignacionIdempotente) UnmarshalText([]byte) error {
	return ErrResultadoAsignacionNoConfiable
}

func (EstadoCandidatoAsignacionIdempotente) MarshalBinary() ([]byte, error) {
	return nil, ErrResultadoAsignacionNoConfiable
}

func (*EstadoCandidatoAsignacionIdempotente) UnmarshalBinary([]byte) error {
	return ErrResultadoAsignacionNoConfiable
}

func (EstadoCandidatoAsignacionIdempotente) GobEncode() ([]byte, error) {
	return nil, ErrResultadoAsignacionNoConfiable
}

func (*EstadoCandidatoAsignacionIdempotente) GobDecode([]byte) error {
	return ErrResultadoAsignacionNoConfiable
}

func (EstadoCandidatoAsignacionIdempotente) MarshalCBOR() ([]byte, error) {
	return nil, ErrResultadoAsignacionNoConfiable
}

func (*EstadoCandidatoAsignacionIdempotente) UnmarshalCBOR([]byte) error {
	return ErrResultadoAsignacionNoConfiable
}

func (EstadoCandidatoAsignacionIdempotente) MarshalYAML() (any, error) {
	return nil, ErrResultadoAsignacionNoConfiable
}

func (*EstadoCandidatoAsignacionIdempotente) UnmarshalYAML(func(any) error) error {
	return ErrResultadoAsignacionNoConfiable
}

func (EstadoCandidatoAsignacionIdempotente) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrResultadoAsignacionNoConfiable
}

func (*EstadoCandidatoAsignacionIdempotente) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrResultadoAsignacionNoConfiable
}

// ConsultorAsignacionIdempotente es una frontera exclusivamente de lectura.
// encontrado=false exige estado cero; cualquier otra combinación es ambigua.
type ConsultorAsignacionIdempotente interface {
	ConsultarAsignacion(
		context.Context,
		SolicitudConsultarAsignacionIdempotente,
	) (EstadoCandidatoAsignacionIdempotente, bool, error)
}

func validarEstadoCandidatoAsignacion(
	datos DatosEstadoCandidatoAsignacionIdempotente,
) error {
	p := datos.Preparacion
	if datos.Consulta.Validar() != nil || p.Expediente.Validar() != nil ||
		p.Referencias.Validar() != nil ||
		p.Estado != PreparacionAsignacionConfirmada ||
		p.ReciboConfirmado == nil ||
		p.ReciboConfirmado.ValidarParaPreparacion(p) != nil ||
		p.Operacion != datos.Consulta.Operacion ||
		p.OrganizacionRef != datos.Consulta.OrganizacionRef ||
		p.Expediente.Referencia != datos.Consulta.ExpedienteRef ||
		p.Expediente.Version != datos.Consulta.VersionExpediente ||
		p.ActorRef != datos.Consulta.ActorRef ||
		p.PerfilRef != datos.Consulta.PerfilRef ||
		p.UnidadRef != datos.Consulta.UnidadRef ||
		p.ResponsableRef != datos.Consulta.ResponsableRef ||
		!sellosHMACIguales(
			p.AmbitoIdempotenciaHMAC,
			datos.Consulta.AmbitoIdempotenciaHMACActivo,
		) ||
		!sellosHMACIguales(
			p.HuellaPeticionHMAC,
			datos.Consulta.HuellaPeticionHMACActiva,
		) ||
		!domain.ReferenciaOpacaValida(datos.DestinoEvidenciaRef) ||
		!huellaSHA256OperacionAnalisisValida(
			datos.DestinoEvidenciaHuellaSHA256,
		) ||
		!domain.ReferenciaOpacaValida(datos.PoliticaRef) ||
		!VersionOperacionAnalisisValida(datos.PoliticaVersion) ||
		!huellaSHA256OperacionAnalisisValida(datos.PoliticaHuellaSHA256) ||
		!datos.Finalidad.Valida() {
		return ErrResultadoAsignacionNoConfiable
	}
	return nil
}

func clonarPreparacionAsignacion(
	preparacion PreparacionAsignacion,
) PreparacionAsignacion {
	copia := preparacion
	copia.Expediente = preparacion.Expediente.Clonar()
	if preparacion.ReciboConfirmado != nil {
		recibo := *preparacion.ReciboConfirmado
		copia.ReciboConfirmado = &recibo
	}
	return copia
}

type GeneradorReferenciasAsignacion interface {
	GenerarReferenciasAsignacion(
		context.Context,
	) (ReferenciasEfectoAsignacion, error)
}

type ReciboAsignacion struct {
	Operacion              TipoOperacionAsignacion `json:"operacion"`
	OrganizacionRef        string                  `json:"organizacion_ref"`
	ExpedienteRef          string                  `json:"expediente_ref"`
	VersionAnterior        uint64                  `json:"version_anterior"`
	VersionResultante      uint64                  `json:"version_resultante"`
	UnidadRef              string                  `json:"unidad_ref"`
	ResponsableRef         string                  `json:"responsable_ref"`
	ReciboRef              string                  `json:"recibo_ref"`
	NotificacionRef        string                  `json:"notificacion_ref"`
	BandejaRef             string                  `json:"bandeja_ref"`
	AuditoriaRef           string                  `json:"auditoria_ref"`
	EventoRef              string                  `json:"evento_ref"`
	ConcesionV3DecisionRef string                  `json:"concesion_v3_decision_ref"`
	AmbitoIdempotenciaHMAC string                  `json:"ambito_idempotencia_hmac"`
	HuellaPeticionHMAC     string                  `json:"huella_peticion_hmac"`
	ConfirmadaEn           time.Time               `json:"confirmada_en"`
}

func (r ReciboAsignacion) ValidarParaPreparacion(
	preparacion PreparacionAsignacion,
) error {
	if !r.Operacion.Valida() || r.Operacion != preparacion.Operacion ||
		r.OrganizacionRef != preparacion.OrganizacionRef ||
		r.ExpedienteRef != preparacion.Expediente.Referencia ||
		r.VersionAnterior != preparacion.Expediente.Version ||
		!VersionOperacionAnalisisConIncrementoValida(r.VersionAnterior) ||
		r.VersionResultante != r.VersionAnterior+1 ||
		r.UnidadRef != preparacion.UnidadRef ||
		r.ResponsableRef != preparacion.ResponsableRef ||
		r.ReciboRef != preparacion.Referencias.ReciboRef ||
		r.NotificacionRef != preparacion.Referencias.NotificacionRef ||
		r.BandejaRef != preparacion.Referencias.BandejaRef ||
		r.AuditoriaRef != preparacion.Referencias.AuditoriaRef ||
		r.EventoRef != preparacion.Referencias.EventoRef ||
		!domain.ReferenciaOpacaValida(r.ConcesionV3DecisionRef) ||
		!sellosHMACIguales(
			r.AmbitoIdempotenciaHMAC,
			preparacion.AmbitoIdempotenciaHMAC,
		) ||
		!sellosHMACIguales(
			r.HuellaPeticionHMAC,
			preparacion.HuellaPeticionHMAC,
		) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) {
		return ErrResultadoAsignacionNoConfiable
	}
	return nil
}

func (r ReciboAsignacion) ValidarParaOrden(
	orden OrdenConfirmarAsignacion,
) error {
	evidencia, err := orden.Datos()
	confirmacion, errConfirmacion := evidencia.ConfirmacionV3.Datos()
	if err != nil || errConfirmacion != nil ||
		r.ValidarParaPreparacion(evidencia.Preparacion) != nil ||
		r.ConcesionV3DecisionRef != confirmacion.DecisionRef ||
		r.ConfirmadaEn.Before(evidencia.InstanteEfecto) ||
		orden.ValidarDentroDeTransaccion(r.ConfirmadaEn) != nil {
		return ErrResultadoAsignacionNoConfiable
	}
	return nil
}

func textoAsignacionValido(
	valor string,
	maximo int,
	permiteVacio bool,
) bool {
	if valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) ||
		!norm.NFC.IsNormalString(valor) ||
		utf8.RuneCountInString(valor) > maximo ||
		(!permiteVacio && valor == "") {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) &&
			caracter != '\n' && caracter != '\t' {
			return false
		}
	}
	return true
}
