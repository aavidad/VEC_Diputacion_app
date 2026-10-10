package bootstrap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"reflect"
	"time"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func (p *perfilesNominalesIncorporacion) todos() []*perfilFijoCTDesarrollo {
	if p == nil {
		return nil
	}
	r := []*perfilFijoCTDesarrollo{p.detalle}
	if p.legadoCompuesto {
		r = append(r, p.alta, p.ct)
	}
	for _, d := range operacionesIncorporacionB2() {
		if perfil := p.b2[d.accion]; perfil != nil {
			existe := false
			for _, previo := range r {
				existe = existe || previo == perfil
			}
			if !existe {
				r = append(r, perfil)
			}
		}
	}
	return r
}
func (p *perfilesNominalesIncorporacion) contiene(perfil *perfilFijoCTDesarrollo) bool {
	for _, p := range p.todos() {
		if p == perfil {
			return true
		}
	}
	return false
}
func extenderPerfilesNominalesB2(p *perfilesNominalesIncorporacion, refs ReferenciasCTIncorporacionDesarrollo, c *archivoIncorporacionPersonalB2, ahora time.Time) error {
	if c == nil {
		return nil
	}
	if p == nil || p.soporte == nil || !refs.valida() || c.Protocolo != "personal_b2_v1" || c.OrganismoRef == "" {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	p.b2 = map[string]*perfilFijoCTDesarrollo{}
	s := p.soporte
	base, e := s.contexto.Vinculo.Datos()
	if e != nil {
		return e
	}
	for _, modulo := range gruposPerfilesIncorporacionB2() {
		if modulo == grupoRegistroVinculoRPTB2 && !operacionConfiguradaB2(c, claveRegistroVinculoRPTB2) ||
			modulo == grupoListadoCategoriasRPTB2 && !operacionConfiguradaB2(c, claveListadoCategoriasRPTB2) ||
			modulo == grupoCatalogoEmpleadoB2 && !operacionConfiguradaB2(c, claveCatalogoPublicarB2) {
			continue
		}
		clave := "incorporacion_b2_" + modulo
		ctx, e := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principalCanalNominalIncorporacion(s), ahora, discriminadorPerfilFijoCTDesarrollo(clave))
		if e != nil {
			return e
		}
		v, e := ctx.Vinculo.Datos()
		if e != nil || v.CuentaRef != base.CuentaRef || v.PrincipalID != base.PrincipalID || ctx.Resultado.Contexto.PersonaRef != s.contexto.Resultado.Contexto.PersonaRef {
			return ct.ErrComposicionIncorporacionAplicacion
		}
		ambitos := []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{refs.OrganizacionRef}}}
		if modulo == "ct155" {
			ambitos = append(ambitos, core.AmbitoPerfil{Clave: "unidad_ref", Valores: []string{refs.UnidadRef}})
		}
		// Competencia sobre todo el organismo (consenso B2 del 10/10): plan
		// (preparar, consultar, ejecutar, confirmar, seleccionar), vacantes,
		// alta, hecho, ficha y publicar/retirar del catálogo. El objetivo
		// concreto va en los atributos firmados del recurso y Personal41-44
		// lo cotejan; la unidad, versión y estado del expediente los sigue
		// comprobando la SQL de cada acto. Las lecturas de régimen/modalidad
		// y clases conservan sus perfiles con objetivo_ref cerrado.
		if modulo == "personal" || modulo == grupoCatalogoEmpleadoB2 {
			ambitos = []core.AmbitoPerfil{{Clave: "organismo_ref", Valores: []string{c.OrganismoRef}}}
		}
		if objetivos := objetivosLecturaPersonalB2(modulo, c.OrganismoRef); objetivos != nil {
			ambitos = []core.AmbitoPerfil{{Clave: "organismo_ref", Valores: []string{c.OrganismoRef}}, {Clave: "objetivo_ref", Valores: objetivos}}
		}
		if modulo == "bolsa" {
			ambitos = []core.AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{refs.UnidadRef}}}
		}
		if modulo == "rpt_catalogo" || modulo == "rpt_uso" || modulo == grupoListadoCategoriasRPTB2 {
			ambitos = []core.AmbitoPerfil{{Clave: "catalogo_id", Valores: []string{c.CatalogoRPTID}}, {Clave: "modulo_id", Valores: []string{c.ModuloRPTID}}}
		}
		if modulo == "rpt_uso" {
			ambitos = append(ambitos, core.AmbitoPerfil{Clave: "consumidor", Valores: []string{"personal"}})
		}
		roles := []core.ConcesionRol{}
		for _, d := range operacionesIncorporacionB2() {
			if d.accion != ct.AccionConsultarDetalleRRHH && grupoOperacionIncorporacionB2(d) == modulo {
				moduloRol := d.modulo
				if modulo == "rpt_catalogo" || modulo == "rpt_uso" || modulo == grupoListadoCategoriasRPTB2 {
					moduloRol = c.ModuloRPTID
				}
				concesion := core.ConcesionRol{Accion: d.accion, ModuloID: moduloRol, TipoRecurso: d.tipo, Finalidades: []string{d.finalidad}, GarantiaMinima: core.AuthAssuranceHigh}
				concesion.CamposPermitidos = camposConcesionB2(d.clave)
				roles = append(roles, concesion)
			}
		}
		plantilla, e := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora, clave, clave, clave, roles, ambitos)
		if e != nil {
			return e
		}
		perfil := &perfilFijoCTDesarrollo{clave: clave, contexto: ctx, plantilla: plantilla, actoSesion: actoSesionPerfilFijoCTDesarrollo}
		for _, d := range operacionesIncorporacionB2() {
			if d.accion != ct.AccionConsultarDetalleRRHH && grupoOperacionIncorporacionB2(d) == modulo {
				p.b2[d.accion] = perfil
			}
		}
	}
	p.b2[ct.AccionConsultarDetalleRRHH] = p.detalle
	return nil
}

