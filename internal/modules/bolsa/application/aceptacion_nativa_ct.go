package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var referenciaAceptacionNativaCT = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,511}$`)

func esquemaAceptacionNativaCT(persona bool) string {
	if persona {
		return "vec.bolsa.aceptacion-nativa-ct.persona.consulta.v1"
	}
	return "vec.bolsa.aceptacion-nativa-ct.anclaje.consulta.v1"
}

func PrepararConsultaAceptacionNativaCT(q ports.SolicitudConsultaAceptacionNativaCT, persona bool) (ports.PreparacionConsultaAceptacionNativaCT, error) {
	x := q.Selector
	for _, ref := range []string{x.OrganizacionRef, x.ExpedienteRef, x.ClaveCT, x.UnidadRef, x.CategoriaRef, x.NecesidadRef, x.BolsaRef, x.LlamamientoRef, x.ReciboEmisionRef, x.ReciboRespuestaRef} {
		if !referenciaAceptacionNativaCT.MatchString(ref) {
			return ports.PreparacionConsultaAceptacionNativaCT{}, ports.ErrAceptacionNativaCTInvalida
		}
	}
	if q.ActorConfiable.Resultado.Validar() != nil || q.ActorConfiable.Vinculo.ValidarPara(q.ActorConfiable.Resultado) != nil {
		return ports.PreparacionConsultaAceptacionNativaCT{}, ports.ErrAceptacionNativaCTInvalida
	}
	actor, err := q.ActorConfiable.Resultado.Clonar()
	if err != nil {
		return ports.PreparacionConsultaAceptacionNativaCT{}, ports.ErrAceptacionNativaCTInvalida
	}
	q.ActorConfiable.Resultado = actor
	b, err := json.Marshal(struct {
		Esquema string `json:"esquema"`
		ports.SelectorAceptacionNativaCT
	}{esquemaAceptacionNativaCT(persona), x})
	if err != nil {
		return ports.PreparacionConsultaAceptacionNativaCT{}, ports.ErrAceptacionNativaCTInvalida
	}
	h := sha256.Sum256(b)
	tipo := ports.TipoRecursoAnclajeAceptacionCT
	if persona {
		tipo = ports.TipoRecursoPersonaAceptacionCT
	}
	r := core.RecursoAutorizable{Referencia: x.ReciboRespuestaRef, ModuloID: "bolsa", Tipo: tipo, Ambitos: map[string]string{"unidad_ref": x.UnidadRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if _, err = r.HuellaContextoAutorizacionSHA256(); err != nil {
		return ports.PreparacionConsultaAceptacionNativaCT{}, ports.ErrAceptacionNativaCTInvalida
	}
	return ports.PreparacionConsultaAceptacionNativaCT{Solicitud: q, MaterialCanonico: b, Recurso: r}, nil
}

func ValidarOrdenConsultaAceptacionNativaCT(o ports.OrdenConsultaAceptacionNativaCT, persona bool, ahora time.Time) error {
	p, err := PrepararConsultaAceptacionNativaCT(o.Solicitud, persona)
	if err != nil {
		return err
	}
	h, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	a := o.Material
	c := o.Solicitud.ActorConfiable.Resultado
	x := a.ResumenCapacidad()
	accion, audiencia := ports.AccionConsultaAnclajeAceptacionNativaCT, ports.AudienciaConsultaAnclajeAceptacionNativaCT
	if persona {
		accion, audiencia = ports.AccionConsultaPersonaAceptacionNativaCT, ports.AudienciaConsultaPersonaAceptacionNativaCT
	}
	if err != nil || a.ValidarEstructura() != nil || !instantePersonaAceptacionCT(ahora) || x.Operacion() != accion || x.AudienciaConsumo() != audiencia || x.EfectoRef() != p.Recurso.Referencia || x.EfectoHuellaSHA256() != h || x.ContextoRef() != c.RegistroContextoRef || x.ContextoHuellaSHA256() != c.HuellaSHA256 || !bytes.Equal(a.ContextoActorCanonico(), c.RepresentacionCanonica) || a.PersonaVersion() != c.Contexto.Instantanea.PersonaVersion || a.PerfilVersion() != c.Contexto.Instantanea.PerfilVersion || ahora.Before(x.EmitidaEn()) || !ahora.Before(x.ExpiraEn()) {
		return ports.ErrAceptacionNativaCTDenegada
	}
	return nil
}

func validarEvidenciaAceptacionNativaCT(o ports.OrdenConsultaAceptacionNativaCT, persona bool, e ports.EvidenciaConsultaPersonaAceptacionCT, ahora time.Time) error {
	x := o.Material.ResumenCapacidad()
	if ValidarOrdenConsultaAceptacionNativaCT(o, persona, ahora) != nil || !referenciaPersonaAceptacionCT.MatchString(e.DecisionRef) || e.DecisionRef != x.DecisionRef() || !huellaPersonaAceptacionCT.MatchString(e.ConsumoHuellaSHA256) || !referenciaPersonaAceptacionCT.MatchString(e.AuditoriaRef) || !instantePersonaAceptacionCT(e.ConsultadaEn) || e.ConsultadaEn.Before(x.EmitidaEn()) || !e.ConsultadaEn.Before(x.ExpiraEn()) || e.ConsultadaEn.After(ahora) {
		return ports.ErrAceptacionNativaCTNoDisponible
	}
	return nil
}

func validarAnclajeAceptacionNativaCT(a *ports.AnclajeAceptacionNativaCT, s ports.SelectorAceptacionNativaCT) bool {
	if a == nil || a.SelectorAceptacionNativaCT != s || a.VersionExpediente == 0 || a.VersionBolsa == 0 || a.VersionInstantanea == 0 || a.VersionPolitica == 0 || a.OrdenSeleccionado == 0 || a.RespondidaEn.IsZero() {
		return false
	}
	for _, ref := range []string{a.ActaRef, a.InstantaneaRef, a.PoliticaRef, a.ParticipacionRef, a.CandidatoRef, a.RespuestaRef} {
		if !referenciaAceptacionNativaCT.MatchString(ref) {
			return false
		}
	}
	for _, h := range []string{a.HuellaBolsaSHA256, a.HuellaInstantaneaSHA256, a.HuellaPoliticaSHA256} {
		if !huellaPersonaAceptacionCT.MatchString(h) {
			return false
		}
	}
	return true
}

func ValidarResultadoAnclajeAceptacionNativaCT(o ports.OrdenConsultaAceptacionNativaCT, r ports.ResultadoAnclajeAceptacionNativaCT, ahora time.Time) error {
	if validarEvidenciaAceptacionNativaCT(o, false, r.Evidencia, ahora) != nil {
		return ports.ErrAceptacionNativaCTNoDisponible
	}
	if r.Estado == "no_encontrada" && r.Anclaje == nil {
		return nil
	}
	if r.Estado != "acreditado" || !validarAnclajeAceptacionNativaCT(r.Anclaje, o.Solicitud.Selector) {
		return ports.ErrAceptacionNativaCTNoDisponible
	}
	return nil
}

func ValidarResultadoPersonaAceptacionNativaCT(o ports.OrdenConsultaAceptacionNativaCT, r ports.ResultadoPersonaAceptacionNativaCT, ahora time.Time) error {
	if validarEvidenciaAceptacionNativaCT(o, true, r.Evidencia, ahora) != nil {
		return ports.ErrAceptacionNativaCTNoDisponible
	}
	if r.Estado == "no_encontrada" && r.Aceptacion == nil && r.Persona == nil && r.Vinculo == nil {
		return nil
	}
	if !validarAnclajeAceptacionNativaCT(r.Aceptacion, o.Solicitud.Selector) {
		return ports.ErrAceptacionNativaCTNoDisponible
	}
	if r.Estado == "pendiente" && r.Persona == nil && r.Vinculo == nil {
		return nil
	}
	if r.Estado != "acreditado" || r.Persona == nil || r.Vinculo == nil || !personaAceptacionCT.MatchString(r.Persona.Ref) || r.Persona.Version <= 0 || r.Vinculo.Version <= 0 || r.Vinculo.ProcedenciaVersion <= 0 || !huellaPersonaAceptacionCT.MatchString(r.Vinculo.ProcedenciaSHA256) || !r.Vinculo.VigenteHasta.After(ahora) {
		return ports.ErrAceptacionNativaCTNoDisponible
	}
	return nil
}

type ServicioConsultaAceptacionNativaCT struct {
	proveedor   ports.ProveedorAutorizacionConsultaAceptacionNativaCT
	repositorio ports.RepositorioConsultaAceptacionNativaCT
	ahora       func() time.Time
}

func NuevoServicioConsultaAceptacionNativaCT(p ports.ProveedorAutorizacionConsultaAceptacionNativaCT, r ports.RepositorioConsultaAceptacionNativaCT, ahora func() time.Time) (*ServicioConsultaAceptacionNativaCT, error) {
	if nuloPersonaAceptacionCT(p) || nuloPersonaAceptacionCT(r) || ahora == nil {
		return nil, ports.ErrAceptacionNativaCTNoDisponible
	}
	return &ServicioConsultaAceptacionNativaCT{p, r, ahora}, nil
}

func (s *ServicioConsultaAceptacionNativaCT) orden(ctx context.Context, q ports.SolicitudConsultaAceptacionNativaCT, persona bool) (ports.OrdenConsultaAceptacionNativaCT, error) {
	var cero ports.OrdenConsultaAceptacionNativaCT
	if ctx == nil || s == nil || nuloPersonaAceptacionCT(s.proveedor) || nuloPersonaAceptacionCT(s.repositorio) || s.ahora == nil {
		return cero, ports.ErrAceptacionNativaCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	p, err := PrepararConsultaAceptacionNativaCT(q, persona)
	if err != nil {
		return cero, err
	}
	m, err := s.proveedor.AutorizarConsultaAceptacionNativaCT(ctx, p, persona)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	o := ports.OrdenConsultaAceptacionNativaCT{Solicitud: p.Solicitud, Material: m}
	if err = ValidarOrdenConsultaAceptacionNativaCT(o, persona, s.ahora().UTC().Truncate(time.Microsecond)); err != nil {
		return cero, err
	}
	return o, nil
}

func (s *ServicioConsultaAceptacionNativaCT) ConsultarAnclajeAceptacionNativaCT(ctx context.Context, q ports.SolicitudConsultaAceptacionNativaCT) (ports.ResultadoAnclajeAceptacionNativaCT, error) {
	var cero ports.ResultadoAnclajeAceptacionNativaCT
	o, err := s.orden(ctx, q, false)
	if err != nil {
		return cero, err
	}
	r, err := s.repositorio.ConsultarAnclajeAceptacionNativaCT(ctx, o)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	if err = ValidarResultadoAnclajeAceptacionNativaCT(o, r, s.ahora().UTC().Truncate(time.Microsecond)); err != nil {
		return cero, err
	}
	return r, nil
}

func (s *ServicioConsultaAceptacionNativaCT) ConsultarPersonaAceptacionNativaCT(ctx context.Context, q ports.SolicitudConsultaAceptacionNativaCT) (ports.ResultadoPersonaAceptacionNativaCT, error) {
	var cero ports.ResultadoPersonaAceptacionNativaCT
	o, err := s.orden(ctx, q, true)
	if err != nil {
		return cero, err
	}
	r, err := s.repositorio.ConsultarPersonaAceptacionNativaCT(ctx, o)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	if err = ValidarResultadoPersonaAceptacionNativaCT(o, r, s.ahora().UTC().Truncate(time.Microsecond)); err != nil {
		return cero, err
	}
	return r, nil
}

var _ ports.ConsultaAceptacionNativaCT = (*ServicioConsultaAceptacionNativaCT)(nil)

type ProveedorNominalConsultaAceptacionNativaCT struct {
	emisor      ports.AutorizadorSituacionParticipacionV3
	motivo      core.ReferenciaEntradaCatalogo
	correlacion func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error)
	ahora       func() time.Time
}

func NuevoProveedorNominalConsultaAceptacionNativaCT(e ports.AutorizadorSituacionParticipacionV3, m core.ReferenciaEntradaCatalogo, c func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error), ahora func() time.Time) (*ProveedorNominalConsultaAceptacionNativaCT, error) {
	if nuloPersonaAceptacionCT(e) || !core.ReferenciaMotivoAutorizacionV2Valida(m) || c == nil || ahora == nil {
		return nil, ports.ErrAceptacionNativaCTNoDisponible
	}
	return &ProveedorNominalConsultaAceptacionNativaCT{e, m, c, ahora}, nil
}

func (p *ProveedorNominalConsultaAceptacionNativaCT) AutorizarConsultaAceptacionNativaCT(ctx context.Context, entrada ports.PreparacionConsultaAceptacionNativaCT, persona bool) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || p == nil || nuloPersonaAceptacionCT(p.emisor) || p.correlacion == nil || p.ahora == nil {
		return cero, ports.ErrAceptacionNativaCTNoDisponible
	}
	preparada, err := PrepararConsultaAceptacionNativaCT(entrada.Solicitud, persona)
	if err != nil {
		return cero, err
	}
	correlacion, err := p.correlacion(ctx)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	accion, audiencia := ports.AccionConsultaAnclajeAceptacionNativaCT, ports.AudienciaConsultaAnclajeAceptacionNativaCT
	campos := []string{"anclaje"}
	if persona {
		accion, audiencia = ports.AccionConsultaPersonaAceptacionNativaCT, ports.AudienciaConsultaPersonaAceptacionNativaCT
		campos = []string{"aceptacion", "persona", "vinculo"}
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: preparada.Solicitud.ActorConfiable.Vinculo, ReferenciaMotivo: p.motivo, Accion: accion, Recurso: preparada.Recurso, Finalidad: ports.FinalidadConsultaPersonaAceptacionCT, Correlacion: correlacion})
	if err != nil {
		return cero, ports.ErrAceptacionNativaCTDenegada
	}
	c := preparada.Solicitud.ActorConfiable.Resultado
	d, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	concedida, _, resultadoErr := d.Resultado()
	ahora := p.ahora().UTC().Truncate(time.Microsecond)
	if d.ValidarPara(s) != nil || resultadoErr != nil {
		return cero, ports.ErrAceptacionNativaCTNoDisponible
	}
	if !concedida {
		return cero, ports.ErrAceptacionNativaCTDenegada
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, p.motivo, c)
	if err != nil || confirmacion.ValidarPara(orden) != nil {
		return cero, ports.ErrAceptacionNativaCTNoDisponible
	}
	restricciones, err := d.RestriccionesProyeccionPara(s)
	if err != nil || !reflect.DeepEqual(restricciones.CamposPermitidos, campos) || len(restricciones.Obligaciones) != 0 || !confirmacion.DentroDeVentanaEn(ahora) {
		return cero, ports.ErrAceptacionNativaCTDenegada
	}
	if nuloPersonaAceptacionCT(exportador) {
		return cero, ports.ErrAceptacionNativaCTNoDisponible
	}
	m, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return cero, ErrorConsultaPersonaAceptacionCT(ctx, err)
	}
	if !materialAutorizacionBorradorLlamamientoExacto(s, d, confirmacion, c, p.motivo, m, audiencia) {
		return cero, ports.ErrAceptacionNativaCTNoDisponible
	}
	if err = ValidarOrdenConsultaAceptacionNativaCT(ports.OrdenConsultaAceptacionNativaCT{Solicitud: preparada.Solicitud, Material: m}, persona, ahora); err != nil {
		return cero, err
	}
	return m, nil
}

var _ ports.ProveedorAutorizacionConsultaAceptacionNativaCT = (*ProveedorNominalConsultaAceptacionNativaCT)(nil)
