package bootstrap

import (
	"context"
	"encoding/json"
	"maps"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	rutaOperacionesPeticionCentro = "/api/vec/contratacion-temporal/peticiones-centro/operaciones"
	rutaBandejaPeticionCentro     = "/api/vec/contratacion-temporal/peticiones-centro/bandeja"
	rutaContextoPeticionCentro    = "/api/vec/contratacion-temporal/peticiones-centro/contexto"
	finalidadPeticionCentro       = "gestionar_peticion_centro"
)

func rutaPeticionCentroDesarrollo(ruta string) bool {
	return ruta == rutaOperacionesPeticionCentro || ruta == rutaBandejaPeticionCentro || ruta == rutaContextoPeticionCentro ||
		rutaIncorporacionCentroDesarrollo(ruta) || rutaCancelacionCentroDesarrollo(ruta)
}

func principalPeticionCentroDesarrolloValido(p vecdomain.Principal) bool {
	return principalSinteticoContratacionTemporalDesarrolloValido(p) && len(p.Roles) == 1 &&
		(p.Roles[0] == "solicitante_centro" || p.Roles[0] == "ratificador_centro")
}

type identidadPeticionCentroDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizadorLigadoContratacionTemporalDesarrollo
	actor       domain.ActorPeticionCentro
	adscripcion adscripcionCentroDesarrollo
	principal   vecdomain.Principal
	// cancelacion es el perfil propio con el que esta persona cancela
	// expedientes; nulo si la regla c20 no le da la cancelación.
	cancelacion *perfilCentroDesarrollo
}

// perfilCentroDesarrollo es un perfil V3 de una persona del centro: su
// soporte (fuente de instantánea y registro), su autorizador y el actor que
// lo usa, con la referencia de ese perfil.
type perfilCentroDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizadorLigadoContratacionTemporalDesarrollo
	actor       domain.ActorPeticionCentro
}

func perfilGeneralCentroDesarrollo(id *identidadPeticionCentroDesarrollo) perfilCentroDesarrollo {
	if id == nil {
		return perfilCentroDesarrollo{}
	}
	return perfilCentroDesarrollo{soporte: id.soporte, autorizador: id.autorizador, actor: id.actor}
}

// nuevaIdentidadPeticionCentroDesarrollo compone los perfiles V3 de una
// persona del centro. El general solo consume la asignación publicada; el de
// cancelación, si la regla se la da, es propio y dinámico con guarda.
func nuevaIdentidadPeticionCentroDesarrollo(
	sello *selloConsultasContratacionTemporalDesarrollo, registro registroDecisionesAnalisisContratacionTemporalDesarrollo,
	pool *pgxpool.Pool, principal vecdomain.Principal, adscripcion adscripcionCentroDesarrollo,
	concesiones, concesionCancelar []vecdomain.ConcesionRol, reloj relojContratacionTemporalDesarrollo,
) (*identidadPeticionCentroDesarrollo, error) {
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, reloj.Ahora())
	if err != nil {
		return nil, err
	}
	v, err := contexto.Vinculo.Datos()
	if err != nil {
		return nil, err
	}
	actor := domain.ActorPeticionCentro{ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef, CentroRef: adscripcion.CentroRef, PuestoRef: adscripcion.PuestoRef}
	if actor.Validar() != nil {
		return nil, domain.ErrPeticionCentroInvalida
	}
	s := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: principal.ID, certificadoSHA256: principal.Attributes["certificate_sha256"], contexto: contexto, reloj: reloj,
		peticionesCentro: true, soloConsumePublicada: true, motivo: motivoPeticionCentroDesarrollo(), registroDecisionesAnalisis: registro,
		instantaneasPorSolicitud: make(map[string]vecdomain.InstantaneaAutorizacion), concesiones: make(map[string]struct{})}
	// Comparte el pool, no la identidad de RRHH a la que se liga cada slot.
	s.autoridadAsignaciones = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: s}
	s.instantanea, err = nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(actor.ActorRef, actor.PerfilRef, reloj.Ahora(), principal.Roles[0], "Petición de centro de desarrollo", "peticion-centro-desarrollo-"+principal.ID, concesiones,
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}, {Clave: "centro_ref", Valores: []string{actor.CentroRef}}})
	if err != nil {
		return nil, err
	}
	base, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(s, s, s, s, reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, err
	}
	id := &identidadPeticionCentroDesarrollo{soporte: s, autorizador: base, actor: actor, adscripcion: adscripcion, principal: principal}
	if len(concesionCancelar) != 0 {
		if id.cancelacion, err = nuevoPerfilCancelacionCentroDesarrollo(s, principal, actor, concesionCancelar, reloj); err != nil {
			return nil, err
		}
	}
	return id, nil
}

