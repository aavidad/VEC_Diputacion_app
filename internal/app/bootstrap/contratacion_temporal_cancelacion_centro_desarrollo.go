package bootstrap

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Cancelación del expediente por el centro solicitante (duda 12, CT122 y
// AD3-87) dentro del circuito de peticiones del centro. Usa la identidad
// nominal del centro y el mismo servicio de aplicación que RRHH con el canal
// «centro»: el motivo ha de admitir ese canal en el catálogo y la decisión V3
// liga el ámbito del centro del expediente, que ha de ser el del actor.
// Qué perfiles del centro cancelan lo fija el atributo «roles_centro» de la
// regla c20; sin él, el centro no cancela (la conducta de siempre).
const (
	rutaCancelacionesCentro = "/api/vec/contratacion-temporal/peticiones-centro/cancelaciones"
	rutaCancelacionCentro   = "/api/vec/contratacion-temporal/peticiones-centro/cancelacion"
	// AtributoRolesCentroCancelacion en la regla c20: perfiles del centro que
	// cancelan, separados por comas.
	atributoRolesCentroCancelacion = "roles_centro"
)

func rutaCancelacionCentroDesarrollo(ruta string) bool {
	return ruta == rutaCancelacionesCentro || ruta == rutaCancelacionCentro
}

// piezasCancelacionCTDesarrollo son las piezas que compone la cancelación de
// RRHH y que el canal del centro reutiliza: el proveedor de material de la
// audiencia de AD3-87, la fuente de reglas, los sellos y el repositorio.
type piezasCancelacionCTDesarrollo struct {
	proveedor   *proveedorMaterialAltaContratacionTemporalDesarrollo
	fuente      fuenteReglasCancelacionDesarrollo
	sellos      ports.SelladorOperacionSeguimiento
	repositorio *postgresct.RepositorioOperacionSeguimientoPostgreSQL
	reloj       relojContratacionTemporalDesarrollo
}

// cancelacionCentroDesarrollo es la composición del canal del centro. Inactiva
// no concede nada ni monta rutas.
type cancelacionCentroDesarrollo struct {
	piezas *piezasCancelacionCTDesarrollo
	roles  []string
}

// nuevaCancelacionCentroDesarrollo solo se activa con la cancelación de RRHH
// compuesta y la bandeja de incorporaciones del centro, que es la lectura que
// liga cada petición con su expediente.
func nuevaCancelacionCentroDesarrollo(cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo, incorporacion *incorporacionCentroDesarrollo) *cancelacionCentroDesarrollo {
	if !cancelacionCTSolicitada(cfg) || alta == nil || alta.cancelacion == nil {
		return &cancelacionCentroDesarrollo{}
	}
	if incorporacion == nil || !incorporacion.activo {
		slog.Warn("cancelación por el centro no compuesta: necesita la bandeja de incorporaciones del centro (VEC_CT_INCORPORACION_ACREDITADA_ENABLED)")
		return &cancelacionCentroDesarrollo{}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	regla, err := alta.cancelacion.fuente.reglas.Regla(ctx, reglas.CTCancelacionExpediente)
	if err != nil {
		slog.Error("cancelación por el centro no compuesta: regla c20 no disponible", "causa", err)
		return &cancelacionCentroDesarrollo{}
	}
	var roles []string
	for _, rol := range strings.Split(regla.Atributos[atributoRolesCentroCancelacion], ",") {
		if rol = strings.TrimSpace(rol); (rol == "solicitante_centro" || rol == "ratificador_centro") && !slices.Contains(roles, rol) {
			roles = append(roles, rol)
		}
	}
	if len(roles) == 0 {
		return &cancelacionCentroDesarrollo{}
	}
	return &cancelacionCentroDesarrollo{piezas: alta.cancelacion, roles: roles}
}

func (c *cancelacionCentroDesarrollo) activa() bool {
	return c != nil && c.piezas != nil && len(c.roles) > 0
}

// concesiones da a los perfiles de la regla la cancelación y la consulta de
// sus opciones, siempre sobre el recurso de cancelación.
func (c *cancelacionCentroDesarrollo) concesiones(rol string) []vecdomain.ConcesionRol {
	if !c.activa() || !slices.Contains(c.roles, rol) {
		return nil
	}
	return []vecdomain.ConcesionRol{
		{Accion: string(domain.AccionCancelarExpediente), ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoCancelacion,
			Finalidades: []string{ports.FinalidadCancelarExpediente}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
		{Accion: accionConsultarCancelacionCTDesarrollo, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoCancelacion,
			Finalidades: []string{finalidadPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
	}
}

func (c *cancelacionCentroDesarrollo) rutas(p *proveedorPeticionCentroDesarrollo) ([]vechttp.RutaExacta, error) {
	if !c.activa() {
		return nil, nil
	}
	if p == nil || p.alta == nil {
		return nil, errCancelacionCTDesarrolloNoDisponible
	}
	bandeja, err := postgresct.NuevoRepositorioIncorporacionCentroPostgreSQL(p.alta.postgresql.ejecucion, p)
	if err != nil {
		return nil, err
	}
	autoridad := &autoridadCancelacionCentroDesarrollo{proveedor: p, piezas: c.piezas, roles: c.roles, bandeja: bandeja}
	servicio, err := application.NuevoServicioCancelacionExpediente(application.DependenciasCancelacionExpediente{Canal: domain.CanalCancelacionCentro,
		Contextos: autoridad, Sellos: c.piezas.sellos, Repositorio: c.piezas.repositorio, Reglas: fuenteReglasCancelacionCentroDesarrollo{c.piezas.fuente},
		Autorizador: autoridad, Referencias: seguridadct.NuevoGeneradorReferenciasAltaCriptografico(), Lector: c.piezas.repositorio, Reloj: c.piezas.reloj})
	if err != nil {
		return nil, err
	}
	manejadores, err := httpinterno.NuevosManejadoresCancelacionEnRutas(rutaCancelacionesCentro, rutaCancelacionCentro, autoridad, autoridad,
		&ejecutorCancelacionCentroDesarrollo{servicio: servicio, autoridad: autoridad})
	if err != nil {
		return nil, err
	}
	return []vechttp.RutaExacta{{Ruta: rutaCancelacionesCentro, Manejador: manejadores[rutaCancelacionesCentro]},
		{Ruta: rutaCancelacionCentro, Manejador: manejadores[rutaCancelacionCentro]}}, nil
}

// fuenteReglasCancelacionCentroDesarrollo es la misma regla y el mismo
// catálogo que RRHH; solo cambia el motivo de autorización, que en este
// circuito es el de las peticiones del centro.
type fuenteReglasCancelacionCentroDesarrollo struct {
	base fuenteReglasCancelacionDesarrollo
}

func (f fuenteReglasCancelacionCentroDesarrollo) ReglaCancelacion(ctx context.Context, instante time.Time) (ports.ReglaCancelacion, ports.PoliticaOperacionSeguimiento, error) {
	regla, politica, err := f.base.ReglaCancelacion(ctx, instante)
	if err != nil {
		return regla, politica, err
	}
	politica.MotivoAutorizacion = motivoPeticionCentroDesarrollo()
	return regla, politica, nil
}

// recursoCancelacionCentroDesarrollo liga la solicitud de autorización al
// recurso exacto; sus ámbitos son los de la instantánea de esa solicitud.
type recursoCancelacionCentroDesarrollo struct {
	accion, finalidad string
	recurso           vecdomain.RecursoAutorizable
	actor             domain.ActorPeticionCentro
}

func (r *recursoCancelacionCentroDesarrollo) validaPara(principal, perfil string, d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	if r == nil || r.actor.ActorRef != principal || r.actor.PerfilRef != perfil || d.Accion != r.accion || d.Finalidad != r.finalidad ||
		d.Recurso.Referencia != r.recurso.Referencia || d.Recurso.ModuloID != ports.ModuloContratacion || d.Recurso.Tipo != ports.TipoRecursoCancelacion ||
		r.recurso.Tipo != ports.TipoRecursoCancelacion || !maps.Equal(d.Recurso.Ambitos, r.recurso.Ambitos) || !maps.Equal(d.Recurso.Atributos, r.recurso.Atributos) ||
		d.Recurso.Ambitos["centro_ref"] != r.actor.CentroRef || d.Recurso.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		d.Recurso.Ambitos["expediente_ref"] != d.Recurso.Referencia {
		return false
	}
	switch r.accion {
	case string(domain.AccionCancelarExpediente):
		return r.finalidad == ports.FinalidadCancelarExpediente && len(d.Recurso.Ambitos) == 5 &&
			domain.ClaveFase(d.Recurso.Ambitos["fase_previa"]).Valida() && d.Recurso.Ambitos["estado_previo"] == string(domain.EstadoEnCurso) &&
			d.Recurso.Atributos["canal"] == string(domain.CanalCancelacionCentro)
	case accionConsultarCancelacionCTDesarrollo:
		return r.finalidad == finalidadPeticionCentro && len(d.Recurso.Ambitos) == 3
	}
	return false
}

// ambitos son los de la instantánea ligada a esta solicitud: exactamente los
// del recurso, cuyo centro ya se ha cotejado con el del actor.
func (r *recursoCancelacionCentroDesarrollo) ambitos() []vecdomain.AmbitoPerfil {
	claves := slices.Sorted(maps.Keys(r.recurso.Ambitos))
	ambitos := make([]vecdomain.AmbitoPerfil, 0, len(claves))
	for _, c := range claves {
		ambitos = append(ambitos, vecdomain.AmbitoPerfil{Clave: c, Valores: []string{r.recurso.Ambitos[c]}})
	}
	return ambitos
}

// autoridadCancelacionCentroDesarrollo resuelve la identidad del centro,
// comprueba que el expediente es de una petición del centro y exige la
// decisión V3 con la instantánea del centro.
type autoridadCancelacionCentroDesarrollo struct {
	proveedor *proveedorPeticionCentroDesarrollo
	piezas    *piezasCancelacionCTDesarrollo
	roles     []string
	bandeja   *postgresct.RepositorioIncorporacionCentroPostgreSQL
}

func (a *autoridadCancelacionCentroDesarrollo) identidad(ctx context.Context) (*identidadPeticionCentroDesarrollo, error) {
	if a == nil || a.proveedor == nil {
		return nil, ports.ErrAutorizacionDenegada
	}
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || !rutaCancelacionCentroDesarrollo(c.ruta) {
		return nil, ports.ErrAutorizacionDenegada
	}
	id, err := a.proveedor.identidad(ctx)
	if err != nil || !slices.Contains(a.roles, id.principal.Roles[0]) {
		return nil, ports.ErrAutorizacionDenegada
	}
	return id, nil
}

func (a *autoridadCancelacionCentroDesarrollo) ResolverContextoCanalSeguimiento(ctx context.Context) (application.ContextoCanalSeguimiento, error) {
	id, err := a.identidad(ctx)
	if err != nil {
		return application.ContextoCanalSeguimiento{}, err
	}
	v, err := id.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return application.ContextoCanalSeguimiento{}, ports.ErrAutorizacionDenegada
	}
	return application.ContextoCanalSeguimiento{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo}, nil
}

// ResolverContextoAutorizacionAltaV3 devuelve el contexto nominal del actor
// del centro si la solicitud es exactamente la suya.
func (a *autoridadCancelacionCentroDesarrollo) ResolverContextoAutorizacionAltaV3(ctx context.Context, sol ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	id, err := a.identidad(ctx)
	if err != nil || sol.Validar() != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	v, err := id.soporte.contexto.Vinculo.Datos()
	if err != nil || sol.AutenticacionRef != v.AutenticacionRef || sol.SesionRef != v.SesionRef || sol.PerfilRef != v.PerfilActivoRef {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return id.soporte.contexto, nil
}

// pertenece lee la bandeja del centro (la misma de las incorporaciones) y
// comprueba que el expediente procede de una de sus peticiones.
func (a *autoridadCancelacionCentroDesarrollo) pertenece(ctx context.Context, id *identidadPeticionCentroDesarrollo, expedienteRef string) error {
	filas, err := a.bandeja.ListarIncorporacionesCentro(ctx, ports.ConsultaIncorporacionesCentro{Modo: "bandeja",
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, Actor: id.actor})
	if err != nil {
		return ports.ErrAutorizacionDenegada
	}
	for _, f := range filas {
		if f.ExpedienteRef == expedienteRef {
			return nil
		}
	}
	return ports.ErrAutorizacionDenegada
}

func (a *autoridadCancelacionCentroDesarrollo) exigir(ctx context.Context, id *identidadPeticionCentroDesarrollo, accion, finalidad string, recurso vecdomain.RecursoAutorizable) (vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	var (
		s vecdomain.SolicitudAutorizacionLigadaV3
		d vecdomain.DecisionAutorizacionLigadaV3
		c vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	)
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return s, d, c, err
	}
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: id.soporte.contexto.Vinculo, ReferenciaMotivo: motivoPeticionCentroDesarrollo(),
		Accion: accion, Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion}
	ctx = context.WithValue(ctx, claveMaterialPeticionCentroDesarrollo{}, materialAutorizacionPeticionCentroDesarrollo{
		cancelacion: &recursoCancelacionCentroDesarrollo{accion: accion, finalidad: finalidad, recurso: recurso, actor: id.actor}})
	if !solicitudAutorizacionPeticionCentroDesarrolloValida(ctx, datos) {
		return s, d, c, ports.ErrAutorizacionDenegada
	}
	s, err = vecdomain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return s, d, c, err
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	d, c, err = id.autorizador.ExigirSolicitudLigadaV3(ctx, s, id.soporte.contexto.Resultado)
	return s, d, c, err
}

func (a *autoridadCancelacionCentroDesarrollo) AutorizarOperacionSeguimiento(ctx context.Context, sol ports.SolicitudAutorizarOperacionSeguimiento) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	id, err := a.identidad(ctx)
	if err != nil || a.piezas == nil || a.piezas.proveedor == nil || sol.Audiencia != ports.AudienciaConsumoCancelacionV1 ||
		sol.Accion != domain.AccionCancelarExpediente || sol.Finalidad != ports.FinalidadCancelarExpediente ||
		sol.Motivo != motivoPeticionCentroDesarrollo() || sol.Recurso.Ambitos["centro_ref"] != id.actor.CentroRef {
		return vacio, ports.ErrAutorizacionDenegada
	}
	s, d, c, err := a.exigir(ctx, id, string(sol.Accion), sol.Finalidad, sol.Recurso)
	if err != nil {
		return vacio, err
	}
	return a.piezas.proveedor.proveerMaterialConfirmacion(ctx, s, d, c, motivoPeticionCentroDesarrollo(), id.soporte.contexto.Resultado)
}

// AutorizarLecturaSeguimiento exige que el expediente sea de una petición del
// centro y una decisión de lectura del recurso exacto; no escribe efectos.
func (a *autoridadCancelacionCentroDesarrollo) AutorizarLecturaSeguimiento(ctx context.Context, organizacionRef, expedienteRef string) error {
	id, err := a.identidad(ctx)
	if err != nil || organizacionRef != organizacionAltaContratacionTemporalDesarrollo || !domain.ReferenciaOpacaValida(expedienteRef) {
		return ports.ErrAutorizacionDenegada
	}
	if err := a.pertenece(ctx, id, expedienteRef); err != nil {
		return err
	}
	_, _, _, err = a.exigir(ctx, id, accionConsultarCancelacionCTDesarrollo, finalidadPeticionCentro, vecdomain.RecursoAutorizable{
		Referencia: expedienteRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoCancelacion,
		Ambitos:   map[string]string{"organizacion_ref": organizacionRef, "centro_ref": id.actor.CentroRef, "expediente_ref": expedienteRef},
		Atributos: map[string]string{"lectura": "cancelacion_opciones"}})
	return err
}

// ejecutorCancelacionCentroDesarrollo comprueba, antes de preparar nada, que
// el expediente procede de una petición del centro: ni sus errores de estado
// se revelan para expedientes ajenos.
type ejecutorCancelacionCentroDesarrollo struct {
	servicio  *application.ServicioCancelacionExpediente
	autoridad *autoridadCancelacionCentroDesarrollo
}

func (e *ejecutorCancelacionCentroDesarrollo) CancelarExpediente(ctx context.Context, sol application.SolicitudCancelarExpediente) (ports.ReciboOperacionSeguimiento, error) {
	id, err := e.autoridad.identidad(ctx)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, application.ErrSeguimientoDenegado
	}
	if err := e.autoridad.pertenece(ctx, id, sol.ExpedienteRef); err != nil {
		return ports.ReciboOperacionSeguimiento{}, application.ErrSeguimientoDenegado
	}
	return e.servicio.CancelarExpediente(ctx, sol)
}

func (e *ejecutorCancelacionCentroDesarrollo) Opciones(ctx context.Context) (application.OpcionesCancelacion, error) {
	return e.servicio.Opciones(ctx)
}

func (e *ejecutorCancelacionCentroDesarrollo) Estado(ctx context.Context, organizacionRef, expedienteRef string) (ports.EstadoCancelacionExpediente, error) {
	return e.servicio.Estado(ctx, organizacionRef, expedienteRef)
}