// camposConcesionB2 da, por operación B2, la lista exacta de campos que el
// núcleo V3 instalado (consumir_decision_mutacion_v3_interna, AD3-233) exige
// en la decisión; sin ella PostgreSQL rechaza el consumo. La prueba
// TestIncorporacionB2CamposCoincidenConNucleoSQL la coteja con la SQL.
func camposConcesionB2(clave string) []string {
	switch clave {
	case "bolsa_anclaje":
		return []string{"anclaje"}
	case "bolsa_persona":
		return []string{"aceptacion", "persona", "vinculo"}
	case "personal_clases":
		return []string{"catalogo", "evidencia"}
	case "personal_plan_preparar", "personal_plan_consultar", "personal_plan_ejecutar", "personal_plan_confirmar":
		return []string{"ejecucion_huella_sha256", "ejecucion_recibo_ref", "estado", "evidencia", "plan", "recibo_alta_relacion", "recibo_ocupacion"}
	case "personal_plan_seleccionar":
		return []string{"evidencia", "seleccion"}
	case "personal_vacantes":
		return []string{"cobertura", "corte", "cursor", "cursor_siguiente", "evidencia", "limite", "organismo_ref", "vacantes"}
	case "personal_catalogos":
		return []string{"cursor_siguiente", "entradas", "evidencia", "organismo_ref"}
	case "personal_alta":
		return []string{"eficacia_administrativa", "empleado_ref", "evidencia", "firma_oficial", "persona_ref", "proyeccion_ref", "recibo", "relacion_ref", "version"}
	case "personal_hecho":
		return []string{"eficacia_administrativa", "empleado_ref", "evidencia", "firma_oficial", "hecho_ref", "recibo", "relacion_ref", "tipo", "version"}
	case "personal_ficha":
		return []string{"corte", "eficacia_administrativa", "empleado_ref", "evidencia", "firma_oficial", "ocupaciones", "organismo_ref", "persona_ref", "relaciones", "servicios", "situaciones", "version"}
	case "ct_vinculo_consultar":
		return []string{"analisis", "vinculo"}
	case "ct_plan_consultar":
		return []string{"plan"}
	case "ct_plan_preparar", "ct_origen_confirmar", claveRegistroVinculoRPTB2:
		return []string{"recibo"}
	case "rpt_publicacion":
		return []string{"control_actual", "entrada", "publicacion"}
	case claveCatalogoPublicarB2, claveCatalogoRetirarB2:
		return []string{"entrada", "recibo"}
	case claveListadoCategoriasRPTB2:
		return []string{"categorias", "paginacion", "publicaciones"}
	case "rpt_uso_consultar":
		return []string{"uso"}
	case "rpt_reservar", "rpt_confirmar":
		return []string{"recibo", "uso"}
	}
	return nil
}

// Un emisor por operación, ligado al PDP común y al material de confianza
// existente. Las negativas no se convierten en indisponibilidad ni viceversa.
type operacionAutorizadaIncorporacionB2 struct {
	descriptor descriptorOperacionIncorporacionB2
	motivo     core.ReferenciaEntradaCatalogo
	pdp        vp.AutorizadorSolicitudLigadaV3
	emisor     *confianza.EmisorCapacidadesAtestacionAutorizacionV3
}
type autoridadIncorporacionPersonalB2 struct {
	perfiles                   *perfilesNominalesIncorporacion
	operaciones                map[string]operacionAutorizadaIncorporacionB2
	material                   *proveedorMaterialAltaContratacionTemporalDesarrollo
	organismoRef               string
	rptPool                    *pgxpool.Pool
	preparadorUsosRPT          vp.PreparadorUsosCategoriaRPT
	catalogoRPTID, moduloRPTID string
	reloj                      ct.Reloj
}

