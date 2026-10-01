package incorporacionejercicio

import (
	"context"
	"errors"
	"reflect"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
)

// ProtocoloPlanNominal impide que una incorporación B2 se interprete como el
// ejercicio histórico. La selección pertenece al contrato persistido de CT.
type ProtocoloPlanNominal string

const (
	ProtocoloEjercicioV2  ProtocoloPlanNominal = "ejercicio_v2"
	ProtocoloPersonalB2V1 ProtocoloPlanNominal = "personal_b2_v1"
)

// ContratoPlanNominal es una proyección autorizada de la historia durable de CT.
// El lector propietario verifica su versión, recibo y huella antes de devolverla.
// No se carga de un fichero de planes por expediente ni de la petición HTTP.
type ContratoPlanNominal struct {
	ContratoRef         string
	Protocolo           ProtocoloPlanNominal
	OrganizacionRef     string
	ExpedienteRef       string
	VersionExpediente   uint64
	ContratoVersion     uint64
	ContratoReciboRef   string
	ContratoSHA256      string
	EjercicioSintetico  bool
	SolicitudRef        string
	IntencionRef        string
	IntencionReciboRef  string
	IntencionVersion    uint64
	AceptacionRef       string
	AceptacionReciboRef string
	LlamamientoRef      string
	SeleccionRef        string
	VersionSeleccion    uint64
	SeleccionReciboRef  string
	PersonaRef          string
	PersonaVersion      uint64
	PersonaFuente       dom.FuentePlanPersonalB2
	FuenteRPT           ct.ReferenciaVersionadaPersonalRPT
	CategoriaRef        string
	VinculoRevision     uint64
	VinculoReciboRef    string
	PuestoRef           string
	PlazaRef            string
	ReservaIdempotente  string
	DatosPersonal       DatosActosPersonalB2
	SelectorBolsa       dom.SelectorBolsaPlanB2
}

// DatosActosPersonalB2 conserva selecciones verificadas de RRHH y sus fuentes.
// El periodo civil admite fin abierto. No deriva régimen ni modalidad de etiquetas.
type DatosActosPersonalB2 struct {
	OrganismoRef, UnidadRef                         string
	Regimen, Modalidad                              personal.EntradaCatalogoEmpleadoB2
	Desde, Hasta                                    personal.FechaCivil
	ClaseOcupacion                                  string
	VersionPlantillaRef, VersionRPTRef              string
	RevisionPlaza, RevisionPuesto                   int64
	FuenteOrganizacionRef, FuenteOrganizacionSHA256 string
	CatalogoRPTID, ModuloRPTID, CategoriaID         string
	CatalogoRPTVersion                              uint64
	CatalogoRPTHuellaSHA256                         string
	Procedencia                                     personal.ProcedenciaActoEmpleadoB2
}

func (d DatosActosPersonalB2) validar() bool {
	return dom.ReferenciaOpacaValida(d.OrganismoRef) && dom.ReferenciaOpacaValida(d.UnidadRef) &&
		d.Regimen.Validar() == nil && d.Modalidad.Validar() == nil && d.Desde.Validar() == nil &&
		(d.Hasta == "" || (d.Hasta.Validar() == nil && d.Desde.AntesDe(d.Hasta))) &&
		dom.ClaseOcupacionPlanPersonalB2Valida(d.ClaseOcupacion) &&
		dom.ReferenciaOpacaValida(d.VersionPlantillaRef) && dom.ReferenciaOpacaValida(d.VersionRPTRef) &&
		d.RevisionPlaza > 0 && d.RevisionPuesto > 0 && dom.ReferenciaOpacaValida(d.FuenteOrganizacionRef) &&
		huellaPlanNominalValida(d.FuenteOrganizacionSHA256) && d.CatalogoRPTID != "" && d.ModuloRPTID != "" &&
		d.CategoriaID != "" && d.CatalogoRPTVersion > 0 && d.CatalogoRPTVersion <= 2147483647 &&
		huellaPlanNominalValida(d.CatalogoRPTHuellaSHA256) && d.Procedencia.Validar() == nil
}

// FuenteContratoPlanNominal efectúa la lectura nominal y su autorización
// vigente; una referencia opaca no es una concesión de lectura.
type FuenteContratoPlanNominal interface {
	LeerContratoPlanNominal(context.Context, string, string) (ContratoPlanNominal, error)
}

