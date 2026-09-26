package ports

import (
	"context"
	"errors"
	"slices"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Operaciones de seguimiento: cese (CT115), cierre del expediente (CT115) y
// modificación tras el nombramiento (CT116). Cada una tiene recurso,
// finalidad y audiencia propios (AD3-82 y AD3-83).
const (
	OperacionRegistrarCese             = "registrar_cese"
	OperacionCerrarExpediente          = "cerrar_expediente"
	OperacionModificarTrasNombramiento = "modificar_tras_nombramiento"

	TipoRecursoCese                     = "cese_contratacion_temporal"
	FinalidadRegistrarCese              = "registrar_cese_contratacion_temporal"
	TipoRecursoCierreExpediente         = "cierre_expediente_contratacion_temporal"
	FinalidadCerrarExpediente           = "cerrar_expediente_tras_cese"
	TipoRecursoModificacionNombramiento = "modificacion_contratacion_temporal"
	FinalidadModificarTrasNombramiento  = "modificar_expediente_tras_nombramiento"

	AudienciaConsumoCeseV1                     = "vec_contratacion_temporal.cese.v1"
	AudienciaConsumoCierreExpedienteV1         = "vec_contratacion_temporal.cierre_expediente.v1"
	AudienciaConsumoModificacionNombramientoV1 = "vec_contratacion_temporal.modificacion_tras_nombramiento.v1"

	DominioAmbitoIdempotenciaCese                 = "vec.contratacion-temporal.cese.ambito"
	DominioHuellaPeticionCese                     = "vec.contratacion-temporal.cese.peticion"
	DominioAmbitoIdempotenciaCierreExpediente     = "vec.contratacion-temporal.cierre-expediente.ambito"
	DominioHuellaPeticionCierreExpediente         = "vec.contratacion-temporal.cierre-expediente.peticion"
	DominioAmbitoIdempotenciaModificacionNombrado = "vec.contratacion-temporal.modificacion-nombramiento.ambito"
	DominioHuellaPeticionModificacionNombrado     = "vec.contratacion-temporal.modificacion-nombramiento.peticion"
	maximoOpcionesSeguimiento                     = 64
	vigenciaMaximaPoliticaOperacionSeguimiento    = 5 * time.Minute
	ResultadoPreparacionSeguimientoPreparada      = "preparada"
	ResultadoPreparacionSeguimientoConfirmada     = "confirmada"
)

var (
	ErrOperacionSeguimientoInvalida     = errors.New("contratacion temporal: operacion de seguimiento invalida")
	ErrOperacionSeguimientoNoDisponible = errors.New("contratacion temporal: operacion de seguimiento no disponible")
	ErrResultadoSeguimientoNoConfiable  = errors.New("contratacion temporal: resultado de seguimiento no confiable")
	// Rechazos de negocio: el estado del expediente o la regla no admiten la
	// operación. Ninguno escribe nada.
	ErrCeseSinIncorporacion            = errors.New("contratacion temporal: cese sin incorporacion acreditada")
	ErrCeseFechaAnteriorIncorporacion  = errors.New("contratacion temporal: fecha de cese anterior a la incorporacion")
	ErrCeseYaRegistrado                = errors.New("contratacion temporal: cese ya registrado")
	ErrCierreSinCese                   = errors.New("contratacion temporal: cierre sin cese registrado")
	ErrCierreYaRegistrado              = errors.New("contratacion temporal: cierre ya registrado")
	ErrModificacionSinCambios          = errors.New("contratacion temporal: modificacion sin cambios")
	ErrModificacionCreditoInsuficiente = errors.New("contratacion temporal: coste por encima de la retencion de credito")
)

// PoliticaOperacionSeguimiento es la definición gobernada que ampara la
// operación (catálogo de causas o regla del catálogo) y el motivo de
// autorización de la ruta. Su vigencia es breve y se fija al resolverla.
type PoliticaOperacionSeguimiento struct {
	DefinicionRef          string
	DefinicionVersion      uint64
	DefinicionHuellaSHA256 string
	MotivoAutorizacion     vd.ReferenciaEntradaCatalogo
	EvaluadaEn             time.Time
	ValidaHasta            time.Time
}

func (p PoliticaOperacionSeguimiento) ValidaEn(instante time.Time) bool {
	return domain.ReferenciaOpacaValida(p.DefinicionRef) && VersionOperacionAnalisisValida(p.DefinicionVersion) &&
		huellaSHA256OperacionAnalisisValida(p.DefinicionHuellaSHA256) && vd.ReferenciaMotivoAutorizacionV2Valida(p.MotivoAutorizacion) &&
		domain.InstanteUTCCanonico(p.EvaluadaEn) && domain.InstanteUTCCanonico(p.ValidaHasta) &&
		p.ValidaHasta.After(p.EvaluadaEn) && !p.ValidaHasta.After(p.EvaluadaEn.Add(vigenciaMaximaPoliticaOperacionSeguimiento)) &&
		!instante.Before(p.EvaluadaEn) && instante.Before(p.ValidaHasta)
}

// CausaCese es una entrada vigente del catálogo de causas, con el tipo de
// justificante que exige.
type CausaCese struct {
	Clave            domain.ClaveCatalogo
	Etiqueta         string
	ClaveI18n        string
	JustificanteTipo domain.ClaveCatalogo
}

// ReglaCierreExpediente son las condiciones del cierre (regla c10).
type ReglaCierreExpediente struct {
	Condiciones []string
}

// ReglaModificacionNombramiento es la fase de vuelta y los motivos admitidos
// (regla c09).
type ReglaModificacionNombramiento struct {
	FaseRetorno domain.ClaveFase
	Motivos     []OpcionMotivoSeguimiento
}

type OpcionMotivoSeguimiento struct {
	Clave     domain.ClaveCatalogo
	Etiqueta  string
	ClaveI18n string
}

// FuenteReglasSeguimiento resuelve desde catálogos versionados todo lo que
// el sistema no fija: causas y justificantes, condiciones del cierre, fase de
// vuelta y motivos de la modificación, y la política de cada operación.
type FuenteReglasSeguimiento interface {
	CausasCese(ctx context.Context, instante time.Time) ([]CausaCese, PoliticaOperacionSeguimiento, error)
	ReglaCierre(ctx context.Context, instante time.Time) (ReglaCierreExpediente, PoliticaOperacionSeguimiento, error)
	ReglaModificacion(ctx context.Context, instante time.Time) (ReglaModificacionNombramiento, PoliticaOperacionSeguimiento, error)
}

// SolicitudAutorizarOperacionSeguimiento describe la decisión que la
// composición debe exigir y atestar para la audiencia de la operación.
type SolicitudAutorizarOperacionSeguimiento struct {
	Accion    domain.ClaveCatalogo
	Finalidad string
	Audiencia string
	Motivo    vd.ReferenciaEntradaCatalogo
	Recurso   vd.RecursoAutorizable
}

// AutorizadorOperacionSeguimiento exige la decisión V3 ligada a la identidad
// autenticada del canal y entrega el material atestado que consume SQL en
// la misma transacción del efecto. Nunca se reutiliza entre operaciones.
type AutorizadorOperacionSeguimiento interface {
	AutorizarOperacionSeguimiento(context.Context, SolicitudAutorizarOperacionSeguimiento) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// Sellos HMAC de idempotencia y de petición para las tres operaciones.
type SelladorOperacionSeguimiento interface {
	SellarAmbitoOperacionSeguimiento(ctx context.Context, operacion string, s SolicitudSellarAmbitoIdempotencia) (ColeccionSellosHMAC, error)
	DerivarHuellaOperacionSeguimiento(ctx context.Context, operacion string, material []byte) (ColeccionSellosHMAC, error)
}

// DominiosHMACOperacionSeguimiento devuelve los dominios de una operación.
func DominiosHMACOperacionSeguimiento(operacion string) (ambito, huella string, ok bool) {
	switch operacion {
	case OperacionRegistrarCese:
		return DominioAmbitoIdempotenciaCese, DominioHuellaPeticionCese, true
	case OperacionCerrarExpediente:
		return DominioAmbitoIdempotenciaCierreExpediente, DominioHuellaPeticionCierreExpediente, true
	case OperacionModificarTrasNombramiento:
		return DominioAmbitoIdempotenciaModificacionNombrado, DominioHuellaPeticionModificacionNombrado, true
	case OperacionCancelarExpediente:
		return DominioAmbitoIdempotenciaCancelacion, DominioHuellaPeticionCancelacion, true
	case OperacionConfirmarGINPIX:
		return DominioAmbitoIdempotenciaConfirmacionGINPIX, DominioHuellaPeticionConfirmacionGINPIX, true
	case OperacionRegistrarNoIncorporacion:
		return DominioAmbitoIdempotenciaNoIncorporacion, DominioHuellaPeticionNoIncorporacion, true
	}
	return "", "", false
}

// MaterialCese, MaterialCierreExpediente y MaterialModificacionNombramiento
// son la intención exacta que se sella y persiste.
type MaterialCese struct {
	OrganizacionRef, ExpedienteRef, ActorRef, PerfilRef string
	VersionEsperada                                     uint64
	ClaveIdempotencia                                   string
	Datos                                               domain.DatosCese
}

type MaterialCierreExpediente struct {
	OrganizacionRef, ExpedienteRef, ActorRef, PerfilRef string
	VersionEsperada                                     uint64
	ClaveIdempotencia                                   string
	Datos                                               domain.DatosCierreExpediente
}

type MaterialModificacionNombramiento struct {
	OrganizacionRef, ExpedienteRef, ActorRef, PerfilRef string
	VersionEsperada                                     uint64
	ClaveIdempotencia                                   string
	Datos                                               domain.DatosModificacionTrasNombramiento
}

func identidadOperacionValida(org, exp, actor, perfil string, version uint64, clave string) bool {
	return domain.ReferenciaOpacaValida(org) && domain.ReferenciaOpacaValida(exp) && domain.ReferenciaOpacaValida(actor) &&
		domain.ReferenciaOpacaValida(perfil) && VersionOperacionAnalisisConIncrementoValida(version) && ClaveIdempotenciaValida(clave)
}

func (m MaterialCese) Valido() bool {
	return identidadOperacionValida(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada, m.ClaveIdempotencia) &&
		m.Datos.Validar() == nil
}

func (m MaterialCierreExpediente) Valido() bool {
	return identidadOperacionValida(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada, m.ClaveIdempotencia) &&
		m.Datos.Validar() == nil
}

func (m MaterialModificacionNombramiento) Valido() bool {
	return identidadOperacionValida(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada, m.ClaveIdempotencia) &&
		m.Datos.Validar() == nil
}

// ReferenciasEfectoSeguimiento son las referencias que el servidor propone
// para el efecto; la preparación devuelve las ganadoras si ya se confirmó.
type ReferenciasEfectoSeguimiento struct{ ReservaRef, ReciboRef, EventoRef string }

func (r ReferenciasEfectoSeguimiento) Validas() bool {
	return domain.ReferenciaOpacaValida(r.ReservaRef) && domain.ReferenciaOpacaValida(r.ReciboRef) && domain.ReferenciaOpacaValida(r.EventoRef)
}

type GeneradorReferenciasSeguimiento interface {
	GenerarReferenciasSeguimiento(context.Context) (ReferenciasEfectoSeguimiento, error)
}

// SellosOperacionSeguimiento transporta la colección activa y retenida.
type SellosOperacionSeguimiento struct {
	Ambitos, Huellas ColeccionSellosHMAC
}

// PreparacionOperacionSeguimiento es la lectura previa: expediente vigente o,
// si la misma intención ya se confirmó, su recibo original.
type PreparacionOperacionSeguimiento struct {
	Expediente             domain.Expediente
	Referencias            ReferenciasEfectoSeguimiento
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
	IncorporacionRef       string
	InicioIncorporacion    time.Time
	CeseReciboRef          string
	FuenteCosteRef         string
	// AceptacionRef es la resolución de aceptación de una no incorporación.
	AceptacionRef string
	Confirmada    bool
	Recibo        *ReciboOperacionSeguimiento
}

// ContextoAutorizadoSeguimiento son los ámbitos y atributos exactos del
// recurso autorizado; SQL los recompone y compara antes de consumir.
type ContextoAutorizadoSeguimiento struct {
	Ambitos, Atributos map[string]string
}

// OrdenConfirmarOperacionSeguimiento es el write-set de una operación:
// versión nueva, actuación, consumo de autorización y outbox en una sola
// transacción.
type OrdenConfirmarOperacionSeguimiento struct {
	Operacion      string
	Material       any
	Preparacion    PreparacionOperacionSeguimiento
	Siguiente      domain.Expediente
	Politica       PoliticaOperacionSeguimiento
	Contexto       ContextoAutorizadoSeguimiento
	Autorizacion   vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	InstanteEfecto time.Time
	Accion         domain.ClaveCatalogo
	Finalidad      string
	Audiencia      string
}

// ReciboOperacionSeguimiento es el recibo durable que devuelve SQL.
type ReciboOperacionSeguimiento struct {
	Operacion         string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionAnterior   uint64
	VersionResultante uint64
	FaseResultante    domain.ClaveFase
	EstadoResultante  domain.EstadoOperativo
	ReciboRef         string
	AuditoriaRef      string
	EventoRef         string
	ActorRef          string
	RegistradaEn      time.Time
	CausaClave        domain.ClaveCatalogo
	FechaEfecto       string
	CeseReciboRef     string
	CosteCentimos     int64
	MotivoClave       domain.ClaveCatalogo
	GINPIXNumero      string
}

func (r ReciboOperacionSeguimiento) ValidoPara(operacion, org, exp string, versionAnterior uint64) bool {
	return r.Operacion == operacion && r.OrganizacionRef == org && r.ExpedienteRef == exp &&
		r.VersionAnterior == versionAnterior && r.VersionResultante == versionAnterior+1 &&
		r.FaseResultante.Valida() && r.EstadoResultante.Valido() &&
		domain.ReferenciaOpacaValida(r.ReciboRef) && domain.ReferenciaOpacaValida(r.AuditoriaRef) &&
		domain.ReferenciaOpacaValida(r.EventoRef) && domain.ReferenciaOpacaValida(r.ActorRef) && domain.InstanteUTCCanonico(r.RegistradaEn)
}

// RepositorioOperacionSeguimiento prepara y confirma las tres operaciones.
type RepositorioOperacionSeguimiento interface {
	PrepararOperacionSeguimiento(ctx context.Context, operacion string, material any, sellos SellosOperacionSeguimiento, referencias ReferenciasEfectoSeguimiento) (PreparacionOperacionSeguimiento, error)
	ConfirmarOperacionSeguimiento(ctx context.Context, orden OrdenConfirmarOperacionSeguimiento) (ReciboOperacionSeguimiento, error)
}

// EstadoSeguimientoExpediente resume incorporación, cese y cierre para el
// detalle. Los justificantes solo aparecen por referencia y huella.
type EstadoSeguimientoExpediente struct {
	ExpedienteRef       string
	IncorporacionRef    string
	InicioIncorporacion string
	Cese                *EstadoCeseExpediente
	Cierre              *EstadoCierreExpediente
	// Acreditada solo existe con la incorporación acreditada compuesta
	// (CT124): confirmaciones de GINPIX y del centro.
	Acreditada *EstadoIncorporacionAcreditada
}

type EstadoCeseExpediente struct {
	CausaClave         string
	FechaEfecto        string
	JustificanteTipo   string
	JustificanteRef    string
	JustificanteSHA256 string
	Observaciones      string
	ReciboRef          string
	RegistradaEn       time.Time
}

type EstadoCierreExpediente struct {
	Condiciones        []string
	GINPIXNumero       string
	GINPIXConfirmadaEn string
	Observaciones      string
	ReciboRef          string
	RegistradaEn       time.Time
}

type LectorEstadoSeguimiento interface {
	ConsultarEstadoSeguimiento(ctx context.Context, organizacionRef, expedienteRef string) (EstadoSeguimientoExpediente, error)
}

// OpcionesSeguimiento son las opciones publicadas por los catálogos para la
// pantalla; nada se fija en el navegador.
type OpcionesSeguimiento struct {
	Causas      []CausaCese
	Condiciones []string
	FaseRetorno domain.ClaveFase
	Motivos     []OpcionMotivoSeguimiento
	// ConfirmacionGINPIX indica que la confirmación de GINPIX está compuesta:
	// se registra aparte y el cierre toma de ella su número.
	ConfirmacionGINPIX bool
	// NoIncorporacion son los motivos de c22 y si exige segunda persona,
	// solo con la no incorporación compuesta.
	NoIncorporacion *ReglaNoIncorporacion
}

func (o OpcionesSeguimiento) Validas() bool {
	if len(o.Causas) == 0 || len(o.Causas) > maximoOpcionesSeguimiento || len(o.Motivos) > maximoOpcionesSeguimiento ||
		!domain.CondicionesCierreValidas(o.Condiciones) ||
		(o.FaseRetorno != domain.FaseFiscalizacion && o.FaseRetorno != domain.FaseInformeJuridico) {
		return false
	}
	for _, c := range o.Causas {
		if !c.Clave.Valida() || !c.JustificanteTipo.Valida() || c.Etiqueta == "" {
			return false
		}
	}
	if o.NoIncorporacion != nil && !o.NoIncorporacion.Valida() {
		return false
	}
	return !slices.ContainsFunc(o.Motivos, func(m OpcionMotivoSeguimiento) bool { return !m.Clave.Valida() || m.Etiqueta == "" })
}
