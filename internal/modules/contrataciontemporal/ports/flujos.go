package ports

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionRegistrarAsignacion   = "contratacion_temporal.unidad.asignar"
	AccionRegistrarReasignacion = "contratacion_temporal.unidad.reasignar"
	TipoRecursoAsignacion       = "asignacion_contratacion_temporal"
)

var (
	ErrFlujoNoDisponible = errors.New(
		"contratacion temporal: flujo no disponible",
	)
	ErrDestinoAsignacionNoDisponible = errors.New(
		"contratacion temporal: destino de asignacion no disponible",
	)
	ErrPoliticaAsignacionNoDisponible = errors.New(
		"contratacion temporal: politica de asignacion no disponible",
	)
	ErrSegregacionAsignacionIncumplida = errors.New(
		"contratacion temporal: segregacion de asignacion incumplida",
	)
	ErrOrdenAsignacionInvalida = errors.New(
		"contratacion temporal: orden de asignacion invalida",
	)
	ErrPersistenciaAsignacionNoDisponible = errors.New(
		"contratacion temporal: persistencia de asignacion no disponible",
	)
)

type SolicitudResolverFlujo struct {
	OrganizacionRef string
	CentroRef       string
	CategoriaRef    string
	MotivoClave     domain.ClaveCatalogo
	Instante        time.Time
}

func (s SolicitudResolverFlujo) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.CentroRef) ||
		!domain.ReferenciaOpacaValida(s.CategoriaRef) ||
		!s.MotivoClave.Valida() || !domain.InstanteUTCCanonico(s.Instante) {
		return ErrFlujoNoDisponible
	}
	return nil
}

type ConfiguracionAltaFlujo struct {
	Flujo            domain.ReferenciaFlujo
	FaseInicial      domain.ClaveFase
	UnidadInicialRef string
	AccionInicial    domain.ClaveCatalogo
}

func (c ConfiguracionAltaFlujo) Validar() error {
	if c.Flujo.Validar() != nil || !c.FaseInicial.Valida() ||
		!domain.ReferenciaOpacaValida(c.UnidadInicialRef) ||
		!c.AccionInicial.Valida() {
		return ErrFlujoNoDisponible
	}
	return nil
}

type ResolutorFlujoAlta interface {
	ResolverFlujoAlta(context.Context, SolicitudResolverFlujo) (ConfiguracionAltaFlujo, error)
}

// SolicitudResolverDestinoAsignacion identifica las coordenadas mínimas que
// debe comprobar una fuente organizativa autoritativa antes de asignar un
// expediente. ActorRef procede del contexto de autenticación resuelto por el
// servidor; nunca se acepta directamente del cliente.
type SolicitudResolverDestinoAsignacion struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	ActorRef          string
	UnidadRef         string
	ResponsableRef    string
	Instante          time.Time
}

func (s SolicitudResolverDestinoAsignacion) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.UnidadRef) ||
		!domain.ReferenciaOpacaValida(s.ResponsableRef) ||
		!domain.InstanteUTCCanonico(s.Instante) {
		return ErrDestinoAsignacionNoDisponible
	}
	return nil
}

// DestinoAsignacionResuelto acredita una comprobación afirmativa de unidad y
// responsable. No contiene nombres, DNI, correo ni otros datos personales.
// Las referencias y la evidencia quedan ligadas a la solicitud exacta para
// impedir reutilizarlas en otro expediente o versión.
type DestinoAsignacionResuelto struct {
	OrganizacionRef        string
	ExpedienteRef          string
	VersionExpediente      uint64
	ActorRef               string
	UnidadRef              string
	ResponsableRef         string
	DefinicionRef          string
	DefinicionVersion      uint64
	DefinicionHuellaSHA256 string
	EvidenciaRef           string
	EvidenciaHuellaSHA256  string
	EvaluadoEn             time.Time
	ValidoHasta            time.Time
}

