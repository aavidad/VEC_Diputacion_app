package ports

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionRegistrarFiscalizacion             = "contratacion_temporal.fiscalizacion.registrar"
	OperacionRegistrarResultadoFiscalizacion = "registrar_resultado"
	TipoRecursoFiscalizacion                 = "fiscalizacion_contratacion_temporal"
	FinalidadRegistrarFiscalizacion          = "gestionar_contratacion_temporal"
	DominioAmbitoIdempotenciaFiscalizacion   = "vec.contratacion-temporal.fiscalizacion.ambito"
	DominioHuellaPeticionFiscalizacion       = "vec.contratacion-temporal.fiscalizacion.peticion"
)

var (
	ErrPreparacionFiscalizacionInvalida = errors.New(
		"contratacion temporal: preparacion de fiscalizacion invalida",
	)
	ErrResultadoFiscalizacionNoConfiable = errors.New(
		"contratacion temporal: resultado de fiscalizacion no confiable",
	)
	ErrPersistenciaFiscalizacionNoDisponible = errors.New(
		"contratacion temporal: persistencia de fiscalizacion no disponible",
	)
)

type MaterialHuellaFiscalizacion struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	ActorRef          string
	PerfilRef         string
	Resultado         domain.ResultadoFiscalizacion
	Observaciones     string
}