func (a *autoridadIncorporacionPersonalB2) contexto(ctx context.Context, accion string) (ct.ContextoAutorizacionAltaV3, error) {
	if a == nil || a.perfiles == nil || ctx == nil || !operacionPermitidaEnRutaIncorporacionB2(ctx, accion) {
		return ct.ContextoAutorizacionAltaV3{}, ct.ErrAutorizacionDenegada
	}
	p := a.perfiles.b2[accion]
	if p == nil {
		return ct.ContextoAutorizacionAltaV3{}, ct.ErrAutorizacionDenegada
	}
	c, e := a.perfiles.resolver(ctx, p)
	if e != nil {
		return c, e
	}
	if e = a.perfiles.consumir(ctx, p); e != nil {
		return ct.ContextoAutorizacionAltaV3{}, e
	}
	return c, nil
}
func (a *autoridadIncorporacionPersonalB2) actor(ctx context.Context, accion string) (core.ContextoActor, error) {
	c, e := a.contexto(ctx, accion)
	if e != nil {
		return core.ContextoActor{}, e
	}
	return c.Resultado.Contexto.Clonar()
}
func (a *autoridadIncorporacionPersonalB2) emitirRecurso(ctx context.Context, accion string, r core.RecursoAutorizable) (core.SolicitudAutorizacionLigadaV3, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if a == nil || ctx == nil || a.material == nil || a.reloj == nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, ct.ErrConsultaRRHHNoDisponible
	}
	op, ok := a.operaciones[accion]
	if !ok || op.descriptor.accion != accion || r.ModuloID != op.descriptor.modulo || r.Tipo != op.descriptor.tipo {
		return core.SolicitudAutorizacionLigadaV3{}, cero, ct.ErrAutorizacionDenegada
	}
	if p := a.perfiles.b2[accion]; p != nil && (p.clave == "incorporacion_b2_personal" || p.clave == "incorporacion_b2_"+grupoCatalogoEmpleadoB2 ||
		objetivosLecturaPersonalB2(grupoOperacionIncorporacionB2(op.descriptor), a.organismoRef) != nil) && r.Ambitos["organismo_ref"] != a.organismoRef {
		return core.SolicitudAutorizacionLigadaV3{}, cero, ct.ErrAutorizacionDenegada
	}
	c, e := a.contexto(ctx, accion)
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	cor, e := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	s, e := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: c.Vinculo, ReferenciaMotivo: op.motivo, Accion: accion, Recurso: r, Finalidad: op.descriptor.finalidad, Correlacion: cor})
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	d, conf, e := op.pdp.ExigirSolicitudLigadaV3(ctx, s, c.Resultado)
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	orden, e := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, op.motivo, c.Resultado)
	if e != nil || conf.ValidarPara(orden) != nil || !conf.DentroDeVentanaEn(a.reloj.Ahora()) {
		return core.SolicitudAutorizacionLigadaV3{}, cero, ct.ErrAutorizacionDenegada
	}
	// La asignación se relee después del PDP, inmediatamente antes de emitir.
	if e = a.perfiles.consumir(ctx, a.perfiles.b2[accion]); e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	at, e := a.material.atestador.Atestar(ctx, d, op.motivo, c.Resultado)
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	prueba, e := a.material.Verificar(ctx, s, d, op.motivo, c.Resultado, at)
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	cap, e := op.emisor.Emitir(ctx, s, d, op.motivo, c.Resultado, at, prueba)
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	m, e := confianza.NuevoMaterialConsumoAutorizacionAtestadaV3(s, d, op.motivo, c.Resultado, at, prueba, cap, a.material.raiz)
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	x, e := m.ExportarMaterialParaConsumidor()
	return s, x, e
}
func (a *autoridadIncorporacionPersonalB2) autorizarRecurso(ctx context.Context, accion string, r core.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	_, x, e := a.emitirRecurso(ctx, accion, r)
	return x, e
}

func (a *autoridadIncorporacionPersonalB2) actorCoincide(ctx context.Context, accion string, actor core.ContextoActor) error {
	actual, e := a.actor(ctx, accion)
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(actual, actor) {
		return ct.ErrAutorizacionDenegada
	}
	return nil
}
func (a *autoridadIncorporacionPersonalB2) AutorizarPlanIncorporacionCT(ctx context.Context, m personal.MaterialPlanIncorporacionCT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if e := a.actorCoincide(ctx, m.Accion(), m.Actor()); e != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	return a.autorizarRecurso(ctx, m.Accion(), m.Recurso())
}
func (a *autoridadIncorporacionPersonalB2) AutorizarConsultaRegistroEmpleadoB2(ctx context.Context, m personal.MaterialConsultaRegistroEmpleadoB2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	accion := personal.AccionFichaEmpleadoB2
	if m.Operacion() == "vacantes" {
		accion = personal.AccionVacantesB2
	}
	if e := a.actorCoincide(ctx, accion, m.Actor()); e != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	return a.autorizarRecurso(ctx, accion, m.Recurso())
}