func (d DestinoAsignacionResuelto) ValidarPara(
	solicitud SolicitudResolverDestinoAsignacion,
	instanteUso time.Time,
) error {
	if solicitud.Validar() != nil ||
		d.OrganizacionRef != solicitud.OrganizacionRef ||
		d.ExpedienteRef != solicitud.ExpedienteRef ||
		d.VersionExpediente != solicitud.VersionExpediente ||
		d.ActorRef != solicitud.ActorRef ||
		d.UnidadRef != solicitud.UnidadRef ||
		d.ResponsableRef != solicitud.ResponsableRef ||
		!domain.ReferenciaOpacaValida(d.DefinicionRef) ||
		!VersionOperacionAnalisisValida(d.DefinicionVersion) ||
		!huellaSHA256OperacionAnalisisValida(d.DefinicionHuellaSHA256) ||
		!domain.ReferenciaOpacaValida(d.EvidenciaRef) ||
		!huellaSHA256OperacionAnalisisValida(d.EvidenciaHuellaSHA256) ||
		!domain.InstanteUTCCanonico(d.EvaluadoEn) ||
		!domain.InstanteUTCCanonico(d.ValidoHasta) ||
		!domain.InstanteUTCCanonico(instanteUso) ||
		!d.EvaluadoEn.Equal(solicitud.Instante) ||
		!d.ValidoHasta.After(d.EvaluadoEn) ||
		instanteUso.Before(d.EvaluadoEn) ||
		instanteUso.After(d.ValidoHasta) {
		return ErrDestinoAsignacionNoDisponible
	}
	return nil
}

// ResolutorDestinoAsignacion es un puerto de consulta. La implementación debe
// cruzar una fuente organizativa publicada con el directorio autoritativo y
// devolver error si la unidad no está activa o si la persona responsable no
// está activa, adscrita y habilitada para la tarea.
type ResolutorDestinoAsignacion interface {
	ResolverDestinoAsignacion(
		context.Context,
		SolicitudResolverDestinoAsignacion,
	) (DestinoAsignacionResuelto, error)
}

type TipoOperacionAsignacion string

const (
	OperacionRegistrarAsignacion   TipoOperacionAsignacion = "asignar"
	OperacionRegistrarReasignacion TipoOperacionAsignacion = "reasignar"
)

func (t TipoOperacionAsignacion) Valida() bool {
	return t == OperacionRegistrarAsignacion ||
		t == OperacionRegistrarReasignacion
}

type MotivoReasignacionGobernado struct {
	ReferenciaCatalogo dominiovec.ReferenciaEntradaCatalogo
	ClaveMensajeI18N   domain.ClaveCatalogo
}

func (m MotivoReasignacionGobernado) ValidarPara(
	clave domain.ClaveCatalogo,
) error {
	if !clave.Valida() || m.ReferenciaCatalogo.Validar() != nil ||
		uint64(m.ReferenciaCatalogo.CatalogoVersion) >
			MaximoEnteroSeguroOperacionAnalisis ||
		m.ReferenciaCatalogo.EntradaClave != string(clave) ||
		!m.ClaveMensajeI18N.Valida() ||
		!strings.HasPrefix(
			string(m.ClaveMensajeI18N),
			"contratacion_temporal.asignacion.motivo.",
		) {
		return ErrPoliticaAsignacionNoDisponible
	}
	return nil
}

type SolicitudResolverPoliticaAsignacion struct {
	Operacion               TipoOperacionAsignacion
	OrganizacionRef         string
	ExpedienteRef           string
	VersionExpediente       uint64
	Flujo                   domain.ReferenciaFlujo
	FasePrevia              domain.ClaveFase
	EstadoPrevio            domain.EstadoOperativo
	ActorRef                string
	PerfilRef               string
	UnidadAnteriorRef       string
	ResponsableAnteriorRef  string
	Destino                 DestinoAsignacionResuelto
	MotivoReasignacionClave domain.ClaveCatalogo
	Instante                time.Time
}

