package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/telemetria"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	appvec "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const claveFronteraSeleccionOfertaCTDesarrollo = "ct-oferta-bolsa-seleccionar-privada"
const claveFronteraConsultaEstadoOfertaCTDesarrollo = "ct-oferta-bolsa-estado-privada"

// La fuente solo consulta la asignación publicada. El fichero de activación
// identifica la versión aprobada, pero no contiene ni concede un rol.
type fuenteRolSeleccionOfertaCTDesarrollo struct {
	fuente      vp.FuenteAutorizacion
	principalID string
	personaRef  string
	perfilRef   string
	rol         referenciaRolOfertaCTDesarrollo
	politica    politicaPerfilOfertaCTDesarrollo
	reloj       relojContratacionTemporalDesarrollo
}

type politicaPerfilOfertaCTDesarrollo struct {
	accion, modulo, tipo, finalidad, audiencia string
	ambitos                                    map[string]string // vacío exige clave, pero deja su valor a la asignación publicada.
}

func politicaVinculoOfertaCTDesarrollo() politicaPerfilOfertaCTDesarrollo {
	return politicaPerfilOfertaCTDesarrollo{ports.AccionVincularOfertaBolsaCT,
		ports.ModuloContratacion, ports.TipoRecursoVinculoOfertaBolsaCT,
		ports.FinalidadVincularOfertaBolsaCT, ports.AudienciaVincularOfertaBolsaCT,
		map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"unidad_ref": ""}}
}

func politicaCierreOfertaBolsaCTDesarrollo(unidad, ambito string) politicaPerfilOfertaCTDesarrollo {
	return politicaPerfilOfertaCTDesarrollo{"bolsa.oferta.intencion.cerrar", "bolsa", "bolsa_constituida",
		"gestion_llamamientos_bolsa", descriptorMaterialCierreIntencionOfertaCTDesarrollo().Audiencia,
		map[string]string{"unidad_ref": unidad, "ambito_ref": ambito}}
}

func politicaDescarteOfertaCTDesarrollo() politicaPerfilOfertaCTDesarrollo {
	return politicaPerfilOfertaCTDesarrollo{"contratacion_temporal.bolsa.oferta.descartar",
		ports.ModuloContratacion, "seleccion_oferta_expediente", ports.FinalidadVincularOfertaBolsaCT,
		descriptorMaterialDescarteSeleccionOfertaCTDesarrollo().Audiencia,
		map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"unidad_ref": ""}}
}

var _ vp.FuenteAutorizacion = (*fuenteRolSeleccionOfertaCTDesarrollo)(nil)

func (f *fuenteRolSeleccionOfertaCTDesarrollo) ObtenerInstantaneaAutorizacion(ctx context.Context, principal, perfil string) (core.InstantaneaAutorizacion, error) {
	var vacia core.InstantaneaAutorizacion
	if f == nil || ctx == nil || ctx.Err() != nil || dependenciaEsNulaContratacionTemporalDesarrollo(f.fuente) ||
		principal != f.principalID || perfil != f.perfilRef {
		return vacia, ports.ErrAutorizacionDenegada
	}
	i, err := f.fuente.ObtenerInstantaneaAutorizacion(ctx, principal, perfil)
	if err != nil {
		if errors.Is(err, vp.ErrFuenteAutorizacionNoDisponible) || ctx.Err() != nil {
			return vacia, vp.ErrFuenteAutorizacionNoDisponible
		}
		return vacia, ports.ErrAutorizacionDenegada
	}
	if i.Validar() != nil ||
		i.AsignacionPerfil.PrincipalID != principal || i.AsignacionPerfil.PerfilActivoRef != perfil ||
		i.AsignacionPerfil.Estado != core.EstadoAsignacionPerfilActiva ||
		!i.AsignacionPerfil.VigenteEn(f.reloj.Ahora()) ||
		i.VersionRol.Estado != core.EstadoVersionRolPublicada ||
		i.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada ||
		i.VersionRol.RolID != f.rol.RolID || i.VersionRol.Version != f.rol.Version ||
		len(i.VersionRol.Concesiones) != 1 ||
		!concesionExactaPerfilOfertaCTDesarrollo(i.VersionRol.Concesiones[0], f.politica) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	huella, err := i.VersionRol.HuellaSHA256()
	if err != nil || huella != f.rol.HuellaSHA256 {
		return vacia, ports.ErrAutorizacionDenegada
	}
	claves := make(map[string]bool, len(f.politica.ambitos))
	for _, ambito := range i.AsignacionPerfil.Ambitos {
		esperado, existe := f.politica.ambitos[ambito.Clave]
		if !existe || (esperado != "" && !slices.Contains(ambito.Valores, esperado)) {
			return vacia, ports.ErrAutorizacionDenegada
		}
		claves[ambito.Clave] = true
	}
	if len(i.AsignacionPerfil.Ambitos) != len(f.politica.ambitos) || len(claves) != len(f.politica.ambitos) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	return i, nil
}