type proveedorPeticionCentroDesarrollo struct {
	alta      *dependenciasAltaContratacionTemporalDesarrollo
	actores   map[string]*identidadPeticionCentroDesarrollo
	catalogos vecports.ConsultaCatalogosConfigurables
	version   int
	reloj     relojContratacionTemporalDesarrollo
}

type claveMaterialPeticionCentroDesarrollo struct{}
type materialAutorizacionPeticionCentroDesarrollo struct {
	consulta  *ports.ConsultaPeticionCentro
	escritura *ports.MaterialPeticionCentro
	// incorporacion es el recurso exacto de la bandeja o la confirmación
	// de la incorporación por el centro (CT124).
	incorporacion *recursoIncorporacionCentroDesarrollo
	// cancelacion es el recurso exacto de la cancelación por el centro o de
	// la consulta de sus opciones (CT122).
	cancelacion *recursoCancelacionCentroDesarrollo
}

func nuevasRutasPeticionCentroDesarrollo(cfg config.Config, resolvedor *resolvedorIdentidadDesarrollo, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, catalogosAlta ...*catalogosAltaContratacionTemporalDesarrollo) ([]vechttp.RutaExacta, error) {
	var principales []vecdomain.Principal
	for _, p := range resolvedor.porHuella {
		if principalPeticionCentroDesarrolloValido(p) {
			principales = append(principales, clonarPrincipalDesarrollo(p))
		}
	}
	// La identidad opcional puede compartirse con una instancia de consultas.
	// Sin la fuente PostgreSQL de organización, este circuito no se compone.
	if len(principales) == 0 || !cfg.PersonalOrganizacionPostgreSQL {
		return nil, nil
	}
	if !cfg.DevelopmentEnabledByDoubleKey() || alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	incorporacion, err := nuevaIncorporacionCentroDesarrollo(cfg, reloj)
	if err != nil {
		return nil, err
	}
	cancelacion := nuevaCancelacionCentroDesarrollo(cfg, alta, incorporacion)
	catalogos, err := personalpg.NuevoRepositorioOrganizacionPostgreSQL(alta.postgresql.ejecucion, &proveedorOrganizacionDesarrollo{alta: alta, reloj: reloj}, reloj)
	if err != nil {
		return nil, err
	}
	p := &proveedorPeticionCentroDesarrollo{alta: alta, actores: make(map[string]*identidadPeticionCentroDesarrollo), catalogos: catalogos, version: cfg.PersonalOrganizacionVersion, reloj: reloj}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	for _, principal := range principales {
		adscripcion, ok := resolvedor.adscripcionCentro(principal.ID)
		if !ok {
			return nil, ports.ErrPeticionCentroNoDisponible
		}
		acciones := []string{ports.AccionConsultarPeticionCentro}
		if principal.Roles[0] == "solicitante_centro" {
			acciones = append(acciones, ports.AccionPresentarPeticionCentro)
		} else {
			acciones = append(acciones, ports.AccionRatificarPeticionCentro)
		}
		var concesiones []vecdomain.ConcesionRol
		for _, accion := range acciones {
			concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: accion, ModuloID: "contratacion_temporal", TipoRecurso: ports.TipoRecursoPeticionCentro, Finalidades: []string{finalidadPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
		}
		concesiones = append(concesiones, incorporacion.concesiones(principal.Roles[0])...)
		// El rol general conserva exactamente las concesiones de siempre para
		// que su asignación publicada siga siendo la misma. La de cancelar es
		// inerte en él: sus ámbitos (organización y centro) nunca cubren el
		// recurso de cancelación, que exige además expediente, fase y estado;
		// la cancelación se ejecuta con el perfil propio.
		concesiones = append(concesiones, cancelacion.concesiones(principal.Roles[0])...)
		id, err := nuevaIdentidadPeticionCentroDesarrollo(alta.soporte.sello, alta.soporte.registroDecisionesAnalisis, alta.postgresql.gobierno,
			principal, adscripcion, concesiones, cancelacion.concesionCancelar(principal.Roles[0]), reloj)
		if err != nil {
			return nil, err
		}
		p.actores[principal.ID] = id
	}
	aprobacion := aprobacionProvisionCentroDesarrollo{referencia: cfg.CTProvisionPerfilesCentroAprobacionRef(),
		preimagenes: cfg.CTProvisionPerfilesCentroPreimagenes()}
	for _, a := range p.actores {
		if _, err := p.catalogoParaActor(ctx, a.actor); err != nil {
			return nil, err
		}
		if a.principal.Roles[0] == "solicitante_centro" {
			rat, ok := p.actores[a.adscripcion.RatificadorSubject]
			if !ok || rat.principal.Roles[0] != "ratificador_centro" || rat.actor.CentroRef != a.actor.CentroRef || rat.actor.ActorRef == a.actor.ActorRef {
				return nil, domain.ErrRatificacionCentroDenegada
			}
		}
		if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, a.soporte); err != nil {
			return nil, err
		}
		// Arranque sin republicar: inicial solo si falta, provisión solo con
		// aprobación expresa del operador y nunca sobre lo revocado.
		estado, err := asegurarPerfilCentroConsumibleDesarrollo(ctx, alta.postgresql.gobierno, a.soporte, aprobacion)
		if err != nil {
			return nil, err
		}
		// El perfil de cancelación depende del general: nunca se crea para
		// una persona cuyo perfil general no es consumible (revocado,
		// restringido o pendiente de provisión).
		if a.cancelacion != nil && estado.perfilCentroOperativo() {
			if err := publicarContextoCancelacionCentroDesarrollo(ctx, alta.postgresql.gobierno, a.cancelacion.soporte); err != nil {
				return nil, err
			}
			if err := asegurarPerfilCancelacionCentroDesarrollo(ctx, alta.postgresql.gobierno, a.cancelacion.soporte); err != nil {
				return nil, err
			}
		}
	}
	desde, _, _ := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []vecdomain.ReferenciaEntradaCatalogo{motivoPeticionCentroDesarrollo()}, desde); err != nil {
		return nil, err
	}
	repo, err := postgresct.NuevoRepositorioPeticionesCentroPostgreSQL(alta.postgresql.ejecucion, p)
	if err != nil {
		return nil, err
	}
	rutas, err := rutasHTTPPeticionCentroDesarrollo(p, repo, catalogosAlta...)
	if err != nil {
		return nil, err
	}
	rutasIncorporacion, err := incorporacion.rutas(p)
	if err != nil {
		return nil, err
	}
	rutasCancelacion, err := cancelacion.rutas(p)
	if err != nil {
		return nil, err
	}
	return append(append(rutas, rutasIncorporacion...), rutasCancelacion...), nil
}

