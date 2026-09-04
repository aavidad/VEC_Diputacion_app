package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionEmitirInformeJuridico              = "contratacion_temporal.informe_juridico.generar"
	TipoRecursoInformeJuridico               = "informe_juridico_contratacion_temporal"
	DominioAmbitoIdempotenciaInformeJuridico = "vec.contratacion-temporal.informe-juridico.ambito"
	DominioHuellaPeticionInformeJuridico     = "vec.contratacion-temporal.informe-juridico.peticion"
	FormatoInformeJuridicoDesarrollo         = "text/plain; charset=utf-8"
)

var (
	ErrPreparacionInformeJuridicoInvalida = errors.New(
		"contratacion temporal: preparacion de informe juridico invalida",
	)
	ErrResultadoInformeJuridicoNoConfiable = errors.New(
		"contratacion temporal: resultado de informe juridico no confiable",
	)
	ErrPersistenciaInformeJuridicoNoDisponible = errors.New(
		"contratacion temporal: persistencia de informe juridico no disponible",
	)
)

type MaterialHuellaInformeJuridico struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	ActorRef          string
	PerfilRef         string
}

func (m MaterialHuellaInformeJuridico) Validar() error {
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(m.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(m.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(m.ActorRef) ||
		!domain.ReferenciaOpacaValida(m.PerfilRef) {
		return ErrPreparacionInformeJuridicoInvalida
	}
	return nil
}

type SelladorAmbitoInformeJuridico interface {
	SellarAmbitoInformeJuridico(
		context.Context,
		SolicitudSellarAmbitoIdempotencia,
	) (ColeccionSellosHMAC, error)
}

type DerivadorHuellaInformeJuridico interface {
	DerivarHuellaInformeJuridico(
		context.Context,
		MaterialHuellaInformeJuridico,
	) (ColeccionSellosHMAC, error)
}

type SolicitudPrepararInformeJuridico struct {
	ClaveIdempotencia   string
	AmbitosHMAC         ColeccionSellosHMAC
	HuellasPeticionHMAC ColeccionSellosHMAC
	Material            MaterialHuellaInformeJuridico
}

func (s SolicitudPrepararInformeJuridico) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) || s.Material.Validar() != nil ||
		s.AmbitosHMAC.ValidarDominio(DominioAmbitoIdempotenciaInformeJuridico) != nil ||
		s.HuellasPeticionHMAC.ValidarDominio(DominioHuellaPeticionInformeJuridico) != nil {
		return ErrPreparacionInformeJuridicoInvalida
	}
	return nil
}

type ReferenciasEfectoInformeJuridico struct {
	ReservaRef   string
	InformeRef   string
	DocumentoRef string
	ReciboRef    string
	AuditoriaRef string
	EventoRef    string
}

func (r ReferenciasEfectoInformeJuridico) Validar() error {
	for _, referencia := range []string{
		r.ReservaRef, r.InformeRef, r.DocumentoRef,
		r.ReciboRef, r.AuditoriaRef, r.EventoRef,
	} {
		if !domain.ReferenciaOpacaValida(referencia) {
			return ErrPreparacionInformeJuridicoInvalida
		}
	}
	return nil
}

type EstadoPreparacionInformeJuridico string

const (
	PreparacionInformeJuridicoReservada  EstadoPreparacionInformeJuridico = "reservada"
	PreparacionInformeJuridicoConfirmada EstadoPreparacionInformeJuridico = "confirmada"
)

type PreparacionInformeJuridico struct {
	Expediente             domain.Expediente
	Referencias            ReferenciasEfectoInformeJuridico
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
	Material               MaterialHuellaInformeJuridico
	Estado                 EstadoPreparacionInformeJuridico
	ReciboConfirmado       *ReciboInformeJuridico
}

