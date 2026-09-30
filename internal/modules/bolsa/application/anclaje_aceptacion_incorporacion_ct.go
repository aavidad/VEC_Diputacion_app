package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaConsultaAnclajeAceptacionCT = "vec.bolsa.anclaje-aceptacion-ct.consulta.v1"

type ServicioConsultaAnclajeAceptacionCT struct {
	proveedor   ports.ProveedorAutorizacionConsultaAnclajeAceptacionCT
	repositorio ports.RepositorioConsultaAnclajeAceptacionCT
	ahora       func() time.Time
}

func NuevoServicioConsultaAnclajeAceptacionCT(p ports.ProveedorAutorizacionConsultaAnclajeAceptacionCT, r ports.RepositorioConsultaAnclajeAceptacionCT, ahora func() time.Time) (*ServicioConsultaAnclajeAceptacionCT, error) {
	if nuloPersonaAceptacionCT(p) || nuloPersonaAceptacionCT(r) || ahora == nil {
		return nil, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	return &ServicioConsultaAnclajeAceptacionCT{p, r, ahora}, nil
}

func (s *ServicioConsultaAnclajeAceptacionCT) ConsultarAnclajeAceptacionCT(ctx context.Context, q ports.SolicitudConsultaAnclajeAceptacionCT) (ports.ResultadoConsultaAnclajeAceptacionCT, error) {
	var cero ports.ResultadoConsultaAnclajeAceptacionCT
	if ctx == nil || s == nil || nuloPersonaAceptacionCT(s.proveedor) || nuloPersonaAceptacionCT(s.repositorio) || s.ahora == nil {
		return cero, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	p, err := PrepararConsultaAnclajeAceptacionCT(q)
	if err != nil {
		return cero, err
	}
	// La copia enviada al proveedor no comparte los mapas o bytes del actor.
	q.ActorConfiable.Resultado, err = q.ActorConfiable.Resultado.Clonar()
	if err != nil {
		return cero, ports.ErrConsultaAnclajeAceptacionCTInvalida
	}
	m, err := s.proveedor.AutorizarConsultaAnclajeAceptacionCT(ctx, p)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	o := ports.OrdenConsultaAnclajeAceptacionCT{Solicitud: q, Material: m}
	if err = ValidarOrdenConsultaAnclajeAceptacionCT(o, s.ahora().UTC().Truncate(time.Microsecond)); err != nil {
		return cero, err
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	r, err := s.repositorio.ConsultarAnclajeAceptacionCT(ctx, o)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if ValidarResultadoConsultaAnclajeAceptacionCT(o, r, s.ahora().UTC().Truncate(time.Microsecond)) != nil {
		return cero, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	return r, nil
}

// Preparar fija bytes, ámbito y atributo que consumirán AD3-131/Bolsa68.
func PrepararConsultaAnclajeAceptacionCT(q ports.SolicitudConsultaAnclajeAceptacionCT) (ports.PreparacionConsultaAnclajeAceptacionCT, error) {
	x := q.Selector
	for _, ref := range []string{x.UnidadRef, x.CategoriaRef, x.NecesidadRef, x.AceptacionOperacionRef, x.AperturaOperacionRef, x.LlamamientoRef, x.PropuestaRef} {
		if !referenciaPersonaAceptacionCT.MatchString(ref) {
			return ports.PreparacionConsultaAnclajeAceptacionCT{}, ports.ErrConsultaAnclajeAceptacionCTInvalida
		}
	}
	if !huellaPersonaAceptacionCT.MatchString(x.AceptacionRegistroSHA256) || x.AceptacionRegistroSHA256 == strings.Repeat("0", 64) || q.ActorConfiable.Resultado.Validar() != nil || q.ActorConfiable.Vinculo.ValidarPara(q.ActorConfiable.Resultado) != nil {
		return ports.PreparacionConsultaAnclajeAceptacionCT{}, ports.ErrConsultaAnclajeAceptacionCTInvalida
	}
	actor, err := q.ActorConfiable.Resultado.Clonar()
	if err != nil {
		return ports.PreparacionConsultaAnclajeAceptacionCT{}, ports.ErrConsultaAnclajeAceptacionCTInvalida
	}
	q.ActorConfiable.Resultado = actor
	b, err := json.Marshal(struct {
		Esquema string `json:"esquema"`
		ports.SelectorAnclajeAceptacionCT
	}{EsquemaConsultaAnclajeAceptacionCT, x})
	if err != nil {
		return ports.PreparacionConsultaAnclajeAceptacionCT{}, ports.ErrConsultaAnclajeAceptacionCTInvalida
	}
	h := sha256.Sum256(b)
	r := core.RecursoAutorizable{Referencia: x.AceptacionOperacionRef, ModuloID: "bolsa", Tipo: ports.TipoRecursoAnclajeAceptacionCT, Ambitos: map[string]string{"unidad_ref": x.UnidadRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if _, err = r.HuellaContextoAutorizacionSHA256(); err != nil {
		return ports.PreparacionConsultaAnclajeAceptacionCT{}, ports.ErrConsultaAnclajeAceptacionCTInvalida
	}
	return ports.PreparacionConsultaAnclajeAceptacionCT{Solicitud: q, MaterialCanonico: b, Recurso: r}, nil
}

func ValidarOrdenConsultaAnclajeAceptacionCT(o ports.OrdenConsultaAnclajeAceptacionCT, ahora time.Time) error {
	p, err := PrepararConsultaAnclajeAceptacionCT(o.Solicitud)
	if err != nil {
		return err
	}
	h, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	a := o.Material
	c := o.Solicitud.ActorConfiable.Resultado
	x := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || !instantePersonaAceptacionCT(ahora) || x.Operacion() != ports.AccionConsultaAnclajeAceptacionCT || x.AudienciaConsumo() != ports.AudienciaConsultaAnclajeAceptacionCT || x.EfectoRef() != p.Recurso.Referencia || x.EfectoHuellaSHA256() != h || x.ContextoRef() != c.RegistroContextoRef || x.ContextoHuellaSHA256() != c.HuellaSHA256 || !bytes.Equal(a.ContextoActorCanonico(), c.RepresentacionCanonica) || a.PersonaVersion() != c.Contexto.Instantanea.PersonaVersion || a.PerfilVersion() != c.Contexto.Instantanea.PerfilVersion || ahora.Before(x.EmitidaEn()) || !ahora.Before(x.ExpiraEn()) {
		return ports.ErrConsultaAnclajeAceptacionCTDenegada
	}
	return nil
}

func ValidarResultadoConsultaAnclajeAceptacionCT(o ports.OrdenConsultaAnclajeAceptacionCT, r ports.ResultadoConsultaAnclajeAceptacionCT, ahora time.Time) error {
	x := o.Material.ResumenCapacidad()
	e := r.Evidencia
	if ValidarOrdenConsultaAnclajeAceptacionCT(o, ahora) != nil || !referenciaPersonaAceptacionCT.MatchString(e.DecisionRef) || e.DecisionRef != x.DecisionRef() || !huellaPersonaAceptacionCT.MatchString(e.ConsumoHuellaSHA256) || !referenciaPersonaAceptacionCT.MatchString(e.AuditoriaRef) || !instantePersonaAceptacionCT(e.ConsultadaEn) || e.ConsultadaEn.Before(x.EmitidaEn()) || !e.ConsultadaEn.Before(x.ExpiraEn()) || e.ConsultadaEn.After(ahora) {
		return ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	if r.Estado == "no_encontrada" || r.Estado == "pendiente" {
		if r.Anclaje == nil {
			return nil
		}
		return ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	a, s := r.Anclaje, o.Solicitud.Selector
	if r.Estado != "acreditado" || a == nil || a.UnidadRef != s.UnidadRef || a.CategoriaRef != s.CategoriaRef || a.NecesidadRef != s.NecesidadRef || a.AceptacionOperacionRef != s.AceptacionOperacionRef || a.AceptacionRegistroSHA256 != s.AceptacionRegistroSHA256 || a.AperturaOperacionRef != s.AperturaOperacionRef || !huellaPersonaAceptacionCT.MatchString(a.AperturaRegistroSHA256) || a.AperturaRegistroSHA256 == strings.Repeat("0", 64) || a.LlamamientoRef != s.LlamamientoRef || a.PropuestaRef != s.PropuestaRef || !referenciaPersonaAceptacionCT.MatchString(a.AceptacionReciboRef) {
		return ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	return nil
}

var _ ports.ConsultaAnclajeAceptacionCT = (*ServicioConsultaAnclajeAceptacionCT)(nil)

// ProveedorNominalConsultaAnclajeAceptacionCT reutiliza el emisor V3 común.
// Motivo, correlador y reloj son dependencias cerradas del montaje confiable.
type ProveedorNominalConsultaAnclajeAceptacionCT struct {
	emisor      ports.AutorizadorSituacionParticipacionV3
	motivo      core.ReferenciaEntradaCatalogo
	correlacion func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error)
	ahora       func() time.Time
}

func NuevoProveedorNominalConsultaAnclajeAceptacionCT(e ports.AutorizadorSituacionParticipacionV3, m core.ReferenciaEntradaCatalogo, c func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error), ahora func() time.Time) (*ProveedorNominalConsultaAnclajeAceptacionCT, error) {
	if nuloPersonaAceptacionCT(e) || !core.ReferenciaMotivoAutorizacionV2Valida(m) || c == nil || ahora == nil {
		return nil, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	return &ProveedorNominalConsultaAnclajeAceptacionCT{e, m, c, ahora}, nil
}

func (p *ProveedorNominalConsultaAnclajeAceptacionCT) AutorizarConsultaAnclajeAceptacionCT(ctx context.Context, entrada ports.PreparacionConsultaAnclajeAceptacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || p == nil || nuloPersonaAceptacionCT(p.emisor) || p.correlacion == nil || p.ahora == nil {
		return cero, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	preparada, err := PrepararConsultaAnclajeAceptacionCT(entrada.Solicitud)
	if err != nil {
		return cero, err
	}
	correlacion, err := p.correlacion(ctx)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: preparada.Solicitud.ActorConfiable.Vinculo, ReferenciaMotivo: p.motivo, Accion: ports.AccionConsultaAnclajeAceptacionCT, Recurso: preparada.Recurso, Finalidad: ports.FinalidadConsultaAnclajeAceptacionCT, Correlacion: correlacion})
	if err != nil {
		return cero, ports.ErrConsultaAnclajeAceptacionCTDenegada
	}
	c := preparada.Solicitud.ActorConfiable.Resultado
	d, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	concedida, _, resultadoErr := d.Resultado()
	ahora := p.ahora().UTC().Truncate(time.Microsecond)
	if d.ValidarPara(s) != nil || resultadoErr != nil {
		return cero, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	if !concedida {
		return cero, ports.ErrConsultaAnclajeAceptacionCTDenegada
	}
	orden, ordenErr := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, p.motivo, c)
	if ordenErr != nil || confirmacion.ValidarPara(orden) != nil {
		return cero, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	restricciones, restriccionesErr := d.RestriccionesProyeccionPara(s)
	if restriccionesErr != nil {
		return cero, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	if !reflect.DeepEqual(restricciones.CamposPermitidos, []string{"anclaje"}) || len(restricciones.Obligaciones) != 0 || !confirmacion.DentroDeVentanaEn(ahora) {
		return cero, ports.ErrConsultaAnclajeAceptacionCTDenegada
	}
	if nuloPersonaAceptacionCT(exportador) {
		return cero, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	m, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	if !materialAutorizacionBorradorLlamamientoExacto(s, d, confirmacion, c, p.motivo, m, ports.AudienciaConsultaAnclajeAceptacionCT) {
		return cero, ports.ErrConsultaAnclajeAceptacionCTNoDisponible
	}
	if err := ValidarOrdenConsultaAnclajeAceptacionCT(ports.OrdenConsultaAnclajeAceptacionCT{Solicitud: preparada.Solicitud, Material: m}, ahora); err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return m, nil
}

var _ ports.ProveedorAutorizacionConsultaAnclajeAceptacionCT = (*ProveedorNominalConsultaAnclajeAceptacionCT)(nil)