// AutorizarCatalogoRegistroEmpleadoB2 consulta con el perfil propio de lectura
// del catálogo (régimen y modalidad) y
// publica o retira con el perfil nominal propio del gobierno del catálogo.
// Sin esas operaciones en la configuración privada no hay perfil y se deniega.
func (a *autoridadIncorporacionPersonalB2) AutorizarCatalogoRegistroEmpleadoB2(ctx context.Context, m personal.MaterialCatalogoEmpleadoB2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	accion, ok := accionCatalogoEmpleadoB2(m.Operacion())
	if !ok || a == nil || a.perfiles == nil || a.perfiles.b2[accion] == nil || m.OrganismoRef() != a.organismoRef {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, negativaGobiernoCatalogoEmpleadoB2(accion, ct.ErrAutorizacionDenegada)
	}
	if e := a.actorCoincide(ctx, accion, m.Actor()); e != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, negativaGobiernoCatalogoEmpleadoB2(accion, e)
	}
	x, e := a.autorizarRecurso(ctx, accion, m.Recurso())
	return x, negativaGobiernoCatalogoEmpleadoB2(accion, e)
}
func (a *autoridadIncorporacionPersonalB2) AutorizarActoRegistroEmpleadoB2(ctx context.Context, m personal.MaterialActoRegistroEmpleadoB2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	accion := personal.AccionAltaEmpleadoB2
	if m.Tipo() == "hecho" {
		accion = personal.AccionHechoEmpleadoB2
	}
	if e := a.actorCoincide(ctx, accion, m.Actor()); e != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	return a.autorizarRecurso(ctx, accion, m.Recurso())
}
func errorAutorizacionIncorporacionB2(err error) error {
	if errors.Is(err, ct.ErrAutorizacionDenegada) || errors.Is(err, core.ErrAutorizacionDenegada) {
		return ct.ErrAutorizacionDenegada
	}
	return errors.Join(ct.ErrConsultaRRHHNoDisponible, err)
}

func (a *autoridadIncorporacionPersonalB2) ActorPreparacionPlanB2(ctx context.Context) (core.ContextoActor, error) {
	return a.actor(ctx, "personal.plan_incorporacion_ct.preparar")
}
func (a *autoridadIncorporacionPersonalB2) ActorConsultaPlanB2(ctx context.Context) (core.ContextoActor, error) {
	return a.actor(ctx, "personal.plan_incorporacion_ct.consultar")
}
func (a *autoridadIncorporacionPersonalB2) ActorEjecucionPlanB2(ctx context.Context) (core.ContextoActor, error) {
	return a.actor(ctx, "personal.plan_incorporacion_ct.ejecutar")
}
func (a *autoridadIncorporacionPersonalB2) ActorLecturaHechosB2(ctx context.Context) (core.ContextoActor, error) {
	return a.actor(ctx, personal.AccionFichaEmpleadoB2)
}
func (a *autoridadIncorporacionPersonalB2) ActorHechoB2(ctx context.Context) (core.ContextoActor, error) {
	return a.actor(ctx, personal.AccionHechoEmpleadoB2)
}

func asignarPerfilesNominalesB2EnFronteras(s *soporteAltaContratacionTemporalDesarrollo, declaraciones []descriptorFronteraComunDesarrollo) ([]descriptorFronteraComunDesarrollo, error) {
	if s == nil {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	// B2 puro crea el perfil de detalle sin los perfiles de alta y confirmación
	// del protocolo anterior. La subconsulta de detalle necesita ese perfil.
	detalle, e := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principalCanalNominalIncorporacion(s), s.reloj.Ahora(), discriminadorPerfilFijoCTDesarrollo(claveIncorporacionDetalle))
	if e != nil {
		return nil, e
	}
	perfilDetalle := detalle.Resultado.Contexto.PerfilActivoRef
	ids := []string{}
	for _, grupo := range gruposPerfilesIncorporacionB2() {
		c, e := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principalCanalNominalIncorporacion(s), s.reloj.Ahora(), discriminadorPerfilFijoCTDesarrollo("incorporacion_b2_"+grupo))
		if e != nil {
			return nil, e
		}
		ids = append(ids, c.Resultado.Contexto.PerfilActivoRef)
	}
	r := append([]descriptorFronteraComunDesarrollo(nil), declaraciones...)
	detalleEncontrado := false
	for i := range r {
		if r[i].Ruta == httpct.RutaConsultaDetalleRRHH && r[i].Metodo == http.MethodPost {
			if !esDescriptorDetalleContratacionTemporalDesarrollo(r[i], s.contexto.Resultado.Contexto.PerfilActivoRef) || detalleEncontrado {
				return nil, ct.ErrComposicionIncorporacionAplicacion
			}
			if !r[i].admitePerfil(perfilDetalle) {
				r[i].PerfilesActivosRef = append(append([]string(nil), r[i].PerfilesActivosRef...), perfilDetalle)
			}
			detalleEncontrado = true
		}
		if r[i].Ruta == httpct.RutaConsultaDetalleRRHH && r[i].Metodo == http.MethodPost ||
			r[i].Ruta == httpct.RutaPlanB2 && (r[i].Metodo == http.MethodGet || r[i].Metodo == http.MethodPost) ||
			r[i].Ruta == httpct.RutaConfirmacionB2 && r[i].Metodo == http.MethodPost ||
			r[i].Ruta == httpct.RutaVinculoCategoriaRPTB2 && (r[i].Metodo == http.MethodGet || r[i].Metodo == http.MethodPost) ||
			r[i].Ruta == httpct.RutaCategoriasRPTB2 && r[i].Metodo == http.MethodGet ||
			r[i].Ruta == rutaCatalogosRegistroEmpleadoB2 && r[i].Metodo == http.MethodPost {
			r[i].PerfilesActivosRef = append(append([]string(nil), r[i].PerfilesActivosRef...), ids...)
		}
	}
	if !detalleEncontrado {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return r, nil
}

