package ports

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConfirmarIncorporacion                                             = "contratacion_temporal.incorporacion.confirmar"
	FinalidadConfirmarIncorporacion                                          = "registrar_incorporacion_confirmada_por_personal"
	TipoRecursoConfirmacionIncorporacion                                     = "confirmacion_incorporacion_contratacion_temporal"
	TransicionConfirmarIncorporacion                    domain.ClaveCatalogo = "confirmar_incorporacion"
	ambitoOrganizacionConfirmacionIncorporacion                              = "organizacion_ref"
	ambitoUnidadConfirmacionIncorporacion                                    = "unidad_ref"
	atributoResultadoConfirmacionIncorporacion                               = "resultado_personal_ref"
	atributoRelacionConfirmacionIncorporacion                                = "relacion_ref"
	atributoOcupacionConfirmacionIncorporacion                               = "ocupacion_ref"
	atributoVersionExpedienteConfirmacionIncorporacion                       = "version_expediente_esperada"
	atributoVersionSeguimientoConfirmacionIncorporacion                      = "version_seguimiento_esperada"
	maximoDocumentosIncorporacion                                            = 32
)

var (
	ErrContextoConfirmacionIncorporacionInvalido = errors.New(
		"contratacion temporal: contexto de confirmacion de incorporacion invalido",
	)
	ErrOrdenConfirmacionIncorporacionInvalida = errors.New(
		"contratacion temporal: orden de confirmacion de incorporacion invalida",
	)
	ErrReciboConfirmacionIncorporacionInvalido = errors.New(
		"contratacion temporal: recibo de confirmacion de incorporacion invalido",
	)
)