func (p PreparacionInformeJuridico) ValidarPara(
	solicitud SolicitudPrepararInformeJuridico,
) error {
	if solicitud.Validar() != nil || p.Expediente.Validar() != nil ||
		p.Referencias.Validar() != nil || p.Material != solicitud.Material ||
		p.Expediente.Referencia != p.Material.ExpedienteRef ||
		p.Expediente.OrganizacionRef != p.Material.OrganizacionRef ||
		p.Expediente.Version != p.Material.VersionExpediente ||
		p.Expediente.Asignacion == nil || p.Expediente.InformeJuridico != nil ||
		!ColeccionesHMACContienenPar(
			solicitud.AmbitosHMAC, DominioAmbitoIdempotenciaInformeJuridico,
			solicitud.HuellasPeticionHMAC, DominioHuellaPeticionInformeJuridico,
			p.AmbitoIdempotenciaHMAC, p.HuellaPeticionHMAC,
		) || (p.Estado != PreparacionInformeJuridicoReservada &&
		p.Estado != PreparacionInformeJuridicoConfirmada) {
		return ErrPreparacionInformeJuridicoInvalida
	}
	if p.Estado == PreparacionInformeJuridicoReservada && p.ReciboConfirmado != nil {
		return ErrPreparacionInformeJuridicoInvalida
	}
	if p.Estado == PreparacionInformeJuridicoConfirmada &&
		(p.ReciboConfirmado == nil ||
			p.ReciboConfirmado.ValidarParaPreparacion(p) != nil) {
		return ErrPreparacionInformeJuridicoInvalida
	}
	return nil
}

type PreparadorInformeJuridicoIdempotente interface {
	PrepararInformeJuridico(
		context.Context,
		SolicitudPrepararInformeJuridico,
	) (PreparacionInformeJuridico, error)
}

type GeneradorReferenciasInformeJuridico interface {
	GenerarReferenciasInformeJuridico(
		context.Context,
	) (ReferenciasEfectoInformeJuridico, error)
}

type SolicitudResolverConfiguracionInformeJuridico struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	ActorRef          string
	PerfilRef         string
	FaseActual        domain.ClaveFase
	EstadoActual      domain.EstadoOperativo
	UnidadAsignadaRef string
	Instante          time.Time
}

func (s SolicitudResolverConfiguracionInformeJuridico) Validar() error {
	if (MaterialHuellaInformeJuridico{
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		VersionExpediente: s.VersionExpediente, ActorRef: s.ActorRef, PerfilRef: s.PerfilRef,
	}).Validar() != nil || !s.FaseActual.Valida() || !s.EstadoActual.Valido() ||
		!domain.ReferenciaOpacaValida(s.UnidadAsignadaRef) ||
		!domain.InstanteUTCCanonico(s.Instante) {
		return ErrPreparacionInformeJuridicoInvalida
	}
	return nil
}

type ConfiguracionInformeJuridico struct {
	Plantilla             domain.ReferenciaPlantillaInformeJuridico
	ReferenciasNormativas []domain.ReferenciaNormativaInformeJuridico
	Anexos                []domain.AnexoDocumentalInformeJuridico
	DefinicionRef         string
	DefinicionVersion     uint64
	DefinicionHuella      string
	Accion                domain.ClaveCatalogo
	Finalidad             domain.ClaveCatalogo
	UnidadEjecutoraRef    string
	MotivoAutorizacion    dominiovec.ReferenciaEntradaCatalogo
	EvaluadaEn            time.Time
	ValidaHasta           time.Time
}

