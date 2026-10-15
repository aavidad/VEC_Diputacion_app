package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	appvec "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const claveFronteraSeleccionOfertaCTDesarrollo = "ct-oferta-bolsa-seleccionar-privada"

// La fuente solo consulta la asignación publicada. El fichero de activación
// identifica la versión aprobada, pero no contiene ni concede un rol.
type fuenteRolSeleccionOfertaCTDesarrollo struct {
	fuente          vp.FuenteAutorizacion
	principalID     string
	personaRef      string
	perfilRef       string
	organizacionRef string
	rol             referenciaRolOfertaCTDesarrollo
	reloj           relojContratacionTemporalDesarrollo
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
		!concesionExactaSeleccionOfertaCTDesarrollo(i.VersionRol.Concesiones[0]) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	huella, err := i.VersionRol.HuellaSHA256()
	if err != nil || huella != f.rol.HuellaSHA256 {
		return vacia, ports.ErrAutorizacionDenegada
	}
	org := false
	claves := map[string]bool{"organizacion_ref": false, "centro_ref": false, "categoria_ref": false}
	for _, ambito := range i.AsignacionPerfil.Ambitos {
		if _, existe := claves[ambito.Clave]; !existe {
			return vacia, ports.ErrAutorizacionDenegada
		}
		claves[ambito.Clave] = true
		if ambito.Clave == "organizacion_ref" && slices.Contains(ambito.Valores, f.organizacionRef) {
			org = true
		}
	}
	if !org || len(i.AsignacionPerfil.Ambitos) != len(claves) || !claves["centro_ref"] || !claves["categoria_ref"] {
		return vacia, ports.ErrAutorizacionDenegada
	}
	return i, nil
}

func concesionExactaSeleccionOfertaCTDesarrollo(c core.ConcesionRol) bool {
	return c.Accion == ports.AccionVincularOfertaBolsaCT &&
		c.ModuloID == ports.ModuloContratacion &&
		c.TipoRecurso == ports.TipoRecursoVinculoOfertaBolsaCT &&
		len(c.Finalidades) == 1 && c.Finalidades[0] == ports.FinalidadVincularOfertaBolsaCT &&
		c.GarantiaMinima == core.AuthAssuranceHigh &&
		len(c.CamposPermitidos) == 0 && len(c.Obligaciones) == 0
}

type autoridadSeleccionOfertaCTDesarrollo struct {
	alta     *dependenciasAltaContratacionTemporalDesarrollo
	publico  catalogoFronterasComunDesarrollo
	privado  catalogoFronterasComunDesarrollo
	sesion   *seguridadComunDesarrollo
	fuente   *fuenteRolSeleccionOfertaCTDesarrollo
	pdp      vp.AutorizadorSolicitudLigadaV3
	material *proveedorMaterialAltaContratacionTemporalDesarrollo
	motivo   core.ReferenciaEntradaCatalogo
	reloj    relojContratacionTemporalDesarrollo
}

var _ ports.AutorizadorSeleccionOfertaCT = (*autoridadSeleccionOfertaCTDesarrollo)(nil)

func (a *autoridadSeleccionOfertaCTDesarrollo) MotivoOferta() core.ReferenciaEntradaCatalogo {
	if a == nil {
		return core.ReferenciaEntradaCatalogo{}
	}
	return a.motivo
}