func (a *autoridadIncorporacionPersonalB2) AutorizarPlanNominalB2(ctx context.Context, accion string, b []byte, actor core.ContextoActor) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var selector struct {
		OrganizacionRef string                             `json:"organizacion_ref"`
		ExpedienteRef   string                             `json:"expediente_ref"`
		UnidadRef       string                             `json:"unidad_ref"`
		UnidadCTRef     string                             `json:"unidad_ct_ref"`
		Material        *domct.PlanIncorporacionPersonalB2 `json:"material"`
		Solicitud       *ct.SolicitudPlanNominalB2         `json:"solicitud"`
	}
	if len(b) == 0 || len(b) > 65536 || json.Unmarshal(b, &selector) != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrPlanNominalB2Invalido
	}
	if accion != ct.AccionRegistrarPlanNominalB2 && accion != ct.AccionLeerPlanNominalB2 && accion != ct.AccionConfirmarOrigenB2 {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrAutorizacionDenegada
	}
	if selector.Solicitud != nil {
		selector.OrganizacionRef, selector.ExpedienteRef = selector.Solicitud.OrganizacionRef, selector.Solicitud.ExpedienteRef
	}
	if selector.UnidadRef == "" {
		selector.UnidadRef = selector.UnidadCTRef
	}
	if selector.Material != nil {
		selector.UnidadRef = selector.Material.UnidadCTRef
	}
	perfil := a.perfiles.b2[accion]
	if perfil == nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrAutorizacionDenegada
	}
	permitida := false
	for _, amb := range perfil.plantilla.AsignacionPerfil.Ambitos {
		if amb.Clave == "organizacion_ref" {
			for _, v := range amb.Valores {
				permitida = permitida || v == selector.OrganizacionRef
			}
		}
	}
	if !permitida {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrAutorizacionDenegada
	}
	if accion != ct.AccionLeerPlanNominalB2 {
		if e := a.actorCoincide(ctx, accion, actor); e != nil {
			return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
		}
	}
	return a.autorizarRecurso(ctx, accion, recursoVinculoCategoriaRPT(selector.ExpedienteRef, ct.ModuloContratacion, ct.TipoRecursoPlanNominalB2, map[string]string{"organizacion_ref": selector.OrganizacionRef, "unidad_ref": selector.UnidadRef}, b))
}

func (a *autoridadIncorporacionPersonalB2) ConsultaCT(ctx context.Context, c ct.ConsultaVinculoCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	b, e := c.Canonico()
	if e != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	return a.autorizarRecurso(ctx, ct.AccionConsultarVinculoCategoriaRPT, recursoVinculoCategoriaRPT(c.ExpedienteRef, ct.ModuloContratacion, "vinculo_categoria_rpt_ct", map[string]string{"organizacion_ref": c.OrganizacionRef}, b))
}

// RegistroCT autoriza el acto CT154 con el perfil nominal propio del registro,
// distinto del que lee la publicación RPT, como exige registrar_vinculo_categoria_rpt_v1.
// Sin la operación en la configuración privada no hay perfil y se deniega.
func (a *autoridadIncorporacionPersonalB2) RegistroCT(ctx context.Context, m ct.RegistroVinculoCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if a == nil || a.perfiles == nil || a.perfiles.b2[ct.AccionRegistrarVinculoCategoriaRPT] == nil ||
		m.CatalogoID != a.catalogoRPTID || m.ModuloID != a.moduloRPTID {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrAutorizacionDenegada
	}
	b, e := m.Canonico()
	if e != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	defer clear(b)
	return a.autorizarRecurso(ctx, ct.AccionRegistrarVinculoCategoriaRPT, recursoVinculoCategoriaRPT(m.ExpedienteRef, ct.ModuloContratacion, "vinculo_categoria_rpt_ct", map[string]string{"organizacion_ref": m.OrganizacionRef}, b))
}
func (a *autoridadIncorporacionPersonalB2) LecturaRPT(ctx context.Context, p domct.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if a == nil || p.Validar() != nil || p.CatalogoID != a.catalogoRPTID || p.ModuloID != a.moduloRPTID {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrAutorizacionDenegada
	}
	b, e := pgct.MaterialPublicacionCategoriaRPTPostgreSQL(ctx, a.rptPool, p)
	if e != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e
	}
	defer clear(b)
	return a.autorizarRecurso(ctx, "vec.catalogos.categorias.consultar_historica", recursoVinculoCategoriaRPT(p.CatalogoID, p.ModuloID, "catalogo_configurable", map[string]string{"catalogo_id": p.CatalogoID, "modulo_id": p.ModuloID}, b))
}