func (p *proveedorPeticionCentroDesarrollo) identidad(ctx context.Context) (*identidadPeticionCentroDesarrollo, error) {
	if p == nil || ctx == nil || ctx.Err() != nil {
		return nil, domain.ErrRatificacionCentroDenegada
	}
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || !rutaPeticionCentroDesarrollo(c.ruta) || c.certificadoVerificadoEn.IsZero() || !p.reloj.Ahora().Before(c.certificadoValidoHasta) {
		return nil, domain.ErrRatificacionCentroDenegada
	}
	a, ok := p.actores[c.principal.ID]
	if !ok {
		return nil, domain.ErrRatificacionCentroDenegada
	}
	if _, ok := a.soporte.capacidadValida(ctx); !ok {
		return nil, domain.ErrRatificacionCentroDenegada
	}
	if _, err := p.catalogoParaActor(ctx, a.actor); err != nil {
		return nil, err
	}
	return a, nil
}

func (p *proveedorPeticionCentroDesarrollo) ActorPeticionCentro(ctx context.Context) (domain.ActorPeticionCentro, error) {
	a, err := p.identidad(ctx)
	if err != nil {
		return domain.ActorPeticionCentro{}, err
	}
	return a.actor, nil
}

func (p *proveedorPeticionCentroDesarrollo) ConfiguracionPeticionCentro(ctx context.Context, actor domain.ActorPeticionCentro) (domain.ConfiguracionPeticionCentro, error) {
	var vacia domain.ConfiguracionPeticionCentro
	a, err := p.identidad(ctx)
	if err != nil || a.actor != actor || a.principal.Roles[0] != "solicitante_centro" {
		return vacia, domain.ErrRatificacionCentroDenegada
	}
	rat, ok := p.actores[a.adscripcion.RatificadorSubject]
	if !ok || rat.principal.Roles[0] != "ratificador_centro" {
		return vacia, domain.ErrRatificacionCentroDenegada
	}
	c, err := p.catalogoParaActor(ctx, rat.actor)
	if err != nil {
		return vacia, err
	}
	material, _ := json.Marshal([]domain.ActorPeticionCentro{a.actor, rat.actor})
	resultado := domain.ConfiguracionPeticionCentro{Referencia: referenciaAltaContratacionTemporalDesarrollo("config_", string(material)), Version: uint64(c.Revision), Solicitante: a.actor, Ratificador: rat.actor}
	return resultado, resultado.Validar()
}