func (c ConfiguracionInformeJuridico) ValidarPara(
	solicitud SolicitudResolverConfiguracionInformeJuridico,
	instante time.Time,
) error {
	borrador, err := domain.NuevoBorradorInformeJuridico(
		domain.DatosBorradorInformeJuridico{
			Canon:                     domain.CanonBorradorInformeJuridicoV1(),
			ExpedienteRef:             solicitud.ExpedienteRef,
			VersionEsperadaExpediente: solicitud.VersionExpediente,
			Plantilla:                 c.Plantilla, ReferenciasNormativas: c.ReferenciasNormativas,
			Anexos: c.Anexos,
		},
	)
	if solicitud.Validar() != nil || err != nil || borrador.Validar() != nil ||
		!domain.InstanteUTCCanonico(instante) ||
		!domain.ReferenciaOpacaValida(c.DefinicionRef) ||
		!VersionOperacionAnalisisValida(c.DefinicionVersion) ||
		!huellaSHA256OperacionAnalisisValida(c.DefinicionHuella) ||
		c.Accion != domain.AccionEmitirInformeJuridico ||
		c.Finalidad != domain.ClaveCatalogo("gestionar_contratacion_temporal") ||
		c.UnidadEjecutoraRef != solicitud.UnidadAsignadaRef ||
		c.MotivoAutorizacion.Validar() != nil ||
		!domain.InstanteUTCCanonico(c.EvaluadaEn) ||
		!domain.InstanteUTCCanonico(c.ValidaHasta) ||
		!c.EvaluadaEn.Equal(solicitud.Instante) ||
		!c.ValidaHasta.After(c.EvaluadaEn) || instante.Before(c.EvaluadaEn) ||
		!instante.Before(c.ValidaHasta) {
		return ErrPreparacionInformeJuridicoInvalida
	}
	return nil
}

type ResolutorConfiguracionInformeJuridico interface {
	ResolverConfiguracionInformeJuridico(
		context.Context,
		SolicitudResolverConfiguracionInformeJuridico,
	) (ConfiguracionInformeJuridico, error)
}

type SolicitudGenerarDocumentoInformeJuridico struct {
	DocumentoRef string
	Borrador     domain.BorradorInformeJuridico
}

type DocumentoInformeJuridico struct {
	DocumentoRef          string `json:"documento_ref"`
	VersionDocumento      uint64 `json:"version_documento"`
	Formato               string `json:"formato"`
	Nombre                string `json:"nombre"`
	HuellaDocumentoSHA256 string `json:"huella_documento_sha256"`
	HuellaPaqueteSHA256   string `json:"huella_paquete_sha256"`
	ContenidoDesarrollo   string `json:"contenido_desarrollo"`
}

func (d DocumentoInformeJuridico) ValidarPara(
	solicitud SolicitudGenerarDocumentoInformeJuridico,
) error {
	huellaContenido := sha256.Sum256([]byte(d.ContenidoDesarrollo))
	if solicitud.Borrador.Validar() != nil ||
		d.DocumentoRef != solicitud.DocumentoRef ||
		!domain.ReferenciaOpacaValida(d.DocumentoRef) ||
		!VersionOperacionAnalisisValida(d.VersionDocumento) ||
		d.Formato != FormatoInformeJuridicoDesarrollo || strings.TrimSpace(d.Nombre) == "" ||
		d.Nombre != strings.TrimSpace(d.Nombre) || len(d.Nombre) > 512 ||
		!utf8.ValidString(d.Nombre) ||
		!huellaSHA256OperacionAnalisisValida(d.HuellaDocumentoSHA256) ||
		!huellaSHA256OperacionAnalisisValida(d.HuellaPaqueteSHA256) ||
		len(d.ContenidoDesarrollo) == 0 || len(d.ContenidoDesarrollo) > 256*1024 ||
		!utf8.ValidString(d.ContenidoDesarrollo) ||
		!strings.Contains(d.ContenidoDesarrollo, "DOCUMENTO DE DESARROLLO") ||
		hex.EncodeToString(huellaContenido[:]) != d.HuellaDocumentoSHA256 {
		return ErrResultadoInformeJuridicoNoConfiable
	}
	return nil
}

type GeneradorDocumentoInformeJuridico interface {
	GenerarDocumentoInformeJuridico(
		context.Context,
		SolicitudGenerarDocumentoInformeJuridico,
	) (DocumentoInformeJuridico, error)
}