func (a *autoridadIncorporacionPersonalB2) ExigirSolicitudLigadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	vacia, confirmacion := core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	d, e := s.Datos()
	if a == nil || e != nil {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	op, ok := a.operaciones[d.Accion]
	if !ok || d.Finalidad != op.descriptor.finalidad || d.ReferenciaMotivo != op.motivo || d.Recurso.ModuloID != op.descriptor.modulo || d.Recurso.Tipo != op.descriptor.tipo {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	actual, e := a.contexto(ctx, d.Accion)
	if e != nil {
		return vacia, confirmacion, e
	}
	if !d.VinculoAutenticacionActor.CoincideExactamenteCon(actual.Vinculo) || d.VinculoAutenticacionActor.ValidarPara(resultado) != nil || !reflect.DeepEqual(actual.Resultado.Contexto, resultado.Contexto) {
		return vacia, confirmacion, ct.ErrAutorizacionDenegada
	}
	return op.pdp.ExigirSolicitudLigadaV3(ctx, s, resultado)
}
func (a *autoridadIncorporacionPersonalB2) emisorMaterial(accion string) (*confianza.EmisorMaterialAutorizacionAtestadaV3, error) {
	if a == nil || a.material == nil {
		return nil, ct.ErrConsultaRRHHNoDisponible
	}
	op, ok := a.operaciones[accion]
	if !ok {
		return nil, ct.ErrAutorizacionDenegada
	}
	return confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(a, a.material.atestador, a.material.confianza, op.emisor)
}

// La lectura del uso tiene su propio contrato; las mutaciones emplean el
// material preparado por el gestor RPT que después consumirá la autorización.
func (a *autoridadIncorporacionPersonalB2) materialConsultaUsoRPT(ctx context.Context, plan personal.PlanIncorporacionCT) (core.SolicitudAutorizacionLigadaV3, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if a == nil || plan.Validar() != nil || plan.Datos.CatalogoRPTID != a.catalogoRPTID || plan.Datos.CatalogoRPTModulo != a.moduloRPTID {
		return core.SolicitudAutorizacionLigadaV3{}, cero, ct.ErrAutorizacionDenegada
	}
	d := plan.Datos
	m := map[string]any{"catalogo_id": d.CatalogoRPTID, "modulo_id": d.CatalogoRPTModulo, "consumidor": "personal", "uso_ref": plan.UsoRPTRef, "reserva_recibo_ref": plan.ReservaRPTRef}
	b, e := json.Marshal(m)
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	defer clear(b)
	var sha string
	// La lectura usa la huella de su consulta propia, independiente del efecto.
	if e = a.rptPool.QueryRow(ctx, "SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to($1::jsonb::text,'UTF8')),'hex')", string(b)).Scan(&sha); e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, errors.Join(ct.ErrConsultaRRHHNoDisponible, e)
	}
	r := core.RecursoAutorizable{Referencia: plan.UsoRPTRef, ModuloID: a.moduloRPTID, Tipo: "uso_categoria", Ambitos: map[string]string{"catalogo_id": a.catalogoRPTID, "modulo_id": a.moduloRPTID, "consumidor": "personal"}, Atributos: map[string]string{"material_sha256": sha}}
	return a.emitirRecurso(ctx, "vec.catalogos.categorias.consultar_uso", r)
}
func (a *autoridadIncorporacionPersonalB2) AutorizarReservaPlanB2(ctx context.Context, p pp.PlanIncorporacionCT, m vp.MaterialReservaUsoCategoriaRPT) (vp.OrdenReservaUsoCategoriaRPT, error) {
	if a == nil || a.preparadorUsosRPT == nil || p.Datos.CatalogoRPTID != a.catalogoRPTID || p.Datos.CatalogoRPTModulo != a.moduloRPTID || !materialReservaCorrespondePlanB2(p, m) {
		return vp.OrdenReservaUsoCategoriaRPT{}, ct.ErrAutorizacionDenegada
	}
	preparada, e := a.preparadorUsosRPT.PrepararReservaUsoCategoriaRPT(ctx, m)
	if e != nil {
		return vp.OrdenReservaUsoCategoriaRPT{}, e
	}
	const accion = "vec.catalogos.categorias.reservar_uso"
	if !a.preparacionUsoRPTValida(accion, m.UsoRef, preparada) {
		return vp.OrdenReservaUsoCategoriaRPT{}, ct.ErrAutorizacionDenegada
	}
	s, x, e := a.emitirRecurso(ctx, accion, preparada.Recurso)
	return vp.OrdenReservaUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: x}, e
}
func (a *autoridadIncorporacionPersonalB2) AutorizarConfirmacionPlanB2(ctx context.Context, p pp.PlanIncorporacionCT, m vp.MaterialTerminalUsoCategoriaRPT) (vp.OrdenConfirmacionUsoCategoriaRPT, error) {
	if a == nil || a.preparadorUsosRPT == nil || p.Datos.CatalogoRPTID != a.catalogoRPTID || p.Datos.CatalogoRPTModulo != a.moduloRPTID || !materialReservaCorrespondePlanB2(p, m.Reserva) || m.TerminalReciboRef != p.ConfirmacionRPTRef {
		return vp.OrdenConfirmacionUsoCategoriaRPT{}, ct.ErrAutorizacionDenegada
	}
	preparada, e := a.preparadorUsosRPT.PrepararConfirmacionUsoCategoriaRPT(ctx, m)
	if e != nil {
		return vp.OrdenConfirmacionUsoCategoriaRPT{}, e
	}
	const accion = "vec.catalogos.categorias.confirmar_uso"
	if !a.preparacionUsoRPTValida(accion, m.Reserva.UsoRef, preparada) {
		return vp.OrdenConfirmacionUsoCategoriaRPT{}, ct.ErrAutorizacionDenegada
	}
	s, x, e := a.emitirRecurso(ctx, accion, preparada.Recurso)
	return vp.OrdenConfirmacionUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: x}, e
}
func (a *autoridadIncorporacionPersonalB2) preparacionUsoRPTValida(accion, usoRef string, p vp.PreparacionAutorizacionUsoCategoriaRPT) bool {
	if a == nil {
		return false
	}
	op, ok := a.operaciones[accion]
	r := p.Recurso
	huella := r.Atributos["material_sha256"]
	bytes, err := hex.DecodeString(huella)
	return ok && op.descriptor.accion == accion && p.Accion == accion &&
		p.Finalidad == op.descriptor.finalidad && p.AudienciaConsumo == op.descriptor.audiencia &&
		r.Referencia == usoRef && r.ModuloID == a.moduloRPTID && r.Tipo == op.descriptor.tipo &&
		len(r.Ambitos) == 3 && r.Ambitos["catalogo_id"] == a.catalogoRPTID &&
		r.Ambitos["modulo_id"] == a.moduloRPTID && r.Ambitos["consumidor"] == "personal" &&
		len(r.Atributos) == 1 && len(huella) == 64 && err == nil && len(bytes) == 32 && hex.EncodeToString(bytes) == huella &&
		r.Validar() == nil
}
func materialReservaCorrespondePlanB2(p pp.PlanIncorporacionCT, m vp.MaterialReservaUsoCategoriaRPT) bool {
	return p.Validar() == nil && m.Consumidor == "personal" && m.UsoRef == p.UsoRPTRef && m.CategoriaID == p.Datos.CatalogoRPTCategoria && m.ReservaReciboRef == p.ReservaRPTRef && m.Publicacion == (vp.ReferenciaPublicacionRPT{CatalogoID: p.Datos.CatalogoRPTID, Version: int(p.Datos.CatalogoRPTVersion), HuellaSHA256: p.Datos.CatalogoRPTHuellaSHA256})
}