// ContextoConfirmacionIncorporacion conserva la autorizacion nominal V3 ya
// resuelta desde contexto confiable. Ninguna referencia aislada concede el
// efecto: solicitud, decision y confirmacion durable se cotejan juntas.
type ContextoConfirmacionIncorporacion struct {
	SolicitudContexto       SolicitudResolverContextoAutorizacionAltaV3
	ContextoAutorizacion    ContextoAutorizacionAltaV3
	SolicitudAutorizacionV3 dominiovec.SolicitudAutorizacionLigadaV3
	DecisionAutorizacionV3  dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionRegistroV3  puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

func (c ContextoConfirmacionIncorporacion) ValidarPara(
	datos DatosConfirmacionIncorporacion,
	instante time.Time,
) error {
	normalizados, err := normalizarDatosConfirmacionIncorporacion(datos)
	if err != nil || !domain.InstanteUTCCanonico(instante) ||
		c.ContextoAutorizacion.ValidarPara(c.SolicitudContexto, instante) != nil {
		return ErrContextoConfirmacionIncorporacionInvalido
	}
	solicitudV3, errSolicitud := c.SolicitudAutorizacionV3.Datos()
	vinculo, errVinculo := solicitudV3.VinculoAutenticacionActor.Datos()
	correlacion, errCorrelacion := solicitudV3.Correlacion.ValorCanonico()
	concedida, _, errDecision := c.DecisionAutorizacionV3.Resultado()
	emitidaEn, validaHasta, errVentana := c.DecisionAutorizacionV3.VentanaValidez()
	huellaDecision, errHuella := dominiovec.HuellaSHA256DecisionAutorizacionV3(
		c.DecisionAutorizacionV3,
	)
	confirmacion, errConfirmacion := c.ConfirmacionRegistroV3.Datos()
	recurso := solicitudV3.Recurso
	if errSolicitud != nil || errVinculo != nil || errCorrelacion != nil ||
		errDecision != nil || errVentana != nil || errHuella != nil ||
		errConfirmacion != nil || !concedida ||
		!solicitudV3.VinculoAutenticacionActor.CoincideExactamenteCon(
			c.ContextoAutorizacion.Vinculo,
		) || vinculo.PerfilActivoRef != c.SolicitudContexto.PerfilRef ||
		vinculo.GarantiaObservada != dominiovec.AuthAssuranceHigh ||
		c.DecisionAutorizacionV3.ValidarPara(c.SolicitudAutorizacionV3) != nil ||
		solicitudV3.Accion != AccionConfirmarIncorporacion ||
		solicitudV3.Finalidad != FinalidadConfirmarIncorporacion ||
		correlacion != normalizados.ResultadoPersonal.CorrelacionRef ||
		recurso.Referencia != normalizados.SolicitudPersonal.ExpedienteRef ||
		recurso.ModuloID != ModuloContratacion ||
		recurso.Tipo != TipoRecursoConfirmacionIncorporacion ||
		!recursoConfirmacionIncorporacionExacto(recurso, normalizados) ||
		confirmacion.DecisionRef == "" ||
		confirmacion.DecisionHuellaSHA256 != huellaDecision ||
		!confirmacion.EmitidaEn.Equal(emitidaEn) ||
		!confirmacion.ValidaHasta.Equal(validaHasta) ||
		!c.ConfirmacionRegistroV3.DentroDeVentanaEn(instante) {
		return ErrContextoConfirmacionIncorporacionInvalido
	}
	return nil
}

func recursoConfirmacionIncorporacionExacto(
	recurso dominiovec.RecursoAutorizable,
	datos DatosConfirmacionIncorporacion,
) bool {
	return len(recurso.Ambitos) == 2 && len(recurso.Atributos) == 5 &&
		domain.ReferenciaOpacaValida(recurso.Ambitos[ambitoOrganizacionConfirmacionIncorporacion]) &&
		domain.ReferenciaOpacaValida(recurso.Ambitos[ambitoUnidadConfirmacionIncorporacion]) &&
		recurso.Atributos[atributoResultadoConfirmacionIncorporacion] == datos.ResultadoPersonal.ResultadoRef &&
		recurso.Atributos[atributoRelacionConfirmacionIncorporacion] == datos.ResultadoPersonal.RelacionRef &&
		recurso.Atributos[atributoOcupacionConfirmacionIncorporacion] == datos.ResultadoPersonal.OcupacionRef &&
		recurso.Atributos[atributoVersionExpedienteConfirmacionIncorporacion] ==
			strconv.FormatUint(datos.SolicitudPersonal.VersionExpediente, 10) &&
		recurso.Atributos[atributoVersionSeguimientoConfirmacionIncorporacion] ==
			strconv.FormatUint(datos.VersionSeguimientoEsperada, 10)
}

// ResolutorContextoConfirmacionIncorporacion obtiene de la frontera confiable
// el material V3 completo; no recibe identidad, perfil ni autoridad del canal.
type ResolutorContextoConfirmacionIncorporacion interface {
	ResolverContextoConfirmacionIncorporacion(
		context.Context,
	) (ContextoConfirmacionIncorporacion, error)
}

type DatosConfirmacionIncorporacion struct {
	SolicitudPersonal          SolicitudAltaPersonalRPT
	ResultadoPersonal          ResultadoAltaPersonalRPT
	VersionSeguimientoEsperada uint64
	PeriodoIncorporacion       domain.IntervaloSeguimiento
	MotivoClave                domain.ClaveCatalogo
	Documentos                 []domain.DocumentoSeguimiento
}

func (d DatosConfirmacionIncorporacion) Validar() error {
	_, err := normalizarDatosConfirmacionIncorporacion(d)
	return err
}

type OrdenConfirmarIncorporacion struct {
	datos *EvidenciaOrdenConfirmarIncorporacion
}

type EvidenciaOrdenConfirmarIncorporacion struct {
	Contexto     ContextoConfirmacionIncorporacion
	Confirmacion DatosConfirmacionIncorporacion
	EvaluadaEn   time.Time
}

func NuevaOrdenConfirmarIncorporacion(
	contexto ContextoConfirmacionIncorporacion,
	datos DatosConfirmacionIncorporacion,
	evaluadaEn time.Time,
) (OrdenConfirmarIncorporacion, error) {
	normalizados, err := normalizarDatosConfirmacionIncorporacion(datos)
	if err != nil || contexto.ValidarPara(normalizados, evaluadaEn) != nil {
		return OrdenConfirmarIncorporacion{}, ErrOrdenConfirmacionIncorporacionInvalida
	}
	return OrdenConfirmarIncorporacion{datos: &EvidenciaOrdenConfirmarIncorporacion{
		Contexto: contexto, Confirmacion: normalizados, EvaluadaEn: evaluadaEn,
	}}, nil
}

func (o OrdenConfirmarIncorporacion) Datos() (
	EvidenciaOrdenConfirmarIncorporacion,
	error,
) {
	if o.datos == nil {
		return EvidenciaOrdenConfirmarIncorporacion{}, ErrOrdenConfirmacionIncorporacionInvalida
	}
	normalizados, err := normalizarDatosConfirmacionIncorporacion(o.datos.Confirmacion)
	if err != nil || o.datos.Contexto.ValidarPara(normalizados, o.datos.EvaluadaEn) != nil {
		return EvidenciaOrdenConfirmarIncorporacion{}, ErrOrdenConfirmacionIncorporacionInvalida
	}
	return EvidenciaOrdenConfirmarIncorporacion{
		Contexto: o.datos.Contexto, Confirmacion: normalizados, EvaluadaEn: o.datos.EvaluadaEn,
	}, nil
}

// ValidarDentroDeTransaccion obliga a revalidar contexto y vigencia V3 con el
// instante autoritativo de la transaccion que consumira la concesion.
func (o OrdenConfirmarIncorporacion) ValidarDentroDeTransaccion(instante time.Time) error {
	evidencia, err := o.Datos()
	if err != nil || evidencia.Contexto.ValidarPara(evidencia.Confirmacion, instante) != nil {
		return ErrOrdenConfirmacionIncorporacionInvalida
	}
	return nil
}

// ReferenciasDurablesConfirmacionIncorporacion son referencias locales
// estables que la transaccion resuelve para el resultado exacto de Personal.
type ReferenciasDurablesConfirmacionIncorporacion struct {
	ActuacionRef   string
	ReciboRef      string
	CorrelacionRef string
	ActorRef       string
}

func (r ReferenciasDurablesConfirmacionIncorporacion) validar() error {
	if !domain.ReferenciaOpacaValida(r.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.CorrelacionRef) ||
		!domain.ReferenciaOpacaValida(r.ActorRef) {
		return ErrOrdenConfirmacionIncorporacionInvalida
	}
	return nil
}

// DatosTransicionSeguimiento construye la unica transicion permitida. El
// adaptador aporta instante y referencias despues de resolverlos en la misma
// transaccion durable.
func (o OrdenConfirmarIncorporacion) DatosTransicionSeguimiento(
	confirmadaEn time.Time,
	referencias ReferenciasDurablesConfirmacionIncorporacion,
) (domain.DatosTransicionSeguimiento, error) {
	evidencia, err := o.Datos()
	solicitudV3, errSolicitud := evidencia.Contexto.SolicitudAutorizacionV3.Datos()
	if err != nil || errSolicitud != nil ||
		o.ValidarDentroDeTransaccion(confirmadaEn) != nil || referencias.validar() != nil {
		return domain.DatosTransicionSeguimiento{}, ErrOrdenConfirmacionIncorporacionInvalida
	}
	datos := evidencia.Confirmacion
	periodo := datos.PeriodoIncorporacion
	return domain.DatosTransicionSeguimiento{
		ActuacionRef: referencias.ActuacionRef, TransicionClave: TransicionConfirmarIncorporacion,
		MotivoClave: datos.MotivoClave, ActorRef: referencias.ActorRef,
		UnidadRef:  solicitudV3.Recurso.Ambitos[ambitoUnidadConfirmacionIncorporacion],
		EfectivoEn: periodo.Desde, RegistradaEn: confirmadaEn,
		Documentos: append([]domain.DocumentoSeguimiento(nil), datos.Documentos...),
		Periodo:    &periodo, ReciboRef: referencias.ReciboRef,
		CorrelacionRef: referencias.CorrelacionRef,
	}, nil
}

type ReciboConfirmacionIncorporacion struct {
	ReciboRef               string
	ActuacionRef            string
	CorrelacionRef          string
	ActorRef                string
	ExpedienteRef           string
	DecisionAutorizacionRef string
	ResultadoPersonalRef    string
	ReciboPersonalRef       string
	RelacionRef             string
	OcupacionRef            string
	TransicionClave         domain.ClaveCatalogo
	VersionAnterior         uint64
	VersionResultante       uint64
	FechaIncorporacion      time.Time
	ConfirmadaEn            time.Time
	Documentos              []domain.DocumentoSeguimiento
}

func (r ReciboConfirmacionIncorporacion) ValidarPara(
	orden OrdenConfirmarIncorporacion,
) error {
	evidencia, err := orden.Datos()
	confirmacionV3, errConfirmacion := evidencia.Contexto.ConfirmacionRegistroV3.Datos()
	if err != nil || errConfirmacion != nil ||
		orden.ValidarDentroDeTransaccion(r.ConfirmadaEn) != nil ||
		r.ConfirmadaEn.Before(evidencia.EvaluadaEn) {
		return ErrReciboConfirmacionIncorporacionInvalido
	}
	datos := evidencia.Confirmacion
	resultado := datos.ResultadoPersonal
	if (ReferenciasDurablesConfirmacionIncorporacion{
		ActuacionRef: r.ActuacionRef, ReciboRef: r.ReciboRef,
		CorrelacionRef: r.CorrelacionRef, ActorRef: r.ActorRef,
	}).validar() != nil || r.ExpedienteRef != datos.SolicitudPersonal.ExpedienteRef ||
		r.DecisionAutorizacionRef != confirmacionV3.DecisionRef ||
		r.ResultadoPersonalRef != resultado.ResultadoRef ||
		r.ReciboPersonalRef != resultado.ReciboRef ||
		r.RelacionRef != resultado.RelacionRef || r.OcupacionRef != resultado.OcupacionRef ||
		r.TransicionClave != TransicionConfirmarIncorporacion ||
		r.VersionAnterior != datos.VersionSeguimientoEsperada ||
		r.VersionResultante != r.VersionAnterior+1 ||
		!r.FechaIncorporacion.Equal(datos.PeriodoIncorporacion.Desde) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) ||
		!slices.Equal(r.Documentos, datos.Documentos) {
		return ErrReciboConfirmacionIncorporacionInvalido
	}
	return nil
}