// IntencionCTDurable se relee desde CT antes de reservar en Personal. Ni la
// petición HTTP ni una referencia opaca pueden sustituir este recibo.
type IntencionCTDurable struct {
	OrganizacionRef, ExpedienteRef string
	ContratoReciboRef              string
	IntencionRef, ReciboRef        string
	Version                        uint64
	ReservaIdempotente             string
	Confirmada                     bool
}

type FuenteIntencionCTDurable interface {
	LeerIntencionCTDurable(context.Context, ContratoPlanNominal) (IntencionCTDurable, error)
}

// AntecedenteCT124 acredita la aceptación vigente y la ausencia de una no
// incorporación confirmada. CT conserva esa decisión y su historia.
type AntecedenteCT124 struct {
	OrganizacionRef, ExpedienteRef string
	AceptacionRef, LlamamientoRef  string
	VersionExpediente              uint64
	ReciboRef                      string
	NoIncorporacion                bool
}

type FuenteAntecedenteCT124 interface {
	LeerAntecedenteCT124(context.Context, ContratoPlanNominal) (AntecedenteCT124, error)
}

// PersonaSeleccionadaBolsa sólo puede proceder de una consulta nominal de
// Bolsa, con versión y recibo originales de la selección aceptada.
type PersonaSeleccionadaBolsa struct {
	OrganizacionRef, ExpedienteRef string
	AceptacionRef, LlamamientoRef  string
	SeleccionRef                   string
	VersionSeleccion               uint64
	PersonaRef                     string
	PersonaVersion                 uint64
	ReciboRef                      string
	FuenteRef, FuenteSHA256        string
	FuenteVersion                  int64
}

type FuentePersonaSeleccionadaBolsa interface {
	LeerPersonaSeleccionada(context.Context, ContratoPlanNominal) (PersonaSeleccionadaBolsa, error)
}

// PuestoRPTNominal debe comprobar la publicación exacta y el vínculo CT154,
// incluidos su revisión y recibo. CT154 prospectivo no acredita procedencia
// histórica ni convierte una propuesta en nombramiento eficaz.
type PuestoRPTNominal struct {
	OrganizacionRef, ExpedienteRef string
	Fuente                         ct.ReferenciaVersionadaPersonalRPT
	CategoriaRef                   string
	PuestoRef, PlazaRef            string
	VinculoRevision                uint64
	VinculoReciboRef               string
	Prospectivo                    bool
	AcreditaProcedenciaHistorica   bool
}

type FuentePuestoRPTNominal interface {
	LeerPuestoRPT(context.Context, ContratoPlanNominal) (PuestoRPTNominal, error)
}

// SolicitudReservaPersonalB2 conserva las referencias del contrato y la
// persona que devolvió Bolsa. No contiene un empleado deducido del actor.
type SolicitudReservaPersonalB2 struct {
	Contrato ContratoPlanNominal
	Persona  PersonaSeleccionadaBolsa
	Puesto   PuestoRPTNominal
}

// ReservaPersonalB2 es el resultado durable que emitirá Personal. La fábrica
// sólo la acepta si liga exactamente el contrato, la persona y el puesto.
type ReservaPersonalB2 struct {
	OrganizacionRef, ExpedienteRef string
	SolicitudRef, IdempotenciaRef  string
	PersonaRef                     string
	PersonaVersion                 uint64
	ContratoReciboRef              string
	ContratoSHA256                 string
	FuenteRPT                      ct.ReferenciaVersionadaPersonalRPT
	PuestoRef, PlazaRef            string
	ReservaRef, ReciboRef          string
	VersionReserva                 uint64
	EmpleadoRef                    string
	EjercicioSintetico             bool
	Plan                           pp.PlanIncorporacionCT
}

// ReservadorPersonalB2 pertenece a Personal. Su implementación debe consumir
// permiso, idempotencia, estado, auditoría y outbox en el mismo efecto durable.
// Nunca se conecta un doble de prueba en composición real.
type ReservadorPersonalB2 interface {
	ReservarPersonalB2(context.Context, SolicitudReservaPersonalB2) (ReservaPersonalB2, error)
}