func concesionExactaSeleccionOfertaCTDesarrollo(c core.ConcesionRol) bool {
	return concesionExactaPerfilOfertaCTDesarrollo(c, politicaVinculoOfertaCTDesarrollo())
}

func concesionExactaPerfilOfertaCTDesarrollo(c core.ConcesionRol, p politicaPerfilOfertaCTDesarrollo) bool {
	return c.Accion == p.accion && c.ModuloID == p.modulo && c.TipoRecurso == p.tipo &&
		len(c.Finalidades) == 1 && c.Finalidades[0] == p.finalidad &&
		c.GarantiaMinima == core.AuthAssuranceHigh &&
		len(c.CamposPermitidos) == 0 && len(c.Obligaciones) == 0
}

// El sujeto del certificado que conserva el canal CT no es la referencia
// canónica de Persona del contexto durable. Ambos quedan ligados por la
// frontera mTLS; los perfiles publicados se cotejan contra esa Persona y
// Cuenta, sin sustituir la comprobación del sujeto/certificado original.
func identidadContextoPublicadoOfertaCTDesarrollo(esperado, original core.ResultadoContextoActorRegistradoV2,
	perfil string, ahora time.Time,
) bool {
	return esperado.Validar() == nil && original.Validar() == nil &&
		esperado.AutoridadEfectiva == core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1 &&
		esperado.Contexto.Principal.ID == original.Contexto.Principal.ID &&
		esperado.Contexto.Principal.ID == esperado.Contexto.PersonaRef &&
		esperado.Contexto.PersonaRef == original.Contexto.PersonaRef &&
		esperado.Contexto.Instantanea.CuentaRef == original.Contexto.Instantanea.CuentaRef &&
		esperado.Contexto.PerfilActivoRef == perfil &&
		esperado.Contexto.Principal.AuthMethod == core.AuthMethodCertificate &&
		esperado.Contexto.Principal.AuthAssurance == core.AuthAssuranceHigh &&
		esperado.Contexto.Instantanea.VigenteEn(ahora)
}

type autoridadPerfilOfertaCTDesarrollo struct {
	privado  catalogoFronterasComunDesarrollo
	sesion   *seguridadComunDesarrollo
	fuente   *fuenteRolSeleccionOfertaCTDesarrollo
	pdp      vp.AutorizadorSolicitudLigadaV3
	material *proveedorMaterialAltaContratacionTemporalDesarrollo
	motivo   core.ReferenciaEntradaCatalogo
}

type autoridadSeleccionOfertaCTDesarrollo struct {
	alta        *dependenciasAltaContratacionTemporalDesarrollo
	publico     catalogoFronterasComunDesarrollo
	vinculo     *autoridadPerfilOfertaCTDesarrollo
	cierreBolsa *autoridadPerfilOfertaCTDesarrollo
	descarteCT  *autoridadPerfilOfertaCTDesarrollo
	reloj       relojContratacionTemporalDesarrollo
}

var _ ports.AutorizadorSeleccionOfertaCT = (*autoridadSeleccionOfertaCTDesarrollo)(nil)
var _ ports.AutorizadorDescarteSeleccionOfertaCT = (*autoridadSeleccionOfertaCTDesarrollo)(nil)

func (a *autoridadSeleccionOfertaCTDesarrollo) MotivoOferta() core.ReferenciaEntradaCatalogo {
	if a == nil || a.vinculo == nil {
		return core.ReferenciaEntradaCatalogo{}
	}
	return a.vinculo.motivo
}

func (a *autoridadSeleccionOfertaCTDesarrollo) MotivoCierreBolsa() core.ReferenciaEntradaCatalogo {
	if a == nil || a.cierreBolsa == nil {
		return core.ReferenciaEntradaCatalogo{}
	}
	return a.cierreBolsa.motivo
}

