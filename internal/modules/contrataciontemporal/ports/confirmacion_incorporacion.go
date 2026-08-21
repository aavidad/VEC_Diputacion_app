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
	AccionConfirmarIncorporacion                                                 = "contratacion_temporal.incorporacion.confirmar"
	FinalidadConfirmarIncorporacion                                              = "registrar_incorporacion_confirmada_por_personal"
	TipoRecursoConfirmacionIncorporacion                                         = "confirmacion_incorporacion_contratacion_temporal"
	TransicionConfirmarIncorporacion                        domain.ClaveCatalogo = "confirmar_incorporacion"
	ambitoOrganizacionConfirmacionIncorporacion                                  = "organizacion_ref"
	ambitoUnidadConfirmacionIncorporacion                                        = "unidad_ref"
	atributoResultadoConfirmacionIncorporacion                                   = "resultado_personal_ref"
	atributoRelacionConfirmacionIncorporacion                                    = "relacion_ref"
	atributoOcupacionConfirmacionIncorporacion                                   = "ocupacion_ref"
	atributoVersionExpedienteConfirmacionIncorporacion                           = "version_expediente_esperada"
	atributoVersionSeguimientoConfirmacionIncorporacion                          = "version_seguimiento_esperada"
	atributoPrincipalV3ConfirmacionIncorporacion                                 = "principal_v3_ref"
	atributoActorSeguimientoConfirmacionIncorporacion                            = "actor_seguimiento_ref"
	atributoCorrelacionV3ConfirmacionIncorporacion                               = "correlacion_v3_ref"
	atributoCorrelacionSeguimientoConfirmacionIncorporacion                      = "correlacion_seguimiento_ref"
	atributoMotivoV3ConfirmacionIncorporacion                                    = "motivo_v3_ref"
	maximoDocumentosIncorporacion                                                = 32
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
	PreparacionSeguimiento  PreparacionSeguimientoConfirmacionIncorporacion
	SolicitudAutorizacionV3 dominiovec.SolicitudAutorizacionLigadaV3
	DecisionAutorizacionV3  dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionRegistroV3  puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

// PreparacionSeguimientoConfirmacionIncorporacion conserva las referencias
// locales resueltas antes de solicitar la autorizacion V3. No traduce ni
// deriva el principal o la correlacion V3: el recurso compromete ambos pares.
type PreparacionSeguimientoConfirmacionIncorporacion struct {
	OrganizacionRef string
	UnidadRef       string
	ActorRef        string
	CorrelacionRef  string
}