func (s SolicitudResolverPoliticaAsignacion) Validar() error {
	destinoSolicitado := SolicitudResolverDestinoAsignacion{
		OrganizacionRef:   s.OrganizacionRef,
		ExpedienteRef:     s.ExpedienteRef,
		VersionExpediente: s.VersionExpediente,
		ActorRef:          s.ActorRef,
		UnidadRef:         s.Destino.UnidadRef,
		ResponsableRef:    s.Destino.ResponsableRef,
		Instante:          s.Instante,
	}
	if !s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		s.Flujo.Validar() != nil || !s.FasePrevia.Valida() ||
		!s.EstadoPrevio.Valido() ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!domain.InstanteUTCCanonico(s.Instante) ||
		s.Destino.ValidarPara(destinoSolicitado, s.Instante) != nil {
		return ErrPoliticaAsignacionNoDisponible
	}
	if s.Operacion == OperacionRegistrarAsignacion {
		if s.UnidadAnteriorRef != "" ||
			s.ResponsableAnteriorRef != "" ||
			s.MotivoReasignacionClave != "" {
			return ErrPoliticaAsignacionNoDisponible
		}
		return nil
	}
	if !domain.ReferenciaOpacaValida(s.UnidadAnteriorRef) ||
		!domain.ReferenciaOpacaValida(s.ResponsableAnteriorRef) ||
		!s.MotivoReasignacionClave.Valida() ||
		(s.UnidadAnteriorRef == s.Destino.UnidadRef &&
			s.ResponsableAnteriorRef == s.Destino.ResponsableRef) {
		return ErrPoliticaAsignacionNoDisponible
	}
	return nil
}

type PoliticaAsignacion struct {
	Operacion                     TipoOperacionAsignacion
	OrganizacionRef               string
	ExpedienteRef                 string
	VersionExpediente             uint64
	ActorRef                      string
	PerfilRef                     string
	DestinoEvidenciaRef           string
	DestinoEvidenciaHuellaSHA256  string
	DefinicionRef                 string
	DefinicionVersion             uint64
	DefinicionHuellaSHA256        string
	Accion                        domain.ClaveCatalogo
	Finalidad                     domain.ClaveCatalogo
	UnidadEjecutoraRef            string
	MotivoAutorizacion            dominiovec.ReferenciaEntradaCatalogo
	ExigeActorDistintoResponsable bool
	MotivoReasignacion            MotivoReasignacionGobernado
	EvaluadaEn                    time.Time
	ValidaHasta                   time.Time
}

func (p PoliticaAsignacion) ValidarPara(
	solicitud SolicitudResolverPoliticaAsignacion,
	instanteUso time.Time,
) error {
	if solicitud.Validar() != nil ||
		p.Operacion != solicitud.Operacion ||
		p.OrganizacionRef != solicitud.OrganizacionRef ||
		p.ExpedienteRef != solicitud.ExpedienteRef ||
		p.VersionExpediente != solicitud.VersionExpediente ||
		p.ActorRef != solicitud.ActorRef ||
		p.PerfilRef != solicitud.PerfilRef ||
		p.DestinoEvidenciaRef != solicitud.Destino.EvidenciaRef ||
		p.DestinoEvidenciaHuellaSHA256 !=
			solicitud.Destino.EvidenciaHuellaSHA256 ||
		!domain.ReferenciaOpacaValida(p.DefinicionRef) ||
		!VersionOperacionAnalisisValida(p.DefinicionVersion) ||
		!huellaSHA256OperacionAnalisisValida(p.DefinicionHuellaSHA256) ||
		!p.Accion.Valida() || !p.Finalidad.Valida() ||
		!domain.ReferenciaOpacaValida(p.UnidadEjecutoraRef) ||
		p.MotivoAutorizacion.Validar() != nil ||
		uint64(p.MotivoAutorizacion.CatalogoVersion) >
			MaximoEnteroSeguroOperacionAnalisis ||
		!domain.InstanteUTCCanonico(p.EvaluadaEn) ||
		!domain.InstanteUTCCanonico(p.ValidaHasta) ||
		!domain.InstanteUTCCanonico(instanteUso) ||
		!p.EvaluadaEn.Equal(solicitud.Instante) ||
		!p.ValidaHasta.After(p.EvaluadaEn) ||
		instanteUso.Before(p.EvaluadaEn) ||
		instanteUso.After(p.ValidaHasta) ||
		(p.ExigeActorDistintoResponsable &&
			p.ActorRef == solicitud.Destino.ResponsableRef) {
		return ErrPoliticaAsignacionNoDisponible
	}
	if p.Operacion == OperacionRegistrarAsignacion {
		if p.Accion != domain.ClaveCatalogo(AccionRegistrarAsignacion) ||
			p.MotivoReasignacion != (MotivoReasignacionGobernado{}) {
			return ErrPoliticaAsignacionNoDisponible
		}
		return nil
	}
	if p.Accion != domain.ClaveCatalogo(AccionRegistrarReasignacion) ||
		p.MotivoReasignacion.ValidarPara(
			solicitud.MotivoReasignacionClave,
		) != nil {
		return ErrPoliticaAsignacionNoDisponible
	}
	return nil
}