// PlanNominal mantiene las dos rutas disjuntas. Sólo una rama se rellena; un
// plan B2 no se convierte en PlanPreparacionDurableV2 ni activa CT75 por sí solo.
type PlanNominal struct {
	Protocolo ProtocoloPlanNominal
	Ejercicio *PlanPreparacionDurableV2
	Personal  *PlanPersonalB2
}

type PlanPersonalB2 struct {
	Contrato    ContratoPlanNominal
	Antecedente AntecedenteCT124
	Persona     PersonaSeleccionadaBolsa
	Puesto      PuestoRPTNominal
	Reserva     ReservaPersonalB2
}

type ConfiguracionPlanesNominales struct {
	Contratos   FuenteContratoPlanNominal
	Ejercicio   FuentePlanesPreparacionV2
	IntencionCT FuenteIntencionCTDurable
	CT124       FuenteAntecedenteCT124
	Bolsa       FuentePersonaSeleccionadaBolsa
	RPT         FuentePuestoRPTNominal
	Personal    ReservadorPersonalB2
}

type PlanesNominales struct{ c ConfiguracionPlanesNominales }

// Se admite montar el histórico mientras las capacidades B2 están pendientes.
// Un contrato personal_b2_v1 sin alguna de ellas devuelve pendiente sin alta.
func NuevosPlanesNominales(c ConfiguracionPlanesNominales) (*PlanesNominales, error) {
	if nuloPlanNominal(c.Contratos) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return &PlanesNominales{c: c}, nil
}