// TransaccionConfirmacionIncorporacion es la unica frontera de efecto. En un
// commit bloquea expediente y seguimiento, comprueba versiones y relacion,
// revalida la orden con su reloj autoritativo, coteja y consume DecisionRef y
// huella de ConfirmacionRegistroV3, aplica confirmar_incorporacion y anexa
// resultado, documentos, auditoria, recibo y outbox. Resuelve las mismas
// referencias locales —incluida la proyeccion del principal V3— en replay
// exacto; divergencia o rollback devuelven error
// y recibo cero.
type TransaccionConfirmacionIncorporacion interface {
	ConfirmarIncorporacion(
		context.Context,
		OrdenConfirmarIncorporacion,
	) (ReciboConfirmacionIncorporacion, error)
}

func normalizarDatosConfirmacionIncorporacion(
	d DatosConfirmacionIncorporacion,
) (DatosConfirmacionIncorporacion, error) {
	if d.SolicitudPersonal.Validar() != nil ||
		d.ResultadoPersonal.ValidarPara(d.SolicitudPersonal) != nil ||
		d.ResultadoPersonal.Estado != AltaPersonalRPTConfirmada ||
		d.VersionSeguimientoEsperada >= MaximoEnteroSeguroOperacionAnalisis ||
		d.PeriodoIncorporacion.Validar() != nil || !d.MotivoClave.Valida() ||
		len(d.Documentos) == 0 || len(d.Documentos) > maximoDocumentosIncorporacion {
		return DatosConfirmacionIncorporacion{}, ErrOrdenConfirmacionIncorporacionInvalida
	}
	normalizados := d
	normalizados.Documentos = append([]domain.DocumentoSeguimiento(nil), d.Documentos...)
	sort.Slice(normalizados.Documentos, func(i, j int) bool {
		if normalizados.Documentos[i].TipoClave == normalizados.Documentos[j].TipoClave {
			return normalizados.Documentos[i].Referencia < normalizados.Documentos[j].Referencia
		}
		return normalizados.Documentos[i].TipoClave < normalizados.Documentos[j].TipoClave
	})
	referencias := make(map[string]struct{}, len(normalizados.Documentos))
	for _, documento := range normalizados.Documentos {
		if !documento.TipoClave.Valida() || !domain.ReferenciaOpacaValida(documento.Referencia) {
			return DatosConfirmacionIncorporacion{}, ErrOrdenConfirmacionIncorporacionInvalida
		}
		if _, repetida := referencias[documento.Referencia]; repetida {
			return DatosConfirmacionIncorporacion{}, ErrOrdenConfirmacionIncorporacionInvalida
		}
		referencias[documento.Referencia] = struct{}{}
	}
	return normalizados, nil
}