type ResolutorPoliticaAsignacion interface {
	ResolverPoliticaAsignacion(
		context.Context,
		SolicitudResolverPoliticaAsignacion,
	) (PoliticaAsignacion, error)
}

const (
	AtributoOperacionAsignacion       = "operacion"
	AtributoVersionAsignacion         = "version_expediente_esperada"
	AtributoPoliticaAsignacionRef     = "politica_ref"
	AtributoPoliticaAsignacionVersion = "politica_version"
	AtributoPoliticaAsignacionHuella  = "politica_huella_sha256"
	AtributoEvidenciaDestinoRef       = "evidencia_destino_ref"
	AtributoEvidenciaDestinoHuella    = "evidencia_destino_huella_sha256"
	AtributoUnidadDestino             = "unidad_destino_ref"
	AtributoResponsableDestino        = "responsable_destino_ref"
	AtributoHuellaPeticionAsignacion  = "huella_peticion_hmac"
	AtributoSegregacionAsignacion     = "exige_actor_distinto_responsable"
)

type DatosOrdenConfirmarAsignacion struct {
	SolicitudContexto    SolicitudResolverContextoAutorizacionAltaV3
	ContextoAutorizacion ContextoAutorizacionAltaV3
	Material             MaterialHuellaAsignacion
	SolicitudPreparacion SolicitudPrepararAsignacion
	Preparacion          PreparacionAsignacion
	SolicitudDestino     SolicitudResolverDestinoAsignacion
	Destino              DestinoAsignacionResuelto
	SolicitudPolitica    SolicitudResolverPoliticaAsignacion
	Politica             PoliticaAsignacion
	SolicitudV3          dominiovec.SolicitudAutorizacionLigadaV3
	DecisionV3           dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionV3       puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	InstanteEfecto       time.Time
	ExpedienteSiguiente  domain.Expediente
}

type OrdenConfirmarAsignacion struct {
	datos *datosOrdenConfirmarAsignacion
}

type datosOrdenConfirmarAsignacion struct {
	entrada             DatosOrdenConfirmarAsignacion
	expedienteAnterior  domain.Expediente
	expedienteSiguiente domain.Expediente
}

type EvidenciaOrdenConfirmarAsignacion struct {
	SolicitudContexto    SolicitudResolverContextoAutorizacionAltaV3
	ContextoAutorizacion ContextoAutorizacionAltaV3
	Material             MaterialHuellaAsignacion
	SolicitudPreparacion SolicitudPrepararAsignacion
	Preparacion          PreparacionAsignacion
	SolicitudDestino     SolicitudResolverDestinoAsignacion
	Destino              DestinoAsignacionResuelto
	SolicitudPolitica    SolicitudResolverPoliticaAsignacion
	Politica             PoliticaAsignacion
	SolicitudV3          dominiovec.SolicitudAutorizacionLigadaV3
	DecisionV3           dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionV3       puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	InstanteEfecto       time.Time
	ExpedienteAnterior   domain.Expediente
	ExpedienteSiguiente  domain.Expediente
}

func NuevaOrdenConfirmarAsignacion(
	datos DatosOrdenConfirmarAsignacion,
) (OrdenConfirmarAsignacion, error) {
	if validarOrdenConfirmarAsignacion(datos) != nil {
		return OrdenConfirmarAsignacion{}, ErrOrdenAsignacionInvalida
	}
	copia := datos
	copia.Preparacion.Expediente = datos.Preparacion.Expediente.Clonar()
	copia.ExpedienteSiguiente = datos.ExpedienteSiguiente.Clonar()
	return OrdenConfirmarAsignacion{
		datos: &datosOrdenConfirmarAsignacion{
			entrada:             copia,
			expedienteAnterior:  datos.Preparacion.Expediente.Clonar(),
			expedienteSiguiente: datos.ExpedienteSiguiente.Clonar(),
		},
	}, nil
}

