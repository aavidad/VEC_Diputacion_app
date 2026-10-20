package bootstrap

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
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
			if !esDescriptorDetalleContratacionTemporalDesarrollo(resultado[i], s) {
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
	capturaPerfilesNominalesCT
	propietario       *perfilesNominalesIncorporacion
	asignacionesB2    map[*perfilFijoCTDesarrollo]instantaneaPublicadaDesarrollo
	muPersonasB2      chan struct{}
	personasB2        map[clavePersonaBolsaB2]lecturaPersonaBolsaB2
	personasNativasB2 map[clavePersonaNativaB2]lecturaPersonaNativaB2
	preparacionB2     *preparacionIncorporacionB2
}

func (p *perfilesNominalesIncorporacion) resolver(ctx context.Context, perfil *perfilFijoCTDesarrollo) (ct.ContextoAutorizacionAltaV3, error) {
	vacio := ct.ContextoAutorizacionAltaV3{}
	if p == nil || p.soporte == nil || perfil == nil || ctx == nil || ctx.Value(claveIncorporacionV2Desarrollo{}) != p.soporte.sello {
		return vacio, ct.ErrAutorizacionDenegada
	}
	holder, ok := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	if !ok || holder == nil || !p.contiene(perfil) {
		return vacio, ct.ErrAutorizacionDenegada
	}
	holder.mu.Lock()
	ajeno := holder.propietario != nil && holder.propietario != p
	holder.mu.Unlock()
	if ajeno {
		return vacio, ct.ErrAutorizacionDenegada
	}
	contexto, err := resolverContextoPerfilNominalCT(ctx, p.soporte, perfil, &holder.capturaPerfilesNominalesCT, p)
	if err != nil {
		return vacio, err
	}
	holder.mu.Lock()
	if holder.propietario != nil && holder.propietario != p {
		holder.mu.Unlock()
		return vacio, ct.ErrAutorizacionDenegada
	}
	holder.propietario = p
	holder.mu.Unlock()
	return contexto, nil
}

func (p *perfilesNominalesIncorporacion) consumir(ctx context.Context, perfil *perfilFijoCTDesarrollo) error {
	_, err := p.leerAsignacionConsumible(ctx, perfil)
	return err
}

// Sólo B2 reutiliza la primera lectura del perfil en la petición. Cada
// emisión conserva consumir después del PDP, que siempre vuelve a la fuente.
func (p *perfilesNominalesIncorporacion) consumirInicialB2(ctx context.Context, perfil *perfilFijoCTDesarrollo) error {
	if p == nil || p.soporte == nil || perfil == nil || ctx == nil || !p.contiene(perfil) || ctx.Value(claveIncorporacionV2Desarrollo{}) != p.soporte.sello {
		return ct.ErrAutorizacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	holder, ok := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	if !ok || holder == nil {
		return ct.ErrAutorizacionDenegada
	}
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.propietario != p {
		return ct.ErrAutorizacionDenegada
	}
	publicada, existe := holder.asignacionesB2[perfil]
	if !existe {
		var err error
		publicada, err = p.leerAsignacionConsumible(ctx, perfil)
		if err != nil {
			return err
		}
		publicada.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(publicada.instantanea)
		if holder.asignacionesB2 == nil {
			holder.asignacionesB2 = map[*perfilFijoCTDesarrollo]instantaneaPublicadaDesarrollo{}
		}
		holder.asignacionesB2[perfil] = publicada
	}
	if _, valida := instantaneaConsumible(publicada, perfil.plantilla, p.soporte.reloj.Ahora()); !valida {
		return ct.ErrAutorizacionDenegada
	}
	return nil
}

func (p *perfilesNominalesIncorporacion) leerAsignacionConsumible(ctx context.Context, perfil *perfilFijoCTDesarrollo) (instantaneaPublicadaDesarrollo, error) {
	if p == nil || !p.contiene(perfil) {
		return instantaneaPublicadaDesarrollo{}, ct.ErrAutorizacionDenegada
	}
	return leerAsignacionConsumiblePerfilNominalCT(ctx, p.soporte, perfil)
}

// La captura sólo vive en la petición nominal. El material existente contiene
// valores y liga la proyección al contrato que devuelve el servicio de CT.
type preparacionIncorporacionB2 struct {
	fuente     *fuentesIncorporacionPersonalB2
	identidad  core.DatosVinculoAutenticacionActorV2
	material   domct.PlanIncorporacionPersonalB2
	proyeccion httpinterno.ProyeccionIncorporacionPersonalB2HTTP
}

func copiarProyeccionPreparacionB2(p httpinterno.ProyeccionIncorporacionPersonalB2HTTP) httpinterno.ProyeccionIncorporacionPersonalB2HTTP {
	p.Prerrequisitos = slices.Clone(p.Prerrequisitos)
	p.Opciones.Vacantes = slices.Clone(p.Opciones.Vacantes)
	p.Opciones.Regimenes = slices.Clone(p.Opciones.Regimenes)
	p.Opciones.Modalidades = slices.Clone(p.Opciones.Modalidades)
	p.Opciones.ClasesOcupacion = slices.Clone(p.Opciones.ClasesOcupacion)
	p.Opciones.Motivos = slices.Clone(p.Opciones.Motivos)
	p.Opciones.Documentos = slices.Clone(p.Opciones.Documentos)
	if p.Plan != nil {
		plan := *p.Plan
		p.Plan = &plan
	}
	if p.Recibo != nil {
		recibo := *p.Recibo
		p.Recibo = &recibo
	}
	return p
}

func (f *fuentesIncorporacionPersonalB2) capturarPreparacionB2(ctx context.Context, m domct.PlanIncorporacionPersonalB2, p httpinterno.ProyeccionIncorporacionPersonalB2HTTP) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	holder, _ := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	if holder == nil || f.autoridad == nil || f.autoridad.perfiles == nil {
		return nil
	}
	perfil := f.autoridad.perfiles.b2[ct.AccionRegistrarPlanNominalB2]
	if perfil == nil {
		return nil
	}
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.propietario != f.autoridad.perfiles {
		return nil
	}
	c, ok := holder.contextos[perfil.clave]
	if !ok {
		return nil
	}
	identidad, err := c.Vinculo.Datos()
	if err != nil {
		return err
	}
	holder.preparacionB2 = &preparacionIncorporacionB2{f, identidad, m, copiarProyeccionPreparacionB2(p)}
	return nil
}