func (p PreparacionSeguimientoConfirmacionIncorporacion) validar() error {
	if !domain.ReferenciaOpacaValida(p.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(p.UnidadRef) ||
		!domain.ReferenciaOpacaValida(p.ActorRef) ||
		!domain.ReferenciaOpacaValida(p.CorrelacionRef) {
		return ErrContextoConfirmacionIncorporacionInvalido
	}
	return nil
}

func (c ContextoConfirmacionIncorporacion) ValidarPara(
	datos DatosConfirmacionIncorporacion,
	instante time.Time,
) error {
	normalizados, err := normalizarDatosConfirmacionIncorporacion(datos)
	if err != nil || !domain.InstanteUTCCanonico(instante) ||
		c.PreparacionSeguimiento.validar() != nil ||
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
		recurso.Referencia != normalizados.SolicitudPersonal.ExpedienteRef ||
		recurso.ModuloID != ModuloContratacion ||
		recurso.Tipo != TipoRecursoConfirmacionIncorporacion ||
		!recursoConfirmacionIncorporacionExacto(
			recurso,
			normalizados,
			c.PreparacionSeguimiento,
			vinculo.PrincipalID,
			correlacion,
			solicitudV3.ReferenciaMotivo.EntradaClave,
		) ||
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
	preparacion PreparacionSeguimientoConfirmacionIncorporacion,
	principalV3Ref string,
	correlacionV3Ref string,
	motivoV3Ref string,
) bool {
	return len(recurso.Ambitos) == 2 && len(recurso.Atributos) == 10 &&
		recurso.Ambitos[ambitoOrganizacionConfirmacionIncorporacion] == preparacion.OrganizacionRef &&
		recurso.Ambitos[ambitoUnidadConfirmacionIncorporacion] == preparacion.UnidadRef &&
		recurso.Atributos[atributoResultadoConfirmacionIncorporacion] == datos.ResultadoPersonal.ResultadoRef &&
		recurso.Atributos[atributoRelacionConfirmacionIncorporacion] == datos.ResultadoPersonal.RelacionRef &&
		recurso.Atributos[atributoOcupacionConfirmacionIncorporacion] == datos.ResultadoPersonal.OcupacionRef &&
		recurso.Atributos[atributoVersionExpedienteConfirmacionIncorporacion] ==
			strconv.FormatUint(datos.SolicitudPersonal.VersionExpediente, 10) &&
		recurso.Atributos[atributoVersionSeguimientoConfirmacionIncorporacion] ==
			strconv.FormatUint(datos.VersionSeguimientoEsperada, 10) &&
		recurso.Atributos[atributoPrincipalV3ConfirmacionIncorporacion] == principalV3Ref &&
		recurso.Atributos[atributoActorSeguimientoConfirmacionIncorporacion] == preparacion.ActorRef &&
		recurso.Atributos[atributoCorrelacionV3ConfirmacionIncorporacion] == correlacionV3Ref &&
		recurso.Atributos[atributoCorrelacionSeguimientoConfirmacionIncorporacion] == preparacion.CorrelacionRef &&
		recurso.Atributos[atributoMotivoV3ConfirmacionIncorporacion] == motivoV3Ref
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

// ReferenciasDurablesConfirmacionIncorporacion son las referencias locales de
// actuacion y recibo que la transaccion resuelve. Actor y correlacion ya vienen
// fijados por la preparacion comprometida en la autorizacion V3.
type ReferenciasDurablesConfirmacionIncorporacion struct {
	ActuacionRef string
	ReciboRef    string
}

func (r ReferenciasDurablesConfirmacionIncorporacion) validar() error {
	if !domain.ReferenciaOpacaValida(r.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) {
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
	if err != nil || o.ValidarDentroDeTransaccion(confirmadaEn) != nil ||
		referencias.validar() != nil {
		return domain.DatosTransicionSeguimiento{}, ErrOrdenConfirmacionIncorporacionInvalida
	}
	datos := evidencia.Confirmacion
	periodo := datos.PeriodoIncorporacion
	preparacion := evidencia.Contexto.PreparacionSeguimiento
	return domain.DatosTransicionSeguimiento{
		ActuacionRef: referencias.ActuacionRef, TransicionClave: TransicionConfirmarIncorporacion,
		MotivoClave: datos.MotivoClave, ActorRef: preparacion.ActorRef,
		UnidadRef:  preparacion.UnidadRef,
		EfectivoEn: periodo.Desde, RegistradaEn: confirmadaEn,
		Documentos: append([]domain.DocumentoSeguimiento(nil), datos.Documentos...),
		Periodo:    &periodo, ReciboRef: referencias.ReciboRef,
		CorrelacionRef: preparacion.CorrelacionRef,
	}, nil
}

type ReciboConfirmacionIncorporacion struct {
	ReciboRef               string
	ActuacionRef            string
	CorrelacionRef          string
	ActorRef                string
	OrganizacionRef         string
	UnidadRef               string
	ExpedienteRef           string
	SolicitudPersonalRef    string
	DecisionAutorizacionRef string
	ResultadoPersonalRef    string
	ReciboPersonalRef       string
	RelacionRef             string
	OcupacionRef            string
	TransicionClave         domain.ClaveCatalogo
	MotivoClave             domain.ClaveCatalogo
	VersionExpediente       uint64
	VersionAnterior         uint64
	VersionResultante       uint64
	FechaIncorporacion      time.Time
	FechaFinPrevista        time.Time
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
	preparacion := evidencia.Contexto.PreparacionSeguimiento
	if (ReferenciasDurablesConfirmacionIncorporacion{
		ActuacionRef: r.ActuacionRef, ReciboRef: r.ReciboRef,
	}).validar() != nil || r.ExpedienteRef != datos.SolicitudPersonal.ExpedienteRef ||
		r.SolicitudPersonalRef != datos.SolicitudPersonal.SolicitudRef ||
		r.OrganizacionRef != preparacion.OrganizacionRef ||
		r.UnidadRef != preparacion.UnidadRef ||
		r.ActorRef != preparacion.ActorRef ||
		r.CorrelacionRef != preparacion.CorrelacionRef ||
		r.DecisionAutorizacionRef != confirmacionV3.DecisionRef ||
		r.ResultadoPersonalRef != resultado.ResultadoRef ||
		r.ReciboPersonalRef != resultado.ReciboRef ||
		r.RelacionRef != resultado.RelacionRef || r.OcupacionRef != resultado.OcupacionRef ||
		r.TransicionClave != TransicionConfirmarIncorporacion ||
		r.MotivoClave != datos.MotivoClave ||
		r.VersionExpediente != datos.SolicitudPersonal.VersionExpediente ||
		r.VersionAnterior != datos.VersionSeguimientoEsperada ||
		r.VersionResultante != r.VersionAnterior+1 ||
		!r.FechaIncorporacion.Equal(datos.PeriodoIncorporacion.Desde) ||
		!r.FechaFinPrevista.Equal(datos.PeriodoIncorporacion.Hasta) ||
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
// resultado, documentos, auditoria, recibo y outbox. Reutiliza actor y
// correlacion locales ya ligados en el recurso, y resuelve actuacion y recibo
// en replay exacto; divergencia o rollback devuelven error y recibo cero.
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