func validarOrdenConfirmarAsignacion(
	datos DatosOrdenConfirmarAsignacion,
) error {
	anterior := datos.Preparacion.Expediente
	if datos.Material.Validar() != nil ||
		datos.SolicitudPreparacion.Validar() != nil ||
		datos.Preparacion.ValidarPara(datos.SolicitudPreparacion) != nil ||
		datos.Preparacion.Estado != PreparacionAsignacionReservada ||
		datos.Preparacion.ReciboConfirmado != nil ||
		anterior.Validar() != nil ||
		!VersionOperacionAnalisisConIncrementoValida(anterior.Version) ||
		!domain.InstanteUTCCanonico(datos.InstanteEfecto) ||
		datos.InstanteEfecto.Before(anterior.ActualizadoEn) ||
		datos.ContextoAutorizacion.ValidarPara(
			datos.SolicitudContexto,
			datos.InstanteEfecto,
		) != nil ||
		datos.Destino.ValidarPara(
			datos.SolicitudDestino,
			datos.InstanteEfecto,
		) != nil ||
		datos.Politica.ValidarPara(
			datos.SolicitudPolitica,
			datos.InstanteEfecto,
		) != nil ||
		!coincidenCoordenadasOrdenAsignacion(datos, anterior) ||
		validarAutorizacionOrdenAsignacion(datos) != nil {
		return ErrOrdenAsignacionInvalida
	}
	esperado, err := reproducirAsignacion(datos, anterior)
	if err != nil || esperado.Validar() != nil ||
		!reflect.DeepEqual(esperado, datos.ExpedienteSiguiente) {
		return ErrOrdenAsignacionInvalida
	}
	return nil
}

func coincidenCoordenadasOrdenAsignacion(
	datos DatosOrdenConfirmarAsignacion,
	anterior domain.Expediente,
) bool {
	material := datos.Material
	preparacion := datos.SolicitudPreparacion
	destino := datos.SolicitudDestino
	politica := datos.SolicitudPolitica
	return material.Operacion == preparacion.Operacion &&
		material.OrganizacionRef == preparacion.OrganizacionRef &&
		material.ExpedienteRef == preparacion.ExpedienteRef &&
		material.VersionExpediente == preparacion.VersionExpediente &&
		material.ActorRef == preparacion.ActorRef &&
		material.PerfilRef == preparacion.PerfilRef &&
		material.UnidadRef == preparacion.UnidadRef &&
		material.ResponsableRef == preparacion.ResponsableRef &&
		destino.OrganizacionRef == material.OrganizacionRef &&
		destino.ExpedienteRef == material.ExpedienteRef &&
		destino.VersionExpediente == material.VersionExpediente &&
		destino.ActorRef == material.ActorRef &&
		destino.UnidadRef == material.UnidadRef &&
		destino.ResponsableRef == material.ResponsableRef &&
		politica.Operacion == material.Operacion &&
		politica.OrganizacionRef == material.OrganizacionRef &&
		politica.ExpedienteRef == material.ExpedienteRef &&
		politica.VersionExpediente == material.VersionExpediente &&
		politica.Flujo == anterior.Flujo &&
		politica.FasePrevia == anterior.FaseActual &&
		politica.EstadoPrevio == anterior.EstadoActual &&
		politica.ActorRef == material.ActorRef &&
		politica.PerfilRef == material.PerfilRef &&
		politica.Destino == datos.Destino &&
		politica.MotivoReasignacionClave ==
			material.MotivoReasignacionClave &&
		coincideAsignacionAnterior(politica, anterior)
}

func coincideAsignacionAnterior(
	solicitud SolicitudResolverPoliticaAsignacion,
	expediente domain.Expediente,
) bool {
	if solicitud.Operacion == OperacionRegistrarAsignacion {
		return expediente.Asignacion == nil &&
			solicitud.UnidadAnteriorRef == "" &&
			solicitud.ResponsableAnteriorRef == ""
	}
	return expediente.Asignacion != nil &&
		solicitud.UnidadAnteriorRef == expediente.Asignacion.UnidadRef &&
		solicitud.ResponsableAnteriorRef ==
			expediente.Asignacion.ResponsableRef
}