func (f *fuentesIncorporacionPersonalB2) opcionesTrasPreparacionB2(ctx context.Context, c ct.ContratoPlanNominalB2) (httpinterno.ProyeccionIncorporacionPersonalB2HTTP, error) {
	var cero httpinterno.ProyeccionIncorporacionPersonalB2HTTP
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	holder, _ := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	if holder == nil {
		return f.ConsultarOpcionesIncorporacionB2(ctx, c.Material.ExpedienteRef)
	}
	holder.mu.Lock()
	captura := holder.preparacionB2
	propietario := holder.propietario
	holder.mu.Unlock()
	if captura == nil || captura.fuente != f || f.autoridad == nil || propietario != f.autoridad.perfiles || captura.material != c.Material {
		return f.ConsultarOpcionesIncorporacionB2(ctx, c.Material.ExpedienteRef)
	}
	actual, err := f.autoridad.contexto(ctx, ct.AccionRegistrarPlanNominalB2)
	if err != nil {
		return cero, err
	}
	identidad, err := actual.Vinculo.Datos()
	if err != nil {
		return cero, err
	}
	if identidad != captura.identidad {
		return f.ConsultarOpcionesIncorporacionB2(ctx, c.Material.ExpedienteRef)
	}
	// La captura no concede acceso: cada perfil relee su asignación vigente.
	for _, accion := range []string{ct.AccionConsultarDetalleRRHH, ct.AccionLeerPlanNominalB2} {
		if _, err := f.autoridad.contexto(ctx, accion); err != nil {
			return cero, err
		}
		if err := propietario.consumir(ctx, propietario.b2[accion]); err != nil {
			return cero, err
		}
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	m := c.Material
	a := ct.AntecedentesPlanNominalB2{OrganizacionRef: m.OrganizacionRef, ExpedienteRef: m.ExpedienteRef, VersionExpediente: m.VersionExpediente, CategoriaRef: m.CategoriaRef, DocumentoRef: m.DocumentoRef, DocumentoSHA256: m.DocumentoSHA256, AceptacionRef: m.AceptacionRef, AceptacionReciboRef: m.AceptacionReciboRef, Bolsa: m.Bolsa}
	return f.completarOpcionesIncorporacionB2(ctx, copiarProyeccionPreparacionB2(captura.proyeccion), a)
}