// grupoRegistroVinculoRPTB2 tiene un perfil propio: el registro CT154 no
// amplía la asignación ya publicada del perfil que sólo consulta el vínculo.
// El listado de categorías RPT tampoco amplía el perfil que lee una
// publicación concreta; tiene el suyo.
const (
	grupoRegistroVinculoRPTB2   = "ct_vinculo_registro"
	grupoListadoCategoriasRPTB2 = "rpt_listado"
)

func gruposPerfilesIncorporacionB2() []string {
	return []string{"ct155", ct.ModuloContratacion, "personal", "bolsa", "rpt_catalogo", "rpt_uso", grupoRegistroVinculoRPTB2, grupoListadoCategoriasRPTB2, grupoCatalogoEmpleadoB2, grupoCatalogoConsultaB2, grupoClasesOcupacionB2}
}

// Las dos lecturas de Personal que hace el plan B2 al preparar sus opciones
// llevan en el recurso, además del organismo, un ámbito objetivo_ref (el
// catálogo «organismo:tipo» y el organismo para las clases). El PDP exige que
// la asignación enumere exactamente las mismas dimensiones (Cubre), así que
// cada una tiene perfil propio con sus valores cerrados; en el perfil común
// de Personal, que sólo fija organismo_ref, se denegaban por ámbito.
const (
	grupoCatalogoConsultaB2 = "personal_catalogo_consulta"
	grupoClasesOcupacionB2  = "personal_clases"
	accionClasesOcupacionB2 = "personal.plan_incorporacion_ct.clases_ocupacion"
)

