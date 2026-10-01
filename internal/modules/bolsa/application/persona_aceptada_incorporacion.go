package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaConsultaPersonaAceptacionCT = "vec.bolsa.persona-aceptacion-ct.consulta.v1"

var (
	referenciaPersonaAceptacionCT = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{2,159}$`)
	huellaPersonaAceptacionCT     = regexp.MustCompile(`^[a-f0-9]{64}$`)
	personaAceptacionCT           = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
)

type ServicioConsultaPersonaAceptacionCT struct {
	proveedor   ports.ProveedorAutorizacionConsultaPersonaAceptacionCT
	repositorio ports.RepositorioConsultaPersonaAceptacionCT
	ahora       func() time.Time
}

func NuevoServicioConsultaPersonaAceptacionCT(p ports.ProveedorAutorizacionConsultaPersonaAceptacionCT, r ports.RepositorioConsultaPersonaAceptacionCT, ahora func() time.Time) (*ServicioConsultaPersonaAceptacionCT, error) {
	if nuloPersonaAceptacionCT(p) || nuloPersonaAceptacionCT(r) || ahora == nil {
		return nil, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	return &ServicioConsultaPersonaAceptacionCT{p, r, ahora}, nil
}

func (s *ServicioConsultaPersonaAceptacionCT) ConsultarPersonaAceptacionCT(ctx context.Context, q ports.SolicitudConsultaPersonaAceptacionCT) (ports.ResultadoConsultaPersonaAceptacionCT, error) {
	var cero ports.ResultadoConsultaPersonaAceptacionCT
	if ctx == nil || s == nil || nuloPersonaAceptacionCT(s.proveedor) || nuloPersonaAceptacionCT(s.repositorio) || s.ahora == nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	p, err := PrepararConsultaPersonaAceptacionCT(q)
	if err != nil {
		return cero, err
	}
	// La copia enviada al proveedor no comparte los mapas o bytes del actor.
	q.ActorConfiable.Resultado, err = q.ActorConfiable.Resultado.Clonar()
	if err != nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTInvalida
	}
	m, err := s.proveedor.AutorizarConsultaPersonaAceptacionCT(ctx, p)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	o := ports.OrdenConsultaPersonaAceptacionCT{Solicitud: q, Material: m}
	if err = ValidarOrdenConsultaPersonaAceptacionCT(o, s.ahora().UTC().Truncate(time.Microsecond)); err != nil {
		return cero, err
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	r, err := s.repositorio.ConsultarPersonaAceptacionCT(ctx, o)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if ValidarResultadoConsultaPersonaAceptacionCT(o, r, s.ahora().UTC().Truncate(time.Microsecond)) != nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	return r, nil
}

// Preparar fija bytes, ámbito y atributo que consumirán AD3/Bolsa67.
func PrepararConsultaPersonaAceptacionCT(q ports.SolicitudConsultaPersonaAceptacionCT) (ports.PreparacionConsultaPersonaAceptacionCT, error) {
	x := q.Selector
	for _, ref := range []string{x.UnidadRef, x.CategoriaRef, x.NecesidadRef, x.AceptacionOperacionRef, x.AperturaOperacionRef, x.LlamamientoRef, x.PropuestaRef} {
		if !referenciaPersonaAceptacionCT.MatchString(ref) {
			return ports.PreparacionConsultaPersonaAceptacionCT{}, ports.ErrConsultaPersonaAceptacionCTInvalida
		}
	}
	if !huellaPersonaAceptacionCT.MatchString(x.AceptacionRegistroSHA256) || !huellaPersonaAceptacionCT.MatchString(x.AperturaRegistroSHA256) || q.ActorConfiable.Resultado.Validar() != nil || q.ActorConfiable.Vinculo.ValidarPara(q.ActorConfiable.Resultado) != nil {
		return ports.PreparacionConsultaPersonaAceptacionCT{}, ports.ErrConsultaPersonaAceptacionCTInvalida
	}
	actor, err := q.ActorConfiable.Resultado.Clonar()
	if err != nil {
		return ports.PreparacionConsultaPersonaAceptacionCT{}, ports.ErrConsultaPersonaAceptacionCTInvalida
	}
	q.ActorConfiable.Resultado = actor
	b, err := json.Marshal(struct {
		Esquema string `json:"esquema"`
		ports.SelectorPersonaAceptacionCT
	}{EsquemaConsultaPersonaAceptacionCT, x})
	if err != nil {
		return ports.PreparacionConsultaPersonaAceptacionCT{}, ports.ErrConsultaPersonaAceptacionCTInvalida
	}
	h := sha256.Sum256(b)
	r := core.RecursoAutorizable{Referencia: x.AceptacionOperacionRef, ModuloID: "bolsa", Tipo: ports.TipoRecursoPersonaAceptacionCT, Ambitos: map[string]string{"unidad_ref": x.UnidadRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if _, err = r.HuellaContextoAutorizacionSHA256(); err != nil {
		return ports.PreparacionConsultaPersonaAceptacionCT{}, ports.ErrConsultaPersonaAceptacionCTInvalida
	}
	return ports.PreparacionConsultaPersonaAceptacionCT{Solicitud: q, MaterialCanonico: b, Recurso: r}, nil
}

func ValidarOrdenConsultaPersonaAceptacionCT(o ports.OrdenConsultaPersonaAceptacionCT, ahora time.Time) error {
	p, err := PrepararConsultaPersonaAceptacionCT(o.Solicitud)
	if err != nil {
		return err
	}
	h, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	a := o.Material
	c := o.Solicitud.ActorConfiable.Resultado
	x := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || !instantePersonaAceptacionCT(ahora) || x.Operacion() != ports.AccionConsultaPersonaAceptacionCT || x.AudienciaConsumo() != ports.AudienciaConsultaPersonaAceptacionCT || x.EfectoRef() != p.Recurso.Referencia || x.EfectoHuellaSHA256() != h || x.ContextoRef() != c.RegistroContextoRef || x.ContextoHuellaSHA256() != c.HuellaSHA256 || !bytes.Equal(a.ContextoActorCanonico(), c.RepresentacionCanonica) || a.PersonaVersion() != c.Contexto.Instantanea.PersonaVersion || a.PerfilVersion() != c.Contexto.Instantanea.PerfilVersion || ahora.Before(x.EmitidaEn()) || !ahora.Before(x.ExpiraEn()) {
		return ports.ErrConsultaPersonaAceptacionCTDenegada
	}
	return nil
}

func ValidarResultadoConsultaPersonaAceptacionCT(o ports.OrdenConsultaPersonaAceptacionCT, r ports.ResultadoConsultaPersonaAceptacionCT, ahora time.Time) error {
	x := o.Material.ResumenCapacidad()
	e := r.Evidencia
	if ValidarOrdenConsultaPersonaAceptacionCT(o, ahora) != nil || !referenciaPersonaAceptacionCT.MatchString(e.DecisionRef) || e.DecisionRef != x.DecisionRef() || !huellaPersonaAceptacionCT.MatchString(e.ConsumoHuellaSHA256) || !referenciaPersonaAceptacionCT.MatchString(e.AuditoriaRef) || !instantePersonaAceptacionCT(e.ConsultadaEn) || e.ConsultadaEn.Before(x.EmitidaEn()) || !e.ConsultadaEn.Before(x.ExpiraEn()) || e.ConsultadaEn.After(ahora) {
		return ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	if r.Estado == "no_encontrada" {
		if r.Aceptacion == nil && r.Persona == nil && r.Vinculo == nil {
			return nil
		}
		return ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	a, s := r.Aceptacion, o.Solicitud.Selector
	if a == nil || a.OperacionRef != s.AceptacionOperacionRef || !referenciaPersonaAceptacionCT.MatchString(a.ReciboRef) || a.RegistroSHA256 != s.AceptacionRegistroSHA256 || a.AperturaOperacionRef != s.AperturaOperacionRef || a.AperturaRegistroSHA256 != s.AperturaRegistroSHA256 || a.LlamamientoRef != s.LlamamientoRef {
		return ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	if r.Estado == "pendiente" {
		if r.Persona == nil && r.Vinculo == nil {
			return nil
		}
		return ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	v, p := r.Vinculo, r.Persona
	if r.Estado != "acreditado" || p == nil || v == nil || !personaAceptacionCT.MatchString(p.Ref) || p.Version < 1 || !referenciaPersonaAceptacionCT.MatchString(v.Ref) || v.Version < 1 || !referenciaPersonaAceptacionCT.MatchString(v.ProcedenciaRef) || v.ProcedenciaVersion < 1 || !huellaPersonaAceptacionCT.MatchString(v.ProcedenciaSHA256) || (v.Poblacion != "interna" && v.Poblacion != "externa") || !instantePersonaAceptacionCT(v.VigenteHasta) || !ahora.Before(v.VigenteHasta) {
		return ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	return nil
}

func instantePersonaAceptacionCT(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}
func nuloPersonaAceptacionCT(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}
func ErrorConsultaPersonaAceptacionCT(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	// La autoridad común puede conservar denegación y fallo de registro juntos.
	// El fallo técnico mantiene su clasificación y nunca se expone su causa.
	for _, dependencia := range []error{ports.ErrConsultaPersonaAceptacionCTNoDisponible, vecports.ErrFuenteAutorizacionNoDisponible, vecports.ErrRegistroDecisionNoDisponible, vecports.ErrRegistroDenegacionNoDisponible, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, vecports.ErrInstantaneaAutorizacionObsoleta} {
		if errors.Is(err, dependencia) {
			return ports.ErrConsultaPersonaAceptacionCTNoDisponible
		}
	}
	if errors.Is(err, ports.ErrConsultaPersonaAceptacionCTDenegada) || errors.Is(err, core.ErrAutorizacionDenegada) || errors.Is(err, core.ErrPermissionDenied) || errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
		return ports.ErrConsultaPersonaAceptacionCTDenegada
	}
	return ports.ErrConsultaPersonaAceptacionCTNoDisponible
}

var _ ports.ConsultaPersonaAceptacionCT = (*ServicioConsultaPersonaAceptacionCT)(nil)

// ProveedorNominalConsultaPersonaAceptacionCT reutiliza el emisor V3 común.
// Motivo, correlador y reloj son dependencias cerradas del montaje confiable.
type ProveedorNominalConsultaPersonaAceptacionCT struct {
	emisor      ports.AutorizadorSituacionParticipacionV3
	motivo      core.ReferenciaEntradaCatalogo
	correlacion func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error)
	ahora       func() time.Time
}

func NuevoProveedorNominalConsultaPersonaAceptacionCT(e ports.AutorizadorSituacionParticipacionV3, m core.ReferenciaEntradaCatalogo, c func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error), ahora func() time.Time) (*ProveedorNominalConsultaPersonaAceptacionCT, error) {
	if nuloPersonaAceptacionCT(e) || !core.ReferenciaMotivoAutorizacionV2Valida(m) || c == nil || ahora == nil {
		return nil, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	return &ProveedorNominalConsultaPersonaAceptacionCT{e, m, c, ahora}, nil
}

func (p *ProveedorNominalConsultaPersonaAceptacionCT) AutorizarConsultaPersonaAceptacionCT(ctx context.Context, entrada ports.PreparacionConsultaPersonaAceptacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || p == nil || nuloPersonaAceptacionCT(p.emisor) || p.correlacion == nil || p.ahora == nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	preparada, err := PrepararConsultaPersonaAceptacionCT(entrada.Solicitud)
	if err != nil {
		return cero, err
	}
	correlacion, err := p.correlacion(ctx)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: preparada.Solicitud.ActorConfiable.Vinculo, ReferenciaMotivo: p.motivo, Accion: ports.AccionConsultaPersonaAceptacionCT, Recurso: preparada.Recurso, Finalidad: ports.FinalidadConsultaPersonaAceptacionCT, Correlacion: correlacion})
	if err != nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTDenegada
	}
	c := preparada.Solicitud.ActorConfiable.Resultado
	d, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	concedida, _, resultadoErr := d.Resultado()
	ahora := p.ahora().UTC().Truncate(time.Microsecond)
	if d.ValidarPara(s) != nil || resultadoErr != nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	if !concedida {
		return cero, ports.ErrConsultaPersonaAceptacionCTDenegada
	}
	orden, ordenErr := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, p.motivo, c)
	if ordenErr != nil || confirmacion.ValidarPara(orden) != nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	restricciones, restriccionesErr := d.RestriccionesProyeccionPara(s)
	if restriccionesErr != nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	if !reflect.DeepEqual(restricciones.CamposPermitidos, []string{"aceptacion", "persona", "vinculo"}) || len(restricciones.Obligaciones) != 0 || !confirmacion.DentroDeVentanaEn(ahora) {
		return cero, ports.ErrConsultaPersonaAceptacionCTDenegada
	}
	if nuloPersonaAceptacionCT(exportador) {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	m, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	if !materialAutorizacionBorradorLlamamientoExacto(s, d, confirmacion, c, p.motivo, m, ports.AudienciaConsultaPersonaAceptacionCT) {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	if err := ValidarOrdenConsultaPersonaAceptacionCT(ports.OrdenConsultaPersonaAceptacionCT{Solicitud: preparada.Solicitud, Material: m}, ahora); err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return m, nil
}

var _ ports.ProveedorAutorizacionConsultaPersonaAceptacionCT = (*ProveedorNominalConsultaPersonaAceptacionCT)(nil)