type EvidenciaAutorizacionInformeJuridico struct {
	Contexto       ContextoAutorizacionAltaV3
	SolicitudV3    dominiovec.SolicitudAutorizacionLigadaV3
	DecisionV3     dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionV3 puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

type OrdenConfirmarInformeJuridico struct {
	Preparacion         PreparacionInformeJuridico
	Configuracion       ConfiguracionInformeJuridico
	Borrador            domain.BorradorInformeJuridico
	Documento           DocumentoInformeJuridico
	ExpedienteSiguiente domain.Expediente
	Evidencia           EvidenciaAutorizacionInformeJuridico
	InstanteEfecto      time.Time
}

type TransaccionInformesJuridicos interface {
	ConfirmarInformeJuridico(
		context.Context,
		OrdenConfirmarInformeJuridico,
	) (ReciboInformeJuridico, error)
}

type ReciboInformeJuridico struct {
	Operacion              string    `json:"operacion"`
	OrganizacionRef        string    `json:"organizacion_ref"`
	ExpedienteRef          string    `json:"expediente_ref"`
	VersionAnterior        uint64    `json:"version_anterior"`
	VersionResultante      uint64    `json:"version_resultante"`
	InformeRef             string    `json:"informe_ref"`
	DocumentoRef           string    `json:"documento_ref"`
	VersionDocumento       uint64    `json:"version_documento"`
	Formato                string    `json:"formato"`
	Nombre                 string    `json:"nombre"`
	HuellaDocumentoSHA256  string    `json:"huella_documento_sha256"`
	HuellaBorradorSHA256   string    `json:"huella_borrador_sha256"`
	ReciboRef              string    `json:"recibo_ref"`
	AuditoriaRef           string    `json:"auditoria_ref"`
	EventoRef              string    `json:"evento_ref"`
	ConcesionV3DecisionRef string    `json:"concesion_v3_decision_ref"`
	AmbitoIdempotenciaHMAC string    `json:"ambito_idempotencia_hmac"`
	HuellaPeticionHMAC     string    `json:"huella_peticion_hmac"`
	ContenidoDesarrollo    string    `json:"contenido_desarrollo"`
	ConfirmadaEn           time.Time `json:"confirmada_en"`
}

func (r ReciboInformeJuridico) ValidarParaPreparacion(
	p PreparacionInformeJuridico,
) error {
	huellaContenido := sha256.Sum256([]byte(r.ContenidoDesarrollo))
	if r.Operacion != "preparar" || r.OrganizacionRef != p.Material.OrganizacionRef ||
		r.ExpedienteRef != p.Material.ExpedienteRef ||
		r.VersionAnterior != p.Material.VersionExpediente ||
		r.VersionResultante != r.VersionAnterior+1 ||
		r.InformeRef != p.Referencias.InformeRef ||
		r.DocumentoRef != p.Referencias.DocumentoRef ||
		!VersionOperacionAnalisisValida(r.VersionDocumento) ||
		r.Formato != FormatoInformeJuridicoDesarrollo || strings.TrimSpace(r.Nombre) == "" ||
		!huellaSHA256OperacionAnalisisValida(r.HuellaDocumentoSHA256) ||
		!huellaSHA256OperacionAnalisisValida(r.HuellaBorradorSHA256) ||
		r.ReciboRef != p.Referencias.ReciboRef ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		r.EventoRef != p.Referencias.EventoRef ||
		!domain.ReferenciaOpacaValida(r.ConcesionV3DecisionRef) ||
		r.AmbitoIdempotenciaHMAC != p.AmbitoIdempotenciaHMAC ||
		r.HuellaPeticionHMAC != p.HuellaPeticionHMAC ||
		!utf8.ValidString(r.ContenidoDesarrollo) ||
		!strings.Contains(r.ContenidoDesarrollo, "DOCUMENTO DE DESARROLLO") ||
		hex.EncodeToString(huellaContenido[:]) != r.HuellaDocumentoSHA256 ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) {
		return ErrResultadoInformeJuridicoNoConfiable
	}
	return nil
}