func (m MaterialHuellaFiscalizacion) Validar() error {
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(m.ExpedienteRef) ||
		m.VersionExpediente != 5 ||
		!domain.ReferenciaOpacaValida(m.ActorRef) ||
		!domain.ReferenciaOpacaValida(m.PerfilRef) ||
		ValidarResultadoFiscalizacion(m.Resultado, m.Observaciones) != nil {
		return ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

func ValidarResultadoFiscalizacion(
	resultado domain.ResultadoFiscalizacion,
	observaciones string,
) error {
	if !resultado.Valido() || !textoFiscalizacionValido(observaciones, true) {
		return ErrPreparacionFiscalizacionInvalida
	}
	if resultado == domain.FiscalizacionFavorable && observaciones != "" {
		return ErrPreparacionFiscalizacionInvalida
	}
	if (resultado == domain.FiscalizacionFavorableConObservaciones ||
		resultado == domain.FiscalizacionDesfavorable) &&
		!textoFiscalizacionValido(observaciones, false) {
		return ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

type SelladorAmbitoFiscalizacion interface {
	SellarAmbitoFiscalizacion(
		context.Context,
		SolicitudSellarAmbitoIdempotencia,
	) (ColeccionSellosHMAC, error)
}

type DerivadorHuellaFiscalizacion interface {
	DerivarHuellaFiscalizacion(
		context.Context,
		MaterialHuellaFiscalizacion,
	) (ColeccionSellosHMAC, error)
}

type SolicitudPrepararFiscalizacion struct {
	ClaveIdempotencia   string
	AmbitosHMAC         ColeccionSellosHMAC
	HuellasPeticionHMAC ColeccionSellosHMAC
	Material            MaterialHuellaFiscalizacion
}

func (s SolicitudPrepararFiscalizacion) Validar() error {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		s.Material.Validar() != nil ||
		s.AmbitosHMAC.ValidarDominio(DominioAmbitoIdempotenciaFiscalizacion) != nil ||
		s.HuellasPeticionHMAC.ValidarDominio(DominioHuellaPeticionFiscalizacion) != nil {
		return ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

type ReferenciasEfectoFiscalizacion struct {
	ReservaRef       string
	FiscalizacionRef string
	ReciboRef        string
	EventoRef        string
	RetornoRef       string
}

func (r ReferenciasEfectoFiscalizacion) ValidarPara(
	resultado domain.ResultadoFiscalizacion,
) error {
	for _, referencia := range []string{
		r.ReservaRef, r.FiscalizacionRef, r.ReciboRef, r.EventoRef,
	} {
		if !domain.ReferenciaOpacaValida(referencia) {
			return ErrPreparacionFiscalizacionInvalida
		}
	}
	if resultado == domain.FiscalizacionDesfavorable {
		if !domain.ReferenciaOpacaValida(r.RetornoRef) {
			return ErrPreparacionFiscalizacionInvalida
		}
	} else if r.RetornoRef != "" {
		return ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

type GeneradorReferenciasFiscalizacion interface {
	GenerarReferenciasFiscalizacion(
		context.Context,
		domain.ResultadoFiscalizacion,
	) (ReferenciasEfectoFiscalizacion, error)
}

type EstadoPreparacionFiscalizacion string

const (
	PreparacionFiscalizacionPreparada  EstadoPreparacionFiscalizacion = "preparada"
	PreparacionFiscalizacionConfirmada EstadoPreparacionFiscalizacion = "confirmada"
)

type PreparacionFiscalizacion struct {
	Expediente             domain.Expediente
	Referencias            ReferenciasEfectoFiscalizacion
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
	Material               MaterialHuellaFiscalizacion
	Estado                 EstadoPreparacionFiscalizacion
	ReciboConfirmado       *ReciboFiscalizacion
}

func (p PreparacionFiscalizacion) ValidarPara(
	solicitud SolicitudPrepararFiscalizacion,
) error {
	if solicitud.Validar() != nil || p.Expediente.Validar() != nil ||
		p.Referencias.ValidarPara(p.Material.Resultado) != nil ||
		p.Material != solicitud.Material ||
		p.Expediente.Referencia != p.Material.ExpedienteRef ||
		p.Expediente.OrganizacionRef != p.Material.OrganizacionRef ||
		p.Expediente.Version != 5 || p.Expediente.FaseActual != domain.FaseInformeJuridico ||
		p.Expediente.EstadoActual != domain.EstadoEnCurso ||
		p.Expediente.Asignacion == nil || p.Expediente.InformeJuridico == nil ||
		p.Expediente.Fiscalizacion != nil ||
		!ColeccionesHMACContienenPar(
			solicitud.AmbitosHMAC, DominioAmbitoIdempotenciaFiscalizacion,
			solicitud.HuellasPeticionHMAC, DominioHuellaPeticionFiscalizacion,
			p.AmbitoIdempotenciaHMAC, p.HuellaPeticionHMAC,
		) || (p.Estado != PreparacionFiscalizacionPreparada &&
		p.Estado != PreparacionFiscalizacionConfirmada) {
		return ErrPreparacionFiscalizacionInvalida
	}
	if p.Estado == PreparacionFiscalizacionPreparada && p.ReciboConfirmado != nil {
		return ErrPreparacionFiscalizacionInvalida
	}
	if p.Estado == PreparacionFiscalizacionConfirmada &&
		(p.ReciboConfirmado == nil ||
			p.ReciboConfirmado.ValidarParaPreparacion(p) != nil) {
		return ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

type PreparadorFiscalizacionIdempotente interface {
	PrepararFiscalizacion(
		context.Context,
		SolicitudPrepararFiscalizacion,
	) (PreparacionFiscalizacion, error)
}

type SolicitudResolverPoliticaFiscalizacion struct {
	OrganizacionRef        string
	ExpedienteRef          string
	VersionExpediente      uint64
	ActorRef               string
	PerfilRef              string
	Resultado              domain.ResultadoFiscalizacion
	Observaciones          string
	FaseActual             domain.ClaveFase
	EstadoActual           domain.EstadoOperativo
	UnidadAsignadaRef      string
	ResponsableAsignadoRef string
	InformeJuridicoRef     string
	DocumentoInformeRef    string
	Instante               time.Time
}

func (s SolicitudResolverPoliticaFiscalizacion) Validar() error {
	if (MaterialHuellaFiscalizacion{
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		VersionExpediente: s.VersionExpediente, ActorRef: s.ActorRef,
		PerfilRef: s.PerfilRef, Resultado: s.Resultado,
		Observaciones: s.Observaciones,
	}).Validar() != nil || s.FaseActual != domain.FaseInformeJuridico ||
		s.EstadoActual != domain.EstadoEnCurso ||
		!domain.ReferenciaOpacaValida(s.UnidadAsignadaRef) ||
		!domain.ReferenciaOpacaValida(s.ResponsableAsignadoRef) ||
		!domain.ReferenciaOpacaValida(s.InformeJuridicoRef) ||
		!domain.ReferenciaOpacaValida(s.DocumentoInformeRef) ||
		!domain.InstanteUTCCanonico(s.Instante) {
		return ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

type PoliticaFiscalizacion struct {
	DefinicionRef          string
	DefinicionVersion      uint64
	DefinicionHuellaSHA256 string
	Accion                 domain.ClaveCatalogo
	Finalidad              domain.ClaveCatalogo
	UnidadFiscalizadoraRef string
	MotivoAutorizacion     dominiovec.ReferenciaEntradaCatalogo
	EvaluadaEn             time.Time
	ValidaHasta            time.Time
}

func (p PoliticaFiscalizacion) ValidarPara(
	solicitud SolicitudResolverPoliticaFiscalizacion,
	instante time.Time,
) error {
	if solicitud.Validar() != nil || !domain.InstanteUTCCanonico(instante) ||
		!domain.ReferenciaOpacaValida(p.DefinicionRef) ||
		!VersionOperacionAnalisisValida(p.DefinicionVersion) ||
		!huellaSHA256OperacionAnalisisValida(p.DefinicionHuellaSHA256) ||
		p.Accion != domain.AccionRegistrarFiscalizacion ||
		p.Finalidad != domain.ClaveCatalogo(FinalidadRegistrarFiscalizacion) ||
		!domain.ReferenciaOpacaValida(p.UnidadFiscalizadoraRef) ||
		p.MotivoAutorizacion.Validar() != nil ||
		!domain.InstanteUTCCanonico(p.EvaluadaEn) ||
		!domain.InstanteUTCCanonico(p.ValidaHasta) ||
		!p.EvaluadaEn.Equal(solicitud.Instante) ||
		!p.ValidaHasta.After(p.EvaluadaEn) || instante.Before(p.EvaluadaEn) ||
		!instante.Before(p.ValidaHasta) {
		return ErrPreparacionFiscalizacionInvalida
	}
	return nil
}

type ResolutorPoliticaFiscalizacion interface {
	ResolverPoliticaFiscalizacion(
		context.Context,
		SolicitudResolverPoliticaFiscalizacion,
	) (PoliticaFiscalizacion, error)
}

type EvidenciaAutorizacionFiscalizacion struct {
	Contexto       ContextoAutorizacionAltaV3
	SolicitudV3    dominiovec.SolicitudAutorizacionLigadaV3
	DecisionV3     dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionV3 puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

type OrdenConfirmarFiscalizacion struct {
	Preparacion         PreparacionFiscalizacion
	Politica            PoliticaFiscalizacion
	ExpedienteSiguiente domain.Expediente
	Evidencia           EvidenciaAutorizacionFiscalizacion
	InstanteEfecto      time.Time
}

type TransaccionFiscalizaciones interface {
	ConfirmarFiscalizacion(
		context.Context,
		OrdenConfirmarFiscalizacion,
	) (ReciboFiscalizacion, error)
}

type ReciboFiscalizacion struct {
	Operacion             string                        `json:"operacion"`
	OrganizacionRef       string                        `json:"organizacion_ref"`
	ExpedienteRef         string                        `json:"expediente_ref"`
	VersionAnterior       uint64                        `json:"version_anterior"`
	VersionResultante     uint64                        `json:"version_resultante"`
	Resultado             domain.ResultadoFiscalizacion `json:"resultado"`
	FaseResultante        domain.ClaveFase              `json:"fase_resultante"`
	EstadoResultante      domain.EstadoOperativo        `json:"estado_resultante"`
	ReciboRef             string                        `json:"recibo_ref"`
	AuditoriaRef          string                        `json:"auditoria_ref"`
	EventoRef             string                        `json:"evento_ref"`
	ActorRef              string                        `json:"actor_ref"`
	UnidadRetornoRef      string                        `json:"unidad_retorno_ref,omitempty"`
	ResponsableRetornoRef string                        `json:"responsable_retorno_ref,omitempty"`
	RegistradaEn          time.Time                     `json:"registrada_en"`
}

func (r ReciboFiscalizacion) ValidarParaPreparacion(
	p PreparacionFiscalizacion,
) error {
	if r.Operacion != OperacionRegistrarResultadoFiscalizacion ||
		r.OrganizacionRef != p.Material.OrganizacionRef ||
		r.ExpedienteRef != p.Material.ExpedienteRef ||
		r.VersionAnterior != 5 || r.VersionResultante != 6 ||
		r.Resultado != p.Material.Resultado ||
		r.ActorRef != p.Material.ActorRef ||
		r.ReciboRef != p.Referencias.ReciboRef ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		r.EventoRef != p.Referencias.EventoRef ||
		!domain.InstanteUTCCanonico(r.RegistradaEn) {
		return ErrResultadoFiscalizacionNoConfiable
	}
	if r.Resultado == domain.FiscalizacionDesfavorable {
		if p.Expediente.Asignacion == nil ||
			r.FaseResultante != domain.FaseSubsanacionUnidad ||
			r.EstadoResultante != domain.EstadoIncidencia ||
			r.UnidadRetornoRef != p.Expediente.Asignacion.UnidadRef ||
			r.ResponsableRetornoRef != p.Expediente.Asignacion.ResponsableRef {
			return ErrResultadoFiscalizacionNoConfiable
		}
	} else if r.FaseResultante != domain.FaseFiscalizacion ||
		r.EstadoResultante != domain.EstadoEnCurso ||
		r.UnidadRetornoRef != "" || r.ResponsableRetornoRef != "" {
		return ErrResultadoFiscalizacionNoConfiable
	}
	return nil
}

func textoFiscalizacionValido(valor string, permiteVacio bool) bool {
	if valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) ||
		!norm.NFC.IsNormalString(valor) || utf8.RuneCountInString(valor) > 2000 ||
		(!permiteVacio && valor == "") {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) && caracter != '\n' && caracter != '\t' {
			return false
		}
	}
	return true
}