func reproducirAsignacion(
	datos DatosOrdenConfirmarAsignacion,
	anterior domain.Expediente,
) (domain.Expediente, error) {
	asignacion := domain.AsignacionUnidad{
		UnidadRef:       datos.Material.UnidadRef,
		ResponsableRef:  datos.Material.ResponsableRef,
		NotificacionRef: datos.Preparacion.Referencias.NotificacionRef,
		AsignadaEn:      datos.InstanteEfecto,
		Observaciones:   datos.Material.Observaciones,
	}
	actuacion := domain.DatosActuacion{
		AccionClave:   datos.Politica.Accion,
		ActorRef:      datos.Material.ActorRef,
		UnidadRef:     datos.Politica.UnidadEjecutoraRef,
		ReciboRef:     datos.Preparacion.Referencias.ReciboRef,
		RealizadaEn:   datos.InstanteEfecto,
		FaseDestino:   anterior.FaseActual,
		EstadoDestino: anterior.EstadoActual,
		Observaciones: datos.Material.Observaciones,
	}
	if datos.Material.Operacion == OperacionRegistrarAsignacion {
		return anterior.RegistrarAsignacion(
			datos.Material.VersionExpediente,
			asignacion,
			actuacion,
		)
	}
	asignacion.MotivoClave = datos.Material.MotivoReasignacionClave
	return anterior.ReasignarUnidad(
		datos.Material.VersionExpediente,
		asignacion,
		actuacion,
	)
}

func validarAutorizacionOrdenAsignacion(
	datos DatosOrdenConfirmarAsignacion,
) error {
	solicitudV3, err := datos.SolicitudV3.Datos()
	vinculo, errVinculo := solicitudV3.VinculoAutenticacionActor.Datos()
	vinculoContexto, errContexto :=
		datos.ContextoAutorizacion.Vinculo.Datos()
	concedida, _, errDecision := datos.DecisionV3.Resultado()
	huellaDecision, errHuella :=
		dominiovec.HuellaSHA256DecisionAutorizacionV3(datos.DecisionV3)
	confirmacion, errConfirmacion := datos.ConfirmacionV3.Datos()
	if err != nil || errVinculo != nil || errContexto != nil ||
		errDecision != nil || errHuella != nil ||
		errConfirmacion != nil || !concedida ||
		datos.DecisionV3.ValidarPara(datos.SolicitudV3) != nil ||
		!reflect.DeepEqual(vinculo, vinculoContexto) ||
		vinculo.PrincipalID != datos.Material.ActorRef ||
		vinculo.PerfilActivoRef != datos.Material.PerfilRef ||
		solicitudV3.ReferenciaMotivo !=
			datos.Politica.MotivoAutorizacion ||
		solicitudV3.Accion != string(datos.Politica.Accion) ||
		solicitudV3.Finalidad != string(datos.Politica.Finalidad) ||
		!recursoAutorizacionAsignacionValido(
			solicitudV3.Recurso,
			datos,
		) ||
		confirmacion.DecisionHuellaSHA256 != huellaDecision ||
		!datos.ConfirmacionV3.DentroDeVentanaEn(datos.InstanteEfecto) {
		return ErrOrdenAsignacionInvalida
	}
	return nil
}