// ContextoOferta registra un perfil nominal nuevo para la misma petición
// certificada. El hijo privado no se instala en el contexto exterior que
// después usa Bolsa para comprobar su propio perfil y audiencia.
func (a *autoridadSeleccionOfertaCTDesarrollo) ContextoOferta(ctx context.Context) (context.Context, ports.ContextoAutorizacionAltaV3, error) {
	var vacio ports.ContextoAutorizacionAltaV3
	if a == nil || a.alta == nil || a.alta.soporte == nil || a.sesion == nil ||
		a.fuente == nil || ctx == nil || ctx.Err() != nil {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := a.alta.soporte.capacidadValida(ctx)
	frontera, real := fronteraSeguridadComunDesdeContexto(ctx)
	if !valida || !real || !a.publico.mismaInstancia(frontera.catalogo) ||
		capacidad.metodo != http.MethodPost || capacidad.ruta != cthttp.RutaSeleccionLlamamiento ||
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
	if _, err := a.fuente.ObtenerInstantaneaAutorizacion(ctx, a.fuente.principalID, a.fuente.perfilRef); err != nil {
		if errors.Is(err, vp.ErrFuenteAutorizacionNoDisponible) {
			return nil, vacio, ports.ErrIntegracionBolsaNoDisponible
		}
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	descriptor, existe := a.privado.resolver(capacidad.metodo, capacidad.ruta)
	if !existe || !descriptor.admitePerfil(a.fuente.perfilRef) {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	capacidad.consultaRRHH, capacidad.contextoOperacion, capacidad.original = nil, nil, nil
	hijo := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	hijo = context.WithValue(hijo, claveContextosNominalesIncorporacion{}, (*capturaContextosNominalesIncorporacion)(nil))
	hijo = context.WithValue(hijo, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: capacidad.metodo, ruta: capacidad.ruta,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: a.privado, descriptor: descriptor})
	hijo = context.WithValue(hijo, claveCacheSeguridadComunDesarrollo{}, &cacheSeguridadComunDesarrollo{})
	actor, err := a.sesion.ResolverContexto(hijo)
	if err != nil || actor.Resultado.Validar() != nil || actor.Vinculo.ValidarPara(actor.Resultado) != nil ||
		!actor.Vinculo.VigenteEn(a.reloj.Ahora(), actor.Resultado) ||
		actor.Resultado.Contexto.PerfilActivoRef != a.fuente.perfilRef ||
		actor.Resultado.Contexto.Principal.ID != original.Resultado.Contexto.Principal.ID ||
		actor.Resultado.Contexto.PersonaRef != a.fuente.personaRef ||
		actor.Resultado.Contexto.PersonaRef != original.Resultado.Contexto.PersonaRef ||
		actor.Resultado.Contexto.Instantanea.CuentaRef != original.Resultado.Contexto.Instantanea.CuentaRef {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	actorDatos, errActor := actor.Vinculo.Datos()
	ctDatos, errCT := original.Vinculo.Datos()
	if errActor != nil || errCT != nil || actorDatos.PrincipalID != ctDatos.PrincipalID ||
		actorDatos.CuentaRef != ctDatos.CuentaRef || actorDatos.PerfilActivoRef != a.fuente.perfilRef {
		return nil, vacio, ports.ErrAutorizacionDenegada
	}
	return hijo, ports.ContextoAutorizacionAltaV3{Vinculo: actor.Vinculo, Resultado: actor.Resultado}, nil
}

func (a *autoridadSeleccionOfertaCTDesarrollo) AutorizarSeleccionOfertaCT(ctx context.Context, s ports.SolicitudAutorizacionSeleccionOfertaCT) (ports.AutorizacionSeleccionOfertaCT, error) {
	var vacia ports.AutorizacionSeleccionOfertaCT
	if a == nil || a.pdp == nil || a.material == nil || s.Intencion.Validar() != nil ||
		s.Intencion.MotivoAutorizacion != a.motivo {
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
			"centro_ref": s.Intencion.CentroRef, "categoria_ref": s.Intencion.CategoriaRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(huella[:])}}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: s.Intencion.Vinculo, ReferenciaMotivo: a.motivo,
		Accion: ports.AccionVincularOfertaBolsaCT, Recurso: recurso,
		Finalidad: ports.FinalidadVincularOfertaBolsaCT, Correlacion: s.Intencion.Correlacion})
	if err != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	decision, confirmacion, err := a.pdp.ExigirSolicitudLigadaV3(hijo, solicitud, s.Intencion.ResultadoContexto)
	if err != nil {
		return vacia, errorAutorizacionOfertaCTDesarrollo(hijo, err)
	}
	material, err := a.material.proveerMaterialConfirmacion(hijo, solicitud, decision, confirmacion, a.motivo, s.Intencion.ResultadoContexto)
	if err != nil || material.ValidarEstructura() != nil {
		return vacia, ports.ErrIntegracionBolsaNoDisponible
	}
	return ports.AutorizacionSeleccionOfertaCT{ActorRef: s.Intencion.ActorRef, PerfilRef: s.Intencion.PerfilRef,
		MaterialCanonico: canon, Material: material}, nil
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
	var poolFuente, poolMotivos *pgxpool.Pool
	defer func() {
		if !completo {
			if poolMotivos != nil {
				poolMotivos.Close()
			}
			if poolFuente != nil {
				poolFuente.Close()
			}
			slog.Warn("oferta CT cerrada", "etapa", etapa)
		}
	}()
	c, presente, err := leerActivacionOfertaCTDesarrollo(cfg)
	if !presente || err != nil || c.PerfilRef == alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef {
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
		return nil, cerrarNada, false
	}
	privado, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave:      claveFronteraSeleccionOfertaCTDesarrollo,
		Superficie: superficieInternaSeguridadComunDesarrollo,
		Metodo:     http.MethodPost, Ruta: cthttp.RutaSeleccionLlamamiento,
		PerfilesActivosRef: []string{c.PerfilRef},
		ClavePolitica:      clavePoliticaContratacionTemporalDesarrollo,
		ClaveCapacidad:     ports.AccionVincularOfertaBolsaCT,
	}})
	if err != nil {
		return nil, cerrarNada, false
	}
	etapa = "contexto"
	cuenta := alta.soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef
	esperado, err := baseProveedor.resolutor.ResolverContextoActorRegistradoV2(sonda, core.SolicitudContextoActor{
		Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta,
			Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh},
		PerfilActivoRef: c.PerfilRef})
	if err != nil || esperado.Validar() != nil ||
		esperado.AutoridadEfectiva != core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		esperado.Contexto.Principal.ID != alta.soporte.principalID ||
		esperado.Contexto.PersonaRef != alta.soporte.contexto.Resultado.Contexto.PersonaRef ||
		esperado.Contexto.Instantanea.CuentaRef != cuenta ||
		esperado.Contexto.PerfilActivoRef != c.PerfilRef ||
		!esperado.Contexto.Instantanea.VigenteEn(reloj.Ahora()) {
		return nil, cerrarNada, false
	}
	proveedorPrivado, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(
		baseProveedor.soporte, baseProveedor.registro, baseProveedor.revalidador,
		baseProveedor.reloj, baseProveedor.resolutor, privado)
	if err != nil {
		return nil, cerrarNada, false
	}
	privada, err := nuevaSesionReincorporacionTitularDesarrollo(proveedorPrivado, esperado)
	if err != nil {
		return nil, cerrarNada, false
	}
	sesion, err := nuevaSeguridadComunDesarrollo(privada, reloj)
	if err != nil {
		return nil, cerrarNada, false
	}
	etapa = "fuentes"
	dsnFuente, err := leerDSNOfertaCTDesarrollo(cfg, c.FuenteAutorizacionDSNFile)
	if err != nil {
		return nil, cerrarNada, false
	}
	dsnMotivos, err := leerDSNOfertaCTDesarrollo(cfg, c.MotivosAutorizacionDSNFile)
	if err != nil {
		return nil, cerrarNada, false
	}
	poolFuente, err = abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnFuente,
		config.RolFuenteAutorizacionAuditoriaDesarrollo, "vec-ct-ofertas-fuente")
	if err != nil {
		return nil, cerrarNada, false
	}
	poolMotivos, err = abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnMotivos,
		config.RolMotivosAuditoriaDesarrollo, "vec-ct-ofertas-motivos")
	if err != nil || !loginPoolsOfertaCTSeparados(poolFuente, poolMotivos,
		alta.postgresql.ejecucion, alta.postgresql.gobierno, alta.postgresql.registroAutorizacion, alta.postgresql.bolsa) {
		return nil, cerrarNada, false
	}
	fuenteDB, err := pgvec.NuevoAlmacenAutorizacion(poolFuente)
	if err != nil {
		return nil, cerrarNada, false
	}
	f := &fuenteRolSeleccionOfertaCTDesarrollo{fuente: fuenteDB,
		principalID: alta.soporte.principalID, personaRef: esperado.Contexto.PersonaRef,
		perfilRef: c.PerfilRef, organizacionRef: organizacionAltaContratacionTemporalDesarrollo, rol: c.Rol, reloj: reloj}
	if _, err = f.ObtenerInstantaneaAutorizacion(sonda, f.principalID, f.perfilRef); err != nil {
		return nil, cerrarNada, false
	}
	motivos, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(poolMotivos, c.Motivo.CatalogoID)
	if err != nil || motivos.ValidarReferenciaMotivoAutorizacionV2(sonda, c.Motivo, reloj.Ahora()) != nil {
		return nil, cerrarNada, false
	}
	registro, err := pgvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return nil, cerrarNada, false
	}
	pdp, err := appvec.NuevoServicioAutorizacionSolicitudLigadaV3(f, registro, registro, motivos, reloj,
		seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return nil, cerrarNada, false
	}
	etapa = "material_capacidad"
	baseDerivada, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return nil, cerrarNada, false
	}
	defer baseDerivada.borrarCopiasEfimeras()
	baseMaterial := alta.postgresql.proveedorMaterial
	if baseMaterial == nil || baseMaterial.atestador == nil || baseMaterial.confianza == nil ||
		baseMaterial.fuenteConfianza == nil {
		return nil, cerrarNada, false
	}
	baseDerivada.fuenteConfianza = baseMaterial.fuenteConfianza
	// Solo tras comprobar el perfil y el motivo publicados se gobierna la
	// capacidad técnica propia. Esta publicación no asigna rol ni perfil.
	proveedorMaterial, err := nuevoProveedorMaterialConsumidorDesarrollo(sonda,
		alta.postgresql.gobierno, baseDerivada, alta.soporte, reloj,
		alta.postgresql.catalogoMaterial, ports.AudienciaVincularOfertaBolsaCT)
	if err != nil {
		return nil, cerrarNada, false
	}
	a := &autoridadSeleccionOfertaCTDesarrollo{alta: alta, publico: publico, privado: privado,
		sesion: sesion, fuente: f, pdp: pdp, material: proveedorMaterial, motivo: c.Motivo, reloj: reloj}
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