// Preparar pertenece al camino de confirmación: la rama B2 puede reservar en
// Personal. Una consulta GET sólo debe leer antecedentes mediante sus puertos.
func (f *PlanesNominales) Preparar(ctx context.Context, org, exp string) (PlanNominal, error) {
	var cero PlanNominal
	if f == nil || ctx == nil || !dom.ReferenciaOpacaValida(org) || !dom.ReferenciaOpacaValida(exp) {
		return cero, ct.ErrIntencionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	contrato, err := f.c.Contratos.LeerContratoPlanNominal(ctx, org, exp)
	if err != nil {
		return cero, errorFuentePlanNominal(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !contratoBasicoValido(contrato, org, exp) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	switch contrato.Protocolo {
	case ProtocoloEjercicioV2:
		if !contrato.EjercicioSintetico {
			return cero, ct.ErrConflictoIncorporacionAplicacion
		}
		if nuloPlanNominal(f.c.Ejercicio) {
			return cero, ct.ErrPreparacionIncorporacionPendiente
		}
		plan, err := f.c.Ejercicio.ResolverPlan(ctx, org, exp)
		if err != nil {
			return cero, errorFuentePlanNominal(ctx, err)
		}
		if err := ctx.Err(); err != nil {
			return cero, err
		}
		if plan.Validar() != nil || plan.OrganizacionRef != org || plan.SolicitudPersonal.ExpedienteRef != exp ||
			plan.SolicitudPersonal.SolicitudRef != contrato.SolicitudRef || plan.SolicitudPersonal.VersionExpediente != contrato.VersionExpediente {
			return cero, ct.ErrConflictoIncorporacionAplicacion
		}
		copia := plan.Copia()
		return PlanNominal{Protocolo: ProtocoloEjercicioV2, Ejercicio: &copia}, nil
	case ProtocoloPersonalB2V1:
		return f.resolverB2(ctx, contrato)
	default:
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
}

func (f *PlanesNominales) resolverB2(ctx context.Context, c ContratoPlanNominal) (PlanNominal, error) {
	var cero PlanNominal
	if !contratoB2Valido(c) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	if nuloPlanNominal(f.c.IntencionCT) || nuloPlanNominal(f.c.CT124) || nuloPlanNominal(f.c.Bolsa) ||
		nuloPlanNominal(f.c.RPT) || nuloPlanNominal(f.c.Personal) {
		return cero, ct.ErrPreparacionIncorporacionPendiente
	}
	i, err := f.c.IntencionCT.LeerIntencionCTDurable(ctx, c)
	if err != nil {
		return cero, errorFuentePlanNominal(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !i.Confirmada || i.OrganizacionRef != c.OrganizacionRef || i.ExpedienteRef != c.ExpedienteRef ||
		i.ContratoReciboRef != c.ContratoReciboRef || i.IntencionRef != c.IntencionRef ||
		i.ReciboRef != c.IntencionReciboRef || i.Version != c.IntencionVersion ||
		i.ReservaIdempotente != c.ReservaIdempotente {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	a, err := f.c.CT124.LeerAntecedenteCT124(ctx, c)
	if err != nil {
		return cero, errorFuentePlanNominal(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if a.OrganizacionRef != c.OrganizacionRef || a.ExpedienteRef != c.ExpedienteRef || a.AceptacionRef != c.AceptacionRef ||
		a.LlamamientoRef != c.LlamamientoRef || a.VersionExpediente != c.VersionExpediente || a.NoIncorporacion ||
		a.ReciboRef != c.AceptacionReciboRef {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	p, err := f.c.Bolsa.LeerPersonaSeleccionada(ctx, c)
	if err != nil {
		return cero, errorFuentePlanNominal(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if p.OrganizacionRef != c.OrganizacionRef || p.ExpedienteRef != c.ExpedienteRef || p.AceptacionRef != c.AceptacionRef ||
		p.LlamamientoRef != c.LlamamientoRef || p.SeleccionRef != c.SeleccionRef || p.VersionSeleccion != c.VersionSeleccion ||
		!personal.ReferenciaPersonaValida(p.PersonaRef) || p.PersonaVersion == 0 || p.PersonaVersion > ct.MaximoEnteroSeguroOperacionAnalisis ||
		p.ReciboRef != c.SeleccionReciboRef || p.PersonaRef != c.PersonaRef || p.PersonaVersion != c.PersonaVersion ||
		p.FuenteRef != c.PersonaFuente.Ref || p.FuenteVersion < 1 || uint64(p.FuenteVersion) != c.PersonaFuente.Version ||
		p.FuenteSHA256 != c.PersonaFuente.SHA256 || !dom.ReferenciaOpacaValida(p.FuenteRef) || !huellaPlanNominalValida(p.FuenteSHA256) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	r, err := f.c.RPT.LeerPuestoRPT(ctx, c)
	if err != nil {
		return cero, errorFuentePlanNominal(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if r.OrganizacionRef != c.OrganizacionRef || r.ExpedienteRef != c.ExpedienteRef || r.Fuente != c.FuenteRPT ||
		r.CategoriaRef != c.CategoriaRef ||
		r.PuestoRef != c.PuestoRef || r.PlazaRef != c.PlazaRef || r.VinculoRevision != c.VinculoRevision ||
		r.VinculoReciboRef != c.VinculoReciboRef ||
		!r.Prospectivo || r.AcreditaProcedenciaHistorica {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	// Releer justo antes del primer efecto de Personal impide que una selección
	// cambiada durante las otras consultas se convierta en la reserva del plan CT.
	actual, err := f.c.Bolsa.LeerPersonaSeleccionada(ctx, c)
	if err != nil {
		return cero, errorFuentePlanNominal(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if actual != p {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	reserva, err := f.c.Personal.ReservarPersonalB2(ctx, SolicitudReservaPersonalB2{Contrato: c, Persona: p, Puesto: r})
	if err != nil {
		return cero, errorFuentePlanNominal(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if reserva.OrganizacionRef != c.OrganizacionRef || reserva.ExpedienteRef != c.ExpedienteRef ||
		reserva.SolicitudRef != c.SolicitudRef || reserva.IdempotenciaRef != c.ReservaIdempotente ||
		reserva.PersonaRef != p.PersonaRef || reserva.PersonaVersion != p.PersonaVersion ||
		reserva.ContratoReciboRef != c.ContratoReciboRef || reserva.ContratoSHA256 != c.ContratoSHA256 ||
		reserva.FuenteRPT != c.FuenteRPT ||
		reserva.PuestoRef != c.PuestoRef || reserva.PlazaRef != c.PlazaRef || reserva.VersionReserva == 0 ||
		!dom.ReferenciaOpacaValida(reserva.ReservaRef) || !dom.ReferenciaOpacaValida(reserva.ReciboRef) ||
		(reserva.EmpleadoRef != "" && !personal.ReferenciaEmpleadoValida(reserva.EmpleadoRef)) || reserva.EjercicioSintetico != c.EjercicioSintetico {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	plan := PlanPersonalB2{Contrato: c, Antecedente: a, Persona: p, Puesto: r, Reserva: reserva}
	return PlanNominal{Protocolo: ProtocoloPersonalB2V1, Personal: &plan}, nil
}

func contratoBasicoValido(c ContratoPlanNominal, org, exp string) bool {
	return c.OrganizacionRef == org && c.ExpedienteRef == exp && c.ContratoVersion > 0 &&
		c.ContratoVersion <= ct.MaximoEnteroSeguroOperacionAnalisis && c.VersionExpediente > 0 &&
		c.VersionExpediente <= ct.MaximoEnteroSeguroOperacionAnalisis && dom.ReferenciaOpacaValida(c.ContratoReciboRef) &&
		huellaPlanNominalValida(c.ContratoSHA256) && dom.ReferenciaOpacaValida(c.SolicitudRef)
}

func contratoB2Valido(c ContratoPlanNominal) bool {
	return c.DatosPersonal.validar() && selectorBolsaPlanB2Valido(c) && dom.ReferenciaOpacaValida(c.ContratoRef) && dom.ReferenciaOpacaValida(c.IntencionRef) &&
		dom.ReferenciaOpacaValida(c.IntencionReciboRef) && c.IntencionVersion > 0 &&
		c.IntencionVersion <= ct.MaximoEnteroSeguroOperacionAnalisis && dom.ReferenciaOpacaValida(c.AceptacionRef) &&
		dom.ReferenciaOpacaValida(c.AceptacionReciboRef) && dom.ReferenciaOpacaValida(c.LlamamientoRef) &&
		(c.SeleccionRef == "" || dom.ReferenciaOpacaValida(c.SeleccionRef)) &&
		c.VersionSeleccion <= ct.MaximoEnteroSeguroOperacionAnalisis && dom.ReferenciaOpacaValida(c.SeleccionReciboRef) &&
		personal.ReferenciaPersonaValida(c.PersonaRef) && dom.VersionPlanPersonalB2Valida(c.PersonaVersion) && c.PersonaFuente.Valida() &&
		c.FuenteRPT.Validar() == nil && c.VinculoRevision > 0 && c.VinculoRevision <= ct.MaximoEnteroSeguroOperacionAnalisis &&
		dom.ReferenciaOpacaValida(c.VinculoReciboRef) &&
		dom.ReferenciaOpacaValida(c.CategoriaRef) && dom.ReferenciaOpacaValida(c.PuestoRef) && dom.ReferenciaOpacaValida(c.PlazaRef) &&
		ct.ClaveIdempotenciaValida(c.ReservaIdempotente)
}

func selectorBolsaPlanB2Valido(c ContratoPlanNominal) bool {
	s := c.SelectorBolsa
	for _, r := range []string{s.UnidadRef, s.CategoriaRef, s.NecesidadRef, s.AceptacionOperacionRef, s.AperturaOperacionRef, s.LlamamientoRef, s.PropuestaRef} {
		if !dom.ReferenciaOpacaValida(r) {
			return false
		}
	}
	return s.UnidadRef == c.DatosPersonal.UnidadRef && s.CategoriaRef == c.CategoriaRef && s.LlamamientoRef == c.LlamamientoRef &&
		huellaPlanNominalValida(s.AceptacionRegistroSHA256) && huellaPlanNominalValida(s.AperturaRegistroSHA256) &&
		c.DatosPersonal.CategoriaID == c.CategoriaRef
}

func huellaPlanNominalValida(s string) bool {
	if len(s) != 64 || s == "0000000000000000000000000000000000000000000000000000000000000000" {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func errorFuentePlanNominal(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || errors.Is(err, personal.ErrRegistroEmpleadoB2Denegado) {
		return ct.ErrDenegadaIncorporacionAplicacion
	}
	if errors.Is(err, ct.ErrPreparacionIncorporacionPendiente) || errors.Is(err, personal.ErrRegistroEmpleadoB2NoEncontrado) {
		return ct.ErrPreparacionIncorporacionPendiente
	}
	if errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || errors.Is(err, personal.ErrRegistroEmpleadoB2Conflicto) {
		return ct.ErrConflictoIncorporacionAplicacion
	}
	return ct.ErrComposicionIncorporacionAplicacion
}

func nuloPlanNominal(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}