func recursoAutorizacionAsignacionValido(
	recurso dominiovec.RecursoAutorizable,
	datos DatosOrdenConfirmarAsignacion,
) bool {
	return recurso.Referencia == datos.Material.ExpedienteRef &&
		recurso.ModuloID == ModuloContratacion &&
		recurso.Tipo == TipoRecursoAsignacion &&
		len(recurso.Ambitos) == 5 && len(recurso.Atributos) == 11 &&
		recurso.Ambitos["organizacion_ref"] ==
			datos.Material.OrganizacionRef &&
		recurso.Ambitos["expediente_ref"] ==
			datos.Material.ExpedienteRef &&
		recurso.Ambitos["fase_previa"] ==
			string(datos.SolicitudPolitica.FasePrevia) &&
		recurso.Ambitos["estado_previo"] ==
			string(datos.SolicitudPolitica.EstadoPrevio) &&
		recurso.Ambitos["unidad_destino_ref"] ==
			datos.Material.UnidadRef &&
		recurso.Atributos[AtributoOperacionAsignacion] ==
			string(datos.Material.Operacion) &&
		recurso.Atributos[AtributoVersionAsignacion] ==
			strconv.FormatUint(datos.Material.VersionExpediente, 10) &&
		recurso.Atributos[AtributoPoliticaAsignacionRef] ==
			datos.Politica.DefinicionRef &&
		recurso.Atributos[AtributoPoliticaAsignacionVersion] ==
			strconv.FormatUint(datos.Politica.DefinicionVersion, 10) &&
		recurso.Atributos[AtributoPoliticaAsignacionHuella] ==
			datos.Politica.DefinicionHuellaSHA256 &&
		recurso.Atributos[AtributoEvidenciaDestinoRef] ==
			datos.Destino.EvidenciaRef &&
		recurso.Atributos[AtributoEvidenciaDestinoHuella] ==
			datos.Destino.EvidenciaHuellaSHA256 &&
		recurso.Atributos[AtributoUnidadDestino] ==
			datos.Material.UnidadRef &&
		recurso.Atributos[AtributoResponsableDestino] ==
			datos.Material.ResponsableRef &&
		recurso.Atributos[AtributoHuellaPeticionAsignacion] ==
			datos.Preparacion.HuellaPeticionHMAC &&
		recurso.Atributos[AtributoSegregacionAsignacion] ==
			strconv.FormatBool(
				datos.Politica.ExigeActorDistintoResponsable,
			)
}

func (o OrdenConfirmarAsignacion) Datos() (
	EvidenciaOrdenConfirmarAsignacion,
	error,
) {
	if o.datos == nil {
		return EvidenciaOrdenConfirmarAsignacion{},
			ErrOrdenAsignacionInvalida
	}
	entrada := o.datos.entrada
	entrada.Preparacion.Expediente = o.datos.expedienteAnterior.Clonar()
	entrada.ExpedienteSiguiente = o.datos.expedienteSiguiente.Clonar()
	if validarOrdenConfirmarAsignacion(entrada) != nil {
		return EvidenciaOrdenConfirmarAsignacion{},
			ErrOrdenAsignacionInvalida
	}
	return EvidenciaOrdenConfirmarAsignacion{
		SolicitudContexto:    entrada.SolicitudContexto,
		ContextoAutorizacion: entrada.ContextoAutorizacion,
		Material:             entrada.Material,
		SolicitudPreparacion: entrada.SolicitudPreparacion,
		Preparacion:          entrada.Preparacion,
		SolicitudDestino:     entrada.SolicitudDestino,
		Destino:              entrada.Destino,
		SolicitudPolitica:    entrada.SolicitudPolitica,
		Politica:             entrada.Politica,
		SolicitudV3:          entrada.SolicitudV3,
		DecisionV3:           entrada.DecisionV3,
		ConfirmacionV3:       entrada.ConfirmacionV3,
		InstanteEfecto:       entrada.InstanteEfecto,
		ExpedienteAnterior:   o.datos.expedienteAnterior.Clonar(),
		ExpedienteSiguiente:  o.datos.expedienteSiguiente.Clonar(),
	}, nil
}

func (o OrdenConfirmarAsignacion) ValidarDentroDeTransaccion(
	confirmadaEn time.Time,
) error {
	evidencia, err := o.Datos()
	if err != nil || !domain.InstanteUTCCanonico(confirmadaEn) ||
		confirmadaEn.Before(evidencia.InstanteEfecto) ||
		evidencia.ContextoAutorizacion.ValidarPara(
			evidencia.SolicitudContexto,
			confirmadaEn,
		) != nil ||
		evidencia.Destino.ValidarPara(
			evidencia.SolicitudDestino,
			confirmadaEn,
		) != nil ||
		evidencia.Politica.ValidarPara(
			evidencia.SolicitudPolitica,
			confirmadaEn,
		) != nil ||
		!evidencia.ConfirmacionV3.DentroDeVentanaEn(confirmadaEn) {
		return ErrOrdenAsignacionInvalida
	}
	return nil
}

// TransaccionAsignaciones posee la única frontera de efectos. En un solo
// COMMIT debe consumir concesión e idempotencia, aplicar CAS, persistir
// agregado e historia append-only y crear bandeja, auditoría, recibo y outbox.
type TransaccionAsignaciones interface {
	ConfirmarAsignacion(
		context.Context,
		OrdenConfirmarAsignacion,
	) (ReciboAsignacion, error)
}