// Personal determina qué unidad contiene el puesto. No se atribuyen superiores
// por denominación ni por posición en la RPT. La configuración privada nombra
// explícitamente al ratificador; cambiarla exige recargar el servidor.
func (p *proveedorPeticionCentroDesarrollo) catalogoParaActor(ctx context.Context, actor domain.ActorPeticionCentro) (vecdomain.CatalogoConfigurable, error) {
	c, err := p.catalogos.ObtenerCatalogo(ctx, personalports.IDCatalogoOrganizacion, p.version)
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, err
	}
	if !actorPeticionCentroPerteneceCatalogo(c, actor) {
		return vecdomain.CatalogoConfigurable{}, domain.ErrRatificacionCentroDenegada
	}
	return c, nil
}

func actorPeticionCentroPerteneceCatalogo(c vecdomain.CatalogoConfigurable, actor domain.ActorPeticionCentro) bool {
	centro, puesto := false, false
	padres := make(map[string]string, len(c.Entradas))
	for _, u := range c.Entradas {
		padres[u.Clave] = u.Atributos["adscripcion_clave"]
		if u.Clave == actor.CentroRef && u.Atributos["tipo"] == "centro" {
			centro = true
		}
		if u.Clave == actor.PuestoRef && u.Atributos["tipo"] == "puesto_responsabilidad" {
			puesto = true
		}
	}
	if centro && puesto {
		actual := actor.PuestoRef
		for n := 0; n < len(c.Entradas); n++ {
			actual = padres[actual]
			if actual == actor.CentroRef {
				return true
			}
			if actual == "" {
				break
			}
		}
	}
	return false
}

func motivoPeticionCentroDesarrollo() vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_peticion_centro", CatalogoVersion: 1, CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("peticion-centro-desarrollo-v1"), EntradaClave: referenciaAltaContratacionTemporalDesarrollo("motivo_", "peticion-centro")}
}

