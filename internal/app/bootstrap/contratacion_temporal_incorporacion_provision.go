package bootstrap

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	alta "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	lectura "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Los tres perfiles pertenecen a la misma persona. Cada geometría de ámbito
// tiene una asignación fija; lectura y confirmación conservan un único perfil.
const (
	claveIncorporacionDetalle = "incorporacion_detalle"
	claveIncorporacionAlta    = "incorporacion_alta_personal"
	claveIncorporacionCT      = "incorporacion_personal_ct"
)

type perfilesNominalesIncorporacion struct {
	soporte           *soporteAltaContratacionTemporalDesarrollo
	consultas         *autoridadConsultasRRHHDesarrollo
	detalle, alta, ct *perfilFijoCTDesarrollo
	b2                map[string]*perfilFijoCTDesarrollo
	montajeB2         *montajeIncorporacionPersonalB2
	legadoCompuesto   bool
}

func principalCanalNominalIncorporacion(s *soporteAltaContratacionTemporalDesarrollo) core.Principal {
	return core.Principal{ID: s.principalID, AuthMethod: core.AuthMethodCertificate, AuthAssurance: core.AuthAssuranceHigh, Attributes: map[string]string{"certificate_sha256": s.certificadoSHA256, "autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment}}
}

func nuevoPerfilNominalIncorporacion(s *soporteAltaContratacionTemporalDesarrollo, refs ReferenciasCTIncorporacionDesarrollo, clave string, centros []string, ahora time.Time) (*perfilFijoCTDesarrollo, error) {
	if s == nil || !refs.valida() {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	base, err := s.contexto.Vinculo.Datos()
	if err != nil || base.PrincipalID != refs.PrincipalV3Ref || base.PerfilActivoRef != refs.PerfilV3Ref {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principalCanalNominalIncorporacion(s), ahora, discriminadorPerfilFijoCTDesarrollo(clave))
	if err != nil {
		return nil, err
	}
	v, err := contexto.Vinculo.Datos()
	if err != nil || v.PerfilActivoRef == base.PerfilActivoRef || v.CuentaRef != base.CuentaRef || contexto.Resultado.Contexto.PersonaRef != s.contexto.Resultado.Contexto.PersonaRef {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	ambitos := []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{refs.OrganizacionRef}}}
	permiso := func(accion, modulo, tipo, finalidad string) core.ConcesionRol {
		return core.ConcesionRol{Accion: accion, ModuloID: modulo, TipoRecurso: tipo, Finalidades: []string{finalidad}, GarantiaMinima: core.AuthAssuranceHigh}
	}
	var concesiones []core.ConcesionRol
	switch clave {
	case claveIncorporacionDetalle:
		ambitos = append(ambitos, core.AmbitoPerfil{Clave: "clase_ambito", Valores: []string{string(ct.AmbitoOrganizacionRRHH)}}, core.AmbitoPerfil{Clave: "ambito_ref", Valores: []string{refs.OrganizacionRef}})
		concesiones = []core.ConcesionRol{permiso(ct.AccionConsultarDetalleRRHH, ct.ModuloContratacion, ct.TipoRecursoExpediente, ct.FinalidadConsultarDetalleRRHH)}
	case claveIncorporacionAlta:
		centros = slices.Clone(centros)
		slices.Sort(centros)
		if len(centros) == 0 || len(centros) > 256 || len(slices.Compact(slices.Clone(centros))) != len(centros) {
			return nil, ct.ErrComposicionIncorporacionAplicacion
		}
		ambitos = append(ambitos, core.AmbitoPerfil{Clave: "centro_ref", Valores: centros})
		concesiones = []core.ConcesionRol{permiso(alta.AccionAltaEjercicio, "personal", alta.TipoRecursoAltaEjercicio, alta.FinalidadAltaEjercicio)}
	case claveIncorporacionCT:
		ambitos = append(ambitos, core.AmbitoPerfil{Clave: "unidad_ref", Valores: []string{refs.UnidadRef}})
		concesiones = []core.ConcesionRol{permiso(lectura.Accion, "personal", lectura.TipoRecursoV2, lectura.Finalidad), permiso(ct.AccionConfirmarIncorporacion, ct.ModuloContratacion, ct.TipoRecursoConfirmacionIncorporacionV2, ct.FinalidadConfirmarIncorporacion)}
	default:
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	plantilla, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, ahora, clave, clave, clave, concesiones, ambitos)
	if err != nil {
		return nil, err
	}
	return &perfilFijoCTDesarrollo{clave: clave, contexto: contexto, plantilla: plantilla, actoSesion: actoSesionPerfilFijoCTDesarrollo}, nil
}

func nuevosPerfilesNominalesIncorporacion(s *soporteAltaContratacionTemporalDesarrollo, consultas *autoridadConsultasRRHHDesarrollo, refs ReferenciasCTIncorporacionDesarrollo, centros []string, ahora time.Time) (*perfilesNominalesIncorporacion, error) {
	if consultas == nil || consultas.soporte != s {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	detalle, err := nuevoPerfilNominalIncorporacion(s, refs, claveIncorporacionDetalle, nil, ahora)
	if err != nil {
		return nil, err
	}
	alta, err := nuevoPerfilNominalIncorporacion(s, refs, claveIncorporacionAlta, centros, ahora)
	if err != nil {
		return nil, err
	}
	confirmacion, err := nuevoPerfilNominalIncorporacion(s, refs, claveIncorporacionCT, nil, ahora)
	if err != nil {
		return nil, err
	}
	return &perfilesNominalesIncorporacion{soporte: s, consultas: consultas, detalle: detalle, alta: alta, ct: confirmacion, legadoCompuesto: true}, nil
}

// Debe invocarse antes de construir el catálogo inmutable. La selección
// interna sigue cerrada por operación; estos datos nunca vienen del cliente.
func asignarPerfilesNominalesIncorporacionEnFronteras(s *soporteAltaContratacionTemporalDesarrollo, declaraciones []descriptorFronteraComunDesarrollo) ([]descriptorFronteraComunDesarrollo, error) {
	if s == nil {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	principal := principalCanalNominalIncorporacion(s)
	ahora := s.reloj.Ahora()
	perfiles := make([]string, 0, 3)
	for _, clave := range []string{claveIncorporacionDetalle, claveIncorporacionAlta, claveIncorporacionCT} {
		c, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principal, ahora, discriminadorPerfilFijoCTDesarrollo(clave))
		if err != nil {
			return nil, err
		}
		perfiles = append(perfiles, c.Resultado.Contexto.PerfilActivoRef)
	}
	resultado := append([]descriptorFronteraComunDesarrollo(nil), declaraciones...)
	encontrada := false
	for i := range resultado {
		if resultado[i].Metodo == "POST" && resultado[i].Ruta == httpinterno.RutaConsultaDetalleRRHH {
			if !esDescriptorDetalleContratacionTemporalDesarrollo(resultado[i], s.contexto.Resultado.Contexto.PerfilActivoRef) {
				return nil, ct.ErrComposicionIncorporacionAplicacion
			}
			for _, perfil := range perfiles {
				if resultado[i].admitePerfil(perfil) {
					return nil, ct.ErrComposicionIncorporacionAplicacion
				}
			}
			resultado[i].PerfilesActivosRef = append(slices.Clone(resultado[i].PerfilesActivosRef), perfiles...)
			encontrada = true
		}
	}
	if !encontrada {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return resultado, nil
}

// Sólo composición previa al servicio HTTP. El helper común lee la preimagen
// real y publica por CAS; no fabrica versiones anteriores ni restaura revocaciones.
func provisionarPerfilesNominalesIncorporacion(ctx context.Context, pool *pgxpool.Pool, p *perfilesNominalesIncorporacion, aprobacion aprobacionProvisionPerfilesRRHHDesarrollo) error {
	if p == nil || p.soporte == nil {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	for _, perfil := range p.todos() {
		estado, err := asegurarPerfilFijoCTDesarrollo(ctx, pool, p.soporte, perfil, aprobacion, preimagenPropiaPerfilFijoCTDesarrollo(perfil, actoAsignacionPerfilFijoCTDesarrollo))
		if err != nil {
			return err
		}
		if estado == perfilFijoPendienteProvision {
			return errPerfilFijoCTNoConsumible
		}
		if err := publicarContextoPerfilFijoCTDesarrollo(ctx, pool, perfil); err != nil {
			return err
		}
	}
	return nil
}

func configurarSesionesNominalesIncorporacion(ctx context.Context, p *perfilesNominalesIncorporacion, base *proveedorSesionConsultaRRHHDesarrollo) error {
	if p == nil || base == nil || p.soporte != base.soporte {
		return ct.ErrComposicionIncorporacionAplicacion
	}
	for _, perfil := range p.todos() {
		esperado, err := contextoEsperadoRegistradoParaSemillaDesarrollo(ctx, base.resolutor, p.soporte, perfil.contexto.Resultado)
		if err != nil {
			return err
		}
		sesion, err := nuevaSesionReincorporacionTitularDesarrollo(base, esperado)
		if err != nil {
			return err
		}
		proveedor, ok := sesion.(*proveedorSesionConsultaRRHHDesarrollo)
		if !ok || proveedor == nil {
			return ct.ErrComposicionIncorporacionAplicacion
		}
		perfil.contextoEsperadoRegistrado, perfil.sesionOperativa = esperado, &sesionNominalIncorporacion{proveedor: proveedor}
	}
	return nil
}

type claveContextosNominalesIncorporacion struct{}
type capturaContextosNominalesIncorporacion struct {
	mu          sync.Mutex
	propietario *perfilesNominalesIncorporacion
	contextos   map[string]ct.ContextoAutorizacionAltaV3
}

func (p *perfilesNominalesIncorporacion) resolver(ctx context.Context, perfil *perfilFijoCTDesarrollo) (ct.ContextoAutorizacionAltaV3, error) {
	vacio := ct.ContextoAutorizacionAltaV3{}
	if p == nil || p.soporte == nil || perfil == nil || ctx == nil || ctx.Value(claveIncorporacionV2Desarrollo{}) != p.soporte.sello {
		return vacio, ct.ErrAutorizacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	holder, ok := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	if !ok || holder == nil || !p.contiene(perfil) {
		return vacio, ct.ErrAutorizacionDenegada
	}
	if _, valida := p.soporte.capacidadValida(ctx); !valida {
		return vacio, ct.ErrAutorizacionDenegada
	}
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.propietario != nil && holder.propietario != p {
		return vacio, ct.ErrAutorizacionDenegada
	}
	holder.propietario = p
	if holder.contextos == nil {
		holder.contextos = map[string]ct.ContextoAutorizacionAltaV3{}
	}
	contexto, existe := holder.contextos[perfil.clave]
	if !existe {
		if perfil.sesionOperativa == nil {
			return vacio, ct.ErrConsultaRRHHNoDisponible
		}
		comun, err := perfil.sesionOperativa.ResolverContexto(ctx)
		if err != nil {
			return vacio, err
		}
		contexto = ct.ContextoAutorizacionAltaV3{Vinculo: comun.Vinculo, Resultado: comun.Resultado}
		if !mismoContextoEsperadoRegistradoDesarrollo(perfil.contextoEsperadoRegistrado, contexto.Resultado) {
			return vacio, ct.ErrAutorizacionDenegada
		}
		holder.contextos[perfil.clave] = contexto
	}
	v, err := contexto.Vinculo.Datos()
	if err != nil || v.PrincipalID != perfil.plantilla.AsignacionPerfil.PrincipalID || v.PerfilActivoRef != perfil.perfilRef() || contexto.ValidarPara(ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: perfil.perfilRef()}, p.soporte.reloj.Ahora()) != nil {
		return vacio, ct.ErrAutorizacionDenegada
	}
	resultado, err := contexto.Resultado.Clonar()
	if err != nil {
		return vacio, ct.ErrAutorizacionDenegada
	}
	return ct.ContextoAutorizacionAltaV3{Vinculo: contexto.Vinculo, Resultado: resultado}, nil
}

func (p *perfilesNominalesIncorporacion) consumir(ctx context.Context, perfil *perfilFijoCTDesarrollo) error {
	if p == nil || p.soporte == nil || perfil == nil || ctx == nil {
		return ct.ErrAutorizacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	p.soporte.mu.Lock()
	lector, ok := p.soporte.autoridadAsignaciones.(lectorAsignacionPublicadaCTDesarrollo)
	p.soporte.mu.Unlock()
	if !ok || dependenciaEsNulaContratacionTemporalDesarrollo(lector) {
		return ct.ErrConsultaRRHHNoDisponible
	}
	publicada, encontrada, err := lector.leerAsignacionPublicada(ctx, perfil.perfilRef())
	if err != nil {
		return errors.Join(ct.ErrConsultaRRHHNoDisponible, err)
	}
	if !encontrada || publicada.actoAsignacion != actoAsignacionPerfilFijoCTDesarrollo {
		return ct.ErrAutorizacionDenegada
	}
	if _, valida := instantaneaConsumible(publicada, perfil.plantilla, p.soporte.reloj.Ahora()); !valida {
		return ct.ErrAutorizacionDenegada
	}
	return nil
}