// ContextoOferta registra un perfil nominal nuevo para la misma petición
// certificada. El hijo privado no se instala en el contexto exterior que
// después usa Bolsa para comprobar su propio perfil y audiencia.
func (a *autoridadSeleccionOfertaCTDesarrollo) ContextoOferta(ctx context.Context) (context.Context, ports.ContextoAutorizacionAltaV3, error) {
	if a == nil {
		return nil, ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return a.contextoPerfilOferta(ctx, a.vinculo, http.MethodPost, cthttp.RutaSeleccionLlamamiento)
}

func (a *autoridadSeleccionOfertaCTDesarrollo) ContextoCierreBolsa(ctx context.Context) (context.Context, ports.ContextoAutorizacionAltaV3, error) {
	if a == nil {
		return nil, ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return a.contextoPerfilOferta(ctx, a.cierreBolsa, http.MethodPost, cthttp.RutaSeleccionLlamamiento)
}

func (a *autoridadSeleccionOfertaCTDesarrollo) ContextoConsultaEstadoOferta(ctx context.Context) (context.Context, ports.ContextoAutorizacionAltaV3, error) {
	if a == nil {
		return nil, ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return a.contextoPerfilOferta(ctx, a.vinculo, http.MethodGet, cthttp.RutaConsultaEstadoSeleccionOferta)
}

func (a *autoridadSeleccionOfertaCTDesarrollo) contextoPerfilOferta(ctx context.Context, p *autoridadPerfilOfertaCTDesarrollo, metodo, ruta string) (context.Context, ports.ContextoAutorizacionAltaV3, error) {
	var vacio ports.ContextoAutorizacionAltaV3
	if a == nil || a.alta == nil || a.alta.soporte == nil || p == nil || p.sesion == nil ||
		p.fuente == nil || ctx == nil || ctx.Err() != nil ||
		(metodo != http.MethodPost || ruta != cthttp.RutaSeleccionLlamamiento) &&
			(p != a.vinculo || metodo != http.MethodGet || ruta != cthttp.RutaConsultaEstadoSeleccionOferta) {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := a.alta.soporte.capacidadValida(ctx)
	frontera, real := fronteraSeguridadComunDesdeContexto(ctx)
	if !valida || !real || !a.publico.mismaInstancia(frontera.catalogo) ||
		capacidad.metodo != metodo || capacidad.ruta != ruta ||
		frontera.metodo != capacidad.metodo || frontera.ruta != capacidad.ruta ||
		frontera.superficie != superficieInternaSeguridadComunDesarrollo ||
		capacidad.principal.ID != a.alta.soporte.principalID ||
		capacidad.principal.Attributes["certificate_sha256"] != a.alta.soporte.certificadoSHA256 {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	original, err := a.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || original.Resultado.Validar() != nil || original.Vinculo.ValidarPara(original.Resultado) != nil ||
		!original.Vinculo.VigenteEn(a.reloj.Ahora(), original.Resultado) ||
		!frontera.descriptor.admitePerfil(original.Resultado.Contexto.PerfilActivoRef) {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	if _, err := p.fuente.ObtenerInstantaneaAutorizacion(ctx, p.fuente.principalID, p.fuente.perfilRef); err != nil {
		if errors.Is(err, vp.ErrFuenteAutorizacionNoDisponible) {
			return nil, vacio, ports.ErrIntegracionBolsaNoDisponible
		}
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	descriptor, existe := p.privado.resolver(capacidad.metodo, capacidad.ruta)
	if !existe || !descriptor.admitePerfil(p.fuente.perfilRef) {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	capacidad.consultaRRHH, capacidad.contextoOperacion, capacidad.original = nil, nil, nil
	hijo := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	hijo = context.WithValue(hijo, claveContextosNominalesIncorporacion{}, (*capturaContextosNominalesIncorporacion)(nil))
	hijo = context.WithValue(hijo, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: capacidad.metodo, ruta: capacidad.ruta,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: p.privado, descriptor: descriptor})
	hijo = context.WithValue(hijo, claveCacheSeguridadComunDesarrollo{}, &cacheSeguridadComunDesarrollo{})
	actor, err := p.sesion.ResolverContexto(hijo)
	if err != nil || actor.Resultado.Validar() != nil || actor.Vinculo.ValidarPara(actor.Resultado) != nil ||
		!actor.Vinculo.VigenteEn(a.reloj.Ahora(), actor.Resultado) ||
		actor.Resultado.Contexto.PerfilActivoRef != p.fuente.perfilRef ||
		actor.Resultado.Contexto.Principal.ID != original.Resultado.Contexto.Principal.ID ||
		actor.Resultado.Contexto.PersonaRef != p.fuente.personaRef ||
		actor.Resultado.Contexto.PersonaRef != original.Resultado.Contexto.PersonaRef ||
		actor.Resultado.Contexto.Instantanea.CuentaRef != original.Resultado.Contexto.Instantanea.CuentaRef {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	actorDatos, errActor := actor.Vinculo.Datos()
	ctDatos, errCT := original.Vinculo.Datos()
	if errActor != nil || errCT != nil || actorDatos.PrincipalID != ctDatos.PrincipalID ||
		actorDatos.CuentaRef != ctDatos.CuentaRef || actorDatos.PerfilActivoRef != p.fuente.perfilRef {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	return hijo, ports.ContextoAutorizacionAltaV3{Vinculo: actor.Vinculo, Resultado: actor.Resultado}, nil
}

func (a *autoridadSeleccionOfertaCTDesarrollo) AutorizarSeleccionOfertaCT(ctx context.Context, s ports.SolicitudAutorizacionSeleccionOfertaCT) (ports.AutorizacionSeleccionOfertaCT, error) {
	var vacia ports.AutorizacionSeleccionOfertaCT
	if a == nil || a.vinculo == nil || a.vinculo.pdp == nil || a.vinculo.material == nil || s.Intencion.Validar() != nil ||
		s.Intencion.MotivoAutorizacion != a.vinculo.motivo {
		return vacia, ports.ErrAutorizacionDenegada
	}
	canon, err := ports.CanonEtapaSeleccionOfertaCT(s)
	if err != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	hijo, actual, err := a.ContextoOferta(ctx)
	if err != nil || s.Intencion.Vinculo.ValidarPara(s.Intencion.ResultadoContexto) != nil ||
		!s.Intencion.Vinculo.VigenteEn(a.reloj.Ahora(), s.Intencion.ResultadoContexto) ||
		actual.Resultado.Contexto.Principal.ID != s.Intencion.ResultadoContexto.Contexto.Principal.ID ||
		actual.Resultado.Contexto.PersonaRef != s.Intencion.ActorRef ||
		actual.Resultado.Contexto.PerfilActivoRef != s.Intencion.PerfilRef ||
		actual.Resultado.Contexto.PersonaRef != s.Intencion.ResultadoContexto.Contexto.PersonaRef ||
		actual.Resultado.Contexto.Instantanea.CuentaRef != s.Intencion.ResultadoContexto.Contexto.Instantanea.CuentaRef {
		return vacia, ports.ErrAutorizacionDenegada
	}
	huella := sha256.Sum256([]byte(canon))
	recurso := core.RecursoAutorizable{Referencia: s.Intencion.ExpedienteRef,
		ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoVinculoOfertaBolsaCT,
		Ambitos: map[string]string{"organizacion_ref": s.Intencion.OrganizacionRef,
			"unidad_ref": s.Intencion.UnidadGestoraRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(huella[:])}}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: s.Intencion.Vinculo, ReferenciaMotivo: a.vinculo.motivo,
		Accion: ports.AccionVincularOfertaBolsaCT, Recurso: recurso,
		Finalidad: ports.FinalidadVincularOfertaBolsaCT, Correlacion: s.Intencion.Correlacion})
	if err != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	decision, confirmacion, err := a.vinculo.pdp.ExigirSolicitudLigadaV3(hijo, solicitud, s.Intencion.ResultadoContexto)
	if err != nil {
		return vacia, errorAutorizacionOfertaCTDesarrollo(hijo, err)
	}
	material, err := a.vinculo.material.proveerMaterialConfirmacion(hijo, solicitud, decision, confirmacion, a.vinculo.motivo, s.Intencion.ResultadoContexto)
	if err != nil || material.ValidarEstructura() != nil {
		return vacia, ports.ErrIntegracionBolsaNoDisponible
	}
	return ports.AutorizacionSeleccionOfertaCT{ActorRef: s.Intencion.ActorRef, PerfilRef: s.Intencion.PerfilRef,
		MaterialCanonico: canon, Material: material}, nil
}

var _ ports.AutorizadorConsultaEstadoSeleccionOfertaCT = (*autoridadSeleccionOfertaCTDesarrollo)(nil)

// La lectura por expediente obtiene una V3 nueva del perfil vinculador. El
// material no contiene la clave de operación, que sólo recupera CT en SQL.
func (a *autoridadSeleccionOfertaCTDesarrollo) AutorizarConsultaEstadoSeleccionOfertaCT(ctx context.Context,
	q ports.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT,
) (ports.AutorizacionSeleccionOfertaCT, error) {
	var vacia ports.AutorizacionSeleccionOfertaCT
	if a == nil || a.vinculo == nil || a.vinculo.pdp == nil || a.vinculo.material == nil ||
		q.Validar() != nil || q.MotivoAutorizacion != a.vinculo.motivo {
		return vacia, ports.ErrAutorizacionDenegada
	}
	canon, err := ports.CanonConsultaEstadoSeleccionOfertaCT(q)
	if err != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	hijo, actual, err := a.ContextoConsultaEstadoOferta(ctx)
	if err != nil || !q.Vinculo.VigenteEn(a.reloj.Ahora(), q.ResultadoContexto) ||
		actual.Resultado.Contexto.Principal.ID != q.ResultadoContexto.Contexto.Principal.ID ||
		actual.Resultado.Contexto.PersonaRef != q.ActorRef ||
		actual.Resultado.Contexto.PerfilActivoRef != q.PerfilRef ||
		actual.Resultado.Contexto.Instantanea.CuentaRef != q.ResultadoContexto.Contexto.Instantanea.CuentaRef {
		return vacia, ports.ErrAutorizacionDenegada
	}
	huella := sha256.Sum256([]byte(canon))
	recurso := core.RecursoAutorizable{Referencia: q.ExpedienteRef,
		ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoVinculoOfertaBolsaCT,
		Ambitos: map[string]string{"organizacion_ref": q.OrganizacionRef,
			"unidad_ref": q.UnidadGestoraRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(huella[:])}}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: actual.Vinculo, ReferenciaMotivo: a.vinculo.motivo,
		Accion: ports.AccionVincularOfertaBolsaCT, Recurso: recurso,
		Finalidad: ports.FinalidadVincularOfertaBolsaCT, Correlacion: q.Correlacion})
	if err != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	decision, confirmacion, err := a.vinculo.pdp.ExigirSolicitudLigadaV3(hijo, solicitud, actual.Resultado)
	if err != nil {
		return vacia, errorAutorizacionOfertaCTDesarrollo(hijo, err)
	}
	material, err := a.vinculo.material.proveerMaterialConfirmacion(hijo, solicitud,
		decision, confirmacion, a.vinculo.motivo, actual.Resultado)
	if err != nil || material.ValidarEstructura() != nil {
		return vacia, ports.ErrIntegracionBolsaNoDisponible
	}
	return ports.AutorizacionSeleccionOfertaCT{ActorRef: q.ActorRef, PerfilRef: q.PerfilRef,
		MaterialCanonico: canon, Material: material}, nil
}

func identidadPerfilRecuperacionOfertaCTDesarrollo(actual ports.ContextoAutorizacionAltaV3,
	i ports.IntencionSeleccionOfertaCT, perfil string) bool {
	return actual.Resultado.Validar() == nil && actual.Vinculo.ValidarPara(actual.Resultado) == nil &&
		actual.Resultado.Contexto.Principal.ID == i.ResultadoContexto.Contexto.Principal.ID &&
		actual.Resultado.Contexto.PersonaRef == i.ActorRef &&
		actual.Resultado.Contexto.Instantanea.CuentaRef == i.ResultadoContexto.Contexto.Instantanea.CuentaRef &&
		actual.Resultado.Contexto.PerfilActivoRef == perfil && perfil != i.PerfilRef
}

// MaterialCierreBolsa autoriza solo el cierre de la intención congelada. La
// sesión nueva del perfil Bolsa pertenece a la misma persona/cuenta/certificado;
// el permiso de vínculo CT no sirve para este efecto B121.
func (a *autoridadSeleccionOfertaCTDesarrollo) MaterialCierreBolsa(ctx context.Context,
	i ports.IntencionSeleccionOfertaCT, c ports.ComandoOfertaBolsaCT,
	canonB121, unidadBolsaRef, ambitoBolsaRef string,
) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if a == nil || a.cierreBolsa == nil || a.cierreBolsa.pdp == nil || a.cierreBolsa.material == nil ||
		i.Validar() != nil || c.ValidarPara(i) != nil ||
		!ctdomain.ReferenciaOpacaValida(unidadBolsaRef) || !ctdomain.ReferenciaOpacaValida(ambitoBolsaRef) ||
		unidadBolsaRef != a.cierreBolsa.fuente.politica.ambitos["unidad_ref"] ||
		ambitoBolsaRef != a.cierreBolsa.fuente.politica.ambitos["ambito_ref"] {
		return vacio, ports.ErrAutorizacionDenegada
	}
	comando, err := json.Marshal(c)
	if err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	esperado, err := bolsaports.CanonCierreIntencionOfertaCT(i.ActorRef, comando)
	if err != nil || canonB121 != esperado {
		return vacio, ports.ErrAutorizacionDenegada
	}
	hijo, actual, err := a.ContextoCierreBolsa(ctx)
	if err != nil || !identidadPerfilRecuperacionOfertaCTDesarrollo(actual, i, a.cierreBolsa.fuente.perfilRef) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	h := sha256.Sum256([]byte(canonB121))
	recurso := core.RecursoAutorizable{Referencia: c.BolsaRef,
		ModuloID: "bolsa", Tipo: "bolsa_constituida",
		Ambitos:   map[string]string{"unidad_ref": unidadBolsaRef, "ambito_ref": ambitoBolsaRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: actual.Vinculo,
		ReferenciaMotivo:          a.cierreBolsa.motivo,
		Accion:                    a.cierreBolsa.fuente.politica.accion, Recurso: recurso,
		Finalidad: a.cierreBolsa.fuente.politica.finalidad, Correlacion: i.Correlacion})
	if err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	decision, confirmacion, err := a.cierreBolsa.pdp.ExigirSolicitudLigadaV3(hijo, solicitud, actual.Resultado)
	if err != nil {
		return vacio, errorAutorizacionOfertaCTDesarrollo(hijo, err)
	}
	material, err := a.cierreBolsa.material.proveerMaterialConfirmacion(hijo, solicitud,
		decision, confirmacion, a.cierreBolsa.motivo, actual.Resultado)
	if err != nil || material.ValidarEstructura() != nil {
		return vacio, ports.ErrIntegracionBolsaNoDisponible
	}
	return material, nil
}

func (a *autoridadSeleccionOfertaCTDesarrollo) AutorizarDescarteOfertaCT(ctx context.Context,
	s ports.SolicitudAutorizacionDescarteOfertaCT,
) (ports.AutorizacionDescarteSeleccionOfertaCT, error) {
	var vacia ports.AutorizacionDescarteSeleccionOfertaCT
	if a == nil || a.descarteCT == nil || a.descarteCT.pdp == nil || a.descarteCT.material == nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	canon, err := ports.CanonDescarteSeleccionOfertaCT(s)
	if err != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	hijo, actual, err := a.contextoPerfilOferta(ctx, a.descarteCT, http.MethodPost, cthttp.RutaSeleccionLlamamiento)
	if err != nil || !identidadPerfilRecuperacionOfertaCTDesarrollo(actual, s.Intencion, a.descarteCT.fuente.perfilRef) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	h := sha256.Sum256([]byte(canon))
	recurso := core.RecursoAutorizable{Referencia: s.Intencion.ExpedienteRef,
		ModuloID: ports.ModuloContratacion, Tipo: "seleccion_oferta_expediente",
		Ambitos: map[string]string{"organizacion_ref": s.Intencion.OrganizacionRef,
			"unidad_ref": s.Intencion.UnidadGestoraRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: actual.Vinculo,
		ReferenciaMotivo:          a.descarteCT.motivo,
		Accion:                    a.descarteCT.fuente.politica.accion, Recurso: recurso,
		Finalidad: a.descarteCT.fuente.politica.finalidad, Correlacion: s.Intencion.Correlacion})
	if err != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	decision, confirmacion, err := a.descarteCT.pdp.ExigirSolicitudLigadaV3(hijo, solicitud, actual.Resultado)
	if err != nil {
		return vacia, errorAutorizacionOfertaCTDesarrollo(hijo, err)
	}
	material, err := a.descarteCT.material.proveerMaterialConfirmacion(hijo, solicitud,
		decision, confirmacion, a.descarteCT.motivo, actual.Resultado)
	if err != nil || material.ValidarEstructura() != nil {
		return vacia, ports.ErrIntegracionBolsaNoDisponible
	}
	return ports.AutorizacionDescarteSeleccionOfertaCT{ActorRef: s.Intencion.ActorRef,
		PerfilRef: a.descarteCT.fuente.perfilRef, MaterialCanonico: canon, Material: material}, nil
}

func errorAutorizacionOfertaCTDesarrollo(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, vp.ErrFuenteAutorizacionNoDisponible) ||
		errors.Is(err, vp.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
		errors.Is(err, vp.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) {
		return ports.ErrIntegracionBolsaNoDisponible
	}
	return ports.ErrAutorizacionDenegada
}

func descriptoresPerfilOfertaCTDesarrollo(c perfilActivacionOfertaCT,
	politica politicaPerfilOfertaCTDesarrollo, consultaEstado bool,
) ([]descriptorFronteraComunDesarrollo, error) {
	descriptores := []descriptorFronteraComunDesarrollo{{
		Clave:      claveFronteraSeleccionOfertaCTDesarrollo,
		Superficie: superficieInternaSeguridadComunDesarrollo,
		Metodo:     http.MethodPost, Ruta: cthttp.RutaSeleccionLlamamiento,
		PerfilesActivosRef: []string{c.PerfilRef},
		ClavePolitica:      clavePoliticaContratacionTemporalDesarrollo,
		ClaveCapacidad:     politica.accion,
	}}
	if consultaEstado {
		if politica.accion != ports.AccionVincularOfertaBolsaCT {
			return nil, errActivacionOfertaCTDesarrollo
		}
		descriptores = append(descriptores, descriptorFronteraComunDesarrollo{
			Clave:      claveFronteraConsultaEstadoOfertaCTDesarrollo,
			Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo:     http.MethodGet, Ruta: cthttp.RutaConsultaEstadoSeleccionOferta,
			PerfilesActivosRef: []string{c.PerfilRef},
			ClavePolitica:      clavePoliticaContratacionTemporalDesarrollo,
			ClaveCapacidad:     politica.accion,
		})
	}
	return descriptores, nil
}

func componerPerfilAutoridadOfertaCTDesarrollo(ctx context.Context,
	c perfilActivacionOfertaCT, politica politicaPerfilOfertaCTDesarrollo, consultaEstado bool,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	base *proveedorSesionConsultaRRHHDesarrollo,
	lector vp.FuenteAutorizacion, registro *pgvec.AlmacenAutorizacion,
	poolMotivos *pgxpool.Pool, reloj relojContratacionTemporalDesarrollo,
) (*autoridadPerfilOfertaCTDesarrollo, error) {
	if ctx == nil || alta == nil || alta.soporte == nil || base == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(lector) || registro == nil || poolMotivos == nil ||
		politica.accion == "" || politica.audiencia == "" {
		return nil, errActivacionOfertaCTDesarrollo
	}
	descriptores, err := descriptoresPerfilOfertaCTDesarrollo(c, politica, consultaEstado)
	if err != nil {
		return nil, err
	}
	privado, err := nuevoCatalogoFronterasComunDesarrollo(descriptores)
	if err != nil {
		return nil, errActivacionOfertaCTDesarrollo
	}
	cuenta := alta.soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef
	esperado, err := base.resolutor.ResolverContextoActorRegistradoV2(ctx, core.SolicitudContextoActor{
		Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta,
			Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh},
		PerfilActivoRef: c.PerfilRef})
	if err != nil || !identidadContextoPublicadoOfertaCTDesarrollo(
		esperado, alta.soporte.contexto.Resultado, c.PerfilRef, reloj.Ahora()) {
		return nil, errActivacionOfertaCTDesarrollo
	}
	proveedorPrivado, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(
		base.soporte, base.registro, base.revalidador, base.reloj, base.resolutor, privado)
	if err != nil {
		return nil, errActivacionOfertaCTDesarrollo
	}
	privada, err := nuevaSesionReincorporacionTitularDesarrollo(proveedorPrivado, esperado)
	if err != nil {
		return nil, errActivacionOfertaCTDesarrollo
	}
	sesion, err := nuevaSeguridadComunDesarrollo(privada, reloj)
	if err != nil {
		return nil, errActivacionOfertaCTDesarrollo
	}
	f := &fuenteRolSeleccionOfertaCTDesarrollo{fuente: lector,
		principalID: esperado.Contexto.Principal.ID, personaRef: esperado.Contexto.PersonaRef,
		perfilRef: c.PerfilRef, rol: c.Rol, politica: politica, reloj: reloj}
	if _, err := f.ObtenerInstantaneaAutorizacion(ctx, f.principalID, f.perfilRef); err != nil {
		return nil, errActivacionOfertaCTDesarrollo
	}
	motivos, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(poolMotivos, c.Motivo.CatalogoID)
	if err != nil || motivos.ValidarReferenciaMotivoAutorizacionV2(ctx, c.Motivo, reloj.Ahora()) != nil {
		return nil, errActivacionOfertaCTDesarrollo
	}
	pdp, err := appvec.NuevoServicioAutorizacionSolicitudLigadaV3(f, registro, registro, motivos, reloj,
		seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return nil, errActivacionOfertaCTDesarrollo
	}
	return &autoridadPerfilOfertaCTDesarrollo{privado: privado, sesion: sesion,
		fuente: f, pdp: pdp, motivo: c.Motivo}, nil
}

func nuevaAutoridadSeleccionOfertaCTDesarrollo(ctx context.Context, cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	base *preparadorBorradorLlamamientoDesarrollo,
	publico catalogoFronterasComunDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*autoridadSeleccionOfertaCTDesarrollo, func(), bool) {
	cerrarNada := func() {}
	const duracion = 3 * time.Second
	if ctx == nil || alta == nil || alta.soporte == nil || !alta.postgresql.seleccionOferta ||
		alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil ||
		base == nil || base.sesion == nil || base.soporte == nil || base.soporte.soporteCanal == nil ||
		publico.identidad == nil || derivador == nil || !derivador.valido() {
		return nil, cerrarNada, false
	}
	sonda, cancelar := context.WithTimeout(ctx, duracion)
	defer cancelar()
	etapa := "configuracion"
	completo := false
	var causa error
	var poolFuente, poolMotivos *pgxpool.Pool
	defer func() {
		if !completo {
			if poolMotivos != nil {
				poolMotivos.Close()
			}
			if poolFuente != nil {
				poolFuente.Close()
			}
			if causa != nil {
				slog.Warn("oferta CT cerrada", "etapa", etapa,
					"clase", telemetria.ClaseErrorArranque(causa),
					"causa", telemetria.MensajeErrorArranque(causa))
			}
		}
	}()
	fallar := func(err error) (*autoridadSeleccionOfertaCTDesarrollo, func(), bool) {
		causa = err
		return nil, cerrarNada, false
	}
	c, presente, err := leerActivacionOfertaCTDesarrollo(cfg)
	if err != nil {
		return fallar(err)
	}
	if !presente {
		return nil, cerrarNada, false
	}
	etapa = "frontera"
	baseProveedor, ok := base.sesion.sesion.(*proveedorSesionConsultaRRHHDesarrollo)
	frontera, existe := publico.resolver(http.MethodPost, cthttp.RutaSeleccionLlamamiento)
	if !ok || baseProveedor == nil || baseProveedor.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(baseProveedor.registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(baseProveedor.revalidador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(baseProveedor.resolutor) ||
		!existe || !baseProveedor.fronteras.mismaInstancia(publico) ||
		baseProveedor.soporte != base.soporte.soporteCanal ||
		baseProveedor.soporte.sello != alta.soporte.sello ||
		baseProveedor.soporte.principalID != alta.soporte.principalID ||
		baseProveedor.soporte.certificadoSHA256 != alta.soporte.certificadoSHA256 ||
		!frontera.admitePerfil(alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef) {
		return fallar(errActivacionOfertaCTDesarrollo)
	}
	for _, perfil := range []perfilActivacionOfertaCT{c.Vinculo, c.CierreBolsa, c.DescarteCT} {
		if perfil.PerfilRef == alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef ||
			perfil.PerfilRef == baseProveedor.base.Contexto.PerfilActivoRef {
			return fallar(errActivacionOfertaCTDesarrollo)
		}
	}
	if base.soporte.unidadRef == "" || base.soporte.ambitoRef == "" {
		return fallar(errActivacionOfertaCTDesarrollo)
	}
	etapa = "fuentes"
	dsnFuente, err := leerDSNOfertaCTDesarrollo(cfg, c.FuenteAutorizacionDSNFile)
	if err != nil {
		return fallar(err)
	}
	dsnMotivos, err := leerDSNOfertaCTDesarrollo(cfg, c.MotivosAutorizacionDSNFile)
	if err != nil {
		return fallar(err)
	}
	poolFuente, err = abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnFuente,
		config.RolFuenteAutorizacionAuditoriaDesarrollo, "vec-ct-ofertas-fuente")
	if err != nil {
		return fallar(err)
	}
	poolMotivos, err = abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnMotivos,
		config.RolMotivosAuditoriaDesarrollo, "vec-ct-ofertas-motivos")
	if err != nil {
		return fallar(err)
	}
	if !loginPoolsOfertaCTSeparados(poolFuente, poolMotivos,
		alta.postgresql.ejecucion, alta.postgresql.gobierno, alta.postgresql.registroAutorizacion, alta.postgresql.bolsa) {
		return fallar(errActivacionOfertaCTDesarrollo)
	}
	fuenteDB, err := pgvec.NuevoAlmacenAutorizacion(poolFuente)
	if err != nil {
		return fallar(err)
	}
	registro, err := pgvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return fallar(err)
	}
	etapa = "perfiles_publicados"
	configuraciones := []struct {
		perfil         perfilActivacionOfertaCT
		politica       politicaPerfilOfertaCTDesarrollo
		consultaEstado bool
	}{
		{c.Vinculo, politicaVinculoOfertaCTDesarrollo(), true},
		{c.CierreBolsa, politicaCierreOfertaBolsaCTDesarrollo(base.soporte.unidadRef, base.soporte.ambitoRef), false},
		{c.DescarteCT, politicaDescarteOfertaCTDesarrollo(), false},
	}
	perfiles := make([]*autoridadPerfilOfertaCTDesarrollo, 0, len(configuraciones))
	for _, configuracion := range configuraciones {
		perfil, e := componerPerfilAutoridadOfertaCTDesarrollo(sonda, configuracion.perfil,
			configuracion.politica, configuracion.consultaEstado, alta, baseProveedor, fuenteDB, registro, poolMotivos, reloj)
		if e != nil {
			return fallar(e)
		}
		perfiles = append(perfiles, perfil)
	}
	etapa = "material_capacidad"
	baseDerivada, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return fallar(err)
	}
	defer baseDerivada.borrarCopiasEfimeras()
	baseMaterial := alta.postgresql.proveedorMaterial
	if baseMaterial == nil || baseMaterial.atestador == nil || baseMaterial.confianza == nil ||
		baseMaterial.fuenteConfianza == nil {
		return fallar(errActivacionOfertaCTDesarrollo)
	}
	baseDerivada.fuenteConfianza = baseMaterial.fuenteConfianza
	// Solo tras comprobar los tres perfiles y motivos publicados se gobiernan
	// las tres capacidades técnicas. Ninguna asigna un rol o perfil al actor.
	for indice, perfil := range perfiles {
		proveedor, e := nuevoProveedorMaterialConsumidorDesarrollo(sonda,
			alta.postgresql.gobierno, baseDerivada, alta.soporte, reloj,
			alta.postgresql.catalogoMaterial, configuraciones[indice].politica.audiencia)
		if e != nil {
			return fallar(e)
		}
		perfil.material = proveedor
	}
	a := &autoridadSeleccionOfertaCTDesarrollo{alta: alta, publico: publico,
		vinculo: perfiles[0], cierreBolsa: perfiles[1], descarteCT: perfiles[2], reloj: reloj}
	completo = true
	cerrar := func() { poolMotivos.Close(); poolFuente.Close() }
	return a, cerrar, true
}

func loginPoolsOfertaCTSeparados(fuente, motivos *pgxpool.Pool, reservados ...*pgxpool.Pool) bool {
	if fuente == nil || motivos == nil || fuente.Config() == nil || motivos.Config() == nil {
		return false
	}
	uFuente, uMotivos := fuente.Config().ConnConfig.User, motivos.Config().ConnConfig.User
	if uFuente == "" || uMotivos == "" || uFuente == uMotivos {
		return false
	}
	for _, reservado := range reservados {
		if reservado != nil && reservado.Config() != nil &&
			(reservado.Config().ConnConfig.User == uFuente || reservado.Config().ConnConfig.User == uMotivos) {
			return false
		}
	}
	return true
}