func (p *proveedorPeticionCentroDesarrollo) AutorizarLecturaPeticionCentro(ctx context.Context, c ports.ConsultaPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	r, err := postgresct.RecursoConsultaPeticionCentro(c)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return p.autorizar(ctx, c.Actor, ports.AccionConsultarPeticionCentro, r, materialAutorizacionPeticionCentroDesarrollo{consulta: &c})
}
func (p *proveedorPeticionCentroDesarrollo) AutorizarEscrituraPeticionCentro(ctx context.Context, m ports.MaterialPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	r, err := postgresct.RecursoEscrituraPeticionCentro(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return p.autorizar(ctx, m.Actor, postgresct.AccionEscrituraPeticionCentro(m), r, materialAutorizacionPeticionCentroDesarrollo{escritura: &m})
}

func (p *proveedorPeticionCentroDesarrollo) autorizar(ctx context.Context, actor domain.ActorPeticionCentro, accion string, r vecdomain.RecursoAutorizable, m materialAutorizacionPeticionCentroDesarrollo) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	a, err := p.identidad(ctx)
	if err != nil || a.actor != actor {
		return vacio, domain.ErrRatificacionCentroDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, err
	}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: a.soporte.contexto.Vinculo, ReferenciaMotivo: motivoPeticionCentroDesarrollo(), Accion: accion, Recurso: r, Finalidad: finalidadPeticionCentro, Correlacion: correlacion}
	ctx = context.WithValue(ctx, claveMaterialPeticionCentroDesarrollo{}, m)
	if !solicitudAutorizacionPeticionCentroDesarrolloValida(ctx, d) {
		return vacio, domain.ErrRatificacionCentroDenegada
	}
	s, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		return vacio, err
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	decision, confirmacion, err := a.autorizador.ExigirSolicitudLigadaV3(ctx, s, a.soporte.contexto.Resultado)
	if err != nil {
		return vacio, err
	}
	return p.alta.postgresql.proveedorMaterial.proveerMaterialConfirmacion(ctx, s, decision, confirmacion, motivoPeticionCentroDesarrollo(), a.soporte.contexto.Resultado)
}

func solicitudAutorizacionPeticionCentroDesarrolloValida(ctx context.Context, d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil || d.ReferenciaMotivo != motivoPeticionCentroDesarrollo() {
		return false
	}
	m, ok := ctx.Value(claveMaterialPeticionCentroDesarrollo{}).(materialAutorizacionPeticionCentroDesarrollo)
	if !ok {
		return false
	}
	v, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		return false
	}
	// La cancelación tiene su propia finalidad (la que exige AD3-87).
	if m.cancelacion != nil {
		return m.consulta == nil && m.escritura == nil && m.incorporacion == nil && m.cancelacion.validaPara(v.PrincipalID, v.PerfilActivoRef, d)
	}
	if d.Finalidad != finalidadPeticionCentro {
		return false
	}
	var r vecdomain.RecursoAutorizable
	var actor domain.ActorPeticionCentro
	if m.incorporacion != nil {
		return m.consulta == nil && m.escritura == nil && m.cancelacion == nil && m.incorporacion.validaPara(v.PrincipalID, v.PerfilActivoRef, d)
	}
	if m.consulta != nil && m.escritura == nil && d.Accion == ports.AccionConsultarPeticionCentro {
		actor = m.consulta.Actor
		r, err = postgresct.RecursoConsultaPeticionCentro(*m.consulta)
	} else if m.escritura != nil && m.consulta == nil && d.Accion == postgresct.AccionEscrituraPeticionCentro(*m.escritura) {
		actor = m.escritura.Actor
		r, err = postgresct.RecursoEscrituraPeticionCentro(*m.escritura)
	} else {
		return false
	}
	return err == nil && actor.ActorRef == v.PrincipalID && actor.PerfilRef == v.PerfilActivoRef && r.Referencia == d.Recurso.Referencia && r.ModuloID == d.Recurso.ModuloID && r.Tipo == d.Recurso.Tipo && maps.Equal(r.Ambitos, d.Recurso.Ambitos) && maps.Equal(r.Atributos, d.Recurso.Atributos)
}