// objetivosLecturaPersonalB2 da los objetivo_ref que admite cada grupo de
// lectura de Personal; nil para los grupos sin esa dimensión.
func objetivosLecturaPersonalB2(grupo, organismo string) []string {
	switch grupo {
	case grupoCatalogoConsultaB2:
		// El plan sólo lee régimen y modalidad (fuentes_incorporacion_personal_b2).
		return []string{organismo + ":regimen", organismo + ":modalidad"}
	case grupoClasesOcupacionB2:
		return []string{organismo}
	}
	return nil
}
func grupoOperacionIncorporacionB2(d descriptorOperacionIncorporacionB2) string {
	if d.accion == personal.AccionConsultarCatalogoEmpleadoB2 {
		return grupoCatalogoConsultaB2
	}
	if d.accion == accionClasesOcupacionB2 {
		return grupoClasesOcupacionB2
	}
	if d.accion == ct.AccionRegistrarVinculoCategoriaRPT {
		return grupoRegistroVinculoRPTB2
	}
	if d.accion == accionListarCategoriasRPTB2 {
		return grupoListadoCategoriasRPTB2
	}
	if d.accion == personal.AccionPublicarCatalogoEmpleadoB2 || d.accion == personal.AccionRetirarCatalogoEmpleadoB2 {
		return grupoCatalogoEmpleadoB2
	}
	if d.modulo == ct.ModuloContratacion && d.accion != ct.AccionConsultarVinculoCategoriaRPT {
		return "ct155"
	}
	if d.modulo == "rpt" {
		if d.clave == "rpt_publicacion" {
			return "rpt_catalogo"
		}
		return "rpt_uso"
	}
	return d.modulo
}
func operacionPermitidaEnRutaIncorporacionB2(ctx context.Context, accion string) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	ruta, ok := ctx.Value(claveRutaPeticionIncorporacionB2{}).(rutaPeticionIncorporacionB2)
	if !ok {
		return false
	}
	if ruta.ruta == httpct.RutaConfirmacionB2 && ruta.metodo == "POST" {
		_, ok := descriptorIncorporacionB2(accion)
		return ok && accion != ct.AccionRegistrarVinculoCategoriaRPT && accion != accionListarCategoriasRPTB2 && !accionGobiernoCatalogoEmpleadoB2(accion)
	}
	if ruta.ruta == rutaCatalogosRegistroEmpleadoB2 {
		// Sólo publicar o retirar una entrada; la consulta va por el plan.
		return ruta.metodo == "POST" && accionGobiernoCatalogoEmpleadoB2(accion)
	}
	if ruta.ruta == httpct.RutaCategoriasRPTB2 {
		// La pantalla del vínculo sólo lista las categorías publicadas.
		return ruta.metodo == "GET" && accion == accionListarCategoriasRPTB2
	}
	if ruta.ruta == httpct.RutaVinculoCategoriaRPTB2 {
		// GET sólo consulta el vínculo; POST lee la publicación RPT y registra.
		if ruta.metodo == "GET" {
			return accion == ct.AccionConsultarVinculoCategoriaRPT
		}
		return ruta.metodo == "POST" && (accion == ct.AccionRegistrarVinculoCategoriaRPT || accion == ct.AccionConsultarPublicacionCategoriaRPT)
	}
	if ruta.ruta == httpct.RutaCesesNombramiento {
		// Tras un cese CT ya confirmado sólo se lee el origen B2 y la ficha y
		// se registra la revisión finalizada de la relación.
		switch accion {
		case ct.AccionConsultarDetalleRRHH, ct.AccionLeerPlanNominalB2, personal.AccionFichaEmpleadoB2, personal.AccionHechoEmpleadoB2:
			return ruta.metodo == "POST"
		}
		return false
	}
	if ruta.ruta != httpct.RutaPlanB2 || (ruta.metodo != "GET" && ruta.metodo != "POST") {
		return false
	}
	if accion == ct.AccionRegistrarPlanNominalB2 {
		return ruta.metodo == "POST"
	}
	switch accion {
	case ct.AccionConsultarDetalleRRHH, ct.AccionLeerPlanNominalB2, ct.AccionConsultarVinculoCategoriaRPT, personal.AccionVacantesB2, personal.AccionConsultarCatalogoEmpleadoB2, "personal.plan_incorporacion_ct.seleccionar", "personal.plan_incorporacion_ct.clases_ocupacion", "personal.plan_incorporacion_ct.consultar", personal.AccionFichaEmpleadoB2, "vec.catalogos.categorias.consultar_uso", "bolsa.aceptacion_ct.anclaje.consultar", "vec.catalogos.categorias.consultar_historica", "bolsa.aceptacion_ct.persona.consultar":
		return true
	}
	return false
}
